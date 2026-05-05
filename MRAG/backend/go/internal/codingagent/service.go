package codingagent

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
	"mrag-platform/backend/go/internal/traintemplate"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

const codingOutputSchemaRef = "schemas/coding-output-v1.json"

type jobCreator interface {
	Create(context.Context, model.AgentJobCreateRequest) (*model.AgentJob, error)
}

type jobUpdater interface {
	Update(context.Context, model.AgentJob) error
}

type triggerService interface {
	Trigger(context.Context, string, model.AgentJobTriggerRequest) (*model.AgentJob, error)
}

type experimentManager interface {
	GetByID(context.Context, string) (*model.ExperimentDetail, error)
	GenerateSpec(context.Context, string) (*model.ExperimentSpecDetail, error)
	GetLatestSpec(context.Context, string) (*model.ExperimentSpecDetail, error)
}

type ideaReader interface {
	GetByID(context.Context, string) (*model.IdeaDetail, error)
}

type specStore interface {
	GetLatestByExperimentID(context.Context, string) (*model.ExperimentSpec, error)
	GetByID(context.Context, string) (*model.ExperimentSpec, error)
	Create(context.Context, model.ExperimentSpec) error
}

type schedulerExecutor interface {
	QueueExperiment(context.Context, string) (*model.ExperimentQueueResult, error)
	ScheduleRun(context.Context, string) (*model.ScheduleResult, error)
}

type runExecutor interface {
	StartRun(context.Context, string) (*model.ExperimentRun, error)
	GetRun(context.Context, string) (*model.ExperimentRun, error)
}

type runComparer interface {
	CompareRun(context.Context, string) (*model.RunCompareResult, error)
}

type Service struct {
	jobs          jobCreator
	jobUpdates    jobUpdater
	triggers      triggerService
	experiments   experimentManager
	ideas         ideaReader
	specs         specStore
	scheduler     schedulerExecutor
	runs          runExecutor
	comparer      runComparer
	templates     *traintemplate.Service
	workspaceRoot string
	templateRoot  string
}

func NewService(
	jobs jobCreator,
	jobUpdates jobUpdater,
	triggers triggerService,
	experiments experimentManager,
	ideas ideaReader,
	specs specStore,
	scheduler schedulerExecutor,
	runs runExecutor,
	comparer runComparer,
	templates *traintemplate.Service,
	workspaceRoot string,
	templateRoot string,
) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{
		jobs:          jobs,
		jobUpdates:    jobUpdates,
		triggers:      triggers,
		experiments:   experiments,
		ideas:         ideas,
		specs:         specs,
		scheduler:     scheduler,
		runs:          runs,
		comparer:      comparer,
		templates:     templates,
		workspaceRoot: workspaceRoot,
		templateRoot:  templateRoot,
	}
}

func (s *Service) Run(ctx context.Context, req model.CodingRunRequest) (*model.CodingRunResult, error) {
	normalizedReq, expDetail, specDetail, planPath, err := s.normalizeRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ideaDetail, _ := s.resolveIdea(ctx, expDetail, normalizedReq.IdeaID)
	evalProtocolRef := firstNonEmpty(strings.TrimSpace(normalizedReq.EvalProtocolRef), defaultEvalProtocolRef(s.workspaceRoot, expDetail.Experiment.DatasetAssetID))
	specPath := specDetail.WorkspacePath
	if strings.TrimSpace(specPath) == "" {
		specPath = filepath.Join(workspacepkg.New(s.workspaceRoot).ExperimentDir(expDetail.Experiment.ID), "spec.json")
	}
	templatePath := resolveTemplatePath(s.templateRoot, normalizedReq.TrainTemplateRef)
	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:     "coding",
		ExecutionMode: normalizedReq.ExecutionMode,
		ModelProvider: normalizedReq.ModelProvider,
		ModelName:     normalizedReq.ModelName,
		PromptVersion: normalizedReq.PromptVersion,
		InputRefs: []model.AgentInputRef{
			{RefType: "experiment", RefID: expDetail.Experiment.ID},
			buildIdeaInputRef(s.workspaceRoot, expDetail.Experiment.IdeaID, ideaDetail),
			{RefType: "experiment_plan", RefPath: planPath},
			{RefType: "experiment_spec", RefID: specDetail.Spec.ID, RefPath: specPath},
			{RefType: "train_template", RefID: normalizedReq.TrainTemplateRef, RefPath: templatePath},
			{RefType: "dataset_eval_protocol", RefPath: evalProtocolRef},
		},
		OutputSchemaRef: codingOutputSchemaRef,
		SkillRefs:       normalizedReq.SkillRefs,
		ToolRefs:        normalizedReq.ToolRefs,
		MemoryRefs:      normalizedReq.MemoryRefs,
		Metadata: map[string]any{
			"experiment_id":      expDetail.Experiment.ID,
			"idea_id":            expDetail.Experiment.IdeaID,
			"idea":               ideaMetadata(ideaDetail),
			"train_template_ref": normalizedReq.TrainTemplateRef,
			"eval_protocol_ref":  evalProtocolRef,
			"experiment_plan":    readJSON(planPath),
			"experiment_spec":    cloneInterfaceMap(specDetail.Spec.SpecJSON),
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}
	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata:    map[string]any{"agent_type": "coding"},
	})
	if err != nil {
		return nil, err
	}
	return s.resultFromJob(ctx, job)
}

