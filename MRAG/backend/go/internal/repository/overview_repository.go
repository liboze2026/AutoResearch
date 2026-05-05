package repository

import (
	"context"
	"database/sql"

	"mrag-platform/backend/go/internal/model"
)

type OverviewRepository struct {
	db *sql.DB
}

func NewOverviewRepository(db *sql.DB) *OverviewRepository {
	return &OverviewRepository{db: db}
}

func (r *OverviewRepository) Stats(ctx context.Context, trendDays int) (*model.OverviewStats, error) {
	stats := &model.OverviewStats{
		Trend: make([]model.OverviewTrendPoint, 0, trendDays),
		Notes: []string{},
	}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM datasets").Scan(&stats.DatasetCount); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM datasets WHERE last_scan_at IS NOT NULL").Scan(&stats.ScannedDatasets); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM datasets WHERE index_status <> 'ready'").Scan(&stats.PendingIndexes); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM servers WHERE status='online'").Scan(&stats.ServerOnline); err != nil {
		return nil, err
	}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM servers").Scan(&stats.ServerTotal); err != nil {
		return nil, err
	}

	q := `WITH days AS (
		SELECT generate_series(
			(CURRENT_DATE - ($1::int - 1) * INTERVAL '1 day')::date,
			CURRENT_DATE,
			INTERVAL '1 day'
		)::date AS day
	),
	datasets_by_day AS (
		SELECT date_trunc('day', created_at)::date AS day, COUNT(*) AS total
		FROM datasets
		WHERE created_at >= (CURRENT_DATE - ($1::int - 1) * INTERVAL '1 day')
		GROUP BY 1
	),
	scans_by_day AS (
		SELECT date_trunc('day', scanned_at)::date AS day, COUNT(*) AS total
		FROM dataset_scan_records
		WHERE scanned_at >= (CURRENT_DATE - ($1::int - 1) * INTERVAL '1 day')
		GROUP BY 1
	)
	SELECT to_char(days.day, 'MM-DD') AS day_label,
		COALESCE(datasets_by_day.total, 0),
		COALESCE(scans_by_day.total, 0),
		$2::bigint
	FROM days
	LEFT JOIN datasets_by_day ON datasets_by_day.day = days.day
	LEFT JOIN scans_by_day ON scans_by_day.day = days.day
	ORDER BY days.day ASC`
	rows, err := r.db.QueryContext(ctx, q, trendDays, stats.ServerOnline)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var point model.OverviewTrendPoint
		if err = rows.Scan(&point.Date, &point.Datasets, &point.Scanned, &point.OnlineServers); err != nil {
			return nil, err
		}
		stats.Trend = append(stats.Trend, point)
	}
	return stats, nil
}
