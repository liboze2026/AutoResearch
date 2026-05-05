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

func (s *IdeaService) PersistStructuredIdea(ctx context.Context, req model.StructuredIdeaPersistRequest) (*model.IdeaDetail, error) {
	ideaPayload := normalizeStructuredIdea(req.StructuredIdea)
	now := time.Now()
	idea := model.Idea{
		ID:            httpx.NewID("idea"),
		Title:         ideaPayload.Title,
		DescriptionMD: ideaPayload.DescriptionMD,
		Status:        "draft",
		Weight:        ideaWeightFromStructured(ideaPayload),
		Priority:      ideaPayload.Priority,
		Confidence:    normalizeIdeaConfidence(ideaPayload.Confidence),
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
	sources := buildStructuredIdeaSources(idea.ID, req, now)
	for _, source := range sources {
		if err := s.store.AddSource(ctx, source); err != nil {
			return nil, err
		}
	}
	if err := s.writeStructuredIdeaWorkspace(idea, sources, req.GeneratedFrom, &ideaPayload); err != nil {
		return nil, err
	}
	s.publishStructuredIdeaEvent(ctx, idea, sources, &ideaPayload)
	return &model.IdeaDetail{Idea: idea, Sources: sources, StructuredIdea: &ideaPayload}, nil
}

func (s *IdeaService) loadStructuredIdea(ideaID string) (*model.StructuredIdeaPayload, error) {
	paths := workspacepkg.New(s.workspaceRoot)
	metadataPath := filepath.Join(paths.IdeaPool(), ideaID, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var metadata model.IdeaWorkspaceMetadata
	if err = json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	if metadata.StructuredIdea == nil {
		return nil, nil
	}
	copyValue := *metadata.StructuredIdea
	return &copyValue, nil
}

func (s *IdeaService) writeStructuredIdeaWorkspace(idea model.Idea, sources []model.IdeaSource, generatedFrom string, structured *model.StructuredIdeaPayload) error {
	paths := workspacepkg.New(s.workspaceRoot)
	ideaDir := filepath.Join(paths.IdeaPool(), idea.ID)
	if err := os.MkdirAll(ideaDir, 0o755); err != nil {
		return err
	}
	ideaMD := buildStructuredIdeaMarkdown(idea, sources, structured)
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
		StructuredIdea: structured,
		UpdatedAt:      idea.UpdatedAt,
	}
	raw, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(ideaDir, "metadata.json"), raw, 0o644); err != nil {
		return err
	}
	if structured != nil {
		structuredRaw, err := json.MarshalIndent(structured, "", "  ")
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(ideaDir, "structured_idea.json"), structuredRaw, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (s *IdeaService) publishStructuredIdeaEvent(ctx context.Context, idea model.Idea, sources []model.IdeaSource, structured *model.StructuredIdeaPayload) {
	if s.eventPublisher == nil {
		return
	}
	inputRefs := []model.AgentInputRef{{RefType: "idea", RefID: idea.ID}}
	for _, source := range sources {
		if strings.TrimSpace(source.PaperID) != "" {
			inputRefs = append(inputRefs, model.AgentInputRef{RefType: "paper", RefID: source.PaperID})
		}
		if strings.TrimSpace(source.PaperInsightID) != "" {
			inputRefs = append(inputRefs, model.AgentInputRef{RefType: "insight", RefID: source.PaperInsightID})
		}
	}
	if structured != nil {
		for _, ref := range structured.TargetDatasetRefs {
			if strings.TrimSpace(ref) != "" {
				inputRefs = append(inputRefs, model.AgentInputRef{RefType: "dataset_asset", RefID: strings.TrimSpace(ref)})
			}
		}
		for _, ref := range structured.DatasetEvalProtocolRefs {
			if strings.TrimSpace(ref) != "" {
				inputRefs = append(inputRefs, model.AgentInputRef{RefType: "dataset_eval_protocol", RefPath: strings.TrimSpace(ref)})
			}
		}
	}
	_, _ = s.eventPublisher.PublishEvent(ctx, model.AgentEventCreateRequest{
		EventType: "idea_ready",
		SourceRef: "idea:" + idea.ID,
		InputRefs: inputRefs,
		Payload: map[string]any{
			"idea_id":            idea.ID,
			"title":              idea.Title,
			"status":             idea.Status,
			"research_direction": structuredField(structured, func(item *model.StructuredIdeaPayload) string { return item.ResearchDirection }),
			"innovation_type":    structuredField(structured, func(item *model.StructuredIdeaPayload) string { return item.InnovationType }),
			"priority":           idea.Priority,
			"confidence":         idea.Confidence,
		},
	})
}

func buildStructuredIdeaSources(ideaID string, req model.StructuredIdeaPersistRequest, now time.Time) []model.IdeaSource {
	sources := make([]model.IdeaSource, 0, len(req.PaperSources)+len(req.DatasetRefs)+len(req.HumanHints))
	for _, source := range req.PaperSources {
		sources = append(sources, model.IdeaSource{
			IdeaID:         ideaID,
			PaperID:        strings.TrimSpace(source.PaperID),
			PaperInsightID: strings.TrimSpace(source.PaperInsightID),
			SourceNote:     firstNonEmpty(strings.TrimSpace(source.SourceNote), "idea generator paper insight source"),
			PaperTitle:     strings.TrimSpace(source.PaperTitle),
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	for _, ref := range req.DatasetRefs {
		if strings.TrimSpace(ref) == "" {
			continue
		}
		sources = append(sources, model.IdeaSource{
			IdeaID:     ideaID,
			SourceNote: "dataset_asset:" + strings.TrimSpace(ref),
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}
	for _, ref := range req.EvalPlanRefs {
		if strings.TrimSpace(ref) == "" {
			continue
		}
		sources = append(sources, model.IdeaSource{
			IdeaID:     ideaID,
			SourceNote: "dataset_eval_protocol:" + strings.TrimSpace(ref),
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}
	for _, hint := range req.HumanHints {
		if strings.TrimSpace(hint) == "" {
			continue
		}
		sources = append(sources, model.IdeaSource{
			IdeaID:     ideaID,
			SourceNote: "human_hint: " + strings.TrimSpace(hint),
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}
	return sources
}

func buildStructuredIdeaMarkdown(idea model.Idea, sources []model.IdeaSource, structured *model.StructuredIdeaPayload) string {
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
	b.WriteString(idea.DescriptionMD)
	b.WriteString("\n\n")
	if structured != nil {
		b.WriteString("## Structured Fields\n")
		b.WriteString("- Research Direction: ")
		b.WriteString(structured.ResearchDirection)
		b.WriteString("\n")
		b.WriteString("- Innovation Type: ")
		b.WriteString(structured.InnovationType)
		b.WriteString("\n")
		b.WriteString("- Expected Advantage: ")
		b.WriteString(structured.ExpectedAdvantage)
		b.WriteString("\n")
		if len(structured.TargetDatasetRefs) > 0 {
			b.WriteString("- Target Dataset Refs: ")
			b.WriteString(strings.Join(structured.TargetDatasetRefs, ", "))
			b.WriteString("\n")
		}
		if len(structured.RiskPoints) > 0 {
			b.WriteString("\n## Risks\n")
			for _, risk := range structured.RiskPoints {
				b.WriteString("- ")
				b.WriteString(risk)
				b.WriteString("\n")
			}
		}
	}
	if len(sources) > 0 {
		b.WriteString("\n## Sources\n")
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

func normalizeStructuredIdea(value model.StructuredIdeaPayload) model.StructuredIdeaPayload {
	value.Title = firstNonEmpty(strings.TrimSpace(value.Title), "Untitled Structured Idea")
	value.DescriptionMD = firstNonEmpty(strings.TrimSpace(value.DescriptionMD), "## Goal\nStructured idea generated by Idea Generator Agent.")
	value.ResearchDirection = firstNonEmpty(strings.TrimSpace(value.ResearchDirection), value.Title)
	value.TargetDatasetRefs = normalizeStringSlice(value.TargetDatasetRefs)
	value.DatasetEvalProtocolRefs = normalizeStringSlice(value.DatasetEvalProtocolRefs)
	value.InnovationType = firstNonEmpty(strings.TrimSpace(value.InnovationType), "idea_generator")
	value.ExpectedAdvantage = firstNonEmpty(strings.TrimSpace(value.ExpectedAdvantage), "Adds a structured, reusable idea to the pool.")
	value.RiskPoints = normalizeStringSlice(value.RiskPoints)
	if len(value.RiskPoints) == 0 {
		value.RiskPoints = []string{"Requires experimental validation."}
	}
	value.Priority = clampPriority(value.Priority)
	value.Confidence = normalizeIdeaConfidence(value.Confidence)
	value.PaperInsightRefs = normalizeStringSlice(value.PaperInsightRefs)
	value.HumanHints = normalizeStringSlice(value.HumanHints)
	return value
}

func normalizeStringSlice(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func ideaWeightFromStructured(value model.StructuredIdeaPayload) int {
	if value.Priority > 0 {
		return value.Priority
	}
	if value.Confidence > 0 {
		return int(value.Confidence * 100)
	}
	return 60
}

func clampPriority(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func structuredField(structured *model.StructuredIdeaPayload, extract func(*model.StructuredIdeaPayload) string) string {
	if structured == nil {
		return ""
	}
	return strings.TrimSpace(extract(structured))
}
