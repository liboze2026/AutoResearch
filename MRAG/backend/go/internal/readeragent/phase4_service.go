package readeragent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
)

const phase4ReaderOutputSchemaRef = "schemas/reader-phase4-output-v1.json"

type phase4DataService interface {
	GetDatasetProfileByID(context.Context, string) (*model.Phase4DatasetProfile, error)
	ListReaderSources(context.Context, string) ([]model.Phase4ReaderSource, error)
	GetReaderSourceByID(context.Context, string) (*model.Phase4ReaderSource, error)
	CreateReaderSource(context.Context, model.Phase4ReaderSourceCreateRequest) (*model.Phase4ReaderSource, error)
	UpdateReaderSource(context.Context, string, model.Phase4ReaderSourceUpdateRequest) (*model.Phase4ReaderSource, error)
	CreateReaderContext(context.Context, model.Phase4ReaderContextCreateRequest) (*model.Phase4ReaderContext, error)
	GetReaderContextByID(context.Context, string) (*model.Phase4ReaderContext, error)
}

type Phase4Service struct {
	jobs          jobCreator
	jobUpdates    jobUpdater
	triggers      triggerService
	artifacts     artifactLister
	phase4        phase4DataService
	workspaceRoot string
}

func NewPhase4Service(jobs jobCreator, jobUpdates jobUpdater, triggers triggerService, artifacts artifactLister, phase4 phase4DataService, workspaceRoot string) *Phase4Service {
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

func (s *Phase4Service) Run(ctx context.Context, req model.Phase4ReaderRunRequest) (*model.Phase4ReaderRunResult, error) {
	normalizedReq, datasetProfile, err := s.normalizeRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:       "reader_phase4",
		ExecutionMode:   normalizedReq.ExecutionMode,
		ModelProvider:   normalizedReq.ModelProvider,
		ModelName:       normalizedReq.ModelName,
		PromptVersion:   normalizedReq.PromptVersion,
		InputRefs:       buildPhase4ReaderInputRefs(*datasetProfile, normalizedReq.ManualPapers),
		OutputSchemaRef: phase4ReaderOutputSchemaRef,
		SkillRefs:       normalizedReq.SkillRefs,
		ToolRefs:        normalizedReq.ToolRefs,
		MemoryRefs:      normalizedReq.MemoryRefs,
		Metadata: map[string]any{
			"dataset_profile_id": datasetProfile.ID,
			"dataset_profile":    phase4DatasetProfileMetadata(*datasetProfile),
			"manual_papers":      normalizedReq.ManualPapers,
			"user_notes":         normalizedReq.UserNotes,
			"search_mode":        normalizedReq.SearchMode,
			"max_papers":         normalizedReq.MaxPapers,
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}
	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata: map[string]any{
			"agent_type": "reader_phase4",
		},
	})
	if err != nil {
		return nil, err
	}
	return s.resultFromJob(ctx, job)
}

func (s *Phase4Service) GetJob(ctx context.Context, jobID string) (*model.Phase4ReaderJobDetail, error) {
	job, err := s.jobs.GetByID(ctx, jobID)
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
	return &model.Phase4ReaderJobDetail{
		Job:           job,
		Artifacts:     artifacts,
		ReaderContext: result.ReaderContext,
		ReaderSources: result.ReaderSources,
		Warnings:      result.Warnings,
	}, nil
}

