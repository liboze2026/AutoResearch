ALTER TABLE agent_jobs
    ADD COLUMN IF NOT EXISTS trigger_event_id VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS dedup_key VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_retries INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS concurrency_limit INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_agent_jobs_trigger_event_id ON agent_jobs(trigger_event_id);
CREATE INDEX IF NOT EXISTS idx_agent_jobs_dedup_key ON agent_jobs(dedup_key);

CREATE TABLE IF NOT EXISTS agent_events (
    id VARCHAR(64) PRIMARY KEY,
    event_type VARCHAR(64) NOT NULL,
    source_ref TEXT NOT NULL DEFAULT '',
    input_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'processing',
    triggered_job_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_agent_events_event_type ON agent_events(event_type);
CREATE INDEX IF NOT EXISTS idx_agent_events_status ON agent_events(status);

CREATE TABLE IF NOT EXISTS agent_subscriptions (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    agent_type VARCHAR(64) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    execution_mode VARCHAR(32) NOT NULL DEFAULT 'mock',
    model_provider VARCHAR(64) NOT NULL DEFAULT '',
    model_name VARCHAR(128) NOT NULL DEFAULT '',
    prompt_version VARCHAR(64) NOT NULL DEFAULT '',
    output_schema_ref VARCHAR(256) NOT NULL DEFAULT '',
    skill_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    tool_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    memory_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    trigger_rule JSONB NOT NULL DEFAULT '{}'::jsonb,
    max_retries INTEGER NOT NULL DEFAULT 0,
    concurrency_limit INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_agent_subscriptions_event_type ON agent_subscriptions(event_type);
CREATE INDEX IF NOT EXISTS idx_agent_subscriptions_agent_type ON agent_subscriptions(agent_type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_subscriptions_unique_name ON agent_subscriptions(name);
