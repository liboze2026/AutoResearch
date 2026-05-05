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
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

const phase4WriterOutputSchemaRef = "schemas/writer-phase4-output-v1.json"

type phase4WriterJobStore interface {
	Create(context.Context, model.AgentJobCreateRequest) (*model.AgentJob, error)
	GetByID(context.Context, string) (*model.AgentJob, error)
}

type phase4WriterArtifactLister interface {
	ListByJobID(context.Context, string) ([]model.AgentArtifact, error)
}

type phase4WriterDataService interface {
	GetRunManifestByID(context.Context, string) (*model.Phase4RunManifest, error)
	UpdateRunManifest(context.Context, string, model.Phase4RunManifestUpdateRequest) (*model.Phase4RunManifest, error)
	GetDatasetProfileByID(context.Context, string) (*model.Phase4DatasetProfile, error)
	GetReaderContextByID(context.Context, string) (*model.Phase4ReaderContext, error)
	GetReaderSourceByID(context.Context, string) (*model.Phase4ReaderSource, error)
	ListReaderSources(context.Context, string) ([]model.Phase4ReaderSource, error)
	GetIdeaByID(context.Context, string) (*model.Phase4Idea, error)
	CreateStructuredReport(context.Context, model.Phase4StructuredReportCreateRequest) (*model.Phase4StructuredReportRecord, error)
	GetStructuredReportByID(context.Context, string) (*model.Phase4StructuredReportRecord, error)
}

type Phase4Service struct {
	jobs          phase4WriterJobStore
	jobUpdates    jobUpdater
	triggers      triggerService
	artifacts     phase4WriterArtifactLister
	phase4        phase4WriterDataService
	workspaceRoot string
}

func NewPhase4Service(
	jobs phase4WriterJobStore,
	jobUpdates jobUpdater,
	triggers triggerService,
	artifacts phase4WriterArtifactLister,
	phase4 phase4WriterDataService,
	workspaceRoot string,
) *Phase4Service {
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

func (s *Phase4Service) Run(ctx context.Context, req model.Phase4WriterRunRequest) (*model.Phase4WriterRunResult, error) {
	req, runManifest, datasetProfile, readerContext, readerSources, idea, err := s.normalizeRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	metrics := phase4WriterReadJSON(phase4WriterResolvePath(runManifest.MetricsPath, runManifest.ArtifactPaths, "metrics_path"))
	codingMachineReport := phase4WriterReadJSON(stringValue(runManifest.ArtifactPaths["machine_report_path"]))
	codingHumanReport := phase4WriterReadText(stringValue(runManifest.ArtifactPaths["human_report_path"]))
	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:       "writer_phase4",
		ExecutionMode:   req.ExecutionMode,
		ModelProvider:   req.ModelProvider,
		ModelName:       req.ModelName,
		PromptVersion:   req.PromptVersion,
		InputRefs:       buildPhase4WriterInputRefs(*datasetProfile, readerContext, readerSources, *idea, *runManifest),
		OutputSchemaRef: phase4WriterOutputSchemaRef,
		SkillRefs:       req.SkillRefs,
		ToolRefs:        req.ToolRefs,
		MemoryRefs:      req.MemoryRefs,
		Metadata: map[string]any{
			"run_manifest_id":             runManifest.ID,
			"user_notes":                  req.UserNotes,
			"dataset_profile":             phase4WriterDatasetMetadata(*datasetProfile),
			"reader_context":              phase4WriterReaderContextMetadata(readerContext),
			"reader_sources":              phase4WriterReaderSourcesMetadata(readerSources),
			"selected_idea":               phase4WriterIdeaMetadata(*idea),
			"run_manifest":                phase4WriterRunManifestMetadata(*runManifest),
			"metrics":                     metrics,
			"failure_summary":             phase4WriterFailureSummary(*runManifest),
			"artifact_summary":            phase4WriterArtifactSummary(*runManifest),
			"coding_machine_report":       codingMachineReport,
			"coding_human_report_excerpt": phase4WriterExcerpt(codingHumanReport, 12),
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}
	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata: map[string]any{
			"agent_type": "writer_phase4",
		},
	})
	if err != nil {
		return nil, err
	}
	return s.resultFromJob(ctx, job)
}

