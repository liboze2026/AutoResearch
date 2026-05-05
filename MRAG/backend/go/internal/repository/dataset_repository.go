package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"mrag-platform/backend/go/internal/model"
)

type DatasetRepository struct {
	db *sql.DB
}

func NewDatasetRepository(db *sql.DB) *DatasetRepository {
	return &DatasetRepository{db: db}
}

func (r *DatasetRepository) List(ctx context.Context, keyword string, sourceType string, modality string) ([]model.Dataset, error) {
	q := `SELECT d.id,d.name,d.tags,d.source_type,d.modality,d.version,d.size_text,d.sample_count,d.description,d.path,
		COALESCE(d.server_id,''),COALESCE(s.name,''),d.index_status,d.file_count,d.directory_count,d.total_size_bytes,
		d.file_types,d.detected_modality,d.last_scan_status,d.last_scan_at,d.last_modified_at,d.updated_at
	FROM datasets d
	LEFT JOIN servers s ON s.id = d.server_id
	WHERE 1=1`
	args := make([]interface{}, 0)
	idx := 1
	if keyword != "" {
		q += fmt.Sprintf(" AND LOWER(d.name) LIKE LOWER($%d)", idx)
		args = append(args, "%"+keyword+"%")
		idx++
	}
	if sourceType != "" {
		q += fmt.Sprintf(" AND d.source_type=$%d", idx)
		args = append(args, sourceType)
		idx++
	}
	if modality != "" {
		q += fmt.Sprintf(" AND d.modality=$%d", idx)
		args = append(args, modality)
		idx++
	}
	q += " ORDER BY d.updated_at DESC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]model.Dataset, 0)
	for rows.Next() {
		item, scanErr := scanDatasetRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		list = append(list, item)
	}
	return list, nil
}

func (r *DatasetRepository) GetByID(ctx context.Context, id string) (*model.Dataset, error) {
	return r.GetSummaryByID(ctx, id)
}

