package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type Phase4Store interface {
	ListDatasetProfiles(context.Context, string, string) ([]model.Phase4DatasetProfile, error)
	GetDatasetProfileByID(context.Context, string) (*model.Phase4DatasetProfile, error)
	CreateDatasetProfile(context.Context, model.Phase4DatasetProfile) error
	UpdateDatasetProfile(context.Context, model.Phase4DatasetProfile) error
	DeleteDatasetProfile(context.Context, string) error
	ListReaderSources(context.Context, string) ([]model.Phase4ReaderSource, error)
	GetReaderSourceByID(context.Context, string) (*model.Phase4ReaderSource, error)
	CreateReaderSource(context.Context, model.Phase4ReaderSource) error
	UpdateReaderSource(context.Context, model.Phase4ReaderSource) error
	ListReaderContexts(context.Context, string) ([]model.Phase4ReaderContext, error)
	GetReaderContextByID(context.Context, string) (*model.Phase4ReaderContext, error)
	CreateReaderContext(context.Context, model.Phase4ReaderContext) error
	UpdateReaderContext(context.Context, model.Phase4ReaderContext) error
	ListIdeas(context.Context, string, string) ([]model.Phase4Idea, error)
	GetIdeaByID(context.Context, string) (*model.Phase4Idea, error)
	CreateIdea(context.Context, model.Phase4Idea) error
	UpdateIdea(context.Context, model.Phase4Idea) error
	DeleteIdea(context.Context, string) error
	ListRunManifests(context.Context, string, string, string) ([]model.Phase4RunManifest, error)
	GetRunManifestByID(context.Context, string) (*model.Phase4RunManifest, error)
	CreateRunManifest(context.Context, model.Phase4RunManifest) error
	UpdateRunManifest(context.Context, model.Phase4RunManifest) error
	ListStructuredReports(context.Context, string) ([]model.Phase4StructuredReportRecord, error)
	GetStructuredReportByID(context.Context, string) (*model.Phase4StructuredReportRecord, error)
	CreateStructuredReport(context.Context, model.Phase4StructuredReportRecord) error
	UpdateStructuredReport(context.Context, model.Phase4StructuredReportRecord) error
}

type phase4WorkflowStore interface {
	ListWorkflows(context.Context, string, string) ([]model.Phase4Workflow, error)
	GetWorkflowByID(context.Context, string) (*model.Phase4Workflow, error)
	CreateWorkflow(context.Context, model.Phase4Workflow) error
	UpdateWorkflow(context.Context, model.Phase4Workflow) error
	ListWorkflowActions(context.Context, string) ([]model.Phase4WorkflowAction, error)
	CreateWorkflowAction(context.Context, model.Phase4WorkflowAction) error
}

type Phase4Service struct {
	store Phase4Store
}

func NewPhase4Service(store Phase4Store) *Phase4Service {
	return &Phase4Service{store: store}
}

func (s *Phase4Service) ListDatasetProfiles(ctx context.Context, taskType string, status string) ([]model.Phase4DatasetProfile, error) {
	return s.store.ListDatasetProfiles(ctx, strings.TrimSpace(strings.ToLower(taskType)), strings.TrimSpace(strings.ToLower(status)))
}

func (s *Phase4Service) GetDatasetProfileByID(ctx context.Context, id string) (*model.Phase4DatasetProfile, error) {
	return s.store.GetDatasetProfileByID(ctx, strings.TrimSpace(id))
}

