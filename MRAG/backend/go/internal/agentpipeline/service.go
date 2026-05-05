package agentpipeline

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

var validEventTypes = map[string]struct{}{
	"paper_imported":            {},
	"paper_parsed":              {},
	"insights_ready":            {},
	"dataset_asset_ready":       {},
	"idea_ready":                {},
	"plan_ready":                {},
	"experiment_result_ready":   {},
	"phase4_run_ready":          {},
	"phase4_workflow_started":   {},
	"phase4_reader_ready":       {},
	"phase4_idea_batch_ready":   {},
	"phase4_idea_selected":      {},
	"phase4_coding_test_failed": {},
	"phase4_workflow_completed": {},
	"draft_ready":               {},
}

type eventStore interface {
	Create(context.Context, model.AgentEvent) error
	Update(context.Context, model.AgentEvent) error
}

type subscriptionStore interface {
	Create(context.Context, model.AgentSubscription) error
	ListByEventType(context.Context, string) ([]model.AgentSubscription, error)
}

type jobCreator interface {
	Create(context.Context, model.AgentJobCreateRequest) (*model.AgentJob, error)
}

type jobStateStore interface {
	GetByID(context.Context, string) (*model.AgentJob, error)
	Update(context.Context, model.AgentJob) error
	ListByStatus(context.Context, string, int) ([]model.AgentJob, error)
	CountActiveByAgentType(context.Context, string) (int, error)
	FindByDedupKey(context.Context, string) (*model.AgentJob, error)
}

type triggerExecutor interface {
	Trigger(context.Context, string, model.AgentJobTriggerRequest) (*model.AgentJob, error)
}

type Service struct {
	events        eventStore
	subscriptions subscriptionStore
	jobs          jobStateStore
	jobCreator    jobCreator
	triggers      triggerExecutor
}

func NewService(events eventStore, subscriptions subscriptionStore, jobs jobStateStore, jobCreator jobCreator, triggers triggerExecutor) *Service {
	return &Service{
		events:        events,
		subscriptions: subscriptions,
		jobs:          jobs,
		jobCreator:    jobCreator,
		triggers:      triggers,
	}
}

