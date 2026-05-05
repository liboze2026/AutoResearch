ALTER TABLE datasets ADD COLUMN IF NOT EXISTS server_id VARCHAR(64) REFERENCES servers(id);
ALTER TABLE datasets ADD COLUMN IF NOT EXISTS file_count BIGINT NOT NULL DEFAULT 0;
ALTER TABLE datasets ADD COLUMN IF NOT EXISTS directory_count BIGINT NOT NULL DEFAULT 0;
ALTER TABLE datasets ADD COLUMN IF NOT EXISTS total_size_bytes BIGINT NOT NULL DEFAULT 0;
ALTER TABLE datasets ADD COLUMN IF NOT EXISTS file_types JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE datasets ADD COLUMN IF NOT EXISTS detected_modality VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE datasets ADD COLUMN IF NOT EXISTS hierarchy_summary JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE datasets ADD COLUMN IF NOT EXISTS last_scan_status VARCHAR(32) NOT NULL DEFAULT 'none';
ALTER TABLE datasets ADD COLUMN IF NOT EXISTS last_scan_at TIMESTAMPTZ;
ALTER TABLE datasets ADD COLUMN IF NOT EXISTS last_modified_at TIMESTAMPTZ;
ALTER TABLE datasets ALTER COLUMN index_status SET DEFAULT 'none';

UPDATE datasets SET index_status='none' WHERE index_status IS NULL OR index_status='';

CREATE INDEX IF NOT EXISTS idx_datasets_server_id ON datasets(server_id);
CREATE INDEX IF NOT EXISTS idx_datasets_last_scan_at ON datasets(last_scan_at DESC);
CREATE INDEX IF NOT EXISTS idx_datasets_index_status ON datasets(index_status);

CREATE TABLE IF NOT EXISTS dataset_scan_records (
    id VARCHAR(64) PRIMARY KEY,
    dataset_id VARCHAR(64) NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    server_id VARCHAR(64) REFERENCES servers(id),
    runtime_mode VARCHAR(16) NOT NULL,
    scan_status VARCHAR(32) NOT NULL,
    validation_status VARCHAR(32) NOT NULL,
    root_path TEXT NOT NULL,
    file_count BIGINT NOT NULL DEFAULT 0,
    directory_count BIGINT NOT NULL DEFAULT 0,
    total_size_bytes BIGINT NOT NULL DEFAULT 0,
    file_types JSONB NOT NULL DEFAULT '{}'::jsonb,
    hierarchy_summary JSONB NOT NULL DEFAULT '[]'::jsonb,
    inferred_modality VARCHAR(32) NOT NULL DEFAULT '',
    recent_modified_at TIMESTAMPTZ,
    error_message TEXT NOT NULL DEFAULT '',
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    scanned_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_dataset_scan_records_dataset_id ON dataset_scan_records(dataset_id, scanned_at DESC);
CREATE INDEX IF NOT EXISTS idx_dataset_scan_records_server_id ON dataset_scan_records(server_id);

CREATE TABLE IF NOT EXISTS dataset_preview_items (
    id BIGSERIAL PRIMARY KEY,
    dataset_id VARCHAR(64) NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    scan_record_id VARCHAR(64) NOT NULL REFERENCES dataset_scan_records(id) ON DELETE CASCADE,
    item_name VARCHAR(255) NOT NULL,
    item_type VARCHAR(16) NOT NULL,
    item_category VARCHAR(32) NOT NULL DEFAULT '',
    relative_path TEXT NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    depth INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_dataset_preview_items_dataset_id ON dataset_preview_items(dataset_id, id ASC);
CREATE INDEX IF NOT EXISTS idx_dataset_preview_items_scan_record_id ON dataset_preview_items(scan_record_id, id ASC);

CREATE TABLE IF NOT EXISTS dataset_index_tasks (
    id VARCHAR(64) PRIMARY KEY,
    dataset_id VARCHAR(64) NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    server_id VARCHAR(64) REFERENCES servers(id),
    source_type VARCHAR(16) NOT NULL,
    executor_mode VARCHAR(16) NOT NULL,
    status VARCHAR(32) NOT NULL,
    remote_task_id VARCHAR(128) NOT NULL DEFAULT '',
    log_path TEXT NOT NULL DEFAULT '',
    status_path TEXT NOT NULL DEFAULT '',
    result_path TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    request_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    response_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_dataset_index_tasks_dataset_id ON dataset_index_tasks(dataset_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dataset_index_tasks_status ON dataset_index_tasks(status);

CREATE TABLE IF NOT EXISTS dataset_index_task_logs (
    id BIGSERIAL PRIMARY KEY,
    task_id VARCHAR(64) NOT NULL REFERENCES dataset_index_tasks(id) ON DELETE CASCADE,
    level VARCHAR(16) NOT NULL DEFAULT 'info',
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_dataset_index_task_logs_task_id ON dataset_index_task_logs(task_id, id ASC);
