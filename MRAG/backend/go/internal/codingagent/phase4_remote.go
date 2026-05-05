package codingagent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	baseservice "mrag-platform/backend/go/internal/service"
)

const (
	phase4ShenzhenlabServerName = "shenzhenvlab"
	phase4RemoteHomeRoot        = "/home/bzli"
	phase4RemoteMRAGRoot        = "/home/bzli/mrag"
)

var phase4SafeRunIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type phase4ServerSecretReader interface {
	List(context.Context) ([]model.Server, error)
	GetByIDWithSecrets(context.Context, string) (*model.Server, error)
}

type phase4GPUProbeSource interface {
	CheckGPU(context.Context, string) (*model.GPUProbeResult, error)
}

type phase4GPUSelector interface {
	SelectGPU(context.Context, *model.Server) (*phase4GPUSelection, *model.GPUProbeResult, error)
}

type phase4RemoteExecutor interface {
	Execute(context.Context, phase4RemoteRunRequest) (*phase4RemoteRunResult, error)
}

type phase4GPUSelection struct {
	GPUIndex    int
	GPUName     string
	FreeMemMB   int64
	Utilization int64
}

type phase4RemotePaths struct {
	HomeRoot                    string
	MRAGRoot                    string
	DatasetsRoot                string
	RunsRoot                    string
	RunDir                      string
	SnapshotDir                 string
	LogsDir                     string
	ArtifactsRoot               string
	ArtifactDir                 string
	CacheDir                    string
	GPULockDir                  string
	GPULockPath                 string
	EnvsRoot                    string
	EnvDir                      string
	RemoteManifestPath          string
	RemoteConfigPath            string
	RemoteBootstrapScriptPath   string
	RemoteExecuteScriptPath     string
	RemoteDriverLogPath         string
	RemoteRunLogPath            string
	RemoteBootstrapStdoutPath   string
	RemoteBootstrapStderrPath   string
	RemoteRuntimeStdoutPath     string
	RemoteRuntimeStderrPath     string
	RemoteMetricsPath           string
	RemotePredictionsPath       string
	RemoteMachineReportPath     string
	RemoteHumanReportPath       string
	RemoteDatasetToolAssetPath  string
	RemoteDatasetAdapterPath    string
	RemoteEvaluateToolAssetPath string
	RemoteEvalSummaryPath       string
}

type phase4RemoteRunRequest struct {
	Server         *model.Server
	RunManifest    *model.Phase4RunManifest
	DatasetProfile *model.Phase4DatasetProfile
	ArtifactPaths  map[string]any
	RemotePaths    phase4RemotePaths
	GPUSelection   *phase4GPUSelection
	CommandTimeout time.Duration
}

type phase4RemoteRunResult struct {
	RemotePaths phase4RemotePaths
}

type phase4ProbeGPUSelector struct {
	source phase4GPUProbeSource
}

func newPhase4ProbeGPUSelector(source phase4GPUProbeSource) phase4GPUSelector {
	if source == nil {
		return nil
	}
	return &phase4ProbeGPUSelector{source: source}
}

func (s *phase4ProbeGPUSelector) SelectGPU(ctx context.Context, server *model.Server) (*phase4GPUSelection, *model.GPUProbeResult, error) {
	if s == nil || s.source == nil {
		return nil, nil, fmt.Errorf("phase4 gpu selector is not configured")
	}
	if server == nil || strings.TrimSpace(server.ID) == "" {
		return nil, nil, fmt.Errorf("phase4 gpu selection requires a server")
	}
	probe, err := s.source.CheckGPU(ctx, server.ID)
	if err != nil {
		return nil, nil, err
	}
	if probe == nil {
		return nil, nil, fmt.Errorf("phase4 gpu probe returned nil")
	}
	candidates := make([]phase4GPUSelection, 0, len(probe.Devices))
	for _, device := range probe.Devices {
		if !device.Available {
			continue
		}
		candidates = append(candidates, phase4GPUSelection{
			GPUIndex:    device.Index,
			GPUName:     strings.TrimSpace(device.Name),
			FreeMemMB:   device.MemoryTotalMB - device.MemoryUsedMB,
			Utilization: device.Utilization,
		})
	}
	if len(candidates) == 0 {
		return nil, probe, fmt.Errorf("no idle gpu is available on shenzhenvlab")
	}
	sort.Slice(candidates, func(i int, j int) bool {
		if candidates[i].FreeMemMB != candidates[j].FreeMemMB {
			return candidates[i].FreeMemMB > candidates[j].FreeMemMB
		}
		if candidates[i].Utilization != candidates[j].Utilization {
			return candidates[i].Utilization < candidates[j].Utilization
		}
		return candidates[i].GPUIndex < candidates[j].GPUIndex
	})
	selected := candidates[0]
	return &selected, probe, nil
}

