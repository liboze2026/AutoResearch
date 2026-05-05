package datasetagent

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

const datasetOutputSchemaRef = "schemas/dataset-output-v1.json"

type jobCreator interface {
	Create(context.Context, model.AgentJobCreateRequest) (*model.AgentJob, error)
}

type jobUpdater interface {
	Update(context.Context, model.AgentJob) error
}

type triggerService interface {
	Trigger(context.Context, string, model.AgentJobTriggerRequest) (*model.AgentJob, error)
}

type datasetAssetWriter interface {
	Create(context.Context, model.DatasetAssetCreateRequest) (*model.DatasetAssetDetail, error)
	RegisterFromScan(context.Context, model.DatasetAssetRegisterFromScanRequest) (*model.DatasetAssetDetail, error)
	GetByID(context.Context, string) (*model.DatasetAssetDetail, error)
}

type datasetAssetLookup interface {
	GetByExistingDatasetRef(context.Context, string) (*model.DatasetAsset, error)
}

type baselineWriter interface {
	Create(context.Context, model.BaselineCreateRequest) (*model.BaselineDetail, error)
	GetByID(context.Context, string) (*model.BaselineDetail, error)
}

type datasetDiscovery interface {
	List(context.Context, string, string, string) ([]model.Dataset, error)
}

type serverInspector interface {
	List(context.Context) ([]model.Server, error)
	CheckGPU(context.Context, string) (*model.GPUProbeResult, error)
}

type memoryWriter interface {
	Upsert(context.Context, model.AgentMemoryUpsertRequest) (*model.AgentMemoryRecord, error)
}

type Service struct {
	jobs          jobCreator
	jobUpdates    jobUpdater
	triggers      triggerService
	datasetAssets datasetAssetWriter
	assetLookup   datasetAssetLookup
	baselines     baselineWriter
	datasets      datasetDiscovery
	servers       serverInspector
	memories      memoryWriter
	workspaceRoot string
}

func NewService(
	jobs jobCreator,
	jobUpdates jobUpdater,
	triggers triggerService,
	datasetAssets datasetAssetWriter,
	assetLookup datasetAssetLookup,
	baselines baselineWriter,
	datasets datasetDiscovery,
	servers serverInspector,
	memories memoryWriter,
	workspaceRoot string,
) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{
		jobs:          jobs,
		jobUpdates:    jobUpdates,
		triggers:      triggers,
		datasetAssets: datasetAssets,
		assetLookup:   assetLookup,
		baselines:     baselines,
		datasets:      datasets,
		servers:       servers,
		memories:      memories,
		workspaceRoot: workspaceRoot,
	}
}

func (s *Service) Run(ctx context.Context, req model.DatasetRunRequest) (*model.DatasetRunResult, error) {
	normalizedReq, err := s.normalizeRequest(req)
	if err != nil {
		return nil, err
	}
	discoveredDatasets, err := s.discoverDatasets(ctx, normalizedReq)
	if err != nil {
		return nil, err
	}
	serverContext, serverWarnings := s.resolveServerContext(ctx, normalizedReq)
	job, err := s.jobs.Create(ctx, model.AgentJobCreateRequest{
		AgentType:       "dataset",
		ExecutionMode:   normalizedReq.ExecutionMode,
		ModelProvider:   normalizedReq.ModelProvider,
		ModelName:       normalizedReq.ModelName,
		PromptVersion:   normalizedReq.PromptVersion,
		InputRefs:       buildInputRefs(discoveredDatasets, serverContext),
		OutputSchemaRef: datasetOutputSchemaRef,
		SkillRefs:       normalizedReq.SkillRefs,
		ToolRefs:        normalizedReq.ToolRefs,
		MemoryRefs:      normalizedReq.MemoryRefs,
		Metadata: map[string]any{
			"research_direction":       normalizedReq.ResearchDirection,
			"task_type":                normalizedReq.TaskType,
			"keywords":                 normalizedReq.Keywords,
			"target_server_preference": normalizedReq.TargetServerPreference,
			"dataset_constraints":      normalizedReq.DatasetConstraints,
			"discovered_datasets":      datasetMetadataList(discoveredDatasets),
			"server_context":           serverContext,
		},
		Status: "registered",
	})
	if err != nil {
		return nil, err
	}
	job, err = s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
		TriggerType: "manual",
		Metadata: map[string]any{
			"agent_type": "dataset",
		},
	})
	if err != nil {
		return nil, err
	}
	result, err := s.resultFromJob(ctx, job)
	if err != nil {
		return nil, err
	}
	result.Warnings = append(result.Warnings, serverWarnings...)
	return result, nil
}

