# Stage3 Agent Implementation Summary

## 总览

阶段3已经完成的是一套“受控 agent runtime + 受控业务节点 + 受控前端管理页面”的最小实现。

它不是新起一套独立科研系统，而是建立在阶段1科研资产层和阶段2实验自动化层之上的扩展层。

## 1. 控制面实现

### Go 控制面

阶段3核心 Go 模块如下：

- `backend/go/internal/agentruntime`
  - 调用 Python runtime runner
  - 写入 input/output contract
- `backend/go/internal/agentjob`
  - 受控 job 创建、查询
- `backend/go/internal/agenttrigger`
  - job 触发、状态推进、artifact 持久化、post-process 调用
- `backend/go/internal/agentpipeline`
  - event、subscription、最小自动流水线 worker
- `backend/go/internal/agentadmin`
  - agent/job/artifact/event 管理接口
- `backend/go/internal/agentschema`
  - schema 管理接口
- `backend/go/internal/toolregistry`
  - tool 注册与持久化
- `backend/go/internal/skillregistry`
  - skill 注册与持久化
- `backend/go/internal/agentmemory`
  - memory 持久化

### Python runtime

阶段3核心 Python runtime 模块如下：

- `backend/python_agents/runtime/contract.py`
  - 统一输入输出 contract
- `backend/python_agents/runtime/base.py`
  - `BaseAgent / BaseExecutor / BaseValidator / BaseRepairer`
- `backend/python_agents/runtime/executors.py`
  - `api / codex_cli / mock` 三种执行适配
- `backend/python_agents/runtime/schema_registry.py`
  - schema 字段注册
- `backend/python_agents/runtime/normalizer.py`
  - 输出归一化
- `backend/python_agents/runtime/config.py`
  - API 与 Codex CLI 配置加载
- `backend/python_agents/runtime/runner.py`
  - runtime 命令行入口

## 2. Agent 实现状态

| Agent | Go 服务 | Python runtime | 当前状态 | 说明 |
| --- | --- | --- | --- | --- |
| Reader | `internal/readeragent` | `runtime/reader_agent.py` | 已完成最小版 | 导入/生成最小 paper 结果，支持 `codex_cli` fallback |
| Insight | `internal/insightagent` | `runtime/insight_agent.py` | 已完成最小版 | 从 paper/parsed 内容提取结构化 insight |
| Dataset | `internal/datasetagent` | `runtime/dataset_agent.py` | 已完成最小版 | 复用服务器/GPU/数据集扫描能力，生成 dataset asset 与 eval plan |
| Idea Generator | `internal/ideaagent` | `runtime/idea_agent.py` | 已完成最小版 | 消费 insight + dataset 生成结构化 idea |
| Planner | `internal/planneragent` | `runtime/planner_agent.py` | 已完成最小版 | 生成 plan 并落到 `Experiment` / `plan.json` |
| Coding + Evaluator | `internal/codingagent` | `runtime/coding_agent.py` | 已完成最小版 | 合并节点，仍限制在统一训练模板中 |
| Writer + Picture | `internal/writeragent` | `runtime/writer_agent.py` | 已完成最小版 | 合并节点，`Picture` 仍为 mock 路径 |

## 3. 前端交付状态

阶段3前端管理页已经落地，主要集中在 `src/views/agents`：

- `AgentListPage.vue`
- `AgentJobListPage.vue`
- `AgentJobDetailPage.vue`
- `ToolSkillManagerPage.vue`
- `AgentEventPage.vue`

对应路由：

- `/agents`
- `/agents/jobs`
- `/agents/jobs/:id`
- `/agents/catalog`
- `/agents/events`

阶段3自动验收已经实际验证这些关键页面可访问。

## 4. 与阶段1/阶段2对象的集成

阶段3没有重写阶段2，而是复用并扩展已有对象：

- `Reader / Insight`
  - 复用 `Paper` 与 `PaperInsight`
- `Dataset`
  - 复用 `Dataset`、`DatasetAsset`、`Baseline`
  - 复用 `Server / heartbeat / GPU snapshot / dataset scan`
- `Idea Generator`
  - 复用 `Idea`
- `Planner`
  - 落到 `Experiment` 与 `plan.json`
- `Coding + Evaluator`
  - 复用 `ExperimentSpec`、`ExperimentRun`、`RunLog`、`ResultComparison`、`ResultArchive`
- `Writer + Picture`
  - 复用 `ResultArchive`
  - 产出 draft 与 figure plan

## 5. 默认自动流水线

当前默认自动流水线由 Go 启动时注册：

- `paper_parsed -> insight`
- `insights_ready -> idea_generator`
- `idea_ready -> planner`
- `plan_ready -> coding`

说明：

- 这是一条最小可运行流水线
- 当前并未扩展到完全自治闭环
- `Writer` 仍以受控最小调用为主，不自动无限链式扩展

## 6. Tool / Skill / Memory 实现状态

阶段3当前已经有：

- tool registry
  - 可注册工具
  - 可记录 schema、用法、脚本内容
- skill registry
  - 可注册 skill 内容
  - 可持久化保存
- memory store
  - 可按 `agent_type + memory_key` 持久化
  - 可追踪来源

这意味着：

- 工具、技能、记忆都已经从“约定”升级为“受控持久化对象”
- 但还没有扩展到阶段4的复杂生态层

## 7. 自动验收与测试状态

阶段3当前已有两层测试：

### Python runtime 单测

覆盖：

- executor
- validator / repair
- Reader
- Insight
- Dataset
- Idea
- Planner
- Coding
- Writer

### 端到端自动验收

入口：

- `scripts/validate_stage3.sh`

覆盖：

- runtime runner
- `codex_cli` fallback
- schema validator / repair
- tool registry / skill registry / memory store
- 最小 agent 链路
- 前端 agent 页面可访问
- `PASS / FAIL` 与失败定位

## 8. 当前实现边界

阶段3必须明确的边界如下：

- 当前仍是受控智能体，不是完全自治科研系统
- 当前 `Writer / Picture` 仍是最小版本
- 当前 `Coding` 仍限制在统一训练模板中
- 当前真实模型与真实抓取能力只保留受控扩展点

## 9. 实现结论

如果按阶段3边界来评价，当前实现状态可以总结为：

- runtime 已成立
- controlled agent nodes 已成立
- tool / skill / memory 已成立
- pipeline 已成立
- 自动验收与端到端验证已成立

因此，阶段3已经达到“可交付收尾”的状态，可以作为阶段4的稳定起点。
