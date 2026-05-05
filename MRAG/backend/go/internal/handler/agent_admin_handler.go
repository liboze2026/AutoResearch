package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type agentAdminHandlerService interface {
	ListAgents(context.Context) ([]model.AgentSummary, error)
	ListJobs(context.Context, int) ([]model.AgentJob, error)
	GetJob(context.Context, string) (*model.AgentJob, error)
	ListArtifacts(context.Context, string) ([]model.AgentArtifact, error)
	ListEvents(context.Context, int) ([]model.AgentEvent, error)
}

type AgentAdminHandler struct {
	agents agentAdminHandlerService
}

func NewAgentAdminHandler(agents agentAdminHandlerService) *AgentAdminHandler {
	return &AgentAdminHandler{agents: agents}
}

func (h *AgentAdminHandler) ListAgents(c *gin.Context) {
	items, err := h.agents.ListAgents(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *AgentAdminHandler) ListJobs(c *gin.Context) {
	items, err := h.agents.ListJobs(c.Request.Context(), parseAgentAdminLimit(c, 100))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *AgentAdminHandler) GetJob(c *gin.Context) {
	item, err := h.agents.GetJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "agent job not found")
		return
	}
	httpx.OK(c, item)
}

func (h *AgentAdminHandler) ListArtifacts(c *gin.Context) {
	items, err := h.agents.ListArtifacts(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *AgentAdminHandler) ListEvents(c *gin.Context) {
	items, err := h.agents.ListEvents(c.Request.Context(), parseAgentAdminLimit(c, 100))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, items)
}

func parseAgentAdminLimit(c *gin.Context, defaultLimit int) int {
	raw := c.Query("limit")
	if raw == "" {
		return defaultLimit
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultLimit
	}
	if value > 500 {
		return 500
	}
	return value
}
