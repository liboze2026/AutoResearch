package codingagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	baseservice "mrag-platform/backend/go/internal/service"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

const phase4CodingOutputSchemaRef = "schemas/coding-phase4-output-v1.json"

type phase4CodingJobService interface {
	Create(context.Context, model.AgentJobCreateRequest) (*model.AgentJob, error)
	GetByID(context.Context, string) (*model.AgentJob, error)
}

type phase4CodingArtifactStore interface {
	Create(context.Context, model.AgentArtifact) error
	ListByJobID(context.Context, string) ([]model.AgentArtifact, error)
}

type phase4CodingDataService interface {
	GetDatasetProfileByID(context.Context, string) (*model.Phase4DatasetProfile, error)
	GetReaderContextByID(context.Context, string) (*model.Phase4ReaderContext, error)
	GetIdeaByID(context.Context, string) (*model.Phase4Idea, error)
	UpdateIdeaStatus(context.Context, string, model.Phase4IdeaStatusUpdateRequest) (*model.Phase4Idea, error)
	CreateRunManifest(context.Context, model.Phase4RunManifestCreateRequest) (*model.Phase4RunManifest, error)
	GetRunManifestByID(context.Context, string) (*model.Phase4RunManifest, error)
	UpdateRunManifest(context.Context, string, model.Phase4RunManifestUpdateRequest) (*model.Phase4RunManifest, error)
	UpdateRunManifestStatus(context.Context, string, model.Phase4RunManifestStatusUpdateRequest) (*model.Phase4RunManifest, error)
}

type phase4CodingEventPublisher interface {
	PublishEvent(context.Context, model.AgentEventCreateRequest) (*model.AgentEvent, error)
}

type phase4IdeaRevisionGenerator interface {
	GenerateRevisionCandidates(context.Context, string, model.Phase4IdeaRevisionGenerateRequest) (*model.Phase4IdeaRunResult, error)
}

type Phase4Service struct {
	jobs                 phase4CodingJobService
	jobUpdates           jobUpdater
	triggers             triggerService
	artifacts            phase4CodingArtifactStore
	phase4               phase4CodingDataService
	events               phase4CodingEventPublisher
	ideaRevisions        phase4IdeaRevisionGenerator
	workspaceRoot        string
	pythonExec           string
	pythonRunnersDir     string
	phase4RemoteWorkRoot string
	servers              phase4ServerSecretReader
	gpuSelector          phase4GPUSelector
	remoteExecutor       phase4RemoteExecutor
	commandTimeout       time.Duration
}

func NewPhase4Service(
	jobs phase4CodingJobService,
	jobUpdates jobUpdater,
	triggers triggerService,
	artifacts phase4CodingArtifactStore,
	phase4 phase4CodingDataService,
	events phase4CodingEventPublisher,
	workspaceRoot string,
	pythonExec string,
	pythonRunnersDir string,
	phase4RemoteWorkRoot string,
) *Phase4Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	if strings.TrimSpace(pythonExec) == "" {
		pythonExec = "python"
	}
	if strings.TrimSpace(pythonRunnersDir) == "" {
		pythonRunnersDir = filepath.Join("..", "python_runners")
	}
	if strings.TrimSpace(phase4RemoteWorkRoot) == "" {
		phase4RemoteWorkRoot = "/home/bzli/mrag"
	}
	return &Phase4Service{
		jobs:                 jobs,
		jobUpdates:           jobUpdates,
		triggers:             triggers,
		artifacts:            artifacts,
		phase4:               phase4,
		events:               events,
		workspaceRoot:        workspaceRoot,
		pythonExec:           pythonExec,
		pythonRunnersDir:     pythonRunnersDir,
		phase4RemoteWorkRoot: phase4RemoteWorkRoot,
		commandTimeout:       20 * time.Second,
	}
}

func (s *Phase4Service) ConfigureShenzhenlabExecution(servers phase4ServerSecretReader, gpuProber phase4GPUProbeSource, ssh baseservice.SSHGateway, commandTimeoutSec int) *Phase4Service {
	s.servers = servers
	s.gpuSelector = newPhase4ProbeGPUSelector(gpuProber)
	s.remoteExecutor = newPhase4ShenzhenlabRemoteExecutor(ssh)
	if commandTimeoutSec > 0 {
		s.commandTimeout = time.Duration(commandTimeoutSec) * time.Second
	}
	return s
}