func (s *Phase4Service) GetJob(ctx context.Context, jobID string) (*model.Phase4WriterJobDetail, error) {
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
	return &model.Phase4WriterJobDetail{
		Job:       job,
		Artifacts: artifacts,
		Report:    result.Report,
		Warnings:  result.Warnings,
	}, nil
}

func (s *Phase4Service) PostProcess(ctx context.Context, job *model.AgentJob) error {
	if job == nil {
		return nil
	}
	payload, err := decodePhase4WriterRuntimePayload(job.NormalizedPayload)
	if err != nil {
		return err
	}
	if strings.TrimSpace(payload.ReportTitle) == "" {
		return fmt.Errorf("phase4 writer report title is required")
	}
	runManifestID := strings.TrimSpace(stringValue(job.Metadata["run_manifest_id"]))
	if runManifestID == "" {
		return fmt.Errorf("phase4 writer run manifest id is required")
	}
	runManifest, err := s.phase4.GetRunManifestByID(ctx, runManifestID)
	if err != nil {
		return err
	}
	if runManifest == nil {
		return fmt.Errorf("phase4 run manifest not found")
	}
	reportRecord, err := s.phase4.CreateStructuredReport(ctx, model.Phase4StructuredReportCreateRequest{
		RunManifestID:         runManifest.ID,
		DatasetProfileID:      runManifest.DatasetProfileID,
		IdeaID:                runManifest.IdeaID,
		ReaderContextID:       runManifest.ReaderContextID,
		Title:                 payload.ReportTitle,
		MachineReadableReport: phase4WriterCloneMap(payload.MachineReadableReport),
		HumanReadableReportMD: strings.TrimSpace(payload.HumanReadableReportMD),
		CitationRefs:          phase4WriterNormalizeStrings(payload.CitationRefs),
		ReferenceSourceIDs:    phase4WriterNormalizeStrings(payload.ReferenceSourceIDs),
		Status:                model.Phase4ReportStatusFinalized,
	})
	if err != nil {
		return err
	}
	artifactDir := phase4WriterArtifactDir(s.workspaceRoot, *runManifest)
	if err = os.MkdirAll(artifactDir, 0o755); err != nil {
		return err
	}
	machineReportPath := filepath.Join(artifactDir, "phase4_structured_report.json")
	humanReportPath := filepath.Join(artifactDir, "phase4_experiment_report.md")
	citationPath := filepath.Join(artifactDir, "phase4_report_citations.json")
	if err = writeJSON(machineReportPath, payload.MachineReadableReport); err != nil {
		return err
	}
	if err = os.WriteFile(humanReportPath, []byte(ensureTrailingLine(payload.HumanReadableReportMD)), 0o644); err != nil {
		return err
	}
	if err = writeJSON(citationPath, map[string]any{
		"citation_refs":        payload.CitationRefs,
		"reference_source_ids": payload.ReferenceSourceIDs,
	}); err != nil {
		return err
	}
	artifactPaths := phase4WriterCloneMap(runManifest.ArtifactPaths)
	artifactPaths["phase4_structured_report_id"] = reportRecord.ID
	artifactPaths["phase4_structured_report_json_path"] = machineReportPath
	artifactPaths["phase4_human_report_md_path"] = humanReportPath
	artifactPaths["phase4_report_citations_path"] = citationPath
	if _, err = s.phase4.UpdateRunManifest(ctx, runManifest.ID, model.Phase4RunManifestUpdateRequest{
		ArtifactPaths: &artifactPaths,
	}); err != nil {
		return err
	}
	job.NormalizedPayload = phase4WriterUpdateJobPayload(job.NormalizedPayload, reportRecord.ID, machineReportPath, humanReportPath)
	job.UpdatedAt = time.Now()
	return s.jobUpdates.Update(ctx, *job)
}

