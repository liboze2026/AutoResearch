package agentartifact

import (
	"context"

	"mrag-platform/backend/go/internal/model"
)

type artifactStore interface {
	Create(context.Context, model.AgentArtifact) error
	ListByJobID(context.Context, string) ([]model.AgentArtifact, error)
}

type Service struct {
	store artifactStore
}

func NewService(store artifactStore) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, item model.AgentArtifact) error {
	return s.store.Create(ctx, item)
}

func (s *Service) ListByJobID(ctx context.Context, jobID string) ([]model.AgentArtifact, error) {
	return s.store.ListByJobID(ctx, jobID)
}
