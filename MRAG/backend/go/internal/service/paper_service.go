package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type PaperStore interface {
	List(context.Context) ([]model.Paper, error)
	GetByID(context.Context, string) (*model.Paper, error)
	ListFiles(context.Context, string) ([]model.PaperFile, error)
	ListInsightsByPaper(context.Context, string) ([]model.PaperInsight, error)
	Create(context.Context, model.Paper) error
	AddFile(context.Context, model.PaperFile) error
	UpdatePaperMetadata(context.Context, model.Paper) error
	UpsertInsight(context.Context, model.PaperInsight) error
}

type PaperService struct {
	store           PaperStore
	pythonExec      string
	pythonAgentsDir string
	workspaceRoot   string
	eventPublisher  PaperEventPublisher
}

type PaperEventPublisher interface {
	PublishEvent(context.Context, model.AgentEventCreateRequest) (*model.AgentEvent, error)
}

type PaperMetadataPatch struct {
	Title      string
	Abstract   string
	Venue      string
	Year       int
	SourceType string
	SourceURL  string
	ParserNote string
}

type parsePaperScriptOutput struct {
	Status       string `json:"status"`
	Title        string `json:"title"`
	Abstract     string `json:"abstract"`
	Authors      string `json:"authors"`
	Venue        string `json:"venue"`
	Year         int    `json:"year"`
	ParseMode    string `json:"parse_mode"`
	MockParsed   bool   `json:"mock_parsed"`
	ParserNote   string `json:"parser_note"`
	MetadataPath string `json:"metadata_path"`
	ParsedPath   string `json:"parsed_path"`
}

type extractInsightsScriptOutput struct {
	Status            string   `json:"status"`
	ExtractMode       string   `json:"extract_mode"`
	MockExtracted     bool     `json:"mock_extracted"`
	SummaryPath       string   `json:"summary_path"`
	ContributionsPath string   `json:"contributions_path"`
	MethodsPath       string   `json:"methods_path"`
	LimitationsPath   string   `json:"limitations_path"`
	NoveltyPointsPath string   `json:"novelty_points_path"`
	Summary           string   `json:"summary"`
	Contributions     []string `json:"contributions"`
	Methods           []string `json:"methods"`
	Limitations       []string `json:"limitations"`
	NoveltyPoints     []string `json:"novelty_points"`
}

type InsightOutputPatch struct {
	SummaryMD     string
	Contributions any
	Methods       any
	Limitations   any
	NoveltyPoints any
	ExtractMode   string
	ExtractStatus string
	ExtractError  string
	SourceRef     string
	Focus         string
}

func NewPaperService(store PaperStore, pythonExec string, pythonAgentsDir string, workspaceRoot string) *PaperService {
	if strings.TrimSpace(pythonExec) == "" {
		pythonExec = "python"
	}
	if strings.TrimSpace(pythonAgentsDir) == "" {
		pythonAgentsDir = filepath.Join("..", "python_agents")
	}
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &PaperService{store: store, pythonExec: pythonExec, pythonAgentsDir: pythonAgentsDir, workspaceRoot: workspaceRoot}
}

func (s *PaperService) SetEventPublisher(publisher PaperEventPublisher) {
	s.eventPublisher = publisher
}

func (s *PaperService) List(ctx context.Context) ([]model.Paper, error) {
	return s.store.List(ctx)
}

func (s *PaperService) GetByID(ctx context.Context, id string) (*model.PaperDetail, error) {
	paper, err := s.store.GetByID(ctx, id)
	if err != nil || paper == nil {
		return nil, err
	}
	files, err := s.store.ListFiles(ctx, id)
	if err != nil {
		return nil, err
	}
	insights, err := s.store.ListInsightsByPaper(ctx, id)
	if err != nil {
		return nil, err
	}
	return &model.PaperDetail{Paper: *paper, Files: files, InsightList: insights}, nil
}

func (s *PaperService) ListFiles(ctx context.Context, id string) ([]model.PaperFile, error) {
	paper, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if paper == nil {
		return nil, fmt.Errorf("paper not found")
	}
	return s.store.ListFiles(ctx, id)
}

