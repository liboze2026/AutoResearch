package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/gpuresource"
	"mrag-platform/backend/go/internal/heartbeat"
	"mrag-platform/backend/go/internal/model"
)

type fakeServerHandlerService struct{}

func (f *fakeServerHandlerService) List(context.Context) ([]model.Server, error) {
	return []model.Server{{ID: "srv_1", Name: "demo"}}, nil
}
func (f *fakeServerHandlerService) Create(context.Context, model.Server) (*model.Server, error) {
	return &model.Server{ID: "srv_1"}, nil
}
func (f *fakeServerHandlerService) Update(context.Context, string, model.Server) (*model.Server, error) {
	return &model.Server{ID: "srv_1"}, nil
}
func (f *fakeServerHandlerService) Delete(context.Context, string) error { return nil }
func (f *fakeServerHandlerService) TestConnection(context.Context, string) (*model.ServerConnectionTestResult, error) {
	return &model.ServerConnectionTestResult{}, nil
}
func (f *fakeServerHandlerService) RefreshStatus(context.Context, string) (*model.ServerStatusSnapshot, error) {
	return &model.ServerStatusSnapshot{}, nil
}
func (f *fakeServerHandlerService) CheckGPU(context.Context, string) (*model.GPUProbeResult, error) {
	return &model.GPUProbeResult{}, nil
}
func (f *fakeServerHandlerService) ScanDatasets(context.Context, string, model.ServerDatasetScanRequest) (*model.ServerDatasetScanResult, error) {
	return &model.ServerDatasetScanResult{}, nil
}

type fakeHeartbeatHandlerService struct{}

func (f *fakeHeartbeatHandlerService) Collect(context.Context, string) (*heartbeat.CaptureResult, error) {
	return &heartbeat.CaptureResult{}, nil
}
func (f *fakeHeartbeatHandlerService) ListByServerID(context.Context, string, int) ([]model.ServerHeartbeat, error) {
	return []model.ServerHeartbeat{{
		ID:          "shb_1",
		ServerID:    "srv_1",
		HeartbeatAt: time.Now(),
		Status:      "online",
		DetailJSON:  map[string]interface{}{"reachable": true},
	}}, nil
}

type fakeGPUSnapshotHandlerService struct{}

func (f *fakeGPUSnapshotHandlerService) Collect(context.Context, string) (*gpuresource.CaptureResult, error) {
	return &gpuresource.CaptureResult{}, nil
}
func (f *fakeGPUSnapshotHandlerService) ListByServerID(context.Context, string, int) ([]model.GPUResourceSnapshot, error) {
	return []model.GPUResourceSnapshot{{
		ID:          "gsnap_1",
		ServerID:    "srv_1",
		CapturedAt:  time.Now(),
		GPUIndex:    0,
		Name:        "RTX 4090",
		TotalMemMB:  24576,
		FreeMemMB:   23552,
		Utilization: 12,
	}}, nil
}

func TestServerHandlerListHeartbeatAndGPUSnapshots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewServerHandler(&fakeServerHandlerService{}, &fakeHeartbeatHandlerService{}, &fakeGPUSnapshotHandlerService{})

	router := gin.New()
	router.GET("/api/servers/:id/heartbeats", handler.ListHeartbeats)
	router.GET("/api/servers/:id/gpu-snapshots", handler.ListGPUSnapshots)

	req := httptest.NewRequest(http.MethodGet, "/api/servers/srv_1/heartbeats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected heartbeats 200, got %d", rec.Code)
	}

	var heartbeatResp struct {
		Code int                     `json:"code"`
		Data []model.ServerHeartbeat `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &heartbeatResp); err != nil {
		t.Fatalf("unmarshal heartbeat response: %v", err)
	}
	if len(heartbeatResp.Data) != 1 {
		t.Fatalf("expected 1 heartbeat, got %d", len(heartbeatResp.Data))
	}

	req = httptest.NewRequest(http.MethodGet, "/api/servers/srv_1/gpu-snapshots", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected snapshots 200, got %d", rec.Code)
	}

	var snapshotResp struct {
		Code int                         `json:"code"`
		Data []model.GPUResourceSnapshot `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshotResp); err != nil {
		t.Fatalf("unmarshal snapshot response: %v", err)
	}
	if len(snapshotResp.Data) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshotResp.Data))
	}
}