func (s *Phase4Service) AttachIdeaRevisionGenerator(generator phase4IdeaRevisionGenerator) *Phase4Service {
	s.ideaRevisions = generator
	return s
}

func (s *Phase4Service) Run(ctx context.Context, req model.Phase4CodingRunRequest) (*model.Phase4CodingRunResult, error) {
	req, datasetProfile, idea, readerContext, err := s.normalizeRunRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	runManifest, err := s.phase4.CreateRunManifest(ctx, model.Phase4RunManifestCreateRequest{
		DatasetProfileID: datasetProfile.ID,
		IdeaID:           idea.ID,
		ReaderContextID:  readerContext.ID,
		RunnerMode:       req.RunnerMode,
		ServerID:         req.ServerID,
		GPU:              req.GPU,
		Status:           model.Phase4RunStatusDraft,
		RetryCount:       0,
		MaxRetryCount:    req.MaxRetryCount,
		ArtifactPaths: map[string]any{
			"phase4_remote_work_root": s.phase4RemoteWorkRoot,
		},
	})
	if err != nil {
		return nil, err
	}

	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:       "coding_phase4",
		ExecutionMode:   req.ExecutionMode,
		ModelProvider:   req.ModelProvider,
		ModelName:       req.ModelName,
		PromptVersion:   req.PromptVersion,
		InputRefs:       buildPhase4CodingInputRefs(*datasetProfile, *idea, *readerContext, *runManifest),
		OutputSchemaRef: phase4CodingOutputSchemaRef,
		SkillRefs:       req.SkillRefs,
		ToolRefs:        appendPhase4ToolRefs(req.ToolRefs),
		MemoryRefs:      req.MemoryRefs,
		Metadata: map[string]any{
			"run_manifest_id":    runManifest.ID,
			"dataset_profile_id": datasetProfile.ID,
			"idea_id":            idea.ID,
			"reader_context_id":  readerContext.ID,
			"runner_mode":        req.RunnerMode,
			"max_retry_count":    req.MaxRetryCount,
			"user_notes":         req.UserNotes,
			"dataset_profile":    phase4CodingDatasetProfileMetadata(*datasetProfile),
			"idea":               phase4CodingIdeaMetadata(*idea),
			"reader_context":     phase4CodingReaderContextMetadata(*readerContext),
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}
	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata: map[string]any{
			"agent_type": "coding_phase4",
		},
	})
	if err != nil {
		return nil, err
	}
	return s.resultFromJob(ctx, job)
}

func (s *Phase4Service) GetJob(ctx context.Context, jobID string) (*model.Phase4CodingJobDetail, error) {
	job, err := s.jobs.GetByID(ctx, strings.TrimSpace(jobID))
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, nil
	}
	artifacts, err := s.artifacts.ListByJobID(ctx, job.ID)
	if err != nil {
		return nil, err
	}
	result, err := s.resultFromJob(ctx, job)
	if err != nil {
		return nil, err
	}
	return &model.Phase4CodingJobDetail{
		Job:         job,
		Artifacts:   artifacts,
		RunManifest: result.RunManifest,
		Warnings:    result.Warnings,
	}, nil
}

func (s *Phase4Service) resultFromJob(ctx context.Context, job *model.AgentJob) (*model.Phase4CodingRunResult, error) {
	if job == nil {
		return nil, fmt.Errorf("phase4 coding job not found")
	}
	runManifestID := firstNonEmpty(strings.TrimSpace(stringValue(job.NormalizedPayload["run_manifest_id"])), strings.TrimSpace(stringValue(job.Metadata["run_manifest_id"])))
	var runManifest *model.Phase4RunManifest
	if runManifestID != "" {
		item, err := s.phase4.GetRunManifestByID(ctx, runManifestID)
		if err != nil {
			return nil, err
		}
		runManifest = item
	}
	return &model.Phase4CodingRunResult{
		Job:         job,
		RunManifest: runManifest,
		Warnings:    append([]string{}, job.Warnings...),
	}, nil
}

