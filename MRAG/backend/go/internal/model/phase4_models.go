package model

import (
	"fmt"
	"strings"
	"time"
)

const (
	Phase4DatasetProfileSourceRegisteredPath = "registered_path"

	Phase4DatasetProfileStatusActive   = "active"
	Phase4DatasetProfileStatusArchived = "archived"

	Phase4ReaderContextStatusDraft    = "draft"
	Phase4ReaderContextStatusReady    = "ready"
	Phase4ReaderContextStatusArchived = "archived"

	Phase4IdeaStatusDraft       = "draft"
	Phase4IdeaStatusScored      = "scored"
	Phase4IdeaStatusRejected    = "rejected"
	Phase4IdeaStatusSelected    = "selected"
	Phase4IdeaStatusImplemented = "implemented"
	Phase4IdeaStatusFailed      = "failed"
	Phase4IdeaStatusArchived    = "archived"

	Phase4RunStatusDraft      = "draft"
	Phase4RunStatusQueued     = "queued"
	Phase4RunStatusScheduled  = "scheduled"
	Phase4RunStatusRunning    = "running"
	Phase4RunStatusSucceeded  = "succeeded"
	Phase4RunStatusFailed     = "failed"
	Phase4RunStatusTestFailed = "test_failed"
	Phase4RunStatusCanceled   = "canceled"
	Phase4RunStatusArchived   = "archived"

	Phase4ReportStatusDraft     = "draft"
	Phase4ReportStatusFinalized = "finalized"
	Phase4ReportStatusArchived  = "archived"
)

type Phase4DatasetSplit struct {
	Name        string `json:"name"`
	Path        string `json:"path,omitempty"`
	SampleCount int64  `json:"sampleCount,omitempty"`
	Note        string `json:"note,omitempty"`
}

type Phase4DatasetProfile struct {
	ID                    string               `json:"id"`
	DatasetName           string               `json:"datasetName"`
	TaskType              string               `json:"taskType"`
	ModalityComposition   []string             `json:"modalityComposition"`
	Splits                []Phase4DatasetSplit `json:"splits"`
	LabelSchema           map[string]any       `json:"labelSchema"`
	FileStructureSnapshot map[string]any       `json:"fileStructureSnapshot"`
	SampleStatistics      map[string]any       `json:"sampleStatistics"`
	OfficialMetric        string               `json:"officialMetric"`
	OfficialBaseline      string               `json:"officialBaseline"`
	License               string               `json:"license"`
	Citation              string               `json:"citation"`
	KnownDifficulties     []string             `json:"knownDifficulties"`
	UserNotes             string               `json:"userNotes"`
	Metadata              map[string]any       `json:"metadata"`
	SourceMode            string               `json:"sourceMode"`
	ServerID              string               `json:"serverId,omitempty"`
	ServerName            string               `json:"serverName,omitempty"`
	ServerPath            string               `json:"serverPath"`
	Status                string               `json:"status"`
	CreatedAt             time.Time            `json:"createdAt"`
	UpdatedAt             time.Time            `json:"updatedAt"`
}

type Phase4DatasetProfileCreateRequest struct {
	DatasetName           string               `json:"datasetName" binding:"required"`
	TaskType              string               `json:"taskType" binding:"required"`
	ModalityComposition   []string             `json:"modalityComposition"`
	Splits                []Phase4DatasetSplit `json:"splits"`
	LabelSchema           map[string]any       `json:"labelSchema"`
	FileStructureSnapshot map[string]any       `json:"fileStructureSnapshot"`
	SampleStatistics      map[string]any       `json:"sampleStatistics"`
	OfficialMetric        string               `json:"officialMetric"`
	OfficialBaseline      string               `json:"officialBaseline"`
	License               string               `json:"license"`
	Citation              string               `json:"citation"`
	KnownDifficulties     []string             `json:"knownDifficulties"`
	UserNotes             string               `json:"userNotes"`
	Metadata              map[string]any       `json:"metadata"`
	SourceMode            string               `json:"sourceMode"`
	ServerID              string               `json:"serverId"`
	ServerPath            string               `json:"serverPath"`
	Status                string               `json:"status"`
}

