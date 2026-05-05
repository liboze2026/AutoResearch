package writeragent

import (
	"context"
	"os"
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
)

type phase4WriterMemoryJobStore struct {
	items map[string]model.AgentJob
}

func newPhase4WriterMemoryJobStore() *phase4WriterMemoryJobStore {
	return &phase4WriterMemoryJobStore{items: map[string]model.AgentJob{}}
}

func (s *phase4WriterMemoryJobStore) Create(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

func (s *phase4WriterMemoryJobStore) GetByID(_ context.Context, id string) (*model.AgentJob, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *phase4WriterMemoryJobStore) Update(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

type phase4WriterMemoryTriggerStore struct {
	items map[string]model.AgentJobTrigger
}

func newPhase4WriterMemoryTriggerStore() *phase4WriterMemoryTriggerStore {
	return &phase4WriterMemoryTriggerStore{items: map[string]model.AgentJobTrigger{}}
}

func (s *phase4WriterMemoryTriggerStore) Create(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}

func (s *phase4WriterMemoryTriggerStore) Update(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}

type phase4WriterMemoryArtifactStore struct {
	items map[string][]model.AgentArtifact
}

func newPhase4WriterMemoryArtifactStore() *phase4WriterMemoryArtifactStore {
	return &phase4WriterMemoryArtifactStore{items: map[string][]model.AgentArtifact{}}
}

func (s *phase4WriterMemoryArtifactStore) Create(_ context.Context, item model.AgentArtifact) error {
	s.items[item.JobID] = append(s.items[item.JobID], item)
	return nil
}

func (s *phase4WriterMemoryArtifactStore) ListByJobID(_ context.Context, jobID string) ([]model.AgentArtifact, error) {
	items := s.items[jobID]
	out := make([]model.AgentArtifact, len(items))
	copy(out, items)
	return out, nil
}

type memoryPhase4WriterStore struct {
	datasets map[string]model.Phase4DatasetProfile
	contexts map[string]model.Phase4ReaderContext
	sources  map[string]model.Phase4ReaderSource
	ideas    map[string]model.Phase4Idea
	runs     map[string]model.Phase4RunManifest
	reports  map[string]model.Phase4StructuredReportRecord
	seq      int
}

func newMemoryPhase4WriterStore() *memoryPhase4WriterStore {
	return &memoryPhase4WriterStore{
		datasets: map[string]model.Phase4DatasetProfile{},
		contexts: map[string]model.Phase4ReaderContext{},
		sources:  map[string]model.Phase4ReaderSource{},
		ideas:    map[string]model.Phase4Idea{},
		runs:     map[string]model.Phase4RunManifest{},
		reports:  map[string]model.Phase4StructuredReportRecord{},
	}
}

func (s *memoryPhase4WriterStore) GetRunManifestByID(_ context.Context, id string) (*model.Phase4RunManifest, error) {
	item, ok := s.runs[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4WriterStore) UpdateRunManifest(_ context.Context, id string, req model.Phase4RunManifestUpdateRequest) (*model.Phase4RunManifest, error) {
	item := s.runs[id]
	if req.ArtifactPaths != nil {
		item.ArtifactPaths = *req.ArtifactPaths
	}
	if req.LogsPath != nil {
		item.LogsPath = *req.LogsPath
	}
	if req.MetricsPath != nil {
		item.MetricsPath = *req.MetricsPath
	}
	item.UpdatedAt = time.Now()
	s.runs[id] = item
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4WriterStore) GetDatasetProfileByID(_ context.Context, id string) (*model.Phase4DatasetProfile, error) {
	item, ok := s.datasets[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4WriterStore) GetReaderContextByID(_ context.Context, id string) (*model.Phase4ReaderContext, error) {
	item, ok := s.contexts[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4WriterStore) GetReaderSourceByID(_ context.Context, id string) (*model.Phase4ReaderSource, error) {
	item, ok := s.sources[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4WriterStore) ListReaderSources(_ context.Context, datasetProfileID string) ([]model.Phase4ReaderSource, error) {
	items := make([]model.Phase4ReaderSource, 0)
	for _, item := range s.sources {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4WriterStore) GetIdeaByID(_ context.Context, id string) (*model.Phase4Idea, error) {
	item, ok := s.ideas[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4WriterStore) CreateStructuredReport(_ context.Context, req model.Phase4StructuredReportCreateRequest) (*model.Phase4StructuredReportRecord, error) {
	s.seq++
	now := time.Now()
	item := model.Phase4StructuredReportRecord{
		ID:                    "p4report_test_" + string(rune('a'+s.seq-1)),
		RunManifestID:         req.RunManifestID,
		DatasetProfileID:      req.DatasetProfileID,
		IdeaID:                req.IdeaID,
		ReaderContextID:       req.ReaderContextID,
		Title:                 req.Title,
		MachineReadableReport: phase4WriterCloneMap(req.MachineReadableReport),
		HumanReadableReportMD: req.HumanReadableReportMD,
		CitationRefs:          append([]string{}, req.CitationRefs...),
		ReferenceSourceIDs:    append([]string{}, req.ReferenceSourceIDs...),
		Status:                req.Status,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	s.reports[item.ID] = item
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4WriterStore) GetStructuredReportByID(_ context.Context, id string) (*model.Phase4StructuredReportRecord, error) {
	item, ok := s.reports[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func requirePhase4WriterPython(t *testing.T) string {
	t.Helper()
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available")
	}
	return python
}

func phase4WriterPythonAgentsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "python_agents"))
}

func TestPhase4WriterServiceRunMockCreatesStructuredReport(t *testing.T) {
	workspaceRoot := t.TempDir()
	jobStore := newPhase4WriterMemoryJobStore()
	triggerStore := newPhase4WriterMemoryTriggerStore()
	artifactStore := newPhase4WriterMemoryArtifactStore()
	phase4Store := newMemoryPhase4WriterStore()

	artifactDir := filepath.Join(workspaceRoot, "phase4", "artifacts", "p4run_1")
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

	now := time.Now()
	phase4Store.datasets["p4ds_visdom"] = model.Phase4DatasetProfile{
		ID:                    "p4ds_visdom",
		DatasetName:           "VisDoM",
		TaskType:              "page_level_retrieval",
		ModalityComposition:   []string{"document_image", "ocr_text"},
		OfficialMetric:        "recall@5",
		FileStructureSnapshot: map[string]any{"documents": 2},
		SampleStatistics:      map[string]any{"pageCount": 5, "queryCount": 5},
		Citation:              "VisDoM dataset citation.",
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	phase4Store.contexts["p4ctx_visdom"] = model.Phase4ReaderContext{
		ID:               "p4ctx_visdom",
		DatasetProfileID: "p4ds_visdom",
		Title:            "VisDoM Context",
		Summary:          "Reader summary for page-level retrieval.",
		TaskDefinition:   "Retrieve the correct page for each question.",
		SourceIDs:        []string{"p4src_1", "p4src_2"},
		StructuredContext: map[string]any{
			"relevant_methods_landscape": []any{"hybrid sparse+dense retrieval"},
			"implementation_constraints": []any{"page-level first"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	phase4Store.sources["p4src_1"] = model.Phase4ReaderSource{
		ID:               "p4src_1",
		DatasetProfileID: "p4ds_visdom",
		Title:            "VisDoM Paper",
		Authors:          []string{"Author A"},
		Venue:            "ACL",
		PublicationYear:  2025,
		SourceType:       "conference",
		SourceURL:        "https://example.org/visdom-paper",
		Metadata:         map[string]any{"doi": "10.1000/visdom.1"},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	phase4Store.sources["p4src_2"] = model.Phase4ReaderSource{
		ID:               "p4src_2",
		DatasetProfileID: "p4ds_visdom",
		Title:            "Layout Retrieval",
		Authors:          []string{"Author B"},
		Venue:            "arXiv",
		PublicationYear:  2024,
		SourceType:       "arxiv",
		SourceURL:        "https://example.org/layout",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	phase4Store.ideas["p4idea_1"] = model.Phase4Idea{
		ID:                "p4idea_1",
		DatasetProfileID:  "p4ds_visdom",
		ReaderContextID:   "p4ctx_visdom",
		Title:             "Layout-Aware Hard Negative Mining",
		ProblemDefinition: "Improve recall on visually similar pages.",
		CoreMethod:        "Weighted lexical retrieval with layout signals.",
		Differentiators:   "Adds OCR/title weighting.",
		RiskPoints:        []string{"ocr noise"},
		ExpectedGains:     []string{"better recall@5"},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	phase4Store.runs["p4run_1"] = model.Phase4RunManifest{
		ID:               "p4run_1",
		DatasetProfileID: "p4ds_visdom",
		IdeaID:           "p4idea_1",
		ReaderContextID:  "p4ctx_visdom",
		RunnerMode:       "local_dummy",
		Status:           model.Phase4RunStatusSucceeded,
		RetryCount:       1,
		MaxRetryCount:    3,
		ArtifactPaths: map[string]any{
			"artifact_dir":        artifactDir,
			"metrics_path":        metricsPath,
			"human_report_path":   humanReportPath,
			"machine_report_path": machineReportPath,
		},
		MetricsPath: metricsPath,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	pythonExec := requirePhase4WriterPython(t)
	pythonDir := phase4WriterPythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	artifactSvc := agentartifact.NewService(artifactStore)
	writerSvc := NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactSvc, phase4Store, workspaceRoot)
	triggerSvc.RegisterPostProcessor("writer_phase4", writerSvc)

	result, err := writerSvc.Run(context.Background(), model.Phase4WriterRunRequest{
		RunManifestID: "p4run_1",
		ExecutionMode: "mock",
	})
	if err != nil {
		t.Fatalf("phase4 writer run failed: %v", err)
	}
	if result.Job == nil || result.Report == nil {
		t.Fatalf("expected phase4 writer job and report")
	}
	if strings.TrimSpace(result.Report.Title) == "" {
		t.Fatalf("expected report title")
	}
	if len(result.Report.CitationRefs) == 0 {
		t.Fatalf("expected citation refs")
	}
	if strings.TrimSpace(result.Report.HumanReadableReportMD) == "" {
		t.Fatalf("expected human readable report")
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "phase4_structured_report.json")); err != nil {
		t.Fatalf("expected structured report file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "phase4_experiment_report.md")); err != nil {
		t.Fatalf("expected human report file: %v", err)
	}
	if phase4Store.runs["p4run_1"].ArtifactPaths["phase4_structured_report_id"] == "" {
		t.Fatalf("expected run artifact paths to record report id")
	}

	detail, err := writerSvc.GetJob(context.Background(), result.Job.ID)
	if err != nil {
		t.Fatalf("phase4 writer get job failed: %v", err)
	}
	if detail == nil || detail.Report == nil {
		t.Fatalf("expected writer job detail with report")
	}
	if len(detail.Artifacts) == 0 {
		t.Fatalf("expected writer artifacts in job detail")
	}
}
