# Phase4 Stage3 Compatibility And Regression Guard

## Purpose

Phase4 must be introduced without silently degrading the already verified phase3 system. This document defines what must remain stable, what can change additively, and how to roll back if a phase4 increment regresses the repository.

## Stage3 Assets That Must Stay Protected

### Verified chain and orchestration

- `backend/go/cmd/server/main.go`
- `backend/go/internal/agentpipeline`
- `backend/go/internal/readeragent`
- `backend/go/internal/insightagent`
- `backend/go/internal/datasetagent`
- `backend/go/internal/ideaagent`
- `backend/go/internal/planneragent`
- `backend/go/internal/codingagent`
- `backend/go/internal/writeragent`

### Runtime and templates

- `backend/go/internal/agentruntime`
- `backend/python_agents/runtime`
- `backend/python_templates/mock_train_template`

### Experiment and archive infrastructure

- `backend/go/internal/service/experiment_service.go`
- `backend/go/internal/runner`
- `backend/go/internal/experimentrun`
- `backend/go/internal/service/result_archive_service.go`
- `backend/go/internal/resultcompare`

### Existing UI entry points

- `src/views/datasets/DatasetListPage.vue`
- `src/views/papers/PaperDetailPage.vue`
- `src/views/ideas/IdeaPoolPage.vue`
- `src/views/experiments/ExperimentDetailPage.vue`
- `src/views/research/ResultArchivePage.vue`

## Compatibility Strategy

### 1. Prefer additive versioning over breaking mutation

Recommended approach:

- preserve existing `/api/v1` routes and payloads during the transition
- introduce phase4 behavior through additive handlers and versioned endpoints
- migrate the frontend incrementally to the new routes only after the backend surface is stable

### 2. Keep phase3 agents callable during the transition

Even though phase4 removes `Insight`, `Dataset`, and `Planner` as first-class target nodes, the codebase should keep those stage3 services runnable until phase4 closes the same responsibilities through integrated flows.

This means:

- no early deletion of stage3 handlers
- no early deletion of stage3 database fields
- no early deletion of stage3 templates

### 3. Introduce feature gates for phase4 pipeline activation

Recommended future gates:

- `PHASE4_API_ENABLED`
- `PHASE4_PIPELINE_ENABLED`
- `PHASE4_READER_ENABLED`
- `PHASE4_CODING_MAINLINE_ENABLED`
- `PHASE4_REMOTE_ROOT`

These names are recommendations for future implementation. They do not exist yet in the repository.

### 4. Preserve workspace compatibility

Phase4 should add new artifacts without overwriting stage3 assets.

Rules:

- do not overwrite `workspace/experiments/<experiment_id>` data created by stage3 without an explicit compatibility bridge
- append new report assets instead of replacing current result archive records
- keep stage3 validation output location untouched:
  - `workspace/validation/stage3/stage3_validation_summary.json`

### 5. Preserve stage3 template path as a regression target

Even though phase4 aims to remove template dependence from the main path, the current stage3 template surface should remain in place until:

- phase4 local runner is green
- phase4 remote smoke is green
- stage3 regression remains green after the change

## Regression Gate Checklist

Every code-changing phase4 step should keep the following checklist visible:

1. Go build passes.
2. affected Go unit tests pass.
3. Python runtime and runner tests pass.
4. frontend typecheck and build pass.
5. `scripts/validate_stage3.sh` still passes.
6. no stage3 workspace layout is silently overwritten.
7. no stage3 API consumer is broken without an explicit compatibility path.

## Rollback Strategy

### API rollback

If a phase4 route or payload breaks clients:

- disable the phase4 route surface
- leave `/api/v1` intact
- roll back the new handler registration before mutating shared services

### Data model rollback

If new phase4 fields or statuses break persistence:

- keep old fields readable
- avoid irreversible migration steps until the new read/write path is green
- prefer additive columns or additive tables over destructive renames

### Coding path rollback

If phase4 Coding mainline becomes unstable:

- keep the stage3 template path available
- roll back only the phase4 runner bridge
- do not remove `backend/python_templates/mock_train_template` until phase4 replacement is validated

### Remote execution rollback

If `/home/bzli/mrag` migration or remote isolation logic fails:

- disable the phase4 remote path
- keep local or mock smoke coverage running
- do not change stage3 execution assumptions silently

### Frontend rollback

If a page-level retrofit breaks the user flow:

- revert the page to the current stage3-compatible data contract
- keep new phase4 widgets behind explicit route or feature selection

## Explicit Areas To Avoid As Primary Phase4 Carriers

The following locations should not become the main phase4 implementation surface:

- `Auto_v1`
- `useless`

They may be used as reference material only if needed.

## Why Step1 Did Not Change Business Code

Step1 is intentionally limited to baseline inspection, blueprinting, test planning, and regression guard definition because:

- the current repository already has a verified phase3 chain
- phase4 introduces architectural movement, not a single isolated bug fix
- premature code changes would make it harder to identify whether future failures are architectural or accidental

This is why step1 only adds documentation assets and records the current validated baseline.
