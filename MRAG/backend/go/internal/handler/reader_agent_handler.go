package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type readerAgentHandlerService interface {
	Run(context.Context, model.ReaderRunRequest) (*model.ReaderRunResult, error)
	GetJob(context.Context, string) (*model.ReaderJobDetail, error)
}

type ReaderAgentHandler struct {
	svc readerAgentHandlerService
}

func NewReaderAgentHandler(svc readerAgentHandlerService) *ReaderAgentHandler {
	return &ReaderAgentHandler{svc: svc}
}

func (h *ReaderAgentHandler) Run(c *gin.Context) {
	var req model.ReaderRunRequest
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

func (h *ReaderAgentHandler) GetJob(c *gin.Context) {
	result, err := h.svc.GetJob(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if result == nil {
		httpx.Error(c, http.StatusNotFound, "reader agent job not found")
		return
	}
	httpx.OK(c, result)
}
