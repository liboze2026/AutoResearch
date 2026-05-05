package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type DatasetAssetStore interface {
	List(context.Context) ([]model.DatasetAsset, error)
	GetByID(context.Context, string) (*model.DatasetAsset, error)
	GetByExistingDatasetRef(context.Context, string) (*model.DatasetAsset, error)
	Create(context.Context, model.DatasetAsset) error
	Update(context.Context, model.DatasetAsset) error
	AddSource(context.Context, model.DatasetAssetSource) error
	ListSources(context.Context, string) ([]model.DatasetAssetSource, error)
}

type DatasetScanReader interface {
	GetSummaryByID(context.Context, string) (*model.Dataset, error)
	GetScanRecordByID(context.Context, string) (*model.DatasetScanRecord, error)
}

type DatasetAssetService struct {
	store          DatasetAssetStore
	scanReader     DatasetScanReader
	workspaceRoot  string
	eventPublisher DatasetAssetEventPublisher
}

type DatasetAssetEventPublisher interface {
	PublishEvent(context.Context, model.AgentEventCreateRequest) (*model.AgentEvent, error)
}

func NewDatasetAssetService(store DatasetAssetStore, scanReader DatasetScanReader, workspaceRoot string) *DatasetAssetService {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &DatasetAssetService{store: store, scanReader: scanReader, workspaceRoot: workspaceRoot}
}

func (s *DatasetAssetService) SetEventPublisher(publisher DatasetAssetEventPublisher) {
	s.eventPublisher = publisher
}

func (s *DatasetAssetService) List(ctx context.Context) ([]model.DatasetAsset, error) {
	return s.store.List(ctx)
}

func (s *DatasetAssetService) GetByID(ctx context.Context, id string) (*model.DatasetAssetDetail, error) {
	item, err := s.store.GetByID(ctx, id)
	if err != nil || item == nil {
		return nil, err
	}
	sources, err := s.store.ListSources(ctx, id)
	if err != nil {
		return nil, err
	}
	return &model.DatasetAssetDetail{Asset: *item, Sources: sources}, nil
}

func (s *DatasetAssetService) Create(ctx context.Context, req model.DatasetAssetCreateRequest) (*model.DatasetAssetDetail, error) {
	now := time.Now()
	asset := model.DatasetAsset{
		ID:                httpx.NewID("dasset"),
		Name:              strings.TrimSpace(req.Name),
		DescriptionMD:     strings.TrimSpace(req.DescriptionMD),
		TaskType:          normalizeDatasetAssetTaskType(req.TaskType),
		Status:            normalizeDatasetAssetStatus(req.Status),
		SourceType:        normalizeDatasetAssetSourceType(req.SourceType, "manual"),
		LocalOrRemotePath: strings.TrimSpace(req.LocalOrRemotePath),
		ReadmeMD:          strings.TrimSpace(req.ReadmeMD),
		LoaderNoteMD:      strings.TrimSpace(req.LoaderNoteMD),
		SchemaNoteMD:      strings.TrimSpace(req.SchemaNoteMD),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := validateDatasetAsset(asset); err != nil {
		return nil, err
	}
	if err := s.store.Create(ctx, asset); err != nil {
		return nil, err
	}
	if err := s.writeDatasetAssetWorkspace(asset, nil); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, asset, nil)
	return &model.DatasetAssetDetail{Asset: asset, Sources: nil}, nil
}

