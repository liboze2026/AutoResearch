package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/agentartifact"
	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/service"
	"mrag-platform/backend/go/internal/writeragent"
)

func TestPhase4WriterHandlerRunAndGetJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceRoot := t.TempDir()
	jobStore := newPhase4ReaderMemoryJobStore()
	triggerStore := newPhase4ReaderMemoryTriggerStore()
	artifactStore := newPhase4ReaderMemoryArtifactStore()
	phase4Store := newMemoryPhase4HandlerStore()
	phase4Svc := service.NewPhase4Service(phase4Store)

	datasetProfile, err := phase4Svc.CreateDatasetProfile(context.Background(), model.Phase4DatasetProfileCreateRequest{
		DatasetName: "VisDoM",
		TaskType:    "page_level_retrieval",
		SourceMode:  model.Phase4DatasetProfileSourceRegisteredPath,
		ServerID:    "srv_visdom",
		ServerPath:  "/home/bzli/mrag/datasets/visdom",
		Citation:    "VisDoM dataset citation.",
	})
	if err != nil {
		t.Fatalf("create dataset profile failed: %v", err)
	}
	sourceOne, err := phase4Svc.CreateReaderSource(context.Background(), model.Phase4ReaderSourceCreateRequest{
		DatasetProfileID: datasetProfile.ID,
		Title:            "VisDoM Paper",
		Authors:          []string{"Author A"},
		Venue:            "ACL",
		PublicationYear:  2025,
		SourceType:       "conference",
		SourceURL:        "https://example.org/visdom-paper",
		Metadata:         map[string]any{"doi": "10.1000/visdom.1"},
	})
	if err != nil {
		t.Fatalf("create reader source failed: %v", err)
	}
	sourceTwo, err := phase4Svc.CreateReaderSource(context.Background(), model.Phase4ReaderSourceCreateRequest{
		DatasetProfileID: datasetProfile.ID,
		Title:            "Layout Retrieval",
		Authors:          []string{"Author B"},
		Venue:            "arXiv",
		PublicationYear:  2024,
		SourceType:       "arxiv",
		SourceURL:        "https://example.org/layout",
	})
	if err != nil {
		t.Fatalf("create reader source failed: %v", err)
	}
	readerContext, err := phase4Svc.CreateReaderContext(context.Background(), model.Phase4ReaderContextCreateRequest{
		DatasetProfileID: datasetProfile.ID,
		Title:            "VisDoM Context",
		Summary:          "Reader summary for page-level retrieval.",
		TaskDefinition:   "Retrieve the correct page for each question.",
		SourceIDs:        []string{sourceOne.ID, sourceTwo.ID},
		StructuredContext: map[string]any{
			"relevant_methods_landscape": []any{"hybrid sparse+dense retrieval"},
			"implementation_constraints": []any{"page-level first"},
		},
		Status: model.Phase4ReaderContextStatusReady,
	})
	if err != nil {
		t.Fatalf("create reader context failed: %v", err)
	}
	idea, err := phase4Svc.CreateIdea(context.Background(), model.Phase4IdeaCreateRequest{
		DatasetProfileID:  datasetProfile.ID,
		ReaderContextID:   readerContext.ID,
		Title:             "Layout-Aware Hard Negative Mining",
		ProblemDefinition: "Improve recall on visually similar pages.",
		CoreMethod:        "Weighted lexical retrieval with layout signals.",
		Differentiators:   "Adds OCR/title weighting.",
		RiskPoints:        []string{"ocr noise"},
		ExpectedGains:     []string{"better recall@5"},
		Score: model.Phase4IdeaScore{
			Novelty: 0.7, DatasetFit: 0.9, Feasibility: 0.95, ExpectedGain: 0.82, ComputeCost: 0.88, FailureRisk: 0.3, Reproducibility: 0.93,
		},
		Status:     model.Phase4IdeaStatusSelected,
		SourceType: "phase4_generated",
	})
	if err != nil {
		t.Fatalf("create idea failed: %v", err)
	}

	artifactDir := filepath.Join(workspaceRoot, "phase4", "artifacts", "p4run_handler")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	metricsPath := filepath.Join(artifactDir, "metrics.json")
	if err := os.WriteFile(metricsPath, []byte(`{"primary_metric":"recall@5","values":{"recall@1":0.6,"recall@5":0.8,"recall@10":0.9,"mrr":0.7,"ndcg@10":0.76}}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	humanReportPath := filepath.Join(artifactDir, "report.md")
	if err := os.WriteFile(humanReportPath, []byte("# Baseline Report\nStable retrieval baseline.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	machineReportPath := filepath.Join(artifactDir, "machine_report.json")
	if err := os.WriteFile(machineReportPath, []byte(`{"pipeline":"retrieval_mainline","notes":["baseline"]}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	runManifest, err := phase4Svc.CreateRunManifest(context.Background(), model.Phase4RunManifestCreateRequest{
		DatasetProfileID: datasetProfile.ID,
		IdeaID:           idea.ID,
		ReaderContextID:  readerContext.ID,
		RunnerMode:       "local_dummy",
		Status:           model.Phase4RunStatusSucceeded,
		MaxRetryCount:    3,
		ArtifactPaths: map[string]any{
			"artifact_dir":        artifactDir,
			"metrics_path":        metricsPath,
			"human_report_path":   humanReportPath,
			"machine_report_path": machineReportPath,
		},
		MetricsPath: metricsPath,
	})
	if err != nil {
		t.Fatalf("create run manifest failed: %v", err)
	}

	pythonExec := requirePhase4Python(t)
	pythonDir := phase4ReaderPythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	artifactSvc := agentartifact.NewService(artifactStore)
	writerSvc := writeragent.NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactSvc, phase4Svc, workspaceRoot)
	triggerSvc.RegisterPostProcessor("writer_phase4", writerSvc)

	h := NewPhase4WriterHandler(writerSvc)
	router := gin.New()
	router.POST("/api/v2/phase4/writer/run", h.Run)
	router.GET("/api/v2/phase4/writer/jobs/:id", h.GetJob)

	runResp := doPhase4JSON(t, router, "POST", "/api/v2/phase4/writer/run", model.Phase4WriterRunRequest{
		RunManifestID: runManifest.ID,
		ExecutionMode: "mock",
	})
	if runResp.Code != 0 {
		t.Fatalf("expected phase4 writer run success, got %+v", runResp)
	}
	var runResult model.Phase4WriterRunResult
	mustDecodePhase4Data(t, runResp.Data, &runResult)
	if runResult.Job == nil || runResult.Report == nil {
		t.Fatalf("expected writer job and report")
	}

	jobResp := doPhase4JSON(t, router, "GET", "/api/v2/phase4/writer/jobs/"+runResult.Job.ID, nil)
	if jobResp.Code != 0 {
		t.Fatalf("expected phase4 writer job success, got %+v", jobResp)
	}
	var detail model.Phase4WriterJobDetail
	mustDecodePhase4Data(t, jobResp.Data, &detail)
	if detail.Report == nil {
		t.Fatalf("expected report in writer job detail")
	}
	if len(detail.Artifacts) == 0 {
		t.Fatalf("expected writer artifacts in job detail")
	}
}
