# 阶段3规格说明

## 1. 文档目标

本文定义 MRAG 阶段3“受控智能体期”的最小可执行规格。

阶段3的目标不是进入完全自治科研闭环，而是在阶段1科研资产模型与阶段2实验执行基础设施之上，建立一层可控、可校验、可修复、可持久化的 agent runtime 与 agent pipeline。

本文严格遵守以下边界：

- 必须先做 agent runtime / contract / adapter / schema repair / tool registry，再做具体 agent。
- 必须优先复用阶段2已有对象模型、实验生命周期、调度器、日志系统、结果归档系统。
- 必须优先复用 MRAG 现有 SSH / GPU / 数据集扫描等能力。
- 每个 agent 必须是受控节点，不是自由聊天机器人。
- 当前阶段不进入完全自治闭环。

## 2. 阶段3目标

阶段3目标如下：

- 建立统一的 agent runtime
- 定义统一的 agent 输入输出契约
- 支持 `api / codex_cli / mock` 三种 execution mode
- 建立 agent 的 validation 与 repair 链
- 建立 tools / skills / memory 的持久化与引用机制
- 建立由上游结构化输出驱动下游的受控流水线
- 在现有 `Experiment / ExperimentSpec / ExperimentRun / ResultArchive` 体系上承载 agent 工作结果
- 支持有限并行、有限重试、有限回退

## 3. 非目标

以下内容不属于阶段3：

- 完全自治科研闭环
- 长时自主循环 agent
- 自由聊天式多智能体系统
- 独立于 MRAG 的第二套 workflow/agent 平台
- 重做 SSH / GPU / dataset scan 基础设施
- 复杂插件市场、开放式工具生态
- 不受控的 agent 自行联网、自行改配置、自行提交实验

## 4. 阶段3复用原则

### 4.1 必须复用的现有能力

- `Experiment` / `ExperimentSpec` / `ExperimentRun`
- `SchedulerDecision` / `RunLog` / `ResultComparison`
- `ResultArchive`
- `ServerService` / `SSHGateway`
- `heartbeat` / `gpuresource`
- `DatasetService` / `dataset_remote_runtime`
- `PaperService` / `IdeaService` / `DatasetAssetService` / `BaselineService`
- `workspace/experiments` / `workspace/results` / `workspace/memory`

### 4.2 必须扩展而非重写的能力

- `ExperimentSpec.SpecJSON` 扩展 agent 字段
- `runner` 扩展为 agent node 执行器
- `RunLog` 扩展为 agent 审计日志承载体
- `ResultJSON` 扩展为 agent 输出与修复摘要承载体
- `workspace/memory` 扩展为 agent memory 存储根目录

## 5. 阶段3核心对象

阶段3最小对象集合至少包括：

- `AgentDefinition`
- `AgentExecution`
- `AgentInputContract`
- `AgentOutputContract`
- `AgentToolRef`
- `AgentSkillRef`
- `AgentMemoryRef`
- `AgentRepairRecord`
- `AgentPipelineNode`
- `AgentPipelineEdge`

其中，阶段3第一版应尽量映射到现有阶段2对象中：

- agent 规划落到 `ExperimentSpec`
- agent 执行落到 `ExperimentRun`
- agent 日志落到 `RunLog`
- agent 成果落到 `ResultArchive`
- agent 对比结论落到 `ResultComparison`

## 6. Agent 运行方式

### 6.1 受控节点定义

每个 agent 必须满足：

- 有明确输入 schema
- 有明确输出 schema
- 有明确 execution mode
- 有明确 prompt version
- 有明确 skill refs / tool refs
- 有明确日志记录
- 有明确验证步骤
- 失败时进入 repair 或 fail，而不是自由继续生成

### 6.2 Agent 调用入口

阶段3第一版统一通过 runtime 调用 agent，不允许业务模块直接自由拼接 prompt 调 agent。

统一入口至少应负责：

- 加载 agent 定义
- 装配结构化输入
- 选择 execution mode
- 调用 adapter
- 解析输出
- 校验输出
- 执行 repair
- 持久化结果
- 写审计日志

## 7. Agent 执行模式

每个 agent 必须支持以下执行模式字段：

- `execution_mode = api | codex_cli | mock`
- `model_provider`
- `model_name`
- `prompt_version`
- `skill_refs`
- `tool_refs`

### 7.1 `api`

用于后续真实模型 API 调用。

第一版要求：

- 保留统一接口
- 支持配置但不要求当前完成真实 key 注入
- 与 `codex_cli` / `mock` 共用同一套 contract

