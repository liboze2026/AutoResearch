package ideaagent

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
)

const phase4IdeaOutputSchemaRef = "schemas/idea-phase4-output-v1.json"

type phase4IdeaArtifactLister interface {
	ListByJobID(context.Context, string) ([]model.AgentArtifact, error)
}

type phase4IdeaDataService interface {
	GetDatasetProfileByID(context.Context, string) (*model.Phase4DatasetProfile, error)
	GetReaderContextByID(context.Context, string) (*model.Phase4ReaderContext, error)
	GetIdeaByID(context.Context, string) (*model.Phase4Idea, error)
	CreateIdea(context.Context, model.Phase4IdeaCreateRequest) (*model.Phase4Idea, error)
}

type Phase4Service struct {
	jobs          jobCreator
	jobUpdates    jobUpdater
	triggers      triggerService
	artifacts     phase4IdeaArtifactLister
	phase4        phase4IdeaDataService
	workspaceRoot string
}

func NewPhase4Service(jobs jobCreator, jobUpdates jobUpdater, triggers triggerService, artifacts phase4IdeaArtifactLister, phase4 phase4IdeaDataService, workspaceRoot string) *Phase4Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Phase4Service{
		jobs:          jobs,
		jobUpdates:    jobUpdates,
		triggers:      triggers,
		artifacts:     artifacts,
		phase4:        phase4,
		workspaceRoot: workspaceRoot,
	}
}

func (s *Phase4Service) Run(ctx context.Context, req model.Phase4IdeaRunRequest) (*model.Phase4IdeaRunResult, error) {
	req, datasetProfile, readerContext, err := s.normalizeRunRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:       "idea_phase4",
		ExecutionMode:   req.ExecutionMode,
		ModelProvider:   req.ModelProvider,
		ModelName:       req.ModelName,
		PromptVersion:   req.PromptVersion,
		InputRefs:       buildPhase4IdeaInputRefs(*datasetProfile, *readerContext, nil),
		OutputSchemaRef: phase4IdeaOutputSchemaRef,
		SkillRefs:       req.SkillRefs,
		ToolRefs:        req.ToolRefs,
		MemoryRefs:      req.MemoryRefs,
		Metadata: map[string]any{
			"dataset_profile_id": datasetProfile.ID,
			"reader_context_id":  readerContext.ID,
			"dataset_profile":    phase4IdeaDatasetProfileMetadata(*datasetProfile),
			"reader_context":     phase4IdeaReaderContextMetadata(*readerContext),
			"user_notes":         req.UserNotes,
			"manual_idea":        phase4IdeaSeedMetadata(req.ManualIdea),
			"generation_mode":    "new",
			"target_count":       req.TargetCount,
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}
	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata: map[string]any{
			"agent_type": "idea_phase4",
		},
	})
	if err != nil {
		return nil, err
	}
	return s.resultFromJob(ctx, job)
}

func (s *Phase4Service) GenerateRevisionCandidates(ctx context.Context, sourceIdeaID string, req model.Phase4IdeaRevisionGenerateRequest) (*model.Phase4IdeaRunResult, error) {
	sourceIdea, datasetProfile, readerContext, req, err := s.normalizeRevisionRequest(ctx, sourceIdeaID, req)
	if err != nil {
		return nil, err
	}
	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:       "idea_phase4",
		ExecutionMode:   req.ExecutionMode,
		ModelProvider:   req.ModelProvider,
		ModelName:       req.ModelName,
		PromptVersion:   req.PromptVersion,
		InputRefs:       buildPhase4IdeaInputRefs(*datasetProfile, *readerContext, sourceIdea),
		OutputSchemaRef: phase4IdeaOutputSchemaRef,
		SkillRefs:       req.SkillRefs,
		ToolRefs:        req.ToolRefs,
		MemoryRefs:      req.MemoryRefs,
		Metadata: map[string]any{
			"dataset_profile_id":  datasetProfile.ID,
			"reader_context_id":   readerContext.ID,
			"dataset_profile":     phase4IdeaDatasetProfileMetadata(*datasetProfile),
			"reader_context":      phase4IdeaReaderContextMetadata(*readerContext),
			"user_notes":          req.UserNotes,
			"generation_mode":     "revision",
			"target_count":        req.TargetCount,
			"source_idea_id":      sourceIdea.ID,
			"source_idea":         phase4IdeaSeedFromStored(*sourceIdea),
			"failure_feedback":    phase4IdeaCloneMap(req.FailureFeedback),
			"last_failure_run_id": strings.TrimSpace(req.LastFailureRunID),
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}
	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata: map[string]any{
			"agent_type": "idea_phase4",
			"mode":       "revision",
		},
	})
	if err != nil {
		return nil, err
	}
	return s.resultFromJob(ctx, job)
}

