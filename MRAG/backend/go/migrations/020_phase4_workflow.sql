CREATE TABLE IF NOT EXISTS phase4_workflows (
    id TEXT PRIMARY KEY,
    dataset_profile_id TEXT NOT NULL REFERENCES phase4_dataset_profiles(id) ON DELETE CASCADE,
    reader_context_id TEXT REFERENCES phase4_reader_contexts(id) ON DELETE SET NULL,
    selected_idea_id TEXT REFERENCES phase4_ideas(id) ON DELETE SET NULL,
    current_run_manifest_id TEXT REFERENCES phase4_run_manifests(id) ON DELETE SET NULL,
    latest_report_id TEXT REFERENCES phase4_structured_reports(id) ON DELETE SET NULL,
    latest_reader_job_id VARCHAR(64) REFERENCES agent_jobs(id) ON DELETE SET NULL,
    latest_idea_job_id VARCHAR(64) REFERENCES agent_jobs(id) ON DELETE SET NULL,
    latest_coding_job_id VARCHAR(64) REFERENCES agent_jobs(id) ON DELETE SET NULL,
    latest_writer_job_id VARCHAR(64) REFERENCES agent_jobs(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'running_reader',
    next_action TEXT NOT NULL DEFAULT 'none',
    last_error TEXT NOT NULL DEFAULT '',
    manual_inputs_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_phase4_workflows_dataset_profile_id ON phase4_workflows(dataset_profile_id);
CREATE INDEX IF NOT EXISTS idx_phase4_workflows_status ON phase4_workflows(status);
CREATE INDEX IF NOT EXISTS idx_phase4_workflows_updated_at ON phase4_workflows(updated_at DESC);

CREATE TABLE IF NOT EXISTS phase4_workflow_actions (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES phase4_workflows(id) ON DELETE CASCADE,
    stage TEXT NOT NULL DEFAULT 'workflow',
    action_type TEXT NOT NULL DEFAULT '',
    actor_type TEXT NOT NULL DEFAULT 'system',
    status TEXT NOT NULL DEFAULT 'started',
    job_id VARCHAR(64) REFERENCES agent_jobs(id) ON DELETE SET NULL,
    run_manifest_id TEXT REFERENCES phase4_run_manifests(id) ON DELETE SET NULL,
    report_id TEXT REFERENCES phase4_structured_reports(id) ON DELETE SET NULL,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_phase4_workflow_actions_workflow_id ON phase4_workflow_actions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_phase4_workflow_actions_stage ON phase4_workflow_actions(stage);
CREATE INDEX IF NOT EXISTS idx_phase4_workflow_actions_status ON phase4_workflow_actions(status);
CREATE INDEX IF NOT EXISTS idx_phase4_workflow_actions_created_at ON phase4_workflow_actions(created_at DESC);
