package model

import "time"

type AgentInputRef struct {
	RefType    string         `json:"ref_type"`
	RefID      string         `json:"ref_id,omitempty"`
	RefPath    string         `json:"ref_path,omitempty"`
	RefVersion string         `json:"ref_version,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type AgentArtifactManifestItem struct {
	ArtifactType string         `json:"artifact_type"`
	Name         string         `json:"name"`
	FilePath     string         `json:"file_path"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type AgentRepairAction struct {
	Action   string         `json:"action"`
	Status   string         `json:"status"`
	Detail   string         `json:"detail"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type AgentToolUsage struct {
	ToolRef  string         `json:"tool_ref"`
	Status   string         `json:"status"`
	Summary  string         `json:"summary"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type AgentRuntimeInput struct {
	JobID           string          `json:"job_id"`
	AgentType       string          `json:"agent_type"`
	ExecutionMode   string          `json:"execution_mode"`
	ModelProvider   string          `json:"model_provider"`
	ModelName       string          `json:"model_name"`
	PromptVersion   string          `json:"prompt_version"`
	InputRefs       []AgentInputRef `json:"input_refs"`
	OutputSchemaRef string          `json:"output_schema_ref"`
	SkillRefs       []string        `json:"skill_refs"`
	ToolRefs        []string        `json:"tool_refs"`
	MemoryRefs      []string        `json:"memory_refs"`
	WorkspaceDir    string          `json:"workspace_dir"`
	Metadata        map[string]any  `json:"metadata"`
}

type AgentRuntimeOutput struct {
	Status            string                      `json:"status"`
	NormalizedPayload map[string]any              `json:"normalized_payload"`
	ArtifactManifest  []AgentArtifactManifestItem `json:"artifact_manifest"`
	RepairActions     []AgentRepairAction         `json:"repair_actions"`
	ToolUsages        []AgentToolUsage            `json:"tool_usages"`
	Warnings          []string                    `json:"warnings"`
	ValidationStatus  string                      `json:"validation_status"`
	RepairStatus      string                      `json:"repair_status"`
	ValidationErrors  []string                    `json:"validation_errors"`
	ErrorMessage      string                      `json:"error_message"`
}

type AgentJob struct {
	ID                string                      `json:"id"`
	AgentType         string                      `json:"agent_type"`
	ExecutionMode     string                      `json:"execution_mode"`
	ModelProvider     string                      `json:"model_provider"`
	ModelName         string                      `json:"model_name"`
	PromptVersion     string                      `json:"prompt_version"`
	InputRefs         []AgentInputRef             `json:"input_refs"`
	OutputSchemaRef   string                      `json:"output_schema_ref"`
	SkillRefs         []string                    `json:"skill_refs"`
	ToolRefs          []string                    `json:"tool_refs"`
	MemoryRefs        []string                    `json:"memory_refs"`
	WorkspaceDir      string                      `json:"workspace_dir"`
	Metadata          map[string]any              `json:"metadata"`
	TriggerEventID    string                      `json:"trigger_event_id"`
	DedupKey          string                      `json:"dedup_key"`
	RetryCount        int                         `json:"retry_count"`
	MaxRetries        int                         `json:"max_retries"`
	ConcurrencyLimit  int                         `json:"concurrency_limit"`
	Status            string                      `json:"status"`
	NormalizedPayload map[string]any              `json:"normalized_payload"`
	ArtifactManifest  []AgentArtifactManifestItem `json:"artifact_manifest"`
	RepairActions     []AgentRepairAction         `json:"repair_actions"`
	ToolUsages        []AgentToolUsage            `json:"tool_usages"`
	Warnings          []string                    `json:"warnings"`
	ValidationStatus  string                      `json:"validation_status"`
	RepairStatus      string                      `json:"repair_status"`
	ValidationErrors  []string                    `json:"validation_errors"`
	ErrorMessage      string                      `json:"error_message"`
	StartedAt         *time.Time                  `json:"started_at,omitempty"`
	CompletedAt       *time.Time                  `json:"completed_at,omitempty"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
}

type AgentArtifact struct {
	ID           string         `json:"id"`
	JobID        string         `json:"job_id"`
	ArtifactType string         `json:"artifact_type"`
	Name         string         `json:"name"`
	FilePath     string         `json:"file_path"`
	Checksum     string         `json:"checksum"`
	MetadataJSON map[string]any `json:"metadata_json"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type AgentJobTrigger struct {
	ID           string         `json:"id"`
	JobID        string         `json:"job_id"`
	TriggerType  string         `json:"trigger_type"`
	Status       string         `json:"status"`
	Metadata     map[string]any `json:"metadata"`
	ErrorMessage string         `json:"error_message"`
	RequestedAt  time.Time      `json:"requested_at"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type AgentSchema struct {
	ID              string         `json:"id"`
	SchemaName      string         `json:"schema_name"`
	Version         string         `json:"version"`
	AgentType       string         `json:"agent_type"`
	SchemaRef       string         `json:"schema_ref"`
	JSONSchema      map[string]any `json:"json_schema"`
	PythonSchemaRef string         `json:"python_schema_ref"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type AgentEvent struct {
	ID              string          `json:"id"`
	EventType       string          `json:"event_type"`
	SourceRef       string          `json:"source_ref"`
	InputRefs       []AgentInputRef `json:"input_refs"`
	Payload         map[string]any  `json:"payload"`
	Status          string          `json:"status"`
	TriggeredJobIDs []string        `json:"triggered_job_ids"`
	ErrorMessage    string          `json:"error_message"`
	CreatedAt       time.Time       `json:"created_at"`
	ProcessedAt     *time.Time      `json:"processed_at,omitempty"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type AgentSubscription struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	EventType        string         `json:"event_type"`
	AgentType        string         `json:"agent_type"`
	Enabled          bool           `json:"enabled"`
	ExecutionMode    string         `json:"execution_mode"`
	ModelProvider    string         `json:"model_provider"`
	ModelName        string         `json:"model_name"`
	PromptVersion    string         `json:"prompt_version"`
	OutputSchemaRef  string         `json:"output_schema_ref"`
	SkillRefs        []string       `json:"skill_refs"`
	ToolRefs         []string       `json:"tool_refs"`
	MemoryRefs       []string       `json:"memory_refs"`
	TriggerRule      map[string]any `json:"trigger_rule"`
	MaxRetries       int            `json:"max_retries"`
	ConcurrencyLimit int            `json:"concurrency_limit"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}
