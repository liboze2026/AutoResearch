package gpuresource

import (
	"context"
	"fmt"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type gpuProbeReader interface {
	Get(ctx context.Context, id string) (*model.Server, error)
	CheckGPU(ctx context.Context, id string) (*model.GPUProbeResult, error)
}

type snapshotRepository interface {
	Create(ctx context.Context, item model.GPUResourceSnapshot) error
	ListByServerID(ctx context.Context, serverID string, limit int) ([]model.GPUResourceSnapshot, error)
}

type CaptureResult struct {
	ServerID          string                      `json:"serverId"`
	CapturedAt        time.Time                   `json:"capturedAt"`
	AvailableGPUCount int                         `json:"availableGpuCount"`
	TotalGPUCount     int                         `json:"totalGpuCount"`
	Snapshots         []model.GPUResourceSnapshot `json:"snapshots"`
	Probe             *model.GPUProbeResult       `json:"probe,omitempty"`
}

type Service struct {
	servers gpuProbeReader
	repo    snapshotRepository
}

func NewService(servers gpuProbeReader, repo snapshotRepository) *Service {
	return &Service{servers: servers, repo: repo}
}

func (s *Service) Collect(ctx context.Context, serverID string) (*CaptureResult, error) {
	node, err := s.servers.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, fmt.Errorf("server not found")
	}

	probe, err := s.servers.CheckGPU(ctx, serverID)
	if err != nil {
		return nil, err
	}

	capturedAt := probe.CheckedAt
	if capturedAt.IsZero() {
		capturedAt = time.Now()
	}

	snapshots := make([]model.GPUResourceSnapshot, 0, len(probe.Devices))
	for _, device := range probe.Devices {
		freeMem := int(device.MemoryTotalMB - device.MemoryUsedMB)
		if freeMem < 0 {
			freeMem = 0
		}
		now := time.Now()
		snapshot := model.GPUResourceSnapshot{
			ID:          httpx.NewID("gsnap"),
			ServerID:    serverID,
			CapturedAt:  capturedAt,
			GPUIndex:    device.Index,
			Name:        device.Name,
			TotalMemMB:  int(device.MemoryTotalMB),
			FreeMemMB:   freeMem,
			Utilization: int(device.Utilization),
			ProcessJSON: []map[string]interface{}{{"processCount": device.Processes}},
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := s.repo.Create(ctx, snapshot); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	return &CaptureResult{
		ServerID:          serverID,
		CapturedAt:        capturedAt,
		AvailableGPUCount: probe.AvailableGPUCount,
		TotalGPUCount:     probe.TotalGPUCount,
		Snapshots:         snapshots,
		Probe:             probe,
	}, nil
}

func (s *Service) ListByServerID(ctx context.Context, serverID string, limit int) ([]model.GPUResourceSnapshot, error) {
	return s.repo.ListByServerID(ctx, serverID, limit)
}
