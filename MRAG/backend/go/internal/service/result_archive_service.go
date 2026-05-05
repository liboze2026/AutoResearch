package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type ResultArchiveStore interface {
	List(context.Context) ([]model.ResultArchive, error)
	GetByID(context.Context, string) (*model.ResultArchive, error)
	Create(context.Context, model.ResultArchive) error
	Update(context.Context, model.ResultArchive) error
	AddFile(context.Context, model.ArchiveFile) error
	ListFiles(context.Context, string) ([]model.ArchiveFile, error)
}

type ResultArchiveDatasetAssetReader interface {
	GetByID(context.Context, string) (*model.DatasetAsset, error)
}

type ResultArchiveIdeaReader interface {
	GetByID(context.Context, string) (*model.Idea, error)
}

type ResultArchiveService struct {
	store         ResultArchiveStore
	assetReader   ResultArchiveDatasetAssetReader
	ideaReader    ResultArchiveIdeaReader
	workspaceRoot string
}

func NewResultArchiveService(store ResultArchiveStore, assetReader ResultArchiveDatasetAssetReader, ideaReader ResultArchiveIdeaReader, workspaceRoot string) *ResultArchiveService {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &ResultArchiveService{store: store, assetReader: assetReader, ideaReader: ideaReader, workspaceRoot: workspaceRoot}
}

func (s *ResultArchiveService) List(ctx context.Context) ([]model.ResultArchive, error) {
	return s.store.List(ctx)
}

func (s *ResultArchiveService) GetByID(ctx context.Context, id string) (*model.ResultArchiveDetail, error) {
	item, err := s.store.GetByID(ctx, id)
	if err != nil || item == nil {
		return nil, err
	}
	files, err := s.store.ListFiles(ctx, id)
	if err != nil {
		return nil, err
	}
	return &model.ResultArchiveDetail{Archive: *item, Files: files}, nil
}

func (s *ResultArchiveService) Create(ctx context.Context, req model.ResultArchiveCreateRequest) (*model.ResultArchiveDetail, error) {
	asset, err := s.assetReader.GetByID(ctx, strings.TrimSpace(req.DatasetAssetID))
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, fmt.Errorf("dataset asset not found")
	}
	ideaID := strings.TrimSpace(req.IdeaID)
	if ideaID != "" && s.ideaReader != nil {
		idea, err := s.ideaReader.GetByID(ctx, ideaID)
		if err != nil {
			return nil, err
		}
		if idea == nil {
			return nil, fmt.Errorf("idea not found")
		}
	}
	now := time.Now()
	archive := model.ResultArchive{
		ID:             httpx.NewID("archive"),
		Title:          strings.TrimSpace(req.Title),
		DatasetAssetID: asset.ID,
		BaselineID:     strings.TrimSpace(req.BaselineID),
		IdeaID:         ideaID,
		ServerID:       strings.TrimSpace(req.ServerID),
		SummaryMD:      strings.TrimSpace(req.SummaryMD),
		MetricJSON:     ensureArchiveMap(req.MetricJSON),
		Status:         normalizeResultArchiveStatus(req.Status),
		NoteMD:         strings.TrimSpace(req.NoteMD),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := validateResultArchive(archive); err != nil {
		return nil, err
	}
	if err := s.store.Create(ctx, archive); err != nil {
		return nil, err
	}
	files, err := s.persistArchiveFiles(ctx, archive, req.Files, true)
	if err != nil {
		return nil, err
	}
	return &model.ResultArchiveDetail{Archive: archive, Files: files}, nil
}

func (s *ResultArchiveService) Update(ctx context.Context, id string, req model.ResultArchiveUpdateRequest) (*model.ResultArchiveDetail, error) {
	archive, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if archive == nil {
		return nil, fmt.Errorf("result archive not found")
	}
	if req.Title != nil {
		archive.Title = strings.TrimSpace(*req.Title)
	}
	if req.BaselineID != nil {
		archive.BaselineID = strings.TrimSpace(*req.BaselineID)
	}
	if req.IdeaID != nil {
		archive.IdeaID = strings.TrimSpace(*req.IdeaID)
		if archive.IdeaID != "" && s.ideaReader != nil {
			idea, err := s.ideaReader.GetByID(ctx, archive.IdeaID)
			if err != nil {
				return nil, err
			}
			if idea == nil {
				return nil, fmt.Errorf("idea not found")
			}
		}
	}
	if req.ServerID != nil {
		archive.ServerID = strings.TrimSpace(*req.ServerID)
	}
	if req.SummaryMD != nil {
		archive.SummaryMD = strings.TrimSpace(*req.SummaryMD)
	}
	if req.MetricJSON != nil {
		archive.MetricJSON = ensureArchiveMap(req.MetricJSON)
	}
	if req.Status != nil {
		archive.Status = normalizeResultArchiveStatus(*req.Status)
	}
	if req.NoteMD != nil {
		archive.NoteMD = strings.TrimSpace(*req.NoteMD)
	}
	archive.UpdatedAt = time.Now()
	if err := validateResultArchive(*archive); err != nil {
		return nil, err
	}
	if err := s.store.Update(ctx, *archive); err != nil {
		return nil, err
	}
	files, err := s.persistArchiveFiles(ctx, *archive, req.Files, false)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		files, err = s.store.ListFiles(ctx, archive.ID)
		if err != nil {
			return nil, err
		}
	}
	return &model.ResultArchiveDetail{Archive: *archive, Files: files}, nil
}