func (s *Phase4Service) GetJob(ctx context.Context, jobID string) (*model.Phase4IdeaJobDetail, error) {
	job, err := s.jobs.GetByID(ctx, strings.TrimSpace(jobID))
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, nil
	}
	artifacts, err := s.artifacts.ListByJobID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	result, err := s.resultFromJob(ctx, job)
	if err != nil {
		return nil, err
	}
	return &model.Phase4IdeaJobDetail{
		Job:                job,
		Artifacts:          artifacts,
		Ideas:              result.Ideas,
		TopRecommendations: result.TopRecommendations,
		Warnings:           result.Warnings,
	}, nil
}

func (s *Phase4Service) PostProcess(ctx context.Context, job *model.AgentJob) error {
	if job == nil {
		return nil
	}
	payload, err := decodePhase4IdeaRuntimePayload(job.NormalizedPayload)
	if err != nil {
		return err
	}
	if len(payload.Ideas) == 0 {
		return fmt.Errorf("phase4 idea returned no ideas")
	}
	datasetProfileID := phase4IdeaStringValue(job.Metadata["dataset_profile_id"])
	readerContextID := phase4IdeaStringValue(job.Metadata["reader_context_id"])
	if datasetProfileID == "" {
		return fmt.Errorf("phase4 idea dataset profile id is required")
	}
	if readerContextID == "" {
		return fmt.Errorf("phase4 idea reader context id is required")
	}
	createdIDs := make([]string, 0, len(payload.Ideas))
	titleToID := map[string]string{}
	for _, candidate := range payload.Ideas {
		item, createErr := s.phase4.CreateIdea(ctx, model.Phase4IdeaCreateRequest{
			DatasetProfileID:    datasetProfileID,
			ReaderContextID:     readerContextID,
			Title:               strings.TrimSpace(candidate.Title),
			ProblemDefinition:   strings.TrimSpace(candidate.ProblemDefinition),
			CoreMethod:          strings.TrimSpace(candidate.CoreMethod),
			Differentiators:     strings.TrimSpace(candidate.Differentiators),
			DataProcessingNeeds: phase4IdeaNormalizeStrings(candidate.DataProcessingNeeds),
			ModelChanges:        phase4IdeaNormalizeStrings(candidate.ModelChanges),
			TrainingPlan:        strings.TrimSpace(candidate.TrainingPlan),
			EvaluationMetrics:   phase4IdeaNormalizeStrings(candidate.EvaluationMetrics),
			RiskPoints:          phase4IdeaNormalizeStrings(candidate.RiskPoints),
			ExpectedGains:       phase4IdeaNormalizeStrings(candidate.ExpectedGains),
			Score:               candidate.Score,
			ScoreSummary:        phase4IdeaCloneMap(candidate.ScoreSummary),
			Status:              strings.TrimSpace(candidate.Status),
			SourceType:          strings.TrimSpace(candidate.SourceType),
			RevisionOfID:        strings.TrimSpace(candidate.RevisionOfID),
			FailureFeedback:     phase4IdeaCloneMap(candidate.FailureFeedback),
			LastFailureRunID:    strings.TrimSpace(candidate.LastFailureRunID),
		})
		if createErr != nil {
			return createErr
		}
		createdIDs = append(createdIDs, item.ID)
		titleToID[strings.ToLower(strings.TrimSpace(item.Title))] = item.ID
	}
	topIDs := make([]string, 0, len(payload.TopRecommendations))
	for _, item := range payload.TopRecommendations {
		if matchedID := titleToID[strings.ToLower(strings.TrimSpace(item.Title))]; matchedID != "" {
			topIDs = append(topIDs, matchedID)
		}
	}
	job.NormalizedPayload = phase4IdeaUpdateJobPayload(job.NormalizedPayload, createdIDs, topIDs)
	job.UpdatedAt = time.Now()
	return s.jobUpdates.Update(ctx, *job)
}

