package codingagent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/resultcompare"
	"mrag-platform/backend/go/internal/runner"
	"mrag-platform/backend/go/internal/scheduler"
	svc "mrag-platform/backend/go/internal/service"
	"mrag-platform/backend/go/internal/traintemplate"
)

type codingTestExperimentStore struct {
	items map[string]model.Experiment
}

func newCodingTestExperimentStore() *codingTestExperimentStore {
	return &codingTestExperimentStore{items: map[string]model.Experiment{}}
}

func (s *codingTestExperimentStore) List(_ context.Context) ([]model.Experiment, error) {
	items := make([]model.Experiment, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *codingTestExperimentStore) GetByID(_ context.Context, id string) (*model.Experiment, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *codingTestExperimentStore) Create(_ context.Context, item model.Experiment) error {
	s.items[item.ID] = item
	return nil
}

func (s *codingTestExperimentStore) Update(_ context.Context, item model.Experiment) error {
	s.items[item.ID] = item
	return nil
}

type codingTestSpecStore struct {
	byExperiment map[string][]model.ExperimentSpec
	byID         map[string]model.ExperimentSpec
}

func newCodingTestSpecStore() *codingTestSpecStore {
	return &codingTestSpecStore{byExperiment: map[string][]model.ExperimentSpec{}, byID: map[string]model.ExperimentSpec{}}
}

func (s *codingTestSpecStore) ListByExperimentID(_ context.Context, experimentID string) ([]model.ExperimentSpec, error) {
	return append([]model.ExperimentSpec{}, s.byExperiment[experimentID]...), nil
}

func (s *codingTestSpecStore) GetLatestByExperimentID(_ context.Context, experimentID string) (*model.ExperimentSpec, error) {
	items := s.byExperiment[experimentID]
	if len(items) == 0 {
		return nil, nil
	}
	item := items[len(items)-1]
	return &item, nil
}

func (s *codingTestSpecStore) GetByID(_ context.Context, id string) (*model.ExperimentSpec, error) {
	item, ok := s.byID[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *codingTestSpecStore) Create(_ context.Context, item model.ExperimentSpec) error {
	s.byExperiment[item.ExperimentID] = append(s.byExperiment[item.ExperimentID], item)
	s.byID[item.ID] = item
	return nil
}

type codingTestDatasetAssetReader struct {
	items map[string]model.DatasetAsset
}

func (s *codingTestDatasetAssetReader) GetByID(_ context.Context, id string) (*model.DatasetAsset, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type codingTestIdeaBaseReader struct {
	items map[string]model.Idea
}

func (s *codingTestIdeaBaseReader) GetByID(_ context.Context, id string) (*model.Idea, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type codingTestIdeaDetailReader struct {
	items map[string]model.IdeaDetail
}

func (s *codingTestIdeaDetailReader) GetByID(_ context.Context, id string) (*model.IdeaDetail, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type codingTestBaselineStore struct {
	items map[string]model.Baseline
}

func (s *codingTestBaselineStore) GetByID(_ context.Context, id string) (*model.Baseline, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type codingTestArchiveStore struct {
	items map[string]model.ResultArchive
	files map[string][]model.ArchiveFile
}

func newCodingTestArchiveStore() *codingTestArchiveStore {
	return &codingTestArchiveStore{items: map[string]model.ResultArchive{}, files: map[string][]model.ArchiveFile{}}
}

func (s *codingTestArchiveStore) List(_ context.Context) ([]model.ResultArchive, error) {
	items := make([]model.ResultArchive, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *codingTestArchiveStore) GetByID(_ context.Context, id string) (*model.ResultArchive, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *codingTestArchiveStore) Create(_ context.Context, item model.ResultArchive) error {
	s.items[item.ID] = item
	return nil
}

func (s *codingTestArchiveStore) Update(_ context.Context, item model.ResultArchive) error {
	s.items[item.ID] = item
	return nil
}

func (s *codingTestArchiveStore) AddFile(_ context.Context, item model.ArchiveFile) error {
	s.files[item.ArchiveID] = append(s.files[item.ArchiveID], item)
	return nil
}

func (s *codingTestArchiveStore) ListFiles(_ context.Context, archiveID string) ([]model.ArchiveFile, error) {
	return append([]model.ArchiveFile{}, s.files[archiveID]...), nil
}

func (s *codingTestArchiveStore) ListByDatasetAssetID(_ context.Context, datasetAssetID string) ([]model.ResultArchive, error) {
	items := make([]model.ResultArchive, 0)
	for _, item := range s.items {
		if item.DatasetAssetID == datasetAssetID {
			items = append(items, item)
		}
	}
	return items, nil
}

type codingTestComparisonStore struct {
	items []model.ResultComparison
}

func (s *codingTestComparisonStore) ListByExperimentID(_ context.Context, experimentID string) ([]model.ResultComparison, error) {
	out := make([]model.ResultComparison, 0)
	for _, item := range s.items {
		if item.ExperimentID == experimentID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *codingTestComparisonStore) Create(_ context.Context, item model.ResultComparison) error {
	s.items = append(s.items, item)
	return nil
}

type codingTestRunStore struct {
	items            map[string]model.ExperimentRun
	experimentCounts map[string]int
}

func newCodingTestRunStore() *codingTestRunStore {
	return &codingTestRunStore{items: map[string]model.ExperimentRun{}, experimentCounts: map[string]int{}}
}

func (s *codingTestRunStore) GetByID(_ context.Context, id string) (*model.ExperimentRun, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *codingTestRunStore) Create(_ context.Context, item model.ExperimentRun) error {
	s.items[item.ID] = item
	s.experimentCounts[item.ExperimentID]++
	return nil
}

func (s *codingTestRunStore) Update(_ context.Context, item model.ExperimentRun) error {
	s.items[item.ID] = item
	return nil
}

func (s *codingTestRunStore) CountByExperimentID(_ context.Context, experimentID string) (int, error) {
	return s.experimentCounts[experimentID], nil
}

func (s *codingTestRunStore) CountActiveByServerID(_ context.Context, serverID string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.AssignedServerID != serverID {
			continue
		}
		switch item.RunStatus {
		case "queued", "scheduled", "preparing", "running":
			count++
		}
	}
	return count, nil
}

type codingTestDecisionStore struct {
	items map[string]model.SchedulerDecision
}

func (s *codingTestDecisionStore) Create(_ context.Context, item model.SchedulerDecision) error {
	s.items[item.RunID] = item
	return nil
}

func (s *codingTestDecisionStore) GetLatestByRunID(_ context.Context, runID string) (*model.SchedulerDecision, error) {
	item, ok := s.items[runID]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type codingTestServerStore struct {
	items map[string]model.Server
}

func (s *codingTestServerStore) List(_ context.Context) ([]model.Server, error) {
	out := make([]model.Server, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	return out, nil
}

func (s *codingTestServerStore) GetByIDWithSecrets(_ context.Context, id string) (*model.Server, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type codingTestHeartbeatReader struct {
	items map[string][]model.ServerHeartbeat
}

func (s *codingTestHeartbeatReader) ListByServerID(_ context.Context, serverID string, _ int) ([]model.ServerHeartbeat, error) {
	return append([]model.ServerHeartbeat{}, s.items[serverID]...), nil
}

type codingTestGPUSnapshotReader struct {
	items map[string][]model.GPUResourceSnapshot
}

func (s *codingTestGPUSnapshotReader) ListByServerID(_ context.Context, serverID string, _ int) ([]model.GPUResourceSnapshot, error) {
	return append([]model.GPUResourceSnapshot{}, s.items[serverID]...), nil
}

type codingTestRunLogStore struct {
	items []model.RunLog
}

func (s *codingTestRunLogStore) Add(_ context.Context, item model.RunLog) error {
	s.items = append(s.items, item)
	return nil
}

func (s *codingTestRunLogStore) ListByRunID(_ context.Context, runID string) ([]model.RunLog, error) {
	out := make([]model.RunLog, 0)
	for _, item := range s.items {
		if item.RunID == runID {
			out = append(out, item)
		}
	}
	return out, nil
}

type codingTestSSHGateway struct{}

func (g *codingTestSSHGateway) Mode() string { return "mock" }

func (g *codingTestSSHGateway) Probe(context.Context, *model.Server) (*model.ServerConnectionTestResult, error) {
	return &model.ServerConnectionTestResult{}, nil
}

func (g *codingTestSSHGateway) Exec(_ context.Context, _ *model.Server, req svc.SSHExecRequest) (*svc.SSHExecResult, error) {
	switch req.Purpose {
	case "experiment_run_prepare", "experiment_run_upload":
		return &svc.SSHExecResult{ExitCode: 0}, nil
	case "experiment_run_start":
		return &svc.SSHExecResult{ExitCode: 0, Stdout: "runner started"}, nil
	case "experiment_run_read_file":
		command := ""
		if len(req.RemoteCommand) > 0 {
			command = req.RemoteCommand[len(req.RemoteCommand)-1]
		}
		switch {
		case strings.Contains(command, "metrics.json"):
			return &svc.SSHExecResult{ExitCode: 0, Stdout: `{"primary_metric":"accuracy","values":{"accuracy":0.88,"loss":0.12}}`}, nil
		case strings.Contains(command, "result.md"):
			return &svc.SSHExecResult{ExitCode: 0, Stdout: "# Mock Result\n\nAccuracy improved.\n"}, nil
		case strings.Contains(command, "stdout.log"):
			return &svc.SSHExecResult{ExitCode: 0, Stdout: "[mock-train] completed\n[mock-eval] collected metrics\n"}, nil
		case strings.Contains(command, "stderr.log"):
			return &svc.SSHExecResult{ExitCode: 0, Stdout: ""}, nil
		}
	}
	return &svc.SSHExecResult{ExitCode: 0}, nil
}

type codingTestJobRecorder struct {
	lastCreate *model.AgentJobCreateRequest
	lastUpdate *model.AgentJob
}

func (r *codingTestJobRecorder) Create(_ context.Context, req model.AgentJobCreateRequest) (*model.AgentJob, error) {
	copyReq := req
	r.lastCreate = &copyReq
	return &model.AgentJob{
		ID:                "job_coding_1",
		AgentType:         req.AgentType,
		ExecutionMode:     req.ExecutionMode,
		ModelProvider:     req.ModelProvider,
		ModelName:         req.ModelName,
		PromptVersion:     req.PromptVersion,
		InputRefs:         append([]model.AgentInputRef{}, req.InputRefs...),
		OutputSchemaRef:   req.OutputSchemaRef,
		SkillRefs:         append([]string{}, req.SkillRefs...),
		ToolRefs:          append([]string{}, req.ToolRefs...),
		MemoryRefs:        append([]string{}, req.MemoryRefs...),
		Metadata:          req.Metadata,
		NormalizedPayload: map[string]any{},
		Status:            "registered",
	}, nil
}

func (r *codingTestJobRecorder) Update(_ context.Context, item model.AgentJob) error {
	copyItem := item
	r.lastUpdate = &copyItem
	return nil
}

type codingTestTrigger struct {
	job *model.AgentJob
}

func (t *codingTestTrigger) Trigger(_ context.Context, _ string, _ model.AgentJobTriggerRequest) (*model.AgentJob, error) {
	copyJob := *t.job
	return &copyJob, nil
}

func TestCodingServiceRunCreatesTemplateBoundJob(t *testing.T) {
	env := newCodingServiceTestEnv(t)

	jobRecorder := &codingTestJobRecorder{}
	trigger := &codingTestTrigger{
		job: &model.AgentJob{
			ID:        "job_coding_1",
			AgentType: "coding",
			Status:    "succeeded",
			Metadata: map[string]any{
				"experiment_id": env.experimentID,
				"idea_id":       "idea_1",
			},
			NormalizedPayload: map[string]any{
				"code_patch_manifest": []any{
					map[string]any{"patch_type": "spec_override", "target": "spec.hyperparams", "action": "merge", "summary": "Shrink schedule"},
				},
				"execution_result_ref":  "",
				"metrics_summary":       map[string]any{"primary_metric": "accuracy", "status": "pending"},
				"evaluation_summary_md": "pending",
			},
		},
	}

	codingSvc := NewService(
		jobRecorder,
		jobRecorder,
		trigger,
		env.experimentSvc,
		env.ideaDetailReader,
		env.specStore,
		nil,
		nil,
		nil,
		env.templateSvc,
		env.workspaceRoot,
		env.templateRoot,
	)

	result, err := codingSvc.Run(context.Background(), model.CodingRunRequest{
		ExperimentID:     env.experimentID,
		TrainTemplateRef: "mock_train_template",
		ExecutionMode:    "mock",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result == nil || result.Job == nil || result.Job.ID == "" {
		t.Fatalf("expected coding job result")
	}
	if jobRecorder.lastCreate == nil {
		t.Fatalf("expected job create request")
	}
	if jobRecorder.lastCreate.OutputSchemaRef != codingOutputSchemaRef {
		t.Fatalf("unexpected schema ref: %s", jobRecorder.lastCreate.OutputSchemaRef)
	}
	refTypes := map[string]bool{}
	for _, ref := range jobRecorder.lastCreate.InputRefs {
		refTypes[ref.RefType] = true
	}
	for _, required := range []string{"experiment", "idea", "experiment_plan", "experiment_spec", "train_template", "dataset_eval_protocol"} {
		if !refTypes[required] {
			t.Fatalf("expected input ref type %s", required)
		}
	}
}

func TestCodingPostProcessRunsTemplateAndCreatesComparison(t *testing.T) {
	env := newCodingServiceTestEnv(t)

	jobRecorder := &codingTestJobRecorder{}
	codingSvc := NewService(
		nil,
		jobRecorder,
		nil,
		env.experimentSvc,
		env.ideaDetailReader,
		env.specStore,
		env.schedulerSvc,
		env.runnerSvc,
		env.compareSvc,
		env.templateSvc,
		env.workspaceRoot,
		env.templateRoot,
	)

	job := &model.AgentJob{
		ID:        "job_coding_pp_1",
		AgentType: "coding",
		Status:    "succeeded",
		InputRefs: []model.AgentInputRef{
			{RefType: "experiment", RefID: env.experimentID},
			{RefType: "idea", RefID: "idea_1"},
		},
		Metadata: map[string]any{
			"experiment_id": env.experimentID,
			"idea_id":       "idea_1",
		},
		NormalizedPayload: map[string]any{
			"code_patch_manifest": []any{
				map[string]any{
					"patch_type": "spec_override",
					"target":     "spec.hyperparams",
					"action":     "merge",
					"value":      map[string]any{"epochs": 1, "batch_size": 4},
					"summary":    "Shrink schedule for first controlled run.",
				},
				map[string]any{
					"patch_type": "config_file",
					"target":     "coding/spec_overrides.json",
					"action":     "write",
					"value":      map[string]any{"hyperparams": map[string]any{"epochs": 1, "batch_size": 4}},
					"summary":    "Persist overrides.",
				},
			},
			"spec_overrides": map[string]any{
				"train_template_type": "mock_train_template",
				"hyperparams":         map[string]any{"epochs": 1, "batch_size": 4},
				"planner_extensions": map[string]any{
					"coding_agent": map[string]any{
						"mode":         "template_bound_v1",
						"generated_by": "coding_agent",
					},
				},
			},
			"execution_result_ref":  "",
			"metrics_summary":       map[string]any{"primary_metric": "accuracy", "status": "pending"},
			"evaluation_summary_md": "Coding agent generated template-bound overrides.",
		},
	}

	if err := codingSvc.PostProcess(context.Background(), job); err != nil {
		t.Fatalf("PostProcess returned error: %v", err)
	}
	if jobRecorder.lastUpdate == nil {
		t.Fatalf("expected postprocess to update job")
	}
	runID := stringValue(jobRecorder.lastUpdate.NormalizedPayload["run_id"])
	if runID == "" {
		t.Fatalf("expected run_id in normalized payload")
	}
	run, err := env.runStore.GetByID(context.Background(), runID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if run == nil || run.RunStatus != "succeeded" {
		t.Fatalf("expected succeeded run, got %#v", run)
	}
	metrics := mapValue(run.ResultJSON["metrics"])
	if stringValue(metrics["primary_metric"]) != "accuracy" {
		t.Fatalf("expected primary metric accuracy, got %#v", metrics)
	}
	if len(env.comparisonStore.items) == 0 {
		t.Fatalf("expected comparison records")
	}
	if len(env.archiveStore.items) == 0 {
		t.Fatalf("expected auto result archive")
	}
	latestSpec, err := env.specStore.GetLatestByExperimentID(context.Background(), env.experimentID)
	if err != nil {
		t.Fatalf("GetLatestByExperimentID returned error: %v", err)
	}
	if latestSpec == nil || latestSpec.Version < 2 {
		t.Fatalf("expected patched spec version >= 2")
	}
	hyperparams := mapValue(latestSpec.SpecJSON["hyperparams"])
	if hyperparams["epochs"] != 1 {
		t.Fatalf("expected patched epochs=1, got %#v", hyperparams)
	}
	codingDir := filepath.Join(env.workspaceRoot, "experiments", env.experimentID, "coding")
	for _, rel := range []string{"patch_manifest.json", "spec_overrides.json", "patched_spec.json", "evaluation_summary.md"} {
		if _, err := os.Stat(filepath.Join(codingDir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(stringValue(jobRecorder.lastUpdate.NormalizedPayload["execution_result_ref"])); err != nil {
		t.Fatalf("expected execution_result_ref file: %v", err)
	}
	if comparisonCount := len(env.comparisonStore.items); comparisonCount < 2 {
		t.Fatalf("expected at least 2 comparisons, got %d", comparisonCount)
	}
}

type codingServiceTestEnv struct {
	workspaceRoot    string
	templateRoot     string
	experimentID     string
	experimentSvc    *svc.ExperimentService
	templateSvc      *traintemplate.Service
	schedulerSvc     *scheduler.Service
	runnerSvc        *runner.Service
	compareSvc       *resultcompare.Service
	specStore        *codingTestSpecStore
	runStore         *codingTestRunStore
	archiveStore     *codingTestArchiveStore
	comparisonStore  *codingTestComparisonStore
	ideaDetailReader *codingTestIdeaDetailReader
}

func newCodingServiceTestEnv(t *testing.T) *codingServiceTestEnv {
	t.Helper()

	workspaceRoot := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	templateRoot := filepath.Clean(filepath.Join(wd, "..", "..", "..", "python_templates"))

	now := time.Now()
	expStore := newCodingTestExperimentStore()
	specStore := newCodingTestSpecStore()
	assetReader := &codingTestDatasetAssetReader{items: map[string]model.DatasetAsset{
		"dasset_1": {
			ID:                "dasset_1",
			Name:              "Demo Dataset",
			TaskType:          "text",
			Status:            "active",
			SourceType:        "manual",
			LocalOrRemotePath: "/data/demo",
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}}
	ideaBaseReader := &codingTestIdeaBaseReader{items: map[string]model.Idea{
		"idea_1": {
			ID:            "idea_1",
			Title:         "Template-Bound Training",
			DescriptionMD: "Use the shared template only.",
			Status:        "draft",
			SourceType:    "agent",
			Priority:      8,
			Confidence:    0.72,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}}
	ideaDetailReader := &codingTestIdeaDetailReader{items: map[string]model.IdeaDetail{
		"idea_1": {
			Idea: ideaBaseReader.items["idea_1"],
			StructuredIdea: &model.StructuredIdeaPayload{
				Title:             "Template-Bound Training",
				DescriptionMD:     "Use the shared template only.",
				ResearchDirection: "controlled training",
				InnovationType:    "workflow",
				ExpectedAdvantage: "Keeps the run auditable.",
				TargetDatasetRefs: []string{"dasset_1"},
				Priority:          8,
				Confidence:        0.72,
			},
		},
	}}
	baselineStore := &codingTestBaselineStore{items: map[string]model.Baseline{
		"baseline_1": {
			ID:             "baseline_1",
			DatasetAssetID: "dasset_1",
			Name:           "Baseline Accuracy",
			MetricSchemaJSON: map[string]any{
				"primary": "accuracy",
			},
			ResultJSON: map[string]any{
				"accuracy": 0.75,
				"loss":     0.32,
			},
			SourceType: "manual",
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}}
	archiveStore := newCodingTestArchiveStore()
	archiveStore.items["archive_prev_1"] = model.ResultArchive{
		ID:             "archive_prev_1",
		Title:          "Historic Result",
		DatasetAssetID: "dasset_1",
		MetricJSON: map[string]any{
			"accuracy": 0.80,
			"loss":     0.25,
		},
		Status:    "archived",
		CreatedAt: now,
		UpdatedAt: now,
	}

	experimentSvc := svc.NewExperimentService(expStore, specStore, assetReader, ideaBaseReader, baselineStore, archiveStore, workspaceRoot)
	created, err := experimentSvc.Create(context.Background(), model.ExperimentCreateRequest{
		DatasetAssetID: "dasset_1",
		IdeaID:         "idea_1",
		BaselineID:     "baseline_1",
		Title:          "Coding Agent Demo",
		Priority:       8,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err = experimentSvc.GenerateSpec(context.Background(), created.Experiment.ID); err != nil {
		t.Fatalf("GenerateSpec returned error: %v", err)
	}

	expDir := filepath.Join(workspaceRoot, "experiments", created.Experiment.ID)
	planDoc := model.ExperimentPlanDocument{
		ExperimentID:      created.Experiment.ID,
		IdeaID:            "idea_1",
		DatasetAssetID:    "dasset_1",
		BaselineID:        "baseline_1",
		TrainTemplateType: "mock_train_template",
		ExperimentPlanJSON: map[string]any{
			"idea_id":          "idea_1",
			"dataset_asset_id": "dasset_1",
			"baseline_id":      "baseline_1",
			"selected_server": map[string]any{
				"server_name":      "shenzhenvlab",
				"best_free_mem_mb": 32768,
			},
		},
		ResourceEstimate: map[string]any{"gpu_count": 1, "estimated_hours": 1},
		RunSequence:      []string{"validate_inputs", "generate_experiment_spec", "queue_experiment", "schedule_run"},
		SuccessCriteria:  map[string]any{"required_metrics": []any{"accuracy", "loss"}},
		FallbackPlan:     map[string]any{"fallback_server_name": "mock_server"},
		PlanPath:         filepath.Join(expDir, "plan.json"),
		GeneratedAt:      now,
	}
	rawPlan, err := json.MarshalIndent(planDoc, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent returned error: %v", err)
	}
	if err = os.WriteFile(filepath.Join(expDir, "plan.json"), rawPlan, 0o644); err != nil {
		t.Fatalf("WriteFile plan.json returned error: %v", err)
	}

	datasetDir := filepath.Join(workspaceRoot, "datasets", "dasset_1")
	if err = os.MkdirAll(datasetDir, 0o755); err != nil {
		t.Fatalf("MkdirAll dataset dir returned error: %v", err)
	}
	evalPlan := map[string]any{
		"eval_protocol_json": map[string]any{
			"task_type":        "classification",
			"metric_list":      []any{"accuracy", "loss"},
			"evaluation_steps": []any{"run", "collect"},
		},
		"split_strategy": "train_dev_test",
	}
	rawEvalPlan, err := json.MarshalIndent(evalPlan, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent eval plan returned error: %v", err)
	}
	if err = os.WriteFile(filepath.Join(datasetDir, "evalplan.json"), rawEvalPlan, 0o644); err != nil {
		t.Fatalf("WriteFile evalplan.json returned error: %v", err)
	}

	templateSvc := traintemplate.NewService(templateRoot, workspaceRoot)
	runStore := newCodingTestRunStore()
	decisionStore := &codingTestDecisionStore{items: map[string]model.SchedulerDecision{}}
	serverStore := &codingTestServerStore{items: map[string]model.Server{
		"srv_1": {
			ID:          "srv_1",
			Name:        "shenzhenvlab",
			Status:      "online",
			TaskWorkdir: "/remote/work",
		},
	}}
	heartbeatReader := &codingTestHeartbeatReader{items: map[string][]model.ServerHeartbeat{
		"srv_1": {{
			ServerID:    "srv_1",
			Status:      "online",
			HeartbeatAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}},
	}}
	gpuSnapshotReader := &codingTestGPUSnapshotReader{items: map[string][]model.GPUResourceSnapshot{
		"srv_1": {{
			ServerID:    "srv_1",
			CapturedAt:  now,
			GPUIndex:    0,
			Name:        "A100",
			FreeMemMB:   32768,
			Utilization: 8,
			CreatedAt:   now,
			UpdatedAt:   now,
		}},
	}}
	logStore := &codingTestRunLogStore{}
	resultArchiveSvc := svc.NewResultArchiveService(archiveStore, assetReader, ideaBaseReader, workspaceRoot)
	comparisonStore := &codingTestComparisonStore{}
	compareSvc := resultcompare.NewService(runStore, expStore, baselineStore, archiveStore, comparisonStore, resultArchiveSvc, workspaceRoot)
	schedulerSvc := scheduler.NewService(expStore, specStore, runStore, decisionStore, serverStore, heartbeatReader, gpuSnapshotReader, workspaceRoot)
	runnerSvc := runner.NewService(runStore, expStore, specStore, serverStore, logStore, &codingTestSSHGateway{}, templateSvc, compareSvc, "/remote/work")

	return &codingServiceTestEnv{
		workspaceRoot:    workspaceRoot,
		templateRoot:     templateRoot,
		experimentID:     created.Experiment.ID,
		experimentSvc:    experimentSvc,
		templateSvc:      templateSvc,
		schedulerSvc:     schedulerSvc,
		runnerSvc:        runnerSvc,
		compareSvc:       compareSvc,
		specStore:        specStore,
		runStore:         runStore,
		archiveStore:     archiveStore,
		comparisonStore:  comparisonStore,
		ideaDetailReader: ideaDetailReader,
	}
}
