package phase4workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"mrag-platform/backend/go/internal/model"
)

const phase4WorkflowRemoteServerName = "shenzhenvlab"

type dataService interface {
	GetDatasetProfileByID(context.Context, string) (*model.Phase4DatasetProfile, error)
	GetReaderContextByID(context.Context, string) (*model.Phase4ReaderContext, error)
	ListIdeas(context.Context, string, string) ([]model.Phase4Idea, error)
	GetIdeaByID(context.Context, string) (*model.Phase4Idea, error)
	SelectIdea(context.Context, string) (*model.Phase4Idea, error)
	GetRunManifestByID(context.Context, string) (*model.Phase4RunManifest, error)
	GetStructuredReportByID(context.Context, string) (*model.Phase4StructuredReportRecord, error)
	ListWorkflows(context.Context, string, string) ([]model.Phase4Workflow, error)
	GetWorkflowByID(context.Context, string) (*model.Phase4Workflow, error)
	CreateWorkflow(context.Context, model.Phase4Workflow) (*model.Phase4Workflow, error)
	UpdateWorkflow(context.Context, string, model.Phase4Workflow) (*model.Phase4Workflow, error)
	ListWorkflowActions(context.Context, string) ([]model.Phase4WorkflowAction, error)
	CreateWorkflowAction(context.Context, model.Phase4WorkflowAction) (*model.Phase4WorkflowAction, error)
}

type readerRunner interface {
	Run(context.Context, model.Phase4ReaderRunRequest) (*model.Phase4ReaderRunResult, error)
}

type ideaRunner interface {
	Run(context.Context, model.Phase4IdeaRunRequest) (*model.Phase4IdeaRunResult, error)
}

type codingRunner interface {
	Run(context.Context, model.Phase4CodingRunRequest) (*model.Phase4CodingRunResult, error)
}

type writerRunner interface {
	Run(context.Context, model.Phase4WriterRunRequest) (*model.Phase4WriterRunResult, error)
}

type jobGetter interface {
	GetByID(context.Context, string) (*model.AgentJob, error)
}

type eventPublisher interface {
	PublishEvent(context.Context, model.AgentEventCreateRequest) (*model.AgentEvent, error)
}

type Service struct {
	data   dataService
	reader readerRunner
	idea   ideaRunner
	coding codingRunner
	writer writerRunner
	jobs   jobGetter
	events eventPublisher
}

func NewService(
	data dataService,
	reader readerRunner,
	idea ideaRunner,
	coding codingRunner,
	writer writerRunner,
	jobs jobGetter,
	events eventPublisher,
) *Service {
	return &Service{
		data:   data,
		reader: reader,
		idea:   idea,
		coding: coding,
		writer: writer,
		jobs:   jobs,
		events: events,
	}
}

func (s *Service) CreateWorkflow(ctx context.Context, req model.Phase4WorkflowCreateRequest) (*model.Phase4WorkflowDetail, error) {
	if s.data == nil || s.reader == nil || s.idea == nil || s.coding == nil || s.writer == nil {
		return nil, fmt.Errorf("phase4 workflow service is not fully configured")
	}
	req.DatasetProfileID = strings.TrimSpace(req.DatasetProfileID)
	if req.DatasetProfileID == "" {
		return nil, fmt.Errorf("datasetProfileId is required")
	}
	datasetProfile, err := s.data.GetDatasetProfileByID(ctx, req.DatasetProfileID)
	if err != nil {
		return nil, err
	}
	if datasetProfile == nil {
		return nil, fmt.Errorf("phase4 dataset profile not found")
	}
	readerCfg := normalizeReaderConfig(req.Reader)
	ideaCfg := normalizeIdeaConfig(req.Idea)
	codingCfg := normalizeCodingConfig(req.Coding, datasetProfile.ServerID)
	writingCfg := normalizeWritingConfig(req.Writing)
	workflow, err := s.data.CreateWorkflow(ctx, model.Phase4Workflow{
		DatasetProfileID: req.DatasetProfileID,
		Status:           model.Phase4WorkflowStatusRunningReader,
		NextAction:       model.Phase4WorkflowNextActionNone,
		LastError:        "",
		ManualInputs: map[string]any{
			"reader":  configMap(readerCfg),
			"idea":    configMap(ideaCfg),
			"coding":  configMap(codingCfg),
			"writing": configMap(writingCfg),
		},
		Metadata: cloneMap(req.Metadata),
	})
	if err != nil {
		return nil, err
	}
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID: workflow.ID,
		Stage:      model.Phase4WorkflowStageWorkflow,
		ActionType: "create",
		ActorType:  model.Phase4WorkflowActorUser,
		Status:     model.Phase4WorkflowActionStatusSucceeded,
		Payload: map[string]any{
			"dataset_profile_id": workflow.DatasetProfileID,
		},
	})
	s.safePublishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "phase4_workflow_started",
		SourceRef: "phase4_workflow:" + workflow.ID,
		InputRefs: []model.AgentInputRef{
			{RefType: "phase4_workflow", RefID: workflow.ID},
			{RefType: "phase4_dataset_profile", RefID: workflow.DatasetProfileID},
		},
		Payload: map[string]any{
			"workflow_id":        workflow.ID,
			"dataset_profile_id": workflow.DatasetProfileID,
		},
	})
	if workflow, err = s.runReaderStage(ctx, workflow, readerCfg); err != nil {
		return nil, err
	}
	if workflow, err = s.runIdeaStage(ctx, workflow, ideaCfg); err != nil {
		return nil, err
	}
	return s.GetWorkflow(ctx, workflow.ID)
}

func (s *Service) ListWorkflows(ctx context.Context, datasetProfileID string, status string) ([]model.Phase4Workflow, error) {
	if s.data == nil {
		return nil, fmt.Errorf("phase4 workflow data service is not configured")
	}
	return s.data.ListWorkflows(ctx, strings.TrimSpace(datasetProfileID), strings.TrimSpace(status))
}