type phase4ShenzhenlabRemoteExecutor struct {
	ssh baseservice.SSHGateway
}

func newPhase4ShenzhenlabRemoteExecutor(ssh baseservice.SSHGateway) phase4RemoteExecutor {
	if ssh == nil {
		return nil
	}
	return &phase4ShenzhenlabRemoteExecutor{ssh: ssh}
}

func isPhase4ShenzhenlabMode(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), phase4ShenzhenlabServerName)
}

func normalizePhase4RemoteRoot(_ string) string {
	return phase4RemoteMRAGRoot
}

func buildPhase4RemotePaths(runID string, gpuIndex int, remoteRoot string) (phase4RemotePaths, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if !phase4SafeRunIDPattern.MatchString(trimmedRunID) {
		return phase4RemotePaths{}, fmt.Errorf("invalid phase4 run id for remote execution: %s", runID)
	}
	root := normalizePhase4RemoteRoot(remoteRoot)
	cacheDir := path.Join(root, "cache")
	runDir := path.Join(root, "runs", trimmedRunID)
	artifactDir := path.Join(root, "artifacts", trimmedRunID)
	logsDir := path.Join(runDir, "logs")
	paths := phase4RemotePaths{
		HomeRoot:                    phase4RemoteHomeRoot,
		MRAGRoot:                    root,
		DatasetsRoot:                path.Join(root, "datasets"),
		RunsRoot:                    path.Join(root, "runs"),
		RunDir:                      runDir,
		SnapshotDir:                 path.Join(runDir, "snapshot"),
		LogsDir:                     logsDir,
		ArtifactsRoot:               path.Join(root, "artifacts"),
		ArtifactDir:                 artifactDir,
		CacheDir:                    cacheDir,
		GPULockDir:                  path.Join(cacheDir, "gpu_locks"),
		GPULockPath:                 path.Join(cacheDir, "gpu_locks", fmt.Sprintf("%s-gpu-%d.lock", phase4ShenzhenlabServerName, gpuIndex)),
		EnvsRoot:                    path.Join(root, "envs"),
		EnvDir:                      path.Join(root, "envs", trimmedRunID),
		RemoteManifestPath:          path.Join(runDir, "remote_experiment_manifest.json"),
		RemoteConfigPath:            path.Join(runDir, "remote_config.json"),
		RemoteBootstrapScriptPath:   path.Join(runDir, "remote_bootstrap.sh"),
		RemoteExecuteScriptPath:     path.Join(runDir, "remote_execute.sh"),
		RemoteDriverLogPath:         path.Join(logsDir, "driver.log"),
		RemoteRunLogPath:            path.Join(logsDir, "run.log"),
		RemoteBootstrapStdoutPath:   path.Join(logsDir, "bootstrap.stdout.log"),
		RemoteBootstrapStderrPath:   path.Join(logsDir, "bootstrap.stderr.log"),
		RemoteRuntimeStdoutPath:     path.Join(logsDir, "runtime.stdout.log"),
		RemoteRuntimeStderrPath:     path.Join(logsDir, "runtime.stderr.log"),
		RemoteMetricsPath:           path.Join(artifactDir, "metrics.json"),
		RemotePredictionsPath:       path.Join(artifactDir, "predictions.json"),
		RemoteMachineReportPath:     path.Join(artifactDir, "machine_report.json"),
		RemoteHumanReportPath:       path.Join(artifactDir, "report.md"),
		RemoteDatasetToolAssetPath:  path.Join(artifactDir, "dataset_tool_asset.json"),
		RemoteDatasetAdapterPath:    path.Join(artifactDir, "dataset_adapter_contract.json"),
		RemoteEvaluateToolAssetPath: path.Join(artifactDir, "evaluate_tool_asset.json"),
		RemoteEvalSummaryPath:       path.Join(artifactDir, "eval_summary.md"),
	}
	return paths, nil
}

func validatePhase4DatasetRemotePath(datasetPath string) error {
	datasetPath = path.Clean(strings.TrimSpace(datasetPath))
	root := normalizePhase4RemoteRoot("")
	if datasetPath == "" {
		return fmt.Errorf("phase4 remote dataset path is required")
	}
	if datasetPath != path.Join(root, "datasets") && !strings.HasPrefix(datasetPath, path.Join(root, "datasets")+"/") {
		return fmt.Errorf("phase4 remote dataset path must stay under %s", path.Join(root, "datasets"))
	}
	return nil
}

func validatePhase4Server(server *model.Server) error {
	if server == nil {
		return fmt.Errorf("phase4 remote execution requires a server")
	}
	if !strings.EqualFold(strings.TrimSpace(server.Name), phase4ShenzhenlabServerName) {
		return fmt.Errorf("phase4 remote execution only supports %s", phase4ShenzhenlabServerName)
	}
	if username := strings.TrimSpace(server.Username); username != "" && !strings.EqualFold(username, "bzli") {
		return fmt.Errorf("phase4 remote execution requires remote user bzli")
	}
	return nil
}

