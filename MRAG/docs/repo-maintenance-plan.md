# MRAG Repo Maintenance Plan

## Goal

This document captures the first-stage cleanup strategy for making `MRAG` easier to maintain without breaking the current stage3 runtime, API, frontend, or validation flow.

## Current Project Map

### `D:\3`

- `MRAG`
  - Main active project.
- `Auto_v1`
  - Legacy stage0 prototype kept as a reference project.
- `useless`
  - Reserved archive location for reviewed unnecessary files.

### `D:\3\MRAG`

- `backend/go`
  - Main Go API, migrations, runtime control-plane modules, server entrypoint.
- `backend/python_agents/runtime`
  - Controlled stage3 runtime, validators, repairers, executors, and agent implementations.
- `backend/python_runners`
  - Runner-facing placeholder area, currently light-weight.
- `src`
  - Vue frontend.
- `scripts`
  - Stage validation entrypoints.
- `docs`
  - Stage specifications, acceptance reports, audits, and delivery notes.
- `workspace`
  - Runtime artifacts and persisted data.

## High-Value Maintainability Problems Found

### 1. Workspace governance is under-specified

- `D:\3` did not have a root workspace README before cleanup.
- The active project and the legacy project share one top-level repository view, which makes project ownership unclear.

### 2. `MRAG` had no project-local ignore policy

- `MRAG` did not have its own `.gitignore`.
- Generated outputs were allowed to accumulate in the project tree with no local boundary rules.

### 3. Generated artifacts are mixed with source-management areas

Observed tracked files inside `MRAG` include:

- `13262` files under `node_modules`
- `51` files under `dist`
- `47` log-like files
- `4` executable files
- `32` Python bytecode files

This makes repository review, navigation, and future cleanup much harder than necessary.

### 4. Build outputs and test outputs sit beside source files

Examples:

- `backend/go/*.exe`
- `backend/go/*.log`
- `.stage3-logs/*`
- `.stage2-frontend.log`
- `tmp_test_script.sh`

### 5. Documentation entrypoints are fragmented

- There are many useful stage documents, but there was no single maintenance-oriented map for cleanup sequencing.
- `backend/README.md` currently appears to contain encoding damage and should be repaired in a later cleanup pass.

## Phase Plan

### Phase 1. Governance and boundaries

- Add workspace-level and project-level guidance.
- Add ignore rules for generated artifacts.
- Record cleanup boundaries so runtime outputs are not confused with source.

### Phase 2. Source-tree and entrypoint cleanup

- Normalize top-level project entrypoints.
- Repair broken or unclear documentation entry files.
- Reduce ambiguity around which directories are active code, legacy reference, or generated output.

### Phase 3. Reviewed archive pass

- Produce a candidate unnecessary-file list.
- Move only user-approved candidates into `D:\3\useless`, preserving the original relative structure.

### Phase 4. Risk scan and staged validation

- Review architecture-level risks.
- Review code-level bug and fragility risks.
- Review config, script, and environment dependency risks.
- Re-run staged validations to ensure cleanup did not break the project.

## Candidate Unnecessary Files Or Directories

These are candidates only. They should not be moved until explicitly reviewed.

### `MRAG`

- `D:\3\MRAG\node_modules`
  - Reinstallable dependency output, not authored source.
- `D:\3\MRAG\dist`
  - Frontend build output.
- `D:\3\MRAG\.stage3-logs`
  - Validation logs.
- `D:\3\MRAG\.stage2-frontend.log`
- `D:\3\MRAG\.stage2-frontend.err.log`
- `D:\3\MRAG\.stage2_validate_state`
- `D:\3\MRAG\tmp_test_script.sh`
- `D:\3\MRAG\backend\go\mrag-server.exe`
- `D:\3\MRAG\backend\go\server.exe`
- `D:\3\MRAG\backend\go\stage3-validation-server.exe`
- `D:\3\MRAG\backend\go\idea_flow_server.err.log`
- `D:\3\MRAG\backend\go\idea_flow_server.log`
- `D:\3\MRAG\backend\go\paper_flow_server.err.log`
- `D:\3\MRAG\backend\go\paper_flow_server.out.log`
- `D:\3\MRAG\backend\python_agents\**\__pycache__`

### `Auto_v1`

- `D:\3\Auto_v1\.tmp_validation`
- `D:\3\Auto_v1\.tmp_tests`
- `D:\3\Auto_v1\tmp*`
- `D:\3\Auto_v1\dbtool.exe`
- `D:\3\Auto_v1\python_agents\**\__pycache__`

### Not candidates for archive right now

- `D:\3\Auto_v1` as a whole
  - Keep it for now because MRAG docs still use it as a historical reference project.
- `D:\3\MRAG\workspace`
  - Active runtime persistence area.

## Validation Gates After Each Cleanup Stage

- Go backend starts
- Vue frontend starts
- Python runtime unit tests pass
- `scripts/validate_stage3.sh` passes
- Key agent pages and APIs remain reachable

## Immediate Next Step

- Continue with source-tree and documentation entrypoint cleanup.
- Do not move candidate unnecessary files yet.

## Archive Progress

### Executed archive batch

The following reviewed items have already been moved into `D:\3\useless` with their original relative structure preserved:

- `MRAG`
  - `.stage2-frontend.log`
  - `.stage2-frontend.err.log`
  - `.stage2_validate_state`
  - `tmp_test_script.sh`
  - `backend/go/*.exe` validation or local build outputs
  - `backend/go/*.log`
  - `backend/go/*.out.log`
  - `backend/go/*.err.log`
  - `backend/python_agents/**/__pycache__`
- `Auto_v1`
  - `.tmp_validation`
  - `.tmp_tests`
  - `tmp*`
  - `dbtool.exe`
  - `python_agents/**/__pycache__`

### Retained on purpose

- `Auto_v1` root project
  - Not archived as a whole in this round.
  - It is not required for `MRAG` runtime execution, but `MRAG/docs` still uses it as a historical phase0 reference project.
- `MRAG/node_modules`
  - Intentionally not moved yet because doing so would immediately require reinstalling frontend dependencies before the next validation phase.
- `MRAG/dist`
  - Intentionally deferred to a later reviewed archive step.
