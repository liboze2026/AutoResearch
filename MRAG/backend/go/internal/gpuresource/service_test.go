package gpuresource

import (
	"context"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type fakeGPUProbeServer struct {
	server *model.Server
	probe  *model.GPUProbeResult
}

func (f *fakeGPUProbeServer) Get(_ context.Context, id string) (*model.Server, error) {
	if f.server == nil || f.server.ID != id {
		return nil, nil
	}
	return f.server, nil
}

func (f *fakeGPUProbeServer) CheckGPU(_ context.Context, _ string) (*model.GPUProbeResult, error) {
	return f.probe, nil
}

type memorySnapshotRepo struct {
	items []model.GPUResourceSnapshot
}

func (r *memorySnapshotRepo) Create(_ context.Context, item model.GPUResourceSnapshot) error {
	r.items = append(r.items, item)
	return nil
}

func (r *memorySnapshotRepo) ListByServerID(_ context.Context, serverID string, _ int) ([]model.GPUResourceSnapshot, error) {
	items := make([]model.GPUResourceSnapshot, 0)
	for _, item := range r.items {
		if item.ServerID == serverID {
			items = append(items, item)
		}
	}
	return items, nil
}

func TestServiceCollectStoresSnapshots(t *testing.T) {
	repo := &memorySnapshotRepo{}
	svc := NewService(&fakeGPUProbeServer{
		server: &model.Server{ID: "srv_1"},
		probe: &model.GPUProbeResult{
			ServerID:          "srv_1",
			CheckedAt:         time.Now(),
			AvailableGPUCount: 1,
			TotalGPUCount:     1,
			Devices: []model.GPUDeviceStatus{{
				Index:         0,
				Name:          "RTX 4090",
				MemoryUsedMB:  1024,
				MemoryTotalMB: 24576,
				Utilization:   12,
				Processes:     1,
				Available:     true,
			}},
		},
	}, repo)

	result, err := svc.Collect(context.Background(), "srv_1")
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(result.Snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(result.Snapshots))
	}
	if repo.items[0].FreeMemMB != 23552 {
		t.Fatalf("expected free mem 23552, got %d", repo.items[0].FreeMemMB)
	}
}
