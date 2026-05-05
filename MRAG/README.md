# MRAG

MRAG 当前已完成阶段0、阶段1、阶段2、阶段3的受控实现。马上开始第四阶段。

当前系统不是完全自治科研系统，而是一个建立在现有科研资产管理与实验自动化基础设施之上的“受控智能体”平台：

- 阶段0：workspace 协议、mock workflow、mock Python agent 脚本接口
- 阶段1：论文、insight、idea、dataset asset、baseline、result archive 等科研资产管理
- 阶段2：experiment、spec、run、scheduler、heartbeat、GPU snapshot、log collector、recovery、comparison
- 阶段3：统一 agent runtime、受控 pipeline、tool/skill/memory 持久化、前后端 agent 管理页面、自动验收与端到端验证

## 阶段3能力说明

阶段3已经交付的核心能力如下：

- 统一的 Python agent runtime，包含 input/output contract、schema registry、validator、repair、runner
- 统一的 Go 控制面，复用阶段2实验生命周期、调度器、日志系统、结果归档系统
- 受控 agent 角色：
  - `Reader Agent`
  - `Insight Agent`
  - `Dataset Agent`
  - `Idea Generator Agent`
  - `Planner Agent`
  - `Coding + Evaluator Agent`
  - `Writer + Picture Agent`
- 统一配置字段：
  - `execution_mode`
  - `model_provider`
  - `model_name`
  - `prompt_version`
  - `skill_refs`
  - `tool_refs`
- 持久化能力：
  - tool registry
  - skill registry
  - memory store
  - agent job / artifact / event / schema
- 前端 agent 页面：
  - `/agents`
  - `/agents/jobs`
  - `/agents/jobs/:id`
  - `/agents/catalog`
  - `/agents/events`
- 端到端最小链路：
  - `paper -> insight -> dataset -> idea -> plan -> coding/eval -> writer`

## Agent 架构说明

阶段3采用“Go 控制面 + Python runtime + Vue 管理界面”的受控架构。

控制面主要职责：

- Go 后端负责：
  - agent job 创建与触发
  - post-process 持久化
  - pipeline 事件订阅与自动触发
  - 复用阶段2的 `Experiment / ExperimentSpec / ExperimentRun / ResultArchive / ResultComparison`
  - 复用 MRAG 现有 SSH、GPU 探测、数据集扫描能力
- Python runtime 负责：
  - contract 解析
  - executor 选择
  - schema 校验
  - repair
  - 统一结构化输出
- Vue 前端负责：
  - agent 列表、任务列表、任务详情
  - tool/skill catalog
  - event 追踪

关键实现目录：

- Go 控制面：`backend/go/internal/agentruntime`、`backend/go/internal/agenttrigger`、`backend/go/internal/agentpipeline`
- Agent 服务：`backend/go/internal/readeragent`、`insightagent`、`datasetagent`、`ideaagent`、`planneragent`、`codingagent`、`writeragent`
- Python runtime：`backend/python_agents/runtime`
- 前端页面：`src/views/agents`

## 双执行模式说明

阶段3对真实模型调用层面保留两种正式执行模式，同时提供一个稳定的测试/兜底模式：

- `api`
  - 预留真实模型 API 接入位
  - 当前默认不启用 live execution
  - 必须仍然经过统一 contract、validator、repair
- `codex_cli`
  - 当前最小联调与验收优先模式
  - 适合在没有真实 API key 的情况下先完成受控链路测试
  - 如果 CLI 不可用，会按 runtime 规则回退
- `mock`
  - 稳定兜底模式
  - 用于最小测试、自动验收、失败回退
  - 输出仍然必须结构化、可校验、可持久化

对外可以理解为“API / Codex CLI 双执行模式”，`mock` 是验收和回退模式，不是额外的产品能力扩张。

## 如何配置真实 API

真实 API 当前只保留受控扩展点，不默认开启 live execution。

