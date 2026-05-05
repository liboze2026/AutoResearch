package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type schedulerHandlerService interface {
	QueueExperiment(ctx context.Context, experimentID string) (*model.ExperimentQueueResult, error)
	ScheduleRun(ctx context.Context, runID string) (*model.ScheduleResult, error)
	GetLatestDecision(ctx context.Context, runID string) (*model.SchedulerDecision, error)
}

type SchedulerHandler struct {
	svc schedulerHandlerService
}

func NewSchedulerHandler(svc schedulerHandlerService) *SchedulerHandler {
	return &SchedulerHandler{svc: svc}
}

func (h *SchedulerHandler) QueueExperiment(c *gin.Context) {
	result, err := h.svc.QueueExperiment(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, schedulerStatus(err), err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *SchedulerHandler) ScheduleRun(c *gin.Context) {
	result, err := h.svc.ScheduleRun(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, schedulerStatus(err), err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *SchedulerHandler) GetDecision(c *gin.Context) {
	result, err := h.svc.GetLatestDecision(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, schedulerStatus(err), err.Error())
		return
	}
	if result == nil {
		httpx.Error(c, http.StatusNotFound, "scheduler decision not found")
		return
	}
	httpx.OK(c, result)
}

func schedulerStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch err.Error() {
	case "experiment not found", "run not found":
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}
