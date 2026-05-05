CREATE TABLE IF NOT EXISTS datasets (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    source_type VARCHAR(32) NOT NULL,
    modality VARCHAR(32) NOT NULL,
    version VARCHAR(32) NOT NULL,
    size_text VARCHAR(32) NOT NULL,
    sample_count BIGINT NOT NULL DEFAULT 0,
    description TEXT NOT NULL,
    path TEXT NOT NULL,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    index_status VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_datasets_source_type ON datasets(source_type);
CREATE INDEX IF NOT EXISTS idx_datasets_modality ON datasets(modality);
CREATE INDEX IF NOT EXISTS idx_datasets_name ON datasets(name);

CREATE TABLE IF NOT EXISTS dataset_index_records (
    id VARCHAR(64) PRIMARY KEY,
    dataset_id VARCHAR(64) NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    version VARCHAR(32) NOT NULL,
    index_status VARCHAR(32) NOT NULL,
    index_path TEXT NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_dataset_index_records_dataset_id ON dataset_index_records(dataset_id);

CREATE TABLE IF NOT EXISTS servers (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    host VARCHAR(128) NOT NULL,
    ssh_port INT NOT NULL DEFAULT 22,
    username VARCHAR(64) NOT NULL,
    auth_type VARCHAR(16) NOT NULL DEFAULT 'password',
    password_cipher TEXT NOT NULL DEFAULT '',
    private_key_cipher TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'offline',
    gpu_info VARCHAR(256) NOT NULL DEFAULT '',
    remote_root TEXT NOT NULL,
    task_workdir TEXT NOT NULL,
    last_heartbeat TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_servers_status ON servers(status);

-- Legacy phase-0/phase-1 experiment tables were removed in
-- 003_remove_experiments_and_extend_servers.sql and replaced by the
-- stage-2 experiment schema in later migrations.
--
-- The current migration runner replays every SQL file on startup instead of
-- tracking applied versions. Keeping the old experiments(dataset_id) DDL here
-- causes repeat startups to fail once the stage-2 experiments table exists
-- with dataset_asset_id instead of dataset_id.
--
-- Intentionally no-op the legacy experiment DDL in 001_init.sql so fresh
-- installs still converge through later migrations and repeated startups stay
-- idempotent.

CREATE TABLE IF NOT EXISTS system_settings (
    id SMALLINT PRIMARY KEY,
    data_root_local TEXT NOT NULL,
    data_root_remote TEXT NOT NULL,
    experiment_output_root TEXT NOT NULL,
    default_retriever VARCHAR(128) NOT NULL,
    default_reranker VARCHAR(128) NOT NULL,
    default_generator VARCHAR(128) NOT NULL,
    default_evaluator VARCHAR(128) NOT NULL,
    storage_policy VARCHAR(32) NOT NULL,
    ui_theme VARCHAR(16) NOT NULL,
    table_density VARCHAR(16) NOT NULL,
    extra JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO system_settings (
    id, data_root_local, data_root_remote, experiment_output_root,
    default_retriever, default_reranker, default_generator, default_evaluator,
    storage_policy, ui_theme, table_density
) VALUES (
    1, 'D:/datasets', '/srv/mrag/datasets', 'D:/mrag/outputs',
    'bge-m3', 'bge-reranker-large', 'qwen2.5-7b-instruct', 'ragas-v1',
    'hybrid', 'light', 'comfortable'
) ON CONFLICT (id) DO NOTHING;
