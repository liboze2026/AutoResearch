package agenttrigger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type jobStore interface {
	GetByID(context.Context, string) (*model.AgentJob, error)
	Update(context.Context, model.AgentJob) error
}

type triggerStore interface {
	Create(context.Context, model.AgentJobTrigger) error
	Update(context.Context, model.AgentJobTrigger) error
}

type artifactWriter interface {
	Create(context.Context, model.AgentArtifact) error
}

type runtimeExecutor interface {
	Execute(context.Context, model.AgentRuntimeInput) (*model.AgentRuntimeOutput, error)
}

type postProcessor interface {
	PostProcess(context.Context, *model.AgentJob) error
}

type Service struct {
	jobs       jobStore
	triggers   triggerStore
	artifacts  artifactWriter
	runtime    runtimeExecutor
	processors map[string]postProcessor
}

func NewService(jobs jobStore, triggers triggerStore, artifacts artifactWriter, runtime runtimeExecutor) *Service {
	return &Service{jobs: jobs, triggers: triggers, artifacts: artifacts, runtime: runtime, processors: map[string]postProcessor{}}
}

func (s *Service) RegisterPostProcessor(agentType string, processor postProcessor) {
	agentType = strings.ToLower(strings.TrimSpace(agentType))
	if agentType == "" || processor == nil {
		return
	}
	s.processors[agentType] = processor
}

func (s *Service) Trigger(ctx context.Context, jobID string, req model.AgentJobTriggerRequest) (*model.AgentJob, error) {
	item, err := s.jobs.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("agent job not found")
	}
	if item.Status == "running" || item.Status == "validating" || item.Status == "repairing" {
		return nil, fmt.Errorf("agent job is already active")
	}

	now := time.Now()
	trigger := model.AgentJobTrigger{
		ID:           httpx.NewID("atrg"),
		JobID:        item.ID,
		TriggerType:  firstNonEmpty(strings.TrimSpace(req.TriggerType), "manual"),
		Status:       "running",
		Metadata:     ensureMap(req.Metadata),
		ErrorMessage: "",
		RequestedAt:  now,
		StartedAt:    &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err = s.triggers.Create(ctx, trigger); err != nil {
		return nil, err
	}

	item.Status = "running"
	item.StartedAt = &now
	item.UpdatedAt = now
	if err = s.jobs.Update(ctx, *item); err != nil {
		return nil, err
	}

	result, execErr := s.runtime.Execute(ctx, toRuntimeInput(*item))
	if execErr != nil {
		finishedAt := time.Now()
		item.Status = "failed"
		item.ValidationStatus = "failed"
		item.RepairStatus = "failed"
		item.ValidationErrors = []string{execErr.Error()}
		item.ErrorMessage = execErr.Error()
		item.CompletedAt = &finishedAt
		item.UpdatedAt = finishedAt
		_ = s.jobs.Update(ctx, *item)

		trigger.Status = "failed"
		trigger.ErrorMessage = execErr.Error()
		trigger.CompletedAt = &finishedAt
		trigger.UpdatedAt = finishedAt
		_ = s.triggers.Update(ctx, trigger)
		return nil, execErr
	}

	validatingAt := time.Now()
	item.Status = "validating"
	item.ValidationStatus = "validating"
	item.RepairStatus = "pending"
	item.UpdatedAt = validatingAt
	if err = s.jobs.Update(ctx, *item); err != nil {
		return nil, err
	}

	item.NormalizedPayload = ensureMap(result.NormalizedPayload)
	item.ArtifactManifest = ensureManifest(result.ArtifactManifest)
	item.RepairActions = ensureRepairActions(result.RepairActions)
	item.ToolUsages = ensureToolUsages(result.ToolUsages)
	item.Warnings = ensureStrings(result.Warnings)
	item.ValidationStatus = strings.TrimSpace(result.ValidationStatus)
	item.RepairStatus = strings.TrimSpace(result.RepairStatus)
	item.ValidationErrors = ensureStrings(result.ValidationErrors)
	item.ErrorMessage = strings.TrimSpace(result.ErrorMessage)

	finishedAt := time.Now()
	switch strings.TrimSpace(result.Status) {
	case "succeeded", "failed", "paused":
		item.Status = strings.TrimSpace(result.Status)
	default:
		item.Status = "succeeded"
	}
	item.CompletedAt = &finishedAt
	item.UpdatedAt = finishedAt
	if err = s.jobs.Update(ctx, *item); err != nil {
		return nil, err
	}

	if err = s.persistArtifacts(ctx, *item); err != nil {
		return nil, err
	}
	if err = s.runPostProcessor(ctx, item); err != nil {
		failedAt := time.Now()
		item.Status = "failed"
		item.ErrorMessage = err.Error()
		item.CompletedAt = &failedAt
		item.UpdatedAt = failedAt
		if updateErr := s.jobs.Update(ctx, *item); updateErr != nil {
			return nil, updateErr
		}
		trigger.Status = "failed"
		trigger.ErrorMessage = err.Error()
		trigger.CompletedAt = &failedAt
		trigger.UpdatedAt = failedAt
		_ = s.triggers.Update(ctx, trigger)
		return nil, err
	}

	trigger.Status = item.Status
	trigger.ErrorMessage = item.ErrorMessage
	trigger.CompletedAt = &finishedAt
	trigger.UpdatedAt = finishedAt
	if err = s.triggers.Update(ctx, trigger); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *Service) runPostProcessor(ctx context.Context, item *model.AgentJob) error {
	if item == nil {
		return nil
	}
	processor, ok := s.processors[strings.ToLower(strings.TrimSpace(item.AgentType))]
	if !ok || processor == nil {
		return nil
	}
	return processor.PostProcess(ctx, item)
}

