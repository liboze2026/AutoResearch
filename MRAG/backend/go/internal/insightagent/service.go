package insightagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/service"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

const insightOutputSchemaRef = "schemas/insight-output-v1.json"

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

type paperService interface {
	GetByID(context.Context, string) (*model.PaperDetail, error)
	ListInsights(context.Context, string) ([]model.PaperInsight, error)
	ApplyInsightAgentOutput(context.Context, string, string, service.InsightOutputPatch) (model.PaperInsight, string, error)
}

type Service struct {
	jobs          jobCreator
	jobUpdates    jobUpdater
	triggers      triggerService
	papers        paperService
	workspaceRoot string
}

func NewService(jobs jobCreator, jobUpdates jobUpdater, triggers triggerService, papers paperService, workspaceRoot string) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{
		jobs:          jobs,
		jobUpdates:    jobUpdates,
		triggers:      triggers,
		papers:        papers,
		workspaceRoot: workspaceRoot,
	}
}

func (s *Service) Run(ctx context.Context, req model.InsightRunRequest) (*model.InsightRunResult, error) {
	normalizedReq, err := s.normalizeRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:     "insight",
		ExecutionMode: normalizedReq.ExecutionMode,
		ModelProvider: normalizedReq.ModelProvider,
		ModelName:     normalizedReq.ModelName,
		PromptVersion: normalizedReq.PromptVersion,
		InputRefs: []model.AgentInputRef{
			{RefType: "paper", RefID: normalizedReq.PaperID},
			{RefType: "parsed_content", RefID: normalizedReq.PaperID, RefPath: normalizedReq.ParsedContentRef},
		},
		OutputSchemaRef: insightOutputSchemaRef,
		SkillRefs:       normalizedReq.SkillRefs,
		ToolRefs:        normalizedReq.ToolRefs,
		MemoryRefs:      normalizedReq.MemoryRefs,
		Metadata: map[string]any{
			"paper_id":           normalizedReq.PaperID,
			"parsed_content_ref": normalizedReq.ParsedContentRef,
			"focus":              normalizedReq.Focus,
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}
	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata: map[string]any{
			"paper_id": normalizedReq.PaperID,
		},
	})
	if err != nil {
		return nil, err
	}
	insight, summaryPath, err := s.latestInsight(ctx, normalizedReq.PaperID)
	if err != nil {
		return nil, err
	}
	return &model.InsightRunResult{
		Job:         job,
		Insight:     insight,
		SummaryPath: summaryPath,
		Warnings:    append([]string{}, job.Warnings...),
	}, nil
}

func (s *Service) PostProcess(ctx context.Context, job *model.AgentJob) error {
	if job == nil {
		return nil
	}
	req, err := s.requestFromJob(ctx, job)
	if err != nil {
		return err
	}
	patch := extractInsightPatch(job.NormalizedPayload)
	patch.ExtractMode = firstNonEmpty(strings.TrimSpace(patch.ExtractMode), job.ExecutionMode)
	patch.ExtractStatus = firstNonEmpty(strings.TrimSpace(patch.ExtractStatus), "completed")
	patch.SourceRef = firstNonEmpty(strings.TrimSpace(patch.SourceRef), "insight_agent_runtime")
	patch.Focus = req.Focus
	insight, summaryPath, err := s.papers.ApplyInsightAgentOutput(ctx, req.PaperID, req.ParsedContentRef, patch)
	if err != nil {
		return err
	}
	payload := ensureMap(job.NormalizedPayload)
	payload["paper_id"] = req.PaperID
	payload["parsed_content_ref"] = req.ParsedContentRef
	payload["summary_path"] = summaryPath
	payload["insight_id"] = insight.ID
	payload["summary_md"] = insight.SummaryMD
	payload["contributions_json"] = insight.ContributionsJSON
	payload["methods_json"] = insight.MethodsJSON
	payload["limitations_json"] = insight.LimitationsJSON
	payload["novelty_points"] = insight.NoveltyPointsJSON
	data := ensureMap(payload["data"])
	data["paper_id"] = req.PaperID
	data["parsed_content_ref"] = req.ParsedContentRef
	data["insight_id"] = insight.ID
	data["summary_path"] = summaryPath
	payload["data"] = data
	job.NormalizedPayload = payload
	job.UpdatedAt = time.Now()
	return s.jobUpdates.Update(ctx, *job)
}

