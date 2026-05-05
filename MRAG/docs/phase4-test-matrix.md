# Phase4 Test Matrix

## Purpose

This document records the baseline validation executed on 2026-03-26 and defines the regression and acceptance matrix that must stay green while phase4 is introduced.

## Baseline Validation Snapshot

The following checks were executed against the current repository state before any phase4 business implementation:

| Layer | Command | Result | Notes |
| --- | --- | --- | --- |
| Go build | `go build -buildvcs=false ./cmd/server` | PASS | Executed in `backend/go` |
| Go unit / module regression | `go test -buildvcs=false ./internal/toolregistry ./internal/skillregistry ./internal/agentmemory ./internal/agentpipeline ./internal/readeragent ./internal/insightagent ./internal/datasetagent ./internal/ideaagent ./internal/planneragent ./internal/codingagent ./internal/writeragent ./internal/handler` | PASS | Matches current stage3 service surface |
| Python import / syntax | `python -m compileall backend/python_agents` | PASS | Confirms runtime package is importable and syntax-valid |
| Python unit | `python -m unittest discover -s backend/python_agents/runtime/tests -p "test_*.py"` | PASS | 31 tests passed |
| Frontend install | `npm.cmd ci` | PASS | Executed at repo root |
| Frontend type + build | `npm.cmd run test:frontend:basic` | PASS | Runs `typecheck` and `build`; Vite warned about large chunks, but build succeeded |
| Stage3 full validation | `Get-Content .\scripts\validate_stage3.sh -Raw \| Invoke-Expression` | PASS | Full controlled chain passed and wrote summary json |
| Runtime smoke | `python backend/python_agents/runtime/runner.py --input <contract> --output <result>` | PASS | Passed with a valid `workspace_dir` in the contract |

## Baseline Integration Evidence

The stage3 validation summary generated during baseline is:

- `workspace/validation/stage3/stage3_validation_summary.json`

The executed validation path confirmed the following chain:

- Reader
- Insight
- Dataset
- Idea
- Planner
- Coding
- Writer

The validation also confirmed stage3 admin, registry, memory, event, and frontend accessibility surfaces.

## Baseline Findings That Are Not Introduced By Step1

### 1. `shenzhenvlab` real probe was unavailable locally

During the baseline run, the phase3 validation script reported:

- `shenzhenvlab_probe = missing`

Meaning:

- the local validation database did not expose a `shenzhenvlab` server record in this environment
- the validation therefore completed through the existing controlled/mock-compatible path

This is a real environmental limitation, not a failure introduced by this step.

### 2. Frontend production bundle is large

Vite reported chunk-size warnings during `npm.cmd run test:frontend:basic`.

Meaning:

- frontend build is healthy
- bundle-splitting may need attention later
- this is not a blocking failure for phase4 step1

## Required Regression Gates For Every Phase4 Step

Unless a layer is truly blocked by environment constraints, each future step should execute all four layers below.

### 1. Static checks

- Go build for affected packages
- frontend typecheck and build
- Python syntax/import check for affected packages

Recommended commands:

```powershell
cd D:\4\MRAG\backend\go
go build -buildvcs=false ./cmd/server
```

```powershell
cd D:\4\MRAG
python -m compileall backend/python_agents backend/python_runners
```

```powershell
cd D:\4\MRAG
npm.cmd run test:frontend:basic
```

### 2. Unit tests

- Go package tests for modified packages
- Python runtime or runner unit tests for the modified slice
- frontend unit-level checks when such tests are added later

Recommended command pattern:

```powershell
cd D:\4\MRAG\backend\go
go test -buildvcs=false ./internal/<changed-package>
```

```powershell
cd D:\4\MRAG
python -m unittest discover -s backend/python_agents/runtime/tests -p "test_*.py"
```

If phase4 runner tests move into `backend/python_runners/tests`, add them to the same baseline suite.

### 3. Integration tests

The stage3 compatibility gate must continue to run while phase4 is additive:

```powershell
cd D:\4\MRAG
Get-Content .\scripts\validate_stage3.sh -Raw | Invoke-Expression
```

Phase4-specific integration tests should be added incrementally, for example:

- dataset registration to Reader context generation
- idea batch generation to persistence and status transitions
- coding run creation to protocol artifact generation
- writing report generation to result archive export

### 4. Minimum real or semi-real smoke path

If the full real environment is unavailable, provide the smallest auditable substitute:

- local runner smoke with a real phase4 contract
- mock SSH boundary with a real directory layout
- controlled dataset fixture that exercises the adapter contract
- report generation smoke with real artifacts from a previous run

Real smoke should be restored as soon as `shenzhenvlab` becomes available in the local environment.

## Future Phase4 Test Matrix

| Workstream | Static | Unit | Integration | Smoke |
| --- | --- | --- | --- | --- |
| Phase4 API and models | Go build, router compile | handler/service tests | versioned API route tests | create and fetch a phase4 resource |
| Reader | Go build, Python compile | reader parser/ranker tests | dataset -> reader context API flow | open-source paper retrieval smoke with fixture or live metadata |
| Idea | Go build, frontend build | idea scoring/status tests | reader context -> idea pool flow | batch generate 10 ideas and persist them |
| Coding protocol | Go build, Python runner compile | protocol schema/layout tests | idea -> run manifest -> local run flow | local retrieval mainline run using VisDoM fixture |
| Remote execution | Go build | config / scheduler tests | server selection + GPU guard flow | shenzhenvlab run with isolated `run_id` directory |
| Writing | Go build, frontend build | report schema tests | run result -> report archive flow | generate one machine report plus one human report |

## Phase4 Real Smoke Plan

Once real remote access is ready, the minimum accepted smoke path should be:

1. register or upload a VisDoM-like dataset
2. generate Reader research context using open metadata and optional uploaded papers
3. generate 10 structured ideas
4. select one idea
5. launch a Coding run on `shenzhenvlab`
6. write outputs under `/home/bzli/mrag/runs/<run_id>` and `/home/bzli/mrag/artifacts/<run_id>`
7. collect metrics, logs, and artifacts
8. generate both machine-readable and human-readable reports

## Step1 Validation Scope

Step1 only introduced documentation assets. No business logic, API behavior, runner behavior, or frontend runtime behavior was changed in this step.

Therefore:

- the executed baseline remains the authoritative code health record for this step
- later code-changing steps must rerun the same baseline and then add phase4-targeted tests
