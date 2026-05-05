package ideaagent

import (
	"context"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/agentartifact"
	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/model"
	baseservice "mrag-platform/backend/go/internal/service"
)

type memoryPhase4IdeaStore struct {
	datasetProfiles map[string]model.Phase4DatasetProfile
	readerSources   map[string]model.Phase4ReaderSource
	readerContexts  map[string]model.Phase4ReaderContext
	ideas           map[string]model.Phase4Idea
	runs            map[string]model.Phase4RunManifest
	reports         map[string]model.Phase4StructuredReportRecord
}

func newMemoryPhase4IdeaStore() *memoryPhase4IdeaStore {
	return &memoryPhase4IdeaStore{
		datasetProfiles: map[string]model.Phase4DatasetProfile{},
		readerSources:   map[string]model.Phase4ReaderSource{},
		readerContexts:  map[string]model.Phase4ReaderContext{},
		ideas:           map[string]model.Phase4Idea{},
		runs:            map[string]model.Phase4RunManifest{},
		reports:         map[string]model.Phase4StructuredReportRecord{},
	}
}

func (s *memoryPhase4IdeaStore) ListDatasetProfiles(_ context.Context, taskType string, status string) ([]model.Phase4DatasetProfile, error) {
	items := make([]model.Phase4DatasetProfile, 0)
	for _, item := range s.datasetProfiles {
		if taskType != "" && item.TaskType != taskType {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4IdeaStore) GetDatasetProfileByID(_ context.Context, id string) (*model.Phase4DatasetProfile, error) {
	item, ok := s.datasetProfiles[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4IdeaStore) CreateDatasetProfile(_ context.Context, item model.Phase4DatasetProfile) error {
	s.datasetProfiles[item.ID] = item
	return nil
}

func (s *memoryPhase4IdeaStore) UpdateDatasetProfile(_ context.Context, item model.Phase4DatasetProfile) error {
	s.datasetProfiles[item.ID] = item
	return nil
}

func (s *memoryPhase4IdeaStore) DeleteDatasetProfile(_ context.Context, id string) error {
	delete(s.datasetProfiles, id)
	return nil
}

func (s *memoryPhase4IdeaStore) ListReaderSources(_ context.Context, datasetProfileID string) ([]model.Phase4ReaderSource, error) {
	items := make([]model.Phase4ReaderSource, 0)
	for _, item := range s.readerSources {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4IdeaStore) GetReaderSourceByID(_ context.Context, id string) (*model.Phase4ReaderSource, error) {
	item, ok := s.readerSources[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4IdeaStore) CreateReaderSource(_ context.Context, item model.Phase4ReaderSource) error {
	s.readerSources[item.ID] = item
	return nil
}

func (s *memoryPhase4IdeaStore) UpdateReaderSource(_ context.Context, item model.Phase4ReaderSource) error {
	s.readerSources[item.ID] = item
	return nil
}

func (s *memoryPhase4IdeaStore) ListReaderContexts(_ context.Context, datasetProfileID string) ([]model.Phase4ReaderContext, error) {
	items := make([]model.Phase4ReaderContext, 0)
	for _, item := range s.readerContexts {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4IdeaStore) GetReaderContextByID(_ context.Context, id string) (*model.Phase4ReaderContext, error) {
	item, ok := s.readerContexts[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4IdeaStore) CreateReaderContext(_ context.Context, item model.Phase4ReaderContext) error {
	s.readerContexts[item.ID] = item
	return nil
}

func (s *memoryPhase4IdeaStore) UpdateReaderContext(_ context.Context, item model.Phase4ReaderContext) error {
	s.readerContexts[item.ID] = item
	return nil
}

func (s *memoryPhase4IdeaStore) ListIdeas(_ context.Context, datasetProfileID string, status string) ([]model.Phase4Idea, error) {
	items := make([]model.Phase4Idea, 0)
	for _, item := range s.ideas {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4IdeaStore) GetIdeaByID(_ context.Context, id string) (*model.Phase4Idea, error) {
	item, ok := s.ideas[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4IdeaStore) CreateIdea(_ context.Context, item model.Phase4Idea) error {
	s.ideas[item.ID] = item
	return nil
}

func (s *memoryPhase4IdeaStore) UpdateIdea(_ context.Context, item model.Phase4Idea) error {
	s.ideas[item.ID] = item
	return nil
}

func (s *memoryPhase4IdeaStore) DeleteIdea(_ context.Context, id string) error {
	delete(s.ideas, id)
	return nil
}

func (s *memoryPhase4IdeaStore) ListRunManifests(_ context.Context, datasetProfileID string, ideaID string, status string) ([]model.Phase4RunManifest, error) {
	items := make([]model.Phase4RunManifest, 0)
	for _, item := range s.runs {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		if ideaID != "" && item.IdeaID != ideaID {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4IdeaStore) GetRunManifestByID(_ context.Context, id string) (*model.Phase4RunManifest, error) {
	item, ok := s.runs[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4IdeaStore) CreateRunManifest(_ context.Context, item model.Phase4RunManifest) error {
	s.runs[item.ID] = item
	return nil
}

func (s *memoryPhase4IdeaStore) UpdateRunManifest(_ context.Context, item model.Phase4RunManifest) error {
	s.runs[item.ID] = item
	return nil
}

func (s *memoryPhase4IdeaStore) ListStructuredReports(_ context.Context, runManifestID string) ([]model.Phase4StructuredReportRecord, error) {
	items := make([]model.Phase4StructuredReportRecord, 0)
	for _, item := range s.reports {
		if runManifestID != "" && item.RunManifestID != runManifestID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4IdeaStore) GetStructuredReportByID(_ context.Context, id string) (*model.Phase4StructuredReportRecord, error) {
	item, ok := s.reports[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4IdeaStore) CreateStructuredReport(_ context.Context, item model.Phase4StructuredReportRecord) error {
	s.reports[item.ID] = item
	return nil
}

func (s *memoryPhase4IdeaStore) UpdateStructuredReport(_ context.Context, item model.Phase4StructuredReportRecord) error {
	s.reports[item.ID] = item
	return nil
}

func setupPhase4IdeaAgent(t *testing.T) (*Phase4Service, *baseservice.Phase4Service) {
	t.Helper()
	workspaceRoot := t.TempDir()
	jobStore := newMemoryJobStore()
	triggerStore := newMemoryTriggerStore()
	artifactStore := newMemoryArtifactStore()
	phase4BackingStore := newMemoryPhase4IdeaStore()
	phase4BackingStore.datasetProfiles["p4ds_visdom"] = model.Phase4DatasetProfile{
		ID:                  "p4ds_visdom",
		DatasetName:         "VisDoM",
		TaskType:            "retrieval",
		ModalityComposition: []string{"image", "text"},
		OfficialMetric:      "Recall@10",
		ServerID:            "srv_visdom",
		ServerPath:          "/home/bzli/mrag/datasets/visdom",
		Status:              model.Phase4DatasetProfileStatusActive,
	}
	phase4BackingStore.readerContexts["p4ctx_visdom"] = model.Phase4ReaderContext{
		ID:               "p4ctx_visdom",
		DatasetProfileID: "p4ds_visdom",
		Title:            "VisDoM Retrieval Context",
		Summary:          "Focus on page-level retrieval.",
		TaskDefinition:   "Improve page-level retrieval recall for visually rich documents.",
		RelatedWork:      []string{"late interaction reranking", "layout-aware retrieval"},
		RetrievalFocus:   []string{"page-level retrieval", "hard negative mining"},
		RankingNotes:     "Quality-first ranking.",
		StructuredContext: map[string]any{
			"task_definition":               "Improve page-level retrieval recall for visually rich documents.",
			"dataset_specific_challenges":   []string{"OCR noise", "template overlap"},
			"relevant_methods_landscape":    []string{"late interaction reranking", "layout-aware retrievers"},
			"likely_strong_baselines":       []string{"dual-encoder page retriever"},
			"common_failure_points":         []string{"Near-duplicate pages confuse retrieval"},
			"evaluation_caveats":            []string{"Recall@10 is official"},
			"implementation_constraints":    []string{"Keep first version page-level"},
			"promising_research_directions": []string{"hard negative mining", "query-conditioned chunking"},
			"citation_metadata":             []map[string]any{{"title": "VisDoM"}},
		},
		Status: model.Phase4ReaderContextStatusReady,
	}

	pythonExec := requirePython(t)
	pythonDir := pythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	artifactSvc := agentartifact.NewService(artifactStore)
	phase4Svc := baseservice.NewPhase4Service(phase4BackingStore)
	ideaPhase4Svc := NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactSvc, phase4Svc, workspaceRoot)
	triggerSvc.RegisterPostProcessor("idea_phase4", ideaPhase4Svc)
	return ideaPhase4Svc, phase4Svc
}

func TestPhase4IdeaServiceRunPersistsTenIdeasAndTop3(t *testing.T) {
	ideaSvc, _ := setupPhase4IdeaAgent(t)

	result, err := ideaSvc.Run(context.Background(), model.Phase4IdeaRunRequest{
		DatasetProfileID: "p4ds_visdom",
		ReaderContextID:  "p4ctx_visdom",
		UserNotes:        "Prefer concrete retrieval changes.",
		ExecutionMode:    "api",
		TargetCount:      10,
	})
	if err != nil {
		t.Fatalf("phase4 idea run failed: %v", err)
	}
	if result.Job == nil || result.Job.ID == "" {
		t.Fatalf("expected phase4 idea job")
	}
	if len(result.Ideas) != 10 {
		t.Fatalf("expected 10 persisted ideas, got %d", len(result.Ideas))
	}
	if len(result.TopRecommendations) != 3 {
		t.Fatalf("expected 3 top recommendations, got %d", len(result.TopRecommendations))
	}
	if result.TopRecommendations[0].Rank != 1 {
		t.Fatalf("expected top recommendation rank 1, got %d", result.TopRecommendations[0].Rank)
	}
	if len(phase4IdeaStringSliceValue(result.Job.NormalizedPayload["idea_ids"])) != 10 {
		t.Fatalf("expected job payload idea_ids to persist all ideas")
	}
}

func TestPhase4IdeaServiceRevisionGenerationBuildsLineage(t *testing.T) {
	ideaSvc, phase4Svc := setupPhase4IdeaAgent(t)
	original, err := phase4Svc.CreateIdea(context.Background(), model.Phase4IdeaCreateRequest{
		DatasetProfileID:  "p4ds_visdom",
		ReaderContextID:   "p4ctx_visdom",
		Title:             "VisDoM: Layout-Aware Hard Negative Mining",
		ProblemDefinition: "Improve page retrieval recall on VisDoM.",
		CoreMethod:        "Use layout-aware hard negatives.",
		Status:            model.Phase4IdeaStatusFailed,
		FailureFeedback:   map[string]any{"error": "low recall on near-duplicate pages"},
		LastFailureRunID:  "p4run_fail_001",
	})
	if err != nil {
		t.Fatalf("create source idea failed: %v", err)
	}

	result, err := ideaSvc.GenerateRevisionCandidates(context.Background(), original.ID, model.Phase4IdeaRevisionGenerateRequest{
		FailureFeedback:  map[string]any{"error": "low recall on near-duplicate pages"},
		LastFailureRunID: "p4run_fail_001",
		ExecutionMode:    "api",
		TargetCount:      3,
	})
	if err != nil {
		t.Fatalf("phase4 revision generation failed: %v", err)
	}
	if len(result.Ideas) != 3 {
		t.Fatalf("expected 3 revision candidates, got %d", len(result.Ideas))
	}
	for _, item := range result.Ideas {
		if item.RevisionOfID != original.ID {
			t.Fatalf("expected revision_of_id=%s, got %s", original.ID, item.RevisionOfID)
		}
		if item.LineageRootID != original.ID {
			t.Fatalf("expected lineage_root_id=%s, got %s", original.ID, item.LineageRootID)
		}
		if item.LastFailureRunID != "p4run_fail_001" {
			t.Fatalf("expected last failure run id to persist")
		}
	}
}

func TestPhase4IdeaServiceGetJobReturnsArtifactsAndIdeas(t *testing.T) {
	ideaSvc, _ := setupPhase4IdeaAgent(t)

	runResult, err := ideaSvc.Run(context.Background(), model.Phase4IdeaRunRequest{
		DatasetProfileID: "p4ds_visdom",
		ReaderContextID:  "p4ctx_visdom",
		ExecutionMode:    "api",
	})
	if err != nil {
		t.Fatalf("phase4 idea run failed: %v", err)
	}

	detail, err := ideaSvc.GetJob(context.Background(), runResult.Job.ID)
	if err != nil {
		t.Fatalf("phase4 idea get job failed: %v", err)
	}
	if detail == nil || detail.Job == nil {
		t.Fatalf("expected job detail")
	}
	if len(detail.Artifacts) == 0 {
		t.Fatalf("expected artifacts for phase4 idea job")
	}
	if len(detail.Ideas) != 10 {
		t.Fatalf("expected persisted ideas in job detail")
	}
	if len(detail.TopRecommendations) != 3 {
		t.Fatalf("expected top recommendations in job detail")
	}
}

func TestPhase4IdeaScoreViewHelpers(t *testing.T) {
	backingStore := newMemoryPhase4IdeaStore()
	phase4Svc := baseservice.NewPhase4Service(backingStore)
	backingStore.datasetProfiles["p4ds_visdom"] = model.Phase4DatasetProfile{
		ID:          "p4ds_visdom",
		DatasetName: "VisDoM",
		TaskType:    "retrieval",
		ServerID:    "srv_visdom",
		ServerPath:  "/home/bzli/mrag/datasets/visdom",
	}
	created, err := phase4Svc.CreateIdea(context.Background(), model.Phase4IdeaCreateRequest{
		DatasetProfileID:  "p4ds_visdom",
		Title:             "Idea A",
		ProblemDefinition: "p",
		CoreMethod:        "m",
		Score: model.Phase4IdeaScore{
			Novelty:         7,
			DatasetFit:      8,
			Feasibility:     8,
			ExpectedGain:    8,
			ComputeCost:     3,
			FailureRisk:     2,
			Reproducibility: 9,
		},
		ScoreSummary: map[string]any{
			"overallScore":         8.2,
			"rank":                 1,
			"recommendationTier":   "top3",
			"recommendationReason": "Strong balance.",
		},
		Status: model.Phase4IdeaStatusScored,
	})
	if err != nil {
		t.Fatalf("create idea failed: %v", err)
	}
	view, err := phase4Svc.GetIdeaScoreView(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get idea score view failed: %v", err)
	}
	if view.OverallScore != 8.2 {
		t.Fatalf("expected overall score 8.2, got %v", view.OverallScore)
	}
	items, err := phase4Svc.ListIdeaScoreViews(context.Background(), "p4ds_visdom", model.Phase4IdeaStatusScored)
	if err != nil {
		t.Fatalf("list score views failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one score view, got %d", len(items))
	}
}

func TestPhase4IdeaRejectHelper(t *testing.T) {
	backingStore := newMemoryPhase4IdeaStore()
	phase4Svc := baseservice.NewPhase4Service(backingStore)
	backingStore.datasetProfiles["p4ds_visdom"] = model.Phase4DatasetProfile{
		ID:          "p4ds_visdom",
		DatasetName: "VisDoM",
		TaskType:    "retrieval",
		ServerID:    "srv_visdom",
		ServerPath:  "/home/bzli/mrag/datasets/visdom",
	}
	created, err := phase4Svc.CreateIdea(context.Background(), model.Phase4IdeaCreateRequest{
		DatasetProfileID:  "p4ds_visdom",
		Title:             "Idea A",
		ProblemDefinition: "p",
		CoreMethod:        "m",
		Status:            model.Phase4IdeaStatusScored,
	})
	if err != nil {
		t.Fatalf("create idea failed: %v", err)
	}
	rejected, err := phase4Svc.RejectIdea(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("reject idea failed: %v", err)
	}
	if rejected.Status != model.Phase4IdeaStatusRejected {
		t.Fatalf("expected rejected status, got %s", rejected.Status)
	}
}

func TestPhase4IdeaRunManifestFailureStatusStillSeparateFromIdea(t *testing.T) {
	backingStore := newMemoryPhase4IdeaStore()
	phase4Svc := baseservice.NewPhase4Service(backingStore)
	backingStore.datasetProfiles["p4ds_visdom"] = model.Phase4DatasetProfile{
		ID:          "p4ds_visdom",
		DatasetName: "VisDoM",
		TaskType:    "retrieval",
		ServerID:    "srv_visdom",
		ServerPath:  "/home/bzli/mrag/datasets/visdom",
	}
	idea, err := phase4Svc.CreateIdea(context.Background(), model.Phase4IdeaCreateRequest{
		DatasetProfileID:  "p4ds_visdom",
		Title:             "Idea A",
		ProblemDefinition: "p",
		CoreMethod:        "m",
		Status:            model.Phase4IdeaStatusSelected,
	})
	if err != nil {
		t.Fatalf("create idea failed: %v", err)
	}
	run, err := phase4Svc.CreateRunManifest(context.Background(), model.Phase4RunManifestCreateRequest{
		DatasetProfileID: "p4ds_visdom",
		IdeaID:           idea.ID,
		RunnerMode:       "remote",
	})
	if err != nil {
		t.Fatalf("create run failed: %v", err)
	}
	run, err = phase4Svc.UpdateRunManifestStatus(context.Background(), run.ID, model.Phase4RunManifestStatusUpdateRequest{
		Status: model.Phase4RunStatusQueued,
	})
	if err != nil {
		t.Fatalf("queue run failed: %v", err)
	}
	run, err = phase4Svc.UpdateRunManifest(context.Background(), run.ID, model.Phase4RunManifestUpdateRequest{
		Status: ptrString(model.Phase4RunStatusRunning),
	})
	if err != nil {
		t.Fatalf("set run running failed: %v", err)
	}
	now := time.Now()
	run, err = phase4Svc.UpdateRunManifestStatus(context.Background(), run.ID, model.Phase4RunManifestStatusUpdateRequest{
		Status:          model.Phase4RunStatusTestFailed,
		RetryCount:      ptrInt(3),
		FailureFeedback: map[string]any{"error": "test failed after max retries"},
		FinishedAt:      &now,
	})
	if err != nil {
		t.Fatalf("update run failed: %v", err)
	}
	if run.Status != model.Phase4RunStatusTestFailed {
		t.Fatalf("expected run status test_failed, got %s", run.Status)
	}
	storedIdea, err := phase4Svc.GetIdeaByID(context.Background(), idea.ID)
	if err != nil {
		t.Fatalf("get idea failed: %v", err)
	}
	if storedIdea.Status != model.Phase4IdeaStatusSelected {
		t.Fatalf("expected idea status to remain selected, got %s", storedIdea.Status)
	}
}

func ptrInt(value int) *int {
	return &value
}

func ptrString(value string) *string {
	return &value
}
