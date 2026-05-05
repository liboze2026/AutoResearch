package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
)

type fakeResultCompareHandlerService struct{}

func (f *fakeResultCompareHandlerService) ListByExperimentID(context.Context, string) ([]model.ResultComparison, error) {
	now := time.Now()
	return []model.ResultComparison{{
		ID:           "cmp_1",
		ExperimentID: "exp_1",
		RunID:        "run_1",
		BaselineID:   "base_1",
		ComparisonJSON: map[string]interface{}{
			"judgment": "较优",
		},
		SummaryMD: "# Comparison",
		CreatedAt: now,
		UpdatedAt: now,
	}}, nil
}

func (f *fakeResultCompareHandlerService) CompareRun(context.Context, string) (*model.RunCompareResult, error) {
	now := time.Now()
	return &model.RunCompareResult{
		Run: model.ExperimentRun{
			ID:           "run_1",
			ExperimentID: "exp_1",
			RunStatus:    "succeeded",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		Comparisons: []model.ResultComparison{{
			ID:           "cmp_1",
			ExperimentID: "exp_1",
			RunID:        "run_1",
			SummaryMD:    "# Comparison",
			CreatedAt:    now,
			UpdatedAt:    now,
		}},
		WorkspaceDir:    "/workspace/experiments/exp_1/comparisons",
		OverallJudgment: "较优",
	}, nil
}

func TestResultCompareHandlerListByExperiment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewResultCompareHandler(&fakeResultCompareHandlerService{})
	router := gin.New()
	router.GET("/api/experiments/:id/comparisons", h.ListByExperiment)

	req := httptest.NewRequest(http.MethodGet, "/api/experiments/exp_1/comparisons", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Code int                      `json:"code"`
		Data []model.ResultComparison `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 comparison, got %d", len(resp.Data))
	}
}

func TestResultCompareHandlerCompareRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewResultCompareHandler(&fakeResultCompareHandlerService{})
	router := gin.New()
	router.POST("/api/runs/:id/compare", h.CompareRun)

	req := httptest.NewRequest(http.MethodPost, "/api/runs/run_1/compare", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Code int                    `json:"code"`
		Data model.RunCompareResult `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.OverallJudgment != "较优" {
		t.Fatalf("unexpected overall judgment: %s", resp.Data.OverallJudgment)
	}
}
