package handler

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/agentartifact"
	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/readeragent"
	"mrag-platform/backend/go/internal/service"
)

type phase4ReaderMemoryJobStore struct {
	items map[string]model.AgentJob
}

func newPhase4ReaderMemoryJobStore() *phase4ReaderMemoryJobStore {
	return &phase4ReaderMemoryJobStore{items: map[string]model.AgentJob{}}
}

func (s *phase4ReaderMemoryJobStore) Create(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

func (s *phase4ReaderMemoryJobStore) GetByID(_ context.Context, id string) (*model.AgentJob, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *phase4ReaderMemoryJobStore) Update(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

type phase4ReaderMemoryTriggerStore struct {
	items map[string]model.AgentJobTrigger
}

func newPhase4ReaderMemoryTriggerStore() *phase4ReaderMemoryTriggerStore {
	return &phase4ReaderMemoryTriggerStore{items: map[string]model.AgentJobTrigger{}}
}

func (s *phase4ReaderMemoryTriggerStore) Create(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}

func (s *phase4ReaderMemoryTriggerStore) Update(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}

type phase4ReaderMemoryArtifactStore struct {
	items map[string][]model.AgentArtifact
}

func newPhase4ReaderMemoryArtifactStore() *phase4ReaderMemoryArtifactStore {
	return &phase4ReaderMemoryArtifactStore{items: map[string][]model.AgentArtifact{}}
}

func (s *phase4ReaderMemoryArtifactStore) Create(_ context.Context, item model.AgentArtifact) error {
	s.items[item.JobID] = append(s.items[item.JobID], item)
	return nil
}

func (s *phase4ReaderMemoryArtifactStore) ListByJobID(_ context.Context, jobID string) ([]model.AgentArtifact, error) {
	items := s.items[jobID]
	out := make([]model.AgentArtifact, len(items))
	copy(out, items)
	return out, nil
}

func requirePhase4Python(t *testing.T) string {
	t.Helper()
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available")
	}
	return python
}

func phase4ReaderPythonAgentsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "python_agents"))
}

func TestPhase4ReaderHandlerRunAndGetJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workspaceRoot := t.TempDir()
	jobStore := newPhase4ReaderMemoryJobStore()
	triggerStore := newPhase4ReaderMemoryTriggerStore()
	artifactStore := newPhase4ReaderMemoryArtifactStore()
	phase4Store := newMemoryPhase4HandlerStore()
	phase4Svc := service.NewPhase4Service(phase4Store)

	datasetProfile, err := phase4Svc.CreateDatasetProfile(context.Background(), model.Phase4DatasetProfileCreateRequest{
		DatasetName: "VisDoM",
		TaskType:    "retrieval",
		SourceMode:  model.Phase4DatasetProfileSourceRegisteredPath,
		ServerID:    "srv_visdom",
		ServerPath:  "/home/bzli/mrag/datasets/visdom",
	})
	if err != nil {
		t.Fatalf("create dataset profile failed: %v", err)
	}

	pythonExec := requirePhase4Python(t)
	pythonDir := phase4ReaderPythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	artifactSvc := agentartifact.NewService(artifactStore)
	readerSvc := readeragent.NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactSvc, phase4Svc, workspaceRoot)
	triggerSvc.RegisterPostProcessor("reader_phase4", readerSvc)

	h := NewPhase4ReaderHandler(readerSvc)
	router := gin.New()
	router.POST("/api/v2/phase4/reader/run", h.Run)
	router.GET("/api/v2/phase4/reader/jobs/:id", h.GetJob)

	runResp := doPhase4JSON(t, router, "POST", "/api/v2/phase4/reader/run", model.Phase4ReaderRunRequest{
		DatasetProfileID: datasetProfile.ID,
		ExecutionMode:    "mock",
		MaxPapers:        4,
	})
	if runResp.Code != 0 {
		t.Fatalf("expected phase4 reader run success, got %+v", runResp)
	}
	var runResult model.Phase4ReaderRunResult
	mustDecodePhase4Data(t, runResp.Data, &runResult)
	if runResult.Job == nil || runResult.ReaderContext == nil {
		t.Fatalf("expected phase4 reader job and context")
	}
	if len(runResult.ReaderSources) == 0 {
		t.Fatalf("expected phase4 reader sources")
	}

	jobResp := doPhase4JSON(t, router, "GET", "/api/v2/phase4/reader/jobs/"+runResult.Job.ID, nil)
	if jobResp.Code != 0 {
		t.Fatalf("expected phase4 reader job success, got %+v", jobResp)
	}
	var detail model.Phase4ReaderJobDetail
	mustDecodePhase4Data(t, jobResp.Data, &detail)
	if detail.ReaderContext == nil {
		t.Fatalf("expected reader context in job detail")
	}
	if len(detail.ReaderSources) == 0 {
		t.Fatalf("expected reader sources in job detail")
	}
	if len(detail.Artifacts) == 0 {
		t.Fatalf("expected reader artifacts in job detail")
	}
}