func (s *Service) PostProcess(ctx context.Context, job *model.AgentJob) error {
	if job == nil {
		return nil
	}
	req := requestFromJob(job)
	expDetail, err := s.experiments.GetByID(ctx, req.ExperimentID)
	if err != nil {
		return err
	}
	if expDetail == nil {
		return fmt.Errorf("experiment not found")
	}
	specDetail, err := s.ensureLatestSpec(ctx, req.ExperimentID)
	if err != nil {
		return err
	}
	payload := extractCodingPayload(job.NormalizedPayload)
	codingDir := filepath.Join(workspacepkg.New(s.workspaceRoot).ExperimentDir(req.ExperimentID), "coding")
	if err = os.MkdirAll(codingDir, 0o755); err != nil {
		return err
	}
	patchManifest, err := s.persistPatchManifest(codingDir, payload)
	if err != nil {
		return err
	}
	appliedSpec, err := s.applyPatchedSpec(ctx, *specDetail, payload, codingDir)
	if err != nil {
		return err
	}
	queueResult, err := s.scheduler.QueueExperiment(ctx, req.ExperimentID)
	if err != nil {
		return err
	}
	scheduleResult, err := s.scheduler.ScheduleRun(ctx, queueResult.Run.ID)
	if err != nil {
		return err
	}
	startedRun, err := s.runs.StartRun(ctx, scheduleResult.Run.ID)
	if err != nil {
		return err
	}
	var compareResult *model.RunCompareResult
	if strings.TrimSpace(readNestedString(startedRun.ResultJSON, "comparison_status")) != "completed" && s.comparer != nil {
		compareResult, err = s.comparer.CompareRun(ctx, startedRun.ID)
		if err != nil {
			return err
		}
	}
	evaluationSummary := buildEvaluationSummary(*startedRun, compareResult, patchManifest)
	evaluationPath := filepath.Join(codingDir, "evaluation_summary.md")
	if err = os.WriteFile(evaluationPath, []byte(ensureTrailingLine(evaluationSummary)), 0o644); err != nil {
		return err
	}
	job.NormalizedPayload = updateCodingJobPayload(job.NormalizedPayload, patchManifest, appliedSpec, startedRun, compareResult, evaluationSummary)
	job.UpdatedAt = time.Now()
	return s.jobUpdates.Update(ctx, *job)
}

func (s *Service) resultFromJob(ctx context.Context, job *model.AgentJob) (*model.CodingRunResult, error) {
	if job == nil {
		return nil, fmt.Errorf("coding job not found")
	}
	req := requestFromJob(job)
	result := &model.CodingRunResult{
		Job:           job,
		PatchManifest: extractPatchManifest(job.NormalizedPayload["code_patch_manifest"]),
		Warnings:      append([]string{}, job.Warnings...),
	}
	if req.ExperimentID != "" {
		expDetail, err := s.experiments.GetByID(ctx, req.ExperimentID)
		if err != nil {
			return nil, err
		}
		result.Experiment = expDetail
	}
	runID := stringValue(job.NormalizedPayload["run_id"])
	if runID != "" {
		run, err := s.runs.GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}
		result.Run = run
	}
	return result, nil
}

