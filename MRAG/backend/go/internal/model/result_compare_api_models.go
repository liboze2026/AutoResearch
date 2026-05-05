package model

type RunCompareResult struct {
	Run             ExperimentRun        `json:"run"`
	Comparisons     []ResultComparison   `json:"comparisons"`
	ResultArchive   *ResultArchiveDetail `json:"resultArchive,omitempty"`
	WorkspaceDir    string               `json:"workspaceDir"`
	OverallJudgment string               `json:"overallJudgment"`
}
