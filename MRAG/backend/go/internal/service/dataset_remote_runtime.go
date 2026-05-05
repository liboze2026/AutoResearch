package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type remoteDatasetRuntime struct {
	mode           string
	ssh            SSHGateway
	entrypoint     string
	remoteWorkRoot string
	commandTimeout time.Duration
}

type remoteDatasetScanJSON struct {
	ValidationStatus string                              `json:"validationStatus"`
	ScanStatus       string                              `json:"scanStatus"`
	FileCount        int64                               `json:"fileCount"`
	DirectoryCount   int64                               `json:"directoryCount"`
	TotalSizeBytes   int64                               `json:"totalSizeBytes"`
	FileTypes        map[string]int64                    `json:"fileTypes"`
	HierarchySummary []model.DatasetHierarchySummaryItem `json:"hierarchySummary"`
	InferredModality string                              `json:"inferredModality"`
	RecentModifiedAt string                              `json:"recentModifiedAt"`
	PreviewItems     []model.DatasetPreviewItem          `json:"previewItems"`
	ErrorMessage     string                              `json:"errorMessage"`
}

type remoteDatasetIndexResponse struct {
	TaskID       string   `json:"taskId"`
	Status       string   `json:"status"`
	LogPath      string   `json:"logPath"`
	StatusPath   string   `json:"statusPath"`
	ResultPath   string   `json:"resultPath"`
	ErrorMessage string   `json:"errorMessage"`
	Message      string   `json:"message"`
	Logs         []string `json:"logs"`
}

func NewRemoteDatasetRuntime(mode string, ssh SSHGateway, entrypoint string, remoteWorkRoot string, commandTimeoutSec int) datasetRuntime {
	if commandTimeoutSec <= 0 {
		commandTimeoutSec = 20
	}
	return &remoteDatasetRuntime{
		mode:           normalizeMode(mode),
		ssh:            ssh,
		entrypoint:     strings.TrimSpace(entrypoint),
		remoteWorkRoot: strings.TrimSpace(remoteWorkRoot),
		commandTimeout: time.Duration(commandTimeoutSec) * time.Second,
	}
}

func (r *remoteDatasetRuntime) Mode() string {
	return r.mode
}

func (r *remoteDatasetRuntime) ValidatePath(ctx context.Context, path string, server *model.Server) (*model.DatasetPathValidationResult, error) {
	if r.mode == "mock" {
		return mockDatasetValidation("remote", path, server), nil
	}
	if server == nil {
		return nil, fmt.Errorf("remote server is required")
	}
	result, err := r.ssh.Exec(ctx, server, SSHExecRequest{
		Purpose:       "dataset_validate",
		RemoteCommand: []string{"sh", "-s", "--", "validate", r.entrypoint, path},
		Stdin:         remoteDatasetValidateScript(),
		Timeout:       r.commandTimeout,
	})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("remote dataset validate failed: %s", firstNonEmpty(result.Stderr, result.Stdout))
	}
	parsed, err := parseRemoteValidationOutput(result.Stdout)
	if err != nil {
		return nil, err
	}
	parsed.SourceType = "remote"
	parsed.Path = path
	parsed.Mode = r.mode
	parsed.ServerID = server.ID
	parsed.ServerName = server.Name
	if parsed.CheckedAt.IsZero() {
		parsed.CheckedAt = time.Now()
	}
	return parsed, nil
}

func (r *remoteDatasetRuntime) Scan(ctx context.Context, root string, server *model.Server, previewLimit int) (*datasetScanSnapshot, error) {
	if r.mode == "mock" {
		return mockDatasetScan("remote", root, previewLimit), nil
	}
	if server == nil {
		return nil, fmt.Errorf("remote server is required")
	}
	result, err := r.ssh.Exec(ctx, server, SSHExecRequest{
		Purpose:       "dataset_scan",
		RemoteCommand: []string{"sh", "-s", "--", "scan", r.entrypoint, root, strconv.Itoa(previewLimit)},
		Stdin:         remoteDatasetScanScript(),
		Timeout:       r.commandTimeout,
	})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("remote dataset scan failed: %s", firstNonEmpty(result.Stderr, result.Stdout))
	}
	return parseRemoteScanOutput(result.Stdout)
}

