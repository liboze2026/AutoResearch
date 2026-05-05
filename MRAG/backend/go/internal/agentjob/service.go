package agentjob

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type jobStore interface {
	Create(context.Context, model.AgentJob) error
	GetByID(context.Context, string) (*model.AgentJob, error)
	Update(context.Context, model.AgentJob) error
}

type Service struct {
	store         jobStore
	workspaceRoot string
}

func NewService(store jobStore, workspaceRoot string) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{store: store, workspaceRoot: workspaceRoot}
}

func (s *Service) Create(ctx context.Context, req model.AgentJobCreateRequest) (*model.AgentJob, error) {
	if err := validateCreate(req); err != nil {
		return nil, err
	}
	now := time.Now()
	jobID := httpx.NewID("ajob")
	workspaceDir := strings.TrimSpace(req.WorkspaceDir)
	if workspaceDir == "" {
		workspaceDir = workspacepkg.New(s.workspaceRoot).AgentJobDir(jobID)
	}
	item := model.AgentJob{
		ID:                jobID,
		AgentType:         strings.TrimSpace(req.AgentType),
		ExecutionMode:     strings.TrimSpace(req.ExecutionMode),
		ModelProvider:     strings.TrimSpace(req.ModelProvider),
		ModelName:         strings.TrimSpace(req.ModelName),
		PromptVersion:     strings.TrimSpace(req.PromptVersion),
		InputRefs:         ensureInputRefs(req.InputRefs),
		OutputSchemaRef:   strings.TrimSpace(req.OutputSchemaRef),
		SkillRefs:         ensureStringSlice(req.SkillRefs),
		ToolRefs:          ensureStringSlice(req.ToolRefs),
		MemoryRefs:        ensureStringSlice(req.MemoryRefs),
		WorkspaceDir:      workspaceDir,
		Metadata:          ensureMap(req.Metadata),
		TriggerEventID:    strings.TrimSpace(req.TriggerEventID),
		DedupKey:          strings.TrimSpace(req.DedupKey),
		RetryCount:        0,
		MaxRetries:        normalizeMaxRetries(req.MaxRetries),
		ConcurrencyLimit:  normalizeConcurrencyLimit(req.ConcurrencyLimit),
		Status:            normalizeJobStatus(req.Status),
		NormalizedPayload: map[string]any{},
		ArtifactManifest:  []model.AgentArtifactManifestItem{},
		RepairActions:     []model.AgentRepairAction{},
		ToolUsages:        []model.AgentToolUsage{},
		Warnings:          []string{},
		ValidationStatus:  "pending",
		RepairStatus:      "pending",
		ValidationErrors:  []string{},
		ErrorMessage:      "",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.store.Create(ctx, item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*model.AgentJob, error) {
	return s.store.GetByID(ctx, id)
}

func (s *Service) GetStatus(ctx context.Context, id string) (*model.AgentJobStatusDetail, error) {
	item, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return &model.AgentJobStatusDetail{
		ID:               item.ID,
		AgentType:        item.AgentType,
		Status:           item.Status,
		ValidationStatus: item.ValidationStatus,
		RepairStatus:     item.RepairStatus,
		ValidationErrors: ensureStringSlice(item.ValidationErrors),
		RepairCount:      len(item.RepairActions),
		StartedAt:        item.StartedAt,
		CompletedAt:      item.CompletedAt,
		Warnings:         ensureStringSlice(item.Warnings),
		ErrorMessage:     item.ErrorMessage,
	}, nil
}

func validateCreate(req model.AgentJobCreateRequest) error {
	if strings.TrimSpace(req.AgentType) == "" {
		return fmt.Errorf("agent_type is required")
	}
	switch strings.TrimSpace(req.ExecutionMode) {
	case "api", "codex_cli", "mock":
	default:
		return fmt.Errorf("execution_mode must be one of api, codex_cli, mock")
	}
	if strings.TrimSpace(req.OutputSchemaRef) == "" {
		return fmt.Errorf("output_schema_ref is required")
	}
	switch normalizeJobStatus(req.Status) {
	case "registered", "waiting_input", "ready":
	default:
		return fmt.Errorf("status must be one of registered, waiting_input, ready")
	}
	return nil
}

func normalizeJobStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "registered"
	}
	return status
}

func normalizeMaxRetries(value int) int {
	if value < 0 {
		return 0
	}
	if value > 5 {
		return 5
	}
	return value
}

func normalizeConcurrencyLimit(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func ensureStringSlice(items []string) []string {
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

func ensureInputRefs(items []model.AgentInputRef) []model.AgentInputRef {
	if items == nil {
		return []model.AgentInputRef{}
	}
	out := make([]model.AgentInputRef, 0, len(items))
	for _, item := range items {
		out = append(out, model.AgentInputRef{
			RefType:    strings.TrimSpace(item.RefType),
			RefID:      strings.TrimSpace(item.RefID),
			RefPath:    strings.TrimSpace(item.RefPath),
			RefVersion: strings.TrimSpace(item.RefVersion),
			Metadata:   ensureMap(item.Metadata),
		})
	}
	return out
}

func ensureMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	copyValue := make(map[string]any, len(value))
	for key, item := range value {
		copyValue[key] = item
	}
	return copyValue
}