func (s *DatasetAssetService) RegisterFromScan(ctx context.Context, req model.DatasetAssetRegisterFromScanRequest) (*model.DatasetAssetDetail, error) {
	if s.scanReader == nil {
		return nil, fmt.Errorf("dataset scan reader not configured")
	}
	var dataset *model.Dataset
	var scanRecord *model.DatasetScanRecord
	var err error
	if scanID := strings.TrimSpace(req.ScanRecordID); scanID != "" {
		scanRecord, err = s.scanReader.GetScanRecordByID(ctx, scanID)
		if err != nil {
			return nil, err
		}
		if scanRecord == nil {
			return nil, fmt.Errorf("scan record not found")
		}
		dataset, err = s.scanReader.GetSummaryByID(ctx, scanRecord.DatasetID)
		if err != nil {
			return nil, err
		}
	} else {
		datasetRef := strings.TrimSpace(req.ExistingDatasetRef)
		if datasetRef == "" {
			return nil, fmt.Errorf("existingDatasetRef or scanRecordId is required")
		}
		dataset, err = s.scanReader.GetSummaryByID(ctx, datasetRef)
		if err != nil {
			return nil, err
		}
	}
	if dataset == nil {
		return nil, fmt.Errorf("dataset not found")
	}
	existing, err := s.store.GetByExistingDatasetRef(ctx, dataset.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("dataset asset already exists for dataset %s", dataset.ID)
	}
	now := time.Now()
	asset := model.DatasetAsset{
		ID:                  httpx.NewID("dasset"),
		Name:                firstNonEmpty(strings.TrimSpace(req.Name), dataset.Name),
		DescriptionMD:       firstNonEmpty(strings.TrimSpace(req.DescriptionMD), buildDatasetAssetDescription(*dataset)),
		TaskType:            normalizeDatasetAssetTaskType(firstNonEmpty(strings.TrimSpace(req.TaskType), dataset.DetectedModality, dataset.Modality)),
		Status:              normalizeDatasetAssetStatus(req.Status),
		SourceType:          normalizeDatasetAssetSourceType(req.SourceType, "mrag_scan"),
		LocalOrRemotePath:   strings.TrimSpace(dataset.Path),
		ReadmeMD:            firstNonEmpty(strings.TrimSpace(req.ReadmeMD), buildDatasetAssetReadme(*dataset, scanRecord)),
		LoaderNoteMD:        firstNonEmpty(strings.TrimSpace(req.LoaderNoteMD), buildDatasetAssetLoaderNote(*dataset)),
		SchemaNoteMD:        firstNonEmpty(strings.TrimSpace(req.SchemaNoteMD), buildDatasetAssetSchemaNote(*dataset, scanRecord)),
		ExistingDatasetRef:  dataset.ID,
		ExistingDatasetName: dataset.Name,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := validateDatasetAsset(asset); err != nil {
		return nil, err
	}
	if err := s.store.Create(ctx, asset); err != nil {
		return nil, err
	}
	sourceKind := "dataset"
	if scanRecord != nil {
		sourceKind = "scan_record"
	}
	source := model.DatasetAssetSource{
		DatasetAssetID:      asset.ID,
		ExistingDatasetRef:  dataset.ID,
		SourceKind:          sourceKind,
		ExistingDatasetName: dataset.Name,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := s.store.AddSource(ctx, source); err != nil {
		return nil, err
	}
	sources := []model.DatasetAssetSource{source}
	if err := s.writeDatasetAssetWorkspace(asset, sources); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, asset, sources)
	return &model.DatasetAssetDetail{Asset: asset, Sources: sources}, nil
}

func (s *DatasetAssetService) publishEvent(ctx context.Context, asset model.DatasetAsset, sources []model.DatasetAssetSource) {
	if s.eventPublisher == nil {
		return
	}
	inputRefs := []model.AgentInputRef{{RefType: "dataset_asset", RefID: asset.ID, RefPath: asset.LocalOrRemotePath}}
	for _, source := range sources {
		if strings.TrimSpace(source.ExistingDatasetRef) != "" {
			inputRefs = append(inputRefs, model.AgentInputRef{RefType: "dataset", RefID: source.ExistingDatasetRef})
		}
	}
	_, _ = s.eventPublisher.PublishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "dataset_asset_ready",
		SourceRef: "dataset_asset:" + asset.ID,
		InputRefs: inputRefs,
		Payload: map[string]any{
			"dataset_asset_id": asset.ID,
			"name":             asset.Name,
			"task_type":        asset.TaskType,
		},
	})
}

