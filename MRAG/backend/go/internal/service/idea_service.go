package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type IdeaStore interface {
	List(context.Context) ([]model.Idea, error)
	GetByID(context.Context, string) (*model.Idea, error)
	Create(context.Context, model.Idea) error
	Update(context.Context, model.Idea) error
	AddSource(context.Context, model.IdeaSource) error
	ListSources(context.Context, string) ([]model.IdeaSource, error)
}

type IdeaPaperReader interface {
	GetByID(context.Context, string) (*model.Paper, error)
	ListInsightsByPaper(context.Context, string) ([]model.PaperInsight, error)
}

type IdeaService struct {
	store          IdeaStore
	paperReader    IdeaPaperReader
	workspaceRoot  string
	eventPublisher IdeaEventPublisher
}

type IdeaEventPublisher interface {
	PublishEvent(context.Context, model.AgentEventCreateRequest) (*model.AgentEvent, error)
}

func NewIdeaService(store IdeaStore, paperReader IdeaPaperReader, workspaceRoot string) *IdeaService {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &IdeaService{store: store, paperReader: paperReader, workspaceRoot: workspaceRoot}
}

func (s *IdeaService) SetEventPublisher(publisher IdeaEventPublisher) {
	s.eventPublisher = publisher
}

func (s *IdeaService) List(ctx context.Context) ([]model.Idea, error) {
	return s.store.List(ctx)
}

func (s *IdeaService) GetByID(ctx context.Context, id string) (*model.IdeaDetail, error) {
	item, err := s.store.GetByID(ctx, id)
	if err != nil || item == nil {
		return nil, err
	}
	sources, err := s.store.ListSources(ctx, id)
	if err != nil {
		return nil, err
	}
	structured, err := s.loadStructuredIdea(id)
	if err != nil {
		return nil, err
	}
	return &model.IdeaDetail{Idea: *item, Sources: sources, StructuredIdea: structured}, nil
}

