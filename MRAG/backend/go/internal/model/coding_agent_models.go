package model

type CodingPatchManifestItem struct {
	PatchType string         `json:"patch_type"`
	Target    string         `json:"target"`
	Action    string         `json:"action"`
	Value     any            `json:"value,omitempty"`
	FilePath  string         `json:"file_path,omitempty"`
	Summary   string         `json:"summary,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type CodingRunRequest struct {
	ExperimentID      string   `json:"experiment_id"`
	IdeaID            string   `json:"idea_id"`
	ExperimentPlanRef string   `json:"experiment_plan_ref"`
	ExperimentSpecRef string   `json:"experiment_spec_ref"`
	TrainTemplateRef  string   `json:"train_template_ref"`
	EvalProtocolRef   string   `json:"eval_protocol_ref"`
	ExecutionMode     string   `json:"execution_mode"`
	ModelProvider     string   `json:"model_provider"`
	ModelName         string   `json:"model_name"`
	PromptVersion     string   `json:"prompt_version"`
	SkillRefs         []string `json:"skill_refs"`
	ToolRefs          []string `json:"tool_refs"`
	MemoryRefs        []string `json:"memory_refs"`
}

type CodingRunResult struct {
	Job           *AgentJob                 `json:"job"`
	Experiment    *ExperimentDetail         `json:"experiment,omitempty"`
	Run           *ExperimentRun            `json:"run,omitempty"`
	Comparison    *RunCompareResult         `json:"comparison,omitempty"`
	PatchManifest []CodingPatchManifestItem `json:"patch_manifest"`
	Warnings      []string                  `json:"warnings"`
}