func (r *DatasetRepository) GetSummaryByID(ctx context.Context, id string) (*model.Dataset, error) {
	q := `SELECT d.id,d.name,d.tags,d.source_type,d.modality,d.version,d.size_text,d.sample_count,d.description,d.path,
		COALESCE(d.server_id,''),COALESCE(s.name,''),d.index_status,d.file_count,d.directory_count,d.total_size_bytes,
		d.file_types,d.detected_modality,d.last_scan_status,d.last_scan_at,d.last_modified_at,d.updated_at
	FROM datasets d
	LEFT JOIN servers s ON s.id = d.server_id
	WHERE d.id=$1`
	item, err := scanDatasetRow(r.db.QueryRowContext(ctx, q, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DatasetRepository) GetDetail(ctx context.Context, id string) (*model.DatasetDetail, error) {
	dataset, err := r.GetSummaryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dataset == nil {
		return nil, nil
	}
	latestScan, err := r.GetLatestScanRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	previewItems := make([]model.DatasetPreviewItem, 0)
	if latestScan != nil {
		previewItems, err = r.ListPreviewItemsByScan(ctx, latestScan.ID)
		if err != nil {
			return nil, err
		}
	}
	indexTasks, err := r.ListIndexTasks(ctx, id, 10)
	if err != nil {
		return nil, err
	}
	var latestTask *model.DatasetIndexTask
	if len(indexTasks) > 0 {
		latestTask = &indexTasks[0]
		latestTask.Logs, err = r.ListIndexTaskLogs(ctx, latestTask.ID)
		if err != nil {
			return nil, err
		}
	}
	return &model.DatasetDetail{
		Dataset:         *dataset,
		LatestScan:      latestScan,
		PreviewItems:    previewItems,
		LatestIndexTask: latestTask,
		IndexTasks:      indexTasks,
	}, nil
}

func (r *DatasetRepository) CreateImported(ctx context.Context, dataset model.Dataset, scan model.DatasetScanRecord, previews []model.DatasetPreviewItem) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tagsRaw, _ := json.Marshal(dataset.Tags)
	fileTypesRaw, _ := json.Marshal(dataset.FileTypes)
	hierarchyRaw, _ := json.Marshal(scan.HierarchySummary)
	scanFileTypesRaw, _ := json.Marshal(scan.FileTypes)
	rawPayload, _ := json.Marshal(map[string]interface{}{})

	datasetQuery := `INSERT INTO datasets (
		id,name,tags,source_type,modality,version,size_text,sample_count,description,path,server_id,index_status,
		file_count,directory_count,total_size_bytes,file_types,detected_modality,hierarchy_summary,last_scan_status,last_scan_at,last_modified_at
	) VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),$12,$13,$14,$15,$16,$17,$18,$19,$20,$21
	)`
	if _, err = tx.ExecContext(ctx, datasetQuery,
		dataset.ID, dataset.Name, tagsRaw, strings.ToLower(dataset.SourceType), strings.ToLower(dataset.Modality), dataset.Version,
		dataset.Size, dataset.Samples, dataset.Description, dataset.Path, dataset.ServerID, dataset.IndexStatus,
		dataset.FileCount, dataset.DirectoryCount, dataset.TotalSizeBytes, fileTypesRaw, dataset.DetectedModality, hierarchyRaw,
		dataset.LastScanStatus, dataset.LastScanAt, dataset.LastModifiedAt,
	); err != nil {
		return err
	}

	scanQuery := `INSERT INTO dataset_scan_records (
		id,dataset_id,server_id,runtime_mode,scan_status,validation_status,root_path,file_count,directory_count,total_size_bytes,
		file_types,hierarchy_summary,inferred_modality,recent_modified_at,error_message,raw_payload,scanned_at
	) VALUES (
		$1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17
	)`
	if _, err = tx.ExecContext(ctx, scanQuery,
		scan.ID, scan.DatasetID, scan.ServerID, scan.RuntimeMode, scan.ScanStatus, scan.ValidationStatus, scan.RootPath,
		scan.FileCount, scan.DirectoryCount, scan.TotalSizeBytes, scanFileTypesRaw, hierarchyRaw, scan.InferredModality,
		scan.RecentModifiedAt, scan.ErrorMessage, rawPayload, scan.ScannedAt,
	); err != nil {
		return err
	}

	previewQuery := `INSERT INTO dataset_preview_items (dataset_id,scan_record_id,item_name,item_type,item_category,relative_path,size_bytes,depth)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	for _, item := range previews {
		if _, err = tx.ExecContext(ctx, previewQuery, dataset.ID, scan.ID, item.Name, item.ItemType, item.Category, item.RelativePath, item.SizeBytes, item.Depth); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *DatasetRepository) UpdateMetadata(ctx context.Context, id string, req model.DatasetUpdateRequest) error {
	tagsRaw, _ := json.Marshal(req.Tags)
	_, err := r.db.ExecContext(ctx, `UPDATE datasets SET name=$2,description=$3,tags=$4,modality=COALESCE(NULLIF($5,''),modality),version=COALESCE(NULLIF($6,''),version),updated_at=now() WHERE id=$1`, id, req.Name, req.Description, tagsRaw, strings.ToLower(req.Modality), req.Version)
	return err
}

func (r *DatasetRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM datasets WHERE id=$1", id)
	return err
}

func (r *DatasetRepository) GetScanRecordByID(ctx context.Context, scanRecordID string) (*model.DatasetScanRecord, error) {
	q := `SELECT id,dataset_id,COALESCE(server_id,''),runtime_mode,scan_status,validation_status,root_path,file_count,directory_count,total_size_bytes,
		file_types,hierarchy_summary,inferred_modality,recent_modified_at,scanned_at,error_message
	FROM dataset_scan_records WHERE id=$1`
	item, err := scanDatasetScanRecord(r.db.QueryRowContext(ctx, q, scanRecordID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DatasetRepository) GetLatestScanRecord(ctx context.Context, datasetID string) (*model.DatasetScanRecord, error) {
	q := `SELECT id,dataset_id,COALESCE(server_id,''),runtime_mode,scan_status,validation_status,root_path,file_count,directory_count,total_size_bytes,
		file_types,hierarchy_summary,inferred_modality,recent_modified_at,scanned_at,error_message
	FROM dataset_scan_records WHERE dataset_id=$1 ORDER BY scanned_at DESC LIMIT 1`
	item, err := scanDatasetScanRecord(r.db.QueryRowContext(ctx, q, datasetID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DatasetRepository) ListPreviewItemsByScan(ctx context.Context, scanRecordID string) ([]model.DatasetPreviewItem, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,scan_record_id,item_name,item_type,item_category,relative_path,size_bytes,depth FROM dataset_preview_items WHERE scan_record_id=$1 ORDER BY id ASC`, scanRecordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.DatasetPreviewItem, 0)
	for rows.Next() {
		var item model.DatasetPreviewItem
		if err = rows.Scan(&item.ID, &item.ScanRecordID, &item.Name, &item.ItemType, &item.Category, &item.RelativePath, &item.SizeBytes, &item.Depth); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *DatasetRepository) CreateIndexTask(ctx context.Context, task model.DatasetIndexTask) error {
	requestRaw, _ := json.Marshal(task.RequestPayload)
	responseRaw, _ := json.Marshal(task.ResponsePayload)
	_, err := r.db.ExecContext(ctx, `INSERT INTO dataset_index_tasks (
		id,dataset_id,server_id,source_type,executor_mode,status,remote_task_id,log_path,status_path,result_path,error_message,request_payload,response_payload,finished_at
	) VALUES (
		$1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14
	)`, task.ID, task.DatasetID, task.ServerID, task.SourceType, task.ExecutorMode, task.Status, task.RemoteTaskID, task.LogPath, task.StatusPath, task.ResultPath, task.ErrorMessage, requestRaw, responseRaw, task.FinishedAt)
	return err
}

func (r *DatasetRepository) UpdateIndexTask(ctx context.Context, task model.DatasetIndexTask) error {
	requestRaw, _ := json.Marshal(task.RequestPayload)
	responseRaw, _ := json.Marshal(task.ResponsePayload)
	_, err := r.db.ExecContext(ctx, `UPDATE dataset_index_tasks SET
		server_id=NULLIF($2,''),source_type=$3,executor_mode=$4,status=$5,remote_task_id=$6,log_path=$7,status_path=$8,result_path=$9,error_message=$10,
		request_payload=$11,response_payload=$12,updated_at=now(),finished_at=$13
	WHERE id=$1`, task.ID, task.ServerID, task.SourceType, task.ExecutorMode, task.Status, task.RemoteTaskID, task.LogPath, task.StatusPath, task.ResultPath, task.ErrorMessage, requestRaw, responseRaw, task.FinishedAt)
	return err
}

func (r *DatasetRepository) GetIndexTaskByID(ctx context.Context, taskID string) (*model.DatasetIndexTask, error) {
	q := `SELECT id,dataset_id,COALESCE(server_id,''),source_type,executor_mode,status,remote_task_id,log_path,status_path,result_path,error_message,request_payload,response_payload,created_at,updated_at,finished_at
	FROM dataset_index_tasks WHERE id=$1`
	item, err := scanDatasetIndexTask(r.db.QueryRowContext(ctx, q, taskID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.Logs, err = r.ListIndexTaskLogs(ctx, item.ID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DatasetRepository) ListIndexTasks(ctx context.Context, datasetID string, limit int) ([]model.DatasetIndexTask, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,dataset_id,COALESCE(server_id,''),source_type,executor_mode,status,remote_task_id,log_path,status_path,result_path,error_message,request_payload,response_payload,created_at,updated_at,finished_at FROM dataset_index_tasks WHERE dataset_id=$1 ORDER BY created_at DESC LIMIT $2`, datasetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.DatasetIndexTask, 0)
	for rows.Next() {
		item, scanErr := scanDatasetIndexTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *DatasetRepository) AddIndexTaskLog(ctx context.Context, taskID string, level string, content string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO dataset_index_task_logs (task_id,level,content) VALUES ($1,$2,$3)`, taskID, level, content)
	return err
}

func (r *DatasetRepository) ListIndexTaskLogs(ctx context.Context, taskID string) ([]model.DatasetIndexTaskLog, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,task_id,level,content,created_at FROM dataset_index_task_logs WHERE task_id=$1 ORDER BY id ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.DatasetIndexTaskLog, 0)
	for rows.Next() {
		var item model.DatasetIndexTaskLog
		if err = rows.Scan(&item.ID, &item.TaskID, &item.Level, &item.Content, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *DatasetRepository) UpdateDatasetIndexStatus(ctx context.Context, datasetID string, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE datasets SET index_status=$2,updated_at=now() WHERE id=$1`, datasetID, status)
	return err
}

type datasetScanner interface {
	Scan(dest ...interface{}) error
}

func scanDatasetRow(scanner datasetScanner) (model.Dataset, error) {
	var item model.Dataset
	var tagsRaw []byte
	var fileTypesRaw []byte
	var serverID string
	var serverName string
	err := scanner.Scan(
		&item.ID, &item.Name, &tagsRaw, &item.SourceType, &item.Modality, &item.Version, &item.Size, &item.Samples,
		&item.Description, &item.Path, &serverID, &serverName, &item.IndexStatus, &item.FileCount, &item.DirectoryCount,
		&item.TotalSizeBytes, &fileTypesRaw, &item.DetectedModality, &item.LastScanStatus, &item.LastScanAt, &item.LastModifiedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	item.ServerID = serverID
	item.ServerName = serverName
	item.Tags = []string{}
	item.FileTypes = map[string]int64{}
	_ = json.Unmarshal(tagsRaw, &item.Tags)
	_ = json.Unmarshal(fileTypesRaw, &item.FileTypes)
	return item, nil
}

func scanDatasetScanRecord(scanner datasetScanner) (model.DatasetScanRecord, error) {
	var item model.DatasetScanRecord
	var serverID string
	var fileTypesRaw []byte
	var hierarchyRaw []byte
	err := scanner.Scan(&item.ID, &item.DatasetID, &serverID, &item.RuntimeMode, &item.ScanStatus, &item.ValidationStatus, &item.RootPath, &item.FileCount, &item.DirectoryCount, &item.TotalSizeBytes, &fileTypesRaw, &hierarchyRaw, &item.InferredModality, &item.RecentModifiedAt, &item.ScannedAt, &item.ErrorMessage)
	if err != nil {
		return item, err
	}
	item.ServerID = serverID
	item.FileTypes = map[string]int64{}
	item.HierarchySummary = []model.DatasetHierarchySummaryItem{}
	_ = json.Unmarshal(fileTypesRaw, &item.FileTypes)
	_ = json.Unmarshal(hierarchyRaw, &item.HierarchySummary)
	return item, nil
}

func scanDatasetIndexTask(scanner datasetScanner) (model.DatasetIndexTask, error) {
	var item model.DatasetIndexTask
	var serverID string
	var requestRaw []byte
	var responseRaw []byte
	err := scanner.Scan(&item.ID, &item.DatasetID, &serverID, &item.SourceType, &item.ExecutorMode, &item.Status, &item.RemoteTaskID, &item.LogPath, &item.StatusPath, &item.ResultPath, &item.ErrorMessage, &requestRaw, &responseRaw, &item.CreatedAt, &item.UpdatedAt, &item.FinishedAt)
	if err != nil {
		return item, err
	}
	item.ServerID = serverID
	item.RequestPayload = map[string]interface{}{}
	item.ResponsePayload = map[string]interface{}{}
	item.Logs = []model.DatasetIndexTaskLog{}
	_ = json.Unmarshal(requestRaw, &item.RequestPayload)
	_ = json.Unmarshal(responseRaw, &item.ResponsePayload)
	return item, nil
}