func (s *ResultArchiveService) persistArchiveFiles(ctx context.Context, archive model.ResultArchive, extraFiles []model.ArchiveFileInput, createRecords bool) ([]model.ArchiveFile, error) {
	paths := workspacepkg.New(s.workspaceRoot)
	archiveDir := paths.ResultArchiveDir(archive.ID)
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return nil, err
	}
	created := make([]model.ArchiveFile, 0)
	coreFiles := []struct {
		name    string
		kind    string
		content []byte
	}{
		{name: "result.md", kind: "result_md", content: []byte(ensureTrailingLine(firstNonEmpty(archive.SummaryMD, "No summary yet.")))},
		{name: "metrics.json", kind: "metrics_json", content: mustJSON(ensureArchiveMap(archive.MetricJSON))},
		{name: "note.md", kind: "note_md", content: []byte(ensureTrailingLine(firstNonEmpty(archive.NoteMD, "No note yet.")))},
	}
	for _, file := range coreFiles {
		filePath := filepath.Join(archiveDir, file.name)
		if err := os.WriteFile(filePath, file.content, 0o644); err != nil {
			return nil, err
		}
		if createRecords {
			rec := buildArchiveFileRecord(archive.ID, filePath, file.kind, file.content)
			if err := s.store.AddFile(ctx, rec); err != nil {
				return nil, err
			}
			created = append(created, rec)
		}
	}
	for _, extra := range extraFiles {
		fileName := sanitizeArchiveFileName(extra.FileName)
		fileKind := normalizeArchiveFileKind(extra.FileKind)
		if strings.TrimSpace(extra.Content) == "" {
			continue
		}
		filePath := filepath.Join(archiveDir, fileName)
		content := []byte(extra.Content)
		if err := os.WriteFile(filePath, content, 0o644); err != nil {
			return nil, err
		}
		if createRecords {
			rec := buildArchiveFileRecord(archive.ID, filePath, fileKind, content)
			if err := s.store.AddFile(ctx, rec); err != nil {
				return nil, err
			}
			created = append(created, rec)
		}
	}
	if createRecords {
		return created, nil
	}
	return s.store.ListFiles(ctx, archive.ID)
}

func buildArchiveFileRecord(archiveID string, filePath string, fileKind string, content []byte) model.ArchiveFile {
	now := time.Now()
	sum := sha256.Sum256(content)
	return model.ArchiveFile{
		ArchiveID: archiveID,
		FilePath:  filePath,
		FileKind:  fileKind,
		Checksum:  hex.EncodeToString(sum[:]),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func validateResultArchive(item model.ResultArchive) error {
	if strings.TrimSpace(item.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(item.DatasetAssetID) == "" {
		return fmt.Errorf("datasetAssetId is required")
	}
	switch item.Status {
	case "draft", "archived", "reviewed":
	default:
		return fmt.Errorf("status must be one of draft, archived, reviewed")
	}
	return nil
}

func normalizeResultArchiveStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "archived"
	}
	return status
}

func normalizeArchiveFileKind(kind string) string {
	kind = strings.TrimSpace(strings.ToLower(kind))
	if kind == "" {
		return "attachment"
	}
	return kind
}

func sanitizeArchiveFileName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	name = strings.ReplaceAll(name, " ", "_")
	if name == "" {
		return "attachment.txt"
	}
	return name
}

func ensureArchiveMap(raw map[string]any) map[string]any {
	if raw == nil {
		return map[string]any{}
	}
	copyValue := make(map[string]any, len(raw))
	for key, value := range raw {
		copyValue[key] = value
	}
	return copyValue
}

func mustJSON(value map[string]any) []byte {
	data, _ := json.MarshalIndent(value, "", "  ")
	return data
}

func ensureTrailingLine(value string) string {
	if strings.HasSuffix(value, "\n") {
		return value
	}
	return value + "\n"
}
