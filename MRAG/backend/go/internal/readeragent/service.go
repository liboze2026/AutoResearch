package readeragent

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

const readerOutputSchemaRef = "schemas/reader-output-v1.json"

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

type artifactLister interface {
	ListByJobID(context.Context, string) ([]model.AgentArtifact, error)
}

type paperService interface {
	ImportExistingFile(context.Context, string) (*model.PaperImportResult, error)
	ApplyReaderMetadata(context.Context, string, service.PaperMetadataPatch) (*model.Paper, error)
}

type Service struct {
	jobs          jobCreator
	jobUpdates    jobUpdater
	triggers      triggerService
	artifacts     artifactLister
	papers        paperService
	workspaceRoot string
}

func NewService(jobs jobCreator, jobUpdates jobUpdater, triggers triggerService, artifacts artifactLister, papers paperService, workspaceRoot string) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{
		jobs:          jobs,
		jobUpdates:    jobUpdates,
		triggers:      triggers,
		artifacts:     artifacts,
		papers:        papers,
		workspaceRoot: workspaceRoot,
	}
}

func (s *Service) Run(ctx context.Context, req model.ReaderRunRequest) (*model.ReaderRunResult, error) {
	normalizedReq, err := s.normalizeRequest(req)
	if err != nil {
		return nil, err
	}
	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:       "reader",
		ExecutionMode:   normalizedReq.ExecutionMode,
		ModelProvider:   normalizedReq.ModelProvider,
		ModelName:       normalizedReq.ModelName,
		PromptVersion:   normalizedReq.PromptVersion,
		InputRefs:       buildInputRefs(normalizedReq.ManualPapers),
		OutputSchemaRef: readerOutputSchemaRef,
		SkillRefs:       normalizedReq.SkillRefs,
		ToolRefs:        normalizedReq.ToolRefs,
		MemoryRefs:      normalizedReq.MemoryRefs,
		Metadata: map[string]any{
			"research_direction": normalizedReq.ResearchDirection,
			"keywords":           normalizedReq.Keywords,
			"source_scope":       normalizedReq.SourceScope,
			"time_range":         normalizedReq.TimeRange,
			"max_papers":         normalizedReq.MaxPapers,
			"manual_papers":      normalizedReq.ManualPapers,
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}

	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata: map[string]any{
			"agent_type": "reader",
		},
	})
	if err != nil {
		return nil, err
	}

	candidatePapers := extractCandidatePapers(job.NormalizedPayload)
	importedPapers := make([]model.ReaderImportedPaper, 0, len(candidatePapers))
	warnings := append([]string{}, job.Warnings...)
	for index, candidate := range candidatePapers {
		stagedPath, stageWarning, stageErr := s.stageCandidateFile(candidate, job.ID, index)
		if stageWarning != "" {
			warnings = append(warnings, stageWarning)
		}
		if stageErr != nil {
			warnings = append(warnings, stageErr.Error())
			continue
		}

		result, importErr := s.papers.ImportExistingFile(ctx, stagedPath)
		if importErr != nil {
			warnings = append(warnings, importErr.Error())
			continue
		}
		updatedPaper, patchErr := s.papers.ApplyReaderMetadata(ctx, result.Paper.ID, service.PaperMetadataPatch{
			Title:      candidate.Title,
			Abstract:   candidate.Abstract,
			Venue:      candidate.Source,
			Year:       candidate.Year,
			SourceType: candidate.Source,
			SourceURL:  candidate.URL,
			ParserNote: "Reader Agent imported structured candidate metadata.",
		})
		if patchErr != nil {
			warnings = append(warnings, patchErr.Error())
			continue
		}
		result.Paper = *updatedPaper
		importedPapers = append(importedPapers, model.ReaderImportedPaper{
			Candidate: candidate,
			Result:    *result,
		})
	}

	if len(candidatePapers) > 0 && len(importedPapers) == 0 {
		job.Status = "failed"
		job.ErrorMessage = "reader candidate import failed for all papers"
		job.Warnings = warnings
		job.UpdatedAt = time.Now()
		if updateErr := s.jobUpdates.Update(ctx, *job); updateErr != nil {
			return nil, updateErr
		}
		return nil, fmt.Errorf(job.ErrorMessage)
	}

	if err = s.persistImportSummary(ctx, job, candidatePapers, importedPapers, warnings); err != nil {
		return nil, err
	}

	return &model.ReaderRunResult{
		Job:             job,
		CandidatePapers: candidatePapers,
		ImportedPapers:  importedPapers,
		Warnings:        warnings,
	}, nil
}

