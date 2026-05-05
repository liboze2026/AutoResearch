package service

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type localDatasetRuntime struct {
	mode string
}

func NewLocalDatasetRuntime(mode string) datasetRuntime {
	return &localDatasetRuntime{mode: normalizeMode(mode)}
}

func (r *localDatasetRuntime) Mode() string {
	return r.mode
}

func (r *localDatasetRuntime) ValidatePath(ctx context.Context, path string, server *model.Server) (*model.DatasetPathValidationResult, error) {
	if r.mode == "mock" {
		return mockDatasetValidation("local", path, nil), nil
	}
	result := &model.DatasetPathValidationResult{
		SourceType: "local",
		Path:       path,
		Mode:       r.mode,
		CheckedAt:  time.Now(),
	}
	info, err := os.Stat(path)
	if err != nil {
		result.Valid = false
		result.Exists = false
		switch {
		case os.IsNotExist(err):
			result.ErrorType = "not_found"
			result.Message = "Local path does not exist"
		case errors.Is(err, fs.ErrPermission):
			result.ErrorType = "permission_denied"
			result.Message = "Local path is not accessible"
		default:
			result.ErrorType = "not_found"
			result.Message = err.Error()
		}
		return result, nil
	}
	result.Exists = true
	result.IsDirectory = info.IsDir()
	if !info.IsDir() {
		result.ErrorType = "not_directory"
		result.Message = "Local path is not a directory"
		return result, nil
	}
	result.Valid = true
	result.Message = "Local dataset directory is available"
	return result, nil
}

