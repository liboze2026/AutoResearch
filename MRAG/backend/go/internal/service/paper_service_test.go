package service

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type memoryPaperStore struct {
	papers   map[string]model.Paper
	files    map[string][]model.PaperFile
	insights map[string][]model.PaperInsight
}

func newMemoryPaperStore() *memoryPaperStore {
	return &memoryPaperStore{
		papers:   map[string]model.Paper{},
		files:    map[string][]model.PaperFile{},
		insights: map[string][]model.PaperInsight{},
	}
}

func (s *memoryPaperStore) List(_ context.Context) ([]model.Paper, error) {
	items := make([]model.Paper, 0, len(s.papers))
	for _, item := range s.papers {
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPaperStore) GetByID(_ context.Context, id string) (*model.Paper, error) {
	item, ok := s.papers[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPaperStore) ListFiles(_ context.Context, paperID string) ([]model.PaperFile, error) {
	items := s.files[paperID]
	out := make([]model.PaperFile, len(items))
	copy(out, items)
	return out, nil
}

func (s *memoryPaperStore) ListInsightsByPaper(_ context.Context, paperID string) ([]model.PaperInsight, error) {
	items := s.insights[paperID]
	out := make([]model.PaperInsight, len(items))
	copy(out, items)
	return out, nil
}

func (s *memoryPaperStore) Create(_ context.Context, paper model.Paper) error {
	s.papers[paper.ID] = paper
	return nil
}

func (s *memoryPaperStore) AddFile(_ context.Context, file model.PaperFile) error {
	s.files[file.PaperID] = append(s.files[file.PaperID], file)
	return nil
}

func (s *memoryPaperStore) UpdatePaperMetadata(_ context.Context, paper model.Paper) error {
	s.papers[paper.ID] = paper
	return nil
}

func (s *memoryPaperStore) UpsertInsight(_ context.Context, insight model.PaperInsight) error {
	s.insights[insight.PaperID] = []model.PaperInsight{insight}
	return nil
}

func requirePython(t *testing.T) string {
	t.Helper()
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available")
	}
	return python
}

func pythonAgentsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "python_agents"))
}

func TestPaperServiceImportSuccess(t *testing.T) {
	store := newMemoryPaperStore()
	workspaceRoot := t.TempDir()
	svc := NewPaperService(store, requirePython(t), pythonAgentsDir(t), workspaceRoot)

	result, err := svc.ImportUploadedFile(context.Background(), "my_first_paper.pdf", bytes.NewBufferString("# Demo Paper\nThis is a deterministic test paper."))
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if result.Paper.ID == "" {
		t.Fatalf("expected paper id")
	}
	if result.Paper.Status != "parsed" {
		t.Fatalf("expected parsed status, got %s", result.Paper.Status)
	}
	if !result.MockParsed {
		t.Fatalf("expected mock parsed")
	}
	files := store.files[result.Paper.ID]
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}

func TestPaperServiceQueryListSuccess(t *testing.T) {
	store := newMemoryPaperStore()
	store.papers["paper_1"] = model.Paper{ID: "paper_1", Title: "Demo", Status: "parsed", SourceType: "upload", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	svc := NewPaperService(store, requirePython(t), pythonAgentsDir(t), t.TempDir())

	items, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 paper, got %d", len(items))
	}
}

func TestPaperServiceQueryDetailSuccess(t *testing.T) {
	store := newMemoryPaperStore()
	now := time.Now()
	store.papers["paper_1"] = model.Paper{ID: "paper_1", Title: "Demo", Status: "parsed", SourceType: "upload", CreatedAt: now, UpdatedAt: now}
	store.files["paper_1"] = []model.PaperFile{{ID: "file_1", PaperID: "paper_1", FilePath: "workspace/papers/incoming/demo.pdf", FileType: "pdf", CreatedAt: now, UpdatedAt: now}}
	svc := NewPaperService(store, requirePython(t), pythonAgentsDir(t), t.TempDir())

	detail, err := svc.GetByID(context.Background(), "paper_1")
	if err != nil {
		t.Fatalf("detail failed: %v", err)
	}
	if detail == nil {
		t.Fatalf("expected paper detail")
	}
	if detail.Paper.ID != "paper_1" {
		t.Fatalf("unexpected paper id %s", detail.Paper.ID)
	}
	if len(detail.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(detail.Files))
	}
}

func TestPaperServiceExtractInsightsSuccess(t *testing.T) {
	store := newMemoryPaperStore()
	workspaceRoot := t.TempDir()
	svc := NewPaperService(store, requirePython(t), pythonAgentsDir(t), workspaceRoot)

	result, err := svc.ImportUploadedFile(context.Background(), "insight_demo.md", bytes.NewBufferString("# Insight Demo\nThis is a deterministic test paper."))
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	extracted, err := svc.ExtractInsights(context.Background(), result.Paper.ID)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if extracted.Insight.ExtractStatus != "completed" {
		t.Fatalf("expected completed extract status, got %s", extracted.Insight.ExtractStatus)
	}
	if len(store.insights[result.Paper.ID]) != 1 {
		t.Fatalf("expected one insight record")
	}
}
