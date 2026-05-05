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

type fakeInsightAgentHandlerService struct{}

func (f *fakeInsightAgentHandlerService) Run(context.Context, model.InsightRunRequest) (*model.InsightRunResult, error) {
	return &model.InsightRunResult{
		Job: &model.AgentJob{
			ID:        "ajob_insight_1",
			AgentType: "insight",
			Status:    "succeeded",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Insight: model.PaperInsight{
			ID:                "pinsight_1",
			PaperID:           "paper_1",
			SummaryMD:         "Insight summary",
			ContributionsJSON: []string{"Contribution"},
			MethodsJSON:       []string{"Method"},
			LimitationsJSON:   []string{"Limitation"},
			NoveltyPointsJSON: []string{"Novelty"},
			ExtractStatus:     "completed",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		SummaryPath: "workspace/papers/insights/paper_1/summary.md",
	}, nil
}

func TestInsightAgentHandlerRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewInsightAgentHandler(&fakeInsightAgentHandlerService{})
	router := gin.New()
	router.POST("/api/agents/insight/run", handler.Run)

	body := bytes.NewBufferString(`{"paper_id":"paper_1","parsed_content_ref":"workspace/papers/parsed/paper_1/parsed.md","execution_mode":"mock"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/agents/insight/run", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Code int                    `json:"code"`
		Data model.InsightRunResult `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Job == nil || resp.Data.Job.AgentType != "insight" {
		t.Fatalf("expected insight job")
	}
	if resp.Data.Insight.PaperID != "paper_1" {
		t.Fatalf("unexpected paper id %s", resp.Data.Insight.PaperID)
	}
}