func (s *PaperService) ListInsights(ctx context.Context, id string) ([]model.PaperInsight, error) {
	paper, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if paper == nil {
		return nil, fmt.Errorf("paper not found")
	}
	return s.store.ListInsightsByPaper(ctx, id)
}

func (s *PaperService) ApplyReaderMetadata(ctx context.Context, paperID string, patch PaperMetadataPatch) (*model.Paper, error) {
	paper, err := s.store.GetByID(ctx, paperID)
	if err != nil {
		return nil, err
	}
	if paper == nil {
		return nil, fmt.Errorf("paper not found")
	}
	if strings.TrimSpace(patch.Title) != "" {
		paper.Title = strings.TrimSpace(patch.Title)
	}
	if strings.TrimSpace(patch.Abstract) != "" {
		paper.Abstract = strings.TrimSpace(patch.Abstract)
	}
	if strings.TrimSpace(patch.Venue) != "" {
		paper.Venue = strings.TrimSpace(patch.Venue)
	}
	if patch.Year > 0 {
		paper.Year = patch.Year
	}
	if strings.TrimSpace(patch.SourceType) != "" {
		paper.SourceType = strings.TrimSpace(patch.SourceType)
	}
	if strings.TrimSpace(patch.SourceURL) != "" {
		paper.SourceURL = strings.TrimSpace(patch.SourceURL)
	}
	if strings.TrimSpace(patch.ParserNote) != "" {
		if strings.TrimSpace(paper.ParserNote) == "" {
			paper.ParserNote = strings.TrimSpace(patch.ParserNote)
		} else if !strings.Contains(paper.ParserNote, strings.TrimSpace(patch.ParserNote)) {
			paper.ParserNote = strings.TrimSpace(paper.ParserNote) + " " + strings.TrimSpace(patch.ParserNote)
		}
	}
	paper.UpdatedAt = time.Now()
	if err = s.store.UpdatePaperMetadata(ctx, *paper); err != nil {
		return nil, err
	}
	return paper, nil
}

func (s *PaperService) ImportUploadedFile(ctx context.Context, fileName string, reader io.Reader) (*model.PaperImportResult, error) {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return nil, fmt.Errorf("file name is required")
	}
	paperID := httpx.NewID("paper")
	paperFileID := httpx.NewID("pfile")
	paths := workspacepkg.New(s.workspaceRoot)
	if err := os.MkdirAll(paths.PapersIncoming(), 0o755); err != nil {
		return nil, err
	}
	incomingFileName := fmt.Sprintf("%s_%s", paperID, sanitizePaperFileName(fileName))
	incomingPath := filepath.Join(paths.PapersIncoming(), incomingFileName)
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if err = os.WriteFile(incomingPath, content, 0o644); err != nil {
		return nil, err
	}
	return s.importFromPath(ctx, paperID, paperFileID, incomingPath, fileName, "upload")
}

func (s *PaperService) ImportExistingFile(ctx context.Context, existingPath string) (*model.PaperImportResult, error) {
	existingPath = strings.TrimSpace(existingPath)
	if existingPath == "" {
		return nil, fmt.Errorf("existingPath is required")
	}
	paperID := httpx.NewID("paper")
	paperFileID := httpx.NewID("pfile")
	paths := workspacepkg.New(s.workspaceRoot)
	cleanBase := filepath.Clean(paths.PapersIncoming())
	cleanPath := filepath.Clean(existingPath)
	if !filepath.IsAbs(cleanPath) {
		rel := filepath.ToSlash(cleanPath)
		rootPrefix := strings.TrimSuffix(filepath.ToSlash(s.workspaceRoot), "/") + "/"
		if strings.HasPrefix(rel, rootPrefix) {
			cleanPath = filepath.Clean(cleanPath)
		} else {
			cleanPath = filepath.Clean(filepath.Join(s.workspaceRoot, cleanPath))
		}
	}
	absBase, _ := filepath.Abs(cleanBase)
	absPath, _ := filepath.Abs(cleanPath)
	if !strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(absBase)) {
		return nil, fmt.Errorf("existingPath must be under workspace/papers/incoming")
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("existingPath must be a file")
	}
	return s.importFromPath(ctx, paperID, paperFileID, absPath, filepath.Base(absPath), "workspace")
}

