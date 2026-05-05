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

type fakeExperimentHandlerService struct{}

func (f *fakeExperimentHandlerService) List(context.Context) ([]model.Experiment, error) {
	return []model.Experiment{{ID: "exp_1", Title: "Demo Experiment"}}, nil
}

func (f *fakeExperimentHandlerService) GetByID(context.Context, string) (*model.ExperimentDetail, error) {
	now := time.Now()
	return &model.ExperimentDetail{
		Experiment:   model.Experiment{ID: "exp_1", Title: "Demo Experiment", Status: "spec_ready", CreatedAt: now, UpdatedAt: now},
		DatasetAsset: model.DatasetAsset{ID: "dasset_1", Name: "Demo Dataset", CreatedAt: now, UpdatedAt: now},
	}, nil
}

func (f *fakeExperimentHandlerService) Create(context.Context, model.ExperimentCreateRequest) (*model.ExperimentDetail, error) {
	return f.GetByID(context.Background(), "exp_1")
}

func (f *fakeExperimentHandlerService) GenerateSpec(context.Context, string) (*model.ExperimentSpecDetail, error) {
	return f.GetLatestSpec(context.Background(), "exp_1")
}

func (f *fakeExperimentHandlerService) GetLatestSpec(context.Context, string) (*model.ExperimentSpecDetail, error) {
	now := time.Now()
	return &model.ExperimentSpecDetail{
		Spec: model.ExperimentSpec{
			ID:           "espec_1",
			ExperimentID: "exp_1",
			SpecJSON: map[string]interface{}{
				"model_name": "mock/llama3.1-8b-instruct",
			},
			TemplateType:  "text_finetune_v1",
			GeneratedFrom: map[string]interface{}{"strategy": "rule_based_v1"},
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		WorkspacePath:   "/workspace/experiments/exp_1/spec.json",
		GeneratorSource: "rule_based_v1",
	}, nil
}

func TestExperimentHandlerGetSpec(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewExperimentHandler(&fakeExperimentHandlerService{})

	router := gin.New()
	router.GET("/api/experiments/:id/spec", h.GetSpec)

	req := httptest.NewRequest(http.MethodGet, "/api/experiments/exp_1/spec", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Code int                        `json:"code"`
		Data model.ExperimentSpecDetail `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Spec.ExperimentID != "exp_1" {
		t.Fatalf("expected exp_1, got %s", resp.Data.Spec.ExperimentID)
	}
	if resp.Data.GeneratorSource != "rule_based_v1" {
		t.Fatalf("expected rule_based_v1, got %s", resp.Data.GeneratorSource)
	}
}