func (r *remoteDatasetRuntime) StartIndex(ctx context.Context, dataset *model.Dataset, task *model.DatasetIndexTask, server *model.Server) (*datasetIndexTaskUpdate, error) {
	if r.mode == "mock" {
		paths := r.remoteIndexPaths(task.ID)
		return &datasetIndexTaskUpdate{
			Status:       "building",
			RemoteTaskID: task.ID,
			LogPath:      paths["logPath"],
			StatusPath:   paths["statusPath"],
			ResultPath:   paths["resultPath"],
			Logs:         []string{"Mock remote index task created", "Use task sync to move the task to completed"},
			ResponsePayload: map[string]interface{}{
				"runner": "remote-mock",
			},
		}, nil
	}
	if server == nil {
		return nil, fmt.Errorf("remote server is required")
	}
	requestPayload, err := json.MarshalIndent(task.RequestPayload, "", "  ")
	if err != nil {
		return nil, err
	}
	result, err := r.ssh.Exec(ctx, server, SSHExecRequest{
		Purpose:       "dataset_index_start",
		RemoteCommand: []string{"sh", "-s", "--", "index-start", r.entrypoint, r.resolveRemoteRoot(), task.ID},
		Stdin:         remoteDatasetIndexStartScript(string(requestPayload)),
		Timeout:       r.commandTimeout,
	})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("remote index start failed: %s", firstNonEmpty(result.Stderr, result.Stdout))
	}
	resp, err := parseRemoteIndexResponse(result.Stdout)
	if err != nil {
		return nil, err
	}
	return &datasetIndexTaskUpdate{
		Status:       resp.Status,
		RemoteTaskID: firstNonEmpty(resp.TaskID, task.ID),
		LogPath:      resp.LogPath,
		StatusPath:   resp.StatusPath,
		ResultPath:   resp.ResultPath,
		ErrorMessage: resp.ErrorMessage,
		Logs:         append([]string{resp.Message}, resp.Logs...),
		ResponsePayload: map[string]interface{}{
			"runner": "remote-ssh",
		},
	}, nil
}

func (r *remoteDatasetRuntime) SyncIndex(ctx context.Context, dataset *model.Dataset, task *model.DatasetIndexTask, server *model.Server) (*datasetIndexTaskUpdate, error) {
	if r.mode == "mock" {
		now := time.Now()
		return &datasetIndexTaskUpdate{
			Status:       "completed",
			RemoteTaskID: firstNonEmpty(task.RemoteTaskID, task.ID),
			LogPath:      task.LogPath,
			StatusPath:   task.StatusPath,
			ResultPath:   task.ResultPath,
			FinishedAt:   &now,
			Logs:         []string{"Mock remote index task completed"},
			ResponsePayload: map[string]interface{}{
				"runner":      "remote-mock",
				"completedAt": now.Format(time.RFC3339),
			},
		}, nil
	}
	if server == nil {
		return nil, fmt.Errorf("remote server is required")
	}
	result, err := r.ssh.Exec(ctx, server, SSHExecRequest{
		Purpose:       "dataset_index_status",
		RemoteCommand: []string{"sh", "-s", "--", "index-status", r.entrypoint, r.resolveRemoteRoot(), task.ID},
		Stdin:         remoteDatasetIndexStatusScript(),
		Timeout:       r.commandTimeout,
	})
	if err != nil {
		return nil, err
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("remote index status failed: %s", firstNonEmpty(result.Stderr, result.Stdout))
	}
	resp, err := parseRemoteIndexResponse(result.Stdout)
	if err != nil {
		return nil, err
	}
	var finishedAt *time.Time
	if resp.Status == "completed" || resp.Status == "failed" {
		now := time.Now()
		finishedAt = &now
	}
	return &datasetIndexTaskUpdate{
		Status:       resp.Status,
		RemoteTaskID: firstNonEmpty(resp.TaskID, task.RemoteTaskID, task.ID),
		LogPath:      firstNonEmpty(resp.LogPath, task.LogPath),
		StatusPath:   firstNonEmpty(resp.StatusPath, task.StatusPath),
		ResultPath:   firstNonEmpty(resp.ResultPath, task.ResultPath),
		ErrorMessage: resp.ErrorMessage,
		FinishedAt:   finishedAt,
		Logs:         append([]string{resp.Message}, resp.Logs...),
		ResponsePayload: map[string]interface{}{
			"runner": "remote-ssh",
		},
	}, nil
}

func (r *remoteDatasetRuntime) resolveRemoteRoot() string {
	if r.remoteWorkRoot != "" {
		return r.remoteWorkRoot
	}
	return "/home/bzli/lbz"
}