func (s *PaperService) ParsePaper(ctx context.Context, paperID string) (*model.PaperParseResult, error) {
	paper, err := s.store.GetByID(ctx, paperID)
	if err != nil {
		return nil, err
	}
	if paper == nil {
		return nil, fmt.Errorf("paper not found")
	}
	files, err := s.store.ListFiles(ctx, paperID)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("paper file not found")
	}
	primary := files[0]
	output, err := s.runParsePaperScript(ctx, paperID, primary.FilePath)
	if err != nil {
		paper.ParseError = err.Error()
		paper.Status = "imported"
		paper.UpdatedAt = time.Now()
		_ = s.store.UpdatePaperMetadata(ctx, *paper)
		return nil, err
	}
	paper.Title = output.Title
	paper.Abstract = output.Abstract
	paper.Authors = output.Authors
	paper.Venue = output.Venue
	paper.Year = output.Year
	paper.Status = "parsed"
	paper.ParseMode = output.ParseMode
	paper.ParseError = ""
	paper.ParserNote = output.ParserNote
	paper.UpdatedAt = time.Now()
	if err = s.store.UpdatePaperMetadata(ctx, *paper); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "paper_parsed",
		SourceRef: "paper:" + paperID,
		InputRefs: []model.AgentInputRef{
			{RefType: "paper", RefID: paperID, RefPath: primary.FilePath},
			{RefType: "parsed_content", RefID: paperID, RefPath: output.ParsedPath},
		},
		Payload: map[string]any{
			"paper_id":           paperID,
			"parse_mode":         output.ParseMode,
			"paper_title":        paper.Title,
			"parsed_content_ref": output.ParsedPath,
		},
	})
	return &model.PaperParseResult{Paper: *paper, ParseMode: output.ParseMode, MockParsed: output.MockParsed, ParserNote: output.ParserNote}, nil
}

