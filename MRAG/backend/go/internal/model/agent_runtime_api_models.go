package model

import "time"

type AgentJobCreateRequest struct {
	AgentType        string          `json:"agent_type"`
	ExecutionMode    string          `json:"execution_mode"`
	ModelProvider    string          `json:"model_provider"`
	ModelName        string          `json:"model_name"`
	PromptVersion    string          `json:"prompt_version"`
	InputRefs        []AgentInputRef `json:"input_refs"`
	OutputSchemaRef  string          `json:"output_schema_ref"`
	SkillRefs        []string        `json:"skill_refs"`
	ToolRefs         []string        `json:"tool_refs"`
	MemoryRefs       []string        `json:"memory_refs"`
	WorkspaceDir     string          `json:"workspace_dir"`
	Metadata         map[string]any  `json:"metadata"`
	Status           string          `json:"status"`
	TriggerEventID   string          `json:"trigger_event_id"`
	DedupKey         string          `json:"dedup_key"`
	MaxRetries       int             `json:"max_retries"`
	ConcurrencyLimit int             `json:"concurrency_limit"`
}

type AgentJobTriggerRequest struct {
	TriggerType string         `json:"trigger_type"`
	Metadata    map[string]any `json:"metadata"`
}

type AgentJobStatusDetail struct {
	ID               string     `json:"id"`
	AgentType        string     `json:"agent_type"`
	Status           string     `json:"status"`
	ValidationStatus string     `json:"validation_status"`
	RepairStatus     string     `json:"repair_status"`
	ValidationErrors []string   `json:"validation_errors"`
	RepairCount      int        `json:"repair_count"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	Warnings         []string   `json:"warnings"`
	ErrorMessage     string     `json:"error_message"`
}

type AgentSchemaCreateRequest struct {
	SchemaName      string         `json:"schema_name"`
	Version         string         `json:"version"`
	AgentType       string         `json:"agent_type"`
	SchemaRef       string         `json:"schema_ref"`
	JSONSchema      map[string]any `json:"json_schema"`
	PythonSchemaRef string         `json:"python_schema_ref"`
}

type AgentEventCreateRequest struct {
	EventType string          `json:"event_type"`
	SourceRef string          `json:"source_ref"`
	InputRefs []AgentInputRef `json:"input_refs"`
	Payload   map[string]any  `json:"payload"`
}

type AgentSubscriptionCreateRequest struct {
	Name             string         `json:"name"`
	EventType        string         `json:"event_type"`
	AgentType        string         `json:"agent_type"`
	Enabled          *bool          `json:"enabled,omitempty"`
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
}
