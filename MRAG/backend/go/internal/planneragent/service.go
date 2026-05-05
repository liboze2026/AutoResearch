package planneragent

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

const plannerOutputSchemaRef = "schemas/planner-output-v1.json"

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
	Create(context.Context, model.ExperimentCreateRequest) (*model.ExperimentDetail, error)
	GetByID(context.Context, string) (*model.ExperimentDetail, error)
}

type ideaReader interface {
	GetByID(context.Context, string) (*model.IdeaDetail, error)
}

type datasetAssetReader interface {
	GetByID(context.Context, string) (*model.DatasetAssetDetail, error)
}

type baselineReader interface {
	GetByID(context.Context, string) (*model.BaselineDetail, error)
}

type serverLister interface {
	List(context.Context) ([]model.Server, error)
}

type heartbeatReader interface {
	ListByServerID(context.Context, string, int) ([]model.ServerHeartbeat, error)
}

type gpuSnapshotReader interface {
	ListByServerID(context.Context, string, int) ([]model.GPUResourceSnapshot, error)
}

type eventPublisher interface {
	PublishEvent(context.Context, model.AgentEventCreateRequest) (*model.AgentEvent, error)
}

type Service struct {
	jobs          jobCreator
	jobUpdates    jobUpdater
	triggers      triggerService
	experiments   experimentManager
	ideas         ideaReader
	datasets      datasetAssetReader
	baselines     baselineReader
	servers       serverLister
	heartbeats    heartbeatReader
	gpuSnapshots  gpuSnapshotReader
	events        eventPublisher
	workspaceRoot string
}

func NewService(
	jobs jobCreator,
	jobUpdates jobUpdater,
	triggers triggerService,
	experiments experimentManager,
	ideas ideaReader,
	datasets datasetAssetReader,
	baselines baselineReader,
	servers serverLister,
	heartbeats heartbeatReader,
	gpuSnapshots gpuSnapshotReader,
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
		experiments:   experiments,
		ideas:         ideas,
		datasets:      datasets,
		baselines:     baselines,
		servers:       servers,
		heartbeats:    heartbeats,
		gpuSnapshots:  gpuSnapshots,
		events:        events,
		workspaceRoot: workspaceRoot,
	}
}

func (s *Service) Run(ctx context.Context, req model.PlannerRunRequest) (*model.PlannerRunResult, error) {
	normalizedReq, ideaDetail, err := s.normalizeRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ideaRef := buildIdeaInputRef(s.workspaceRoot, normalizedReq.IdeaID, ideaDetail)
	datasets, datasetRefs, err := s.resolveDatasets(ctx, normalizedReq.DatasetAssetRefs)
	if err != nil {
		return nil, err
	}
	evalPlans, evalRefs := resolveEvalPlans(datasets, normalizedReq.EvalProtocolRefs)
	baselines, baselineRefs, err := s.resolveBaselines(ctx, normalizedReq.BaselineRefs)
	if err != nil {
		return nil, err
	}
	serverSnapshots, serverRefs, err := s.resolveServerSnapshots(ctx, normalizedReq.ServerResourceSnapshotRefs)
	if err != nil {
		return nil, err
	}
	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:       "planner",
		ExecutionMode:   normalizedReq.ExecutionMode,
		ModelProvider:   normalizedReq.ModelProvider,
		ModelName:       normalizedReq.ModelName,
		PromptVersion:   normalizedReq.PromptVersion,
		InputRefs:       append(append(append(append([]model.AgentInputRef{ideaRef}, datasetRefs...), evalRefs...), serverRefs...), baselineRefs...),
		OutputSchemaRef: plannerOutputSchemaRef,
		SkillRefs:       normalizedReq.SkillRefs,
		ToolRefs:        normalizedReq.ToolRefs,
		MemoryRefs:      normalizedReq.MemoryRefs,
		Metadata: map[string]any{
			"idea_id":                       normalizedReq.IdeaID,
			"dataset_asset_refs":            normalizedReq.DatasetAssetRefs,
			"eval_protocol_refs":            normalizedReq.EvalProtocolRefs,
			"server_resource_snapshot_refs": normalizedReq.ServerResourceSnapshotRefs,
			"baseline_refs":                 normalizedReq.BaselineRefs,
			"human_hints":                   normalizedReq.HumanHints,
			"idea":                          ideaMetadata(ideaDetail),
			"dataset_assets":                datasets,
			"eval_plans":                    evalPlans,
			"baselines":                     baselines,
			"server_resource_snapshots":     serverSnapshots,
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}
	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata:    map[string]any{"agent_type": "planner"},
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
	payload := extractPlannerPayload(job.NormalizedPayload)
	datasetAssetID := firstNonEmpty(payload.DatasetAssetID, firstString(req.DatasetAssetRefs), firstString(payload.TargetDatasetRefs))
	if datasetAssetID == "" {
		return fmt.Errorf("planner postprocess requires dataset asset")
	}
	priority := payload.Priority
	if priority <= 0 {
		priority = intValue(mapValue(job.Metadata["idea"])["priority"])
	}
	experimentDetail, err := s.experiments.Create(ctx, model.ExperimentCreateRequest{
		DatasetAssetID: datasetAssetID,
		IdeaID:         req.IdeaID,
		BaselineID:     firstNonEmpty(payload.BaselineID, firstString(req.BaselineRefs)),
		Title:          buildExperimentTitle(req.IdeaTitle, datasetAssetID),
		Priority:       priority,
		SummaryMD:      buildExperimentSummary(payload),
		OwnerNoteMD:    buildExperimentOwnerNote(payload),
	})
	if err != nil {
		return err
	}
	planDoc, err := s.writePlan(experimentDetail.Experiment.ID, job, req, payload)
	if err != nil {
		return err
	}
	job.NormalizedPayload = updateJobPayload(job.NormalizedPayload, experimentDetail, planDoc)
	job.UpdatedAt = time.Now()
	if err = s.jobUpdates.Update(ctx, *job); err != nil {
		return err
	}
	return s.publishPlanReady(ctx, experimentDetail, planDoc)
}

