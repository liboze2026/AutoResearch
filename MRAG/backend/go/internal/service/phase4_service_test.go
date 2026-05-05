package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type memoryPhase4Store struct {
	datasetProfiles map[string]model.Phase4DatasetProfile
	readerSources   map[string]model.Phase4ReaderSource
	readerContexts  map[string]model.Phase4ReaderContext
	ideas           map[string]model.Phase4Idea
	runs            map[string]model.Phase4RunManifest
	reports         map[string]model.Phase4StructuredReportRecord
}

func newMemoryPhase4Store() *memoryPhase4Store {
	return &memoryPhase4Store{
		datasetProfiles: map[string]model.Phase4DatasetProfile{},
		readerSources:   map[string]model.Phase4ReaderSource{},
		readerContexts:  map[string]model.Phase4ReaderContext{},
		ideas:           map[string]model.Phase4Idea{},
		runs:            map[string]model.Phase4RunManifest{},
		reports:         map[string]model.Phase4StructuredReportRecord{},
	}
}

func (s *memoryPhase4Store) ListDatasetProfiles(_ context.Context, taskType string, status string) ([]model.Phase4DatasetProfile, error) {
	items := make([]model.Phase4DatasetProfile, 0)
	for _, item := range s.datasetProfiles {
		if taskType != "" && item.TaskType != taskType {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4Store) GetDatasetProfileByID(_ context.Context, id string) (*model.Phase4DatasetProfile, error) {
	item, ok := s.datasetProfiles[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4Store) CreateDatasetProfile(_ context.Context, item model.Phase4DatasetProfile) error {
	s.datasetProfiles[item.ID] = item
	return nil
}

func (s *memoryPhase4Store) UpdateDatasetProfile(_ context.Context, item model.Phase4DatasetProfile) error {
	s.datasetProfiles[item.ID] = item
	return nil
}

func (s *memoryPhase4Store) DeleteDatasetProfile(_ context.Context, id string) error {
	delete(s.datasetProfiles, id)
	return nil
}

func (s *memoryPhase4Store) ListReaderSources(_ context.Context, datasetProfileID string) ([]model.Phase4ReaderSource, error) {
	items := make([]model.Phase4ReaderSource, 0)
	for _, item := range s.readerSources {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4Store) GetReaderSourceByID(_ context.Context, id string) (*model.Phase4ReaderSource, error) {
	item, ok := s.readerSources[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4Store) CreateReaderSource(_ context.Context, item model.Phase4ReaderSource) error {
	s.readerSources[item.ID] = item
	return nil
}

func (s *memoryPhase4Store) UpdateReaderSource(_ context.Context, item model.Phase4ReaderSource) error {
	s.readerSources[item.ID] = item
	return nil
}

func (s *memoryPhase4Store) ListReaderContexts(_ context.Context, datasetProfileID string) ([]model.Phase4ReaderContext, error) {
	items := make([]model.Phase4ReaderContext, 0)
	for _, item := range s.readerContexts {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4Store) GetReaderContextByID(_ context.Context, id string) (*model.Phase4ReaderContext, error) {
	item, ok := s.readerContexts[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4Store) CreateReaderContext(_ context.Context, item model.Phase4ReaderContext) error {
	s.readerContexts[item.ID] = item
	return nil
}

func (s *memoryPhase4Store) UpdateReaderContext(_ context.Context, item model.Phase4ReaderContext) error {
	s.readerContexts[item.ID] = item
	return nil
}

func (s *memoryPhase4Store) ListIdeas(_ context.Context, datasetProfileID string, status string) ([]model.Phase4Idea, error) {
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

func (s *memoryPhase4Store) GetIdeaByID(_ context.Context, id string) (*model.Phase4Idea, error) {
	item, ok := s.ideas[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4Store) CreateIdea(_ context.Context, item model.Phase4Idea) error {
	s.ideas[item.ID] = item
	return nil
}

func (s *memoryPhase4Store) UpdateIdea(_ context.Context, item model.Phase4Idea) error {
	s.ideas[item.ID] = item
	return nil
}

func (s *memoryPhase4Store) DeleteIdea(_ context.Context, id string) error {
	delete(s.ideas, id)
	return nil
}

func (s *memoryPhase4Store) ListRunManifests(_ context.Context, datasetProfileID string, ideaID string, status string) ([]model.Phase4RunManifest, error) {
	items := make([]model.Phase4RunManifest, 0)
	for _, item := range s.runs {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		if ideaID != "" && item.IdeaID != ideaID {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4Store) GetRunManifestByID(_ context.Context, id string) (*model.Phase4RunManifest, error) {
	item, ok := s.runs[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4Store) CreateRunManifest(_ context.Context, item model.Phase4RunManifest) error {
	s.runs[item.ID] = item
	return nil
}

func (s *memoryPhase4Store) UpdateRunManifest(_ context.Context, item model.Phase4RunManifest) error {
	s.runs[item.ID] = item
	return nil
}

func (s *memoryPhase4Store) ListStructuredReports(_ context.Context, runManifestID string) ([]model.Phase4StructuredReportRecord, error) {
	items := make([]model.Phase4StructuredReportRecord, 0)
	for _, item := range s.reports {
		if runManifestID != "" && item.RunManifestID != runManifestID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4Store) GetStructuredReportByID(_ context.Context, id string) (*model.Phase4StructuredReportRecord, error) {
	item, ok := s.reports[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4Store) CreateStructuredReport(_ context.Context, item model.Phase4StructuredReportRecord) error {
	s.reports[item.ID] = item
	return nil
}

func (s *memoryPhase4Store) UpdateStructuredReport(_ context.Context, item model.Phase4StructuredReportRecord) error {
	s.reports[item.ID] = item
	return nil
}

func TestPhase4DatasetProfileValidation(t *testing.T) {
	svc := NewPhase4Service(newMemoryPhase4Store())

	if _, err := svc.CreateDatasetProfile(context.Background(), model.Phase4DatasetProfileCreateRequest{
		DatasetName: "VisDoM",
		TaskType:    "retrieval",
		SourceMode:  model.Phase4DatasetProfileSourceRegisteredPath,
		ServerPath:  "/home/bzli/mrag/datasets/visdom",
	}); err == nil {
		t.Fatalf("expected serverId validation error")
	}

	item, err := svc.CreateDatasetProfile(context.Background(), model.Phase4DatasetProfileCreateRequest{
		DatasetName:         "VisDoM Retrieval",
		TaskType:            "retrieval",
		ModalityComposition: []string{"image", "text", "text"},
		Splits:              []model.Phase4DatasetSplit{{Name: "train", SampleCount: 1200}, {Name: "test", SampleCount: 200}},
		OfficialMetric:      "Recall@10",
		KnownDifficulties:   []string{"hard negatives"},
		SourceMode:          model.Phase4DatasetProfileSourceRegisteredPath,
		ServerID:            "srv_1",
		ServerPath:          "/home/bzli/mrag/datasets/visdom",
	})
	if err != nil {
		t.Fatalf("create dataset profile failed: %v", err)
	}
	if item.ID == "" {
		t.Fatalf("expected dataset profile id")
	}
	if len(item.ModalityComposition) != 2 {
		t.Fatalf("expected deduped modality composition, got %d", len(item.ModalityComposition))
	}
}

func TestPhase4IdeaStatusFlowAndScoreValidation(t *testing.T) {
	store := newMemoryPhase4Store()
	svc := NewPhase4Service(store)
	profile, err := svc.CreateDatasetProfile(context.Background(), model.Phase4DatasetProfileCreateRequest{
		DatasetName: "VisDoM",
		TaskType:    "retrieval",
		SourceMode:  model.Phase4DatasetProfileSourceRegisteredPath,
		ServerID:    "srv_1",
		ServerPath:  "/home/bzli/mrag/datasets/visdom",
	})
	if err != nil {
		t.Fatalf("create dataset profile failed: %v", err)
	}

	if _, err = svc.CreateIdea(context.Background(), model.Phase4IdeaCreateRequest{
		DatasetProfileID:  profile.ID,
		Title:             "bad score",
		ProblemDefinition: "p",
		CoreMethod:        "m",
		Score:             model.Phase4IdeaScore{Novelty: 11},
	}); err == nil {
		t.Fatalf("expected score validation error")
	}

	idea, err := svc.CreateIdea(context.Background(), model.Phase4IdeaCreateRequest{
		DatasetProfileID:  profile.ID,
		Title:             "Page retrieval with layout-aware negatives",
		ProblemDefinition: "Improve page-level retrieval recall on VisDoM",
		CoreMethod:        "layout-aware negatives",
		Score: model.Phase4IdeaScore{
			Novelty:         7,
			DatasetFit:      9,
			Feasibility:     8,
			ExpectedGain:    8,
			ComputeCost:     4,
			FailureRisk:     3,
			Reproducibility: 8,
		},
	})
	if err != nil {
		t.Fatalf("create idea failed: %v", err)
	}

	idea, err = svc.UpdateIdeaStatus(context.Background(), idea.ID, model.Phase4IdeaStatusUpdateRequest{Status: model.Phase4IdeaStatusScored})
	if err != nil {
		t.Fatalf("expected draft -> scored to succeed: %v", err)
	}
	idea, err = svc.SelectIdea(context.Background(), idea.ID)
	if err != nil {
		t.Fatalf("expected scored -> selected to succeed: %v", err)
	}
	if idea.Status != model.Phase4IdeaStatusSelected {
		t.Fatalf("expected selected status, got %s", idea.Status)
	}
	if _, err = svc.UpdateIdeaStatus(context.Background(), idea.ID, model.Phase4IdeaStatusUpdateRequest{Status: model.Phase4IdeaStatusDraft}); err == nil {
		t.Fatalf("expected invalid selected -> draft transition")
	}
}

func TestPhase4RunManifestStatusAndRetry(t *testing.T) {
	svc := NewPhase4Service(newMemoryPhase4Store())
	profile, err := svc.CreateDatasetProfile(context.Background(), model.Phase4DatasetProfileCreateRequest{
		DatasetName: "VisDoM",
		TaskType:    "retrieval",
		SourceMode:  model.Phase4DatasetProfileSourceRegisteredPath,
		ServerID:    "srv_1",
		ServerPath:  "/home/bzli/mrag/datasets/visdom",
	})
	if err != nil {
		t.Fatalf("create dataset profile failed: %v", err)
	}
	idea, err := svc.CreateIdea(context.Background(), model.Phase4IdeaCreateRequest{
		DatasetProfileID:  profile.ID,
		Title:             "Idea",
		ProblemDefinition: "p",
		CoreMethod:        "m",
	})
	if err != nil {
		t.Fatalf("create idea failed: %v", err)
	}

	run, err := svc.CreateRunManifest(context.Background(), model.Phase4RunManifestCreateRequest{
		DatasetProfileID: profile.ID,
		IdeaID:           idea.ID,
		RunnerMode:       "remote",
		MaxRetryCount:    3,
	})
	if err != nil {
		t.Fatalf("create run manifest failed: %v", err)
	}
	run, err = svc.UpdateRunManifestStatus(context.Background(), run.ID, model.Phase4RunManifestStatusUpdateRequest{Status: model.Phase4RunStatusQueued})
	if err != nil {
		t.Fatalf("expected draft -> queued to succeed: %v", err)
	}
	startedAt := time.Now()
	run, err = svc.UpdateRunManifest(context.Background(), run.ID, model.Phase4RunManifestUpdateRequest{
		Status:     ptrString(model.Phase4RunStatusRunning),
		RetryCount: ptrInt(1),
		StartedAt:  &startedAt,
	})
	if err != nil {
		t.Fatalf("expected queued -> running to succeed: %v", err)
	}
	if run.RetryCount != 1 {
		t.Fatalf("expected retryCount=1, got %d", run.RetryCount)
	}
	if _, err = svc.UpdateRunManifest(context.Background(), run.ID, model.Phase4RunManifestUpdateRequest{RetryCount: ptrInt(4)}); err == nil {
		t.Fatalf("expected retry count overflow validation")
	}
}

func TestPhase4StructuredReportSerialization(t *testing.T) {
	svc := NewPhase4Service(newMemoryPhase4Store())
	profile, _ := svc.CreateDatasetProfile(context.Background(), model.Phase4DatasetProfileCreateRequest{
		DatasetName: "VisDoM",
		TaskType:    "retrieval",
		SourceMode:  model.Phase4DatasetProfileSourceRegisteredPath,
		ServerID:    "srv_1",
		ServerPath:  "/home/bzli/mrag/datasets/visdom",
	})
	idea, _ := svc.CreateIdea(context.Background(), model.Phase4IdeaCreateRequest{
		DatasetProfileID:  profile.ID,
		Title:             "Idea",
		ProblemDefinition: "p",
		CoreMethod:        "m",
	})
	run, _ := svc.CreateRunManifest(context.Background(), model.Phase4RunManifestCreateRequest{
		DatasetProfileID: profile.ID,
		IdeaID:           idea.ID,
		RunnerMode:       "remote",
	})
	report, err := svc.CreateStructuredReport(context.Background(), model.Phase4StructuredReportCreateRequest{
		RunManifestID:         run.ID,
		Title:                 "VisDoM page retrieval report",
		MachineReadableReport: map[string]any{"metric": "Recall@10", "value": 0.81},
		HumanReadableReportMD: "# Report\n\nGood run.",
		CitationRefs:          []string{"paper:demo"},
	})
	if err != nil {
		t.Fatalf("create report failed: %v", err)
	}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report failed: %v", err)
	}
	var decoded model.Phase4StructuredReportRecord
	if err = json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal report failed: %v", err)
	}
	if decoded.HumanReadableReportMD == "" || decoded.MachineReadableReport["metric"] != "Recall@10" {
		t.Fatalf("unexpected serialized report payload: %#v", decoded)
	}
}

func ptrInt(value int) *int { return &value }

func ptrString(value string) *string { return &value }
