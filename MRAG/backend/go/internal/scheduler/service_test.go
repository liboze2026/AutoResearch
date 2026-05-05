package scheduler

import (
	"context"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type memExperimentReader struct {
	items map[string]model.Experiment
}

func (m *memExperimentReader) GetByID(_ context.Context, id string) (*model.Experiment, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (m *memExperimentReader) Update(_ context.Context, item model.Experiment) error {
	m.items[item.ID] = item
	return nil
}

type memSpecReader struct {
	items map[string]model.ExperimentSpec
}

func (m *memSpecReader) GetLatestByExperimentID(_ context.Context, experimentID string) (*model.ExperimentSpec, error) {
	item, ok := m.items[experimentID]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type memRunStore struct {
	items            map[string]model.ExperimentRun
	activeQueueCount map[string]int
	experimentCounts map[string]int
}

func (m *memRunStore) GetByID(_ context.Context, id string) (*model.ExperimentRun, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (m *memRunStore) Create(_ context.Context, item model.ExperimentRun) error {
	m.items[item.ID] = item
	m.experimentCounts[item.ExperimentID]++
	return nil
}

func (m *memRunStore) Update(_ context.Context, item model.ExperimentRun) error {
	m.items[item.ID] = item
	return nil
}

func (m *memRunStore) CountByExperimentID(_ context.Context, experimentID string) (int, error) {
	return m.experimentCounts[experimentID], nil
}

func (m *memRunStore) CountActiveByServerID(_ context.Context, serverID string) (int, error) {
	return m.activeQueueCount[serverID], nil
}

type memDecisionStore struct {
	items map[string]model.SchedulerDecision
}

func (m *memDecisionStore) Create(_ context.Context, item model.SchedulerDecision) error {
	m.items[item.RunID] = item
	return nil
}

func (m *memDecisionStore) GetLatestByRunID(_ context.Context, runID string) (*model.SchedulerDecision, error) {
	item, ok := m.items[runID]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type memServerLister struct {
	items []model.Server
}

func (m *memServerLister) List(context.Context) ([]model.Server, error) {
	return m.items, nil
}

type memHeartbeatReader struct {
	items map[string][]model.ServerHeartbeat
}

func (m *memHeartbeatReader) ListByServerID(_ context.Context, serverID string, _ int) ([]model.ServerHeartbeat, error) {
	return m.items[serverID], nil
}

type memGPUSnapshotReader struct {
	items map[string][]model.GPUResourceSnapshot
}

func (m *memGPUSnapshotReader) ListByServerID(_ context.Context, serverID string, _ int) ([]model.GPUResourceSnapshot, error) {
	return m.items[serverID], nil
}

func TestScheduleRunChoosesBestOnlineServer(t *testing.T) {
	now := time.Now()
	svc := NewService(
		&memExperimentReader{items: map[string]model.Experiment{"exp_1": {ID: "exp_1"}}},
		&memSpecReader{},
		&memRunStore{
			items: map[string]model.ExperimentRun{
				"run_1": {ID: "run_1", ExperimentID: "exp_1", RunStatus: "queued", ResultJSON: map[string]interface{}{}},
			},
			activeQueueCount: map[string]int{"srv_a": 1, "srv_b": 0},
			experimentCounts: map[string]int{},
		},
		&memDecisionStore{items: map[string]model.SchedulerDecision{}},
		&memServerLister{items: []model.Server{{ID: "srv_a", Name: "A"}, {ID: "srv_b", Name: "B"}}},
		&memHeartbeatReader{items: map[string][]model.ServerHeartbeat{
			"srv_a": {{ServerID: "srv_a", HeartbeatAt: now.Add(-1 * time.Minute), Status: "online"}},
			"srv_b": {{ServerID: "srv_b", HeartbeatAt: now, Status: "online"}},
		}},
		&memGPUSnapshotReader{items: map[string][]model.GPUResourceSnapshot{
			"srv_a": {{ServerID: "srv_a", CapturedAt: now, GPUIndex: 0, Name: "A100", FreeMemMB: 12000, Utilization: 20}},
			"srv_b": {{ServerID: "srv_b", CapturedAt: now, GPUIndex: 0, Name: "A100", FreeMemMB: 24000, Utilization: 5}},
		}},
		t.TempDir(),
	)

	result, err := svc.ScheduleRun(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("ScheduleRun returned error: %v", err)
	}
	if result.Chosen.ServerID != "srv_b" {
		t.Fatalf("expected srv_b, got %s", result.Chosen.ServerID)
	}
}

func TestScheduleRunSkipsOfflineServers(t *testing.T) {
	now := time.Now()
	svc := NewService(
		&memExperimentReader{items: map[string]model.Experiment{"exp_1": {ID: "exp_1"}}},
		&memSpecReader{},
		&memRunStore{
			items: map[string]model.ExperimentRun{
				"run_1": {ID: "run_1", ExperimentID: "exp_1", RunStatus: "queued", ResultJSON: map[string]interface{}{}},
			},
			activeQueueCount: map[string]int{},
			experimentCounts: map[string]int{},
		},
		&memDecisionStore{items: map[string]model.SchedulerDecision{}},
		&memServerLister{items: []model.Server{{ID: "srv_off", Name: "offline"}, {ID: "srv_on", Name: "online"}}},
		&memHeartbeatReader{items: map[string][]model.ServerHeartbeat{
			"srv_off": {{ServerID: "srv_off", HeartbeatAt: now, Status: "offline"}},
			"srv_on":  {{ServerID: "srv_on", HeartbeatAt: now, Status: "online"}},
		}},
		&memGPUSnapshotReader{items: map[string][]model.GPUResourceSnapshot{
			"srv_off": {{ServerID: "srv_off", CapturedAt: now, GPUIndex: 0, FreeMemMB: 64000, Utilization: 0}},
			"srv_on":  {{ServerID: "srv_on", CapturedAt: now, GPUIndex: 0, FreeMemMB: 8000, Utilization: 10}},
		}},
		t.TempDir(),
	)

	result, err := svc.ScheduleRun(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("ScheduleRun returned error: %v", err)
	}
	if result.Chosen.ServerID != "srv_on" {
		t.Fatalf("expected srv_on, got %s", result.Chosen.ServerID)
	}
}

func TestScheduleRunFailsWhenNoAvailableServer(t *testing.T) {
	now := time.Now()
	svc := NewService(
		&memExperimentReader{items: map[string]model.Experiment{"exp_1": {ID: "exp_1"}}},
		&memSpecReader{},
		&memRunStore{
			items: map[string]model.ExperimentRun{
				"run_1": {ID: "run_1", ExperimentID: "exp_1", RunStatus: "queued", ResultJSON: map[string]interface{}{}},
			},
			activeQueueCount: map[string]int{},
			experimentCounts: map[string]int{},
		},
		&memDecisionStore{items: map[string]model.SchedulerDecision{}},
		&memServerLister{items: []model.Server{{ID: "srv_off", Name: "offline"}}},
		&memHeartbeatReader{items: map[string][]model.ServerHeartbeat{
			"srv_off": {{ServerID: "srv_off", HeartbeatAt: now, Status: "offline"}},
		}},
		&memGPUSnapshotReader{items: map[string][]model.GPUResourceSnapshot{
			"srv_off": {{ServerID: "srv_off", CapturedAt: now, GPUIndex: 0, FreeMemMB: 0, Utilization: 99}},
		}},
		t.TempDir(),
	)

	_, err := svc.ScheduleRun(context.Background(), "run_1")
	if err == nil || err.Error() != "no available server for scheduling" {
		t.Fatalf("expected no available server error, got %v", err)
	}
}
