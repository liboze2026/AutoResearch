package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	"mrag-platform/backend/go/internal/repository"
)

type DatasetService struct {
	repo          *repository.DatasetRepository
	serverRepo    *repository.ServerRepository
	scanAdapters  DatasetScanAdapterResolver
	indexAdapters DatasetIndexAdapterResolver
	previewLimit  int
}

func NewDatasetService(repo *repository.DatasetRepository, serverRepo *repository.ServerRepository, scanAdapters DatasetScanAdapterResolver, indexAdapters DatasetIndexAdapterResolver, previewLimit int) *DatasetService {
	if previewLimit <= 0 {
		previewLimit = 12
	}
	return &DatasetService{
		repo:          repo,
		serverRepo:    serverRepo,
		scanAdapters:  scanAdapters,
		indexAdapters: indexAdapters,
		previewLimit:  previewLimit,
	}
}

func (s *DatasetService) List(ctx context.Context, keyword string, sourceType string, modality string) ([]model.Dataset, error) {
	return s.repo.List(ctx, keyword, sourceType, modality)
}

func (s *DatasetService) GetByID(ctx context.Context, id string) (*model.DatasetDetail, error) {
	return s.repo.GetDetail(ctx, id)
}

func (s *DatasetService) ValidatePath(ctx context.Context, req model.DatasetPathValidationRequest) (*model.DatasetPathValidationResult, error) {
	sourceType, err := normalizeDatasetSourceType(req.SourceType)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return nil, fmt.Errorf("dataset path must not be empty")
	}
	server, err := s.resolveDatasetServer(ctx, sourceType, strings.TrimSpace(req.ServerID))
	if err != nil {
		return nil, err
	}
	scanAdapter, err := s.scanAdapters.Resolve(sourceType)
	if err != nil {
		return nil, err
	}
	return scanAdapter.ValidatePath(ctx, path, server)
}

