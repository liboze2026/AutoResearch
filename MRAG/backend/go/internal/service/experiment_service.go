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

type ExperimentStore interface {
	List(context.Context) ([]model.Experiment, error)
	GetByID(context.Context, string) (*model.Experiment, error)
	Create(context.Context, model.Experiment) error
	Update(context.Context, model.Experiment) error
}

type ExperimentSpecStore interface {
	ListByExperimentID(context.Context, string) ([]model.ExperimentSpec, error)
	GetLatestByExperimentID(context.Context, string) (*model.ExperimentSpec, error)
	Create(context.Context, model.ExperimentSpec) error
}

type ExperimentDatasetAssetReader interface {
	GetByID(context.Context, string) (*model.DatasetAsset, error)
}

type ExperimentIdeaReader interface {
	GetByID(context.Context, string) (*model.Idea, error)
}

type ExperimentBaselineReader interface {
	GetByID(context.Context, string) (*model.Baseline, error)
}

type ExperimentResultArchiveReader interface {
	ListByDatasetAssetID(context.Context, string) ([]model.ResultArchive, error)
}

type ExperimentService struct {
	store          ExperimentStore
	specStore      ExperimentSpecStore
	assetReader    ExperimentDatasetAssetReader
	ideaReader     ExperimentIdeaReader
	baselineReader ExperimentBaselineReader
	archiveReader  ExperimentResultArchiveReader
	workspaceRoot  string
}

func NewExperimentService(
	store ExperimentStore,
	specStore ExperimentSpecStore,
	assetReader ExperimentDatasetAssetReader,
	ideaReader ExperimentIdeaReader,
	baselineReader ExperimentBaselineReader,
	archiveReader ExperimentResultArchiveReader,
	workspaceRoot string,
) *ExperimentService {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &ExperimentService{
		store:          store,
		specStore:      specStore,
		assetReader:    assetReader,
		ideaReader:     ideaReader,
		baselineReader: baselineReader,
		archiveReader:  archiveReader,
		workspaceRoot:  workspaceRoot,
	}
}

func (s *ExperimentService) List(ctx context.Context) ([]model.Experiment, error) {
	return s.store.List(ctx)
}

func (s *ExperimentService) GetByID(ctx context.Context, id string) (*model.ExperimentDetail, error) {
	item, err := s.store.GetByID(ctx, id)
	if err != nil || item == nil {
		return nil, err
	}
	return s.buildDetail(ctx, *item)
}