func (s *Phase4Service) CreateDatasetProfile(ctx context.Context, req model.Phase4DatasetProfileCreateRequest) (*model.Phase4DatasetProfile, error) {
	if err := model.ValidatePhase4DatasetProfileCreateRequest(req); err != nil {
		return nil, err
	}
	now := time.Now()
	item := model.Phase4DatasetProfile{
		ID:                    httpx.NewID("p4ds"),
		DatasetName:           strings.TrimSpace(req.DatasetName),
		TaskType:              strings.TrimSpace(strings.ToLower(req.TaskType)),
		ModalityComposition:   normalizeStringList(req.ModalityComposition),
		Splits:                normalizeDatasetSplits(req.Splits),
		LabelSchema:           clonePhase4AnyMap(req.LabelSchema),
		FileStructureSnapshot: clonePhase4AnyMap(req.FileStructureSnapshot),
		SampleStatistics:      clonePhase4AnyMap(req.SampleStatistics),
		OfficialMetric:        strings.TrimSpace(req.OfficialMetric),
		OfficialBaseline:      strings.TrimSpace(req.OfficialBaseline),
		License:               strings.TrimSpace(req.License),
		Citation:              strings.TrimSpace(req.Citation),
		KnownDifficulties:     normalizeStringList(req.KnownDifficulties),
		UserNotes:             strings.TrimSpace(req.UserNotes),
		Metadata:              clonePhase4AnyMap(req.Metadata),
		SourceMode:            model.NormalizePhase4DatasetProfileSourceMode(req.SourceMode),
		ServerID:              strings.TrimSpace(req.ServerID),
		ServerPath:            strings.TrimSpace(req.ServerPath),
		Status:                model.NormalizePhase4DatasetProfileStatus(req.Status),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := s.store.CreateDatasetProfile(ctx, item); err != nil {
		return nil, err
	}
	return s.store.GetDatasetProfileByID(ctx, item.ID)
}

func (s *Phase4Service) UpdateDatasetProfile(ctx context.Context, id string, req model.Phase4DatasetProfileUpdateRequest) (*model.Phase4DatasetProfile, error) {
	item, err := s.store.GetDatasetProfileByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("phase4 dataset profile not found")
	}
	if req.DatasetName != nil {
		item.DatasetName = strings.TrimSpace(*req.DatasetName)
	}
	if req.TaskType != nil {
		item.TaskType = strings.TrimSpace(strings.ToLower(*req.TaskType))
	}
	if req.ModalityComposition != nil {
		item.ModalityComposition = normalizeStringList(*req.ModalityComposition)
	}
	if req.Splits != nil {
		item.Splits = normalizeDatasetSplits(*req.Splits)
	}
	if req.LabelSchema != nil {
		item.LabelSchema = clonePhase4AnyMap(*req.LabelSchema)
	}
	if req.FileStructureSnapshot != nil {
		item.FileStructureSnapshot = clonePhase4AnyMap(*req.FileStructureSnapshot)
	}
	if req.SampleStatistics != nil {
		item.SampleStatistics = clonePhase4AnyMap(*req.SampleStatistics)
	}
	if req.OfficialMetric != nil {
		item.OfficialMetric = strings.TrimSpace(*req.OfficialMetric)
	}
	if req.OfficialBaseline != nil {
		item.OfficialBaseline = strings.TrimSpace(*req.OfficialBaseline)
	}
	if req.License != nil {
		item.License = strings.TrimSpace(*req.License)
	}
	if req.Citation != nil {
		item.Citation = strings.TrimSpace(*req.Citation)
	}
	if req.KnownDifficulties != nil {
		item.KnownDifficulties = normalizeStringList(*req.KnownDifficulties)
	}
	if req.UserNotes != nil {
		item.UserNotes = strings.TrimSpace(*req.UserNotes)
	}
	if req.Metadata != nil {
		item.Metadata = clonePhase4AnyMap(*req.Metadata)
	}
	if req.SourceMode != nil {
		item.SourceMode = model.NormalizePhase4DatasetProfileSourceMode(*req.SourceMode)
	}
	if req.ServerID != nil {
		item.ServerID = strings.TrimSpace(*req.ServerID)
	}
	if req.ServerPath != nil {
		item.ServerPath = strings.TrimSpace(*req.ServerPath)
	}
	if req.Status != nil {
		item.Status = model.NormalizePhase4DatasetProfileStatus(*req.Status)
	}
	validateReq := model.Phase4DatasetProfileCreateRequest{
		DatasetName:           item.DatasetName,
		TaskType:              item.TaskType,
		ModalityComposition:   item.ModalityComposition,
		Splits:                item.Splits,
		LabelSchema:           item.LabelSchema,
		FileStructureSnapshot: item.FileStructureSnapshot,
		SampleStatistics:      item.SampleStatistics,
		OfficialMetric:        item.OfficialMetric,
		OfficialBaseline:      item.OfficialBaseline,
		License:               item.License,
		Citation:              item.Citation,
		KnownDifficulties:     item.KnownDifficulties,
		UserNotes:             item.UserNotes,
		Metadata:              item.Metadata,
		SourceMode:            item.SourceMode,
		ServerID:              item.ServerID,
		ServerPath:            item.ServerPath,
		Status:                item.Status,
	}
	if err = model.ValidatePhase4DatasetProfileCreateRequest(validateReq); err != nil {
		return nil, err
	}
	item.UpdatedAt = time.Now()
	if err = s.store.UpdateDatasetProfile(ctx, *item); err != nil {
		return nil, err
	}
	return s.store.GetDatasetProfileByID(ctx, item.ID)
}

func (s *Phase4Service) DeleteDatasetProfile(ctx context.Context, id string) error {
	existing, err := s.store.GetDatasetProfileByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("phase4 dataset profile not found")
	}
	return s.store.DeleteDatasetProfile(ctx, id)
}

func (s *Phase4Service) ListReaderSources(ctx context.Context, datasetProfileID string) ([]model.Phase4ReaderSource, error) {
	return s.store.ListReaderSources(ctx, strings.TrimSpace(datasetProfileID))
}

func (s *Phase4Service) GetReaderSourceByID(ctx context.Context, id string) (*model.Phase4ReaderSource, error) {
	return s.store.GetReaderSourceByID(ctx, strings.TrimSpace(id))
}

