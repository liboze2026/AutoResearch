package model

import "time"

type ToolDefinition struct {
	ToolID         string         `json:"tool_id"`
	Name           string         `json:"name"`
	OwnerAgentType string         `json:"owner_agent_type"`
	Path           string         `json:"path"`
	Description    string         `json:"description"`
	UsageMD        string         `json:"usage_md"`
	InputSchema    map[string]any `json:"input_schema"`
	OutputSchema   map[string]any `json:"output_schema"`
	TestStatus     string         `json:"test_status"`
	Version        string         `json:"version"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type ToolRegisterRequest struct {
	Name           string         `json:"name"`
	OwnerAgentType string         `json:"owner_agent_type"`
	Path           string         `json:"path"`
	Description    string         `json:"description"`
	UsageMD        string         `json:"usage_md"`
	InputSchema    map[string]any `json:"input_schema"`
	OutputSchema   map[string]any `json:"output_schema"`
	TestStatus     string         `json:"test_status,omitempty"`
	Version        string         `json:"version"`
	ScriptName     string         `json:"script_name"`
	ScriptContent  string         `json:"script_content"`
}

type SkillDefinition struct {
	SkillID      string    `json:"skill_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SkillDir     string    `json:"skill_dir"`
	Entrypoint   string    `json:"entrypoint"`
	Dependencies []string  `json:"dependencies"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SkillRegisterRequest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SkillDir     string   `json:"skill_dir"`
	Entrypoint   string   `json:"entrypoint"`
	Dependencies []string `json:"dependencies"`
	EntryContent string   `json:"entry_content"`
}

type AgentMemoryRecord struct {
	ID        string    `json:"id"`
	AgentType string    `json:"agent_type"`
	MemoryKey string    `json:"memory_key"`
	ContentMD string    `json:"content_md"`
	SourceRef string    `json:"source_ref"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

type AgentMemoryUpsertRequest struct {
	AgentType string `json:"agent_type"`
	MemoryKey string `json:"memory_key"`
	ContentMD string `json:"content_md"`
	SourceRef string `json:"source_ref"`
}