func buildPhase4RemoteBootstrapScript(paths phase4RemotePaths) string {
	return strings.TrimSpace(fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

RUN_ID=%s
ENV_DIR=%s
SNAPSHOT_DIR=%s

mkdir -p "$ENV_DIR"
python3 -m venv "$ENV_DIR"
. "$ENV_DIR/bin/activate"
cd "$SNAPSHOT_DIR"

if [ -f "bootstrap_env.sh" ]; then
  bash "bootstrap_env.sh"
else
  python -m pip install --upgrade pip
  if [ -f "requirements.txt" ]; then
    python -m pip install -r "requirements.txt"
  fi
fi

echo "phase4 remote bootstrap complete for ${RUN_ID}"
`, phase4ShellQuote(path.Base(paths.RunDir)), phase4ShellQuote(paths.EnvDir), phase4ShellQuote(paths.SnapshotDir))) + "\n"
}

func buildPhase4RemoteExecuteScript(paths phase4RemotePaths, gpuIndex int) string {
	return strings.TrimSpace(fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

RUN_ID=%s
ENV_DIR=%s
SNAPSHOT_DIR=%s
MANIFEST_PATH=%s
DRIVER_LOG=%s
export CUDA_VISIBLE_DEVICES=%s

. "$ENV_DIR/bin/activate"
cd "$SNAPSHOT_DIR"

{
  echo "[phase4_remote] run_id=${RUN_ID}"
  echo "[phase4_remote] selected_gpu=${CUDA_VISIBLE_DEVICES}"
  python "run_entrypoint.py" --manifest "$MANIFEST_PATH"
  python "eval_entrypoint.py" --manifest "$MANIFEST_PATH"
  echo "[phase4_remote] completed"
} >> "$DRIVER_LOG" 2>&1
`, phase4ShellQuote(path.Base(paths.RunDir)), phase4ShellQuote(paths.EnvDir), phase4ShellQuote(paths.SnapshotDir), phase4ShellQuote(paths.RemoteManifestPath), phase4ShellQuote(paths.RemoteDriverLogPath), phase4ShellQuote(fmt.Sprintf("%d", gpuIndex)))) + "\n"
}

func (e *phase4ShenzhenlabRemoteExecutor) Execute(ctx context.Context, req phase4RemoteRunRequest) (*phase4RemoteRunResult, error) {
	if e == nil || e.ssh == nil {
		return nil, fmt.Errorf("phase4 remote executor is not configured")
	}
	if err := validatePhase4Server(req.Server); err != nil {
		return nil, err
	}
	if req.RunManifest == nil {
		return nil, fmt.Errorf("phase4 remote run manifest is required")
	}
	if req.DatasetProfile == nil {
		return nil, fmt.Errorf("phase4 remote dataset profile is required")
	}
	if req.GPUSelection == nil {
		return nil, fmt.Errorf("phase4 remote gpu selection is required")
	}
	if err := validatePhase4DatasetRemotePath(req.DatasetProfile.ServerPath); err != nil {
		return nil, err
	}
	artifactPaths := phase4CodingEnsureMap(req.ArtifactPaths)
	localRunDir := strings.TrimSpace(stringValue(artifactPaths["run_dir"]))
	if localRunDir == "" {
		return nil, fmt.Errorf("phase4 remote local run directory is required")
	}
	if req.CommandTimeout <= 0 {
		req.CommandTimeout = 20 * time.Second
	}
	if err := e.prepareRemoteLayout(ctx, req.Server, req.RemotePaths, req.RunManifest.ID, req.CommandTimeout); err != nil {
		return nil, err
	}
	lockReleased := false
	defer func() {
		if lockReleased {
			return
		}
		_ = e.releaseGPULock(context.Background(), req.Server, req.RemotePaths, req.RunManifest.ID, req.CommandTimeout)
	}()
	if err := e.uploadDirectory(ctx, req.Server, localRunDir, req.RemotePaths.RunDir, req.CommandTimeout); err != nil {
		return nil, err
	}
	if err := e.runBootstrap(ctx, req.Server, req.RemotePaths, req.CommandTimeout); err != nil {
		_ = e.collectRemoteOutputs(ctx, req.Server, artifactPaths, req.RemotePaths, req.CommandTimeout)
		return nil, err
	}
	if err := e.runMainline(ctx, req.Server, req.RemotePaths, req.CommandTimeout); err != nil {
		_ = e.collectRemoteOutputs(ctx, req.Server, artifactPaths, req.RemotePaths, req.CommandTimeout)
		return nil, err
	}
	if err := e.collectRemoteOutputs(ctx, req.Server, artifactPaths, req.RemotePaths, req.CommandTimeout); err != nil {
		return nil, err
	}
	if err := e.releaseGPULock(ctx, req.Server, req.RemotePaths, req.RunManifest.ID, req.CommandTimeout); err != nil {
		return nil, err
	}
	lockReleased = true
	return &phase4RemoteRunResult{RemotePaths: req.RemotePaths}, nil
}

