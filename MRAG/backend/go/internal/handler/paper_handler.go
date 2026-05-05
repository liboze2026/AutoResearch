package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	"mrag-platform/backend/go/internal/service"
)

type PaperHandler struct {
	svc *service.PaperService
}

func NewPaperHandler(svc *service.PaperService) *PaperHandler { return &PaperHandler{svc: svc} }

func (h *PaperHandler) Import(c *gin.Context) {
	file, err := c.FormFile("file")
	if err == nil && file != nil {
		src, openErr := file.Open()
		if openErr != nil {
			httpx.Error(c, http.StatusBadRequest, openErr.Error())
			return
		}
		defer src.Close()
		result, importErr := h.svc.ImportUploadedFile(c.Request.Context(), file.Filename, src)
		if importErr != nil {
			httpx.Error(c, http.StatusBadRequest, importErr.Error())
			return
		}
		httpx.OK(c, result)
		return
	}

	var req model.PaperImportRequest
	if strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		if !httpx.MustBindJSON(c, &req) {
			return
		}
	} else {
		req.ExistingPath = c.PostForm("existingPath")
	}
	result, importErr := h.svc.ImportExistingFile(c.Request.Context(), req.ExistingPath)
	if importErr != nil {
		httpx.Error(c, http.StatusBadRequest, importErr.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *PaperHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, list)
}

func (h *PaperHandler) Get(c *gin.Context) {
	item, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "paper not found")
		return
	}
	httpx.OK(c, item)
}

func (h *PaperHandler) ListFiles(c *gin.Context) {
	files, err := h.svc.ListFiles(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "paper not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	httpx.OK(c, files)
}

func (h *PaperHandler) Parse(c *gin.Context) {
	result, err := h.svc.ParsePaper(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "paper not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *PaperHandler) ExtractInsights(c *gin.Context) {
	result, err := h.svc.ExtractInsights(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "paper not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *PaperHandler) ListInsights(c *gin.Context) {
	items, err := h.svc.ListInsights(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "paper not found" {
			status = http.StatusNotFound
		}
		httpx.Error(c, status, err.Error())
		return
	}
	httpx.OK(c, items)
}
