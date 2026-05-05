package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type phase4WriterHandlerService interface {
	Run(context.Context, model.Phase4WriterRunRequest) (*model.Phase4WriterRunResult, error)
	GetJob(context.Context, string) (*model.Phase4WriterJobDetail, error)
}

type Phase4WriterHandler struct {
	svc phase4WriterHandlerService
}

func NewPhase4WriterHandler(svc phase4WriterHandlerService) *Phase4WriterHandler {
	return &Phase4WriterHandler{svc: svc}
}

func (h *Phase4WriterHandler) Run(c *gin.Context) {
	var req model.Phase4WriterRunRequest
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

func (h *Phase4WriterHandler) GetJob(c *gin.Context) {
	result, err := h.svc.GetJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	if result == nil {
		httpx.Error(c, http.StatusNotFound, "phase4 writer job not found")
		return
	}
	httpx.OK(c, result)
}
