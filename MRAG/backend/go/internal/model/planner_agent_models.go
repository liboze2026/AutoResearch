package model

import "time"

type PlannerRunRequest struct {
	IdeaID                     string   `json:"idea_id"`
	DatasetAssetRefs           []string `json:"dataset_asset_refs"`
	EvalProtocolRefs           []string `json:"eval_protocol_refs"`
	ServerResourceSnapshotRefs []string `json:"server_resource_snapshot_refs"`
	BaselineRefs               []string `json:"baseline_refs"`
	HumanHints                 []string `json:"human_hints"`
	ExecutionMode              string   `json:"execution_mode"`
	ModelProvider              string   `json:"model_provider"`
	ModelName                  string   `json:"model_name"`
	PromptVersion              string   `json:"prompt_version"`
	SkillRefs                  []string `json:"skill_refs"`
	ToolRefs                   []string `json:"tool_refs"`
	MemoryRefs                 []string `json:"memory_refs"`
}

type ExperimentPlanDocument struct {
	ExperimentID               string         `json:"experiment_id"`
	IdeaID                     string         `json:"idea_id"`
	DatasetAssetID             string         `json:"dataset_asset_id"`
	BaselineID                 string         `json:"baseline_id,omitempty"`
	EvalProtocolRefs           []string       `json:"eval_protocol_refs"`
	ServerResourceSnapshotRefs []string       `json:"server_resource_snapshot_refs"`
	ExperimentPlanJSON         map[string]any `json:"experiment_plan_json"`
	TrainTemplateType          string         `json:"train_template_type"`
	ResourceEstimate           map[string]any `json:"resource_estimate"`
	RunSequence                []string       `json:"run_sequence"`
	SuccessCriteria            map[string]any `json:"success_criteria"`
	FallbackPlan               map[string]any `json:"fallback_plan"`
	PlanPath                   string         `json:"plan_path"`
	PlanMarkdownPath           string         `json:"plan_markdown_path"`
	RuntimeOutputPath          string         `json:"runtime_output_path,omitempty"`
	GeneratedAt                time.Time      `json:"generated_at"`
}

type PlannerRunResult struct {
	Job        *AgentJob               `json:"job"`
	Experiment *ExperimentDetail       `json:"experiment,omitempty"`
	Plan       *ExperimentPlanDocument `json:"plan,omitempty"`
	Warnings   []string                `json:"warnings"`
}

type ExperimentPlanResponse struct {
	Experiment *ExperimentDetail      `json:"experiment"`
	Plan       ExperimentPlanDocument `json:"plan"`
}
