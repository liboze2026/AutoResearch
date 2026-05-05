package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
)

type fakeCodingHandlerService struct{}

func (f *fakeCodingHandlerService) Run(context.Context, model.CodingRunRequest) (*model.CodingRunResult, error) {
	return &model.CodingRunResult{
		Job:           &model.AgentJob{ID: "job_1", AgentType: "coding"},
		PatchManifest: []model.CodingPatchManifestItem{{PatchType: "spec_override", Target: "spec.hyperparams", Action: "merge"}},
	}, nil
}

func TestCodingAgentHandlerRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewCodingAgentHandler(&fakeCodingHandlerService{})
	router := gin.New()
	router.POST("/api/agents/coding/run", h.Run)

	req := httptest.NewRequest(http.MethodPost, "/api/agents/coding/run", strings.NewReader(`{"experiment_id":"exp_1","train_template_ref":"mock_train_template"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCodingAgentHandlerRunEvaluator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewCodingAgentHandler(&fakeCodingHandlerService{})
	router := gin.New()
	router.POST("/api/agents/evaluator/run", h.RunEvaluator)

	req := httptest.NewRequest(http.MethodPost, "/api/agents/evaluator/run", strings.NewReader(`{"experiment_id":"exp_1","train_template_ref":"mock_train_template"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
