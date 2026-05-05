package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"mrag-platform/backend/go/internal/model"
)

type SSHExecRequest struct {
	Purpose       string
	RemoteCommand []string
	Stdin         string
	Metadata      map[string]string
	Timeout       time.Duration
}

type SSHExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

type SSHGateway interface {
	Mode() string
	Probe(ctx context.Context, node *model.Server) (*model.ServerConnectionTestResult, error)
	Exec(ctx context.Context, node *model.Server, req SSHExecRequest) (*SSHExecResult, error)
}

func NewSSHGateway(binary string, mode string, connectTimeoutSec int) SSHGateway {
	if strings.EqualFold(strings.TrimSpace(mode), "mock") {
		return &MockSSHGateway{}
	}
	if connectTimeoutSec <= 0 {
		connectTimeoutSec = 4
	}
	return &SystemSSHGateway{
		binary:         strings.TrimSpace(binary),
		connectTimeout: time.Duration(connectTimeoutSec) * time.Second,
	}
}

type SystemSSHGateway struct {
	binary         string
	connectTimeout time.Duration
}

type resolvedSSHTarget struct {
	Alias        string
	Host         string
	Port         int
	User         string
	ProxyCommand string
}

type pipeAddr string

func (a pipeAddr) Network() string { return "ssh-proxy" }
func (a pipeAddr) String() string  { return string(a) }

type proxyCommandConn struct {
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	cmd       *exec.Cmd
	stderr    bytes.Buffer
	address   string
	closeOnce sync.Once
}

func (c *proxyCommandConn) Read(p []byte) (int, error)  { return c.stdout.Read(p) }
func (c *proxyCommandConn) Write(p []byte) (int, error) { return c.stdin.Write(p) }

func (c *proxyCommandConn) Close() error {
	c.closeOnce.Do(func() {
		if c.stdin != nil {
			_ = c.stdin.Close()
		}
		if c.stdout != nil {
			_ = c.stdout.Close()
		}
		if c.cmd != nil {
			if c.cmd.Process != nil {
				_ = c.cmd.Process.Kill()
			}
			_ = c.cmd.Wait()
		}
	})
	return nil
}

func (c *proxyCommandConn) LocalAddr() net.Addr                { return pipeAddr("local-proxy") }
func (c *proxyCommandConn) RemoteAddr() net.Addr               { return pipeAddr(c.address) }
func (c *proxyCommandConn) SetDeadline(t time.Time) error      { return nil }
func (c *proxyCommandConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *proxyCommandConn) SetWriteDeadline(t time.Time) error { return nil }
func (c *proxyCommandConn) StderrString() string               { return strings.TrimSpace(c.stderr.String()) }

func (g *SystemSSHGateway) Mode() string {
	return "real"
}

func (g *SystemSSHGateway) Probe(ctx context.Context, node *model.Server) (*model.ServerConnectionTestResult, error) {
	probeTimeout := g.connectTimeout + 15*time.Second
	if probeTimeout < 20*time.Second {
		probeTimeout = 20 * time.Second
	}
	probeResult, err := g.Exec(ctx, node, SSHExecRequest{
		Purpose:       "probe",
		RemoteCommand: []string{"sh", "-s", "--", "probe"},
		Stdin:         "printf '__MRAG_SSH_OK__:%s:%s\n' \"$(hostname)\" \"$(whoami)\"\n",
		Timeout:       probeTimeout,
	})
	if err != nil {
		return nil, err
	}
	checkedAt := time.Now()
	result := &model.ServerConnectionTestResult{
		ServerID:   node.ID,
		ServerName: node.Name,
		Mode:       g.Mode(),
		Target:     buildSSHTarget(node),
		Stdout:     probeResult.Stdout,
		Stderr:     probeResult.Stderr,
		ExitCode:   probeResult.ExitCode,
		LatencyMs:  probeResult.Duration.Milliseconds(),
		CheckedAt:  checkedAt,
	}
	if host, user, ok := parseProbeToken(probeResult.Stdout); ok && probeResult.ExitCode == 0 {
		result.Result = "login_success"
		result.Reachable = true
		result.Message = "SSH \u767b\u5f55\u6210\u529f\uff0c\u5df2\u5efa\u7acb\u975e\u4ea4\u4e92\u4f1a\u8bdd\u3002"
		result.RemoteHost = host
		result.RemoteUser = user
		return result, nil
	}
	result.Result = classifySSHFailure(probeResult.Stderr, probeResult.ExitCode)
	result.Message = probeFailureMessage(result.Result)
	return result, nil
}

