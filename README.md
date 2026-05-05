<div align="center">

# 🔬 AutoResearch · MRAG

**A Controlled Multi-Agent Platform for AI-Assisted Scientific Research**

*Go control plane · Python agent runtime · Vue management console*

[![Stage](https://img.shields.io/badge/stage-3%20delivered%20·%204%20in%20progress-blue)]()
[![Backend](https://img.shields.io/badge/backend-Go-00ADD8?logo=go)]()
[![Runtime](https://img.shields.io/badge/runtime-Python-3776AB?logo=python)]()
[![Frontend](https://img.shields.io/badge/frontend-Vue%203-4FC08D?logo=vuedotjs)]()
[![License](https://img.shields.io/badge/status-research%20preview-orange)]()

</div>

---

## ✨ What is this?

**AutoResearch (codename `MRAG`)** is **not** a black-box "fully autonomous scientist". It is a **controlled agent platform** built on top of a real research-asset management and experiment-automation infrastructure — designed so that every step a model takes is **observable, replayable, validated, and persisted**.

Where "agent" usually means *"an LLM that does whatever it wants"*, here every agent runs through a unified **contract → executor → schema-validate → repair → persist** pipeline. That's the whole point.

> 🎯 **Goal:** make AI-driven research *reproducible*, not just *impressive*.

---

## 🧠 The 7 Controlled Agents

A complete `paper → publication` minimum loop, end-to-end:

| # | Agent | Role |
|---|---|---|
| 1 | 📖 **Reader** | Ingest papers, extract structured knowledge |
| 2 | 💡 **Insight** | Distill insights, gaps, open problems |
| 3 | 📊 **Dataset** | Survey / scan / index dataset assets |
| 4 | 🌟 **Idea Generator** | Propose research ideas grounded in insights + datasets |
| 5 | 🗺️ **Planner** | Turn ideas into executable experiment plans |
| 6 | 💻 **Coding + Evaluator** | Run the experiment, evaluate baselines |
| 7 | ✍️ **Writer + Picture** | Draft the writeup with figures |

Each agent shares the **same configuration surface**:
`execution_mode · model_provider · model_name · prompt_version · skill_refs · tool_refs`.

---

## 🏗️ Architecture at a Glance

```
                ┌────────────────────────────────────────────┐
                │            Vue 3 Management Console        │
                │  /agents · /jobs · /catalog · /events       │
                └──────────────────┬─────────────────────────┘
                                   │ REST
                ┌──────────────────▼─────────────────────────┐
                │         Go Control Plane (Gin)              │
                │  agentruntime · agenttrigger · agentpipeline│
                │  experiment lifecycle · scheduler · archive │
                │  SSH gateway · GPU probe · dataset scan     │
                └──────────────────┬─────────────────────────┘
                                   │ contract IPC
                ┌──────────────────▼─────────────────────────┐
                │         Python Agent Runtime                │
                │  contract · executor · validator · repair   │
                │  Reader · Insight · Dataset · Idea ·        │
                │  Planner · Coding · Writer                  │
                └──────────────────┬─────────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────────┐
                │  PostgreSQL · Workspace Filesystem · GPU/SSH │
                └────────────────────────────────────────────┘
```

---

## 🚀 Highlights

- 🧩 **Unified contract layer** — every agent input / output passes a schema validator + auto-repair before persistence.
- 🔁 **Replayable pipelines** — `paper → insight → dataset → idea → plan → coding/eval → writer` runs end-to-end and is fully traceable.
- 🧪 **Real experiment infrastructure** — reuses Stage 2's `Experiment / ExperimentSpec / ExperimentRun / ResultArchive / ResultComparison`.
- 🛠️ **Persistent registries** — tool registry, skill registry, memory store, artifact / event / job / schema stores.
- 🧷 **Three execution modes**:
  - `api` — real model API hook (off by default; controlled extension point)
  - `codex_cli` — Codex CLI for minimum integration testing
  - `mock` — stable fallback used by automated acceptance
- 🖥️ **First-class management UI** — list, drill, replay every job; live event tracing.
- 🧬 **Real-host capability** — SSH gateway, GPU snapshot, dataset scan against real remote nodes.

---

## 📦 Tech Stack

| Layer | Tech |
|---|---|
| Control plane | Go · Gin · PostgreSQL |
| Agent runtime | Python (contract + validator + executors) |
| Frontend | Vue 3 · TypeScript · Vite · Element Plus |
| Infra | Docker Compose · SSH · optional Codex CLI |

---

## ⚡ Quickstart

```bash
# 1. Backend (Docker)
docker compose up -d postgres go-backend

# 2. Frontend
cd MRAG
npm install
npm run dev -- --host 127.0.0.1
```

Or run the Go backend directly:

```bash
cd MRAG/backend/go
go run ./cmd/server
```

### Stage 3 automated acceptance

```powershell
Get-Content .\MRAG\scripts\validate_stage3.sh -Raw | Invoke-Expression
```

This covers: runtime runner, codex_cli fallback, schema validation/repair, tool/skill/memory registries, the 7-agent minimum chain, and frontend reachability.

### Wiring real model APIs (optional)

Live execution is **off by default**. To enable, set in `.env` (or via `AGENT_RUNTIME_CONFIG_FILE`):

```env
AGENT_API_ENABLED=true
AGENT_API_ALLOW_LIVE_EXECUTION=true
AGENT_API_BASE_URL=https://your-model-endpoint.example.com/v1
AGENT_API_KEY=your-real-api-key
```

---

## 🛣️ Stage Roadmap

| Stage | Theme | Status |
|---|---|---|
| 0 | Workspace protocol · mock workflow · Python agent script interface | ✅ |
| 1 | Research-asset management (papers, insights, ideas, datasets, baselines, archives) | ✅ |
| 2 | Experiment lifecycle · scheduler · heartbeat · GPU snapshot · log collector · recovery · comparison | ✅ |
| 3 | Unified agent runtime · controlled pipeline · tool/skill/memory persistence · agent management UI · automated acceptance | ✅ |
| 4 | Real-host live execution · richer Picture agent · open-ended coding loop | 🚧 |

---

## 📂 Repository Layout

```
.
├── MRAG/                           # Main project
│   ├── backend/
│   │   ├── go/                     # Go control plane
│   │   ├── python_agents/          # Agent runtime
│   │   ├── python_runners/         # Experiment runners
│   │   └── python_templates/
│   ├── src/                        # Vue 3 frontend
│   ├── scripts/                    # Stage validation scripts
│   ├── docs/                       # Stage specs, architecture, validation reports
│   ├── workspace/                  # Runtime workspace (templates, skills, tools)
│   └── docker-compose.yml
└── README.md                       # This file
```

---

## 📜 Documentation

- [Stage 3 Specification](./MRAG/docs/stage3-spec.md)
- [Stage 3 Agent Architecture](./MRAG/docs/stage3-agent-architecture.md)
- [Stage 3 Implementation Summary](./MRAG/docs/stage3-agent-implementation-summary.md)
- [Stage 3 Demo Notes](./MRAG/docs/stage3-demo.md)
- [Stage 3 Validation Report](./MRAG/docs/stage3-validation-report.md)
- [Stage 3 Known Limitations](./MRAG/docs/stage3-known-limitations.md)

---

## 🧭 Design Boundaries

These are **deliberate constraints**, not gaps:

- This is a **controlled agent system**, not a fully autonomous scientist.
- `Writer / Picture` is a minimal version; `Picture` currently rides inside `Writer`.
- `Coding` is restricted to a unified training template — no free-form codegen training loop yet.
- Real model APIs and real scraping are kept as **controlled extension points**, off by default.

---

## 🤝 Contributing

PRs and issues welcome. Please run `validate_stage3.sh` before submitting.

---

> ⚠️ **声明：项目还需进一步测试与功能性验证。**
> *This project still requires further testing and functional validation.*
