package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	"mrag-platform/backend/go/internal/service"
)

type DatasetAssetHandler struct {
	svc *service.DatasetAssetService
}

func NewDatasetAssetHandler(svc *service.DatasetAssetService) *DatasetAssetHandler {
	return &DatasetAssetHandler{svc: svc}
}

func (h *DatasetAssetHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *DatasetAssetHandler) Create(c *gin.Context) {
	var req model.DatasetAssetCreateRequest
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

func (h *DatasetAssetHandler) RegisterFromScan(c *gin.Context) {
	var req model.DatasetAssetRegisterFromScanRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.RegisterFromScan(c.Request.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "dataset not found" || err.Error() == "scan record not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *DatasetAssetHandler) Get(c *gin.Context) {
	item, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "dataset asset not found")
		return
	}
	httpx.OK(c, item)
}
