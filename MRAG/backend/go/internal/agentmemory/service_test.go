package agentmemory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mrag-platform/backend/go/internal/model"
)

type memoryStoreStub struct {
	items map[string]model.AgentMemoryRecord
}

func newMemoryStoreStub() *memoryStoreStub {
	return &memoryStoreStub{items: map[string]model.AgentMemoryRecord{}}
}

func (s *memoryStoreStub) Upsert(_ context.Context, item model.AgentMemoryRecord) error {
	s.items[item.AgentType+"::"+item.MemoryKey] = item
	return nil
}

func (s *memoryStoreStub) ListByAgentType(_ context.Context, agentType string) ([]model.AgentMemoryRecord, error) {
	items := make([]model.AgentMemoryRecord, 0)
	for _, item := range s.items {
		if item.AgentType == agentType {
			items = append(items, item)
		}
	}
	return items, nil
}

func TestAgentMemoryWriteAndQuerySuccess(t *testing.T) {
	store := newMemoryStoreStub()
	workspaceRoot := t.TempDir()
	svc := NewService(store, workspaceRoot)

	item, err := svc.Upsert(context.Background(), model.AgentMemoryUpsertRequest{
		AgentType: "reader",
		MemoryKey: "paper-summary-style",
		ContentMD: "Prefer concise bullet summaries with schema-safe wording.",
		SourceRef: "paper:demo-001",
	})
	if err != nil {
		t.Fatalf("upsert memory failed: %v", err)
	}
	if item.ID == "" {
		t.Fatalf("expected memory id")
	}
	items, err := svc.ListByAgentType(context.Background(), "reader")
	if err != nil {
		t.Fatalf("list memory failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 memory record, got %d", len(items))
	}
	expectedFile := filepath.Join(workspaceRoot, "memory", "agents", "reader", "paper_summary_style.md")
	if _, err = os.Stat(expectedFile); err != nil {
		t.Fatalf("expected memory file: %v", err)
	}
}
