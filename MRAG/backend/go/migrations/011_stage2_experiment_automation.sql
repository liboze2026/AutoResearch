CREATE TABLE IF NOT EXISTS experiments (
    id VARCHAR(64) PRIMARY KEY,
    idea_id VARCHAR(64) REFERENCES ideas(id) ON DELETE SET NULL,
    dataset_asset_id VARCHAR(64) NOT NULL REFERENCES dataset_assets(id) ON DELETE RESTRICT,
    baseline_id VARCHAR(64) REFERENCES baselines(id) ON DELETE SET NULL,
    title VARCHAR(256) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    priority INT NOT NULL DEFAULT 0,
    current_run_id VARCHAR(64),
    summary_md TEXT NOT NULL DEFAULT '',
    owner_note_md TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_experiments_dataset_asset_id ON experiments(dataset_asset_id);
CREATE INDEX IF NOT EXISTS idx_experiments_idea_id ON experiments(idea_id);
CREATE INDEX IF NOT EXISTS idx_experiments_baseline_id ON experiments(baseline_id);
CREATE INDEX IF NOT EXISTS idx_experiments_status ON experiments(status);
CREATE INDEX IF NOT EXISTS idx_experiments_priority ON experiments(priority DESC, updated_at DESC);

CREATE TABLE IF NOT EXISTS experiment_specs (
    id VARCHAR(64) PRIMARY KEY,
    experiment_id VARCHAR(64) NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    spec_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    template_type VARCHAR(64) NOT NULL DEFAULT '',
    generated_from JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (experiment_id, version)
);
CREATE INDEX IF NOT EXISTS idx_experiment_specs_experiment_id ON experiment_specs(experiment_id);
CREATE INDEX IF NOT EXISTS idx_experiment_specs_template_type ON experiment_specs(template_type);

CREATE TABLE IF NOT EXISTS experiment_runs (
    id VARCHAR(64) PRIMARY KEY,
    experiment_id VARCHAR(64) NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    spec_id VARCHAR(64) REFERENCES experiment_specs(id) ON DELETE SET NULL,
    assigned_server_id VARCHAR(64) REFERENCES servers(id) ON DELETE SET NULL,
    run_status VARCHAR(32) NOT NULL DEFAULT 'queued',
    remote_workdir TEXT NOT NULL DEFAULT '',
    remote_job_id TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    retry_count INT NOT NULL DEFAULT 0,
    exit_code INT,
    result_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_experiment_runs_experiment_id ON experiment_runs(experiment_id);
CREATE INDEX IF NOT EXISTS idx_experiment_runs_spec_id ON experiment_runs(spec_id);
CREATE INDEX IF NOT EXISTS idx_experiment_runs_assigned_server_id ON experiment_runs(assigned_server_id);
CREATE INDEX IF NOT EXISTS idx_experiment_runs_status ON experiment_runs(run_status);
CREATE INDEX IF NOT EXISTS idx_experiment_runs_created_at ON experiment_runs(created_at DESC);

CREATE TABLE IF NOT EXISTS run_logs (
    id BIGSERIAL PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES experiment_runs(id) ON DELETE CASCADE,
    log_type VARCHAR(32) NOT NULL DEFAULT 'event',
    log_path TEXT NOT NULL DEFAULT '',
    tail_text TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_run_logs_run_id ON run_logs(run_id);
CREATE INDEX IF NOT EXISTS idx_run_logs_log_type ON run_logs(log_type);

CREATE TABLE IF NOT EXISTS scheduler_decisions (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES experiment_runs(id) ON DELETE CASCADE,
    chosen_server_id VARCHAR(64) REFERENCES servers(id) ON DELETE SET NULL,
    decision_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_scheduler_decisions_run_id ON scheduler_decisions(run_id);
CREATE INDEX IF NOT EXISTS idx_scheduler_decisions_server_id ON scheduler_decisions(chosen_server_id);

CREATE TABLE IF NOT EXISTS server_heartbeats (
    id VARCHAR(64) PRIMARY KEY,
    server_id VARCHAR(64) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    heartbeat_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    detail_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_server_heartbeats_server_id ON server_heartbeats(server_id);
CREATE INDEX IF NOT EXISTS idx_server_heartbeats_status ON server_heartbeats(status);
CREATE INDEX IF NOT EXISTS idx_server_heartbeats_heartbeat_at ON server_heartbeats(heartbeat_at DESC);

CREATE TABLE IF NOT EXISTS gpu_resource_snapshots (
    id VARCHAR(64) PRIMARY KEY,
    server_id VARCHAR(64) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    captured_at TIMESTAMPTZ NOT NULL,
    gpu_index INT NOT NULL,
    name VARCHAR(256) NOT NULL DEFAULT '',
    total_mem_mb INT NOT NULL DEFAULT 0,
    free_mem_mb INT NOT NULL DEFAULT 0,
    utilization INT NOT NULL DEFAULT 0,
    process_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_gpu_resource_snapshots_server_id ON gpu_resource_snapshots(server_id);
CREATE INDEX IF NOT EXISTS idx_gpu_resource_snapshots_captured_at ON gpu_resource_snapshots(captured_at DESC);
CREATE INDEX IF NOT EXISTS idx_gpu_resource_snapshots_server_gpu ON gpu_resource_snapshots(server_id, gpu_index, captured_at DESC);

CREATE TABLE IF NOT EXISTS result_comparisons (
    id VARCHAR(64) PRIMARY KEY,
    experiment_id VARCHAR(64) NOT NULL REFERENCES experiments(id) ON DELETE CASCADE,
    run_id VARCHAR(64) NOT NULL REFERENCES experiment_runs(id) ON DELETE CASCADE,
    baseline_id VARCHAR(64) REFERENCES baselines(id) ON DELETE SET NULL,
    target_result_archive_id VARCHAR(64) REFERENCES result_archives(id) ON DELETE SET NULL,
    comparison_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    summary_md TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_result_comparisons_experiment_id ON result_comparisons(experiment_id);
CREATE INDEX IF NOT EXISTS idx_result_comparisons_run_id ON result_comparisons(run_id);
CREATE INDEX IF NOT EXISTS idx_result_comparisons_baseline_id ON result_comparisons(baseline_id);
CREATE INDEX IF NOT EXISTS idx_result_comparisons_target_archive_id ON result_comparisons(target_result_archive_id);
