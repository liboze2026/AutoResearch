package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type codingAgentHandlerService interface {
	Run(context.Context, model.CodingRunRequest) (*model.CodingRunResult, error)
}

type CodingAgentHandler struct {
	svc codingAgentHandlerService
}

func NewCodingAgentHandler(svc codingAgentHandlerService) *CodingAgentHandler {
	return &CodingAgentHandler{svc: svc}
}

func (h *CodingAgentHandler) Run(c *gin.Context) {
	var req model.CodingRunRequest
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

func (h *CodingAgentHandler) RunEvaluator(c *gin.Context) {
	h.Run(c)
}