func (s *Phase4Service) PostProcess(ctx context.Context, job *model.AgentJob) error {
	if job == nil {
		return nil
	}
	payload, err := decodePhase4CodingRuntimePayload(job.NormalizedPayload)
	if err != nil {
		return err
	}
	runManifestID := firstNonEmpty(strings.TrimSpace(stringValue(job.Metadata["run_manifest_id"])), strings.TrimSpace(stringValue(job.NormalizedPayload["run_manifest_id"])))
	if runManifestID == "" {
		return fmt.Errorf("phase4 coding run manifest id is required")
	}
	runManifest, err := s.phase4.GetRunManifestByID(ctx, runManifestID)
	if err != nil {
		return err
	}
	if runManifest == nil {
		return fmt.Errorf("phase4 run manifest not found")
	}
	datasetProfile, err := s.phase4.GetDatasetProfileByID(ctx, runManifest.DatasetProfileID)
	if err != nil {
		return err
	}
	if datasetProfile == nil {
		return fmt.Errorf("phase4 dataset profile not found")
	}
	idea, err := s.phase4.GetIdeaByID(ctx, runManifest.IdeaID)
	if err != nil {
		return err
	}
	if idea == nil {
		return fmt.Errorf("phase4 idea not found")
	}
	readerContext, err := s.phase4.GetReaderContextByID(ctx, runManifest.ReaderContextID)
	if err != nil {
		return err
	}
	if readerContext == nil {
		return fmt.Errorf("phase4 reader context not found")
	}

	if _, err = s.phase4.UpdateRunManifestStatus(ctx, runManifest.ID, model.Phase4RunManifestStatusUpdateRequest{
		Status: model.Phase4RunStatusQueued,
	}); err != nil {
		return err
	}

	artifactPaths, snapshotID, err := s.preparePhase4Run(ctx, runManifest, datasetProfile, idea, readerContext, payload)
	if err != nil {
		_, _ = s.phase4.UpdateRunManifestStatus(ctx, runManifest.ID, model.Phase4RunManifestStatusUpdateRequest{
			Status:          model.Phase4RunStatusFailed,
			FailureFeedback: map[string]any{"stage": "prepare_phase4_run", "error": err.Error()},
		})
		return err
	}
	if _, err = s.phase4.UpdateRunManifest(ctx, runManifest.ID, model.Phase4RunManifestUpdateRequest{
		CodeSnapshotID: &snapshotID,
		ArtifactPaths:  &artifactPaths,
		LogsPath:       stringPtr(stringValue(artifactPaths["logs_dir"])),
		MetricsPath:    stringPtr(stringValue(artifactPaths["metrics_path"])),
	}); err != nil {
		return err
	}
	return s.executePhase4ManagedRun(ctx, job, runManifest, datasetProfile, idea, readerContext, artifactPaths, snapshotID, payload)
}