func (s *DatasetAssetService) writeDatasetAssetWorkspace(asset model.DatasetAsset, sources []model.DatasetAssetSource) error {
	paths := workspacepkg.New(s.workspaceRoot)
	assetDir := paths.DatasetAssetDir(asset.ID)
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(assetDir, "README.md"), []byte(ensureTrailingNewline(firstNonEmpty(asset.ReadmeMD, buildDatasetAssetReadmeFromAsset(asset)))), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(assetDir, "loader.md"), []byte(ensureTrailingNewline(firstNonEmpty(asset.LoaderNoteMD, "Loader note is not provided yet."))), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(assetDir, "schema.md"), []byte(ensureTrailingNewline(firstNonEmpty(asset.SchemaNoteMD, "Schema note is not provided yet."))), 0o644); err != nil {
		return err
	}
	metadata := model.DatasetAssetWorkspaceMetadata{
		DatasetAssetID:     asset.ID,
		Name:               asset.Name,
		TaskType:           asset.TaskType,
		Status:             asset.Status,
		SourceType:         asset.SourceType,
		LocalOrRemotePath:  asset.LocalOrRemotePath,
		ExistingDatasetRef: asset.ExistingDatasetRef,
		Sources:            sources,
		UpdatedAt:          asset.UpdatedAt,
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(assetDir, "metadata.json"), data, 0o644)
}

func validateDatasetAsset(asset model.DatasetAsset) error {
	if strings.TrimSpace(asset.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(asset.LocalOrRemotePath) == "" {
		return fmt.Errorf("localOrRemotePath is required")
	}
	switch asset.Status {
	case "draft", "active", "archived":
	default:
		return fmt.Errorf("status must be one of draft, active, archived")
	}
	switch asset.SourceType {
	case "manual", "mrag_scan", "mixed", "mrag_dataset":
	default:
		return fmt.Errorf("sourceType must be one of manual, mrag_scan, mixed, mrag_dataset")
	}
	return nil
}

func normalizeDatasetAssetStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "active"
	}
	return status
}

func normalizeDatasetAssetSourceType(sourceType string, fallback string) string {
	sourceType = strings.TrimSpace(strings.ToLower(sourceType))
	if sourceType == "" {
		return fallback
	}
	return sourceType
}

func normalizeDatasetAssetTaskType(taskType string) string {
	taskType = strings.TrimSpace(strings.ToLower(taskType))
	if taskType == "" {
		return "text"
	}
	return taskType
}

func buildDatasetAssetDescription(dataset model.Dataset) string {
	return fmt.Sprintf("Imported from MRAG scanned dataset `%s`.\n\n- Source type: %s\n- Path: %s", dataset.Name, dataset.SourceType, dataset.Path)
}

func buildDatasetAssetReadme(dataset model.Dataset, scanRecord *model.DatasetScanRecord) string {
	lines := []string{
		fmt.Sprintf("# %s", dataset.Name),
		"",
		"This dataset asset is registered from an existing MRAG scan result.",
		"",
		fmt.Sprintf("- Source type: %s", dataset.SourceType),
		fmt.Sprintf("- Path: %s", dataset.Path),
		fmt.Sprintf("- File count: %d", dataset.FileCount),
		fmt.Sprintf("- Directory count: %d", dataset.DirectoryCount),
	}
	if dataset.ServerName != "" {
		lines = append(lines, fmt.Sprintf("- Server: %s", dataset.ServerName))
	}
	if scanRecord != nil {
		lines = append(lines, fmt.Sprintf("- Scan record: %s", scanRecord.ID))
	}
	return strings.Join(lines, "\n")
}

func buildDatasetAssetReadmeFromAsset(asset model.DatasetAsset) string {
	return fmt.Sprintf("# %s\n\n- Task type: %s\n- Path: %s\n", asset.Name, asset.TaskType, asset.LocalOrRemotePath)
}

func buildDatasetAssetLoaderNote(dataset model.Dataset) string {
	return fmt.Sprintf("- Use the existing MRAG dataset record `%s` as the canonical scan source.\n- Read files from `%s`.\n- Preserve the current source type `%s` instead of re-scanning here.", dataset.ID, dataset.Path, dataset.SourceType)
}

func buildDatasetAssetSchemaNote(dataset model.Dataset, scanRecord *model.DatasetScanRecord) string {
	lines := []string{
		fmt.Sprintf("- Detected modality: %s", firstNonEmpty(dataset.DetectedModality, dataset.Modality)),
		fmt.Sprintf("- Known file types: %v", dataset.FileTypes),
	}
	if scanRecord != nil && len(scanRecord.HierarchySummary) > 0 {
		lines = append(lines, fmt.Sprintf("- Hierarchy summary items: %d", len(scanRecord.HierarchySummary)))
	}
	return strings.Join(lines, "\n")
}

func ensureTrailingNewline(value string) string {
	if strings.HasSuffix(value, "\n") {
		return value
	}
	return value + "\n"
}