func (s *DatasetService) Create(ctx context.Context, req model.DatasetImportRequest) (*model.DatasetDetail, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("dataset name must not be empty")
	}
	description := strings.TrimSpace(req.Description)
	if description == "" {
		return nil, fmt.Errorf("dataset description must not be empty")
	}
	sourceType, err := normalizeDatasetSourceType(req.SourceType)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return nil, fmt.Errorf("dataset path must not be empty")
	}
	server, err := s.resolveDatasetServer(ctx, sourceType, strings.TrimSpace(req.ServerID))
	if err != nil {
		return nil, err
	}
	scanAdapter, err := s.scanAdapters.Resolve(sourceType)
	if err != nil {
		return nil, err
	}
	validation, err := scanAdapter.ValidatePath(ctx, path, server)
	if err != nil {
		return nil, err
	}
	if validation == nil || !validation.Valid {
		if validation != nil && strings.TrimSpace(validation.Message) != "" {
			return nil, fmt.Errorf(validation.Message)
		}
		return nil, fmt.Errorf("dataset path validation failed")
	}
	snapshot, err := scanAdapter.Scan(ctx, path, server, s.previewLimit)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, fmt.Errorf("dataset scan returned empty result")
	}
	if snapshot.ScanStatus != "" && snapshot.ScanStatus != "completed" {
		return nil, fmt.Errorf(firstNonEmpty(snapshot.ErrorMessage, "dataset scan did not complete successfully"))
	}
	datasetID := httpx.NewID("ds")
	scanID := httpx.NewID("dscan")
	scannedAt := snapshot.ScannedAt
	if scannedAt.IsZero() {
		scannedAt = time.Now()
	}
	modality := normalizeDatasetModality(req.Modality, snapshot.InferredModality)
	dataset := model.Dataset{
		ID:               datasetID,
		Name:             name,
		Tags:             normalizeDatasetTags(req.Tags),
		SourceType:       sourceType,
		Modality:         modality,
		Version:          normalizeDatasetVersion(req.Version),
		Size:             humanizeBytes(snapshot.TotalSizeBytes),
		Samples:          snapshot.FileCount,
		Description:      description,
		Path:             path,
		IndexStatus:      "none",
		FileCount:        snapshot.FileCount,
		DirectoryCount:   snapshot.DirectoryCount,
		TotalSizeBytes:   snapshot.TotalSizeBytes,
		FileTypes:        ensureFileTypeMap(snapshot.FileTypes),
		DetectedModality: normalizeDatasetModality("", snapshot.InferredModality),
		LastScanStatus:   firstNonEmpty(snapshot.ScanStatus, "completed"),
		LastScanAt:       &scannedAt,
		LastModifiedAt:   snapshot.RecentModifiedAt,
		UpdatedAt:        scannedAt,
	}
	if server != nil {
		dataset.ServerID = server.ID
		dataset.ServerName = server.Name
	}
	scanRecord := model.DatasetScanRecord{
		ID:               scanID,
		DatasetID:        datasetID,
		RuntimeMode:      scanAdapter.Mode(),
		ScanStatus:       firstNonEmpty(snapshot.ScanStatus, "completed"),
		ValidationStatus: firstNonEmpty(snapshot.ValidationStatus, "ok"),
		RootPath:         path,
		FileCount:        snapshot.FileCount,
		DirectoryCount:   snapshot.DirectoryCount,
		TotalSizeBytes:   snapshot.TotalSizeBytes,
		FileTypes:        ensureFileTypeMap(snapshot.FileTypes),
		HierarchySummary: snapshot.HierarchySummary,
		InferredModality: normalizeDatasetModality("", snapshot.InferredModality),
		RecentModifiedAt: snapshot.RecentModifiedAt,
		ScannedAt:        scannedAt,
		ErrorMessage:     strings.TrimSpace(snapshot.ErrorMessage),
	}
	if server != nil {
		scanRecord.ServerID = server.ID
	}
	previews := make([]model.DatasetPreviewItem, 0, len(snapshot.PreviewItems))
	for _, item := range snapshot.PreviewItems {
		previews = append(previews, model.DatasetPreviewItem{
			ScanRecordID: scanID,
			Name:         firstNonEmpty(strings.TrimSpace(item.Name), previewName(item.RelativePath)),
			ItemType:     firstNonEmpty(strings.TrimSpace(item.ItemType), "file"),
			Category:     firstNonEmpty(strings.TrimSpace(item.Category), "other"),
			RelativePath: normalizeRelativePath(item.RelativePath),
			SizeBytes:    item.SizeBytes,
			Depth:        item.Depth,
		})
	}
	if err = s.repo.CreateImported(ctx, dataset, scanRecord, previews); err != nil {
		return nil, err
	}
	return s.repo.GetDetail(ctx, datasetID)
}

func (s *DatasetService) Update(ctx context.Context, id string, req model.DatasetUpdateRequest) (*model.DatasetDetail, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("dataset name must not be empty")
	}
	description := strings.TrimSpace(req.Description)
	if description == "" {
		return nil, fmt.Errorf("dataset description must not be empty")
	}
	existing, err := s.repo.GetSummaryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}
	req.Name = name
	req.Description = description
	req.Tags = normalizeDatasetTags(req.Tags)
	req.Modality = normalizeDatasetModality(req.Modality, existing.Modality)
	req.Version = firstNonEmpty(strings.TrimSpace(req.Version), existing.Version)
	if err = s.repo.UpdateMetadata(ctx, id, req); err != nil {
		return nil, err
	}
	return s.repo.GetDetail(ctx, id)
}