func (s *Service) GetWorkflow(ctx context.Context, workflowID string) (*model.Phase4WorkflowDetail, error) {
	if s.data == nil {
		return nil, fmt.Errorf("phase4 workflow data service is not configured")
	}
	workflow, err := s.data.GetWorkflowByID(ctx, strings.TrimSpace(workflowID))
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, nil
	}
	detail := &model.Phase4WorkflowDetail{
		Workflow:           workflow,
		Ideas:              []model.Phase4Idea{},
		TopRecommendations: []model.Phase4IdeaScoreView{},
		NextActions:        workflowNextActions(workflow.Status),
		Timeline:           []model.Phase4WorkflowAction{},
	}
	if detail.DatasetProfile, err = s.data.GetDatasetProfileByID(ctx, workflow.DatasetProfileID); err != nil {
		return nil, err
	}
	if workflow.ReaderContextID != "" {
		if detail.ReaderContext, err = s.data.GetReaderContextByID(ctx, workflow.ReaderContextID); err != nil {
			return nil, err
		}
	}
	if workflow.SelectedIdeaID != "" {
		if detail.SelectedIdea, err = s.data.GetIdeaByID(ctx, workflow.SelectedIdeaID); err != nil {
			return nil, err
		}
	}
	if workflow.CurrentRunManifestID != "" {
		if detail.CurrentRunManifest, err = s.data.GetRunManifestByID(ctx, workflow.CurrentRunManifestID); err != nil {
			return nil, err
		}
	}
	if workflow.LatestReportID != "" {
		if detail.LatestReport, err = s.data.GetStructuredReportByID(ctx, workflow.LatestReportID); err != nil {
			return nil, err
		}
	}
	if detail.Timeline, err = s.data.ListWorkflowActions(ctx, workflow.ID); err != nil {
		return nil, err
	}
	if detail.LatestJobs.Reader, err = s.getJob(ctx, workflow.LatestReaderJobID); err != nil {
		return nil, err
	}
	if detail.LatestJobs.Idea, err = s.getJob(ctx, workflow.LatestIdeaJobID); err != nil {
		return nil, err
	}
	if detail.LatestJobs.Coding, err = s.getJob(ctx, workflow.LatestCodingJobID); err != nil {
		return nil, err
	}
	if detail.LatestJobs.Writer, err = s.getJob(ctx, workflow.LatestWriterJobID); err != nil {
		return nil, err
	}
	ideaIDs := orderedUniqueStrings(
		stringSliceValue(workflow.Metadata["idea_ids"]),
		stringSliceValue(workflow.Metadata["revision_idea_ids"]),
		[]string{workflow.SelectedIdeaID},
	)
	for _, ideaID := range ideaIDs {
		item, getErr := s.data.GetIdeaByID(ctx, ideaID)
		if getErr != nil {
			return nil, getErr
		}
		if item != nil {
			detail.Ideas = append(detail.Ideas, *item)
		}
	}
	topIDs := stringSliceValue(workflow.Metadata["top_recommendation_ids"])
	if workflow.Status == model.Phase4WorkflowStatusAwaitingRevisionSelect {
		topIDs = stringSliceValue(workflow.Metadata["revision_top_recommendation_ids"])
	}
	for _, ideaID := range topIDs {
		item, getErr := s.data.GetIdeaByID(ctx, ideaID)
		if getErr != nil {
			return nil, getErr
		}
		if item != nil {
			detail.TopRecommendations = append(detail.TopRecommendations, ideaToScoreView(*item))
		}
	}
	sort.SliceStable(detail.Ideas, func(i, j int) bool {
		if detail.Ideas[i].CreatedAt.Equal(detail.Ideas[j].CreatedAt) {
			return detail.Ideas[i].ID < detail.Ideas[j].ID
		}
		return detail.Ideas[i].CreatedAt.Before(detail.Ideas[j].CreatedAt)
	})
	return detail, nil
}

func (s *Service) ArchiveWorkflow(ctx context.Context, workflowID string) (*model.Phase4WorkflowDetail, error) {
	workflow, err := s.data.GetWorkflowByID(ctx, strings.TrimSpace(workflowID))
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, fmt.Errorf("phase4 workflow not found")
	}
	workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusArchived, model.Phase4WorkflowNextActionNone, workflow.LastError))
	if err != nil {
		return nil, err
	}
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID: workflow.ID,
		Stage:      model.Phase4WorkflowStageWorkflow,
		ActionType: "archive",
		ActorType:  model.Phase4WorkflowActorUser,
		Status:     model.Phase4WorkflowActionStatusSucceeded,
		Payload:    map[string]any{},
	})
	return s.GetWorkflow(ctx, workflow.ID)
}

func (s *Service) SelectIdea(ctx context.Context, workflowID string, req model.Phase4WorkflowSelectIdeaRequest) (*model.Phase4WorkflowDetail, error) {
	return s.selectIdea(ctx, workflowID, req, false)
}

func (s *Service) SelectRevision(ctx context.Context, workflowID string, req model.Phase4WorkflowSelectIdeaRequest) (*model.Phase4WorkflowDetail, error) {
	return s.selectIdea(ctx, workflowID, req, true)
}

func (s *Service) RetryStage(ctx context.Context, workflowID string, req model.Phase4WorkflowRetryStageRequest) (*model.Phase4WorkflowDetail, error) {
	workflow, err := s.data.GetWorkflowByID(ctx, strings.TrimSpace(workflowID))
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, fmt.Errorf("phase4 workflow not found")
	}
	if workflow.Status != model.Phase4WorkflowStatusBlocked {
		return nil, fmt.Errorf("phase4 workflow is not blocked")
	}
	failedStage := strings.TrimSpace(strings.ToLower(stringValue(workflow.Metadata["failed_stage"])))
	if failedStage == "" {
		return nil, fmt.Errorf("phase4 workflow has no failed stage to retry")
	}
	datasetProfile, err := s.data.GetDatasetProfileByID(ctx, workflow.DatasetProfileID)
	if err != nil {
		return nil, err
	}
	datasetServerID := ""
	if datasetProfile != nil {
		datasetServerID = datasetProfile.ServerID
	}
	workflow.ManualInputs = cloneMap(workflow.ManualInputs)
	codingCfg := mergeCodingConfig(decodeCodingConfig(workflow.ManualInputs["coding"]), req.Coding, datasetServerID)
	writingCfg := mergeWritingConfig(decodeWritingConfig(workflow.ManualInputs["writing"]), req.Writing)
	workflow.ManualInputs["coding"] = configMap(codingCfg)
	workflow.ManualInputs["writing"] = configMap(writingCfg)
	workflow.Metadata = cloneMap(workflow.Metadata)
	workflow.Metadata["retry_stage"] = failedStage
	workflow.Metadata["retry_user_notes"] = strings.TrimSpace(req.UserNotes)
	workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, *workflow)
	if err != nil {
		return nil, err
	}
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID: workflow.ID,
		Stage:      model.Phase4WorkflowStageWorkflow,
		ActionType: "retry_stage",
		ActorType:  model.Phase4WorkflowActorUser,
		Status:     model.Phase4WorkflowActionStatusSucceeded,
		Payload: map[string]any{
			"failed_stage": failedStage,
			"user_notes":   strings.TrimSpace(req.UserNotes),
		},
	})
	switch failedStage {
	case model.Phase4WorkflowStageReader:
		workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusRunningReader, model.Phase4WorkflowNextActionNone, ""))
		if err != nil {
			return nil, err
		}
		if workflow, err = s.runReaderStage(ctx, workflow, decodeReaderConfig(workflow.ManualInputs["reader"])); err != nil {
			return nil, err
		}
		if workflow.Status == model.Phase4WorkflowStatusRunningIdea {
			if workflow, err = s.runIdeaStage(ctx, workflow, decodeIdeaConfig(workflow.ManualInputs["idea"])); err != nil {
				return nil, err
			}
		}
	case model.Phase4WorkflowStageIdea:
		workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusRunningIdea, model.Phase4WorkflowNextActionNone, ""))
		if err != nil {
			return nil, err
		}
		if workflow, err = s.runIdeaStage(ctx, workflow, decodeIdeaConfig(workflow.ManualInputs["idea"])); err != nil {
			return nil, err
		}
	case model.Phase4WorkflowStageCoding:
		selectedIdea, getErr := s.data.GetIdeaByID(ctx, workflow.SelectedIdeaID)
		if getErr != nil {
			return nil, getErr
		}
		if selectedIdea == nil {
			return nil, fmt.Errorf("phase4 workflow selected idea not found")
		}
		workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusRunningCoding, model.Phase4WorkflowNextActionNone, ""))
		if err != nil {
			return nil, err
		}
		if workflow, err = s.runCodingStage(ctx, workflow, selectedIdea, codingCfg, writingCfg); err != nil {
			return nil, err
		}
	case model.Phase4WorkflowStageWriting:
		runManifest, getErr := s.data.GetRunManifestByID(ctx, workflow.CurrentRunManifestID)
		if getErr != nil {
			return nil, getErr
		}
		if runManifest == nil {
			return nil, fmt.Errorf("phase4 workflow current run manifest not found")
		}
		workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusRunningWriting, model.Phase4WorkflowNextActionNone, ""))
		if err != nil {
			return nil, err
		}
		if workflow, err = s.runWritingStage(ctx, workflow, runManifest, writingCfg); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported failed stage: %s", failedStage)
	}
	return s.GetWorkflow(ctx, workflow.ID)
}