func (s *Phase4Service) normalizeRequest(ctx context.Context, req model.Phase4WriterRunRequest) (model.Phase4WriterRunRequest, *model.Phase4RunManifest, *model.Phase4DatasetProfile, *model.Phase4ReaderContext, []model.Phase4ReaderSource, *model.Phase4Idea, error) {
	req.RunManifestID = strings.TrimSpace(req.RunManifestID)
	req.UserNotes = strings.TrimSpace(req.UserNotes)
	req.ExecutionMode = strings.TrimSpace(strings.ToLower(req.ExecutionMode))
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	if req.RunManifestID == "" {
		return req, nil, nil, nil, nil, nil, fmt.Errorf("runManifestId is required")
	}
	runManifest, err := s.phase4.GetRunManifestByID(ctx, req.RunManifestID)
	if err != nil {
		return req, nil, nil, nil, nil, nil, err
	}
	if runManifest == nil {
		return req, nil, nil, nil, nil, nil, fmt.Errorf("phase4 run manifest not found")
	}
	datasetProfile, err := s.phase4.GetDatasetProfileByID(ctx, runManifest.DatasetProfileID)
	if err != nil {
		return req, nil, nil, nil, nil, nil, err
	}
	if datasetProfile == nil {
		return req, nil, nil, nil, nil, nil, fmt.Errorf("phase4 dataset profile not found")
	}
	idea, err := s.phase4.GetIdeaByID(ctx, runManifest.IdeaID)
	if err != nil {
		return req, nil, nil, nil, nil, nil, err
	}
	if idea == nil {
		return req, nil, nil, nil, nil, nil, fmt.Errorf("phase4 idea not found")
	}
	var readerContext *model.Phase4ReaderContext
	if strings.TrimSpace(runManifest.ReaderContextID) != "" {
		readerContext, err = s.phase4.GetReaderContextByID(ctx, runManifest.ReaderContextID)
		if err != nil {
			return req, nil, nil, nil, nil, nil, err
		}
		if readerContext == nil {
			return req, nil, nil, nil, nil, nil, fmt.Errorf("phase4 reader context not found")
		}
	}
	readerSources, err := s.resolveReaderSources(ctx, datasetProfile.ID, readerContext)
	if err != nil {
		return req, nil, nil, nil, nil, nil, err
	}
	switch req.ExecutionMode {
	case "":
		req.ExecutionMode = "api"
	case "mock", "api", "codex_cli":
	default:
		return req, nil, nil, nil, nil, nil, fmt.Errorf("executionMode must be one of mock, api, codex_cli")
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "phase4_writer"
	}
	if req.ModelName == "" {
		req.ModelName = "writer-phase4-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	return req, runManifest, datasetProfile, readerContext, readerSources, idea, nil
}

func (s *Phase4Service) resolveReaderSources(ctx context.Context, datasetProfileID string, readerContext *model.Phase4ReaderContext) ([]model.Phase4ReaderSource, error) {
	if readerContext != nil && len(readerContext.SourceIDs) > 0 {
		items := make([]model.Phase4ReaderSource, 0, len(readerContext.SourceIDs))
		for _, sourceID := range readerContext.SourceIDs {
			item, err := s.phase4.GetReaderSourceByID(ctx, sourceID)
			if err != nil {
				return nil, err
			}
			if item != nil {
				items = append(items, *item)
			}
		}
		return items, nil
	}
	return s.phase4.ListReaderSources(ctx, datasetProfileID)
}

func (s *Phase4Service) resultFromJob(ctx context.Context, job *model.AgentJob) (*model.Phase4WriterRunResult, error) {
	if job == nil {
		return nil, fmt.Errorf("phase4 writer job not found")
	}
	reportID := strings.TrimSpace(stringValue(job.NormalizedPayload["report_id"]))
	if reportID == "" {
		return &model.Phase4WriterRunResult{
			Job:      job,
			Warnings: append([]string{}, job.Warnings...),
		}, nil
	}
	report, err := s.phase4.GetStructuredReportByID(ctx, reportID)
	if err != nil {
		return nil, err
	}
	return &model.Phase4WriterRunResult{
		Job:      job,
		Report:   report,
		Warnings: append([]string{}, job.Warnings...),
	}, nil
}

func buildPhase4WriterInputRefs(datasetProfile model.Phase4DatasetProfile, readerContext *model.Phase4ReaderContext, readerSources []model.Phase4ReaderSource, idea model.Phase4Idea, runManifest model.Phase4RunManifest) []model.AgentInputRef {
	inputRefs := []model.AgentInputRef{
		{
			RefType: "dataset_profile",
			RefID:   datasetProfile.ID,
			Metadata: map[string]any{
				"dataset_name":    datasetProfile.DatasetName,
				"task_type":       datasetProfile.TaskType,
				"official_metric": datasetProfile.OfficialMetric,
			},
		},
		{
			RefType: "phase4_run_manifest",
			RefID:   runManifest.ID,
			Metadata: map[string]any{
				"runner_mode": runManifest.RunnerMode,
				"status":      runManifest.Status,
				"retry_count": runManifest.RetryCount,
			},
		},
		{
			RefType: "idea",
			RefID:   idea.ID,
			Metadata: map[string]any{
				"title":              idea.Title,
				"problem_definition": idea.ProblemDefinition,
				"core_method":        idea.CoreMethod,
			},
		},
	}
	if readerContext != nil {
		inputRefs = append(inputRefs, model.AgentInputRef{
			RefType: "reader_context",
			RefID:   readerContext.ID,
			Metadata: map[string]any{
				"title":           readerContext.Title,
				"task_definition": readerContext.TaskDefinition,
				"source_ids":      readerContext.SourceIDs,
			},
		})
	}
	for _, source := range readerSources {
		inputRefs = append(inputRefs, model.AgentInputRef{
			RefType: "citation",
			RefID:   source.ID,
			RefPath: firstNonEmpty(source.OpenAccessURL, source.SourceURL),
			Metadata: map[string]any{
				"title":            source.Title,
				"venue":            source.Venue,
				"publication_year": source.PublicationYear,
				"source_type":      source.SourceType,
			},
		})
	}
	return inputRefs
}

func phase4WriterDatasetMetadata(item model.Phase4DatasetProfile) map[string]any {
	return map[string]any{
		"id":                    item.ID,
		"datasetName":           item.DatasetName,
		"taskType":              item.TaskType,
		"modalityComposition":   append([]string{}, item.ModalityComposition...),
		"splits":                item.Splits,
		"officialMetric":        item.OfficialMetric,
		"officialBaseline":      item.OfficialBaseline,
		"knownDifficulties":     append([]string{}, item.KnownDifficulties...),
		"sampleStatistics":      phase4WriterCloneMap(item.SampleStatistics),
		"fileStructureSnapshot": phase4WriterCloneMap(item.FileStructureSnapshot),
		"citation":              item.Citation,
		"serverPath":            item.ServerPath,
	}
}

func phase4WriterReaderContextMetadata(item *model.Phase4ReaderContext) map[string]any {
	if item == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":                item.ID,
		"title":             item.Title,
		"summary":           item.Summary,
		"taskDefinition":    item.TaskDefinition,
		"relatedWork":       append([]string{}, item.RelatedWork...),
		"retrievalFocus":    append([]string{}, item.RetrievalFocus...),
		"rankingNotes":      item.RankingNotes,
		"sourceIds":         append([]string{}, item.SourceIDs...),
		"structuredContext": phase4WriterCloneMap(item.StructuredContext),
	}
}