func (e *phase4ShenzhenlabRemoteExecutor) prepareRemoteLayout(ctx context.Context, server *model.Server, paths phase4RemotePaths, runID string, timeout time.Duration) error {
	runMarkerPath := path.Join(paths.RunDir, ".phase4_run_id")
	script := fmt.Sprintf(`set -euo pipefail
if [ "$(whoami)" != "bzli" ]; then
  echo "unexpected remote user: $(whoami)" >&2
  exit 31
fi
mkdir -p %s %s %s %s %s %s
for target in %s %s %s; do
  if [ -e "$target" ] && [ ! -f %s ]; then
    echo "remote path already exists without run marker: $target" >&2
    exit 32
  fi
  if [ -e "$target" ] && [ -f %s ] && ! grep -q %s %s; then
    echo "remote path belongs to another run: $target" >&2
    exit 32
  fi
done
mkdir -p %s %s %s %s %s
printf '%%s\n' %s > %s
( set -o noclobber; printf '%%s\n' %s > %s ) 2>/dev/null || {
  echo "gpu lock already exists: %s" >&2
  exit 33
}
`, phase4ShellQuote(paths.DatasetsRoot), phase4ShellQuote(paths.RunsRoot), phase4ShellQuote(paths.ArtifactsRoot), phase4ShellQuote(paths.CacheDir), phase4ShellQuote(paths.EnvsRoot), phase4ShellQuote(paths.GPULockDir), phase4ShellQuote(paths.RunDir), phase4ShellQuote(paths.ArtifactDir), phase4ShellQuote(paths.EnvDir), phase4ShellQuote(runMarkerPath), phase4ShellQuote(runMarkerPath), phase4ShellQuote(strings.TrimSpace(runID)), phase4ShellQuote(runMarkerPath), phase4ShellQuote(paths.RunDir), phase4ShellQuote(paths.ArtifactDir), phase4ShellQuote(paths.LogsDir), phase4ShellQuote(paths.SnapshotDir), phase4ShellQuote(paths.EnvDir), phase4ShellQuote(strings.TrimSpace(runID)), phase4ShellQuote(runMarkerPath), phase4ShellQuote(strings.TrimSpace(runID)), phase4ShellQuote(paths.GPULockPath), phase4ShellQuote(paths.GPULockPath))
	result, err := e.ssh.Exec(ctx, server, baseservice.SSHExecRequest{
		Purpose:       "phase4_remote_prepare",
		RemoteCommand: []string{"bash", "-lc", script},
		Metadata: map[string]string{
			"runId":        runID,
			"runDir":       paths.RunDir,
			"artifactDir":  paths.ArtifactDir,
			"envDir":       paths.EnvDir,
			"gpuLockPath":  paths.GPULockPath,
			"remoteRoot":   paths.MRAGRoot,
			"datasetsRoot": paths.DatasetsRoot,
		},
		Timeout: timeout,
	})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, "phase4 remote prepare failed"))
	}
	return nil
}

func (e *phase4ShenzhenlabRemoteExecutor) uploadDirectory(ctx context.Context, server *model.Server, localDir string, remoteDir string, timeout time.Duration) error {
	return filepath.Walk(localDir, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(localDir, current)
		if err != nil {
			return err
		}
		remotePath := path.Join(remoteDir, filepath.ToSlash(rel))
		if info.IsDir() {
			result, err := e.ssh.Exec(ctx, server, baseservice.SSHExecRequest{
				Purpose:       "phase4_remote_prepare",
				RemoteCommand: []string{"bash", "-lc", fmt.Sprintf("mkdir -p %s", phase4ShellQuote(remotePath))},
				Metadata: map[string]string{
					"remotePath": remotePath,
				},
				Timeout: timeout,
			})
			if err != nil {
				return err
			}
			if result.ExitCode != 0 {
				return fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, "phase4 remote mkdir failed"))
			}
			return nil
		}
		data, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		payload := base64.StdEncoding.EncodeToString(data)
		result, err := e.ssh.Exec(ctx, server, baseservice.SSHExecRequest{
			Purpose:       "phase4_remote_upload",
			RemoteCommand: []string{"python3", "-c", "import base64, pathlib, sys; p=pathlib.Path(sys.argv[1]); p.parent.mkdir(parents=True, exist_ok=True); p.write_bytes(base64.b64decode(sys.stdin.buffer.read()))", remotePath},
			Stdin:         payload,
			Metadata: map[string]string{
				"remotePath": remotePath,
			},
			Timeout: maxDuration(timeout, 30*time.Second),
		})
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, "phase4 remote upload failed"))
		}
		return nil
	})
}

