package recovery

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type runStore interface {
	GetByID(context.Context, string) (*model.ExperimentRun, error)
	Create(context.Context, model.ExperimentRun) error
	Update(context.Context, model.ExperimentRun) error
	CountByExperimentID(context.Context, string) (int, error)
}

type experimentStore interface {
	GetByID(context.Context, string) (*model.Experiment, error)
	Update(context.Context, model.Experiment) error
}

type logReader interface {
	ListByRunID(context.Context, string) ([]model.RunLog, error)
}

type Service struct {
	runs          runStore
	experiments   experimentStore
	logs          logReader
	workspaceRoot string
}

func NewService(runs runStore, experiments experimentStore, logs logReader, workspaceRoot string) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{runs: runs, experiments: experiments, logs: logs, workspaceRoot: workspaceRoot}
}

func (s *Service) Retry(ctx context.Context, runID string) (*model.ExperimentQueueResult, error) {
	run, err := s.runs.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("run not found")
	}
	if run.RunStatus != "failed" {
		return nil, fmt.Errorf("run is not retryable")
	}

	exp, err := s.experiments.GetByID(ctx, run.ExperimentID)
	if err != nil {
		return nil, err
	}
	if exp == nil {
		return nil, fmt.Errorf("experiment not found")
	}

	count, err := s.runs.CountByExperimentID(ctx, run.ExperimentID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	paths := workspacepkg.New(s.workspaceRoot)
	newRun := model.ExperimentRun{
		ID:            httpx.NewID("run"),
		ExperimentID:  run.ExperimentID,
		SpecID:        run.SpecID,
		RunStatus:     "queued",
		RemoteWorkdir: filepath.Join(paths.ExperimentDir(run.ExperimentID), fmt.Sprintf("run_%d", count+1)),
		RetryCount:    run.RetryCount + 1,
		ResultJSON: map[string]interface{}{
			"retry_of_run_id": run.ID,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.runs.Create(ctx, newRun); err != nil {
		return nil, err
	}

	exp.CurrentRunID = newRun.ID
	exp.Status = "queued"
	exp.UpdatedAt = now
	if err := s.experiments.Update(ctx, *exp); err != nil {
		return nil, err
	}
	return &model.ExperimentQueueResult{ExperimentID: exp.ID, Run: newRun}, nil
}

func (s *Service) GetRecovery(ctx context.Context, runID string) (*model.RunRecoveryDetail, error) {
	run, err := s.runs.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("run not found")
	}

	logs, err := s.logs.ListByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}

	stage := readString(run.ResultJSON, "failure_stage")
	summary := readString(run.ResultJSON, "last_log_summary")
	if summary == "" {
		for i := len(logs) - 1; i >= 0; i-- {
			if strings.TrimSpace(logs[i].TailText) != "" {
				summary = logs[i].TailText
				break
			}
		}
	}
	suggestRetry := readBool(run.ResultJSON, "recovery", "suggest_retry")
	if !suggestRetry {
		suggestRetry = inferSuggestRetry(stage, run.ErrorMessage)
	}

	return &model.RunRecoveryDetail{
		RunID:                  run.ID,
		ExperimentID:           run.ExperimentID,
		RunStatus:              run.RunStatus,
		FailureReason:          firstNonEmpty(run.ErrorMessage, "run did not record an explicit error"),
		FailureStage:           firstNonEmpty(stage, inferFailureStage(run.ErrorMessage)),
		LastLogSummary:         summary,
		SuggestRetry:           suggestRetry,
		RetryCount:             run.RetryCount,
		LatestAssignedServerID: run.AssignedServerID,
	}, nil
}

func inferFailureStage(errorMessage string) string {
	lower := strings.ToLower(errorMessage)
	switch {
	case strings.Contains(lower, "server"), strings.Contains(lower, "ssh"):
		return "prepare_remote"
	case strings.Contains(lower, "output"):
		return "collect_outputs"
	case strings.Contains(lower, "runner"):
		return "runner_exit"
	default:
		return "unknown"
	}
}

func inferSuggestRetry(stage string, errorMessage string) bool {
	switch stage {
	case "prepare_remote", "upload_files", "start_runner", "collect_outputs":
		return true
	case "runner_exit":
		return false
	}
	lower := strings.ToLower(errorMessage)
	return strings.Contains(lower, "unreachable") || strings.Contains(lower, "timeout")
}

func readString(root map[string]interface{}, key string) string {
	if root == nil {
		return ""
	}
	value, _ := root[key].(string)
	return value
}

func readBool(root map[string]interface{}, parent string, key string) bool {
	if root == nil {
		return false
	}
	nested, _ := root[parent].(map[string]interface{})
	if nested == nil {
		return false
	}
	value, _ := nested[key].(bool)
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
