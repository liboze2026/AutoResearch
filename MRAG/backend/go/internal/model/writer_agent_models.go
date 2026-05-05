package model

import "time"

type FigurePlanItem struct {
	FigureID         string   `json:"figure_id"`
	FigureType       string   `json:"figure_type"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	SourceRefs       []string `json:"source_refs,omitempty"`
	PlaceholderNotes []string `json:"placeholder_notes,omitempty"`
}

type WriterRunRequest struct {
	PaperTemplateRef     string   `json:"paper_template_ref"`
	IdeaRefs             []string `json:"idea_refs"`
	ExperimentResultRefs []string `json:"experiment_result_refs"`
	ComparisonRefs       []string `json:"comparison_refs"`
	CitationRefs         []string `json:"citation_refs"`
	ExecutionMode        string   `json:"execution_mode"`
	ModelProvider        string   `json:"model_provider"`
	ModelName            string   `json:"model_name"`
	PromptVersion        string   `json:"prompt_version"`
	SkillRefs            []string `json:"skill_refs"`
	ToolRefs             []string `json:"tool_refs"`
	MemoryRefs           []string `json:"memory_refs"`
}

type DraftDocument struct {
	DraftID           string           `json:"draft_id"`
	Title             string           `json:"title"`
	Abstract          string           `json:"abstract"`
	Introduction      string           `json:"introduction"`
	Method            string           `json:"method"`
	Experiments       string           `json:"experiments"`
	Conclusion        string           `json:"conclusion"`
	ReferencesStub    []string         `json:"references_stub"`
	FigurePlan        []FigurePlanItem `json:"figure_plan"`
	PaperTemplateRef  string           `json:"paper_template_ref"`
	IdeaRefs          []string         `json:"idea_refs"`
	ExperimentRunRefs []string         `json:"experiment_result_refs"`
	ComparisonRefs    []string         `json:"comparison_refs"`
	CitationRefs      []string         `json:"citation_refs"`
	ResultArchiveID   string           `json:"result_archive_id,omitempty"`
	DraftPath         string           `json:"draft_path"`
	DraftMarkdownPath string           `json:"draft_markdown_path"`
	FigurePlanPath    string           `json:"figure_plan_path"`
	GeneratedAt       time.Time        `json:"generated_at"`
}

type WriterRunResult struct {
	Job      *AgentJob      `json:"job"`
	Draft    *DraftDocument `json:"draft,omitempty"`
	Warnings []string       `json:"warnings"`
}