type Phase4DatasetProfileUpdateRequest struct {
	DatasetName           *string               `json:"datasetName,omitempty"`
	TaskType              *string               `json:"taskType,omitempty"`
	ModalityComposition   *[]string             `json:"modalityComposition,omitempty"`
	Splits                *[]Phase4DatasetSplit `json:"splits,omitempty"`
	LabelSchema           *map[string]any       `json:"labelSchema,omitempty"`
	FileStructureSnapshot *map[string]any       `json:"fileStructureSnapshot,omitempty"`
	SampleStatistics      *map[string]any       `json:"sampleStatistics,omitempty"`
	OfficialMetric        *string               `json:"officialMetric,omitempty"`
	OfficialBaseline      *string               `json:"officialBaseline,omitempty"`
	License               *string               `json:"license,omitempty"`
	Citation              *string               `json:"citation,omitempty"`
	KnownDifficulties     *[]string             `json:"knownDifficulties,omitempty"`
	UserNotes             *string               `json:"userNotes,omitempty"`
	Metadata              *map[string]any       `json:"metadata,omitempty"`
	SourceMode            *string               `json:"sourceMode,omitempty"`
	ServerID              *string               `json:"serverId,omitempty"`
	ServerPath            *string               `json:"serverPath,omitempty"`
	Status                *string               `json:"status,omitempty"`
}

