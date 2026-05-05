package writeragent

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

const writerOutputSchemaRef = "schemas/writer-output-v1.json"

type jobCreator interface {
	Create(context.Context, model.AgentJobCreateRequest) (*model.AgentJob, error)
}

type jobUpdater interface {
	Update(context.Context, model.AgentJob) error
}

type triggerService interface {
	Trigger(context.Context, string, model.AgentJobTriggerRequest) (*model.AgentJob, error)
}

type ideaReader interface {
	GetByID(context.Context, string) (*model.IdeaDetail, error)
}

type runReader interface {
	GetRun(context.Context, string) (*model.ExperimentRun, error)
}

type comparisonReader interface {
	ListByExperimentID(context.Context, string) ([]model.ResultComparison, error)
}

type archiveUpdater interface {
	Update(context.Context, string, model.ResultArchiveUpdateRequest) (*model.ResultArchiveDetail, error)
}

type eventPublisher interface {
	PublishEvent(context.Context, model.AgentEventCreateRequest) (*model.AgentEvent, error)
}

type Service struct {
	jobs          jobCreator
	jobUpdates    jobUpdater
	triggers      triggerService
	ideas         ideaReader
	runs          runReader
	comparisons   comparisonReader
	archives      archiveUpdater
	events        eventPublisher
	workspaceRoot string
}

func NewService(
	jobs jobCreator,
	jobUpdates jobUpdater,
	triggers triggerService,
	ideas ideaReader,
	runs runReader,
	comparisons comparisonReader,
	archives archiveUpdater,
	events eventPublisher,
	workspaceRoot string,
) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{
		jobs:          jobs,
		jobUpdates:    jobUpdates,
		triggers:      triggers,
		ideas:         ideas,
		runs:          runs,
		comparisons:   comparisons,
		archives:      archives,
		events:        events,
		workspaceRoot: workspaceRoot,
	}
}

func (s *Service) Run(ctx context.Context, req model.WriterRunRequest) (*model.WriterRunResult, error) {
	normalizedReq, ideas, results, comparisons, citations, err := s.normalizeRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	inputRefs := []model.AgentInputRef{
		{RefType: "paper_template", RefPath: normalizedReq.PaperTemplateRef},
	}
	for _, item := range ideas {
		inputRefs = append(inputRefs, model.AgentInputRef{
			RefType: "idea",
			RefID:   stringValue(item["idea_ref"]),
			Metadata: map[string]any{
				"title":               item["title"],
				"research_direction":  item["research_direction"],
				"expected_advantage":  item["expected_advantage"],
				"innovation_type":     item["innovation_type"],
				"dataset_eval_refs":   item["dataset_eval_protocol_refs"],
				"target_dataset_refs": item["target_dataset_refs"],
			},
		})
	}
	for _, item := range results {
		inputRefs = append(inputRefs, model.AgentInputRef{
			RefType: "experiment_result",
			RefID:   stringValue(item["run_id"]),
			Metadata: map[string]any{
				"experiment_id":     item["experiment_id"],
				"result_archive_id": item["result_archive_id"],
				"metrics":           item["metrics"],
				"result_path":       item["result_path"],
			},
		})
	}
	for _, item := range comparisons {
		inputRefs = append(inputRefs, model.AgentInputRef{
			RefType: "comparison",
			RefID:   stringValue(item["comparison_id"]),
			RefPath: stringValue(item["comparison_path"]),
			Metadata: map[string]any{
				"summary_md": item["summary_md"],
			},
		})
	}
	for _, item := range citations {
		inputRefs = append(inputRefs, model.AgentInputRef{
			RefType: "citation",
			RefID:   stringValue(item["citation_ref"]),
			RefPath: stringValue(item["citation_path"]),
			Metadata: map[string]any{
				"citation_text": item["citation_text"],
			},
		})
	}
	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:       "writer",
		ExecutionMode:   normalizedReq.ExecutionMode,
		ModelProvider:   normalizedReq.ModelProvider,
		ModelName:       normalizedReq.ModelName,
		PromptVersion:   normalizedReq.PromptVersion,
		InputRefs:       inputRefs,
		OutputSchemaRef: writerOutputSchemaRef,
		SkillRefs:       normalizedReq.SkillRefs,
		ToolRefs:        normalizedReq.ToolRefs,
		MemoryRefs:      normalizedReq.MemoryRefs,
		Metadata: map[string]any{
			"paper_template_ref":     normalizedReq.PaperTemplateRef,
			"idea_refs":              normalizedReq.IdeaRefs,
			"experiment_result_refs": normalizedReq.ExperimentResultRefs,
			"comparison_refs":        normalizedReq.ComparisonRefs,
			"citation_refs":          normalizedReq.CitationRefs,
			"ideas":                  ideas,
			"experiment_results":     results,
			"comparisons":            comparisons,
			"citations":              citations,
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}
	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata:    map[string]any{"agent_type": "writer"},
	})
	if err != nil {
		return nil, err
	}
	return s.resultFromJob(job)
}

