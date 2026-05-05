package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type phase4IdeaHandlerService interface {
	Run(context.Context, model.Phase4IdeaRunRequest) (*model.Phase4IdeaRunResult, error)
	GenerateRevisionCandidates(context.Context, string, model.Phase4IdeaRevisionGenerateRequest) (*model.Phase4IdeaRunResult, error)
	GetJob(context.Context, string) (*model.Phase4IdeaJobDetail, error)
}

type Phase4IdeaHandler struct {
	svc phase4IdeaHandlerService
}

func NewPhase4IdeaHandler(svc phase4IdeaHandlerService) *Phase4IdeaHandler {
	return &Phase4IdeaHandler{svc: svc}
}

func (h *Phase4IdeaHandler) Run(c *gin.Context) {
	var req model.Phase4IdeaRunRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	result, err := h.svc.Run(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *Phase4IdeaHandler) GenerateRevisionCandidates(c *gin.Context) {
	var req model.Phase4IdeaRevisionGenerateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	result, err := h.svc.GenerateRevisionCandidates(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *Phase4IdeaHandler) GetJob(c *gin.Context) {
	result, err := h.svc.GetJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	if result == nil {
		httpx.Error(c, http.StatusNotFound, "phase4 idea job not found")
		return
	}
	httpx.OK(c, result)
}
