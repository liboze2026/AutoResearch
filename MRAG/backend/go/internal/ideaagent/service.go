package ideaagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

const ideaOutputSchemaRef = "schemas/idea-generator-output-v1.json"

type jobCreator interface {
	Create(context.Context, model.AgentJobCreateRequest) (*model.AgentJob, error)
	GetByID(context.Context, string) (*model.AgentJob, error)
}

type jobUpdater interface {
	Update(context.Context, model.AgentJob) error
}

type triggerService interface {
	Trigger(context.Context, string, model.AgentJobTriggerRequest) (*model.AgentJob, error)
}

type ideaWriter interface {
	PersistStructuredIdea(context.Context, model.StructuredIdeaPersistRequest) (*model.IdeaDetail, error)
	GetByID(context.Context, string) (*model.IdeaDetail, error)
}

type paperReader interface {
	List(context.Context) ([]model.Paper, error)
	GetByID(context.Context, string) (*model.PaperDetail, error)
	ListInsights(context.Context, string) ([]model.PaperInsight, error)
}

type datasetAssetReader interface {
	List(context.Context) ([]model.DatasetAsset, error)
	GetByID(context.Context, string) (*model.DatasetAssetDetail, error)
}

type Service struct {
	jobs          jobCreator
	jobUpdates    jobUpdater
	triggers      triggerService
	ideas         ideaWriter
	papers        paperReader
	datasets      datasetAssetReader
	workspaceRoot string
}

func NewService(jobs jobCreator, jobUpdates jobUpdater, triggers triggerService, ideas ideaWriter, papers paperReader, datasets datasetAssetReader, workspaceRoot string) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{
		jobs:          jobs,
		jobUpdates:    jobUpdates,
		triggers:      triggers,
		ideas:         ideas,
		papers:        papers,
		datasets:      datasets,
		workspaceRoot: workspaceRoot,
	}
}

func (s *Service) Run(ctx context.Context, req model.IdeaGeneratorRunRequest) (*model.IdeaGeneratorRunResult, error) {
	normalizedReq, err := s.normalizeRequest(req)
	if err != nil {
		return nil, err
	}
	insights, insightRefs, paperSources, err := s.resolveInsights(ctx, normalizedReq.PaperInsightRefs)
	if err != nil {
		return nil, err
	}
	datasets, datasetRefs, err := s.resolveDatasets(ctx, normalizedReq.DatasetAssetRefs)
	if err != nil {
		return nil, err
	}
	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:       "idea_generator",
		ExecutionMode:   normalizedReq.ExecutionMode,
		ModelProvider:   normalizedReq.ModelProvider,
		ModelName:       normalizedReq.ModelName,
		PromptVersion:   normalizedReq.PromptVersion,
		InputRefs:       append(insightRefs, datasetRefs...),
		OutputSchemaRef: ideaOutputSchemaRef,
		SkillRefs:       normalizedReq.SkillRefs,
		ToolRefs:        normalizedReq.ToolRefs,
		MemoryRefs:      normalizedReq.MemoryRefs,
		Metadata: map[string]any{
			"paper_insight_refs": normalizedReq.PaperInsightRefs,
			"dataset_asset_refs": normalizedReq.DatasetAssetRefs,
			"human_hints":        normalizedReq.HumanHints,
			"manual_idea":        manualIdeaMap(normalizedReq.ManualIdea),
			"paper_insights":     insights,
			"dataset_assets":     datasets,
			"paper_sources":      paperSources,
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}
	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata: map[string]any{
			"agent_type": "idea_generator",
		},
	})
	if err != nil {
		return nil, err
	}
	result, err := s.resultFromJob(ctx, job)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) PostProcess(ctx context.Context, job *model.AgentJob) error {
	if job == nil {
		return nil
	}
	payload := extractStructuredIdea(job.NormalizedPayload)
	payload.PaperInsightRefs = normalizeStringSlice(stringSliceValue(job.Metadata["paper_insight_refs"]))
	payload.HumanHints = normalizeStringSlice(stringSliceValue(job.Metadata["human_hints"]))
	payload.TargetDatasetRefs = normalizeStringSlice(payload.TargetDatasetRefs)
	payload.DatasetEvalProtocolRefs = normalizeStringSlice(payload.DatasetEvalProtocolRefs)
	if len(payload.TargetDatasetRefs) == 0 || len(payload.DatasetEvalProtocolRefs) == 0 {
		enriched, err := s.enrichWithAvailableDatasets(ctx, payload)
		if err != nil {
			return err
		}
		payload = enriched
	}
	paperSources := parsePaperSources(job.Metadata["paper_sources"])
	if len(paperSources) == 0 {
		paperSources = buildPaperSourcesFromInputRefs(job.InputRefs)
	}
	sourceType := "auto"
	if hasManualIdeaMetadata(job.Metadata["manual_idea"]) {
		sourceType = "human"
	}
	if sourceType == "human" && len(paperSources) > 0 {
		sourceType = "mixed"
	}
	detail, err := s.ideas.PersistStructuredIdea(ctx, model.StructuredIdeaPersistRequest{
		StructuredIdea: payload,
		SourceType:     sourceType,
		PaperSources:   paperSources,
		DatasetRefs:    payload.TargetDatasetRefs,
		EvalPlanRefs:   payload.DatasetEvalProtocolRefs,
		HumanHints:     payload.HumanHints,
		GeneratedFrom:  "idea_generator_agent",
	})
	if err != nil {
		return err
	}
	job.NormalizedPayload = updateJobPayload(job.NormalizedPayload, detail)
	job.UpdatedAt = time.Now()
	return s.jobUpdates.Update(ctx, *job)
}