func (s *Service) PostProcess(ctx context.Context, job *model.AgentJob) error {
	if job == nil {
		return nil
	}
	req := requestFromJob(job)
	payload := extractWriterPayload(job.NormalizedPayload)
	if payload.Title == "" {
		return fmt.Errorf("writer payload title is required")
	}
	draftID := firstNonEmpty(payload.DraftID, httpx.NewID("draft"))
	paths := workspacepkg.New(s.workspaceRoot)
	draftDir := paths.DraftDir(draftID)
	if err := os.MkdirAll(draftDir, 0o755); err != nil {
		return err
	}
	figurePlanPath := filepath.Join(draftDir, "figure_plan.json")
	if err := writeJSON(figurePlanPath, payload.FigurePlan); err != nil {
		return err
	}
	draftMarkdown := buildDraftMarkdown(payload)
	draftMarkdownPath := filepath.Join(draftDir, "draft.md")
	if err := os.WriteFile(draftMarkdownPath, []byte(ensureTrailingLine(draftMarkdown)), 0o644); err != nil {
		return err
	}
	resultArchiveID := firstNonEmpty(payload.ResultArchiveID, findResultArchiveID(req.ExperimentResultRefs, job.InputRefs, s.runs))
	doc := model.DraftDocument{
		DraftID:           draftID,
		Title:             payload.Title,
		Abstract:          payload.Abstract,
		Introduction:      payload.Introduction,
		Method:            payload.Method,
		Experiments:       payload.Experiments,
		Conclusion:        payload.Conclusion,
		ReferencesStub:    append([]string{}, payload.ReferencesStub...),
		FigurePlan:        append([]model.FigurePlanItem{}, payload.FigurePlan...),
		PaperTemplateRef:  req.PaperTemplateRef,
		IdeaRefs:          append([]string{}, req.IdeaRefs...),
		ExperimentRunRefs: append([]string{}, req.ExperimentResultRefs...),
		ComparisonRefs:    append([]string{}, req.ComparisonRefs...),
		CitationRefs:      append([]string{}, req.CitationRefs...),
		ResultArchiveID:   resultArchiveID,
		DraftPath:         filepath.Join(draftDir, "draft.json"),
		DraftMarkdownPath: draftMarkdownPath,
		FigurePlanPath:    figurePlanPath,
		GeneratedAt:       time.Now(),
	}
	if err := writeJSON(doc.DraftPath, doc); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(draftDir, "sources.json"), map[string]any{
		"paper_template_ref":     req.PaperTemplateRef,
		"idea_refs":              req.IdeaRefs,
		"experiment_result_refs": req.ExperimentResultRefs,
		"comparison_refs":        req.ComparisonRefs,
		"citation_refs":          req.CitationRefs,
	}); err != nil {
		return err
	}
	if resultArchiveID != "" && s.archives != nil {
		if _, err := s.archives.Update(ctx, resultArchiveID, model.ResultArchiveUpdateRequest{
			Files: []model.ArchiveFileInput{
				{FileName: draftID + "_draft.md", FileKind: "paper_draft", Content: draftMarkdown},
				{FileName: draftID + "_figure_plan.json", FileKind: "figure_plan", Content: prettyJSON(payload.FigurePlan)},
				{FileName: draftID + "_draft.json", FileKind: "draft_json", Content: prettyJSON(doc)},
			},
		}); err != nil {
			return err
		}
	}
	job.NormalizedPayload = updateJobPayload(job.NormalizedPayload, doc)
	job.UpdatedAt = time.Now()
	if err := s.jobUpdates.Update(ctx, *job); err != nil {
		return err
	}
	return s.publishDraftReady(ctx, doc)
}

