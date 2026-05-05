package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type agentSchemaHandlerService interface {
	Create(context.Context, model.AgentSchemaCreateRequest) (*model.AgentSchema, error)
	GetByID(context.Context, string) (*model.AgentSchema, error)
	List(context.Context) ([]model.AgentSchema, error)
}

type AgentSchemaHandler struct {
	schemas agentSchemaHandlerService
}

func NewAgentSchemaHandler(schemas agentSchemaHandlerService) *AgentSchemaHandler {
	return &AgentSchemaHandler{schemas: schemas}
}

func (h *AgentSchemaHandler) Create(c *gin.Context) {
	var req model.AgentSchemaCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.schemas.Create(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *AgentSchemaHandler) Get(c *gin.Context) {
	item, err := h.schemas.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "agent schema not found")
		return
	}
	httpx.OK(c, item)
}

func (h *AgentSchemaHandler) List(c *gin.Context) {
	items, err := h.schemas.List(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, items)
}
