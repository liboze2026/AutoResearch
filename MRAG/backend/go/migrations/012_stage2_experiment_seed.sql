INSERT INTO experiments (
    id, idea_id, dataset_asset_id, baseline_id, title, status, priority, summary_md, owner_note_md
)
SELECT
    'exp_stage2_demo_001',
    'idea_stage1_demo_001',
    'dasset_stage1_demo_001',
    'baseline_stage1_demo_001',
    'Stage2 Demo Experiment',
    'spec_ready',
    5,
    'Seeded stage2 experiment linked to the stage1 demo idea, dataset asset, and baseline.',
    'Seeded for stage2 schema and repository development.'
WHERE EXISTS (SELECT 1 FROM dataset_assets WHERE id = 'dasset_stage1_demo_001')
  AND NOT EXISTS (SELECT 1 FROM experiments WHERE id = 'exp_stage2_demo_001');

INSERT INTO experiment_specs (
    id, experiment_id, spec_json, template_type, generated_from, version
)
SELECT
    'espec_stage2_demo_001',
    'exp_stage2_demo_001',
    '{
      "ideaId": "idea_stage1_demo_001",
      "datasetAssetId": "dasset_stage1_demo_001",
      "baselineId": "baseline_stage1_demo_001",
      "objective": "Validate stage2 experiment lifecycle wiring against existing stage1 assets.",
      "resourceRequirements": {
        "gpuCount": 1,
        "minFreeMemMb": 8192
      },
      "comparisonTargets": {
        "baselineId": "baseline_stage1_demo_001",
        "historicalResultArchiveId": "rarch_stage1_demo_001"
      }
    }'::jsonb,
    'demo-train-template',
    '{
      "ideaId": "idea_stage1_demo_001",
      "datasetAssetId": "dasset_stage1_demo_001",
      "baselineId": "baseline_stage1_demo_001",
      "source": "stage2_seed"
    }'::jsonb,
    1
WHERE EXISTS (SELECT 1 FROM experiments WHERE id = 'exp_stage2_demo_001')
  AND NOT EXISTS (SELECT 1 FROM experiment_specs WHERE id = 'espec_stage2_demo_001');

INSERT INTO experiment_runs (
    id, experiment_id, spec_id, assigned_server_id, run_status, remote_workdir, remote_job_id, started_at, ended_at, retry_count, exit_code, result_json, error_message
)
SELECT
    'erun_stage2_demo_001',
    'exp_stage2_demo_001',
    'espec_stage2_demo_001',
    (SELECT id FROM servers ORDER BY created_at ASC LIMIT 1),
    'succeeded',
    '/home/bzli/lbz/experiments/exp_stage2_demo_001/run_001',
    'demo-job-001',
    now() - interval '15 minutes',
    now() - interval '5 minutes',
    0,
    0,
    '{
      "primaryMetric": "macro_f1",
      "macro_f1": 0.78,
      "accuracy": 0.81,
      "status": "succeeded"
    }'::jsonb,
    ''
WHERE EXISTS (SELECT 1 FROM experiments WHERE id = 'exp_stage2_demo_001')
  AND NOT EXISTS (SELECT 1 FROM experiment_runs WHERE id = 'erun_stage2_demo_001');

UPDATE experiments
SET current_run_id = 'erun_stage2_demo_001',
    updated_at = now()
WHERE id = 'exp_stage2_demo_001'
  AND EXISTS (SELECT 1 FROM experiment_runs WHERE id = 'erun_stage2_demo_001');

INSERT INTO run_logs (run_id, log_type, log_path, tail_text)
SELECT
    'erun_stage2_demo_001',
    'event',
    'workspace/experiments/exp_stage2_demo_001/runs/erun_stage2_demo_001/logs/events.log',
    'experiment queued -> scheduled -> running -> succeeded'
