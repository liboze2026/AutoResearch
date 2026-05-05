package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/pkg/httpx"
	"mrag-platform/backend/go/internal/service"
)

type OverviewHandler struct {
	svc *service.OverviewService
}

func NewOverviewHandler(svc *service.OverviewService) *OverviewHandler { return &OverviewHandler{svc: svc} }

func (h *OverviewHandler) Stats(c *gin.Context) {
	item, err := h.svc.Stats(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, item)
}
