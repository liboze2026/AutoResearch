package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
)

type fakePlannerHandlerService struct{}

func (f *fakePlannerHandlerService) Run(context.Context, model.PlannerRunRequest) (*model.PlannerRunResult, error) {
	return &model.PlannerRunResult{
		Job: &model.AgentJob{ID: "job_1", AgentType: "planner"},
		Experiment: &model.ExperimentDetail{
			Experiment: model.Experiment{ID: "exp_1", DatasetAssetID: "dasset_1", IdeaID: "idea_1"},
		},
		Plan: &model.ExperimentPlanDocument{
			ExperimentID:      "exp_1",
			IdeaID:            "idea_1",
			DatasetAssetID:    "dasset_1",
			TrainTemplateType: "text_finetune_v1",
			RunSequence:       []string{"generate_experiment_spec"},
			GeneratedAt:       time.Now(),
		},
	}, nil
}

func (f *fakePlannerHandlerService) GetPlan(context.Context, string) (*model.ExperimentPlanResponse, error) {
	return &model.ExperimentPlanResponse{
		Experiment: &model.ExperimentDetail{
			Experiment: model.Experiment{ID: "exp_1", DatasetAssetID: "dasset_1", IdeaID: "idea_1"},
		},
		Plan: model.ExperimentPlanDocument{
			ExperimentID:      "exp_1",
			IdeaID:            "idea_1",
			DatasetAssetID:    "dasset_1",
			TrainTemplateType: "text_finetune_v1",
			RunSequence:       []string{"generate_experiment_spec"},
			GeneratedAt:       time.Now(),
		},
	}, nil
}

func TestPlannerAgentHandlerRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPlannerAgentHandler(&fakePlannerHandlerService{})
	router := gin.New()
	router.POST("/api/agents/planner/run", h.Run)

	req := httptest.NewRequest(http.MethodPost, "/api/agents/planner/run", strings.NewReader(`{"idea_id":"idea_1","dataset_asset_refs":["dasset_1"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestPlannerAgentHandlerGetPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPlannerAgentHandler(&fakePlannerHandlerService{})
	router := gin.New()
	router.GET("/api/experiments/:id/plan", h.GetPlan)

	req := httptest.NewRequest(http.MethodGet, "/api/experiments/exp_1/plan", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