func (s *Service) selectIdea(ctx context.Context, workflowID string, req model.Phase4WorkflowSelectIdeaRequest, revision bool) (*model.Phase4WorkflowDetail, error) {
	workflow, err := s.data.GetWorkflowByID(ctx, strings.TrimSpace(workflowID))
	if err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, fmt.Errorf("phase4 workflow not found")
	}
	expectedStatus := model.Phase4WorkflowStatusAwaitingSelection
	actionType := "select_idea"
	allowedIdeaIDs := stringSliceValue(workflow.Metadata["idea_ids"])
	if revision {
		expectedStatus = model.Phase4WorkflowStatusAwaitingRevisionSelect
		actionType = "select_revision"
		allowedIdeaIDs = stringSliceValue(workflow.Metadata["revision_idea_ids"])
	}
	if workflow.Status != expectedStatus {
		return nil, fmt.Errorf("phase4 workflow is not ready for %s", actionType)
	}
	ideaID := strings.TrimSpace(req.IdeaID)
	if ideaID == "" {
		return nil, fmt.Errorf("ideaId is required")
	}
	if len(allowedIdeaIDs) > 0 && !containsStringFold(allowedIdeaIDs, ideaID) {
		return nil, fmt.Errorf("idea is not available for this workflow state")
	}
	idea, err := s.data.GetIdeaByID(ctx, ideaID)
	if err != nil {
		return nil, err
	}
	if idea == nil {
		return nil, fmt.Errorf("phase4 idea not found")
	}
	if idea.DatasetProfileID != workflow.DatasetProfileID {
		return nil, fmt.Errorf("phase4 idea does not belong to the workflow dataset")
	}
	datasetProfile, err := s.data.GetDatasetProfileByID(ctx, workflow.DatasetProfileID)
	if err != nil {
		return nil, err
	}
	datasetServerID := ""
	if datasetProfile != nil {
		datasetServerID = datasetProfile.ServerID
	}
	selectedIdea, err := s.data.SelectIdea(ctx, idea.ID)
	if err != nil {
		_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
			WorkflowID:   workflow.ID,
			Stage:        model.Phase4WorkflowStageIdea,
			ActionType:   actionType,
			ActorType:    model.Phase4WorkflowActorUser,
			Status:       model.Phase4WorkflowActionStatusFailed,
			ErrorMessage: err.Error(),
			Payload: map[string]any{
				"idea_id": ideaID,
			},
		})
		return nil, err
	}
	workflow.ManualInputs = cloneMap(workflow.ManualInputs)
	codingCfg := mergeCodingConfig(decodeCodingConfig(workflow.ManualInputs["coding"]), req.Coding, datasetServerID)
	writingCfg := mergeWritingConfig(decodeWritingConfig(workflow.ManualInputs["writing"]), req.Writing)
	workflow.ManualInputs["coding"] = configMap(codingCfg)
	workflow.ManualInputs["writing"] = configMap(writingCfg)
	workflow.Metadata = cloneMap(workflow.Metadata)
	workflow.Metadata["selected_idea_id"] = selectedIdea.ID
	workflow.Metadata["selection_mode"] = actionType
	workflow.Metadata["selection_user_notes"] = strings.TrimSpace(req.UserNotes)
	workflow.Metadata["failed_stage"] = ""
	workflow.Metadata["last_failure_feedback"] = map[string]any{}
	workflow.Metadata["latest_failed_run_id"] = ""
	if revision {
		if len(stringSliceValue(workflow.Metadata["revision_top_recommendation_ids"])) > 0 {
			workflow.Metadata["top_recommendation_ids"] = stringSliceValue(workflow.Metadata["revision_top_recommendation_ids"])
		}
	}
	workflow.SelectedIdeaID = selectedIdea.ID
	workflow.LatestReportID = ""
	workflow.LastError = ""
	workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusRunningCoding, model.Phase4WorkflowNextActionNone, ""))
	if err != nil {
		return nil, err
	}
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID: workflow.ID,
		Stage:      model.Phase4WorkflowStageIdea,
		ActionType: actionType,
		ActorType:  model.Phase4WorkflowActorUser,
		Status:     model.Phase4WorkflowActionStatusSucceeded,
		Payload: map[string]any{
			"idea_id":    selectedIdea.ID,
			"idea_title": selectedIdea.Title,
			"user_notes": strings.TrimSpace(req.UserNotes),
		},
	})
	s.safePublishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "phase4_idea_selected",
		SourceRef: "phase4_workflow:" + workflow.ID,
		InputRefs: []model.AgentInputRef{
			{RefType: "phase4_workflow", RefID: workflow.ID},
			{RefType: "phase4_dataset_profile", RefID: workflow.DatasetProfileID},
			{RefType: "phase4_idea", RefID: selectedIdea.ID},
		},
		Payload: map[string]any{
			"workflow_id":        workflow.ID,
			"dataset_profile_id": workflow.DatasetProfileID,
			"idea_id":            selectedIdea.ID,
			"selection_mode":     actionType,
		},
	})
	if workflow, err = s.runCodingStage(ctx, workflow, selectedIdea, codingCfg, writingCfg); err != nil {
		return nil, err
	}
	return s.GetWorkflow(ctx, workflow.ID)
}

