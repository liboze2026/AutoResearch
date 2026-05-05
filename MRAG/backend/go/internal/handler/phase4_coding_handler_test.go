package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
)

type fakePhase4CodingHandlerService struct{}

func (f *fakePhase4CodingHandlerService) Run(_ context.Context, req model.Phase4CodingRunRequest) (*model.Phase4CodingRunResult, error) {
	return &model.Phase4CodingRunResult{
		Job: &model.AgentJob{ID: "ajob_phase4_coding_1"},
		RunManifest: &model.Phase4RunManifest{
			ID:               "p4run_1",
			DatasetProfileID: req.DatasetProfileID,
			IdeaID:           req.IdeaID,
			Status:           model.Phase4RunStatusSucceeded,
		},
	}, nil
}

func (f *fakePhase4CodingHandlerService) GetJob(_ context.Context, id string) (*model.Phase4CodingJobDetail, error) {
	return &model.Phase4CodingJobDetail{
		Job: &model.AgentJob{ID: id},
		RunManifest: &model.Phase4RunManifest{
			ID:     "p4run_1",
			Status: model.Phase4RunStatusSucceeded,
		},
	}, nil
}

func TestPhase4CodingHandlerRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewPhase4CodingHandler(&fakePhase4CodingHandlerService{})
	router.POST("/api/v2/phase4/coding/run", handler.Run)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/phase4/coding/run", strings.NewReader(`{"datasetProfileId":"p4ds_1","ideaId":"p4idea_1"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if data, ok := payload["data"].(map[string]any); !ok || data["runManifest"] == nil {
		t.Fatalf("expected runManifest in response: %#v", payload)
	}
}

func TestPhase4CodingHandlerGetJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewPhase4CodingHandler(&fakePhase4CodingHandlerService{})
	router.GET("/api/v2/phase4/coding/jobs/:id", handler.GetJob)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/phase4/coding/jobs/ajob_phase4_coding_1", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}