func (s *Service) normalizeRequest(req model.IdeaGeneratorRunRequest) (model.IdeaGeneratorRunRequest, error) {
	req.PaperInsightRefs = normalizeStringSlice(req.PaperInsightRefs)
	req.DatasetAssetRefs = normalizeStringSlice(req.DatasetAssetRefs)
	req.HumanHints = normalizeStringSlice(req.HumanHints)
	req.ExecutionMode = strings.TrimSpace(req.ExecutionMode)
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	if len(req.PaperInsightRefs) == 0 && req.ManualIdea == nil {
		return req, fmt.Errorf("paper_insight_refs is required unless manual_idea is provided")
	}
	switch req.ExecutionMode {
	case "", "mock":
		req.ExecutionMode = "mock"
	case "api", "codex_cli":
	default:
		return req, fmt.Errorf("execution_mode must be one of api, codex_cli, mock")
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "codex"
	}
	if req.ModelName == "" {
		req.ModelName = "idea-generator-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	return req, nil
}

func (s *Service) resultFromJob(ctx context.Context, job *model.AgentJob) (*model.IdeaGeneratorRunResult, error) {
	if job == nil {
		return nil, fmt.Errorf("idea generator job not found")
	}
	ideaID := stringValue(job.NormalizedPayload["idea_id"])
	result := &model.IdeaGeneratorRunResult{
		Job:      job,
		Warnings: append([]string{}, job.Warnings...),
	}
	if ideaID == "" {
		return result, nil
	}
	detail, err := s.ideas.GetByID(ctx, ideaID)
	if err != nil {
		return nil, err
	}
	result.Idea = detail
	return result, nil
}

func (s *Service) resolveInsights(ctx context.Context, refs []string) ([]map[string]any, []model.AgentInputRef, []model.IdeaSource, error) {
	out := make([]map[string]any, 0, len(refs))
	inputRefs := make([]model.AgentInputRef, 0, len(refs))
	sources := make([]model.IdeaSource, 0, len(refs))
	for _, insightID := range refs {
		paperID, insight, paperTitle, err := s.findInsight(ctx, insightID)
		if err != nil {
			return nil, nil, nil, err
		}
		if insight == nil {
			return nil, nil, nil, fmt.Errorf("paper insight not found")
		}
		insightPath := filepath.Join(workspacepkg.New(s.workspaceRoot).PapersInsights(), paperID, "insight_agent_output.json")
		out = append(out, map[string]any{
			"insight_id":          insight.ID,
			"paper_id":            paperID,
			"paper_title":         paperTitle,
			"summary_md":          strings.TrimSpace(insight.SummaryMD),
			"contributions_json":  toStringSlice(insight.ContributionsJSON),
			"novelty_points":      toStringSlice(insight.NoveltyPointsJSON),
			"insight_output_path": insightPath,
		})
		inputRefs = append(inputRefs, model.AgentInputRef{RefType: "insight", RefID: insight.ID, RefPath: insightPath})
		sources = append(sources, model.IdeaSource{
			PaperID:        paperID,
			PaperInsightID: insight.ID,
			PaperTitle:     paperTitle,
			SourceNote:     "idea generator paper insight source",
		})
	}
	return out, inputRefs, sources, nil
}

func (s *Service) resolveDatasets(ctx context.Context, refs []string) ([]map[string]any, []model.AgentInputRef, error) {
	out := make([]map[string]any, 0, len(refs))
	inputRefs := make([]model.AgentInputRef, 0, len(refs))
	for _, datasetAssetID := range refs {
		item, err := s.datasets.GetByID(ctx, datasetAssetID)
		if err != nil {
			return nil, nil, err
		}
		if item == nil {
			return nil, nil, fmt.Errorf("dataset asset not found")
		}
		evalplanPath := filepath.Join(workspacepkg.New(s.workspaceRoot).DatasetAssetDir(item.Asset.ID), "evalplan.json")
		evalProtocol := map[string]any{}
		splitStrategy := ""
		if raw, readErr := os.ReadFile(evalplanPath); readErr == nil {
			var parsed map[string]any
			if json.Unmarshal(raw, &parsed) == nil {
				evalProtocol = mapValue(parsed["eval_protocol_json"])
				splitStrategy = stringValue(parsed["split_strategy"])
			}
		}
		out = append(out, map[string]any{
			"dataset_asset_id":   item.Asset.ID,
			"name":               item.Asset.Name,
			"task_type":          item.Asset.TaskType,
			"evalplan_path":      evalplanPath,
			"eval_protocol_json": evalProtocol,
			"split_strategy":     splitStrategy,
		})
		inputRefs = append(inputRefs, model.AgentInputRef{
			RefType: "dataset_asset",
			RefID:   item.Asset.ID,
			RefPath: evalplanPath,
			Metadata: map[string]any{
				"name":          item.Asset.Name,
				"task_type":     item.Asset.TaskType,
				"evalplan_path": evalplanPath,
			},
		})
	}
	return out, inputRefs, nil
}

func (s *Service) findInsight(ctx context.Context, insightID string) (string, *model.PaperInsight, string, error) {
	papers, err := s.papers.List(ctx)
	if err != nil {
		return "", nil, "", err
	}
	for _, paper := range papers {
		insights, listErr := s.papers.ListInsights(ctx, paper.ID)
		if listErr != nil {
			return "", nil, "", listErr
		}
		for _, insight := range insights {
			if strings.EqualFold(insight.ID, insightID) {
				copyInsight := insight
				return paper.ID, &copyInsight, paper.Title, nil
			}
		}
	}
	return "", nil, "", nil
}

func (s *Service) enrichWithAvailableDatasets(ctx context.Context, payload model.StructuredIdeaPayload) (model.StructuredIdeaPayload, error) {
	items, err := s.datasets.List(ctx)
	if err != nil {
		return payload, err
	}
	if len(payload.TargetDatasetRefs) > 0 && len(payload.DatasetEvalProtocolRefs) > 0 {
		return payload, nil
	}
	for _, item := range items {
		if len(payload.TargetDatasetRefs) >= 2 {
			break
		}
		payload.TargetDatasetRefs = appendIfMissing(payload.TargetDatasetRefs, item.ID)
		evalplanPath := filepath.Join(workspacepkg.New(s.workspaceRoot).DatasetAssetDir(item.ID), "evalplan.json")
		if _, statErr := os.Stat(evalplanPath); statErr == nil {
			payload.DatasetEvalProtocolRefs = appendIfMissing(payload.DatasetEvalProtocolRefs, evalplanPath)
		}
	}
	return payload, nil
}

func extractStructuredIdea(payload map[string]any) model.StructuredIdeaPayload {
	return model.StructuredIdeaPayload{
		Title:                   stringValue(payload["title"]),
		DescriptionMD:           stringValue(payload["description_md"]),
		ResearchDirection:       stringValue(payload["research_direction"]),
		TargetDatasetRefs:       normalizeStringSlice(stringSliceValue(payload["target_dataset_refs"])),
		DatasetEvalProtocolRefs: normalizeStringSlice(stringSliceValue(payload["dataset_eval_protocol_refs"])),
		InnovationType:          stringValue(payload["innovation_type"]),
		ExpectedAdvantage:       stringValue(payload["expected_advantage"]),
		RiskPoints:              normalizeStringSlice(stringSliceValue(payload["risk_points"])),
		Priority:                intValue(payload["priority"]),
		Confidence:              floatValue(payload["confidence"]),
	}
}

func updateJobPayload(payload map[string]any, detail *model.IdeaDetail) map[string]any {
	out := mapValue(payload)
	if detail == nil {
		return out
	}
	out["idea_id"] = detail.Idea.ID
	out["title"] = detail.Idea.Title
	if detail.StructuredIdea != nil {
		out["description_md"] = detail.StructuredIdea.DescriptionMD
		out["research_direction"] = detail.StructuredIdea.ResearchDirection
		out["target_dataset_refs"] = detail.StructuredIdea.TargetDatasetRefs
		out["dataset_eval_protocol_refs"] = detail.StructuredIdea.DatasetEvalProtocolRefs
		out["innovation_type"] = detail.StructuredIdea.InnovationType
		out["expected_advantage"] = detail.StructuredIdea.ExpectedAdvantage
		out["risk_points"] = detail.StructuredIdea.RiskPoints
		out["priority"] = detail.StructuredIdea.Priority
		out["confidence"] = detail.StructuredIdea.Confidence
	}
	data := mapValue(out["data"])
	data["idea_id"] = detail.Idea.ID
	out["data"] = data
	return out
}

func parsePaperSources(value any) []model.IdeaSource {
	if value == nil {
		return []model.IdeaSource{}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return []model.IdeaSource{}
	}
	var items []model.IdeaSource
	if err = json.Unmarshal(raw, &items); err != nil {
		return []model.IdeaSource{}
	}
	return items
}

func buildPaperSourcesFromInputRefs(refs []model.AgentInputRef) []model.IdeaSource {
	paperID := ""
	paperTitle := ""
	out := make([]model.IdeaSource, 0)
	for _, ref := range refs {
		if ref.RefType == "paper" && strings.TrimSpace(ref.RefID) != "" {
			paperID = strings.TrimSpace(ref.RefID)
			paperTitle = stringValue(ref.Metadata["paper_title"])
		}
		if ref.RefType == "insight" && strings.TrimSpace(ref.RefID) != "" {
			out = append(out, model.IdeaSource{
				PaperID:        paperID,
				PaperInsightID: strings.TrimSpace(ref.RefID),
				PaperTitle:     paperTitle,
				SourceNote:     "idea generator insight event source",
			})
		}
	}
	return out
}

func manualIdeaMap(value *model.StructuredIdeaPayload) map[string]any {
	if value == nil {
		return nil
	}
	return map[string]any{
		"title":                      value.Title,
		"description_md":             value.DescriptionMD,
		"research_direction":         value.ResearchDirection,
		"target_dataset_refs":        value.TargetDatasetRefs,
		"dataset_eval_protocol_refs": value.DatasetEvalProtocolRefs,
		"innovation_type":            value.InnovationType,
		"expected_advantage":         value.ExpectedAdvantage,
		"risk_points":                value.RiskPoints,
		"priority":                   value.Priority,
		"confidence":                 value.Confidence,
	}
}

func hasManualIdeaMetadata(value any) bool {
	typed, ok := value.(map[string]any)
	if !ok || len(typed) == 0 {
		return false
	}
	return true
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := stringValue(item)
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := stringValue(value)
		if text == "" {
			return []string{}
		}
		return []string{text}
	}
}

func normalizeStringSlice(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func appendIfMissing(items []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return items
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), value) {
			return items
		}
	}
	return append(items, value)
}

func mapValue(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok || typed == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(typed))
	for key, item := range typed {
		out[key] = item
	}
	return out
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		text := strings.TrimSpace(fmt.Sprint(typed))
		if text == "<nil>" {
			return ""
		}
		return text
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func toStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := stringValue(item)
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := stringValue(value)
		if text == "" {
			return []string{}
		}
		return []string{text}
	}
}
