package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"mrag-platform/backend/go/internal/model"
)

type ServerHeartbeatRepository struct {
	db *sql.DB
}

func NewServerHeartbeatRepository(db *sql.DB) *ServerHeartbeatRepository {
	return &ServerHeartbeatRepository{db: db}
}

func (r *ServerHeartbeatRepository) ListByServerID(ctx context.Context, serverID string, limit int) ([]model.ServerHeartbeat, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,server_id,heartbeat_at,status,detail_json,created_at,updated_at FROM server_heartbeats WHERE server_id=$1 ORDER BY heartbeat_at DESC LIMIT $2`, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ServerHeartbeat, 0)
	for rows.Next() {
		item, scanErr := scanServerHeartbeat(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ServerHeartbeatRepository) Create(ctx context.Context, item model.ServerHeartbeat) error {
	detailRaw, _ := json.Marshal(item.DetailJSON)
	_, err := r.db.ExecContext(ctx, `INSERT INTO server_heartbeats (id,server_id,heartbeat_at,status,detail_json,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		item.ID, item.ServerID, item.HeartbeatAt, item.Status, detailRaw, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func scanServerHeartbeat(scanner researchAssetScanner) (model.ServerHeartbeat, error) {
	var item model.ServerHeartbeat
	var detailRaw []byte
	err := scanner.Scan(&item.ID, &item.ServerID, &item.HeartbeatAt, &item.Status, &detailRaw, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.DetailJSON = map[string]interface{}{}
	decodeJSON(detailRaw, &item.DetailJSON)
	return item, nil
}

type GPUResourceSnapshotRepository struct {
	db *sql.DB
}

func NewGPUResourceSnapshotRepository(db *sql.DB) *GPUResourceSnapshotRepository {
	return &GPUResourceSnapshotRepository{db: db}
}

func (r *GPUResourceSnapshotRepository) ListByServerID(ctx context.Context, serverID string, limit int) ([]model.GPUResourceSnapshot, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,server_id,captured_at,gpu_index,name,total_mem_mb,free_mem_mb,utilization,process_json,created_at,updated_at FROM gpu_resource_snapshots WHERE server_id=$1 ORDER BY captured_at DESC, gpu_index ASC LIMIT $2`, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.GPUResourceSnapshot, 0)
	for rows.Next() {
		item, scanErr := scanGPUResourceSnapshot(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *GPUResourceSnapshotRepository) Create(ctx context.Context, item model.GPUResourceSnapshot) error {
	processRaw, _ := json.Marshal(item.ProcessJSON)
	_, err := r.db.ExecContext(ctx, `INSERT INTO gpu_resource_snapshots (id,server_id,captured_at,gpu_index,name,total_mem_mb,free_mem_mb,utilization,process_json,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		item.ID, item.ServerID, item.CapturedAt, item.GPUIndex, item.Name, item.TotalMemMB, item.FreeMemMB, item.Utilization, processRaw, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func scanGPUResourceSnapshot(scanner researchAssetScanner) (model.GPUResourceSnapshot, error) {
	var item model.GPUResourceSnapshot
	var processRaw []byte
	err := scanner.Scan(&item.ID, &item.ServerID, &item.CapturedAt, &item.GPUIndex, &item.Name, &item.TotalMemMB, &item.FreeMemMB, &item.Utilization, &processRaw, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.ProcessJSON = []map[string]interface{}{}
	decodeJSON(processRaw, &item.ProcessJSON)
	return item, nil
}
