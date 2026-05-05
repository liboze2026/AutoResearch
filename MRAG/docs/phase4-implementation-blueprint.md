# Phase4 Implementation Blueprint

## Purpose

This document captures a repository-grounded implementation blueprint for MRAG phase4 as of 2026-03-26. It is based on the current codebase state in `D:\4\MRAG` and is intentionally additive: the goal is to enable phase4 without breaking the verified phase3 chain.

This step does not introduce phase4 business logic. It only fixes the implementation target, module landing points, sequencing, and validation strategy.

## Baseline Inventory

### Real repository root

- MRAG root: `D:\4\MRAG`

### Current backend structure

- Go backend root: `backend/go`
- Server entrypoint: `backend/go/cmd/server/main.go`
- HTTP router: `backend/go/internal/router/router.go`
- Agent pipeline and runtime:
  - `backend/go/internal/agentpipeline`
  - `backend/go/internal/agentruntime`
  - `backend/go/internal/agenttrigger`
  - `backend/go/internal/agentjob`
- Stage3 first-class agents:
  - `backend/go/internal/readeragent`
  - `backend/go/internal/insightagent`
  - `backend/go/internal/datasetagent`
  - `backend/go/internal/ideaagent`
  - `backend/go/internal/planneragent`
  - `backend/go/internal/codingagent`
  - `backend/go/internal/writeragent`
- Experiment / run / archive:
  - `backend/go/internal/service/experiment_service.go`
  - `backend/go/internal/runner`
  - `backend/go/internal/experimentrun`
  - `backend/go/internal/service/result_archive_service.go`
  - `backend/go/internal/resultcompare`
- Remote execution / server / GPU:
  - `backend/go/internal/service/server_service.go`
  - `backend/go/internal/service/ssh_gateway.go`
  - `backend/go/internal/heartbeat`
  - `backend/go/internal/gpuresource`

### Current Python runtime structure

- Agent runtime root: `backend/python_agents/runtime`
- Runtime entrypoint: `backend/python_agents/runtime/runner.py`
- Current stage3 agents:
  - `reader_agent.py`
  - `insight_agent.py`
  - `dataset_agent.py`
  - `idea_agent.py`
  - `planner_agent.py`
  - `coding_agent.py`
  - `writer_agent.py`
- Runtime tests: `backend/python_agents/runtime/tests`
- Stage2/stage3 template surface:
  - `backend/python_templates/mock_train_template`
- Reserved runner surface suitable for phase4 mainline:
  - `backend/python_runners`

### Current frontend structure

- Frontend root: `src`
- Router: `src/router/index.ts`
- Current reusable pages:
  - dataset entry: `src/views/datasets/DatasetListPage.vue`
  - paper / reader outputs: `src/views/papers/PaperDetailPage.vue`
  - idea pool: `src/views/ideas/IdeaPoolPage.vue`
  - experiment / run / logs: `src/views/experiments/ExperimentDetailPage.vue`
  - result archive: `src/views/research/ResultArchivePage.vue`
- Current domain types: `src/types/domain.ts`
- Current API clients:
  - `src/api/modules/datasets.ts`
  - `src/api/modules/papers.ts`
  - `src/api/modules/ideas.ts`
  - `src/api/modules/experiments.ts`
  - `src/api/modules/researchAssets.ts`
  - `src/api/modules/servers.ts`

### Current stage3 chain

The current verified stage3 chain wired in `backend/go/cmd/server/main.go` is:

`paper -> insight -> dataset -> idea -> plan -> coding/eval -> writer`

Current pipeline event flow in `backend/go/internal/agentpipeline/service.go` uses:

- `paper_imported`
- `paper_parsed`
- `insights_ready`
- `dataset_asset_ready`
- `idea_ready`
- `plan_ready`
- `experiment_result_ready`
- `draft_ready`

### Current workspace and artifact landing points

- papers: `workspace/papers`
- datasets: `workspace/datasets`
- ideas: `workspace/ideas`
- experiments: `workspace/experiments`
- results: `workspace/results`
- writing drafts: `workspace/writing`
- agent memory: `workspace/memory`
- agent jobs: `workspace/agents/jobs`
- validation summaries: `workspace/validation`

## Phase4 Target Topology

Phase4 target topology is:

`Reader -> Idea -> Coding -> Writing`

And the system entry becomes dataset-driven:

1. manual dataset upload / registration is the primary entry
2. manual paper upload is an optional Reader input
3. manual idea input is an optional Idea input