func phase4WriterReaderSourcesMetadata(items []model.Phase4ReaderSource) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"id":              item.ID,
			"title":           item.Title,
			"authors":         append([]string{}, item.Authors...),
			"venue":           item.Venue,
			"publicationYear": item.PublicationYear,
			"sourceType":      item.SourceType,
			"sourceUrl":       item.SourceURL,
			"openAccessUrl":   item.OpenAccessURL,
			"qualityTier":     item.QualityTier,
			"rankingScore":    item.RankingScore,
			"qualityScore":    item.QualityScore,
			"relevanceScore":  item.RelevanceScore,
			"citationCount":   item.CitationCount,
			"metadata":        phase4WriterCloneMap(item.Metadata),
		})
	}
	return out
}

func phase4WriterIdeaMetadata(item model.Phase4Idea) map[string]any {
	return map[string]any{
		"id":                  item.ID,
		"title":               item.Title,
		"problemDefinition":   item.ProblemDefinition,
		"coreMethod":          item.CoreMethod,
		"differentiators":     item.Differentiators,
		"dataProcessingNeeds": append([]string{}, item.DataProcessingNeeds...),
		"modelChanges":        append([]string{}, item.ModelChanges...),
		"trainingPlan":        item.TrainingPlan,
		"evaluationMetrics":   append([]string{}, item.EvaluationMetrics...),
		"riskPoints":          append([]string{}, item.RiskPoints...),
		"expectedGains":       append([]string{}, item.ExpectedGains...),
		"scoreSummary":        phase4WriterCloneMap(item.ScoreSummary),
	}
}

