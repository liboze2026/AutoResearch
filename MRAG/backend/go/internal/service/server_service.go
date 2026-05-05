package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/repository"
)

type ServerService struct {
	repo           *repository.ServerRepository
	ssh            SSHGateway
	commandTimeout time.Duration
}

func NewServerService(repo *repository.ServerRepository, ssh SSHGateway, commandTimeoutSec int) *ServerService {
	if commandTimeoutSec <= 0 {
		commandTimeoutSec = 20
	}
	return &ServerService{
		repo:           repo,
		ssh:            ssh,
		commandTimeout: time.Duration(commandTimeoutSec) * time.Second,
	}
}

func (s *ServerService) List(ctx context.Context) ([]model.Server, error) { return s.repo.List(ctx) }

func (s *ServerService) Get(ctx context.Context, id string) (*model.Server, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ServerService) Create(ctx context.Context, req model.Server) (*model.Server, error) {
	req.Status = "offline"
	return s.repo.Create(ctx, req)
}

func (s *ServerService) Update(ctx context.Context, id string, req model.Server) (*model.Server, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *ServerService) Delete(ctx context.Context, id string) error { return s.repo.Delete(ctx, id) }

func (s *ServerService) TestConnection(ctx context.Context, id string) (*model.ServerConnectionTestResult, error) {
	node, err := s.repo.GetByIDWithSecrets(ctx, id)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, fmt.Errorf("server not found")
	}
	result, err := s.ssh.Probe(ctx, node)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, id, "offline", err.Error())
		return nil, err
	}
	status := "offline"
	if result.Reachable {
		status = "online"
	}
	_ = s.repo.UpdateStatus(ctx, id, status, result.Message)
	return result, nil
}

func (s *ServerService) RefreshStatus(ctx context.Context, id string) (*model.ServerStatusSnapshot, error) {
	probe, err := s.TestConnection(ctx, id)
	if err != nil {
		return nil, err
	}
	status := "offline"
	if probe.Reachable {
		status = "online"
	}
	return &model.ServerStatusSnapshot{
		ServerID:  probe.ServerID,
		Status:    status,
		Message:   probe.Message,
		CheckedAt: probe.CheckedAt,
	}, nil
}

func (s *ServerService) CheckGPU(ctx context.Context, id string) (*model.GPUProbeResult, error) {
	node, err := s.repo.GetByIDWithSecrets(ctx, id)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, fmt.Errorf("server not found")
	}

	result, err := s.ssh.Exec(ctx, node, SSHExecRequest{
		Purpose:       "check_gpu",
		RemoteCommand: []string{"sh", "-lc", gpuProbeScript()},
		Timeout:       s.commandTimeout,
	})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, fmt.Sprintf("gpu probe exited with code %d", result.ExitCode)))
	}

	probe := &model.GPUProbeResult{}
	if err = decodeLastJSONLine(result.Stdout, probe); err != nil {
		return nil, err
	}
	probe.ServerID = node.ID
	probe.ServerName = node.Name
	if probe.Mode == "" {
		probe.Mode = s.ssh.Mode()
	}
	_ = s.repo.UpdateGPUInfo(ctx, id, probe)
	return probe, nil
}

func (s *ServerService) ScanDatasets(ctx context.Context, id string, req model.ServerDatasetScanRequest) (*model.ServerDatasetScanResult, error) {
	node, err := s.repo.GetByIDWithSecrets(ctx, id)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, fmt.Errorf("server not found")
	}

	rootPath := strings.TrimSpace(req.RootPath)
	if rootPath == "" {
		rootPath = strings.TrimSpace(node.RemoteRoot)
	}
	if rootPath == "" {
		return nil, fmt.Errorf("rootPath is required")
	}
	maxDepth := req.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 2
	}

	result, err := s.ssh.Exec(ctx, node, SSHExecRequest{
		Purpose:       "scan_datasets",
		RemoteCommand: []string{"sh", "-lc", scanDatasetsScript(rootPath, maxDepth)},
		Metadata: map[string]string{
			"rootPath": rootPath,
			"maxDepth": strconv.Itoa(maxDepth),
		},
		Timeout: s.commandTimeout * 3,
	})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf(firstNonEmpty(result.Stderr, result.Stdout, fmt.Sprintf("dataset scan exited with code %d", result.ExitCode)))
	}

	scan := &model.ServerDatasetScanResult{}
	if err = decodeLastJSONLine(result.Stdout, scan); err != nil {
		return nil, err
	}
	if scan.ServerID == "" {
		scan.ServerID = node.ID
	}
	if scan.ServerName == "" {
		scan.ServerName = node.Name
	}
	if scan.Mode == "" {
		scan.Mode = s.ssh.Mode()
	}
	if scan.RootPath == "" {
		scan.RootPath = rootPath
	}
	if scan.ScannedAt.IsZero() {
		scan.ScannedAt = time.Now()
	}
	return scan, nil
}

