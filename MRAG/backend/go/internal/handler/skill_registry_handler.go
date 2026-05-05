package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type skillRegistryHandlerService interface {
	Register(context.Context, model.SkillRegisterRequest) (*model.SkillDefinition, error)
	List(context.Context) ([]model.SkillDefinition, error)
}

type SkillRegistryHandler struct {
	skills skillRegistryHandlerService
}

func NewSkillRegistryHandler(skills skillRegistryHandlerService) *SkillRegistryHandler {
	return &SkillRegistryHandler{skills: skills}
}

func (h *SkillRegistryHandler) List(c *gin.Context) {
	items, err := h.skills.List(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *SkillRegistryHandler) Register(c *gin.Context) {
	var req model.SkillRegisterRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.skills.Register(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, item)
}