func (e *phase4ShenzhenlabRemoteExecutor) runBootstrap(ctx context.Context, server *model.Server, paths phase4RemotePaths, timeout time.Duration) error {
	command := fmt.Sprintf("bash %s > %s 2> %s", phase4ShellQuote(paths.RemoteBootstrapScriptPath), phase4ShellQuote(paths.RemoteBootstrapStdoutPath), phase4ShellQuote(paths.RemoteBootstrapStderrPath))
	result, err := e.ssh.Exec(ctx, server, baseservice.SSHExecRequest{
		Purpose:       "phase4_remote_bootstrap",
		RemoteCommand: []string{"bash", "-lc", command},
		Metadata: map[string]string{
			"scriptPath": paths.RemoteBootstrapScriptPath,
		},
		Timeout: maxDuration(timeout*4, 2*time.Minute),
	})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, "phase4 remote bootstrap failed"))
	}
	return nil
}

func (e *phase4ShenzhenlabRemoteExecutor) runMainline(ctx context.Context, server *model.Server, paths phase4RemotePaths, timeout time.Duration) error {
	command := fmt.Sprintf("bash %s > %s 2> %s", phase4ShellQuote(paths.RemoteExecuteScriptPath), phase4ShellQuote(paths.RemoteRuntimeStdoutPath), phase4ShellQuote(paths.RemoteRuntimeStderrPath))
	result, err := e.ssh.Exec(ctx, server, baseservice.SSHExecRequest{
		Purpose:       "phase4_remote_run",
		RemoteCommand: []string{"bash", "-lc", command},
		Metadata: map[string]string{
			"scriptPath": paths.RemoteExecuteScriptPath,
		},
		Timeout: maxDuration(timeout*90, 60*time.Minute),
	})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, "phase4 remote run failed"))
	}
	return nil
}

func (e *phase4ShenzhenlabRemoteExecutor) releaseGPULock(ctx context.Context, server *model.Server, paths phase4RemotePaths, runID string, timeout time.Duration) error {
	script := fmt.Sprintf(`set -euo pipefail
if [ -f %s ] && grep -q %s %s; then
  rm -f %s
fi
`, phase4ShellQuote(paths.GPULockPath), phase4ShellQuote(strings.TrimSpace(runID)), phase4ShellQuote(paths.GPULockPath), phase4ShellQuote(paths.GPULockPath))
	result, err := e.ssh.Exec(ctx, server, baseservice.SSHExecRequest{
		Purpose:       "phase4_remote_release_lock",
		RemoteCommand: []string{"bash", "-lc", script},
		Metadata: map[string]string{
			"gpuLockPath": paths.GPULockPath,
			"runId":       runID,
		},
		Timeout: timeout,
	})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, "phase4 remote release lock failed"))
	}
	return nil
}

func (e *phase4ShenzhenlabRemoteExecutor) collectRemoteOutputs(ctx context.Context, server *model.Server, artifactPaths map[string]any, paths phase4RemotePaths, timeout time.Duration) error {
	files := []struct {
		remote   string
		localKey string
		required bool
	}{
		{paths.RemoteMetricsPath, "metrics_path", true},
		{paths.RemoteMachineReportPath, "machine_report_path", true},
		{paths.RemoteHumanReportPath, "human_report_path", true},
		{paths.RemoteDatasetToolAssetPath, "dataset_tool_asset_path", true},
		{paths.RemoteDatasetAdapterPath, "dataset_adapter_path", true},
		{paths.RemoteEvaluateToolAssetPath, "evaluate_tool_asset_path", true},
		{paths.RemoteEvalSummaryPath, "eval_summary_path", true},
		{paths.RemotePredictionsPath, "predictions_path", true},
		{paths.RemoteDriverLogPath, "driver_log_path", true},
		{paths.RemoteRunLogPath, "run_log_path", false},
		{paths.RemoteBootstrapStdoutPath, "bootstrap_stdout_path", false},
		{paths.RemoteBootstrapStderrPath, "bootstrap_stderr_path", false},
		{paths.RemoteRuntimeStdoutPath, "runtime_stdout_path", false},
		{paths.RemoteRuntimeStderrPath, "runtime_stderr_path", false},
	}
	for _, item := range files {
		localPath := strings.TrimSpace(stringValue(artifactPaths[item.localKey]))
		if localPath == "" {
			continue
		}
		content, err := e.readRemoteFile(ctx, server, item.remote, timeout)
		if err != nil {
			if item.required {
				return err
			}
			continue
		}
		if strings.TrimSpace(content) == "" && item.required {
			return fmt.Errorf("phase4 remote required output is missing: %s", item.remote)
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		if err = os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
			return err
		}
		if err = os.WriteFile(localPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (e *phase4ShenzhenlabRemoteExecutor) readRemoteFile(ctx context.Context, server *model.Server, remotePath string, timeout time.Duration) (string, error) {
	result, err := e.ssh.Exec(ctx, server, baseservice.SSHExecRequest{
		Purpose:       "phase4_remote_read_file",
		RemoteCommand: []string{"sh", "-lc", fmt.Sprintf("if [ -f %s ]; then cat %s; fi", phase4ShellQuote(remotePath), phase4ShellQuote(remotePath))},
		Metadata: map[string]string{
			"remotePath": remotePath,
		},
		Timeout: timeout,
	})
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, "phase4 remote read failed"))
	}
	return result.Stdout, nil
}