func (s *Service) CreateSubscription(ctx context.Context, req model.AgentSubscriptionCreateRequest) (*model.AgentSubscription, error) {
	if err := validateSubscription(req); err != nil {
		return nil, err
	}
	now := time.Now()
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item := model.AgentSubscription{
		ID:               newID("asub"),
		Name:             strings.TrimSpace(req.Name),
		EventType:        strings.TrimSpace(req.EventType),
		AgentType:        strings.TrimSpace(req.AgentType),
		Enabled:          enabled,
		ExecutionMode:    strings.TrimSpace(req.ExecutionMode),
		ModelProvider:    strings.TrimSpace(req.ModelProvider),
		ModelName:        strings.TrimSpace(req.ModelName),
		PromptVersion:    strings.TrimSpace(req.PromptVersion),
		OutputSchemaRef:  strings.TrimSpace(req.OutputSchemaRef),
		SkillRefs:        ensureStrings(req.SkillRefs),
		ToolRefs:         ensureStrings(req.ToolRefs),
		MemoryRefs:       ensureStrings(req.MemoryRefs),
		TriggerRule:      ensureMap(req.TriggerRule),
		MaxRetries:       normalizeMaxRetries(req.MaxRetries),
		ConcurrencyLimit: normalizeConcurrencyLimit(req.ConcurrencyLimit, req.AgentType),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.subscriptions.Create(ctx, item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) PublishEvent(ctx context.Context, req model.AgentEventCreateRequest) (*model.AgentEvent, error) {
	if err := validateEvent(req); err != nil {
		return nil, err
	}
	now := time.Now()
	event := model.AgentEvent{
		ID:              newID("aevt"),
		EventType:       strings.TrimSpace(req.EventType),
		SourceRef:       strings.TrimSpace(req.SourceRef),
		InputRefs:       ensureInputRefs(req.InputRefs),
		Payload:         ensureMap(req.Payload),
		Status:          "processing",
		TriggeredJobIDs: []string{},
		ErrorMessage:    "",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.events.Create(ctx, event); err != nil {
		return nil, err
	}

	subscriptions, err := s.subscriptions.ListByEventType(ctx, event.EventType)
	if err != nil {
		event.Status = "failed"
		event.ErrorMessage = err.Error()
		event.UpdatedAt = time.Now()
		_ = s.events.Update(ctx, event)
		return nil, err
	}

	for _, sub := range subscriptions {
		if !matchesTriggerRule(sub.TriggerRule, event) {
			continue
		}
		dedupKey := buildDedupKey(event, sub)
		existing, err := s.jobs.FindByDedupKey(ctx, dedupKey)
		if err != nil {
			event.Status = "failed"
			event.ErrorMessage = err.Error()
			event.UpdatedAt = time.Now()
			_ = s.events.Update(ctx, event)
			return nil, err
		}
		if existing != nil {
			continue
		}

		job, createErr := s.jobCreator.Create(ctx, model.AgentJobCreateRequest{
			AgentType:        sub.AgentType,
			ExecutionMode:    sub.ExecutionMode,
			ModelProvider:    sub.ModelProvider,
			ModelName:        sub.ModelName,
			PromptVersion:    sub.PromptVersion,
			InputRefs:        ensureInputRefs(event.InputRefs),
			OutputSchemaRef:  sub.OutputSchemaRef,
			SkillRefs:        ensureStrings(sub.SkillRefs),
			ToolRefs:         ensureStrings(sub.ToolRefs),
			MemoryRefs:       ensureStrings(sub.MemoryRefs),
			Metadata:         mergeMaps(event.Payload, map[string]any{"event_id": event.ID, "event_type": event.EventType, "subscription_id": sub.ID}),
			Status:           "ready",
			TriggerEventID:   event.ID,
			DedupKey:         dedupKey,
			MaxRetries:       sub.MaxRetries,
			ConcurrencyLimit: sub.ConcurrencyLimit,
		})
		if createErr != nil {
			event.Status = "failed"
			event.ErrorMessage = createErr.Error()
			event.UpdatedAt = time.Now()
			_ = s.events.Update(ctx, event)
			return nil, createErr
		}
		event.TriggeredJobIDs = append(event.TriggeredJobIDs, job.ID)
	}

	processedAt := time.Now()
	event.Status = "processed"
	event.ProcessedAt = &processedAt
	event.UpdatedAt = processedAt
	if err := s.events.Update(ctx, event); err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *Service) DispatchReadyJobs(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 20
	}
	readyJobs, err := s.jobs.ListByStatus(ctx, "ready", limit)
	if err != nil {
		return 0, err
	}
	sort.SliceStable(readyJobs, func(i, j int) bool {
		if readyJobs[i].CreatedAt.Equal(readyJobs[j].CreatedAt) {
			return readyJobs[i].ID < readyJobs[j].ID
		}
		return readyJobs[i].CreatedAt.Before(readyJobs[j].CreatedAt)
	})

	reserved := map[string]int{}
	dispatched := 0
	for _, job := range readyJobs {
		active, countErr := s.jobs.CountActiveByAgentType(ctx, job.AgentType)
		if countErr != nil {
			return dispatched, countErr
		}
		limitForType := normalizeConcurrencyLimit(job.ConcurrencyLimit, job.AgentType)
		if active+reserved[job.AgentType] >= limitForType {
			continue
		}
		reserved[job.AgentType]++
		dispatched++
		triggeredJob, triggerErr := s.triggers.Trigger(ctx, job.ID, model.AgentJobTriggerRequest{
			TriggerType: "subscription_worker",
			Metadata: map[string]any{
				"trigger_event_id": job.TriggerEventID,
				"retry_count":      job.RetryCount,
			},
		})
		if triggerErr != nil {
			if retryErr := s.retryOrFail(ctx, job, triggerErr.Error()); retryErr != nil {
				return dispatched, retryErr
			}
			continue
		}
		if triggeredJob != nil && triggeredJob.Status == "failed" {
			if retryErr := s.retryOrFail(ctx, *triggeredJob, triggeredJob.ErrorMessage); retryErr != nil {
				return dispatched, retryErr
			}
		}
	}
	return dispatched, nil
}

func (s *Service) retryOrFail(ctx context.Context, job model.AgentJob, reason string) error {
	current, err := s.jobs.GetByID(ctx, job.ID)
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}
	if current.RetryCount >= current.MaxRetries {
		current.Status = "failed"
		current.ErrorMessage = firstNonEmpty(strings.TrimSpace(reason), current.ErrorMessage)
		current.UpdatedAt = time.Now()
		return s.jobs.Update(ctx, *current)
	}
	current.RetryCount++
	current.Status = "ready"
	current.ErrorMessage = firstNonEmpty(strings.TrimSpace(reason), current.ErrorMessage)
	current.CompletedAt = nil
	current.UpdatedAt = time.Now()
	return s.jobs.Update(ctx, *current)
}

type Worker struct {
	service  *Service
	interval time.Duration
	batch    int
}

func NewWorker(service *Service, interval time.Duration, batch int) *Worker {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if batch <= 0 {
		batch = 20
	}
	return &Worker{service: service, interval: interval, batch: batch}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = w.service.DispatchReadyJobs(ctx, w.batch)
			}
		}
	}()
}