func (s *Service) GetPlan(ctx context.Context, experimentID string) (*model.ExperimentPlanResponse, error) {
	if strings.TrimSpace(experimentID) == "" {
		return nil, fmt.Errorf("experiment id is required")
	}
	detail, err := s.experiments.GetByID(ctx, strings.TrimSpace(experimentID))
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, nil
	}
	plan, err := s.readPlan(strings.TrimSpace(experimentID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return &model.ExperimentPlanResponse{Experiment: detail, Plan: *plan}, nil
}

func (s *Service) resultFromJob(ctx context.Context, job *model.AgentJob) (*model.PlannerRunResult, error) {
	if job == nil {
		return nil, fmt.Errorf("planner job not found")
	}
	result := &model.PlannerRunResult{Job: job, Warnings: append([]string{}, job.Warnings...)}
	experimentID := stringValue(job.NormalizedPayload["experiment_id"])
	if experimentID == "" {
		return result, nil
	}
	detail, err := s.experiments.GetByID(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	result.Experiment = detail
	planResp, err := s.GetPlan(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	if planResp != nil {
		result.Plan = &planResp.Plan
	}
	return result, nil
}

func (s *Service) normalizeRequest(ctx context.Context, req model.PlannerRunRequest) (model.PlannerRunRequest, *model.IdeaDetail, error) {
	req.IdeaID = strings.TrimSpace(req.IdeaID)
	req.DatasetAssetRefs = normalizeStringSlice(req.DatasetAssetRefs)
	req.EvalProtocolRefs = normalizeStringSlice(req.EvalProtocolRefs)
	req.ServerResourceSnapshotRefs = normalizeStringSlice(req.ServerResourceSnapshotRefs)
	req.BaselineRefs = normalizeStringSlice(req.BaselineRefs)
	req.HumanHints = normalizeStringSlice(req.HumanHints)
	req.ExecutionMode = strings.TrimSpace(req.ExecutionMode)
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	if req.IdeaID == "" {
		return req, nil, fmt.Errorf("idea_id is required")
	}
	ideaDetail, err := s.ideas.GetByID(ctx, req.IdeaID)
	if err != nil {
		return req, nil, err
	}
	if ideaDetail == nil {
		return req, nil, fmt.Errorf("idea not found")
	}
	switch req.ExecutionMode {
	case "", "mock":
		req.ExecutionMode = "mock"
	case "api", "codex_cli":
	default:
		return req, nil, fmt.Errorf("execution_mode must be one of api, codex_cli, mock")
	}
	if req.ModelProvider == "" {
		req.ModelProvider = "codex"
	}
	if req.ModelName == "" {
		req.ModelName = "planner-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	if ideaDetail.StructuredIdea != nil {
		if len(req.DatasetAssetRefs) == 0 {
			req.DatasetAssetRefs = cloneStringSlice(ideaDetail.StructuredIdea.TargetDatasetRefs)
		}
		if len(req.EvalProtocolRefs) == 0 {
			req.EvalProtocolRefs = cloneStringSlice(ideaDetail.StructuredIdea.DatasetEvalProtocolRefs)
		}
		if len(req.HumanHints) == 0 {
			req.HumanHints = cloneStringSlice(ideaDetail.StructuredIdea.HumanHints)
		}
	}
	if len(req.DatasetAssetRefs) == 0 {
		return req, nil, fmt.Errorf("dataset_asset_refs is required")
	}
	return req, ideaDetail, nil
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

func (s *Service) resolveDatasets(ctx context.Context, refs []string) ([]map[string]any, []model.AgentInputRef, error) {
	out := make([]map[string]any, 0, len(refs))
	inputRefs := make([]model.AgentInputRef, 0, len(refs))
	for _, id := range refs {
		item, err := s.datasets.GetByID(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if item == nil {
			return nil, nil, fmt.Errorf("dataset asset not found")
		}
		evalplanPath := filepath.Join(workspacepkg.New(s.workspaceRoot).DatasetAssetDir(id), "evalplan.json")
		out = append(out, map[string]any{
			"dataset_asset_id": id,
			"name":             item.Asset.Name,
			"task_type":        item.Asset.TaskType,
			"path":             item.Asset.LocalOrRemotePath,
			"evalplan_path":    evalplanPath,
		})
		inputRefs = append(inputRefs, model.AgentInputRef{
			RefType: "dataset_asset",
			RefID:   id,
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

func resolveEvalPlans(datasets []map[string]any, refs []string) ([]map[string]any, []model.AgentInputRef) {
	out := make([]map[string]any, 0)
	inputRefs := make([]model.AgentInputRef, 0)
	seen := map[string]struct{}{}
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, map[string]any{"eval_protocol_ref": ref})
		inputRefs = append(inputRefs, model.AgentInputRef{RefType: "dataset_eval_protocol", RefPath: ref})
	}
	for _, item := range datasets {
		ref := stringValue(item["evalplan_path"])
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, map[string]any{
			"eval_protocol_ref": ref,
			"dataset_asset_id":  stringValue(item["dataset_asset_id"]),
			"task_type":         stringValue(item["task_type"]),
		})
		inputRefs = append(inputRefs, model.AgentInputRef{
			RefType: "dataset_eval_protocol",
			RefPath: ref,
			Metadata: map[string]any{
				"dataset_asset_id": stringValue(item["dataset_asset_id"]),
				"task_type":        stringValue(item["task_type"]),
			},
		})
	}
	return out, inputRefs
}

func (s *Service) resolveBaselines(ctx context.Context, refs []string) ([]map[string]any, []model.AgentInputRef, error) {
	out := make([]map[string]any, 0, len(refs))
	inputRefs := make([]model.AgentInputRef, 0, len(refs))
	for _, id := range refs {
		item, err := s.baselines.GetByID(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if item == nil {
			continue
		}
		out = append(out, map[string]any{
			"baseline_id":        item.Baseline.ID,
			"name":               item.Baseline.Name,
			"dataset_asset_id":   item.Baseline.DatasetAssetID,
			"metric_schema_json": cloneMap(item.Baseline.MetricSchemaJSON),
		})
		inputRefs = append(inputRefs, model.AgentInputRef{
			RefType: "baseline",
			RefID:   item.Baseline.ID,
			Metadata: map[string]any{
				"name":             item.Baseline.Name,
				"dataset_asset_id": item.Baseline.DatasetAssetID,
			},
		})
	}
	return out, inputRefs, nil
}

func (s *Service) resolveServerSnapshots(ctx context.Context, refs []string) ([]map[string]any, []model.AgentInputRef, error) {
	if s.servers == nil {
		return nil, nil, nil
	}
	servers, err := s.servers.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	selected := make([]model.Server, 0)
	if len(refs) == 0 {
		selected = append(selected, servers...)
	} else {
		for _, ref := range refs {
			ref = strings.TrimSpace(ref)
			for _, server := range servers {
				if server.ID == ref || strings.EqualFold(server.Name, ref) {
					selected = append(selected, server)
					break
				}
			}
		}
	}
	out := make([]map[string]any, 0, len(selected))
	inputRefs := make([]model.AgentInputRef, 0, len(selected))
	for _, server := range selected {
		snapshot, err := s.buildServerSnapshot(ctx, server)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, snapshot)
		inputRefs = append(inputRefs, model.AgentInputRef{RefType: "server_resource_snapshot", RefID: server.ID, Metadata: snapshot})
	}
	return out, inputRefs, nil
}

func (s *Service) buildServerSnapshot(ctx context.Context, server model.Server) (map[string]any, error) {
	out := map[string]any{
		"server_id":        server.ID,
		"server_name":      server.Name,
		"status":           server.Status,
		"best_free_mem_mb": 0,
		"best_utilization": 1000,
	}
	if s.heartbeats != nil {
		heartbeats, err := s.heartbeats.ListByServerID(ctx, server.ID, 1)
		if err != nil {
			return nil, err
		}
		if len(heartbeats) > 0 {
			out["status"] = heartbeats[0].Status
			out["heartbeat_at"] = heartbeats[0].HeartbeatAt.Format(time.RFC3339)
		}
	}
	if s.gpuSnapshots != nil {
		snapshots, err := s.gpuSnapshots.ListByServerID(ctx, server.ID, 20)
		if err != nil {
			return nil, err
		}
		bestFree := 0
		bestUtil := 1000
		for _, item := range snapshots {
			if item.FreeMemMB > bestFree || (item.FreeMemMB == bestFree && item.Utilization < bestUtil) {
				bestFree = item.FreeMemMB
				bestUtil = item.Utilization
			}
		}
		out["best_free_mem_mb"] = bestFree
		if bestUtil == 1000 {
			bestUtil = 0
		}
		out["best_utilization"] = bestUtil
	}
	return out, nil
}

func (s *Service) writePlan(experimentID string, job *model.AgentJob, req plannerRequest, payload plannerPayload) (*model.ExperimentPlanDocument, error) {
	expDir := workspacepkg.New(s.workspaceRoot).ExperimentDir(experimentID)
	if err := os.MkdirAll(expDir, 0o755); err != nil {
		return nil, err
	}
	doc := &model.ExperimentPlanDocument{
		ExperimentID:               experimentID,
		IdeaID:                     req.IdeaID,
		DatasetAssetID:             firstNonEmpty(payload.DatasetAssetID, firstString(req.DatasetAssetRefs)),
		BaselineID:                 firstNonEmpty(payload.BaselineID, firstString(req.BaselineRefs)),
		EvalProtocolRefs:           cloneStringSlice(firstNonEmptySlice(payload.EvalProtocolRefs, req.EvalProtocolRefs)),
		ServerResourceSnapshotRefs: cloneStringSlice(firstNonEmptySlice(payload.ServerResourceSnapshotRefs, req.ServerResourceSnapshotRefs)),
		ExperimentPlanJSON:         cloneMap(payload.ExperimentPlanJSON),
		TrainTemplateType:          payload.TrainTemplateType,
		ResourceEstimate:           cloneMap(payload.ResourceEstimate),
		RunSequence:                cloneStringSlice(payload.RunSequence),
		SuccessCriteria:            cloneMap(payload.SuccessCriteria),
		FallbackPlan:               cloneMap(payload.FallbackPlan),
		PlanPath:                   filepath.Join(expDir, "plan.json"),
		PlanMarkdownPath:           filepath.Join(expDir, "plan.md"),
		RuntimeOutputPath:          filepath.Join(job.WorkspaceDir, "output.json"),
		GeneratedAt:                time.Now(),
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	if err = os.WriteFile(doc.PlanPath, raw, 0o644); err != nil {
		return nil, err
	}
	if err = os.WriteFile(doc.PlanMarkdownPath, []byte(buildPlanMarkdown(*doc)), 0o644); err != nil {
		return nil, err
	}
	return doc, nil
}

func (s *Service) readPlan(experimentID string) (*model.ExperimentPlanDocument, error) {
	path := filepath.Join(workspacepkg.New(s.workspaceRoot).ExperimentDir(experimentID), "plan.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc model.ExperimentPlanDocument
	if err = json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Service) publishPlanReady(ctx context.Context, detail *model.ExperimentDetail, plan *model.ExperimentPlanDocument) error {
	if s.events == nil || detail == nil || plan == nil {
		return nil
	}
	_, err := s.events.PublishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "plan_ready",
		SourceRef: "experiment:" + detail.Experiment.ID,
		InputRefs: []model.AgentInputRef{
			{RefType: "experiment", RefID: detail.Experiment.ID},
			{RefType: "idea", RefID: detail.Experiment.IdeaID},
			{RefType: "dataset_asset", RefID: detail.Experiment.DatasetAssetID},
			{RefType: "experiment_plan", RefPath: plan.PlanPath},
		},
		Payload: map[string]any{
			"experiment_id":       detail.Experiment.ID,
			"idea_id":             detail.Experiment.IdeaID,
			"dataset_asset_id":    detail.Experiment.DatasetAssetID,
			"baseline_id":         detail.Experiment.BaselineID,
			"train_template_type": plan.TrainTemplateType,
		},
	})
	return err
}

type plannerRequest struct {
	IdeaID                     string
	DatasetAssetRefs           []string
	EvalProtocolRefs           []string
	ServerResourceSnapshotRefs []string
	BaselineRefs               []string
	HumanHints                 []string
	IdeaTitle                  string
}

type plannerPayload struct {
	DatasetAssetID             string
	BaselineID                 string
	TargetDatasetRefs          []string
	EvalProtocolRefs           []string
	ServerResourceSnapshotRefs []string
	ExperimentPlanJSON         map[string]any
	TrainTemplateType          string
	ResourceEstimate           map[string]any
	RunSequence                []string
	SuccessCriteria            map[string]any
	FallbackPlan               map[string]any
	Priority                   int
}

func requestFromJob(job *model.AgentJob) plannerRequest {
	ideaID := stringValue(job.Metadata["idea_id"])
	datasetRefs := normalizeStringSlice(stringSliceValue(job.Metadata["dataset_asset_refs"]))
	evalRefs := normalizeStringSlice(stringSliceValue(job.Metadata["eval_protocol_refs"]))
	baselineRefs := normalizeStringSlice(stringSliceValue(job.Metadata["baseline_refs"]))
	serverRefs := normalizeStringSlice(stringSliceValue(job.Metadata["server_resource_snapshot_refs"]))
	ideaTitle := stringValue(mapValue(job.Metadata["idea"])["title"])
	for _, ref := range job.InputRefs {
		switch strings.TrimSpace(ref.RefType) {
		case "idea":
			if ideaID == "" {
				ideaID = strings.TrimSpace(ref.RefID)
			}
			if ideaTitle == "" {
				ideaTitle = stringValue(mapValue(ref.Metadata)["title"])
			}
		case "dataset_asset":
			if strings.TrimSpace(ref.RefID) != "" && len(datasetRefs) == 0 {
				datasetRefs = append(datasetRefs, strings.TrimSpace(ref.RefID))
			}
		case "dataset_eval_protocol":
			if len(evalRefs) == 0 {
				if strings.TrimSpace(ref.RefPath) != "" {
					evalRefs = append(evalRefs, strings.TrimSpace(ref.RefPath))
				} else if strings.TrimSpace(ref.RefID) != "" {
					evalRefs = append(evalRefs, strings.TrimSpace(ref.RefID))
				}
			}
		case "baseline":
			if strings.TrimSpace(ref.RefID) != "" && len(baselineRefs) == 0 {
				baselineRefs = append(baselineRefs, strings.TrimSpace(ref.RefID))
			}
		case "server_resource_snapshot":
			if strings.TrimSpace(ref.RefID) != "" && len(serverRefs) == 0 {
				serverRefs = append(serverRefs, strings.TrimSpace(ref.RefID))
			}
		}
	}
	return plannerRequest{
		IdeaID:                     ideaID,
		DatasetAssetRefs:           datasetRefs,
		EvalProtocolRefs:           evalRefs,
		ServerResourceSnapshotRefs: serverRefs,
		BaselineRefs:               baselineRefs,
		HumanHints:                 normalizeStringSlice(stringSliceValue(job.Metadata["human_hints"])),
		IdeaTitle:                  ideaTitle,
	}
}

func extractPlannerPayload(payload map[string]any) plannerPayload {
	planJSON := mapValue(payload["experiment_plan_json"])
	return plannerPayload{
		DatasetAssetID:             firstNonEmpty(stringValue(planJSON["dataset_asset_id"]), stringValue(payload["dataset_asset_id"])),
		BaselineID:                 firstNonEmpty(stringValue(planJSON["baseline_id"]), stringValue(payload["baseline_id"])),
		TargetDatasetRefs:          normalizeStringSlice(stringSliceValue(payload["target_dataset_refs"])),
		EvalProtocolRefs:           normalizeStringSlice(stringSliceValue(payload["eval_protocol_refs"])),
		ServerResourceSnapshotRefs: normalizeStringSlice(stringSliceValue(payload["server_resource_snapshot_refs"])),
		ExperimentPlanJSON:         cloneMap(planJSON),
		TrainTemplateType:          stringValue(payload["train_template_type"]),
		ResourceEstimate:           cloneMap(mapValue(payload["resource_estimate"])),
		RunSequence:                normalizeStringSlice(stringSliceValue(payload["run_sequence"])),
		SuccessCriteria:            cloneMap(mapValue(payload["success_criteria"])),
		FallbackPlan:               cloneMap(mapValue(payload["fallback_plan"])),
		Priority:                   intValue(payload["priority"]),
	}
}

func updateJobPayload(payload map[string]any, experiment *model.ExperimentDetail, plan *model.ExperimentPlanDocument) map[string]any {
	out := cloneMap(payload)
	if out == nil {
		out = map[string]any{}
	}
	if experiment != nil {
		out["experiment_id"] = experiment.Experiment.ID
		out["dataset_asset_id"] = experiment.Experiment.DatasetAssetID
		out["idea_id"] = experiment.Experiment.IdeaID
		out["baseline_id"] = experiment.Experiment.BaselineID
	}
	if plan != nil {
		out["plan_path"] = plan.PlanPath
		out["plan_markdown_path"] = plan.PlanMarkdownPath
		out["train_template_type"] = plan.TrainTemplateType
	}
	return out
}

func buildPlanMarkdown(plan model.ExperimentPlanDocument) string {
	lines := []string{
		"# Experiment Plan",
		"",
		"- Experiment ID: " + plan.ExperimentID,
		"- Idea ID: " + plan.IdeaID,
		"- Dataset Asset ID: " + plan.DatasetAssetID,
		"- Baseline ID: " + firstNonEmpty(plan.BaselineID, "none"),
		"- Train Template Type: " + plan.TrainTemplateType,
		"",
		"## Run Sequence",
	}
	for _, step := range plan.RunSequence {
		lines = append(lines, "- "+step)
	}
	lines = append(lines, "", "## Success Criteria", prettyJSON(plan.SuccessCriteria))
	lines = append(lines, "", "## Fallback Plan", prettyJSON(plan.FallbackPlan))
	return strings.Join(lines, "\n")
}

func buildExperimentTitle(ideaTitle string, datasetAssetID string) string {
	if strings.TrimSpace(ideaTitle) == "" {
		return "Planned Experiment | " + datasetAssetID
	}
	return strings.TrimSpace(ideaTitle) + " | Planned"
}

func buildExperimentSummary(payload plannerPayload) string {
	lines := []string{
		"Planner-generated experiment plan.",
		"",
		"- Train template: " + payload.TrainTemplateType,
		"- Dataset asset: " + payload.DatasetAssetID,
	}
	if baseline := strings.TrimSpace(payload.BaselineID); baseline != "" {
		lines = append(lines, "- Baseline: "+baseline)
	}
	if len(payload.RunSequence) > 0 {
		lines = append(lines, "- First step: "+payload.RunSequence[0])
	}
	return strings.Join(lines, "\n")
}

func buildExperimentOwnerNote(payload plannerPayload) string {
	return "Planner resource estimate:\n" + prettyJSON(payload.ResourceEstimate)
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
		out["risk_points"] = cloneStringSlice(detail.StructuredIdea.RiskPoints)
		out["target_dataset_refs"] = cloneStringSlice(detail.StructuredIdea.TargetDatasetRefs)
		out["dataset_eval_protocol_refs"] = cloneStringSlice(detail.StructuredIdea.DatasetEvalProtocolRefs)
	}
	return out
}

func normalizeStringSlice(input []string) []string {
	out := make([]string, 0, len(input))
	for _, item := range input {
		if text := strings.TrimSpace(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func stringSliceValue(value any) []string {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(stringValue(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		if text := strings.TrimSpace(stringValue(value)); text != "" {
			return []string{text}
		}
	}
	return nil
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

func cloneStringSlice(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	out := make([]string, len(input))
	copy(out, input)
	return out
}

func firstString(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.TrimSpace(items[0])
}

func firstNonEmpty(values ...string) string {
	for _, item := range values {
		if text := strings.TrimSpace(item); text != "" {
			return text
		}
	}
	return ""
}

func firstNonEmptySlice(primary []string, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
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

func prettyJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}
