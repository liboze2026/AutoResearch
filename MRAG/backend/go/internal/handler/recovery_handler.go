package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type recoveryHandlerService interface {
	Retry(ctx context.Context, runID string) (*model.ExperimentQueueResult, error)
	GetRecovery(ctx context.Context, runID string) (*model.RunRecoveryDetail, error)
}

type RecoveryHandler struct {
	svc recoveryHandlerService
}

func NewRecoveryHandler(svc recoveryHandlerService) *RecoveryHandler {
	return &RecoveryHandler{svc: svc}
}

func (h *RecoveryHandler) Retry(c *gin.Context) {
	item, err := h.svc.Retry(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, recoveryStatus(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *RecoveryHandler) GetRecovery(c *gin.Context) {
	item, err := h.svc.GetRecovery(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, recoveryStatus(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func recoveryStatus(err error) int {
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
