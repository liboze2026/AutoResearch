CREATE TABLE IF NOT EXISTS papers (
    id VARCHAR(64) PRIMARY KEY,
    title TEXT NOT NULL,
    abstract TEXT NOT NULL DEFAULT '',
    authors TEXT NOT NULL DEFAULT '',
    venue TEXT NOT NULL DEFAULT '',
    year INT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'imported',
    source_type VARCHAR(32) NOT NULL DEFAULT 'manual',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_papers_status ON papers(status);
CREATE INDEX IF NOT EXISTS idx_papers_source_type ON papers(source_type);
CREATE INDEX IF NOT EXISTS idx_papers_year ON papers(year DESC);

CREATE TABLE IF NOT EXISTS paper_files (
    id VARCHAR(64) PRIMARY KEY,
    paper_id VARCHAR(64) NOT NULL REFERENCES papers(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    file_type VARCHAR(32) NOT NULL DEFAULT 'original',
    checksum VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (paper_id, file_path)
);
CREATE INDEX IF NOT EXISTS idx_paper_files_paper_id ON paper_files(paper_id);
CREATE INDEX IF NOT EXISTS idx_paper_files_file_type ON paper_files(file_type);

CREATE TABLE IF NOT EXISTS paper_insights (
    id VARCHAR(64) PRIMARY KEY,
    paper_id VARCHAR(64) NOT NULL REFERENCES papers(id) ON DELETE CASCADE,
    summary_md TEXT NOT NULL DEFAULT '',
    contributions_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    methods_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    limitations_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_paper_insights_paper_id ON paper_insights(paper_id);

CREATE TABLE IF NOT EXISTS ideas (
    id VARCHAR(64) PRIMARY KEY,
    title TEXT NOT NULL,
    description_md TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    weight INT NOT NULL DEFAULT 0,
    source_type VARCHAR(32) NOT NULL DEFAULT 'manual',
    priority INT NOT NULL DEFAULT 0,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ideas_status ON ideas(status);
CREATE INDEX IF NOT EXISTS idx_ideas_source_type ON ideas(source_type);
CREATE INDEX IF NOT EXISTS idx_ideas_priority ON ideas(priority DESC, weight DESC);

CREATE TABLE IF NOT EXISTS idea_sources (
    id BIGSERIAL PRIMARY KEY,
    idea_id VARCHAR(64) NOT NULL REFERENCES ideas(id) ON DELETE CASCADE,
    paper_id VARCHAR(64) REFERENCES papers(id) ON DELETE SET NULL,
    paper_insight_id VARCHAR(64) REFERENCES paper_insights(id) ON DELETE SET NULL,
    source_note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_idea_sources_idea_id ON idea_sources(idea_id);
CREATE INDEX IF NOT EXISTS idx_idea_sources_paper_id ON idea_sources(paper_id);
CREATE INDEX IF NOT EXISTS idx_idea_sources_paper_insight_id ON idea_sources(paper_insight_id);

CREATE TABLE IF NOT EXISTS dataset_assets (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description_md TEXT NOT NULL DEFAULT '',
    task_type VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    source_type VARCHAR(32) NOT NULL DEFAULT 'mrag_dataset',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_dataset_assets_status ON dataset_assets(status);
CREATE INDEX IF NOT EXISTS idx_dataset_assets_source_type ON dataset_assets(source_type);

CREATE TABLE IF NOT EXISTS dataset_asset_sources (
    id BIGSERIAL PRIMARY KEY,
    dataset_asset_id VARCHAR(64) NOT NULL REFERENCES dataset_assets(id) ON DELETE CASCADE,
    existing_dataset_ref VARCHAR(64) NOT NULL REFERENCES datasets(id) ON DELETE RESTRICT,
    source_kind VARCHAR(32) NOT NULL DEFAULT 'mrag_dataset',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (dataset_asset_id, existing_dataset_ref, source_kind),
    UNIQUE (existing_dataset_ref)
);
CREATE INDEX IF NOT EXISTS idx_dataset_asset_sources_asset_id ON dataset_asset_sources(dataset_asset_id);
CREATE INDEX IF NOT EXISTS idx_dataset_asset_sources_dataset_ref ON dataset_asset_sources(existing_dataset_ref);

CREATE TABLE IF NOT EXISTS baselines (
    id VARCHAR(64) PRIMARY KEY,
    dataset_asset_id VARCHAR(64) NOT NULL REFERENCES dataset_assets(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    metric_schema_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    note_md TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (dataset_asset_id, name)
);
CREATE INDEX IF NOT EXISTS idx_baselines_dataset_asset_id ON baselines(dataset_asset_id);

CREATE TABLE IF NOT EXISTS result_archives (
    id VARCHAR(64) PRIMARY KEY,
    title VARCHAR(256) NOT NULL,
    dataset_asset_id VARCHAR(64) NOT NULL REFERENCES dataset_assets(id) ON DELETE RESTRICT,
    baseline_id VARCHAR(64) REFERENCES baselines(id) ON DELETE SET NULL,
    idea_id VARCHAR(64) REFERENCES ideas(id) ON DELETE SET NULL,
    server_id VARCHAR(64) REFERENCES servers(id) ON DELETE SET NULL,
    summary_md TEXT NOT NULL DEFAULT '',
    metric_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'archived',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_result_archives_dataset_asset_id ON result_archives(dataset_asset_id);
CREATE INDEX IF NOT EXISTS idx_result_archives_baseline_id ON result_archives(baseline_id);
CREATE INDEX IF NOT EXISTS idx_result_archives_idea_id ON result_archives(idea_id);
CREATE INDEX IF NOT EXISTS idx_result_archives_server_id ON result_archives(server_id);
CREATE INDEX IF NOT EXISTS idx_result_archives_status ON result_archives(status);

CREATE TABLE IF NOT EXISTS archive_files (
    id BIGSERIAL PRIMARY KEY,
    archive_id VARCHAR(64) NOT NULL REFERENCES result_archives(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    file_kind VARCHAR(32) NOT NULL DEFAULT 'attachment',
    checksum VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (archive_id, file_path)
);
CREATE INDEX IF NOT EXISTS idx_archive_files_archive_id ON archive_files(archive_id);
CREATE INDEX IF NOT EXISTS idx_archive_files_file_kind ON archive_files(file_kind);
