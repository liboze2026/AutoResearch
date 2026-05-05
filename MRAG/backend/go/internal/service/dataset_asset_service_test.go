package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type memoryDatasetAssetStore struct {
	assets          map[string]model.DatasetAsset
	sources         map[string][]model.DatasetAssetSource
	byDatasetSource map[string]string
}

func newMemoryDatasetAssetStore() *memoryDatasetAssetStore {
	return &memoryDatasetAssetStore{
		assets:          map[string]model.DatasetAsset{},
		sources:         map[string][]model.DatasetAssetSource{},
		byDatasetSource: map[string]string{},
	}
}

func (s *memoryDatasetAssetStore) List(_ context.Context) ([]model.DatasetAsset, error) {
	items := make([]model.DatasetAsset, 0, len(s.assets))
	for _, item := range s.assets {
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryDatasetAssetStore) GetByID(_ context.Context, id string) (*model.DatasetAsset, error) {
	item, ok := s.assets[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryDatasetAssetStore) GetByExistingDatasetRef(_ context.Context, datasetRef string) (*model.DatasetAsset, error) {
	assetID, ok := s.byDatasetSource[datasetRef]
	if !ok {
		return nil, nil
	}
	item := s.assets[assetID]
	copyItem := item
	return &copyItem, nil
}

func (s *memoryDatasetAssetStore) Create(_ context.Context, asset model.DatasetAsset) error {
	s.assets[asset.ID] = asset
	return nil
}

func (s *memoryDatasetAssetStore) Update(_ context.Context, asset model.DatasetAsset) error {
	s.assets[asset.ID] = asset
	return nil
}

func (s *memoryDatasetAssetStore) AddSource(_ context.Context, source model.DatasetAssetSource) error {
	s.sources[source.DatasetAssetID] = append(s.sources[source.DatasetAssetID], source)
	s.byDatasetSource[source.ExistingDatasetRef] = source.DatasetAssetID
	return nil
}

func (s *memoryDatasetAssetStore) ListSources(_ context.Context, datasetAssetID string) ([]model.DatasetAssetSource, error) {
	items := s.sources[datasetAssetID]
	out := make([]model.DatasetAssetSource, len(items))
	copy(out, items)
	return out, nil
}

type memoryDatasetScanReader struct {
	datasets map[string]model.Dataset
	scans    map[string]model.DatasetScanRecord
}

func newMemoryDatasetScanReader() *memoryDatasetScanReader {
	return &memoryDatasetScanReader{datasets: map[string]model.Dataset{}, scans: map[string]model.DatasetScanRecord{}}
}

func (s *memoryDatasetScanReader) GetSummaryByID(_ context.Context, id string) (*model.Dataset, error) {
	item, ok := s.datasets[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryDatasetScanReader) GetScanRecordByID(_ context.Context, id string) (*model.DatasetScanRecord, error) {
	item, ok := s.scans[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func TestDatasetAssetServiceRegisterFromScan(t *testing.T) {
	store := newMemoryDatasetAssetStore()
	scanReader := newMemoryDatasetScanReader()
	now := time.Now()
	scanReader.datasets["ds_1"] = model.Dataset{ID: "ds_1", Name: "Scanned OCR Dataset", SourceType: "remote", Modality: "text", Description: "Scanned by MRAG", Path: "/remote/data/ocr", ServerName: "gpu-box", FileCount: 120, DirectoryCount: 4, FileTypes: map[string]int64{"jsonl": 120}, DetectedModality: "text", UpdatedAt: now}
	scanReader.scans["scan_1"] = model.DatasetScanRecord{ID: "scan_1", DatasetID: "ds_1", ScanStatus: "completed", RootPath: "/remote/data/ocr", FileCount: 120, DirectoryCount: 4, FileTypes: map[string]int64{"jsonl": 120}, HierarchySummary: []model.DatasetHierarchySummaryItem{{Level: 1, Path: "train", ItemCount: 100}}, ScannedAt: now}
	svc := NewDatasetAssetService(store, scanReader, t.TempDir())

	item, err := svc.RegisterFromScan(context.Background(), model.DatasetAssetRegisterFromScanRequest{ScanRecordID: "scan_1"})
	if err != nil {
		t.Fatalf("register from scan failed: %v", err)
	}
	if item.Asset.ExistingDatasetRef != "ds_1" {
		t.Fatalf("expected linked dataset ref ds_1, got %s", item.Asset.ExistingDatasetRef)
	}
	assetDir := filepath.Join(svc.workspaceRoot, "datasets", item.Asset.ID)
	if _, err := os.Stat(filepath.Join(assetDir, "README.md")); err != nil {
		t.Fatalf("expected dataset asset readme: %v", err)
	}
}

func TestDatasetAssetServiceManualCreate(t *testing.T) {
	store := newMemoryDatasetAssetStore()
	svc := NewDatasetAssetService(store, newMemoryDatasetScanReader(), t.TempDir())

	item, err := svc.Create(context.Background(), model.DatasetAssetCreateRequest{
		Name:              "Manual Benchmark Asset",
		DescriptionMD:     "Manually curated benchmark asset.",
		TaskType:          "text",
		Status:            "active",
		SourceType:        "manual",
		LocalOrRemotePath: "D:/datasets/manual-benchmark",
		ReadmeMD:          "# Manual Benchmark Asset",
		LoaderNoteMD:      "Use pandas to read the csv.",
		SchemaNoteMD:      "Columns: prompt, answer",
	})
	if err != nil {
		t.Fatalf("manual create failed: %v", err)
	}
	if item.Asset.ID == "" {
		t.Fatalf("expected asset id")
	}
	assetDir := filepath.Join(svc.workspaceRoot, "datasets", item.Asset.ID)
	if _, err := os.Stat(filepath.Join(assetDir, "metadata.json")); err != nil {
		t.Fatalf("expected dataset asset metadata: %v", err)
	}
}

func TestDatasetAssetServiceQueryDetail(t *testing.T) {
	store := newMemoryDatasetAssetStore()
	now := time.Now()
	store.assets["dasset_1"] = model.DatasetAsset{ID: "dasset_1", Name: "Detail Asset", TaskType: "text", Status: "active", SourceType: "manual", LocalOrRemotePath: "D:/datasets/detail", CreatedAt: now, UpdatedAt: now}
	store.sources["dasset_1"] = []model.DatasetAssetSource{{ID: 1, DatasetAssetID: "dasset_1", ExistingDatasetRef: "ds_1", SourceKind: "dataset", ExistingDatasetName: "Scanned Dataset", CreatedAt: now, UpdatedAt: now}}
	svc := NewDatasetAssetService(store, newMemoryDatasetScanReader(), t.TempDir())

	detail, err := svc.GetByID(context.Background(), "dasset_1")
	if err != nil {
		t.Fatalf("detail failed: %v", err)
	}
	if detail == nil {
		t.Fatalf("expected detail")
	}
	if len(detail.Sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(detail.Sources))
	}
}
