package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type ideaAgentHandlerService interface {
	Run(context.Context, model.IdeaGeneratorRunRequest) (*model.IdeaGeneratorRunResult, error)
}

type IdeaAgentHandler struct {
	svc ideaAgentHandlerService
}

func NewIdeaAgentHandler(svc ideaAgentHandlerService) *IdeaAgentHandler {
	return &IdeaAgentHandler{svc: svc}
}

func (h *IdeaAgentHandler) Run(c *gin.Context) {
	var req model.IdeaGeneratorRunRequest
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
