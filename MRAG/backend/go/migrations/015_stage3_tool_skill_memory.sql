CREATE TABLE IF NOT EXISTS tool_registry (
    tool_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    owner_agent_type VARCHAR(64) NOT NULL,
    path TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    usage_md TEXT NOT NULL DEFAULT '',
    input_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    output_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    test_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    version VARCHAR(64) NOT NULL DEFAULT 'v1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_tool_registry_owner_agent_type ON tool_registry(owner_agent_type);
CREATE INDEX IF NOT EXISTS idx_tool_registry_test_status ON tool_registry(test_status);

CREATE TABLE IF NOT EXISTS skill_registry (
    skill_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    skill_dir TEXT NOT NULL,
    entrypoint TEXT NOT NULL,
    dependencies JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_skill_registry_name ON skill_registry(name);

CREATE TABLE IF NOT EXISTS agent_memory (
    id VARCHAR(64) PRIMARY KEY,
    agent_type VARCHAR(64) NOT NULL,
    memory_key VARCHAR(128) NOT NULL,
    content_md TEXT NOT NULL DEFAULT '',
    source_ref TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (agent_type, memory_key)
);
CREATE INDEX IF NOT EXISTS idx_agent_memory_agent_type ON agent_memory(agent_type);