func (s *Service) GetDraft(_ context.Context, draftID string) (*model.DraftDocument, error) {
	draftID = strings.TrimSpace(draftID)
	if draftID == "" {
		return nil, fmt.Errorf("draft id is required")
	}
	path := filepath.Join(workspacepkg.New(s.workspaceRoot).DraftDir(draftID), "draft.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc model.DraftDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Service) resultFromJob(job *model.AgentJob) (*model.WriterRunResult, error) {
	if job == nil {
		return nil, fmt.Errorf("writer job not found")
	}
	result := &model.WriterRunResult{
		Job:      job,
		Warnings: append([]string{}, job.Warnings...),
	}
	draftID := stringValue(job.NormalizedPayload["draft_id"])
	if draftID == "" {
		return result, nil
	}
	draft, err := s.GetDraft(context.Background(), draftID)
	if err != nil {
		return nil, err
	}
	result.Draft = draft
	return result, nil
}

func (s *Service) normalizeRequest(ctx context.Context, req model.WriterRunRequest) (model.WriterRunRequest, []map[string]any, []map[string]any, []map[string]any, []map[string]any, error) {
	req.PaperTemplateRef = strings.TrimSpace(req.PaperTemplateRef)
	req.IdeaRefs = normalizeStringSlice(req.IdeaRefs)
	req.ExperimentResultRefs = normalizeStringSlice(req.ExperimentResultRefs)
	req.ComparisonRefs = normalizeStringSlice(req.ComparisonRefs)
	req.CitationRefs = normalizeStringSlice(req.CitationRefs)
	req.ExecutionMode = strings.TrimSpace(req.ExecutionMode)
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	if req.PaperTemplateRef == "" {
		return req, nil, nil, nil, nil, fmt.Errorf("paper_template_ref is required")
	}
	if len(req.IdeaRefs) == 0 {
		return req, nil, nil, nil, nil, fmt.Errorf("idea_refs is required")
	}
	if len(req.ExperimentResultRefs) == 0 {
		return req, nil, nil, nil, nil, fmt.Errorf("experiment_result_refs is required")
	}
	switch req.ExecutionMode {
	case "", "mock":
		req.ExecutionMode = "mock"
	case "api", "codex_cli":
	default:
		return req, nil, nil, nil, nil, fmt.Errorf("execution_mode must be one of api, codex_cli, mock")
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "codex"
	}
	if req.ModelName == "" {
		req.ModelName = "writer-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	ideas, err := s.resolveIdeas(ctx, req.IdeaRefs)
	if err != nil {
		return req, nil, nil, nil, nil, err
	}
	results, err := s.resolveResults(ctx, req.ExperimentResultRefs)
	if err != nil {
		return req, nil, nil, nil, nil, err
	}
	comparisons, err := s.resolveComparisons(ctx, req.ComparisonRefs, results)
	if err != nil {
		return req, nil, nil, nil, nil, err
	}
	citations := s.resolveCitations(req.CitationRefs)
	return req, ideas, results, comparisons, citations, nil
}

func (s *Service) resolveIdeas(ctx context.Context, refs []string) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(refs))
	for _, id := range refs {
		if s.ideas == nil {
			out = append(out, map[string]any{"idea_ref": id})
			continue
		}
		item, err := s.ideas.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if item == nil {
			return nil, fmt.Errorf("idea not found")
		}
		mapped := map[string]any{
			"idea_ref":           id,
			"title":              item.Idea.Title,
			"description_md":     item.Idea.DescriptionMD,
			"priority":           item.Idea.Priority,
			"confidence":         item.Idea.Confidence,
			"research_direction": "",
			"innovation_type":    "",
			"expected_advantage": "",
		}
		if item.StructuredIdea != nil {
			mapped["research_direction"] = item.StructuredIdea.ResearchDirection
			mapped["innovation_type"] = item.StructuredIdea.InnovationType
			mapped["expected_advantage"] = item.StructuredIdea.ExpectedAdvantage
			mapped["target_dataset_refs"] = append([]string{}, item.StructuredIdea.TargetDatasetRefs...)
			mapped["dataset_eval_protocol_refs"] = append([]string{}, item.StructuredIdea.DatasetEvalProtocolRefs...)
		}
		out = append(out, mapped)
	}
	return out, nil
}