func (g *SystemSSHGateway) Exec(ctx context.Context, node *model.Server, req SSHExecRequest) (*SSHExecResult, error) {
	if strings.TrimSpace(node.Host) == "" {
		return nil, fmt.Errorf("ssh target is empty")
	}
	if strings.EqualFold(strings.TrimSpace(node.AuthType), "password") {
		return g.execWithPassword(ctx, node, req)
	}
	if strings.TrimSpace(g.binary) == "" {
		g.binary = "ssh"
	}

	execCtx := ctx
	cancel := func() {}
	if req.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, req.Timeout)
	}
	defer cancel()

	args := g.buildArgs(node, req.RemoteCommand)
	cmd := exec.CommandContext(execCtx, g.binary, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if req.Stdin != "" {
		cmd.Stdin = strings.NewReader(req.Stdin)
	}

	startedAt := time.Now()
	runErr := cmd.Run()
	result := &SSHExecResult{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		Duration: time.Since(startedAt),
	}
	if runErr == nil {
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}
	var execErr *exec.Error
	if errors.As(runErr, &execErr) {
		return nil, fmt.Errorf("ssh client unavailable: %w", runErr)
	}
	if execCtx.Err() != nil {
		return nil, execCtx.Err()
	}
	return nil, runErr
}

func (g *SystemSSHGateway) execWithPassword(ctx context.Context, node *model.Server, req SSHExecRequest) (*SSHExecResult, error) {
	if strings.TrimSpace(node.Password) == "" {
		return nil, fmt.Errorf("password auth selected but no password is stored")
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = g.connectTimeout + 10*time.Second
	}

	target, err := g.resolveSSHTarget(ctx, node)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(target.Host) == "" {
		return nil, fmt.Errorf("resolved ssh hostname is empty")
	}
	if strings.TrimSpace(target.User) == "" {
		return nil, fmt.Errorf("username is required for password auth")
	}
	if target.Port <= 0 {
		target.Port = 22
	}

	address := net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	config := &ssh.ClientConfig{
		User: target.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(node.Password),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range questions {
					answers[i] = node.Password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	startedAt := time.Now()
	conn, proxyConn, err := g.openTargetConn(ctx, target)
	if err != nil {
		return nil, err
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		_ = conn.Close()
		if proxyConn != nil && proxyConn.StderrString() != "" {
			return nil, fmt.Errorf("%s", proxyConn.StderrString())
		}
		return nil, err
	}
	client := ssh.NewClient(clientConn, chans, reqs)
	defer client.Close()
	if proxyConn != nil {
		defer proxyConn.Close()
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if req.Stdin != "" {
		session.Stdin = strings.NewReader(req.Stdin)
	}

	command := shellJoin(req.RemoteCommand)
	runDone := make(chan error, 1)
	go func() {
		if command == "" {
			runDone <- session.Run("true")
			return
		}
		runDone <- session.Run(command)
	}()

	var runErr error
	select {
	case <-ctx.Done():
		_ = client.Close()
		return nil, ctx.Err()
	case <-time.After(timeout):
		_ = client.Close()
		return nil, context.DeadlineExceeded
	case runErr = <-runDone:
	}

	result := &SSHExecResult{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		Duration: time.Since(startedAt),
	}
	if runErr == nil {
		return result, nil
	}

	var exitErr *ssh.ExitError
	if errors.As(runErr, &exitErr) {
		result.ExitCode = exitErr.ExitStatus()
		return result, nil
	}
	if errors.Is(runErr, io.EOF) {
		return result, nil
	}
	return nil, runErr
}

func (g *SystemSSHGateway) openTargetConn(ctx context.Context, target resolvedSSHTarget) (net.Conn, *proxyCommandConn, error) {
	address := net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	if strings.TrimSpace(target.ProxyCommand) == "" || strings.EqualFold(strings.TrimSpace(target.ProxyCommand), "none") {
		conn, err := net.DialTimeout("tcp", address, g.connectTimeout)
		if err != nil {
			return nil, nil, err
		}
		return conn, nil, nil
	}

	expanded := expandProxyCommand(target.ProxyCommand, target)
	proxyConn, err := g.startProxyCommand(ctx, expanded, address)
	if err != nil {
		return nil, nil, err
	}
	return proxyConn, proxyConn, nil
}

func (g *SystemSSHGateway) startProxyCommand(ctx context.Context, command string, address string) (*proxyCommandConn, error) {
	shellName, shellArgs := localShellCommand(command)
	cmd := exec.CommandContext(ctx, shellName, shellArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	conn := &proxyCommandConn{
		stdin:   stdin,
		stdout:  stdout,
		cmd:     cmd,
		address: address,
	}
	cmd.Stderr = &conn.stderr
	if err = cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, err
	}
	return conn, nil
}

func localShellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/d", "/c", command}
	}
	return "sh", []string{"-lc", command}
}

func (g *SystemSSHGateway) resolveSSHTarget(ctx context.Context, node *model.Server) (resolvedSSHTarget, error) {
	target := resolvedSSHTarget{
		Alias: strings.TrimSpace(node.Host),
		Host:  strings.TrimSpace(node.Host),
		Port:  node.SSHPort,
		User:  strings.TrimSpace(node.Username),
	}
	if target.Port <= 0 {
		target.Port = 22
	}
	if target.Alias == "" {
		return target, nil
	}

	if strings.TrimSpace(g.binary) == "" {
		g.binary = "ssh"
	}

	resolveCtx := ctx
	cancel := func() {}
	if g.connectTimeout > 0 {
		resolveCtx, cancel = context.WithTimeout(ctx, g.connectTimeout)
	}
	defer cancel()

	cmd := exec.CommandContext(resolveCtx, g.binary, "-G", target.Alias)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if target.Host != "" {
			return target, nil
		}
		return target, err
	}

	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		switch key {
		case "hostname":
			if value != "" {
				target.Host = value
			}
		case "port":
			if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
				target.Port = parsed
			}
		case "user":
			if target.User == "" && value != "" {
				target.User = value
			}
		case "proxycommand":
			if value != "" {
				target.ProxyCommand = value
			}
		}
	}

	return target, nil
}

