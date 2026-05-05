# Phase4 File-Level Change Map

## Purpose

This map translates the phase4 target into concrete repository landing points so the next steps can proceed in small, testable slices.

The map is intentionally conservative: it prioritizes additive changes in existing MRAG modules and avoids unnecessary rewrites.

## Step1 Output Boundary

Step1 changed documentation only. No backend, Python runtime, runner, migration, or frontend business files were modified.

## Step2 Recommendation: phase4 API and schema skeleton

### Likely files to modify

- `backend/go/cmd/server/main.go`
- `backend/go/internal/router/router.go`
- `backend/go/internal/config/config.go`
- `src/router/index.ts`
- `src/types/domain.ts`

### Likely files to add

- `backend/go/internal/handler/phase4_reader_handler.go`
- `backend/go/internal/handler/phase4_idea_handler.go`
- `backend/go/internal/handler/phase4_coding_handler.go`
- `backend/go/internal/handler/phase4_writing_handler.go`
- `backend/go/internal/model/phase4_reader_models.go`
- `backend/go/internal/model/phase4_idea_models.go`
- `backend/go/internal/model/phase4_coding_models.go`
- `backend/go/internal/model/phase4_writing_models.go`
- `backend/go/migrations/019_phase4_idea_statuses.sql`
- `backend/go/migrations/020_phase4_run_protocol.sql`
- `backend/go/migrations/021_phase4_reports.sql`

### Why these files first

They create the minimum safe shell for phase4 without forcing the full behavior switch.

## Step3 Recommendation: Reader phase4 landing

### Likely files to modify

- `backend/go/internal/readeragent/service.go`
- `backend/python_agents/runtime/reader_agent.py`
- `backend/python_agents/runtime/tests/test_reader_agent.py`
- `src/api/modules/datasets.ts`
- `src/api/modules/papers.ts`
- `src/views/datasets/DatasetListPage.vue`
- `src/views/papers/PaperDetailPage.vue`

### Likely files to add

- `backend/go/internal/readeragent/phase4_service.go`
- `backend/go/internal/model/phase4_reader_models.go`
- `backend/go/internal/readeragent/phase4_service_test.go`

### Core change

Make dataset registration the primary Reader entry and produce a structured research context while preserving current paper import behavior.

## Step4 Recommendation: Idea batch, scoring, and status system

### Likely files to modify

- `backend/go/internal/ideaagent/service.go`
- `backend/go/internal/service/idea_service.go`
- `backend/go/internal/service/idea_service_structured.go`
- `backend/go/internal/model/idea_agent_models.go`
- `backend/python_agents/runtime/idea_agent.py`
- `backend/python_agents/runtime/tests/test_idea_agent.py`
- `src/types/domain.ts`
- `src/api/modules/ideas.ts`
- `src/views/ideas/IdeaPoolPage.vue`

### Likely files to add

- `backend/go/internal/ideaagent/phase4_service.go`
- `backend/go/internal/ideaagent/phase4_service_test.go`
- `backend/go/internal/model/phase4_idea_models.go`

### Core change

Support batch generation of 10 structured ideas, new scoring dimensions, and the full phase4 status lifecycle.

## Step5 Recommendation: Coding protocol and local mainline

### Likely files to modify

- `backend/go/internal/codingagent/service.go`
- `backend/go/internal/runner/service.go`
- `backend/go/internal/experimentrun`
- `backend/go/internal/model/coding_agent_models.go`
- `backend/python_agents/runtime/coding_agent.py`
- `backend/python_agents/runtime/tests/test_coding_agent.py`

### Likely files to add

- `backend/go/internal/codingagent/phase4_service.go`
- `backend/go/internal/codingagent/repair_policy.go`
- `backend/go/internal/runner/phase4_runner_bridge.go`
- `backend/go/internal/experimentrun/protocol.go`
- `backend/go/internal/experimentrun/protocol_test.go`
- `backend/python_runners/phase4_protocol/__init__.py`
- `backend/python_runners/phase4_protocol/manifest.py`
- `backend/python_runners/phase4_protocol/config.py`
- `backend/python_runners/phase4_protocol/metrics.py`
- `backend/python_runners/phase4_protocol/layout.py`
- `backend/python_runners/retrieval_mainline/main.py`
- `backend/python_runners/retrieval_mainline/eval.py`
- `backend/python_runners/retrieval_mainline/bootstrap.sh`
- `backend/python_runners/retrieval_mainline/adapters/visdom_adapter.py`
- `backend/python_runners/retrieval_mainline/evaluators/page_retrieval.py`

### Files to avoid using as the primary phase4 engine

- `backend/python_templates/mock_train_template/train.py`
- `backend/python_templates/mock_train_template/evaluate.py`

### Core change

Move the primary path from template patching to a unified protocol plus a reusable retrieval mainline snapshot model.

## Step6 Recommendation: remote execution and server isolation

### Likely files to modify

- `backend/go/internal/config/config.go`
- `backend/go/internal/service/server_service.go`
- `backend/go/internal/service/ssh_gateway.go`
- `backend/go/internal/heartbeat`
- `backend/go/internal/gpuresource`

### Likely files to add

- `backend/go/internal/service/remote_layout.go`
- `backend/go/internal/service/remote_layout_test.go`

### Core change

Switch the real execution boundary to:

- `/home/bzli/mrag/datasets`
- `/home/bzli/mrag/runs/<run_id>`
- `/home/bzli/mrag/artifacts/<run_id>`
- `/home/bzli/mrag/cache`
- `/home/bzli/mrag/envs`

while keeping GPU-idle selection and per-run isolation auditable.

## Step7 Recommendation: Writing report closure

### Likely files to modify

- `backend/go/internal/writeragent/service.go`
- `backend/go/internal/service/result_archive_service.go`
- `backend/go/internal/model/writer_agent_models.go`
- `backend/python_agents/runtime/writer_agent.py`
- `backend/python_agents/runtime/tests/test_writer_agent.py`
- `src/api/modules/experiments.ts`
- `src/views/research/ResultArchivePage.vue`

### Likely files to add

- `backend/go/internal/writeragent/phase4_service.go`
- `backend/go/internal/model/phase4_writing_models.go`

### Core change

Produce both structured and human-readable experiment reports from real run artifacts.

## Step8 Recommendation: frontend completion

### Likely files to modify

- `src/router/index.ts`
- `src/types/domain.ts`
- `src/api/modules/datasets.ts`
- `src/api/modules/papers.ts`
- `src/api/modules/ideas.ts`
- `src/api/modules/experiments.ts`
- `src/views/datasets/DatasetListPage.vue`
- `src/views/papers/PaperDetailPage.vue`
- `src/views/ideas/IdeaPoolPage.vue`
- `src/views/experiments/ExperimentDetailPage.vue`
- `src/views/research/ResultArchivePage.vue`

### Likely files to add

- `src/components/phase4/`
- `src/views/phase4/` only if existing pages become too crowded

### Core change

Complete the dataset-driven operator flow without replacing the current frontend shell.

## Files That Should Remain Out Of Scope Unless Evidence Forces A Change

- `Auto_v1`
- `useless`
- broad rewrites of stage3 handlers
- deletion of stage3 templates
- deletion of stage3 planner, dataset, or insight flows before phase4 replacement is validated

## Minimal Safe Rule For The Next Steps

For each future step:

1. change the smallest file set that can deliver the slice
2. add tests next to the changed module
3. rerun the baseline regression gate
4. keep rollback localized to the touched slice
