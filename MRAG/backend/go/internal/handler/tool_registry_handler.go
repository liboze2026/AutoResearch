package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type toolRegistryHandlerService interface {
	Register(context.Context, model.ToolRegisterRequest) (*model.ToolDefinition, error)
	List(context.Context) ([]model.ToolDefinition, error)
}

type ToolRegistryHandler struct {
	tools toolRegistryHandlerService
}

func NewToolRegistryHandler(tools toolRegistryHandlerService) *ToolRegistryHandler {
	return &ToolRegistryHandler{tools: tools}
}

func (h *ToolRegistryHandler) List(c *gin.Context) {
	items, err := h.tools.List(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *ToolRegistryHandler) Register(c *gin.Context) {
	var req model.ToolRegisterRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.tools.Register(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, item)
}
