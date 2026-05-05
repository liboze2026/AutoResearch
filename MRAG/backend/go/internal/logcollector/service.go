package logcollector

import (
	"context"
	"strings"

	"mrag-platform/backend/go/internal/model"
)

type runLogReader interface {
	ListByRunID(context.Context, string) ([]model.RunLog, error)
}

type Service struct {
	logs runLogReader
}

func NewService(logs runLogReader) *Service {
	return &Service{logs: logs}
}

func (s *Service) ListByRunID(ctx context.Context, runID string) ([]model.RunLog, error) {
	return s.logs.ListByRunID(ctx, runID)
}

func (s *Service) Tail(ctx context.Context, runID string, logType string) (string, error) {
	items, err := s.logs.ListByRunID(ctx, runID)
	if err != nil {
		return "", err
	}
	targetType := strings.TrimSpace(logType)
	if targetType == "" {
		targetType = "stdout"
	}
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].LogType == targetType {
			return items[i].TailText, nil
		}
	}
	return "", nil
}