Phase4 first-class agents are reduced to:

- Reader Agent
- Idea Agent
- Coding Agent
- Writing Agent

Phase4 capabilities that must be demoted from first-class agents into tools or embedded abilities:

- Insight -> internal tool of Idea Agent
- Dataset -> internal tool of Coding Agent
- Evaluate -> internal tool of Coding Agent
- Planner -> merged into upper-layer orchestration

## Hard Grounded Gaps Between Stage3 and Phase4

The repository inspection shows the following real gaps:

1. `planneragent`, `datasetagent`, and `insightagent` are still first-class stage3 nodes and pipeline subscriptions.
2. `codingagent` is still coupled to `backend/go/internal/traintemplate/service.go` and `backend/python_templates/mock_train_template`.
3. current reader flow in `backend/python_agents/runtime/reader_agent.py` is controlled and mock-friendly, not a true open academic retrieval implementation.
4. current idea persistence and UI only support the limited stage3 status set (`draft`, `shortlisted`, `archived`).
5. current writer outputs a draft scaffold, not the required phase4 experiment report contract.
6. current remote work-root defaults still point to `/home/bzli/lbz`; phase4 requires `/home/bzli/mrag`.

These gaps define the real implementation scope. Phase4 should be built as an additive transition rather than a destructive rewrite.

## Implementation Principles

1. Keep phase3 callable and testable while phase4 is being introduced.
2. Prefer additive modules, compatibility layers, or versioned APIs over in-place breaking rewrites.
3. Reuse current services, repositories, workspace contracts, and pages whenever possible.
4. Move template-bound code out of the critical path gradually; do not delete stage3 templates until phase4 has real green smoke coverage.
5. Every phase4 increment must remain compilable, testable, explainable, and rollbackable.

## Module-Level Implementation Blueprint

## 1. API and data model layer

### Current landing points

- `backend/go/internal/router/router.go`
- `backend/go/internal/model`
- `backend/go/internal/handler`
- `backend/go/migrations`

### Phase4 plan

Add a phase4 additive surface under new versioned routes instead of mutating all `/api/v1` behavior in place.

Recommended additions:

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

### Why this is the right landing point

These folders already own HTTP contract, persistence models, and schema evolution. Adding phase4 here keeps the change traceable and reversible.

## 2. Reader Agent

### Current landing points

- Go orchestration: `backend/go/internal/readeragent/service.go`
- Python execution: `backend/python_agents/runtime/reader_agent.py`
- paper ingestion and storage are already reachable through existing paper services and handlers

### Phase4 plan

Reader must support:

- dataset-first research context generation
- optional user-uploaded paper augmentation
- true open academic retrieval without paid download dependency
- paper ranking by quality, not only source label

Recommended file-level additions and edits:

- add `backend/go/internal/readeragent/phase4_service.go`
- extend `backend/go/internal/readeragent/service.go` only where compatibility bridging is unavoidable
- extend `backend/python_agents/runtime/reader_agent.py`
- add `backend/python_agents/runtime/tests/test_reader_agent.py` coverage for phase4 structured output
- add `backend/go/internal/model/phase4_reader_models.go`

Recommended future source connectors should remain open-access oriented, for example:

- OpenAlex / Crossref metadata
- arXiv
- open-access links when available
- manual uploaded PDFs and metadata

### Output contract to add

Reader output should include a structured research context payload with at least:

- dataset summary
- task framing
- ranked papers
- extracted related-work observations
- retrieval-stage-specific notes
- citations and source provenance

## 3. Idea Agent

### Current landing points

- orchestration: `backend/go/internal/ideaagent/service.go`
- persistence: `backend/go/internal/service/idea_service.go`
- structured persistence: `backend/go/internal/service/idea_service_structured.go`
- Python execution: `backend/python_agents/runtime/idea_agent.py`
- frontend page: `src/views/ideas/IdeaPoolPage.vue`

### Phase4 plan

Idea Agent must:

- generate 10 structured ideas by default
- directly produce high-granularity plans
- score ideas on the required dimensions
- support new phase4 idea statuses
- accept manual idea input as augmentation

Recommended edits:

- update `backend/go/internal/service/idea_service.go`
- update `backend/go/internal/service/idea_service_structured.go`
- update `backend/go/internal/model/idea_agent_models.go`
- add `backend/go/internal/model/phase4_idea_models.go`
- add `backend/go/internal/ideaagent/phase4_service.go`
- extend `backend/python_agents/runtime/idea_agent.py`
- extend `backend/python_agents/runtime/tests/test_idea_agent.py`
- update `src/types/domain.ts`
- update `src/api/modules/ideas.ts`
- update `src/views/ideas/IdeaPoolPage.vue`

