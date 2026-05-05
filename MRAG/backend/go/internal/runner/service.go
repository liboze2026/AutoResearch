package runner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/service"
	"mrag-platform/backend/go/internal/traintemplate"
)

type runReader interface {
	GetByID(context.Context, string) (*model.ExperimentRun, error)
	Update(context.Context, model.ExperimentRun) error
}

type experimentReader interface {
	GetByID(context.Context, string) (*model.Experiment, error)
}

type specReader interface {
	GetLatestByExperimentID(context.Context, string) (*model.ExperimentSpec, error)
	GetByID(context.Context, string) (*model.ExperimentSpec, error)
}

type serverSecretReader interface {
	GetByIDWithSecrets(context.Context, string) (*model.Server, error)
}

type runLogWriter interface {
	Add(context.Context, model.RunLog) error
	ListByRunID(context.Context, string) ([]model.RunLog, error)
}

type Service struct {
	runs           runReader
	experiments    experimentReader
	specs          specReader
	servers        serverSecretReader
	logs           runLogWriter
	ssh            service.SSHGateway
	templates      *traintemplate.Service
	comparer       runComparer
	remoteWorkRoot string
}

type runComparer interface {
	CompareRun(context.Context, string) (*model.RunCompareResult, error)
}

func NewService(
	runs runReader,
	experiments experimentReader,
	specs specReader,
	servers serverSecretReader,
	logs runLogWriter,
	ssh service.SSHGateway,
	templates *traintemplate.Service,
	comparer runComparer,
	remoteWorkRoot string,
) *Service {
	return &Service{
		runs:           runs,
		experiments:    experiments,
		specs:          specs,
		servers:        servers,
		logs:           logs,
		ssh:            ssh,
		templates:      templates,
		comparer:       comparer,
		remoteWorkRoot: strings.TrimSpace(remoteWorkRoot),
	}
}

func (s *Service) GetRun(ctx context.Context, runID string) (*model.ExperimentRun, error) {
	return s.runs.GetByID(ctx, runID)
}