func (s *IdeaService) Create(ctx context.Context, req model.IdeaCreateRequest) (*model.IdeaDetail, error) {
	now := time.Now()
	idea := model.Idea{
		ID:            httpx.NewID("idea"),
		Title:         strings.TrimSpace(req.Title),
		DescriptionMD: strings.TrimSpace(req.DescriptionMD),
		Status:        normalizeIdeaStatus(req.Status),
		Weight:        req.Weight,
		Priority:      req.Priority,
		Confidence:    normalizeIdeaConfidence(req.Confidence),
		SourceType:    normalizeIdeaSourceType(req.SourceType),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := validateIdea(idea); err != nil {
		return nil, err
	}
	if err := s.store.Create(ctx, idea); err != nil {
		return nil, err
	}
	sources := make([]model.IdeaSource, 0, 1)
	if sourceNote := strings.TrimSpace(req.SourceNote); sourceNote != "" {
		source := model.IdeaSource{IdeaID: idea.ID, SourceNote: sourceNote, CreatedAt: now, UpdatedAt: now}
		if err := s.store.AddSource(ctx, source); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	if err := s.writeIdeaWorkspace(idea, sources, "manual"); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, idea, sources, "idea_ready")
	return &model.IdeaDetail{Idea: idea, Sources: sources}, nil
}

func (s *IdeaService) Update(ctx context.Context, id string, req model.IdeaUpdateRequest) (*model.IdeaDetail, error) {
	item, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("idea not found")
	}
	if req.Title != nil {
		item.Title = strings.TrimSpace(*req.Title)
	}
	if req.DescriptionMD != nil {
		item.DescriptionMD = strings.TrimSpace(*req.DescriptionMD)
	}
	if req.Status != nil {
		item.Status = normalizeIdeaStatus(*req.Status)
	}
	if req.Weight != nil {
		item.Weight = *req.Weight
	}
	if req.Priority != nil {
		item.Priority = *req.Priority
	}
	if req.Confidence != nil {
		item.Confidence = normalizeIdeaConfidence(*req.Confidence)
	}
	if req.SourceType != nil {
		item.SourceType = normalizeIdeaSourceType(*req.SourceType)
	}
	item.UpdatedAt = time.Now()
	if err := validateIdea(*item); err != nil {
		return nil, err
	}
	if err := s.store.Update(ctx, *item); err != nil {
		return nil, err
	}
	sources, err := s.store.ListSources(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.writeIdeaWorkspace(*item, sources, "update"); err != nil {
		return nil, err
	}
	return &model.IdeaDetail{Idea: *item, Sources: sources}, nil
}

func (s *IdeaService) GenerateFromPaper(ctx context.Context, paperID string) (*model.IdeaGenerationResult, error) {
	if s.paperReader == nil {
		return nil, fmt.Errorf("paper reader not configured")
	}
	paper, err := s.paperReader.GetByID(ctx, paperID)
	if err != nil {
		return nil, err
	}
	if paper == nil {
		return nil, fmt.Errorf("paper not found")
	}
	insights, err := s.paperReader.ListInsightsByPaper(ctx, paperID)
	if err != nil {
		return nil, err
	}
	if len(insights) == 0 {
		return nil, fmt.Errorf("paper insights not found")
	}
	seed := buildIdeaSeed(paper.Title, insights[0])
	items := generateMockIdeas(seed)
	created := make([]model.Idea, 0, len(items))
	for idx, generated := range items {
		now := time.Now()
		idea := model.Idea{
			ID:            httpx.NewID("idea"),
			Title:         generated.Title,
			DescriptionMD: generated.DescriptionMD,
			Status:        "draft",
			Weight:        generated.Weight,
			Priority:      generated.Priority,
			Confidence:    generated.Confidence,
			SourceType:    "auto",
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := s.store.Create(ctx, idea); err != nil {
			return nil, err
		}
		source := model.IdeaSource{
			IdeaID:         idea.ID,
			PaperID:        paperID,
			PaperInsightID: insights[0].ID,
			SourceNote:     fmt.Sprintf("mock idea #%d generated from paper insight summary", idx+1),
			PaperTitle:     paper.Title,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := s.store.AddSource(ctx, source); err != nil {
			return nil, err
		}
		if err := s.writeIdeaWorkspace(idea, []model.IdeaSource{source}, fmt.Sprintf("paper:%s", paperID)); err != nil {
			return nil, err
		}
		s.publishEvent(ctx, idea, []model.IdeaSource{source}, "idea_ready")
		created = append(created, idea)
	}
	return &model.IdeaGenerationResult{PaperID: paperID, Ideas: created}, nil
}

func (s *IdeaService) publishEvent(ctx context.Context, idea model.Idea, sources []model.IdeaSource, eventType string) {
	if s.eventPublisher == nil {
		return
	}
	inputRefs := []model.AgentInputRef{{RefType: "idea", RefID: idea.ID}}
	for _, source := range sources {
		if strings.TrimSpace(source.PaperID) != "" {
			inputRefs = append(inputRefs, model.AgentInputRef{RefType: "paper", RefID: source.PaperID})
		}
	}
	_, _ = s.eventPublisher.PublishEvent(ctx, model.AgentEventCreateRequest{
		EventType: eventType,
		SourceRef: "idea:" + idea.ID,
		InputRefs: inputRefs,
		Payload: map[string]any{
			"idea_id": idea.ID,
			"title":   idea.Title,
			"status":  idea.Status,
		},
	})
}

type generatedIdea struct {
	Title         string
	DescriptionMD string
	Weight        int
	Priority      int
	Confidence    float64
}

type ideaSeed struct {
	PaperTitle    string
	Summary       string
	Contributions []string
	Methods       []string
	Limitations   []string
}

func buildIdeaSeed(paperTitle string, insight model.PaperInsight) ideaSeed {
	seed := ideaSeed{PaperTitle: strings.TrimSpace(paperTitle), Summary: strings.TrimSpace(insight.SummaryMD)}
	seed.Contributions = toStringSlice(insight.ContributionsJSON)
	seed.Methods = toStringSlice(insight.MethodsJSON)
	seed.Limitations = toStringSlice(insight.LimitationsJSON)
	return seed
}

func generateMockIdeas(seed ideaSeed) []generatedIdea {
	baseTitle := safeSeedText(seed.PaperTitle, "Untitled Paper")
	summary := safeSeedText(firstSentence(seed.Summary), "Explore a controllable follow-up direction from the paper summary.")
	contribution := safeSeedText(firstNonEmpty(seed.Contributions...), "the paper's main contribution")
	method := safeSeedText(firstNonEmpty(seed.Methods...), "the paper's central method")
	limitation := safeSeedText(firstNonEmpty(seed.Limitations...), "limited evaluation breadth")
	return []generatedIdea{
		{
			Title:         fmt.Sprintf("Reframe %s for a narrower benchmark", baseTitle),
			DescriptionMD: fmt.Sprintf("## Hypothesis\nAdapt %s into a lighter benchmark setting.\n\n## Why now\n%s\n\n## Starting point\nReuse %s as the initial implementation anchor.", contribution, summary, method),
			Weight:        78,
			Priority:      85,
			Confidence:    0.74,
		},
		{
			Title:         fmt.Sprintf("Turn %s into a data-centric variant", baseTitle),
			DescriptionMD: fmt.Sprintf("## Hypothesis\nKeep the core method but shift effort toward data curation and error slicing.\n\n## Insight source\nThe paper highlights %s.\n\n## Expected value\nAddress %s with a more controllable dataset protocol.", method, limitation),
			Weight:        71,
			Priority:      76,
			Confidence:    0.68,
		},
		{
			Title:         fmt.Sprintf("Combine %s with a lightweight retrieval step", baseTitle),
			DescriptionMD: fmt.Sprintf("## Hypothesis\nAdd a simple retrieval or memory component before applying the paper method.\n\n## Insight source\nLeverage %s while keeping the implementation small for stage1 validation.", contribution),
			Weight:        66,
			Priority:      69,
			Confidence:    0.61,
		},
	}
}

func (s *IdeaService) writeIdeaWorkspace(idea model.Idea, sources []model.IdeaSource, generatedFrom string) error {
	paths := workspacepkg.New(s.workspaceRoot)
	ideaDir := filepath.Join(paths.IdeaPool(), idea.ID)
	if err := os.MkdirAll(ideaDir, 0o755); err != nil {
		return err
	}
	ideaMD := buildIdeaMarkdown(idea, sources)
	if err := os.WriteFile(filepath.Join(ideaDir, "idea.md"), []byte(ideaMD), 0o644); err != nil {
		return err
	}
	metadata := model.IdeaWorkspaceMetadata{
		IdeaID:         idea.ID,
		Title:          idea.Title,
		DescriptionMD:  idea.DescriptionMD,
		Status:         idea.Status,
		Weight:         idea.Weight,
		Priority:       idea.Priority,
		Confidence:     idea.Confidence,
		SourceType:     idea.SourceType,
		GeneratedFrom:  generatedFrom,
		SourceSnapshot: sources,
		UpdatedAt:      idea.UpdatedAt,
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(ideaDir, "metadata.json"), data, 0o644)
}

func buildIdeaMarkdown(idea model.Idea, sources []model.IdeaSource) string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(idea.Title)
	b.WriteString("\n\n")
	b.WriteString("- Status: ")
	b.WriteString(idea.Status)
	b.WriteString("\n")
	b.WriteString("- Source Type: ")
	b.WriteString(idea.SourceType)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("- Weight: %d\n", idea.Weight))
	b.WriteString(fmt.Sprintf("- Priority: %d\n", idea.Priority))
	b.WriteString(fmt.Sprintf("- Confidence: %.2f\n\n", idea.Confidence))
	if idea.DescriptionMD != "" {
		b.WriteString(idea.DescriptionMD)
		b.WriteString("\n\n")
	}
	if len(sources) > 0 {
		b.WriteString("## Sources\n")
		for _, source := range sources {
			line := strings.TrimSpace(source.SourceNote)
			if source.PaperTitle != "" {
				line = fmt.Sprintf("%s (%s)", line, source.PaperTitle)
			}
			if line == "" {
				line = "manual note"
			}
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String()) + "\n"
}

func validateIdea(idea model.Idea) error {
	if strings.TrimSpace(idea.Title) == "" {
		return fmt.Errorf("title is required")
	}
	switch idea.Status {
	case "draft", "shortlisted", "archived":
	default:
		return fmt.Errorf("status must be one of draft, shortlisted, archived")
	}
	switch idea.SourceType {
	case "auto", "human", "mixed":
	default:
		return fmt.Errorf("sourceType must be one of auto, human, mixed")
	}
	if idea.Confidence < 0 || idea.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	return nil
}

func normalizeIdeaStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "draft"
	}
	return status
}

func normalizeIdeaSourceType(sourceType string) string {
	sourceType = strings.TrimSpace(strings.ToLower(sourceType))
	if sourceType == "" {
		return "human"
	}
	return sourceType
}

func normalizeIdeaConfidence(confidence float64) float64 {
	if confidence < 0 {
		return 0
	}
	if confidence > 1 {
		return 1
	}
	return confidence
}

func toStringSlice(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case []string:
		return append([]string(nil), v...)
	case []any:
		items := make([]string, 0, len(v))
		for _, item := range v {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text != "" {
				items = append(items, text)
			}
		}
		return items
	default:
		text := strings.TrimSpace(fmt.Sprint(v))
		if text == "" || text == "<nil>" {
			return nil
		}
		return []string{text}
	}
}

func firstSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	for _, sep := range []string{". ", "\n"} {
		parts := strings.Split(text, sep)
		if len(parts) > 0 {
			first := strings.TrimSpace(parts[0])
			if first != "" {
				return first
			}
		}
	}
	return text
}

func safeSeedText(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