func (s *DatasetService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *DatasetService) BuildIndex(ctx context.Context, datasetID string) (*model.DatasetIndexTask, error) {
	dataset, err := s.repo.GetSummaryByID(ctx, datasetID)
	if err != nil {
		return nil, err
	}
	if dataset == nil {
		return nil, fmt.Errorf("dataset not found")
	}
	server, err := s.resolveDatasetServer(ctx, dataset.SourceType, dataset.ServerID)
	if err != nil {
		return nil, err
	}
	indexAdapter, err := s.indexAdapters.Resolve(dataset.SourceType)
	if err != nil {
		return nil, err
	}
	task := model.DatasetIndexTask{
		ID:           httpx.NewID("didx"),
		DatasetID:    dataset.ID,
		ServerID:     dataset.ServerID,
		SourceType:   dataset.SourceType,
		ExecutorMode: indexAdapter.Mode(),
		Status:       "building",
		RequestPayload: map[string]interface{}{
			"requestVersion": "2026-03-24",
			"taskId":         "",
			"submittedAt":    time.Now().Format(time.RFC3339),
			"dataset": map[string]interface{}{
				"id":               dataset.ID,
				"name":             dataset.Name,
				"path":             dataset.Path,
				"sourceType":       dataset.SourceType,
				"modality":         dataset.Modality,
				"detectedModality": dataset.DetectedModality,
				"fileCount":        dataset.FileCount,
				"directoryCount":   dataset.DirectoryCount,
				"totalSizeBytes":   dataset.TotalSizeBytes,
			},
		},
		ResponsePayload: map[string]interface{}{},
	}
	if server != nil {
		task.RequestPayload["server"] = map[string]interface{}{
			"id":         server.ID,
			"name":       server.Name,
			"host":       server.Host,
			"authType":   server.AuthType,
			"remoteRoot": server.RemoteRoot,
		}
	}
	task.RequestPayload["taskId"] = task.ID
	if err = s.repo.CreateIndexTask(ctx, task); err != nil {
		return nil, err
	}
	_ = s.repo.AddIndexTaskLog(ctx, task.ID, "info", fmt.Sprintf("Index task created for dataset %s", dataset.Name))
	if err = s.repo.UpdateDatasetIndexStatus(ctx, dataset.ID, "building"); err != nil {
		return nil, err
	}
	update, runErr := indexAdapter.StartIndex(ctx, dataset, &task, server)
	if runErr != nil {
		now := time.Now()
		task.Status = "failed"
		task.ErrorMessage = runErr.Error()
		task.FinishedAt = &now
		_ = s.repo.UpdateIndexTask(ctx, task)
		_ = s.repo.AddIndexTaskLog(ctx, task.ID, "error", runErr.Error())
		_ = s.repo.UpdateDatasetIndexStatus(ctx, dataset.ID, "failed")
		return s.repo.GetIndexTaskByID(ctx, task.ID)
	}
	if err = s.applyIndexTaskUpdate(ctx, &task, update); err != nil {
		return nil, err
	}
	if err = s.repo.UpdateDatasetIndexStatus(ctx, dataset.ID, datasetIndexStatusFromTask(task.Status)); err != nil {
		return nil, err
	}
	return s.repo.GetIndexTaskByID(ctx, task.ID)
}

func (s *DatasetService) SyncIndexTask(ctx context.Context, datasetID string, taskID string) (*model.DatasetIndexTask, error) {
	dataset, err := s.repo.GetSummaryByID(ctx, datasetID)
	if err != nil {
		return nil, err
	}
	if dataset == nil {
		return nil, fmt.Errorf("dataset not found")
	}
	task, err := s.repo.GetIndexTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil || task.DatasetID != datasetID {
		return nil, fmt.Errorf("dataset index task not found")
	}
	server, err := s.resolveDatasetServer(ctx, dataset.SourceType, firstNonEmpty(task.ServerID, dataset.ServerID))
	if err != nil {
		return nil, err
	}
	indexAdapter, err := s.indexAdapters.Resolve(dataset.SourceType)
	if err != nil {
		return nil, err
	}
	update, err := indexAdapter.SyncIndex(ctx, dataset, task, server)
	if err != nil {
		_ = s.repo.AddIndexTaskLog(ctx, task.ID, "error", err.Error())
		return nil, err
	}
	if err = s.applyIndexTaskUpdate(ctx, task, update); err != nil {
		return nil, err
	}
	if err = s.repo.UpdateDatasetIndexStatus(ctx, dataset.ID, datasetIndexStatusFromTask(task.Status)); err != nil {
		return nil, err
	}
	return s.repo.GetIndexTaskByID(ctx, task.ID)
}

