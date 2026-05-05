# 阶段3能力审计

## 1. 审计目标

本文用于在不编写阶段3业务代码的前提下，审计当前 `MRAG` 与历史 `Auto_v1` 代码库中哪些能力已经存在、哪些能力可以直接复用、哪些能力需要在原模块上扩展、哪些能力必须在阶段3新建。

本次审计严格遵守以下边界：

- 先复用阶段2已有对象模型、实验生命周期、调度、日志、结果归档，再谈 agent。
- 先复用 MRAG 已有 SSH / GPU / 数据集扫描能力，不重造底层远程执行链。
- 阶段3 agent 必须是“受控节点”，不是自由聊天机器人。
- 当前只做能力审计与清洗方案，不进入阶段3业务实现。

## 2. 结论摘要

当前系统已经具备阶段3非常关键的“执行基础设施”，但还没有“受控 agent runtime”本身。

已经具备并可直接复用的底座主要有：

- 阶段1科研资产对象：`Paper`、`PaperInsight`、`Idea`、`DatasetAsset`、`Baseline`、`ResultArchive`
- 阶段2实验对象：`Experiment`、`ExperimentSpec`、`ExperimentRun`、`RunLog`、`SchedulerDecision`、`ResultComparison`
- 服务器与远程执行底座：`ServerService`、`SSHGateway`、GPU 探测、服务器 heartbeat、GPU snapshot
- 训练运行底座：`scheduler`、`runner`、`recovery`、`logcollector`、`resultcompare`
- workspace 与结果落盘：`workspace/experiments`、`workspace/results`、`workspace/memory`
- Python CLI 脚本接入经验：阶段1 `backend/python_agents/*` 与阶段0 `Auto_v1/python_agents/*`

当前缺的不是实验系统，而是 agent 层公共能力：

- 统一 agent runtime / contract
- execution mode 适配器：`api / codex_cli / mock`
- 统一 Python agent 调用层
- tool registry / skill registry / memory registry
- schema validation + repair 链
- agent 输出持久化协议
- agent 并行控制、重试控制、容错修复策略

结论：阶段3应建立在现有阶段2执行链之上，新增“受控 agent runtime 层”，而不是另起一套 agent workflow 系统。

## 3. 本次重点检查范围

- 阶段2：`experiment_service`、`scheduler`、`runner`、`logcollector`、`recovery`、`resultcompare`
- 阶段1：`paper`、`idea`、`dataset asset`、`baseline`、`result archive`
- MRAG 现有：`SSHGateway`、`ServerService.CheckGPU()`、远程数据集扫描与索引
- 阶段0/1 mock：`Auto_v1/internal/workflow/phase0.go`、`Auto_v1/python_agents/*`、`MRAG/backend/python_agents/*`
- workspace：`docs/workspace-contract-v2.md`、`workspace/memory/*`

## 4. 已有可复用能力

### 4.1 阶段2实验生命周期已成型，可直接复用

已有能力：

- `Experiment` / `ExperimentSpec` / `ExperimentRun` 数据模型已落库
- `ExperimentService.GenerateSpec()` 已能从 `dataset asset / idea / baseline / historical result archive` 生成结构化 spec
- `scheduler.Service` 已支持 queue、candidate 计算、server 选择、decision 持久化
- `runner.Service` 已支持本地准备 run dir、上传远程目录、SSH 启动、采集输出、写回 run 状态
- `recovery.Service` 已支持失败原因聚合与 retry
- `logcollector.Service` 已支持 run log 聚合与 tail
- `resultcompare.Service` 已支持与 baseline / historical result archive 的结构化对比，并可自动生成新的 result archive

阶段3复用方式：

- agent 生成的规划结果应优先落到 `ExperimentSpec.SpecJSON`，而不是新造平行 spec 表
- agent 执行结果应尽量复用 `ExperimentRun.ResultJSON`、`RunLog`、`ResultArchive`
- agent pipeline 的状态推进应借用阶段2现有状态机思路，而不是重新发明 workflow 事件流

### 4.2 后台 worker / background job 机制已有“最小够用”基础

已有能力：

- `heartbeat.Monitor` 已支持定时循环采集 heartbeat 与 GPU snapshot
- dataset index 已经有 `StartIndex / SyncIndex` 这类异步任务接口与任务状态记录
- `scheduler -> runner -> compare -> archive` 构成了最小后台执行链

