package model

type InsightRunRequest struct {
	PaperID          string   `json:"paper_id"`
	ParsedContentRef string   `json:"parsed_content_ref"`
	Focus            string   `json:"focus"`
	ExecutionMode    string   `json:"execution_mode"`
	ModelProvider    string   `json:"model_provider"`
	ModelName        string   `json:"model_name"`
	PromptVersion    string   `json:"prompt_version"`
	SkillRefs        []string `json:"skill_refs"`
	ToolRefs         []string `json:"tool_refs"`
	MemoryRefs       []string `json:"memory_refs"`
}

type InsightRunResult struct {
	Job         *AgentJob    `json:"job"`
	Insight     PaperInsight `json:"insight"`
	SummaryPath string       `json:"summary_path"`
	Warnings    []string     `json:"warnings"`
}
