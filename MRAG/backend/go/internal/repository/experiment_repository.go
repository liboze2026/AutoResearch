package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"mrag-platform/backend/go/internal/model"
)

type ExperimentRepository struct {
	db *sql.DB
}

func NewExperimentRepository(db *sql.DB) *ExperimentRepository {
	return &ExperimentRepository{db: db}
}

func (r *ExperimentRepository) List(ctx context.Context) ([]model.Experiment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,COALESCE(idea_id,''),dataset_asset_id,COALESCE(baseline_id,''),title,status,priority,COALESCE(current_run_id,''),summary_md,owner_note_md,created_at,updated_at FROM experiments ORDER BY priority DESC, updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Experiment, 0)
	for rows.Next() {
		item, scanErr := scanExperiment(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ExperimentRepository) GetByID(ctx context.Context, id string) (*model.Experiment, error) {
	item, err := scanExperiment(r.db.QueryRowContext(ctx, `SELECT id,COALESCE(idea_id,''),dataset_asset_id,COALESCE(baseline_id,''),title,status,priority,COALESCE(current_run_id,''),summary_md,owner_note_md,created_at,updated_at FROM experiments WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ExperimentRepository) Create(ctx context.Context, item model.Experiment) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO experiments (id,idea_id,dataset_asset_id,baseline_id,title,status,priority,current_run_id,summary_md,owner_note_md,created_at,updated_at) VALUES ($1,NULLIF($2,''),$3,NULLIF($4,''),$5,$6,$7,NULLIF($8,''),$9,$10,$11,$12)`,
		item.ID, item.IdeaID, item.DatasetAssetID, item.BaselineID, item.Title, item.Status, item.Priority, item.CurrentRunID, item.SummaryMD, item.OwnerNoteMD, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *ExperimentRepository) Update(ctx context.Context, item model.Experiment) error {
	_, err := r.db.ExecContext(ctx, `UPDATE experiments SET idea_id=NULLIF($2,''),dataset_asset_id=$3,baseline_id=NULLIF($4,''),title=$5,status=$6,priority=$7,current_run_id=NULLIF($8,''),summary_md=$9,owner_note_md=$10,updated_at=$11 WHERE id=$1`,
		item.ID, item.IdeaID, item.DatasetAssetID, item.BaselineID, item.Title, item.Status, item.Priority, item.CurrentRunID, item.SummaryMD, item.OwnerNoteMD, item.UpdatedAt,
	)
	return err
}

func scanExperiment(scanner researchAssetScanner) (model.Experiment, error) {
	var item model.Experiment
	err := scanner.Scan(&item.ID, &item.IdeaID, &item.DatasetAssetID, &item.BaselineID, &item.Title, &item.Status, &item.Priority, &item.CurrentRunID, &item.SummaryMD, &item.OwnerNoteMD, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

type ExperimentSpecRepository struct {
	db *sql.DB
}

func NewExperimentSpecRepository(db *sql.DB) *ExperimentSpecRepository {
	return &ExperimentSpecRepository{db: db}
}

func (r *ExperimentSpecRepository) ListByExperimentID(ctx context.Context, experimentID string) ([]model.ExperimentSpec, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,experiment_id,spec_json,template_type,generated_from,version,created_at,updated_at FROM experiment_specs WHERE experiment_id=$1 ORDER BY version DESC, created_at DESC`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ExperimentSpec, 0)
	for rows.Next() {
		item, scanErr := scanExperimentSpec(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ExperimentSpecRepository) GetByID(ctx context.Context, id string) (*model.ExperimentSpec, error) {
	item, err := scanExperimentSpec(r.db.QueryRowContext(ctx, `SELECT id,experiment_id,spec_json,template_type,generated_from,version,created_at,updated_at FROM experiment_specs WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ExperimentSpecRepository) GetLatestByExperimentID(ctx context.Context, experimentID string) (*model.ExperimentSpec, error) {
	item, err := scanExperimentSpec(r.db.QueryRowContext(ctx, `SELECT id,experiment_id,spec_json,template_type,generated_from,version,created_at,updated_at FROM experiment_specs WHERE experiment_id=$1 ORDER BY version DESC, created_at DESC LIMIT 1`, experimentID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ExperimentSpecRepository) Create(ctx context.Context, item model.ExperimentSpec) error {
	specRaw, _ := json.Marshal(item.SpecJSON)
	generatedRaw, _ := json.Marshal(item.GeneratedFrom)
	_, err := r.db.ExecContext(ctx, `INSERT INTO experiment_specs (id,experiment_id,spec_json,template_type,generated_from,version,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		item.ID, item.ExperimentID, specRaw, item.TemplateType, generatedRaw, item.Version, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func scanExperimentSpec(scanner researchAssetScanner) (model.ExperimentSpec, error) {
	var item model.ExperimentSpec
	var specRaw []byte
	var generatedRaw []byte
	err := scanner.Scan(&item.ID, &item.ExperimentID, &specRaw, &item.TemplateType, &generatedRaw, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.SpecJSON = map[string]interface{}{}
	item.GeneratedFrom = map[string]interface{}{}
	decodeJSON(specRaw, &item.SpecJSON)
	decodeJSON(generatedRaw, &item.GeneratedFrom)
	return item, nil
}
