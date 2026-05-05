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

type fakeAgentAdminHandlerService struct{}

func (f *fakeAgentAdminHandlerService) ListAgents(context.Context) ([]model.AgentSummary, error) {
	now := time.Now()
	return []model.AgentSummary{{
		AgentType:       "reader",
		EventTypes:      []string{"paper_imported"},
		ExecutionMode:   "codex_cli",
		ModelProvider:   "codex",
		ModelName:       "reader-default",
		PromptVersion:   "v1",
		OutputSchemaRef: "schemas/reader-output-v1.json",
		SkillRefs:       []string{"skills/reader/base"},
		ToolRefs:        []string{"tools/fs.read"},
		LatestJob: &model.AgentJob{
			ID:        "job_reader_1",
			AgentType: "reader",
			Status:    "succeeded",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}}, nil
}

func (f *fakeAgentAdminHandlerService) ListJobs(context.Context, int) ([]model.AgentJob, error) {
	now := time.Now()
	return []model.AgentJob{{
		ID:               "job_reader_1",
		AgentType:        "reader",
		Status:           "succeeded",
		ValidationStatus: "passed",
		RepairStatus:     "not_needed",
		CreatedAt:        now,
		UpdatedAt:        now,
	}}, nil
}

func (f *fakeAgentAdminHandlerService) GetJob(context.Context, string) (*model.AgentJob, error) {
	now := time.Now()
	return &model.AgentJob{
		ID:               "job_reader_1",
		AgentType:        "reader",
		Status:           "succeeded",
		ValidationStatus: "passed",
		RepairStatus:     "not_needed",
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (f *fakeAgentAdminHandlerService) ListArtifacts(context.Context, string) ([]model.AgentArtifact, error) {
	now := time.Now()
	return []model.AgentArtifact{{
		ID:           "artifact_1",
		JobID:        "job_reader_1",
		ArtifactType: "output_contract",
		Name:         "output.json",
		CreatedAt:    now,
		UpdatedAt:    now,
	}}, nil
}

func (f *fakeAgentAdminHandlerService) ListEvents(context.Context, int) ([]model.AgentEvent, error) {
	now := time.Now()
	return []model.AgentEvent{{
		ID:        "event_1",
		EventType: "paper_imported",
		SourceRef: "paper:paper_1",
		Status:    "processed",
		CreatedAt: now,
		UpdatedAt: now,
	}}, nil
}

func TestAgentAdminHandlerRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAgentAdminHandler(&fakeAgentAdminHandlerService{})
	router := gin.New()
	router.GET("/api/agents", h.ListAgents)
	router.GET("/api/agents/jobs", h.ListJobs)
	router.GET("/api/agents/jobs/:id", h.GetJob)
	router.GET("/api/agents/artifacts/:id", h.ListArtifacts)
	router.GET("/api/agent-events", h.ListEvents)

	for _, path := range []string{
		"/api/agents",
		"/api/agents/jobs",
		"/api/agents/jobs/job_reader_1",
		"/api/agents/artifacts/job_reader_1",
		"/api/agent-events",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, rec.Code)
		}
		var envelope struct {
			Code int             `json:"code"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("unmarshal %s response: %v", path, err)
		}
		if envelope.Code != 0 {
			t.Fatalf("expected code 0 for %s", path)
		}
	}
}
