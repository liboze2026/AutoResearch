package service

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type datasetRuntime interface {
	Mode() string
	ValidatePath(ctx context.Context, path string, server *model.Server) (*model.DatasetPathValidationResult, error)
	Scan(ctx context.Context, path string, server *model.Server, previewLimit int) (*datasetScanSnapshot, error)
	StartIndex(ctx context.Context, dataset *model.Dataset, task *model.DatasetIndexTask, server *model.Server) (*datasetIndexTaskUpdate, error)
	SyncIndex(ctx context.Context, dataset *model.Dataset, task *model.DatasetIndexTask, server *model.Server) (*datasetIndexTaskUpdate, error)
}

type datasetScanSnapshot struct {
	ValidationStatus string
	ScanStatus       string
	FileCount        int64
	DirectoryCount   int64
	TotalSizeBytes   int64
	FileTypes        map[string]int64
	HierarchySummary []model.DatasetHierarchySummaryItem
	InferredModality string
	RecentModifiedAt *time.Time
	ScannedAt        time.Time
	PreviewItems     []model.DatasetPreviewItem
	ErrorMessage     string
}

type datasetIndexTaskUpdate struct {
	Status          string
	RemoteTaskID    string
	LogPath         string
	StatusPath      string
	ResultPath      string
	ErrorMessage    string
	ResponsePayload map[string]interface{}
	Logs            []string
	FinishedAt      *time.Time
}

func normalizeMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), "mock") {
		return "mock"
	}
	return "real"
}

func detectFileCategory(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".txt"), strings.HasSuffix(lower, ".md"), strings.HasSuffix(lower, ".csv"), strings.HasSuffix(lower, ".tsv"), strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"), strings.HasSuffix(lower, ".xml"), strings.HasSuffix(lower, ".html"), strings.HasSuffix(lower, ".htm"):
		return "text"
	case strings.HasSuffix(lower, ".pdf"):
		return "pdf"
	case strings.HasSuffix(lower, ".json"), strings.HasSuffix(lower, ".jsonl"):
		return "json"
	case strings.HasSuffix(lower, ".png"), strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"), strings.HasSuffix(lower, ".gif"), strings.HasSuffix(lower, ".bmp"), strings.HasSuffix(lower, ".webp"), strings.HasSuffix(lower, ".tif"), strings.HasSuffix(lower, ".tiff"):
		return "image"
	case strings.HasSuffix(lower, ".wav"), strings.HasSuffix(lower, ".mp3"), strings.HasSuffix(lower, ".m4a"), strings.HasSuffix(lower, ".flac"), strings.HasSuffix(lower, ".aac"):
		return "audio"
	case strings.HasSuffix(lower, ".mp4"), strings.HasSuffix(lower, ".avi"), strings.HasSuffix(lower, ".mov"), strings.HasSuffix(lower, ".mkv"), strings.HasSuffix(lower, ".webm"):
		return "video"
	default:
		return "other"
	}
}

func inferDatasetModality(fileTypes map[string]int64) string {
	buckets := 0
	textLike := fileTypes["text"] + fileTypes["pdf"] + fileTypes["json"]
	if textLike > 0 {
		buckets++
	}
	if fileTypes["image"] > 0 {
		buckets++
	}
	if fileTypes["audio"] > 0 {
		buckets++
	}
	if fileTypes["video"] > 0 {
		buckets++
	}
	switch {
	case buckets == 0:
		return "text"
	case buckets > 1:
		return "multimodal"
	case fileTypes["video"] > 0:
		return "video"
	case fileTypes["audio"] > 0:
		return "audio"
	case fileTypes["image"] > 0:
		return "image"
	default:
		return "text"
	}
}

func humanizeBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	value := float64(size)
	units := []string{"KB", "MB", "GB", "TB"}
	unit := "B"
	for _, next := range units {
		value = value / 1024
		unit = next
		if value < 1024 {
			break
		}
	}
	return fmt.Sprintf("%.1f %s", value, unit)
}

func normalizeRelativePath(value string) string {
	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.TrimPrefix(value, "./")
	return strings.TrimPrefix(value, "/")
}

func previewName(relativePath string) string {
	relativePath = normalizeRelativePath(relativePath)
	if relativePath == "" {
		return ""
	}
	return path.Base(relativePath)
}

func previewDepth(relativePath string) int {
	relativePath = normalizeRelativePath(relativePath)
	if relativePath == "" {
		return 0
	}
	return len(strings.Split(relativePath, "/")) - 1
}

func hierarchySummaryFromCounts(counts map[string]int64, limit int) []model.DatasetHierarchySummaryItem {
	items := make([]model.DatasetHierarchySummaryItem, 0, len(counts))
	for key, count := range counts {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) != 2 {
			continue
		}
		level := 0
		if parts[0] == "1" {
			level = 1
		}
		items = append(items, model.DatasetHierarchySummaryItem{Level: level, Path: parts[1], ItemCount: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ItemCount == items[j].ItemCount {
			if items[i].Level == items[j].Level {
				return items[i].Path < items[j].Path
			}
			return items[i].Level < items[j].Level
		}
		return items[i].ItemCount > items[j].ItemCount
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func sortPreviewItems(items []model.DatasetPreviewItem, limit int) []model.DatasetPreviewItem {
	sort.Slice(items, func(i, j int) bool {
		if items[i].RelativePath == items[j].RelativePath {
			return items[i].ItemType < items[j].ItemType
		}
		return items[i].RelativePath < items[j].RelativePath
	})
	if limit > 0 && len(items) > limit {
		return items[:limit]
	}
	return items
}