func gpuProbeScript() string {
	return `if ! command -v nvidia-smi >/dev/null 2>&1; then
  printf '{"mode":"real","summary":"\u670d\u52a1\u5668\u672a\u5b89\u88c5 nvidia-smi","availableGpuCount":0,"totalGpuCount":0,"checkedAt":"%s","devices":[]}\n' "$(date -Iseconds)"
  exit 0
fi
nvidia-smi --query-gpu=index,name,memory.used,memory.total,utilization.gpu --format=csv,noheader,nounits | awk -F', ' '
BEGIN {
  checked=""; total=0; available=0; devices="[";
}
{
  if (checked == "") {
    cmd="date -Iseconds";
    cmd | getline checked;
    close(cmd);
  }
  idx=$1+0; name=$2; used=$3+0; totalMem=$4+0; util=$5+0;
  isAvailable=(util < 10 && used < 1024) ? "true" : "false";
  if (isAvailable == "true") { available++; }
  if (total > 0) { devices=devices ","; }
  devices=devices sprintf("{\"index\":%d,\"name\":\"%s\",\"memoryUsedMb\":%d,\"memoryTotalMb\":%d,\"utilization\":%d,\"processes\":0,\"available\":%s}", idx, name, used, totalMem, util, isAvailable);
  total++;
}
END {
  devices=devices "]";
  summary=sprintf("\\u68c0\\u6d4b\\u5230 %d \\u5f20 GPU\\uff0c\\u53ef\\u7528 %d \\u5f20", total, available);
  printf "{\"mode\":\"real\",\"summary\":\"%s\",\"availableGpuCount\":%d,\"totalGpuCount\":%d,\"checkedAt\":\"%s\",\"devices\":%s}\n", summary, available, total, checked, devices;
}'`
}

func scanDatasetsScript(rootPath string, maxDepth int) string {
	quotedRoot := shellQuote(rootPath)
	depth := strconv.Itoa(maxDepth)
	return fmt.Sprintf(`ROOT=%s
MAX_DEPTH=%s
if [ ! -d "$ROOT" ]; then
  printf '{"mode":"real","rootPath":"%%s","scannedAt":"%%s","candidates":[]}\n' "$ROOT" "$(date -Iseconds)"
  exit 0
fi
python3 - <<'PY'
import json
import os
from datetime import datetime, timezone
root = %q
max_depth = %d
def humanize(size: int) -> str:
    units = ["B", "KB", "MB", "GB", "TB"]
    value = float(size)
    unit = units[0]
    for next_unit in units[1:]:
        if value < 1024:
            break
        value /= 1024
        unit = next_unit
    if unit == "B":
        return f"{int(value)} {unit}"
    return f"{value:.1f} {unit}"

items = []
for entry in sorted(os.scandir(root), key=lambda item: item.name.lower()):
    if not entry.is_dir(follow_symlinks=False):
        continue
    file_count = 0
    dir_count = 0
    total_size = 0
    last_modified = None
    modality = "text"
    for current_root, dirnames, filenames in os.walk(entry.path):
        rel = os.path.relpath(current_root, entry.path)
        depth = 0 if rel == "." else rel.count(os.sep) + 1
        if depth > max_depth:
            dirnames[:] = []
            continue
        dir_count += len(dirnames)
        for name in filenames:
            file_count += 1
            path = os.path.join(current_root, name)
            try:
                stat = os.stat(path)
            except OSError:
                continue
            total_size += stat.st_size
            ts = datetime.fromtimestamp(stat.st_mtime, tz=timezone.utc).isoformat()
            if last_modified is None or ts > last_modified:
                last_modified = ts
            lower = name.lower()
            if lower.endswith((".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp")):
                modality = "image" if modality == "text" else "multimodal"
            elif lower.endswith((".wav", ".mp3", ".flac", ".m4a")):
                modality = "audio" if modality == "text" else "multimodal"
            elif lower.endswith((".mp4", ".avi", ".mov", ".mkv")):
                modality = "video" if modality == "text" else "multimodal"
    items.append({
        "name": entry.name,
        "path": entry.path.replace('\\\\', '/'),
        "size": humanize(total_size),
        "totalSizeBytes": total_size,
        "fileCount": file_count,
        "directoryCount": dir_count,
        "lastModifiedAt": last_modified,
        "modality": modality,
        "status": "new",
        "description": f"\u4ece\u670d\u52a1\u5668\u76ee\u5f55 {entry.path} \u626b\u63cf\u53d1\u73b0"
    })

print(json.dumps({
    "mode": "real",
    "rootPath": root,
    "scannedAt": datetime.now(timezone.utc).isoformat(),
    "candidates": items,
}, ensure_ascii=False))
PY`, quotedRoot, depth, rootPath, maxDepth)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func decodeLastJSONLine(raw string, dest interface{}) error {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), dest); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no valid JSON payload found in command output")
}