### New idea status set to introduce

- `draft`
- `scored`
- `rejected`
- `selected`
- `implemented`
- `failed`
- `archived`

### Minimum phase4 idea structure

- problem definition
- core method
- differentiators
- data processing needs
- model changes
- training plan
- evaluation metrics
- risk points
- expected gains

### Minimum scoring dimensions

- novelty
- dataset fit
- feasibility
- expected gain
- compute cost
- failure risk
- reproducibility

## 4. Coding Agent and unified run protocol

### Current landing points

- orchestration: `backend/go/internal/codingagent/service.go`
- runner orchestration: `backend/go/internal/runner`
- experiment run package: `backend/go/internal/experimentrun`
- template coupling: `backend/go/internal/traintemplate/service.go`
- Python execution: `backend/python_agents/runtime/coding_agent.py`
- current template surface: `backend/python_templates/mock_train_template`
- current reserved runner surface: `backend/python_runners`

### Phase4 plan

Coding Agent is the phase4 pivot. The implementation should:

- remove fixed train/evaluate template dependence from the primary path
- keep a unified minimum run protocol
- generate and execute real retrieval research code from dataset plus idea
- support repair-first retries with configurable max retry count

Recommended implementation split:

- keep `backend/python_templates/mock_train_template` for stage3 regression only
- add phase4 mainline under `backend/python_runners/retrieval_mainline`
- add shared run protocol helpers under `backend/python_runners/phase4_protocol`
- add Go protocol bridge under `backend/go/internal/experimentrun/protocol.go`
- add retry / repair coordination under `backend/go/internal/codingagent/phase4_service.go`
- add compatibility bridge in `backend/go/internal/runner`

Recommended future files:

- `backend/go/internal/experimentrun/protocol.go`
- `backend/go/internal/experimentrun/protocol_test.go`
- `backend/go/internal/codingagent/phase4_service.go`
- `backend/go/internal/codingagent/repair_policy.go`
- `backend/go/internal/runner/phase4_runner_bridge.go`
- `backend/python_runners/phase4_protocol/__init__.py`
- `backend/python_runners/phase4_protocol/manifest.py`
- `backend/python_runners/phase4_protocol/config.py`
- `backend/python_runners/phase4_protocol/metrics.py`
- `backend/python_runners/phase4_protocol/layout.py`
- `backend/python_runners/retrieval_mainline/main.py`
- `backend/python_runners/retrieval_mainline/eval.py`
- `backend/python_runners/retrieval_mainline/bootstrap.sh`
- `backend/python_runners/retrieval_mainline/adapters/`
- `backend/python_runners/retrieval_mainline/methods/`

### Minimum unified run protocol that must be enforced

- experiment manifest
- dataset adapter contract
- config format
- run entrypoint
- eval entrypoint
- metrics output schema
- artifact directory layout
- logs layout
- retry / repair hooks
- environment bootstrap script

### Repair and retry policy to implement

- configurable `max_auto_retries`, default `3`
- first priority: fix runtime failures
- second priority: small code or parameter adjustment
- third priority: fallback to previous stable snapshot
- after repeated failure: mark run as `test_failed` and feed failure context back to Idea flow

## 5. Dataset analysis and evaluation as Coding internal tools

### Current landing points

- dataset services:
  - `backend/go/internal/service/dataset_service.go`
  - `backend/go/internal/service/dataset_asset_service.go`
- dataset agent:
  - `backend/go/internal/datasetagent/service.go`
- evaluation artifacts:
  - `backend/go/internal/resultcompare`
  - `backend/go/internal/service/result_archive_service.go`

### Phase4 plan

Do not delete stage3 dataset assets. Instead, move phase4 dataset analysis and evaluation logic under Coding-owned services and runners.

Recommended additions:

- `backend/go/internal/codingagent/dataset_tool.go`
- `backend/go/internal/codingagent/evaluate_tool.go`
- `backend/python_runners/retrieval_mainline/adapters/visdom_adapter.py`
- `backend/python_runners/retrieval_mainline/evaluators/page_retrieval.py`

This preserves current dataset asset management while shifting execution authority to Coding Agent.

## 6. Shenzhenvlab execution boundary

### Current landing points

