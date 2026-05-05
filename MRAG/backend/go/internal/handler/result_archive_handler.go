package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	"mrag-platform/backend/go/internal/service"
)

type ResultArchiveHandler struct {
	svc *service.ResultArchiveService
}

func NewResultArchiveHandler(svc *service.ResultArchiveService) *ResultArchiveHandler {
	return &ResultArchiveHandler{svc: svc}
}

func (h *ResultArchiveHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *ResultArchiveHandler) Create(c *gin.Context) {
	var req model.ResultArchiveCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "dataset asset not found" || err.Error() == "idea not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *ResultArchiveHandler) Get(c *gin.Context) {
	item, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "result archive not found")
		return
	}
	httpx.OK(c, item)
}

func (h *ResultArchiveHandler) Update(c *gin.Context) {
	var req model.ResultArchiveUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "result archive not found" || err.Error() == "idea not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	httpx.OK(c, item)
}
