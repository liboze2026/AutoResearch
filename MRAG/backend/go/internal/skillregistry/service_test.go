package skillregistry

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mrag-platform/backend/go/internal/model"
)

type memorySkillStore struct {
	items map[string]model.SkillDefinition
}

func newMemorySkillStore() *memorySkillStore {
	return &memorySkillStore{items: map[string]model.SkillDefinition{}}
}

func (s *memorySkillStore) Create(_ context.Context, item model.SkillDefinition) error {
	s.items[item.SkillID] = item
	return nil
}

func (s *memorySkillStore) List(_ context.Context) ([]model.SkillDefinition, error) {
	items := make([]model.SkillDefinition, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func TestSkillRegistryRegisterSuccess(t *testing.T) {
	store := newMemorySkillStore()
	svc := NewService(store, t.TempDir())

	item, err := svc.Register(context.Background(), model.SkillRegisterRequest{
		Name:         "Reader Summary Skill",
		Description:  "Provides structured summary instructions.",
		Entrypoint:   "SKILL.md",
		Dependencies: []string{"tools/fs.read", "schemas/reader-output-v1.json"},
		EntryContent: "# Reader Summary Skill\n\nReturn structured summaries.\n",
	})
	if err != nil {
		t.Fatalf("register skill failed: %v", err)
	}
	if item.SkillID == "" {
		t.Fatalf("expected skill id")
	}
	if _, err = os.Stat(item.Entrypoint); err != nil {
		t.Fatalf("expected skill entrypoint file: %v", err)
	}
	if filepath.Base(item.Entrypoint) != "SKILL.md" {
		t.Fatalf("expected SKILL.md entrypoint")
	}
}
