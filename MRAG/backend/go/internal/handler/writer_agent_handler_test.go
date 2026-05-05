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

type fakeWriterHandlerService struct{}

func (f *fakeWriterHandlerService) Run(context.Context, model.WriterRunRequest) (*model.WriterRunResult, error) {
	return &model.WriterRunResult{
		Job: &model.AgentJob{ID: "job_writer_1", AgentType: "writer"},
		Draft: &model.DraftDocument{
			DraftID: "draft_1",
			Title:   "Demo Draft",
		},
	}, nil
}

func (f *fakeWriterHandlerService) GetDraft(context.Context, string) (*model.DraftDocument, error) {
	return &model.DraftDocument{DraftID: "draft_1", Title: "Demo Draft"}, nil
}

func TestWriterAgentHandlerRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWriterAgentHandler(&fakeWriterHandlerService{})
	router := gin.New()
	router.POST("/api/agents/writer/run", h.Run)

	req := httptest.NewRequest(http.MethodPost, "/api/agents/writer/run", strings.NewReader(`{"paper_template_ref":"workspace/templates/paper.md","idea_refs":["idea_1"],"experiment_result_refs":["run_1"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestWriterAgentHandlerGetDraft(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewWriterAgentHandler(&fakeWriterHandlerService{})
	router := gin.New()
	router.GET("/api/drafts/:id", h.GetDraft)

	req := httptest.NewRequest(http.MethodGet, "/api/drafts/draft_1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
