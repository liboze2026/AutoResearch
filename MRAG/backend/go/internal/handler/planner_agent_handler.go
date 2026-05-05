package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type plannerAgentHandlerService interface {
	Run(context.Context, model.PlannerRunRequest) (*model.PlannerRunResult, error)
	GetPlan(context.Context, string) (*model.ExperimentPlanResponse, error)
}

type PlannerAgentHandler struct {
	svc plannerAgentHandlerService
}

func NewPlannerAgentHandler(svc plannerAgentHandlerService) *PlannerAgentHandler {
	return &PlannerAgentHandler{svc: svc}
}

func (h *PlannerAgentHandler) Run(c *gin.Context) {
	var req model.PlannerRunRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	result, err := h.svc.Run(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *PlannerAgentHandler) GetPlan(c *gin.Context) {
	result, err := h.svc.GetPlan(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if result == nil {
		httpx.Error(c, http.StatusNotFound, "experiment plan not found")
		return
	}
	httpx.OK(c, result)
}
