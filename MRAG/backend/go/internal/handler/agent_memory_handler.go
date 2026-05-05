package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type agentMemoryHandlerService interface {
	ListByAgentType(context.Context, string) ([]model.AgentMemoryRecord, error)
}

type AgentMemoryHandler struct {
	memory agentMemoryHandlerService
}

func NewAgentMemoryHandler(memory agentMemoryHandlerService) *AgentMemoryHandler {
	return &AgentMemoryHandler{memory: memory}
}

func (h *AgentMemoryHandler) ListByAgentType(c *gin.Context) {
	items, err := h.memory.ListByAgentType(c.Request.Context(), c.Param("type"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, items)
}
