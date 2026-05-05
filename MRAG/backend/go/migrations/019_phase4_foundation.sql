CREATE TABLE IF NOT EXISTS phase4_dataset_profiles (
    id TEXT PRIMARY KEY,
    dataset_name TEXT NOT NULL,
    task_type TEXT NOT NULL,
    modality_composition_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    splits_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    label_schema_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    file_structure_snapshot_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    sample_statistics_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    official_metric TEXT NOT NULL DEFAULT '',
    official_baseline TEXT NOT NULL DEFAULT '',
    license TEXT NOT NULL DEFAULT '',
    citation TEXT NOT NULL DEFAULT '',
    known_difficulties_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    user_notes TEXT NOT NULL DEFAULT '',
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    source_mode TEXT NOT NULL DEFAULT 'registered_path',
    server_id TEXT REFERENCES servers(id) ON DELETE SET NULL,
    server_path TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_phase4_dataset_profiles_task_type ON phase4_dataset_profiles(task_type);
CREATE INDEX IF NOT EXISTS idx_phase4_dataset_profiles_status ON phase4_dataset_profiles(status);

CREATE TABLE IF NOT EXISTS phase4_reader_sources (
    id TEXT PRIMARY KEY,
    dataset_profile_id TEXT REFERENCES phase4_dataset_profiles(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    authors_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    venue TEXT NOT NULL DEFAULT '',
    publication_year INTEGER NOT NULL DEFAULT 0,
    source_type TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    open_access_url TEXT NOT NULL DEFAULT '',
    quality_tier TEXT NOT NULL DEFAULT '',
    ranking_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    quality_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    relevance_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    citation_count INTEGER NOT NULL DEFAULT 0,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_phase4_reader_sources_dataset_profile_id ON phase4_reader_sources(dataset_profile_id);

CREATE TABLE IF NOT EXISTS phase4_reader_contexts (
    id TEXT PRIMARY KEY,
    dataset_profile_id TEXT REFERENCES phase4_dataset_profiles(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    task_definition TEXT NOT NULL DEFAULT '',
    related_work_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    retrieval_focus_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    ranking_notes TEXT NOT NULL DEFAULT '',
    source_ids_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    structured_context_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_phase4_reader_contexts_dataset_profile_id ON phase4_reader_contexts(dataset_profile_id);
CREATE INDEX IF NOT EXISTS idx_phase4_reader_contexts_status ON phase4_reader_contexts(status);

CREATE TABLE IF NOT EXISTS phase4_ideas (
    id TEXT PRIMARY KEY,
    dataset_profile_id TEXT REFERENCES phase4_dataset_profiles(id) ON DELETE SET NULL,
    reader_context_id TEXT REFERENCES phase4_reader_contexts(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    problem_definition TEXT NOT NULL DEFAULT '',
    core_method TEXT NOT NULL DEFAULT '',
    differentiators TEXT NOT NULL DEFAULT '',
    data_processing_needs_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    model_changes_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    training_plan TEXT NOT NULL DEFAULT '',
    evaluation_metrics_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    risk_points_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    expected_gains_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    score_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    score_summary_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'draft',
    source_type TEXT NOT NULL DEFAULT 'manual',
    revision_of_id TEXT REFERENCES phase4_ideas(id) ON DELETE SET NULL,
    lineage_root_id TEXT REFERENCES phase4_ideas(id) ON DELETE SET NULL,
    failure_feedback_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_failure_run_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_phase4_ideas_dataset_profile_id ON phase4_ideas(dataset_profile_id);
CREATE INDEX IF NOT EXISTS idx_phase4_ideas_reader_context_id ON phase4_ideas(reader_context_id);
CREATE INDEX IF NOT EXISTS idx_phase4_ideas_status ON phase4_ideas(status);
CREATE INDEX IF NOT EXISTS idx_phase4_ideas_revision_of_id ON phase4_ideas(revision_of_id);

CREATE TABLE IF NOT EXISTS phase4_run_manifests (
    id TEXT PRIMARY KEY,
    dataset_profile_id TEXT NOT NULL REFERENCES phase4_dataset_profiles(id) ON DELETE CASCADE,
    idea_id TEXT NOT NULL REFERENCES phase4_ideas(id) ON DELETE CASCADE,
    reader_context_id TEXT REFERENCES phase4_reader_contexts(id) ON DELETE SET NULL,
    code_snapshot_id TEXT NOT NULL DEFAULT '',
    runner_mode TEXT NOT NULL DEFAULT '',
    server_id TEXT REFERENCES servers(id) ON DELETE SET NULL,
    gpu TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retry_count INTEGER NOT NULL DEFAULT 3,
    artifact_paths_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    logs_path TEXT NOT NULL DEFAULT '',
    metrics_path TEXT NOT NULL DEFAULT '',
    failure_feedback_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_phase4_run_manifests_dataset_profile_id ON phase4_run_manifests(dataset_profile_id);
CREATE INDEX IF NOT EXISTS idx_phase4_run_manifests_idea_id ON phase4_run_manifests(idea_id);
CREATE INDEX IF NOT EXISTS idx_phase4_run_manifests_status ON phase4_run_manifests(status);

CREATE TABLE IF NOT EXISTS phase4_structured_reports (
    id TEXT PRIMARY KEY,
    run_manifest_id TEXT NOT NULL REFERENCES phase4_run_manifests(id) ON DELETE CASCADE,
    dataset_profile_id TEXT REFERENCES phase4_dataset_profiles(id) ON DELETE SET NULL,
    idea_id TEXT REFERENCES phase4_ideas(id) ON DELETE SET NULL,
    reader_context_id TEXT REFERENCES phase4_reader_contexts(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    machine_readable_report_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    human_readable_report_md TEXT NOT NULL DEFAULT '',
    citation_refs_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    reference_source_ids_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    status TEXT NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_phase4_structured_reports_run_manifest_id ON phase4_structured_reports(run_manifest_id);
CREATE INDEX IF NOT EXISTS idx_phase4_structured_reports_status ON phase4_structured_reports(status);
