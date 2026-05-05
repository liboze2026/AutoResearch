package writeragent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type writerTestJobRecorder struct {
	lastCreate *model.AgentJobCreateRequest
	lastUpdate *model.AgentJob
}

func (r *writerTestJobRecorder) Create(_ context.Context, req model.AgentJobCreateRequest) (*model.AgentJob, error) {
	copyReq := req
	r.lastCreate = &copyReq
	return &model.AgentJob{
		ID:                "job_writer_1",
		AgentType:         "writer",
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

func (r *writerTestJobRecorder) Update(_ context.Context, item model.AgentJob) error {
	copyItem := item
	r.lastUpdate = &copyItem
	return nil
}

type writerTestTrigger struct {
	job *model.AgentJob
}

func (t *writerTestTrigger) Trigger(_ context.Context, _ string, _ model.AgentJobTriggerRequest) (*model.AgentJob, error) {
	copyJob := *t.job
	return &copyJob, nil
}

type writerTestIdeaReader struct {
	items map[string]model.IdeaDetail
}

func (r *writerTestIdeaReader) GetByID(_ context.Context, id string) (*model.IdeaDetail, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type writerTestRunReader struct {
	items map[string]model.ExperimentRun
}

func (r *writerTestRunReader) GetRun(_ context.Context, id string) (*model.ExperimentRun, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

type writerTestComparisonReader struct {
	items map[string][]model.ResultComparison
}

func (r *writerTestComparisonReader) ListByExperimentID(_ context.Context, experimentID string) ([]model.ResultComparison, error) {
	return append([]model.ResultComparison{}, r.items[experimentID]...), nil
}

type writerTestArchiveUpdater struct {
	lastID  string
	lastReq *model.ResultArchiveUpdateRequest
}

func (u *writerTestArchiveUpdater) Update(_ context.Context, id string, req model.ResultArchiveUpdateRequest) (*model.ResultArchiveDetail, error) {
	copyReq := req
	u.lastID = id
	u.lastReq = &copyReq
	return &model.ResultArchiveDetail{Archive: model.ResultArchive{ID: id}}, nil
}

type writerTestEventPublisher struct {
	published []model.AgentEventCreateRequest
}

func (p *writerTestEventPublisher) PublishEvent(_ context.Context, req model.AgentEventCreateRequest) (*model.AgentEvent, error) {
	p.published = append(p.published, req)
	return &model.AgentEvent{ID: "evt_1", EventType: req.EventType}, nil
}

func TestWriterRunCreatesJobFromTemplateAndRefs(t *testing.T) {
	workspaceRoot := t.TempDir()
	templatePath := filepath.Join(workspaceRoot, "templates", "paper.md")
	if err := os.MkdirAll(filepath.Dir(templatePath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(templatePath, []byte("# Template\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	jobRecorder := &writerTestJobRecorder{}
	trigger := &writerTestTrigger{
		job: &model.AgentJob{
			ID:        "job_writer_1",
			AgentType: "writer",
			Status:    "succeeded",
			Metadata: map[string]any{
				"paper_template_ref":     templatePath,
				"idea_refs":              []string{"idea_1"},
				"experiment_result_refs": []string{"run_1"},
			},
			NormalizedPayload: map[string]any{
				"title":           "Demo Draft",
				"abstract":        "Demo abstract.",
				"introduction":    "Demo intro.",
				"method":          "Demo method.",
				"experiments":     "Demo experiments.",
				"conclusion":      "Demo conclusion.",
				"references_stub": []any{"[Ref 1] Demo"},
				"figure_plan":     []any{map[string]any{"figure_id": "fig_1", "title": "Demo Figure"}},
			},
		},
	}

	svc := NewService(
		jobRecorder,
		jobRecorder,
		trigger,
		&writerTestIdeaReader{items: map[string]model.IdeaDetail{
			"idea_1": {
				Idea: model.Idea{ID: "idea_1", Title: "Controlled Writing", Priority: 8, Confidence: 0.8},
				StructuredIdea: &model.StructuredIdeaPayload{
					ResearchDirection: "controlled writing",
					ExpectedAdvantage: "improve traceability",
				},
			},
		}},
		&writerTestRunReader{items: map[string]model.ExperimentRun{
			"run_1": {
				ID:           "run_1",
				ExperimentID: "exp_1",
				RunStatus:    "succeeded",
				ResultJSON: map[string]interface{}{
					"result_archive_id": "archive_1",
					"metrics": map[string]interface{}{
						"primary_metric": "accuracy",
						"values":         map[string]interface{}{"accuracy": 0.88},
					},
				},
			},
		}},
		&writerTestComparisonReader{items: map[string][]model.ResultComparison{
			"exp_1": {{ID: "cmp_1", ExperimentID: "exp_1", SummaryMD: "Higher accuracy than baseline."}},
		}},
		nil,
		nil,
		workspaceRoot,
	)

	result, err := svc.Run(context.Background(), model.WriterRunRequest{
		PaperTemplateRef:     templatePath,
		IdeaRefs:             []string{"idea_1"},
		ExperimentResultRefs: []string{"run_1"},
		ComparisonRefs:       []string{"exp_1"},
		CitationRefs:         []string{"paper:demo"},
		ExecutionMode:        "mock",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result == nil || result.Job == nil {
		t.Fatalf("expected writer run result")
	}
	if jobRecorder.lastCreate == nil {
		t.Fatalf("expected create request")
	}
	refTypes := map[string]bool{}
	for _, ref := range jobRecorder.lastCreate.InputRefs {
		refTypes[ref.RefType] = true
	}
	for _, required := range []string{"paper_template", "idea", "experiment_result", "comparison", "citation"} {
		if !refTypes[required] {
			t.Fatalf("expected ref type %s", required)
		}
	}
}

func TestWriterPostProcessWritesDraftAndArchiveFiles(t *testing.T) {
	workspaceRoot := t.TempDir()
	archiveUpdater := &writerTestArchiveUpdater{}
	events := &writerTestEventPublisher{}
	jobRecorder := &writerTestJobRecorder{}

	svc := NewService(
		nil,
		jobRecorder,
		nil,
		nil,
		&writerTestRunReader{items: map[string]model.ExperimentRun{
			"run_1": {
				ID:           "run_1",
				ExperimentID: "exp_1",
				RunStatus:    "succeeded",
				ResultJSON:   map[string]interface{}{"result_archive_id": "archive_1"},
			},
		}},
		nil,
		archiveUpdater,
		events,
		workspaceRoot,
	)

	job := &model.AgentJob{
		ID:        "job_writer_pp_1",
		AgentType: "writer",
		Status:    "succeeded",
		Metadata: map[string]any{
			"paper_template_ref":     filepath.Join(workspaceRoot, "templates", "paper.md"),
			"idea_refs":              []string{"idea_1"},
			"experiment_result_refs": []string{"run_1"},
			"comparison_refs":        []string{"exp_1"},
			"citation_refs":          []string{"paper:demo"},
		},
		InputRefs: []model.AgentInputRef{
			{RefType: "experiment_result", RefID: "run_1", Metadata: map[string]any{"result_archive_id": "archive_1"}},
		},
		NormalizedPayload: map[string]any{
			"title":        "Controlled Draft",
			"abstract":     "Abstract section.",
			"introduction": "Introduction section.",
			"method":       "Method section.",
			"experiments":  "Experiments section.",
			"conclusion":   "Conclusion section.",
			"references_stub": []any{
				"[Ref 1] Controlled Drafting, 2026.",
			},
			"figure_plan": []any{
				map[string]any{
					"figure_id":         "fig_1",
					"figure_type":       "workflow_mock",
					"title":             "Pipeline Overview",
					"description":       "Mock pipeline figure.",
					"placeholder_notes": []any{"Picture agent is mock in v1."},
				},
			},
		},
	}

	if err := svc.PostProcess(context.Background(), job); err != nil {
		t.Fatalf("PostProcess returned error: %v", err)
	}
	if jobRecorder.lastUpdate == nil {
		t.Fatalf("expected job update")
	}
	draftID := stringValue(jobRecorder.lastUpdate.NormalizedPayload["draft_id"])
	if draftID == "" {
		t.Fatalf("expected draft_id")
	}
	draftDir := filepath.Join(workspaceRoot, "writing", draftID)
	for _, rel := range []string{"draft.json", "draft.md", "figure_plan.json", "sources.json"} {
		if _, err := os.Stat(filepath.Join(draftDir, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
	draft, err := svc.GetDraft(context.Background(), draftID)
	if err != nil {
		t.Fatalf("GetDraft returned error: %v", err)
	}
	if draft == nil || draft.Title != "Controlled Draft" {
		t.Fatalf("expected persisted draft")
	}
	if archiveUpdater.lastID != "archive_1" || archiveUpdater.lastReq == nil || len(archiveUpdater.lastReq.Files) < 3 {
		t.Fatalf("expected archive update with draft files")
	}
	if len(events.published) != 1 || events.published[0].EventType != "draft_ready" {
		t.Fatalf("expected draft_ready event")
	}
}

func TestWriterGetDraftMissingReturnsNil(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, nil, nil, nil, nil, t.TempDir())

	draft, err := svc.GetDraft(context.Background(), "draft_missing")
	if err != nil {
		t.Fatalf("GetDraft returned error: %v", err)
	}
	if draft != nil {
		t.Fatalf("expected nil draft")
	}
}

func TestWriterDraftDocumentTimestampPersists(t *testing.T) {
	workspaceRoot := t.TempDir()
	jobRecorder := &writerTestJobRecorder{}
	svc := NewService(nil, jobRecorder, nil, nil, nil, nil, nil, nil, workspaceRoot)

	job := &model.AgentJob{
		ID:     "job_writer_pp_2",
		Status: "succeeded",
		Metadata: map[string]any{
			"paper_template_ref":     "workspace/templates/paper.md",
			"idea_refs":              []string{"idea_1"},
			"experiment_result_refs": []string{"run_1"},
		},
		NormalizedPayload: map[string]any{
			"title":           "Timestamp Draft",
			"abstract":        "A",
			"introduction":    "B",
			"method":          "C",
			"experiments":     "D",
			"conclusion":      "E",
			"references_stub": []any{"R"},
			"figure_plan":     []any{},
		},
	}
	before := time.Now()
	if err := svc.PostProcess(context.Background(), job); err != nil {
		t.Fatalf("PostProcess returned error: %v", err)
	}
	draftID := stringValue(jobRecorder.lastUpdate.NormalizedPayload["draft_id"])
	draft, err := svc.GetDraft(context.Background(), draftID)
	if err != nil {
		t.Fatalf("GetDraft returned error: %v", err)
	}
	if draft == nil || draft.GeneratedAt.Before(before) {
		t.Fatalf("expected generated_at to persist")
	}
}