func (s *Service) runReaderStage(ctx context.Context, workflow *model.Phase4Workflow, cfg model.Phase4WorkflowReaderConfig) (*model.Phase4Workflow, error) {
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID: workflow.ID,
		Stage:      model.Phase4WorkflowStageReader,
		ActionType: "run_reader",
		ActorType:  model.Phase4WorkflowActorSystem,
		Status:     model.Phase4WorkflowActionStatusStarted,
		Payload: map[string]any{
			"dataset_profile_id": workflow.DatasetProfileID,
			"search_mode":        cfg.SearchMode,
			"max_papers":         cfg.MaxPapers,
		},
	})
	result, err := s.reader.Run(ctx, model.Phase4ReaderRunRequest{
		DatasetProfileID: workflow.DatasetProfileID,
		ManualPapers:     cfg.ManualPapers,
		UserNotes:        cfg.UserNotes,
		SearchMode:       cfg.SearchMode,
		MaxPapers:        cfg.MaxPapers,
		ExecutionMode:    cfg.ExecutionMode,
		ModelProvider:    cfg.ModelProvider,
		ModelName:        cfg.ModelName,
		PromptVersion:    cfg.PromptVersion,
		SkillRefs:        cfg.SkillRefs,
		ToolRefs:         cfg.ToolRefs,
		MemoryRefs:       cfg.MemoryRefs,
	})
	if err != nil {
		return s.failWorkflowStage(ctx, workflow, model.Phase4WorkflowStageReader, "run_reader", "", "", map[string]any{
			"search_mode": cfg.SearchMode,
			"max_papers":  cfg.MaxPapers,
		}, err)
	}
	if result == nil || result.Job == nil || result.ReaderContext == nil {
		return s.failWorkflowStage(ctx, workflow, model.Phase4WorkflowStageReader, "run_reader", "", "", map[string]any{
			"reason": "phase4 reader returned incomplete result",
		}, fmt.Errorf("phase4 reader returned incomplete result"))
	}
	workflow.Metadata = cloneMap(workflow.Metadata)
	workflow.Metadata["reader_source_ids"] = extractReaderSourceIDs(result.ReaderSources)
	workflow.Metadata["failed_stage"] = ""
	workflow.Metadata["last_failure_feedback"] = map[string]any{}
	workflow.ReaderContextID = result.ReaderContext.ID
	workflow.LatestReaderJobID = result.Job.ID
	workflow.LastError = ""
	workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusRunningIdea, model.Phase4WorkflowNextActionNone, ""))
	if err != nil {
		return nil, err
	}
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID: workflow.ID,
		Stage:      model.Phase4WorkflowStageReader,
		ActionType: "run_reader",
		ActorType:  model.Phase4WorkflowActorSystem,
		Status:     model.Phase4WorkflowActionStatusSucceeded,
		JobID:      result.Job.ID,
		Payload: map[string]any{
			"reader_context_id": result.ReaderContext.ID,
			"source_ids":        extractReaderSourceIDs(result.ReaderSources),
			"warnings":          append([]string{}, result.Warnings...),
		},
	})
	s.safePublishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "phase4_reader_ready",
		SourceRef: "phase4_workflow:" + workflow.ID,
		InputRefs: []model.AgentInputRef{
			{RefType: "phase4_workflow", RefID: workflow.ID},
			{RefType: "phase4_dataset_profile", RefID: workflow.DatasetProfileID},
			{RefType: "phase4_reader_context", RefID: result.ReaderContext.ID},
		},
		Payload: map[string]any{
			"workflow_id":        workflow.ID,
			"dataset_profile_id": workflow.DatasetProfileID,
			"reader_context_id":  result.ReaderContext.ID,
			"reader_job_id":      result.Job.ID,
		},
	})
	return workflow, nil
}

func (s *Service) runIdeaStage(ctx context.Context, workflow *model.Phase4Workflow, cfg model.Phase4WorkflowIdeaConfig) (*model.Phase4Workflow, error) {
	if strings.TrimSpace(workflow.ReaderContextID) == "" {
		return s.failWorkflowStage(ctx, workflow, model.Phase4WorkflowStageIdea, "run_idea", "", "", map[string]any{}, fmt.Errorf("phase4 workflow reader context is required before idea generation"))
	}
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID: workflow.ID,
		Stage:      model.Phase4WorkflowStageIdea,
		ActionType: "run_idea",
		ActorType:  model.Phase4WorkflowActorSystem,
		Status:     model.Phase4WorkflowActionStatusStarted,
		Payload: map[string]any{
			"reader_context_id": workflow.ReaderContextID,
			"target_count":      cfg.TargetCount,
		},
	})
	result, err := s.idea.Run(ctx, model.Phase4IdeaRunRequest{
		DatasetProfileID: workflow.DatasetProfileID,
		ReaderContextID:  workflow.ReaderContextID,
		UserNotes:        cfg.UserNotes,
		ManualIdea:       cfg.ManualIdea,
		TargetCount:      cfg.TargetCount,
		ExecutionMode:    cfg.ExecutionMode,
		ModelProvider:    cfg.ModelProvider,
		ModelName:        cfg.ModelName,
		PromptVersion:    cfg.PromptVersion,
		SkillRefs:        cfg.SkillRefs,
		ToolRefs:         cfg.ToolRefs,
		MemoryRefs:       cfg.MemoryRefs,
	})
	if err != nil {
		return s.failWorkflowStage(ctx, workflow, model.Phase4WorkflowStageIdea, "run_idea", "", "", map[string]any{
			"reader_context_id": workflow.ReaderContextID,
			"target_count":      cfg.TargetCount,
		}, err)
	}
	if result == nil || result.Job == nil || len(result.Ideas) == 0 {
		return s.failWorkflowStage(ctx, workflow, model.Phase4WorkflowStageIdea, "run_idea", "", "", map[string]any{
			"reader_context_id": workflow.ReaderContextID,
		}, fmt.Errorf("phase4 idea returned no persisted ideas"))
	}
	ideaIDs := extractIdeaIDs(result.Ideas)
	topIDs := extractTopIdeaIDs(result.TopRecommendations)
	workflow.Metadata = cloneMap(workflow.Metadata)
	workflow.Metadata["idea_ids"] = ideaIDs
	workflow.Metadata["top_recommendation_ids"] = topIDs
	workflow.Metadata["revision_idea_ids"] = []string{}
	workflow.Metadata["revision_top_recommendation_ids"] = []string{}
	workflow.Metadata["failed_stage"] = ""
	workflow.Metadata["last_failure_feedback"] = map[string]any{}
	workflow.LatestIdeaJobID = result.Job.ID
	workflow.LastError = ""
	workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusAwaitingSelection, model.Phase4WorkflowNextActionSelectIdea, ""))
	if err != nil {
		return nil, err
	}
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID: workflow.ID,
		Stage:      model.Phase4WorkflowStageIdea,
		ActionType: "run_idea",
		ActorType:  model.Phase4WorkflowActorSystem,
		Status:     model.Phase4WorkflowActionStatusSucceeded,
		JobID:      result.Job.ID,
		Payload: map[string]any{
			"idea_ids":               ideaIDs,
			"top_recommendation_ids": topIDs,
			"warnings":               append([]string{}, result.Warnings...),
		},
	})
	s.safePublishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "phase4_idea_batch_ready",
		SourceRef: "phase4_workflow:" + workflow.ID,
		InputRefs: []model.AgentInputRef{
			{RefType: "phase4_workflow", RefID: workflow.ID},
			{RefType: "phase4_dataset_profile", RefID: workflow.DatasetProfileID},
			{RefType: "phase4_reader_context", RefID: workflow.ReaderContextID},
		},
		Payload: map[string]any{
			"workflow_id":            workflow.ID,
			"dataset_profile_id":     workflow.DatasetProfileID,
			"reader_context_id":      workflow.ReaderContextID,
			"idea_job_id":            result.Job.ID,
			"idea_ids":               ideaIDs,
			"top_recommendation_ids": topIDs,
		},
	})
	return workflow, nil
}