func (s *Phase4Service) executePhase4RemoteRun(ctx context.Context, job *model.AgentJob, runManifest *model.Phase4RunManifest, datasetProfile *model.Phase4DatasetProfile, idea *model.Phase4Idea, readerContext *model.Phase4ReaderContext, artifactPaths map[string]any, snapshotID string) error {
	if s.servers == nil || s.gpuSelector == nil || s.remoteExecutor == nil {
		return s.failPhase4RunSetup(ctx, job, runManifest.ID, artifactPaths, "remote_configure", fmt.Errorf("phase4 shenzhenvlab execution is not configured"))
	}
	serverNode, err := s.resolvePhase4ShenzhenlabServer(ctx, strings.TrimSpace(runManifest.ServerID))
	if err != nil {
		return s.failPhase4RunSetup(ctx, job, runManifest.ID, artifactPaths, "resolve_server", err)
	}
	selection, probe, err := s.gpuSelector.SelectGPU(ctx, serverNode)
	if err != nil {
		return s.failPhase4RunSetup(ctx, job, runManifest.ID, artifactPaths, "select_gpu", err)
	}
	remotePaths, err := buildPhase4RemotePaths(runManifest.ID, selection.GPUIndex, s.phase4RemoteWorkRoot)
	if err != nil {
		return s.failPhase4RunSetup(ctx, job, runManifest.ID, artifactPaths, "remote_paths", err)
	}
	if err = s.materializePhase4RemoteFiles(runManifest, datasetProfile, artifactPaths, remotePaths, *selection); err != nil {
		return s.failPhase4RunSetup(ctx, job, runManifest.ID, artifactPaths, "materialize_remote_files", err)
	}
	artifactPaths = s.attachPhase4RemoteArtifactPaths(artifactPaths, remotePaths, *selection, probe)
	serverID := serverNode.ID
	gpu := fmt.Sprintf("%d", selection.GPUIndex)
	if _, err = s.phase4.UpdateRunManifest(ctx, runManifest.ID, model.Phase4RunManifestUpdateRequest{
		CodeSnapshotID: &snapshotID,
		ServerID:       &serverID,
		GPU:            &gpu,
		ArtifactPaths:  &artifactPaths,
		LogsPath:       stringPtr(stringValue(artifactPaths["logs_dir"])),
		MetricsPath:    stringPtr(stringValue(artifactPaths["metrics_path"])),
	}); err != nil {
		return s.failPhase4RunSetup(ctx, job, runManifest.ID, artifactPaths, "update_manifest_remote", err)
	}
	if _, err = s.phase4.UpdateRunManifestStatus(ctx, runManifest.ID, model.Phase4RunManifestStatusUpdateRequest{
		Status: model.Phase4RunStatusScheduled,
	}); err != nil {
		return s.failPhase4RunSetup(ctx, job, runManifest.ID, artifactPaths, "schedule_remote", err)
	}
	startedAt := time.Now()
	if _, err = s.phase4.UpdateRunManifestStatus(ctx, runManifest.ID, model.Phase4RunManifestStatusUpdateRequest{
		Status:    model.Phase4RunStatusRunning,
		StartedAt: &startedAt,
	}); err != nil {
		return s.failPhase4RunSetup(ctx, job, runManifest.ID, artifactPaths, "remote_start", err)
	}
	if _, err = s.remoteExecutor.Execute(ctx, phase4RemoteRunRequest{
		Server:         serverNode,
		RunManifest:    runManifest,
		DatasetProfile: datasetProfile,
		ArtifactPaths:  artifactPaths,
		RemotePaths:    remotePaths,
		GPUSelection:   selection,
		CommandTimeout: s.commandTimeout,
	}); err != nil {
		return s.failPhase4Run(ctx, job, runManifest.ID, artifactPaths, "shenzhenvlab_remote_run", err)
	}
	metricsRaw := readJSON(stringValue(artifactPaths["metrics_path"]))
	metricsSummary := map[string]any{}
	if values := mapValue(metricsRaw["values"]); len(values) > 0 {
		metricsSummary = values
		if primary := stringValue(metricsRaw["primary_metric"]); primary != "" {
			metricsSummary["primary_metric"] = primary
		}
	}
	if len(metricsSummary) == 0 {
		metricsSummary["status"] = "missing"
	}
	job.NormalizedPayload = updatePhase4CodingJobPayload(job.NormalizedPayload, runManifest.ID, snapshotID, artifactPaths, metricsSummary, stringValue(artifactPaths["human_report_path"]))
	job.UpdatedAt = time.Now()
	if err = s.jobUpdates.Update(ctx, *job); err != nil {
		return err
	}
	if err = s.persistPhase4Artifacts(ctx, job.ID, artifactPaths); err != nil {
		return err
	}
	finishedAt := time.Now()
	if _, err = s.phase4.UpdateRunManifestStatus(ctx, runManifest.ID, model.Phase4RunManifestStatusUpdateRequest{
		Status:     model.Phase4RunStatusSucceeded,
		FinishedAt: &finishedAt,
	}); err != nil {
		return err
	}
	return s.publishPhase4RunReady(ctx, *runManifest, *datasetProfile, *idea, *readerContext, artifactPaths, metricsRaw)
}

