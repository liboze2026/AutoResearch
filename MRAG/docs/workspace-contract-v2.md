# Workspace Contract V2

## 1. Purpose

This document defines the MRAG stage 1 workspace contract for research asset management.

It adapts the phase0 workspace idea into MRAG without reintroducing the phase0 workflow system.

## 2. Principles

- MRAG remains the primary system.
- PostgreSQL remains the source of truth for object state.
- Workspace files are audit-friendly supporting artifacts.
- Existing MRAG dataset scanning and server capabilities are not replaced by workspace files.

## 3. Directory Layout

```text
workspace/
  papers/
    incoming/
    parsed/
    insights/
  ideas/
    pool/
  datasets/
  results/
  memory/
    agents/
```

## 4. Directory Semantics

### `workspace/papers`

- `incoming/`: raw paper imports
- `parsed/`: parsed paper intermediate outputs
- `insights/`: per-paper insight artifacts

### `workspace/ideas`

- `pool/`: idea snapshots and supplementary metadata

### `workspace/datasets`

- stores supporting files for research dataset assets
- may contain baseline summaries or attachments
- does not replace MRAG dataset metadata or scan records

### `workspace/results`

- stores supporting files for result archives
- may contain markdown summaries, manifests, metrics exports, and attachments
- does not imply automated experiment execution

### `workspace/memory`

- stores optional helper prompts, reusable notes, or light memory files for research helpers
- does not define a workflow engine

## 5. Mapping to MRAG Objects

- `Paper` and `PaperFile` map to `workspace/papers/*`
- `Insight` may produce supporting files under `workspace/papers/insights/`
- `Idea` may produce supporting files under `workspace/ideas/pool/`
- `DatasetAsset` may use `workspace/datasets/<dataset_asset_id>/`
- `Baseline` files may live under `workspace/datasets/<dataset_asset_id>/baselines/`
- `ResultArchive` may use `workspace/results/<result_archive_id>/`

## 6. Constraints

- `DatasetAsset` must reference an existing MRAG dataset.
- `ResultArchive` is an archive object, not an execution object.
- New stage 1 modules should reuse shared helpers instead of copying phase0 workflow packages wholesale.