func (s *Service) runCodingStage(ctx context.Context, workflow *model.Phase4Workflow, selectedIdea *model.Phase4Idea, cfg model.Phase4WorkflowCodingConfig, writingCfg model.Phase4WorkflowWritingConfig) (*model.Phase4Workflow, error) {
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID: workflow.ID,
		Stage:      model.Phase4WorkflowStageCoding,
		ActionType: "run_coding",
		ActorType:  model.Phase4WorkflowActorSystem,
		Status:     model.Phase4WorkflowActionStatusStarted,
		Payload: map[string]any{
			"idea_id":         selectedIdea.ID,
			"runner_mode":     cfg.RunnerMode,
			"server_id":       cfg.ServerID,
			"gpu":             cfg.GPU,
			"max_retry_count": cfg.MaxRetryCount,
		},
	})
	result, err := s.coding.Run(ctx, model.Phase4CodingRunRequest{
		DatasetProfileID: workflow.DatasetProfileID,
		IdeaID:           selectedIdea.ID,
		ReaderContextID:  workflow.ReaderContextID,
		RunnerMode:       cfg.RunnerMode,
		ServerID:         cfg.ServerID,
		GPU:              cfg.GPU,
		MaxRetryCount:    cfg.MaxRetryCount,
		UserNotes:        cfg.UserNotes,
		ExecutionMode:    cfg.ExecutionMode,
		ModelProvider:    cfg.ModelProvider,
		ModelName:        cfg.ModelName,
		PromptVersion:    cfg.PromptVersion,
		SkillRefs:        cfg.SkillRefs,
		ToolRefs:         cfg.ToolRefs,
		MemoryRefs:       cfg.MemoryRefs,
	})
	if err != nil {
		return s.failWorkflowStage(ctx, workflow, model.Phase4WorkflowStageCoding, "run_coding", "", "", map[string]any{
			"idea_id":     selectedIdea.ID,
			"runner_mode": cfg.RunnerMode,
		}, err)
	}
	if result == nil || result.Job == nil || result.RunManifest == nil {
		return s.failWorkflowStage(ctx, workflow, model.Phase4WorkflowStageCoding, "run_coding", "", "", map[string]any{
			"idea_id": selectedIdea.ID,
		}, fmt.Errorf("phase4 coding returned incomplete result"))
	}
	runManifest := result.RunManifest
	workflow.Metadata = cloneMap(workflow.Metadata)
	workflow.Metadata["current_run_status"] = runManifest.Status
	workflow.Metadata["current_run_manifest_id"] = runManifest.ID
	workflow.CurrentRunManifestID = runManifest.ID
	workflow.LatestCodingJobID = result.Job.ID
	switch runManifest.Status {
	case model.Phase4RunStatusSucceeded:
		workflow.Metadata["failed_stage"] = ""
		workflow.Metadata["last_failure_feedback"] = map[string]any{}
		workflow.LastError = ""
		workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusRunningWriting, model.Phase4WorkflowNextActionNone, ""))
		if err != nil {
			return nil, err
		}
		_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
			WorkflowID:    workflow.ID,
			Stage:         model.Phase4WorkflowStageCoding,
			ActionType:    "run_coding",
			ActorType:     model.Phase4WorkflowActorSystem,
			Status:        model.Phase4WorkflowActionStatusSucceeded,
			JobID:         result.Job.ID,
			RunManifestID: runManifest.ID,
			Payload: map[string]any{
				"run_status": runManifest.Status,
				"warnings":   append([]string{}, result.Warnings...),
			},
		})
		return s.runWritingStage(ctx, workflow, runManifest, writingCfg)
	case model.Phase4RunStatusTestFailed:
		workflow.Metadata["failed_stage"] = model.Phase4WorkflowStageCoding
		workflow.Metadata["last_failure_feedback"] = cloneMap(runManifest.FailureFeedback)
		workflow.Metadata["latest_failed_run_id"] = runManifest.ID
		workflow.Metadata["revision_idea_ids"] = stringSliceValue(runManifest.ArtifactPaths["revision_idea_ids"])
		workflow.Metadata["revision_top_recommendation_ids"] = stringSliceValue(runManifest.ArtifactPaths["revision_top_idea_ids"])
		workflow.LastError = summarizeFailure(runManifest.FailureFeedback)
		workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusAwaitingRevisionSelect, model.Phase4WorkflowNextActionSelectRevision, workflow.LastError))
		if err != nil {
			return nil, err
		}
		_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
			WorkflowID:    workflow.ID,
			Stage:         model.Phase4WorkflowStageCoding,
			ActionType:    "run_coding",
			ActorType:     model.Phase4WorkflowActorSystem,
			Status:        model.Phase4WorkflowActionStatusFailed,
			JobID:         result.Job.ID,
			RunManifestID: runManifest.ID,
			Payload: map[string]any{
				"run_status":        runManifest.Status,
				"failure_feedback":  cloneMap(runManifest.FailureFeedback),
				"revision_idea_ids": stringSliceValue(runManifest.ArtifactPaths["revision_idea_ids"]),
				"revision_top_ids":  stringSliceValue(runManifest.ArtifactPaths["revision_top_idea_ids"]),
			},
			ErrorMessage: workflow.LastError,
		})
		s.safePublishEvent(ctx, model.AgentEventCreateRequest{
			EventType: "phase4_coding_test_failed",
			SourceRef: "phase4_workflow:" + workflow.ID,
			InputRefs: []model.AgentInputRef{
				{RefType: "phase4_workflow", RefID: workflow.ID},
				{RefType: "phase4_run_manifest", RefID: runManifest.ID},
				{RefType: "phase4_idea", RefID: selectedIdea.ID},
			},
			Payload: map[string]any{
				"workflow_id":           workflow.ID,
				"run_manifest_id":       runManifest.ID,
				"idea_id":               selectedIdea.ID,
				"failure_feedback":      cloneMap(runManifest.FailureFeedback),
				"revision_idea_ids":     stringSliceValue(runManifest.ArtifactPaths["revision_idea_ids"]),
				"revision_top_idea_ids": stringSliceValue(runManifest.ArtifactPaths["revision_top_idea_ids"]),
			},
		})
		return workflow, nil
	default:
		return s.failWorkflowStage(ctx, workflow, model.Phase4WorkflowStageCoding, "run_coding", result.Job.ID, runManifest.ID, map[string]any{
			"run_status":       runManifest.Status,
			"failure_feedback": cloneMap(runManifest.FailureFeedback),
		}, fmt.Errorf("phase4 coding ended with run status %s", runManifest.Status))
	}
}

func (s *Service) runWritingStage(ctx context.Context, workflow *model.Phase4Workflow, runManifest *model.Phase4RunManifest, cfg model.Phase4WorkflowWritingConfig) (*model.Phase4Workflow, error) {
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID:    workflow.ID,
		Stage:         model.Phase4WorkflowStageWriting,
		ActionType:    "run_writing",
		ActorType:     model.Phase4WorkflowActorSystem,
		Status:        model.Phase4WorkflowActionStatusStarted,
		RunManifestID: runManifest.ID,
		Payload: map[string]any{
			"run_manifest_id": runManifest.ID,
		},
	})
	result, err := s.writer.Run(ctx, model.Phase4WriterRunRequest{
		RunManifestID: runManifest.ID,
		UserNotes:     cfg.UserNotes,
		ExecutionMode: cfg.ExecutionMode,
		ModelProvider: cfg.ModelProvider,
		ModelName:     cfg.ModelName,
		PromptVersion: cfg.PromptVersion,
		SkillRefs:     cfg.SkillRefs,
		ToolRefs:      cfg.ToolRefs,
		MemoryRefs:    cfg.MemoryRefs,
	})
	if err != nil {
		return s.failWorkflowStage(ctx, workflow, model.Phase4WorkflowStageWriting, "run_writing", "", runManifest.ID, map[string]any{
			"run_manifest_id": runManifest.ID,
		}, err)
	}
	if result == nil || result.Job == nil || result.Report == nil {
		return s.failWorkflowStage(ctx, workflow, model.Phase4WorkflowStageWriting, "run_writing", "", runManifest.ID, map[string]any{
			"run_manifest_id": runManifest.ID,
		}, fmt.Errorf("phase4 writer returned incomplete result"))
	}
	workflow.Metadata = cloneMap(workflow.Metadata)
	workflow.Metadata["failed_stage"] = ""
	workflow.Metadata["last_failure_feedback"] = map[string]any{}
	workflow.Metadata["latest_report_id"] = result.Report.ID
	workflow.LatestWriterJobID = result.Job.ID
	workflow.LatestReportID = result.Report.ID
	workflow.LastError = ""
	workflow, err = s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusCompleted, model.Phase4WorkflowNextActionViewReport, ""))
	if err != nil {
		return nil, err
	}
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID:    workflow.ID,
		Stage:         model.Phase4WorkflowStageWriting,
		ActionType:    "run_writing",
		ActorType:     model.Phase4WorkflowActorSystem,
		Status:        model.Phase4WorkflowActionStatusSucceeded,
		JobID:         result.Job.ID,
		RunManifestID: runManifest.ID,
		ReportID:      result.Report.ID,
		Payload: map[string]any{
			"report_id": result.Report.ID,
			"title":     result.Report.Title,
		},
	})
	s.safePublishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "phase4_workflow_completed",
		SourceRef: "phase4_workflow:" + workflow.ID,
		InputRefs: []model.AgentInputRef{
			{RefType: "phase4_workflow", RefID: workflow.ID},
			{RefType: "phase4_run_manifest", RefID: runManifest.ID},
			{RefType: "phase4_report", RefID: result.Report.ID},
		},
		Payload: map[string]any{
			"workflow_id":     workflow.ID,
			"run_manifest_id": runManifest.ID,
			"report_id":       result.Report.ID,
		},
	})
	return workflow, nil
}

