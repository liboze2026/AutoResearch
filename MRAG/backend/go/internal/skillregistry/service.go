package skillregistry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type skillStore interface {
	Create(context.Context, model.SkillDefinition) error
	List(context.Context) ([]model.SkillDefinition, error)
}

type Service struct {
	store         skillStore
	workspaceRoot string
}

func NewService(store skillStore, workspaceRoot string) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	paths := workspacepkg.New(workspaceRoot)
	_ = os.MkdirAll(paths.SkillsRoot(), 0o755)
	return &Service{store: store, workspaceRoot: workspaceRoot}
}

func (s *Service) Register(ctx context.Context, req model.SkillRegisterRequest) (*model.SkillDefinition, error) {
	if err := validateRegister(req); err != nil {
		return nil, err
	}
	skillID := httpx.NewID("skill")
	skillDir, entrypoint, err := s.resolveSkillFiles(skillID, req)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	item := model.SkillDefinition{
		SkillID:      skillID,
		Name:         strings.TrimSpace(req.Name),
		Description:  strings.TrimSpace(req.Description),
		SkillDir:     skillDir,
		Entrypoint:   entrypoint,
		Dependencies: sanitizeStrings(req.Dependencies),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err = s.store.Create(ctx, item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) List(ctx context.Context) ([]model.SkillDefinition, error) {
	return s.store.List(ctx)
}

func (s *Service) resolveSkillFiles(skillID string, req model.SkillRegisterRequest) (string, string, error) {
	paths := workspacepkg.New(s.workspaceRoot)
	entrypoint := strings.TrimSpace(req.Entrypoint)
	if entrypoint == "" {
		entrypoint = "SKILL.md"
	}
	if strings.TrimSpace(req.EntryContent) != "" {
		dirName := slugify(req.Name)
		dir := paths.SkillDir(dirName)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", "", err
		}
		path := filepath.Join(dir, filepath.Base(entrypoint))
		if err := os.WriteFile(path, []byte(req.EntryContent), 0o644); err != nil {
			return "", "", err
		}
		return dir, path, nil
	}

	if strings.TrimSpace(req.SkillDir) == "" {
		return "", "", fmt.Errorf("skill_dir or entry_content is required")
	}
	dir, err := ensurePathWithin(paths.SkillsRoot(), req.SkillDir)
	if err != nil {
		return "", "", err
	}
	path := filepath.Join(dir, filepath.Base(entrypoint))
	if _, err = os.Stat(path); err != nil {
		return "", "", err
	}
	_ = skillID
	return dir, path, nil
}

func validateRegister(req model.SkillRegisterRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if strings.TrimSpace(req.SkillDir) == "" && strings.TrimSpace(req.EntryContent) == "" {
		return fmt.Errorf("skill_dir or entry_content is required")
	}
	return nil
}

func sanitizeStrings(items []string) []string {
	if items == nil {
		return []string{}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func ensurePathWithin(root string, candidate string) (string, error) {
	base, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(candidate)
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, absPath)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path must stay within %s", base)
	}
	if _, err = os.Stat(absPath); err != nil {
		return "", err
	}
	return absPath, nil
}

func slugify(value string) string {
	builder := strings.Builder{}
	lastUnderscore := false
	for _, ch := range strings.ToLower(strings.TrimSpace(value)) {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			builder.WriteRune(ch)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return "skill"
	}
	return out
}