审计判断：

- 当前系统已经有“后台周期任务”和“异步任务状态记录”的基础
- 但还没有通用的 agent worker 池、并发配额、租约、任务抢占、全局队列

阶段3结论：

- 阶段3第一步不必先建设重型 worker 平台
- 可以先复用现有 run/scheduler 背景执行方式，再补 agent task queue 与并发控制

### 4.3 远程执行、SSH、GPU 探测能力成熟，可直接复用

已有能力：

- `service.SSHGateway` 支持 real/mock 两种模式
- `SystemSSHGateway` 支持 system ssh、password、`.ssh/config`、`ProxyCommand`
- `ServerService.TestConnection()` / `RefreshStatus()` / `CheckGPU()` 已可直接调用
- `gpuresource.Service` 已把 GPU probe 结果持久化为 `GPUResourceSnapshot`
- `scheduler.Service` 已把 heartbeat 与 GPU snapshot 作为调度输入

阶段3复用方式：

- agent runtime 不应自己处理 SSH、GPU 探测
- agent 只应通过已注册的 Go 侧服务或注册工具调用这些能力
- “如果 `shenzhenvlab` 空闲则优先真实执行，否则回退 mock server”应建立在现有 heartbeat + GPU snapshot + scheduler 信息之上

### 4.4 数据集扫描与索引链可作为 agent 工具输入

已有能力：

- `DatasetService` 支持路径校验、导入、扫描记录、预览、索引任务
- `dataset_remote_runtime.go` 已支持 SSH 远程 scan/index
- `dataset_local_runtime.go` 已支持本地 scan 与 mock scan
- 数据集模型中已有 `Dataset`、`DatasetScanRecord`、`DatasetPreviewItem`、`DatasetIndexTask`

阶段3复用方式：

- agent 不应自己直接遍历远程目录
- 需要目录、样本、模态、索引状态时，应直接使用 MRAG 已有 dataset scan/index 结果

### 4.5 阶段1科研资产模型已足够支撑受控 agent 的上游/下游输入

已有能力：

- `PaperService` 已支持论文导入、解析、insight 抽取
- `IdeaService` 已支持 idea 持久化与来源追踪
- `DatasetAssetService` 已支持从 MRAG 数据集扫描结果登记 asset
- `BaselineService` 已支持结构化指标 schema 与结果
- `ResultArchiveService` 已支持指标、摘要、文件落盘与归档

阶段3复用方式：

- paper parser / insight extractor / planner agent 的输入输出应优先落到这些现有对象
- agent 之间的自动流水线应尽量以这些结构化对象 ID 驱动，而不是直接传大片自由文本

### 4.6 experiment spec 与 result comparison 对 agent 非常有复用价值

审计结果：

- 已有 `ExperimentSpec.SpecJSON`，且 `GenerateSpec()` 已把 dataset、baseline、comparison target、template type 等信息装配进去
- 已有 `ResultComparison` 与 workspace 比较输出
- `runner.Service` 成功后会自动尝试 compare，并可自动归档 result archive

阶段3意义：

- planner 类 agent 可以先复用现有 spec 结构扩展 agent 专用字段
- reviewer / comparator 类 agent 可直接消费 `ResultComparison` 与 `ResultArchive`
- 当前系统已经有“结构化科研结果对比”的雏形，不需要从零开始

### 4.7 Python CLI 脚本合同已有先例，但尚未统一

已有能力：

- `MRAG/backend/python_agents/README.md` 明确要求：CLI 参数输入、成功时单个 JSON 输出、失败时非零退出
- `PaperService` 已用 `exec.CommandContext(...)` 直接调用 Python 脚本
- `Auto_v1/python_agents/*` 已验证了一套 deterministic CLI + workspace 落盘模式

限制：

- 当前每个脚本是“独立直连”，没有统一调用器
- 没有 agent 级 schema 验证、repair、重试、超时、日志标准化
- 没有 execution mode 适配器抽象

结论：

- 阶段3可以复用“CLI 输入 + JSON 输出”的脚本合同
- 但必须新建统一 Python agent runtime / adapter 层

### 4.8 workspace memory 与 scripts 目录可复用，但目前只是预留位

