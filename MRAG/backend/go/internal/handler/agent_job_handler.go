package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type agentJobHandlerService interface {
	Create(context.Context, model.AgentJobCreateRequest) (*model.AgentJob, error)
	GetByID(context.Context, string) (*model.AgentJob, error)
	GetStatus(context.Context, string) (*model.AgentJobStatusDetail, error)
}

type agentTriggerHandlerService interface {
	Trigger(context.Context, string, model.AgentJobTriggerRequest) (*model.AgentJob, error)
}

type agentArtifactHandlerService interface {
	ListByJobID(context.Context, string) ([]model.AgentArtifact, error)
}

type AgentJobHandler struct {
	jobs      agentJobHandlerService
	triggers  agentTriggerHandlerService
	artifacts agentArtifactHandlerService
}

func NewAgentJobHandler(jobs agentJobHandlerService, triggers agentTriggerHandlerService, artifacts agentArtifactHandlerService) *AgentJobHandler {
	return &AgentJobHandler{jobs: jobs, triggers: triggers, artifacts: artifacts}
}

func (h *AgentJobHandler) Create(c *gin.Context) {
	var req model.AgentJobCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.jobs.Create(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *AgentJobHandler) Get(c *gin.Context) {
	item, err := h.jobs.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, agentJobStatus(err), err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "agent job not found")
		return
	}
	httpx.OK(c, item)
}

func (h *AgentJobHandler) GetStatus(c *gin.Context) {
	item, err := h.jobs.GetStatus(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, agentJobStatus(err), err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "agent job not found")
		return
	}
	httpx.OK(c, item)
}

func (h *AgentJobHandler) Trigger(c *gin.Context) {
	var req model.AgentJobTriggerRequest
	if c.Request.ContentLength > 0 {
		if !httpx.MustBindJSON(c, &req) {
			return
		}
	}
	item, err := h.triggers.Trigger(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, agentJobStatus(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *AgentJobHandler) ListArtifacts(c *gin.Context) {
	items, err := h.artifacts.ListByJobID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, agentJobStatus(err), err.Error())
		return
	}
	httpx.OK(c, items)
}

func AgentSchemaStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch err.Error() {
	case "agent schema not found":
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}

func agentJobStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch err.Error() {
	case "agent job not found":
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}