可以通过仓库根目录 `.env`、环境变量或 runtime config file 进行配置。核心变量如下：

```env
AGENT_API_ENABLED=true
AGENT_API_ALLOW_LIVE_EXECUTION=true
AGENT_API_BASE_URL=https://your-model-endpoint.example.com/v1
AGENT_API_KEY=your-real-api-key
AGENT_API_TIMEOUT_SECONDS=60

CODEX_CLI_BIN=codex
CODEX_CLI_TIMEOUT_SECONDS=60
CODEX_CLI_OUTPUT_MODE=stdout
```

也可以通过 `AGENT_RUNTIME_CONFIG_FILE` 指向一个 JSON 配置文件，例如：

```json
{
  "api": {
    "enabled": true,
    "allow_live_execution": true,
    "base_url": "https://your-model-endpoint.example.com/v1",
    "api_key": "your-real-api-key",
    "timeout_seconds": 60
  },
  "codex_cli": {
    "command": "codex",
    "args": [],
    "timeout_seconds": 60,
    "output_mode": "stdout"
  }
}
```

注意：

- 阶段3不要求默认接通真实 key
- 真实 API 调用必须显式开启 `AGENT_API_ALLOW_LIVE_EXECUTION=true`
- 当前真实模型能力仍属于“受控扩展点”，不是默认生产闭环

## 如何使用 `codex_cli` 进行最小测试

推荐方式是直接运行阶段3自动验收脚本。它会优先验证 `codex_cli` 路径，并在 CLI 不可用时自动验证 fallback 到 `mock`：

```powershell
Get-Content .\scripts\validate_stage3.sh -Raw | Invoke-Expression
```

如果本机已经安装 Codex CLI，可以显式指定：

```powershell
$env:CODEX_CLI_BIN="codex"
Get-Content .\scripts\validate_stage3.sh -Raw | Invoke-Expression
```

最小测试会覆盖：

- agent runtime 启动
- `mock / codex_cli fallback`
- schema validator / repair
- tool registry / skill registry / memory store
- Reader / Insight / Dataset / Idea / Planner / Coding-Evaluator / Writer
- 前端 agent 页面可访问

## 本地启动

### 后端

推荐直接使用 Docker Compose：

```bash
docker compose up -d postgres go-backend
```

本地直接启动 Go 后端：

```bash
cd backend/go
go run ./cmd/server
```

### 前端

```bash
npm install
npm run dev -- --host 127.0.0.1
```

## 阶段3自动验收

阶段3自动验收入口：

```powershell
Get-Content .\scripts\validate_stage3.sh -Raw | Invoke-Expression
```

当前验收脚本已经覆盖：

- runtime runner
- `codex_cli` fallback
- schema validation / repair
- tool / skill / memory
- 7 段最小 agent 链路
- agent 页面可访问
- `shenzhenvlab` 可用性探测与 mock 自动回退

## 当前阶段3边界

以下边界是当前设计的明确约束，不应误判为阶段3未完成：

- 当前仍是受控智能体系统，不是完全自治科研系统
- 当前 `Writer / Picture` 仍是最小版本，`Picture` 仍并入 `Writer` 的 mock 能力
- 当前 `Coding` 仍限制在统一训练模板中，不放开自由代码生成式训练闭环
- 当前真实模型接入与真实抓取能力只保留受控扩展点，不默认打开

## 文档入口

- [仓库维护计划](./docs/repo-maintenance-plan.md)
- [工作区总览](../README.md)
- [阶段3规格](./docs/stage3-spec.md)
- [阶段3架构说明](./docs/stage3-agent-architecture.md)
- [阶段3实现总结](./docs/stage3-agent-implementation-summary.md)
- [阶段3演示说明](./docs/stage3-demo.md)
- [阶段3验收报告](./docs/stage3-validation-report.md)
- [阶段3已知限制](./docs/stage3-known-limitations.md)
- [阶段3验收模板](./docs/stage3-validation-report-template.md)
