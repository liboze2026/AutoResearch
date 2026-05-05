package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type datasetAgentHandlerService interface {
	Run(context.Context, model.DatasetRunRequest) (*model.DatasetRunResult, error)
	GetEvalPlan(context.Context, string) (*model.DatasetEvalPlanResponse, error)
}

type DatasetAgentHandler struct {
	svc datasetAgentHandlerService
}

func NewDatasetAgentHandler(svc datasetAgentHandlerService) *DatasetAgentHandler {
	return &DatasetAgentHandler{svc: svc}
}

func (h *DatasetAgentHandler) Run(c *gin.Context) {
	var req model.DatasetRunRequest
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

func (h *DatasetAgentHandler) GetEvalPlan(c *gin.Context) {
	result, err := h.svc.GetEvalPlan(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if result == nil {
		httpx.Error(c, http.StatusNotFound, "dataset eval plan not found")
		return
	}
	httpx.OK(c, result)
}
