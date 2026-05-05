package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"mrag-platform/backend/go/internal/model"
)

type BaselineRepository struct {
	db *sql.DB
}

func NewBaselineRepository(db *sql.DB) *BaselineRepository {
	return &BaselineRepository{db: db}
}

func (r *BaselineRepository) List(ctx context.Context) ([]model.Baseline, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,dataset_asset_id,name,metric_schema_json,result_json,note_md,source_type,created_at,updated_at FROM baselines ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Baseline, 0)
	for rows.Next() {
		item, scanErr := scanBaseline(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *BaselineRepository) ListByDatasetAssetID(ctx context.Context, datasetAssetID string) ([]model.Baseline, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,dataset_asset_id,name,metric_schema_json,result_json,note_md,source_type,created_at,updated_at FROM baselines WHERE dataset_asset_id=$1 ORDER BY updated_at DESC`, datasetAssetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Baseline, 0)
	for rows.Next() {
		item, scanErr := scanBaseline(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *BaselineRepository) GetByID(ctx context.Context, id string) (*model.Baseline, error) {
	item, err := scanBaseline(r.db.QueryRowContext(ctx, `SELECT id,dataset_asset_id,name,metric_schema_json,result_json,note_md,source_type,created_at,updated_at FROM baselines WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *BaselineRepository) Create(ctx context.Context, baseline model.Baseline) error {
	metricSchemaRaw, _ := json.Marshal(baseline.MetricSchemaJSON)
	resultRaw, _ := json.Marshal(baseline.ResultJSON)
	_, err := r.db.ExecContext(ctx, `INSERT INTO baselines (id,dataset_asset_id,name,metric_schema_json,result_json,note_md,source_type,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		baseline.ID, baseline.DatasetAssetID, baseline.Name, metricSchemaRaw, resultRaw, baseline.NoteMD, baseline.SourceType, baseline.CreatedAt, baseline.UpdatedAt,
	)
	return err
}

func (r *BaselineRepository) Update(ctx context.Context, baseline model.Baseline) error {
	metricSchemaRaw, _ := json.Marshal(baseline.MetricSchemaJSON)
	resultRaw, _ := json.Marshal(baseline.ResultJSON)
	_, err := r.db.ExecContext(ctx, `UPDATE baselines SET name=$2,metric_schema_json=$3,result_json=$4,note_md=$5,source_type=$6,updated_at=$7 WHERE id=$1`,
		baseline.ID, baseline.Name, metricSchemaRaw, resultRaw, baseline.NoteMD, baseline.SourceType, baseline.UpdatedAt,
	)
	return err
}

func scanBaseline(scanner researchAssetScanner) (model.Baseline, error) {
	var item model.Baseline
	var metricSchemaRaw []byte
	var resultRaw []byte
	err := scanner.Scan(&item.ID, &item.DatasetAssetID, &item.Name, &metricSchemaRaw, &resultRaw, &item.NoteMD, &item.SourceType, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.MetricSchemaJSON = map[string]any{}
	item.ResultJSON = map[string]any{}
	decodeJSON(metricSchemaRaw, &item.MetricSchemaJSON)
	decodeJSON(resultRaw, &item.ResultJSON)
	return item, nil
}