func (s *Service) resolveResults(ctx context.Context, refs []string) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(refs))
	for _, id := range refs {
		if s.runs == nil {
			out = append(out, map[string]any{"run_id": id})
			continue
		}
		run, err := s.runs.GetRun(ctx, id)
		if err != nil {
			return nil, err
		}
		if run == nil {
			return nil, fmt.Errorf("experiment run not found")
		}
		out = append(out, map[string]any{
			"run_id":            run.ID,
			"experiment_id":     run.ExperimentID,
			"metrics":           mapValue(run.ResultJSON["metrics"]),
			"result_archive_id": stringValue(run.ResultJSON["result_archive_id"]),
			"result_path":       readNestedString(run.ResultJSON, "artifacts.result_path"),
		})
	}
	return out, nil
}

func (s *Service) resolveComparisons(ctx context.Context, refs []string, results []map[string]any) ([]map[string]any, error) {
	out := make([]map[string]any, 0)
	if len(refs) == 0 && len(results) > 0 {
		if expID := stringValue(results[0]["experiment_id"]); expID != "" {
			refs = []string{expID}
		}
	}
	for _, ref := range refs {
		if strings.HasSuffix(strings.ToLower(ref), ".json") || strings.HasSuffix(strings.ToLower(ref), ".md") {
			out = append(out, map[string]any{"comparison_ref": ref, "comparison_path": ref})
			continue
		}
		if s.comparisons == nil {
			out = append(out, map[string]any{"comparison_ref": ref})
			continue
		}
		items, err := s.comparisons.ListByExperimentID(ctx, ref)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			out = append(out, map[string]any{
				"comparison_id":   item.ID,
				"comparison_ref":  ref,
				"summary_md":      item.SummaryMD,
				"comparison_json": cloneMap(item.ComparisonJSON),
			})
		}
	}
	return out, nil
}

func (s *Service) resolveCitations(refs []string) []map[string]any {
	out := make([]map[string]any, 0, len(refs))
	for _, ref := range refs {
		out = append(out, map[string]any{
			"citation_ref":  ref,
			"citation_text": "[Citation] " + ref,
		})
	}
	return out
}

func (s *Service) publishDraftReady(ctx context.Context, doc model.DraftDocument) error {
	if s.events == nil {
		return nil
	}
	_, err := s.events.PublishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "draft_ready",
		SourceRef: "draft:" + doc.DraftID,
		InputRefs: []model.AgentInputRef{
			{RefType: "draft", RefID: doc.DraftID, RefPath: doc.DraftPath},
		},
		Payload: map[string]any{
			"draft_id":           doc.DraftID,
			"title":              doc.Title,
			"result_archive_id":  doc.ResultArchiveID,
			"experiment_results": doc.ExperimentRunRefs,
		},
	})
	return err
}

type writerRequest struct {
	PaperTemplateRef     string
	IdeaRefs             []string
	ExperimentResultRefs []string
	ComparisonRefs       []string
	CitationRefs         []string
}

type writerPayload struct {
	DraftID         string
	ResultArchiveID string
	Title           string
	Abstract        string
	Introduction    string
	Method          string
	Experiments     string
	Conclusion      string
	ReferencesStub  []string
	FigurePlan      []model.FigurePlanItem
}

func requestFromJob(job *model.AgentJob) writerRequest {
	return writerRequest{
		PaperTemplateRef:     stringValue(job.Metadata["paper_template_ref"]),
		IdeaRefs:             normalizeStringSlice(stringSliceValue(job.Metadata["idea_refs"])),
		ExperimentResultRefs: normalizeStringSlice(stringSliceValue(job.Metadata["experiment_result_refs"])),
		ComparisonRefs:       normalizeStringSlice(stringSliceValue(job.Metadata["comparison_refs"])),
		CitationRefs:         normalizeStringSlice(stringSliceValue(job.Metadata["citation_refs"])),
	}
}

