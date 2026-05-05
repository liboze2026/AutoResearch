package model

type AgentSummary struct {
	AgentType        string              `json:"agent_type"`
	EventTypes       []string            `json:"event_types"`
	ExecutionMode    string              `json:"execution_mode"`
	ModelProvider    string              `json:"model_provider"`
	ModelName        string              `json:"model_name"`
	PromptVersion    string              `json:"prompt_version"`
	OutputSchemaRef  string              `json:"output_schema_ref"`
	SkillRefs        []string            `json:"skill_refs"`
	ToolRefs         []string            `json:"tool_refs"`
	MemoryRefs       []string            `json:"memory_refs"`
	ConcurrencyLimit int                 `json:"concurrency_limit"`
	MaxRetries       int                 `json:"max_retries"`
	JobCount         int                 `json:"job_count"`
	LatestJob        *AgentJob           `json:"latest_job,omitempty"`
	Subscriptions    []AgentSubscription `json:"subscriptions"`
}
