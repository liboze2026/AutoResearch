package recovery

import (
	"context"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type memRunStore struct {
	items  map[string]model.ExperimentRun
	counts map[string]int
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
	m.counts[item.ExperimentID]++
	return nil
}

func (m *memRunStore) Update(_ context.Context, item model.ExperimentRun) error {
	m.items[item.ID] = item
	return nil
}

func (m *memRunStore) CountByExperimentID(_ context.Context, experimentID string) (int, error) {
	return m.counts[experimentID], nil
}

type memExperimentStore struct {
	items map[string]model.Experiment
}

func (m *memExperimentStore) GetByID(_ context.Context, id string) (*model.Experiment, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (m *memExperimentStore) Update(_ context.Context, item model.Experiment) error {
	m.items[item.ID] = item
	return nil
}

type memLogReader struct {
	items []model.RunLog
}

func (m *memLogReader) ListByRunID(_ context.Context, runID string) ([]model.RunLog, error) {
	out := make([]model.RunLog, 0)
	for _, item := range m.items {
		if item.RunID == runID {
			out = append(out, item)
		}
	}
	return out, nil
}

func TestRetryCreatesQueuedRun(t *testing.T) {
	now := time.Now()
	runStore := &memRunStore{
		items: map[string]model.ExperimentRun{
			"run_1": {
				ID:           "run_1",
				ExperimentID: "exp_1",
				SpecID:       "spec_1",
				RunStatus:    "failed",
				RetryCount:   0,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
		counts: map[string]int{"exp_1": 1},
	}
	expStore := &memExperimentStore{
		items: map[string]model.Experiment{
			"exp_1": {ID: "exp_1", Status: "failed", CreatedAt: now, UpdatedAt: now},
		},
	}
	svc := NewService(runStore, expStore, &memLogReader{}, t.TempDir())

	result, err := svc.Retry(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("Retry returned error: %v", err)
	}
	if result.Run.RunStatus != "queued" {
		t.Fatalf("expected queued, got %s", result.Run.RunStatus)
	}
	if result.Run.RetryCount != 1 {
		t.Fatalf("expected retry_count=1, got %d", result.Run.RetryCount)
	}
}

func TestGetRecoveryReturnsFailureInfo(t *testing.T) {
	now := time.Now()
	svc := NewService(
		&memRunStore{
			items: map[string]model.ExperimentRun{
				"run_1": {
					ID:               "run_1",
					ExperimentID:     "exp_1",
					AssignedServerID: "srv_1",
					RunStatus:        "failed",
					RetryCount:       1,
					ErrorMessage:     "required output files are missing",
					ResultJSON: map[string]interface{}{
						"failure_stage":    "collect_outputs",
						"last_log_summary": "stdout tail",
						"recovery":         map[string]interface{}{"suggest_retry": true},
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			counts: map[string]int{},
		},
		&memExperimentStore{items: map[string]model.Experiment{"exp_1": {ID: "exp_1"}}},
		&memLogReader{items: []model.RunLog{{RunID: "run_1", LogType: "stdout", TailText: "stdout tail"}}},
		t.TempDir(),
	)

	info, err := svc.GetRecovery(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("GetRecovery returned error: %v", err)
	}
	if info.FailureStage != "collect_outputs" {
		t.Fatalf("expected collect_outputs, got %s", info.FailureStage)
	}
	if !info.SuggestRetry {
		t.Fatalf("expected suggest retry")
	}
}
