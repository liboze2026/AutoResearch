package model

import "time"

type DatasetAssetCreateRequest struct {
	Name              string `json:"name"`
	DescriptionMD     string `json:"descriptionMd"`
	TaskType          string `json:"taskType"`
	Status            string `json:"status"`
	SourceType        string `json:"sourceType"`
	LocalOrRemotePath string `json:"localOrRemotePath"`
	ReadmeMD          string `json:"readmeMd"`
	LoaderNoteMD      string `json:"loaderNoteMd"`
	SchemaNoteMD      string `json:"schemaNoteMd"`
}

type DatasetAssetRegisterFromScanRequest struct {
	ExistingDatasetRef string `json:"existingDatasetRef"`
	ScanRecordID       string `json:"scanRecordId"`
	Name               string `json:"name"`
	DescriptionMD      string `json:"descriptionMd"`
	TaskType           string `json:"taskType"`
	Status             string `json:"status"`
	SourceType         string `json:"sourceType"`
	ReadmeMD           string `json:"readmeMd"`
	LoaderNoteMD       string `json:"loaderNoteMd"`
	SchemaNoteMD       string `json:"schemaNoteMd"`
}

type DatasetAssetDetail struct {
	Asset   DatasetAsset         `json:"asset"`
	Sources []DatasetAssetSource `json:"sources,omitempty"`
}

type DatasetAssetWorkspaceMetadata struct {
	DatasetAssetID     string               `json:"datasetAssetId"`
	Name               string               `json:"name"`
	TaskType           string               `json:"taskType"`
	Status             string               `json:"status"`
	SourceType         string               `json:"sourceType"`
	LocalOrRemotePath  string               `json:"localOrRemotePath"`
	ExistingDatasetRef string               `json:"existingDatasetRef,omitempty"`
	Sources            []DatasetAssetSource `json:"sources,omitempty"`
	UpdatedAt          time.Time            `json:"updatedAt"`
}