func (s *Service) GetJob(ctx context.Context, jobID string) (*model.ReaderJobDetail, error) {
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
	return &model.ReaderJobDetail{
		Job:             job,
		Artifacts:       artifacts,
		CandidatePapers: extractCandidatePapers(job.NormalizedPayload),
		ImportedPapers:  extractImportedPapers(job.NormalizedPayload),
	}, nil
}

func (s *Service) normalizeRequest(req model.ReaderRunRequest) (model.ReaderRunRequest, error) {
	req.ResearchDirection = strings.TrimSpace(req.ResearchDirection)
	req.SourceScope = strings.ToLower(strings.TrimSpace(req.SourceScope))
	req.ExecutionMode = strings.TrimSpace(req.ExecutionMode)
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	if req.ResearchDirection == "" && len(req.ManualPapers) == 0 {
		return req, fmt.Errorf("research_direction is required unless manual_papers is provided")
	}
	if req.SourceScope == "" {
		req.SourceScope = "mixed"
	}
	switch req.SourceScope {
	case "arxiv", "conference", "journal", "mixed":
	default:
		return req, fmt.Errorf("source_scope must be one of arxiv, conference, journal, mixed")
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
		req.ModelName = "reader-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	if req.MaxPapers <= 0 {
		req.MaxPapers = 5
	}
	if req.MaxPapers > 20 {
		req.MaxPapers = 20
	}
	if req.TimeRange == nil {
		req.TimeRange = map[string]any{}
	}
	return req, nil
}

func buildInputRefs(items []model.ReaderManualPaperInput) []model.AgentInputRef {
	refs := make([]model.AgentInputRef, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.FilePath) == "" {
			continue
		}
		refs = append(refs, model.AgentInputRef{
			RefType: "paper_upload",
			RefPath: strings.TrimSpace(item.FilePath),
			Metadata: map[string]any{
				"title":       strings.TrimSpace(item.Title),
				"abstract":    strings.TrimSpace(item.Abstract),
				"source":      strings.TrimSpace(item.Source),
				"year":        item.Year,
				"url":         strings.TrimSpace(item.URL),
				"file_status": strings.TrimSpace(item.FileStatus),
			},
		})
	}
	return refs
}

func (s *Service) stageCandidateFile(candidate model.ReaderCandidatePaper, jobID string, index int) (string, string, error) {
	paths := workspacepkg.New(s.workspaceRoot)
	if err := os.MkdirAll(paths.PapersIncoming(), 0o755); err != nil {
		return "", "", err
	}
	sourcePath := strings.TrimSpace(candidate.FilePath)
	if sourcePath != "" {
		staged, copied, err := stageExistingFile(sourcePath, paths.PapersIncoming(), jobID, index, candidate.Title)
		if err == nil {
			if copied {
				return staged, "reader candidate file was copied into workspace/papers/incoming before import", nil
			}
			return staged, "", nil
		}
	}
	targetName := fmt.Sprintf("%s_%02d_%s.md", jobID, index+1, sanitizeFileName(candidate.Title))
	targetPath := filepath.Join(paths.PapersIncoming(), targetName)
	content := buildReaderPaperDocument(candidate)
	if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
		return "", "", err
	}
	return targetPath, "", nil
}

func buildReaderPaperDocument(candidate model.ReaderCandidatePaper) string {
	lines := []string{
		"# " + firstNonEmpty(strings.TrimSpace(candidate.Title), "Untitled Reader Paper"),
		"",
		"## Abstract",
		firstNonEmpty(strings.TrimSpace(candidate.Abstract), "No abstract provided by Reader Agent."),
		"",
		"## Source",
		firstNonEmpty(strings.TrimSpace(candidate.Source), "unknown"),
		"",
		"## Year",
		fmt.Sprintf("%d", candidate.Year),
		"",
		"## URL",
		firstNonEmpty(strings.TrimSpace(candidate.URL), ""),
		"",
		"## File Status",
		firstNonEmpty(strings.TrimSpace(candidate.FileStatus), "metadata_only"),
		"",
	}
	return strings.Join(lines, "\n")
}