func (s *Service) PostProcess(ctx context.Context, job *model.AgentJob) error {
	if job == nil {
		return nil
	}
	req := requestFromJob(job)
	payload := extractDatasetPayload(job.NormalizedPayload)
	assetDetail, err := s.persistDatasetAsset(ctx, req, payload, job)
	if err != nil {
		return err
	}
	evalPlan := s.buildEvalPlan(assetDetail.Asset, payload)
	var baselineDetail *model.BaselineDetail
	if evalPlan.BaselineNeeded && s.baselines != nil {
		baselineDetail, err = s.baselines.Create(ctx, model.BaselineCreateRequest{
			DatasetAssetID:   assetDetail.Asset.ID,
			Name:             buildBaselineName(req.TaskType),
			MetricSchemaJSON: cloneMap(evalPlan.MetricSchemaJSON),
			ResultJSON: map[string]any{
				"status":        "planned",
				"protocol_path": evalPlan.EvalPlanPath,
			},
			NoteMD:     buildBaselineNote(req, assetDetail.Asset, evalPlan),
			SourceType: "manual",
		})
		if err != nil {
			return err
		}
		evalPlan.BaselineID = baselineDetail.Baseline.ID
	}
	if s.memories != nil {
		evalPlan.MemoryKey = "dataset_evalplan_" + assetDetail.Asset.ID
		_, err = s.memories.Upsert(ctx, model.AgentMemoryUpsertRequest{
			AgentType: "dataset",
			MemoryKey: evalPlan.MemoryKey,
			ContentMD: buildMemoryContent(req, assetDetail.Asset, evalPlan),
			SourceRef: "dataset_asset:" + assetDetail.Asset.ID,
		})
		if err != nil {
			return err
		}
	}
	job.NormalizedPayload = updateJobPayload(job.NormalizedPayload, assetDetail.Asset, evalPlan)
	if err = s.writeEvalPlan(assetDetail.Asset, *evalPlan, job); err != nil {
		return err
	}
	job.UpdatedAt = time.Now()
	return s.jobUpdates.Update(ctx, *job)
}

