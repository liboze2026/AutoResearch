package model

import (
	"fmt"
	"strings"
	"time"
)

const (
	Phase4WorkflowStatusRunningReader          = "running_reader"
	Phase4WorkflowStatusRunningIdea            = "running_idea"
	Phase4WorkflowStatusAwaitingSelection      = "awaiting_selection"
	Phase4WorkflowStatusRunningCoding          = "running_coding"
	Phase4WorkflowStatusAwaitingRevisionSelect = "awaiting_revision_selection"
	Phase4WorkflowStatusRunningWriting         = "running_writing"
	Phase4WorkflowStatusCompleted              = "completed"
	Phase4WorkflowStatusBlocked                = "blocked"
	Phase4WorkflowStatusArchived               = "archived"

	Phase4WorkflowStageWorkflow = "workflow"
	Phase4WorkflowStageReader   = "reader"
	Phase4WorkflowStageIdea     = "idea"
	Phase4WorkflowStageCoding   = "coding"
	Phase4WorkflowStageWriting  = "writing"

	Phase4WorkflowActorSystem = "system"
	Phase4WorkflowActorUser   = "user"

	Phase4WorkflowActionStatusStarted   = "started"
	Phase4WorkflowActionStatusSucceeded = "succeeded"
	Phase4WorkflowActionStatusFailed    = "failed"

	Phase4WorkflowNextActionNone           = "none"
	Phase4WorkflowNextActionSelectIdea     = "select_idea"
	Phase4WorkflowNextActionSelectRevision = "select_revision"
	Phase4WorkflowNextActionRetryStage     = "retry_stage"
	Phase4WorkflowNextActionArchive        = "archive"
	Phase4WorkflowNextActionViewReport     = "view_report"
)

type Phase4WorkflowReaderConfig struct {
	ManualPapers  []Phase4ReaderManualPaperInput `json:"manualPapers"`
	UserNotes     string                         `json:"userNotes"`
	SearchMode    string                         `json:"searchMode"`
	MaxPapers     int                            `json:"maxPapers"`
	ExecutionMode string                         `json:"executionMode"`
	ModelProvider string                         `json:"modelProvider"`
	ModelName     string                         `json:"modelName"`
	PromptVersion string                         `json:"promptVersion"`
	SkillRefs     []string                       `json:"skillRefs"`
	ToolRefs      []string                       `json:"toolRefs"`
	MemoryRefs    []string                       `json:"memoryRefs"`
}

type Phase4WorkflowIdeaConfig struct {
	UserNotes     string               `json:"userNotes"`
	ManualIdea    *Phase4IdeaSeedInput `json:"manualIdea,omitempty"`
	TargetCount   int                  `json:"targetCount"`
	ExecutionMode string               `json:"executionMode"`
	ModelProvider string               `json:"modelProvider"`
	ModelName     string               `json:"modelName"`
	PromptVersion string               `json:"promptVersion"`
	SkillRefs     []string             `json:"skillRefs"`
	ToolRefs      []string             `json:"toolRefs"`
	MemoryRefs    []string             `json:"memoryRefs"`
}

type Phase4WorkflowCodingConfig struct {
	RunnerMode    string   `json:"runnerMode"`
	ServerID      string   `json:"serverId,omitempty"`
	GPU           string   `json:"gpu,omitempty"`
	MaxRetryCount int      `json:"maxRetryCount"`
	UserNotes     string   `json:"userNotes"`
	ExecutionMode string   `json:"executionMode"`
	ModelProvider string   `json:"modelProvider"`
	ModelName     string   `json:"modelName"`
	PromptVersion string   `json:"promptVersion"`
	SkillRefs     []string `json:"skillRefs"`
	ToolRefs      []string `json:"toolRefs"`
	MemoryRefs    []string `json:"memoryRefs"`
}