func extractWriterPayload(payload map[string]any) writerPayload {
	return writerPayload{
		DraftID:         stringValue(payload["draft_id"]),
		ResultArchiveID: stringValue(payload["result_archive_id"]),
		Title:           stringValue(payload["title"]),
		Abstract:        stringValue(payload["abstract"]),
		Introduction:    stringValue(payload["introduction"]),
		Method:          stringValue(payload["method"]),
		Experiments:     stringValue(payload["experiments"]),
		Conclusion:      stringValue(payload["conclusion"]),
		ReferencesStub:  normalizeStringSlice(stringSliceValue(payload["references_stub"])),
		FigurePlan:      parseFigurePlan(payload["figure_plan"]),
	}
}

func updateJobPayload(payload map[string]any, doc model.DraftDocument) map[string]any {
	out := cloneMap(payload)
	if out == nil {
		out = map[string]any{}
	}
	out["draft_id"] = doc.DraftID
	out["draft_path"] = doc.DraftPath
	out["draft_markdown_path"] = doc.DraftMarkdownPath
	out["figure_plan_path"] = doc.FigurePlanPath
	out["result_archive_id"] = doc.ResultArchiveID
	return out
}

func parseFigurePlan(value any) []model.FigurePlanItem {
	switch typed := value.(type) {
	case []model.FigurePlanItem:
		return append([]model.FigurePlanItem{}, typed...)
	case []any:
		out := make([]model.FigurePlanItem, 0, len(typed))
		for _, item := range typed {
			mapped := mapValue(item)
			out = append(out, model.FigurePlanItem{
				FigureID:         stringValue(mapped["figure_id"]),
				FigureType:       stringValue(mapped["figure_type"]),
				Title:            stringValue(mapped["title"]),
				Description:      stringValue(mapped["description"]),
				SourceRefs:       normalizeStringSlice(stringSliceValue(mapped["source_refs"])),
				PlaceholderNotes: normalizeStringSlice(stringSliceValue(mapped["placeholder_notes"])),
			})
		}
		return out
	default:
		return []model.FigurePlanItem{}
	}
}

func findResultArchiveID(resultRefs []string, inputRefs []model.AgentInputRef, runs runReader) string {
	for _, ref := range inputRefs {
		if ref.RefType == "experiment_result" {
			if value := stringValue(ref.Metadata["result_archive_id"]); value != "" {
				return value
			}
		}
	}
	for _, id := range resultRefs {
		if runs == nil {
			continue
		}
		run, err := runs.GetRun(context.Background(), id)
		if err == nil && run != nil {
			if value := stringValue(run.ResultJSON["result_archive_id"]); value != "" {
				return value
			}
		}
	}
	return ""
}

func buildDraftMarkdown(payload writerPayload) string {
	lines := []string{
		"# " + payload.Title,
		"",
		"## Abstract",
		payload.Abstract,
		"",
		"## Introduction",
		payload.Introduction,
		"",
		"## Method",
		payload.Method,
		"",
		"## Experiments",
		payload.Experiments,
		"",
		"## Conclusion",
		payload.Conclusion,
		"",
		"## References Stub",
	}
	for _, item := range payload.ReferencesStub {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "", "## Figure Plan")
	for _, item := range payload.FigurePlan {
		lines = append(lines, "- "+firstNonEmpty(item.Title, item.FigureID)+": "+item.Description)
	}
	return strings.Join(lines, "\n")
}

func prettyJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func ensureTrailingLine(text string) string {
	if strings.HasSuffix(text, "\n") {
		return text
	}
	return text + "\n"
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
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := stringValue(item); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		if text := stringValue(value); text != "" {
			return []string{text}
		}
		return []string{}
	}
}

func mapValue(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	if typed, ok := value.(map[string]interface{}); ok {
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = item
		}
		return out
	}
	return map[string]any{}
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func readNestedString(root map[string]interface{}, path string) string {
	current := any(root)
	for _, part := range strings.Split(path, ".") {
		mapped, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = mapped[part]
	}
	return stringValue(current)
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "<nil>" {
			return ""
		}
		return text
	}
}

func firstNonEmpty(values ...string) string {
	for _, item := range values {
		if text := strings.TrimSpace(item); text != "" {
			return text
		}
	}
	return ""
}
