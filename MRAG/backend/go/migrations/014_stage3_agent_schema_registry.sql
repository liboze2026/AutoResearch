ALTER TABLE agent_jobs
    ADD COLUMN IF NOT EXISTS validation_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS repair_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS validation_errors JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_agent_jobs_validation_status ON agent_jobs(validation_status);
CREATE INDEX IF NOT EXISTS idx_agent_jobs_repair_status ON agent_jobs(repair_status);

CREATE TABLE IF NOT EXISTS agent_schemas (
    id VARCHAR(64) PRIMARY KEY,
    schema_name VARCHAR(128) NOT NULL,
    version VARCHAR(64) NOT NULL,
    agent_type VARCHAR(64) NOT NULL,
    schema_ref VARCHAR(256) NOT NULL,
    json_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    python_schema_ref TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_schemas_agent_type ON agent_schemas(agent_type);
CREATE INDEX IF NOT EXISTS idx_agent_schemas_schema_name ON agent_schemas(schema_name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_schemas_unique_ref ON agent_schemas(schema_ref, version, agent_type);