func (s *Service) failWorkflowStage(ctx context.Context, workflow *model.Phase4Workflow, stage string, actionType string, jobID string, runManifestID string, payload map[string]any, stageErr error) (*model.Phase4Workflow, error) {
	errorMessage := stageErr.Error()
	failurePayload := cloneMap(payload)
	failurePayload["stage"] = stage
	failurePayload["error"] = errorMessage
	_, _ = s.data.CreateWorkflowAction(ctx, model.Phase4WorkflowAction{
		WorkflowID:    workflow.ID,
		Stage:         stage,
		ActionType:    actionType,
		ActorType:     model.Phase4WorkflowActorSystem,
		Status:        model.Phase4WorkflowActionStatusFailed,
		JobID:         strings.TrimSpace(jobID),
		RunManifestID: strings.TrimSpace(runManifestID),
		Payload:       failurePayload,
		ErrorMessage:  errorMessage,
	})
	workflow.Metadata = cloneMap(workflow.Metadata)
	workflow.Metadata["failed_stage"] = stage
	workflow.Metadata["last_failure_feedback"] = failurePayload
	if strings.TrimSpace(runManifestID) != "" {
		workflow.Metadata["latest_failed_run_id"] = runManifestID
	}
	workflow.LastError = summarizeFailure(failurePayload)
	workflow, err := s.data.UpdateWorkflow(ctx, workflow.ID, withWorkflowState(*workflow, model.Phase4WorkflowStatusBlocked, model.Phase4WorkflowNextActionRetryStage, workflow.LastError))
	if err != nil {
		return nil, err
	}
	return workflow, nil
}

func (s *Service) getJob(ctx context.Context, jobID string) (*model.AgentJob, error) {
	if s.jobs == nil || strings.TrimSpace(jobID) == "" {
		return nil, nil
	}
	return s.jobs.GetByID(ctx, strings.TrimSpace(jobID))
}

func (s *Service) safePublishEvent(ctx context.Context, req model.AgentEventCreateRequest) {
	if s.events == nil || len(req.InputRefs) == 0 {
		return
	}
	_, _ = s.events.PublishEvent(ctx, req)
}

func withWorkflowState(item model.Phase4Workflow, status string, nextAction string, lastError string) model.Phase4Workflow {
	item.Status = model.NormalizePhase4WorkflowStatus(status)
	item.NextAction = model.NormalizePhase4WorkflowNextAction(nextAction)
	item.LastError = strings.TrimSpace(lastError)
	item.ManualInputs = cloneMap(item.ManualInputs)
	item.Metadata = cloneMap(item.Metadata)
	return item
}

func normalizeReaderConfig(cfg model.Phase4WorkflowReaderConfig) model.Phase4WorkflowReaderConfig {
	cfg.UserNotes = strings.TrimSpace(cfg.UserNotes)
	cfg.SearchMode = strings.TrimSpace(strings.ToLower(cfg.SearchMode))
	cfg.ExecutionMode = strings.TrimSpace(strings.ToLower(cfg.ExecutionMode))
	cfg.ModelProvider = strings.TrimSpace(cfg.ModelProvider)
	cfg.ModelName = strings.TrimSpace(cfg.ModelName)
	cfg.PromptVersion = strings.TrimSpace(cfg.PromptVersion)
	switch cfg.SearchMode {
	case "", "auto":
		cfg.SearchMode = "auto"
	case "fixture", "live":
	default:
		cfg.SearchMode = "auto"
	}
	switch cfg.ExecutionMode {
	case "", "api":
		cfg.ExecutionMode = "api"
	case "mock", "codex_cli":
	default:
		cfg.ExecutionMode = "api"
	}
	if cfg.ModelProvider == "" {
		cfg.ModelProvider = "phase4_reader"
	}
	if cfg.ModelName == "" {
		cfg.ModelName = "reader-phase4-default"
	}
	if cfg.PromptVersion == "" {
		cfg.PromptVersion = "v1"
	}
	if cfg.MaxPapers <= 0 {
		cfg.MaxPapers = 10
	}
	if cfg.MaxPapers > 20 {
		cfg.MaxPapers = 20
	}
	cfg.SkillRefs = orderedUniqueStrings(cfg.SkillRefs)
	cfg.ToolRefs = orderedUniqueStrings(cfg.ToolRefs)
	cfg.MemoryRefs = orderedUniqueStrings(cfg.MemoryRefs)
	cfg.ManualPapers = normalizeManualPapers(cfg.ManualPapers)
	return cfg
}

func normalizeIdeaConfig(cfg model.Phase4WorkflowIdeaConfig) model.Phase4WorkflowIdeaConfig {
	cfg.UserNotes = strings.TrimSpace(cfg.UserNotes)
	cfg.ExecutionMode = strings.TrimSpace(strings.ToLower(cfg.ExecutionMode))
	cfg.ModelProvider = strings.TrimSpace(cfg.ModelProvider)
	cfg.ModelName = strings.TrimSpace(cfg.ModelName)
	cfg.PromptVersion = strings.TrimSpace(cfg.PromptVersion)
	if cfg.TargetCount <= 0 {
		cfg.TargetCount = 10
	}
	if cfg.TargetCount > 10 {
		cfg.TargetCount = 10
	}
	switch cfg.ExecutionMode {
	case "", "api":
		cfg.ExecutionMode = "api"
	case "mock", "codex_cli":
	default:
		cfg.ExecutionMode = "api"
	}
	if cfg.ModelProvider == "" {
		cfg.ModelProvider = "phase4_idea"
	}
	if cfg.ModelName == "" {
		cfg.ModelName = "idea-phase4-default"
	}
	if cfg.PromptVersion == "" {
		cfg.PromptVersion = "v1"
	}
	cfg.SkillRefs = orderedUniqueStrings(cfg.SkillRefs)
	cfg.ToolRefs = orderedUniqueStrings(cfg.ToolRefs)
	cfg.MemoryRefs = orderedUniqueStrings(cfg.MemoryRefs)
	cfg.ManualIdea = normalizeIdeaSeed(cfg.ManualIdea)
	return cfg
}

