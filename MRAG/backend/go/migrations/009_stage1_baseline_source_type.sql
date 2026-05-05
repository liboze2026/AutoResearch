ALTER TABLE baselines ADD COLUMN IF NOT EXISTS source_type VARCHAR(32) NOT NULL DEFAULT 'manual';
CREATE INDEX IF NOT EXISTS idx_baselines_source_type ON baselines(source_type);