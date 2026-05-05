package service

import (
	"context"
	"fmt"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/repository"
)

type OverviewStatsAdapter interface {
	Mode() string
	Stats(ctx context.Context, trendDays int) (*model.OverviewStats, error)
}

func NewOverviewStatsAdapter(mode string, repo *repository.OverviewRepository) OverviewStatsAdapter {
	if normalizeMode(mode) == "mock" {
		return &MockOverviewStatsAdapter{}
	}
	return &RealOverviewStatsAdapter{repo: repo}
}

type RealOverviewStatsAdapter struct {
	repo *repository.OverviewRepository
}

func (a *RealOverviewStatsAdapter) Mode() string {
	return "real"
}

func (a *RealOverviewStatsAdapter) Stats(ctx context.Context, trendDays int) (*model.OverviewStats, error) {
	stats, err := a.repo.Stats(ctx, trendDays)
	if err != nil {
		return nil, err
	}
	stats.StatsMode = a.Mode()
	stats.StatsGeneratedAt = time.Now()
	stats.PlatformIntro = "总览统计现在聚焦数据集资产、扫描覆盖率和服务器可用性，不再展示实验与结果维度。"
	stats.Notes = []string{
		fmt.Sprintf("趋势图统计最近 %d 天的数据集新增量、扫描次数和在线服务器数量。", trendDays),
		"服务器在线数量以最近一次连接探测后的节点状态为准。",
		"待构建索引表示 indexStatus 不是 ready 的数据集数量。",
	}
	return stats, nil
}

type MockOverviewStatsAdapter struct{}

func (a *MockOverviewStatsAdapter) Mode() string {
	return "mock"
}

func (a *MockOverviewStatsAdapter) Stats(ctx context.Context, trendDays int) (*model.OverviewStats, error) {
	now := time.Now()
	trend := make([]model.OverviewTrendPoint, 0, trendDays)
	baseDatasets := []int64{1, 1, 2, 2, 3, 3, 4, 5, 5, 6}
	baseScanned := []int64{0, 1, 1, 2, 2, 3, 2, 4, 3, 4}
	baseServers := []int64{1, 1, 1, 2, 2, 1, 2, 2, 2, 2}
	for i := trendDays - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		idx := (trendDays - 1) - i
		trend = append(trend, model.OverviewTrendPoint{
			Date:          day.Format("01-02"),
			Datasets:      baseDatasets[idx%len(baseDatasets)],
			Scanned:       baseScanned[idx%len(baseScanned)],
			OnlineServers: baseServers[idx%len(baseServers)],
		})
	}
	return &model.OverviewStats{
		PlatformIntro:    "当前总览使用后端演示统计数据，用于未接入真实数据库或联调环境下的页面验证。",
		StatsMode:        a.Mode(),
		StatsGeneratedAt: now,
		DatasetCount:     6,
		ScannedDatasets:  5,
		PendingIndexes:   2,
		ServerOnline:     1,
		ServerTotal:      2,
		Trend:            trend,
		Notes: []string{
			"演示统计由后端 mock adapter 输出，前端不再维护写死的实验/结果数据。",
			"将 OVERVIEW_STATS_MODE 设为 real 后，可切换为数据库聚合统计。",
		},
	}, nil
}
