package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
)

type phase4WorkflowHandlerServiceStub struct {
	createResp         *model.Phase4WorkflowDetail
	listResp           []model.Phase4Workflow
	getResp            *model.Phase4WorkflowDetail
	selectIdeaResp     *model.Phase4WorkflowDetail
	selectRevisionResp *model.Phase4WorkflowDetail
	retryResp          *model.Phase4WorkflowDetail
	archiveResp        *model.Phase4WorkflowDetail
}

func (s *phase4WorkflowHandlerServiceStub) CreateWorkflow(_ context.Context, _ model.Phase4WorkflowCreateRequest) (*model.Phase4WorkflowDetail, error) {
	return s.createResp, nil
}

func (s *phase4WorkflowHandlerServiceStub) ListWorkflows(_ context.Context, _ string, _ string) ([]model.Phase4Workflow, error) {
	return s.listResp, nil
}

func (s *phase4WorkflowHandlerServiceStub) GetWorkflow(_ context.Context, _ string) (*model.Phase4WorkflowDetail, error) {
	return s.getResp, nil
}

func (s *phase4WorkflowHandlerServiceStub) SelectIdea(_ context.Context, _ string, _ model.Phase4WorkflowSelectIdeaRequest) (*model.Phase4WorkflowDetail, error) {
	return s.selectIdeaResp, nil
}

func (s *phase4WorkflowHandlerServiceStub) SelectRevision(_ context.Context, _ string, _ model.Phase4WorkflowSelectIdeaRequest) (*model.Phase4WorkflowDetail, error) {
	return s.selectRevisionResp, nil
}

func (s *phase4WorkflowHandlerServiceStub) RetryStage(_ context.Context, _ string, _ model.Phase4WorkflowRetryStageRequest) (*model.Phase4WorkflowDetail, error) {
	return s.retryResp, nil
}

func (s *phase4WorkflowHandlerServiceStub) ArchiveWorkflow(_ context.Context, _ string) (*model.Phase4WorkflowDetail, error) {
	return s.archiveResp, nil
}