func normalizeCodingConfig(cfg model.Phase4WorkflowCodingConfig, datasetServerID string) model.Phase4WorkflowCodingConfig {
	cfg.RunnerMode = strings.TrimSpace(strings.ToLower(cfg.RunnerMode))
	cfg.ServerID = strings.TrimSpace(cfg.ServerID)
	cfg.GPU = strings.TrimSpace(cfg.GPU)
	cfg.UserNotes = strings.TrimSpace(cfg.UserNotes)
	cfg.ExecutionMode = strings.TrimSpace(strings.ToLower(cfg.ExecutionMode))
	cfg.ModelProvider = strings.TrimSpace(cfg.ModelProvider)
	cfg.ModelName = strings.TrimSpace(cfg.ModelName)
	cfg.PromptVersion = strings.TrimSpace(cfg.PromptVersion)
	if cfg.ServerID == "" {
		cfg.ServerID = strings.TrimSpace(datasetServerID)
	}
	switch cfg.RunnerMode {
	case "":
		if cfg.ServerID != "" {
			cfg.RunnerMode = phase4WorkflowRemoteServerName
		} else {
			cfg.RunnerMode = "local_dummy"
		}
	case "local_dummy", phase4WorkflowRemoteServerName:
	default:
		if cfg.ServerID != "" {
			cfg.RunnerMode = phase4WorkflowRemoteServerName
		} else {
			cfg.RunnerMode = "local_dummy"
		}
	}
	if cfg.RunnerMode == phase4WorkflowRemoteServerName && cfg.ServerID == "" {
		cfg.RunnerMode = "local_dummy"
	}
	if cfg.MaxRetryCount <= 0 {
		cfg.MaxRetryCount = 3
	}
	switch cfg.ExecutionMode {
	case "", "api":
		cfg.ExecutionMode = "api"
	case "mock", "codex_cli":
	default:
		cfg.ExecutionMode = "api"
	}
	if cfg.ModelProvider == "" {
		cfg.ModelProvider = "phase4_coding"
	}
	if cfg.ModelName == "" {
		cfg.ModelName = "coding-phase4-default"
	}
	if cfg.PromptVersion == "" {
		cfg.PromptVersion = "v1"
	}
	cfg.SkillRefs = orderedUniqueStrings(cfg.SkillRefs)
	cfg.ToolRefs = orderedUniqueStrings(cfg.ToolRefs)
	cfg.MemoryRefs = orderedUniqueStrings(cfg.MemoryRefs)
	return cfg
}

func normalizeWritingConfig(cfg model.Phase4WorkflowWritingConfig) model.Phase4WorkflowWritingConfig {
	cfg.UserNotes = strings.TrimSpace(cfg.UserNotes)
	cfg.ExecutionMode = strings.TrimSpace(strings.ToLower(cfg.ExecutionMode))
	cfg.ModelProvider = strings.TrimSpace(cfg.ModelProvider)
	cfg.ModelName = strings.TrimSpace(cfg.ModelName)
	cfg.PromptVersion = strings.TrimSpace(cfg.PromptVersion)
	switch cfg.ExecutionMode {
	case "", "api":
		cfg.ExecutionMode = "api"
	case "mock", "codex_cli":
	default:
		cfg.ExecutionMode = "api"
	}
	if cfg.ModelProvider == "" {
		cfg.ModelProvider = "phase4_writer"
	}
	if cfg.ModelName == "" {
		cfg.ModelName = "writer-phase4-default"
	}
	if cfg.PromptVersion == "" {
		cfg.PromptVersion = "v1"
	}
	cfg.SkillRefs = orderedUniqueStrings(cfg.SkillRefs)
	cfg.ToolRefs = orderedUniqueStrings(cfg.ToolRefs)
	cfg.MemoryRefs = orderedUniqueStrings(cfg.MemoryRefs)
	return cfg
}

func decodeReaderConfig(raw any) model.Phase4WorkflowReaderConfig {
	var cfg model.Phase4WorkflowReaderConfig
	decodeWorkflowConfig(raw, &cfg)
	return normalizeReaderConfig(cfg)
}

func decodeIdeaConfig(raw any) model.Phase4WorkflowIdeaConfig {
	var cfg model.Phase4WorkflowIdeaConfig
	decodeWorkflowConfig(raw, &cfg)
	return normalizeIdeaConfig(cfg)
}

func decodeCodingConfig(raw any) model.Phase4WorkflowCodingConfig {
	var cfg model.Phase4WorkflowCodingConfig
	decodeWorkflowConfig(raw, &cfg)
	return cfg
}

func decodeWritingConfig(raw any) model.Phase4WorkflowWritingConfig {
	var cfg model.Phase4WorkflowWritingConfig
	decodeWorkflowConfig(raw, &cfg)
	return normalizeWritingConfig(cfg)
}

func decodeWorkflowConfig(raw any, target any) {
	if raw == nil || target == nil {
		return
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return
	}
	_ = json.Unmarshal(payload, target)
}

func mergeCodingConfig(base model.Phase4WorkflowCodingConfig, override *model.Phase4WorkflowCodingConfig, datasetServerID string) model.Phase4WorkflowCodingConfig {
	base = normalizeCodingConfig(base, datasetServerID)
	if override == nil {
		return base
	}
	merged := base
	if strings.TrimSpace(override.RunnerMode) != "" {
		merged.RunnerMode = override.RunnerMode
	}
	if strings.TrimSpace(override.ServerID) != "" {
		merged.ServerID = override.ServerID
	}
	if strings.TrimSpace(override.GPU) != "" {
		merged.GPU = override.GPU
	}
	if override.MaxRetryCount > 0 {
		merged.MaxRetryCount = override.MaxRetryCount
	}
	if strings.TrimSpace(override.UserNotes) != "" {
		merged.UserNotes = mergeNotes(merged.UserNotes, override.UserNotes)
	}
	if strings.TrimSpace(override.ExecutionMode) != "" {
		merged.ExecutionMode = override.ExecutionMode
	}
	if strings.TrimSpace(override.ModelProvider) != "" {
		merged.ModelProvider = override.ModelProvider
	}
	if strings.TrimSpace(override.ModelName) != "" {
		merged.ModelName = override.ModelName
	}
	if strings.TrimSpace(override.PromptVersion) != "" {
		merged.PromptVersion = override.PromptVersion
	}
	if len(override.SkillRefs) > 0 {
		merged.SkillRefs = orderedUniqueStrings(override.SkillRefs)
	}
	if len(override.ToolRefs) > 0 {
		merged.ToolRefs = orderedUniqueStrings(override.ToolRefs)
	}
	if len(override.MemoryRefs) > 0 {
		merged.MemoryRefs = orderedUniqueStrings(override.MemoryRefs)
	}
	return normalizeCodingConfig(merged, datasetServerID)
}

func mergeWritingConfig(base model.Phase4WorkflowWritingConfig, override *model.Phase4WorkflowWritingConfig) model.Phase4WorkflowWritingConfig {
	base = normalizeWritingConfig(base)
	if override == nil {
		return base
	}
	merged := base
	if strings.TrimSpace(override.UserNotes) != "" {
		merged.UserNotes = mergeNotes(merged.UserNotes, override.UserNotes)
	}
	if strings.TrimSpace(override.ExecutionMode) != "" {
		merged.ExecutionMode = override.ExecutionMode
	}
	if strings.TrimSpace(override.ModelProvider) != "" {
		merged.ModelProvider = override.ModelProvider
	}
	if strings.TrimSpace(override.ModelName) != "" {
		merged.ModelName = override.ModelName
	}
	if strings.TrimSpace(override.PromptVersion) != "" {
		merged.PromptVersion = override.PromptVersion
	}
	if len(override.SkillRefs) > 0 {
		merged.SkillRefs = orderedUniqueStrings(override.SkillRefs)
	}
	if len(override.ToolRefs) > 0 {
		merged.ToolRefs = orderedUniqueStrings(override.ToolRefs)
	}
	if len(override.MemoryRefs) > 0 {
		merged.MemoryRefs = orderedUniqueStrings(override.MemoryRefs)
	}
	return normalizeWritingConfig(merged)
}