func (s *PaperService) ExtractInsights(ctx context.Context, paperID string) (*model.PaperInsightExtractionResult, error) {
	paper, err := s.store.GetByID(ctx, paperID)
	if err != nil {
		return nil, err
	}
	if paper == nil {
		return nil, fmt.Errorf("paper not found")
	}
	parsedPath := filepath.Join(workspacepkg.New(s.workspaceRoot).PapersParsed(), paperID, "parsed.md")
	if _, err = os.Stat(parsedPath); err != nil {
		return nil, fmt.Errorf("parsed paper not found, run parse first")
	}
	output, err := s.runExtractInsightsScript(ctx, paperID, parsedPath)
	if err != nil {
		insight := model.PaperInsight{
			ID:            httpx.NewID("pinsight"),
			PaperID:       paperID,
			SummaryMD:     "",
			ExtractStatus: "failed",
			ExtractError:  err.Error(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		_ = s.store.UpsertInsight(ctx, insight)
		return nil, err
	}
	insight, _, err := s.ApplyInsightAgentOutput(ctx, paperID, parsedPath, InsightOutputPatch{
		SummaryMD:     output.Summary,
		Contributions: output.Contributions,
		Methods:       output.Methods,
		Limitations:   output.Limitations,
		NoveltyPoints: output.NoveltyPoints,
		ExtractMode:   output.ExtractMode,
		ExtractStatus: "completed",
		ExtractError:  "",
		SourceRef:     "legacy_extract_insights",
	})
	if err != nil {
		return nil, err
	}
	return &model.PaperInsightExtractionResult{PaperID: paperID, ExtractMode: output.ExtractMode, MockExtracted: output.MockExtracted, SummaryPath: output.SummaryPath, Insight: insight}, nil
}

func (s *PaperService) ApplyInsightAgentOutput(ctx context.Context, paperID string, parsedPath string, patch InsightOutputPatch) (model.PaperInsight, string, error) {
	paper, err := s.store.GetByID(ctx, paperID)
	if err != nil {
		return model.PaperInsight{}, "", err
	}
	if paper == nil {
		return model.PaperInsight{}, "", fmt.Errorf("paper not found")
	}
	if strings.TrimSpace(parsedPath) == "" {
		parsedPath = filepath.Join(workspacepkg.New(s.workspaceRoot).PapersParsed(), paperID, "parsed.md")
	}
	insightDir := filepath.Join(workspacepkg.New(s.workspaceRoot).PapersInsights(), paperID)
	if err = os.MkdirAll(insightDir, 0o755); err != nil {
		return model.PaperInsight{}, "", err
	}
	summaryPath := filepath.Join(insightDir, "summary.md")
	contributionsPath := filepath.Join(insightDir, "contributions.json")
	methodsPath := filepath.Join(insightDir, "methods.json")
	limitationsPath := filepath.Join(insightDir, "limitations.json")
	noveltyPointsPath := filepath.Join(insightDir, "novelty_points.json")
	metadataPath := filepath.Join(insightDir, "insight_agent_output.json")

	summaryMD := strings.TrimSpace(patch.SummaryMD)
	if summaryMD == "" {
		summaryMD = fmt.Sprintf("Insight summary for %s.", firstNonEmpty(paper.Title, paperID))
	}
	contributions := normalizeJSONArray(patch.Contributions)
	methods := normalizeJSONArray(patch.Methods)
	limitations := normalizeJSONArray(patch.Limitations)
	noveltyPoints := normalizeJSONArray(patch.NoveltyPoints)
	extractStatus := firstNonEmpty(strings.TrimSpace(patch.ExtractStatus), "completed")

	if err = os.WriteFile(summaryPath, []byte(summaryMD+"\n"), 0o644); err != nil {
		return model.PaperInsight{}, "", err
	}
	if err = writeJSONFile(contributionsPath, contributions); err != nil {
		return model.PaperInsight{}, "", err
	}
	if err = writeJSONFile(methodsPath, methods); err != nil {
		return model.PaperInsight{}, "", err
	}
	if err = writeJSONFile(limitationsPath, limitations); err != nil {
		return model.PaperInsight{}, "", err
	}
	if err = writeJSONFile(noveltyPointsPath, noveltyPoints); err != nil {
		return model.PaperInsight{}, "", err
	}
	if err = writeJSONFile(metadataPath, map[string]any{
		"paper_id":           paperID,
		"parsed_content_ref": parsedPath,
		"summary_md":         summaryMD,
		"contributions_json": contributions,
		"methods_json":       methods,
		"limitations_json":   limitations,
		"novelty_points":     noveltyPoints,
		"extract_mode":       strings.TrimSpace(patch.ExtractMode),
		"extract_status":     extractStatus,
		"extract_error":      strings.TrimSpace(patch.ExtractError),
		"focus":              strings.TrimSpace(patch.Focus),
		"source_ref":         strings.TrimSpace(patch.SourceRef),
	}); err != nil {
		return model.PaperInsight{}, "", err
	}

	insight := model.PaperInsight{
		ID:                httpx.NewID("pinsight"),
		PaperID:           paperID,
		SummaryMD:         summaryMD,
		ContributionsJSON: contributions,
		MethodsJSON:       methods,
		LimitationsJSON:   limitations,
		NoveltyPointsJSON: noveltyPoints,
		ExtractStatus:     extractStatus,
		ExtractError:      strings.TrimSpace(patch.ExtractError),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	if err = s.store.UpsertInsight(ctx, insight); err != nil {
		return model.PaperInsight{}, "", err
	}
	paper.Status = "insight_extracted"
	paper.UpdatedAt = time.Now()
	if err = s.store.UpdatePaperMetadata(ctx, *paper); err != nil {
		return model.PaperInsight{}, "", err
	}
	s.publishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "insights_ready",
		SourceRef: "paper:" + paperID,
		InputRefs: []model.AgentInputRef{
			{RefType: "paper", RefID: paperID, RefPath: parsedPath},
			{RefType: "parsed_content", RefID: paperID, RefPath: parsedPath},
			{RefType: "insight", RefID: insight.ID, RefPath: metadataPath},
		},
		Payload: map[string]any{
			"paper_id":            paperID,
			"insight_id":          insight.ID,
			"extract_mode":        strings.TrimSpace(patch.ExtractMode),
			"focus":               strings.TrimSpace(patch.Focus),
			"parsed_content_ref":  parsedPath,
			"novelty_point_count": len(noveltyPoints),
		},
	})
	return insight, summaryPath, nil
}

func (s *PaperService) importFromPath(ctx context.Context, paperID string, paperFileID string, storedPath string, originalFileName string, sourceType string) (*model.PaperImportResult, error) {
	now := time.Now()
	checksum, err := checksumFile(storedPath)
	if err != nil {
		return nil, err
	}
	paper := model.Paper{
		ID:         paperID,
		Title:      humanizePaperTitle(strings.TrimSuffix(filepath.Base(originalFileName), filepath.Ext(originalFileName))),
		Abstract:   "",
		Authors:    "",
		Venue:      "",
		Year:       0,
		Status:     "imported",
		SourceType: sourceType,
		SourceURL:  "",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	file := model.PaperFile{
		ID:        paperFileID,
		PaperID:   paperID,
		FilePath:  storedPath,
		FileType:  detectPaperFileType(originalFileName),
		Checksum:  checksum,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err = s.store.Create(ctx, paper); err != nil {
		return nil, err
	}
	if err = s.store.AddFile(ctx, file); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "paper_imported",
		SourceRef: "paper:" + paperID,
		InputRefs: []model.AgentInputRef{
			{RefType: "paper", RefID: paperID, RefPath: storedPath},
		},
		Payload: map[string]any{
			"paper_id":    paperID,
			"file_path":   storedPath,
			"source_type": sourceType,
		},
	})
	parseResult, err := s.ParsePaper(ctx, paperID)
	if err != nil {
		return nil, err
	}
	return &model.PaperImportResult{Paper: parseResult.Paper, Files: []model.PaperFile{file}, ParseMode: parseResult.ParseMode, MockParsed: parseResult.MockParsed, ParserNote: parseResult.ParserNote}, nil
}

func (s *PaperService) publishEvent(ctx context.Context, req model.AgentEventCreateRequest) {
	if s.eventPublisher == nil {
		return
	}
	_, _ = s.eventPublisher.PublishEvent(ctx, req)
}

func (s *PaperService) runParsePaperScript(ctx context.Context, paperID string, paperFilePath string) (*parsePaperScriptOutput, error) {
	paths := workspacepkg.New(s.workspaceRoot)
	outputDir := filepath.Join(paths.PapersParsed(), paperID)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	scriptPath, err := filepath.Abs(filepath.Join(s.pythonAgentsDir, "parse_paper.py"))
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, s.pythonExec, scriptPath, "--paper-file", paperFilePath, "--output-dir", outputDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("parse_paper.py failed: %s", strings.TrimSpace(string(out)))
	}
	var result parsePaperScriptOutput
	if err = json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parse_paper.py output parse failed: %w", err)
	}
	return &result, nil
}