func (s *Phase4Service) PostProcess(ctx context.Context, job *model.AgentJob) error {
	if job == nil {
		return nil
	}
	payload, err := decodePhase4ReaderRuntimePayload(job.NormalizedPayload)
	if err != nil {
		return err
	}
	if len(payload.Sources) == 0 {
		return fmt.Errorf("phase4 reader returned no sources")
	}
	datasetProfileID := strings.TrimSpace(stringValue(job.Metadata["dataset_profile_id"]))
	if datasetProfileID == "" {
		return fmt.Errorf("phase4 reader dataset profile id is required")
	}
	datasetProfile, err := s.phase4.GetDatasetProfileByID(ctx, datasetProfileID)
	if err != nil {
		return err
	}
	if datasetProfile == nil {
		return fmt.Errorf("phase4 dataset profile not found")
	}
	existingSources, err := s.phase4.ListReaderSources(ctx, datasetProfileID)
	if err != nil {
		return err
	}
	sourceIDs := make([]string, 0, len(payload.Sources))
	for _, source := range payload.Sources {
		item, persistErr := s.persistReaderSource(ctx, datasetProfileID, existingSources, source)
		if persistErr != nil {
			return persistErr
		}
		existingSources = appendOrReplaceReaderSource(existingSources, *item)
		sourceIDs = append(sourceIDs, item.ID)
	}
	context, err := s.phase4.CreateReaderContext(ctx, model.Phase4ReaderContextCreateRequest{
		DatasetProfileID:  datasetProfileID,
		Title:             phase4ReaderContextTitle(*datasetProfile),
		Summary:           firstNonEmpty(strings.TrimSpace(payload.ReadingSummary), strings.TrimSpace(payload.Summary)),
		TaskDefinition:    strings.TrimSpace(payload.ReaderContext.TaskDefinition),
		RelatedWork:       phase4NormalizeStrings(payload.ReaderContext.RelevantMethodsLandscape),
		RetrievalFocus:    phase4ContextRetrievalFocus(payload.ReaderContext),
		RankingNotes:      phase4RankingNotes(payload.Metadata),
		SourceIDs:         phase4NormalizeStrings(sourceIDs),
		StructuredContext: phase4ReaderStructuredContext(payload),
		Status:            model.Phase4ReaderContextStatusReady,
	})
	if err != nil {
		return err
	}
	job.NormalizedPayload = phase4UpdateJobPayload(job.NormalizedPayload, context.ID, sourceIDs)
	job.UpdatedAt = time.Now()
	return s.jobUpdates.Update(ctx, *job)
}

func (s *Phase4Service) normalizeRequest(ctx context.Context, req model.Phase4ReaderRunRequest) (model.Phase4ReaderRunRequest, *model.Phase4DatasetProfile, error) {
	req.DatasetProfileID = strings.TrimSpace(req.DatasetProfileID)
	req.UserNotes = strings.TrimSpace(req.UserNotes)
	req.SearchMode = strings.TrimSpace(strings.ToLower(req.SearchMode))
	req.ExecutionMode = strings.TrimSpace(strings.ToLower(req.ExecutionMode))
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	if req.DatasetProfileID == "" {
		return req, nil, fmt.Errorf("datasetProfileId is required")
	}
	datasetProfile, err := s.phase4.GetDatasetProfileByID(ctx, req.DatasetProfileID)
	if err != nil {
		return req, nil, err
	}
	if datasetProfile == nil {
		return req, nil, fmt.Errorf("phase4 dataset profile not found")
	}
	switch req.SearchMode {
	case "", "auto":
		req.SearchMode = "auto"
	case "fixture", "live":
	default:
		return req, nil, fmt.Errorf("searchMode must be one of auto, fixture, live")
	}
	switch req.ExecutionMode {
	case "":
		req.ExecutionMode = "api"
	case "mock", "api", "codex_cli":
	default:
		return req, nil, fmt.Errorf("executionMode must be one of mock, api, codex_cli")
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "phase4_reader"
	}
	if req.ModelName == "" {
		req.ModelName = "reader-phase4-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	if req.MaxPapers <= 0 {
		req.MaxPapers = 10
	}
	if req.MaxPapers > 20 {
		req.MaxPapers = 20
	}
	req.ManualPapers = phase4NormalizeManualPapers(req.ManualPapers)
	return req, datasetProfile, nil
}

func (s *Phase4Service) resultFromJob(ctx context.Context, job *model.AgentJob) (*model.Phase4ReaderRunResult, error) {
	if job == nil {
		return nil, fmt.Errorf("phase4 reader job not found")
	}
	readerSources, readerContext, err := s.resolvePersistedReaderObjects(ctx, job.NormalizedPayload)
	if err != nil {
		return nil, err
	}
	return &model.Phase4ReaderRunResult{
		Job:           job,
		ReaderContext: readerContext,
		ReaderSources: readerSources,
		Warnings:      append([]string{}, job.Warnings...),
	}, nil
}

func (s *Phase4Service) resolvePersistedReaderObjects(ctx context.Context, payload map[string]any) ([]model.Phase4ReaderSource, *model.Phase4ReaderContext, error) {
	sourceIDs := phase4NormalizeStrings(stringSliceValue(payload["source_ids"]))
	readerSources := make([]model.Phase4ReaderSource, 0, len(sourceIDs))
	for _, sourceID := range sourceIDs {
		item, err := s.phase4.GetReaderSourceByID(ctx, sourceID)
		if err != nil {
			return nil, nil, err
		}
		if item != nil {
			readerSources = append(readerSources, *item)
		}
	}
	contextID := strings.TrimSpace(stringValue(payload["reader_context_id"]))
	if contextID == "" {
		return readerSources, nil, nil
	}
	item, err := s.phase4.GetReaderContextByID(ctx, contextID)
	if err != nil {
		return nil, nil, err
	}
	return readerSources, item, nil
}

