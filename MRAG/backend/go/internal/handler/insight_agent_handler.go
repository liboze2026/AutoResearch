package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type insightAgentHandlerService interface {
	Run(context.Context, model.InsightRunRequest) (*model.InsightRunResult, error)
}

type InsightAgentHandler struct {
	svc insightAgentHandlerService
}

func NewInsightAgentHandler(svc insightAgentHandlerService) *InsightAgentHandler {
	return &InsightAgentHandler{svc: svc}
}

func (h *InsightAgentHandler) Run(c *gin.Context) {
	var req model.InsightRunRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	result, err := h.svc.Run(c.Request.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "paper not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	httpx.OK(c, result)
}
