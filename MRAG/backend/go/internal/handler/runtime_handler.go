package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/pkg/httpx"
	"mrag-platform/backend/go/internal/service"
)

type RuntimeHandler struct {
	svc *service.RuntimeProfileService
}

func NewRuntimeHandler(svc *service.RuntimeProfileService) *RuntimeHandler {
	return &RuntimeHandler{svc: svc}
}

func (h *RuntimeHandler) Profile(c *gin.Context) {
	item, err := h.svc.Profile(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, item)
}