func phase4WriterRunManifestMetadata(item model.Phase4RunManifest) map[string]any {
	return map[string]any{
		"id":               item.ID,
		"datasetProfileId": item.DatasetProfileID,
		"ideaId":           item.IdeaID,
		"readerContextId":  item.ReaderContextID,
		"codeSnapshotId":   item.CodeSnapshotID,
		"runnerMode":       item.RunnerMode,
		"serverId":         item.ServerID,
		"gpu":              item.GPU,
		"status":           item.Status,
		"retryCount":       item.RetryCount,
		"maxRetryCount":    item.MaxRetryCount,
		"artifactPaths":    phase4WriterCloneMap(item.ArtifactPaths),
		"logsPath":         item.LogsPath,
		"metricsPath":      item.MetricsPath,
		"startedAt":        item.StartedAt,
		"finishedAt":       item.FinishedAt,
	}
}

func phase4WriterFailureSummary(item model.Phase4RunManifest) map[string]any {
	out := phase4WriterCloneMap(item.FailureFeedback)
	if len(out) == 0 {
		path := stringValue(item.ArtifactPaths["failure_feedback_path"])
		out = phase4WriterReadJSON(path)
	}
	if len(out) == 0 && strings.TrimSpace(item.Status) == model.Phase4RunStatusTestFailed {
		out["status"] = item.Status
		out["final_error"] = "run marked as test_failed without a persisted failure feedback payload"
	}
	driverLog := phase4WriterResolvePath("", item.ArtifactPaths, "driver_log_path")
	runLog := phase4WriterResolvePath(item.LogsPath, item.ArtifactPaths, "run_log_path")
	if tail := phase4WriterTail(driverLog, 20); tail != "" {
		out["driver_log_tail"] = tail
	}
	if tail := phase4WriterTail(runLog, 20); tail != "" {
		out["run_log_tail"] = tail
	}
	return out
}

func phase4WriterArtifactSummary(item model.Phase4RunManifest) map[string]any {
	out := phase4WriterCloneMap(item.ArtifactPaths)
	out["logsPath"] = item.LogsPath
	out["metricsPath"] = item.MetricsPath
	return out
}

func decodePhase4WriterRuntimePayload(payload map[string]any) (model.Phase4WriterRuntimePayload, error) {
	raw, err := json.Marshal(phase4WriterCloneMap(payload))
	if err != nil {
		return model.Phase4WriterRuntimePayload{}, err
	}
	var out model.Phase4WriterRuntimePayload
	if err = json.Unmarshal(raw, &out); err != nil {
		return model.Phase4WriterRuntimePayload{}, err
	}
	return out, nil
}

func phase4WriterUpdateJobPayload(payload map[string]any, reportID string, machineReportPath string, humanReportPath string) map[string]any {
	out := phase4WriterCloneMap(payload)
	out["report_id"] = reportID
	out["machine_report_path"] = machineReportPath
	out["human_report_path"] = humanReportPath
	return out
}

func phase4WriterCloneMap(value any) map[string]any {
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

func phase4WriterNormalizeStrings(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		text := strings.TrimSpace(item)
		if text == "" {
			continue
		}
		key := strings.ToLower(text)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, text)
	}
	return out
}

func phase4WriterReadJSON(path string) map[string]any {
	path = strings.TrimSpace(path)
	if path == "" {
		return map[string]any{}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if err = json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func phase4WriterReadText(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func phase4WriterResolvePath(primary string, artifactPaths map[string]any, fallbackKey string) string {
	if text := strings.TrimSpace(primary); text != "" {
		return text
	}
	return stringValue(artifactPaths[fallbackKey])
}

func phase4WriterTail(path string, maxLines int) string {
	text := strings.TrimSpace(phase4WriterReadText(path))
	if text == "" {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			filtered = append(filtered, line)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	if maxLines > 0 && len(filtered) > maxLines {
		filtered = filtered[len(filtered)-maxLines:]
	}
	return strings.Join(filtered, "\n")
}

func phase4WriterExcerpt(text string, maxLines int) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func phase4WriterArtifactDir(workspaceRoot string, runManifest model.Phase4RunManifest) string {
	artifactDir := stringValue(runManifest.ArtifactPaths["artifact_dir"])
	if strings.TrimSpace(artifactDir) != "" {
		return artifactDir
	}
	paths := workspacepkg.New(workspaceRoot)
	return paths.Phase4ArtifactDir(runManifest.ID)
}
