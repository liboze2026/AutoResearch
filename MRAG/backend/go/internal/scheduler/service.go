package scheduler

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type experimentReader interface {
	GetByID(context.Context, string) (*model.Experiment, error)
	Update(context.Context, model.Experiment) error
}

type experimentSpecReader interface {
	GetLatestByExperimentID(context.Context, string) (*model.ExperimentSpec, error)
}

type runStore interface {
	GetByID(context.Context, string) (*model.ExperimentRun, error)
	Create(context.Context, model.ExperimentRun) error
	Update(context.Context, model.ExperimentRun) error
	CountByExperimentID(context.Context, string) (int, error)
	CountActiveByServerID(context.Context, string) (int, error)
}

type decisionStore interface {
	Create(context.Context, model.SchedulerDecision) error
	GetLatestByRunID(context.Context, string) (*model.SchedulerDecision, error)
}

type serverLister interface {
	List(context.Context) ([]model.Server, error)
}

type heartbeatReader interface {
	ListByServerID(context.Context, string, int) ([]model.ServerHeartbeat, error)
}

type gpuSnapshotReader interface {
	ListByServerID(context.Context, string, int) ([]model.GPUResourceSnapshot, error)
}

type Service struct {
	experiments   experimentReader
	specs         experimentSpecReader
	runs          runStore
	decisions     decisionStore
	servers       serverLister
	heartbeats    heartbeatReader
	gpuSnapshots  gpuSnapshotReader
	workspaceRoot string
}

func NewService(
	experiments experimentReader,
	specs experimentSpecReader,
	runs runStore,
	decisions decisionStore,
	servers serverLister,
	heartbeats heartbeatReader,
	gpuSnapshots gpuSnapshotReader,
	workspaceRoot string,
) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{
		experiments:   experiments,
		specs:         specs,
		runs:          runs,
		decisions:     decisions,
		servers:       servers,
		heartbeats:    heartbeats,
		gpuSnapshots:  gpuSnapshots,
		workspaceRoot: workspaceRoot,
	}
}