func validateEvent(req model.AgentEventCreateRequest) error {
	eventType := strings.TrimSpace(req.EventType)
	if eventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if _, ok := validEventTypes[eventType]; !ok {
		return fmt.Errorf("event_type is not supported")
	}
	if strings.TrimSpace(req.SourceRef) == "" {
		return fmt.Errorf("source_ref is required")
	}
	if len(req.InputRefs) == 0 {
		return fmt.Errorf("input_refs is required")
	}
	return nil
}

func validateSubscription(req model.AgentSubscriptionCreateRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if err := validateEvent(model.AgentEventCreateRequest{
		EventType: req.EventType,
		SourceRef: "subscription",
		InputRefs: []model.AgentInputRef{{RefType: "subscription"}},
	}); err != nil && err.Error() != "source_ref is required" && err.Error() != "input_refs is required" {
		return err
	}
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
	return nil
}

func matchesTriggerRule(rule map[string]any, event model.AgentEvent) bool {
	if len(rule) == 0 {
		return true
	}
	raw, ok := rule["required_ref_types"]
	if !ok {
		return true
	}
	required := toStringSlice(raw)
	if len(required) == 0 {
		return true
	}
	available := map[string]struct{}{}
	for _, ref := range event.InputRefs {
		available[strings.TrimSpace(ref.RefType)] = struct{}{}
	}
	for _, value := range required {
		if _, ok := available[value]; !ok {
			return false
		}
	}
	return true
}

func buildDedupKey(event model.AgentEvent, sub model.AgentSubscription) string {
	payload := map[string]any{
		"event_type":        event.EventType,
		"source_ref":        event.SourceRef,
		"input_refs":        event.InputRefs,
		"agent_type":        sub.AgentType,
		"execution_mode":    sub.ExecutionMode,
		"output_schema_ref": sub.OutputSchemaRef,
	}
	raw, _ := json.Marshal(payload)
	sum := sha1.Sum(raw)
	return "dedup_" + hex.EncodeToString(sum[:])
}

func normalizeConcurrencyLimit(value int, agentType string) int {
	if value > 0 {
		return value
	}
	switch strings.ToLower(strings.TrimSpace(agentType)) {
	case "reader", "insight", "dataset", "idea", "idea_generator":
		return 4
	case "planner", "coding", "writer", "picture":
		return 1
	default:
		return 2
	}
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

func toStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return ensureStrings(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" {
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

func mergeMaps(base map[string]any, extra map[string]any) map[string]any {
	out := ensureMap(base)
	for key, value := range extra {
		out[key] = value
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func newID(prefix string) string {
	return httpx.NewID(prefix)
}
