package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type phase4WorkflowHandlerService interface {
	CreateWorkflow(context.Context, model.Phase4WorkflowCreateRequest) (*model.Phase4WorkflowDetail, error)
	ListWorkflows(context.Context, string, string) ([]model.Phase4Workflow, error)
	GetWorkflow(context.Context, string) (*model.Phase4WorkflowDetail, error)
	SelectIdea(context.Context, string, model.Phase4WorkflowSelectIdeaRequest) (*model.Phase4WorkflowDetail, error)
	SelectRevision(context.Context, string, model.Phase4WorkflowSelectIdeaRequest) (*model.Phase4WorkflowDetail, error)
	RetryStage(context.Context, string, model.Phase4WorkflowRetryStageRequest) (*model.Phase4WorkflowDetail, error)
	ArchiveWorkflow(context.Context, string) (*model.Phase4WorkflowDetail, error)
}

type Phase4WorkflowHandler struct {
	svc phase4WorkflowHandlerService
}

func NewPhase4WorkflowHandler(svc phase4WorkflowHandlerService) *Phase4WorkflowHandler {
	return &Phase4WorkflowHandler{svc: svc}
}

func (h *Phase4WorkflowHandler) Create(c *gin.Context) {
	var req model.Phase4WorkflowCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.CreateWorkflow(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4WorkflowHandler) List(c *gin.Context) {
	items, err := h.svc.ListWorkflows(c.Request.Context(), c.Query("datasetProfileId"), c.Query("status"))
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *Phase4WorkflowHandler) Get(c *gin.Context) {
	item, err := h.svc.GetWorkflow(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "phase4 workflow not found")
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4WorkflowHandler) SelectIdea(c *gin.Context) {
	var req model.Phase4WorkflowSelectIdeaRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.SelectIdea(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4WorkflowHandler) SelectRevision(c *gin.Context) {
	var req model.Phase4WorkflowSelectIdeaRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.SelectRevision(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4WorkflowHandler) RetryStage(c *gin.Context) {
	var req model.Phase4WorkflowRetryStageRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.RetryStage(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4WorkflowHandler) Archive(c *gin.Context) {
	item, err := h.svc.ArchiveWorkflow(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}
