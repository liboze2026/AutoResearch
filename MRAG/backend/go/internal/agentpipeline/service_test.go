package agentpipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type memoryEventStore struct {
	items map[string]model.AgentEvent
}

func newMemoryEventStore() *memoryEventStore {
	return &memoryEventStore{items: map[string]model.AgentEvent{}}
}

func (s *memoryEventStore) Create(_ context.Context, item model.AgentEvent) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryEventStore) Update(_ context.Context, item model.AgentEvent) error {
	s.items[item.ID] = item
	return nil
}

type memorySubscriptionStore struct {
	items map[string]model.AgentSubscription
}

func newMemorySubscriptionStore() *memorySubscriptionStore {
	return &memorySubscriptionStore{items: map[string]model.AgentSubscription{}}
}

func (s *memorySubscriptionStore) Create(_ context.Context, item model.AgentSubscription) error {
	s.items[item.ID] = item
	return nil
}

func (s *memorySubscriptionStore) ListByEventType(_ context.Context, eventType string) ([]model.AgentSubscription, error) {
	items := make([]model.AgentSubscription, 0)
	for _, item := range s.items {
		if item.Enabled && item.EventType == eventType {
			items = append(items, item)
		}
	}
	return items, nil
}

type memoryJobStore struct {
	items   map[string]model.AgentJob
	counter int
}

func newMemoryJobStore() *memoryJobStore {
	return &memoryJobStore{items: map[string]model.AgentJob{}}
}

func (s *memoryJobStore) Create(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryJobStore) GetByID(_ context.Context, id string) (*model.AgentJob, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryJobStore) Update(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryJobStore) ListByStatus(_ context.Context, status string, limit int) ([]model.AgentJob, error) {
	items := make([]model.AgentJob, 0)
	for _, item := range s.items {
		if item.Status == status {
			items = append(items, item)
		}
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *memoryJobStore) CountActiveByAgentType(_ context.Context, agentType string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.AgentType == agentType && (item.Status == "running" || item.Status == "validating" || item.Status == "repairing") {
			count++
		}
	}
	return count, nil
}

func (s *memoryJobStore) FindByDedupKey(_ context.Context, dedupKey string) (*model.AgentJob, error) {
	for _, item := range s.items {
		if item.DedupKey == dedupKey {
			copyItem := item
			return &copyItem, nil
		}
	}
	return nil, nil
}