func (s *Phase4Service) CreateReaderSource(ctx context.Context, req model.Phase4ReaderSourceCreateRequest) (*model.Phase4ReaderSource, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(req.SourceType) == "" {
		return nil, fmt.Errorf("sourceType is required")
	}
	if req.DatasetProfileID != "" {
		if _, err := s.requireDatasetProfile(ctx, req.DatasetProfileID); err != nil {
			return nil, err
		}
	}
	now := time.Now()
	item := model.Phase4ReaderSource{
		ID:               httpx.NewID("p4src"),
		DatasetProfileID: strings.TrimSpace(req.DatasetProfileID),
		Title:            strings.TrimSpace(req.Title),
		Authors:          normalizeStringList(req.Authors),
		Venue:            strings.TrimSpace(req.Venue),
		PublicationYear:  req.PublicationYear,
		SourceType:       strings.TrimSpace(strings.ToLower(req.SourceType)),
		SourceURL:        strings.TrimSpace(req.SourceURL),
		OpenAccessURL:    strings.TrimSpace(req.OpenAccessURL),
		QualityTier:      strings.TrimSpace(req.QualityTier),
		RankingScore:     req.RankingScore,
		QualityScore:     req.QualityScore,
		RelevanceScore:   req.RelevanceScore,
		CitationCount:    req.CitationCount,
		Metadata:         clonePhase4AnyMap(req.Metadata),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.store.CreateReaderSource(ctx, item); err != nil {
		return nil, err
	}
	return s.store.GetReaderSourceByID(ctx, item.ID)
}

func (s *Phase4Service) UpdateReaderSource(ctx context.Context, id string, req model.Phase4ReaderSourceUpdateRequest) (*model.Phase4ReaderSource, error) {
	item, err := s.store.GetReaderSourceByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("phase4 reader source not found")
	}
	if req.Title != nil {
		item.Title = strings.TrimSpace(*req.Title)
	}
	if req.Authors != nil {
		item.Authors = normalizeStringList(*req.Authors)
	}
	if req.Venue != nil {
		item.Venue = strings.TrimSpace(*req.Venue)
	}
	if req.PublicationYear != nil {
		item.PublicationYear = *req.PublicationYear
	}
	if req.SourceType != nil {
		item.SourceType = strings.TrimSpace(strings.ToLower(*req.SourceType))
	}
	if req.SourceURL != nil {
		item.SourceURL = strings.TrimSpace(*req.SourceURL)
	}
	if req.OpenAccessURL != nil {
		item.OpenAccessURL = strings.TrimSpace(*req.OpenAccessURL)
	}
	if req.QualityTier != nil {
		item.QualityTier = strings.TrimSpace(*req.QualityTier)
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
		item.Metadata = clonePhase4AnyMap(*req.Metadata)
	}
	if strings.TrimSpace(item.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(item.SourceType) == "" {
		return nil, fmt.Errorf("sourceType is required")
	}
	item.UpdatedAt = time.Now()
	if err = s.store.UpdateReaderSource(ctx, *item); err != nil {
		return nil, err
	}
	return s.store.GetReaderSourceByID(ctx, item.ID)
}

func (s *Phase4Service) ListReaderContexts(ctx context.Context, datasetProfileID string) ([]model.Phase4ReaderContext, error) {
	return s.store.ListReaderContexts(ctx, strings.TrimSpace(datasetProfileID))
}

func (s *Phase4Service) GetReaderContextByID(ctx context.Context, id string) (*model.Phase4ReaderContext, error) {
	return s.store.GetReaderContextByID(ctx, strings.TrimSpace(id))
}

func (s *Phase4Service) CreateReaderContext(ctx context.Context, req model.Phase4ReaderContextCreateRequest) (*model.Phase4ReaderContext, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if req.DatasetProfileID != "" {
		if _, err := s.requireDatasetProfile(ctx, req.DatasetProfileID); err != nil {
			return nil, err
		}
	}
	if err := s.ensureReaderSourceIDsExist(ctx, req.SourceIDs); err != nil {
		return nil, err
	}
	now := time.Now()
	item := model.Phase4ReaderContext{
		ID:                httpx.NewID("p4ctx"),
		DatasetProfileID:  strings.TrimSpace(req.DatasetProfileID),
		Title:             strings.TrimSpace(req.Title),
		Summary:           strings.TrimSpace(req.Summary),
		TaskDefinition:    strings.TrimSpace(req.TaskDefinition),
		RelatedWork:       normalizeStringList(req.RelatedWork),
		RetrievalFocus:    normalizeStringList(req.RetrievalFocus),
		RankingNotes:      strings.TrimSpace(req.RankingNotes),
		SourceIDs:         normalizeStringList(req.SourceIDs),
		StructuredContext: clonePhase4AnyMap(req.StructuredContext),
		Status:            normalizeReaderContextStatus(req.Status),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.store.CreateReaderContext(ctx, item); err != nil {
		return nil, err
	}
	return s.store.GetReaderContextByID(ctx, item.ID)
}

func (s *Phase4Service) UpdateReaderContext(ctx context.Context, id string, req model.Phase4ReaderContextUpdateRequest) (*model.Phase4ReaderContext, error) {
	item, err := s.store.GetReaderContextByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("phase4 reader context not found")
	}
	if req.Title != nil {
		item.Title = strings.TrimSpace(*req.Title)
	}
	if req.Summary != nil {
		item.Summary = strings.TrimSpace(*req.Summary)
	}
	if req.TaskDefinition != nil {
		item.TaskDefinition = strings.TrimSpace(*req.TaskDefinition)
	}
	if req.RelatedWork != nil {
		item.RelatedWork = normalizeStringList(*req.RelatedWork)
	}
	if req.RetrievalFocus != nil {
		item.RetrievalFocus = normalizeStringList(*req.RetrievalFocus)
	}
	if req.RankingNotes != nil {
		item.RankingNotes = strings.TrimSpace(*req.RankingNotes)
	}
	if req.SourceIDs != nil {
		item.SourceIDs = normalizeStringList(*req.SourceIDs)
	}
	if req.StructuredContext != nil {
		item.StructuredContext = clonePhase4AnyMap(*req.StructuredContext)
	}
	if req.Status != nil {
		item.Status = normalizeReaderContextStatus(*req.Status)
	}
	if strings.TrimSpace(item.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if err = s.ensureReaderSourceIDsExist(ctx, item.SourceIDs); err != nil {
		return nil, err
	}
	item.UpdatedAt = time.Now()
	if err = s.store.UpdateReaderContext(ctx, *item); err != nil {
		return nil, err
	}
	return s.store.GetReaderContextByID(ctx, item.ID)
}

func (s *Phase4Service) ListIdeas(ctx context.Context, datasetProfileID string, status string) ([]model.Phase4Idea, error) {
	return s.store.ListIdeas(ctx, strings.TrimSpace(datasetProfileID), strings.TrimSpace(strings.ToLower(status)))
}

func (s *Phase4Service) GetIdeaByID(ctx context.Context, id string) (*model.Phase4Idea, error) {
	return s.store.GetIdeaByID(ctx, strings.TrimSpace(id))
}

func (s *Phase4Service) CreateIdea(ctx context.Context, req model.Phase4IdeaCreateRequest) (*model.Phase4Idea, error) {
	if err := s.validatePhase4IdeaCreate(ctx, req); err != nil {
		return nil, err
	}
	now := time.Now()
	status := model.NormalizePhase4IdeaStatus(req.Status)
	lineageRootID := ""
	if strings.TrimSpace(req.RevisionOfID) != "" {
		parent, err := s.requireIdea(ctx, req.RevisionOfID)
		if err != nil {
			return nil, err
		}
		lineageRootID = parent.LineageRootID
		if strings.TrimSpace(lineageRootID) == "" {
			lineageRootID = parent.ID
		}
	}
	item := model.Phase4Idea{
		ID:                  httpx.NewID("p4idea"),
		DatasetProfileID:    strings.TrimSpace(req.DatasetProfileID),
		ReaderContextID:     strings.TrimSpace(req.ReaderContextID),
		Title:               strings.TrimSpace(req.Title),
		ProblemDefinition:   strings.TrimSpace(req.ProblemDefinition),
		CoreMethod:          strings.TrimSpace(req.CoreMethod),
		Differentiators:     strings.TrimSpace(req.Differentiators),
		DataProcessingNeeds: normalizeStringList(req.DataProcessingNeeds),
		ModelChanges:        normalizeStringList(req.ModelChanges),
		TrainingPlan:        strings.TrimSpace(req.TrainingPlan),
		EvaluationMetrics:   normalizeStringList(req.EvaluationMetrics),
		RiskPoints:          normalizeStringList(req.RiskPoints),
		ExpectedGains:       normalizeStringList(req.ExpectedGains),
		Score:               req.Score,
		ScoreSummary:        clonePhase4AnyMap(req.ScoreSummary),
		Status:              status,
		SourceType:          normalizeSourceType(req.SourceType),
		RevisionOfID:        strings.TrimSpace(req.RevisionOfID),
		LineageRootID:       strings.TrimSpace(lineageRootID),
		FailureFeedback:     clonePhase4AnyMap(req.FailureFeedback),
		LastFailureRunID:    strings.TrimSpace(req.LastFailureRunID),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := s.store.CreateIdea(ctx, item); err != nil {
		return nil, err
	}
	return s.store.GetIdeaByID(ctx, item.ID)
}

func (s *Phase4Service) UpdateIdea(ctx context.Context, id string, req model.Phase4IdeaUpdateRequest) (*model.Phase4Idea, error) {
	item, err := s.requireIdea(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Title != nil {
		item.Title = strings.TrimSpace(*req.Title)
	}
	if req.ProblemDefinition != nil {
		item.ProblemDefinition = strings.TrimSpace(*req.ProblemDefinition)
	}
	if req.CoreMethod != nil {
		item.CoreMethod = strings.TrimSpace(*req.CoreMethod)
	}
	if req.Differentiators != nil {
		item.Differentiators = strings.TrimSpace(*req.Differentiators)
	}
	if req.DataProcessingNeeds != nil {
		item.DataProcessingNeeds = normalizeStringList(*req.DataProcessingNeeds)
	}
	if req.ModelChanges != nil {
		item.ModelChanges = normalizeStringList(*req.ModelChanges)
	}
	if req.TrainingPlan != nil {
		item.TrainingPlan = strings.TrimSpace(*req.TrainingPlan)
	}
	if req.EvaluationMetrics != nil {
		item.EvaluationMetrics = normalizeStringList(*req.EvaluationMetrics)
	}
	if req.RiskPoints != nil {
		item.RiskPoints = normalizeStringList(*req.RiskPoints)
	}
	if req.ExpectedGains != nil {
		item.ExpectedGains = normalizeStringList(*req.ExpectedGains)
	}
	if req.Score != nil {
		item.Score = *req.Score
	}
	if req.ScoreSummary != nil {
		item.ScoreSummary = clonePhase4AnyMap(*req.ScoreSummary)
	}
	if req.Status != nil {
		nextStatus := model.NormalizePhase4IdeaStatus(*req.Status)
		if err = model.ValidatePhase4IdeaStatusTransition(item.Status, nextStatus); err != nil {
			return nil, err
		}
		item.Status = nextStatus
	}
	if req.SourceType != nil {
		item.SourceType = normalizeSourceType(*req.SourceType)
	}
	if req.FailureFeedback != nil {
		item.FailureFeedback = clonePhase4AnyMap(*req.FailureFeedback)
	}
	if req.LastFailureRunID != nil {
		item.LastFailureRunID = strings.TrimSpace(*req.LastFailureRunID)
	}
	if err = s.validateStoredIdea(ctx, *item); err != nil {
		return nil, err
	}
	item.UpdatedAt = time.Now()
	if err = s.store.UpdateIdea(ctx, *item); err != nil {
		return nil, err
	}
	return s.store.GetIdeaByID(ctx, item.ID)
}

func (s *Phase4Service) UpdateIdeaStatus(ctx context.Context, id string, req model.Phase4IdeaStatusUpdateRequest) (*model.Phase4Idea, error) {
	item, err := s.requireIdea(ctx, id)
	if err != nil {
		return nil, err
	}
	nextStatus := model.NormalizePhase4IdeaStatus(req.Status)
	if err = model.ValidatePhase4IdeaStatusTransition(item.Status, nextStatus); err != nil {
		return nil, err
	}
	item.Status = nextStatus
	if len(req.FailureFeedback) > 0 {
		item.FailureFeedback = clonePhase4AnyMap(req.FailureFeedback)
	}
	if strings.TrimSpace(req.LastFailureRunID) != "" {
		item.LastFailureRunID = strings.TrimSpace(req.LastFailureRunID)
	}
	item.UpdatedAt = time.Now()
	if err = s.store.UpdateIdea(ctx, *item); err != nil {
		return nil, err
	}
	return s.store.GetIdeaByID(ctx, item.ID)
}

func (s *Phase4Service) SelectIdea(ctx context.Context, id string) (*model.Phase4Idea, error) {
	return s.UpdateIdeaStatus(ctx, id, model.Phase4IdeaStatusUpdateRequest{Status: model.Phase4IdeaStatusSelected})
}

func (s *Phase4Service) ArchiveIdea(ctx context.Context, id string) (*model.Phase4Idea, error) {
	return s.UpdateIdeaStatus(ctx, id, model.Phase4IdeaStatusUpdateRequest{Status: model.Phase4IdeaStatusArchived})
}

func (s *Phase4Service) RejectIdea(ctx context.Context, id string) (*model.Phase4Idea, error) {
	return s.UpdateIdeaStatus(ctx, id, model.Phase4IdeaStatusUpdateRequest{Status: model.Phase4IdeaStatusRejected})
}

func (s *Phase4Service) GetIdeaScoreView(ctx context.Context, id string) (*model.Phase4IdeaScoreView, error) {
	item, err := s.requireIdea(ctx, id)
	if err != nil {
		return nil, err
	}
	view := buildPhase4IdeaScoreView(*item)
	return &view, nil
}

func (s *Phase4Service) ListIdeaScoreViews(ctx context.Context, datasetProfileID string, status string) ([]model.Phase4IdeaScoreView, error) {
	items, err := s.ListIdeas(ctx, datasetProfileID, status)
	if err != nil {
		return nil, err
	}
	out := make([]model.Phase4IdeaScoreView, 0, len(items))
	for _, item := range items {
		out = append(out, buildPhase4IdeaScoreView(item))
	}
	return out, nil
}

func (s *Phase4Service) DeleteIdea(ctx context.Context, id string) error {
	if _, err := s.requireIdea(ctx, id); err != nil {
		return err
	}
	return s.store.DeleteIdea(ctx, strings.TrimSpace(id))
}

func (s *Phase4Service) ListRunManifests(ctx context.Context, datasetProfileID string, ideaID string, status string) ([]model.Phase4RunManifest, error) {
	return s.store.ListRunManifests(ctx, strings.TrimSpace(datasetProfileID), strings.TrimSpace(ideaID), strings.TrimSpace(strings.ToLower(status)))
}

func (s *Phase4Service) GetRunManifestByID(ctx context.Context, id string) (*model.Phase4RunManifest, error) {
	return s.store.GetRunManifestByID(ctx, strings.TrimSpace(id))
}

func (s *Phase4Service) CreateRunManifest(ctx context.Context, req model.Phase4RunManifestCreateRequest) (*model.Phase4RunManifest, error) {
	if _, err := s.requireDatasetProfile(ctx, req.DatasetProfileID); err != nil {
		return nil, err
	}
	if _, err := s.requireIdea(ctx, req.IdeaID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.ReaderContextID) != "" {
		if _, err := s.requireReaderContext(ctx, req.ReaderContextID); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(req.RunnerMode) == "" {
		return nil, fmt.Errorf("runnerMode is required")
	}
	status := model.NormalizePhase4RunStatus(req.Status)
	maxRetryCount := req.MaxRetryCount
	if maxRetryCount <= 0 {
		maxRetryCount = 3
	}
	if err := model.ValidatePhase4RetryCounts(req.RetryCount, maxRetryCount); err != nil {
		return nil, err
	}
	now := time.Now()
	item := model.Phase4RunManifest{
		ID:               httpx.NewID("p4run"),
		DatasetProfileID: strings.TrimSpace(req.DatasetProfileID),
		IdeaID:           strings.TrimSpace(req.IdeaID),
		ReaderContextID:  strings.TrimSpace(req.ReaderContextID),
		CodeSnapshotID:   strings.TrimSpace(req.CodeSnapshotID),
		RunnerMode:       strings.TrimSpace(strings.ToLower(req.RunnerMode)),
		ServerID:         strings.TrimSpace(req.ServerID),
		GPU:              strings.TrimSpace(req.GPU),
		Status:           status,
		RetryCount:       req.RetryCount,
		MaxRetryCount:    maxRetryCount,
		ArtifactPaths:    clonePhase4AnyMap(req.ArtifactPaths),
		LogsPath:         strings.TrimSpace(req.LogsPath),
		MetricsPath:      strings.TrimSpace(req.MetricsPath),
		FailureFeedback:  clonePhase4AnyMap(req.FailureFeedback),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.store.CreateRunManifest(ctx, item); err != nil {
		return nil, err
	}
	return s.store.GetRunManifestByID(ctx, item.ID)
}

func (s *Phase4Service) UpdateRunManifest(ctx context.Context, id string, req model.Phase4RunManifestUpdateRequest) (*model.Phase4RunManifest, error) {
	item, err := s.requireRunManifest(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.CodeSnapshotID != nil {
		item.CodeSnapshotID = strings.TrimSpace(*req.CodeSnapshotID)
	}
	if req.RunnerMode != nil {
		item.RunnerMode = strings.TrimSpace(strings.ToLower(*req.RunnerMode))
	}
	if req.ServerID != nil {
		item.ServerID = strings.TrimSpace(*req.ServerID)
	}
	if req.GPU != nil {
		item.GPU = strings.TrimSpace(*req.GPU)
	}
	if req.Status != nil {
		nextStatus := model.NormalizePhase4RunStatus(*req.Status)
		if err = model.ValidatePhase4RunStatusTransition(item.Status, nextStatus); err != nil {
			return nil, err
		}
		item.Status = nextStatus
	}
	if req.RetryCount != nil {
		item.RetryCount = *req.RetryCount
	}
	if req.MaxRetryCount != nil {
		item.MaxRetryCount = *req.MaxRetryCount
	}
	if req.ArtifactPaths != nil {
		item.ArtifactPaths = clonePhase4AnyMap(*req.ArtifactPaths)
	}
	if req.LogsPath != nil {
		item.LogsPath = strings.TrimSpace(*req.LogsPath)
	}
	if req.MetricsPath != nil {
		item.MetricsPath = strings.TrimSpace(*req.MetricsPath)
	}
	if req.FailureFeedback != nil {
		item.FailureFeedback = clonePhase4AnyMap(*req.FailureFeedback)
	}
	if req.StartedAt != nil {
		item.StartedAt = req.StartedAt
	}
	if req.FinishedAt != nil {
		item.FinishedAt = req.FinishedAt
	}
	if strings.TrimSpace(item.RunnerMode) == "" {
		return nil, fmt.Errorf("runnerMode is required")
	}
	if err = model.ValidatePhase4RetryCounts(item.RetryCount, item.MaxRetryCount); err != nil {
		return nil, err
	}
	item.UpdatedAt = time.Now()
	if err = s.store.UpdateRunManifest(ctx, *item); err != nil {
		return nil, err
	}
	return s.store.GetRunManifestByID(ctx, item.ID)
}

func (s *Phase4Service) UpdateRunManifestStatus(ctx context.Context, id string, req model.Phase4RunManifestStatusUpdateRequest) (*model.Phase4RunManifest, error) {
	item, err := s.requireRunManifest(ctx, id)
	if err != nil {
		return nil, err
	}
	nextStatus := model.NormalizePhase4RunStatus(req.Status)
	if err = model.ValidatePhase4RunStatusTransition(item.Status, nextStatus); err != nil {
		return nil, err
	}
	item.Status = nextStatus
	if req.RetryCount != nil {
		item.RetryCount = *req.RetryCount
	}
	if len(req.FailureFeedback) > 0 {
		item.FailureFeedback = clonePhase4AnyMap(req.FailureFeedback)
	}
	if req.StartedAt != nil {
		item.StartedAt = req.StartedAt
	}
	if req.FinishedAt != nil {
		item.FinishedAt = req.FinishedAt
	}
	if err = model.ValidatePhase4RetryCounts(item.RetryCount, item.MaxRetryCount); err != nil {
		return nil, err
	}
	item.UpdatedAt = time.Now()
	if err = s.store.UpdateRunManifest(ctx, *item); err != nil {
		return nil, err
	}
	return s.store.GetRunManifestByID(ctx, item.ID)
}

func (s *Phase4Service) ListStructuredReports(ctx context.Context, runManifestID string) ([]model.Phase4StructuredReportRecord, error) {
	return s.store.ListStructuredReports(ctx, strings.TrimSpace(runManifestID))
}

func (s *Phase4Service) GetStructuredReportByID(ctx context.Context, id string) (*model.Phase4StructuredReportRecord, error) {
	return s.store.GetStructuredReportByID(ctx, strings.TrimSpace(id))
}

func (s *Phase4Service) CreateStructuredReport(ctx context.Context, req model.Phase4StructuredReportCreateRequest) (*model.Phase4StructuredReportRecord, error) {
	if strings.TrimSpace(req.RunManifestID) == "" {
		return nil, fmt.Errorf("runManifestId is required")
	}
	run, err := s.requireRunManifest(ctx, req.RunManifestID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	now := time.Now()
	item := model.Phase4StructuredReportRecord{
		ID:                    httpx.NewID("p4report"),
		RunManifestID:         run.ID,
		DatasetProfileID:      firstNonEmpty(strings.TrimSpace(req.DatasetProfileID), run.DatasetProfileID),
		IdeaID:                firstNonEmpty(strings.TrimSpace(req.IdeaID), run.IdeaID),
		ReaderContextID:       firstNonEmpty(strings.TrimSpace(req.ReaderContextID), run.ReaderContextID),
		Title:                 strings.TrimSpace(req.Title),
		MachineReadableReport: clonePhase4AnyMap(req.MachineReadableReport),
		HumanReadableReportMD: strings.TrimSpace(req.HumanReadableReportMD),
		CitationRefs:          normalizeStringList(req.CitationRefs),
		ReferenceSourceIDs:    normalizeStringList(req.ReferenceSourceIDs),
		Status:                model.NormalizePhase4ReportStatus(req.Status),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err = s.store.CreateStructuredReport(ctx, item); err != nil {
		return nil, err
	}
	return s.store.GetStructuredReportByID(ctx, item.ID)
}

func (s *Phase4Service) UpdateStructuredReport(ctx context.Context, id string, req model.Phase4StructuredReportUpdateRequest) (*model.Phase4StructuredReportRecord, error) {
	item, err := s.store.GetStructuredReportByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("phase4 structured report not found")
	}
	if req.Title != nil {
		item.Title = strings.TrimSpace(*req.Title)
	}
	if req.MachineReadableReport != nil {
		item.MachineReadableReport = clonePhase4AnyMap(*req.MachineReadableReport)
	}
	if req.HumanReadableReportMD != nil {
		item.HumanReadableReportMD = strings.TrimSpace(*req.HumanReadableReportMD)
	}
	if req.CitationRefs != nil {
		item.CitationRefs = normalizeStringList(*req.CitationRefs)
	}
	if req.ReferenceSourceIDs != nil {
		item.ReferenceSourceIDs = normalizeStringList(*req.ReferenceSourceIDs)
	}
	if req.Status != nil {
		item.Status = model.NormalizePhase4ReportStatus(*req.Status)
	}
	if strings.TrimSpace(item.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	item.UpdatedAt = time.Now()
	if err = s.store.UpdateStructuredReport(ctx, *item); err != nil {
		return nil, err
	}
	return s.store.GetStructuredReportByID(ctx, item.ID)
}

func (s *Phase4Service) ListWorkflows(ctx context.Context, datasetProfileID string, status string) ([]model.Phase4Workflow, error) {
	store, err := s.requireWorkflowStore()
	if err != nil {
		return nil, err
	}
	return store.ListWorkflows(ctx, strings.TrimSpace(datasetProfileID), strings.TrimSpace(strings.ToLower(status)))
}

func (s *Phase4Service) GetWorkflowByID(ctx context.Context, id string) (*model.Phase4Workflow, error) {
	store, err := s.requireWorkflowStore()
	if err != nil {
		return nil, err
	}
	return store.GetWorkflowByID(ctx, strings.TrimSpace(id))
}

func (s *Phase4Service) CreateWorkflow(ctx context.Context, item model.Phase4Workflow) (*model.Phase4Workflow, error) {
	store, err := s.requireWorkflowStore()
	if err != nil {
		return nil, err
	}
	if _, err = s.requireDatasetProfile(ctx, item.DatasetProfileID); err != nil {
		return nil, err
	}
	item.Status = model.NormalizePhase4WorkflowStatus(item.Status)
	item.NextAction = model.NormalizePhase4WorkflowNextAction(item.NextAction)
	item.LastError = strings.TrimSpace(item.LastError)
	item.ManualInputs = clonePhase4AnyMap(item.ManualInputs)
	item.Metadata = clonePhase4AnyMap(item.Metadata)
	if item.ID == "" {
		item.ID = httpx.NewID("p4wf")
	}
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	if err = store.CreateWorkflow(ctx, item); err != nil {
		return nil, err
	}
	return store.GetWorkflowByID(ctx, item.ID)
}

func (s *Phase4Service) UpdateWorkflow(ctx context.Context, id string, item model.Phase4Workflow) (*model.Phase4Workflow, error) {
	store, err := s.requireWorkflowStore()
	if err != nil {
		return nil, err
	}
	existing, err := store.GetWorkflowByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("phase4 workflow not found")
	}
	if strings.TrimSpace(item.DatasetProfileID) == "" {
		item.DatasetProfileID = existing.DatasetProfileID
	}
	if _, err = s.requireDatasetProfile(ctx, item.DatasetProfileID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(item.ReaderContextID) != "" {
		if _, err = s.requireReaderContext(ctx, item.ReaderContextID); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(item.SelectedIdeaID) != "" {
		if _, err = s.requireIdea(ctx, item.SelectedIdeaID); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(item.CurrentRunManifestID) != "" {
		if _, err = s.requireRunManifest(ctx, item.CurrentRunManifestID); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(item.LatestReportID) != "" {
		report, getErr := s.store.GetStructuredReportByID(ctx, item.LatestReportID)
		if getErr != nil {
			return nil, getErr
		}
		if report == nil {
			return nil, fmt.Errorf("phase4 structured report not found")
		}
	}
	if err = model.ValidatePhase4WorkflowTransition(existing.Status, item.Status); err != nil {
		return nil, err
	}
	item.ID = existing.ID
	item.Status = model.NormalizePhase4WorkflowStatus(item.Status)
	item.NextAction = model.NormalizePhase4WorkflowNextAction(item.NextAction)
	item.LastError = strings.TrimSpace(item.LastError)
	item.ManualInputs = clonePhase4AnyMap(item.ManualInputs)
	item.Metadata = clonePhase4AnyMap(item.Metadata)
	item.CreatedAt = existing.CreatedAt
	item.UpdatedAt = time.Now()
	if err = store.UpdateWorkflow(ctx, item); err != nil {
		return nil, err
	}
	return store.GetWorkflowByID(ctx, item.ID)
}

func (s *Phase4Service) ListWorkflowActions(ctx context.Context, workflowID string) ([]model.Phase4WorkflowAction, error) {
	store, err := s.requireWorkflowStore()
	if err != nil {
		return nil, err
	}
	return store.ListWorkflowActions(ctx, strings.TrimSpace(workflowID))
}

func (s *Phase4Service) CreateWorkflowAction(ctx context.Context, item model.Phase4WorkflowAction) (*model.Phase4WorkflowAction, error) {
	store, err := s.requireWorkflowStore()
	if err != nil {
		return nil, err
	}
	workflow, err := s.GetWorkflowByID(ctx, item.WorkflowID)
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, fmt.Errorf("phase4 workflow not found")
	}
	if strings.TrimSpace(item.JobID) != "" && s.store != nil {
		// job existence is validated by FK in repository-backed stores.
	}
	item.Stage = normalizePhase4WorkflowStage(item.Stage)
	item.ActorType = normalizePhase4WorkflowActor(item.ActorType)
	item.Status = normalizePhase4WorkflowActionStatus(item.Status)
	item.ActionType = strings.TrimSpace(item.ActionType)
	item.Payload = clonePhase4AnyMap(item.Payload)
	item.ErrorMessage = strings.TrimSpace(item.ErrorMessage)
	if item.ID == "" {
		item.ID = httpx.NewID("p4wfa")
	}
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	if err = store.CreateWorkflowAction(ctx, item); err != nil {
		return nil, err
	}
	items, err := store.ListWorkflowActions(ctx, item.WorkflowID)
	if err != nil {
		return nil, err
	}
	for _, candidate := range items {
		if candidate.ID == item.ID {
			copyItem := candidate
			return &copyItem, nil
		}
	}
	return &item, nil
}

func (s *Phase4Service) validatePhase4IdeaCreate(ctx context.Context, req model.Phase4IdeaCreateRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(req.ProblemDefinition) == "" {
		return fmt.Errorf("problemDefinition is required")
	}
	if strings.TrimSpace(req.CoreMethod) == "" {
		return fmt.Errorf("coreMethod is required")
	}
	if req.DatasetProfileID != "" {
		if _, err := s.requireDatasetProfile(ctx, req.DatasetProfileID); err != nil {
			return err
		}
	}
	if req.ReaderContextID != "" {
		if _, err := s.requireReaderContext(ctx, req.ReaderContextID); err != nil {
			return err
		}
	}
	if strings.TrimSpace(req.RevisionOfID) != "" {
		if _, err := s.requireIdea(ctx, req.RevisionOfID); err != nil {
			return err
		}
	}
	if err := model.ValidatePhase4IdeaScore(req.Score); err != nil {
		return err
	}
	status := model.NormalizePhase4IdeaStatus(req.Status)
	switch status {
	case model.Phase4IdeaStatusDraft, model.Phase4IdeaStatusScored, model.Phase4IdeaStatusRejected, model.Phase4IdeaStatusSelected, model.Phase4IdeaStatusImplemented, model.Phase4IdeaStatusFailed, model.Phase4IdeaStatusArchived:
		return nil
	default:
		return fmt.Errorf("unsupported idea status: %s", status)
	}
}

func (s *Phase4Service) validateStoredIdea(ctx context.Context, item model.Phase4Idea) error {
	if strings.TrimSpace(item.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(item.ProblemDefinition) == "" {
		return fmt.Errorf("problemDefinition is required")
	}
	if strings.TrimSpace(item.CoreMethod) == "" {
		return fmt.Errorf("coreMethod is required")
	}
	if item.DatasetProfileID != "" {
		if _, err := s.requireDatasetProfile(ctx, item.DatasetProfileID); err != nil {
			return err
		}
	}
	if item.ReaderContextID != "" {
		if _, err := s.requireReaderContext(ctx, item.ReaderContextID); err != nil {
			return err
		}
	}
	if item.RevisionOfID != "" {
		if _, err := s.requireIdea(ctx, item.RevisionOfID); err != nil {
			return err
		}
	}
	return model.ValidatePhase4IdeaScore(item.Score)
}

func (s *Phase4Service) requireDatasetProfile(ctx context.Context, id string) (*model.Phase4DatasetProfile, error) {
	item, err := s.store.GetDatasetProfileByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("phase4 dataset profile not found")
	}
	return item, nil
}

func (s *Phase4Service) requireReaderContext(ctx context.Context, id string) (*model.Phase4ReaderContext, error) {
	item, err := s.store.GetReaderContextByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("phase4 reader context not found")
	}
	return item, nil
}

func (s *Phase4Service) requireIdea(ctx context.Context, id string) (*model.Phase4Idea, error) {
	item, err := s.store.GetIdeaByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("phase4 idea not found")
	}
	return item, nil
}

func (s *Phase4Service) requireRunManifest(ctx context.Context, id string) (*model.Phase4RunManifest, error) {
	item, err := s.store.GetRunManifestByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("phase4 run manifest not found")
	}
	return item, nil
}

func (s *Phase4Service) ensureReaderSourceIDsExist(ctx context.Context, sourceIDs []string) error {
	for _, id := range normalizeStringList(sourceIDs) {
		item, err := s.store.GetReaderSourceByID(ctx, id)
		if err != nil {
			return err
		}
		if item == nil {
			return fmt.Errorf("phase4 reader source not found: %s", id)
		}
	}
	return nil
}

func normalizeDatasetSplits(items []model.Phase4DatasetSplit) []model.Phase4DatasetSplit {
	out := make([]model.Phase4DatasetSplit, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		if item.SampleCount < 0 {
			item.SampleCount = 0
		}
		out = append(out, model.Phase4DatasetSplit{
			Name:        name,
			Path:        strings.TrimSpace(item.Path),
			SampleCount: item.SampleCount,
			Note:        strings.TrimSpace(item.Note),
		})
	}
	return out
}

func normalizeStringList(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
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

func clonePhase4AnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func normalizeReaderContextStatus(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", model.Phase4ReaderContextStatusDraft:
		return model.Phase4ReaderContextStatusDraft
	case model.Phase4ReaderContextStatusReady:
		return model.Phase4ReaderContextStatusReady
	case model.Phase4ReaderContextStatusArchived:
		return model.Phase4ReaderContextStatusArchived
	default:
		return model.Phase4ReaderContextStatusDraft
	}
}

func normalizeSourceType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "manual"
	}
	return value
}

func buildPhase4IdeaScoreView(item model.Phase4Idea) model.Phase4IdeaScoreView {
	return model.Phase4IdeaScoreView{
		ID:                   item.ID,
		DatasetProfileID:     item.DatasetProfileID,
		ReaderContextID:      item.ReaderContextID,
		Title:                item.Title,
		Status:               item.Status,
		SourceType:           item.SourceType,
		RevisionOfID:         item.RevisionOfID,
		LineageRootID:        item.LineageRootID,
		LastFailureRunID:     item.LastFailureRunID,
		Score:                item.Score,
		OverallScore:         phase4FloatValue(item.ScoreSummary["overallScore"]),
		Rank:                 phase4IntValue(item.ScoreSummary["rank"]),
		RecommendationTier:   strings.TrimSpace(fmt.Sprint(item.ScoreSummary["recommendationTier"])),
		RecommendationReason: strings.TrimSpace(fmt.Sprint(item.ScoreSummary["recommendationReason"])),
		ExpectedGains:        append([]string{}, item.ExpectedGains...),
		RiskPoints:           append([]string{}, item.RiskPoints...),
	}
}

func phase4IntValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		number, _ := strconv.Atoi(strings.TrimSpace(typed))
		return number
	default:
		return 0
	}
}

func phase4FloatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		number, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return number
	default:
		return 0
	}
}

func (s *Phase4Service) requireWorkflowStore() (phase4WorkflowStore, error) {
	store, ok := s.store.(phase4WorkflowStore)
	if !ok || store == nil {
		return nil, fmt.Errorf("phase4 workflow store is not configured")
	}
	return store, nil
}

func normalizePhase4WorkflowStage(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", model.Phase4WorkflowStageWorkflow:
		return model.Phase4WorkflowStageWorkflow
	case model.Phase4WorkflowStageReader, model.Phase4WorkflowStageIdea, model.Phase4WorkflowStageCoding, model.Phase4WorkflowStageWriting:
		return value
	default:
		return model.Phase4WorkflowStageWorkflow
	}
}

func normalizePhase4WorkflowActor(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", model.Phase4WorkflowActorSystem:
		return model.Phase4WorkflowActorSystem
	case model.Phase4WorkflowActorUser:
		return model.Phase4WorkflowActorUser
	default:
		return model.Phase4WorkflowActorSystem
	}
}

func normalizePhase4WorkflowActionStatus(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", model.Phase4WorkflowActionStatusStarted:
		return model.Phase4WorkflowActionStatusStarted
	case model.Phase4WorkflowActionStatusSucceeded, model.Phase4WorkflowActionStatusFailed:
		return value
	default:
		return model.Phase4WorkflowActionStatusStarted
	}
}
