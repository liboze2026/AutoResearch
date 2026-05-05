# 阶段3 Agent 架构说明

## 1. 文档目标

本文描述阶段3的 agent 运行架构、角色拆分、执行链路、并发策略、修复策略以及与阶段1/2对象模型的映射关系。

本文只定义架构，不涉及代码实现。

## 2. 总体架构

阶段3采用“受控 agent runtime + 结构化对象驱动 + 有限并行 pipeline”架构。

总体分层如下：

1. `Research Asset Layer`
2. `Experiment Control Layer`
3. `Agent Runtime Layer`
4. `Execution Adapter Layer`
5. `Persistence Layer`

## 3. 分层定义

### 3.1 Research Asset Layer

复用阶段1对象：

- `Paper`
- `PaperInsight`
- `Idea`
- `DatasetAsset`
- `Baseline`
- `ResultArchive`

作用：

- 提供 agent 的主要上游输入和最终成果落点

### 3.2 Experiment Control Layer

复用阶段2对象：

- `Experiment`
- `ExperimentSpec`
- `ExperimentRun`
- `RunLog`
- `SchedulerDecision`
- `ResultComparison`
- `ServerHeartbeat`
- `GPUResourceSnapshot`

作用：

- 提供受控执行状态机、调度、日志、恢复、对比、归档能力

### 3.3 Agent Runtime Layer

阶段3新建：

- `AgentRegistry`
- `AgentRuntime`
- `AgentContractRegistry`
- `PipelineCoordinator`
- `ValidationEngine`
- `RepairEngine`

作用：

- 把每个 agent 变成受控节点

### 3.4 Execution Adapter Layer

阶段3新建：

- `API Adapter`
- `Codex CLI Adapter`
- `Mock Adapter`

作用：

- 屏蔽不同执行方式差异
- 对上统一 contract

### 3.5 Persistence Layer

阶段3新增但依托现有基础：

- `Tool Registry`
- `Skill Registry`
- `Memory Registry`
- `Agent Execution Log`
- `Agent Repair Record`

## 4. 核心执行链

阶段3的最小执行链建议如下：

- `Reader Agent`
- `Insight Agent`
- `Dataset Agent`
- `Idea Generator Agent`
- `Planner Agent`
- `Coding+Evaluator Agent`
- `Writer+Picture Agent`

其中：

- `Picture` 第一版使用 mock
- `Coding + Evaluator` 第一版合并
- `Writer + Picture` 第一版合并

## 5. 角色架构

### 5.1 Reader Agent

输入：

- `Paper`
- paper file refs

输出：

- 结构化阅读摘要
- section map
- evidence refs

第一版定位：

- 必做最小版

### 5.2 Insight Agent

输入：

- `Reader` 输出
- `Paper`

输出：

- `summary`
- `methods`
- `contributions`
- `limitations`

第一版定位：

- 必做最小版

### 5.3 Dataset Agent

输入：

- `Dataset`
- `DatasetScanRecord`
- `DatasetAsset`
- `Baseline`

输出：

- 数据集约束
- 加载建议
- 风险提示
- 任务类型判断

第一版定位：

- 必做最小版

### 5.4 Idea Generator Agent

输入：

- `Insight` 输出
- `Dataset`/`DatasetAsset` 输出

输出：

- 候选 idea 列表
- idea rationale
- idea ranking

第一版定位：

- 必做最小版

### 5.5 Planner Agent

输入：

- `Idea`
- `DatasetAsset`
- `Baseline`
- `ResultArchive`
- `ResultComparison`

输出：

- 结构化 plan
- `ExperimentSpec` 草案
- 资源需求
- 比较目标

第一版定位：

- 必做最小版
- 阶段3第一核心 agent

### 5.6 Coding Agent

输入：

- `Planner` 输出
- train template ref
- tool refs

输出：

- 最小执行实现建议
- 代码/脚本草案
- 实验实现摘要

第一版定位：

- 与 Evaluator 合并

### 5.7 Evaluator Agent

输入：

- `Coding` 输出
- run outputs
- metrics
- logs

输出：

- 结构化评估
- 合规判断
- 是否需要 repair

第一版定位：

- 与 Coding 合并

### 5.8 Writer Agent

