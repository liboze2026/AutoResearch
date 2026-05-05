package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	"mrag-platform/backend/go/internal/service"
)

type IdeaHandler struct {
	svc *service.IdeaService
}

func NewIdeaHandler(svc *service.IdeaService) *IdeaHandler { return &IdeaHandler{svc: svc} }

func (h *IdeaHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *IdeaHandler) Create(c *gin.Context) {
	var req model.IdeaCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *IdeaHandler) Get(c *gin.Context) {
	item, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "idea not found")
		return
	}
	httpx.OK(c, item)
}

func (h *IdeaHandler) Update(c *gin.Context) {
	var req model.IdeaUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "idea not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *IdeaHandler) GenerateFromPaper(c *gin.Context) {
	result, err := h.svc.GenerateFromPaper(c.Request.Context(), c.Param("paperId"))
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
