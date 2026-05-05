package model

import "time"

type DatasetRunRequest struct {
	ResearchDirection      string         `json:"research_direction"`
	TaskType               string         `json:"task_type"`
	Keywords               []string       `json:"keywords"`
	TargetServerPreference string         `json:"target_server_preference"`
	DatasetConstraints     map[string]any `json:"dataset_constraints"`
	ExecutionMode          string         `json:"execution_mode"`
	ModelProvider          string         `json:"model_provider"`
	ModelName              string         `json:"model_name"`
	PromptVersion          string         `json:"prompt_version"`
	SkillRefs              []string       `json:"skill_refs"`
	ToolRefs               []string       `json:"tool_refs"`
	MemoryRefs             []string       `json:"memory_refs"`
}

type DatasetEvalPlanDocument struct {
	DatasetAssetID     string         `json:"dataset_asset_id"`
	DatasetLocation    string         `json:"dataset_location"`
	FetchAction        string         `json:"fetch_action"`
	SelectedDatasetRef string         `json:"selected_dataset_ref"`
	ServerDecision     map[string]any `json:"server_decision"`
	EvalProtocolJSON   map[string]any `json:"eval_protocol_json"`
	MetricSchemaJSON   map[string]any `json:"metric_schema_json"`
	SplitStrategy      string         `json:"split_strategy"`
	NotesMD            string         `json:"notes_md"`
	EvalPlanPath       string         `json:"evalplan_path"`
	NotesPath          string         `json:"notes_path"`
	RuntimeOutputPath  string         `json:"runtime_output_path"`
	BaselineID         string         `json:"baseline_id,omitempty"`
	BaselineNeeded     bool           `json:"baseline_needed"`
	MemoryKey          string         `json:"memory_key,omitempty"`
	TargetServerName   string         `json:"target_server_name"`
	GeneratedAt        time.Time      `json:"generated_at"`
}

type DatasetRunResult struct {
	Job          *AgentJob                `json:"job"`
	DatasetAsset *DatasetAssetDetail      `json:"dataset_asset,omitempty"`
	Baseline     *BaselineDetail          `json:"baseline,omitempty"`
	EvalPlan     *DatasetEvalPlanDocument `json:"eval_plan,omitempty"`
	Warnings     []string                 `json:"warnings"`
}

type DatasetEvalPlanResponse struct {
	DatasetAsset *DatasetAssetDetail     `json:"dataset_asset"`
	Baseline     *BaselineDetail         `json:"baseline,omitempty"`
	EvalPlan     DatasetEvalPlanDocument `json:"eval_plan"`
}
