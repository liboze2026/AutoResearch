package model

type Phase4IdeaSeedInput struct {
	Title               string   `json:"title"`
	ProblemDefinition   string   `json:"problemDefinition"`
	CoreMethod          string   `json:"coreMethod"`
	Differentiators     string   `json:"differentiators"`
	DataProcessingNeeds []string `json:"dataProcessingNeeds"`
	ModelChanges        []string `json:"modelChanges"`
	TrainingPlan        string   `json:"trainingPlan"`
	EvaluationMetrics   []string `json:"evaluationMetrics"`
	RiskPoints          []string `json:"riskPoints"`
	ExpectedGains       []string `json:"expectedGains"`
	SourceType          string   `json:"sourceType"`
	RevisionOfID        string   `json:"revisionOfId"`
}

type Phase4IdeaRunRequest struct {
	DatasetProfileID string               `json:"datasetProfileId" binding:"required"`
	ReaderContextID  string               `json:"readerContextId" binding:"required"`
	UserNotes        string               `json:"userNotes"`
	ManualIdea       *Phase4IdeaSeedInput `json:"manualIdea,omitempty"`
	TargetCount      int                  `json:"targetCount"`
	ExecutionMode    string               `json:"executionMode"`
	ModelProvider    string               `json:"modelProvider"`
	ModelName        string               `json:"modelName"`
	PromptVersion    string               `json:"promptVersion"`
	SkillRefs        []string             `json:"skillRefs"`
	ToolRefs         []string             `json:"toolRefs"`
	MemoryRefs       []string             `json:"memoryRefs"`
}

type Phase4IdeaRevisionGenerateRequest struct {
	FailureFeedback  map[string]any `json:"failureFeedback"`
	LastFailureRunID string         `json:"lastFailureRunId"`
	UserNotes        string         `json:"userNotes"`
	TargetCount      int            `json:"targetCount"`
	ExecutionMode    string         `json:"executionMode"`
	ModelProvider    string         `json:"modelProvider"`
	ModelName        string         `json:"modelName"`
	PromptVersion    string         `json:"promptVersion"`
	SkillRefs        []string       `json:"skillRefs"`
	ToolRefs         []string       `json:"toolRefs"`
	MemoryRefs       []string       `json:"memoryRefs"`
}

type Phase4IdeaRuntimeCandidate struct {
	Title               string          `json:"title"`
	ProblemDefinition   string          `json:"problem_definition"`
	CoreMethod          string          `json:"core_method"`
	Differentiators     string          `json:"differentiators"`
	DataProcessingNeeds []string        `json:"data_processing_needs"`
	ModelChanges        []string        `json:"model_changes"`
	TrainingPlan        string          `json:"training_plan"`
	EvaluationMetrics   []string        `json:"evaluation_metrics"`
	RiskPoints          []string        `json:"risk_points"`
	ExpectedGains       []string        `json:"expected_gains"`
	Score               Phase4IdeaScore `json:"score"`
	ScoreSummary        map[string]any  `json:"score_summary"`
	Status              string          `json:"status"`
	SourceType          string          `json:"source_type"`
	RevisionOfID        string          `json:"revision_of_id"`
	FailureFeedback     map[string]any  `json:"failure_feedback"`
	LastFailureRunID    string          `json:"last_failure_run_id"`
}

type Phase4IdeaTopRecommendation struct {
	Title                string          `json:"title"`
	OverallScore         float64         `json:"overallScore"`
	Rank                 int             `json:"rank"`
	RecommendationReason string          `json:"recommendationReason"`
	Score                Phase4IdeaScore `json:"score"`
}

type Phase4IdeaRuntimePayload struct {
	Summary            string                        `json:"summary"`
	Ideas              []Phase4IdeaRuntimeCandidate  `json:"ideas"`
	TopRecommendations []Phase4IdeaTopRecommendation `json:"top_recommendations"`
	GenerationMode     string                        `json:"generation_mode"`
	Data               map[string]any                `json:"data"`
	Metadata           map[string]any                `json:"metadata"`
}

type Phase4IdeaScoreView struct {
	ID                   string          `json:"id"`
	DatasetProfileID     string          `json:"datasetProfileId,omitempty"`
	ReaderContextID      string          `json:"readerContextId,omitempty"`
	Title                string          `json:"title"`
	Status               string          `json:"status"`
	SourceType           string          `json:"sourceType"`
	RevisionOfID         string          `json:"revisionOfId,omitempty"`
	LineageRootID        string          `json:"lineageRootId,omitempty"`
	LastFailureRunID     string          `json:"lastFailureRunId,omitempty"`
	Score                Phase4IdeaScore `json:"score"`
	OverallScore         float64         `json:"overallScore"`
	Rank                 int             `json:"rank"`
	RecommendationTier   string          `json:"recommendationTier"`
	RecommendationReason string          `json:"recommendationReason"`
	ExpectedGains        []string        `json:"expectedGains"`
	RiskPoints           []string        `json:"riskPoints"`
}

type Phase4IdeaRunResult struct {
	Job                *AgentJob             `json:"job"`
	Ideas              []Phase4Idea          `json:"ideas"`
	TopRecommendations []Phase4IdeaScoreView `json:"topRecommendations"`
	Warnings           []string              `json:"warnings"`
}

type Phase4IdeaJobDetail struct {
	Job                *AgentJob             `json:"job"`
	Artifacts          []AgentArtifact       `json:"artifacts"`
	Ideas              []Phase4Idea          `json:"ideas"`
	TopRecommendations []Phase4IdeaScoreView `json:"topRecommendations"`
	Warnings           []string              `json:"warnings"`
}
