package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/gpuresource"
	"mrag-platform/backend/go/internal/heartbeat"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type serverHandlerService interface {
	List(ctx context.Context) ([]model.Server, error)
	Create(ctx context.Context, req model.Server) (*model.Server, error)
	Update(ctx context.Context, id string, req model.Server) (*model.Server, error)
	Delete(ctx context.Context, id string) error
	TestConnection(ctx context.Context, id string) (*model.ServerConnectionTestResult, error)
	RefreshStatus(ctx context.Context, id string) (*model.ServerStatusSnapshot, error)
	CheckGPU(ctx context.Context, id string) (*model.GPUProbeResult, error)
	ScanDatasets(ctx context.Context, id string, req model.ServerDatasetScanRequest) (*model.ServerDatasetScanResult, error)
}

type heartbeatHandlerService interface {
	Collect(ctx context.Context, serverID string) (*heartbeat.CaptureResult, error)
	ListByServerID(ctx context.Context, serverID string, limit int) ([]model.ServerHeartbeat, error)
}

type gpuSnapshotHandlerService interface {
	Collect(ctx context.Context, serverID string) (*gpuresource.CaptureResult, error)
	ListByServerID(ctx context.Context, serverID string, limit int) ([]model.GPUResourceSnapshot, error)
}

type ServerHandler struct {
	svc            serverHandlerService
	heartbeatSvc   heartbeatHandlerService
	gpuSnapshotSvc gpuSnapshotHandlerService
}

func NewServerHandler(svc serverHandlerService, heartbeatSvc heartbeatHandlerService, gpuSnapshotSvc gpuSnapshotHandlerService) *ServerHandler {
	return &ServerHandler{
		svc:            svc,
		heartbeatSvc:   heartbeatSvc,
		gpuSnapshotSvc: gpuSnapshotSvc,
	}
}

func (h *ServerHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, list)
}

func (h *ServerHandler) Create(c *gin.Context) {
	var req model.Server
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *ServerHandler) Update(c *gin.Context) {
	var req model.Server
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	item, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if item == nil {
		httpx.Error(c, http.StatusNotFound, "server not found")
		return
	}
	httpx.OK(c, item)
}

func (h *ServerHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		httpx.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

func (h *ServerHandler) TestConnection(c *gin.Context) {
	result, err := h.svc.TestConnection(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, serverStatus(err), err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *ServerHandler) RefreshStatus(c *gin.Context) {
	result, err := h.svc.RefreshStatus(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, serverStatus(err), err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *ServerHandler) CheckGPU(c *gin.Context) {
	result, err := h.svc.CheckGPU(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, serverStatus(err), err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *ServerHandler) Heartbeat(c *gin.Context) {
	result, err := h.heartbeatSvc.Collect(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, serverStatus(err), err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *ServerHandler) ListHeartbeats(c *gin.Context) {
	items, err := h.heartbeatSvc.ListByServerID(c.Request.Context(), c.Param("id"), parseLimit(c.Query("limit")))
	if err != nil {
		httpx.Error(c, serverStatus(err), err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *ServerHandler) GPUSnapshot(c *gin.Context) {
	result, err := h.gpuSnapshotSvc.Collect(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Error(c, serverStatus(err), err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *ServerHandler) ListGPUSnapshots(c *gin.Context) {
	items, err := h.gpuSnapshotSvc.ListByServerID(c.Request.Context(), c.Param("id"), parseLimit(c.Query("limit")))
	if err != nil {
		httpx.Error(c, serverStatus(err), err.Error())
		return
	}
	httpx.OK(c, items)
}

func (h *ServerHandler) ScanDatasets(c *gin.Context) {
	var req model.ServerDatasetScanRequest
	if !httpx.MustBindJSON(c, &req) {
		return
	}
	result, err := h.svc.ScanDatasets(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		httpx.Error(c, serverStatus(err), err.Error())
		return
	}
	httpx.OK(c, result)
}

func parseLimit(raw string) int {
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 20
	}
	return limit
}

func serverStatus(err error) int {
	if err != nil && err.Error() == "server not found" {
		return http.StatusNotFound
	}
	return http.StatusBadGateway
}
