package codingagent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/model"
)

type phase4CodingMemoryJobStore struct {
	items map[string]model.AgentJob
}

func newPhase4CodingMemoryJobStore() *phase4CodingMemoryJobStore {
	return &phase4CodingMemoryJobStore{items: map[string]model.AgentJob{}}
}

func (s *phase4CodingMemoryJobStore) Create(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

func (s *phase4CodingMemoryJobStore) GetByID(_ context.Context, id string) (*model.AgentJob, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *phase4CodingMemoryJobStore) Update(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

type phase4CodingMemoryTriggerStore struct {
	items map[string]model.AgentJobTrigger
}

func newPhase4CodingMemoryTriggerStore() *phase4CodingMemoryTriggerStore {
	return &phase4CodingMemoryTriggerStore{items: map[string]model.AgentJobTrigger{}}
}

func (s *phase4CodingMemoryTriggerStore) Create(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}

func (s *phase4CodingMemoryTriggerStore) Update(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}

type phase4CodingMemoryArtifactStore struct {
	items []model.AgentArtifact
}

func (s *phase4CodingMemoryArtifactStore) Create(_ context.Context, item model.AgentArtifact) error {
	s.items = append(s.items, item)
	return nil
}

func (s *phase4CodingMemoryArtifactStore) ListByJobID(_ context.Context, jobID string) ([]model.AgentArtifact, error) {
	out := make([]model.AgentArtifact, 0)
	for _, item := range s.items {
		if item.JobID == jobID {
			out = append(out, item)
		}
	}
	return out, nil
}

type phase4CodingMemoryDataService struct {
	datasets map[string]model.Phase4DatasetProfile
	contexts map[string]model.Phase4ReaderContext
	ideas    map[string]model.Phase4Idea
	runs     map[string]model.Phase4RunManifest
}

func newPhase4CodingMemoryDataService() *phase4CodingMemoryDataService {
	now := time.Now()
	return &phase4CodingMemoryDataService{
		datasets: map[string]model.Phase4DatasetProfile{
			"p4ds_1": {
				ID:                "p4ds_1",
				DatasetName:       "VisDoM",
				TaskType:          "multimodal_retrieval",
				OfficialMetric:    "recall@5",
				ServerPath:        "/datasets/visdom",
				KnownDifficulties: []string{"page-level retrieval first"},
				Splits:            []model.Phase4DatasetSplit{{Name: "train"}, {Name: "val"}, {Name: "test"}},
				Status:            model.Phase4DatasetProfileStatusActive,
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
		contexts: map[string]model.Phase4ReaderContext{
			"p4ctx_1": {
				ID:               "p4ctx_1",
				DatasetProfileID: "p4ds_1",
				Title:            "VisDoM retrieval context",
				TaskDefinition:   "Page-level retrieval for visually rich documents.",
				StructuredContext: map[string]any{
					"task_definition":            "Page-level retrieval for visually rich documents.",
					"implementation_constraints": []any{"page-level first", "retrieval accuracy first"},
					"likely_strong_baselines":    []any{"hybrid lexical-dense retrieval"},
				},
				Status:    model.Phase4ReaderContextStatusReady,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		ideas: map[string]model.Phase4Idea{
			"p4idea_1": {
				ID:                  "p4idea_1",
				DatasetProfileID:    "p4ds_1",
				ReaderContextID:     "p4ctx_1",
				Title:               "Layout-Aware Hard Negative Mining",
				ProblemDefinition:   "Improve page-level retrieval recall on VisDoM.",
				CoreMethod:          "Use layout-aware hard negative sampling inside retrieval training.",
				DataProcessingNeeds: []string{"page graph", "hard negatives"},
				ModelChanges:        []string{"layout-aware encoder", "negative mining head"},
				TrainingPlan:        "Start with dummy retrieval mainline then replace with real retrieval training.",
				EvaluationMetrics:   []string{"recall@5"},
				RiskPoints:          []string{"layout noise"},
				Status:              model.Phase4IdeaStatusSelected,
				CreatedAt:           now,
				UpdatedAt:           now,
			},
		},
		runs: map[string]model.Phase4RunManifest{},
	}
}

func (s *phase4CodingMemoryDataService) GetDatasetProfileByID(_ context.Context, id string) (*model.Phase4DatasetProfile, error) {
	item, ok := s.datasets[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *phase4CodingMemoryDataService) GetReaderContextByID(_ context.Context, id string) (*model.Phase4ReaderContext, error) {
	item, ok := s.contexts[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *phase4CodingMemoryDataService) GetIdeaByID(_ context.Context, id string) (*model.Phase4Idea, error) {
	item, ok := s.ideas[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *phase4CodingMemoryDataService) UpdateIdeaStatus(_ context.Context, id string, req model.Phase4IdeaStatusUpdateRequest) (*model.Phase4Idea, error) {
	item := s.ideas[id]
	if strings.TrimSpace(req.Status) != "" {
		item.Status = req.Status
	}
	if len(req.FailureFeedback) > 0 {
		item.FailureFeedback = req.FailureFeedback
	}
	if strings.TrimSpace(req.LastFailureRunID) != "" {
		item.LastFailureRunID = req.LastFailureRunID
	}
	item.UpdatedAt = time.Now()
	s.ideas[id] = item
	copyItem := item
	return &copyItem, nil
}

func (s *phase4CodingMemoryDataService) CreateRunManifest(_ context.Context, req model.Phase4RunManifestCreateRequest) (*model.Phase4RunManifest, error) {
	now := time.Now()
	item := model.Phase4RunManifest{
		ID:               "p4run_1",
		DatasetProfileID: req.DatasetProfileID,
		IdeaID:           req.IdeaID,
		ReaderContextID:  req.ReaderContextID,
		RunnerMode:       req.RunnerMode,
		ServerID:         req.ServerID,
		GPU:              req.GPU,
		Status:           model.NormalizePhase4RunStatus(req.Status),
		RetryCount:       req.RetryCount,
		MaxRetryCount:    req.MaxRetryCount,
		ArtifactPaths:    req.ArtifactPaths,
		LogsPath:         req.LogsPath,
		MetricsPath:      req.MetricsPath,
		FailureFeedback:  req.FailureFeedback,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.runs[item.ID] = item
	copyItem := item
	return &copyItem, nil
}

func (s *phase4CodingMemoryDataService) GetRunManifestByID(_ context.Context, id string) (*model.Phase4RunManifest, error) {
	item, ok := s.runs[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *phase4CodingMemoryDataService) UpdateRunManifest(_ context.Context, id string, req model.Phase4RunManifestUpdateRequest) (*model.Phase4RunManifest, error) {
	item := s.runs[id]
	if req.CodeSnapshotID != nil {
		item.CodeSnapshotID = *req.CodeSnapshotID
	}
	if req.RunnerMode != nil {
		item.RunnerMode = *req.RunnerMode
	}
	if req.ServerID != nil {
		item.ServerID = *req.ServerID
	}
	if req.GPU != nil {
		item.GPU = *req.GPU
	}
	if req.Status != nil {
		item.Status = *req.Status
	}
	if req.RetryCount != nil {
		item.RetryCount = *req.RetryCount
	}
	if req.MaxRetryCount != nil {
		item.MaxRetryCount = *req.MaxRetryCount
	}
	if req.ArtifactPaths != nil {
		item.ArtifactPaths = *req.ArtifactPaths
	}
	if req.LogsPath != nil {
		item.LogsPath = *req.LogsPath
	}
	if req.MetricsPath != nil {
		item.MetricsPath = *req.MetricsPath
	}
	if req.FailureFeedback != nil {
		item.FailureFeedback = *req.FailureFeedback
	}
	if req.StartedAt != nil {
		item.StartedAt = req.StartedAt
	}
	if req.FinishedAt != nil {
		item.FinishedAt = req.FinishedAt
	}
	item.UpdatedAt = time.Now()
	s.runs[id] = item
	copyItem := item
	return &copyItem, nil
}

func (s *phase4CodingMemoryDataService) UpdateRunManifestStatus(_ context.Context, id string, req model.Phase4RunManifestStatusUpdateRequest) (*model.Phase4RunManifest, error) {
	item := s.runs[id]
	item.Status = req.Status
	if req.RetryCount != nil {
		item.RetryCount = *req.RetryCount
	}
	if req.FailureFeedback != nil {
		item.FailureFeedback = req.FailureFeedback
	}
	if req.StartedAt != nil {
		item.StartedAt = req.StartedAt
	}
	if req.FinishedAt != nil {
		item.FinishedAt = req.FinishedAt
	}
	item.UpdatedAt = time.Now()
	s.runs[id] = item
	copyItem := item
	return &copyItem, nil
}

type phase4CodingEventPublisherStub struct {
	items []model.AgentEventCreateRequest
}

func (p *phase4CodingEventPublisherStub) PublishEvent(_ context.Context, req model.AgentEventCreateRequest) (*model.AgentEvent, error) {
	p.items = append(p.items, req)
	return &model.AgentEvent{ID: "evt_phase4_1", EventType: req.EventType}, nil
}

type phase4IdeaRevisionGeneratorStub struct {
	calls  []model.Phase4IdeaRevisionGenerateRequest
	ideaID string
	result *model.Phase4IdeaRunResult
}

func (s *phase4IdeaRevisionGeneratorStub) GenerateRevisionCandidates(_ context.Context, ideaID string, req model.Phase4IdeaRevisionGenerateRequest) (*model.Phase4IdeaRunResult, error) {
	s.ideaID = ideaID
	s.calls = append(s.calls, req)
	if s.result != nil {
		return s.result, nil
	}
	return &model.Phase4IdeaRunResult{
		Job: &model.AgentJob{ID: "ajob_phase4_idea_revision_1"},
		Ideas: []model.Phase4Idea{
			{ID: "p4idea_rev_1", RevisionOfID: ideaID, LastFailureRunID: req.LastFailureRunID},
		},
		TopRecommendations: []model.Phase4IdeaScoreView{
			{ID: "p4idea_rev_1", Title: "revision"},
		},
	}, nil
}

func TestPhase4CodingServiceRunAndPostProcessIntegration(t *testing.T) {
	workspaceRoot := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	pythonAgentsDir := filepath.Clean(filepath.Join(wd, "..", "..", "..", "python_agents"))
	pythonRunnersDir := filepath.Clean(filepath.Join(wd, "..", "..", "..", "python_runners"))

	jobStore := newPhase4CodingMemoryJobStore()
	triggerStore := newPhase4CodingMemoryTriggerStore()
	artifactStore := &phase4CodingMemoryArtifactStore{}
	phase4Data := newPhase4CodingMemoryDataService()
	events := &phase4CodingEventPublisherStub{}

	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService("python", pythonAgentsDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	codingSvc := NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactStore, phase4Data, events, workspaceRoot, "python", pythonRunnersDir, "/home/bzli/mrag")
	triggerSvc.RegisterPostProcessor("coding_phase4", codingSvc)

	result, err := codingSvc.Run(context.Background(), model.Phase4CodingRunRequest{
		DatasetProfileID: "p4ds_1",
		IdeaID:           "p4idea_1",
		ReaderContextID:  "p4ctx_1",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result == nil || result.Job == nil {
		t.Fatalf("expected coding result job")
	}
	if result.RunManifest == nil {
		t.Fatalf("expected run manifest in result")
	}
	if result.RunManifest.Status != model.Phase4RunStatusSucceeded {
		t.Fatalf("expected succeeded run manifest, got %s", result.RunManifest.Status)
	}

	runDir := filepath.Join(workspaceRoot, "phase4", "runs", result.RunManifest.ID)
	artifactDir := filepath.Join(workspaceRoot, "phase4", "artifacts", result.RunManifest.ID)
	for _, path := range []string{
		filepath.Join(runDir, "experiment_manifest.json"),
		filepath.Join(runDir, "config.json"),
		filepath.Join(runDir, "snapshot", "run_entrypoint.py"),
		filepath.Join(artifactDir, "metrics.json"),
		filepath.Join(artifactDir, "report.md"),
		filepath.Join(artifactDir, "dataset_tool_asset.json"),
		filepath.Join(artifactDir, "snapshot_manifest.json"),
		filepath.Join(runDir, "logs", "driver.log"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}

	metricsRaw, err := os.ReadFile(filepath.Join(artifactDir, "metrics.json"))
	if err != nil {
		t.Fatalf("ReadFile metrics returned error: %v", err)
	}
	var metrics map[string]any
	if err = json.Unmarshal(metricsRaw, &metrics); err != nil {
		t.Fatalf("Unmarshal metrics returned error: %v", err)
	}
	if strings.TrimSpace(stringValue(metrics["primary_metric"])) == "" {
		t.Fatalf("expected primary_metric in metrics: %#v", metrics)
	}
	if len(artifactStore.items) < 6 {
		t.Fatalf("expected persisted artifacts, got %d", len(artifactStore.items))
	}
	if len(events.items) != 1 || events.items[0].EventType != "phase4_run_ready" {
		t.Fatalf("expected phase4_run_ready event, got %#v", events.items)
	}
	snapshotManifestRaw, err := os.ReadFile(filepath.Join(artifactDir, "snapshot_manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile snapshot_manifest returned error: %v", err)
	}
	if !strings.Contains(string(snapshotManifestRaw), "run_entrypoint.py") {
		t.Fatalf("expected snapshot manifest to mention run_entrypoint.py")
	}
}

func TestPhase4RollbackRestoresSnapshot(t *testing.T) {
	workspaceRoot := t.TempDir()
	service := NewPhase4Service(nil, nil, nil, nil, nil, nil, workspaceRoot, "python", filepath.Join(workspaceRoot, "python_runners"), "/home/bzli/mrag")
	sourceRoot := filepath.Join(workspaceRoot, "source")
	snapshotDir := filepath.Join(workspaceRoot, "run", "snapshot")
	if err := os.MkdirAll(filepath.Join(sourceRoot, "methods", "generated"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "run_entrypoint.py"), []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(snapshotDir, "methods", "generated"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(snapshotDir, "run_entrypoint.py"), []byte("broken\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	artifactPaths := map[string]any{
		"source_root":        sourceRoot,
		"snapshot_dir":       snapshotDir,
		"method_module_path": filepath.Join(snapshotDir, "methods", "generated", "generated_method.py"),
	}
	if err := service.rollbackPhase4Snapshot(artifactPaths); err != nil {
		t.Fatalf("rollbackPhase4Snapshot returned error: %v", err)
	}
	restored, err := os.ReadFile(filepath.Join(snapshotDir, "run_entrypoint.py"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if string(restored) != "print('ok')\n" {
		t.Fatalf("expected snapshot rollback to restore source content, got %q", string(restored))
	}
}

func TestPhase4RepairLoopRecoversFromBrokenMethod(t *testing.T) {
	workspaceRoot := t.TempDir()
	now := time.Now()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	pythonRunnersDir := filepath.Clean(filepath.Join(wd, "..", "..", "..", "python_runners"))
	phase4Data := newPhase4CodingMemoryDataService()
	runManifest, err := phase4Data.CreateRunManifest(context.Background(), model.Phase4RunManifestCreateRequest{
		DatasetProfileID: "p4ds_1",
		IdeaID:           "p4idea_1",
		ReaderContextID:  "p4ctx_1",
		RunnerMode:       "local_dummy",
		Status:           model.Phase4RunStatusDraft,
		MaxRetryCount:    3,
	})
	if err != nil {
		t.Fatalf("CreateRunManifest returned error: %v", err)
	}
	artifactStore := &phase4CodingMemoryArtifactStore{}
	jobStore := newPhase4CodingMemoryJobStore()
	job := &model.AgentJob{
		ID:            "ajob_phase4_coding_repair_1",
		AgentType:     "coding_phase4",
		Status:        "succeeded",
		ExecutionMode: "mock",
		Metadata: map[string]any{
			"run_manifest_id": runManifest.ID,
		},
		NormalizedPayload: map[string]any{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	jobStore.items[job.ID] = *job
	service := NewPhase4Service(nil, jobStore, nil, artifactStore, phase4Data, &phase4CodingEventPublisherStub{}, workspaceRoot, "python", pythonRunnersDir, "/home/bzli/mrag")
	datasetProfile, _ := phase4Data.GetDatasetProfileByID(context.Background(), "p4ds_1")
	idea, _ := phase4Data.GetIdeaByID(context.Background(), "p4idea_1")
	readerContext, _ := phase4Data.GetReaderContextByID(context.Background(), "p4ctx_1")
	payload := model.Phase4CodingRuntimePayload{
		MethodModule: model.Phase4CodingMethodModule{
			ModuleName:   "broken_method",
			RelativePath: "methods/generated/broken_method.py",
			Content:      "this is not valid python\n",
		},
		Phase4Config: map[string]any{
			"protocol_version":   "phase4-retrieval-mainline-v1",
			"method_name":        "broken_method",
			"method_module_path": "methods/generated/broken_method.py",
			"runner_mode":        "local_dummy",
			"method_branch":      "method/broken_method",
			"dataset_adapter": map[string]any{
				"dataset_profile_id": "p4ds_1",
				"dataset_name":       "VisDoM",
				"task_type":          "multimodal_retrieval",
				"server_path":        "/datasets/visdom",
				"official_metric":    "recall@5",
			},
			"evaluate": map[string]any{
				"primary_metric": "recall@5",
			},
			"retry_policy": map[string]any{
				"max_retries": 3,
			},
		},
	}
	artifactPaths, snapshotID, err := service.preparePhase4Run(context.Background(), runManifest, datasetProfile, idea, readerContext, payload)
	if err != nil {
		t.Fatalf("preparePhase4Run returned error: %v", err)
	}
	if err := service.executePhase4ManagedRun(context.Background(), job, runManifest, datasetProfile, idea, readerContext, artifactPaths, snapshotID, payload); err != nil {
		t.Fatalf("executePhase4ManagedRun returned error: %v", err)
	}
	updatedRun, _ := phase4Data.GetRunManifestByID(context.Background(), runManifest.ID)
	if updatedRun == nil || updatedRun.Status != model.Phase4RunStatusSucceeded {
		t.Fatalf("expected succeeded run manifest after repair, got %#v", updatedRun)
	}
	if updatedRun.RetryCount < 1 {
		t.Fatalf("expected repair loop to consume at least one retry, got %d", updatedRun.RetryCount)
	}
	if _, err := os.Stat(stringValue(artifactPaths["repair_log_path"])); err != nil {
		t.Fatalf("expected repair log to exist: %v", err)
	}
}

func TestPhase4FinalTestFailedTriggersIdeaRevision(t *testing.T) {
	workspaceRoot := t.TempDir()
	now := time.Now()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	pythonRunnersDir := filepath.Clean(filepath.Join(wd, "..", "..", "..", "python_runners"))
	phase4Data := newPhase4CodingMemoryDataService()
	runManifest, err := phase4Data.CreateRunManifest(context.Background(), model.Phase4RunManifestCreateRequest{
		DatasetProfileID: "p4ds_1",
		IdeaID:           "p4idea_1",
		ReaderContextID:  "p4ctx_1",
		RunnerMode:       "local_dummy",
		Status:           model.Phase4RunStatusDraft,
		MaxRetryCount:    2,
	})
	if err != nil {
		t.Fatalf("CreateRunManifest returned error: %v", err)
	}
	artifactStore := &phase4CodingMemoryArtifactStore{}
	jobStore := newPhase4CodingMemoryJobStore()
	job := &model.AgentJob{
		ID:            "ajob_phase4_coding_fail_1",
		AgentType:     "coding_phase4",
		Status:        "succeeded",
		ExecutionMode: "mock",
		Metadata: map[string]any{
			"run_manifest_id": runManifest.ID,
		},
		NormalizedPayload: map[string]any{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	jobStore.items[job.ID] = *job
	service := NewPhase4Service(nil, jobStore, nil, artifactStore, phase4Data, &phase4CodingEventPublisherStub{}, workspaceRoot, "python", pythonRunnersDir, "/home/bzli/mrag")
	revisionStub := &phase4IdeaRevisionGeneratorStub{}
	service.AttachIdeaRevisionGenerator(revisionStub)
	datasetProfile, _ := phase4Data.GetDatasetProfileByID(context.Background(), "p4ds_1")
	idea, _ := phase4Data.GetIdeaByID(context.Background(), "p4idea_1")
	readerContext, _ := phase4Data.GetReaderContextByID(context.Background(), "p4ctx_1")
	payload := model.Phase4CodingRuntimePayload{
		MethodModule: model.Phase4CodingMethodModule{
			ModuleName:   "still_broken",
			RelativePath: "methods/generated/still_broken.py",
			Content:      "raise SyntaxError('broken')\n",
		},
		Phase4Config: map[string]any{
			"protocol_version":   "phase4-retrieval-mainline-v1",
			"method_name":        "still_broken",
			"method_module_path": "methods/generated/still_broken.py",
			"runner_mode":        "local_dummy",
			"method_branch":      "method/still_broken",
			"dataset_adapter": map[string]any{
				"dataset_profile_id": "p4ds_1",
				"dataset_name":       "BrokenDataset",
				"task_type":          "multimodal_retrieval",
				"server_path":        "/this/path/does/not/exist",
				"official_metric":    "recall@5",
				"metadata": map[string]any{
					"adapter_type": "generic_page_retrieval",
				},
			},
			"evaluate": map[string]any{
				"primary_metric": "recall@5",
			},
			"retry_policy": map[string]any{
				"max_retries": 2,
			},
		},
	}
	artifactPaths, snapshotID, err := service.preparePhase4Run(context.Background(), runManifest, datasetProfile, idea, readerContext, payload)
	if err != nil {
		t.Fatalf("preparePhase4Run returned error: %v", err)
	}
	if err := service.executePhase4ManagedRun(context.Background(), job, runManifest, datasetProfile, idea, readerContext, artifactPaths, snapshotID, payload); err != nil {
		t.Fatalf("executePhase4ManagedRun returned error: %v", err)
	}
	updatedRun, _ := phase4Data.GetRunManifestByID(context.Background(), runManifest.ID)
	if updatedRun == nil || updatedRun.Status != model.Phase4RunStatusTestFailed {
		t.Fatalf("expected test_failed run manifest, got %#v", updatedRun)
	}
	if len(revisionStub.calls) != 1 || revisionStub.ideaID != "p4idea_1" {
		t.Fatalf("expected revision generation call, got %+v", revisionStub)
	}
	if stringValue(updatedRun.FailureFeedback["status"]) != model.Phase4RunStatusTestFailed {
		t.Fatalf("expected structured failure feedback, got %#v", updatedRun.FailureFeedback)
	}
	storedIdea, _ := phase4Data.GetIdeaByID(context.Background(), "p4idea_1")
	if storedIdea == nil || storedIdea.LastFailureRunID != runManifest.ID {
		t.Fatalf("expected source idea failure metadata to be updated, got %#v", storedIdea)
	}
	if _, err := os.Stat(stringValue(artifactPaths["failure_feedback_path"])); err != nil {
		t.Fatalf("expected failure feedback artifact: %v", err)
	}
}