func (s *PaperService) runExtractInsightsScript(ctx context.Context, paperID string, parsedPath string) (*extractInsightsScriptOutput, error) {
	paths := workspacepkg.New(s.workspaceRoot)
	outputDir := filepath.Join(paths.PapersInsights(), paperID)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	scriptPath, err := filepath.Abs(filepath.Join(s.pythonAgentsDir, "extract_insights.py"))
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, s.pythonExec, scriptPath, "--parsed-paper", parsedPath, "--output-dir", outputDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("extract_insights.py failed: %s", strings.TrimSpace(string(out)))
	}
	var result extractInsightsScriptOutput
	if err = json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("extract_insights.py output parse failed: %w", err)
	}
	return &result, nil
}

func checksumFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func detectPaperFileType(fileName string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if ext == "" {
		return "unknown"
	}
	return ext
}

func sanitizePaperFileName(fileName string) string {
	fileName = strings.TrimSpace(filepath.Base(fileName))
	fileName = strings.ReplaceAll(fileName, " ", "_")
	if fileName == "" {
		return "paper.bin"
	}
	return fileName
}

func humanizePaperTitle(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "Untitled Paper"
	}
	raw = strings.ReplaceAll(raw, "_", " ")
	raw = strings.ReplaceAll(raw, "-", " ")
	parts := strings.Fields(raw)
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	return strings.Join(parts, " ")
}

func normalizeJSONArray(value any) []string {
	switch typed := value.(type) {
	case nil:
		return []string{}
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" && text != "<nil>" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := strings.TrimSpace(fmt.Sprint(typed))
		if text == "" || text == "<nil>" {
			return []string{}
		}
		return []string{text}
	}
}

func writeJSONFile(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