func (s *Phase4Service) normalizeRunRequest(ctx context.Context, req model.Phase4CodingRunRequest) (model.Phase4CodingRunRequest, *model.Phase4DatasetProfile, *model.Phase4Idea, *model.Phase4ReaderContext, error) {
	req.DatasetProfileID = strings.TrimSpace(req.DatasetProfileID)
	req.IdeaID = strings.TrimSpace(req.IdeaID)
	req.ReaderContextID = strings.TrimSpace(req.ReaderContextID)
	req.RunnerMode = strings.TrimSpace(strings.ToLower(req.RunnerMode))
	req.ServerID = strings.TrimSpace(req.ServerID)
	req.GPU = strings.TrimSpace(req.GPU)
	req.UserNotes = strings.TrimSpace(req.UserNotes)
	req.ExecutionMode = strings.TrimSpace(strings.ToLower(req.ExecutionMode))
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	if req.DatasetProfileID == "" {
		return req, nil, nil, nil, fmt.Errorf("datasetProfileId is required")
	}
	if req.IdeaID == "" {
		return req, nil, nil, nil, fmt.Errorf("ideaId is required")
	}
	datasetProfile, err := s.phase4.GetDatasetProfileByID(ctx, req.DatasetProfileID)
	if err != nil {
		return req, nil, nil, nil, err
	}
	if datasetProfile == nil {
		return req, nil, nil, nil, fmt.Errorf("phase4 dataset profile not found")
	}
	idea, err := s.phase4.GetIdeaByID(ctx, req.IdeaID)
	if err != nil {
		return req, nil, nil, nil, err
	}
	if idea == nil {
		return req, nil, nil, nil, fmt.Errorf("phase4 idea not found")
	}
	if req.ReaderContextID == "" {
		req.ReaderContextID = strings.TrimSpace(idea.ReaderContextID)
	}
	if req.ReaderContextID == "" {
		return req, nil, nil, nil, fmt.Errorf("readerContextId is required")
	}
	readerContext, err := s.phase4.GetReaderContextByID(ctx, req.ReaderContextID)
	if err != nil {
		return req, nil, nil, nil, err
	}
	if readerContext == nil {
		return req, nil, nil, nil, fmt.Errorf("phase4 reader context not found")
	}
	if req.MaxRetryCount <= 0 {
		req.MaxRetryCount = 3
	}
	switch req.ExecutionMode {
	case "":
		req.ExecutionMode = "mock"
	case "mock", "api", "codex_cli":
	default:
		return req, nil, nil, nil, fmt.Errorf("executionMode must be one of mock, api, codex_cli")
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "phase4_coding"
	}
	if req.ModelName == "" {
		req.ModelName = "coding-phase4-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	switch req.RunnerMode {
	case "":
		if req.ServerID != "" {
			req.RunnerMode = phase4ShenzhenlabServerName
		} else {
			req.RunnerMode = "local_dummy"
		}
	case "local_dummy", phase4ShenzhenlabServerName:
	default:
		return req, nil, nil, nil, fmt.Errorf("runnerMode must be one of local_dummy, %s", phase4ShenzhenlabServerName)
	}
	return req, datasetProfile, idea, readerContext, nil
}

func decodePhase4CodingRuntimePayload(payload map[string]any) (model.Phase4CodingRuntimePayload, error) {
	raw, err := json.Marshal(phase4CodingEnsureMap(payload))
	if err != nil {
		return model.Phase4CodingRuntimePayload{}, err
	}
	var out model.Phase4CodingRuntimePayload
	if err = json.Unmarshal(raw, &out); err != nil {
		return model.Phase4CodingRuntimePayload{}, err
	}
	if out.Phase4RunManifest == nil {
		out.Phase4RunManifest = map[string]any{}
	}
	if out.Phase4Config == nil {
		out.Phase4Config = map[string]any{}
	}
	if out.RetryPlan == nil {
		out.RetryPlan = map[string]any{}
	}
	if out.DatasetToolAssets == nil {
		out.DatasetToolAssets = map[string]any{}
	}
	if out.EvaluateToolAssets == nil {
		out.EvaluateToolAssets = map[string]any{}
	}
	if out.Entrypoints == nil {
		out.Entrypoints = map[string]any{}
	}
	if out.Data == nil {
		out.Data = map[string]any{}
	}
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	return out, nil
}

func buildPhase4CodingInputRefs(datasetProfile model.Phase4DatasetProfile, idea model.Phase4Idea, readerContext model.Phase4ReaderContext, runManifest model.Phase4RunManifest) []model.AgentInputRef {
	return []model.AgentInputRef{
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
			RefType: "idea",
			RefID:   idea.ID,
			Metadata: map[string]any{
				"title": idea.Title,
			},
		},
		{
			RefType: "reader_context",
			RefID:   readerContext.ID,
			Metadata: map[string]any{
				"title": readerContext.Title,
			},
		},
		{
			RefType: "phase4_run_manifest",
			RefID:   runManifest.ID,
			Metadata: map[string]any{
				"runner_mode": runManifest.RunnerMode,
			},
		},
	}
}

