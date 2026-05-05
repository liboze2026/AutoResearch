package model

import "time"

type Paper struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Abstract   string    `json:"abstract"`
	Authors    string    `json:"authors"`
	Venue      string    `json:"venue"`
	Year       int       `json:"year"`
	Status     string    `json:"status"`
	SourceType string    `json:"sourceType"`
	SourceURL  string    `json:"sourceUrl,omitempty"`
	ParseMode  string    `json:"parseMode,omitempty"`
	ParseError string    `json:"parseError,omitempty"`
	ParserNote string    `json:"parserNote,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type PaperFile struct {
	ID        string    `json:"id"`
	PaperID   string    `json:"paperId"`
	FilePath  string    `json:"filePath"`
	FileType  string    `json:"fileType"`
	Checksum  string    `json:"checksum"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PaperInsight struct {
	ID                string    `json:"id"`
	PaperID           string    `json:"paperId"`
	SummaryMD         string    `json:"summaryMd"`
	ContributionsJSON any       `json:"contributionsJson,omitempty"`
	MethodsJSON       any       `json:"methodsJson,omitempty"`
	LimitationsJSON   any       `json:"limitationsJson,omitempty"`
	NoveltyPointsJSON any       `json:"noveltyPointsJson,omitempty"`
	ExtractStatus     string    `json:"extractStatus"`
	ExtractError      string    `json:"extractError,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type Idea struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	DescriptionMD string    `json:"descriptionMd"`
	Status        string    `json:"status"`
	Weight        int       `json:"weight"`
	SourceType    string    `json:"sourceType"`
	Priority      int       `json:"priority"`
	Confidence    float64   `json:"confidence"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type IdeaSource struct {
	ID             int64     `json:"id"`
	IdeaID         string    `json:"ideaId"`
	PaperID        string    `json:"paperId,omitempty"`
	PaperInsightID string    `json:"paperInsightId,omitempty"`
	SourceNote     string    `json:"sourceNote"`
	PaperTitle     string    `json:"paperTitle,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type DatasetAsset struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	DescriptionMD       string    `json:"descriptionMd"`
	TaskType            string    `json:"taskType"`
	Status              string    `json:"status"`
	SourceType          string    `json:"sourceType"`
	LocalOrRemotePath   string    `json:"localOrRemotePath"`
	ReadmeMD            string    `json:"readmeMd,omitempty"`
	LoaderNoteMD        string    `json:"loaderNoteMd,omitempty"`
	SchemaNoteMD        string    `json:"schemaNoteMd,omitempty"`
	ExistingDatasetRef  string    `json:"existingDatasetRef,omitempty"`
	ExistingDatasetName string    `json:"existingDatasetName,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type DatasetAssetSource struct {
	ID                  int64     `json:"id"`
	DatasetAssetID      string    `json:"datasetAssetId"`
	ExistingDatasetRef  string    `json:"existingDatasetRef"`
	SourceKind          string    `json:"sourceKind"`
	ExistingDatasetName string    `json:"existingDatasetName,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type Baseline struct {
	ID               string         `json:"id"`
	DatasetAssetID   string         `json:"datasetAssetId"`
	Name             string         `json:"name"`
	MetricSchemaJSON map[string]any `json:"metricSchemaJson,omitempty"`
	ResultJSON       map[string]any `json:"resultJson,omitempty"`
	NoteMD           string         `json:"noteMd"`
	SourceType       string         `json:"sourceType"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type ResultArchive struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	DatasetAssetID string         `json:"datasetAssetId"`
	BaselineID     string         `json:"baselineId,omitempty"`
	IdeaID         string         `json:"ideaId,omitempty"`
	ServerID       string         `json:"serverId,omitempty"`
	SummaryMD      string         `json:"summaryMd"`
	MetricJSON     map[string]any `json:"metricJson,omitempty"`
	Status         string         `json:"status"`
	NoteMD         string         `json:"noteMd"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type ArchiveFile struct {
	ID        int64     `json:"id"`
	ArchiveID string    `json:"archiveId"`
	FilePath  string    `json:"filePath"`
	FileKind  string    `json:"fileKind"`
	Checksum  string    `json:"checksum"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