- `backend/go/internal/config/config.go`
- `backend/go/internal/service/server_service.go`
- `backend/go/internal/service/ssh_gateway.go`
- `backend/go/internal/heartbeat`
- `backend/go/internal/gpuresource`

### Phase4 plan

Phase4 real execution must be constrained to:

- server: `shenzhenvlab`
- root: `/home/bzli/mrag`
- directories:
  - `/home/bzli/mrag/datasets`
  - `/home/bzli/mrag/runs/<run_id>`
  - `/home/bzli/mrag/artifacts/<run_id>`
  - `/home/bzli/mrag/cache`
  - `/home/bzli/mrag/envs`

Required future changes:

- update remote root defaults in `backend/go/internal/config/config.go`
- update server-side directory creation logic in `backend/go/internal/service/server_service.go`
- keep GPU-idle checks in the scheduling path
- ensure every run is isolated by `run_id`
- never overwrite unrelated user directories

No remote-root mutation is made in step1. This remains a planned phase4 change because it affects verified stage3 behavior.

## 7. Writing Agent

### Current landing points

- orchestration: `backend/go/internal/writeragent/service.go`
- Python execution: `backend/python_agents/runtime/writer_agent.py`
- result archive: `backend/go/internal/service/result_archive_service.go`
- frontend result page: `src/views/research/ResultArchivePage.vue`

### Phase4 plan

Writing Agent should become an experiment-report closer instead of a generic paper draft generator.

Recommended edits:

- add `backend/go/internal/writeragent/phase4_service.go`
- add `backend/go/internal/model/phase4_writing_models.go`
- extend `backend/python_agents/runtime/writer_agent.py`
- extend `backend/python_agents/runtime/tests/test_writer_agent.py`
- update `src/api/modules/experiments.ts`
- update `src/views/research/ResultArchivePage.vue`

### Required output layers

- machine-readable structured report
- human-readable experiment report

### Minimum human-readable report sections

- title
- dataset and task
- related work
- idea summary
- implementation method
- experimental setup
- results and analysis
- limitations and next steps
- references

## 8. Frontend incremental retrofit

### Reuse-first targets

- `src/views/datasets/DatasetListPage.vue`
- `src/views/papers/PaperDetailPage.vue`
- `src/views/ideas/IdeaPoolPage.vue`
- `src/views/experiments/ExperimentDetailPage.vue`
- `src/views/research/ResultArchivePage.vue`

### Phase4 plan

Prefer incremental page upgrades over a frontend rewrite:

- dataset page becomes the main phase4 entry
- paper page exposes Reader research context results
- idea pool page adds batch scoring, filtering, archiving, selection, and status transitions
- experiment page exposes coding run start, logs, metrics, retries, and artifacts
- result page exposes report preview and export

If shared widgets are needed, add small focused components instead of replacing the page shell.

## Proposed Transition Strategy

### Step A: additive API and schema foundation

- introduce phase4 models, migrations, and route surface
- keep stage3 routes intact

### Step B: Reader dataset-first context generation

- add phase4 Reader service and structured output
- keep current paper import and stage3 reader available

### Step C: Idea batch generation, scoring, and status model

- extend idea persistence and UI
- keep old status values readable for backward compatibility

### Step D: Coding run protocol and local phase4 mainline

- introduce retrieval mainline and run protocol
- keep stage3 template path as fallback and regression target only

### Step E: shenzhenvlab bridge and real smoke path

- move remote path defaults to `/home/bzli/mrag`
- verify GPU-idle gating and per-run isolation

### Step F: Writing report closing and frontend completion

- add dual report outputs
- complete page-level report viewing and export

## Proposed Phase4 Events

These are recommended additions, not current implemented events:

- `dataset_registered`
- `reader_context_ready`
- `idea_batch_ready`
- `idea_selected`
- `coding_run_started`
- `coding_run_failed`
- `coding_run_succeeded`
- `report_generated`

These should be added to `backend/go/internal/agentpipeline/service.go` only after phase4 services exist and are feature-gated.

## Explicit Non-Goals For Step1

Step1 must not:

- rewrite the stage3 chain
- delete planner / dataset / insight services
- replace current template code with incomplete phase4 code
- change remote server root in production paths without a regression-safe bridge
- push a frontend redesign

## Acceptance Definition For The Next Steps

The first acceptable phase4 increments are the ones that:

1. add the phase4 surface without breaking stage3 validation
2. keep Go, Python, and frontend baselines green
3. add small, test-backed, rollbackable slices
4. converge toward a real VisDoM retrieval run and report generation
