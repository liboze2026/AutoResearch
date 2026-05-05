package ideaagent

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentpipeline"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/model"
	baseservice "mrag-platform/backend/go/internal/service"
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
		if item.AgentType == agentType && (item.Status == "running" || item.Status == "validating" || item.Status == "repairing") {
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

type memoryIdeaStore struct {
	ideas   map[string]model.Idea
	sources map[string][]model.IdeaSource
}

func newMemoryIdeaStore() *memoryIdeaStore {
	return &memoryIdeaStore{ideas: map[string]model.Idea{}, sources: map[string][]model.IdeaSource{}}
}
func (s *memoryIdeaStore) List(_ context.Context) ([]model.Idea, error) {
	items := make([]model.Idea, 0, len(s.ideas))
	for _, item := range s.ideas {
		items = append(items, item)
	}
	return items, nil
}
func (s *memoryIdeaStore) GetByID(_ context.Context, id string) (*model.Idea, error) {
	item, ok := s.ideas[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}
func (s *memoryIdeaStore) Create(_ context.Context, idea model.Idea) error {
	s.ideas[idea.ID] = idea
	return nil
}
func (s *memoryIdeaStore) Update(_ context.Context, idea model.Idea) error {
	s.ideas[idea.ID] = idea
	return nil
}
func (s *memoryIdeaStore) AddSource(_ context.Context, source model.IdeaSource) error {
	s.sources[source.IdeaID] = append(s.sources[source.IdeaID], source)
	return nil
}
func (s *memoryIdeaStore) ListSources(_ context.Context, ideaID string) ([]model.IdeaSource, error) {
	items := s.sources[ideaID]
	out := make([]model.IdeaSource, len(items))
	copy(out, items)
	return out, nil
}

type fakePaperReader struct {
	papers   map[string]model.Paper
	insights map[string][]model.PaperInsight
}

func (f *fakePaperReader) List(_ context.Context) ([]model.Paper, error) {
	items := make([]model.Paper, 0, len(f.papers))
	for _, item := range f.papers {
		items = append(items, item)
	}
	return items, nil
}
func (f *fakePaperReader) GetByID(_ context.Context, id string) (*model.PaperDetail, error) {
	item, ok := f.papers[id]
	if !ok {
		return nil, nil
	}
	return &model.PaperDetail{Paper: item, InsightList: f.insights[id]}, nil
}
func (f *fakePaperReader) ListInsights(_ context.Context, id string) ([]model.PaperInsight, error) {
	items := f.insights[id]
	out := make([]model.PaperInsight, len(items))
	copy(out, items)
	return out, nil
}

type fakeDatasetAssetReader struct {
	items map[string]model.DatasetAssetDetail
}

func (f *fakeDatasetAssetReader) List(_ context.Context) ([]model.DatasetAsset, error) {
	out := make([]model.DatasetAsset, 0, len(f.items))
	for _, item := range f.items {
		out = append(out, item.Asset)
	}
	return out, nil
}
func (f *fakeDatasetAssetReader) GetByID(_ context.Context, id string) (*model.DatasetAssetDetail, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
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
	out := make([]model.AgentSubscription, 0)
	for _, item := range s.items {
		if item.Enabled && item.EventType == eventType {
			out = append(out, item)
		}
	}
	return out, nil
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

func writeInsightFixture(t *testing.T, workspaceRoot string, paperID string, insightID string) string {
	t.Helper()
	dir := filepath.Join(workspaceRoot, "papers", "insights", paperID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir insight dir: %v", err)
	}
	path := filepath.Join(dir, "insight_agent_output.json")
	payload := map[string]any{
		"paper_id":           paperID,
		"paper_title":        "Controlled Retrieval Paper",
		"insight_id":         insightID,
		"summary_md":         "This paper improves retrieval reliability.",
		"contributions_json": []string{"Controlled retrieval orchestration."},
		"novelty_points":     []string{"Turns retrieval into a controlled pipeline."},
	}
	raw, _ := json.Marshal(payload)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write insight fixture: %v", err)
	}
	return path
}

func writeEvalPlanFixture(t *testing.T, workspaceRoot string, datasetAssetID string) string {
	t.Helper()
	dir := filepath.Join(workspaceRoot, "datasets", datasetAssetID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir dataset dir: %v", err)
	}
	path := filepath.Join(dir, "evalplan.json")
	payload := map[string]any{
		"dataset_asset_id": datasetAssetID,
		"eval_protocol_json": map[string]any{
			"metric_list": []string{"mrr", "ndcg@10"},
		},
		"split_strategy": "query_document_train_dev_test",
	}
	raw, _ := json.Marshal(payload)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write evalplan fixture: %v", err)
	}
	return path
}

func setupIdeaAgent(t *testing.T) (*Service, *baseservice.IdeaService, *agentpipeline.Service, *memoryIdeaStore, string) {
	t.Helper()
	workspaceRoot := t.TempDir()
	jobStore := newMemoryJobStore()
	triggerStore := newMemoryTriggerStore()
	artifactStore := newMemoryArtifactStore()
	ideaStore := newMemoryIdeaStore()
	eventStore := newMemoryEventStore()
	subscriptionStore := newMemorySubscriptionStore()

	insightPath := writeInsightFixture(t, workspaceRoot, "paper_1", "pinsight_1")
	evalplanPath := writeEvalPlanFixture(t, workspaceRoot, "dasset_1")

	paperReader := &fakePaperReader{
		papers: map[string]model.Paper{
			"paper_1": {ID: "paper_1", Title: "Controlled Retrieval Paper", Status: "insight_extracted", SourceType: "workspace", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		},
		insights: map[string][]model.PaperInsight{
			"paper_1": {{
				ID:                "pinsight_1",
				PaperID:           "paper_1",
				SummaryMD:         "This paper improves retrieval reliability.",
				ContributionsJSON: []string{"Controlled retrieval orchestration."},
				NoveltyPointsJSON: []string{"Turns retrieval into a controlled pipeline."},
				ExtractStatus:     "completed",
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			}},
		},
	}
	_ = insightPath
	datasetReader := &fakeDatasetAssetReader{
		items: map[string]model.DatasetAssetDetail{
			"dasset_1": {
				Asset: model.DatasetAsset{
					ID:                "dasset_1",
					Name:              "Retrieval Benchmark",
					TaskType:          "retrieval",
					Status:            "active",
					SourceType:        "mrag_scan",
					LocalOrRemotePath: "/data/retrieval",
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				},
			},
		},
	}
	_ = evalplanPath

	pythonExec := requirePython(t)
	pythonDir := pythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	ideaSvc := baseservice.NewIdeaService(ideaStore, nil, workspaceRoot)
	pipelineSvc := agentpipeline.NewService(eventStore, subscriptionStore, jobStore, jobSvc, triggerSvc)
	ideaSvc.SetEventPublisher(pipelineSvc)
	ideaAgentSvc := NewService(jobSvc, jobStore, triggerSvc, ideaSvc, paperReader, datasetReader, workspaceRoot)
	triggerSvc.RegisterPostProcessor("idea_generator", ideaAgentSvc)
	return ideaAgentSvc, ideaSvc, pipelineSvc, ideaStore, workspaceRoot
}

func TestIdeaAgentRunFromInsightAndDataset(t *testing.T) {
	ideaAgentSvc, _, _, _, workspaceRoot := setupIdeaAgent(t)

	result, err := ideaAgentSvc.Run(context.Background(), model.IdeaGeneratorRunRequest{
		PaperInsightRefs: []string{"pinsight_1"},
		DatasetAssetRefs: []string{"dasset_1"},
		HumanHints:       []string{"Focus on retrieval robustness."},
		ExecutionMode:    "mock",
	})
	if err != nil {
		t.Fatalf("idea agent run failed: %v", err)
	}
	if result.Idea == nil || result.Idea.StructuredIdea == nil {
		t.Fatalf("expected persisted structured idea")
	}
	if len(result.Idea.StructuredIdea.TargetDatasetRefs) != 1 || result.Idea.StructuredIdea.TargetDatasetRefs[0] != "dasset_1" {
		t.Fatalf("expected structured dataset refs, got %#v", result.Idea.StructuredIdea.TargetDatasetRefs)
	}
	if _, err = os.Stat(filepath.Join(workspaceRoot, "ideas", "pool", result.Idea.Idea.ID, "idea.md")); err != nil {
		t.Fatalf("expected idea workspace markdown: %v", err)
	}
}

func TestIdeaAgentManualIdeaStandardization(t *testing.T) {
	ideaAgentSvc, _, _, _, _ := setupIdeaAgent(t)

	result, err := ideaAgentSvc.Run(context.Background(), model.IdeaGeneratorRunRequest{
		HumanHints: []string{"Make it smaller and easier to validate."},
		ManualIdea: &model.StructuredIdeaPayload{
			Title:             "Human Seed Idea",
			DescriptionMD:     "Build a smaller retrieval controller.",
			ResearchDirection: "retrieval control",
			InnovationType:    "human_curated",
			ExpectedAdvantage: "Faster iteration.",
			RiskPoints:        []string{"May be too incremental."},
			Priority:          88,
			Confidence:        0.81,
		},
		ExecutionMode: "mock",
	})
	if err != nil {
		t.Fatalf("manual idea run failed: %v", err)
	}
	if result.Idea == nil || result.Idea.StructuredIdea == nil {
		t.Fatalf("expected structured idea")
	}
	if result.Idea.StructuredIdea.Title != "Human Seed Idea" {
		t.Fatalf("unexpected title %s", result.Idea.StructuredIdea.Title)
	}
	if result.Idea.Idea.SourceType != "human" {
		t.Fatalf("expected human source type, got %s", result.Idea.Idea.SourceType)
	}
}

func TestIdeaAgentAutoFlowFromInsightsReadyEvent(t *testing.T) {
	ideaAgentSvc, ideaSvc, pipelineSvc, _, workspaceRoot := setupIdeaAgent(t)
	_, err := pipelineSvc.CreateSubscription(context.Background(), model.AgentSubscriptionCreateRequest{
		Name:            "test-idea-subscription",
		EventType:       "insights_ready",
		AgentType:       "idea_generator",
		ExecutionMode:   "mock",
		ModelProvider:   "codex",
		ModelName:       "idea-generator-default",
		PromptVersion:   "v1",
		OutputSchemaRef: "schemas/idea-generator-output-v1.json",
		TriggerRule: map[string]any{
			"required_ref_types": []string{"insight"},
		},
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	insightPath := filepath.Join(workspaceRoot, "papers", "insights", "paper_1", "insight_agent_output.json")
	_, err = pipelineSvc.PublishEvent(context.Background(), model.AgentEventCreateRequest{
		EventType: "insights_ready",
		SourceRef: "paper:paper_1",
		InputRefs: []model.AgentInputRef{
			{RefType: "paper", RefID: "paper_1", Metadata: map[string]any{"paper_title": "Controlled Retrieval Paper"}},
			{RefType: "insight", RefID: "pinsight_1", RefPath: insightPath},
		},
		Payload: map[string]any{
			"paper_id":   "paper_1",
			"insight_id": "pinsight_1",
		},
	})
	if err != nil {
		t.Fatalf("publish event failed: %v", err)
	}
	dispatched, err := pipelineSvc.DispatchReadyJobs(context.Background(), 10)
	if err != nil {
		t.Fatalf("dispatch ready jobs failed: %v", err)
	}
	if dispatched == 0 {
		t.Fatalf("expected dispatched idea job")
	}
	items, err := ideaSvc.List(context.Background())
	if err != nil {
		t.Fatalf("list ideas failed: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected persisted idea after auto flow")
	}
	detail, err := ideaSvc.GetByID(context.Background(), items[0].ID)
	if err != nil {
		t.Fatalf("get idea failed: %v", err)
	}
	if detail == nil || detail.StructuredIdea == nil {
		t.Fatalf("expected structured idea detail")
	}
	if len(detail.StructuredIdea.TargetDatasetRefs) == 0 {
		t.Fatalf("expected auto flow to enrich dataset refs")
	}
	_ = ideaAgentSvc
}
