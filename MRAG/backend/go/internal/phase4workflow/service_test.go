package phase4workflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type workflowDataStore struct {
	datasets  map[string]model.Phase4DatasetProfile
	contexts  map[string]model.Phase4ReaderContext
	ideas     map[string]model.Phase4Idea
	runs      map[string]model.Phase4RunManifest
	reports   map[string]model.Phase4StructuredReportRecord
	workflows map[string]model.Phase4Workflow
	actions   []model.Phase4WorkflowAction
	counter   int
}

func newWorkflowDataStore() *workflowDataStore {
	now := time.Now()
	return &workflowDataStore{
		datasets: map[string]model.Phase4DatasetProfile{
			"p4ds_workflow": {
				ID:          "p4ds_workflow",
				DatasetName: "VisDoM",
				TaskType:    "page_level_retrieval",
				ServerID:    "srv_shenzhenvlab",
				ServerPath:  "/home/bzli/mrag/datasets/visdom_subset",
				Status:      model.Phase4DatasetProfileStatusActive,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
		contexts: map[string]model.Phase4ReaderContext{
			"p4ctx_workflow": {
				ID:               "p4ctx_workflow",
				DatasetProfileID: "p4ds_workflow",
				Title:            "VisDoM Reader Context",
				Summary:          "Reader summary",
				TaskDefinition:   "Retrieve the correct page for each visual question.",
				SourceIDs:        []string{"src_1", "src_2"},
				StructuredContext: map[string]any{
					"likely_strong_baselines": []any{"hybrid lexical dense"},
				},
				Status:    model.Phase4ReaderContextStatusReady,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		ideas: map[string]model.Phase4Idea{
			"p4idea_1": {
				ID:                "p4idea_1",
				DatasetProfileID:  "p4ds_workflow",
				ReaderContextID:   "p4ctx_workflow",
				Title:             "Template-Debias Contrastive Retrieval",
				ProblemDefinition: "Reduce template bias in page retrieval.",
				CoreMethod:        "Template debias contrastive training.",
				Status:            model.Phase4IdeaStatusScored,
				ScoreSummary: map[string]any{
					"overallScore":         9.2,
					"rank":                 1,
					"recommendationTier":   "top3",
					"recommendationReason": "Strong fit.",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			"p4idea_2": {
				ID:                "p4idea_2",
				DatasetProfileID:  "p4ds_workflow",
				ReaderContextID:   "p4ctx_workflow",
				Title:             "Layout-Aware Hard Negative Mining",
				ProblemDefinition: "Improve near-duplicate discrimination.",
				CoreMethod:        "Hard negatives with layout cues.",
				Status:            model.Phase4IdeaStatusScored,
				ScoreSummary: map[string]any{
					"overallScore":         8.9,
					"rank":                 2,
					"recommendationTier":   "top3",
					"recommendationReason": "Good fit.",
				},
				CreatedAt: now.Add(time.Second),
				UpdatedAt: now.Add(time.Second),
			},
			"p4idea_3": {
				ID:                "p4idea_3",
				DatasetProfileID:  "p4ds_workflow",
				ReaderContextID:   "p4ctx_workflow",
				Title:             "Hybrid Lexical Dense Gating",
				ProblemDefinition: "Balance lexical and dense recall.",
				CoreMethod:        "Confidence-gated hybrid retriever.",
				Status:            model.Phase4IdeaStatusScored,
				ScoreSummary: map[string]any{
					"overallScore":         8.5,
					"rank":                 3,
					"recommendationTier":   "top3",
					"recommendationReason": "Balanced risk.",
				},
				CreatedAt: now.Add(2 * time.Second),
				UpdatedAt: now.Add(2 * time.Second),
			},
			"p4idea_rev_1": {
				ID:                "p4idea_rev_1",
				DatasetProfileID:  "p4ds_workflow",
				ReaderContextID:   "p4ctx_workflow",
				Title:             "Revision: OCR Noise Robust Retrieval",
				ProblemDefinition: "Address OCR-driven failures.",
				CoreMethod:        "OCR noise robust matching.",
				Status:            model.Phase4IdeaStatusScored,
				RevisionOfID:      "p4idea_1",
				LineageRootID:     "p4idea_1",
				LastFailureRunID:  "p4run_fail",
				ScoreSummary: map[string]any{
					"overallScore":         8.7,
					"rank":                 1,
					"recommendationTier":   "top3",
					"recommendationReason": "Directly addresses failure.",
				},
				CreatedAt: now.Add(3 * time.Second),
				UpdatedAt: now.Add(3 * time.Second),
			},
			"p4idea_rev_2": {
				ID:                "p4idea_rev_2",
				DatasetProfileID:  "p4ds_workflow",
				ReaderContextID:   "p4ctx_workflow",
				Title:             "Revision: Section-Aware Reweighting",
				ProblemDefinition: "Reduce retrieval misses from section bias.",
				CoreMethod:        "Section-aware reweighting.",
				Status:            model.Phase4IdeaStatusScored,
				RevisionOfID:      "p4idea_1",
				LineageRootID:     "p4idea_1",
				LastFailureRunID:  "p4run_fail",
				ScoreSummary: map[string]any{
					"overallScore":         8.4,
					"rank":                 2,
					"recommendationTier":   "top3",
					"recommendationReason": "Smaller change.",
				},
				CreatedAt: now.Add(4 * time.Second),
				UpdatedAt: now.Add(4 * time.Second),
			},
		},
		runs:      map[string]model.Phase4RunManifest{},
		reports:   map[string]model.Phase4StructuredReportRecord{},
		workflows: map[string]model.Phase4Workflow{},
		actions:   []model.Phase4WorkflowAction{},
	}
}

func (s *workflowDataStore) GetDatasetProfileByID(_ context.Context, id string) (*model.Phase4DatasetProfile, error) {
	item, ok := s.datasets[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *workflowDataStore) GetReaderContextByID(_ context.Context, id string) (*model.Phase4ReaderContext, error) {
	item, ok := s.contexts[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *workflowDataStore) ListIdeas(_ context.Context, datasetProfileID string, status string) ([]model.Phase4Idea, error) {
	items := make([]model.Phase4Idea, 0)
	for _, item := range s.ideas {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *workflowDataStore) GetIdeaByID(_ context.Context, id string) (*model.Phase4Idea, error) {
	item, ok := s.ideas[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *workflowDataStore) SelectIdea(_ context.Context, id string) (*model.Phase4Idea, error) {
	item, ok := s.ideas[id]
	if !ok {
		return nil, nil
	}
	item.Status = model.Phase4IdeaStatusSelected
	item.UpdatedAt = time.Now()
	s.ideas[id] = item
	copyItem := item
	return &copyItem, nil
}

func (s *workflowDataStore) GetRunManifestByID(_ context.Context, id string) (*model.Phase4RunManifest, error) {
	item, ok := s.runs[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *workflowDataStore) GetStructuredReportByID(_ context.Context, id string) (*model.Phase4StructuredReportRecord, error) {
	item, ok := s.reports[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *workflowDataStore) ListWorkflows(_ context.Context, datasetProfileID string, status string) ([]model.Phase4Workflow, error) {
	items := make([]model.Phase4Workflow, 0)
	for _, item := range s.workflows {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *workflowDataStore) GetWorkflowByID(_ context.Context, id string) (*model.Phase4Workflow, error) {
	item, ok := s.workflows[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *workflowDataStore) CreateWorkflow(_ context.Context, item model.Phase4Workflow) (*model.Phase4Workflow, error) {
	s.workflows[item.ID] = item
	copyItem := item
	return &copyItem, nil
}

func (s *workflowDataStore) UpdateWorkflow(_ context.Context, id string, item model.Phase4Workflow) (*model.Phase4Workflow, error) {
	s.workflows[id] = item
	copyItem := item
	return &copyItem, nil
}

func (s *workflowDataStore) ListWorkflowActions(_ context.Context, workflowID string) ([]model.Phase4WorkflowAction, error) {
	out := make([]model.Phase4WorkflowAction, 0)
	for _, item := range s.actions {
		if item.WorkflowID == workflowID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *workflowDataStore) CreateWorkflowAction(_ context.Context, item model.Phase4WorkflowAction) (*model.Phase4WorkflowAction, error) {
	s.counter++
	if item.ID == "" {
		item.ID = fmt.Sprintf("p4wfa_%d", s.counter)
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	item.UpdatedAt = item.CreatedAt
	s.actions = append(s.actions, item)
	copyItem := item
	return &copyItem, nil
}

type workflowReaderRunner struct {
	result *model.Phase4ReaderRunResult
	err    error
	calls  int
}

func (r *workflowReaderRunner) Run(_ context.Context, _ model.Phase4ReaderRunRequest) (*model.Phase4ReaderRunResult, error) {
	r.calls++
	return r.result, r.err
}

type workflowIdeaRunner struct {
	result *model.Phase4IdeaRunResult
	err    error
	calls  int
}

func (r *workflowIdeaRunner) Run(_ context.Context, _ model.Phase4IdeaRunRequest) (*model.Phase4IdeaRunResult, error) {
	r.calls++
	return r.result, r.err
}

type workflowCodingRunner struct {
	results []*model.Phase4CodingRunResult
	errs    []error
	calls   int
}

func (r *workflowCodingRunner) Run(_ context.Context, _ model.Phase4CodingRunRequest) (*model.Phase4CodingRunResult, error) {
	index := r.calls
	r.calls++
	if index < len(r.errs) && r.errs[index] != nil {
		return nil, r.errs[index]
	}
	if index < len(r.results) {
		return r.results[index], nil
	}
	return nil, fmt.Errorf("unexpected coding run call %d", r.calls)
}

type workflowWriterRunner struct {
	results []*model.Phase4WriterRunResult
	errs    []error
	calls   int
}

func (r *workflowWriterRunner) Run(_ context.Context, _ model.Phase4WriterRunRequest) (*model.Phase4WriterRunResult, error) {
	index := r.calls
	r.calls++
	if index < len(r.errs) && r.errs[index] != nil {
		return nil, r.errs[index]
	}
	if index < len(r.results) {
		return r.results[index], nil
	}
	return nil, fmt.Errorf("unexpected writer run call %d", r.calls)
}

type workflowJobStore struct {
	items map[string]model.AgentJob
}

func (s *workflowJobStore) GetByID(_ context.Context, id string) (*model.AgentJob, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type workflowEventRecorder struct {
	eventTypes []string
}

func (r *workflowEventRecorder) PublishEvent(_ context.Context, req model.AgentEventCreateRequest) (*model.AgentEvent, error) {
	r.eventTypes = append(r.eventTypes, req.EventType)
	return &model.AgentEvent{ID: fmt.Sprintf("evt_%d", len(r.eventTypes)), EventType: req.EventType}, nil
}

type workflowFixture struct {
	data   *workflowDataStore
	reader *workflowReaderRunner
	idea   *workflowIdeaRunner
	coding *workflowCodingRunner
	writer *workflowWriterRunner
	jobs   *workflowJobStore
	events *workflowEventRecorder
	svc    *Service
}

func workflowJobPointer(items map[string]model.AgentJob, id string) *model.AgentJob {
	item := items[id]
	copyItem := item
	return &copyItem
}

func workflowContextPointer(items map[string]model.Phase4ReaderContext, id string) *model.Phase4ReaderContext {
	item := items[id]
	copyItem := item
	return &copyItem
}

func workflowRunPointer(items map[string]model.Phase4RunManifest, id string) *model.Phase4RunManifest {
	item := items[id]
	copyItem := item
	return &copyItem
}

func workflowReportPointer(items map[string]model.Phase4StructuredReportRecord, id string) *model.Phase4StructuredReportRecord {
	item := items[id]
	copyItem := item
	return &copyItem
}

func newWorkflowFixture() *workflowFixture {
	data := newWorkflowDataStore()
	jobs := &workflowJobStore{
		items: map[string]model.AgentJob{
			"ajob_reader": {ID: "ajob_reader", AgentType: "reader_phase4", Status: "succeeded"},
			"ajob_idea":   {ID: "ajob_idea", AgentType: "idea_phase4", Status: "succeeded"},
			"ajob_code1":  {ID: "ajob_code1", AgentType: "coding_phase4", Status: "succeeded"},
			"ajob_code2":  {ID: "ajob_code2", AgentType: "coding_phase4", Status: "succeeded"},
			"ajob_writer": {ID: "ajob_writer", AgentType: "writer_phase4", Status: "succeeded"},
		},
	}
	reader := &workflowReaderRunner{result: &model.Phase4ReaderRunResult{
		Job:           workflowJobPointer(jobs.items, "ajob_reader"),
		ReaderContext: workflowContextPointer(data.contexts, "p4ctx_workflow"),
		ReaderSources: []model.Phase4ReaderSource{{ID: "src_1"}, {ID: "src_2"}},
	}}
	idea := &workflowIdeaRunner{result: &model.Phase4IdeaRunResult{
		Job: workflowJobPointer(jobs.items, "ajob_idea"),
		Ideas: []model.Phase4Idea{
			data.ideas["p4idea_1"],
			data.ideas["p4idea_2"],
			data.ideas["p4idea_3"],
		},
		TopRecommendations: []model.Phase4IdeaScoreView{
			{ID: "p4idea_1", Title: data.ideas["p4idea_1"].Title, Rank: 1},
			{ID: "p4idea_2", Title: data.ideas["p4idea_2"].Title, Rank: 2},
			{ID: "p4idea_3", Title: data.ideas["p4idea_3"].Title, Rank: 3},
		},
	}}
	events := &workflowEventRecorder{}
	f := &workflowFixture{
		data:   data,
		reader: reader,
		idea:   idea,
		coding: &workflowCodingRunner{},
		writer: &workflowWriterRunner{},
		jobs:   jobs,
		events: events,
	}
	f.svc = NewService(data, reader, idea, f.coding, f.writer, jobs, events)
	return f
}

func (f *workflowFixture) createWorkflow(t *testing.T) *model.Phase4WorkflowDetail {
	t.Helper()
	detail, err := f.svc.CreateWorkflow(context.Background(), model.Phase4WorkflowCreateRequest{
		DatasetProfileID: "p4ds_workflow",
		Reader: model.Phase4WorkflowReaderConfig{
			ExecutionMode: "mock",
		},
		Idea: model.Phase4WorkflowIdeaConfig{
			ExecutionMode: "mock",
		},
		Coding: model.Phase4WorkflowCodingConfig{
			ExecutionMode: "mock",
			RunnerMode:    "local_dummy",
		},
		Writing: model.Phase4WorkflowWritingConfig{
			ExecutionMode: "mock",
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkflow returned error: %v", err)
	}
	return detail
}

func TestCreateWorkflowAutoAdvancesToAwaitingSelection(t *testing.T) {
	fixture := newWorkflowFixture()

	detail := fixture.createWorkflow(t)
	if detail.Workflow == nil || detail.Workflow.Status != model.Phase4WorkflowStatusAwaitingSelection {
		t.Fatalf("expected awaiting_selection workflow, got %#v", detail.Workflow)
	}
	if detail.ReaderContext == nil || detail.ReaderContext.ID != "p4ctx_workflow" {
		t.Fatalf("expected persisted reader context, got %#v", detail.ReaderContext)
	}
	if len(detail.TopRecommendations) != 3 {
		t.Fatalf("expected 3 top recommendations, got %d", len(detail.TopRecommendations))
	}
	if detail.LatestJobs.Reader == nil || detail.LatestJobs.Idea == nil {
		t.Fatalf("expected reader and idea jobs in detail")
	}
	if len(detail.Timeline) < 5 {
		t.Fatalf("expected workflow timeline to include create + reader/idea actions, got %d", len(detail.Timeline))
	}
	if got := fixture.events.eventTypes; len(got) != 3 || got[0] != "phase4_workflow_started" || got[1] != "phase4_reader_ready" || got[2] != "phase4_idea_batch_ready" {
		t.Fatalf("unexpected event sequence: %#v", got)
	}
}

func TestSelectIdeaSuccessCompletesWorkflow(t *testing.T) {
	fixture := newWorkflowFixture()
	fixture.data.runs["p4run_success"] = model.Phase4RunManifest{
		ID:               "p4run_success",
		DatasetProfileID: "p4ds_workflow",
		IdeaID:           "p4idea_1",
		ReaderContextID:  "p4ctx_workflow",
		RunnerMode:       "local_dummy",
		Status:           model.Phase4RunStatusSucceeded,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	fixture.data.reports["p4report_success"] = model.Phase4StructuredReportRecord{
		ID:            "p4report_success",
		RunManifestID: "p4run_success",
		Title:         "Workflow Success Report",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	fixture.coding.results = []*model.Phase4CodingRunResult{{
		Job:         workflowJobPointer(fixture.jobs.items, "ajob_code1"),
		RunManifest: workflowRunPointer(fixture.data.runs, "p4run_success"),
	}}
	fixture.writer.results = []*model.Phase4WriterRunResult{{
		Job:    workflowJobPointer(fixture.jobs.items, "ajob_writer"),
		Report: workflowReportPointer(fixture.data.reports, "p4report_success"),
	}}

	create := fixture.createWorkflow(t)
	detail, err := fixture.svc.SelectIdea(context.Background(), create.Workflow.ID, model.Phase4WorkflowSelectIdeaRequest{IdeaID: "p4idea_1"})
	if err != nil {
		t.Fatalf("SelectIdea returned error: %v", err)
	}
	if detail.Workflow == nil || detail.Workflow.Status != model.Phase4WorkflowStatusCompleted {
		t.Fatalf("expected completed workflow, got %#v", detail.Workflow)
	}
	if detail.LatestReport == nil || detail.LatestReport.ID != "p4report_success" {
		t.Fatalf("expected latest report, got %#v", detail.LatestReport)
	}
	if detail.CurrentRunManifest == nil || detail.CurrentRunManifest.ID != "p4run_success" {
		t.Fatalf("expected current run manifest, got %#v", detail.CurrentRunManifest)
	}
	if detail.SelectedIdea == nil || detail.SelectedIdea.ID != "p4idea_1" {
		t.Fatalf("expected selected idea p4idea_1, got %#v", detail.SelectedIdea)
	}
	if fixture.coding.calls != 1 || fixture.writer.calls != 1 {
		t.Fatalf("expected one coding and one writing call, got coding=%d writer=%d", fixture.coding.calls, fixture.writer.calls)
	}
	if got := fixture.events.eventTypes[len(fixture.events.eventTypes)-2:]; len(got) != 2 || got[0] != "phase4_idea_selected" || got[1] != "phase4_workflow_completed" {
		t.Fatalf("unexpected tail events: %#v", got)
	}
}

func TestCodingTestFailedTransitionsToRevisionSelection(t *testing.T) {
	fixture := newWorkflowFixture()
	fixture.data.runs["p4run_fail"] = model.Phase4RunManifest{
		ID:               "p4run_fail",
		DatasetProfileID: "p4ds_workflow",
		IdeaID:           "p4idea_1",
		ReaderContextID:  "p4ctx_workflow",
		RunnerMode:       "local_dummy",
		Status:           model.Phase4RunStatusTestFailed,
		FailureFeedback:  map[string]any{"stage": "eval", "error": "Recall@5 below threshold"},
		ArtifactPaths: map[string]any{
			"revision_idea_ids":     []string{"p4idea_rev_1", "p4idea_rev_2"},
			"revision_top_idea_ids": []string{"p4idea_rev_1"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	fixture.coding.results = []*model.Phase4CodingRunResult{{
		Job:         workflowJobPointer(fixture.jobs.items, "ajob_code1"),
		RunManifest: workflowRunPointer(fixture.data.runs, "p4run_fail"),
	}}

	create := fixture.createWorkflow(t)
	detail, err := fixture.svc.SelectIdea(context.Background(), create.Workflow.ID, model.Phase4WorkflowSelectIdeaRequest{IdeaID: "p4idea_1"})
	if err != nil {
		t.Fatalf("SelectIdea returned error: %v", err)
	}
	if detail.Workflow == nil || detail.Workflow.Status != model.Phase4WorkflowStatusAwaitingRevisionSelect {
		t.Fatalf("expected awaiting_revision_selection, got %#v", detail.Workflow)
	}
	if detail.Workflow.CurrentRunManifestID != "p4run_fail" {
		t.Fatalf("expected failed run manifest id to persist, got %s", detail.Workflow.CurrentRunManifestID)
	}
	if len(detail.TopRecommendations) != 1 || detail.TopRecommendations[0].ID != "p4idea_rev_1" {
		t.Fatalf("expected revision top recommendation, got %#v", detail.TopRecommendations)
	}
	if detail.Workflow.LastError == "" {
		t.Fatalf("expected summarized last error")
	}
	if tail := fixture.events.eventTypes[len(fixture.events.eventTypes)-2:]; len(tail) != 2 || tail[0] != "phase4_idea_selected" || tail[1] != "phase4_coding_test_failed" {
		t.Fatalf("unexpected tail events: %#v", tail)
	}
}

func TestSelectRevisionReRunsCodingAndCompletes(t *testing.T) {
	fixture := newWorkflowFixture()
	fixture.data.runs["p4run_fail"] = model.Phase4RunManifest{
		ID:               "p4run_fail",
		DatasetProfileID: "p4ds_workflow",
		IdeaID:           "p4idea_1",
		ReaderContextID:  "p4ctx_workflow",
		RunnerMode:       "local_dummy",
		Status:           model.Phase4RunStatusTestFailed,
		FailureFeedback:  map[string]any{"stage": "train", "error": "nan loss"},
		ArtifactPaths: map[string]any{
			"revision_idea_ids":     []string{"p4idea_rev_1", "p4idea_rev_2"},
			"revision_top_idea_ids": []string{"p4idea_rev_1"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	fixture.data.runs["p4run_recovered"] = model.Phase4RunManifest{
		ID:               "p4run_recovered",
		DatasetProfileID: "p4ds_workflow",
		IdeaID:           "p4idea_rev_1",
		ReaderContextID:  "p4ctx_workflow",
		RunnerMode:       "local_dummy",
		Status:           model.Phase4RunStatusSucceeded,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	fixture.data.reports["p4report_recovered"] = model.Phase4StructuredReportRecord{
		ID:            "p4report_recovered",
		RunManifestID: "p4run_recovered",
		Title:         "Recovered Report",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	fixture.coding.results = []*model.Phase4CodingRunResult{
		{Job: workflowJobPointer(fixture.jobs.items, "ajob_code1"), RunManifest: workflowRunPointer(fixture.data.runs, "p4run_fail")},
		{Job: workflowJobPointer(fixture.jobs.items, "ajob_code2"), RunManifest: workflowRunPointer(fixture.data.runs, "p4run_recovered")},
	}
	fixture.writer.results = []*model.Phase4WriterRunResult{
		{Job: workflowJobPointer(fixture.jobs.items, "ajob_writer"), Report: workflowReportPointer(fixture.data.reports, "p4report_recovered")},
	}

	create := fixture.createWorkflow(t)
	failedDetail, err := fixture.svc.SelectIdea(context.Background(), create.Workflow.ID, model.Phase4WorkflowSelectIdeaRequest{IdeaID: "p4idea_1"})
	if err != nil {
		t.Fatalf("SelectIdea returned error: %v", err)
	}
	if failedDetail.Workflow.Status != model.Phase4WorkflowStatusAwaitingRevisionSelect {
		t.Fatalf("expected awaiting revision selection, got %s", failedDetail.Workflow.Status)
	}
	detail, err := fixture.svc.SelectRevision(context.Background(), failedDetail.Workflow.ID, model.Phase4WorkflowSelectIdeaRequest{IdeaID: "p4idea_rev_1"})
	if err != nil {
		t.Fatalf("SelectRevision returned error: %v", err)
	}
	if detail.Workflow == nil || detail.Workflow.Status != model.Phase4WorkflowStatusCompleted {
		t.Fatalf("expected completed workflow after revision, got %#v", detail.Workflow)
	}
	if detail.CurrentRunManifest == nil || detail.CurrentRunManifest.ID != "p4run_recovered" {
		t.Fatalf("expected recovered run manifest, got %#v", detail.CurrentRunManifest)
	}
	if detail.LatestReport == nil || detail.LatestReport.ID != "p4report_recovered" {
		t.Fatalf("expected recovered report, got %#v", detail.LatestReport)
	}
	if len(detail.Timeline) == 0 {
		t.Fatalf("expected workflow timeline")
	}
	foundFailedRun := false
	foundRecoveredRun := false
	for _, item := range detail.Timeline {
		if item.RunManifestID == "p4run_fail" {
			foundFailedRun = true
		}
		if item.RunManifestID == "p4run_recovered" {
			foundRecoveredRun = true
		}
	}
	if !foundFailedRun || !foundRecoveredRun {
		t.Fatalf("expected timeline to preserve failed and recovered run records: %#v", detail.Timeline)
	}
}

func TestRetryStageOnlyRetriesBlockedStage(t *testing.T) {
	fixture := newWorkflowFixture()
	fixture.data.runs["p4run_success"] = model.Phase4RunManifest{
		ID:               "p4run_success",
		DatasetProfileID: "p4ds_workflow",
		IdeaID:           "p4idea_1",
		ReaderContextID:  "p4ctx_workflow",
		RunnerMode:       "local_dummy",
		Status:           model.Phase4RunStatusSucceeded,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	fixture.data.reports["p4report_success"] = model.Phase4StructuredReportRecord{
		ID:            "p4report_success",
		RunManifestID: "p4run_success",
		Title:         "Success Report",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	fixture.coding.results = []*model.Phase4CodingRunResult{
		{Job: workflowJobPointer(fixture.jobs.items, "ajob_code1"), RunManifest: workflowRunPointer(fixture.data.runs, "p4run_success")},
	}
	fixture.writer.errs = []error{fmt.Errorf("writer storage timeout")}
	fixture.writer.results = []*model.Phase4WriterRunResult{
		nil,
		{Job: workflowJobPointer(fixture.jobs.items, "ajob_writer"), Report: workflowReportPointer(fixture.data.reports, "p4report_success")},
	}

	create := fixture.createWorkflow(t)
	blockedDetail, err := fixture.svc.SelectIdea(context.Background(), create.Workflow.ID, model.Phase4WorkflowSelectIdeaRequest{IdeaID: "p4idea_1"})
	if err != nil {
		t.Fatalf("SelectIdea returned error: %v", err)
	}
	if blockedDetail.Workflow == nil || blockedDetail.Workflow.Status != model.Phase4WorkflowStatusBlocked {
		t.Fatalf("expected blocked workflow after writing failure, got %#v", blockedDetail.Workflow)
	}
	if stringValue(blockedDetail.Workflow.Metadata["failed_stage"]) != model.Phase4WorkflowStageWriting {
		t.Fatalf("expected failed_stage=writing, got %#v", blockedDetail.Workflow.Metadata)
	}
	detail, err := fixture.svc.RetryStage(context.Background(), blockedDetail.Workflow.ID, model.Phase4WorkflowRetryStageRequest{})
	if err != nil {
		t.Fatalf("RetryStage returned error: %v", err)
	}
	if detail.Workflow == nil || detail.Workflow.Status != model.Phase4WorkflowStatusCompleted {
		t.Fatalf("expected completed workflow after retry, got %#v", detail.Workflow)
	}
	if fixture.coding.calls != 1 {
		t.Fatalf("expected coding stage not to rerun, got %d coding calls", fixture.coding.calls)
	}
	if fixture.writer.calls != 2 {
		t.Fatalf("expected writer stage to rerun once, got %d writer calls", fixture.writer.calls)
	}
	if detail.SelectedIdea == nil || detail.SelectedIdea.ID != "p4idea_1" {
		t.Fatalf("expected selected idea to stay unchanged, got %#v", detail.SelectedIdea)
	}
}
