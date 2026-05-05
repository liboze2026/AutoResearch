# Stage3 Validation Report

## Scope

- Validation date: 2026-03-25
- Validator: Codex desktop local rerun
- Validation entrypoint: `scripts/validate_stage3.sh`
- Run command: `Get-Content .\scripts\validate_stage3.sh -Raw | Invoke-Expression`
- Backend API base: `http://127.0.0.1:18080/api/v1`
- Frontend base: `http://127.0.0.1:4173`
- Runtime mode: `mock + codex_cli fallback + controlled real-server probe`
- Optional real-server probe result: `missing` (`shenzhenvlab record not found`)

## Checklist

- [x] Agent runtime runner starts successfully
- [x] Mock executor works
- [x] `codex_cli` fallback works
- [x] Schema validator works
- [x] Schema repair works
- [x] Tool registry works
- [x] Skill registry works
- [x] Memory store works
- [x] Reader Agent minimal test passed
- [x] Insight Agent minimal test passed
- [x] Dataset Agent minimal test passed
- [x] Idea Generator Agent minimal test passed
- [x] Planner Agent minimal test passed
- [x] Coding/Evaluator Agent minimal test passed
- [x] Writer Agent minimal test passed
- [x] Frontend agent pages are reachable
- [x] End-to-end chain `paper -> insight -> dataset -> idea -> plan -> coding/eval -> writer` passed
- [x] Final result is PASS

## Environment

- Git commit / branch: unavailable, current workspace is not a git repository snapshot
- OS: `Microsoft Windows NT 10.0.22631.0`
- Docker version: `Docker version 29.2.1, build a5c7197`
- Node / npm version: `v20.12.0 / 10.5.0`
- Go version: `go version go1.26.1 windows/amd64`
- Python version: `Python 3.12.4`
- Database: `mrag_stage3_validation`

## Real Server Probe

- `shenzhenvlab` record found: no
- Connection / heartbeat result: not executed
- GPU check result: not executed
- Idle GPU available: unknown
- Validation path chosen: fallback to controlled mock path

## Validation Output

```text
[stage3] go stage3 package tests
[stage3] python stage3 runtime unit tests
[stage3] runtime runner mock and codex fallback checks
[stage3] frontend typecheck
[stage3] prepare local validation database
[stage3] build local backend executable
[stage3] start stage3 probe backend
[stage3] probe shenzhenvlab availability
[stage3] real_probe_result: missing
[stage3] real_probe_message: shenzhenvlab record not found
[stage3] start main stage3 validation backend
[stage3] start frontend dev server
[stage3] create validation mock server
[stage3] register validation tool and skill
[stage3] prepare writer template
[stage3] reader agent minimal test
[stage3] insight agent minimal test
[stage3] dataset agent minimal test
[stage3] idea generator agent minimal test
[stage3] planner agent minimal test
[stage3] coding evaluator agent minimal test
[stage3] writer agent minimal test
[stage3] agent admin and frontend accessibility checks
PASS: stage3 validation passed
- runtime_runner_mock: succeeded
- runtime_runner_codex_fallback: succeeded
- schema_validator_repair: succeeded
- tool_registry: tool_1774449090_6fbe5d20
- skill_registry: skill_1774449090_ad0ef29a
- memory_store: mem_1774449093_71fb93f3
- reader_job_id: ajob_1774449091_2afc287a
- insight_job_id: ajob_1774449091_9aacc5aa
- dataset_job_id: ajob_1774449092_d6e6de08
- idea_job_id: ajob_1774449093_750dca7a
- planner_job_id: ajob_1774449094_0e1fea5d
- coding_job_id: ajob_1774449095_a5a94f7e
- writer_job_id: ajob_1774449097_7967df27
- experiment_id: exp_1774449095_ca6cdb1e
- run_id: run_1774449096_455a14bb
- draft_id: draft_1774449098_c3fb342e
- shenzhenvlab_probe: missing
- summary_path: D:\3\MRAG\workspace\validation\stage3\stage3_validation_summary.json
```

- Validation summary json: `workspace/validation/stage3/stage3_validation_summary.json`

## Created Objects

- tool_id: `tool_1774449090_6fbe5d20`
- skill_id: `skill_1774449090_ad0ef29a`
- reader_job_id: `ajob_1774449091_2afc287a`
- paper_id: `paper_1774449091_0b9df0c2`
- insight_job_id: `ajob_1774449091_9aacc5aa`
- insight_id: `pinsight_1774449092_8407117f`
- dataset_job_id: `ajob_1774449092_d6e6de08`
- dataset_asset_id: `dasset_1774449093_0b0decce`
- dataset_memory_id: `mem_1774449093_71fb93f3`
- idea_job_id: `ajob_1774449093_750dca7a`
- idea_id: `idea_1774449094_3bd371d4`
- planner_job_id: `ajob_1774449094_0e1fea5d`
- experiment_id: `exp_1774449095_ca6cdb1e`
- coding_job_id: `ajob_1774449095_a5a94f7e`
- run_id: `run_1774449096_455a14bb`
- result_archive_id: `archive_1774449097_73c8507c`
- writer_job_id: `ajob_1774449097_7967df27`
- draft_id: `draft_1774449098_c3fb342e`

## Failure Localization

- Failed step: none in final rerun
- Error message: n/a
- Log path: `D:\3\MRAG\.stage3-logs\backend-main-1774449048.log`
- Backend / frontend evidence:
  - backend log shows successful `POST /api/v1/agents/*/run` and subsequent `GET /api/v1/agents/jobs/*`
  - backend log also shows successful access to `/api/v1/agents`, `/api/v1/agents/jobs`, `/api/v1/agent-events`
  - summary file persists all created object ids for rerun audit

If a future rerun fails, `validate_stage3.sh` will print the failed step, error message, log path, and the tail of the relevant log file.

## Issues Found During Closure

- Issue: `Dataset Agent` could emit an empty `dataset_location` when no scanned dataset was available.
  - Impact: validator/repair acceptance could fail on the minimal dataset path.
  - Fix: add planned fallback dataset location generation and extend runtime unit tests.
- Issue: `Planner / Coding / Writer` codex-cli fallback normalization did not consistently preserve fallback metadata.
  - Impact: acceptance evidence around `codex_cli -> mock` fallback was incomplete or misleading.
  - Fix: normalize through the shared base executor path and add dedicated fallback tests.
- Issue: validation reruns could collide on reused backend/frontend log names or leave stale processes.
  - Impact: reruns were less stable and failures were harder to localize.
  - Fix: move to stamp-specific logs and strengthen background process cleanup.
- Issue: validation script assumed `result_archive_id` at the wrong response level for coding verification.
  - Impact: coding/evaluator acceptance could fail even when the run itself succeeded.
  - Fix: read the normalized payload correctly and verify via the experiment comparison API.

## Final Result

- Result: PASS
- Stage3 boundary respected: yes
- Notes:
  - current system is still a controlled-agent system, not a fully autonomous research system
  - current `Writer / Picture` remains a minimal version
  - current `Coding` remains constrained within the unified training template
  - real model execution and real crawling/retrieval remain controlled extension points
  - within the agreed stage3 boundary, completion is 100%
