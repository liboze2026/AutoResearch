package model

type StructuredIdeaPayload struct {
	Title                   string   `json:"title"`
	DescriptionMD           string   `json:"description_md"`
	ResearchDirection       string   `json:"research_direction"`
	TargetDatasetRefs       []string `json:"target_dataset_refs"`
	DatasetEvalProtocolRefs []string `json:"dataset_eval_protocol_refs"`
	InnovationType          string   `json:"innovation_type"`
	ExpectedAdvantage       string   `json:"expected_advantage"`
	RiskPoints              []string `json:"risk_points"`
	Priority                int      `json:"priority"`
	Confidence              float64  `json:"confidence"`
	PaperInsightRefs        []string `json:"paper_insight_refs,omitempty"`
	HumanHints              []string `json:"human_hints,omitempty"`
}

type IdeaGeneratorRunRequest struct {
	PaperInsightRefs []string               `json:"paper_insight_refs"`
	DatasetAssetRefs []string               `json:"dataset_asset_refs"`
	HumanHints       []string               `json:"human_hints"`
	ManualIdea       *StructuredIdeaPayload `json:"manual_idea,omitempty"`
	ExecutionMode    string                 `json:"execution_mode"`
	ModelProvider    string                 `json:"model_provider"`
	ModelName        string                 `json:"model_name"`
	PromptVersion    string                 `json:"prompt_version"`
	SkillRefs        []string               `json:"skill_refs"`
	ToolRefs         []string               `json:"tool_refs"`
	MemoryRefs       []string               `json:"memory_refs"`
}

type IdeaGeneratorRunResult struct {
	Job      *AgentJob   `json:"job"`
	Idea     *IdeaDetail `json:"idea,omitempty"`
	Warnings []string    `json:"warnings"`
}

type StructuredIdeaPersistRequest struct {
	StructuredIdea StructuredIdeaPayload
	SourceType     string
	PaperSources   []IdeaSource
	DatasetRefs    []string
	EvalPlanRefs   []string
	HumanHints     []string
	GeneratedFrom  string
}