func (s *Service) QueueExperiment(ctx context.Context, experimentID string) (*model.ExperimentQueueResult, error) {
	exp, err := s.experiments.GetByID(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	if exp == nil {
		return nil, fmt.Errorf("experiment not found")
	}
	spec, err := s.specs.GetLatestByExperimentID(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	if spec == nil {
		return nil, fmt.Errorf("experiment spec not found")
	}

	runCount, err := s.runs.CountByExperimentID(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	paths := workspacepkg.New(s.workspaceRoot)
	runNumber := runCount + 1
	run := model.ExperimentRun{
		ID:            httpx.NewID("run"),
		ExperimentID:  experimentID,
		SpecID:        spec.ID,
		RunStatus:     "queued",
		RemoteWorkdir: filepath.Join(paths.ExperimentDir(experimentID), fmt.Sprintf("run_%d", runNumber)),
		RetryCount:    0,
		ResultJSON:    map[string]interface{}{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.runs.Create(ctx, run); err != nil {
		return nil, err
	}

	exp.CurrentRunID = run.ID
	exp.Status = "queued"
	exp.UpdatedAt = now
	if err := s.experiments.Update(ctx, *exp); err != nil {
		return nil, err
	}
	return &model.ExperimentQueueResult{ExperimentID: experimentID, Run: run}, nil
}

func (s *Service) ScheduleRun(ctx context.Context, runID string) (*model.ScheduleResult, error) {
	run, err := s.runs.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("run not found")
	}
	if run.RunStatus != "queued" && run.RunStatus != "scheduled" {
		return nil, fmt.Errorf("run is not queueable for scheduling")
	}

	candidates, err := s.buildCandidates(ctx)
	if err != nil {
		return nil, err
	}
	eligible := make([]model.SchedulerCandidate, 0)
	rejected := make([]map[string]interface{}, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Eligible {
			eligible = append(eligible, candidate)
		} else {
			rejected = append(rejected, map[string]interface{}{
				"serverId": candidate.ServerID,
				"reason":   candidate.IneligibleReason,
			})
		}
	}
	if len(eligible) == 0 {
		return nil, fmt.Errorf("no available server for scheduling")
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].BestFreeMemMB != eligible[j].BestFreeMemMB {
			return eligible[i].BestFreeMemMB > eligible[j].BestFreeMemMB
		}
		if !eligible[i].HeartbeatAt.Equal(eligible[j].HeartbeatAt) {
			return eligible[i].HeartbeatAt.After(eligible[j].HeartbeatAt)
		}
		if eligible[i].QueueLength != eligible[j].QueueLength {
			return eligible[i].QueueLength < eligible[j].QueueLength
		}
		return eligible[i].BestUtilization < eligible[j].BestUtilization
	})
	chosen := eligible[0]

	now := time.Now()
	run.AssignedServerID = chosen.ServerID
	run.RunStatus = "scheduled"
	run.UpdatedAt = now
	if err := s.runs.Update(ctx, *run); err != nil {
		return nil, err
	}

	decision := model.SchedulerDecision{
		ID:             httpx.NewID("sdec"),
		RunID:          run.ID,
		ChosenServerID: chosen.ServerID,
		DecisionJSON: map[string]interface{}{
			"ruleVersion": "stage2_scheduler_v1",
			"chosen":      chosen,
			"candidates":  candidates,
			"rejected":    rejected,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.decisions.Create(ctx, decision); err != nil {
		return nil, err
	}

	exp, err := s.experiments.GetByID(ctx, run.ExperimentID)
	if err != nil {
		return nil, err
	}
	if exp != nil {
		exp.CurrentRunID = run.ID
		exp.Status = "scheduled"
		exp.UpdatedAt = now
		if err := s.experiments.Update(ctx, *exp); err != nil {
			return nil, err
		}
	}

	return &model.ScheduleResult{Run: *run, Decision: decision, Chosen: chosen}, nil
}

func (s *Service) GetLatestDecision(ctx context.Context, runID string) (*model.SchedulerDecision, error) {
	run, err := s.runs.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("run not found")
	}
	return s.decisions.GetLatestByRunID(ctx, runID)
}

func (s *Service) buildCandidates(ctx context.Context) ([]model.SchedulerCandidate, error) {
	servers, err := s.servers.List(ctx)
	if err != nil {
		return nil, err
	}
	candidates := make([]model.SchedulerCandidate, 0, len(servers))
	for _, server := range servers {
		candidate, err := s.buildCandidate(ctx, server)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func (s *Service) buildCandidate(ctx context.Context, server model.Server) (model.SchedulerCandidate, error) {
	candidate := model.SchedulerCandidate{
		ServerID:        server.ID,
		ServerName:      server.Name,
		Status:          "offline",
		BestGPUIndex:    -1,
		BestFreeMemMB:   -1,
		BestUtilization: 1000,
	}

	heartbeats, err := s.heartbeats.ListByServerID(ctx, server.ID, 1)
	if err != nil {
		return candidate, err
	}
	if len(heartbeats) == 0 {
		candidate.IneligibleReason = "missing heartbeat"
		return candidate, nil
	}
	latestHeartbeat := heartbeats[0]
	candidate.HeartbeatAt = latestHeartbeat.HeartbeatAt
	candidate.Status = latestHeartbeat.Status
	if latestHeartbeat.Status != "online" {
		candidate.IneligibleReason = "server offline"
		return candidate, nil
	}

	queueLength, err := s.runs.CountActiveByServerID(ctx, server.ID)
	if err != nil {
		return candidate, err
	}
	candidate.QueueLength = queueLength

	snapshots, err := s.gpuSnapshots.ListByServerID(ctx, server.ID, 20)
	if err != nil {
		return candidate, err
	}
	latestSnapshots, capturedAt := latestSnapshotGroup(snapshots)
	if len(latestSnapshots) == 0 {
		candidate.IneligibleReason = "missing gpu snapshot"
		return candidate, nil
	}
	candidate.SnapshotCaptured = capturedAt
	for _, snapshot := range latestSnapshots {
		if snapshot.FreeMemMB > candidate.BestFreeMemMB || (snapshot.FreeMemMB == candidate.BestFreeMemMB && snapshot.Utilization < candidate.BestUtilization) {
			candidate.BestGPUIndex = snapshot.GPUIndex
			candidate.BestGPUName = snapshot.Name
			candidate.BestFreeMemMB = snapshot.FreeMemMB
			candidate.BestUtilization = snapshot.Utilization
		}
	}
	if candidate.BestFreeMemMB <= 0 {
		candidate.IneligibleReason = "no free gpu memory"
		return candidate, nil
	}
	candidate.Eligible = true
	return candidate, nil
}

func latestSnapshotGroup(items []model.GPUResourceSnapshot) ([]model.GPUResourceSnapshot, *time.Time) {
	if len(items) == 0 {
		return nil, nil
	}
	latest := items[0].CapturedAt
	group := make([]model.GPUResourceSnapshot, 0)
	for _, item := range items {
		if item.CapturedAt.Equal(latest) {
			group = append(group, item)
		}
	}
	return group, &latest
}
