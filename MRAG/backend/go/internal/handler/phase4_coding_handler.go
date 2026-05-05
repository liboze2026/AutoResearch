package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type phase4CodingHandlerService interface {
	Run(context.Context, model.Phase4CodingRunRequest) (*model.Phase4CodingRunResult, error)
	GetJob(context.Context, string) (*model.Phase4CodingJobDetail, error)
}

type Phase4CodingHandler struct {
	svc phase4CodingHandlerService
}

func NewPhase4CodingHandler(svc phase4CodingHandlerService) *Phase4CodingHandler {
	return &Phase4CodingHandler{svc: svc}
}

func (h *Phase4CodingHandler) Run(c *gin.Context) {
	var req model.Phase4CodingRunRequest
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

func (h *Phase4CodingHandler) GetJob(c *gin.Context) {
	result, err := h.svc.GetJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	if result == nil {
		httpx.Error(c, http.StatusNotFound, "phase4 coding job not found")
		return
	}
	httpx.OK(c, result)
}