func (s *Service) GetEvalPlan(ctx context.Context, datasetAssetID string) (*model.DatasetEvalPlanResponse, error) {
	if strings.TrimSpace(datasetAssetID) == "" {
		return nil, fmt.Errorf("dataset asset id is required")
	}
	assetDetail, err := s.datasetAssets.GetByID(ctx, strings.TrimSpace(datasetAssetID))
	if err != nil {
		return nil, err
	}
	if assetDetail == nil {
		return nil, nil
	}
	evalPlan, err := s.readEvalPlan(datasetAssetID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var baselineDetail *model.BaselineDetail
	if evalPlan.BaselineID != "" && s.baselines != nil {
		baselineDetail, err = s.baselines.GetByID(ctx, evalPlan.BaselineID)
		if err != nil {
			return nil, err
		}
	}
	return &model.DatasetEvalPlanResponse{
		DatasetAsset: assetDetail,
		Baseline:     baselineDetail,
		EvalPlan:     *evalPlan,
	}, nil
}

func (s *Service) resultFromJob(ctx context.Context, job *model.AgentJob) (*model.DatasetRunResult, error) {
	if job == nil {
		return nil, fmt.Errorf("dataset agent job not found")
	}
	payload := extractDatasetPayload(job.NormalizedPayload)
	result := &model.DatasetRunResult{
		Job:      job,
		Warnings: append([]string{}, job.Warnings...),
	}
	if payload.DatasetAssetRef == "" {
		return result, nil
	}
	assetDetail, err := s.datasetAssets.GetByID(ctx, payload.DatasetAssetRef)
	if err != nil {
		return nil, err
	}
	result.DatasetAsset = assetDetail
	evalPlanResp, err := s.GetEvalPlan(ctx, payload.DatasetAssetRef)
	if err != nil {
		return nil, err
	}
	if evalPlanResp != nil {
		result.EvalPlan = &evalPlanResp.EvalPlan
		result.Baseline = evalPlanResp.Baseline
	}
	return result, nil
}

func (s *Service) normalizeRequest(req model.DatasetRunRequest) (model.DatasetRunRequest, error) {
	req.ResearchDirection = strings.TrimSpace(req.ResearchDirection)
	req.TaskType = strings.ToLower(strings.TrimSpace(req.TaskType))
	req.TargetServerPreference = strings.TrimSpace(req.TargetServerPreference)
	req.ExecutionMode = strings.TrimSpace(req.ExecutionMode)
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.ModelName = strings.TrimSpace(req.ModelName)
	req.PromptVersion = strings.TrimSpace(req.PromptVersion)
	req.Keywords = normalizeKeywords(req.Keywords)
	if req.ResearchDirection == "" {
		return req, fmt.Errorf("research_direction is required")
	}
	if req.TaskType == "" {
		return req, fmt.Errorf("task_type is required")
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
		req.ModelName = "dataset-default"
	}
	if req.PromptVersion == "" {
		req.PromptVersion = "v1"
	}
	if req.TargetServerPreference == "" {
		req.TargetServerPreference = "shenzhenvlab"
	}
	if req.DatasetConstraints == nil {
		req.DatasetConstraints = map[string]any{}
	}
	return req, nil
}

func (s *Service) discoverDatasets(ctx context.Context, req model.DatasetRunRequest) ([]model.Dataset, error) {
	if s.datasets == nil {
		return []model.Dataset{}, nil
	}
	queries := buildDiscoveryQueries(req)
	seen := map[string]struct{}{}
	items := make([]model.Dataset, 0)
	modality := requestedDatasetModality(req.TaskType)
	for _, query := range queries {
		found, err := s.datasets.List(ctx, query, "", modality)
		if err != nil {
			return nil, err
		}
		for _, item := range found {
			if _, exists := seen[item.ID]; exists {
				continue
			}
			seen[item.ID] = struct{}{}
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *Service) resolveServerContext(ctx context.Context, req model.DatasetRunRequest) (map[string]any, []string) {
	contextValue := map[string]any{
		"selected_server_name": firstNonEmpty(req.TargetServerPreference, "mock_server"),
		"selected_server_id":   "",
		"decision_mode":        "mock",
		"gpu_available":        false,
		"fallback_reason":      "",
	}
	if s.servers == nil {
		contextValue["selected_server_name"] = "mock_server"
		contextValue["fallback_reason"] = "server inspector not configured"
		return contextValue, []string{"server inspector not configured; dataset agent fell back to mock server context"}
	}
	servers, err := s.servers.List(ctx)
	if err != nil {
		contextValue["selected_server_name"] = "mock_server"
		contextValue["fallback_reason"] = err.Error()
		return contextValue, []string{"server discovery failed; dataset agent fell back to mock server context"}
	}
	preferred := findServerByName(servers, req.TargetServerPreference)
	if preferred == nil {
		contextValue["selected_server_name"] = fallbackServerName(servers)
		contextValue["fallback_reason"] = "preferred server not found"
		return contextValue, []string{fmt.Sprintf("preferred server %q not found; dataset agent used mock fallback", req.TargetServerPreference)}
	}
	contextValue["selected_server_name"] = preferred.Name
	contextValue["selected_server_id"] = preferred.ID
	if !strings.EqualFold(preferred.Name, "shenzhenvlab") {
		contextValue["decision_mode"] = "real"
		return contextValue, nil
	}
	allowWithoutGPU := boolValue(req.DatasetConstraints["allow_server_without_idle_gpu"]) || boolValue(req.DatasetConstraints["register_only"])
	probe, err := s.servers.CheckGPU(ctx, preferred.ID)
	if err != nil {
		if allowWithoutGPU {
			contextValue["decision_mode"] = "real"
			contextValue["fallback_reason"] = "gpu probe failed but policy allows server usage"
			return contextValue, []string{"shenzhenvlab gpu probe failed, but policy allowed dataset registration without idle gpu"}
		}
		contextValue["selected_server_name"] = fallbackServerName(servers)
		contextValue["selected_server_id"] = ""
		contextValue["fallback_reason"] = err.Error()
		return contextValue, []string{"shenzhenvlab is unavailable for dataset work; dataset agent fell back to mock server context"}
	}
	if probe != nil && probe.AvailableGPUCount > 0 {
		contextValue["decision_mode"] = "real"
		contextValue["gpu_available"] = true
		return contextValue, nil
	}
	if allowWithoutGPU {
		contextValue["decision_mode"] = "real"
		contextValue["gpu_available"] = false
		contextValue["fallback_reason"] = "gpu is busy but policy allows dataset registration"
		return contextValue, []string{"shenzhenvlab gpu is busy, but policy allowed dataset registration on the preferred server"}
	}
	contextValue["selected_server_name"] = fallbackServerName(servers)
	contextValue["selected_server_id"] = ""
	contextValue["fallback_reason"] = "shenzhenvlab gpu not idle"
	return contextValue, []string{"shenzhenvlab gpu is not idle; dataset agent fell back to mock server context"}
}

func (s *Service) persistDatasetAsset(ctx context.Context, req datasetRequestSnapshot, payload datasetPayload, job *model.AgentJob) (*model.DatasetAssetDetail, error) {
	if payload.FetchAction == "register_existing" && payload.SelectedDatasetRef != "" && !isRemoteAliasDatasetLocation(payload.DatasetLocation) {
		if s.assetLookup != nil {
			existing, err := s.assetLookup.GetByExistingDatasetRef(ctx, payload.SelectedDatasetRef)
			if err != nil {
				return nil, err
			}
			if existing != nil {
				return s.datasetAssets.GetByID(ctx, existing.ID)
			}
		}
		return s.datasetAssets.RegisterFromScan(ctx, model.DatasetAssetRegisterFromScanRequest{
			ExistingDatasetRef: payload.SelectedDatasetRef,
			Name:               buildDatasetAssetName(req, payload),
			DescriptionMD:      buildDatasetAssetDescription(req, payload),
			TaskType:           req.TaskType,
			Status:             "active",
			SourceType:         "mrag_scan",
			ReadmeMD:           buildDatasetReadme(req, payload),
			LoaderNoteMD:       buildDatasetLoaderNote(payload),
			SchemaNoteMD:       buildDatasetSchemaNote(payload),
		})
	}
	location := strings.TrimSpace(payload.DatasetLocation)
	if location == "" {
		location = filepath.Join(workspacepkg.New(s.workspaceRoot).DatasetsRoot(), "downloads", job.ID, "mock_dataset")
	}
	if shouldMaterializeDatasetLocation(location, payload.FetchAction, s.workspaceRoot) {
		if err := os.MkdirAll(location, 0o755); err != nil {
			return nil, err
		}
		readmePath := filepath.Join(location, "README.md")
		if _, err := os.Stat(readmePath); err != nil {
			_ = os.WriteFile(readmePath, []byte(ensureTrailingNewline(buildDownloadedDatasetReadme(req, payload))), 0o644)
		}
	}
	return s.datasetAssets.Create(ctx, model.DatasetAssetCreateRequest{
		Name:              buildDatasetAssetName(req, payload),
		DescriptionMD:     buildDatasetAssetDescription(req, payload),
		TaskType:          req.TaskType,
		Status:            "active",
		SourceType:        "manual",
		LocalOrRemotePath: location,
		ReadmeMD:          buildDatasetReadme(req, payload),
		LoaderNoteMD:      buildDatasetLoaderNote(payload),
		SchemaNoteMD:      buildDatasetSchemaNote(payload),
	})
}

func (s *Service) buildEvalPlan(asset model.DatasetAsset, payload datasetPayload) *model.DatasetEvalPlanDocument {
	assetDir := workspacepkg.New(s.workspaceRoot).DatasetAssetDir(asset.ID)
	return &model.DatasetEvalPlanDocument{
		DatasetAssetID:     asset.ID,
		DatasetLocation:    asset.LocalOrRemotePath,
		FetchAction:        payload.FetchAction,
		SelectedDatasetRef: payload.SelectedDatasetRef,
		ServerDecision:     cloneMap(payload.ServerDecision),
		EvalProtocolJSON:   cloneMap(payload.EvalProtocolJSON),
		MetricSchemaJSON:   cloneMap(payload.MetricSchemaJSON),
		SplitStrategy:      payload.SplitStrategy,
		NotesMD:            payload.NotesMD,
		EvalPlanPath:       filepath.Join(assetDir, "evalplan.json"),
		NotesPath:          filepath.Join(assetDir, "dataset_agent_notes.md"),
		RuntimeOutputPath:  filepath.Join(assetDir, "dataset_agent_output.json"),
		BaselineNeeded:     boolValue(payload.EvalProtocolJSON["baseline_needed"]),
		TargetServerName:   stringValue(payload.ServerDecision["selected_server_name"]),
		GeneratedAt:        time.Now(),
	}
}

func (s *Service) writeEvalPlan(asset model.DatasetAsset, evalPlan model.DatasetEvalPlanDocument, job *model.AgentJob) error {
	assetDir := workspacepkg.New(s.workspaceRoot).DatasetAssetDir(asset.ID)
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(evalPlan, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(evalPlan.EvalPlanPath, raw, 0o644); err != nil {
		return err
	}
	if err = os.WriteFile(evalPlan.NotesPath, []byte(ensureTrailingNewline(evalPlan.NotesMD)), 0o644); err != nil {
		return err
	}
	outputRaw, err := json.MarshalIndent(job.NormalizedPayload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(evalPlan.RuntimeOutputPath, outputRaw, 0o644)
}

func (s *Service) readEvalPlan(datasetAssetID string) (*model.DatasetEvalPlanDocument, error) {
	path := filepath.Join(workspacepkg.New(s.workspaceRoot).DatasetAssetDir(datasetAssetID), "evalplan.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var item model.DatasetEvalPlanDocument
	if err = json.Unmarshal(data, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

type datasetRequestSnapshot struct {
	ResearchDirection      string
	TaskType               string
	Keywords               []string
	TargetServerPreference string
	DatasetConstraints     map[string]any
}

type datasetPayload struct {
	DatasetAssetRef    string
	DatasetLocation    string
	FetchAction        string
	SelectedDatasetRef string
	ServerDecision     map[string]any
	EvalProtocolJSON   map[string]any
	MetricSchemaJSON   map[string]any
	SplitStrategy      string
	NotesMD            string
}

func requestFromJob(job *model.AgentJob) datasetRequestSnapshot {
	return datasetRequestSnapshot{
		ResearchDirection:      stringValue(job.Metadata["research_direction"]),
		TaskType:               strings.ToLower(stringValue(job.Metadata["task_type"])),
		Keywords:               stringSliceValue(job.Metadata["keywords"]),
		TargetServerPreference: stringValue(job.Metadata["target_server_preference"]),
		DatasetConstraints:     mapValue(job.Metadata["dataset_constraints"]),
	}
}

func extractDatasetPayload(payload map[string]any) datasetPayload {
	return datasetPayload{
		DatasetAssetRef:    stringValue(payload["dataset_asset_ref"]),
		DatasetLocation:    stringValue(payload["dataset_location"]),
		FetchAction:        normalizeFetchAction(firstNonEmpty(stringValue(payload["fetch_action"]), "register_existing")),
		SelectedDatasetRef: stringValue(payload["selected_dataset_ref"]),
		ServerDecision:     mapValue(payload["server_decision"]),
		EvalProtocolJSON:   mapValue(payload["eval_protocol_json"]),
		MetricSchemaJSON:   mapValue(payload["metric_schema_json"]),
		SplitStrategy:      stringValue(payload["split_strategy"]),
		NotesMD:            stringValue(payload["notes_md"]),
	}
}

func updateJobPayload(payload map[string]any, asset model.DatasetAsset, evalPlan *model.DatasetEvalPlanDocument) map[string]any {
	out := cloneMap(payload)
	out["dataset_asset_ref"] = asset.ID
	out["dataset_location"] = asset.LocalOrRemotePath
	out["eval_protocol_json"] = cloneMap(evalPlan.EvalProtocolJSON)
	out["metric_schema_json"] = cloneMap(evalPlan.MetricSchemaJSON)
	out["split_strategy"] = evalPlan.SplitStrategy
	out["notes_md"] = evalPlan.NotesMD
	out["evalplan_path"] = evalPlan.EvalPlanPath
	out["notes_path"] = evalPlan.NotesPath
	out["runtime_output_path"] = evalPlan.RuntimeOutputPath
	out["server_decision"] = cloneMap(evalPlan.ServerDecision)
	out["baseline_id"] = evalPlan.BaselineID
	out["memory_key"] = evalPlan.MemoryKey
	data := mapValue(out["data"])
	data["dataset_asset_id"] = asset.ID
	data["dataset_location"] = asset.LocalOrRemotePath
	data["evalplan_path"] = evalPlan.EvalPlanPath
	if evalPlan.BaselineID != "" {
		data["baseline_id"] = evalPlan.BaselineID
	}
	out["data"] = data
	return out
}

func buildInputRefs(datasets []model.Dataset, serverContext map[string]any) []model.AgentInputRef {
	refs := make([]model.AgentInputRef, 0, len(datasets)+1)
	for _, item := range datasets {
		refs = append(refs, model.AgentInputRef{
			RefType: "dataset",
			RefID:   item.ID,
			RefPath: item.Path,
			Metadata: map[string]any{
				"name":        item.Name,
				"server_name": item.ServerName,
				"modality":    firstNonEmpty(item.DetectedModality, item.Modality),
			},
		})
	}
	serverName := stringValue(serverContext["selected_server_name"])
	if serverName != "" {
		refs = append(refs, model.AgentInputRef{
			RefType: "server",
			RefID:   stringValue(serverContext["selected_server_id"]),
			Metadata: map[string]any{
				"name":          serverName,
				"decision_mode": stringValue(serverContext["decision_mode"]),
				"gpu_available": boolValue(serverContext["gpu_available"]),
			},
		})
	}
	return refs
}

func datasetMetadataList(items []model.Dataset) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"id":          item.ID,
			"name":        item.Name,
			"description": item.Description,
			"path":        item.Path,
			"server_name": item.ServerName,
			"modality":    firstNonEmpty(item.DetectedModality, item.Modality),
		})
	}
	return out
}

func buildDiscoveryQueries(req model.DatasetRunRequest) []string {
	values := append([]string{}, req.Keywords...)
	values = append(values, req.ResearchDirection)
	values = append(values, strings.Fields(req.ResearchDirection)...)
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, item := range values {
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

func normalizeFetchAction(value string) string {
	action := strings.ToLower(strings.TrimSpace(value))
	switch {
	case action == "":
		return "register_existing"
	case strings.Contains(action, "register_existing"):
		return "register_existing"
	case strings.Contains(action, "mock") && strings.Contains(action, "download"):
		return "mock_download"
	default:
		return action
	}
}

func shouldMaterializeDatasetLocation(location string, fetchAction string, workspaceRoot string) bool {
	cleaned := strings.TrimSpace(location)
	if cleaned == "" {
		return true
	}
	if normalizeFetchAction(fetchAction) == "mock_download" {
		return true
	}
	if isRemoteDatasetLocation(cleaned) {
		return false
	}
	datasetsRoot := workspacepkg.New(workspaceRoot).DatasetsRoot()
	return hasPathPrefix(cleaned, datasetsRoot)
}

func isRemoteDatasetLocation(value string) bool {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return false
	}
	if strings.HasPrefix(cleaned, "/") {
		return true
	}
	return isRemoteAliasDatasetLocation(cleaned)
}

func isRemoteAliasDatasetLocation(value string) bool {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return false
	}
	if len(cleaned) >= 3 && cleaned[1] == ':' && (cleaned[2] == '\\' || cleaned[2] == '/') {
		return false
	}
	return strings.Contains(cleaned, ":/")
}

func hasPathPrefix(value string, prefix string) bool {
	value = strings.ToLower(filepath.Clean(value))
	prefix = strings.ToLower(filepath.Clean(prefix))
	if value == prefix {
		return true
	}
	return strings.HasPrefix(value, prefix+string(os.PathSeparator))
}

func requestedDatasetModality(taskType string) string {
	switch strings.ToLower(strings.TrimSpace(taskType)) {
	case "multimodal":
		return "multimodal"
	case "image":
		return "image"
	case "audio":
		return "audio"
	case "video":
		return "video"
	default:
		return ""
	}
}

func findServerByName(items []model.Server, name string) *model.Server {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Name), strings.TrimSpace(name)) {
			copyItem := item
			return &copyItem
		}
	}
	return nil
}

func fallbackServerName(items []model.Server) string {
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), "mock") {
			return item.Name
		}
	}
	return "mock_server"
}