func (s *DatasetService) applyIndexTaskUpdate(ctx context.Context, task *model.DatasetIndexTask, update *datasetIndexTaskUpdate) error {
	if update == nil {
		return nil
	}
	if strings.TrimSpace(update.Status) != "" {
		task.Status = strings.TrimSpace(update.Status)
	}
	if strings.TrimSpace(update.RemoteTaskID) != "" {
		task.RemoteTaskID = strings.TrimSpace(update.RemoteTaskID)
	}
	if strings.TrimSpace(update.LogPath) != "" {
		task.LogPath = strings.TrimSpace(update.LogPath)
	}
	if strings.TrimSpace(update.StatusPath) != "" {
		task.StatusPath = strings.TrimSpace(update.StatusPath)
	}
	if strings.TrimSpace(update.ResultPath) != "" {
		task.ResultPath = strings.TrimSpace(update.ResultPath)
	}
	task.ErrorMessage = strings.TrimSpace(update.ErrorMessage)
	if update.ResponsePayload != nil {
		task.ResponsePayload = update.ResponsePayload
	}
	if update.FinishedAt != nil {
		task.FinishedAt = update.FinishedAt
	} else if task.Status == "completed" || task.Status == "failed" {
		now := time.Now()
		task.FinishedAt = &now
	}
	if err := s.repo.UpdateIndexTask(ctx, *task); err != nil {
		return err
	}
	for _, line := range update.Logs {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		level := "info"
		if task.Status == "failed" {
			level = "error"
		}
		if err := s.repo.AddIndexTaskLog(ctx, task.ID, level, trimmed); err != nil {
			return err
		}
	}
	if task.Status == "failed" && task.ErrorMessage != "" {
		if err := s.repo.AddIndexTaskLog(ctx, task.ID, "error", task.ErrorMessage); err != nil {
			return err
		}
	}
	return nil
}

func (s *DatasetService) resolveDatasetServer(ctx context.Context, sourceType string, serverID string) (*model.Server, error) {
	if !strings.EqualFold(sourceType, "remote") {
		return nil, nil
	}
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return nil, fmt.Errorf("serverId is required for remote datasets")
	}
	server, err := s.serverRepo.GetByIDWithSecrets(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, fmt.Errorf("server not found: %s", serverID)
	}
	return server, nil
}

func normalizeDatasetSourceType(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "local", "remote":
		return normalized, nil
	default:
		return "", fmt.Errorf("sourceType must be local or remote")
	}
}

func normalizeDatasetTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func normalizeDatasetVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "imported"
	}
	return value
}

func normalizeDatasetModality(requested string, fallback string) string {
	requested = strings.ToLower(strings.TrimSpace(requested))
	switch requested {
	case "text", "image", "audio", "video", "multimodal":
		return requested
	}
	fallback = strings.ToLower(strings.TrimSpace(fallback))
	switch fallback {
	case "text", "image", "audio", "video", "multimodal":
		return fallback
	default:
		return "text"
	}
}

func ensureFileTypeMap(raw map[string]int64) map[string]int64 {
	if raw == nil {
		return map[string]int64{}
	}
	copyValue := make(map[string]int64, len(raw))
	for key, value := range raw {
		copyValue[key] = value
	}
	return copyValue
}

func datasetIndexStatusFromTask(taskStatus string) string {
	switch strings.ToLower(strings.TrimSpace(taskStatus)) {
	case "completed", "ready":
		return "ready"
	case "failed":
		return "failed"
	case "building", "running", "queued":
		return "building"
	default:
		return "none"
	}
}