func appendPhase4ToolRefs(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items)+2)
	for _, item := range append(append([]string{}, items...), "dataset_tool", "evaluate_tool") {
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

func phase4CodingDatasetProfileMetadata(item model.Phase4DatasetProfile) map[string]any {
	return map[string]any{
		"id":                    item.ID,
		"datasetName":           item.DatasetName,
		"taskType":              item.TaskType,
		"serverPath":            item.ServerPath,
		"officialMetric":        item.OfficialMetric,
		"splits":                append([]model.Phase4DatasetSplit{}, item.Splits...),
		"knownDifficulties":     append([]string{}, item.KnownDifficulties...),
		"fileStructureSnapshot": phase4CodingEnsureMap(item.FileStructureSnapshot),
		"sampleStatistics":      phase4CodingEnsureMap(item.SampleStatistics),
		"userNotes":             item.UserNotes,
	}
}

func phase4CodingIdeaMetadata(item model.Phase4Idea) map[string]any {
	return map[string]any{
		"id":                  item.ID,
		"title":               item.Title,
		"problemDefinition":   item.ProblemDefinition,
		"coreMethod":          item.CoreMethod,
		"trainingPlan":        item.TrainingPlan,
		"dataProcessingNeeds": append([]string{}, item.DataProcessingNeeds...),
		"modelChanges":        append([]string{}, item.ModelChanges...),
		"evaluationMetrics":   append([]string{}, item.EvaluationMetrics...),
		"riskPoints":          append([]string{}, item.RiskPoints...),
	}
}

func phase4CodingReaderContextMetadata(item model.Phase4ReaderContext) map[string]any {
	out := phase4CodingEnsureMap(item.StructuredContext)
	out["task_definition"] = firstNonEmpty(stringValue(out["task_definition"]), item.TaskDefinition)
	out["implementation_constraints"] = append([]string{}, phase4CodingStringSlice(out["implementation_constraints"])...)
	out["likely_strong_baselines"] = append([]string{}, phase4CodingStringSlice(out["likely_strong_baselines"])...)
	return out
}

func phase4CodingEnsureMap(value any) map[string]any {
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

func phase4CodingStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text != "" && text != "<nil>" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "" || text == "<nil>" {
			return []string{}
		}
		return []string{text}
	}
}

