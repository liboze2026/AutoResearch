package model

type Phase4ReaderManualPaperInput struct {
	Title         string   `json:"title"`
	Abstract      string   `json:"abstract"`
	SourceType    string   `json:"sourceType"`
	SourceURL     string   `json:"sourceUrl"`
	OpenAccessURL string   `json:"openAccessUrl"`
	Venue         string   `json:"venue"`
	Year          int      `json:"year"`
	Authors       []string `json:"authors"`
	FilePath      string   `json:"filePath"`
	Note          string   `json:"note"`
}

type Phase4ReaderRunRequest struct {
	DatasetProfileID string                         `json:"datasetProfileId" binding:"required"`
	ManualPapers     []Phase4ReaderManualPaperInput `json:"manualPapers"`
	UserNotes        string                         `json:"userNotes"`
	SearchMode       string                         `json:"searchMode"`
	MaxPapers        int                            `json:"maxPapers"`
	ExecutionMode    string                         `json:"executionMode"`
	ModelProvider    string                         `json:"modelProvider"`
	ModelName        string                         `json:"modelName"`
	PromptVersion    string                         `json:"promptVersion"`
	SkillRefs        []string                       `json:"skillRefs"`
	ToolRefs         []string                       `json:"toolRefs"`
	MemoryRefs       []string                       `json:"memoryRefs"`
}

type Phase4ReaderSourcePayload struct {
	Title           string         `json:"title"`
	Abstract        string         `json:"abstract"`
	Authors         []string       `json:"authors"`
	Venue           string         `json:"venue"`
	PublicationYear int            `json:"publication_year"`
	SourceType      string         `json:"source_type"`
	SourceURL       string         `json:"source_url"`
	OpenAccessURL   string         `json:"open_access_url"`
	QualityTier     string         `json:"quality_tier"`
	RankingScore    float64        `json:"ranking_score"`
	QualityScore    float64        `json:"quality_score"`
	RelevanceScore  float64        `json:"relevance_score"`
	CitationCount   int            `json:"citation_count"`
	Metadata        map[string]any `json:"metadata"`
}

type Phase4ReaderContextPayload struct {
	TaskDefinition              string           `json:"task_definition"`
	DatasetSpecificChallenges   []string         `json:"dataset_specific_challenges"`
	RelevantMethodsLandscape    []string         `json:"relevant_methods_landscape"`
	LikelyStrongBaselines       []string         `json:"likely_strong_baselines"`
	CommonFailurePoints         []string         `json:"common_failure_points"`
	EvaluationCaveats           []string         `json:"evaluation_caveats"`
	ImplementationConstraints   []string         `json:"implementation_constraints"`
	PromisingResearchDirections []string         `json:"promising_research_directions"`
	CitationMetadata            []map[string]any `json:"citation_metadata"`
	ReadingSummary              string           `json:"reading_summary"`
	UserNotes                   string           `json:"user_notes"`
}

type Phase4ReaderRuntimePayload struct {
	Summary          string                      `json:"summary"`
	ReadingSummary   string                      `json:"reading_summary"`
	Sources          []Phase4ReaderSourcePayload `json:"sources"`
	ReaderContext    Phase4ReaderContextPayload  `json:"reader_context"`
	CitationMetadata []map[string]any            `json:"citation_metadata"`
	Data             map[string]any              `json:"data"`
	Metadata         map[string]any              `json:"metadata"`
}

type Phase4ReaderRunResult struct {
	Job           *AgentJob            `json:"job"`
	ReaderContext *Phase4ReaderContext `json:"readerContext,omitempty"`
	ReaderSources []Phase4ReaderSource `json:"readerSources"`
	Warnings      []string             `json:"warnings"`
}

type Phase4ReaderJobDetail struct {
	Job           *AgentJob            `json:"job"`
	Artifacts     []AgentArtifact      `json:"artifacts"`
	ReaderContext *Phase4ReaderContext `json:"readerContext,omitempty"`
	ReaderSources []Phase4ReaderSource `json:"readerSources"`
	Warnings      []string             `json:"warnings"`
}
