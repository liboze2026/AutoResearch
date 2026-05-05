package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type runHandlerService interface {
	StartRun(ctx context.Context, runID string) (*model.ExperimentRun, error)
	GetRun(ctx context.Context, runID string) (*model.ExperimentRun, error)
}

type runLogHandlerService interface {
	ListByRunID(ctx context.Context, runID string) ([]model.RunLog, error)
	Tail(ctx context.Context, runID string, logType string) (string, error)
}

type RunHandler struct {
	runs runHandlerService
	logs runLogHandlerService
}

func NewRunHandler(runs runHandlerService, logs runLogHandlerService) *RunHandler {
	return &RunHandler{runs: runs, logs: logs}
}

func (h *RunHandler) Start(c *gin.Context) {
	item, err := h.runs.StartRun(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, runStatus(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *RunHandler) Get(c *gin.Context) {
	item, err := h.runs.GetRun(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, runStatus(err), err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "run not found")
		return
	}
	httpx.OK(c, item)
}

func (h *RunHandler) ListLogs(c *gin.Context) {
	items, err := h.logs.ListByRunID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, runStatus(err), err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *RunHandler) TailLogs(c *gin.Context) {
	text, err := h.logs.Tail(c.Request.Context(), c.Param("id"), c.Query("type"))
	if err != nil {
		httpx.Error(c, runStatus(err), err.Error())
		return
	}
	httpx.OK(c, gin.H{"tail": text})
}

func runStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch err.Error() {
	case "run not found", "experiment not found", "assigned server not found":
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}
