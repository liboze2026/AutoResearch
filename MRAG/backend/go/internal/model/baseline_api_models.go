package model

type BaselineCreateRequest struct {
	DatasetAssetID   string         `json:"datasetAssetId"`
	Name             string         `json:"name"`
	MetricSchemaJSON map[string]any `json:"metricSchemaJson"`
	ResultJSON       map[string]any `json:"resultJson"`
	NoteMD           string         `json:"noteMd"`
	SourceType       string         `json:"sourceType"`
}

type BaselineUpdateRequest struct {
	Name             *string        `json:"name,omitempty"`
	MetricSchemaJSON map[string]any `json:"metricSchemaJson,omitempty"`
	ResultJSON       map[string]any `json:"resultJson,omitempty"`
	NoteMD           *string        `json:"noteMd,omitempty"`
	SourceType       *string        `json:"sourceType,omitempty"`
}

type BaselineDetail struct {
	Baseline     Baseline     `json:"baseline"`
	DatasetAsset DatasetAsset `json:"datasetAsset"`
}