type Phase4WorkflowWritingConfig struct {
	UserNotes     string   `json:"userNotes"`
	ExecutionMode string   `json:"executionMode"`
	ModelProvider string   `json:"modelProvider"`
	ModelName     string   `json:"modelName"`
	PromptVersion string   `json:"promptVersion"`
	SkillRefs     []string `json:"skillRefs"`
	ToolRefs      []string `json:"toolRefs"`
	MemoryRefs    []string `json:"memoryRefs"`
}

type Phase4Workflow struct {
	ID                   string         `json:"id"`
	DatasetProfileID     string         `json:"datasetProfileId"`
	ReaderContextID      string         `json:"readerContextId,omitempty"`
	SelectedIdeaID       string         `json:"selectedIdeaId,omitempty"`
	CurrentRunManifestID string         `json:"currentRunManifestId,omitempty"`
	LatestReportID       string         `json:"latestReportId,omitempty"`
	LatestReaderJobID    string         `json:"latestReaderJobId,omitempty"`
	LatestIdeaJobID      string         `json:"latestIdeaJobId,omitempty"`
	LatestCodingJobID    string         `json:"latestCodingJobId,omitempty"`
	LatestWriterJobID    string         `json:"latestWriterJobId,omitempty"`
	Status               string         `json:"status"`
	NextAction           string         `json:"nextAction"`
	LastError            string         `json:"lastError"`
	ManualInputs         map[string]any `json:"manualInputs"`
	Metadata             map[string]any `json:"metadata"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

type Phase4WorkflowAction struct {
	ID            string         `json:"id"`
	WorkflowID    string         `json:"workflowId"`
	Stage         string         `json:"stage"`
	ActionType    string         `json:"actionType"`
	ActorType     string         `json:"actorType"`
	Status        string         `json:"status"`
	JobID         string         `json:"jobId,omitempty"`
	RunManifestID string         `json:"runManifestId,omitempty"`
	ReportID      string         `json:"reportId,omitempty"`
	Payload       map[string]any `json:"payload"`
	ErrorMessage  string         `json:"errorMessage"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

type Phase4WorkflowCreateRequest struct {
	DatasetProfileID string                      `json:"datasetProfileId" binding:"required"`
	Reader           Phase4WorkflowReaderConfig  `json:"reader"`
	Idea             Phase4WorkflowIdeaConfig    `json:"idea"`
	Coding           Phase4WorkflowCodingConfig  `json:"coding"`
	Writing          Phase4WorkflowWritingConfig `json:"writing"`
	Metadata         map[string]any              `json:"metadata"`
}

type Phase4WorkflowSelectIdeaRequest struct {
	IdeaID    string                       `json:"ideaId" binding:"required"`
	Coding    *Phase4WorkflowCodingConfig  `json:"coding,omitempty"`
	Writing   *Phase4WorkflowWritingConfig `json:"writing,omitempty"`
	UserNotes string                       `json:"userNotes"`
}

type Phase4WorkflowRetryStageRequest struct {
	Coding    *Phase4WorkflowCodingConfig  `json:"coding,omitempty"`
	Writing   *Phase4WorkflowWritingConfig `json:"writing,omitempty"`
	UserNotes string                       `json:"userNotes"`
}

type Phase4WorkflowNextAction struct {
	Action      string `json:"action"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type Phase4WorkflowLatestJobs struct {
	Reader *AgentJob `json:"reader,omitempty"`
	Idea   *AgentJob `json:"idea,omitempty"`
	Coding *AgentJob `json:"coding,omitempty"`
	Writer *AgentJob `json:"writer,omitempty"`
}

type Phase4WorkflowDetail struct {
	Workflow           *Phase4Workflow               `json:"workflow"`
	DatasetProfile     *Phase4DatasetProfile         `json:"datasetProfile,omitempty"`
	ReaderContext      *Phase4ReaderContext          `json:"readerContext,omitempty"`
	Ideas              []Phase4Idea                  `json:"ideas"`
	TopRecommendations []Phase4IdeaScoreView         `json:"topRecommendations"`
	SelectedIdea       *Phase4Idea                   `json:"selectedIdea,omitempty"`
	CurrentRunManifest *Phase4RunManifest            `json:"currentRunManifest,omitempty"`
	LatestReport       *Phase4StructuredReportRecord `json:"latestReport,omitempty"`
	LatestJobs         Phase4WorkflowLatestJobs      `json:"latestJobs"`
	NextActions        []Phase4WorkflowNextAction    `json:"nextActions"`
	Timeline           []Phase4WorkflowAction        `json:"timeline"`
}

func NormalizePhase4WorkflowStatus(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", Phase4WorkflowStatusRunningReader:
		return Phase4WorkflowStatusRunningReader
	case Phase4WorkflowStatusRunningIdea,
		Phase4WorkflowStatusAwaitingSelection,
		Phase4WorkflowStatusRunningCoding,
		Phase4WorkflowStatusAwaitingRevisionSelect,
		Phase4WorkflowStatusRunningWriting,
		Phase4WorkflowStatusCompleted,
		Phase4WorkflowStatusBlocked,
		Phase4WorkflowStatusArchived:
		return value
	default:
		return Phase4WorkflowStatusRunningReader
	}
}

func NormalizePhase4WorkflowNextAction(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", Phase4WorkflowNextActionNone:
		return Phase4WorkflowNextActionNone
	case Phase4WorkflowNextActionSelectIdea,
		Phase4WorkflowNextActionSelectRevision,
		Phase4WorkflowNextActionRetryStage,
		Phase4WorkflowNextActionArchive,
		Phase4WorkflowNextActionViewReport:
		return value
	default:
		return Phase4WorkflowNextActionNone
	}
}

func ValidatePhase4WorkflowTransition(current string, next string) error {
	current = NormalizePhase4WorkflowStatus(current)
	next = NormalizePhase4WorkflowStatus(next)
	if current == next {
		return nil
	}
	allowed := map[string]map[string]struct{}{
		Phase4WorkflowStatusRunningReader: {
			Phase4WorkflowStatusRunningIdea: {},
			Phase4WorkflowStatusBlocked:     {},
			Phase4WorkflowStatusArchived:    {},
		},
		Phase4WorkflowStatusRunningIdea: {
			Phase4WorkflowStatusAwaitingSelection: {},
			Phase4WorkflowStatusBlocked:           {},
			Phase4WorkflowStatusArchived:          {},
		},
		Phase4WorkflowStatusAwaitingSelection: {
			Phase4WorkflowStatusRunningCoding: {},
			Phase4WorkflowStatusArchived:      {},
		},
		Phase4WorkflowStatusRunningCoding: {
			Phase4WorkflowStatusRunningWriting:         {},
			Phase4WorkflowStatusAwaitingRevisionSelect: {},
			Phase4WorkflowStatusBlocked:                {},
			Phase4WorkflowStatusArchived:               {},
		},
		Phase4WorkflowStatusAwaitingRevisionSelect: {
			Phase4WorkflowStatusRunningCoding: {},
			Phase4WorkflowStatusArchived:      {},
		},
		Phase4WorkflowStatusRunningWriting: {
			Phase4WorkflowStatusCompleted: {},
			Phase4WorkflowStatusBlocked:   {},
			Phase4WorkflowStatusArchived:  {},
		},
		Phase4WorkflowStatusCompleted: {
			Phase4WorkflowStatusArchived: {},
		},
		Phase4WorkflowStatusBlocked: {
			Phase4WorkflowStatusRunningReader:  {},
			Phase4WorkflowStatusRunningIdea:    {},
			Phase4WorkflowStatusRunningCoding:  {},
			Phase4WorkflowStatusRunningWriting: {},
			Phase4WorkflowStatusArchived:       {},
		},
		Phase4WorkflowStatusArchived: {},
	}
	nextSet, ok := allowed[current]
	if !ok {
		return fmt.Errorf("unsupported current workflow status: %s", current)
	}
	if _, ok = nextSet[next]; !ok {
		return fmt.Errorf("invalid workflow status transition: %s -> %s", current, next)
	}
	return nil
}