func stageExistingFile(sourcePath string, incomingRoot string, jobID string, index int, title string) (string, bool, error) {
	cleanSource := filepath.Clean(sourcePath)
	if !filepath.IsAbs(cleanSource) {
		cleanSource = filepath.Clean(filepath.Join(".", cleanSource))
	}
	info, err := os.Stat(cleanSource)
	if err != nil {
		return "", false, err
	}
	if info.IsDir() {
		return "", false, fmt.Errorf("candidate file path must be a file")
	}
	incomingRootAbs, _ := filepath.Abs(incomingRoot)
	sourceAbs, _ := filepath.Abs(cleanSource)
	if hasPathPrefix(sourceAbs, incomingRootAbs) {
		return sourceAbs, false, nil
	}
	targetName := fmt.Sprintf("%s_%02d_%s%s", jobID, index+1, sanitizeFileName(title), filepath.Ext(sourceAbs))
	targetPath := filepath.Join(incomingRootAbs, targetName)
	content, err := os.ReadFile(sourceAbs)
	if err != nil {
		return "", false, err
	}
	if err = os.WriteFile(targetPath, content, 0o644); err != nil {
		return "", false, err
	}
	return targetPath, true, nil
}

func sanitizeFileName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "reader_paper"
	}
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(value)
}

func hasPathPrefix(value string, prefix string) bool {
	value = strings.ToLower(filepath.Clean(value))
	prefix = strings.ToLower(filepath.Clean(prefix))
	return strings.HasPrefix(value, prefix)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *Service) persistImportSummary(ctx context.Context, job *model.AgentJob, candidatePapers []model.ReaderCandidatePaper, importedPapers []model.ReaderImportedPaper, warnings []string) error {
	if job == nil {
		return nil
	}
	payload := ensureMap(job.NormalizedPayload)
	payload["candidate_papers"] = candidatePapers
	payload["items"] = candidatePapers
	payload["imported_papers"] = importedPapers
	data := ensureMap(payload["data"])
	data["candidate_count"] = len(candidatePapers)
	data["imported_paper_count"] = len(importedPapers)
	importedIDs := make([]string, 0, len(importedPapers))
	for _, item := range importedPapers {
		importedIDs = append(importedIDs, item.Result.Paper.ID)
	}
	data["imported_paper_ids"] = importedIDs
	payload["data"] = data
	job.NormalizedPayload = payload
	job.Warnings = warnings
	job.UpdatedAt = time.Now()
	return s.jobUpdates.Update(ctx, *job)
}

func extractCandidatePapers(payload map[string]any) []model.ReaderCandidatePaper {
	raw, ok := payload["candidate_papers"]
	if !ok {
		raw = payload["items"]
	}
	items, err := decodeReaderCandidatePapers(raw)
	if err == nil && len(items) > 0 {
		return items
	}
	data, _ := payload["data"].(map[string]any)
	if data != nil {
		if items, err = decodeReaderCandidatePapers(data["candidate_papers"]); err == nil && len(items) > 0 {
			return items
		}
	}
	return []model.ReaderCandidatePaper{}
}

func extractImportedPapers(payload map[string]any) []model.ReaderImportedPaper {
	items, err := decodeReaderImportedPapers(payload["imported_papers"])
	if err != nil {
		return []model.ReaderImportedPaper{}
	}
	return items
}

func decodeReaderCandidatePapers(value any) ([]model.ReaderCandidatePaper, error) {
	if value == nil {
		return nil, fmt.Errorf("candidate papers not found")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var items []model.ReaderCandidatePaper
	if err = json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func decodeReaderImportedPapers(value any) ([]model.ReaderImportedPaper, error) {
	if value == nil {
		return nil, fmt.Errorf("imported papers not found")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var items []model.ReaderImportedPaper
	if err = json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func ensureMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	typed, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	out := make(map[string]any, len(typed))
	for key, item := range typed {
		out[key] = item
	}
	return out
}
