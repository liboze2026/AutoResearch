package model

type ResultArchiveCreateRequest struct {
	Title          string             `json:"title"`
	DatasetAssetID string             `json:"datasetAssetId"`
	BaselineID     string             `json:"baselineId"`
	IdeaID         string             `json:"ideaId"`
	ServerID       string             `json:"serverId"`
	SummaryMD      string             `json:"summaryMd"`
	MetricJSON     map[string]any     `json:"metricJson"`
	Status         string             `json:"status"`
	NoteMD         string             `json:"noteMd"`
	Files          []ArchiveFileInput `json:"files"`
}

type ResultArchiveUpdateRequest struct {
	BaselineID *string            `json:"baselineId,omitempty"`
	Title      *string            `json:"title,omitempty"`
	IdeaID     *string            `json:"ideaId,omitempty"`
	ServerID   *string            `json:"serverId,omitempty"`
	SummaryMD  *string            `json:"summaryMd,omitempty"`
	MetricJSON map[string]any     `json:"metricJson,omitempty"`
	Status     *string            `json:"status,omitempty"`
	NoteMD     *string            `json:"noteMd,omitempty"`
	Files      []ArchiveFileInput `json:"files,omitempty"`
}

type ArchiveFileInput struct {
	FileName string `json:"fileName"`
	FileKind string `json:"fileKind"`
	Content  string `json:"content"`
}

type ResultArchiveDetail struct {
	Archive ResultArchive `json:"archive"`
	Files   []ArchiveFile `json:"files,omitempty"`
}
