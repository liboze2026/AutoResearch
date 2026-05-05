# Multimodal RAG + VisDoM Real E2E Test Report

## Scope

- Date: 2026-03-26
- Workspace: `D:\3\MRAG`
- Backend API: `http://127.0.0.1:18082`
- Database: Docker PostgreSQL `mrag-postgres` / `mrag_platform`
- Real server kept after cleanup: `shenzhenvlab`
- Real dataset source confirmed on server: `/home/bzli/lbz/data/VisDoM-main`
- Registered dataset asset location used by the pipeline: `shenzhenvlab:/datasets/VisDoM`
- Agent execution mode under test: `codex_cli`
- Experiment theme: `multimodal RAG for visual document retrieval on VisDoM`

## Environment Confirmation

- Server retained in DB after cleanup:
  - `srv_1774366607_14235076 / shenzhenvlab / shenzhen-vlab:2266 / bzli`
- Real GPU probe succeeded:
  - `4 / 8` GPUs available
- Real dataset scan succeeded:
  - candidate `VisDoM-main`
  - path `/home/bzli/lbz/data/VisDoM-main`
  - size `1.9 GB`
- Local Codex CLI used:
  - `C:\Users\Zionjuer\.vscode\extensions\openai.chatgpt-26.323.20928-win32-x64\bin\windows-x86_64\codex.exe`
  - model `gpt-5.4-mini`
  - reasoning `low`

## End-to-End Chain

### 1. Reader

- Job: `ajob_1774459337_1c4583a3`
- Requested mode: `codex_cli`
- Used mode: `codex_cli`
- Imported paper:
  - `paper_1774459454_ae352c82`
  - `VisDoM: A Benchmark for Visual Document Retrieval`

### 2. Insight

- Job: `ajob_1774459466_bad415fd`
- Requested mode: `codex_cli`
- Used mode: `codex_cli`
- Generated insight:
  - `pinsight_1774459571_1afeeb00`

### 3. Dataset

- Job: `ajob_1774459603_fc8f3153`
- Requested mode: `codex_cli`
- Used mode: `codex_cli`
- Dataset asset:
  - `dasset_1774459658_85800333`
- Baseline:
  - `baseline_1774459658_44cdce0c`
- Eval plan:
  - [evalplan.json](D:/3/MRAG/workspace/datasets/dasset_1774459658_85800333/evalplan.json)

### 4. Idea

- Job: `ajob_1774459675_472e643c`
- Requested mode: `codex_cli`
- Used mode: `codex_cli`
- Idea:
  - `idea_1774459704_d3869629`
  - `Late-Fusion OCR + Vision Retriever for VisDoM`

### 5. Planner

- Job: `ajob_1774459731_602a358d`
- Requested mode: `codex_cli`
- Used mode: `codex_cli`
- Experiment:
  - `exp_1774459762_cf54f970`
- Plan:
  - [plan.json](D:/3/MRAG/workspace/experiments/exp_1774459762_cf54f970/plan.json)

### 6. Coding / Evaluator

- First blocking issue:
  - scheduler failed with `no available server for scheduling`
  - root cause: latest heartbeat in DB was stale `offline`
  - resolution during test: collected a fresh real heartbeat before scheduling
- Second blocking issue:
  - `coding` Codex schema rejected by structured output
  - root cause: `code_patch_manifest[].value` used an open object schema with `additionalProperties: true`
  - code fix applied: changed that field to compact string payloads for Codex schema
- Final successful coding job:
  - `ajob_1774460519_8e491831`
  - requested mode `codex_cli`
  - used mode `codex_cli`
- Final successful run:
  - `run_1774460637_13fca6dd`
  - assigned server `srv_1774366607_14235076`
  - remote workdir `/home/bzli/lbz/experiments/exp_1774459762_cf54f970/run_5`
  - status `succeeded`
- Result archive:
  - `archive_1774460673_b7f25eee`
- Metrics:
  - `recall@1 = 0.88`
  - `loss = 0.12`
  - `latency_ms = 37`

### 7. Writer

- First blocking issue:
  - runtime crashed with `UnicodeEncodeError: 'gbk' codec can't encode character '\\ufeff'`
  - root cause: Windows console output encoding + BOM in template
  - code fix applied:
    - runtime `runner.py` now forces UTF-8 stdout/stderr
    - writer template loader now reads `utf-8-sig`
- Final successful writer job:
  - `ajob_1774460922_c0048497`
  - requested mode `codex_cli`
  - used mode `codex_cli`
- Draft:
  - `draft_1774461044_e460d625`
  - [draft.md](D:/3/MRAG/workspace/writing/draft_1774461044_e460d625/draft.md)

## Code Fixes Applied During This Run

- [coding_agent.py](D:/3/MRAG/backend/python_agents/runtime/coding_agent.py)
  - changed Codex schema for `code_patch_manifest[].value` from open object to string
  - stringified patch manifest values
  - clarified coding prompt instructions for compact string values
- [runner.py](D:/3/MRAG/backend/python_agents/runtime/runner.py)
  - forced UTF-8 stdout/stderr for Windows runtime output
- [writer_agent.py](D:/3/MRAG/backend/python_agents/runtime/writer_agent.py)
  - switched template loading from `utf-8` to `utf-8-sig`
- [test_coding_agent.py](D:/3/MRAG/backend/python_agents/runtime/tests/test_coding_agent.py)
  - added coverage for coding Codex schema and stringified patch values

## Commands Used

```powershell
Invoke-RestMethod http://127.0.0.1:18082/healthz
Invoke-RestMethod -Method Post http://127.0.0.1:18082/api/v1/servers/srv_1774366607_14235076/gpu-snapshot
Invoke-RestMethod -Method Post http://127.0.0.1:18082/api/v1/servers/srv_1774366607_14235076/scan-datasets
Invoke-RestMethod -Method Post http://127.0.0.1:18082/api/v1/agents/reader/run
Invoke-RestMethod -Method Post http://127.0.0.1:18082/api/v1/agents/insight/run
Invoke-RestMethod -Method Post http://127.0.0.1:18082/api/v1/agents/dataset/run
Invoke-RestMethod -Method Post http://127.0.0.1:18082/api/v1/agents/idea-generator/run
Invoke-RestMethod -Method Post http://127.0.0.1:18082/api/v1/agents/planner/run
Invoke-RestMethod -Method Post http://127.0.0.1:18082/api/v1/servers/srv_1774366607_14235076/heartbeat
Invoke-RestMethod -Method Post http://127.0.0.1:18082/api/v1/agents/coding/run
Invoke-RestMethod -Method Post http://127.0.0.1:18082/api/v1/agents/writer/run
python -m unittest D:\3\MRAG\backend\python_agents\runtime\tests\test_coding_agent.py
python -m unittest D:\3\MRAG\backend\python_agents\runtime\tests\test_writer_agent.py
```

## Final Verdict

- PASS

The controlled stage3 chain ran end to end on the new `multimodal RAG + VisDoM` theme. All seven agents completed through the backend API, all generation nodes used real local `codex_cli`, the coding/evaluation stage successfully scheduled and executed a real run on `shenzhenvlab`, and the writer produced a final structured paper draft.

## Remaining Limitations

- The system is still a controlled agent system, not a fully autonomous research loop.
- Reader parsing is still based on the existing mock parser path for paper contents.
- Coding remains template-bound to the shared stage2 training template surface.
- Writer succeeded with real `codex_cli`, but picture generation remains mock-only in v1.
- Scheduling still depends on a fresh heartbeat being present for the selected server.