func (s *ExperimentService) Create(ctx context.Context, req model.ExperimentCreateRequest) (*model.ExperimentDetail, error) {
	asset, idea, baseline, err := s.resolveDependencies(ctx, req.DatasetAssetID, req.IdeaID, req.BaselineID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = defaultExperimentTitle(asset, idea)
	}
	item := model.Experiment{
		ID:             httpx.NewID("exp"),
		IdeaID:         strings.TrimSpace(req.IdeaID),
		DatasetAssetID: asset.ID,
		BaselineID:     strings.TrimSpace(req.BaselineID),
		Title:          title,
		Status:         "draft",
		Priority:       req.Priority,
		SummaryMD:      strings.TrimSpace(req.SummaryMD),
		OwnerNoteMD:    strings.TrimSpace(req.OwnerNoteMD),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := validateExperiment(item); err != nil {
		return nil, err
	}
	if err := s.store.Create(ctx, item); err != nil {
		return nil, err
	}
	if err := s.writeExperimentWorkspace(item); err != nil {
		return nil, err
	}

	detail := &model.ExperimentDetail{
		Experiment:   item,
		DatasetAsset: *asset,
		Idea:         idea,
		Baseline:     baseline,
	}
	return detail, nil
}

func (s *ExperimentService) GenerateSpec(ctx context.Context, experimentID string) (*model.ExperimentSpecDetail, error) {
	item, err := s.store.GetByID(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("experiment not found")
	}
	asset, idea, baseline, err := s.resolveDependencies(ctx, item.DatasetAssetID, item.IdeaID, item.BaselineID)
	if err != nil {
		return nil, err
	}

	archives := make([]model.ResultArchive, 0)
	if s.archiveReader != nil {
		archives, err = s.archiveReader.ListByDatasetAssetID(ctx, item.DatasetAssetID)
		if err != nil {
			return nil, err
		}
	}

	latest, err := s.specStore.GetLatestByExperimentID(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	version := 1
	if latest != nil {
		version = latest.Version + 1
	}

	now := time.Now()
	specPayload := buildExperimentSpecPayload(*item, asset, idea, baseline, archives, s.workspaceRoot)
	planDoc, err := s.loadPlannerPlan(experimentID)
	if err != nil {
		return nil, err
	}
	specPayload = applyPlannerPlanToSpec(specPayload, planDoc)
	spec := model.ExperimentSpec{
		ID:           httpx.NewID("espec"),
		ExperimentID: experimentID,
		SpecJSON:     specPayload,
		TemplateType: asString(specPayload["train_template_type"]),
		GeneratedFrom: map[string]interface{}{
			"strategy":             "rule_based_v1",
			"supportsPlannerAgent": true,
			"supportsCodingAgent":  true,
			"sourceObjects": map[string]interface{}{
				"datasetAssetId": asset.ID,
				"ideaId":         item.IdeaID,
				"baselineId":     item.BaselineID,
			},
		},
		Version:   version,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.specStore.Create(ctx, spec); err != nil {
		return nil, err
	}

	item.Status = "spec_ready"
	item.UpdatedAt = now
	if err := s.store.Update(ctx, *item); err != nil {
		return nil, err
	}

	workspacePath, err := s.writeSpecWorkspace(*item, spec)
	if err != nil {
		return nil, err
	}
	return &model.ExperimentSpecDetail{
		Spec:            spec,
		WorkspacePath:   workspacePath,
		GeneratorSource: "rule_based_v1",
	}, nil
}

func (s *ExperimentService) GetLatestSpec(ctx context.Context, experimentID string) (*model.ExperimentSpecDetail, error) {
	item, err := s.store.GetByID(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("experiment not found")
	}
	spec, err := s.specStore.GetLatestByExperimentID(ctx, experimentID)
	if err != nil || spec == nil {
		return nil, err
	}
	paths := workspacepkg.New(s.workspaceRoot)
	return &model.ExperimentSpecDetail{
		Spec:            *spec,
		WorkspacePath:   filepath.Join(paths.ExperimentDir(experimentID), "spec.json"),
		GeneratorSource: asString(spec.GeneratedFrom["strategy"]),
	}, nil
}

func (s *ExperimentService) buildDetail(ctx context.Context, item model.Experiment) (*model.ExperimentDetail, error) {
	asset, idea, baseline, err := s.resolveDependencies(ctx, item.DatasetAssetID, item.IdeaID, item.BaselineID)
	if err != nil {
		return nil, err
	}
	latestSpec, err := s.specStore.GetLatestByExperimentID(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	return &model.ExperimentDetail{
		Experiment:   item,
		DatasetAsset: *asset,
		Idea:         idea,
		Baseline:     baseline,
		LatestSpec:   latestSpec,
	}, nil
}

func (s *ExperimentService) resolveDependencies(ctx context.Context, datasetAssetID string, ideaID string, baselineID string) (*model.DatasetAsset, *model.Idea, *model.Baseline, error) {
	asset, err := s.assetReader.GetByID(ctx, strings.TrimSpace(datasetAssetID))
	if err != nil {
		return nil, nil, nil, err
	}
	if asset == nil {
		return nil, nil, nil, fmt.Errorf("dataset asset not found")
	}

	var idea *model.Idea
	if strings.TrimSpace(ideaID) != "" {
		idea, err = s.ideaReader.GetByID(ctx, strings.TrimSpace(ideaID))
		if err != nil {
			return nil, nil, nil, err
		}
		if idea == nil {
			return nil, nil, nil, fmt.Errorf("idea not found")
		}
	}

	var baseline *model.Baseline
	if strings.TrimSpace(baselineID) != "" {
		baseline, err = s.baselineReader.GetByID(ctx, strings.TrimSpace(baselineID))
		if err != nil {
			return nil, nil, nil, err
		}
		if baseline == nil {
			return nil, nil, nil, fmt.Errorf("baseline not found")
		}
		if baseline.DatasetAssetID != asset.ID {
			return nil, nil, nil, fmt.Errorf("baseline does not belong to dataset asset")
		}
	}
	return asset, idea, baseline, nil
}

func (s *ExperimentService) writeExperimentWorkspace(item model.Experiment) error {
	paths := workspacepkg.New(s.workspaceRoot)
	expDir := paths.ExperimentDir(item.ID)
	if err := os.MkdirAll(expDir, 0o755); err != nil {
		return err
	}
	metadata := model.ExperimentWorkspaceMetadata{
		ExperimentID:   item.ID,
		DatasetAssetID: item.DatasetAssetID,
		IdeaID:         item.IdeaID,
		BaselineID:     item.BaselineID,
		Title:          item.Title,
		Status:         item.Status,
		Priority:       item.Priority,
		UpdatedAt:      item.UpdatedAt,
	}
	raw, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(expDir, "metadata.json"), raw, 0o644)
}

func (s *ExperimentService) writeSpecWorkspace(item model.Experiment, spec model.ExperimentSpec) (string, error) {
	paths := workspacepkg.New(s.workspaceRoot)
	expDir := paths.ExperimentDir(item.ID)
	if err := os.MkdirAll(expDir, 0o755); err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(spec.SpecJSON, "", "  ")
	if err != nil {
		return "", err
	}
	specPath := filepath.Join(expDir, "spec.json")
	if err := os.WriteFile(specPath, raw, 0o644); err != nil {
		return "", err
	}
	return specPath, nil
}

func validateExperiment(item model.Experiment) error {
	if strings.TrimSpace(item.DatasetAssetID) == "" {
		return fmt.Errorf("datasetAssetId is required")
	}
	if strings.TrimSpace(item.Title) == "" {
		return fmt.Errorf("title is required")
	}
	switch item.Status {
	case "draft", "spec_ready", "queued", "scheduled", "running", "succeeded", "failed", "cancelled", "archived":
	default:
		return fmt.Errorf("invalid experiment status")
	}
	return nil
}

func defaultExperimentTitle(asset *model.DatasetAsset, idea *model.Idea) string {
	if asset == nil {
		return "New Experiment"
	}
	if idea != nil && strings.TrimSpace(idea.Title) != "" {
		return fmt.Sprintf("%s | %s", idea.Title, asset.Name)
	}
	return fmt.Sprintf("Experiment | %s", asset.Name)
}

func buildExperimentSpecPayload(exp model.Experiment, asset *model.DatasetAsset, idea *model.Idea, baseline *model.Baseline, archives []model.ResultArchive, workspaceRoot string) map[string]interface{} {
	paths := workspacepkg.New(workspaceRoot)
	outputDir := filepath.Join(paths.ExperimentDir(exp.ID), "outputs")
	return map[string]interface{}{
		"experiment_id":       exp.ID,
		"dataset_ref":         buildDatasetRef(asset),
		"dataset_loader_ref":  buildDatasetLoaderRef(asset),
		"train_template_type": buildTrainTemplateType(asset, idea),
		"model_name":          defaultModelName(asset),
		"hyperparams":         defaultHyperparams(asset),
		"output_dir":          outputDir,
		"expected_metrics":    buildExpectedMetrics(asset, baseline),
		"comparison_targets":  buildComparisonTargets(exp, baseline, archives),
		"planner_extensions": map[string]interface{}{
			"planner_agent": "reserved",
			"coding_agent":  "reserved",
		},
	}
}

func buildDatasetRef(asset *model.DatasetAsset) map[string]interface{} {
	return map[string]interface{}{
		"dataset_asset_id":     asset.ID,
		"existing_dataset_ref": asset.ExistingDatasetRef,
		"name":                 asset.Name,
		"task_type":            asset.TaskType,
		"source_type":          asset.SourceType,
		"local_or_remote_path": asset.LocalOrRemotePath,
	}
}

func buildDatasetLoaderRef(asset *model.DatasetAsset) map[string]interface{} {
	loaderID := "mrag.dataset_asset_loader.v1"
	if strings.TrimSpace(asset.ExistingDatasetRef) != "" {
		loaderID = "mrag.scanned_dataset_loader.v1"
	}
	return map[string]interface{}{
		"loader_id": loaderID,
		"task_type": asset.TaskType,
		"notes":     firstNonEmpty(strings.TrimSpace(asset.LoaderNoteMD), "Use stage1 dataset asset metadata as the canonical dataset entry."),
	}
}

func buildTrainTemplateType(asset *model.DatasetAsset, idea *model.Idea) string {
	if idea != nil && strings.Contains(strings.ToLower(idea.Title), "lora") {
		return "lora_sft_v1"
	}
	switch strings.ToLower(strings.TrimSpace(asset.TaskType)) {
	case "text", "qa", "rag":
		return "text_finetune_v1"
	default:
		return "generic_train_v1"
	}
}

func defaultModelName(asset *model.DatasetAsset) string {
	switch strings.ToLower(strings.TrimSpace(asset.TaskType)) {
	case "text", "qa", "rag":
		return "mock/llama3.1-8b-instruct"
	default:
		return "mock/base-model-v1"
	}
}

func defaultHyperparams(asset *model.DatasetAsset) map[string]interface{} {
	return map[string]interface{}{
		"epochs":                3,
		"batch_size":            8,
		"learning_rate":         1e-4,
		"gradient_accumulation": 4,
		"max_seq_len":           hyperparamSeqLen(asset),
	}
}

func hyperparamSeqLen(asset *model.DatasetAsset) int {
	if strings.EqualFold(strings.TrimSpace(asset.TaskType), "text") {
		return 2048
	}
	return 1024
}

func buildExpectedMetrics(asset *model.DatasetAsset, baseline *model.Baseline) map[string]interface{} {
	if baseline != nil && len(baseline.MetricSchemaJSON) > 0 {
		return cloneAnyMap(baseline.MetricSchemaJSON)
	}
	switch strings.ToLower(strings.TrimSpace(asset.TaskType)) {
	case "text", "qa", "rag":
		return map[string]interface{}{"primary": "accuracy", "secondary": []string{"loss", "latency_ms"}}
	default:
		return map[string]interface{}{"primary": "score"}
	}
}

func buildComparisonTargets(exp model.Experiment, baseline *model.Baseline, archives []model.ResultArchive) []map[string]interface{} {
	targets := make([]map[string]interface{}, 0)
	if baseline != nil {
		targets = append(targets, map[string]interface{}{
			"type":       "baseline",
			"id":         baseline.ID,
			"name":       baseline.Name,
			"sourceType": baseline.SourceType,
		})
	}
	for _, archive := range archives {
		targets = append(targets, map[string]interface{}{
			"type":           "result_archive",
			"id":             archive.ID,
			"title":          archive.Title,
			"status":         archive.Status,
			"baselineId":     archive.BaselineID,
			"ideaId":         archive.IdeaID,
			"datasetAssetId": archive.DatasetAssetID,
		})
	}
	return targets
}

func cloneAnyMap(input map[string]any) map[string]interface{} {
	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func asString(value interface{}) string {
	raw, _ := value.(string)
	return raw
}
