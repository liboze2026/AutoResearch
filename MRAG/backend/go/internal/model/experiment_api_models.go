package model

import "time"

type ExperimentCreateRequest struct {
	DatasetAssetID string `json:"datasetAssetId"`
	IdeaID         string `json:"ideaId"`
	BaselineID     string `json:"baselineId"`
	Title          string `json:"title"`
	Priority       int    `json:"priority"`
	SummaryMD      string `json:"summaryMd"`
	OwnerNoteMD    string `json:"ownerNoteMd"`
}

type ExperimentDetail struct {
	Experiment   Experiment      `json:"experiment"`
	DatasetAsset DatasetAsset    `json:"datasetAsset"`
	Idea         *Idea           `json:"idea,omitempty"`
	Baseline     *Baseline       `json:"baseline,omitempty"`
	LatestSpec   *ExperimentSpec `json:"latestSpec,omitempty"`
}

type ExperimentSpecDetail struct {
	Spec            ExperimentSpec `json:"spec"`
	WorkspacePath   string         `json:"workspacePath"`
	GeneratorSource string         `json:"generatorSource"`
}

type ExperimentWorkspaceMetadata struct {
	ExperimentID   string    `json:"experimentId"`
	DatasetAssetID string    `json:"datasetAssetId"`
	IdeaID         string    `json:"ideaId,omitempty"`
	BaselineID     string    `json:"baselineId,omitempty"`
	Title          string    `json:"title"`
	Status         string    `json:"status"`
	Priority       int       `json:"priority"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
