package repository

import (
	"context"
	"database/sql"

	"mrag-platform/backend/go/internal/model"
)

type DatasetAssetRepository struct {
	db *sql.DB
}

func NewDatasetAssetRepository(db *sql.DB) *DatasetAssetRepository {
	return &DatasetAssetRepository{db: db}
}

func (r *DatasetAssetRepository) List(ctx context.Context) ([]model.DatasetAsset, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT a.id,a.name,a.description_md,a.task_type,a.status,a.source_type,a.local_or_remote_path,a.readme_md,a.loader_note_md,a.schema_note_md,COALESCE(s.existing_dataset_ref,''),COALESCE(d.name,''),a.created_at,a.updated_at FROM dataset_assets a LEFT JOIN dataset_asset_sources s ON s.dataset_asset_id = a.id LEFT JOIN datasets d ON d.id = s.existing_dataset_ref ORDER BY a.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.DatasetAsset, 0)
	for rows.Next() {
		item, scanErr := scanDatasetAsset(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *DatasetAssetRepository) GetByID(ctx context.Context, id string) (*model.DatasetAsset, error) {
	item, err := scanDatasetAsset(r.db.QueryRowContext(ctx, `SELECT a.id,a.name,a.description_md,a.task_type,a.status,a.source_type,a.local_or_remote_path,a.readme_md,a.loader_note_md,a.schema_note_md,COALESCE(s.existing_dataset_ref,''),COALESCE(d.name,''),a.created_at,a.updated_at FROM dataset_assets a LEFT JOIN dataset_asset_sources s ON s.dataset_asset_id = a.id LEFT JOIN datasets d ON d.id = s.existing_dataset_ref WHERE a.id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DatasetAssetRepository) GetByExistingDatasetRef(ctx context.Context, datasetRef string) (*model.DatasetAsset, error) {
	item, err := scanDatasetAsset(r.db.QueryRowContext(ctx, `SELECT a.id,a.name,a.description_md,a.task_type,a.status,a.source_type,a.local_or_remote_path,a.readme_md,a.loader_note_md,a.schema_note_md,COALESCE(s.existing_dataset_ref,''),COALESCE(d.name,''),a.created_at,a.updated_at FROM dataset_assets a JOIN dataset_asset_sources s ON s.dataset_asset_id = a.id LEFT JOIN datasets d ON d.id = s.existing_dataset_ref WHERE s.existing_dataset_ref=$1`, datasetRef))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DatasetAssetRepository) Create(ctx context.Context, asset model.DatasetAsset) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO dataset_assets (id,name,description_md,task_type,status,source_type,local_or_remote_path,readme_md,loader_note_md,schema_note_md,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		asset.ID, asset.Name, asset.DescriptionMD, asset.TaskType, asset.Status, asset.SourceType, asset.LocalOrRemotePath, asset.ReadmeMD, asset.LoaderNoteMD, asset.SchemaNoteMD, asset.CreatedAt, asset.UpdatedAt,
	)
	return err
}

func (r *DatasetAssetRepository) Update(ctx context.Context, asset model.DatasetAsset) error {
	_, err := r.db.ExecContext(ctx, `UPDATE dataset_assets SET name=$2,description_md=$3,task_type=$4,status=$5,source_type=$6,local_or_remote_path=$7,readme_md=$8,loader_note_md=$9,schema_note_md=$10,updated_at=$11 WHERE id=$1`,
		asset.ID, asset.Name, asset.DescriptionMD, asset.TaskType, asset.Status, asset.SourceType, asset.LocalOrRemotePath, asset.ReadmeMD, asset.LoaderNoteMD, asset.SchemaNoteMD, asset.UpdatedAt,
	)
	return err
}

func (r *DatasetAssetRepository) AddSource(ctx context.Context, source model.DatasetAssetSource) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO dataset_asset_sources (dataset_asset_id,existing_dataset_ref,source_kind,created_at,updated_at) VALUES ($1,$2,$3,$4,$5)`,
		source.DatasetAssetID, source.ExistingDatasetRef, source.SourceKind, source.CreatedAt, source.UpdatedAt,
	)
	return err
}

func (r *DatasetAssetRepository) ListSources(ctx context.Context, datasetAssetID string) ([]model.DatasetAssetSource, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT s.id,s.dataset_asset_id,s.existing_dataset_ref,s.source_kind,COALESCE(d.name,''),s.created_at,s.updated_at FROM dataset_asset_sources s LEFT JOIN datasets d ON d.id = s.existing_dataset_ref WHERE s.dataset_asset_id=$1 ORDER BY s.id ASC`, datasetAssetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.DatasetAssetSource, 0)
	for rows.Next() {
		var item model.DatasetAssetSource
		if err = rows.Scan(&item.ID, &item.DatasetAssetID, &item.ExistingDatasetRef, &item.SourceKind, &item.ExistingDatasetName, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func scanDatasetAsset(scanner researchAssetScanner) (model.DatasetAsset, error) {
	var item model.DatasetAsset
	err := scanner.Scan(&item.ID, &item.Name, &item.DescriptionMD, &item.TaskType, &item.Status, &item.SourceType, &item.LocalOrRemotePath, &item.ReadmeMD, &item.LoaderNoteMD, &item.SchemaNoteMD, &item.ExistingDatasetRef, &item.ExistingDatasetName, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}
