package model

type Phase4WriterRunRequest struct {
	RunManifestID string   `json:"runManifestId" binding:"required"`
	UserNotes     string   `json:"userNotes"`
	ExecutionMode string   `json:"executionMode"`
	ModelProvider string   `json:"modelProvider"`
	ModelName     string   `json:"modelName"`
	PromptVersion string   `json:"promptVersion"`
	SkillRefs     []string `json:"skillRefs"`
	ToolRefs      []string `json:"toolRefs"`
	MemoryRefs    []string `json:"memoryRefs"`
}

type Phase4WriterRuntimePayload struct {
	Summary               string         `json:"summary"`
	ReportTitle           string         `json:"report_title"`
	MachineReadableReport map[string]any `json:"machine_readable_report"`
	HumanReadableReportMD string         `json:"human_readable_report_md"`
	CitationRefs          []string       `json:"citation_refs"`
	ReferenceSourceIDs    []string       `json:"reference_source_ids"`
}

type Phase4WriterRunResult struct {
	Job      *AgentJob                     `json:"job"`
	Report   *Phase4StructuredReportRecord `json:"report,omitempty"`
	Warnings []string                      `json:"warnings"`
}

type Phase4WriterJobDetail struct {
	Job       *AgentJob                     `json:"job"`
	Artifacts []AgentArtifact               `json:"artifacts"`
	Report    *Phase4StructuredReportRecord `json:"report,omitempty"`
	Warnings  []string                      `json:"warnings"`
}
