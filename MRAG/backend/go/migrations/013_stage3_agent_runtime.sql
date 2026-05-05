CREATE TABLE IF NOT EXISTS agent_jobs (
    id VARCHAR(64) PRIMARY KEY,
    agent_type VARCHAR(64) NOT NULL,
    execution_mode VARCHAR(32) NOT NULL,
    model_provider VARCHAR(64) NOT NULL DEFAULT '',
    model_name VARCHAR(128) NOT NULL DEFAULT '',
    prompt_version VARCHAR(64) NOT NULL DEFAULT '',
    input_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    output_schema_ref VARCHAR(256) NOT NULL DEFAULT '',
    skill_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    tool_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    memory_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    workspace_dir TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'registered',
    normalized_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    artifact_manifest JSONB NOT NULL DEFAULT '[]'::jsonb,
    repair_actions JSONB NOT NULL DEFAULT '[]'::jsonb,
    tool_usages JSONB NOT NULL DEFAULT '[]'::jsonb,
    warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_agent_jobs_status ON agent_jobs(status);
CREATE INDEX IF NOT EXISTS idx_agent_jobs_agent_type ON agent_jobs(agent_type);
CREATE INDEX IF NOT EXISTS idx_agent_jobs_execution_mode ON agent_jobs(execution_mode);

CREATE TABLE IF NOT EXISTS agent_artifacts (
    id VARCHAR(64) PRIMARY KEY,
    job_id VARCHAR(64) NOT NULL REFERENCES agent_jobs(id) ON DELETE CASCADE,
    artifact_type VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    file_path TEXT NOT NULL,
    checksum VARCHAR(128) NOT NULL DEFAULT '',
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_agent_artifacts_job_id ON agent_artifacts(job_id);
CREATE INDEX IF NOT EXISTS idx_agent_artifacts_type ON agent_artifacts(artifact_type);

CREATE TABLE IF NOT EXISTS agent_job_triggers (
    id VARCHAR(64) PRIMARY KEY,
    job_id VARCHAR(64) NOT NULL REFERENCES agent_jobs(id) ON DELETE CASCADE,
    trigger_type VARCHAR(32) NOT NULL DEFAULT 'manual',
    status VARCHAR(32) NOT NULL DEFAULT 'requested',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_agent_job_triggers_job_id ON agent_job_triggers(job_id);
CREATE INDEX IF NOT EXISTS idx_agent_job_triggers_status ON agent_job_triggers(status);