func (s *Phase4Service) normalizeRunRequest(ctx context.Context, req model.Phase4IdeaRunRequest) (model.Phase4IdeaRunRequest, *model.Phase4DatasetProfile, *model.Phase4ReaderContext, error) {
	req.DatasetProfileID = strings.TrimSpace(req.DatasetProfileID)
	req.ReaderContextID = strings.TrimSpace(req.ReaderContextID)
	req.UserNotes = strings.TrimSpace(req.UserNotes)
	req.ExecutionMode = strings.TrimSpace(strings.ToLower(req.ExecutionMode))
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	if req.DatasetProfileID == "" {
		return req, nil, nil, fmt.Errorf("datasetProfileId is required")
	}
	if req.ReaderContextID == "" {
		return req, nil, nil, fmt.Errorf("readerContextId is required")
	}
	datasetProfile, err := s.phase4.GetDatasetProfileByID(ctx, req.DatasetProfileID)
	if err != nil {
		return req, nil, nil, err
	}
	if datasetProfile == nil {
		return req, nil, nil, fmt.Errorf("phase4 dataset profile not found")
	}
	readerContext, err := s.phase4.GetReaderContextByID(ctx, req.ReaderContextID)
	if err != nil {
		return req, nil, nil, err
	}
	if readerContext == nil {
		return req, nil, nil, fmt.Errorf("phase4 reader context not found")
	}
	if req.TargetCount <= 0 || req.TargetCount > 10 {
		req.TargetCount = 10
	}
	switch req.ExecutionMode {
	case "":
		req.ExecutionMode = "api"
	case "mock", "api", "codex_cli":
	default:
		return req, nil, nil, fmt.Errorf("executionMode must be one of mock, api, codex_cli")
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "phase4_idea"
	}
	if req.ModelName == "" {
		req.ModelName = "idea-phase4-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	return req, datasetProfile, readerContext, nil
}

func (s *Phase4Service) normalizeRevisionRequest(ctx context.Context, sourceIdeaID string, req model.Phase4IdeaRevisionGenerateRequest) (*model.Phase4Idea, *model.Phase4DatasetProfile, *model.Phase4ReaderContext, model.Phase4IdeaRevisionGenerateRequest, error) {
	sourceIdeaID = strings.TrimSpace(sourceIdeaID)
	req.UserNotes = strings.TrimSpace(req.UserNotes)
	req.LastFailureRunID = strings.TrimSpace(req.LastFailureRunID)
	req.ExecutionMode = strings.TrimSpace(strings.ToLower(req.ExecutionMode))
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	if sourceIdeaID == "" {
		return nil, nil, nil, req, fmt.Errorf("idea id is required")
	}
	sourceIdea, err := s.phase4.GetIdeaByID(ctx, sourceIdeaID)
	if err != nil {
		return nil, nil, nil, req, err
	}
	if sourceIdea == nil {
		return nil, nil, nil, req, fmt.Errorf("phase4 idea not found")
	}
	datasetProfile, err := s.phase4.GetDatasetProfileByID(ctx, sourceIdea.DatasetProfileID)
	if err != nil {
		return nil, nil, nil, req, err
	}
	if datasetProfile == nil {
		return nil, nil, nil, req, fmt.Errorf("phase4 dataset profile not found")
	}
	readerContext, err := s.phase4.GetReaderContextByID(ctx, sourceIdea.ReaderContextID)
	if err != nil {
		return nil, nil, nil, req, err
	}
	if readerContext == nil {
		return nil, nil, nil, req, fmt.Errorf("phase4 reader context not found")
	}
	if len(req.FailureFeedback) == 0 && len(sourceIdea.FailureFeedback) > 0 {
		req.FailureFeedback = phase4IdeaCloneMap(sourceIdea.FailureFeedback)
	}
	if req.LastFailureRunID == "" {
		req.LastFailureRunID = strings.TrimSpace(sourceIdea.LastFailureRunID)
	}
	if req.TargetCount <= 0 || req.TargetCount > 5 {
		req.TargetCount = 3
	}
	switch req.ExecutionMode {
	case "":
		req.ExecutionMode = "api"
	case "mock", "api", "codex_cli":
	default:
		return nil, nil, nil, req, fmt.Errorf("executionMode must be one of mock, api, codex_cli")
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "phase4_idea"
	}
	if req.ModelName == "" {
		req.ModelName = "idea-phase4-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	return sourceIdea, datasetProfile, readerContext, req, nil
}

func (s *Phase4Service) resultFromJob(ctx context.Context, job *model.AgentJob) (*model.Phase4IdeaRunResult, error) {
	if job == nil {
		return nil, fmt.Errorf("phase4 idea job not found")
	}
	ideas, top, err := s.resolvePersistedIdeas(ctx, job.NormalizedPayload)
	if err != nil {
		return nil, err
	}
	return &model.Phase4IdeaRunResult{
		Job:                job,
		Ideas:              ideas,
		TopRecommendations: top,
		Warnings:           append([]string{}, job.Warnings...),
	}, nil
}

func (s *Phase4Service) resolvePersistedIdeas(ctx context.Context, payload map[string]any) ([]model.Phase4Idea, []model.Phase4IdeaScoreView, error) {
	ideaIDs := phase4IdeaNormalizeStrings(phase4IdeaStringSliceValue(payload["idea_ids"]))
	items := make([]model.Phase4Idea, 0, len(ideaIDs))
	scoreViewByID := map[string]model.Phase4IdeaScoreView{}
	for _, ideaID := range ideaIDs {
		item, err := s.phase4.GetIdeaByID(ctx, ideaID)
		if err != nil {
			return nil, nil, err
		}
		if item == nil {
			continue
		}
		items = append(items, *item)
		scoreViewByID[item.ID] = phase4IdeaScoreView(*item)
	}
	topIDs := phase4IdeaNormalizeStrings(phase4IdeaStringSliceValue(payload["top_idea_ids"]))
	top := make([]model.Phase4IdeaScoreView, 0, len(topIDs))
	for _, ideaID := range topIDs {
		if item, ok := scoreViewByID[ideaID]; ok {
			top = append(top, item)
		}
	}
	if len(top) == 0 && len(items) > 0 {
		top = phase4IdeaTopViews(items)
	}
	return items, top, nil
}

func buildPhase4IdeaInputRefs(datasetProfile model.Phase4DatasetProfile, readerContext model.Phase4ReaderContext, sourceIdea *model.Phase4Idea) []model.AgentInputRef {
	refs := []model.AgentInputRef{
		{
			RefType: "dataset_profile",
			RefID:   datasetProfile.ID,
			RefPath: datasetProfile.ServerPath,
			Metadata: map[string]any{
				"dataset_name": datasetProfile.DatasetName,
				"task_type":    datasetProfile.TaskType,
			},
		},
		{
			RefType: "reader_context",
			RefID:   readerContext.ID,
			Metadata: map[string]any{
				"title": readerContext.Title,
			},
		},
	}
	if sourceIdea != nil {
		refs = append(refs, model.AgentInputRef{
			RefType: "idea",
			RefID:   sourceIdea.ID,
			Metadata: map[string]any{
				"title": sourceIdea.Title,
			},
		})
	}
	return refs
}

func phase4IdeaDatasetProfileMetadata(item model.Phase4DatasetProfile) map[string]any {
	return map[string]any{
		"id":                  item.ID,
		"datasetName":         item.DatasetName,
		"taskType":            item.TaskType,
		"modalityComposition": append([]string{}, item.ModalityComposition...),
		"splits":              append([]model.Phase4DatasetSplit{}, item.Splits...),
		"officialMetric":      item.OfficialMetric,
		"officialBaseline":    item.OfficialBaseline,
		"knownDifficulties":   append([]string{}, item.KnownDifficulties...),
		"userNotes":           item.UserNotes,
		"metadata":            phase4IdeaCloneMap(item.Metadata),
	}
}

func phase4IdeaReaderContextMetadata(item model.Phase4ReaderContext) map[string]any {
	out := phase4IdeaCloneMap(item.StructuredContext)
	out["title"] = item.Title
	out["summary"] = item.Summary
	out["task_definition"] = firstNonEmpty(phase4IdeaStringValue(out["task_definition"]), item.TaskDefinition)
	out["related_work"] = phase4IdeaNormalizeStrings(item.RelatedWork)
	out["retrieval_focus"] = phase4IdeaNormalizeStrings(item.RetrievalFocus)
	out["ranking_notes"] = item.RankingNotes
	out["source_ids"] = phase4IdeaNormalizeStrings(item.SourceIDs)
	return out
}

func phase4IdeaSeedMetadata(seed *model.Phase4IdeaSeedInput) map[string]any {
	if seed == nil {
		return map[string]any{}
	}
	return map[string]any{
		"title":                 strings.TrimSpace(seed.Title),
		"problem_definition":    strings.TrimSpace(seed.ProblemDefinition),
		"core_method":           strings.TrimSpace(seed.CoreMethod),
		"differentiators":       strings.TrimSpace(seed.Differentiators),
		"data_processing_needs": phase4IdeaNormalizeStrings(seed.DataProcessingNeeds),
		"model_changes":         phase4IdeaNormalizeStrings(seed.ModelChanges),
		"training_plan":         strings.TrimSpace(seed.TrainingPlan),
		"evaluation_metrics":    phase4IdeaNormalizeStrings(seed.EvaluationMetrics),
		"risk_points":           phase4IdeaNormalizeStrings(seed.RiskPoints),
		"expected_gains":        phase4IdeaNormalizeStrings(seed.ExpectedGains),
		"source_type":           strings.TrimSpace(seed.SourceType),
		"revision_of_id":        strings.TrimSpace(seed.RevisionOfID),
	}
}

func phase4IdeaSeedFromStored(item model.Phase4Idea) map[string]any {
	return map[string]any{
		"title":                 item.Title,
		"problem_definition":    item.ProblemDefinition,
		"core_method":           item.CoreMethod,
		"differentiators":       item.Differentiators,
		"data_processing_needs": append([]string{}, item.DataProcessingNeeds...),
		"model_changes":         append([]string{}, item.ModelChanges...),
		"training_plan":         item.TrainingPlan,
		"evaluation_metrics":    append([]string{}, item.EvaluationMetrics...),
		"risk_points":           append([]string{}, item.RiskPoints...),
		"expected_gains":        append([]string{}, item.ExpectedGains...),
		"source_type":           item.SourceType,
		"revision_of_id":        item.ID,
	}
}

func decodePhase4IdeaRuntimePayload(payload map[string]any) (model.Phase4IdeaRuntimePayload, error) {
	raw, err := json.Marshal(phase4IdeaEnsureMap(payload))
	if err != nil {
		return model.Phase4IdeaRuntimePayload{}, err
	}
	var out model.Phase4IdeaRuntimePayload
	if err = json.Unmarshal(raw, &out); err != nil {
		return model.Phase4IdeaRuntimePayload{}, err
	}
	if out.Data == nil {
		out.Data = map[string]any{}
	}
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	return out, nil
}

func phase4IdeaUpdateJobPayload(payload map[string]any, ideaIDs []string, topIdeaIDs []string) map[string]any {
	out := phase4IdeaEnsureMap(payload)
	out["idea_ids"] = phase4IdeaNormalizeStrings(ideaIDs)
	out["top_idea_ids"] = phase4IdeaNormalizeStrings(topIdeaIDs)
	data := phase4IdeaEnsureMap(out["data"])
	data["idea_ids"] = phase4IdeaNormalizeStrings(ideaIDs)
	data["top_idea_ids"] = phase4IdeaNormalizeStrings(topIdeaIDs)
	out["data"] = data
	return out
}

func phase4IdeaTopViews(items []model.Phase4Idea) []model.Phase4IdeaScoreView {
	views := make([]model.Phase4IdeaScoreView, 0, len(items))
	for _, item := range items {
		views = append(views, phase4IdeaScoreView(item))
	}
	sort.SliceStable(views, func(i, j int) bool {
		if views[i].Rank > 0 && views[j].Rank > 0 && views[i].Rank != views[j].Rank {
			return views[i].Rank < views[j].Rank
		}
		return views[i].OverallScore > views[j].OverallScore
	})
	if len(views) > 3 {
		return views[:3]
	}
	return views
}

func phase4IdeaScoreView(item model.Phase4Idea) model.Phase4IdeaScoreView {
	return model.Phase4IdeaScoreView{
		ID:                   item.ID,
		DatasetProfileID:     item.DatasetProfileID,
		ReaderContextID:      item.ReaderContextID,
		Title:                item.Title,
		Status:               item.Status,
		SourceType:           item.SourceType,
		RevisionOfID:         item.RevisionOfID,
		LineageRootID:        item.LineageRootID,
		LastFailureRunID:     item.LastFailureRunID,
		Score:                item.Score,
		OverallScore:         phase4IdeaFloatValue(item.ScoreSummary["overallScore"]),
		Rank:                 phase4IdeaIntValue(item.ScoreSummary["rank"]),
		RecommendationTier:   phase4IdeaStringValue(item.ScoreSummary["recommendationTier"]),
		RecommendationReason: phase4IdeaStringValue(item.ScoreSummary["recommendationReason"]),
		ExpectedGains:        append([]string{}, item.ExpectedGains...),
		RiskPoints:           append([]string{}, item.RiskPoints...),
	}
}

func phase4IdeaStringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := phase4IdeaStringValue(item)
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := phase4IdeaStringValue(value)
		if text == "" {
			return []string{}
		}
		return []string{text}
	}
}

func phase4IdeaNormalizeStrings(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func phase4IdeaEnsureMap(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok || typed == nil {
		return map[string]any{}
	}
	return phase4IdeaCloneMap(typed)
}

func phase4IdeaCloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func phase4IdeaStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "<nil>" {
			return ""
		}
		return text
	}
}

func phase4IdeaIntValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		number, _ := typed.Int64()
		return int(number)
	case string:
		number, _ := strconv.Atoi(strings.TrimSpace(typed))
		return number
	default:
		return 0
	}
}

func phase4IdeaFloatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		number, _ := typed.Float64()
		return number
	case string:
		number, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return number
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
