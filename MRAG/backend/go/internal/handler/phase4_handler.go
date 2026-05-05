package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	"mrag-platform/backend/go/internal/service"
)

type Phase4Handler struct {
	svc *service.Phase4Service
}

func NewPhase4Handler(svc *service.Phase4Service) *Phase4Handler {
	return &Phase4Handler{svc: svc}
}

func (h *Phase4Handler) ListDatasetProfiles(c *gin.Context) {
	items, err := h.svc.ListDatasetProfiles(c.Request.Context(), c.Query("taskType"), c.Query("status"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *Phase4Handler) CreateDatasetProfile(c *gin.Context) {
	var req model.Phase4DatasetProfileCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.CreateDatasetProfile(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) GetDatasetProfile(c *gin.Context) {
	item, err := h.svc.GetDatasetProfileByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "phase4 dataset profile not found")
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) UpdateDatasetProfile(c *gin.Context) {
	var req model.Phase4DatasetProfileUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.UpdateDatasetProfile(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) DeleteDatasetProfile(c *gin.Context) {
	if err := h.svc.DeleteDatasetProfile(c.Request.Context(), c.Param("id")); err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

func (h *Phase4Handler) ListReaderSources(c *gin.Context) {
	items, err := h.svc.ListReaderSources(c.Request.Context(), c.Query("datasetProfileId"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *Phase4Handler) CreateReaderSource(c *gin.Context) {
	var req model.Phase4ReaderSourceCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.CreateReaderSource(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) GetReaderSource(c *gin.Context) {
	item, err := h.svc.GetReaderSourceByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "phase4 reader source not found")
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) UpdateReaderSource(c *gin.Context) {
	var req model.Phase4ReaderSourceUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.UpdateReaderSource(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) ListReaderContexts(c *gin.Context) {
	items, err := h.svc.ListReaderContexts(c.Request.Context(), c.Query("datasetProfileId"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *Phase4Handler) CreateReaderContext(c *gin.Context) {
	var req model.Phase4ReaderContextCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.CreateReaderContext(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) GetReaderContext(c *gin.Context) {
	item, err := h.svc.GetReaderContextByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "phase4 reader context not found")
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) UpdateReaderContext(c *gin.Context) {
	var req model.Phase4ReaderContextUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.UpdateReaderContext(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) ListIdeas(c *gin.Context) {
	items, err := h.svc.ListIdeas(c.Request.Context(), c.Query("datasetProfileId"), c.Query("status"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *Phase4Handler) CreateIdea(c *gin.Context) {
	var req model.Phase4IdeaCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.CreateIdea(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) GetIdea(c *gin.Context) {
	item, err := h.svc.GetIdeaByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "phase4 idea not found")
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) UpdateIdea(c *gin.Context) {
	var req model.Phase4IdeaUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.UpdateIdea(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) DeleteIdea(c *gin.Context) {
	if err := h.svc.DeleteIdea(c.Request.Context(), c.Param("id")); err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

func (h *Phase4Handler) UpdateIdeaStatus(c *gin.Context) {
	var req model.Phase4IdeaStatusUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.UpdateIdeaStatus(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) SelectIdea(c *gin.Context) {
	item, err := h.svc.SelectIdea(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) ArchiveIdea(c *gin.Context) {
	item, err := h.svc.ArchiveIdea(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) RejectIdea(c *gin.Context) {
	item, err := h.svc.RejectIdea(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) ListIdeaScoreViews(c *gin.Context) {
	items, err := h.svc.ListIdeaScoreViews(c.Request.Context(), c.Query("datasetProfileId"), c.Query("status"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *Phase4Handler) GetIdeaScoreView(c *gin.Context) {
	item, err := h.svc.GetIdeaScoreView(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) ListRunManifests(c *gin.Context) {
	items, err := h.svc.ListRunManifests(c.Request.Context(), c.Query("datasetProfileId"), c.Query("ideaId"), c.Query("status"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *Phase4Handler) CreateRunManifest(c *gin.Context) {
	var req model.Phase4RunManifestCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.CreateRunManifest(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) GetRunManifest(c *gin.Context) {
	item, err := h.svc.GetRunManifestByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "phase4 run manifest not found")
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) UpdateRunManifest(c *gin.Context) {
	var req model.Phase4RunManifestUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.UpdateRunManifest(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) UpdateRunManifestStatus(c *gin.Context) {
	var req model.Phase4RunManifestStatusUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.UpdateRunManifestStatus(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) ListStructuredReports(c *gin.Context) {
	items, err := h.svc.ListStructuredReports(c.Request.Context(), c.Query("runManifestId"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *Phase4Handler) CreateStructuredReport(c *gin.Context) {
	var req model.Phase4StructuredReportCreateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.CreateStructuredReport(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) GetStructuredReport(c *gin.Context) {
	item, err := h.svc.GetStructuredReportByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "phase4 structured report not found")
		return
	}
	httpx.OK(c, item)
}

func (h *Phase4Handler) UpdateStructuredReport(c *gin.Context) {
	var req model.Phase4StructuredReportUpdateRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.UpdateStructuredReport(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, phase4Status(err), err.Error())
		return
	}
	httpx.OK(c, item)
}

func phase4Status(err error) int {
	if err == nil {
		return http.StatusOK
	}
	switch err.Error() {
	case "phase4 dataset profile not found", "phase4 reader source not found", "phase4 reader context not found", "phase4 idea not found", "phase4 run manifest not found", "phase4 structured report not found", "phase4 workflow not found":
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}
