package heartbeat

import (
	"context"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type fakeHeartbeatServer struct {
	server *model.Server
	probe  *model.ServerConnectionTestResult
	err    error
}

func (f *fakeHeartbeatServer) Get(_ context.Context, id string) (*model.Server, error) {
	if f.server == nil || f.server.ID != id {
		return nil, nil
	}
	return f.server, nil
}

func (f *fakeHeartbeatServer) TestConnection(_ context.Context, _ string) (*model.ServerConnectionTestResult, error) {
	return f.probe, f.err
}

type memoryHeartbeatRepo struct {
	items []model.ServerHeartbeat
}

func (r *memoryHeartbeatRepo) Create(_ context.Context, item model.ServerHeartbeat) error {
	r.items = append(r.items, item)
	return nil
}

func (r *memoryHeartbeatRepo) ListByServerID(_ context.Context, serverID string, _ int) ([]model.ServerHeartbeat, error) {
	items := make([]model.ServerHeartbeat, 0)
	for _, item := range r.items {
		if item.ServerID == serverID {
			items = append(items, item)
		}
	}
	return items, nil
}

func TestServiceCollectStoresHeartbeat(t *testing.T) {
	checkedAt := time.Now()
	svc := NewService(&fakeHeartbeatServer{
		server: &model.Server{ID: "srv_1", Name: "mock-server", Host: "mock-host"},
		probe: &model.ServerConnectionTestResult{
			ServerID:   "srv_1",
			ServerName: "mock-server",
			Target:     "mock-host",
			Result:     "login_success",
			Reachable:  true,
			Message:    "ok",
			CheckedAt:  checkedAt,
		},
	}, &memoryHeartbeatRepo{})

	result, err := svc.Collect(context.Background(), "srv_1")
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if result.Heartbeat == nil {
		t.Fatalf("expected heartbeat record")
	}
	if result.Heartbeat.Status != "online" {
		t.Fatalf("expected online heartbeat, got %s", result.Heartbeat.Status)
	}
	if got := result.Heartbeat.DetailJSON["result"]; got != "login_success" {
		t.Fatalf("expected result login_success, got %v", got)
	}
}
