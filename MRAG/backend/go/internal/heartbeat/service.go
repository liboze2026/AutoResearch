package heartbeat

import (
	"context"
	"fmt"
	"log"
	"time"

	"mrag-platform/backend/go/internal/gpuresource"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type serverHeartbeatReader interface {
	Get(ctx context.Context, id string) (*model.Server, error)
	TestConnection(ctx context.Context, id string) (*model.ServerConnectionTestResult, error)
}

type heartbeatRepository interface {
	Create(ctx context.Context, item model.ServerHeartbeat) error
	ListByServerID(ctx context.Context, serverID string, limit int) ([]model.ServerHeartbeat, error)
}

type serverInventory interface {
	List(ctx context.Context) ([]model.Server, error)
}

type heartbeatCollector interface {
	Collect(ctx context.Context, serverID string) (*CaptureResult, error)
}

type gpuSnapshotCollector interface {
	Collect(ctx context.Context, serverID string) (*gpuresource.CaptureResult, error)
}

type CaptureResult struct {
	Heartbeat *model.ServerHeartbeat            `json:"heartbeat"`
	Probe     *model.ServerConnectionTestResult `json:"probe,omitempty"`
}

type Service struct {
	servers serverHeartbeatReader
	repo    heartbeatRepository
}

func NewService(servers serverHeartbeatReader, repo heartbeatRepository) *Service {
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

	now := time.Now()
	heartbeat := model.ServerHeartbeat{
		ID:          httpx.NewID("shb"),
		ServerID:    serverID,
		HeartbeatAt: now,
		Status:      "offline",
		DetailJSON: map[string]interface{}{
			"serverName": node.Name,
			"host":       node.Host,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	probe, probeErr := s.servers.TestConnection(ctx, serverID)
	if probe != nil {
		heartbeat.HeartbeatAt = probe.CheckedAt
		if probe.Reachable {
			heartbeat.Status = "online"
		}
		heartbeat.DetailJSON = map[string]interface{}{
			"serverName":  node.Name,
			"host":        node.Host,
			"target":      probe.Target,
			"result":      probe.Result,
			"reachable":   probe.Reachable,
			"message":     probe.Message,
			"remoteHost":  probe.RemoteHost,
			"remoteUser":  probe.RemoteUser,
			"stdout":      probe.Stdout,
			"stderr":      probe.Stderr,
			"exitCode":    probe.ExitCode,
			"latencyMs":   probe.LatencyMs,
			"checkedMode": probe.Mode,
		}
	}
	if probeErr != nil {
		heartbeat.Status = "offline"
		heartbeat.DetailJSON["errorMessage"] = probeErr.Error()
	}

	if err := s.repo.Create(ctx, heartbeat); err != nil {
		return nil, err
	}
	return &CaptureResult{Heartbeat: &heartbeat, Probe: probe}, nil
}

func (s *Service) ListByServerID(ctx context.Context, serverID string, limit int) ([]model.ServerHeartbeat, error) {
	return s.repo.ListByServerID(ctx, serverID, limit)
}

type Monitor struct {
	servers           serverInventory
	heartbeats        heartbeatCollector
	gpuSnapshots      gpuSnapshotCollector
	heartbeatInterval time.Duration
	gpuInterval       time.Duration
}

func NewMonitor(servers serverInventory, heartbeats heartbeatCollector, gpuSnapshots gpuSnapshotCollector, heartbeatIntervalSec int, gpuIntervalSec int) *Monitor {
	return &Monitor{
		servers:           servers,
		heartbeats:        heartbeats,
		gpuSnapshots:      gpuSnapshots,
		heartbeatInterval: time.Duration(heartbeatIntervalSec) * time.Second,
		gpuInterval:       time.Duration(gpuIntervalSec) * time.Second,
	}
}

func (m *Monitor) Start(ctx context.Context) {
	if m.heartbeatInterval > 0 {
		go m.runLoop(ctx, m.heartbeatInterval, func(loopCtx context.Context, serverID string) error {
			_, err := m.heartbeats.Collect(loopCtx, serverID)
			return err
		})
	}
	if m.gpuInterval > 0 {
		go m.runLoop(ctx, m.gpuInterval, func(loopCtx context.Context, serverID string) error {
			_, err := m.gpuSnapshots.Collect(loopCtx, serverID)
			return err
		})
	}
}

func (m *Monitor) runLoop(ctx context.Context, interval time.Duration, fn func(context.Context, string) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	m.collectOnce(ctx, fn)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collectOnce(ctx, fn)
		}
	}
}

func (m *Monitor) collectOnce(ctx context.Context, fn func(context.Context, string) error) {
	servers, err := m.servers.List(ctx)
	if err != nil {
		log.Printf("stage2 monitor list servers failed: %v", err)
		return
	}
	for _, server := range servers {
		if err := fn(ctx, server.ID); err != nil {
			log.Printf("stage2 monitor collect failed for %s: %v", server.ID, err)
		}
	}
}
