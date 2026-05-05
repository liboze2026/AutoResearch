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

type fakeRecoveryHandlerService struct{}

func (f *fakeRecoveryHandlerService) Retry(context.Context, string) (*model.ExperimentQueueResult, error) {
	now := time.Now()
	return &model.ExperimentQueueResult{
		ExperimentID: "exp_1",
		Run: model.ExperimentRun{
			ID:           "run_retry_1",
			ExperimentID: "exp_1",
			RunStatus:    "queued",
			RetryCount:   1,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}, nil
}

func (f *fakeRecoveryHandlerService) GetRecovery(context.Context, string) (*model.RunRecoveryDetail, error) {
	return &model.RunRecoveryDetail{
		RunID:                  "run_1",
		ExperimentID:           "exp_1",
		RunStatus:              "failed",
		FailureReason:          "required output files are missing",
		FailureStage:           "collect_outputs",
		LastLogSummary:         "stdout tail",
		SuggestRetry:           true,
		RetryCount:             0,
		LatestAssignedServerID: "srv_1",
	}, nil
}

func TestRecoveryHandlerGetRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRecoveryHandler(&fakeRecoveryHandlerService{})
	router := gin.New()
	router.GET("/api/runs/:id/recovery", h.GetRecovery)

	req := httptest.NewRequest(http.MethodGet, "/api/runs/run_1/recovery", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Code int                     `json:"code"`
		Data model.RunRecoveryDetail `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.FailureStage != "collect_outputs" {
		t.Fatalf("expected collect_outputs, got %s", resp.Data.FailureStage)
	}
	if !resp.Data.SuggestRetry {
		t.Fatalf("expected suggest retry")
	}
}

func TestRecoveryHandlerRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewRecoveryHandler(&fakeRecoveryHandlerService{})
	router := gin.New()
	router.POST("/api/runs/:id/retry", h.Retry)

	req := httptest.NewRequest(http.MethodPost, "/api/runs/run_1/retry", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Code int                         `json:"code"`
		Data model.ExperimentQueueResult `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Run.RunStatus != "queued" {
		t.Fatalf("expected queued, got %s", resp.Data.Run.RunStatus)
	}
	if resp.Data.Run.RetryCount != 1 {
		t.Fatalf("expected retry_count 1, got %d", resp.Data.Run.RetryCount)
	}
}
