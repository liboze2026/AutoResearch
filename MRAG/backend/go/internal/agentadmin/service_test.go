package agentadmin

import (
	"context"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type fakeJobStore struct {
	jobs []model.AgentJob
}

func (s *fakeJobStore) List(_ context.Context, _ int) ([]model.AgentJob, error) {
	return append([]model.AgentJob(nil), s.jobs...), nil
}

func (s *fakeJobStore) GetByID(_ context.Context, id string) (*model.AgentJob, error) {
	for _, item := range s.jobs {
		if item.ID == id {
			copyItem := item
			return &copyItem, nil
		}
	}
	return nil, nil
}

func (s *fakeJobStore) LatestByAgentType(_ context.Context, agentType string) (*model.AgentJob, error) {
	var latest *model.AgentJob
	for _, item := range s.jobs {
		if item.AgentType != agentType {
			continue
		}
		copyItem := item
		if latest == nil || copyItem.CreatedAt.After(latest.CreatedAt) {
			latest = &copyItem
		}
	}
	return latest, nil
}

func (s *fakeJobStore) CountByAgentType(_ context.Context, agentType string) (int, error) {
	count := 0
	for _, item := range s.jobs {
		if item.AgentType == agentType {
			count++
		}
	}
	return count, nil
}

type fakeArtifactStore struct {
	items map[string][]model.AgentArtifact
}

func (s *fakeArtifactStore) ListByJobID(_ context.Context, jobID string) ([]model.AgentArtifact, error) {
	return append([]model.AgentArtifact(nil), s.items[jobID]...), nil
}

type fakeEventStore struct {
	items []model.AgentEvent
}

func (s *fakeEventStore) List(_ context.Context, _ int) ([]model.AgentEvent, error) {
	return append([]model.AgentEvent(nil), s.items...), nil
}

type fakeSubscriptionStore struct {
	items []model.AgentSubscription
}

func (s *fakeSubscriptionStore) List(_ context.Context) ([]model.AgentSubscription, error) {
	return append([]model.AgentSubscription(nil), s.items...), nil
}

func TestListAgentsAggregatesSubscriptionsAndLatestJobs(t *testing.T) {
	now := time.Now()
	svc := NewService(
		&fakeJobStore{jobs: []model.AgentJob{{
			ID:               "job_reader_1",
			AgentType:        "reader",
			ExecutionMode:    "mock",
			ModelProvider:    "codex",
			ModelName:        "reader-default",
			Status:           "succeeded",
			ValidationStatus: "passed",
			RepairStatus:     "not_needed",
			InputRefs:        []model.AgentInputRef{{RefType: "paper"}},
			CreatedAt:        now,
			UpdatedAt:        now,
		}}},
		&fakeArtifactStore{},
		&fakeEventStore{},
		&fakeSubscriptionStore{items: []model.AgentSubscription{{
			ID:               "sub_reader_1",
			Name:             "reader-from-paper",
			EventType:        "paper_imported",
			AgentType:        "reader",
			ExecutionMode:    "codex_cli",
			ModelProvider:    "codex",
			ModelName:        "reader-default",
			PromptVersion:    "v1",
			OutputSchemaRef:  "schemas/reader-output-v1.json",
			SkillRefs:        []string{"skills/reader/base"},
			ToolRefs:         []string{"tools/fs.read"},
			MemoryRefs:       []string{"memory/reader/default"},
			MaxRetries:       2,
			ConcurrencyLimit: 4,
		}}},
	)

	items, err := svc.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("list agents failed: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected agent summaries")
	}
	reader := items[0]
	if reader.AgentType != "reader" {
		t.Fatalf("expected reader first, got %s", reader.AgentType)
	}
	if reader.JobCount != 1 {
		t.Fatalf("expected job count 1, got %d", reader.JobCount)
	}
	if reader.LatestJob == nil || reader.LatestJob.ID != "job_reader_1" {
		t.Fatalf("expected latest reader job")
	}
	if len(reader.EventTypes) != 1 || reader.EventTypes[0] != "paper_imported" {
		t.Fatalf("expected reader event types")
	}
	if len(reader.SkillRefs) != 1 || reader.SkillRefs[0] != "skills/reader/base" {
		t.Fatalf("expected reader skill refs")
	}
}

func TestListJobsArtifactsAndEvents(t *testing.T) {
	now := time.Now()
	svc := NewService(
		&fakeJobStore{jobs: []model.AgentJob{{
			ID:        "job_writer_1",
			AgentType: "writer",
			Status:    "succeeded",
			CreatedAt: now,
			UpdatedAt: now,
		}}},
		&fakeArtifactStore{items: map[string][]model.AgentArtifact{
			"job_writer_1": {{
				ID:           "artifact_1",
				JobID:        "job_writer_1",
				ArtifactType: "draft",
				Name:         "draft.md",
			}},
		}},
		&fakeEventStore{items: []model.AgentEvent{{
			ID:        "event_1",
			EventType: "draft_ready",
			SourceRef: "draft:draft_1",
			Status:    "processed",
			CreatedAt: now,
			UpdatedAt: now,
		}}},
		&fakeSubscriptionStore{},
	)

	jobs, err := svc.ListJobs(context.Background(), 20)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("expected 1 job, err=%v", err)
	}
	artifacts, err := svc.ListArtifacts(context.Background(), "job_writer_1")
	if err != nil || len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, err=%v", err)
	}
	events, err := svc.ListEvents(context.Background(), 20)
	if err != nil || len(events) != 1 {
		t.Fatalf("expected 1 event, err=%v", err)
	}
}
