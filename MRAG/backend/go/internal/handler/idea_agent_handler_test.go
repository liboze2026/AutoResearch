package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
)

type fakeIdeaAgentHandlerService struct{}

func (f *fakeIdeaAgentHandlerService) Run(context.Context, model.IdeaGeneratorRunRequest) (*model.IdeaGeneratorRunResult, error) {
	now := time.Now()
	return &model.IdeaGeneratorRunResult{
		Job: &model.AgentJob{
			ID:        "ajob_idea_1",
			AgentType: "idea_generator",
			Status:    "succeeded",
			CreatedAt: now,
			UpdatedAt: now,
		},
		Idea: &model.IdeaDetail{
			Idea: model.Idea{
				ID:            "idea_1",
				Title:         "Structured Idea",
				DescriptionMD: "Idea description",
				Status:        "draft",
				Priority:      80,
				Confidence:    0.7,
				SourceType:    "auto",
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			StructuredIdea: &model.StructuredIdeaPayload{
				Title:                   "Structured Idea",
				DescriptionMD:           "Idea description",
				ResearchDirection:       "retrieval",
				TargetDatasetRefs:       []string{"dasset_1"},
				DatasetEvalProtocolRefs: []string{"workspace/datasets/dasset_1/evalplan.json"},
				InnovationType:          "insight_plus_dataset",
				ExpectedAdvantage:       "Higher robustness",
				RiskPoints:              []string{"Needs validation"},
				Priority:                80,
				Confidence:              0.7,
			},
		},
	}, nil
}

func TestIdeaAgentHandlerRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewIdeaAgentHandler(&fakeIdeaAgentHandlerService{})
	router := gin.New()
	router.POST("/api/agents/idea-generator/run", handler.Run)

	body := bytes.NewBufferString(`{"paper_insight_refs":["pinsight_1"],"dataset_asset_refs":["dasset_1"],"execution_mode":"mock"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/agents/idea-generator/run", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Code int                          `json:"code"`
		Data model.IdeaGeneratorRunResult `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Job == nil || resp.Data.Job.AgentType != "idea_generator" {
		t.Fatalf("expected idea generator job")
	}
}
