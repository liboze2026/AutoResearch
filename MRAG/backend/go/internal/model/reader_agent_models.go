package model

type ReaderManualPaperInput struct {
	Title      string `json:"title"`
	Abstract   string `json:"abstract"`
	Source     string `json:"source"`
	Year       int    `json:"year"`
	URL        string `json:"url"`
	FileStatus string `json:"file_status"`
	FilePath   string `json:"file_path"`
}

type ReaderRunRequest struct {
	ResearchDirection string                   `json:"research_direction"`
	Keywords          []string                 `json:"keywords"`
	SourceScope       string                   `json:"source_scope"`
	TimeRange         map[string]any           `json:"time_range"`
	MaxPapers         int                      `json:"max_papers"`
	ExecutionMode     string                   `json:"execution_mode"`
	ModelProvider     string                   `json:"model_provider"`
	ModelName         string                   `json:"model_name"`
	PromptVersion     string                   `json:"prompt_version"`
	SkillRefs         []string                 `json:"skill_refs"`
	ToolRefs          []string                 `json:"tool_refs"`
	MemoryRefs        []string                 `json:"memory_refs"`
	ManualPapers      []ReaderManualPaperInput `json:"manual_papers"`
}

type ReaderCandidatePaper struct {
	Title      string `json:"title"`
	Abstract   string `json:"abstract"`
	Source     string `json:"source"`
	Year       int    `json:"year"`
	URL        string `json:"url"`
	FileStatus string `json:"file_status"`
	FilePath   string `json:"file_path,omitempty"`
}

type ReaderImportedPaper struct {
	Candidate ReaderCandidatePaper `json:"candidate"`
	Result    PaperImportResult    `json:"result"`
}

type ReaderRunResult struct {
	Job             *AgentJob              `json:"job"`
	CandidatePapers []ReaderCandidatePaper `json:"candidate_papers"`
	ImportedPapers  []ReaderImportedPaper  `json:"imported_papers"`
	Warnings        []string               `json:"warnings"`
}

type ReaderJobDetail struct {
	Job             *AgentJob              `json:"job"`
	Artifacts       []AgentArtifact        `json:"artifacts"`
	CandidatePapers []ReaderCandidatePaper `json:"candidate_papers"`
	ImportedPapers  []ReaderImportedPaper  `json:"imported_papers"`
}