已有目录：

- `workspace/memory/`
- `workspace/memory/agents/`
- `backend/scripts/`
- `backend/python_agents/`
- `backend/python_templates/`
- `backend/python_runners/`

审计判断：

- 这些目录为阶段3提供了天然落点
- 但目前没有真正的 skill registry、tool registry、memory manifest、script registration 机制

建议：

- 阶段3直接在现有目录上扩展，不另建新的顶层目录体系

## 5. 待扩展模块

以下模块建议在原模块上扩展，而不是重写：

### 5.1 `ExperimentSpec` 结构

需要扩展的方向：

- agent pipeline 上游输入引用
- agent 专用输出 schema 引用
- `execution_mode`
- `model_provider`
- `model_name`
- `prompt_version`
- `skill_refs`
- `tool_refs`

原因：

- `ExperimentSpec` 已经是阶段2可校验、可持久化、可落盘的结构化执行入口
- 阶段3 planner agent 最自然的落点就是补全此 spec，而不是新建平行 spec

### 5.2 `runner` 与 `scheduler`

需要扩展的方向：

- agent node 级别的执行单元
- 并发控制
- retry policy
- 对 Codex CLI / API / mock 的执行适配
- 对 agent 输出 schema 校验失败时的 repair 分支

原因：

- 当前 runner 适合“实验训练脚本”
- 阶段3需要在其上扩展为“受控 agent 任务执行器”

### 5.3 `RunLog` / `ResultJSON` / `ResultArchive`

需要扩展的方向：

- agent 请求与响应摘要
- schema 校验报告
- repair 过程记录
- tool 调用记录
- skill 使用记录
- memory 读写摘要

原因：

- 这些现有对象已经是审计与持久化的自然位置

### 5.4 `workspace/memory`

需要扩展的方向：

- 每个 agent 的 memory manifest
- memory item 分类
- 最近上下文快照
- memory 读写规则

原因：

- 目录已存在，但目前只有 README，占位性质明显

## 6. 待新建模块

以下是阶段3必须新建、且当前代码中基本不存在的模块：

### 6.1 Agent Runtime

必须新建内容：

- 统一 agent 定义
- 受控执行入口
- 标准输入上下文
- 标准输出合同
- 超时 / 重试 / fallback / repair 钩子

这是阶段3第一优先级模块。

### 6.2 Agent Contract / Schema Registry

必须新建内容：

- 每类 agent 的 input schema / output schema
- schema version
- validation 与 repair hook
- 结构化错误分类

原因：

- 当前 Python 脚本只有“打印 JSON”约束，没有正式 contract registry

### 6.3 Execution Adapter

必须新建内容：

- `api` adapter
- `codex_cli` adapter
- `mock` adapter

要求：

- 三种模式共享统一 agent contract
- 当前最小测试优先走 `codex_cli`

### 6.4 Python Agent Invoker

必须新建内容：

- 统一 Python CLI 调用器
- 参数组装
- stdout/stderr 捕获
- JSON 提取
- 超时控制
- 重试与日志标准化

原因：

- 当前 `PaperService` 直调 Python，不适合扩展成多 agent runtime

### 6.5 Tool Registry

必须新建内容：

- 工具注册表
- 工具元数据
- 调用方式说明
- 输入输出 schema
- 测试结果记录
- 是否允许 agent 直接使用的受控标记

原因：

- 当前没有任何正式 tool registry
- 阶段3要求 agent 新增 Python 工具脚本前必须先注册

### 6.6 Skill Registry

必须新建内容：

- skill 定义
- skill 内容存储
- skill 版本
- skill 引用关系
- skill 与 agent 的绑定

原因：

- 当前只有 `workspace/memory/agents` 占位目录，没有技能系统

### 6.7 Memory Registry

必须新建内容：

- memory item 表
- memory 文件索引
- 读写策略
- 生命周期与清理策略

### 6.8 Schema Repair / Output Repair

必须新建内容：

- agent 输出解析失败后的 repair 流程
- JSON 修复、字段补齐、格式纠偏
- repair 结果审计记录

原因：

- 当前没有任何统一的 agent 输出修复链

### 6.9 Agent Node / Pipeline Definition

必须新建内容：