func (s *Service) normalizeRequest(ctx context.Context, req model.CodingRunRequest) (model.CodingRunRequest, *model.ExperimentDetail, *model.ExperimentSpecDetail, string, error) {
	req.ExperimentID = firstNonEmpty(strings.TrimSpace(req.ExperimentID), inferExperimentIDFromPath(req.ExperimentPlanRef), inferExperimentIDFromPath(req.ExperimentSpecRef))
	req.IdeaID = strings.TrimSpace(req.IdeaID)
	req.ExperimentPlanRef = strings.TrimSpace(req.ExperimentPlanRef)
	req.ExperimentSpecRef = strings.TrimSpace(req.ExperimentSpecRef)
	req.TrainTemplateRef = strings.TrimSpace(req.TrainTemplateRef)
	req.EvalProtocolRef = strings.TrimSpace(req.EvalProtocolRef)
	req.ExecutionMode = strings.TrimSpace(req.ExecutionMode)
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	if req.ExperimentID == "" {
		return req, nil, nil, "", fmt.Errorf("experiment_id is required")
	}
	expDetail, err := s.experiments.GetByID(ctx, req.ExperimentID)
	if err != nil {
		return req, nil, nil, "", err
	}
	if expDetail == nil {
		return req, nil, nil, "", fmt.Errorf("experiment not found")
	}
	switch req.ExecutionMode {
	case "", "mock":
		req.ExecutionMode = "mock"
	case "api", "codex_cli":
	default:
		return req, nil, nil, "", fmt.Errorf("execution_mode must be one of api, codex_cli, mock")
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "codex"
	}
	if req.ModelName == "" {
		req.ModelName = "coding-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	if req.IdeaID == "" {
		req.IdeaID = expDetail.Experiment.IdeaID
	}
	planPath := firstNonEmpty(req.ExperimentPlanRef, filepath.Join(workspacepkg.New(s.workspaceRoot).ExperimentDir(req.ExperimentID), "plan.json"))
	specDetail, err := s.ensureLatestSpec(ctx, req.ExperimentID)
	if err != nil {
		return req, nil, nil, "", err
	}
	if req.ExperimentSpecRef == "" && specDetail != nil {
		req.ExperimentSpecRef = specDetail.WorkspacePath
	}
	if req.TrainTemplateRef == "" && specDetail != nil {
		req.TrainTemplateRef = firstNonEmpty(specDetail.Spec.TemplateType, stringValue(specDetail.Spec.SpecJSON["train_template_type"]), "mock_train_template")
	}
	if req.EvalProtocolRef == "" {
		req.EvalProtocolRef = defaultEvalProtocolRef(s.workspaceRoot, expDetail.Experiment.DatasetAssetID)
	}
	return req, expDetail, specDetail, planPath, nil
}