func (r *remoteDatasetRuntime) remoteIndexPaths(taskID string) map[string]string {
	base := filepath.ToSlash(filepath.Join(r.resolveRemoteRoot(), "dataset-index-tasks", taskID))
	return map[string]string{
		"logPath":    filepath.ToSlash(filepath.Join(base, "logs", "runtime.log")),
		"statusPath": filepath.ToSlash(filepath.Join(base, "status.json")),
		"resultPath": filepath.ToSlash(filepath.Join(base, "result.json")),
	}
}

func parseRemoteValidationOutput(raw string) (*model.DatasetPathValidationResult, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("empty validation output")
	}
	if strings.HasPrefix(trimmed, "{") {
		var item model.DatasetPathValidationResult
		if err := json.Unmarshal([]byte(trimmed), &item); err != nil {
			return nil, err
		}
		return &item, nil
	}
	result := &model.DatasetPathValidationResult{CheckedAt: time.Now()}
	for _, line := range strings.Split(trimmed, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "\t", 3)
		if len(parts) < 2 {
			continue
		}
		switch parts[0] {
		case "STATUS":
			result.Valid = parts[1] == "ok"
			result.Exists = result.Valid || parts[1] != "not_found"
			result.IsDirectory = result.Valid
			if !result.Valid {
				result.ErrorType = parts[1]
			}
		case "MESSAGE":
			result.Message = parts[1]
		}
	}
	if result.Message == "" {
		if result.Valid {
			result.Message = "Remote dataset directory is available"
		} else {
			result.Message = result.ErrorType
		}
	}
	return result, nil
}

func parseRemoteScanOutput(raw string) (*datasetScanSnapshot, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("empty scan output")
	}
	if strings.HasPrefix(trimmed, "{") {
		var payload remoteDatasetScanJSON
		if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
			return nil, err
		}
		var recentModified *time.Time
		if payload.RecentModifiedAt != "" {
			parsed, err := time.Parse(time.RFC3339, payload.RecentModifiedAt)
			if err == nil {
				recentModified = &parsed
			}
		}
		return &datasetScanSnapshot{
			ValidationStatus: payload.ValidationStatus,
			ScanStatus:       payload.ScanStatus,
			FileCount:        payload.FileCount,
			DirectoryCount:   payload.DirectoryCount,
			TotalSizeBytes:   payload.TotalSizeBytes,
			FileTypes:        payload.FileTypes,
			HierarchySummary: payload.HierarchySummary,
			InferredModality: payload.InferredModality,
			RecentModifiedAt: recentModified,
			ScannedAt:        time.Now(),
			PreviewItems:     sortPreviewItems(payload.PreviewItems, len(payload.PreviewItems)),
			ErrorMessage:     payload.ErrorMessage,
		}, nil
	}
	snapshot := &datasetScanSnapshot{
		ValidationStatus: "ok",
		ScanStatus:       "completed",
		FileTypes:        map[string]int64{},
		HierarchySummary: []model.DatasetHierarchySummaryItem{},
		PreviewItems:     []model.DatasetPreviewItem{},
		ScannedAt:        time.Now(),
	}
	hierarchyCounts := map[string]int64{}
	for _, rawLine := range strings.Split(trimmed, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		switch parts[0] {
		case "SCAN_STATUS":
			if len(parts) > 1 {
				snapshot.ScanStatus = parts[1]
			}
		case "VALIDATION_STATUS":
			if len(parts) > 1 {
				snapshot.ValidationStatus = parts[1]
			}
		case "FILE_COUNT":
			snapshot.FileCount = parseInt64(parts, 1)
		case "DIRECTORY_COUNT":
			snapshot.DirectoryCount = parseInt64(parts, 1)
		case "TOTAL_SIZE_BYTES":
			snapshot.TotalSizeBytes = parseInt64(parts, 1)
		case "RECENT_MODIFIED_UNIX":
			if len(parts) > 1 {
				parsed := parseUnixSeconds(parts[1])
				if parsed != nil {
					snapshot.RecentModifiedAt = parsed
				}
			}
		case "TYPE":
			if len(parts) >= 3 {
				snapshot.FileTypes[parts[1]] = parseInt64(parts, 2)
			}
		case "HIER":
			if len(parts) >= 4 {
				hierarchyCounts[parts[1]+"|"+parts[2]] = parseInt64(parts, 3)
			}
		case "PREVIEW":
			if len(parts) >= 5 {
				rel := normalizeRelativePath(parts[4])
				snapshot.PreviewItems = append(snapshot.PreviewItems, model.DatasetPreviewItem{
					Name:         previewName(rel),
					ItemType:     parts[1],
					Category:     parts[2],
					RelativePath: rel,
					SizeBytes:    parseInt64(parts, 3),
					Depth:        previewDepth(rel),
				})
			}
		case "ERROR":
			if len(parts) > 1 {
				snapshot.ErrorMessage = parts[1]
			}
		}
	}
	snapshot.HierarchySummary = hierarchySummaryFromCounts(hierarchyCounts, 8)
	if snapshot.InferredModality == "" {
		snapshot.InferredModality = inferDatasetModality(snapshot.FileTypes)
	}
	snapshot.PreviewItems = sortPreviewItems(snapshot.PreviewItems, len(snapshot.PreviewItems))
	return snapshot, nil
}

