package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/service"
	"mrag-platform/backend/go/internal/traintemplate"
)

type memRunStore struct {
	items   map[string]model.ExperimentRun
	history []string
}

func (m *memRunStore) GetByID(_ context.Context, id string) (*model.ExperimentRun, error) {
	item, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (m *memRunStore) Update(_ context.Context, item model.ExperimentRun) error {
	m.items[item.ID] = item
	m.history = append(m.history, item.RunStatus)
	return nil
}

type memExperimentReader struct {
	item model.Experiment
}

func (m *memExperimentReader) GetByID(_ context.Context, id string) (*model.Experiment, error) {
	if m.item.ID != id {
		return nil, nil
	}
	copyItem := m.item
	return &copyItem, nil
}

type memSpecReader struct {
	spec model.ExperimentSpec
}

func (m *memSpecReader) GetLatestByExperimentID(_ context.Context, experimentID string) (*model.ExperimentSpec, error) {
	if m.spec.ExperimentID != experimentID {
		return nil, nil
	}
	copyItem := m.spec
	return &copyItem, nil
}

func (m *memSpecReader) GetByID(_ context.Context, id string) (*model.ExperimentSpec, error) {
	if m.spec.ID != id {
		return nil, nil
	}
	copyItem := m.spec
	return &copyItem, nil
}

type memServerReader struct {
	server model.Server
}

func (m *memServerReader) GetByIDWithSecrets(_ context.Context, id string) (*model.Server, error) {
	if m.server.ID != id {
		return nil, nil
	}
	copyItem := m.server
	return &copyItem, nil
}

type memRunLogStore struct {
	items []model.RunLog
}

func (m *memRunLogStore) Add(_ context.Context, item model.RunLog) error {
	m.items = append(m.items, item)
	return nil
}

func (m *memRunLogStore) ListByRunID(_ context.Context, runID string) ([]model.RunLog, error) {
	items := make([]model.RunLog, 0)
	for _, item := range m.items {
		if item.RunID == runID {
			items = append(items, item)
		}
	}
	return items, nil
}

type fakeSSHGateway struct{}

func (g *fakeSSHGateway) Mode() string { return "mock" }
func (g *fakeSSHGateway) Probe(context.Context, *model.Server) (*model.ServerConnectionTestResult, error) {
	return &model.ServerConnectionTestResult{}, nil
}
func (g *fakeSSHGateway) Exec(_ context.Context, _ *model.Server, req service.SSHExecRequest) (*service.SSHExecResult, error) {
	switch req.Purpose {
	case "experiment_run_prepare", "experiment_run_upload":
		return &service.SSHExecResult{ExitCode: 0}, nil
	case "experiment_run_start":
		return &service.SSHExecResult{ExitCode: 0, Stdout: "runner started", Stderr: ""}, nil
	case "experiment_run_read_file":
		command := ""
		if len(req.RemoteCommand) > 0 {
			command = req.RemoteCommand[len(req.RemoteCommand)-1]
		}
		switch {
		case strings.Contains(command, "metrics.json"):
			return &service.SSHExecResult{ExitCode: 0, Stdout: `{"primary_metric":"accuracy","values":{"accuracy":0.88}}`}, nil
		case strings.Contains(command, "result.md"):
			return &service.SSHExecResult{ExitCode: 0, Stdout: "# Mock Result\n"}, nil
		case strings.Contains(command, "stdout.log"):
			return &service.SSHExecResult{ExitCode: 0, Stdout: "[mock-train] completed\n[mock-eval] evaluating outputs\n"}, nil
		case strings.Contains(command, "stderr.log"):
			return &service.SSHExecResult{ExitCode: 0, Stdout: ""}, nil
		}
	}
	return &service.SSHExecResult{ExitCode: 0}, nil
}

func TestStartRunSuccessAndLogs(t *testing.T) {
	workspaceRoot := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	templateRoot := filepath.Clean(filepath.Join(wd, "..", "..", "..", "python_templates"))
	templateSvc := traintemplate.NewService(templateRoot, workspaceRoot)

	runStore := &memRunStore{items: map[string]model.ExperimentRun{
		"run_1": {
			ID:               "run_1",
			ExperimentID:     "exp_1",
			SpecID:           "spec_1",
			AssignedServerID: "srv_1",
			RunStatus:        "scheduled",
			RemoteWorkdir:    filepath.Join(workspaceRoot, "experiments", "exp_1", "run_1"),
			ResultJSON:       map[string]interface{}{},
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}}
	spec := model.ExperimentSpec{
		ID:           "spec_1",
		ExperimentID: "exp_1",
		SpecJSON: map[string]interface{}{
			"dataset_ref":         map[string]interface{}{"dataset_asset_id": "dasset_1"},
			"dataset_loader_ref":  map[string]interface{}{"loader_id": "mrag.dataset_asset_loader.v1"},
			"train_template_type": "mock_train_template",
			"model_name":          "mock/llama3.1-8b-instruct",
			"hyperparams":         map[string]interface{}{"epochs": 3},
			"output_dir":          filepath.Join(workspaceRoot, "experiments", "exp_1", "run_1", "outputs"),
			"expected_metrics":    map[string]interface{}{"primary": "accuracy"},
			"comparison_targets":  []map[string]interface{}{{"type": "baseline", "id": "baseline_1"}},
		},
	}
	logStore := &memRunLogStore{}
	svc := NewService(
		runStore,
		&memExperimentReader{item: model.Experiment{ID: "exp_1"}},
		&memSpecReader{spec: spec},
		&memServerReader{server: model.Server{ID: "srv_1", TaskWorkdir: "/remote/work"}},
		logStore,
		&fakeSSHGateway{},
		templateSvc,
		nil,
		"/remote/work",
	)

	run, err := svc.StartRun(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("StartRun returned error: %v", err)
	}
	if run.RunStatus != "succeeded" {
		t.Fatalf("expected succeeded, got %s", run.RunStatus)
	}
	if len(logStore.items) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logStore.items))
	}
	if len(runStore.history) < 3 || runStore.history[0] != "preparing" || runStore.history[1] != "running" || runStore.history[len(runStore.history)-1] != "succeeded" {
		t.Fatalf("unexpected status history: %#v", runStore.history)
	}
	if _, ok := run.ResultJSON["metrics"]; !ok {
		t.Fatalf("expected metrics in result_json")
	}
}
