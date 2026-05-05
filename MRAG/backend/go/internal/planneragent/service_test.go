package planneragent

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentpipeline"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/model"
	baseservice "mrag-platform/backend/go/internal/service"
)

type memoryPlannerJobStore struct{ items map[string]model.AgentJob }

func newMemoryPlannerJobStore() *memoryPlannerJobStore {
	return &memoryPlannerJobStore{items: map[string]model.AgentJob{}}
}
func (s *memoryPlannerJobStore) Create(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}
func (s *memoryPlannerJobStore) GetByID(_ context.Context, id string) (*model.AgentJob, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}
func (s *memoryPlannerJobStore) Update(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}
func (s *memoryPlannerJobStore) ListByStatus(_ context.Context, status string, limit int) ([]model.AgentJob, error) {
	items := make([]model.AgentJob, 0)
	for _, item := range s.items {
		if item.Status == status {
			items = append(items, item)
		}
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items, nil
}
func (s *memoryPlannerJobStore) CountActiveByAgentType(_ context.Context, agentType string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.AgentType == agentType && (item.Status == "running" || item.Status == "validating" || item.Status == "repairing") {
			count++
		}
	}
	return count, nil
}
func (s *memoryPlannerJobStore) FindByDedupKey(_ context.Context, dedupKey string) (*model.AgentJob, error) {
	for _, item := range s.items {
		if item.DedupKey == dedupKey {
			copyItem := item
			return &copyItem, nil
		}
	}
	return nil, nil
}

type memoryPlannerTriggerStore struct {
	items map[string]model.AgentJobTrigger
}

func newMemoryPlannerTriggerStore() *memoryPlannerTriggerStore {
	return &memoryPlannerTriggerStore{items: map[string]model.AgentJobTrigger{}}
}
func (s *memoryPlannerTriggerStore) Create(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}
func (s *memoryPlannerTriggerStore) Update(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}

type memoryPlannerArtifactStore struct {
	items map[string][]model.AgentArtifact
}

func newMemoryPlannerArtifactStore() *memoryPlannerArtifactStore {
	return &memoryPlannerArtifactStore{items: map[string][]model.AgentArtifact{}}
}
func (s *memoryPlannerArtifactStore) Create(_ context.Context, item model.AgentArtifact) error {
	s.items[item.JobID] = append(s.items[item.JobID], item)
	return nil
}

type memoryPlannerEventStore struct{ items map[string]model.AgentEvent }

func newMemoryPlannerEventStore() *memoryPlannerEventStore {
	return &memoryPlannerEventStore{items: map[string]model.AgentEvent{}}
}
func (s *memoryPlannerEventStore) Create(_ context.Context, item model.AgentEvent) error {
	s.items[item.ID] = item
	return nil
}
func (s *memoryPlannerEventStore) Update(_ context.Context, item model.AgentEvent) error {
	s.items[item.ID] = item
	return nil
}

type memoryPlannerSubscriptionStore struct {
	items map[string]model.AgentSubscription
}

func newMemoryPlannerSubscriptionStore() *memoryPlannerSubscriptionStore {
	return &memoryPlannerSubscriptionStore{items: map[string]model.AgentSubscription{}}
}
func (s *memoryPlannerSubscriptionStore) Create(_ context.Context, item model.AgentSubscription) error {
	s.items[item.ID] = item
	return nil
}
func (s *memoryPlannerSubscriptionStore) ListByEventType(_ context.Context, eventType string) ([]model.AgentSubscription, error) {
	out := make([]model.AgentSubscription, 0)
	for _, item := range s.items {
		if item.Enabled && item.EventType == eventType {
			out = append(out, item)
		}
	}
	return out, nil
}

type memoryPlannerExperimentStore struct{ items map[string]model.Experiment }