func workflowNextActions(status string) []model.Phase4WorkflowNextAction {
	switch model.NormalizePhase4WorkflowStatus(status) {
	case model.Phase4WorkflowStatusAwaitingSelection:
		return []model.Phase4WorkflowNextAction{
			{Action: model.Phase4WorkflowNextActionSelectIdea, Label: "Select Idea", Description: "Choose one idea and start coding."},
			{Action: model.Phase4WorkflowNextActionArchive, Label: "Archive", Description: "Archive this workflow without running coding."},
		}
	case model.Phase4WorkflowStatusAwaitingRevisionSelect:
		return []model.Phase4WorkflowNextAction{
			{Action: model.Phase4WorkflowNextActionSelectRevision, Label: "Select Revision", Description: "Choose a revised idea candidate and rerun coding."},
			{Action: model.Phase4WorkflowNextActionArchive, Label: "Archive", Description: "Archive this workflow and keep the failure trail."},
		}
	case model.Phase4WorkflowStatusBlocked:
		return []model.Phase4WorkflowNextAction{
			{Action: model.Phase4WorkflowNextActionRetryStage, Label: "Retry Stage", Description: "Retry the most recent blocked stage with the current selection."},
			{Action: model.Phase4WorkflowNextActionArchive, Label: "Archive", Description: "Archive this blocked workflow."},
		}
	case model.Phase4WorkflowStatusCompleted:
		return []model.Phase4WorkflowNextAction{
			{Action: model.Phase4WorkflowNextActionViewReport, Label: "View Report", Description: "Open the finalized phase4 report."},
			{Action: model.Phase4WorkflowNextActionArchive, Label: "Archive", Description: "Archive this completed workflow."},
		}
	default:
		return []model.Phase4WorkflowNextAction{}
	}
}

func ideaToScoreView(item model.Phase4Idea) model.Phase4IdeaScoreView {
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
		OverallScore:         floatValue(item.ScoreSummary["overallScore"]),
		Rank:                 intValue(item.ScoreSummary["rank"]),
		RecommendationTier:   stringValue(item.ScoreSummary["recommendationTier"]),
		RecommendationReason: stringValue(item.ScoreSummary["recommendationReason"]),
		ExpectedGains:        append([]string{}, item.ExpectedGains...),
		RiskPoints:           append([]string{}, item.RiskPoints...),
	}
}

func configMap(value any) map[string]any {
	raw, err := json.Marshal(value)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if err = json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func extractIdeaIDs(items []model.Phase4Idea) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.ID) != "" {
			out = append(out, item.ID)
		}
	}
	return orderedUniqueStrings(out)
}

func extractTopIdeaIDs(items []model.Phase4IdeaScoreView) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.ID) != "" {
			out = append(out, item.ID)
		}
	}
	return orderedUniqueStrings(out)
}

func extractReaderSourceIDs(items []model.Phase4ReaderSource) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.ID) != "" {
			out = append(out, item.ID)
		}
	}
	return orderedUniqueStrings(out)
}

func normalizeManualPapers(items []model.Phase4ReaderManualPaperInput) []model.Phase4ReaderManualPaperInput {
	out := make([]model.Phase4ReaderManualPaperInput, 0, len(items))
	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		filePath := strings.TrimSpace(item.FilePath)
		if title == "" && filePath == "" {
			continue
		}
		out = append(out, model.Phase4ReaderManualPaperInput{
			Title:         title,
			Abstract:      strings.TrimSpace(item.Abstract),
			SourceType:    strings.TrimSpace(strings.ToLower(item.SourceType)),
			SourceURL:     strings.TrimSpace(item.SourceURL),
			OpenAccessURL: strings.TrimSpace(item.OpenAccessURL),
			Venue:         strings.TrimSpace(item.Venue),
			Year:          item.Year,
			Authors:       orderedUniqueStrings(item.Authors),
			FilePath:      filePath,
			Note:          strings.TrimSpace(item.Note),
		})
	}
	return out
}

func normalizeIdeaSeed(seed *model.Phase4IdeaSeedInput) *model.Phase4IdeaSeedInput {
	if seed == nil {
		return nil
	}
	copySeed := *seed
	copySeed.Title = strings.TrimSpace(copySeed.Title)
	copySeed.ProblemDefinition = strings.TrimSpace(copySeed.ProblemDefinition)
	copySeed.CoreMethod = strings.TrimSpace(copySeed.CoreMethod)
	copySeed.Differentiators = strings.TrimSpace(copySeed.Differentiators)
	copySeed.DataProcessingNeeds = orderedUniqueStrings(copySeed.DataProcessingNeeds)
	copySeed.ModelChanges = orderedUniqueStrings(copySeed.ModelChanges)
	copySeed.TrainingPlan = strings.TrimSpace(copySeed.TrainingPlan)
	copySeed.EvaluationMetrics = orderedUniqueStrings(copySeed.EvaluationMetrics)
	copySeed.RiskPoints = orderedUniqueStrings(copySeed.RiskPoints)
	copySeed.ExpectedGains = orderedUniqueStrings(copySeed.ExpectedGains)
	copySeed.SourceType = strings.TrimSpace(copySeed.SourceType)
	copySeed.RevisionOfID = strings.TrimSpace(copySeed.RevisionOfID)
	if copySeed.Title == "" && copySeed.CoreMethod == "" && copySeed.ProblemDefinition == "" {
		return nil
	}
	return &copySeed
}

func orderedUniqueStrings(groups ...[]string) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, items := range groups {
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
	}
	return out
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return orderedUniqueStrings(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := stringValue(item); text != "" {
				out = append(out, text)
			}
		}
		return orderedUniqueStrings(out)
	default:
		if text := stringValue(value); text != "" {
			return []string{text}
		}
		return []string{}
	}
}

func cloneMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func containsStringFold(items []string, needle string) bool {
	needle = strings.TrimSpace(strings.ToLower(needle))
	for _, item := range items {
		if strings.TrimSpace(strings.ToLower(item)) == needle {
			return true
		}
	}
	return false
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

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func floatValue(value any) float64 {
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
	default:
		return 0
	}
}

func mergeNotes(existing string, incoming string) string {
	existing = strings.TrimSpace(existing)
	incoming = strings.TrimSpace(incoming)
	switch {
	case existing == "":
		return incoming
	case incoming == "":
		return existing
	case strings.EqualFold(existing, incoming):
		return existing
	default:
		return existing + "\n" + incoming
	}
}

func summarizeFailure(feedback map[string]any) string {
	stage := strings.TrimSpace(stringValue(feedback["stage"]))
	errText := strings.TrimSpace(stringValue(feedback["error"]))
	switch {
	case stage != "" && errText != "":
		return stage + ": " + errText
	case errText != "":
		return errText
	case stage != "":
		return stage
	default:
		return "workflow blocked"
	}
}
