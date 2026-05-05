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

type fakeDatasetAgentHandlerService struct{}

func (f *fakeDatasetAgentHandlerService) Run(context.Context, model.DatasetRunRequest) (*model.DatasetRunResult, error) {
	now := time.Now()
	return &model.DatasetRunResult{
		Job: &model.AgentJob{
			ID:        "ajob_dataset_1",
			AgentType: "dataset",
			Status:    "succeeded",
			CreatedAt: now,
			UpdatedAt: now,
		},
		DatasetAsset: &model.DatasetAssetDetail{
			Asset: model.DatasetAsset{
				ID:                "dasset_1",
				Name:              "Dataset Asset",
				TaskType:          "retrieval",
				Status:            "active",
				SourceType:        "mrag_scan",
				LocalOrRemotePath: "/data/retrieval",
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
		EvalPlan: &model.DatasetEvalPlanDocument{
			DatasetAssetID:   "dasset_1",
			DatasetLocation:  "/data/retrieval",
			FetchAction:      "register_existing",
			SplitStrategy:    "query_document_train_dev_test",
			EvalProtocolJSON: map[string]any{"task_type": "retrieval"},
			MetricSchemaJSON: map[string]any{"primary_metric": "mrr"},
			ServerDecision:   map[string]any{"selected_server_name": "shenzhenvlab"},
			NotesMD:          "Dataset plan",
			GeneratedAt:      now,
		},
	}, nil
}

func (f *fakeDatasetAgentHandlerService) GetEvalPlan(context.Context, string) (*model.DatasetEvalPlanResponse, error) {
	now := time.Now()
	return &model.DatasetEvalPlanResponse{
		DatasetAsset: &model.DatasetAssetDetail{
			Asset: model.DatasetAsset{
				ID:                "dasset_1",
				Name:              "Dataset Asset",
				TaskType:          "retrieval",
				Status:            "active",
				SourceType:        "mrag_scan",
				LocalOrRemotePath: "/data/retrieval",
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
		EvalPlan: model.DatasetEvalPlanDocument{
			DatasetAssetID:   "dasset_1",
			DatasetLocation:  "/data/retrieval",
			FetchAction:      "register_existing",
			SplitStrategy:    "query_document_train_dev_test",
			EvalProtocolJSON: map[string]any{"task_type": "retrieval"},
			MetricSchemaJSON: map[string]any{"primary_metric": "mrr"},
			ServerDecision:   map[string]any{"selected_server_name": "shenzhenvlab"},
			NotesMD:          "Dataset plan",
			GeneratedAt:      now,
		},
	}, nil
}

func TestDatasetAgentHandlerRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewDatasetAgentHandler(&fakeDatasetAgentHandlerService{})
	router := gin.New()
	router.POST("/api/agents/dataset/run", handler.Run)

	body := bytes.NewBufferString(`{"research_direction":"multimodal retrieval","task_type":"retrieval","execution_mode":"mock"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/agents/dataset/run", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Code int                    `json:"code"`
		Data model.DatasetRunResult `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Job == nil || resp.Data.Job.AgentType != "dataset" {
		t.Fatalf("expected dataset job")
	}
}

func TestDatasetAgentHandlerGetEvalPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewDatasetAgentHandler(&fakeDatasetAgentHandlerService{})
	router := gin.New()
	router.GET("/api/dataset-assets/:id/evalplan", handler.GetEvalPlan)

	req := httptest.NewRequest(http.MethodGet, "/api/dataset-assets/dasset_1/evalplan", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