func (s *Service) normalizeRequest(ctx context.Context, req model.InsightRunRequest) (model.InsightRunRequest, error) {
	req.PaperID = strings.TrimSpace(req.PaperID)
	req.ParsedContentRef = strings.TrimSpace(req.ParsedContentRef)
	req.Focus = strings.ToLower(strings.TrimSpace(req.Focus))
	req.ExecutionMode = strings.TrimSpace(req.ExecutionMode)
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	if req.PaperID == "" {
		return req, fmt.Errorf("paper_id is required")
	}
	if req.Focus != "" {
		switch req.Focus {
		case "method", "contribution", "limitation", "novelty":
		default:
			return req, fmt.Errorf("focus must be one of method, contribution, limitation, novelty")
		}
	}
	if req.ExecutionMode == "" {
		req.ExecutionMode = "mock"
	}
	switch req.ExecutionMode {
	case "api", "codex_cli", "mock":
	default:
		return req, fmt.Errorf("execution_mode must be one of api, codex_cli, mock")
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "codex"
	}
	if req.ModelName == "" {
		req.ModelName = "insight-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	if req.ParsedContentRef == "" {
		req.ParsedContentRef = filepath.Join(workspacepkg.New(s.workspaceRoot).PapersParsed(), req.PaperID, "parsed.md")
	}
	paper, err := s.papers.GetByID(ctx, req.PaperID)
	if err != nil {
		return req, err
	}
	if paper == nil {
		return req, fmt.Errorf("paper not found")
	}
	if _, err = os.Stat(req.ParsedContentRef); err != nil {
		return req, fmt.Errorf("parsed_content_ref not found")
	}
	return req, nil
}

func (s *Service) requestFromJob(ctx context.Context, job *model.AgentJob) (model.InsightRunRequest, error) {
	if job == nil {
		return model.InsightRunRequest{}, fmt.Errorf("agent job not found")
	}
	req := model.InsightRunRequest{
		PaperID:          stringValue(job.Metadata["paper_id"]),
		ParsedContentRef: stringValue(job.Metadata["parsed_content_ref"]),
		Focus:            stringValue(job.Metadata["focus"]),
		ExecutionMode:    job.ExecutionMode,
		ModelProvider:    job.ModelProvider,
		ModelName:        job.ModelName,
		PromptVersion:    job.PromptVersion,
		SkillRefs:        append([]string{}, job.SkillRefs...),
		ToolRefs:         append([]string{}, job.ToolRefs...),
		MemoryRefs:       append([]string{}, job.MemoryRefs...),
	}
	for _, ref := range job.InputRefs {
		if req.PaperID == "" && ref.RefType == "paper" && ref.RefID != "" {
			req.PaperID = ref.RefID
		}
		if req.ParsedContentRef == "" && ref.RefType == "parsed_content" {
			req.ParsedContentRef = ref.RefPath
			if req.PaperID == "" && ref.RefID != "" {
				req.PaperID = ref.RefID
			}
		}
	}
	return s.normalizeRequest(ctx, req)
}

func (s *Service) latestInsight(ctx context.Context, paperID string) (model.PaperInsight, string, error) {
	items, err := s.papers.ListInsights(ctx, paperID)
	if err != nil {
		return model.PaperInsight{}, "", err
	}
	if len(items) == 0 {
		return model.PaperInsight{}, "", fmt.Errorf("paper insight not found")
	}
	return items[0], filepath.Join(workspacepkg.New(s.workspaceRoot).PapersInsights(), paperID, "summary.md"), nil
}

func extractInsightPatch(payload map[string]any) service.InsightOutputPatch {
	return service.InsightOutputPatch{
		SummaryMD:     strings.TrimSpace(stringValue(payload["summary_md"])),
		Contributions: arrayValue(payload["contributions_json"]),
		Methods:       arrayValue(payload["methods_json"]),
		Limitations:   arrayValue(payload["limitations_json"]),
		NoveltyPoints: arrayValue(payload["novelty_points"]),
		ExtractMode:   firstNonEmpty(stringValue(payload["extract_mode"]), stringValue(ensureMap(payload["data"])["extract_mode"])),
		ExtractStatus: firstNonEmpty(stringValue(payload["extract_status"]), "completed"),
		ExtractError:  stringValue(payload["extract_error"]),
		SourceRef:     firstNonEmpty(stringValue(payload["source_ref"]), "insight_agent_runtime"),
		Focus:         stringValue(ensureMap(payload["data"])["focus"]),
	}
}

func ensureMap(value any) map[string]any {
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

func arrayValue(value any) []string {
	if value == nil {
		return []string{}
	}
	raw, err := json.Marshal(value)
	if err == nil {
		var stringsOnly []string
		if json.Unmarshal(raw, &stringsOnly) == nil {
			return stringsOnly
		}
		var anys []any
		if json.Unmarshal(raw, &anys) == nil {
			out := make([]string, 0, len(anys))
			for _, item := range anys {
				text := stringValue(item)
				if text != "" {
					out = append(out, text)
				}
			}
			return out
		}
	}
	text := stringValue(value)
	if text == "" {
		return []string{}
	}
	return []string{text}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
