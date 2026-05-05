package insightagent

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentpipeline"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/model"
	paperservice "mrag-platform/backend/go/internal/service"
)

type memoryJobStore struct {
	items map[string]model.AgentJob
}

func newMemoryJobStore() *memoryJobStore { return &memoryJobStore{items: map[string]model.AgentJob{}} }
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
func (s *memoryJobStore) ListByStatus(_ context.Context, status string, limit int) ([]model.AgentJob, error) {
	items := make([]model.AgentJob, 0)
	for _, item := range s.items {
		if item.Status == status {
			items = append(items, item)
		}
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items, nil
}
func (s *memoryJobStore) CountActiveByAgentType(_ context.Context, agentType string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.AgentType != agentType {
			continue
		}
		if item.Status == "running" || item.Status == "validating" || item.Status == "repairing" {
			count++
		}
	}
	return count, nil
}
func (s *memoryJobStore) FindByDedupKey(_ context.Context, dedupKey string) (*model.AgentJob, error) {
	for _, item := range s.items {
		if item.DedupKey == dedupKey {
			copyItem := item
			return &copyItem, nil
		}
	}
	return nil, nil
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

type memoryEventStore struct{ items map[string]model.AgentEvent }

func newMemoryEventStore() *memoryEventStore {
	return &memoryEventStore{items: map[string]model.AgentEvent{}}
}
func (s *memoryEventStore) Create(_ context.Context, item model.AgentEvent) error {
	s.items[item.ID] = item
	return nil
}
func (s *memoryEventStore) Update(_ context.Context, item model.AgentEvent) error {
	s.items[item.ID] = item
	return nil
}

type memorySubscriptionStore struct {
	items map[string]model.AgentSubscription
}

func newMemorySubscriptionStore() *memorySubscriptionStore {
	return &memorySubscriptionStore{items: map[string]model.AgentSubscription{}}
}
func (s *memorySubscriptionStore) Create(_ context.Context, item model.AgentSubscription) error {
	s.items[item.ID] = item
	return nil
}
func (s *memorySubscriptionStore) ListByEventType(_ context.Context, eventType string) ([]model.AgentSubscription, error) {
	items := make([]model.AgentSubscription, 0)
	for _, item := range s.items {
		if item.Enabled && item.EventType == eventType {
			items = append(items, item)
		}
	}
	return items, nil
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

func setupInsightServices(t *testing.T) (*Service, *agenttrigger.Service, *agentpipeline.Service, *paperservice.PaperService, *memoryPaperStore, string) {
	t.Helper()
	workspaceRoot := t.TempDir()
	jobStore := newMemoryJobStore()
	triggerStore := newMemoryTriggerStore()
	artifactStore := newMemoryArtifactStore()
	paperStore := newMemoryPaperStore()
	eventStore := newMemoryEventStore()
	subscriptionStore := newMemorySubscriptionStore()

	pythonExec := requirePython(t)
	pythonDir := pythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	paperSvc := paperservice.NewPaperService(paperStore, pythonExec, pythonDir, workspaceRoot)
	insightSvc := NewService(jobSvc, jobStore, triggerSvc, paperSvc, workspaceRoot)
	triggerSvc.RegisterPostProcessor("insight", insightSvc)
	pipelineSvc := agentpipeline.NewService(eventStore, subscriptionStore, jobStore, jobSvc, triggerSvc)
	paperSvc.SetEventPublisher(pipelineSvc)
	return insightSvc, triggerSvc, pipelineSvc, paperSvc, paperStore, workspaceRoot
}

func TestInsightServiceRunMockWritesInsight(t *testing.T) {
	insightSvc, _, _, paperSvc, _, workspaceRoot := setupInsightServices(t)
	importResult, err := paperSvc.ImportUploadedFile(context.Background(), "insight_source.md", bytes.NewBufferString("# Insight Source\n\nDeterministic content."))
	if err != nil {
		t.Fatalf("import paper failed: %v", err)
	}

	result, err := insightSvc.Run(context.Background(), model.InsightRunRequest{
		PaperID:          importResult.Paper.ID,
		ParsedContentRef: filepath.Join(workspaceRoot, "papers", "parsed", importResult.Paper.ID, "parsed.md"),
		ExecutionMode:    "mock",
	})
	if err != nil {
		t.Fatalf("run insight failed: %v", err)
	}
	if result.Job == nil || result.Job.ID == "" {
		t.Fatalf("expected insight job")
	}
	if result.Insight.PaperID != importResult.Paper.ID {
		t.Fatalf("unexpected paper id %s", result.Insight.PaperID)
	}
	if len(result.Insight.NoveltyPointsJSON.([]string)) == 0 {
		t.Fatalf("expected novelty points")
	}
	if _, err = os.Stat(result.SummaryPath); err != nil {
		t.Fatalf("expected summary path: %v", err)
	}
}

func TestInsightServiceRunCodexCLIFallsBackToMock(t *testing.T) {
	t.Setenv("CODEX_CLI_BIN", "definitely-missing-codex-cli")
	insightSvc, _, _, paperSvc, _, workspaceRoot := setupInsightServices(t)
	importResult, err := paperSvc.ImportUploadedFile(context.Background(), "insight_codex.md", bytes.NewBufferString("# Insight Codex\n\nDeterministic content."))
	if err != nil {
		t.Fatalf("import paper failed: %v", err)
	}

	result, err := insightSvc.Run(context.Background(), model.InsightRunRequest{
		PaperID:          importResult.Paper.ID,
		ParsedContentRef: filepath.Join(workspaceRoot, "papers", "parsed", importResult.Paper.ID, "parsed.md"),
		ExecutionMode:    "codex_cli",
	})
	if err != nil {
		t.Fatalf("run insight fallback failed: %v", err)
	}
	foundFallback := false
	for _, warning := range result.Warnings {
		if strings.Contains(strings.ToLower(warning), "falling back to mock executor") {
			foundFallback = true
			break
		}
	}
	if !foundFallback {
		t.Fatalf("expected fallback warning")
	}
}

func TestInsightAutoFlowFromPaperParsedEvent(t *testing.T) {
	insightSvc, _, pipelineSvc, paperSvc, _, workspaceRoot := setupInsightServices(t)
	_, err := pipelineSvc.CreateSubscription(context.Background(), model.AgentSubscriptionCreateRequest{
		Name:            "test-insight-subscription",
		EventType:       "paper_parsed",
		AgentType:       "insight",
		ExecutionMode:   "mock",
		ModelProvider:   "codex",
		ModelName:       "insight-default",
		PromptVersion:   "v1",
		OutputSchemaRef: "schemas/insight-output-v1.json",
		TriggerRule: map[string]any{
			"required_ref_types": []string{"parsed_content"},
		},
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	importResult, err := paperSvc.ImportUploadedFile(context.Background(), "auto_insight.md", bytes.NewBufferString("# Auto Insight\n\nDeterministic content."))
	if err != nil {
		t.Fatalf("import paper failed: %v", err)
	}
	dispatched, err := pipelineSvc.DispatchReadyJobs(context.Background(), 10)
	if err != nil {
		t.Fatalf("dispatch ready jobs failed: %v", err)
	}
	if dispatched == 0 {
		t.Fatalf("expected dispatched insight job")
	}

	insights, err := paperSvc.ListInsights(context.Background(), importResult.Paper.ID)
	if err != nil {
		t.Fatalf("list insights failed: %v", err)
	}
	if len(insights) == 0 {
		t.Fatalf("expected persisted insight after auto flow")
	}
	if _, err = os.Stat(filepath.Join(workspaceRoot, "papers", "insights", importResult.Paper.ID, "summary.md")); err != nil {
		t.Fatalf("expected auto insight summary artifact: %v", err)
	}
	_ = insightSvc
}
