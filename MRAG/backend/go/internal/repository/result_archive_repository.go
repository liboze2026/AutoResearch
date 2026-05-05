package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"mrag-platform/backend/go/internal/model"
)

type ResultArchiveRepository struct {
	db *sql.DB
}

func NewResultArchiveRepository(db *sql.DB) *ResultArchiveRepository {
	return &ResultArchiveRepository{db: db}
}

func (r *ResultArchiveRepository) List(ctx context.Context) ([]model.ResultArchive, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,title,dataset_asset_id,COALESCE(baseline_id,''),COALESCE(idea_id,''),COALESCE(server_id,''),summary_md,metric_json,status,note_md,created_at,updated_at FROM result_archives ORDER BY updated_at DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ResultArchive, 0)
	for rows.Next() {
		item, scanErr := scanResultArchive(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ResultArchiveRepository) GetByID(ctx context.Context, id string) (*model.ResultArchive, error) {
	item, err := scanResultArchive(r.db.QueryRowContext(ctx, `SELECT id,title,dataset_asset_id,COALESCE(baseline_id,''),COALESCE(idea_id,''),COALESCE(server_id,''),summary_md,metric_json,status,note_md,created_at,updated_at FROM result_archives WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ResultArchiveRepository) ListByDatasetAssetID(ctx context.Context, datasetAssetID string) ([]model.ResultArchive, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,title,dataset_asset_id,COALESCE(baseline_id,''),COALESCE(idea_id,''),COALESCE(server_id,''),summary_md,metric_json,status,note_md,created_at,updated_at FROM result_archives WHERE dataset_asset_id=$1 ORDER BY updated_at DESC, created_at DESC`, datasetAssetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ResultArchive, 0)
	for rows.Next() {
		item, scanErr := scanResultArchive(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ResultArchiveRepository) Create(ctx context.Context, archive model.ResultArchive) error {
	metricRaw, _ := json.Marshal(archive.MetricJSON)
	_, err := r.db.ExecContext(ctx, `INSERT INTO result_archives (id,title,dataset_asset_id,baseline_id,idea_id,server_id,summary_md,metric_json,status,note_md,created_at,updated_at) VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),$7,$8,$9,$10,$11,$12)`,
		archive.ID, archive.Title, archive.DatasetAssetID, archive.BaselineID, archive.IdeaID, archive.ServerID, archive.SummaryMD, metricRaw, archive.Status, archive.NoteMD, archive.CreatedAt, archive.UpdatedAt,
	)
	return err
}

func (r *ResultArchiveRepository) Update(ctx context.Context, archive model.ResultArchive) error {
	metricRaw, _ := json.Marshal(archive.MetricJSON)
	_, err := r.db.ExecContext(ctx, `UPDATE result_archives SET title=$2,dataset_asset_id=$3,baseline_id=NULLIF($4,''),idea_id=NULLIF($5,''),server_id=NULLIF($6,''),summary_md=$7,metric_json=$8,status=$9,note_md=$10,updated_at=$11 WHERE id=$1`,
		archive.ID, archive.Title, archive.DatasetAssetID, archive.BaselineID, archive.IdeaID, archive.ServerID, archive.SummaryMD, metricRaw, archive.Status, archive.NoteMD, archive.UpdatedAt,
	)
	return err
}

func (r *ResultArchiveRepository) AddFile(ctx context.Context, file model.ArchiveFile) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO archive_files (archive_id,file_path,file_kind,checksum,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		file.ArchiveID, file.FilePath, file.FileKind, file.Checksum, file.CreatedAt, file.UpdatedAt,
	)
	return err
}

func (r *ResultArchiveRepository) ListFiles(ctx context.Context, archiveID string) ([]model.ArchiveFile, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,archive_id,file_path,file_kind,checksum,created_at,updated_at FROM archive_files WHERE archive_id=$1 ORDER BY id ASC`, archiveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ArchiveFile, 0)
	for rows.Next() {
		var item model.ArchiveFile
		if err = rows.Scan(&item.ID, &item.ArchiveID, &item.FilePath, &item.FileKind, &item.Checksum, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func scanResultArchive(scanner researchAssetScanner) (model.ResultArchive, error) {
	var item model.ResultArchive
	var metricRaw []byte
	err := scanner.Scan(&item.ID, &item.Title, &item.DatasetAssetID, &item.BaselineID, &item.IdeaID, &item.ServerID, &item.SummaryMD, &metricRaw, &item.Status, &item.NoteMD, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.MetricJSON = map[string]any{}
	decodeJSON(metricRaw, &item.MetricJSON)
	return item, nil
}
