package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type experimentHandlerService interface {
	List(ctx context.Context) ([]model.Experiment, error)
	GetByID(ctx context.Context, id string) (*model.ExperimentDetail, error)
	Create(ctx context.Context, req model.ExperimentCreateRequest) (*model.ExperimentDetail, error)
	GenerateSpec(ctx context.Context, experimentID string) (*model.ExperimentSpecDetail, error)
	GetLatestSpec(ctx context.Context, experimentID string) (*model.ExperimentSpecDetail, error)
}

type ExperimentHandler struct {
	svc experimentHandlerService
}

func NewExperimentHandler(svc experimentHandlerService) *ExperimentHandler {
	return &ExperimentHandler{svc: svc}
}

func (h *ExperimentHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *ExperimentHandler) Get(c *gin.Context) {
	item, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, experimentStatus(err), err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "experiment not found")
		return
	}
	httpx.OK(c, item)
}

func (h *ExperimentHandler) Create(c *gin.Context) {
	var req model.ExperimentCreateRequest
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

func (h *ExperimentHandler) GenerateSpec(c *gin.Context) {
	item, err := h.svc.GenerateSpec(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, experimentStatus(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *ExperimentHandler) GetSpec(c *gin.Context) {
	item, err := h.svc.GetLatestSpec(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, experimentStatus(err), err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "experiment spec not found")
		return
	}
	httpx.OK(c, item)
}

func experimentStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch err.Error() {
	case "experiment not found":
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}
