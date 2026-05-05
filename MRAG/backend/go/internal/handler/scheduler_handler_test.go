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

type fakeSchedulerHandlerService struct{}

func (f *fakeSchedulerHandlerService) QueueExperiment(context.Context, string) (*model.ExperimentQueueResult, error) {
	now := time.Now()
	return &model.ExperimentQueueResult{
		ExperimentID: "exp_1",
		Run: model.ExperimentRun{
			ID:           "run_1",
			ExperimentID: "exp_1",
			RunStatus:    "queued",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}, nil
}

func (f *fakeSchedulerHandlerService) ScheduleRun(context.Context, string) (*model.ScheduleResult, error) {
	now := time.Now()
	return &model.ScheduleResult{
		Run:      model.ExperimentRun{ID: "run_1", ExperimentID: "exp_1", AssignedServerID: "srv_1", RunStatus: "scheduled", CreatedAt: now, UpdatedAt: now},
		Decision: model.SchedulerDecision{ID: "sdec_1", RunID: "run_1", ChosenServerID: "srv_1", DecisionJSON: map[string]interface{}{"ruleVersion": "stage2_scheduler_v1"}, CreatedAt: now, UpdatedAt: now},
		Chosen:   model.SchedulerCandidate{ServerID: "srv_1", ServerName: "best-server", Eligible: true},
	}, nil
}

func (f *fakeSchedulerHandlerService) GetLatestDecision(context.Context, string) (*model.SchedulerDecision, error) {
	now := time.Now()
	return &model.SchedulerDecision{
		ID:             "sdec_1",
		RunID:          "run_1",
		ChosenServerID: "srv_1",
		DecisionJSON:   map[string]interface{}{"ruleVersion": "stage2_scheduler_v1"},
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func TestSchedulerHandlerGetDecision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSchedulerHandler(&fakeSchedulerHandlerService{})
	router := gin.New()
	router.GET("/api/runs/:id/scheduler-decision", h.GetDecision)

	req := httptest.NewRequest(http.MethodGet, "/api/runs/run_1/scheduler-decision", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Code int                     `json:"code"`
		Data model.SchedulerDecision `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.ChosenServerID != "srv_1" {
		t.Fatalf("expected srv_1, got %s", resp.Data.ChosenServerID)
	}
}