func (s *Phase4Service) preparePhase4Run(ctx context.Context, runManifest *model.Phase4RunManifest, datasetProfile *model.Phase4DatasetProfile, idea *model.Phase4Idea, readerContext *model.Phase4ReaderContext, payload model.Phase4CodingRuntimePayload) (map[string]any, string, error) {
	paths := workspacepkg.New(s.workspaceRoot)
	runDir := paths.Phase4RunDir(runManifest.ID)
	artifactDir := paths.Phase4ArtifactDir(runManifest.ID)
	logsDir := filepath.Join(runDir, "logs")
	snapshotDir := filepath.Join(runDir, "snapshot")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return nil, "", err
	}

	sourceRoot, err := filepath.Abs(filepath.Join(s.pythonRunnersDir, "retrieval_mainline"))
	if err != nil {
		return nil, "", err
	}
	if err = copyDirectory(sourceRoot, snapshotDir); err != nil {
		return nil, "", err
	}

	methodModulePath, err := s.materializeMethodModule(snapshotDir, payload.MethodModule)
	if err != nil {
		return nil, "", err
	}
	configPath := filepath.Join(runDir, "config.json")
	configPayload := phase4CodingEnsureMap(payload.Phase4Config)
	configPayload["method_module_path"] = filepath.ToSlash(relPathWithin(snapshotDir, methodModulePath))
	if _, ok := configPayload["protocol_version"]; !ok {
		configPayload["protocol_version"] = "phase4-retrieval-mainline-v1"
	}
	if _, ok := configPayload["runner_mode"]; !ok {
		configPayload["runner_mode"] = runManifest.RunnerMode
	}
	if err = writeJSON(configPath, configPayload); err != nil {
		return nil, "", err
	}

	metricsPath := filepath.Join(artifactDir, "metrics.json")
	manifestPath := filepath.Join(runDir, "experiment_manifest.json")
	machineReportPath := filepath.Join(artifactDir, "machine_report.json")
	humanReportPath := filepath.Join(artifactDir, "report.md")
	datasetToolAssetPath := filepath.Join(artifactDir, "dataset_tool_asset.json")
	datasetAdapterPath := filepath.Join(artifactDir, "dataset_adapter_contract.json")
	evaluateToolAssetPath := filepath.Join(artifactDir, "evaluate_tool_asset.json")
	evalSummaryPath := filepath.Join(artifactDir, "eval_summary.md")
	predictionsPath := filepath.Join(artifactDir, "predictions.json")
	bootstrapPath := filepath.Join(snapshotDir, "bootstrap_env.sh")
	driverLogPath := filepath.Join(logsDir, "driver.log")
	snapshotManifestPath := filepath.Join(artifactDir, "snapshot_manifest.json")

	manifestPayload := map[string]any{
		"protocol_version":         "phase4-retrieval-mainline-v1",
		"run_id":                   runManifest.ID,
		"dataset_profile_id":       runManifest.DatasetProfileID,
		"idea_id":                  runManifest.IdeaID,
		"reader_context_id":        runManifest.ReaderContextID,
		"run_dir":                  runDir,
		"snapshot_dir":             snapshotDir,
		"artifact_dir":             artifactDir,
		"logs_dir":                 logsDir,
		"config_path":              configPath,
		"metrics_path":             metricsPath,
		"predictions_path":         predictionsPath,
		"machine_report_path":      machineReportPath,
		"human_report_path":        humanReportPath,
		"dataset_tool_asset_path":  datasetToolAssetPath,
		"dataset_adapter_path":     datasetAdapterPath,
		"evaluate_tool_asset_path": evaluateToolAssetPath,
		"eval_summary_path":        evalSummaryPath,
		"bootstrap_script_path":    bootstrapPath,
		"metadata": map[string]any{
			"dataset_name":            datasetProfile.DatasetName,
			"idea_title":              idea.Title,
			"reader_context_title":    readerContext.Title,
			"phase4_remote_work_root": s.phase4RemoteWorkRoot,
		},
	}
	if err = writeJSON(manifestPath, manifestPayload); err != nil {
		return nil, "", err
	}

	snapshotFiles, err := listFiles(snapshotDir)
	if err != nil {
		return nil, "", err
	}
	if err = writeJSON(snapshotManifestPath, map[string]any{
		"source_root":      sourceRoot,
		"snapshot_dir":     snapshotDir,
		"copied_files":     snapshotFiles,
		"file_count":       len(snapshotFiles),
		"method_module":    filepath.ToSlash(relPathWithin(snapshotDir, methodModulePath)),
		"protocol_version": "phase4-retrieval-mainline-v1",
	}); err != nil {
		return nil, "", err
	}

	artifactPaths := map[string]any{
		"run_dir":                  runDir,
		"artifact_dir":             artifactDir,
		"snapshot_dir":             snapshotDir,
		"source_root":              sourceRoot,
		"manifest_path":            manifestPath,
		"config_path":              configPath,
		"method_module_path":       methodModulePath,
		"metrics_path":             metricsPath,
		"machine_report_path":      machineReportPath,
		"human_report_path":        humanReportPath,
		"dataset_tool_asset_path":  datasetToolAssetPath,
		"dataset_adapter_path":     datasetAdapterPath,
		"evaluate_tool_asset_path": evaluateToolAssetPath,
		"eval_summary_path":        evalSummaryPath,
		"predictions_path":         predictionsPath,
		"logs_dir":                 logsDir,
		"driver_log_path":          driverLogPath,
		"repair_log_path":          filepath.Join(logsDir, "repair.jsonl"),
		"snapshot_manifest_path":   snapshotManifestPath,
		"bootstrap_script_path":    bootstrapPath,
		"failure_feedback_path":    filepath.Join(artifactDir, "failure_feedback.json"),
		"phase4_remote_work_root":  s.phase4RemoteWorkRoot,
	}
	return artifactPaths, httpx.NewID("p4snap"), nil
}

