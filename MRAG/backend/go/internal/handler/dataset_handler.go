package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	"mrag-platform/backend/go/internal/service"
)

type DatasetHandler struct {
	svc *service.DatasetService
}

func NewDatasetHandler(svc *service.DatasetService) *DatasetHandler { return &DatasetHandler{svc: svc} }

func (h *DatasetHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), c.Query("keyword"), c.Query("sourceType"), c.Query("modality"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, list)
}

func (h *DatasetHandler) Get(c *gin.Context) {
	item, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "dataset not found")
		return
	}
	httpx.OK(c, item)
}

func (h *DatasetHandler) ValidatePath(c *gin.Context) {
	var req model.DatasetPathValidationRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.ValidatePath(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *DatasetHandler) Create(c *gin.Context) {
	var req model.DatasetImportRequest
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

func (h *DatasetHandler) Update(c *gin.Context) {
	var req model.DatasetUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "dataset not found")
		return
	}
	httpx.OK(c, item)
}

func (h *DatasetHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

func (h *DatasetHandler) BuildIndex(c *gin.Context) {
	item, err := h.svc.BuildIndex(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *DatasetHandler) SyncIndexTask(c *gin.Context) {
	item, err := h.svc.SyncIndexTask(c.Request.Context(), c.Param("id"), c.Param("taskId"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, item)
}