func TestPhase4WorkflowHandlerCRUDAndTransitions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	workflow := &model.Phase4Workflow{
		ID:               "p4wf_1",
		DatasetProfileID: "p4ds_1",
		Status:           model.Phase4WorkflowStatusAwaitingSelection,
		NextAction:       model.Phase4WorkflowNextActionSelectIdea,
	}
	completed := &model.Phase4WorkflowDetail{
		Workflow: &model.Phase4Workflow{
			ID:               "p4wf_1",
			DatasetProfileID: "p4ds_1",
			Status:           model.Phase4WorkflowStatusCompleted,
			NextAction:       model.Phase4WorkflowNextActionViewReport,
			SelectedIdeaID:   "p4idea_1",
			LatestReportID:   "p4report_1",
		},
		NextActions: []model.Phase4WorkflowNextAction{{Action: model.Phase4WorkflowNextActionViewReport}},
	}
	blocked := &model.Phase4WorkflowDetail{
		Workflow: &model.Phase4Workflow{
			ID:               "p4wf_1",
			DatasetProfileID: "p4ds_1",
			Status:           model.Phase4WorkflowStatusBlocked,
			NextAction:       model.Phase4WorkflowNextActionRetryStage,
		},
	}
	stub := &phase4WorkflowHandlerServiceStub{
		createResp: &model.Phase4WorkflowDetail{
			Workflow:    workflow,
			NextActions: []model.Phase4WorkflowNextAction{{Action: model.Phase4WorkflowNextActionSelectIdea}},
		},
		listResp: []model.Phase4Workflow{*workflow},
		getResp: &model.Phase4WorkflowDetail{
			Workflow: workflow,
			Timeline: []model.Phase4WorkflowAction{{WorkflowID: "p4wf_1", ActionType: "create"}},
		},
		selectIdeaResp:     blocked,
		selectRevisionResp: completed,
		retryResp:          completed,
		archiveResp: &model.Phase4WorkflowDetail{
			Workflow: &model.Phase4Workflow{
				ID:               "p4wf_1",
				DatasetProfileID: "p4ds_1",
				Status:           model.Phase4WorkflowStatusArchived,
				NextAction:       model.Phase4WorkflowNextActionNone,
			},
		},
	}

	h := NewPhase4WorkflowHandler(stub)
	router := gin.New()
	router.POST("/api/v2/phase4/workflows", h.Create)
	router.GET("/api/v2/phase4/workflows", h.List)
	router.GET("/api/v2/phase4/workflows/:id", h.Get)
	router.POST("/api/v2/phase4/workflows/:id/select-idea", h.SelectIdea)
	router.POST("/api/v2/phase4/workflows/:id/select-revision", h.SelectRevision)
	router.POST("/api/v2/phase4/workflows/:id/retry-stage", h.RetryStage)
	router.POST("/api/v2/phase4/workflows/:id/archive", h.Archive)

	createResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/workflows", model.Phase4WorkflowCreateRequest{
		DatasetProfileID: "p4ds_1",
	})
	if createResp.Code != 0 {
		t.Fatalf("expected create workflow success, got %+v", createResp)
	}
	var created model.Phase4WorkflowDetail
	mustDecodePhase4Data(t, createResp.Data, &created)
	if created.Workflow == nil || created.Workflow.ID != "p4wf_1" {
		t.Fatalf("unexpected created workflow: %#v", created)
	}

	listResp := doPhase4JSON(t, router, http.MethodGet, "/api/v2/phase4/workflows?datasetProfileId=p4ds_1", nil)
	if listResp.Code != 0 {
		t.Fatalf("expected list workflow success, got %+v", listResp)
	}
	var listed []model.Phase4Workflow
	mustDecodePhase4Data(t, listResp.Data, &listed)
	if len(listed) != 1 || listed[0].ID != "p4wf_1" {
		t.Fatalf("unexpected workflow list: %#v", listed)
	}

	getResp := doPhase4JSON(t, router, http.MethodGet, "/api/v2/phase4/workflows/p4wf_1", nil)
	if getResp.Code != 0 {
		t.Fatalf("expected get workflow success, got %+v", getResp)
	}
	var detail model.Phase4WorkflowDetail
	mustDecodePhase4Data(t, getResp.Data, &detail)
	if detail.Workflow == nil || len(detail.Timeline) != 1 {
		t.Fatalf("unexpected workflow detail: %#v", detail)
	}

	selectIdeaResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/workflows/p4wf_1/select-idea", model.Phase4WorkflowSelectIdeaRequest{
		IdeaID: "p4idea_1",
	})
	if selectIdeaResp.Code != 0 {
		t.Fatalf("expected select-idea success, got %+v", selectIdeaResp)
	}
	mustDecodePhase4Data(t, selectIdeaResp.Data, &detail)
	if detail.Workflow == nil || detail.Workflow.Status != model.Phase4WorkflowStatusBlocked {
		t.Fatalf("expected blocked workflow after select-idea stub, got %#v", detail)
	}

	selectRevisionResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/workflows/p4wf_1/select-revision", model.Phase4WorkflowSelectIdeaRequest{
		IdeaID: "p4idea_rev_1",
	})
	if selectRevisionResp.Code != 0 {
		t.Fatalf("expected select-revision success, got %+v", selectRevisionResp)
	}
	mustDecodePhase4Data(t, selectRevisionResp.Data, &detail)
	if detail.Workflow == nil || detail.Workflow.Status != model.Phase4WorkflowStatusCompleted {
		t.Fatalf("expected completed workflow after select-revision stub, got %#v", detail)
	}

	retryResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/workflows/p4wf_1/retry-stage", model.Phase4WorkflowRetryStageRequest{})
	if retryResp.Code != 0 {
		t.Fatalf("expected retry-stage success, got %+v", retryResp)
	}
	mustDecodePhase4Data(t, retryResp.Data, &detail)
	if detail.Workflow == nil || detail.Workflow.Status != model.Phase4WorkflowStatusCompleted {
		t.Fatalf("expected completed workflow after retry stub, got %#v", detail)
	}

	archiveResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/workflows/p4wf_1/archive", map[string]any{})
	if archiveResp.Code != 0 {
		t.Fatalf("expected archive success, got %+v", archiveResp)
	}
	mustDecodePhase4Data(t, archiveResp.Data, &detail)
	if detail.Workflow == nil || detail.Workflow.Status != model.Phase4WorkflowStatusArchived {
		t.Fatalf("expected archived workflow, got %#v", detail)
	}
}