func toRuntimeInput(job model.AgentJob) model.AgentRuntimeInput {
	return model.AgentRuntimeInput{
		JobID:           job.ID,
		AgentType:       job.AgentType,
		ExecutionMode:   job.ExecutionMode,
		ModelProvider:   job.ModelProvider,
		ModelName:       job.ModelName,
		PromptVersion:   job.PromptVersion,
		InputRefs:       ensureInputRefs(job.InputRefs),
		OutputSchemaRef: job.OutputSchemaRef,
		SkillRefs:       ensureStrings(job.SkillRefs),
		ToolRefs:        ensureStrings(job.ToolRefs),
		MemoryRefs:      ensureStrings(job.MemoryRefs),
		WorkspaceDir:    job.WorkspaceDir,
		Metadata:        ensureMap(job.Metadata),
	}
}

func (s *Service) persistArtifacts(ctx context.Context, job model.AgentJob) error {
	manifest := append([]model.AgentArtifactManifestItem{}, job.ArtifactManifest...)
	defaultItems := []model.AgentArtifactManifestItem{
		{ArtifactType: "input_contract", Name: "input.json", FilePath: filepath.Join(job.WorkspaceDir, "input.json"), Metadata: map[string]any{"source": "runtime"}},
		{ArtifactType: "output_contract", Name: "output.json", FilePath: filepath.Join(job.WorkspaceDir, "output.json"), Metadata: map[string]any{"source": "runtime"}},
	}
	for _, item := range defaultItems {
		if !containsArtifact(manifest, item.FilePath, item.ArtifactType) {
			manifest = append(manifest, item)
		}
	}
	for _, item := range manifest {
		now := time.Now()
		rec := model.AgentArtifact{
			ID:           httpx.NewID("aart"),
			JobID:        job.ID,
			ArtifactType: strings.TrimSpace(item.ArtifactType),
			Name:         firstNonEmpty(strings.TrimSpace(item.Name), filepath.Base(item.FilePath)),
			FilePath:     strings.TrimSpace(item.FilePath),
			Checksum:     checksumFile(item.FilePath),
			MetadataJSON: ensureMap(item.Metadata),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.artifacts.Create(ctx, rec); err != nil {
			return err
		}
	}
	return nil
}

func containsArtifact(items []model.AgentArtifactManifestItem, filePath string, artifactType string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.FilePath), strings.TrimSpace(filePath)) && strings.EqualFold(strings.TrimSpace(item.ArtifactType), strings.TrimSpace(artifactType)) {
			return true
		}
	}
	return false
}

func checksumFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func ensureStrings(items []string) []string {
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

func ensureManifest(items []model.AgentArtifactManifestItem) []model.AgentArtifactManifestItem {
	if items == nil {
		return []model.AgentArtifactManifestItem{}
	}
	out := make([]model.AgentArtifactManifestItem, 0, len(items))
	for _, item := range items {
		out = append(out, model.AgentArtifactManifestItem{
			ArtifactType: strings.TrimSpace(item.ArtifactType),
			Name:         strings.TrimSpace(item.Name),
			FilePath:     strings.TrimSpace(item.FilePath),
			Metadata:     ensureMap(item.Metadata),
		})
	}
	return out
}

func ensureRepairActions(items []model.AgentRepairAction) []model.AgentRepairAction {
	if items == nil {
		return []model.AgentRepairAction{}
	}
	out := make([]model.AgentRepairAction, 0, len(items))
	for _, item := range items {
		out = append(out, model.AgentRepairAction{
			Action:   strings.TrimSpace(item.Action),
			Status:   strings.TrimSpace(item.Status),
			Detail:   strings.TrimSpace(item.Detail),
			Metadata: ensureMap(item.Metadata),
		})
	}
	return out
}

func ensureToolUsages(items []model.AgentToolUsage) []model.AgentToolUsage {
	if items == nil {
		return []model.AgentToolUsage{}
	}
	out := make([]model.AgentToolUsage, 0, len(items))
	for _, item := range items {
		out = append(out, model.AgentToolUsage{
			ToolRef:  strings.TrimSpace(item.ToolRef),
			Status:   strings.TrimSpace(item.Status),
			Summary:  strings.TrimSpace(item.Summary),
			Metadata: ensureMap(item.Metadata),
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
