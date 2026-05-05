package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type resultCompareHandlerService interface {
	ListByExperimentID(ctx context.Context, experimentID string) ([]model.ResultComparison, error)
	CompareRun(ctx context.Context, runID string) (*model.RunCompareResult, error)
}

type ResultCompareHandler struct {
	svc resultCompareHandlerService
}

func NewResultCompareHandler(svc resultCompareHandlerService) *ResultCompareHandler {
	return &ResultCompareHandler{svc: svc}
}

func (h *ResultCompareHandler) ListByExperiment(c *gin.Context) {
	items, err := h.svc.ListByExperimentID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *ResultCompareHandler) CompareRun(c *gin.Context) {
	item, err := h.svc.CompareRun(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, compareStatus(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func compareStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch err.Error() {
	case "run not found", "experiment not found":
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}