### 7.2 `codex_cli`

用于当前最小测试优先模式。

第一版要求：

- 作为主要联调与最小闭环验证方式
- 输出必须仍然走统一 schema 校验与 repair
- 不允许绕过 runtime 直接自由使用

### 7.3 `mock`

用于稳定演示、兜底、回退。

第一版要求：

- 每个关键 agent 都要有 mock 路径
- mock 输出也必须符合真实 output contract

## 8. Agent 输入输出契约

### 8.1 输入契约

每个 agent 输入至少包含：

- `agent_id`
- `agent_role`
- `execution_mode`
- `input_version`
- `trigger_source`
- `workspace_refs`
- `upstream_object_refs`
- `tool_refs`
- `skill_refs`
- `memory_refs`
- `payload`

输入原则：

- 尽量引用对象 ID，而不是传递大块自由文本
- 原始文本类输入必须有来源引用
- 上游输入必须可追溯到 paper / insight / dataset asset / baseline / experiment spec / run / archive

### 8.2 输出契约

每个 agent 输出至少包含：

- `status`
- `output_version`
- `agent_role`
- `summary`
- `structured_output`
- `artifact_refs`
- `tool_usage`
- `skill_usage`
- `memory_writes`
- `validation_report`
- `repair_report`

输出原则：

- 输出必须结构化
- 输出必须可校验
- 输出必须可持久化
- 输出必须可作为下游输入

### 8.3 校验原则

每次 agent 输出后必须至少经过：

1. JSON 结构校验
2. 必填字段校验
3. 类型校验
4. 语义完整性检查
5. 下游可消费性检查

## 9. Agent 生命周期

阶段3统一 agent 生命周期如下：

- `registered`
- `idle`
- `waiting_input`
- `ready`
- `running`
- `validating`
- `repairing`
- `succeeded`
- `failed`
- `paused`

### 9.1 生命周期含义

- `registered`：agent 已注册到系统，但尚未进入待执行状态
- `idle`：agent 可用，但当前没有待处理任务
- `waiting_input`：上游输入尚未满足，不能执行
- `ready`：输入已满足，可进入执行队列
- `running`：agent 正在执行
- `validating`：原始输出已产生，正在进行 schema 校验
- `repairing`：校验未通过，进入自动修复
- `succeeded`：输出通过校验并已持久化
- `failed`：执行失败或修复失败
- `paused`：人工或系统暂停，不继续推进

### 9.2 最小状态流转

至少应支持：

- `registered -> idle`
- `idle -> waiting_input`
- `waiting_input -> ready`
- `ready -> running`
- `running -> validating`
- `validating -> succeeded`
- `validating -> repairing`
- `repairing -> validating`
- `repairing -> failed`
- `running -> failed`
- `any active state -> paused`
- `paused -> ready`

## 10. Agent 角色范围

阶段3至少定义以下角色：

- `Reader Agent`
- `Insight Agent`
- `Dataset Agent`
- `Idea Generator Agent`
- `Planner Agent`
- `Coding Agent`
- `Evaluator Agent`
- `Writer Agent`
- `Picture Agent`

### 10.1 每个角色的最小职责

`Reader Agent`

- 消费 paper 输入
- 产出结构化阅读摘要与章节映射

`Insight Agent`

- 从 paper/reader 输出提炼结构化 insight
- 补充 methods / contributions / limitations

`Dataset Agent`

- 读取 MRAG 现有 dataset scan / dataset asset 信息
- 输出结构化数据集理解、约束、加载建议

`Idea Generator Agent`

- 基于 insight + dataset 产出候选 idea
- 输出结构化候选，而不是自由文本 brainstorm

`Planner Agent`

- 基于 idea + dataset asset + baseline + historical result 产出可执行 spec 草案
- 是阶段3第一版核心 agent 之一

`Coding Agent`

- 生成最小实现方案、代码变更建议、实验实现骨架
- 第一版可与 Evaluator 合并

`Evaluator Agent`

- 审查输出是否符合 spec、结果是否可比、是否需要 repair
- 第一版可与 Coding 合并

`Writer Agent`

- 负责结构化报告、总结、对外可读文档
- 第一版可与 Picture 合并

`Picture Agent`

- 负责图示、图表、插图型产出
- 第一版先 mock

## 11. 第一版合并策略

### 11.1 推荐合并

第一版推荐合并如下：

- `Coding Agent + Evaluator Agent`
- `Writer Agent + Picture Agent`

其中：

- `Coding + Evaluator` 先作为一个受控节点实现
- `Writer + Picture` 先作为一个受控节点实现，且 `Picture` 先走 mock

