package model

type Phase4CodingRunRequest struct {
	DatasetProfileID string   `json:"datasetProfileId" binding:"required"`
	IdeaID           string   `json:"ideaId" binding:"required"`
	ReaderContextID  string   `json:"readerContextId,omitempty"`
	RunnerMode       string   `json:"runnerMode"`
	ServerID         string   `json:"serverId,omitempty"`
	GPU              string   `json:"gpu,omitempty"`
	MaxRetryCount    int      `json:"maxRetryCount"`
	UserNotes        string   `json:"userNotes"`
	ExecutionMode    string   `json:"executionMode"`
	ModelProvider    string   `json:"modelProvider"`
	ModelName        string   `json:"modelName"`
	PromptVersion    string   `json:"promptVersion"`
	SkillRefs        []string `json:"skillRefs"`
	ToolRefs         []string `json:"toolRefs"`
	MemoryRefs       []string `json:"memoryRefs"`
}

type Phase4CodingMethodModule struct {
	ModuleName   string         `json:"module_name"`
	RelativePath string         `json:"relative_path"`
	BranchName   string         `json:"branch_name"`
	Summary      string         `json:"summary"`
	Content      string         `json:"content"`
	Metadata     map[string]any `json:"metadata"`
}

type Phase4CodingRuntimePayload struct {
	Summary            string                   `json:"summary"`
	ProtocolVersion    string                   `json:"protocol_version"`
	Phase4RunManifest  map[string]any           `json:"phase4_run_manifest"`
	Phase4Config       map[string]any           `json:"phase4_config"`
	MethodModule       Phase4CodingMethodModule `json:"method_module"`
	RetryPlan          map[string]any           `json:"retry_plan"`
	DatasetToolAssets  map[string]any           `json:"dataset_tool_assets"`
	EvaluateToolAssets map[string]any           `json:"evaluate_tool_assets"`
	Entrypoints        map[string]any           `json:"entrypoints"`
	Data               map[string]any           `json:"data"`
	Metadata           map[string]any           `json:"metadata"`
}

type Phase4CodingRunResult struct {
	Job         *AgentJob          `json:"job"`
	RunManifest *Phase4RunManifest `json:"runManifest,omitempty"`
	Warnings    []string           `json:"warnings"`
}

type Phase4CodingJobDetail struct {
	Job         *AgentJob          `json:"job"`
	Artifacts   []AgentArtifact    `json:"artifacts"`
	RunManifest *Phase4RunManifest `json:"runManifest,omitempty"`
	Warnings    []string           `json:"warnings"`
}