- 受控 agent node 定义
- node 输入来源
- 上游输出依赖
- 并发与串行编排
- 节点级 retry / fallback

原因：

- 当前只有实验执行链，没有 agent 流水线定义层

## 7. 特别检查结论

### 7.1 是否已有足够的 worker / background job 机制

结论：

- 有“最小够用”的基础，但还不够直接承载完整 agent 平台。

已有：

- heartbeat/gpu snapshot 定时循环
- dataset index 异步任务
- scheduler/run/recovery 任务状态流

缺少：

- 通用 agent 队列
- 并发配额
- worker lease
- 全局 retry/backoff
- 节点级熔断

### 7.2 是否已有统一 Python 脚本调用层

结论：

- 没有。

现状：

- `PaperService` 直接 `exec.CommandContext` 调 Python
- 阶段0 `Auto_v1` 也是 workflow 内直接起 Python 子进程
- 没有统一 invoker、统一日志格式、统一 JSON 解析与 repair

### 7.3 是否已有工具脚本目录或技能目录可复用

结论：

- 有目录落点，但没有注册机制。

可复用目录：

- `backend/scripts/`
- `backend/python_agents/`
- `workspace/memory/agents/`

当前不足：

- 没有 registry
- 没有 manifest
- 没有版本
- 没有测试记录索引

### 7.4 是否已有 experiment spec / result comparison 可供 agent 使用

结论：

- 有，而且应直接复用。

已有：

- `ExperimentSpec.SpecJSON`
- `ResultComparison`
- 自动归档 `ResultArchive`
- comparison workspace 输出

这部分是阶段3 planner / evaluator / reviewer agent 最应复用的资产。

## 8. 不建议当前做的模块

以下内容不建议在当前阶段3起步时就做：

- 完全自治科研闭环
- 长时自主循环 agent
- 自由对话式多智能体系统
- 新建独立于 `Experiment` 体系之外的 agent workflow 数据库
- 新建独立 SSH / GPU / dataset scan 基础设施
- 重型分布式 worker 平台
- 复杂 prompt 市场、动态插件生态
- 多模态科研闭环的端到端自治执行

原因：

- 会直接突破本阶段“受控节点”边界
- 会和阶段2已有执行设施重复建设
- 会把最关键的 contract / adapter / schema repair 工作推迟

## 9. 阶段3最小起步模块建议

建议按以下顺序启动：

1. `agent runtime`
2. `agent contract + schema registry`
3. `execution adapter (api / codex_cli / mock)`
4. `python agent invoker`
5. `tool registry`
6. `skill registry`
7. `memory registry`
8. `schema repair`
9. 第一个最小 planner agent

原因：

- 这条顺序与当前工程原则一致
- 能最大化复用阶段2已有实验生命周期
- 能最小化对真实服务器、真实模型配置的依赖

## 10. 复用清单

阶段3应优先复用：

- `backend/go/internal/service/experiment_service.go`
- `backend/go/internal/scheduler/service.go`
- `backend/go/internal/runner/service.go`
- `backend/go/internal/recovery/service.go`
- `backend/go/internal/logcollector/service.go`
- `backend/go/internal/resultcompare/service.go`
- `backend/go/internal/service/server_service.go`
- `backend/go/internal/service/ssh_gateway.go`
- `backend/go/internal/gpuresource/service.go`
- `backend/go/internal/heartbeat/service.go`
- `backend/go/internal/service/dataset_service.go`
- `backend/go/internal/service/dataset_remote_runtime.go`
- `backend/go/internal/service/paper_service.go`
- `backend/go/internal/service/dataset_asset_service.go`
- `backend/go/internal/service/baseline_service.go`
- `backend/go/internal/service/result_archive_service.go`
- `backend/go/internal/model/experiment_models.go`
- `backend/go/internal/model/research_asset_models.go`
- `backend/python_agents/`
- `backend/python_templates/`
- `workspace/memory/`

## 11. 审计结论

当前系统已经具备阶段3所需的大部分“基础设施层”和“对象模型层”，真正缺失的是“受控 agent 公共层”。

因此阶段3不应从 agent 业务开始，而应先建设：

- runtime
- contract
- adapter
- schema repair
- tool registry

在这五块完成前，不建议直接实现具体 agent 业务节点。