func (s *Phase4Service) failPhase4RunSetup(ctx context.Context, job *model.AgentJob, runManifestID string, artifactPaths map[string]any, stage string, execErr error) error {
	feedback := map[string]any{
		"stage": stage,
		"error": execErr.Error(),
	}
	if _, err := s.phase4.UpdateRunManifest(ctx, runManifestID, model.Phase4RunManifestUpdateRequest{
		ArtifactPaths:   &artifactPaths,
		LogsPath:        stringPtr(stringValue(artifactPaths["logs_dir"])),
		MetricsPath:     stringPtr(stringValue(artifactPaths["metrics_path"])),
		FailureFeedback: &feedback,
	}); err != nil {
		return err
	}
	if _, err := s.phase4.UpdateRunManifestStatus(ctx, runManifestID, model.Phase4RunManifestStatusUpdateRequest{
		Status:          model.Phase4RunStatusFailed,
		FailureFeedback: feedback,
	}); err != nil {
		return err
	}
	if job != nil {
		job.Warnings = append(job.Warnings, fmt.Sprintf("phase4 remote setup failed during %s: %s", stage, execErr.Error()))
		job.UpdatedAt = time.Now()
		_ = s.jobUpdates.Update(ctx, *job)
	}
	return execErr
}

func (s *Phase4Service) materializePhase4RemoteFiles(runManifest *model.Phase4RunManifest, datasetProfile *model.Phase4DatasetProfile, artifactPaths map[string]any, remotePaths phase4RemotePaths, selection phase4GPUSelection) error {
	localRunDir := strings.TrimSpace(stringValue(artifactPaths["run_dir"]))
	localLogsDir := strings.TrimSpace(stringValue(artifactPaths["logs_dir"]))
	localConfigPath := strings.TrimSpace(stringValue(artifactPaths["config_path"]))
	if localRunDir == "" || localLogsDir == "" || localConfigPath == "" {
		return fmt.Errorf("phase4 remote run requires prepared local run layout")
	}
	remoteConfigSourcePath := filepath.Join(localRunDir, "remote_config.json")
	configPayload := readJSON(localConfigPath)
	configPayload["runner_mode"] = phase4ShenzhenlabServerName
	if datasetAdapter := mapValue(configPayload["dataset_adapter"]); len(datasetAdapter) > 0 {
		datasetAdapter["server_path"] = strings.TrimSpace(datasetProfile.ServerPath)
		configPayload["dataset_adapter"] = datasetAdapter
	}
	if err := writeJSON(remoteConfigSourcePath, configPayload); err != nil {
		return err
	}
	remoteManifestSourcePath := filepath.Join(localRunDir, "remote_experiment_manifest.json")
	manifestPayload := map[string]any{
		"protocol_version":         "phase4-retrieval-mainline-v1",
		"run_id":                   runManifest.ID,
		"dataset_profile_id":       runManifest.DatasetProfileID,
		"idea_id":                  runManifest.IdeaID,
		"reader_context_id":        runManifest.ReaderContextID,
		"run_dir":                  remotePaths.RunDir,
		"snapshot_dir":             remotePaths.SnapshotDir,
		"artifact_dir":             remotePaths.ArtifactDir,
		"logs_dir":                 remotePaths.LogsDir,
		"config_path":              remotePaths.RemoteConfigPath,
		"metrics_path":             remotePaths.RemoteMetricsPath,
		"predictions_path":         remotePaths.RemotePredictionsPath,
		"machine_report_path":      remotePaths.RemoteMachineReportPath,
		"human_report_path":        remotePaths.RemoteHumanReportPath,
		"dataset_tool_asset_path":  remotePaths.RemoteDatasetToolAssetPath,
		"dataset_adapter_path":     remotePaths.RemoteDatasetAdapterPath,
		"evaluate_tool_asset_path": remotePaths.RemoteEvaluateToolAssetPath,
		"eval_summary_path":        remotePaths.RemoteEvalSummaryPath,
		"bootstrap_script_path":    remotePaths.RemoteBootstrapScriptPath,
		"metadata": map[string]any{
			"dataset_name":            datasetProfile.DatasetName,
			"phase4_remote_work_root": remotePaths.MRAGRoot,
			"selected_gpu_index":      selection.GPUIndex,
			"selected_gpu_name":       selection.GPUName,
		},
	}
	if err := writeJSON(remoteManifestSourcePath, manifestPayload); err != nil {
		return err
	}
	remoteBootstrapSourcePath := filepath.Join(localRunDir, "remote_bootstrap.sh")
	if err := os.WriteFile(remoteBootstrapSourcePath, []byte(buildPhase4RemoteBootstrapScript(remotePaths)), 0o644); err != nil {
		return err
	}
	remoteExecuteSourcePath := filepath.Join(localRunDir, "remote_execute.sh")
	if err := os.WriteFile(remoteExecuteSourcePath, []byte(buildPhase4RemoteExecuteScript(remotePaths, selection.GPUIndex)), 0o644); err != nil {
		return err
	}
	artifactPaths["remote_manifest_source_path"] = remoteManifestSourcePath
	artifactPaths["remote_config_source_path"] = remoteConfigSourcePath
	artifactPaths["remote_bootstrap_source_path"] = remoteBootstrapSourcePath
	artifactPaths["remote_execute_source_path"] = remoteExecuteSourcePath
	artifactPaths["run_log_path"] = filepath.Join(localLogsDir, "run.log")
	artifactPaths["bootstrap_stdout_path"] = filepath.Join(localLogsDir, "bootstrap.stdout.log")
	artifactPaths["bootstrap_stderr_path"] = filepath.Join(localLogsDir, "bootstrap.stderr.log")
	artifactPaths["runtime_stdout_path"] = filepath.Join(localLogsDir, "runtime.stdout.log")
	artifactPaths["runtime_stderr_path"] = filepath.Join(localLogsDir, "runtime.stderr.log")
	return nil
}