func (s *Service) ensureLatestSpec(ctx context.Context, experimentID string) (*model.ExperimentSpecDetail, error) {
	specDetail, err := s.experiments.GetLatestSpec(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	if specDetail != nil {
		return specDetail, nil
	}
	return s.experiments.GenerateSpec(ctx, experimentID)
}

func (s *Service) resolveIdea(ctx context.Context, expDetail *model.ExperimentDetail, ideaID string) (*model.IdeaDetail, error) {
	if ideaID == "" {
		ideaID = expDetail.Experiment.IdeaID
	}
	if ideaID == "" || s.ideas == nil {
		return nil, nil
	}
	return s.ideas.GetByID(ctx, ideaID)
}

func (s *Service) persistPatchManifest(codingDir string, payload codingPayload) ([]model.CodingPatchManifestItem, error) {
	manifest := payload.CodePatchManifest
	specOverridesPath := filepath.Join(codingDir, "spec_overrides.json")
	if err := writeJSON(specOverridesPath, payload.SpecOverrides); err != nil {
		return nil, err
	}
	for idx := range manifest {
		if strings.EqualFold(manifest[idx].PatchType, "config_file") && strings.TrimSpace(manifest[idx].FilePath) == "" {
			manifest[idx].FilePath = specOverridesPath
		}
	}
	notesPath := filepath.Join(codingDir, "template_patch_notes.md")
	if err := os.WriteFile(notesPath, []byte(ensureTrailingLine(buildPatchNotes(markdownPatchSummaries(manifest)))), 0o644); err != nil {
		return nil, err
	}
	for idx := range manifest {
		if strings.Contains(strings.ToLower(manifest[idx].Target), "template_patch_notes") {
			manifest[idx].FilePath = notesPath
		}
	}
	if err := writeJSON(filepath.Join(codingDir, "patch_manifest.json"), manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func (s *Service) applyPatchedSpec(ctx context.Context, latest model.ExperimentSpecDetail, payload codingPayload, codingDir string) (*model.ExperimentSpec, error) {
	specJSON := cloneInterfaceMap(latest.Spec.SpecJSON)
	if specJSON == nil {
		specJSON = map[string]interface{}{}
	}
	mergeInto(specJSON, cloneInterfaceMap(payload.SpecOverrides))
	if s.templates != nil {
		templateName, err := s.templates.SelectTemplate(specJSON)
		if err != nil {
			return nil, err
		}
		if err = s.templates.ValidateSpec(templateName, specJSON); err != nil {
			return nil, err
		}
	}
	version := latest.Spec.Version + 1
	now := time.Now()
	spec := model.ExperimentSpec{
		ID:            httpx.NewID("espec"),
		ExperimentID:  latest.Spec.ExperimentID,
		SpecJSON:      specJSON,
		TemplateType:  firstNonEmpty(stringValue(specJSON["train_template_type"]), latest.Spec.TemplateType),
		GeneratedFrom: mergeGeneratedFrom(latest.Spec.GeneratedFrom, payload),
		Version:       version,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.specs.Create(ctx, spec); err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(codingDir, "patched_spec.json"), spec.SpecJSON); err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(workspacepkg.New(s.workspaceRoot).ExperimentDir(spec.ExperimentID), "spec.json"), spec.SpecJSON); err != nil {
		return nil, err
	}
	return &spec, nil
}

func mergeGeneratedFrom(existing map[string]interface{}, payload codingPayload) map[string]interface{} {
	out := map[string]interface{}{
		"strategy":             "coding_agent_template_bound_v1",
		"supportsPlannerAgent": true,
		"supportsCodingAgent":  true,
		"codingAgent": map[string]interface{}{
			"patch_count": len(payload.CodePatchManifest),
			"mode":        "template_bound_v1",
		},
	}
	for key, value := range existing {
		out[key] = value
	}
	out["codingAgent"] = map[string]interface{}{
		"patch_count": len(payload.CodePatchManifest),
		"mode":        "template_bound_v1",
	}
	return out
}

type codingRequest struct {
	ExperimentID string
	IdeaID       string
}

type codingPayload struct {
	CodePatchManifest  []model.CodingPatchManifestItem
	SpecOverrides      map[string]any
	ExecutionResultRef string
	MetricsSummary     map[string]any
	EvaluationSummary  string
	TargetRunID        string
}

func requestFromJob(job *model.AgentJob) codingRequest {
	experimentID := stringValue(job.Metadata["experiment_id"])
	ideaID := stringValue(job.Metadata["idea_id"])
	for _, ref := range job.InputRefs {
		switch strings.TrimSpace(ref.RefType) {
		case "experiment":
			if experimentID == "" {
				experimentID = strings.TrimSpace(ref.RefID)
			}
		case "idea":
			if ideaID == "" {
				ideaID = strings.TrimSpace(ref.RefID)
			}
		case "experiment_plan", "experiment_spec":
			if experimentID == "" {
				experimentID = inferExperimentIDFromPath(ref.RefPath)
			}
		}
	}
	return codingRequest{ExperimentID: experimentID, IdeaID: ideaID}
}

func extractCodingPayload(payload map[string]any) codingPayload {
	return codingPayload{
		CodePatchManifest:  extractPatchManifest(payload["code_patch_manifest"]),
		SpecOverrides:      cloneMap(mapValue(payload["spec_overrides"])),
		ExecutionResultRef: stringValue(payload["execution_result_ref"]),
		MetricsSummary:     cloneMap(mapValue(payload["metrics_summary"])),
		EvaluationSummary:  stringValue(payload["evaluation_summary_md"]),
		TargetRunID:        stringValue(payload["run_id"]),
	}
}

func extractPatchManifest(value any) []model.CodingPatchManifestItem {
	switch typed := value.(type) {
	case []model.CodingPatchManifestItem:
		return append([]model.CodingPatchManifestItem(nil), typed...)
	case []any:
		out := make([]model.CodingPatchManifestItem, 0, len(typed))
		for _, item := range typed {
			mapped := mapValue(item)
			out = append(out, model.CodingPatchManifestItem{
				PatchType: stringValue(mapped["patch_type"]),
				Target:    stringValue(mapped["target"]),
				Action:    stringValue(mapped["action"]),
				Value:     mapped["value"],
				FilePath:  stringValue(mapped["file_path"]),
				Summary:   stringValue(mapped["summary"]),
				Metadata:  cloneMap(mapValue(mapped["metadata"])),
			})
		}
		return out
	default:
		return []model.CodingPatchManifestItem{}
	}
}

func updateCodingJobPayload(payload map[string]any, manifest []model.CodingPatchManifestItem, appliedSpec *model.ExperimentSpec, run *model.ExperimentRun, compareResult *model.RunCompareResult, evaluationSummary string) map[string]any {
	out := cloneMap(payload)
	if out == nil {
		out = map[string]any{}
	}
	out["code_patch_manifest"] = manifest
	if appliedSpec != nil {
		out["applied_spec_id"] = appliedSpec.ID
		out["train_template_type"] = appliedSpec.TemplateType
	}
	if run != nil {
		out["run_id"] = run.ID
		out["execution_result_ref"] = firstNonEmpty(readNestedString(run.ResultJSON, "artifacts.result_path"), "run:"+run.ID)
		out["metrics_summary"] = flattenMetricsSummary(run.ResultJSON)
	}
	if strings.TrimSpace(evaluationSummary) != "" {
		out["evaluation_summary_md"] = evaluationSummary
	}
	if compareResult != nil {
		out["comparison_count"] = len(compareResult.Comparisons)
		if compareResult.ResultArchive != nil {
			out["result_archive_id"] = compareResult.ResultArchive.Archive.ID
		}
	}
	return out
}

func flattenMetricsSummary(resultJSON map[string]interface{}) map[string]any {
	metrics := mapValue(resultJSON["metrics"])
	out := map[string]any{}
	if primary := stringValue(metrics["primary_metric"]); primary != "" {
		out["primary_metric"] = primary
	}
	values := mapValue(metrics["values"])
	for key, value := range values {
		out[key] = value
	}
	if len(out) == 0 {
		out["status"] = "missing"
	}
	return out
}

func buildEvaluationSummary(run model.ExperimentRun, compareResult *model.RunCompareResult, manifest []model.CodingPatchManifestItem) string {
	lines := []string{
		"# Coding + Evaluator Summary",
		"",
		"- Run ID: `" + run.ID + "`",
		"- Run Status: `" + run.RunStatus + "`",
		"- Patch Count: " + fmt.Sprintf("%d", len(manifest)),
	}
	metrics := flattenMetricsSummary(run.ResultJSON)
	if primary := stringValue(metrics["primary_metric"]); primary != "" {
		lines = append(lines, "- Primary Metric: `"+primary+"`")
		if value, ok := metrics[primary]; ok {
			lines = append(lines, "- Primary Value: `"+fmt.Sprintf("%v", value)+"`")
		}
	}
	if compareResult != nil {
		lines = append(lines, "- Comparison Count: "+fmt.Sprintf("%d", len(compareResult.Comparisons)))
		lines = append(lines, "- Overall Judgment: `"+compareResult.OverallJudgment+"`")
		if compareResult.ResultArchive != nil {
			lines = append(lines, "- Result Archive: `"+compareResult.ResultArchive.Archive.ID+"`")
		}
	}
	lines = append(lines, "", "The first version stays inside the shared train template so every generated change remains auditable and runnable by the stage2 pipeline.")
	return strings.Join(lines, "\n")
}

func buildPatchNotes(items []string) string {
	lines := []string{
		"# Template Patch Notes",
		"",
		"These changes are constrained to the shared stage2 template surface.",
		"",
	}
	for _, item := range items {
		lines = append(lines, "- "+item)
	}
	return strings.Join(lines, "\n")
}

func markdownPatchSummaries(items []model.CodingPatchManifestItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, firstNonEmpty(item.Summary, item.Target))
	}
	return out
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func readJSON(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]any{}
	}
	var value map[string]any
	if err = json.Unmarshal(data, &value); err != nil {
		return map[string]any{}
	}
	return value
}

