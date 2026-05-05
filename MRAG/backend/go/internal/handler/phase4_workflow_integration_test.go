package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/agentartifact"
	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/codingagent"
	"mrag-platform/backend/go/internal/ideaagent"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/phase4workflow"
	"mrag-platform/backend/go/internal/readeragent"
	"mrag-platform/backend/go/internal/service"
	"mrag-platform/backend/go/internal/writeragent"
)

type phase4WorkflowIntegrationEventRecorder struct {
	items []model.AgentEventCreateRequest
}

func (r *phase4WorkflowIntegrationEventRecorder) PublishEvent(_ context.Context, req model.AgentEventCreateRequest) (*model.AgentEvent, error) {
	r.items = append(r.items, req)
	return &model.AgentEvent{
		ID:        "evt_" + req.EventType,
		EventType: req.EventType,
		SourceRef: req.SourceRef,
	}, nil
}

func phase4WorkflowPythonRunnersDir(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join(phase4ReaderPythonAgentsDir(t), "..", "python_runners"))
}

func phase4WorkflowFixtureDatasetDir(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join(phase4WorkflowPythonRunnersDir(t), "retrieval_mainline", "tests", "fixtures", "visdom_like_dataset"))
}

func TestPhase4WorkflowHandlerEndToEndSmoke(t *testing.T) {
	gin.SetMode(gin.TestMode)

	workspaceRoot := t.TempDir()
	jobStore := newPhase4ReaderMemoryJobStore()
	triggerStore := newPhase4ReaderMemoryTriggerStore()
	artifactStore := newPhase4ReaderMemoryArtifactStore()
	phase4Store := newMemoryPhase4HandlerStore()
	phase4Svc := service.NewPhase4Service(phase4Store)
	events := &phase4WorkflowIntegrationEventRecorder{}

	datasetProfile, err := phase4Svc.CreateDatasetProfile(context.Background(), model.Phase4DatasetProfileCreateRequest{
		DatasetName:         "VisDoM Fixture",
		TaskType:            "page_retrieval",
		ModalityComposition: []string{"image", "text"},
		SourceMode:          model.Phase4DatasetProfileSourceRegisteredPath,
		ServerID:            "srv_fixture_local",
		ServerPath:          phase4WorkflowFixtureDatasetDir(t),
		OfficialMetric:      "Recall@5",
		KnownDifficulties:   []string{"OCR noise", "template overlap"},
		UserNotes:           "Workflow integration smoke dataset.",
	})
	if err != nil {
		t.Fatalf("CreateDatasetProfile returned error: %v", err)
	}
	// Keep the dataset profile valid at creation time, then clear server binding in the
	// in-memory test store so the workflow uses local_dummy instead of remote execution.
	item := phase4Store.datasetProfiles[datasetProfile.ID]
	item.ServerID = ""
	phase4Store.datasetProfiles[datasetProfile.ID] = item
	datasetProfile.ServerID = ""

	pythonExec := requirePhase4Python(t)
	pythonAgentsDir := phase4ReaderPythonAgentsDir(t)
	pythonRunnersDir := phase4WorkflowPythonRunnersDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonAgentsDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	artifactSvc := agentartifact.NewService(artifactStore)

	readerSvc := readeragent.NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactSvc, phase4Svc, workspaceRoot)
	ideaSvc := ideaagent.NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactSvc, phase4Svc, workspaceRoot)
	codingSvc := codingagent.NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactSvc, phase4Svc, events, workspaceRoot, pythonExec, pythonRunnersDir, "/home/bzli/mrag")
	codingSvc.AttachIdeaRevisionGenerator(ideaSvc)
	writerSvc := writeragent.NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactSvc, phase4Svc, workspaceRoot)

	triggerSvc.RegisterPostProcessor("reader_phase4", readerSvc)
	triggerSvc.RegisterPostProcessor("idea_phase4", ideaSvc)
	triggerSvc.RegisterPostProcessor("coding_phase4", codingSvc)
	triggerSvc.RegisterPostProcessor("writer_phase4", writerSvc)

	workflowSvc := phase4workflow.NewService(phase4Svc, readerSvc, ideaSvc, codingSvc, writerSvc, jobSvc, events)
	h := NewPhase4WorkflowHandler(workflowSvc)
	router := gin.New()
	router.POST("/api/v2/phase4/workflows", h.Create)
	router.GET("/api/v2/phase4/workflows/:id", h.Get)
	router.POST("/api/v2/phase4/workflows/:id/select-idea", h.SelectIdea)

	createResp := doPhase4JSON(t, router, "POST", "/api/v2/phase4/workflows", model.Phase4WorkflowCreateRequest{
		DatasetProfileID: datasetProfile.ID,
		Reader: model.Phase4WorkflowReaderConfig{
			ExecutionMode: "mock",
			SearchMode:    "dataset_first",
			MaxPapers:     6,
		},
		Idea: model.Phase4WorkflowIdeaConfig{
			ExecutionMode: "mock",
			TargetCount:   10,
		},
		Coding: model.Phase4WorkflowCodingConfig{
			ExecutionMode: "mock",
			RunnerMode:    "local_dummy",
			MaxRetryCount: 3,
		},
		Writing: model.Phase4WorkflowWritingConfig{
			ExecutionMode: "mock",
		},
		Metadata: map[string]any{
			"smoke": "phase4-workflow-e2e",
		},
	})
	if createResp.Code != 0 {
		t.Fatalf("expected workflow create success, got %+v", createResp)
	}

	var created model.Phase4WorkflowDetail
	mustDecodePhase4Data(t, createResp.Data, &created)
	if created.Workflow == nil {
		t.Fatalf("expected workflow detail payload")
	}
	if created.Workflow.Status != model.Phase4WorkflowStatusAwaitingSelection {
		t.Fatalf("expected awaiting_selection after create, got %s", created.Workflow.Status)
	}
	if created.ReaderContext == nil {
		t.Fatalf("expected reader context after create")
	}
	if len(created.Ideas) != 10 {
		t.Fatalf("expected 10 ideas after create, got %d", len(created.Ideas))
	}
	if len(created.TopRecommendations) < 3 {
		t.Fatalf("expected top recommendations after create, got %d", len(created.TopRecommendations))
	}
	if len(created.Timeline) < 4 {
		t.Fatalf("expected workflow timeline to include reader/idea actions, got %d", len(created.Timeline))
	}

	selectedIdeaID := created.TopRecommendations[0].ID
	selectResp := doPhase4JSON(t, router, "POST", "/api/v2/phase4/workflows/"+created.Workflow.ID+"/select-idea", model.Phase4WorkflowSelectIdeaRequest{
		IdeaID: selectedIdeaID,
	})
	if selectResp.Code != 0 {
		t.Fatalf("expected workflow select success, got %+v", selectResp)
	}

	var completed model.Phase4WorkflowDetail
	mustDecodePhase4Data(t, selectResp.Data, &completed)
	if completed.Workflow == nil {
		t.Fatalf("expected completed workflow detail")
	}
	if completed.Workflow.Status != model.Phase4WorkflowStatusCompleted {
		t.Fatalf("expected completed workflow, got %s", completed.Workflow.Status)
	}
	if completed.SelectedIdea == nil || completed.SelectedIdea.ID != selectedIdeaID {
		t.Fatalf("expected selected idea %s, got %#v", selectedIdeaID, completed.SelectedIdea)
	}
	if completed.CurrentRunManifest == nil {
		t.Fatalf("expected current run manifest in completed workflow")
	}
	if completed.CurrentRunManifest.Status != model.Phase4RunStatusSucceeded {
		t.Fatalf("expected succeeded run manifest, got %s", completed.CurrentRunManifest.Status)
	}
	if completed.LatestReport == nil {
		t.Fatalf("expected latest report in completed workflow")
	}
	if completed.LatestJobs.Reader == nil || completed.LatestJobs.Idea == nil || completed.LatestJobs.Coding == nil || completed.LatestJobs.Writer == nil {
		t.Fatalf("expected latest jobs for all workflow stages, got %#v", completed.LatestJobs)
	}

	requiredPaths := []string{
		phase4WorkflowStringValue(completed.CurrentRunManifest.ArtifactPaths["metrics_path"]),
		phase4WorkflowStringValue(completed.CurrentRunManifest.ArtifactPaths["human_report_path"]),
		phase4WorkflowStringValue(completed.CurrentRunManifest.ArtifactPaths["driver_log_path"]),
		phase4WorkflowStringValue(completed.CurrentRunManifest.ArtifactPaths["phase4_human_report_md_path"]),
		phase4WorkflowStringValue(completed.CurrentRunManifest.ArtifactPaths["phase4_structured_report_json_path"]),
	}
	for _, path := range requiredPaths {
		if path == "" {
			t.Fatalf("expected artifact path to be populated: %#v", completed.CurrentRunManifest.ArtifactPaths)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected workflow artifact %s: %v", path, err)
		}
	}
	if strings.TrimSpace(completed.CurrentRunManifest.LogsPath) == "" {
		t.Fatalf("expected logs path to be populated")
	}
	if _, err := os.Stat(completed.CurrentRunManifest.LogsPath); err != nil {
		t.Fatalf("expected logs path %s: %v", completed.CurrentRunManifest.LogsPath, err)
	}

	reportMarkdownPath := phase4WorkflowStringValue(completed.CurrentRunManifest.ArtifactPaths["phase4_human_report_md_path"])
	reportMarkdown, err := os.ReadFile(reportMarkdownPath)
	if err != nil {
		t.Fatalf("ReadFile report markdown returned error: %v", err)
	}
	if !containsAll(string(reportMarkdown), "Dataset:", "Core method:", "Run manifest:", "Artifact summary keys:") {
		t.Fatalf("expected report markdown sections, got %s", string(reportMarkdown))
	}

	getResp := doPhase4JSON(t, router, "GET", "/api/v2/phase4/workflows/"+created.Workflow.ID, nil)
	if getResp.Code != 0 {
		t.Fatalf("expected workflow get success, got %+v", getResp)
	}
	var fetched model.Phase4WorkflowDetail
	mustDecodePhase4Data(t, getResp.Data, &fetched)
	if fetched.Workflow == nil || fetched.Workflow.Status != model.Phase4WorkflowStatusCompleted {
		t.Fatalf("expected completed workflow detail on GET, got %#v", fetched.Workflow)
	}
	if len(fetched.NextActions) == 0 || fetched.NextActions[0].Action != model.Phase4WorkflowNextActionViewReport {
		t.Fatalf("expected view_report next action, got %#v", fetched.NextActions)
	}
	if len(fetched.Timeline) < 8 {
		t.Fatalf("expected full workflow timeline, got %d", len(fetched.Timeline))
	}

	for _, jobID := range []string{
		completed.LatestJobs.Reader.ID,
		completed.LatestJobs.Idea.ID,
		completed.LatestJobs.Coding.ID,
		completed.LatestJobs.Writer.ID,
	} {
		if len(artifactStore.items[jobID]) == 0 {
			t.Fatalf("expected persisted artifacts for job %s", jobID)
		}
	}

	eventTypes := make(map[string]struct{}, len(events.items))
	for _, item := range events.items {
		eventTypes[item.EventType] = struct{}{}
	}
	for _, expected := range []string{
		"phase4_workflow_started",
		"phase4_reader_ready",
		"phase4_idea_batch_ready",
		"phase4_idea_selected",
		"phase4_run_ready",
		"phase4_workflow_completed",
	} {
		if _, ok := eventTypes[expected]; !ok {
			t.Fatalf("expected event %s, got %#v", expected, events.items)
		}
	}
}

func containsAll(text string, fragments ...string) bool {
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			return false
		}
	}
	return true
}

func phase4WorkflowStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
