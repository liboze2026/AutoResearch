package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type phase4ReaderHandlerService interface {
	Run(context.Context, model.Phase4ReaderRunRequest) (*model.Phase4ReaderRunResult, error)
	GetJob(context.Context, string) (*model.Phase4ReaderJobDetail, error)
}

type Phase4ReaderHandler struct {
	svc phase4ReaderHandlerService
}

func NewPhase4ReaderHandler(svc phase4ReaderHandlerService) *Phase4ReaderHandler {
	return &Phase4ReaderHandler{svc: svc}
}

func (h *Phase4ReaderHandler) Run(c *gin.Context) {
	var req model.Phase4ReaderRunRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	result, err := h.svc.Run(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *Phase4ReaderHandler) GetJob(c *gin.Context) {
	result, err := h.svc.GetJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	if result == nil {
		httpx.Error(c, http.StatusNotFound, "phase4 reader job not found")
		return
	}
	httpx.OK(c, result)
}
