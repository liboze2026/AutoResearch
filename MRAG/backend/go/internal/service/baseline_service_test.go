package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type memoryBaselineStore struct {
	items map[string]model.Baseline
}

func newMemoryBaselineStore() *memoryBaselineStore {
	return &memoryBaselineStore{items: map[string]model.Baseline{}}
}

func (s *memoryBaselineStore) List(_ context.Context) ([]model.Baseline, error) {
	items := make([]model.Baseline, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryBaselineStore) GetByID(_ context.Context, id string) (*model.Baseline, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryBaselineStore) Create(_ context.Context, item model.Baseline) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryBaselineStore) Update(_ context.Context, item model.Baseline) error {
	s.items[item.ID] = item
	return nil
}

type memoryBaselineAssetReader struct {
	assets map[string]model.DatasetAsset
}

func newMemoryBaselineAssetReader() *memoryBaselineAssetReader {
	return &memoryBaselineAssetReader{assets: map[string]model.DatasetAsset{}}
}

func (s *memoryBaselineAssetReader) GetByID(_ context.Context, id string) (*model.DatasetAsset, error) {
	item, ok := s.assets[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func TestBaselineServiceCreateBaseline(t *testing.T) {
	store := newMemoryBaselineStore()
	assetReader := newMemoryBaselineAssetReader()
	now := time.Now()
	assetReader.assets["dasset_1"] = model.DatasetAsset{ID: "dasset_1", Name: "Demo Dataset Asset", TaskType: "text", Status: "active", SourceType: "mrag_scan", LocalOrRemotePath: "/data/demo", CreatedAt: now, UpdatedAt: now}
	svc := NewBaselineService(store, assetReader, t.TempDir())

	item, err := svc.Create(context.Background(), model.BaselineCreateRequest{
		DatasetAssetID:   "dasset_1",
		Name:             "BM25 Baseline",
		MetricSchemaJSON: map[string]any{"primary": "accuracy", "higherIsBetter": true},
		ResultJSON:       map[string]any{"accuracy": 0.81, "latencyMs": 25},
		NoteMD:           "Manual baseline from prior report.",
		SourceType:       "manual",
	})
	if err != nil {
		t.Fatalf("create baseline failed: %v", err)
	}
	if item.Baseline.ID == "" {
		t.Fatalf("expected baseline id")
	}
	baselineDir := filepath.Join(svc.workspaceRoot, "datasets", "dasset_1", "baselines", item.Baseline.ID)
	if _, err := os.Stat(filepath.Join(baselineDir, "result.json")); err != nil {
		t.Fatalf("expected baseline result workspace file: %v", err)
	}
}

func TestBaselineServiceQueryBaseline(t *testing.T) {
	store := newMemoryBaselineStore()
	assetReader := newMemoryBaselineAssetReader()
	now := time.Now()
	store.items["baseline_1"] = model.Baseline{ID: "baseline_1", DatasetAssetID: "dasset_1", Name: "Dense Retriever", SourceType: "manual", CreatedAt: now, UpdatedAt: now}
	assetReader.assets["dasset_1"] = model.DatasetAsset{ID: "dasset_1", Name: "Demo Dataset Asset", TaskType: "text", Status: "active", SourceType: "mrag_scan", LocalOrRemotePath: "/data/demo", CreatedAt: now, UpdatedAt: now}
	svc := NewBaselineService(store, assetReader, t.TempDir())

	detail, err := svc.GetByID(context.Background(), "baseline_1")
	if err != nil {
		t.Fatalf("get baseline failed: %v", err)
	}
	if detail == nil || detail.Baseline.ID != "baseline_1" {
		t.Fatalf("expected baseline detail")
	}
}

func TestBaselineServiceDatasetAssetLinked(t *testing.T) {
	store := newMemoryBaselineStore()
	assetReader := newMemoryBaselineAssetReader()
	now := time.Now()
	assetReader.assets["dasset_2"] = model.DatasetAsset{ID: "dasset_2", Name: "Another Dataset Asset", TaskType: "text", Status: "active", SourceType: "manual", LocalOrRemotePath: "/data/another", CreatedAt: now, UpdatedAt: now}
	svc := NewBaselineService(store, assetReader, t.TempDir())

	item, err := svc.Create(context.Background(), model.BaselineCreateRequest{DatasetAssetID: "dasset_2", Name: "Reference Baseline", SourceType: "manual"})
	if err != nil {
		t.Fatalf("create baseline failed: %v", err)
	}
	if item.Baseline.DatasetAssetID != "dasset_2" {
		t.Fatalf("expected linked dataset asset id dasset_2, got %s", item.Baseline.DatasetAssetID)
	}
}