WHERE EXISTS (SELECT 1 FROM experiment_runs WHERE id = 'erun_stage2_demo_001')
  AND NOT EXISTS (
      SELECT 1 FROM run_logs
      WHERE run_id = 'erun_stage2_demo_001'
        AND log_type = 'event'
        AND log_path = 'workspace/experiments/exp_stage2_demo_001/runs/erun_stage2_demo_001/logs/events.log'
  );

INSERT INTO run_logs (run_id, log_type, log_path, tail_text)
SELECT
    'erun_stage2_demo_001',
    'stdout',
    'workspace/experiments/exp_stage2_demo_001/runs/erun_stage2_demo_001/logs/stdout.log',
    'epoch=3 macro_f1=0.78 accuracy=0.81'
WHERE EXISTS (SELECT 1 FROM experiment_runs WHERE id = 'erun_stage2_demo_001')
  AND NOT EXISTS (
      SELECT 1 FROM run_logs
      WHERE run_id = 'erun_stage2_demo_001'
        AND log_type = 'stdout'
        AND log_path = 'workspace/experiments/exp_stage2_demo_001/runs/erun_stage2_demo_001/logs/stdout.log'
  );

INSERT INTO scheduler_decisions (
    id, run_id, chosen_server_id, decision_json
)
SELECT
    'sdec_stage2_demo_001',
    'erun_stage2_demo_001',
    (SELECT assigned_server_id FROM experiment_runs WHERE id = 'erun_stage2_demo_001'),
    jsonb_build_object(
        'strategy', 'seeded-demo',
        'reason', 'selected first available server for seeded stage2 development record',
        'requiredResource', jsonb_build_object('gpuCount', 1, 'minFreeMemMb', 8192)
    )
WHERE EXISTS (SELECT 1 FROM experiment_runs WHERE id = 'erun_stage2_demo_001')
  AND NOT EXISTS (SELECT 1 FROM scheduler_decisions WHERE id = 'sdec_stage2_demo_001');

INSERT INTO server_heartbeats (
    id, server_id, heartbeat_at, status, detail_json
)
SELECT
    'shb_stage2_demo_001',
    s.id,
    now() - interval '2 minutes',
    'online',
    jsonb_build_object('source', 'stage2_seed', 'message', 'seeded heartbeat for stage2 development')
FROM servers s
WHERE NOT EXISTS (SELECT 1 FROM server_heartbeats WHERE id = 'shb_stage2_demo_001')
ORDER BY s.created_at ASC
LIMIT 1;

INSERT INTO gpu_resource_snapshots (
    id, server_id, captured_at, gpu_index, name, total_mem_mb, free_mem_mb, utilization, process_json
)
SELECT
    'gsnap_stage2_demo_001',
    s.id,
    now() - interval '2 minutes',
    0,
    'Seeded RTX Demo',
    24576,
    20480,
    8,
    '[]'::jsonb
FROM servers s
WHERE NOT EXISTS (SELECT 1 FROM gpu_resource_snapshots WHERE id = 'gsnap_stage2_demo_001')
ORDER BY s.created_at ASC
LIMIT 1;

INSERT INTO result_comparisons (
    id, experiment_id, run_id, baseline_id, target_result_archive_id, comparison_json, summary_md
)
SELECT
    'rcmp_stage2_demo_001',
    'exp_stage2_demo_001',
    'erun_stage2_demo_001',
    'baseline_stage1_demo_001',
    'rarch_stage1_demo_001',
    '{
      "primaryMetric": "macro_f1",
      "current": 0.78,
      "baseline": 0.71,
      "historicalResult": 0.74,
      "deltaVsBaseline": 0.07,
      "deltaVsHistoricalResult": 0.04,
      "direction": "improved"
    }'::jsonb,
    'Seeded comparison shows improvement over both the stage1 baseline and the archived historical result.'
WHERE EXISTS (SELECT 1 FROM experiment_runs WHERE id = 'erun_stage2_demo_001')
  AND NOT EXISTS (SELECT 1 FROM result_comparisons WHERE id = 'rcmp_stage2_demo_001');
