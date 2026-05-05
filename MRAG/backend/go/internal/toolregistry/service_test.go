package toolregistry

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mrag-platform/backend/go/internal/model"
)

type memoryToolStore struct {
	items map[string]model.ToolDefinition
}

func newMemoryToolStore() *memoryToolStore {
	return &memoryToolStore{items: map[string]model.ToolDefinition{}}
}

func (s *memoryToolStore) Create(_ context.Context, item model.ToolDefinition) error {
	s.items[item.ToolID] = item
	return nil
}

func (s *memoryToolStore) List(_ context.Context) ([]model.ToolDefinition, error) {
	items := make([]model.ToolDefinition, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func TestToolRegistryRegisterSuccess(t *testing.T) {
	store := newMemoryToolStore()
	svc := NewService(store, t.TempDir(), "python")

	item, err := svc.Register(context.Background(), model.ToolRegisterRequest{
		Name:           "Echo Tool",
		OwnerAgentType: "coding",
		Description:    "Minimal echo tool for controlled registration.",
		UsageMD:        "Run this tool as a Python helper script.",
		InputSchema:    map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}}},
		OutputSchema:   map[string]any{"type": "object", "properties": map[string]any{"ok": map[string]any{"type": "boolean"}}},
		Version:        "v1",
		ScriptName:     "echo_tool.py",
		ScriptContent:  "from __future__ import annotations\n\ndef main() -> int:\n    return 0\n\nif __name__ == '__main__':\n    raise SystemExit(main())\n",
	})
	if err != nil {
		t.Fatalf("register tool failed: %v", err)
	}
	if item.ToolID == "" {
		t.Fatalf("expected tool id")
	}
	if item.TestStatus != "passed" {
		t.Fatalf("expected passed test status, got %s", item.TestStatus)
	}
	if _, err = os.Stat(item.Path); err != nil {
		t.Fatalf("expected tool file: %v", err)
	}
	if filepath.Ext(item.Path) != ".py" {
		t.Fatalf("expected python script path")
	}
}
