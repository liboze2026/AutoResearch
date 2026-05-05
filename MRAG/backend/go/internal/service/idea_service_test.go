package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type memoryIdeaStore struct {
	ideas   map[string]model.Idea
	sources map[string][]model.IdeaSource
}

func newMemoryIdeaStore() *memoryIdeaStore {
	return &memoryIdeaStore{ideas: map[string]model.Idea{}, sources: map[string][]model.IdeaSource{}}
}

func (s *memoryIdeaStore) List(_ context.Context) ([]model.Idea, error) {
	items := make([]model.Idea, 0, len(s.ideas))
	for _, item := range s.ideas {
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryIdeaStore) GetByID(_ context.Context, id string) (*model.Idea, error) {
	item, ok := s.ideas[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryIdeaStore) Create(_ context.Context, idea model.Idea) error {
	s.ideas[idea.ID] = idea
	return nil
}

func (s *memoryIdeaStore) Update(_ context.Context, idea model.Idea) error {
	s.ideas[idea.ID] = idea
	return nil
}

func (s *memoryIdeaStore) AddSource(_ context.Context, source model.IdeaSource) error {
	s.sources[source.IdeaID] = append(s.sources[source.IdeaID], source)
	return nil
}

func (s *memoryIdeaStore) ListSources(_ context.Context, ideaID string) ([]model.IdeaSource, error) {
	items := s.sources[ideaID]
	out := make([]model.IdeaSource, len(items))
	copy(out, items)
	return out, nil
}

type memoryIdeaPaperReader struct {
	papers   map[string]model.Paper
	insights map[string][]model.PaperInsight
}

func newMemoryIdeaPaperReader() *memoryIdeaPaperReader {
	return &memoryIdeaPaperReader{papers: map[string]model.Paper{}, insights: map[string][]model.PaperInsight{}}
}

func (s *memoryIdeaPaperReader) GetByID(_ context.Context, id string) (*model.Paper, error) {
	item, ok := s.papers[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryIdeaPaperReader) ListInsightsByPaper(_ context.Context, paperID string) ([]model.PaperInsight, error) {
	items := s.insights[paperID]
	out := make([]model.PaperInsight, len(items))
	copy(out, items)
	return out, nil
}

func TestIdeaServiceManualCreateIdea(t *testing.T) {
	store := newMemoryIdeaStore()
	svc := NewIdeaService(store, newMemoryIdeaPaperReader(), t.TempDir())

	created, err := svc.Create(context.Background(), model.IdeaCreateRequest{
		Title:         "Manual MRAG Idea",
		DescriptionMD: "## Goal\nCreate a manually curated idea.",
		Status:        "draft",
		Weight:        80,
		Priority:      90,
		Confidence:    0.82,
		SourceType:    "human",
		SourceNote:    "Captured during researcher review",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.Idea.ID == "" {
		t.Fatalf("expected idea id")
	}
	if created.Idea.SourceType != "human" {
		t.Fatalf("expected source type human, got %s", created.Idea.SourceType)
	}
	ideaDir := filepath.Join(svc.workspaceRoot, "ideas", "pool", created.Idea.ID)
	if _, err := os.Stat(filepath.Join(ideaDir, "idea.md")); err != nil {
		t.Fatalf("expected idea workspace markdown: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ideaDir, "metadata.json")); err != nil {
		t.Fatalf("expected idea workspace metadata: %v", err)
	}
}

func TestIdeaServiceGenerateFromPaper(t *testing.T) {
	store := newMemoryIdeaStore()
	paperReader := newMemoryIdeaPaperReader()
	now := time.Now()
	paperReader.papers["paper_1"] = model.Paper{ID: "paper_1", Title: "Mock Retrieval Paper", Status: "insight_extracted", SourceType: "workspace", CreatedAt: now, UpdatedAt: now}
	paperReader.insights["paper_1"] = []model.PaperInsight{{
		ID:                "pinsight_1",
		PaperID:           "paper_1",
		SummaryMD:         "This paper improves retrieval-guided generation with a compact controller.",
		ContributionsJSON: []string{"compact controller for retrieval-guided generation"},
		MethodsJSON:       []string{"retrieval-guided generation"},
		LimitationsJSON:   []string{"limited evaluation breadth"},
		ExtractStatus:     "completed",
		CreatedAt:         now,
		UpdatedAt:         now,
	}}
	svc := NewIdeaService(store, paperReader, t.TempDir())

	result, err := svc.GenerateFromPaper(context.Background(), "paper_1")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(result.Ideas) != 3 {
		t.Fatalf("expected 3 ideas, got %d", len(result.Ideas))
	}
	if len(store.sources[result.Ideas[0].ID]) != 1 {
		t.Fatalf("expected source linked to generated idea")
	}
	ideaDir := filepath.Join(svc.workspaceRoot, "ideas", "pool", result.Ideas[0].ID)
	if _, err := os.Stat(filepath.Join(ideaDir, "idea.md")); err != nil {
		t.Fatalf("expected generated idea workspace markdown: %v", err)
	}
}

func TestIdeaServiceListIdeas(t *testing.T) {
	store := newMemoryIdeaStore()
	now := time.Now()
	store.ideas["idea_1"] = model.Idea{ID: "idea_1", Title: "Idea One", Status: "draft", Weight: 70, SourceType: "human", Priority: 60, Confidence: 0.5, CreatedAt: now, UpdatedAt: now}
	svc := NewIdeaService(store, newMemoryIdeaPaperReader(), t.TempDir())

	items, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 idea, got %d", len(items))
	}
}
