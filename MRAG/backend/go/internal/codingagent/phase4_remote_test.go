package codingagent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/model"
	baseservice "mrag-platform/backend/go/internal/service"
)

type phase4ServerRepoStub struct {
	items map[string]model.Server
}

func (s *phase4ServerRepoStub) List(_ context.Context) ([]model.Server, error) {
	out := make([]model.Server, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	return out, nil
}

func (s *phase4ServerRepoStub) GetByIDWithSecrets(_ context.Context, id string) (*model.Server, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type phase4GPUProbeStub struct {
	result *model.GPUProbeResult
	err    error
}

func (s *phase4GPUProbeStub) CheckGPU(_ context.Context, _ string) (*model.GPUProbeResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.result == nil {
		return nil, fmt.Errorf("gpu probe not configured")
	}
	copyResult := *s.result
	copyResult.Devices = append([]model.GPUDeviceStatus{}, s.result.Devices...)
	return &copyResult, nil
}

type capturePhase4SSHGateway struct {
	requests []baseservice.SSHExecRequest
}

func (g *capturePhase4SSHGateway) Mode() string {
	return "mock"
}

func (g *capturePhase4SSHGateway) Probe(_ context.Context, node *model.Server) (*model.ServerConnectionTestResult, error) {
	return &model.ServerConnectionTestResult{
		ServerID:   node.ID,
		ServerName: node.Name,
		Mode:       "mock",
		Reachable:  true,
		CheckedAt:  time.Now(),
		Result:     "login_success",
	}, nil
}

func (g *capturePhase4SSHGateway) Exec(_ context.Context, _ *model.Server, req baseservice.SSHExecRequest) (*baseservice.SSHExecResult, error) {
	g.requests = append(g.requests, req)
	switch req.Purpose {
	case "phase4_remote_prepare", "phase4_remote_upload", "phase4_remote_bootstrap", "phase4_remote_release_lock":
		return &baseservice.SSHExecResult{Stdout: "ok", ExitCode: 0}, nil
	case "phase4_remote_run":
		return &baseservice.SSHExecResult{Stdout: "run ok", ExitCode: 0}, nil
	case "phase4_remote_read_file":
		remotePath := req.Metadata["remotePath"]
		switch {
		case strings.HasSuffix(remotePath, "/metrics.json"):
			return &baseservice.SSHExecResult{Stdout: `{"protocol_version":"phase4-retrieval-mainline-v1","run_id":"p4run_safe_1","primary_metric":"recall@5","values":{"recall@5":0.66,"query_count":2},"status":"succeeded"}`, ExitCode: 0}, nil
		case strings.HasSuffix(remotePath, "/machine_report.json"):
			return &baseservice.SSHExecResult{Stdout: `{"run_id":"p4run_safe_1","status":"succeeded"}`, ExitCode: 0}, nil
		case strings.HasSuffix(remotePath, "/report.md"):
			return &baseservice.SSHExecResult{Stdout: "# report\n", ExitCode: 0}, nil
		case strings.HasSuffix(remotePath, "/dataset_tool_asset.json"), strings.HasSuffix(remotePath, "/dataset_adapter_contract.json"), strings.HasSuffix(remotePath, "/evaluate_tool_asset.json"), strings.HasSuffix(remotePath, "/predictions.json"):
			return &baseservice.SSHExecResult{Stdout: `{"ok":true}`, ExitCode: 0}, nil
		case strings.HasSuffix(remotePath, "/eval_summary.md"):
			return &baseservice.SSHExecResult{Stdout: "# eval\n", ExitCode: 0}, nil
		case strings.HasSuffix(remotePath, "/driver.log"):
			return &baseservice.SSHExecResult{Stdout: "[phase4_remote] completed\n", ExitCode: 0}, nil
		case strings.HasSuffix(remotePath, "/run.log"):
			return &baseservice.SSHExecResult{Stdout: "[retrieval_mainline] run started\n", ExitCode: 0}, nil
		case strings.HasSuffix(remotePath, ".log"):
			return &baseservice.SSHExecResult{Stdout: "", ExitCode: 0}, nil
		default:
			return &baseservice.SSHExecResult{Stdout: "", ExitCode: 0}, nil
		}
	default:
		return &baseservice.SSHExecResult{Stdout: "", ExitCode: 0}, nil
	}
}

func TestBuildPhase4RemotePathsIsolated(t *testing.T) {
	paths, err := buildPhase4RemotePaths("p4run_safe_1", 2, "/tmp/ignored")
	if err != nil {
		t.Fatalf("buildPhase4RemotePaths returned error: %v", err)
	}
	if paths.MRAGRoot != "/home/bzli/mrag" {
		t.Fatalf("expected fixed remote root, got %s", paths.MRAGRoot)
	}
	if !strings.HasPrefix(paths.RunDir, "/home/bzli/mrag/runs/") {
		t.Fatalf("expected remote run dir under /home/bzli/mrag/runs, got %s", paths.RunDir)
	}
	if !strings.HasPrefix(paths.ArtifactDir, "/home/bzli/mrag/artifacts/") {
		t.Fatalf("expected remote artifact dir under /home/bzli/mrag/artifacts, got %s", paths.ArtifactDir)
	}
	if !strings.HasPrefix(paths.EnvDir, "/home/bzli/mrag/envs/") {
		t.Fatalf("expected remote env dir under /home/bzli/mrag/envs, got %s", paths.EnvDir)
	}
	if _, err = buildPhase4RemotePaths("../bad", 0, "/home/bzli/mrag"); err == nil {
		t.Fatalf("expected invalid run id to fail")
	}
}

func TestPhase4GPUSelectorChoosesBestIdleGPU(t *testing.T) {
	selector := newPhase4ProbeGPUSelector(&phase4GPUProbeStub{
		result: &model.GPUProbeResult{
			AvailableGPUCount: 2,
			TotalGPUCount:     3,
			Devices: []model.GPUDeviceStatus{
				{Index: 1, Name: "GPU-1", MemoryUsedMB: 1024, MemoryTotalMB: 24576, Utilization: 8, Available: true},
				{Index: 0, Name: "GPU-0", MemoryUsedMB: 512, MemoryTotalMB: 24576, Utilization: 9, Available: true},
				{Index: 2, Name: "GPU-2", MemoryUsedMB: 20000, MemoryTotalMB: 24576, Utilization: 80, Available: false},
			},
		},
	})
	selected, probe, err := selector.SelectGPU(context.Background(), &model.Server{ID: "srv_1", Name: phase4ShenzhenlabServerName})
	if err != nil {
		t.Fatalf("SelectGPU returned error: %v", err)
	}
	if probe == nil || selected == nil {
		t.Fatalf("expected gpu selection and probe")
	}
	if selected.GPUIndex != 0 {
		t.Fatalf("expected GPU 0 to win by free memory, got %d", selected.GPUIndex)
	}
}

func TestBuildPhase4RemoteBootstrapScriptUsesPerRunEnv(t *testing.T) {
	paths, err := buildPhase4RemotePaths("p4run_bootstrap_1", 0, "/home/bzli/mrag")
	if err != nil {
		t.Fatalf("buildPhase4RemotePaths returned error: %v", err)
	}
	script := buildPhase4RemoteBootstrapScript(paths)
	if !strings.Contains(script, "/home/bzli/mrag/envs/p4run_bootstrap_1") {
		t.Fatalf("expected per-run env dir in bootstrap script: %s", script)
	}
	if !strings.Contains(script, "bootstrap_env.sh") {
		t.Fatalf("expected bootstrap script to call snapshot bootstrap: %s", script)
	}
}

func TestPhase4RemoteExecutorDryRunGeneratesSafeCommands(t *testing.T) {
	workspaceRoot := t.TempDir()
	runDir := filepath.Join(workspaceRoot, "run")
	artifactDir := filepath.Join(workspaceRoot, "artifacts")
	if err := os.MkdirAll(filepath.Join(runDir, "snapshot"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(runDir, "logs"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	for _, file := range []string{
		filepath.Join(runDir, "remote_experiment_manifest.json"),
		filepath.Join(runDir, "remote_config.json"),
		filepath.Join(runDir, "remote_bootstrap.sh"),
		filepath.Join(runDir, "remote_execute.sh"),
		filepath.Join(runDir, "snapshot", "run_entrypoint.py"),
	} {
		if err := os.WriteFile(file, []byte("placeholder\n"), 0o644); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}
	}
	artifactPaths := map[string]any{
		"run_dir":                  runDir,
		"artifact_dir":             artifactDir,
		"metrics_path":             filepath.Join(artifactDir, "metrics.json"),
		"machine_report_path":      filepath.Join(artifactDir, "machine_report.json"),
		"human_report_path":        filepath.Join(artifactDir, "report.md"),
		"dataset_tool_asset_path":  filepath.Join(artifactDir, "dataset_tool_asset.json"),
		"dataset_adapter_path":     filepath.Join(artifactDir, "dataset_adapter_contract.json"),
		"evaluate_tool_asset_path": filepath.Join(artifactDir, "evaluate_tool_asset.json"),
		"eval_summary_path":        filepath.Join(artifactDir, "eval_summary.md"),
		"predictions_path":         filepath.Join(artifactDir, "predictions.json"),
		"driver_log_path":          filepath.Join(runDir, "logs", "driver.log"),
		"run_log_path":             filepath.Join(runDir, "logs", "run.log"),
		"bootstrap_stdout_path":    filepath.Join(runDir, "logs", "bootstrap.stdout.log"),
		"bootstrap_stderr_path":    filepath.Join(runDir, "logs", "bootstrap.stderr.log"),
		"runtime_stdout_path":      filepath.Join(runDir, "logs", "runtime.stdout.log"),
		"runtime_stderr_path":      filepath.Join(runDir, "logs", "runtime.stderr.log"),
	}
	paths, err := buildPhase4RemotePaths("p4run_safe_1", 0, "/home/bzli/mrag")
	if err != nil {
		t.Fatalf("buildPhase4RemotePaths returned error: %v", err)
	}
	ssh := &capturePhase4SSHGateway{}
	executor := newPhase4ShenzhenlabRemoteExecutor(ssh)
	_, err = executor.Execute(context.Background(), phase4RemoteRunRequest{
		Server: &model.Server{
			ID:       "srv_1",
			Name:     phase4ShenzhenlabServerName,
			Host:     phase4ShenzhenlabServerName,
			Username: "bzli",
			AuthType: "ssh_config",
		},
		RunManifest: &model.Phase4RunManifest{ID: "p4run_safe_1"},
		DatasetProfile: &model.Phase4DatasetProfile{
			ID:          "p4ds_1",
			DatasetName: "VisDoM",
			ServerPath:  "/home/bzli/mrag/datasets/visdom",
		},
		ArtifactPaths:  artifactPaths,
		RemotePaths:    paths,
		GPUSelection:   &phase4GPUSelection{GPUIndex: 0, GPUName: "GPU-0"},
		CommandTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(ssh.requests) == 0 {
		t.Fatalf("expected captured ssh requests")
	}
	prepareCommand := ""
	for _, req := range ssh.requests {
		if req.Purpose == "phase4_remote_prepare" {
			prepareCommand = strings.Join(req.RemoteCommand, " ")
			break
		}
	}
	if !strings.Contains(prepareCommand, "whoami") {
		t.Fatalf("expected whoami guard in prepare command: %s", prepareCommand)
	}
	if !strings.Contains(prepareCommand, "/home/bzli/mrag/cache/gpu_locks/shenzhenvlab-gpu-0.lock") {
		t.Fatalf("expected gpu lock path in prepare command: %s", prepareCommand)
	}
	if strings.Contains(prepareCommand, "rm -rf") {
		t.Fatalf("prepare command must not contain rm -rf: %s", prepareCommand)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "metrics.json")); err != nil {
		t.Fatalf("expected metrics file to be collected locally: %v", err)
	}
}

func TestPhase4CodingServiceRemoteRunIntegration(t *testing.T) {
	workspaceRoot := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	pythonAgentsDir := filepath.Clean(filepath.Join(wd, "..", "..", "..", "python_agents"))
	pythonRunnersDir := filepath.Clean(filepath.Join(wd, "..", "..", "..", "python_runners"))

	jobStore := newPhase4CodingMemoryJobStore()
	triggerStore := newPhase4CodingMemoryTriggerStore()
	artifactStore := &phase4CodingMemoryArtifactStore{}
	phase4Data := newPhase4CodingMemoryDataService()
	phase4Data.datasets["p4ds_1"] = model.Phase4DatasetProfile{
		ID:                "p4ds_1",
		DatasetName:       "VisDoM",
		TaskType:          "multimodal_retrieval",
		OfficialMetric:    "recall@5",
		ServerPath:        "/home/bzli/mrag/datasets/visdom",
		KnownDifficulties: []string{"page-level retrieval first"},
		Splits:            []model.Phase4DatasetSplit{{Name: "train"}, {Name: "val"}, {Name: "test"}},
		Status:            model.Phase4DatasetProfileStatusActive,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	events := &phase4CodingEventPublisherStub{}

	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService("python", pythonAgentsDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	codingSvc := NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactStore, phase4Data, events, workspaceRoot, "python", pythonRunnersDir, "/home/bzli/mrag")
	codingSvc.ConfigureShenzhenlabExecution(&phase4ServerRepoStub{
		items: map[string]model.Server{
			"srv_shenzhen": {
				ID:       "srv_shenzhen",
				Name:     phase4ShenzhenlabServerName,
				Host:     phase4ShenzhenlabServerName,
				Username: "bzli",
				AuthType: "ssh_config",
			},
		},
	}, &phase4GPUProbeStub{
		result: &model.GPUProbeResult{
			AvailableGPUCount: 1,
			TotalGPUCount:     2,
			Devices: []model.GPUDeviceStatus{
				{Index: 0, Name: "RTX-4090", MemoryUsedMB: 512, MemoryTotalMB: 24576, Utilization: 3, Available: true},
				{Index: 1, Name: "RTX-4090", MemoryUsedMB: 20000, MemoryTotalMB: 24576, Utilization: 88, Available: false},
			},
		},
	}, &baseservice.MockSSHGateway{}, 20)
	triggerSvc.RegisterPostProcessor("coding_phase4", codingSvc)

	result, err := codingSvc.Run(context.Background(), model.Phase4CodingRunRequest{
		DatasetProfileID: "p4ds_1",
		IdeaID:           "p4idea_1",
		ReaderContextID:  "p4ctx_1",
		RunnerMode:       phase4ShenzhenlabServerName,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result == nil || result.RunManifest == nil {
		t.Fatalf("expected phase4 coding run manifest")
	}
	if result.RunManifest.Status != model.Phase4RunStatusSucceeded {
		t.Fatalf("expected succeeded run manifest, got %s", result.RunManifest.Status)
	}
	if result.RunManifest.ServerID != "srv_shenzhen" {
		t.Fatalf("expected shenzhenvlab server to be assigned, got %s", result.RunManifest.ServerID)
	}
	if result.RunManifest.GPU != "0" {
		t.Fatalf("expected GPU 0, got %s", result.RunManifest.GPU)
	}
	artifactPaths := result.RunManifest.ArtifactPaths
	if stringValue(artifactPaths["remote_run_dir"]) == "" || stringValue(artifactPaths["remote_gpu_lock_path"]) == "" {
		t.Fatalf("expected remote artifact paths to be recorded: %#v", artifactPaths)
	}
	for _, path := range []string{
		stringValue(artifactPaths["metrics_path"]),
		stringValue(artifactPaths["human_report_path"]),
		stringValue(artifactPaths["driver_log_path"]),
		stringValue(artifactPaths["run_log_path"]),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected local collected file %s: %v", path, err)
		}
	}
	if len(events.items) != 1 || events.items[0].EventType != "phase4_run_ready" {
		t.Fatalf("expected phase4_run_ready event, got %#v", events.items)
	}
}
