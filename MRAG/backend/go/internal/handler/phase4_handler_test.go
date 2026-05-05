package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/service"
)

type memoryPhase4HandlerStore struct {
	datasetProfiles map[string]model.Phase4DatasetProfile
	readerSources   map[string]model.Phase4ReaderSource
	readerContexts  map[string]model.Phase4ReaderContext
	ideas           map[string]model.Phase4Idea
	runs            map[string]model.Phase4RunManifest
	reports         map[string]model.Phase4StructuredReportRecord
	workflows       map[string]model.Phase4Workflow
	workflowActions []model.Phase4WorkflowAction
}

func newMemoryPhase4HandlerStore() *memoryPhase4HandlerStore {
	return &memoryPhase4HandlerStore{
		datasetProfiles: map[string]model.Phase4DatasetProfile{},
		readerSources:   map[string]model.Phase4ReaderSource{},
		readerContexts:  map[string]model.Phase4ReaderContext{},
		ideas:           map[string]model.Phase4Idea{},
		runs:            map[string]model.Phase4RunManifest{},
		reports:         map[string]model.Phase4StructuredReportRecord{},
		workflows:       map[string]model.Phase4Workflow{},
		workflowActions: []model.Phase4WorkflowAction{},
	}
}

func (s *memoryPhase4HandlerStore) ListDatasetProfiles(_ context.Context, taskType string, status string) ([]model.Phase4DatasetProfile, error) {
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

func (s *memoryPhase4HandlerStore) GetDatasetProfileByID(_ context.Context, id string) (*model.Phase4DatasetProfile, error) {
	item, ok := s.datasetProfiles[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4HandlerStore) CreateDatasetProfile(_ context.Context, item model.Phase4DatasetProfile) error {
	s.datasetProfiles[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) UpdateDatasetProfile(_ context.Context, item model.Phase4DatasetProfile) error {
	s.datasetProfiles[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) DeleteDatasetProfile(_ context.Context, id string) error {
	delete(s.datasetProfiles, id)
	return nil
}

func (s *memoryPhase4HandlerStore) ListReaderSources(_ context.Context, datasetProfileID string) ([]model.Phase4ReaderSource, error) {
	items := make([]model.Phase4ReaderSource, 0)
	for _, item := range s.readerSources {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4HandlerStore) GetReaderSourceByID(_ context.Context, id string) (*model.Phase4ReaderSource, error) {
	item, ok := s.readerSources[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4HandlerStore) CreateReaderSource(_ context.Context, item model.Phase4ReaderSource) error {
	s.readerSources[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) UpdateReaderSource(_ context.Context, item model.Phase4ReaderSource) error {
	s.readerSources[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) ListReaderContexts(_ context.Context, datasetProfileID string) ([]model.Phase4ReaderContext, error) {
	items := make([]model.Phase4ReaderContext, 0)
	for _, item := range s.readerContexts {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4HandlerStore) GetReaderContextByID(_ context.Context, id string) (*model.Phase4ReaderContext, error) {
	item, ok := s.readerContexts[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4HandlerStore) CreateReaderContext(_ context.Context, item model.Phase4ReaderContext) error {
	s.readerContexts[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) UpdateReaderContext(_ context.Context, item model.Phase4ReaderContext) error {
	s.readerContexts[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) ListIdeas(_ context.Context, datasetProfileID string, status string) ([]model.Phase4Idea, error) {
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

func (s *memoryPhase4HandlerStore) GetIdeaByID(_ context.Context, id string) (*model.Phase4Idea, error) {
	item, ok := s.ideas[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4HandlerStore) CreateIdea(_ context.Context, item model.Phase4Idea) error {
	s.ideas[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) UpdateIdea(_ context.Context, item model.Phase4Idea) error {
	s.ideas[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) DeleteIdea(_ context.Context, id string) error {
	delete(s.ideas, id)
	return nil
}

func (s *memoryPhase4HandlerStore) ListRunManifests(_ context.Context, datasetProfileID string, ideaID string, status string) ([]model.Phase4RunManifest, error) {
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

func (s *memoryPhase4HandlerStore) GetRunManifestByID(_ context.Context, id string) (*model.Phase4RunManifest, error) {
	item, ok := s.runs[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4HandlerStore) CreateRunManifest(_ context.Context, item model.Phase4RunManifest) error {
	s.runs[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) UpdateRunManifest(_ context.Context, item model.Phase4RunManifest) error {
	s.runs[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) ListStructuredReports(_ context.Context, runManifestID string) ([]model.Phase4StructuredReportRecord, error) {
	items := make([]model.Phase4StructuredReportRecord, 0)
	for _, item := range s.reports {
		if runManifestID != "" && item.RunManifestID != runManifestID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4HandlerStore) GetStructuredReportByID(_ context.Context, id string) (*model.Phase4StructuredReportRecord, error) {
	item, ok := s.reports[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4HandlerStore) CreateStructuredReport(_ context.Context, item model.Phase4StructuredReportRecord) error {
	s.reports[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) UpdateStructuredReport(_ context.Context, item model.Phase4StructuredReportRecord) error {
	s.reports[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) ListWorkflows(_ context.Context, datasetProfileID string, status string) ([]model.Phase4Workflow, error) {
	items := make([]model.Phase4Workflow, 0)
	for _, item := range s.workflows {
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

func (s *memoryPhase4HandlerStore) GetWorkflowByID(_ context.Context, id string) (*model.Phase4Workflow, error) {
	item, ok := s.workflows[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4HandlerStore) CreateWorkflow(_ context.Context, item model.Phase4Workflow) error {
	s.workflows[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) UpdateWorkflow(_ context.Context, item model.Phase4Workflow) error {
	s.workflows[item.ID] = item
	return nil
}

func (s *memoryPhase4HandlerStore) ListWorkflowActions(_ context.Context, workflowID string) ([]model.Phase4WorkflowAction, error) {
	items := make([]model.Phase4WorkflowAction, 0)
	for _, item := range s.workflowActions {
		if workflowID != "" && item.WorkflowID != workflowID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4HandlerStore) CreateWorkflowAction(_ context.Context, item model.Phase4WorkflowAction) error {
	s.workflowActions = append(s.workflowActions, item)
	return nil
}

func TestPhase4HandlerDatasetIdeaAndRunFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newMemoryPhase4HandlerStore()
	h := NewPhase4Handler(service.NewPhase4Service(store))

	router := gin.New()
	router.POST("/api/v2/phase4/dataset-profiles", h.CreateDatasetProfile)
	router.GET("/api/v2/phase4/dataset-profiles/:id", h.GetDatasetProfile)
	router.POST("/api/v2/phase4/ideas", h.CreateIdea)
	router.GET("/api/v2/phase4/ideas/:id", h.GetIdea)
	router.POST("/api/v2/phase4/runs", h.CreateRunManifest)
	router.POST("/api/v2/phase4/runs/:id/status", h.UpdateRunManifestStatus)

	datasetReq := model.Phase4DatasetProfileCreateRequest{
		DatasetName: "VisDoM",
		TaskType:    "retrieval",
		SourceMode:  model.Phase4DatasetProfileSourceRegisteredPath,
		ServerID:    "srv_1",
		ServerPath:  "/home/bzli/mrag/datasets/visdom",
	}
	datasetResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/dataset-profiles", datasetReq)
	if datasetResp.Code != 0 {
		t.Fatalf("expected dataset create success, got %+v", datasetResp)
	}
	var dataset model.Phase4DatasetProfile
	mustDecodePhase4Data(t, datasetResp.Data, &dataset)

	datasetGet := doPhase4JSON(t, router, http.MethodGet, "/api/v2/phase4/dataset-profiles/"+dataset.ID, nil)
	if datasetGet.Code != 0 {
		t.Fatalf("expected dataset get success, got %+v", datasetGet)
	}

	ideaResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/ideas", model.Phase4IdeaCreateRequest{
		DatasetProfileID:  dataset.ID,
		Title:             "Layout aware retrieval",
		ProblemDefinition: "Improve page retrieval",
		CoreMethod:        "layout aware negatives",
	})
	if ideaResp.Code != 0 {
		t.Fatalf("expected idea create success, got %+v", ideaResp)
	}
	var idea model.Phase4Idea
	mustDecodePhase4Data(t, ideaResp.Data, &idea)

	ideaGet := doPhase4JSON(t, router, http.MethodGet, "/api/v2/phase4/ideas/"+idea.ID, nil)
	if ideaGet.Code != 0 {
		t.Fatalf("expected idea get success, got %+v", ideaGet)
	}

	runResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/runs", model.Phase4RunManifestCreateRequest{
		DatasetProfileID: dataset.ID,
		IdeaID:           idea.ID,
		RunnerMode:       "remote",
	})
	if runResp.Code != 0 {
		t.Fatalf("expected run create success, got %+v", runResp)
	}
	var run model.Phase4RunManifest
	mustDecodePhase4Data(t, runResp.Data, &run)

	updateResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/runs/"+run.ID+"/status", model.Phase4RunManifestStatusUpdateRequest{
		Status: model.Phase4RunStatusQueued,
	})
	if updateResp.Code != 0 {
		t.Fatalf("expected run status update success, got %+v", updateResp)
	}
	mustDecodePhase4Data(t, updateResp.Data, &run)
	if run.Status != model.Phase4RunStatusQueued {
		t.Fatalf("expected queued status, got %s", run.Status)
	}
}

func TestPhase4HandlerIdeaScoreViewAndReject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newMemoryPhase4HandlerStore()
	h := NewPhase4Handler(service.NewPhase4Service(store))

	router := gin.New()
	router.POST("/api/v2/phase4/dataset-profiles", h.CreateDatasetProfile)
	router.POST("/api/v2/phase4/ideas", h.CreateIdea)
	router.GET("/api/v2/phase4/ideas/score-view", h.ListIdeaScoreViews)
	router.GET("/api/v2/phase4/ideas/:id/score-view", h.GetIdeaScoreView)
	router.POST("/api/v2/phase4/ideas/:id/reject", h.RejectIdea)

	datasetResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/dataset-profiles", model.Phase4DatasetProfileCreateRequest{
		DatasetName: "VisDoM",
		TaskType:    "retrieval",
		SourceMode:  model.Phase4DatasetProfileSourceRegisteredPath,
		ServerID:    "srv_1",
		ServerPath:  "/home/bzli/mrag/datasets/visdom",
	})
	if datasetResp.Code != 0 {
		t.Fatalf("expected dataset create success, got %+v", datasetResp)
	}
	var dataset model.Phase4DatasetProfile
	mustDecodePhase4Data(t, datasetResp.Data, &dataset)

	ideaResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/ideas", model.Phase4IdeaCreateRequest{
		DatasetProfileID:  dataset.ID,
		Title:             "Layout aware retrieval",
		ProblemDefinition: "Improve page retrieval",
		CoreMethod:        "layout aware negatives",
		Status:            model.Phase4IdeaStatusScored,
		Score: model.Phase4IdeaScore{
			Novelty:         7,
			DatasetFit:      9,
			Feasibility:     8,
			ExpectedGain:    8,
			ComputeCost:     3,
			FailureRisk:     2,
			Reproducibility: 8,
		},
		ScoreSummary: map[string]any{
			"overallScore":         8.1,
			"rank":                 1,
			"recommendationTier":   "top3",
			"recommendationReason": "Strong balance.",
		},
	})
	if ideaResp.Code != 0 {
		t.Fatalf("expected idea create success, got %+v", ideaResp)
	}
	var idea model.Phase4Idea
	mustDecodePhase4Data(t, ideaResp.Data, &idea)

	listResp := doPhase4JSON(t, router, http.MethodGet, "/api/v2/phase4/ideas/score-view?datasetProfileId="+dataset.ID+"&status=scored", nil)
	if listResp.Code != 0 {
		t.Fatalf("expected score view list success, got %+v", listResp)
	}
	var scoreViews []model.Phase4IdeaScoreView
	mustDecodePhase4Data(t, listResp.Data, &scoreViews)
	if len(scoreViews) != 1 || scoreViews[0].OverallScore != 8.1 {
		t.Fatalf("unexpected score views: %#v", scoreViews)
	}

	itemResp := doPhase4JSON(t, router, http.MethodGet, "/api/v2/phase4/ideas/"+idea.ID+"/score-view", nil)
	if itemResp.Code != 0 {
		t.Fatalf("expected score view detail success, got %+v", itemResp)
	}
	var scoreView model.Phase4IdeaScoreView
	mustDecodePhase4Data(t, itemResp.Data, &scoreView)
	if scoreView.ID != idea.ID {
		t.Fatalf("expected score view for idea %s, got %s", idea.ID, scoreView.ID)
	}

	rejectResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/ideas/"+idea.ID+"/reject", map[string]any{})
	if rejectResp.Code != 0 {
		t.Fatalf("expected reject success, got %+v", rejectResp)
	}
	mustDecodePhase4Data(t, rejectResp.Data, &idea)
	if idea.Status != model.Phase4IdeaStatusRejected {
		t.Fatalf("expected rejected status, got %s", idea.Status)
	}
}

type phase4Envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func doPhase4JSON(t *testing.T, router *gin.Engine, method string, path string, body any) phase4Envelope {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body failed: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var resp phase4Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	return resp
}

func mustDecodePhase4Data(t *testing.T, raw json.RawMessage, dest any) {
	t.Helper()
	if err := json.Unmarshal(raw, dest); err != nil {
		t.Fatalf("decode response data failed: %v", err)
	}
}