func expandProxyCommand(template string, target resolvedSSHTarget) string {
	replacer := strings.NewReplacer(
		"%%", "%",
		"%h", target.Host,
		"%p", strconv.Itoa(target.Port),
		"%r", target.User,
	)
	return replacer.Replace(template)
}

func (g *SystemSSHGateway) buildArgs(node *model.Server, remoteCommand []string) []string {
	args := []string{
		"-T",
		"-o", "BatchMode=yes",
		"-o", fmt.Sprintf("ConnectTimeout=%d", int(g.connectTimeout.Seconds())),
	}
	target := strings.TrimSpace(node.Host)
	if !usesSSHConfig(node) {
		if node.SSHPort > 0 {
			args = append(args, "-p", strconv.Itoa(node.SSHPort))
		}
		if strings.TrimSpace(node.Username) != "" && !strings.Contains(target, "@") {
			target = node.Username + "@" + target
		}
	}
	args = append(args, target)
	args = append(args, remoteCommand...)
	return args
}

type MockSSHGateway struct{}

func (g *MockSSHGateway) Mode() string {
	return "mock"
}

func (g *MockSSHGateway) Probe(ctx context.Context, node *model.Server) (*model.ServerConnectionTestResult, error) {
	lowerHost := strings.ToLower(node.Host)
	result := &model.ServerConnectionTestResult{
		ServerID:   node.ID,
		ServerName: node.Name,
		Mode:       g.Mode(),
		Target:     buildSSHTarget(node),
		LatencyMs:  18,
		CheckedAt:  time.Now(),
	}
	switch {
	case strings.Contains(lowerHost, "auth"):
		result.Result = "auth_failed"
		result.Message = "\u6f14\u793a\u6a21\u5f0f\uff1a\u6a21\u62df SSH \u8ba4\u8bc1\u5931\u8d25\u3002"
	case strings.Contains(lowerHost, "handshake"):
		result.Result = "handshake_failed"
		result.Message = "\u6f14\u793a\u6a21\u5f0f\uff1a\u6a21\u62df SSH \u63e1\u624b\u5931\u8d25\u3002"
	case strings.Contains(lowerHost, "offline") || strings.Contains(lowerHost, "unreachable"):
		result.Result = "host_unreachable"
		result.Message = "\u6f14\u793a\u6a21\u5f0f\uff1a\u6a21\u62df\u4e3b\u673a\u4e0d\u53ef\u8fbe\u3002"
	default:
		result.Result = "login_success"
		result.Reachable = true
		result.Message = "\u6f14\u793a\u6a21\u5f0f\uff1a\u6a21\u62df SSH \u767b\u5f55\u6210\u529f\u3002"
		result.RemoteHost = fmt.Sprintf("mock-%s", node.Name)
		if strings.TrimSpace(node.Username) != "" {
			result.RemoteUser = node.Username
		} else {
			result.RemoteUser = "demo"
		}
		result.Stdout = fmt.Sprintf("__MRAG_SSH_OK__:%s:%s", result.RemoteHost, result.RemoteUser)
	}
	return result, nil
}

