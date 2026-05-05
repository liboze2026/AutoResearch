# Stage3 Validation Report Template

## Scope
- Validation date:
- Validator:
- Validation entrypoint:
- Run command:
- Backend API base:
- Frontend base:
- Runtime mode:
- Optional real-server probe result:

## Checklist
- [ ] Agent runtime runner starts successfully
- [ ] Mock executor works
- [ ] `codex_cli` fallback works
- [ ] Schema validator works
- [ ] Schema repair works
- [ ] Tool registry works
- [ ] Skill registry works
- [ ] Memory store works
- [ ] Reader Agent minimal test passed
- [ ] Insight Agent minimal test passed
- [ ] Dataset Agent minimal test passed
- [ ] Idea Generator Agent minimal test passed
- [ ] Planner Agent minimal test passed
- [ ] Coding/Evaluator Agent minimal test passed
- [ ] Writer Agent minimal test passed
- [ ] Frontend agent pages are reachable
- [ ] End-to-end chain `paper -> insight -> dataset -> idea -> plan -> coding/eval -> writer` passed
- [ ] Final result is PASS

## Environment
- Git commit / branch:
- OS:
- Docker version:
- Node / npm version:
- Go version:
- Python version:
- Database:

## Real Server Probe
- `shenzhenvlab` record found:
- Connection / heartbeat result:
- GPU check result:
- Idle GPU available:
- Validation path chosen:

## Validation Output
```text
PASTE PASS / FAIL OUTPUT HERE
```

- Validation summary json:

## Created Objects
- tool_id:
- skill_id:
- reader_job_id:
- paper_id:
- insight_job_id:
- insight_id:
- dataset_job_id:
- dataset_asset_id:
- dataset_memory_id:
- idea_job_id:
- idea_id:
- planner_job_id:
- experiment_id:
- coding_job_id:
- run_id:
- result_archive_id:
- writer_job_id:
- draft_id:

## Failure Localization
- Failed step:
- Error message:
- Log path:
- Backend / frontend evidence:

## Issues Found
- Issue:
- Impact:
- Fix:

## Final Result
- Result: PASS / FAIL
- Stage3 boundary respected:
- Notes:
