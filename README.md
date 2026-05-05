<div align="center">

# 🔬 AutoResearch · MRAG

**面向科研场景的「受控多智能体平台」**

*Go 控制面 · Python 智能体运行时 · Vue 管理控制台*

[![阶段](https://img.shields.io/badge/阶段-3已交付%20·%204进行中-blue)]()
[![后端](https://img.shields.io/badge/后端-Go-00ADD8?logo=go)]()
[![运行时](https://img.shields.io/badge/运行时-Python-3776AB?logo=python)]()
[![前端](https://img.shields.io/badge/前端-Vue%203-4FC08D?logo=vuedotjs)]()
[![状态](https://img.shields.io/badge/状态-研究预览版-orange)]()

</div>

---

## ✨ 项目简介

**AutoResearch（代号 `MRAG`）** 不是一个"完全自治"的黑盒科学家系统，而是一个建立在真实科研资产管理与实验自动化基础设施之上的**受控智能体平台**——核心目标是让模型走出的每一步都**可观察、可重放、可校验、可持久化**。

业内常说的"agent"往往等同于"让 LLM 自由发挥"，本项目反其道而行：所有智能体都必须经过统一的 **contract → executor → schema 校验 → 自动修复 → 持久化** 链路。这正是项目的立项初衷。

> 🎯 **目标：让 AI 驱动的科研真正具备"可复现性"，而非仅仅"看起来很厉害"。**

---

## 🧠 七位受控智能体

完整覆盖 `论文 → 论文产出` 最小闭环：

| # | 智能体 | 职责 |
|---|---|---|
| 1 | 📖 **Reader（论文阅读）** | 摄取论文，抽取结构化知识 |
| 2 | 💡 **Insight（洞察提炼）** | 提炼洞见、研究空白、开放问题 |
| 3 | 📊 **Dataset（数据集）** | 数据集资产的扫描、索引、综述 |
| 4 | 🌟 **Idea Generator（点子生成）** | 基于洞见与数据集生成研究 idea |
| 5 | 🗺️ **Planner（实验规划）** | 把 idea 转化为可执行实验方案 |
| 6 | 💻 **Coding + Evaluator（编码与评测）** | 跑实验、跑 baseline、做评测 |
| 7 | ✍️ **Writer + Picture（写作与配图）** | 起草论文文本与配图 |

每个智能体共享同一套配置接口：
`execution_mode · model_provider · model_name · prompt_version · skill_refs · tool_refs`

---

## 🏗️ 整体架构

```
                ┌────────────────────────────────────────────┐
                │         Vue 3 管理控制台                     │
                │  /agents · /jobs · /catalog · /events       │
                └──────────────────┬─────────────────────────┘
                                   │ REST
                ┌──────────────────▼─────────────────────────┐
                │         Go 控制面（Gin）                     │
                │  agentruntime · agenttrigger · agentpipeline│
                │  实验生命周期 · 调度器 · 结果归档              │
                │  SSH 网关 · GPU 探测 · 数据集扫描              │
                └──────────────────┬─────────────────────────┘
                                   │ contract IPC
                ┌──────────────────▼─────────────────────────┐
                │         Python 智能体运行时                   │
                │  contract · executor · validator · repair   │
                │  Reader · Insight · Dataset · Idea ·        │
                │  Planner · Coding · Writer                  │
                └──────────────────┬─────────────────────────┘
                                   │
                ┌──────────────────▼─────────────────────────┐
                │  PostgreSQL · Workspace 文件系统 · GPU/SSH   │
                └────────────────────────────────────────────┘
```

---

## 🚀 核心亮点

- 🧩 **统一 Contract 层**——所有智能体输入输出都经过 schema 校验 + 自动修复后才落库。
- 🔁 **可重放流水线**——`论文 → 洞察 → 数据集 → idea → 计划 → 编码评测 → 写作` 全链路可追溯。
- 🧪 **真实实验基础设施**——直接复用阶段 2 的 `Experiment / ExperimentSpec / ExperimentRun / ResultArchive / ResultComparison`。
- 🛠️ **持久化注册表**——tool registry、skill registry、memory store、artifact / event / job / schema 全部入库。
- 🧷 **三种执行模式**：
  - `api`——真实模型 API 接入位（默认关闭，受控扩展点）
  - `codex_cli`——基于 Codex CLI 的最小联调路径
  - `mock`——稳定兜底模式，自动验收使用
- 🖥️ **一流的管理界面**——任务列表、详情、回放、事件流追踪。
- 🧬 **真实主机能力**——SSH 网关、GPU 快照、远程数据集扫描全部对真实节点可用。

---

## 📦 技术栈

| 层 | 技术 |
|---|---|
| 控制面 | Go · Gin · PostgreSQL |
| 智能体运行时 | Python（contract + validator + executors） |
| 前端 | Vue 3 · TypeScript · Vite · Element Plus |
| 基础设施 | Docker Compose · SSH · 可选 Codex CLI |

---

## ⚡ 快速开始

```bash
# 1. 启动后端（Docker）
docker compose up -d postgres go-backend

# 2. 启动前端
cd MRAG
npm install
npm run dev -- --host 127.0.0.1
```

或直接本地起 Go 后端：

```bash
cd MRAG/backend/go
go run ./cmd/server
```

### 阶段 3 自动验收

```powershell
Get-Content .\MRAG\scripts\validate_stage3.sh -Raw | Invoke-Expression
```

覆盖范围：runtime runner、codex_cli fallback、schema 校验/修复、tool/skill/memory 注册表、7 段最小智能体链路、前端 agent 页面可达性。

### 接入真实模型 API（可选）

真实执行**默认关闭**。如需启用，在 `.env` 或 `AGENT_RUNTIME_CONFIG_FILE` 中配置：

```env
AGENT_API_ENABLED=true
AGENT_API_ALLOW_LIVE_EXECUTION=true
AGENT_API_BASE_URL=https://your-model-endpoint.example.com/v1
AGENT_API_KEY=your-real-api-key
```

---

## 🛣️ 阶段路线图

| 阶段 | 主题 | 状态 |
|---|---|---|
| 0 | workspace 协议 · mock workflow · Python agent 脚本接口 | ✅ |
| 1 | 科研资产管理（论文、洞察、idea、数据集、baseline、归档） | ✅ |
| 2 | 实验生命周期 · 调度 · 心跳 · GPU 快照 · 日志收集 · 恢复 · 对比 | ✅ |
| 3 | 统一 agent runtime · 受控 pipeline · tool/skill/memory 持久化 · agent 管理页面 · 自动验收 | ✅ |
| 4 | 真实主机 live execution · 更完整的 Picture 智能体 · 开放式编码闭环 | 🚧 |

---

## 📂 仓库结构

```
.
├── MRAG/                           # 主项目
│   ├── backend/
│   │   ├── go/                     # Go 控制面
│   │   ├── python_agents/          # 智能体运行时
│   │   ├── python_runners/         # 实验运行器
│   │   └── python_templates/
│   ├── src/                        # Vue 3 前端
│   ├── scripts/                    # 各阶段验收脚本
│   ├── docs/                       # 阶段规格、架构、验收报告
│   ├── workspace/                  # 运行时工作区（模板、技能、工具）
│   └── docker-compose.yml
└── README.md                       # 当前文档
```

---

## 📜 文档入口

- [阶段 3 规格说明](./MRAG/docs/stage3-spec.md)
- [阶段 3 架构说明](./MRAG/docs/stage3-agent-architecture.md)
- [阶段 3 实现总结](./MRAG/docs/stage3-agent-implementation-summary.md)
- [阶段 3 演示说明](./MRAG/docs/stage3-demo.md)
- [阶段 3 验收报告](./MRAG/docs/stage3-validation-report.md)
- [阶段 3 已知限制](./MRAG/docs/stage3-known-limitations.md)

---

## 🧭 设计边界

以下是项目**有意为之**的约束，并非未完成项：

- 当前是**受控智能体系统**，不是完全自治的科学家。
- `Writer / Picture` 仍是最小版本，`Picture` 暂时并入 `Writer`。
- `Coding` 限制在统一训练模板内，尚未放开自由代码生成式训练闭环。
- 真实模型接入与真实抓取保留为**受控扩展点**，默认不开启。

---

## 🤝 参与贡献

欢迎提 PR / Issue。提交前请先跑通 `validate_stage3.sh`。

---

> ⚠️ **声明：项目还需进一步测试与功能性验证。**
