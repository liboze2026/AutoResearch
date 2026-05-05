package agentadmin

import (
	"context"
	"sort"
	"strings"

	"mrag-platform/backend/go/internal/model"
)

var preferredAgentTypes = []string{
	"reader",
	"insight",
	"dataset",
	"idea_generator",
	"planner",
	"coding",
	"writer",
}

type jobStore interface {
	List(context.Context, int) ([]model.AgentJob, error)
	GetByID(context.Context, string) (*model.AgentJob, error)
	LatestByAgentType(context.Context, string) (*model.AgentJob, error)
	CountByAgentType(context.Context, string) (int, error)
}

type artifactStore interface {
	ListByJobID(context.Context, string) ([]model.AgentArtifact, error)
}

type eventStore interface {
	List(context.Context, int) ([]model.AgentEvent, error)
}

type subscriptionStore interface {
	List(context.Context) ([]model.AgentSubscription, error)
}

type Service struct {
	jobs          jobStore
	artifacts     artifactStore
	events        eventStore
	subscriptions subscriptionStore
}

func NewService(jobs jobStore, artifacts artifactStore, events eventStore, subscriptions subscriptionStore) *Service {
	return &Service{
		jobs:          jobs,
		artifacts:     artifacts,
		events:        events,
		subscriptions: subscriptions,
	}
}

func (s *Service) ListAgents(ctx context.Context) ([]model.AgentSummary, error) {
	subscriptionItems, err := s.subscriptions.List(ctx)
	if err != nil {
		return nil, err
	}

	summaryMap := map[string]*model.AgentSummary{}
	agentTypes := make([]string, 0)
	for _, item := range preferredAgentTypes {
		agentTypes = appendAgentType(agentTypes, item)
	}

	for _, item := range subscriptionItems {
		agentTypes = appendAgentType(agentTypes, item.AgentType)
		summary := ensureAgentSummary(summaryMap, item.AgentType)
		summary.EventTypes = appendUnique(summary.EventTypes, item.EventType)
		summary.ExecutionMode = firstNonEmpty(summary.ExecutionMode, item.ExecutionMode)
		summary.ModelProvider = firstNonEmpty(summary.ModelProvider, item.ModelProvider)
		summary.ModelName = firstNonEmpty(summary.ModelName, item.ModelName)
		summary.PromptVersion = firstNonEmpty(summary.PromptVersion, item.PromptVersion)
		summary.OutputSchemaRef = firstNonEmpty(summary.OutputSchemaRef, item.OutputSchemaRef)
		summary.SkillRefs = appendUnique(summary.SkillRefs, item.SkillRefs...)
		summary.ToolRefs = appendUnique(summary.ToolRefs, item.ToolRefs...)
		summary.MemoryRefs = appendUnique(summary.MemoryRefs, item.MemoryRefs...)
		if item.ConcurrencyLimit > summary.ConcurrencyLimit {
			summary.ConcurrencyLimit = item.ConcurrencyLimit
		}
		if item.MaxRetries > summary.MaxRetries {
			summary.MaxRetries = item.MaxRetries
		}
		summary.Subscriptions = append(summary.Subscriptions, item)
	}

	for _, agentType := range agentTypes {
		summary := ensureAgentSummary(summaryMap, agentType)
		jobCount, countErr := s.jobs.CountByAgentType(ctx, agentType)
		if countErr != nil {
			return nil, countErr
		}
		summary.JobCount = jobCount

		latestJob, latestErr := s.jobs.LatestByAgentType(ctx, agentType)
		if latestErr != nil {
			return nil, latestErr
		}
		summary.LatestJob = latestJob
		if latestJob == nil {
			continue
		}
		summary.ExecutionMode = firstNonEmpty(summary.ExecutionMode, latestJob.ExecutionMode)
		summary.ModelProvider = firstNonEmpty(summary.ModelProvider, latestJob.ModelProvider)
		summary.ModelName = firstNonEmpty(summary.ModelName, latestJob.ModelName)
		summary.PromptVersion = firstNonEmpty(summary.PromptVersion, latestJob.PromptVersion)
		summary.OutputSchemaRef = firstNonEmpty(summary.OutputSchemaRef, latestJob.OutputSchemaRef)
		summary.SkillRefs = appendUnique(summary.SkillRefs, latestJob.SkillRefs...)
		summary.ToolRefs = appendUnique(summary.ToolRefs, latestJob.ToolRefs...)
		summary.MemoryRefs = appendUnique(summary.MemoryRefs, latestJob.MemoryRefs...)
		if latestJob.ConcurrencyLimit > summary.ConcurrencyLimit {
			summary.ConcurrencyLimit = latestJob.ConcurrencyLimit
		}
		if latestJob.MaxRetries > summary.MaxRetries {
			summary.MaxRetries = latestJob.MaxRetries
		}
	}

	items := make([]model.AgentSummary, 0, len(summaryMap))
	for _, agentType := range agentTypes {
		items = append(items, *summaryMap[agentType])
	}
	sort.SliceStable(items, func(i, j int) bool {
		return agentTypeOrder(items[i].AgentType) < agentTypeOrder(items[j].AgentType)
	})
	return items, nil
}

func (s *Service) ListJobs(ctx context.Context, limit int) ([]model.AgentJob, error) {
	return s.jobs.List(ctx, limit)
}

func (s *Service) GetJob(ctx context.Context, id string) (*model.AgentJob, error) {
	return s.jobs.GetByID(ctx, id)
}

func (s *Service) ListArtifacts(ctx context.Context, jobID string) ([]model.AgentArtifact, error) {
	return s.artifacts.ListByJobID(ctx, jobID)
}

func (s *Service) ListEvents(ctx context.Context, limit int) ([]model.AgentEvent, error) {
	return s.events.List(ctx, limit)
}

func ensureAgentSummary(summaryMap map[string]*model.AgentSummary, agentType string) *model.AgentSummary {
	if item, ok := summaryMap[agentType]; ok {
		return item
	}
	item := &model.AgentSummary{
		AgentType:     agentType,
		EventTypes:    []string{},
		SkillRefs:     []string{},
		ToolRefs:      []string{},
		MemoryRefs:    []string{},
		Subscriptions: []model.AgentSubscription{},
	}
	summaryMap[agentType] = item
	return item
}

func appendAgentType(items []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return items
	}
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func appendUnique(items []string, values ...string) []string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		exists := false
		for _, item := range items {
			if item == trimmed {
				exists = true
				break
			}
		}
		if !exists {
			items = append(items, trimmed)
		}
	}
	return items
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

func agentTypeOrder(agentType string) int {
	for index, item := range preferredAgentTypes {
		if item == agentType {
			return index
		}
	}
	return len(preferredAgentTypes) + 1
}
