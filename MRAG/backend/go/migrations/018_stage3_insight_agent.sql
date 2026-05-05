ALTER TABLE paper_insights ADD COLUMN IF NOT EXISTS novelty_points_json JSONB NOT NULL DEFAULT '[]'::jsonb;