### 11.2 第一版必须先做最小版本的 agent

必须先做最小版本：

- `Reader Agent`
- `Insight Agent`
- `Dataset Agent`
- `Idea Generator Agent`
- `Planner Agent`
- `Coding+Evaluator Agent`
- `Writer+Picture Agent`

原因：

- 它们刚好覆盖“读入 -> 理解 -> 数据集约束 -> 产出 idea -> 形成计划 -> 实现/评估 -> 写结果”的最小链路

### 11.3 第一版可以延后加强的 agent

可以延后的是“单独拆开的高级版本”，而不是角色本身：

- 独立 `Evaluator Agent` 高级版
- 独立 `Picture Agent` 真实版
- 更强的 `Writer Agent` 多文体版

## 12. 并行流水线策略

### 12.1 总体原则

- 上游输出可触发下游
- 只能由通过校验的上游结构化输出触发下游
- 并行必须受限于角色级并发策略
- 并行不能破坏审计与可回放性

### 12.2 默认并发级别

相对高并发：

- `Reader`
- `Insight`
- `Dataset`
- `Idea Generator`

默认低并发：

- `Coding`
- `Writer`

说明：

- `Coding` 低并发是为了降低冲突、减少对共享代码或 spec 的覆盖风险
- `Writer` 低并发是为了减少重复报告和输出竞争

### 12.3 触发规则

可触发关系建议如下：

- `Reader -> Insight`
- `Insight + Dataset -> Idea Generator`
- `Idea Generator + Dataset Asset + Baseline -> Planner`
- `Planner -> Coding+Evaluator`
- `Coding+Evaluator -> Writer+Picture`

### 12.4 并行控制规则

必须至少支持：

- 节点级最大并发数
- 角色级默认并发数
- 同一对象上的互斥执行
- 重试次数上限
- 修复次数上限

## 13. 容错与修复策略

### 13.1 失败类型

阶段3至少区分以下失败：

- 调用失败
- 超时失败
- 输出非 JSON
- JSON 合法但 schema 不合法
- 字段缺失
- 语义不完整
- 工具调用失败
- 上游依赖失效

### 13.2 修复顺序

建议统一 repair 顺序：

1. 轻量解析修复
2. 字段补齐修复
3. 格式重建修复
4. 单次重试执行
5. 回退到 mock
6. 标记失败并暂停下游

### 13.3 Repair 边界

repair 只能做：

- JSON 修复
- 字段归一化
- schema 对齐
- 可追踪的最小补齐

repair 不能做：

- 擅自引入新的未授权工具
- 擅自改写上游事实
- 擅自跳过关键缺失输入

## 14. Tools / Skills / Memory 持久化策略

### 14.1 Tools

每个工具必须持久化记录：

- `tool_id`
- `name`
- `version`
- `owner`
- `entrypoint`
- `usage_contract`
- `input_schema`
- `output_schema`
- `test_status`
- `test_summary`

规则：

- agent 新增 Python 工具脚本前必须先注册
- 未注册工具不能被复用

### 14.2 Skills

每个 skill 必须持久化记录：

- `skill_id`
- `name`
- `version`
- `content_ref`
- `scope`
- `compatible_agents`
- `status`

规则：

- skill 必须可版本化
- skill 引用必须出现在 agent 配置中

### 14.3 Memory

每个 memory item 必须持久化记录：

- `memory_id`
- `agent_id`
- `memory_type`
- `scope`
- `content_ref`
- `source_run_id`
- `created_at`
- `expires_at`

规则：

- memory 默认是辅助性，不替代数据库主事实
- memory 必须可追踪来源
- memory 不允许偷偷篡改主对象

## 15. 阶段3最小闭环

阶段3第一版推荐形成如下最小受控闭环：

1. `Reader`
2. `Insight`
3. `Dataset`
4. `Idea Generator`
5. `Planner`
6. `Coding+Evaluator`
7. `Writer+Picture`

其中：

- `Planner` 输出进入 `ExperimentSpec`
- `Coding+Evaluator` 输出进入 `ExperimentRun`/`RunLog`
- `Writer+Picture` 输出进入 `ResultArchive`

## 16. 通过标准摘要

阶段3规格成立的前提是：

- agent 受控执行，而非自由对话
- 三种 execution mode 共享统一 contract
- 输出结构化、可校验、可修复、可持久化
- tools / skills / memory 有持久化机制
- 流水线可由上游输出驱动，但并发受控
- 当前仍不进入完全自治闭环
