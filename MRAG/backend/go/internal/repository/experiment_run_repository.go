package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"mrag-platform/backend/go/internal/model"
)

type ExperimentRunRepository struct {
	db *sql.DB
}

func NewExperimentRunRepository(db *sql.DB) *ExperimentRunRepository {
	return &ExperimentRunRepository{db: db}
}

func (r *ExperimentRunRepository) ListByExperimentID(ctx context.Context, experimentID string) ([]model.ExperimentRun, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,experiment_id,COALESCE(spec_id,''),COALESCE(assigned_server_id,''),run_status,remote_workdir,remote_job_id,started_at,ended_at,retry_count,exit_code,result_json,error_message,created_at,updated_at FROM experiment_runs WHERE experiment_id=$1 ORDER BY created_at DESC`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ExperimentRun, 0)
	for rows.Next() {
		item, scanErr := scanExperimentRun(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ExperimentRunRepository) CountByExperimentID(ctx context.Context, experimentID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM experiment_runs WHERE experiment_id=$1`, experimentID).Scan(&count)
	return count, err
}

func (r *ExperimentRunRepository) GetByID(ctx context.Context, id string) (*model.ExperimentRun, error) {
	item, err := scanExperimentRun(r.db.QueryRowContext(ctx, `SELECT id,experiment_id,COALESCE(spec_id,''),COALESCE(assigned_server_id,''),run_status,remote_workdir,remote_job_id,started_at,ended_at,retry_count,exit_code,result_json,error_message,created_at,updated_at FROM experiment_runs WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ExperimentRunRepository) CountActiveByServerID(ctx context.Context, serverID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM experiment_runs WHERE assigned_server_id=$1 AND run_status IN ('queued','scheduled','running')`, serverID).Scan(&count)
	return count, err
}

func (r *ExperimentRunRepository) Create(ctx context.Context, item model.ExperimentRun) error {
	resultRaw, _ := json.Marshal(item.ResultJSON)
	_, err := r.db.ExecContext(ctx, `INSERT INTO experiment_runs (id,experiment_id,spec_id,assigned_server_id,run_status,remote_workdir,remote_job_id,started_at,ended_at,retry_count,exit_code,result_json,error_message,created_at,updated_at) VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		item.ID, item.ExperimentID, item.SpecID, item.AssignedServerID, item.RunStatus, item.RemoteWorkdir, item.RemoteJobID, item.StartedAt, item.EndedAt, item.RetryCount, item.ExitCode, resultRaw, item.ErrorMessage, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *ExperimentRunRepository) Update(ctx context.Context, item model.ExperimentRun) error {
	resultRaw, _ := json.Marshal(item.ResultJSON)
	_, err := r.db.ExecContext(ctx, `UPDATE experiment_runs SET spec_id=NULLIF($2,''),assigned_server_id=NULLIF($3,''),run_status=$4,remote_workdir=$5,remote_job_id=$6,started_at=$7,ended_at=$8,retry_count=$9,exit_code=$10,result_json=$11,error_message=$12,updated_at=$13 WHERE id=$1`,
		item.ID, item.SpecID, item.AssignedServerID, item.RunStatus, item.RemoteWorkdir, item.RemoteJobID, item.StartedAt, item.EndedAt, item.RetryCount, item.ExitCode, resultRaw, item.ErrorMessage, item.UpdatedAt,
	)
	return err
}

func scanExperimentRun(scanner researchAssetScanner) (model.ExperimentRun, error) {
	var item model.ExperimentRun
	var resultRaw []byte
	err := scanner.Scan(&item.ID, &item.ExperimentID, &item.SpecID, &item.AssignedServerID, &item.RunStatus, &item.RemoteWorkdir, &item.RemoteJobID, &item.StartedAt, &item.EndedAt, &item.RetryCount, &item.ExitCode, &resultRaw, &item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.ResultJSON = map[string]interface{}{}
	decodeJSON(resultRaw, &item.ResultJSON)
	return item, nil
}

type RunLogRepository struct {
	db *sql.DB
}

func NewRunLogRepository(db *sql.DB) *RunLogRepository {
	return &RunLogRepository{db: db}
}

func (r *RunLogRepository) ListByRunID(ctx context.Context, runID string) ([]model.RunLog, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,run_id,log_type,log_path,tail_text,created_at,updated_at FROM run_logs WHERE run_id=$1 ORDER BY id ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.RunLog, 0)
	for rows.Next() {
		var item model.RunLog
		if err = rows.Scan(&item.ID, &item.RunID, &item.LogType, &item.LogPath, &item.TailText, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *RunLogRepository) Add(ctx context.Context, item model.RunLog) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO run_logs (run_id,log_type,log_path,tail_text,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		item.RunID, item.LogType, item.LogPath, item.TailText, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

type SchedulerDecisionRepository struct {
	db *sql.DB
}

func NewSchedulerDecisionRepository(db *sql.DB) *SchedulerDecisionRepository {
	return &SchedulerDecisionRepository{db: db}
}

func (r *SchedulerDecisionRepository) ListByRunID(ctx context.Context, runID string) ([]model.SchedulerDecision, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,run_id,COALESCE(chosen_server_id,''),decision_json,created_at,updated_at FROM scheduler_decisions WHERE run_id=$1 ORDER BY created_at DESC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.SchedulerDecision, 0)
	for rows.Next() {
		item, scanErr := scanSchedulerDecision(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *SchedulerDecisionRepository) GetLatestByRunID(ctx context.Context, runID string) (*model.SchedulerDecision, error) {
	item, err := scanSchedulerDecision(r.db.QueryRowContext(ctx, `SELECT id,run_id,COALESCE(chosen_server_id,''),decision_json,created_at,updated_at FROM scheduler_decisions WHERE run_id=$1 ORDER BY created_at DESC LIMIT 1`, runID))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *SchedulerDecisionRepository) Create(ctx context.Context, item model.SchedulerDecision) error {
	decisionRaw, _ := json.Marshal(item.DecisionJSON)
	_, err := r.db.ExecContext(ctx, `INSERT INTO scheduler_decisions (id,run_id,chosen_server_id,decision_json,created_at,updated_at) VALUES ($1,$2,NULLIF($3,''),$4,$5,$6)`,
		item.ID, item.RunID, item.ChosenServerID, decisionRaw, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func scanSchedulerDecision(scanner researchAssetScanner) (model.SchedulerDecision, error) {
	var item model.SchedulerDecision
	var decisionRaw []byte
	err := scanner.Scan(&item.ID, &item.RunID, &item.ChosenServerID, &decisionRaw, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.DecisionJSON = map[string]interface{}{}
	decodeJSON(decisionRaw, &item.DecisionJSON)
	return item, nil
}