func (s *Service) StartRun(ctx context.Context, runID string) (*model.ExperimentRun, error) {
	run, err := s.runs.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("run not found")
	}
	if run.RunStatus != "scheduled" && run.RunStatus != "queued" {
		return nil, fmt.Errorf("run is not startable")
	}

	exp, err := s.experiments.GetByID(ctx, run.ExperimentID)
	if err != nil {
		return nil, err
	}
	if exp == nil {
		return nil, fmt.Errorf("experiment not found")
	}

	spec, err := s.loadSpec(ctx, run)
	if err != nil {
		return nil, err
	}
	serverNode, err := s.servers.GetByIDWithSecrets(ctx, run.AssignedServerID)
	if err != nil {
		return nil, err
	}
	if serverNode == nil {
		return nil, fmt.Errorf("assigned server not found")
	}

	run.RunStatus = "preparing"
	run.UpdatedAt = time.Now()
	if err := s.runs.Update(ctx, *run); err != nil {
		return nil, err
	}

	runNumber := parseRunNumber(run.RemoteWorkdir)
	prepared, err := s.templates.PrepareRunDir(ctx, exp.ID, runNumber, spec.SpecJSON)
	if err != nil {
		return s.failRun(ctx, run, "prepare_local", err, nil, "")
	}

	remoteDir := s.remoteRunDir(serverNode, exp.ID, filepath.Base(prepared.RunDir))
	if err := s.prepareRemoteDir(ctx, serverNode, remoteDir); err != nil {
		return s.failRun(ctx, run, "prepare_remote", err, nil, "")
	}
	if err := s.uploadRunDir(ctx, serverNode, prepared.RunDir, remoteDir); err != nil {
		return s.failRun(ctx, run, "upload_files", err, nil, "")
	}

	startedAt := time.Now()
	run.RunStatus = "running"
	run.StartedAt = &startedAt
	run.RemoteWorkdir = remoteDir
	run.ResultJSON = mergeRunResult(run.ResultJSON, map[string]interface{}{
		"local_run_dir":  prepared.RunDir,
		"remote_run_dir": remoteDir,
	})
	run.UpdatedAt = startedAt
	if err := s.runs.Update(ctx, *run); err != nil {
		return nil, err
	}

	execResult, err := s.ssh.Exec(ctx, serverNode, service.SSHExecRequest{
		Purpose:       "experiment_run_start",
		RemoteCommand: []string{"sh", "-lc", fmt.Sprintf("cd %s && python3 runner.py --spec spec.json --output-dir outputs", shellQuote(remoteDir))},
		Timeout:       60 * time.Second,
	})
	if err != nil {
		return s.failRun(ctx, run, "start_runner", err, nil, "")
	}

	stdoutText, _ := s.readRemoteFile(ctx, serverNode, path.Join(remoteDir, "outputs", "stdout.log"))
	stderrText, _ := s.readRemoteFile(ctx, serverNode, path.Join(remoteDir, "outputs", "stderr.log"))
	metricsText, metricsErr := s.readRemoteFile(ctx, serverNode, path.Join(remoteDir, "outputs", "metrics.json"))
	resultMD, _ := s.readRemoteFile(ctx, serverNode, path.Join(remoteDir, "outputs", "result.md"))

	if strings.TrimSpace(stdoutText) == "" {
		stdoutText = execResult.Stdout
	}
	if strings.TrimSpace(stderrText) == "" {
		stderrText = execResult.Stderr
	}
	lastSummary := tailText(firstNonEmpty(stderrText, stdoutText, execResult.Stderr, execResult.Stdout), 4000)

	if err := s.persistLog(ctx, run.ID, "stdout", path.Join(remoteDir, "outputs", "stdout.log"), stdoutText); err != nil {
		return nil, err
	}
	if err := s.persistLog(ctx, run.ID, "stderr", path.Join(remoteDir, "outputs", "stderr.log"), stderrText); err != nil {
		return nil, err
	}

	localOutputDir := filepath.Join(prepared.RunDir, "outputs")
	_ = os.WriteFile(filepath.Join(localOutputDir, "stdout.log"), []byte(stdoutText), 0o644)
	_ = os.WriteFile(filepath.Join(localOutputDir, "stderr.log"), []byte(stderrText), 0o644)
	if metricsErr == nil && strings.TrimSpace(metricsText) != "" {
		_ = os.WriteFile(filepath.Join(localOutputDir, "metrics.json"), []byte(metricsText), 0o644)
	}
	if strings.TrimSpace(resultMD) != "" {
		_ = os.WriteFile(filepath.Join(localOutputDir, "result.md"), []byte(resultMD), 0o644)
	}

	finishedAt := time.Now()
	run.EndedAt = &finishedAt
	run.UpdatedAt = finishedAt
	run.ExitCode = &execResult.ExitCode

	resultJSON := mergeRunResult(run.ResultJSON, map[string]interface{}{
		"remote_run_dir": remoteDir,
		"local_run_dir":  prepared.RunDir,
		"artifacts": map[string]interface{}{
			"metrics_path": filepath.Join(localOutputDir, "metrics.json"),
			"result_path":  filepath.Join(localOutputDir, "result.md"),
			"stdout_path":  filepath.Join(localOutputDir, "stdout.log"),
			"stderr_path":  filepath.Join(localOutputDir, "stderr.log"),
		},
	})
	if metricsErr == nil && strings.TrimSpace(metricsText) != "" {
		var metrics map[string]interface{}
		if err := json.Unmarshal([]byte(metricsText), &metrics); err == nil {
			resultJSON["metrics"] = metrics
		}
	}
	run.ResultJSON = resultJSON

	if execResult.ExitCode != 0 {
		exitCode := execResult.ExitCode
		return s.failRun(ctx, run, "runner_exit", fmt.Errorf(firstNonEmpty(stderrText, execResult.Stderr, "remote runner failed")), &exitCode, lastSummary)
	}
	if strings.TrimSpace(metricsText) == "" || strings.TrimSpace(resultMD) == "" {
		return s.failRun(ctx, run, "collect_outputs", fmt.Errorf("required output files are missing"), &execResult.ExitCode, lastSummary)
	}

	run.RunStatus = "succeeded"
	run.ErrorMessage = ""
	if err := s.runs.Update(ctx, *run); err != nil {
		return nil, err
	}
	if s.comparer != nil {
		if _, err := s.comparer.CompareRun(ctx, run.ID); err != nil {
			run.ResultJSON = mergeRunResult(run.ResultJSON, map[string]interface{}{
				"comparison_status": "failed",
				"comparison_error":  err.Error(),
			})
			run.UpdatedAt = time.Now()
			_ = s.runs.Update(ctx, *run)
		}
	}
	return run, nil
}

func (s *Service) loadSpec(ctx context.Context, run *model.ExperimentRun) (*model.ExperimentSpec, error) {
	if strings.TrimSpace(run.SpecID) != "" {
		spec, err := s.specs.GetByID(ctx, run.SpecID)
		if err != nil {
			return nil, err
		}
		if spec != nil {
			return spec, nil
		}
	}
	spec, err := s.specs.GetLatestByExperimentID(ctx, run.ExperimentID)
	if err != nil {
		return nil, err
	}
	if spec == nil {
		return nil, fmt.Errorf("experiment spec not found")
	}
	return spec, nil
}