func cloneInterfaceMap(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
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

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		rendered := strings.TrimSpace(fmt.Sprintf("%v", value))
		if rendered == "<nil>" {
			return ""
		}
		return rendered
	}
}

func readNestedString(root map[string]interface{}, path string) string {
	current := any(root)
	parts := strings.Split(path, ".")
	for _, part := range parts {
		mapped, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = mapped[part]
	}
	return stringValue(current)
}

func ensureTrailingLine(text string) string {
	if strings.HasSuffix(text, "\n") {
		return text
	}
	return text + "\n"
}

func mergeInto(dst map[string]interface{}, src map[string]interface{}) {
	for key, value := range src {
		if valueMap, ok := value.(map[string]interface{}); ok {
			if current, ok := dst[key].(map[string]interface{}); ok {
				mergeInto(current, valueMap)
				dst[key] = current
				continue
			}
		}
		dst[key] = value
	}
}

func normalizeSpecOverrideMap(input map[string]any) map[string]interface{} {
	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		if nested, ok := value.(map[string]any); ok {
			out[key] = normalizeSpecOverrideMap(nested)
			continue
		}
		out[key] = value
	}
	return out
}

func defaultEvalProtocolRef(workspaceRoot string, datasetAssetID string) string {
	if strings.TrimSpace(datasetAssetID) == "" {
		return ""
	}
	return filepath.Join(workspacepkg.New(workspaceRoot).DatasetAssetDir(datasetAssetID), "evalplan.json")
}