func buildDatasetAssetName(req datasetRequestSnapshot, payload datasetPayload) string {
	if payload.SelectedDatasetRef != "" {
		return fmt.Sprintf("%s dataset asset", titleLike(req.ResearchDirection))
	}
	return fmt.Sprintf("%s mock dataset", titleLike(req.ResearchDirection))
}

func buildDatasetAssetDescription(req datasetRequestSnapshot, payload datasetPayload) string {
	return fmt.Sprintf(
		"Dataset Agent prepared a controlled dataset asset for `%s` (%s).\n\n- Fetch action: %s\n- Split strategy: %s",
		req.ResearchDirection,
		req.TaskType,
		payload.FetchAction,
		payload.SplitStrategy,
	)
}

func buildDatasetReadme(req datasetRequestSnapshot, payload datasetPayload) string {
	lines := []string{
		fmt.Sprintf("# %s Dataset Plan", titleLike(req.ResearchDirection)),
		"",
		fmt.Sprintf("- Task type: %s", req.TaskType),
		fmt.Sprintf("- Fetch action: %s", payload.FetchAction),
		fmt.Sprintf("- Target server: %s", stringValue(payload.ServerDecision["selected_server_name"])),
	}
	if payload.SelectedDatasetRef != "" {
		lines = append(lines, fmt.Sprintf("- Existing dataset ref: %s", payload.SelectedDatasetRef))
	}
	lines = append(lines, "", payload.NotesMD)
	return strings.Join(lines, "\n")
}

