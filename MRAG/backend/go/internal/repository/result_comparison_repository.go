package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"mrag-platform/backend/go/internal/model"
)

type ResultComparisonRepository struct {
	db *sql.DB
}

func NewResultComparisonRepository(db *sql.DB) *ResultComparisonRepository {
	return &ResultComparisonRepository{db: db}
}

func (r *ResultComparisonRepository) ListByExperimentID(ctx context.Context, experimentID string) ([]model.ResultComparison, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,experiment_id,run_id,COALESCE(baseline_id,''),COALESCE(target_result_archive_id,''),comparison_json,summary_md,created_at,updated_at FROM result_comparisons WHERE experiment_id=$1 ORDER BY created_at DESC`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ResultComparison, 0)
	for rows.Next() {
		item, scanErr := scanResultComparison(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ResultComparisonRepository) GetByID(ctx context.Context, id string) (*model.ResultComparison, error) {
	item, err := scanResultComparison(r.db.QueryRowContext(ctx, `SELECT id,experiment_id,run_id,COALESCE(baseline_id,''),COALESCE(target_result_archive_id,''),comparison_json,summary_md,created_at,updated_at FROM result_comparisons WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ResultComparisonRepository) Create(ctx context.Context, item model.ResultComparison) error {
	comparisonRaw, _ := json.Marshal(item.ComparisonJSON)
	_, err := r.db.ExecContext(ctx, `INSERT INTO result_comparisons (id,experiment_id,run_id,baseline_id,target_result_archive_id,comparison_json,summary_md,created_at,updated_at) VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),$6,$7,$8,$9)`,
		item.ID, item.ExperimentID, item.RunID, item.BaselineID, item.TargetResultArchiveID, comparisonRaw, item.SummaryMD, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func scanResultComparison(scanner researchAssetScanner) (model.ResultComparison, error) {
	var item model.ResultComparison
	var comparisonRaw []byte
	err := scanner.Scan(&item.ID, &item.ExperimentID, &item.RunID, &item.BaselineID, &item.TargetResultArchiveID, &comparisonRaw, &item.SummaryMD, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.ComparisonJSON = map[string]interface{}{}
	decodeJSON(comparisonRaw, &item.ComparisonJSON)
	return item, nil
}