func resolveTemplatePath(templateRoot string, templateRef string) string {
	if strings.TrimSpace(templateRef) == "" {
		return ""
	}
	if filepath.IsAbs(templateRef) {
		return templateRef
	}
	if strings.TrimSpace(templateRoot) == "" {
		return templateRef
	}
	return filepath.Join(templateRoot, templateRef)
}

func inferExperimentIDFromPath(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	for idx, item := range parts {
		if item == "experiments" && idx+1 < len(parts) {
			return strings.TrimSpace(parts[idx+1])
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, item := range values {
		if text := strings.TrimSpace(item); text != "" {
			return text
		}
	}
	return ""
}

func buildIdeaInputRef(workspaceRoot string, ideaID string, detail *model.IdeaDetail) model.AgentInputRef {
	ref := model.AgentInputRef{
		RefType: "idea",
		RefID:   ideaID,
		RefPath: filepath.Join(workspacepkg.New(workspaceRoot).IdeaPool(), ideaID, "structured_idea.json"),
	}
	if detail != nil {
		ref.Metadata = map[string]any{"title": detail.Idea.Title}
	}
	return ref
}

func ideaMetadata(detail *model.IdeaDetail) map[string]any {
	if detail == nil {
		return map[string]any{}
	}
	out := map[string]any{
		"id":             detail.Idea.ID,
		"title":          detail.Idea.Title,
		"description_md": detail.Idea.DescriptionMD,
		"priority":       detail.Idea.Priority,
		"confidence":     detail.Idea.Confidence,
	}
	if detail.StructuredIdea != nil {
		out["research_direction"] = detail.StructuredIdea.ResearchDirection
		out["innovation_type"] = detail.StructuredIdea.InnovationType
		out["expected_advantage"] = detail.StructuredIdea.ExpectedAdvantage
		out["risk_points"] = append([]string{}, detail.StructuredIdea.RiskPoints...)
		out["target_dataset_refs"] = append([]string{}, detail.StructuredIdea.TargetDatasetRefs...)
		out["dataset_eval_protocol_refs"] = append([]string{}, detail.StructuredIdea.DatasetEvalProtocolRefs...)
	}
	return out
}