func buildDownloadedDatasetReadme(req datasetRequestSnapshot, payload datasetPayload) string {
	return fmt.Sprintf("# Mock Downloaded Dataset\n\nThis placeholder dataset was created by Dataset Agent for `%s`.\n\n%s\n", req.ResearchDirection, payload.NotesMD)
}

func buildDatasetLoaderNote(payload datasetPayload) string {
	if payload.FetchAction == "register_existing" {
		return fmt.Sprintf("- Reuse MRAG scanned dataset ref `%s`.\n- Read from the registered existing location without re-downloading.", payload.SelectedDatasetRef)
	}
	return "- Use the staged mock dataset directory prepared by Dataset Agent.\n- Replace this path with a real downloader in the next iteration."
}

func buildDatasetSchemaNote(payload datasetPayload) string {
	return fmt.Sprintf("- Split strategy: %s\n- Metric list: %s", payload.SplitStrategy, strings.Join(stringSliceValue(payload.EvalProtocolJSON["metric_list"]), ", "))
}

func buildBaselineName(taskType string) string {
	taskType = strings.TrimSpace(taskType)
	if taskType == "" {
		taskType = "dataset"
	}
	return fmt.Sprintf("%s-baseline-plan", taskType)
}

func buildBaselineNote(req datasetRequestSnapshot, asset model.DatasetAsset, evalPlan *model.DatasetEvalPlanDocument) string {
	return fmt.Sprintf(
		"Dataset Agent created this baseline placeholder for `%s`.\n\n- Dataset asset: %s\n- Split strategy: %s\n- Metrics: %s",
		req.ResearchDirection,
		asset.Name,
		evalPlan.SplitStrategy,
		strings.Join(stringSliceValue(evalPlan.EvalProtocolJSON["metric_list"]), ", "),
	)
}

func buildMemoryContent(req datasetRequestSnapshot, asset model.DatasetAsset, evalPlan *model.DatasetEvalPlanDocument) string {
	return fmt.Sprintf(
		"# Dataset Eval Plan Memory\n\n- Research direction: %s\n- Dataset asset: %s\n- Location: %s\n- Split strategy: %s\n- Metrics: %s\n",
		req.ResearchDirection,
		asset.ID,
		asset.LocalOrRemotePath,
		evalPlan.SplitStrategy,
		strings.Join(stringSliceValue(evalPlan.EvalProtocolJSON["metric_list"]), ", "),
	)
}

func titleLike(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Dataset"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func normalizeKeywords(items []string) []string {
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
	return cloneMap(typed)
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	copyValue := make(map[string]any, len(value))
	for key, item := range value {
		copyValue[key] = item
	}
	return copyValue
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

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func ensureTrailingNewline(value string) string {
	if strings.HasSuffix(value, "\n") {
		return value
	}
	return value + "\n"
}