输入：

- plan
- run result
- compare result

输出：

- 结构化报告
- 归档摘要
- 对外可读 markdown

第一版定位：

- 与 Picture 合并

### 5.9 Picture Agent

输入：

- Writer 草稿
- metrics/compare 数据

输出：

- 图示占位物
- 图表说明
- 图片产物 ref

第一版定位：

- 与 Writer 合并
- 先 mock

## 6. 第一版角色合并策略

### 6.1 必须保留为独立节点

- `Reader`
- `Insight`
- `Dataset`
- `Idea Generator`
- `Planner`

原因：

- 它们对应不同上游对象与下游触发条件
- 合并过早会削弱结构化边界

### 6.2 第一版允许合并

- `Coding + Evaluator`
- `Writer + Picture`

原因：

- 可以显著降低第一版节点数量
- 仍不破坏受控链路

## 7. 生命周期架构

阶段3统一生命周期如下：

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

### 7.1 生命周期摘要

- `registered`：已注册，可被编排器识别
- `idle`：等待任务
- `waiting_input`：依赖不足
- `ready`：可调度
- `running`：执行中
- `validating`：输出校验中
- `repairing`：自动修复中
- `succeeded`：完成并持久化
- `failed`：执行或修复失败
- `paused`：人工/系统暂停

## 8. 并行架构

### 8.1 并行原则

- 上游输出通过校验后，允许触发下游
- 只有结构化输出可以作为触发依据
- 默认并发按角色分级，不全局放开

### 8.2 角色并发策略

相对高并发：

- `Reader`
- `Insight`
- `Dataset`
- `Idea Generator`

默认低并发：

- `Coding`
- `Writer`

第一版建议：

- Reader/Insight/Dataset/Idea：支持小规模并行
- Planner：中低并发
- Coding+Evaluator：低并发
- Writer+Picture：低并发

### 8.3 互斥规则

必须至少支持以下互斥：

- 同一 `ExperimentSpec` 上仅允许一个 `Coding+Evaluator` 活跃执行
- 同一 `ResultArchive` 目标上仅允许一个 `Writer+Picture` 活跃执行
- 同一 primary object 上避免多 planner 竞争写入

## 9. 验证与修复架构

### 9.1 Validation Engine

职责：

- 解析 adapter 返回
- 进行 schema 校验
- 进行必填字段校验
- 进行语义完整性检查
- 生成 validation report

### 9.2 Repair Engine

职责：

- 非法 JSON 修复
- 字段缺失补齐
- 格式归一化
- 必要时触发单次重试
- 达到上限后 fail

### 9.3 修复边界

允许：

- 修复格式
- 修复缺字段
- 重建合法输出结构

不允许：

- 伪造事实
- 跳过关键依赖
- 改写上游已确认内容

## 10. Tools / Skills / Memory 架构

### 10.1 Tools

工具必须注册后才能使用。

第一版最小要求：

- 有唯一 ID
- 有调用方式
- 有 schema
- 有测试结果
- 有是否可复用的状态

### 10.2 Skills

skills 作为 agent 行为约束与模板，不是自由 prompt 文本堆。

第一版最小要求：

- 可版本化
- 可引用
- 可绑定到 agent role

### 10.3 Memory

memory 作为辅助上下文，不是主事实源。

第一版最小要求：

- 有来源
- 有作用域
- 有可失效机制
- 不能覆盖数据库主对象

## 11. 与现有对象的映射关系

建议映射如下：

- `Reader/Insight/Dataset/Idea` 中间输出：落到 agent execution artifacts
- `Planner` 产物：写入 `ExperimentSpec`
- `Coding+Evaluator` 产物：写入 `ExperimentRun.ResultJSON` 与 `RunLog`
- `Writer+Picture` 产物：写入 `ResultArchive`
- compare 类结论：复用 `ResultComparison`

## 12. 架构结论

阶段3不是“再做一个智能体系统”，而是在现有 MRAG 资产层和实验层之上，补一层受控 agent runtime。

第一版最重要的架构决策是：

- 角色受控
- 契约统一
- adapter 统一
- 校验必经
- repair 受限
- 并行受控
- 持久化优先
- 不进入完全自治闭环
