package readeragent

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/agentartifact"
	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/model"
	paperservice "mrag-platform/backend/go/internal/service"
)

type memoryJobStore struct {
	items map[string]model.AgentJob
}

func newMemoryJobStore() *memoryJobStore {
	return &memoryJobStore{items: map[string]model.AgentJob{}}
}

func (s *memoryJobStore) Create(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryJobStore) GetByID(_ context.Context, id string) (*model.AgentJob, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryJobStore) Update(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

type memoryTriggerStore struct {
	items map[string]model.AgentJobTrigger
}

func newMemoryTriggerStore() *memoryTriggerStore {
	return &memoryTriggerStore{items: map[string]model.AgentJobTrigger{}}
}

func (s *memoryTriggerStore) Create(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryTriggerStore) Update(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}

type memoryArtifactStore struct {
	items map[string][]model.AgentArtifact
}

func newMemoryArtifactStore() *memoryArtifactStore {
	return &memoryArtifactStore{items: map[string][]model.AgentArtifact{}}
}

func (s *memoryArtifactStore) Create(_ context.Context, item model.AgentArtifact) error {
	s.items[item.JobID] = append(s.items[item.JobID], item)
	return nil
}

func (s *memoryArtifactStore) ListByJobID(_ context.Context, jobID string) ([]model.AgentArtifact, error) {
	items := s.items[jobID]
	out := make([]model.AgentArtifact, len(items))
	copy(out, items)
	return out, nil
}

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

func TestReaderServiceRunMockImportsPapers(t *testing.T) {
	workspaceRoot := t.TempDir()
	jobStore := newMemoryJobStore()
	triggerStore := newMemoryTriggerStore()
	artifactStore := newMemoryArtifactStore()
	paperStore := newMemoryPaperStore()

	pythonExec := requirePython(t)
	pythonDir := pythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	artifactSvc := agentartifact.NewService(artifactStore)
	paperSvc := paperservice.NewPaperService(paperStore, pythonExec, pythonDir, workspaceRoot)
	readerSvc := NewService(jobSvc, jobStore, triggerSvc, artifactSvc, paperSvc, workspaceRoot)

	result, err := readerSvc.Run(context.Background(), model.ReaderRunRequest{
		ResearchDirection: "multimodal retrieval",
		Keywords:          []string{"retrieval", "reasoning"},
		SourceScope:       "mixed",
		TimeRange:         map[string]any{"year": 2026},
		MaxPapers:         3,
		ExecutionMode:     "mock",
	})
	if err != nil {
		t.Fatalf("reader run failed: %v", err)
	}
	if result.Job == nil || result.Job.ID == "" {
		t.Fatalf("expected reader job")
	}
	if len(result.CandidatePapers) != 3 {
		t.Fatalf("expected 3 candidate papers, got %d", len(result.CandidatePapers))
	}
	if len(result.ImportedPapers) != 3 {
		t.Fatalf("expected 3 imported papers, got %d", len(result.ImportedPapers))
	}
	for _, item := range result.ImportedPapers {
		if item.Result.Paper.SourceURL == "" {
			t.Fatalf("expected imported paper source url to be set")
		}
		if item.Result.Paper.Status != "parsed" {
			t.Fatalf("expected parsed paper status, got %s", item.Result.Paper.Status)
		}
	}
}

func TestReaderServiceRunCodexCLIFallsBackToMock(t *testing.T) {
	t.Setenv("CODEX_CLI_BIN", "definitely-missing-codex-cli")

	workspaceRoot := t.TempDir()
	jobStore := newMemoryJobStore()
	triggerStore := newMemoryTriggerStore()
	artifactStore := newMemoryArtifactStore()
	paperStore := newMemoryPaperStore()

	pythonExec := requirePython(t)
	pythonDir := pythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	artifactSvc := agentartifact.NewService(artifactStore)
	paperSvc := paperservice.NewPaperService(paperStore, pythonExec, pythonDir, workspaceRoot)
	readerSvc := NewService(jobSvc, jobStore, triggerSvc, artifactSvc, paperSvc, workspaceRoot)

	result, err := readerSvc.Run(context.Background(), model.ReaderRunRequest{
		ResearchDirection: "vision language",
		Keywords:          []string{"vlm"},
		SourceScope:       "arxiv",
		TimeRange:         map[string]any{"year": 2026},
		MaxPapers:         2,
		ExecutionMode:     "codex_cli",
	})
	if err != nil {
		t.Fatalf("reader codex_cli fallback failed: %v", err)
	}
	if result.Job == nil {
		t.Fatalf("expected reader job")
	}
	if len(result.ImportedPapers) != 2 {
		t.Fatalf("expected 2 imported papers, got %d", len(result.ImportedPapers))
	}
	foundFallbackWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(strings.ToLower(warning), "falling back to mock executor") {
			foundFallbackWarning = true
			break
		}
	}
	if !foundFallbackWarning {
		t.Fatalf("expected codex_cli fallback warning")
	}
}

func TestReaderServiceGetJobReturnsArtifactsAndImportedPapers(t *testing.T) {
	workspaceRoot := t.TempDir()
	jobStore := newMemoryJobStore()
	triggerStore := newMemoryTriggerStore()
	artifactStore := newMemoryArtifactStore()
	paperStore := newMemoryPaperStore()

	pythonExec := requirePython(t)
	pythonDir := pythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	artifactSvc := agentartifact.NewService(artifactStore)
	paperSvc := paperservice.NewPaperService(paperStore, pythonExec, pythonDir, workspaceRoot)
	readerSvc := NewService(jobSvc, jobStore, triggerSvc, artifactSvc, paperSvc, workspaceRoot)

	runResult, err := readerSvc.Run(context.Background(), model.ReaderRunRequest{
		ResearchDirection: "efficient reasoning",
		Keywords:          []string{"reasoning"},
		SourceScope:       "journal",
		TimeRange:         map[string]any{"year": 2025},
		MaxPapers:         1,
		ExecutionMode:     "mock",
	})
	if err != nil {
		t.Fatalf("reader run failed: %v", err)
	}

	detail, err := readerSvc.GetJob(context.Background(), runResult.Job.ID)
	if err != nil {
		t.Fatalf("get job failed: %v", err)
	}
	if detail == nil || detail.Job == nil {
		t.Fatalf("expected reader job detail")
	}
	if len(detail.Artifacts) == 0 {
		t.Fatalf("expected reader artifacts")
	}
	if len(detail.ImportedPapers) != 1 {
		t.Fatalf("expected 1 imported paper, got %d", len(detail.ImportedPapers))
	}
	if detail.Job.UpdatedAt.Before(time.Now().Add(-time.Minute)) {
		t.Fatalf("expected fresh updated time")
	}
}
