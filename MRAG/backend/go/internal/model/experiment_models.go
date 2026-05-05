package model

import "time"

type Experiment struct {
	ID             string    `json:"id"`
	IdeaID         string    `json:"ideaId,omitempty"`
	DatasetAssetID string    `json:"datasetAssetId"`
	BaselineID     string    `json:"baselineId,omitempty"`
	Title          string    `json:"title"`
	Status         string    `json:"status"`
	Priority       int       `json:"priority"`
	CurrentRunID   string    `json:"currentRunId,omitempty"`
	SummaryMD      string    `json:"summaryMd"`
	OwnerNoteMD    string    `json:"ownerNoteMd"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type ExperimentSpec struct {
	ID            string                 `json:"id"`
	ExperimentID  string                 `json:"experimentId"`
	SpecJSON      map[string]interface{} `json:"specJson"`
	TemplateType  string                 `json:"templateType"`
	GeneratedFrom map[string]interface{} `json:"generatedFrom"`
	Version       int                    `json:"version"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

type ExperimentRun struct {
	ID               string                 `json:"id"`
	ExperimentID     string                 `json:"experimentId"`
	SpecID           string                 `json:"specId,omitempty"`
	AssignedServerID string                 `json:"assignedServerId,omitempty"`
	RunStatus        string                 `json:"runStatus"`
	RemoteWorkdir    string                 `json:"remoteWorkdir"`
	RemoteJobID      string                 `json:"remoteJobId,omitempty"`
	StartedAt        *time.Time             `json:"startedAt,omitempty"`
	EndedAt          *time.Time             `json:"endedAt,omitempty"`
	RetryCount       int                    `json:"retryCount"`
	ExitCode         *int                   `json:"exitCode,omitempty"`
	ResultJSON       map[string]interface{} `json:"resultJson"`
	ErrorMessage     string                 `json:"errorMessage"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
}

type RunLog struct {
	ID        int64     `json:"id"`
	RunID     string    `json:"runId"`
	LogType   string    `json:"logType"`
	LogPath   string    `json:"logPath"`
	TailText  string    `json:"tailText"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SchedulerDecision struct {
	ID             string                 `json:"id"`
	RunID          string                 `json:"runId"`
	ChosenServerID string                 `json:"chosenServerId,omitempty"`
	DecisionJSON   map[string]interface{} `json:"decisionJson"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type ServerHeartbeat struct {
	ID          string                 `json:"id"`
	ServerID    string                 `json:"serverId"`
	HeartbeatAt time.Time              `json:"heartbeatAt"`
	Status      string                 `json:"status"`
	DetailJSON  map[string]interface{} `json:"detailJson"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

type GPUResourceSnapshot struct {
	ID          string                   `json:"id"`
	ServerID    string                   `json:"serverId"`
	CapturedAt  time.Time                `json:"capturedAt"`
	GPUIndex    int                      `json:"gpuIndex"`
	Name        string                   `json:"name"`
	TotalMemMB  int                      `json:"totalMemMb"`
	FreeMemMB   int                      `json:"freeMemMb"`
	Utilization int                      `json:"utilization"`
	ProcessJSON []map[string]interface{} `json:"processJson"`
	CreatedAt   time.Time                `json:"createdAt"`
	UpdatedAt   time.Time                `json:"updatedAt"`
}

type ResultComparison struct {
	ID                    string                 `json:"id"`
	ExperimentID          string                 `json:"experimentId"`
	RunID                 string                 `json:"runId"`
	BaselineID            string                 `json:"baselineId,omitempty"`
	TargetResultArchiveID string                 `json:"targetResultArchiveId,omitempty"`
	ComparisonJSON        map[string]interface{} `json:"comparisonJson"`
	SummaryMD             string                 `json:"summaryMd"`
	CreatedAt             time.Time              `json:"createdAt"`
	UpdatedAt             time.Time              `json:"updatedAt"`
}
