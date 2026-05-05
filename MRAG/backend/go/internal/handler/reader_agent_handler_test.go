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

type fakeReaderAgentHandlerService struct{}

func (f *fakeReaderAgentHandlerService) Run(context.Context, model.ReaderRunRequest) (*model.ReaderRunResult, error) {
	return &model.ReaderRunResult{
		Job: &model.AgentJob{
			ID:        "ajob_reader_1",
			AgentType: "reader",
			Status:    "succeeded",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		CandidatePapers: []model.ReaderCandidatePaper{{
			Title:      "Demo Reader Paper",
			Abstract:   "Demo abstract",
			Source:     "arxiv",
			Year:       2026,
			URL:        "https://arxiv.org/abs/2603.0001",
			FileStatus: "metadata_only",
		}},
	}, nil
}

func (f *fakeReaderAgentHandlerService) GetJob(context.Context, string) (*model.ReaderJobDetail, error) {
	return &model.ReaderJobDetail{
		Job: &model.AgentJob{
			ID:        "ajob_reader_1",
			AgentType: "reader",
			Status:    "succeeded",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		CandidatePapers: []model.ReaderCandidatePaper{{
			Title:      "Demo Reader Paper",
			Abstract:   "Demo abstract",
			Source:     "arxiv",
			Year:       2026,
			URL:        "https://arxiv.org/abs/2603.0001",
			FileStatus: "metadata_only",
		}},
	}, nil
}

func TestReaderAgentHandlerRunAndGetJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewReaderAgentHandler(&fakeReaderAgentHandlerService{})
	router := gin.New()
	router.POST("/api/agents/reader/run", handler.Run)
	router.GET("/api/agents/reader/jobs/:id", handler.GetJob)

	body := bytes.NewBufferString(`{"research_direction":"multimodal retrieval","keywords":["retrieval"],"source_scope":"arxiv","max_papers":1,"execution_mode":"mock"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/agents/reader/run", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected run 200, got %d", rec.Code)
	}

	var runResp struct {
		Code int                   `json:"code"`
		Data model.ReaderRunResult `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("unmarshal run response: %v", err)
	}
	if runResp.Data.Job == nil || runResp.Data.Job.ID == "" {
		t.Fatalf("expected reader run job")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/agents/reader/jobs/ajob_reader_1", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected get job 200, got %d", rec.Code)
	}

	var jobResp struct {
		Code int                   `json:"code"`
		Data model.ReaderJobDetail `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &jobResp); err != nil {
		t.Fatalf("unmarshal job response: %v", err)
	}
	if len(jobResp.Data.CandidatePapers) != 1 {
		t.Fatalf("expected 1 candidate paper")
	}
}
