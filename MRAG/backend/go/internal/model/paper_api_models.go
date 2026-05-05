package model

import "time"

type PaperImportRequest struct {
	ExistingPath string `json:"existingPath"`
}

type PaperImportResult struct {
	Paper      Paper       `json:"paper"`
	Files      []PaperFile `json:"files"`
	ParseMode  string      `json:"parseMode"`
	MockParsed bool        `json:"mockParsed"`
	ParserNote string      `json:"parserNote"`
}

type PaperDetail struct {
	Paper       Paper          `json:"paper"`
	Files       []PaperFile    `json:"files"`
	InsightList []PaperInsight `json:"insightList,omitempty"`
}

type PaperParseResult struct {
	Paper      Paper  `json:"paper"`
	ParseMode  string `json:"parseMode"`
	MockParsed bool   `json:"mockParsed"`
	ParserNote string `json:"parserNote"`
}

type PaperInsightExtractionResult struct {
	PaperID       string       `json:"paperId"`
	ExtractMode   string       `json:"extractMode"`
	MockExtracted bool         `json:"mockExtracted"`
	SummaryPath   string       `json:"summaryPath"`
	Insight       PaperInsight `json:"insight"`
}

type IdeaCreateRequest struct {
	Title         string  `json:"title"`
	DescriptionMD string  `json:"descriptionMd"`
	Status        string  `json:"status"`
	Weight        int     `json:"weight"`
	Priority      int     `json:"priority"`
	Confidence    float64 `json:"confidence"`
	SourceType    string  `json:"sourceType"`
	SourceNote    string  `json:"sourceNote"`
}

type IdeaUpdateRequest struct {
	Title         *string  `json:"title,omitempty"`
	DescriptionMD *string  `json:"descriptionMd,omitempty"`
	Status        *string  `json:"status,omitempty"`
	Weight        *int     `json:"weight,omitempty"`
	Priority      *int     `json:"priority,omitempty"`
	Confidence    *float64 `json:"confidence,omitempty"`
	SourceType    *string  `json:"sourceType,omitempty"`
}

type IdeaDetail struct {
	Idea           Idea                   `json:"idea"`
	Sources        []IdeaSource           `json:"sources,omitempty"`
	StructuredIdea *StructuredIdeaPayload `json:"structuredIdea,omitempty"`
}

type IdeaGenerationResult struct {
	PaperID string `json:"paperId"`
	Ideas   []Idea `json:"ideas"`
}

type IdeaWorkspaceMetadata struct {
	IdeaID         string                 `json:"ideaId"`
	Title          string                 `json:"title"`
	DescriptionMD  string                 `json:"descriptionMd"`
	Status         string                 `json:"status"`
	Weight         int                    `json:"weight"`
	Priority       int                    `json:"priority"`
	Confidence     float64                `json:"confidence"`
	SourceType     string                 `json:"sourceType"`
	GeneratedFrom  string                 `json:"generatedFrom,omitempty"`
	SourceSnapshot []IdeaSource           `json:"sourceSnapshot,omitempty"`
	StructuredIdea *StructuredIdeaPayload `json:"structuredIdea,omitempty"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}