func (s *memoryJobStore) CreateJob(_ context.Context, req model.AgentJobCreateRequest) (*model.AgentJob, error) {
	s.counter++
	now := time.Now()
	item := model.AgentJob{
		ID:                fmt.Sprintf("job_%d", s.counter),
		AgentType:         req.AgentType,
		ExecutionMode:     req.ExecutionMode,
		ModelProvider:     req.ModelProvider,
		ModelName:         req.ModelName,
		PromptVersion:     req.PromptVersion,
		InputRefs:         req.InputRefs,
		OutputSchemaRef:   req.OutputSchemaRef,
		SkillRefs:         req.SkillRefs,
		ToolRefs:          req.ToolRefs,
		MemoryRefs:        req.MemoryRefs,
		Metadata:          req.Metadata,
		Status:            req.Status,
		TriggerEventID:    req.TriggerEventID,
		DedupKey:          req.DedupKey,
		MaxRetries:        req.MaxRetries,
		ConcurrencyLimit:  req.ConcurrencyLimit,
		NormalizedPayload: map[string]any{},
		ArtifactManifest:  []model.AgentArtifactManifestItem{},
		RepairActions:     []model.AgentRepairAction{},
		ToolUsages:        []model.AgentToolUsage{},
		Warnings:          []string{},
		ValidationErrors:  []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	s.items[item.ID] = item
	return &item, nil
}

type triggerStub struct {
	store     *memoryJobStore
	triggered []string
}

func (t *triggerStub) Trigger(_ context.Context, jobID string, _ model.AgentJobTriggerRequest) (*model.AgentJob, error) {
	item, ok := t.store.items[jobID]
	if !ok {
		return nil, fmt.Errorf("job not found")
	}
	t.triggered = append(t.triggered, jobID)
	item.Status = "succeeded"
	item.UpdatedAt = time.Now()
	t.store.items[jobID] = item
	copyItem := item
	return &copyItem, nil
}

func TestPublishEventCreatesAgentJobs(t *testing.T) {
	eventStore := newMemoryEventStore()
	subscriptionStore := newMemorySubscriptionStore()
	jobStore := newMemoryJobStore()
	trigger := &triggerStub{store: jobStore}
	svc := NewService(eventStore, subscriptionStore, jobStore, creatorAdapter{store: jobStore}, trigger)

	_, _ = svc.CreateSubscription(context.Background(), model.AgentSubscriptionCreateRequest{
		Name:            "paper-import-reader",
		EventType:       "paper_imported",
		AgentType:       "reader",
		ExecutionMode:   "mock",
		OutputSchemaRef: "schemas/reader-output-v1.json",
	})
	_, _ = svc.CreateSubscription(context.Background(), model.AgentSubscriptionCreateRequest{
		Name:            "paper-import-insight",
		EventType:       "paper_imported",
		AgentType:       "insight",
		ExecutionMode:   "mock",
		OutputSchemaRef: "schemas/insight-output-v1.json",
	})

	event, err := svc.PublishEvent(context.Background(), model.AgentEventCreateRequest{
		EventType: "paper_imported",
		SourceRef: "paper:paper_1",
		InputRefs: []model.AgentInputRef{{RefType: "paper", RefID: "paper_1"}},
		Payload:   map[string]any{"paper_id": "paper_1"},
	})
	if err != nil {
		t.Fatalf("publish event failed: %v", err)
	}
	if len(event.TriggeredJobIDs) != 2 {
		t.Fatalf("expected 2 triggered jobs, got %d", len(event.TriggeredJobIDs))
	}
	readyJobs, _ := jobStore.ListByStatus(context.Background(), "ready", 10)
	if len(readyJobs) != 2 {
		t.Fatalf("expected 2 ready jobs, got %d", len(readyJobs))
	}
}

func TestPublishPhase4RunReadyEventWithoutSubscriptions(t *testing.T) {
	eventStore := newMemoryEventStore()
	subscriptionStore := newMemorySubscriptionStore()
	jobStore := newMemoryJobStore()
	trigger := &triggerStub{store: jobStore}
	svc := NewService(eventStore, subscriptionStore, jobStore, creatorAdapter{store: jobStore}, trigger)

	event, err := svc.PublishEvent(context.Background(), model.AgentEventCreateRequest{
		EventType: "phase4_run_ready",
		SourceRef: "phase4_run:p4run_1",
		InputRefs: []model.AgentInputRef{{RefType: "phase4_run_manifest", RefID: "p4run_1"}},
		Payload:   map[string]any{"run_manifest_id": "p4run_1"},
	})
	if err != nil {
		t.Fatalf("publish phase4_run_ready failed: %v", err)
	}
	if event == nil {
		t.Fatalf("expected published event")
	}
	if event.EventType != "phase4_run_ready" {
		t.Fatalf("expected phase4_run_ready event type, got %s", event.EventType)
	}
	if len(event.TriggeredJobIDs) != 0 {
		t.Fatalf("expected no triggered jobs without subscriptions, got %d", len(event.TriggeredJobIDs))
	}
}

func TestPublishPhase4WorkflowEventsWithoutSubscriptions(t *testing.T) {
	eventStore := newMemoryEventStore()
	subscriptionStore := newMemorySubscriptionStore()
	jobStore := newMemoryJobStore()
	trigger := &triggerStub{store: jobStore}
	svc := NewService(eventStore, subscriptionStore, jobStore, creatorAdapter{store: jobStore}, trigger)

	for _, eventType := range []string{
		"phase4_workflow_started",
		"phase4_reader_ready",
		"phase4_idea_batch_ready",
		"phase4_idea_selected",
		"phase4_coding_test_failed",
		"phase4_workflow_completed",
	} {
		event, err := svc.PublishEvent(context.Background(), model.AgentEventCreateRequest{
			EventType: eventType,
			SourceRef: "phase4_workflow:p4wf_1",
			InputRefs: []model.AgentInputRef{{RefType: "phase4_workflow", RefID: "p4wf_1"}},
			Payload:   map[string]any{"workflow_id": "p4wf_1"},
		})
		if err != nil {
			t.Fatalf("publish %s failed: %v", eventType, err)
		}
		if event == nil || event.EventType != eventType {
			t.Fatalf("expected event type %s, got %#v", eventType, event)
		}
	}
}

func TestDispatchReadyJobsRespectsConcurrencyLimit(t *testing.T) {
	eventStore := newMemoryEventStore()
	subscriptionStore := newMemorySubscriptionStore()
	jobStore := newMemoryJobStore()
	trigger := &triggerStub{store: jobStore}
	svc := NewService(eventStore, subscriptionStore, jobStore, creatorAdapter{store: jobStore}, trigger)

	now := time.Now()
	jobStore.items["job_1"] = model.AgentJob{ID: "job_1", AgentType: "coding", Status: "ready", ConcurrencyLimit: 1, CreatedAt: now, UpdatedAt: now}
	jobStore.items["job_2"] = model.AgentJob{ID: "job_2", AgentType: "coding", Status: "ready", ConcurrencyLimit: 1, CreatedAt: now.Add(time.Millisecond), UpdatedAt: now.Add(time.Millisecond)}

	dispatched, err := svc.DispatchReadyJobs(context.Background(), 10)
	if err != nil {
		t.Fatalf("dispatch ready jobs failed: %v", err)
	}
	if dispatched != 1 {
		t.Fatalf("expected 1 dispatched job, got %d", dispatched)
	}
	if len(trigger.triggered) != 1 {
		t.Fatalf("expected 1 triggered job, got %d", len(trigger.triggered))
	}
}

func TestPublishEventDedupSkipsDuplicateJobs(t *testing.T) {
	eventStore := newMemoryEventStore()
	subscriptionStore := newMemorySubscriptionStore()
	jobStore := newMemoryJobStore()
	trigger := &triggerStub{store: jobStore}
	svc := NewService(eventStore, subscriptionStore, jobStore, creatorAdapter{store: jobStore}, trigger)

	_, _ = svc.CreateSubscription(context.Background(), model.AgentSubscriptionCreateRequest{
		Name:            "paper-import-reader",
		EventType:       "paper_imported",
		AgentType:       "reader",
		ExecutionMode:   "mock",
		OutputSchemaRef: "schemas/reader-output-v1.json",
	})

	req := model.AgentEventCreateRequest{
		EventType: "paper_imported",
		SourceRef: "paper:paper_1",
		InputRefs: []model.AgentInputRef{{RefType: "paper", RefID: "paper_1"}},
		Payload:   map[string]any{"paper_id": "paper_1"},
	}
	firstEvent, err := svc.PublishEvent(context.Background(), req)
	if err != nil {
		t.Fatalf("first publish failed: %v", err)
	}
	secondEvent, err := svc.PublishEvent(context.Background(), req)
	if err != nil {
		t.Fatalf("second publish failed: %v", err)
	}
	if len(firstEvent.TriggeredJobIDs) != 1 {
		t.Fatalf("expected first event to trigger 1 job")
	}
	if len(secondEvent.TriggeredJobIDs) != 0 {
		t.Fatalf("expected second event to trigger 0 jobs, got %d", len(secondEvent.TriggeredJobIDs))
	}
	if len(jobStore.items) != 1 {
		t.Fatalf("expected 1 deduplicated job, got %d", len(jobStore.items))
	}
}

type creatorAdapter struct {
	store *memoryJobStore
}

func (c creatorAdapter) Create(ctx context.Context, req model.AgentJobCreateRequest) (*model.AgentJob, error) {
	return c.store.CreateJob(ctx, req)
}