func parseRemoteIndexResponse(raw string) (*remoteDatasetIndexResponse, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("empty index task output")
	}
	lines := strings.Split(trimmed, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			var resp remoteDatasetIndexResponse
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				return nil, err
			}
			if resp.Status == "" {
				resp.Status = "building"
			}
			return &resp, nil
		}
	}
	return nil, fmt.Errorf("index task output did not contain JSON")
}

func parseInt64(parts []string, idx int) int64 {
	if len(parts) <= idx {
		return 0
	}
	value, _ := strconv.ParseInt(strings.TrimSpace(parts[idx]), 10, 64)
	return value
}

func parseUnixSeconds(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	seconds := int64(floatValue)
	nanos := int64((floatValue - float64(seconds)) * float64(time.Second))
	parsed := time.Unix(seconds, nanos)
	return &parsed
}

func remoteDatasetValidateScript() string {
	return `set -eu
ENTRYPOINT="${2:?entrypoint is required}"
TARGET="${3:?path is required}"
if [ -x "$ENTRYPOINT" ]; then
  "$ENTRYPOINT" validate --path "$TARGET"
  exit 0
fi
if [ ! -e "$TARGET" ]; then
  printf 'STATUS\tnot_found\nMESSAGE\tRemote path does not exist\n'
  exit 0
fi
if [ ! -d "$TARGET" ]; then
  printf 'STATUS\tnot_directory\nMESSAGE\tRemote path is not a directory\n'
  exit 0
fi
if [ ! -r "$TARGET" ] || [ ! -x "$TARGET" ]; then
  printf 'STATUS\tpermission_denied\nMESSAGE\tRemote path is not accessible\n'
  exit 0
fi
printf 'STATUS\tok\nMESSAGE\tRemote dataset directory is available\n'
`
}