func (s *Phase4Service) persistReaderSource(ctx context.Context, datasetProfileID string, existing []model.Phase4ReaderSource, source model.Phase4ReaderSourcePayload) (*model.Phase4ReaderSource, error) {
	match := findMatchingReaderSource(existing, source)
	if match == nil {
		return s.phase4.CreateReaderSource(ctx, model.Phase4ReaderSourceCreateRequest{
			DatasetProfileID: datasetProfileID,
			Title:            strings.TrimSpace(source.Title),
			Authors:          phase4NormalizeStrings(source.Authors),
			Venue:            strings.TrimSpace(source.Venue),
			PublicationYear:  source.PublicationYear,
			SourceType:       strings.TrimSpace(strings.ToLower(source.SourceType)),
			SourceURL:        strings.TrimSpace(source.SourceURL),
			OpenAccessURL:    strings.TrimSpace(source.OpenAccessURL),
			QualityTier:      strings.TrimSpace(source.QualityTier),
			RankingScore:     source.RankingScore,
			QualityScore:     source.QualityScore,
			RelevanceScore:   source.RelevanceScore,
			CitationCount:    source.CitationCount,
			Metadata:         phase4SourceMetadata(source),
		})
	}
	return s.phase4.UpdateReaderSource(ctx, match.ID, model.Phase4ReaderSourceUpdateRequest{
		Title:           stringPtr(firstNonEmpty(strings.TrimSpace(source.Title), match.Title)),
		Authors:         stringSlicePtr(phase4NormalizeStrings(append(match.Authors, source.Authors...))),
		Venue:           stringPtr(firstNonEmpty(strings.TrimSpace(source.Venue), match.Venue)),
		PublicationYear: intPtr(phase4PreferYear(match.PublicationYear, source.PublicationYear)),
		SourceType:      stringPtr(firstNonEmpty(strings.TrimSpace(strings.ToLower(source.SourceType)), match.SourceType)),
		SourceURL:       stringPtr(firstNonEmpty(strings.TrimSpace(source.SourceURL), match.SourceURL)),
		OpenAccessURL:   stringPtr(firstNonEmpty(strings.TrimSpace(source.OpenAccessURL), match.OpenAccessURL)),
		QualityTier:     stringPtr(phase4PreferQualityTier(match.QualityTier, source.QualityTier)),
		RankingScore:    float64Ptr(maxFloat(match.RankingScore, source.RankingScore)),
		QualityScore:    float64Ptr(maxFloat(match.QualityScore, source.QualityScore)),
		RelevanceScore:  float64Ptr(maxFloat(match.RelevanceScore, source.RelevanceScore)),
		CitationCount:   intPtr(maxInt(match.CitationCount, source.CitationCount)),
		Metadata:        mapPtr(phase4MergeMaps(match.Metadata, phase4SourceMetadata(source))),
	})
}

func buildPhase4ReaderInputRefs(datasetProfile model.Phase4DatasetProfile, items []model.Phase4ReaderManualPaperInput) []model.AgentInputRef {
	refs := []model.AgentInputRef{
		{
			RefType: "dataset_profile",
			RefID:   datasetProfile.ID,
			RefPath: datasetProfile.ServerPath,
			Metadata: map[string]any{
				"dataset_name": datasetProfile.DatasetName,
				"task_type":    datasetProfile.TaskType,
				"server_id":    datasetProfile.ServerID,
			},
		},
	}
	for _, item := range items {
		if strings.TrimSpace(item.FilePath) == "" {
			continue
		}
		refs = append(refs, model.AgentInputRef{
			RefType: "paper_upload",
			RefPath: strings.TrimSpace(item.FilePath),
			Metadata: map[string]any{
				"title":           strings.TrimSpace(item.Title),
				"abstract":        strings.TrimSpace(item.Abstract),
				"source_type":     strings.TrimSpace(item.SourceType),
				"source_url":      strings.TrimSpace(item.SourceURL),
				"open_access_url": strings.TrimSpace(item.OpenAccessURL),
				"venue":           strings.TrimSpace(item.Venue),
				"year":            item.Year,
			},
		})
	}
	return refs
}

func phase4DatasetProfileMetadata(item model.Phase4DatasetProfile) map[string]any {
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
		"metadata":            phase4CloneMap(item.Metadata),
	}
}

