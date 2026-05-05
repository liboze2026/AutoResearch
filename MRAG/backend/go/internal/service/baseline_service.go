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

type BaselineStore interface {
	List(context.Context) ([]model.Baseline, error)
	GetByID(context.Context, string) (*model.Baseline, error)
	Create(context.Context, model.Baseline) error
	Update(context.Context, model.Baseline) error
}

type BaselineDatasetAssetReader interface {
	GetByID(context.Context, string) (*model.DatasetAsset, error)
}

type BaselineService struct {
	store         BaselineStore
	assetReader   BaselineDatasetAssetReader
	workspaceRoot string
}

func NewBaselineService(store BaselineStore, assetReader BaselineDatasetAssetReader, workspaceRoot string) *BaselineService {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &BaselineService{store: store, assetReader: assetReader, workspaceRoot: workspaceRoot}
}

func (s *BaselineService) List(ctx context.Context) ([]model.Baseline, error) {
	return s.store.List(ctx)
}

func (s *BaselineService) GetByID(ctx context.Context, id string) (*model.BaselineDetail, error) {
	item, err := s.store.GetByID(ctx, id)
	if err != nil || item == nil {
		return nil, err
	}
	asset, err := s.assetReader.GetByID(ctx, item.DatasetAssetID)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, fmt.Errorf("dataset asset not found")
	}
	return &model.BaselineDetail{Baseline: *item, DatasetAsset: *asset}, nil
}

func (s *BaselineService) Create(ctx context.Context, req model.BaselineCreateRequest) (*model.BaselineDetail, error) {
	asset, err := s.assetReader.GetByID(ctx, strings.TrimSpace(req.DatasetAssetID))
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, fmt.Errorf("dataset asset not found")
	}
	now := time.Now()
	item := model.Baseline{
		ID:               httpx.NewID("baseline"),
		DatasetAssetID:   asset.ID,
		Name:             strings.TrimSpace(req.Name),
		MetricSchemaJSON: ensureAnyMap(req.MetricSchemaJSON),
		ResultJSON:       ensureAnyMap(req.ResultJSON),
		NoteMD:           strings.TrimSpace(req.NoteMD),
		SourceType:       normalizeBaselineSourceType(req.SourceType),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := validateBaseline(item); err != nil {
		return nil, err
	}
	if err := s.store.Create(ctx, item); err != nil {
		return nil, err
	}
	if err := s.writeBaselineWorkspace(*asset, item); err != nil {
		return nil, err
	}
	return &model.BaselineDetail{Baseline: item, DatasetAsset: *asset}, nil
}

func (s *BaselineService) Update(ctx context.Context, id string, req model.BaselineUpdateRequest) (*model.BaselineDetail, error) {
	item, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("baseline not found")
	}
	if req.Name != nil {
		item.Name = strings.TrimSpace(*req.Name)
	}
	if req.MetricSchemaJSON != nil {
		item.MetricSchemaJSON = ensureAnyMap(req.MetricSchemaJSON)
	}
	if req.ResultJSON != nil {
		item.ResultJSON = ensureAnyMap(req.ResultJSON)
	}
	if req.NoteMD != nil {
		item.NoteMD = strings.TrimSpace(*req.NoteMD)
	}
	if req.SourceType != nil {
		item.SourceType = normalizeBaselineSourceType(*req.SourceType)
	}
	item.UpdatedAt = time.Now()
	if err := validateBaseline(*item); err != nil {
		return nil, err
	}
	if err := s.store.Update(ctx, *item); err != nil {
		return nil, err
	}
	asset, err := s.assetReader.GetByID(ctx, item.DatasetAssetID)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, fmt.Errorf("dataset asset not found")
	}
	if err := s.writeBaselineWorkspace(*asset, *item); err != nil {
		return nil, err
	}
	return &model.BaselineDetail{Baseline: *item, DatasetAsset: *asset}, nil
}

func (s *BaselineService) writeBaselineWorkspace(asset model.DatasetAsset, baseline model.Baseline) error {
	paths := workspacepkg.New(s.workspaceRoot)
	baseDir := paths.DatasetBaselinesDir(asset.ID)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return err
	}
	baselineDir := filepath.Join(baseDir, baseline.ID)
	if err := os.MkdirAll(baselineDir, 0o755); err != nil {
		return err
	}
	metricRaw, err := json.MarshalIndent(ensureAnyMap(baseline.MetricSchemaJSON), "", "  ")
	if err != nil {
		return err
	}
	resultRaw, err := json.MarshalIndent(ensureAnyMap(baseline.ResultJSON), "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(baselineDir, "metric_schema.json"), metricRaw, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(baselineDir, "result.json"), resultRaw, 0o644); err != nil {
		return err
	}
	note := fmt.Sprintf("# %s\n\n- Dataset Asset: %s\n- Source Type: %s\n\n%s\n", baseline.Name, asset.Name, baseline.SourceType, firstNonEmpty(baseline.NoteMD, "No note yet."))
	if err := os.WriteFile(filepath.Join(baselineDir, "note.md"), []byte(note), 0o644); err != nil {
		return err
	}
	metadata := map[string]any{
		"baselineId":     baseline.ID,
		"datasetAssetId": baseline.DatasetAssetID,
		"name":           baseline.Name,
		"sourceType":     baseline.SourceType,
		"updatedAt":      baseline.UpdatedAt,
	}
	metadataRaw, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(baselineDir, "metadata.json"), metadataRaw, 0o644)
}

func validateBaseline(item model.Baseline) error {
	if strings.TrimSpace(item.DatasetAssetID) == "" {
		return fmt.Errorf("datasetAssetId is required")
	}
	if strings.TrimSpace(item.Name) == "" {
		return fmt.Errorf("name is required")
	}
	switch item.SourceType {
	case "manual", "result_archive", "mixed":
	default:
		return fmt.Errorf("sourceType must be one of manual, result_archive, mixed")
	}
	return nil
}

func normalizeBaselineSourceType(sourceType string) string {
	sourceType = strings.TrimSpace(strings.ToLower(sourceType))
	if sourceType == "" {
		return "manual"
	}
	return sourceType
}

func ensureAnyMap(raw map[string]any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	copyValue := make(map[string]any, len(raw))
	for key, value := range raw {
		copyValue[key] = value
	}
	return copyValue
}
