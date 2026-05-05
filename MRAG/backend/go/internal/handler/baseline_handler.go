package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	"mrag-platform/backend/go/internal/service"
)

type BaselineHandler struct {
	svc *service.BaselineService
}

func NewBaselineHandler(svc *service.BaselineService) *BaselineHandler {
	return &BaselineHandler{svc: svc}
}

func (h *BaselineHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *BaselineHandler) Create(c *gin.Context) {
	var req model.BaselineCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "dataset asset not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *BaselineHandler) Get(c *gin.Context) {
	item, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "dataset asset not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "baseline not found")
		return
	}
	httpx.OK(c, item)
}

func (h *BaselineHandler) Update(c *gin.Context) {
	var req model.BaselineUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "baseline not found" || err.Error() == "dataset asset not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	httpx.OK(c, item)
}
