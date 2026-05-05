package resultcompare

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type memRunStore struct {
	items map[string]model.ExperimentRun
}

func (m *memRunStore) GetByID(_ context.Context, id string) (*model.ExperimentRun, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (m *memRunStore) Update(_ context.Context, item model.ExperimentRun) error {
	m.items[item.ID] = item
	return nil
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

type memBaselineReader struct {
	items map[string]model.Baseline
}

func (m *memBaselineReader) GetByID(_ context.Context, id string) (*model.Baseline, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type memArchiveReader struct {
	items map[string]model.ResultArchive
}

func (m *memArchiveReader) GetByID(_ context.Context, id string) (*model.ResultArchive, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (m *memArchiveReader) ListByDatasetAssetID(_ context.Context, datasetAssetID string) ([]model.ResultArchive, error) {
	out := make([]model.ResultArchive, 0)
	for _, item := range m.items {
		if item.DatasetAssetID == datasetAssetID {
			out = append(out, item)
		}
	}
	return out, nil
}

type memComparisonStore struct {
	items []model.ResultComparison
}

func (m *memComparisonStore) ListByExperimentID(_ context.Context, experimentID string) ([]model.ResultComparison, error) {
	out := make([]model.ResultComparison, 0)
	for _, item := range m.items {
		if item.ExperimentID == experimentID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (m *memComparisonStore) Create(_ context.Context, item model.ResultComparison) error {
	m.items = append(m.items, item)
	return nil
}

type memArchiveWriter struct {
	created []*model.ResultArchiveDetail
}

func (m *memArchiveWriter) Create(_ context.Context, req model.ResultArchiveCreateRequest) (*model.ResultArchiveDetail, error) {
	now := time.Now()
	item := &model.ResultArchiveDetail{
		Archive: model.ResultArchive{
			ID:             "archive_auto_1",
			Title:          req.Title,
			DatasetAssetID: req.DatasetAssetID,
			BaselineID:     req.BaselineID,
			IdeaID:         req.IdeaID,
			ServerID:       req.ServerID,
			SummaryMD:      req.SummaryMD,
			MetricJSON:     req.MetricJSON,
			Status:         req.Status,
			NoteMD:         req.NoteMD,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}
	m.created = append(m.created, item)
	return item, nil
}

func TestCompareRunWithBaselineCreatesComparisons(t *testing.T) {
	workspace := t.TempDir()
	now := time.Now()
	runStore := &memRunStore{items: map[string]model.ExperimentRun{
		"run_1": {
			ID:           "run_1",
			ExperimentID: "exp_1",
			RunStatus:    "succeeded",
			ResultJSON: map[string]interface{}{
				"metrics": map[string]interface{}{
					"primary_metric": "accuracy",
					"values":         map[string]interface{}{"accuracy": 0.90, "loss": 0.20},
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}}
	comparisonStore := &memComparisonStore{}
	archiveWriter := &memArchiveWriter{}
	svc := NewService(
		runStore,
		&memExperimentStore{items: map[string]model.Experiment{
			"exp_1": {ID: "exp_1", DatasetAssetID: "dasset_1", BaselineID: "base_1", Title: "Demo Experiment"},
		}},
		&memBaselineReader{items: map[string]model.Baseline{
			"base_1": {ID: "base_1", Name: "BM25", ResultJSON: map[string]any{"accuracy": 0.75, "loss": 0.35}},
		}},
		&memArchiveReader{items: map[string]model.ResultArchive{
			"arch_old": {ID: "arch_old", Title: "Historic Run", DatasetAssetID: "dasset_1", MetricJSON: map[string]any{"accuracy": 0.82, "loss": 0.29}},
		}},
		comparisonStore,
		archiveWriter,
		workspace,
	)

	result, err := svc.CompareRun(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("CompareRun returned error: %v", err)
	}
	if len(result.Comparisons) != 2 {
		t.Fatalf("expected 2 comparisons, got %d", len(result.Comparisons))
	}
	if result.ResultArchive == nil {
		t.Fatalf("expected auto result archive")
	}
	if result.OverallJudgment == "" {
		t.Fatalf("expected overall judgment")
	}
	files, err := os.ReadDir(filepath.Join(workspace, "experiments", "exp_1", "comparisons"))
	if err != nil {
		t.Fatalf("ReadDir returned error: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("expected comparison workspace files")
	}
	if len(comparisonStore.items) != 2 {
		t.Fatalf("expected persisted comparisons, got %d", len(comparisonStore.items))
	}
}

func TestCompareRunWithoutBaselineStillReturnsResult(t *testing.T) {
	now := time.Now()
	runStore := &memRunStore{items: map[string]model.ExperimentRun{
		"run_1": {
			ID:           "run_1",
			ExperimentID: "exp_1",
			RunStatus:    "succeeded",
			ResultJSON: map[string]interface{}{
				"metrics": map[string]interface{}{
					"values": map[string]interface{}{"accuracy": 0.88},
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}}
	comparisonStore := &memComparisonStore{}
	svc := NewService(
		runStore,
		&memExperimentStore{items: map[string]model.Experiment{
			"exp_1": {ID: "exp_1", DatasetAssetID: "dasset_1", Title: "No Baseline Experiment"},
		}},
		&memBaselineReader{items: map[string]model.Baseline{}},
		&memArchiveReader{items: map[string]model.ResultArchive{}},
		comparisonStore,
		&memArchiveWriter{},
		t.TempDir(),
	)

	result, err := svc.CompareRun(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("CompareRun returned error: %v", err)
	}
	if len(result.Comparisons) != 0 {
		t.Fatalf("expected 0 comparisons, got %d", len(result.Comparisons))
	}
	if result.OverallJudgment != "无可用对比对象" {
		t.Fatalf("unexpected overall judgment: %s", result.OverallJudgment)
	}
	if len(comparisonStore.items) != 0 {
		t.Fatalf("expected no persisted comparisons, got %d", len(comparisonStore.items))
	}
}