func remoteDatasetScanScript() string {
	return `set -eu
ENTRYPOINT="${2:?entrypoint is required}"
TARGET="${3:?path is required}"
LIMIT="${4:-12}"
if [ -x "$ENTRYPOINT" ]; then
  "$ENTRYPOINT" scan --path "$TARGET" --preview-limit "$LIMIT"
  exit 0
fi
if [ ! -d "$TARGET" ]; then
  printf 'SCAN_STATUS\tfailed\nERROR\tRemote path is not a directory\n'
  exit 0
fi
printf 'SCAN_STATUS\tcompleted\n'
printf 'VALIDATION_STATUS\tok\n'
printf 'FILE_COUNT\t%s\n' "$(find "$TARGET" -type f | wc -l | tr -d ' ')"
printf 'DIRECTORY_COUNT\t%s\n' "$(find "$TARGET" -mindepth 1 -type d | wc -l | tr -d ' ')"
printf 'TOTAL_SIZE_BYTES\t%s\n' "$(find "$TARGET" -type f -printf '%s\n' 2>/dev/null | awk '{s+=$1} END {print s+0}')"
printf 'RECENT_MODIFIED_UNIX\t%s\n' "$(find "$TARGET" -printf '%T@\n' 2>/dev/null | sort -nr | head -n 1)"
find "$TARGET" -type f -printf '%f\n' 2>/dev/null | awk '
function category(name, lower) {
  lower = tolower(name)
  if (lower ~ /\.(txt|md|csv|tsv|yaml|yml|xml|html|htm)$/) return "text"
  if (lower ~ /\.pdf$/) return "pdf"
  if (lower ~ /\.(json|jsonl)$/) return "json"
  if (lower ~ /\.(png|jpg|jpeg|gif|bmp|webp|tif|tiff)$/) return "image"
  if (lower ~ /\.(wav|mp3|m4a|flac|aac)$/) return "audio"
  if (lower ~ /\.(mp4|avi|mov|mkv|webm)$/) return "video"
  return "other"
}
{ counts[category($0)]++ }
END { for (k in counts) printf "TYPE\t%s\t%d\n", k, counts[k] }
'
find "$TARGET" -mindepth 1 -printf '%P\n' 2>/dev/null | awk -F'/' '
NF >= 1 && $1 != "" { level0[$1]++ }
NF >= 2 && $2 != "" { level1[$1 "/" $2]++ }
END {
  for (k in level0) printf "HIER\t0\t%s\t%d\n", k, level0[k]
  for (k in level1) printf "HIER\t1\t%s\t%d\n", k, level1[k]
}
'
find "$TARGET" -mindepth 1 -maxdepth 2 -printf '%y\t%s\t%P\n' 2>/dev/null | sort | head -n "$LIMIT" | while IFS='\t' read -r kind size rel; do
  item_type="file"
  category="other"
  if [ "$kind" = "d" ]; then
    item_type="directory"
    category="directory"
    size=0
  else
    lower="$(printf '%s' "$rel" | tr '[:upper:]' '[:lower:]')"
    case "$lower" in
      *.txt|*.md|*.csv|*.tsv|*.yaml|*.yml|*.xml|*.html|*.htm) category="text" ;;
      *.pdf) category="pdf" ;;
      *.json|*.jsonl) category="json" ;;
      *.png|*.jpg|*.jpeg|*.gif|*.bmp|*.webp|*.tif|*.tiff) category="image" ;;
      *.wav|*.mp3|*.m4a|*.flac|*.aac) category="audio" ;;
      *.mp4|*.avi|*.mov|*.mkv|*.webm) category="video" ;;
      *) category="other" ;;
    esac
  fi
  printf 'PREVIEW\t%s\t%s\t%s\t%s\n' "$item_type" "$category" "$size" "$rel"
done
`
}

func remoteDatasetIndexStartScript(requestPayload string) string {
	return fmt.Sprintf(`set -eu
ENTRYPOINT="${2:?entrypoint is required}"
ROOT="${3:?remote root is required}"
TASK_ID="${4:?task id is required}"
TASK_DIR="$ROOT/dataset-index-tasks/$TASK_ID"
REQUEST_FILE="$TASK_DIR/request.json"
STATUS_FILE="$TASK_DIR/status.json"
LOG_FILE="$TASK_DIR/logs/runtime.log"
RESULT_FILE="$TASK_DIR/result.json"
mkdir -p "$TASK_DIR/logs"
cat > "$REQUEST_FILE" <<'__MRAG_DATASET_REQUEST__'
%s
__MRAG_DATASET_REQUEST__
if [ -x "$ENTRYPOINT" ]; then
  "$ENTRYPOINT" index-start --request-file "$REQUEST_FILE"
  exit 0
fi
NOW_VALUE="$(date -Iseconds)"
cat > "$STATUS_FILE" <<__MRAG_DATASET_STATUS__
{"taskId":"$TASK_ID","status":"building","logPath":"$LOG_FILE","statusPath":"$STATUS_FILE","resultPath":"$RESULT_FILE","errorMessage":"","message":"remote placeholder index task accepted","logs":["remote placeholder index task accepted at $NOW_VALUE"]}
__MRAG_DATASET_STATUS__
printf '[%%s] remote placeholder index task accepted\n' "$NOW_VALUE" >> "$LOG_FILE"
cat "$STATUS_FILE"
`, requestPayload)
}

func remoteDatasetIndexStatusScript() string {
	return `set -eu
ENTRYPOINT="${2:?entrypoint is required}"
ROOT="${3:?remote root is required}"
TASK_ID="${4:?task id is required}"
TASK_DIR="$ROOT/dataset-index-tasks/$TASK_ID"
STATUS_FILE="$TASK_DIR/status.json"
if [ -x "$ENTRYPOINT" ]; then
  "$ENTRYPOINT" index-status --task-dir "$TASK_DIR"
  exit 0
fi
if [ -f "$STATUS_FILE" ]; then
  cat "$STATUS_FILE"
  exit 0
fi
printf '{"taskId":"%s","status":"failed","logPath":"","statusPath":"%s","resultPath":"","errorMessage":"index status file not found","message":"remote index status file not found","logs":[]}' "$TASK_ID" "$STATUS_FILE"
`
}
