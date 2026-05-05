package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type writerAgentHandlerService interface {
	Run(context.Context, model.WriterRunRequest) (*model.WriterRunResult, error)
	GetDraft(context.Context, string) (*model.DraftDocument, error)
}

type WriterAgentHandler struct {
	svc writerAgentHandlerService
}

func NewWriterAgentHandler(svc writerAgentHandlerService) *WriterAgentHandler {
	return &WriterAgentHandler{svc: svc}
}

func (h *WriterAgentHandler) Run(c *gin.Context) {
	var req model.WriterRunRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	result, err := h.svc.Run(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *WriterAgentHandler) GetDraft(c *gin.Context) {
	result, err := h.svc.GetDraft(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if result == nil {
		httpx.Error(c, http.StatusNotFound, "draft not found")
		return
	}
	httpx.OK(c, result)
}