func (s *Phase4Service) attachPhase4RemoteArtifactPaths(artifactPaths map[string]any, remotePaths phase4RemotePaths, selection phase4GPUSelection, probe *model.GPUProbeResult) map[string]any {
	out := artifactPaths
	if out == nil {
		out = map[string]any{}
	}
	out["phase4_remote_work_root"] = remotePaths.MRAGRoot
	out["remote_run_dir"] = remotePaths.RunDir
	out["remote_snapshot_dir"] = remotePaths.SnapshotDir
	out["remote_logs_dir"] = remotePaths.LogsDir
	out["remote_artifact_dir"] = remotePaths.ArtifactDir
	out["remote_env_dir"] = remotePaths.EnvDir
	out["remote_manifest_path"] = remotePaths.RemoteManifestPath
	out["remote_config_path"] = remotePaths.RemoteConfigPath
	out["remote_bootstrap_script_path"] = remotePaths.RemoteBootstrapScriptPath
	out["remote_execute_script_path"] = remotePaths.RemoteExecuteScriptPath
	out["remote_gpu_lock_path"] = remotePaths.GPULockPath
	out["selected_gpu_index"] = selection.GPUIndex
	out["selected_gpu_name"] = selection.GPUName
	if probe != nil {
		out["gpu_probe_summary"] = probe.Summary
	}
	return out
}

func (s *Phase4Service) resolvePhase4ShenzhenlabServer(ctx context.Context, serverID string) (*model.Server, error) {
	if s.servers == nil {
		return nil, fmt.Errorf("phase4 server registry is not configured")
	}
	if serverID != "" {
		server, err := s.servers.GetByIDWithSecrets(ctx, serverID)
		if err != nil {
			return nil, err
		}
		if err = validatePhase4Server(server); err != nil {
			return nil, err
		}
		return server, nil
	}
	items, err := s.servers.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if !strings.EqualFold(strings.TrimSpace(item.Name), phase4ShenzhenlabServerName) {
			continue
		}
		server, getErr := s.servers.GetByIDWithSecrets(ctx, item.ID)
		if getErr != nil {
			return nil, getErr
		}
		if err = validatePhase4Server(server); err != nil {
			return nil, err
		}
		return server, nil
	}
	return nil, fmt.Errorf("server %s is not registered", phase4ShenzhenlabServerName)
}

func phase4ShellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func maxDuration(values ...time.Duration) time.Duration {
	var best time.Duration
	for _, value := range values {
		if value > best {
			best = value
		}
	}
	return best
}

func phase4DecodeJSONLine(raw string) map[string]any {
	var payload map[string]any
	_ = json.Unmarshal([]byte(raw), &payload)
	return payload
}
