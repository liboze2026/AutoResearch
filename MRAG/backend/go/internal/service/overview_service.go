package service

import (
	"context"

	"mrag-platform/backend/go/internal/model"
)

type OverviewService struct {
	adapter   OverviewStatsAdapter
	trendDays int
}

func NewOverviewService(adapter OverviewStatsAdapter, trendDays int) *OverviewService {
	if trendDays <= 0 {
		trendDays = 7
	}
	return &OverviewService{adapter: adapter, trendDays: trendDays}
}

func (s *OverviewService) Stats(ctx context.Context) (*model.OverviewStats, error) {
	return s.adapter.Stats(ctx, s.trendDays)
}