type Phase4ReaderSource struct {
	ID               string         `json:"id"`
	DatasetProfileID string         `json:"datasetProfileId,omitempty"`
	Title            string         `json:"title"`
	Authors          []string       `json:"authors"`
	Venue            string         `json:"venue"`
	PublicationYear  int            `json:"publicationYear,omitempty"`
	SourceType       string         `json:"sourceType"`
	SourceURL        string         `json:"sourceUrl"`
	OpenAccessURL    string         `json:"openAccessUrl,omitempty"`
	QualityTier      string         `json:"qualityTier"`
	RankingScore     float64        `json:"rankingScore"`
	QualityScore     float64        `json:"qualityScore"`
	RelevanceScore   float64        `json:"relevanceScore"`
	CitationCount    int            `json:"citationCount"`
	Metadata         map[string]any `json:"metadata"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type Phase4ReaderSourceCreateRequest struct {
	DatasetProfileID string         `json:"datasetProfileId,omitempty"`
	Title            string         `json:"title" binding:"required"`
	Authors          []string       `json:"authors"`
	Venue            string         `json:"venue"`
	PublicationYear  int            `json:"publicationYear,omitempty"`
	SourceType       string         `json:"sourceType" binding:"required"`
	SourceURL        string         `json:"sourceUrl"`
	OpenAccessURL    string         `json:"openAccessUrl,omitempty"`
	QualityTier      string         `json:"qualityTier"`
	RankingScore     float64        `json:"rankingScore"`
	QualityScore     float64        `json:"qualityScore"`
	RelevanceScore   float64        `json:"relevanceScore"`
	CitationCount    int            `json:"citationCount"`
	Metadata         map[string]any `json:"metadata"`
}

type Phase4ReaderSourceUpdateRequest struct {
	Title           *string         `json:"title,omitempty"`
	Authors         *[]string       `json:"authors,omitempty"`
	Venue           *string         `json:"venue,omitempty"`
	PublicationYear *int            `json:"publicationYear,omitempty"`
	SourceType      *string         `json:"sourceType,omitempty"`
	SourceURL       *string         `json:"sourceUrl,omitempty"`
	OpenAccessURL   *string         `json:"openAccessUrl,omitempty"`
	QualityTier     *string         `json:"qualityTier,omitempty"`
	RankingScore    *float64        `json:"rankingScore,omitempty"`
	QualityScore    *float64        `json:"qualityScore,omitempty"`
	RelevanceScore  *float64        `json:"relevanceScore,omitempty"`
	CitationCount   *int            `json:"citationCount,omitempty"`
	Metadata        *map[string]any `json:"metadata,omitempty"`
}

type Phase4ReaderContext struct {
	ID                string         `json:"id"`
	DatasetProfileID  string         `json:"datasetProfileId,omitempty"`
	Title             string         `json:"title"`
	Summary           string         `json:"summary"`
	TaskDefinition    string         `json:"taskDefinition"`
	RelatedWork       []string       `json:"relatedWork"`
	RetrievalFocus    []string       `json:"retrievalFocus"`
	RankingNotes      string         `json:"rankingNotes"`
	SourceIDs         []string       `json:"sourceIds"`
	StructuredContext map[string]any `json:"structuredContext"`
	Status            string         `json:"status"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

type Phase4ReaderContextCreateRequest struct {
	DatasetProfileID  string         `json:"datasetProfileId,omitempty"`
	Title             string         `json:"title" binding:"required"`
	Summary           string         `json:"summary"`
	TaskDefinition    string         `json:"taskDefinition"`
	RelatedWork       []string       `json:"relatedWork"`
	RetrievalFocus    []string       `json:"retrievalFocus"`
	RankingNotes      string         `json:"rankingNotes"`
	SourceIDs         []string       `json:"sourceIds"`
	StructuredContext map[string]any `json:"structuredContext"`
	Status            string         `json:"status"`
}

type Phase4ReaderContextUpdateRequest struct {
	Title             *string         `json:"title,omitempty"`
	Summary           *string         `json:"summary,omitempty"`
	TaskDefinition    *string         `json:"taskDefinition,omitempty"`
	RelatedWork       *[]string       `json:"relatedWork,omitempty"`
	RetrievalFocus    *[]string       `json:"retrievalFocus,omitempty"`
	RankingNotes      *string         `json:"rankingNotes,omitempty"`
	SourceIDs         *[]string       `json:"sourceIds,omitempty"`
	StructuredContext *map[string]any `json:"structuredContext,omitempty"`
	Status            *string         `json:"status,omitempty"`
}

type Phase4IdeaScore struct {
	Novelty         float64 `json:"novelty"`
	DatasetFit      float64 `json:"datasetFit"`
	Feasibility     float64 `json:"feasibility"`
	ExpectedGain    float64 `json:"expectedGain"`
	ComputeCost     float64 `json:"computeCost"`
	FailureRisk     float64 `json:"failureRisk"`
	Reproducibility float64 `json:"reproducibility"`
}

type Phase4Idea struct {
	ID                  string          `json:"id"`
	DatasetProfileID    string          `json:"datasetProfileId,omitempty"`
	ReaderContextID     string          `json:"readerContextId,omitempty"`
	Title               string          `json:"title"`
	ProblemDefinition   string          `json:"problemDefinition"`
	CoreMethod          string          `json:"coreMethod"`
	Differentiators     string          `json:"differentiators"`
	DataProcessingNeeds []string        `json:"dataProcessingNeeds"`
	ModelChanges        []string        `json:"modelChanges"`
	TrainingPlan        string          `json:"trainingPlan"`
	EvaluationMetrics   []string        `json:"evaluationMetrics"`
	RiskPoints          []string        `json:"riskPoints"`
	ExpectedGains       []string        `json:"expectedGains"`
	Score               Phase4IdeaScore `json:"score"`
	ScoreSummary        map[string]any  `json:"scoreSummary"`
	Status              string          `json:"status"`
	SourceType          string          `json:"sourceType"`
	RevisionOfID        string          `json:"revisionOfId,omitempty"`
	LineageRootID       string          `json:"lineageRootId,omitempty"`
	FailureFeedback     map[string]any  `json:"failureFeedback"`
	LastFailureRunID    string          `json:"lastFailureRunId,omitempty"`
	CreatedAt           time.Time       `json:"createdAt"`
	UpdatedAt           time.Time       `json:"updatedAt"`
}

type Phase4IdeaCreateRequest struct {
	DatasetProfileID    string          `json:"datasetProfileId,omitempty"`
	ReaderContextID     string          `json:"readerContextId,omitempty"`
	Title               string          `json:"title" binding:"required"`
	ProblemDefinition   string          `json:"problemDefinition" binding:"required"`
	CoreMethod          string          `json:"coreMethod" binding:"required"`
	Differentiators     string          `json:"differentiators"`
	DataProcessingNeeds []string        `json:"dataProcessingNeeds"`
	ModelChanges        []string        `json:"modelChanges"`
	TrainingPlan        string          `json:"trainingPlan"`
	EvaluationMetrics   []string        `json:"evaluationMetrics"`
	RiskPoints          []string        `json:"riskPoints"`
	ExpectedGains       []string        `json:"expectedGains"`
	Score               Phase4IdeaScore `json:"score"`
	ScoreSummary        map[string]any  `json:"scoreSummary"`
	Status              string          `json:"status"`
	SourceType          string          `json:"sourceType"`
	RevisionOfID        string          `json:"revisionOfId,omitempty"`
	FailureFeedback     map[string]any  `json:"failureFeedback"`
	LastFailureRunID    string          `json:"lastFailureRunId,omitempty"`
}

type Phase4IdeaUpdateRequest struct {
	Title               *string          `json:"title,omitempty"`
	ProblemDefinition   *string          `json:"problemDefinition,omitempty"`
	CoreMethod          *string          `json:"coreMethod,omitempty"`
	Differentiators     *string          `json:"differentiators,omitempty"`
	DataProcessingNeeds *[]string        `json:"dataProcessingNeeds,omitempty"`
	ModelChanges        *[]string        `json:"modelChanges,omitempty"`
	TrainingPlan        *string          `json:"trainingPlan,omitempty"`
	EvaluationMetrics   *[]string        `json:"evaluationMetrics,omitempty"`
	RiskPoints          *[]string        `json:"riskPoints,omitempty"`
	ExpectedGains       *[]string        `json:"expectedGains,omitempty"`
	Score               *Phase4IdeaScore `json:"score,omitempty"`
	ScoreSummary        *map[string]any  `json:"scoreSummary,omitempty"`
	Status              *string          `json:"status,omitempty"`
	SourceType          *string          `json:"sourceType,omitempty"`
	FailureFeedback     *map[string]any  `json:"failureFeedback,omitempty"`
	LastFailureRunID    *string          `json:"lastFailureRunId,omitempty"`
}

type Phase4IdeaStatusUpdateRequest struct {
	Status           string         `json:"status" binding:"required"`
	FailureFeedback  map[string]any `json:"failureFeedback,omitempty"`
	LastFailureRunID string         `json:"lastFailureRunId,omitempty"`
}

type Phase4RunManifest struct {
	ID               string         `json:"id"`
	DatasetProfileID string         `json:"datasetProfileId"`
	IdeaID           string         `json:"ideaId"`
	ReaderContextID  string         `json:"readerContextId,omitempty"`
	CodeSnapshotID   string         `json:"codeSnapshotId,omitempty"`
	RunnerMode       string         `json:"runnerMode"`
	ServerID         string         `json:"serverId,omitempty"`
	GPU              string         `json:"gpu,omitempty"`
	Status           string         `json:"status"`
	RetryCount       int            `json:"retryCount"`
	MaxRetryCount    int            `json:"maxRetryCount"`
	ArtifactPaths    map[string]any `json:"artifactPaths"`
	LogsPath         string         `json:"logsPath,omitempty"`
	MetricsPath      string         `json:"metricsPath,omitempty"`
	FailureFeedback  map[string]any `json:"failureFeedback"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	StartedAt        *time.Time     `json:"startedAt,omitempty"`
	FinishedAt       *time.Time     `json:"finishedAt,omitempty"`
}

type Phase4RunManifestCreateRequest struct {
	DatasetProfileID string         `json:"datasetProfileId" binding:"required"`
	IdeaID           string         `json:"ideaId" binding:"required"`
	ReaderContextID  string         `json:"readerContextId,omitempty"`
	CodeSnapshotID   string         `json:"codeSnapshotId,omitempty"`
	RunnerMode       string         `json:"runnerMode" binding:"required"`
	ServerID         string         `json:"serverId,omitempty"`
	GPU              string         `json:"gpu,omitempty"`
	Status           string         `json:"status"`
	RetryCount       int            `json:"retryCount"`
	MaxRetryCount    int            `json:"maxRetryCount"`
	ArtifactPaths    map[string]any `json:"artifactPaths"`
	LogsPath         string         `json:"logsPath,omitempty"`
	MetricsPath      string         `json:"metricsPath,omitempty"`
	FailureFeedback  map[string]any `json:"failureFeedback"`
}

type Phase4RunManifestUpdateRequest struct {
	CodeSnapshotID  *string         `json:"codeSnapshotId,omitempty"`
	RunnerMode      *string         `json:"runnerMode,omitempty"`
	ServerID        *string         `json:"serverId,omitempty"`
	GPU             *string         `json:"gpu,omitempty"`
	Status          *string         `json:"status,omitempty"`
	RetryCount      *int            `json:"retryCount,omitempty"`
	MaxRetryCount   *int            `json:"maxRetryCount,omitempty"`
	ArtifactPaths   *map[string]any `json:"artifactPaths,omitempty"`
	LogsPath        *string         `json:"logsPath,omitempty"`
	MetricsPath     *string         `json:"metricsPath,omitempty"`
	FailureFeedback *map[string]any `json:"failureFeedback,omitempty"`
	StartedAt       *time.Time      `json:"startedAt,omitempty"`
	FinishedAt      *time.Time      `json:"finishedAt,omitempty"`
}

type Phase4RunManifestStatusUpdateRequest struct {
	Status          string         `json:"status" binding:"required"`
	RetryCount      *int           `json:"retryCount,omitempty"`
	FailureFeedback map[string]any `json:"failureFeedback,omitempty"`
	StartedAt       *time.Time     `json:"startedAt,omitempty"`
	FinishedAt      *time.Time     `json:"finishedAt,omitempty"`
}

type Phase4StructuredReportRecord struct {
	ID                    string         `json:"id"`
	RunManifestID         string         `json:"runManifestId"`
	DatasetProfileID      string         `json:"datasetProfileId,omitempty"`
	IdeaID                string         `json:"ideaId,omitempty"`
	ReaderContextID       string         `json:"readerContextId,omitempty"`
	Title                 string         `json:"title"`
	MachineReadableReport map[string]any `json:"machineReadableReport"`
	HumanReadableReportMD string         `json:"humanReadableReportMd"`
	CitationRefs          []string       `json:"citationRefs"`
	ReferenceSourceIDs    []string       `json:"referenceSourceIds"`
	Status                string         `json:"status"`
	CreatedAt             time.Time      `json:"createdAt"`
	UpdatedAt             time.Time      `json:"updatedAt"`
}

type Phase4StructuredReportCreateRequest struct {
	RunManifestID         string         `json:"runManifestId" binding:"required"`
	DatasetProfileID      string         `json:"datasetProfileId,omitempty"`
	IdeaID                string         `json:"ideaId,omitempty"`
	ReaderContextID       string         `json:"readerContextId,omitempty"`
	Title                 string         `json:"title" binding:"required"`
	MachineReadableReport map[string]any `json:"machineReadableReport"`
	HumanReadableReportMD string         `json:"humanReadableReportMd"`
	CitationRefs          []string       `json:"citationRefs"`
	ReferenceSourceIDs    []string       `json:"referenceSourceIds"`
	Status                string         `json:"status"`
}

type Phase4StructuredReportUpdateRequest struct {
	Title                 *string         `json:"title,omitempty"`
	MachineReadableReport *map[string]any `json:"machineReadableReport,omitempty"`
	HumanReadableReportMD *string         `json:"humanReadableReportMd,omitempty"`
	CitationRefs          *[]string       `json:"citationRefs,omitempty"`
	ReferenceSourceIDs    *[]string       `json:"referenceSourceIds,omitempty"`
	Status                *string         `json:"status,omitempty"`
}

func ValidatePhase4DatasetProfileCreateRequest(req Phase4DatasetProfileCreateRequest) error {
	if strings.TrimSpace(req.DatasetName) == "" {
		return fmt.Errorf("datasetName is required")
	}
	if strings.TrimSpace(req.TaskType) == "" {
		return fmt.Errorf("taskType is required")
	}
	sourceMode := NormalizePhase4DatasetProfileSourceMode(req.SourceMode)
	if sourceMode == Phase4DatasetProfileSourceRegisteredPath {
		if strings.TrimSpace(req.ServerID) == "" {
			return fmt.Errorf("serverId is required for registered_path dataset profiles")
		}
		if strings.TrimSpace(req.ServerPath) == "" {
			return fmt.Errorf("serverPath is required for registered_path dataset profiles")
		}
	}
	for _, split := range req.Splits {
		if strings.TrimSpace(split.Name) == "" {
			return fmt.Errorf("dataset split name is required")
		}
		if split.SampleCount < 0 {
			return fmt.Errorf("dataset split sampleCount must be >= 0")
		}
	}
	return nil
}

func NormalizePhase4DatasetProfileSourceMode(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return Phase4DatasetProfileSourceRegisteredPath
	}
	return value
}

func NormalizePhase4DatasetProfileStatus(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return Phase4DatasetProfileStatusActive
	}
	return value
}

func ValidatePhase4IdeaScore(score Phase4IdeaScore) error {
	for label, value := range map[string]float64{
		"novelty":         score.Novelty,
		"datasetFit":      score.DatasetFit,
		"feasibility":     score.Feasibility,
		"expectedGain":    score.ExpectedGain,
		"computeCost":     score.ComputeCost,
		"failureRisk":     score.FailureRisk,
		"reproducibility": score.Reproducibility,
	} {
		if value < 0 || value > 10 {
			return fmt.Errorf("%s must be between 0 and 10", label)
		}
	}
	return nil
}

func NormalizePhase4IdeaStatus(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return Phase4IdeaStatusDraft
	}
	return value
}

func ValidatePhase4IdeaStatusTransition(current string, next string) error {
	current = NormalizePhase4IdeaStatus(current)
	next = NormalizePhase4IdeaStatus(next)
	if current == next {
		return nil
	}
	allowed := map[string]map[string]struct{}{
		Phase4IdeaStatusDraft: {
			Phase4IdeaStatusScored:   {},
			Phase4IdeaStatusRejected: {},
			Phase4IdeaStatusArchived: {},
		},
		Phase4IdeaStatusScored: {
			Phase4IdeaStatusSelected: {},
			Phase4IdeaStatusRejected: {},
			Phase4IdeaStatusArchived: {},
		},
		Phase4IdeaStatusRejected: {
			Phase4IdeaStatusArchived: {},
		},
		Phase4IdeaStatusSelected: {
			Phase4IdeaStatusImplemented: {},
			Phase4IdeaStatusFailed:      {},
			Phase4IdeaStatusArchived:    {},
		},
		Phase4IdeaStatusImplemented: {
			Phase4IdeaStatusArchived: {},
		},
		Phase4IdeaStatusFailed: {
			Phase4IdeaStatusSelected: {},
			Phase4IdeaStatusArchived: {},
		},
		Phase4IdeaStatusArchived: {},
	}
	nextSet, ok := allowed[current]
	if !ok {
		return fmt.Errorf("unsupported current idea status: %s", current)
	}
	if _, ok = nextSet[next]; !ok {
		return fmt.Errorf("invalid idea status transition: %s -> %s", current, next)
	}
	return nil
}

func NormalizePhase4RunStatus(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return Phase4RunStatusDraft
	}
	return value
}

func ValidatePhase4RunStatusTransition(current string, next string) error {
	current = NormalizePhase4RunStatus(current)
	next = NormalizePhase4RunStatus(next)
	if current == next {
		return nil
	}
	allowed := map[string]map[string]struct{}{
		Phase4RunStatusDraft: {
			Phase4RunStatusQueued:   {},
			Phase4RunStatusArchived: {},
			Phase4RunStatusCanceled: {},
		},
		Phase4RunStatusQueued: {
			Phase4RunStatusScheduled: {},
			Phase4RunStatusRunning:   {},
			Phase4RunStatusFailed:    {},
			Phase4RunStatusCanceled:  {},
			Phase4RunStatusArchived:  {},
		},
		Phase4RunStatusScheduled: {
			Phase4RunStatusRunning:  {},
			Phase4RunStatusFailed:   {},
			Phase4RunStatusCanceled: {},
			Phase4RunStatusArchived: {},
		},
		Phase4RunStatusRunning: {
			Phase4RunStatusSucceeded:  {},
			Phase4RunStatusFailed:     {},
			Phase4RunStatusTestFailed: {},
			Phase4RunStatusCanceled:   {},
			Phase4RunStatusArchived:   {},
		},
		Phase4RunStatusFailed: {
			Phase4RunStatusQueued:   {},
			Phase4RunStatusArchived: {},
		},
		Phase4RunStatusTestFailed: {
			Phase4RunStatusQueued:   {},
			Phase4RunStatusArchived: {},
		},
		Phase4RunStatusSucceeded: {
			Phase4RunStatusArchived: {},
		},
		Phase4RunStatusCanceled: {
			Phase4RunStatusArchived: {},
		},
		Phase4RunStatusArchived: {},
	}
	nextSet, ok := allowed[current]
	if !ok {
		return fmt.Errorf("unsupported current run status: %s", current)
	}
	if _, ok = nextSet[next]; !ok {
		return fmt.Errorf("invalid run status transition: %s -> %s", current, next)
	}
	return nil
}

func ValidatePhase4RetryCounts(retryCount int, maxRetryCount int) error {
	if retryCount < 0 {
		return fmt.Errorf("retryCount must be >= 0")
	}
	if maxRetryCount <= 0 {
		return fmt.Errorf("maxRetryCount must be > 0")
	}
	if retryCount > maxRetryCount {
		return fmt.Errorf("retryCount cannot exceed maxRetryCount")
	}
	return nil
}

func NormalizePhase4ReportStatus(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return Phase4ReportStatusDraft
	}
	return value
}
