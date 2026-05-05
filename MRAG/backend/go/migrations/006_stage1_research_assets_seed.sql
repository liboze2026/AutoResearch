INSERT INTO datasets (
    id, name, source_type, modality, version, size_text, sample_count, description, path, tags, index_status,
    file_count, directory_count, total_size_bytes, file_types, detected_modality, hierarchy_summary, last_scan_status, last_scan_at, last_modified_at, updated_at
) VALUES (
    'ds_stage1_demo_nlp',
    'Stage1 Demo NLP Dataset',
    'local',
    'text',
    'v1',
    '12 MB',
    1200,
    'Demo dataset seeded for stage 1 research asset development.',
    'D:/1/MRAG/workspace/datasets/demo_nlp',
    '["stage1","demo","nlp"]'::jsonb,
    'none',
    1200,
    24,
    12582912,
    '{"text": 1200}'::jsonb,
    'text',
    '[]'::jsonb,
    'seeded',
    now(),
    now(),
    now()
) ON CONFLICT (id) DO NOTHING;

INSERT INTO papers (id, title, abstract, authors, venue, year, status, source_type)
VALUES (
    'paper_stage1_demo_001',
    'Stage1 Demo Paper on Research Asset Management',
    'A seeded demo paper used to exercise stage 1 research asset flows in MRAG.',
    'OpenAI Demo Team',
    'MRAG Workshop',
    2026,
    'parsed',
    'seed'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO paper_files (id, paper_id, file_path, file_type, checksum)
VALUES (
    'pfile_stage1_demo_001',
    'paper_stage1_demo_001',
    'workspace/papers/incoming/paper_stage1_demo_001.md',
    'original',
    'seed-paper-file-001'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO paper_insights (id, paper_id, summary_md, contributions_json, methods_json, limitations_json)
VALUES (
    'pinsight_stage1_demo_001',
    'paper_stage1_demo_001',
    'This seeded paper argues for research objects as first-class assets in MRAG.',
    '["Defines research assets as manageable system objects","Separates archival results from execution workflows"]'::jsonb,
    '["Incremental system integration","Schema-first modeling"]'::jsonb,
    '["No automatic execution in stage 1"]'::jsonb
) ON CONFLICT (id) DO NOTHING;

INSERT INTO ideas (id, title, description_md, status, weight, source_type, priority, confidence)
VALUES (
    'idea_stage1_demo_001',
    'Asset-first experiment review workflow',
    'Use archived results, baselines, and paper insights to compare future work without enabling automatic execution.',
    'candidate',
    10,
    'seed',
    5,
    0.82
) ON CONFLICT (id) DO NOTHING;

INSERT INTO idea_sources (idea_id, paper_id, paper_insight_id, source_note)
VALUES (
    'idea_stage1_demo_001',
    'paper_stage1_demo_001',
    'pinsight_stage1_demo_001',
    'Seeded from the demo paper insight to illustrate traceable idea origins.'
) ON CONFLICT DO NOTHING;

INSERT INTO dataset_assets (id, name, description_md, task_type, status, source_type)
VALUES (
    'dasset_stage1_demo_001',
    'Demo NLP Dataset Asset',
    'Research asset wrapper for the seeded MRAG dataset.',
    'text-classification',
    'active',
    'mrag_dataset'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO dataset_asset_sources (dataset_asset_id, existing_dataset_ref, source_kind)
VALUES (
    'dasset_stage1_demo_001',
    'ds_stage1_demo_nlp',
    'mrag_dataset'
) ON CONFLICT DO NOTHING;

INSERT INTO baselines (id, dataset_asset_id, name, metric_schema_json, result_json, note_md)
VALUES (
    'baseline_stage1_demo_001',
    'dasset_stage1_demo_001',
    'demo-bert-tiny',
    '{"primary_metric":"macro_f1","tracked_metrics":["macro_f1","accuracy"]}'::jsonb,
    '{"macro_f1":0.71,"accuracy":0.76}'::jsonb,
    'Seeded baseline for local stage 1 demos.'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO result_archives (id, title, dataset_asset_id, baseline_id, idea_id, summary_md, metric_json, status)
VALUES (
    'rarch_stage1_demo_001',
    'Demo archived comparison result',
    'dasset_stage1_demo_001',
    'baseline_stage1_demo_001',
    'idea_stage1_demo_001',
    'This archived result records a seeded comparison summary for stage 1 page and API validation.',
    '{"macro_f1":0.74,"accuracy":0.79}'::jsonb,
    'archived'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO archive_files (archive_id, file_path, file_kind, checksum)
VALUES (
    'rarch_stage1_demo_001',
    'workspace/results/rarch_stage1_demo_001/summary.md',
    'summary',
    'seed-archive-file-001'
) ON CONFLICT DO NOTHING;