func (g *MockSSHGateway) Exec(ctx context.Context, node *model.Server, req SSHExecRequest) (*SSHExecResult, error) {
	now := time.Now().Format(time.RFC3339)
	rootPath := firstNonEmpty(req.Metadata["rootPath"], node.RemoteRoot, "/srv/mrag/datasets")
	taskID := firstNonEmpty(req.Metadata["taskId"], "mock-task")
	switch req.Purpose {
	case "server_command":
		return &SSHExecResult{Stdout: fmt.Sprintf("demo command executed on %s", buildSSHTarget(node)), ExitCode: 0, Duration: 20 * time.Millisecond}, nil
	case "check_gpu":
		return &SSHExecResult{Stdout: fmt.Sprintf(`{"mode":"mock","summary":"mock gpu probe detected 2 gpus, 1 available","availableGpuCount":1,"totalGpuCount":2,"checkedAt":"%s","devices":[{"index":0,"name":"RTX 4090","memoryUsedMb":512,"memoryTotalMb":24564,"utilization":3,"processes":0,"available":true},{"index":1,"name":"RTX 4090","memoryUsedMb":12000,"memoryTotalMb":24564,"utilization":78,"processes":2,"available":false}]}`,
			now), ExitCode: 0, Duration: 10 * time.Millisecond}, nil
	case "scan_datasets":
		return &SSHExecResult{Stdout: fmt.Sprintf(`{"serverId":"%s","serverName":"%s","mode":"mock","rootPath":"%s","scannedAt":"%s","candidates":[{"name":"mmrag-sample","path":"%s/mmrag-sample","size":"2.3 GB","totalSizeBytes":2469606195,"fileCount":1320,"directoryCount":48,"lastModifiedAt":"%s","modality":"multimodal","status":"new","description":"濠曟梻銇氬Ο鈥崇础閹殿偅寮跨紒鎾寸亯"},{"name":"papers-demo","path":"%s/papers-demo","size":"846.5 MB","totalSizeBytes":887619584,"fileCount":420,"directoryCount":19,"lastModifiedAt":"%s","modality":"text","status":"new","description":"濠曟梻銇氬Ο鈥崇础閹殿偅寮跨紒鎾寸亯"}]}`,
			node.ID, node.Name, rootPath, now, rootPath, now, rootPath, now), ExitCode: 0, Duration: 15 * time.Millisecond}, nil
	case "dataset_validate":
		return &SSHExecResult{Stdout: `{"valid":true,"exists":true,"isDirectory":true,"message":"Mock remote dataset directory is available"}`, ExitCode: 0, Duration: 10 * time.Millisecond}, nil
	case "dataset_scan":
		return &SSHExecResult{Stdout: `{"validationStatus":"ok","scanStatus":"completed","fileCount":23,"directoryCount":4,"totalSizeBytes":2304000,"fileTypes":{"text":14,"image":4,"json":2,"pdf":3},"hierarchySummary":[{"level":0,"path":"docs","itemCount":12},{"level":0,"path":"images","itemCount":11},{"level":1,"path":"docs/train","itemCount":6}],"inferredModality":"multimodal","recentModifiedAt":"` + now + `","previewItems":[{"name":"docs","itemType":"directory","category":"directory","relativePath":"docs","sizeBytes":0,"depth":0},{"name":"readme.md","itemType":"file","category":"text","relativePath":"docs/readme.md","sizeBytes":4096,"depth":1}],"errorMessage":""}`, ExitCode: 0, Duration: 10 * time.Millisecond}, nil
	case "dataset_index_start":
		return &SSHExecResult{Stdout: fmt.Sprintf(`{"taskId":"%s","status":"building","logPath":"%s/dataset-index-tasks/%s/logs/runtime.log","statusPath":"%s/dataset-index-tasks/%s/status.json","resultPath":"%s/dataset-index-tasks/%s/result.json","message":"mock remote index accepted","logs":["mock remote index accepted"]}`,
			taskID, rootPath, taskID, rootPath, taskID, rootPath, taskID), ExitCode: 0, Duration: 12 * time.Millisecond}, nil
	case "dataset_index_status":
		return &SSHExecResult{Stdout: fmt.Sprintf(`{"taskId":"%s","status":"completed","logPath":"%s/dataset-index-tasks/%s/logs/runtime.log","statusPath":"%s/dataset-index-tasks/%s/status.json","resultPath":"%s/dataset-index-tasks/%s/result.json","message":"mock remote index completed","logs":["mock remote index completed"]}`,
			taskID, rootPath, taskID, rootPath, taskID, rootPath, taskID), ExitCode: 0, Duration: 12 * time.Millisecond}, nil
	case "experiment_run_prepare", "experiment_run_upload":
		return &SSHExecResult{Stdout: "mock remote run path prepared", ExitCode: 0, Duration: 8 * time.Millisecond}, nil
	case "experiment_run_start":
		return &SSHExecResult{
			Stdout:   "[mock-train] start model=mock/model\n[mock-train] completed\n[mock-eval] evaluating outputs\n",
			Stderr:   "",
			ExitCode: 0,
			Duration: 25 * time.Millisecond,
		}, nil
	case "experiment_run_read_file":
		command := ""
		if len(req.RemoteCommand) > 0 {
			command = req.RemoteCommand[len(req.RemoteCommand)-1]
		}
		switch {
		case strings.Contains(command, "metrics.json"):
			return &SSHExecResult{Stdout: `{"primary_metric":"accuracy","values":{"accuracy":0.88,"loss":0.12}}`, ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.Contains(command, "result.md"):
			return &SSHExecResult{Stdout: "# Mock Result\n\n- accuracy: 0.88\n- loss: 0.12\n", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.Contains(command, "stdout.log"):
			return &SSHExecResult{Stdout: "[mock-train] start model=mock/model\n[mock-train] completed\n[mock-eval] evaluating outputs\n", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.Contains(command, "stderr.log"):
			return &SSHExecResult{Stdout: "", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		default:
			return &SSHExecResult{Stdout: "", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		}
	case "phase4_remote_prepare", "phase4_remote_upload", "phase4_remote_bootstrap", "phase4_remote_release_lock":
		return &SSHExecResult{Stdout: "mock phase4 remote command completed", ExitCode: 0, Duration: 8 * time.Millisecond}, nil
	case "phase4_remote_run":
		return &SSHExecResult{
			Stdout:   "mock phase4 remote run completed",
			Stderr:   "",
			ExitCode: 0,
			Duration: 20 * time.Millisecond,
		}, nil
	case "phase4_remote_read_file":
		remotePath := firstNonEmpty(req.Metadata["remotePath"], commandTail(req.RemoteCommand))
		switch {
		case strings.HasSuffix(remotePath, "/metrics.json"):
			return &SSHExecResult{Stdout: `{"protocol_version":"phase4-retrieval-mainline-v1","run_id":"p4run_1","primary_metric":"recall@5","values":{"recall@5":0.71,"average_candidate_count":4.0,"query_count":3},"status":"succeeded","retrieval_summary":{"method_name":"dummy_retrieval"},"metadata":{"runner_mode":"shenzhenvlab"}}`, ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/machine_report.json"):
			return &SSHExecResult{Stdout: `{"run_id":"p4run_1","status":"succeeded","metrics":{"primary_metric":"recall@5","values":{"recall@5":0.71}}}`, ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/report.md"):
			return &SSHExecResult{Stdout: "# Phase4 Retrieval Evaluation\n\n- Run ID: `p4run_1`\n- Primary Metric: `recall@5`\n", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/dataset_tool_asset.json"):
			return &SSHExecResult{Stdout: `{"dataset_profile_id":"p4ds_1","dataset_name":"VisDoM","server_path":"/home/bzli/mrag/datasets/visdom","official_metric":"recall@5","run_id":"p4run_1"}`, ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/dataset_adapter_contract.json"):
			return &SSHExecResult{Stdout: `{"contract_version":"dataset-adapter-v1","dataset_adapter":{"dataset_profile_id":"p4ds_1","dataset_name":"VisDoM"}}`, ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/evaluate_tool_asset.json"):
			return &SSHExecResult{Stdout: `{"tool":"evaluate_tool","primary_metric":"recall@5","values":{"recall@5":0.71},"status":"succeeded"}`, ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/eval_summary.md"):
			return &SSHExecResult{Stdout: "# Evaluation Summary\n\n- Recall@5: `0.71`\n", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/predictions.json"):
			return &SSHExecResult{Stdout: `{"method_name":"dummy_retrieval","predictions":[{"query_id":"q1","candidates":[{"page_id":"p1","score":0.71}]}]}`, ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/driver.log"):
			return &SSHExecResult{Stdout: "[phase4_remote] run_id=p4run_1\n[phase4_remote] completed\n", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/run.log"):
			return &SSHExecResult{Stdout: "[retrieval_mainline] run started\n[retrieval_mainline] method=dummy_retrieval\n", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/bootstrap.stdout.log"):
			return &SSHExecResult{Stdout: "phase4 retrieval mainline bootstrap complete\n", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/bootstrap.stderr.log"), strings.HasSuffix(remotePath, "/runtime.stderr.log"):
			return &SSHExecResult{Stdout: "", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		case strings.HasSuffix(remotePath, "/runtime.stdout.log"):
			return &SSHExecResult{Stdout: "mock phase4 remote run completed\n", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		default:
			return &SSHExecResult{Stdout: "", ExitCode: 0, Duration: 5 * time.Millisecond}, nil
		}
	default:
		return &SSHExecResult{ExitCode: 0, Duration: 10 * time.Millisecond}, nil
	}
}

func usesSSHConfig(node *model.Server) bool {
	return strings.EqualFold(strings.TrimSpace(node.AuthType), "ssh_config")
}

func buildSSHTarget(node *model.Server) string {
	target := strings.TrimSpace(node.Host)
	if target == "" {
		return ""
	}
	if usesSSHConfig(node) {
		return target
	}
	if strings.TrimSpace(node.Username) != "" && !strings.Contains(target, "@") {
		target = node.Username + "@" + target
	}
	if node.SSHPort > 0 {
		return fmt.Sprintf("%s:%d", target, node.SSHPort)
	}
	return target
}

func shellJoin(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, sshShellQuote(part))
	}
	return strings.Join(quoted, " ")
}

func commandTail(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func sshShellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func parseProbeToken(raw string) (string, string, bool) {
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, "__MRAG_SSH_OK__:") {
			continue
		}
		parts := strings.SplitN(trimmed, "__MRAG_SSH_OK__:", 2)
		if len(parts) != 2 {
			continue
		}
		meta := strings.Split(strings.TrimSpace(parts[1]), ":")
		if len(meta) != 2 {
			continue
		}
		return meta[0], meta[1], true
	}
	return "", "", false
}

func classifySSHFailure(stderr string, exitCode int) string {
	message := strings.ToLower(stderr)
	switch {
	case strings.Contains(message, "permission denied"),
		strings.Contains(message, "authentication failed"),
		strings.Contains(message, "too many authentication failures"):
		return "auth_failed"
	case strings.Contains(message, "no route to host"),
		strings.Contains(message, "could not resolve hostname"),
		strings.Contains(message, "connection timed out"),
		strings.Contains(message, "network is unreachable"),
		strings.Contains(message, "connection refused"),
		strings.Contains(message, "name or service not known"),
		strings.Contains(message, "temporary failure in name resolution"):
		return "host_unreachable"
	default:
		if exitCode == 255 {
			return "handshake_failed"
		}
		return "handshake_failed"
	}
}

func probeFailureMessage(result string) string {
	switch result {
	case "host_unreachable":
		return "\u4e3b\u673a\u4e0d\u53ef\u8fbe\uff0c\u6216 SSH \u7aef\u53e3\u672a\u80fd\u5efa\u7acb\u8fde\u63a5\u3002"
	case "auth_failed":
		return "SSH \u63e1\u624b\u5df2\u5efa\u7acb\uff0c\u4f46\u8ba4\u8bc1\u5931\u8d25\u3002"
	default:
		return "SSH \u63e1\u624b\u672a\u5b8c\u6210\uff0c\u8bf7\u68c0\u67e5\u4e3b\u673a\u5bc6\u94a5\u3001\u8df3\u677f\u94fe\u8def\u6216\u670d\u52a1\u7aef SSH \u914d\u7f6e\u3002"
	}
}
