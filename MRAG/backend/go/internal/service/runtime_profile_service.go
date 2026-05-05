package service

import (
	"context"
	"time"

	"mrag-platform/backend/go/internal/config"
	"mrag-platform/backend/go/internal/model"
)

type RuntimeProfileService struct {
	cfg config.AppConfig
}

func NewRuntimeProfileService(cfg config.AppConfig) *RuntimeProfileService {
	return &RuntimeProfileService{cfg: cfg}
}

func (s *RuntimeProfileService) Profile(ctx context.Context) (*model.RuntimeProfile, error) {
	modes := []model.RuntimeModeItem{
		{
			Key:          "remote_execution",
			Label:        "远程实验执行",
			Mode:         s.cfg.RemoteExecutionMode,
			Summary:      modeSummary(s.cfg.RemoteExecutionMode, "remote 实验由演示适配器直接返回状态，不会通过 SSH 提交", "remote 实验由 Go 后端通过 SSH 提交到目标服务器"),
			RealBehavior: "通过 SSH 在远程工作目录执行约定命令，不再回落本地 Python 服务。",
			MockBehavior: "返回演示状态与路径信息，便于前端联调和演示。",
		},
		{
			Key:          "dataset_scan",
			Label:        "数据集扫描",
			Mode:         s.cfg.DatasetScanMode,
			Summary:      modeSummary(s.cfg.DatasetScanMode, "路径校验、扫描摘要和样本预览使用演示适配器返回", "local 直接扫描本机目录，remote 通过 SSH 在远程执行扫描命令"),
			RealBehavior: "校验真实路径存在性并返回真实扫描摘要、文件类型分布与预览项。",
			MockBehavior: "返回稳定的演示扫描摘要与预览项，不访问实际目录。",
		},
		{
			Key:          "dataset_index",
			Label:        "索引构建",
			Mode:         s.cfg.DatasetIndexMode,
			Summary:      modeSummary(s.cfg.DatasetIndexMode, "索引任务状态由演示适配器推进", "索引任务记录真实创建；remote 通过 SSH 调远程契约，local 走本地占位构建器"),
			RealBehavior: "创建真实任务记录、日志和状态流转，并为后续真实索引算法预留接入点。",
			MockBehavior: "直接返回可演示的任务状态变化，用于页面验收与联调。",
		},
		{
			Key:          "overview_stats",
			Label:        "总览统计",
			Mode:         s.cfg.OverviewStatsMode,
			Summary:      modeSummary(s.cfg.OverviewStatsMode, "总览页展示后端演示统计数据", "总览页展示数据库聚合统计与服务器状态汇总"),
			RealBehavior: "从数据库和服务状态汇总接口生成统计卡片与趋势图。",
			MockBehavior: "由后端统一返回演示统计，不再由前端写死。",
		},
	}
	profile := &model.RuntimeProfile{
		Preset:               runtimePreset(modes),
		GeneratedAt:          time.Now(),
		RemoteExecutionMode:  s.cfg.RemoteExecutionMode,
		DatasetScanMode:      s.cfg.DatasetScanMode,
		DatasetIndexMode:     s.cfg.DatasetIndexMode,
		OverviewStatsMode:    s.cfg.OverviewStatsMode,
		ServerConnectionMode: s.cfg.SSHClientMode,
		Modes:                modes,
		Notes: []string{
			"当前系统支持按能力维度分别切换 mock 和 real，不需要改业务代码。",
			"server connection mode 仍独立使用 SSH_CLIENT_MODE，用于服务器连接测试与远程命令执行诊断。",
			"远程实验算法与远程索引算法的业务实现仍由远程 Python 侧后续补齐；本地已完成契约、任务记录和调度链路。",
		},
	}
	return profile, nil
}

func modeSummary(mode string, mockText string, realText string) string {
	if mode == "mock" {
		return mockText
	}
	return realText
}

func runtimePreset(items []model.RuntimeModeItem) string {
	mockCount := 0
	for _, item := range items {
		if item.Mode == "mock" {
			mockCount++
		}
	}
	switch {
	case mockCount == 0:
		return "all-real"
	case mockCount == len(items):
		return "all-mock"
	default:
		return "mixed"
	}
}
