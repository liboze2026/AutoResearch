package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type ServerRepository struct {
	db *sql.DB
}

func NewServerRepository(db *sql.DB) *ServerRepository {
	return &ServerRepository{db: db}
}

func (r *ServerRepository) List(ctx context.Context) ([]model.Server, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,host,ssh_port,username,auth_type,(password_cipher <> '') AS has_password,status,status_message,gpu_info,remote_root,task_workdir,config,available_gpus,total_gpus,last_heartbeat,last_gpu_check_at FROM servers ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]model.Server, 0)
	for rows.Next() {
		item, scanErr := scanServerRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		list = append(list, item)
	}
	return list, nil
}

func (r *ServerRepository) GetByID(ctx context.Context, id string) (*model.Server, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,name,host,ssh_port,username,auth_type,(password_cipher <> '') AS has_password,status,status_message,gpu_info,remote_root,task_workdir,config,available_gpus,total_gpus,last_heartbeat,last_gpu_check_at FROM servers WHERE id=$1`, id)
	item, err := scanServerRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ServerRepository) GetByIDWithSecrets(ctx context.Context, id string) (*model.Server, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,name,host,ssh_port,username,auth_type,password_cipher,private_key_cipher,(password_cipher <> '') AS has_password,status,status_message,gpu_info,remote_root,task_workdir,config,available_gpus,total_gpus,last_heartbeat,last_gpu_check_at FROM servers WHERE id=$1`, id)
	item, err := scanServerRowWithSecrets(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ServerRepository) Create(ctx context.Context, req model.Server) (*model.Server, error) {
	id := httpx.NewID("srv")
	if req.AuthType == "" {
		req.AuthType = "ssh_config"
	}
	configRaw, err := json.Marshal(defaultServerConfig(req.Config))
	if err != nil {
		return nil, err
	}
	q := `INSERT INTO servers (id,name,host,ssh_port,username,auth_type,password_cipher,private_key_cipher,status,status_message,gpu_info,remote_root,task_workdir,config,available_gpus,total_gpus,last_heartbeat,last_gpu_check_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,0,0,now(),NULL)`
	_, err = r.db.ExecContext(ctx, q, id, req.Name, req.Host, req.SSHPort, req.Username, req.AuthType, req.Password, req.PrivateKey, defaultServerStatus(req.Status), req.StatusMessage, req.GPUInfo, req.RemoteRoot, req.TaskWorkdir, string(configRaw))
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *ServerRepository) Update(ctx context.Context, id string, req model.Server) (*model.Server, error) {
	if req.AuthType == "" {
		req.AuthType = "ssh_config"
	}
	configRaw, err := json.Marshal(defaultServerConfig(req.Config))
	if err != nil {
		return nil, err
	}
	q := `UPDATE servers
	SET name=$2,
		host=$3,
		ssh_port=$4,
		username=$5,
		auth_type=$6::text,
		password_cipher=CASE
			WHEN $6::text = 'password' AND $7::text <> '' THEN $7::text
			WHEN $6::text = 'password' THEN password_cipher
			ELSE ''
		END,
		private_key_cipher=CASE
			WHEN $6::text = 'key' AND $8::text <> '' THEN $8::text
			WHEN $6::text = 'key' THEN private_key_cipher
			ELSE ''
		END,
		remote_root=$9,
		task_workdir=$10,
		config=$11::jsonb,
		updated_at=now()
	WHERE id=$1`
	res, err := r.db.ExecContext(ctx, q, id, req.Name, req.Host, req.SSHPort, req.Username, req.AuthType, req.Password, req.PrivateKey, req.RemoteRoot, req.TaskWorkdir, string(configRaw))
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, nil
	}
	return r.GetByID(ctx, id)
}

func (r *ServerRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM servers WHERE id=$1", id)
	return err
}

func (r *ServerRepository) UpdateStatus(ctx context.Context, id string, status string, message string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE servers SET status=$2,status_message=$3,last_heartbeat=now(),updated_at=now() WHERE id=$1", id, status, message)
	return err
}

func (r *ServerRepository) UpdateGPUInfo(ctx context.Context, id string, probe *model.GPUProbeResult) error {
	if probe == nil {
		return nil
	}
	gpuInfo := probe.Summary
	if len(probe.Devices) > 0 {
		gpuInfo = fmt.Sprintf("%d/%d \u53ef\u7528", probe.AvailableGPUCount, probe.TotalGPUCount)
	}
	_, err := r.db.ExecContext(ctx, "UPDATE servers SET gpu_info=$2,available_gpus=$3,total_gpus=$4,last_gpu_check_at=$5,updated_at=now() WHERE id=$1", id, gpuInfo, probe.AvailableGPUCount, probe.TotalGPUCount, probe.CheckedAt)
	return err
}

type serverScanner interface {
	Scan(dest ...interface{}) error
}

func scanServerRow(scanner serverScanner) (model.Server, error) {
	var item model.Server
	var configRaw []byte
	err := scanner.Scan(&item.ID, &item.Name, &item.Host, &item.SSHPort, &item.Username, &item.AuthType, &item.HasPassword, &item.Status, &item.StatusMessage, &item.GPUInfo, &item.RemoteRoot, &item.TaskWorkdir, &configRaw, &item.AvailableGPUs, &item.TotalGPUs, &item.LastHeartbeat, &item.LastGPUCheckAt)
	if err != nil {
		return item, err
	}
	item.Config = map[string]interface{}{}
	if len(configRaw) > 0 {
		_ = json.Unmarshal(configRaw, &item.Config)
	}
	return item, nil
}

func scanServerRowWithSecrets(scanner serverScanner) (model.Server, error) {
	var item model.Server
	var configRaw []byte
	err := scanner.Scan(&item.ID, &item.Name, &item.Host, &item.SSHPort, &item.Username, &item.AuthType, &item.Password, &item.PrivateKey, &item.HasPassword, &item.Status, &item.StatusMessage, &item.GPUInfo, &item.RemoteRoot, &item.TaskWorkdir, &configRaw, &item.AvailableGPUs, &item.TotalGPUs, &item.LastHeartbeat, &item.LastGPUCheckAt)
	if err != nil {
		return item, err
	}
	item.Config = map[string]interface{}{}
	if len(configRaw) > 0 {
		_ = json.Unmarshal(configRaw, &item.Config)
	}
	return item, nil
}

func defaultServerConfig(input map[string]interface{}) map[string]interface{} {
	if input == nil {
		return map[string]interface{}{}
	}
	return input
}

func defaultServerStatus(value string) string {
	if value == "" {
		return "offline"
	}
	return value
}

func (r *ServerRepository) ValidateRunMode(ctx context.Context, runMode string, serverID string) error {
	if runMode != "remote" {
		return nil
	}
	if serverID == "" {
		return fmt.Errorf("serverId is required when runMode=remote")
	}
	var count int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM servers WHERE id=$1", serverID).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("server not found: %s", serverID)
	}
	return nil
}