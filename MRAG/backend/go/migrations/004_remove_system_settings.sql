ALTER TABLE IF EXISTS system_settings
    DROP COLUMN IF EXISTS data_root_local,
    DROP COLUMN IF EXISTS data_root_remote,
    DROP COLUMN IF EXISTS experiment_output_root,
    DROP COLUMN IF EXISTS default_retriever,
    DROP COLUMN IF EXISTS default_reranker,
    DROP COLUMN IF EXISTS default_generator,
    DROP COLUMN IF EXISTS default_evaluator,
    DROP COLUMN IF EXISTS storage_policy,
    DROP COLUMN IF EXISTS ui_theme,
    DROP COLUMN IF EXISTS table_density;

DROP TABLE IF EXISTS system_settings;