func (s *Phase4Service) materializeMethodModule(snapshotDir string, module model.Phase4CodingMethodModule) (string, error) {
	relativePath := strings.TrimSpace(module.RelativePath)
	if relativePath == "" {
		relativePath = filepath.ToSlash(filepath.Join("methods", "generated", "generated_method.py"))
	}
	methodPath := filepath.Join(snapshotDir, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(methodPath), 0o755); err != nil {
		return "", err
	}
	content := strings.TrimSpace(module.Content)
	if content == "" {
		content = "from methods.dummy_retrieval import DummyRetrievalMethod\n\n\ndef build_method():\n    return DummyRetrievalMethod(name='generated_method', method_tags=['fallback'], score_bias=0.0)\n"
	}
	if err := os.WriteFile(methodPath, []byte(content), 0o644); err != nil {
		return "", err
	}
	return methodPath, nil
}

func copyDirectory(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, relative)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if err = os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err = io.Copy(out, in); err != nil {
			_ = out.Close()
			return err
		}
		if err = out.Close(); err != nil {
			return err
		}
		return os.Chmod(target, info.Mode())
	})
}

func listFiles(root string) ([]string, error) {
	items := make([]string, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		items = append(items, filepath.ToSlash(relative))
		return nil
	})
	return items, err
}

func relPathWithin(root string, target string) string {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return target
	}
	return relative
}

