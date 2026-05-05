package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type memoryExperimentStore struct {
	items map[string]model.Experiment
}

func newMemoryExperimentStore() *memoryExperimentStore {
	return &memoryExperimentStore{items: map[string]model.Experiment{}}
}

func (s *memoryExperimentStore) List(_ context.Context) ([]model.Experiment, error) {
	items := make([]model.Experiment, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryExperimentStore) GetByID(_ context.Context, id string) (*model.Experiment, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryExperimentStore) Create(_ context.Context, item model.Experiment) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryExperimentStore) Update(_ context.Context, item model.Experiment) error {
	s.items[item.ID] = item
	return nil
}

type memoryExperimentSpecStore struct {
	items map[string][]model.ExperimentSpec
}

func newMemoryExperimentSpecStore() *memoryExperimentSpecStore {
	return &memoryExperimentSpecStore{items: map[string][]model.ExperimentSpec{}}
}

func (s *memoryExperimentSpecStore) ListByExperimentID(_ context.Context, experimentID string) ([]model.ExperimentSpec, error) {
	items := append([]model.ExperimentSpec(nil), s.items[experimentID]...)
	return items, nil
}

func (s *memoryExperimentSpecStore) GetLatestByExperimentID(_ context.Context, experimentID string) (*model.ExperimentSpec, error) {
	items := s.items[experimentID]
	if len(items) == 0 {
		return nil, nil
	}
	item := items[len(items)-1]
	return &item, nil
}

func (s *memoryExperimentSpecStore) Create(_ context.Context, item model.ExperimentSpec) error {
	s.items[item.ExperimentID] = append(s.items[item.ExperimentID], item)
	return nil
}

type memoryExperimentAssetReader struct {
	items map[string]model.DatasetAsset
}

func (s *memoryExperimentAssetReader) GetByID(_ context.Context, id string) (*model.DatasetAsset, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type memoryExperimentIdeaReader struct {
	items map[string]model.Idea
}

func (s *memoryExperimentIdeaReader) GetByID(_ context.Context, id string) (*model.Idea, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type memoryExperimentBaselineReader struct {
	items map[string]model.Baseline
}

func (s *memoryExperimentBaselineReader) GetByID(_ context.Context, id string) (*model.Baseline, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type memoryExperimentArchiveReader struct {
	items []model.ResultArchive
}

func (s *memoryExperimentArchiveReader) ListByDatasetAssetID(_ context.Context, datasetAssetID string) ([]model.ResultArchive, error) {
	items := make([]model.ResultArchive, 0)
	for _, item := range s.items {
		if item.DatasetAssetID == datasetAssetID {
			items = append(items, item)
		}
	}
	return items, nil
}

func TestExperimentServiceCreateGenerateAndQuerySpec(t *testing.T) {
	now := time.Now()
	expStore := newMemoryExperimentStore()
	specStore := newMemoryExperimentSpecStore()
	assetReader := &memoryExperimentAssetReader{items: map[string]model.DatasetAsset{
		"dasset_1": {ID: "dasset_1", Name: "Demo Dataset", TaskType: "text", SourceType: "mrag_scan", LocalOrRemotePath: "/data/demo", ExistingDatasetRef: "dataset_demo", LoaderNoteMD: "Use stage1 dataset loader note.", CreatedAt: now, UpdatedAt: now},
	}}
	ideaReader := &memoryExperimentIdeaReader{items: map[string]model.Idea{
		"idea_1": {ID: "idea_1", Title: "LoRA tuning", CreatedAt: now, UpdatedAt: now},
	}}
	baselineReader := &memoryExperimentBaselineReader{items: map[string]model.Baseline{
		"baseline_1": {ID: "baseline_1", DatasetAssetID: "dasset_1", Name: "BM25 Baseline", MetricSchemaJSON: map[string]any{"primary": "accuracy"}, CreatedAt: now, UpdatedAt: now},
	}}
	archiveReader := &memoryExperimentArchiveReader{items: []model.ResultArchive{
		{ID: "rarch_1", Title: "Historic Run", DatasetAssetID: "dasset_1", Status: "archived", CreatedAt: now, UpdatedAt: now},
	}}

	workspaceRoot := t.TempDir()
	svc := NewExperimentService(expStore, specStore, assetReader, ideaReader, baselineReader, archiveReader, workspaceRoot)

	created, err := svc.Create(context.Background(), model.ExperimentCreateRequest{
		DatasetAssetID: "dasset_1",
		IdeaID:         "idea_1",
		BaselineID:     "baseline_1",
		Title:          "Demo Experiment",
		Priority:       5,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	specDetail, err := svc.GenerateSpec(context.Background(), created.Experiment.ID)
	if err != nil {
		t.Fatalf("GenerateSpec returned error: %v", err)
	}

	gotSpec, err := svc.GetLatestSpec(context.Background(), created.Experiment.ID)
	if err != nil {
		t.Fatalf("GetLatestSpec returned error: %v", err)
	}
	if gotSpec.Spec.Version != 1 {
		t.Fatalf("expected spec version 1, got %d", gotSpec.Spec.Version)
	}

	specPath := filepath.Join(workspaceRoot, "experiments", created.Experiment.ID, "spec.json")
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("ReadFile spec.json returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal spec.json: %v", err)
	}
	if payload["model_name"] == "" {
		t.Fatalf("expected model_name in spec payload")
	}
	targets, ok := payload["comparison_targets"].([]any)
	if !ok || len(targets) != 2 {
		t.Fatalf("expected 2 comparison targets, got %#v", payload["comparison_targets"])
	}
	if specDetail.Spec.TemplateType != "lora_sft_v1" {
		t.Fatalf("expected lora_sft_v1 template, got %s", specDetail.Spec.TemplateType)
	}
}
