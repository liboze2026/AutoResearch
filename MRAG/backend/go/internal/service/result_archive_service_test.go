package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type memoryResultArchiveStore struct {
	items map[string]model.ResultArchive
	files map[string][]model.ArchiveFile
}

func newMemoryResultArchiveStore() *memoryResultArchiveStore {
	return &memoryResultArchiveStore{items: map[string]model.ResultArchive{}, files: map[string][]model.ArchiveFile{}}
}

func (s *memoryResultArchiveStore) List(_ context.Context) ([]model.ResultArchive, error) {
	items := make([]model.ResultArchive, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryResultArchiveStore) GetByID(_ context.Context, id string) (*model.ResultArchive, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryResultArchiveStore) Create(_ context.Context, item model.ResultArchive) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryResultArchiveStore) Update(_ context.Context, item model.ResultArchive) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryResultArchiveStore) AddFile(_ context.Context, file model.ArchiveFile) error {
	s.files[file.ArchiveID] = append(s.files[file.ArchiveID], file)
	return nil
}

func (s *memoryResultArchiveStore) ListFiles(_ context.Context, archiveID string) ([]model.ArchiveFile, error) {
	items := s.files[archiveID]
	out := make([]model.ArchiveFile, len(items))
	copy(out, items)
	return out, nil
}

type memoryResultArchiveAssetReader struct {
	items map[string]model.DatasetAsset
}

func newMemoryResultArchiveAssetReader() *memoryResultArchiveAssetReader {
	return &memoryResultArchiveAssetReader{items: map[string]model.DatasetAsset{}}
}

func (s *memoryResultArchiveAssetReader) GetByID(_ context.Context, id string) (*model.DatasetAsset, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type memoryResultArchiveIdeaReader struct {
	items map[string]model.Idea
}

func newMemoryResultArchiveIdeaReader() *memoryResultArchiveIdeaReader {
	return &memoryResultArchiveIdeaReader{items: map[string]model.Idea{}}
}

func (s *memoryResultArchiveIdeaReader) GetByID(_ context.Context, id string) (*model.Idea, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func TestResultArchiveServiceCreateArchive(t *testing.T) {
	store := newMemoryResultArchiveStore()
	assetReader := newMemoryResultArchiveAssetReader()
	ideaReader := newMemoryResultArchiveIdeaReader()
	now := time.Now()
	assetReader.items["dasset_1"] = model.DatasetAsset{ID: "dasset_1", Name: "Demo Dataset", Status: "active", SourceType: "manual", LocalOrRemotePath: "/data/demo", CreatedAt: now, UpdatedAt: now}
	ideaReader.items["idea_1"] = model.Idea{ID: "idea_1", Title: "Archive Idea", Status: "draft", SourceType: "human", CreatedAt: now, UpdatedAt: now}
	svc := NewResultArchiveService(store, assetReader, ideaReader, t.TempDir())

	item, err := svc.Create(context.Background(), model.ResultArchiveCreateRequest{
		Title:          "Stage1 Archive",
		DatasetAssetID: "dasset_1",
		IdeaID:         "idea_1",
		SummaryMD:      "Archived experiment summary.",
		MetricJSON:     map[string]any{"accuracy": 0.88},
		Status:         "archived",
		NoteMD:         "Reference only.",
		Files:          []model.ArchiveFileInput{{FileName: "figure.txt", FileKind: "figure", Content: "figure placeholder"}},
	})
	if err != nil {
		t.Fatalf("create archive failed: %v", err)
	}
	if item.Archive.ID == "" {
		t.Fatalf("expected archive id")
	}
	archiveDir := filepath.Join(svc.workspaceRoot, "results", item.Archive.ID)
	if _, err := os.Stat(filepath.Join(archiveDir, "result.md")); err != nil {
		t.Fatalf("expected result.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(archiveDir, "metrics.json")); err != nil {
		t.Fatalf("expected metrics.json: %v", err)
	}
}

func TestResultArchiveServiceQueryArchive(t *testing.T) {
	store := newMemoryResultArchiveStore()
	now := time.Now()
	store.items["archive_1"] = model.ResultArchive{ID: "archive_1", Title: "Archive One", DatasetAssetID: "dasset_1", Status: "archived", CreatedAt: now, UpdatedAt: now}
	store.files["archive_1"] = []model.ArchiveFile{{ID: 1, ArchiveID: "archive_1", FilePath: "workspace/results/archive_1/result.md", FileKind: "result_md", CreatedAt: now, UpdatedAt: now}}
	svc := NewResultArchiveService(store, newMemoryResultArchiveAssetReader(), newMemoryResultArchiveIdeaReader(), t.TempDir())

	detail, err := svc.GetByID(context.Background(), "archive_1")
	if err != nil {
		t.Fatalf("get archive failed: %v", err)
	}
	if detail == nil || detail.Archive.ID != "archive_1" {
		t.Fatalf("expected archive detail")
	}
}

func TestResultArchiveServiceFilesWritten(t *testing.T) {
	store := newMemoryResultArchiveStore()
	assetReader := newMemoryResultArchiveAssetReader()
	now := time.Now()
	assetReader.items["dasset_2"] = model.DatasetAsset{ID: "dasset_2", Name: "Dataset Two", Status: "active", SourceType: "manual", LocalOrRemotePath: "/data/two", CreatedAt: now, UpdatedAt: now}
	svc := NewResultArchiveService(store, assetReader, newMemoryResultArchiveIdeaReader(), t.TempDir())

	item, err := svc.Create(context.Background(), model.ResultArchiveCreateRequest{Title: "Archive Two", DatasetAssetID: "dasset_2", SummaryMD: "Summary", MetricJSON: map[string]any{"f1": 0.73}, NoteMD: "note"})
	if err != nil {
		t.Fatalf("create archive failed: %v", err)
	}
	files := store.files[item.Archive.ID]
	if len(files) < 3 {
		t.Fatalf("expected at least 3 stored files, got %d", len(files))
	}
}