func (s *Phase4Service) executeEntrypoint(ctx context.Context, snapshotDir string, entrypoint string, manifestPath string) (string, error) {
	cmd := exec.CommandContext(ctx, s.pythonExec, filepath.Join(snapshotDir, entrypoint), "--manifest", manifestPath)
	cmd.Dir = snapshotDir
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func appendPhase4DriverLog(path string, stage string, output string, execErr error) {
	lines := []string{"[" + stage + "]"}
	if strings.TrimSpace(output) != "" {
		lines = append(lines, output)
	}
	if execErr != nil {
		lines = append(lines, "error="+execErr.Error())
	}
	lines = append(lines, "")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(strings.Join(lines, "\n"))
}

func (s *Phase4Service) failPhase4Run(ctx context.Context, job *model.AgentJob, runManifestID string, artifactPaths map[string]any, stage string, execErr error) error {
	finishedAt := time.Now()
	feedback := map[string]any{
		"stage": stage,
		"error": execErr.Error(),
	}
	if _, err := s.phase4.UpdateRunManifest(ctx, runManifestID, model.Phase4RunManifestUpdateRequest{
		ArtifactPaths:   &artifactPaths,
		LogsPath:        stringPtr(stringValue(artifactPaths["logs_dir"])),
		MetricsPath:     stringPtr(stringValue(artifactPaths["metrics_path"])),
		FailureFeedback: &feedback,
	}); err != nil {
		return err
	}
	if _, err := s.phase4.UpdateRunManifestStatus(ctx, runManifestID, model.Phase4RunManifestStatusUpdateRequest{
		Status:          model.Phase4RunStatusTestFailed,
		FailureFeedback: feedback,
		FinishedAt:      &finishedAt,
	}); err != nil {
		return err
	}
	job.NormalizedPayload = updatePhase4CodingJobPayload(job.NormalizedPayload, runManifestID, stringValue(job.NormalizedPayload["code_snapshot_id"]), artifactPaths, map[string]any{"status": "failed"}, "")
	job.Warnings = append(job.Warnings, fmt.Sprintf("phase4 coding failed during %s: %s", stage, execErr.Error()))
	job.UpdatedAt = time.Now()
	if err := s.jobUpdates.Update(ctx, *job); err != nil {
		return err
	}
	_ = s.persistPhase4Artifacts(ctx, job.ID, artifactPaths)
	return execErr
}

func updatePhase4CodingJobPayload(payload map[string]any, runManifestID string, snapshotID string, artifactPaths map[string]any, metricsSummary map[string]any, executionResultRef string) map[string]any {
	out := phase4CodingEnsureMap(payload)
	out["run_manifest_id"] = runManifestID
	out["code_snapshot_id"] = snapshotID
	out["artifact_paths"] = artifactPaths
	out["metrics_summary"] = phase4CodingEnsureMap(metricsSummary)
	out["execution_result_ref"] = executionResultRef
	return out
}

func (s *Phase4Service) persistPhase4Artifacts(ctx context.Context, jobID string, artifactPaths map[string]any) error {
	artifacts := []struct {
		kind string
		path string
	}{
		{"phase4_manifest", stringValue(artifactPaths["manifest_path"])},
		{"phase4_config", stringValue(artifactPaths["config_path"])},
		{"phase4_remote_manifest", stringValue(artifactPaths["remote_manifest_source_path"])},
		{"phase4_remote_config", stringValue(artifactPaths["remote_config_source_path"])},
		{"phase4_method_module", stringValue(artifactPaths["method_module_path"])},
		{"phase4_metrics", stringValue(artifactPaths["metrics_path"])},
		{"phase4_machine_report", stringValue(artifactPaths["machine_report_path"])},
		{"phase4_human_report", stringValue(artifactPaths["human_report_path"])},
		{"phase4_dataset_tool_asset", stringValue(artifactPaths["dataset_tool_asset_path"])},
		{"phase4_dataset_adapter", stringValue(artifactPaths["dataset_adapter_path"])},
		{"phase4_evaluate_tool_asset", stringValue(artifactPaths["evaluate_tool_asset_path"])},
		{"phase4_eval_summary", stringValue(artifactPaths["eval_summary_path"])},
		{"phase4_predictions", stringValue(artifactPaths["predictions_path"])},
		{"phase4_driver_log", stringValue(artifactPaths["driver_log_path"])},
		{"phase4_repair_log", stringValue(artifactPaths["repair_log_path"])},
		{"phase4_run_log", stringValue(artifactPaths["run_log_path"])},
		{"phase4_bootstrap_stdout", stringValue(artifactPaths["bootstrap_stdout_path"])},
		{"phase4_bootstrap_stderr", stringValue(artifactPaths["bootstrap_stderr_path"])},
		{"phase4_runtime_stdout", stringValue(artifactPaths["runtime_stdout_path"])},
		{"phase4_runtime_stderr", stringValue(artifactPaths["runtime_stderr_path"])},
		{"phase4_failure_feedback", stringValue(artifactPaths["failure_feedback_path"])},
		{"phase4_remote_bootstrap_script", stringValue(artifactPaths["remote_bootstrap_source_path"])},
		{"phase4_remote_execute_script", stringValue(artifactPaths["remote_execute_source_path"])},
		{"phase4_snapshot_manifest", stringValue(artifactPaths["snapshot_manifest_path"])},
	}
	for _, item := range artifacts {
		if strings.TrimSpace(item.path) == "" {
			continue
		}
		if _, err := os.Stat(item.path); err != nil {
			continue
		}
		now := time.Now()
		if err := s.artifacts.Create(ctx, model.AgentArtifact{
			ID:           httpx.NewID("aart"),
			JobID:        jobID,
			ArtifactType: item.kind,
			Name:         filepath.Base(item.path),
			FilePath:     item.path,
			Checksum:     checksumPhase4File(item.path),
			MetadataJSON: map[string]any{"source": "phase4_coding_postprocess"},
			CreatedAt:    now,
			UpdatedAt:    now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func checksumPhase4File(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func stringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func (s *Phase4Service) publishPhase4RunReady(ctx context.Context, runManifest model.Phase4RunManifest, datasetProfile model.Phase4DatasetProfile, idea model.Phase4Idea, readerContext model.Phase4ReaderContext, artifactPaths map[string]any, metrics map[string]any) error {
	if s.events == nil {
		return nil
	}
	_, err := s.events.PublishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "phase4_run_ready",
		SourceRef: "phase4_run:" + runManifest.ID,
		InputRefs: []model.AgentInputRef{
			{RefType: "phase4_run_manifest", RefID: runManifest.ID},
			{RefType: "dataset_profile", RefID: datasetProfile.ID, RefPath: datasetProfile.ServerPath},
			{RefType: "idea", RefID: idea.ID},
			{RefType: "reader_context", RefID: readerContext.ID},
		},
		Payload: map[string]any{
			"run_manifest_id":    runManifest.ID,
			"dataset_profile_id": datasetProfile.ID,
			"idea_id":            idea.ID,
			"reader_context_id":  readerContext.ID,
			"artifact_paths":     artifactPaths,
			"metrics":            metrics,
		},
	})
	return err
}
