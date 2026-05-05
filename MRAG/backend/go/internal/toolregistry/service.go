package toolregistry

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type toolStore interface {
	Create(context.Context, model.ToolDefinition) error
	List(context.Context) ([]model.ToolDefinition, error)
}

type Service struct {
	store         toolStore
	workspaceRoot string
	pythonExec    string
}

func NewService(store toolStore, workspaceRoot string, pythonExec string) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	if strings.TrimSpace(pythonExec) == "" {
		pythonExec = "python"
	}
	paths := workspacepkg.New(workspaceRoot)
	_ = os.MkdirAll(paths.ToolsRoot(), 0o755)
	_ = os.MkdirAll(paths.GeneratedToolsRoot(), 0o755)
	return &Service{store: store, workspaceRoot: workspaceRoot, pythonExec: pythonExec}
}

func (s *Service) Register(ctx context.Context, req model.ToolRegisterRequest) (*model.ToolDefinition, error) {
	if err := validateRegister(req); err != nil {
		return nil, err
	}
	toolID := httpx.NewID("tool")
	toolPath, err := s.resolveToolPath(toolID, req)
	if err != nil {
		return nil, err
	}

	testStatus, err := s.runMinimalTest(ctx, toolPath)
	if err != nil {
		testStatus = "failed"
	}

	now := time.Now()
	item := model.ToolDefinition{
		ToolID:         toolID,
		Name:           strings.TrimSpace(req.Name),
		OwnerAgentType: strings.TrimSpace(req.OwnerAgentType),
		Path:           toolPath,
		Description:    strings.TrimSpace(req.Description),
		UsageMD:        strings.TrimSpace(req.UsageMD),
		InputSchema:    cloneMap(req.InputSchema),
		OutputSchema:   cloneMap(req.OutputSchema),
		TestStatus:     testStatus,
		Version:        firstNonEmpty(strings.TrimSpace(req.Version), "v1"),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err = s.store.Create(ctx, item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) List(ctx context.Context) ([]model.ToolDefinition, error) {
	return s.store.List(ctx)
}

func (s *Service) resolveToolPath(toolID string, req model.ToolRegisterRequest) (string, error) {
	paths := workspacepkg.New(s.workspaceRoot)
	if strings.TrimSpace(req.ScriptContent) != "" {
		scriptName := strings.TrimSpace(req.ScriptName)
		if scriptName == "" {
			scriptName = slugify(req.Name) + ".py"
		}
		if filepath.Ext(scriptName) == "" {
			scriptName += ".py"
		}
		dir := paths.GeneratedToolDir(toolID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		path := filepath.Join(dir, filepath.Base(scriptName))
		if err := os.WriteFile(path, []byte(req.ScriptContent), 0o644); err != nil {
			return "", err
		}
		return path, nil
	}

	if strings.TrimSpace(req.Path) == "" {
		return "", fmt.Errorf("path or script_content is required")
	}
	return ensurePathWithin(paths.ToolsRoot(), req.Path)
}

func (s *Service) runMinimalTest(ctx context.Context, path string) (string, error) {
	if strings.EqualFold(filepath.Ext(path), ".py") {
		cmd := exec.CommandContext(ctx, s.pythonExec, "-m", "py_compile", path)
		if output, err := cmd.CombinedOutput(); err != nil {
			return "failed", fmt.Errorf("python tool test failed: %s", strings.TrimSpace(string(output)))
		}
		return "passed", nil
	}
	if _, err := os.Stat(path); err != nil {
		return "failed", err
	}
	return "passed", nil
}

func validateRegister(req model.ToolRegisterRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.OwnerAgentType) == "" {
		return fmt.Errorf("owner_agent_type is required")
	}
	if strings.TrimSpace(req.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if strings.TrimSpace(req.UsageMD) == "" {
		return fmt.Errorf("usage_md is required")
	}
	if len(req.InputSchema) == 0 {
		return fmt.Errorf("input_schema is required")
	}
	if len(req.OutputSchema) == 0 {
		return fmt.Errorf("output_schema is required")
	}
	if strings.TrimSpace(req.Path) == "" && strings.TrimSpace(req.ScriptContent) == "" {
		return fmt.Errorf("path or script_content is required")
	}
	return nil
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
		return "tool"
	}
	return out
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	copyValue := make(map[string]any, len(value))
	for key, item := range value {
		copyValue[key] = item
	}
	return copyValue
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
