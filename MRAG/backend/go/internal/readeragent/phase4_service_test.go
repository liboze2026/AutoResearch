package readeragent

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/agentartifact"
	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/model"
)

type memoryPhase4ReaderStore struct {
	datasetProfiles map[string]model.Phase4DatasetProfile
	readerSources   map[string]model.Phase4ReaderSource
	readerContexts  map[string]model.Phase4ReaderContext
	sourceSeq       int
	contextSeq      int
}

func newMemoryPhase4ReaderStore() *memoryPhase4ReaderStore {
	return &memoryPhase4ReaderStore{
		datasetProfiles: map[string]model.Phase4DatasetProfile{},
		readerSources:   map[string]model.Phase4ReaderSource{},
		readerContexts:  map[string]model.Phase4ReaderContext{},
	}
}

func (s *memoryPhase4ReaderStore) GetDatasetProfileByID(_ context.Context, id string) (*model.Phase4DatasetProfile, error) {
	item, ok := s.datasetProfiles[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4ReaderStore) ListReaderSources(_ context.Context, datasetProfileID string) ([]model.Phase4ReaderSource, error) {
	items := make([]model.Phase4ReaderSource, 0)
	for _, item := range s.readerSources {
		if datasetProfileID != "" && item.DatasetProfileID != datasetProfileID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryPhase4ReaderStore) GetReaderSourceByID(_ context.Context, id string) (*model.Phase4ReaderSource, error) {
	item, ok := s.readerSources[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4ReaderStore) CreateReaderSource(_ context.Context, req model.Phase4ReaderSourceCreateRequest) (*model.Phase4ReaderSource, error) {
	s.sourceSeq++
	now := time.Now()
	item := model.Phase4ReaderSource{
		ID:               fmt.Sprintf("p4src_test_%d", s.sourceSeq),
		DatasetProfileID: req.DatasetProfileID,
		Title:            req.Title,
		Authors:          append([]string{}, req.Authors...),
		Venue:            req.Venue,
		PublicationYear:  req.PublicationYear,
		SourceType:       req.SourceType,
		SourceURL:        req.SourceURL,
		OpenAccessURL:    req.OpenAccessURL,
		QualityTier:      req.QualityTier,
		RankingScore:     req.RankingScore,
		QualityScore:     req.QualityScore,
		RelevanceScore:   req.RelevanceScore,
		CitationCount:    req.CitationCount,
		Metadata:         phase4CloneMap(req.Metadata),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.readerSources[item.ID] = item
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4ReaderStore) UpdateReaderSource(_ context.Context, id string, req model.Phase4ReaderSourceUpdateRequest) (*model.Phase4ReaderSource, error) {
	item := s.readerSources[id]
	if req.Title != nil {
		item.Title = *req.Title
	}
	if req.Authors != nil {
		item.Authors = append([]string{}, (*req.Authors)...)
	}
	if req.Venue != nil {
		item.Venue = *req.Venue
	}
	if req.PublicationYear != nil {
		item.PublicationYear = *req.PublicationYear
	}
	if req.SourceType != nil {
		item.SourceType = *req.SourceType
	}
	if req.SourceURL != nil {
		item.SourceURL = *req.SourceURL
	}
	if req.OpenAccessURL != nil {
		item.OpenAccessURL = *req.OpenAccessURL
	}
	if req.QualityTier != nil {
		item.QualityTier = *req.QualityTier
	}
	if req.RankingScore != nil {
		item.RankingScore = *req.RankingScore
	}
	if req.QualityScore != nil {
		item.QualityScore = *req.QualityScore
	}
	if req.RelevanceScore != nil {
		item.RelevanceScore = *req.RelevanceScore
	}
	if req.CitationCount != nil {
		item.CitationCount = *req.CitationCount
	}
	if req.Metadata != nil {
		item.Metadata = phase4CloneMap(*req.Metadata)
	}
	item.UpdatedAt = time.Now()
	s.readerSources[id] = item
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4ReaderStore) CreateReaderContext(_ context.Context, req model.Phase4ReaderContextCreateRequest) (*model.Phase4ReaderContext, error) {
	s.contextSeq++
	now := time.Now()
	item := model.Phase4ReaderContext{
		ID:                fmt.Sprintf("p4ctx_test_%d", s.contextSeq),
		DatasetProfileID:  req.DatasetProfileID,
		Title:             req.Title,
		Summary:           req.Summary,
		TaskDefinition:    req.TaskDefinition,
		RelatedWork:       append([]string{}, req.RelatedWork...),
		RetrievalFocus:    append([]string{}, req.RetrievalFocus...),
		RankingNotes:      req.RankingNotes,
		SourceIDs:         append([]string{}, req.SourceIDs...),
		StructuredContext: phase4CloneMap(req.StructuredContext),
		Status:            req.Status,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	s.readerContexts[item.ID] = item
	copyItem := item
	return &copyItem, nil
}

func (s *memoryPhase4ReaderStore) GetReaderContextByID(_ context.Context, id string) (*model.Phase4ReaderContext, error) {
	item, ok := s.readerContexts[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func phase4PythonAgentsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "python_agents"))
}

func TestPhase4ReaderServiceRunMockPersistsSourcesAndContext(t *testing.T) {
	workspaceRoot := t.TempDir()
	jobStore := newMemoryJobStore()
	triggerStore := newMemoryTriggerStore()
	artifactStore := newMemoryArtifactStore()
	phase4Store := newMemoryPhase4ReaderStore()
	phase4Store.datasetProfiles["p4ds_visdom"] = model.Phase4DatasetProfile{
		ID:                  "p4ds_visdom",
		DatasetName:         "VisDoM",
		TaskType:            "retrieval",
		ModalityComposition: []string{"image", "text"},
		OfficialMetric:      "Recall@10",
		ServerID:            "srv_visdom",
		ServerPath:          "/home/bzli/mrag/datasets/visdom",
	}

	pythonExec := requirePython(t)
	pythonDir := phase4PythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	artifactSvc := agentartifact.NewService(artifactStore)
	readerSvc := NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactSvc, phase4Store, workspaceRoot)
	triggerSvc.RegisterPostProcessor("reader_phase4", readerSvc)

	result, err := readerSvc.Run(context.Background(), model.Phase4ReaderRunRequest{
		DatasetProfileID: "p4ds_visdom",
		ExecutionMode:    "mock",
		MaxPapers:        4,
	})
	if err != nil {
		t.Fatalf("phase4 reader run failed: %v", err)
	}
	if result.Job == nil || result.Job.ID == "" {
		t.Fatalf("expected phase4 reader job")
	}
	if result.ReaderContext == nil {
		t.Fatalf("expected persisted reader context")
	}
	if len(result.ReaderSources) < 2 {
		t.Fatalf("expected persisted reader sources, got %d", len(result.ReaderSources))
	}
	if len(result.ReaderContext.SourceIDs) != len(result.ReaderSources) {
		t.Fatalf("expected source ids to align with reader sources")
	}
	firstSource := result.ReaderSources[0]
	if strings.TrimSpace(stringValue(firstSource.Metadata["abstract"])) == "" {
		t.Fatalf("expected persisted abstract metadata")
	}
	if result.Job.NormalizedPayload["reader_context_id"] == "" {
		t.Fatalf("expected job normalized payload to carry reader_context_id")
	}
}

func TestPhase4ReaderServiceGetJobReturnsArtifactsAndPersistedObjects(t *testing.T) {
	workspaceRoot := t.TempDir()
	jobStore := newMemoryJobStore()
	triggerStore := newMemoryTriggerStore()
	artifactStore := newMemoryArtifactStore()
	phase4Store := newMemoryPhase4ReaderStore()
	phase4Store.datasetProfiles["p4ds_visdom"] = model.Phase4DatasetProfile{
		ID:                  "p4ds_visdom",
		DatasetName:         "VisDoM",
		TaskType:            "retrieval",
		ModalityComposition: []string{"image", "text"},
		OfficialMetric:      "Recall@10",
		ServerID:            "srv_visdom",
		ServerPath:          "/home/bzli/mrag/datasets/visdom",
	}

	pythonExec := requirePython(t)
	pythonDir := phase4PythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	artifactSvc := agentartifact.NewService(artifactStore)
	readerSvc := NewPhase4Service(jobSvc, jobStore, triggerSvc, artifactSvc, phase4Store, workspaceRoot)
	triggerSvc.RegisterPostProcessor("reader_phase4", readerSvc)

	runResult, err := readerSvc.Run(context.Background(), model.Phase4ReaderRunRequest{
		DatasetProfileID: "p4ds_visdom",
		ExecutionMode:    "mock",
		MaxPapers:        3,
	})
	if err != nil {
		t.Fatalf("phase4 reader run failed: %v", err)
	}

	detail, err := readerSvc.GetJob(context.Background(), runResult.Job.ID)
	if err != nil {
		t.Fatalf("phase4 reader get job failed: %v", err)
	}
	if detail == nil || detail.Job == nil {
		t.Fatalf("expected reader job detail")
	}
	if len(detail.Artifacts) == 0 {
		t.Fatalf("expected persisted artifacts")
	}
	if detail.ReaderContext == nil {
		t.Fatalf("expected reader context in job detail")
	}
	if len(detail.ReaderSources) == 0 {
		t.Fatalf("expected reader sources in job detail")
	}
}