func newMemoryPlannerExperimentStore() *memoryPlannerExperimentStore {
	return &memoryPlannerExperimentStore{items: map[string]model.Experiment{}}
}
func (s *memoryPlannerExperimentStore) List(_ context.Context) ([]model.Experiment, error) {
	out := make([]model.Experiment, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	return out, nil
}
func (s *memoryPlannerExperimentStore) GetByID(_ context.Context, id string) (*model.Experiment, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}
func (s *memoryPlannerExperimentStore) Create(_ context.Context, item model.Experiment) error {
	s.items[item.ID] = item
	return nil
}
func (s *memoryPlannerExperimentStore) Update(_ context.Context, item model.Experiment) error {
	s.items[item.ID] = item
	return nil
}

type memoryPlannerExperimentSpecStore struct {
	items map[string][]model.ExperimentSpec
}

func newMemoryPlannerExperimentSpecStore() *memoryPlannerExperimentSpecStore {
	return &memoryPlannerExperimentSpecStore{items: map[string][]model.ExperimentSpec{}}
}
func (s *memoryPlannerExperimentSpecStore) ListByExperimentID(_ context.Context, experimentID string) ([]model.ExperimentSpec, error) {
	return append([]model.ExperimentSpec(nil), s.items[experimentID]...), nil
}
func (s *memoryPlannerExperimentSpecStore) GetLatestByExperimentID(_ context.Context, experimentID string) (*model.ExperimentSpec, error) {
	items := s.items[experimentID]
	if len(items) == 0 {
		return nil, nil
	}
	item := items[len(items)-1]
	return &item, nil
}
func (s *memoryPlannerExperimentSpecStore) Create(_ context.Context, item model.ExperimentSpec) error {
	s.items[item.ExperimentID] = append(s.items[item.ExperimentID], item)
	return nil
}

type fakePlannerIdeaReader struct{ items map[string]model.IdeaDetail }

func (f *fakePlannerIdeaReader) GetByID(_ context.Context, id string) (*model.IdeaDetail, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type fakePlannerDatasetReader struct {
	items map[string]model.DatasetAssetDetail
}

func (f *fakePlannerDatasetReader) GetByID(_ context.Context, id string) (*model.DatasetAssetDetail, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type fakePlannerBaselineReader struct {
	items map[string]model.BaselineDetail
}

func (f *fakePlannerBaselineReader) GetByID(_ context.Context, id string) (*model.BaselineDetail, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type fakePlannerServerLister struct{ items []model.Server }

func (f *fakePlannerServerLister) List(_ context.Context) ([]model.Server, error) {
	out := make([]model.Server, len(f.items))
	copy(out, f.items)
	return out, nil
}

type fakePlannerHeartbeatReader struct {
	items map[string][]model.ServerHeartbeat
}

func (f *fakePlannerHeartbeatReader) ListByServerID(_ context.Context, serverID string, _ int) ([]model.ServerHeartbeat, error) {
	return append([]model.ServerHeartbeat(nil), f.items[serverID]...), nil
}

type fakePlannerGPUSnapshotReader struct {
	items map[string][]model.GPUResourceSnapshot
}

func (f *fakePlannerGPUSnapshotReader) ListByServerID(_ context.Context, serverID string, _ int) ([]model.GPUResourceSnapshot, error) {
	return append([]model.GPUResourceSnapshot(nil), f.items[serverID]...), nil
}

type memoryPlannerAssetReader struct{ items map[string]model.DatasetAsset }

func (s *memoryPlannerAssetReader) GetByID(_ context.Context, id string) (*model.DatasetAsset, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type memoryPlannerIdeaCoreReader struct{ items map[string]model.Idea }

func (s *memoryPlannerIdeaCoreReader) GetByID(_ context.Context, id string) (*model.Idea, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type memoryPlannerBaselineCoreReader struct{ items map[string]model.Baseline }

func (s *memoryPlannerBaselineCoreReader) GetByID(_ context.Context, id string) (*model.Baseline, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type memoryPlannerArchiveReader struct{}

func (s *memoryPlannerArchiveReader) ListByDatasetAssetID(_ context.Context, _ string) ([]model.ResultArchive, error) {
	return []model.ResultArchive{}, nil
}

func requirePlannerPython(t *testing.T) string {
	t.Helper()
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available")
	}
	return python
}

func plannerPythonAgentsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "python_agents"))
}

func writePlannerFixtures(t *testing.T, workspaceRoot string) string {
	t.Helper()
	ideaDir := filepath.Join(workspaceRoot, "ideas", "pool", "idea_1")
	if err := os.MkdirAll(ideaDir, 0o755); err != nil {
		t.Fatalf("mkdir idea fixture: %v", err)
	}
	ideaPayload := map[string]any{
		"title":                      "Controlled Retrieval Controller",
		"description_md":             "Generate a protocol-aware retrieval controller.",
		"research_direction":         "multimodal retrieval",
		"target_dataset_refs":        []string{"dasset_1"},
		"dataset_eval_protocol_refs": []string{filepath.Join(workspaceRoot, "datasets", "dasset_1", "evalplan.json")},
		"innovation_type":            "methodology",
		"expected_advantage":         "Better reproducibility.",
		"risk_points":                []string{"May be too incremental."},
		"priority":                   70,
		"confidence":                 0.74,
	}
	raw, _ := json.Marshal(ideaPayload)
	if err := os.WriteFile(filepath.Join(ideaDir, "structured_idea.json"), raw, 0o644); err != nil {
		t.Fatalf("write idea fixture: %v", err)
	}

	datasetDir := filepath.Join(workspaceRoot, "datasets", "dasset_1")
	if err := os.MkdirAll(datasetDir, 0o755); err != nil {
		t.Fatalf("mkdir evalplan fixture: %v", err)
	}
	evalplanPath := filepath.Join(datasetDir, "evalplan.json")
	evalplanPayload := map[string]any{
		"dataset_asset_id": "dasset_1",
		"task_type":        "text",
		"eval_protocol_json": map[string]any{
			"metric_list": []string{"accuracy", "loss"},
		},
		"metric_schema_json": map[string]any{"primary_metric": "accuracy"},
		"split_strategy":     "train_validation_test",
		"baseline_id":        "baseline_1",
	}
	raw, _ = json.Marshal(evalplanPayload)
	if err := os.WriteFile(evalplanPath, raw, 0o644); err != nil {
		t.Fatalf("write evalplan fixture: %v", err)
	}
	return evalplanPath
}

func setupPlannerAgent(t *testing.T) (*Service, *baseservice.ExperimentService, *agentpipeline.Service, *memoryPlannerJobStore, string) {
	t.Helper()
	workspaceRoot := t.TempDir()
	evalplanPath := writePlannerFixtures(t, workspaceRoot)
	now := time.Now()

	jobStore := newMemoryPlannerJobStore()
	triggerStore := newMemoryPlannerTriggerStore()
	artifactStore := newMemoryPlannerArtifactStore()
	eventStore := newMemoryPlannerEventStore()
	subscriptionStore := newMemoryPlannerSubscriptionStore()
	expStore := newMemoryPlannerExperimentStore()
	specStore := newMemoryPlannerExperimentSpecStore()

	ideaReader := &fakePlannerIdeaReader{items: map[string]model.IdeaDetail{
		"idea_1": {
			Idea: model.Idea{
				ID:            "idea_1",
				Title:         "Controlled Retrieval Controller",
				DescriptionMD: "Generate a protocol-aware retrieval controller.",
				Status:        "draft",
				Priority:      70,
				Confidence:    0.74,
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			StructuredIdea: &model.StructuredIdeaPayload{
				Title:                   "Controlled Retrieval Controller",
				DescriptionMD:           "Generate a protocol-aware retrieval controller.",
				ResearchDirection:       "multimodal retrieval",
				TargetDatasetRefs:       []string{"dasset_1"},
				DatasetEvalProtocolRefs: []string{evalplanPath},
				InnovationType:          "methodology",
				ExpectedAdvantage:       "Better reproducibility.",
				RiskPoints:              []string{"May be too incremental."},
				Priority:                70,
				Confidence:              0.74,
			},
		},
	}}
	datasetReader := &fakePlannerDatasetReader{items: map[string]model.DatasetAssetDetail{
		"dasset_1": {
			Asset: model.DatasetAsset{
				ID:                "dasset_1",
				Name:              "Planner Dataset",
				TaskType:          "text",
				Status:            "active",
				SourceType:        "mrag_scan",
				LocalOrRemotePath: "/data/planner",
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
	}}
	baselineReader := &fakePlannerBaselineReader{items: map[string]model.BaselineDetail{
		"baseline_1": {
			Baseline: model.Baseline{
				ID:               "baseline_1",
				DatasetAssetID:   "dasset_1",
				Name:             "Planner Baseline",
				MetricSchemaJSON: map[string]any{"primary": "accuracy"},
				CreatedAt:        now,
				UpdatedAt:        now,
			},
			DatasetAsset: datasetReader.items["dasset_1"].Asset,
		},
	}}
	servers := &fakePlannerServerLister{items: []model.Server{
		{ID: "srv_1", Name: "shenzhenvlab", Status: "online"},
		{ID: "srv_mock", Name: "mock_server", Status: "online"},
	}}
	heartbeats := &fakePlannerHeartbeatReader{items: map[string][]model.ServerHeartbeat{
		"srv_1":    {{ServerID: "srv_1", Status: "online", HeartbeatAt: now}},
		"srv_mock": {{ServerID: "srv_mock", Status: "online", HeartbeatAt: now}},
	}}
	gpuSnapshots := &fakePlannerGPUSnapshotReader{items: map[string][]model.GPUResourceSnapshot{
		"srv_1":    {{ServerID: "srv_1", GPUIndex: 0, FreeMemMB: 32768, Utilization: 10, CapturedAt: now}},
		"srv_mock": {{ServerID: "srv_mock", GPUIndex: 0, FreeMemMB: 8192, Utilization: 30, CapturedAt: now}},
	}}

	pythonExec := requirePlannerPython(t)
	pythonDir := plannerPythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	pipelineSvc := agentpipeline.NewService(eventStore, subscriptionStore, jobStore, jobSvc, triggerSvc)

	experimentSvc := baseservice.NewExperimentService(
		expStore,
		specStore,
		&memoryPlannerAssetReader{items: map[string]model.DatasetAsset{"dasset_1": datasetReader.items["dasset_1"].Asset}},
		&memoryPlannerIdeaCoreReader{items: map[string]model.Idea{"idea_1": ideaReader.items["idea_1"].Idea}},
		&memoryPlannerBaselineCoreReader{items: map[string]model.Baseline{"baseline_1": baselineReader.items["baseline_1"].Baseline}},
		&memoryPlannerArchiveReader{},
		workspaceRoot,
	)
	plannerSvc := NewService(jobSvc, jobStore, triggerSvc, experimentSvc, ideaReader, datasetReader, baselineReader, servers, heartbeats, gpuSnapshots, pipelineSvc, workspaceRoot)
	triggerSvc.RegisterPostProcessor("planner", plannerSvc)
	return plannerSvc, experimentSvc, pipelineSvc, jobStore, workspaceRoot
}

func TestPlannerRunGeneratesExperimentPlan(t *testing.T) {
	plannerSvc, _, _, _, workspaceRoot := setupPlannerAgent(t)

	result, err := plannerSvc.Run(context.Background(), model.PlannerRunRequest{
		IdeaID:           "idea_1",
		DatasetAssetRefs: []string{"dasset_1"},
		EvalProtocolRefs: []string{filepath.Join(workspaceRoot, "datasets", "dasset_1", "evalplan.json")},
		BaselineRefs:     []string{"baseline_1"},
		ExecutionMode:    "mock",
	})
	if err != nil {
		t.Fatalf("planner run failed: %v", err)
	}
	if result.Experiment == nil || result.Plan == nil {
		t.Fatalf("expected experiment and plan")
	}
	if result.Plan.ResourceEstimate["preferred_server_name"] != "shenzhenvlab" {
		t.Fatalf("expected shenzhenvlab preferred server, got %#v", result.Plan.ResourceEstimate)
	}
	if _, err = os.Stat(filepath.Join(workspaceRoot, "experiments", result.Experiment.Experiment.ID, "plan.json")); err != nil {
		t.Fatalf("expected plan.json: %v", err)
	}
}

func TestPlannerPlanCanGenerateExperimentSpec(t *testing.T) {
	plannerSvc, experimentSvc, _, _, workspaceRoot := setupPlannerAgent(t)

	result, err := plannerSvc.Run(context.Background(), model.PlannerRunRequest{
		IdeaID:           "idea_1",
		DatasetAssetRefs: []string{"dasset_1"},
		EvalProtocolRefs: []string{filepath.Join(workspaceRoot, "datasets", "dasset_1", "evalplan.json")},
		BaselineRefs:     []string{"baseline_1"},
		ExecutionMode:    "mock",
	})
	if err != nil {
		t.Fatalf("planner run failed: %v", err)
	}
	specDetail, err := experimentSvc.GenerateSpec(context.Background(), result.Experiment.Experiment.ID)
	if err != nil {
		t.Fatalf("generate spec failed: %v", err)
	}
	if specDetail.Spec.TemplateType == "" {
		t.Fatalf("expected planner-backed template type")
	}
	plannerExtensions, ok := specDetail.Spec.SpecJSON["planner_extensions"].(map[string]interface{})
	if !ok || plannerExtensions["planner_agent"] == nil {
		t.Fatalf("expected planner_agent section in spec")
	}
}

func TestPlannerAutoFlowFromIdeaReadyEvent(t *testing.T) {
	plannerSvc, experimentSvc, pipelineSvc, jobStore, workspaceRoot := setupPlannerAgent(t)
	_, err := pipelineSvc.CreateSubscription(context.Background(), model.AgentSubscriptionCreateRequest{
		Name:            "test-planner-subscription",
		EventType:       "idea_ready",
		AgentType:       "planner",
		ExecutionMode:   "mock",
		ModelProvider:   "codex",
		ModelName:       "planner-default",
		PromptVersion:   "v1",
		OutputSchemaRef: "schemas/planner-output-v1.json",
		TriggerRule: map[string]any{
			"required_ref_types": []string{"idea", "dataset_asset"},
		},
		MaxRetries: 1,
	})
	if err != nil {
		t.Fatalf("create planner subscription failed: %v", err)
	}

	evalplanPath := filepath.Join(workspaceRoot, "datasets", "dasset_1", "evalplan.json")
	_, err = pipelineSvc.PublishEvent(context.Background(), model.AgentEventCreateRequest{
		EventType: "idea_ready",
		SourceRef: "idea:idea_1",
		InputRefs: []model.AgentInputRef{
			{RefType: "idea", RefID: "idea_1", RefPath: filepath.Join(workspaceRoot, "ideas", "pool", "idea_1", "structured_idea.json")},
			{RefType: "dataset_asset", RefID: "dasset_1", RefPath: evalplanPath},
			{RefType: "dataset_eval_protocol", RefPath: evalplanPath},
		},
		Payload: map[string]any{"idea_id": "idea_1"},
	})
	if err != nil {
		t.Fatalf("publish idea_ready failed: %v", err)
	}
	dispatched, err := pipelineSvc.DispatchReadyJobs(context.Background(), 10)
	if err != nil {
		t.Fatalf("dispatch planner jobs failed: %v", err)
	}
	if dispatched == 0 {
		t.Fatalf("expected dispatched planner job")
	}
	items, err := experimentSvc.List(context.Background())
	if err != nil {
		t.Fatalf("list experiments failed: %v", err)
	}
	if len(items) == 0 {
		_ = jobStore
		t.Fatalf("expected experiment after planner auto flow")
	}
	planResp, err := plannerSvc.GetPlan(context.Background(), items[0].ID)
	if err != nil {
		t.Fatalf("get plan failed: %v", err)
	}
	if planResp == nil || planResp.Plan.ResourceEstimate["preferred_server_name"] == nil {
		t.Fatalf("expected resource estimate in persisted plan")
	}
}
