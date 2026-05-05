package agentmemory

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

type memoryStore interface {
	Upsert(context.Context, model.AgentMemoryRecord) error
	ListByAgentType(context.Context, string) ([]model.AgentMemoryRecord, error)
}

type Service struct {
	store         memoryStore
	workspaceRoot string
}

func NewService(store memoryStore, workspaceRoot string) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	paths := workspacepkg.New(workspaceRoot)
	_ = os.MkdirAll(paths.AgentMemoryDir(), 0o755)
	return &Service{store: store, workspaceRoot: workspaceRoot}
}

func (s *Service) Upsert(ctx context.Context, req model.AgentMemoryUpsertRequest) (*model.AgentMemoryRecord, error) {
	if err := validateUpsert(req); err != nil {
		return nil, err
	}
	now := time.Now()
	item := model.AgentMemoryRecord{
		ID:        httpx.NewID("mem"),
		AgentType: strings.TrimSpace(req.AgentType),
		MemoryKey: strings.TrimSpace(req.MemoryKey),
		ContentMD: req.ContentMD,
		SourceRef: strings.TrimSpace(req.SourceRef),
		UpdatedAt: now,
		CreatedAt: now,
	}
	if err := s.writeMemoryFile(item); err != nil {
		return nil, err
	}
	if err := s.store.Upsert(ctx, item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) ListByAgentType(ctx context.Context, agentType string) ([]model.AgentMemoryRecord, error) {
	if strings.TrimSpace(agentType) == "" {
		return nil, fmt.Errorf("agent type is required")
	}
	return s.store.ListByAgentType(ctx, strings.TrimSpace(agentType))
}

func (s *Service) writeMemoryFile(item model.AgentMemoryRecord) error {
	paths := workspacepkg.New(s.workspaceRoot)
	dir := paths.AgentMemoryTypeDir(item.AgentType)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	fileName := slugify(item.MemoryKey) + ".md"
	return os.WriteFile(filepath.Join(dir, fileName), []byte(item.ContentMD), 0o644)
}

func validateUpsert(req model.AgentMemoryUpsertRequest) error {
	if strings.TrimSpace(req.AgentType) == "" {
		return fmt.Errorf("agent_type is required")
	}
	if strings.TrimSpace(req.MemoryKey) == "" {
		return fmt.Errorf("memory_key is required")
	}
	if strings.TrimSpace(req.ContentMD) == "" {
		return fmt.Errorf("content_md is required")
	}
	return nil
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
		return "memory"
	}
	return out
}