func (s *Service) remoteRunDir(server *model.Server, experimentID string, runDirName string) string {
	base := strings.TrimSpace(server.TaskWorkdir)
	if base == "" {
		base = strings.TrimSpace(server.RemoteRoot)
	}
	if base == "" {
		base = s.remoteWorkRoot
	}
	if base == "" {
		base = "/tmp/mrag"
	}
	return path.Join(filepath.ToSlash(base), "experiments", experimentID, runDirName)
}

func (s *Service) prepareRemoteDir(ctx context.Context, server *model.Server, remoteDir string) error {
	result, err := s.ssh.Exec(ctx, server, service.SSHExecRequest{
		Purpose:       "experiment_run_prepare",
		RemoteCommand: []string{"sh", "-lc", fmt.Sprintf("mkdir -p %s %s", shellQuote(remoteDir), shellQuote(path.Join(remoteDir, "outputs")))},
		Timeout:       20 * time.Second,
	})
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, "prepare remote dir failed"))
	}
	return nil
}

func (s *Service) uploadRunDir(ctx context.Context, server *model.Server, localDir string, remoteDir string) error {
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
			result, err := s.ssh.Exec(ctx, server, service.SSHExecRequest{
				Purpose:       "experiment_run_prepare",
				RemoteCommand: []string{"sh", "-lc", fmt.Sprintf("mkdir -p %s", shellQuote(remotePath))},
				Timeout:       20 * time.Second,
			})
			if err != nil {
				return err
			}
			if result.ExitCode != 0 {
				return fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, "mkdir failed"))
			}
			return nil
		}
		if strings.Contains(filepath.ToSlash(rel), "/outputs/") {
			return nil
		}
		data, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		payload := base64.StdEncoding.EncodeToString(data)
		result, err := s.ssh.Exec(ctx, server, service.SSHExecRequest{
			Purpose:       "experiment_run_upload",
			RemoteCommand: []string{"python3", "-c", "import base64, pathlib, sys; p=pathlib.Path(sys.argv[1]); p.parent.mkdir(parents=True, exist_ok=True); p.write_bytes(base64.b64decode(sys.stdin.buffer.read()))", remotePath},
			Stdin:         payload,
			Timeout:       30 * time.Second,
		})
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, "upload failed"))
		}
		return nil
	})
}

func (s *Service) readRemoteFile(ctx context.Context, server *model.Server, remotePath string) (string, error) {
	result, err := s.ssh.Exec(ctx, server, service.SSHExecRequest{
		Purpose:       "experiment_run_read_file",
		RemoteCommand: []string{"sh", "-lc", fmt.Sprintf("if [ -f %s ]; then cat %s; fi", shellQuote(remotePath), shellQuote(remotePath))},
		Timeout:       20 * time.Second,
	})
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, "read remote file failed"))
	}
	return result.Stdout, nil
}

func (s *Service) persistLog(ctx context.Context, runID string, logType string, logPath string, content string) error {
	now := time.Now()
	return s.logs.Add(ctx, model.RunLog{
		RunID:     runID,
		LogType:   logType,
		LogPath:   logPath,
		TailText:  tailText(content, 4000),
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *Service) failRun(ctx context.Context, run *model.ExperimentRun, stage string, err error, exitCode *int, lastSummary string) (*model.ExperimentRun, error) {
	now := time.Now()
	run.RunStatus = "failed"
	run.ErrorMessage = err.Error()
	run.ExitCode = exitCode
	run.EndedAt = &now
	run.UpdatedAt = now
	run.ResultJSON = mergeRunResult(run.ResultJSON, map[string]interface{}{
		"failure_stage":    stage,
		"last_log_summary": tailText(lastSummary, 4000),
		"recovery": map[string]interface{}{
			"suggest_retry": suggestRetry(stage),
		},
	})
	_ = s.runs.Update(ctx, *run)
	return nil, err
}

func suggestRetry(stage string) bool {
	switch stage {
	case "prepare_remote", "upload_files", "start_runner", "collect_outputs":
		return true
	default:
		return false
	}
}

func parseRunNumber(runPath string) int {
	base := filepath.Base(filepath.Clean(runPath))
	if strings.HasPrefix(base, "run_") {
		if value, err := strconv.Atoi(strings.TrimPrefix(base, "run_")); err == nil && value > 0 {
			return value
		}
	}
	return 1
}

func tailText(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mergeRunResult(existing map[string]interface{}, updates map[string]interface{}) map[string]interface{} {
	if existing == nil {
		existing = map[string]interface{}{}
	}
	for key, value := range updates {
		existing[key] = value
	}
	return existing
}