func phase4NormalizeManualPapers(items []model.Phase4ReaderManualPaperInput) []model.Phase4ReaderManualPaperInput {
	out := make([]model.Phase4ReaderManualPaperInput, 0, len(items))
	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		if title == "" && strings.TrimSpace(item.FilePath) == "" {
			continue
		}
		if item.Year < 0 {
			item.Year = 0
		}
		out = append(out, model.Phase4ReaderManualPaperInput{
			Title:         title,
			Abstract:      strings.TrimSpace(item.Abstract),
			SourceType:    strings.TrimSpace(strings.ToLower(item.SourceType)),
			SourceURL:     strings.TrimSpace(item.SourceURL),
			OpenAccessURL: strings.TrimSpace(item.OpenAccessURL),
			Venue:         strings.TrimSpace(item.Venue),
			Year:          item.Year,
			Authors:       phase4NormalizeStrings(item.Authors),
			FilePath:      strings.TrimSpace(item.FilePath),
			Note:          strings.TrimSpace(item.Note),
		})
	}
	return out
}

func decodePhase4ReaderRuntimePayload(payload map[string]any) (model.Phase4ReaderRuntimePayload, error) {
	raw, err := json.Marshal(ensureMap(payload))
	if err != nil {
		return model.Phase4ReaderRuntimePayload{}, err
	}
	var out model.Phase4ReaderRuntimePayload
	if err = json.Unmarshal(raw, &out); err != nil {
		return model.Phase4ReaderRuntimePayload{}, err
	}
	if out.Data == nil {
		out.Data = map[string]any{}
	}
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	return out, nil
}

func appendOrReplaceReaderSource(items []model.Phase4ReaderSource, item model.Phase4ReaderSource) []model.Phase4ReaderSource {
	for index, existing := range items {
		if strings.EqualFold(existing.ID, item.ID) {
			items[index] = item
			return items
		}
	}
	return append(items, item)
}

func findMatchingReaderSource(items []model.Phase4ReaderSource, source model.Phase4ReaderSourcePayload) *model.Phase4ReaderSource {
	keys := phase4ReaderSourceMatchKeys(source.Title, source.SourceURL, source.Metadata)
	for _, item := range items {
		existingKeys := phase4ReaderSourceMatchKeys(item.Title, item.SourceURL, item.Metadata)
		for _, key := range keys {
			for _, existingKey := range existingKeys {
				if key != "" && strings.EqualFold(key, existingKey) {
					copyItem := item
					return &copyItem
				}
			}
		}
	}
	return nil
}

func phase4ReaderSourceMatchKeys(title string, sourceURL string, metadata map[string]any) []string {
	keys := make([]string, 0, 4)
	doi := strings.TrimSpace(strings.ToLower(stringValue(metadata["doi"])))
	if doi != "" {
		keys = append(keys, "doi:"+doi)
	}
	externalIDs := mapValue(metadata["external_ids"])
	arxivID := strings.TrimSpace(strings.ToLower(stringValue(externalIDs["arxiv"])))
	if arxivID != "" {
		keys = append(keys, "arxiv:"+arxivID)
	}
	normalizedTitle := phase4NormalizeTitle(title)
	if normalizedTitle != "" {
		keys = append(keys, "title:"+normalizedTitle)
	}
	sourceURL = strings.TrimSpace(strings.ToLower(sourceURL))
	if sourceURL != "" {
		keys = append(keys, "url:"+sourceURL)
	}
	return phase4NormalizeStrings(keys)
}

func phase4NormalizeTitle(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("/", " ", "\\", " ", "-", " ", "_", " ", ":", " ", ",", " ", ".", " ", "(", " ", ")", " ")
	value = replacer.Replace(value)
	return strings.Join(strings.Fields(value), " ")
}

func phase4ReaderContextTitle(datasetProfile model.Phase4DatasetProfile) string {
	return firstNonEmpty(strings.TrimSpace(datasetProfile.DatasetName)+" Reader Context", "Phase4 Reader Context")
}

func phase4ContextRetrievalFocus(payload model.Phase4ReaderContextPayload) []string {
	return phase4NormalizeStrings(append(payload.LikelyStrongBaselines, payload.PromisingResearchDirections...))
}

func phase4RankingNotes(metadata map[string]any) string {
	statuses := make([]string, 0)
	for _, item := range sliceMapValue(metadata["provider_statuses"]) {
		provider := firstNonEmpty(stringValue(item["provider"]), "provider")
		status := firstNonEmpty(stringValue(item["status"]), "unknown")
		count := maxInt(0, intValue(item["result_count"]))
		statuses = append(statuses, fmt.Sprintf("%s:%s:%d", provider, status, count))
	}
	if len(statuses) == 0 {
		return "Sources are ranked by quality tier first, then relevance and citation strength."
	}
	return "Sources are ranked by quality tier first, then relevance and citation strength. Provider status: " + strings.Join(statuses, ", ")
}