func (r *localDatasetRuntime) Scan(ctx context.Context, root string, server *model.Server, previewLimit int) (*datasetScanSnapshot, error) {
	if r.mode == "mock" {
		return mockDatasetScan("local", root, previewLimit), nil
	}
	fileTypes := map[string]int64{}
	hierarchyCounts := map[string]int64{}
	previewItems := make([]model.DatasetPreviewItem, 0, previewLimit)
	var fileCount int64
	var directoryCount int64
	var totalSize int64
	var latestModified *time.Time

	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if current == root {
			return nil
		}
		rel, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		rel = normalizeRelativePath(rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if latestModified == nil || info.ModTime().After(*latestModified) {
			copyValue := info.ModTime()
			latestModified = &copyValue
		}
		segments := strings.Split(rel, "/")
		if len(segments) >= 1 {
			hierarchyCounts["0|"+segments[0]]++
		}
		if len(segments) >= 2 {
			hierarchyCounts["1|"+segments[0]+"/"+segments[1]]++
		}
		if entry.IsDir() {
			directoryCount++
			previewItems = appendPreview(previewItems, previewLimit*3, model.DatasetPreviewItem{
				Name:         previewName(rel),
				ItemType:     "directory",
				Category:     "directory",
				RelativePath: rel,
				SizeBytes:    0,
				Depth:        previewDepth(rel),
			})
			return nil
		}
		fileCount++
		totalSize += info.Size()
		category := detectFileCategory(entry.Name())
		fileTypes[category]++
		previewItems = appendPreview(previewItems, previewLimit*3, model.DatasetPreviewItem{
			Name:         previewName(rel),
			ItemType:     "file",
			Category:     category,
			RelativePath: rel,
			SizeBytes:    info.Size(),
			Depth:        previewDepth(rel),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	previewItems = sortPreviewItems(previewItems, previewLimit)
	return &datasetScanSnapshot{
		ValidationStatus: "ok",
		ScanStatus:       "completed",
		FileCount:        fileCount,
		DirectoryCount:   directoryCount,
		TotalSizeBytes:   totalSize,
		FileTypes:        fileTypes,
		HierarchySummary: hierarchySummaryFromCounts(hierarchyCounts, 8),
		InferredModality: inferDatasetModality(fileTypes),
		RecentModifiedAt: latestModified,
		ScannedAt:        time.Now(),
		PreviewItems:     previewItems,
	}, nil
}

func (r *localDatasetRuntime) StartIndex(ctx context.Context, dataset *model.Dataset, task *model.DatasetIndexTask, server *model.Server) (*datasetIndexTaskUpdate, error) {
	paths := localIndexPaths(dataset.Path, task.ID)
	return &datasetIndexTaskUpdate{
		Status:     "building",
		LogPath:    paths["logPath"],
		StatusPath: paths["statusPath"],
		ResultPath: paths["resultPath"],
		Logs:       []string{"Local index task created", "Current stage uses a simulated local index builder"},
		ResponsePayload: map[string]interface{}{
			"builder":     "local-simulated",
			"datasetPath": dataset.Path,
		},
	}, nil
}

func (r *localDatasetRuntime) SyncIndex(ctx context.Context, dataset *model.Dataset, task *model.DatasetIndexTask, server *model.Server) (*datasetIndexTaskUpdate, error) {
	if task.Status == "completed" || task.Status == "failed" {
		return &datasetIndexTaskUpdate{
			Status:          task.Status,
			LogPath:         task.LogPath,
			StatusPath:      task.StatusPath,
			ResultPath:      task.ResultPath,
			ErrorMessage:    task.ErrorMessage,
			ResponsePayload: task.ResponsePayload,
		}, nil
	}
	now := time.Now()
	return &datasetIndexTaskUpdate{
		Status:     "completed",
		LogPath:    task.LogPath,
		StatusPath: task.StatusPath,
		ResultPath: task.ResultPath,
		FinishedAt: &now,
		Logs:       []string{"Local simulated index build completed", fmt.Sprintf("Index artifact placeholder: %s", task.ResultPath)},
		ResponsePayload: map[string]interface{}{
			"builder":        "local-simulated",
			"completedAt":    now.Format(time.RFC3339),
			"resultArtifact": task.ResultPath,
		},
	}, nil
}

func appendPreview(items []model.DatasetPreviewItem, limit int, item model.DatasetPreviewItem) []model.DatasetPreviewItem {
	if limit > 0 && len(items) >= limit {
		return items
	}
	return append(items, item)
}

func localIndexPaths(root string, taskID string) map[string]string {
	base := normalizeRelativePath(filepath.ToSlash(filepath.Join(root, ".mrag-index", taskID)))
	if strings.Contains(root, ":") || strings.HasPrefix(root, "/") {
		base = filepath.ToSlash(filepath.Join(root, ".mrag-index", taskID))
	}
	return map[string]string{
		"logPath":    filepath.ToSlash(filepath.Join(base, "runtime.log")),
		"statusPath": filepath.ToSlash(filepath.Join(base, "status.json")),
		"resultPath": filepath.ToSlash(filepath.Join(base, "index.json")),
	}
}

func mockDatasetValidation(sourceType string, path string, server *model.Server) *model.DatasetPathValidationResult {
	result := &model.DatasetPathValidationResult{
		SourceType: sourceType,
		Path:       path,
		Mode:       "mock",
		CheckedAt:  time.Now(),
	}
	if server != nil {
		result.ServerID = server.ID
		result.ServerName = server.Name
	}
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "missing"):
		result.ErrorType = "not_found"
		result.Message = "Mock mode: path does not exist"
	case strings.Contains(lower, "denied"):
		result.Exists = true
		result.ErrorType = "permission_denied"
		result.Message = "Mock mode: path is not accessible"
	case strings.Contains(lower, "file"):
		result.Exists = true
		result.ErrorType = "not_directory"
		result.Message = "Mock mode: path is a file"
	default:
		result.Valid = true
		result.Exists = true
		result.IsDirectory = true
		result.Message = "Mock mode: dataset directory is available"
	}
	return result
}

func mockDatasetScan(sourceType string, root string, previewLimit int) *datasetScanSnapshot {
	now := time.Now()
	fileTypes := map[string]int64{
		"text":  14,
		"image": 4,
		"json":  2,
		"pdf":   3,
	}
	preview := []model.DatasetPreviewItem{
		{Name: "docs", ItemType: "directory", Category: "directory", RelativePath: "docs", SizeBytes: 0, Depth: 0},
		{Name: "images", ItemType: "directory", Category: "directory", RelativePath: "images", SizeBytes: 0, Depth: 0},
		{Name: "readme.md", ItemType: "file", Category: "text", RelativePath: "docs/readme.md", SizeBytes: 4096, Depth: 1},
		{Name: "sample.json", ItemType: "file", Category: "json", RelativePath: "docs/sample.json", SizeBytes: 1024, Depth: 1},
		{Name: "figure01.png", ItemType: "file", Category: "image", RelativePath: "images/figure01.png", SizeBytes: 204800, Depth: 1},
	}
	preview = sortPreviewItems(preview, previewLimit)
	return &datasetScanSnapshot{
		ValidationStatus: "ok",
		ScanStatus:       "completed",
		FileCount:        23,
		DirectoryCount:   4,
		TotalSizeBytes:   2_304_000,
		FileTypes:        fileTypes,
		HierarchySummary: []model.DatasetHierarchySummaryItem{{Level: 0, Path: "docs", ItemCount: 12}, {Level: 0, Path: "images", ItemCount: 11}, {Level: 1, Path: "docs/train", ItemCount: 6}},
		InferredModality: inferDatasetModality(fileTypes),
		RecentModifiedAt: &now,
		ScannedAt:        now,
		PreviewItems:     preview,
	}
}
