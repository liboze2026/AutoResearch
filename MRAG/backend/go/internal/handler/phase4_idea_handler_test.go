package handler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/model"
)

type fakePhase4IdeaHandlerService struct{}

func (f *fakePhase4IdeaHandlerService) Run(_ context.Context, req model.Phase4IdeaRunRequest) (*model.Phase4IdeaRunResult, error) {
	return &model.Phase4IdeaRunResult{
		Job: &model.AgentJob{
			ID:        "job_phase4_idea_001",
			AgentType: "idea_phase4",
			Status:    "succeeded",
			Metadata: map[string]any{
				"dataset_profile_id": req.DatasetProfileID,
				"reader_context_id":  req.ReaderContextID,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Ideas: []model.Phase4Idea{
			{ID: "p4idea_1", Title: "Idea 1", Status: model.Phase4IdeaStatusScored},
			{ID: "p4idea_2", Title: "Idea 2", Status: model.Phase4IdeaStatusScored},
		},
		TopRecommendations: []model.Phase4IdeaScoreView{
			{ID: "p4idea_1", Title: "Idea 1", OverallScore: 8.2, Rank: 1},
		},
	}, nil
}

func (f *fakePhase4IdeaHandlerService) GenerateRevisionCandidates(_ context.Context, ideaID string, req model.Phase4IdeaRevisionGenerateRequest) (*model.Phase4IdeaRunResult, error) {
	return &model.Phase4IdeaRunResult{
		Job: &model.AgentJob{
			ID:        "job_phase4_idea_revision_001",
			AgentType: "idea_phase4",
			Status:    "succeeded",
			Metadata: map[string]any{
				"source_idea_id": ideaID,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Ideas: []model.Phase4Idea{
			{
				ID:               "p4idea_rev_1",
				Title:            "Idea 1 Revision",
				Status:           model.Phase4IdeaStatusScored,
				RevisionOfID:     ideaID,
				LastFailureRunID: req.LastFailureRunID,
				FailureFeedback:  req.FailureFeedback,
			},
		},
		TopRecommendations: []model.Phase4IdeaScoreView{
			{ID: "p4idea_rev_1", Title: "Idea 1 Revision", OverallScore: 7.9, Rank: 1},
		},
	}, nil
}

func (f *fakePhase4IdeaHandlerService) GetJob(_ context.Context, jobID string) (*model.Phase4IdeaJobDetail, error) {
	return &model.Phase4IdeaJobDetail{
		Job: &model.AgentJob{
			ID:        jobID,
			AgentType: "idea_phase4",
			Status:    "succeeded",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Ideas: []model.Phase4Idea{
			{ID: "p4idea_1", Title: "Idea 1", Status: model.Phase4IdeaStatusScored},
		},
		TopRecommendations: []model.Phase4IdeaScoreView{
			{ID: "p4idea_1", Title: "Idea 1", OverallScore: 8.2, Rank: 1},
		},
	}, nil
}

func TestPhase4IdeaHandlerGenerateRevisionAndGetJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPhase4IdeaHandler(&fakePhase4IdeaHandlerService{})
	router := gin.New()
	router.POST("/api/v2/phase4/ideas/generate", h.Run)
	router.POST("/api/v2/phase4/ideas/:id/revisions/generate", h.GenerateRevisionCandidates)
	router.GET("/api/v2/phase4/ideas/jobs/:id", h.GetJob)

	runResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/ideas/generate", model.Phase4IdeaRunRequest{
		DatasetProfileID: "p4ds_visdom",
		ReaderContextID:  "p4ctx_visdom",
		TargetCount:      10,
	})
	if runResp.Code != 0 {
		t.Fatalf("expected idea generate success, got %+v", runResp)
	}
	var runResult model.Phase4IdeaRunResult
	mustDecodePhase4Data(t, runResp.Data, &runResult)
	if len(runResult.Ideas) != 2 {
		t.Fatalf("expected ideas in generate response, got %d", len(runResult.Ideas))
	}

	revisionResp := doPhase4JSON(t, router, http.MethodPost, "/api/v2/phase4/ideas/p4idea_1/revisions/generate", model.Phase4IdeaRevisionGenerateRequest{
		FailureFeedback:  map[string]any{"error": "test failed after max retries"},
		LastFailureRunID: "p4run_fail_001",
	})
	if revisionResp.Code != 0 {
		t.Fatalf("expected revision generate success, got %+v", revisionResp)
	}
	var revisionResult model.Phase4IdeaRunResult
	mustDecodePhase4Data(t, revisionResp.Data, &revisionResult)
	if len(revisionResult.Ideas) != 1 || revisionResult.Ideas[0].RevisionOfID != "p4idea_1" {
		t.Fatalf("unexpected revision result: %#v", revisionResult.Ideas)
	}

	jobResp := doPhase4JSON(t, router, http.MethodGet, "/api/v2/phase4/ideas/jobs/job_phase4_idea_001", nil)
	if jobResp.Code != 0 {
		t.Fatalf("expected idea job get success, got %+v", jobResp)
	}
	var detail model.Phase4IdeaJobDetail
	mustDecodePhase4Data(t, jobResp.Data, &detail)
	if detail.Job == nil || detail.Job.ID != "job_phase4_idea_001" {
		t.Fatalf("unexpected job detail: %#v", detail.Job)
	}
}