func phase4ReaderStructuredContext(payload model.Phase4ReaderRuntimePayload) map[string]any {
	out := phase4CloneMap(payload.Data)
	out["reading_summary"] = firstNonEmpty(strings.TrimSpace(payload.ReadingSummary), strings.TrimSpace(payload.Summary))
	out["task_definition"] = strings.TrimSpace(payload.ReaderContext.TaskDefinition)
	out["dataset_specific_challenges"] = phase4NormalizeStrings(payload.ReaderContext.DatasetSpecificChallenges)
	out["relevant_methods_landscape"] = phase4NormalizeStrings(payload.ReaderContext.RelevantMethodsLandscape)
	out["likely_strong_baselines"] = phase4NormalizeStrings(payload.ReaderContext.LikelyStrongBaselines)
	out["common_failure_points"] = phase4NormalizeStrings(payload.ReaderContext.CommonFailurePoints)
	out["evaluation_caveats"] = phase4NormalizeStrings(payload.ReaderContext.EvaluationCaveats)
	out["implementation_constraints"] = phase4NormalizeStrings(payload.ReaderContext.ImplementationConstraints)
	out["promising_research_directions"] = phase4NormalizeStrings(payload.ReaderContext.PromisingResearchDirections)
	out["citation_metadata"] = append([]map[string]any{}, payload.ReaderContext.CitationMetadata...)
	out["sources"] = append([]model.Phase4ReaderSourcePayload{}, payload.Sources...)
	return out
}

func phase4SourceMetadata(source model.Phase4ReaderSourcePayload) map[string]any {
	metadata := phase4CloneMap(source.Metadata)
	if strings.TrimSpace(source.Abstract) != "" {
		metadata["abstract"] = strings.TrimSpace(source.Abstract)
	}
	return metadata
}

func phase4UpdateJobPayload(payload map[string]any, readerContextID string, sourceIDs []string) map[string]any {
	out := ensureMap(payload)
	out["reader_context_id"] = strings.TrimSpace(readerContextID)
	out["source_ids"] = phase4NormalizeStrings(sourceIDs)
	data := ensureMap(out["data"])
	data["reader_context_id"] = strings.TrimSpace(readerContextID)
	data["source_ids"] = phase4NormalizeStrings(sourceIDs)
	out["data"] = data
	return out
}

func phase4NormalizeStrings(items []string) []string {
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

func phase4CloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func phase4MergeMaps(current map[string]any, incoming map[string]any) map[string]any {
	out := phase4CloneMap(current)
	for key, value := range phase4CloneMap(incoming) {
		switch key {
		case "provider_refs", "matched_queries", "keywords":
			out[key] = phase4NormalizeStrings(append(stringSliceValue(out[key]), stringSliceValue(value)...))
		case "external_ids":
			merged := mapValue(out[key])
			for nestedKey, nestedValue := range mapValue(value) {
				if strings.TrimSpace(stringValue(nestedValue)) != "" {
					merged[nestedKey] = strings.TrimSpace(stringValue(nestedValue))
				}
			}
			out[key] = merged
		default:
			if existing, ok := out[key]; !ok || existing == nil || strings.TrimSpace(stringValue(existing)) == "" {
				out[key] = value
			}
		}
	}
	return out
}

func phase4PreferQualityTier(current string, incoming string) string {
	if phase4QualityRank(incoming) > phase4QualityRank(current) {
		return strings.TrimSpace(incoming)
	}
	return strings.TrimSpace(current)
}

func phase4QualityRank(value string) int {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "top_venue":
		return 4
	case "peer_reviewed":
		return 3
	case "manual":
		return 2
	case "arxiv":
		return 1
	default:
		return 0
	}
}

func phase4PreferYear(current int, incoming int) int {
	if incoming > current {
		return incoming
	}
	return current
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func mapPtr(value map[string]any) *map[string]any {
	return &value
}

func stringSlicePtr(value []string) *[]string {
	return &value
}

func maxInt(current int, incoming int) int {
	if incoming > current {
		return incoming
	}
	return current
}

func maxFloat(current float64, incoming float64) float64 {
	if incoming > current {
		return incoming
	}
	return current
}

func stringValue(value any) string {
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

func mapValue(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok || typed == nil {
		return map[string]any{}
	}
	return phase4CloneMap(typed)
}

func sliceMapValue(value any) []map[string]any {
	typed, ok := value.([]any)
	if !ok {
		if typedMaps, okMaps := value.([]map[string]any); okMaps {
			return append([]map[string]any{}, typedMaps...)
		}
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(typed))
	for _, item := range typed {
		out = append(out, mapValue(item))
	}
	return out
}
