# 阶段3验收标准

## 1. 文档目标

本文定义阶段3“受控智能体期”的验收标准。

阶段3验收关注的是：

- 是否建立了统一 agent runtime
- 是否形成结构化、可校验、可修复、可持久化的 agent 链路
- 是否严格复用了 MRAG 阶段1/2已有基础设施
- 是否仍然停留在受控边界内，而没有滑向完全自治闭环

## 2. 总体验收原则

阶段3通过验收，必须同时满足以下原则：

- 不另起平行系统
- 不重做 SSH / GPU / dataset scan / experiment lifecycle
- agent 必须是受控节点
- execution mode 必须统一
- 输出必须结构化
- 校验与修复必须经过 runtime
- 并行必须受控
- tools / skills / memory 必须可持久化
- 当前不进入完全自治闭环

## 3. 核心验收目标

阶段3完成时，至少应满足：

1. 存在统一 agent runtime
2. 存在统一 agent 生命周期
3. 存在统一输入输出契约
4. 存在 `api / codex_cli / mock` 三种执行模式
5. 存在 validation + repair 链
6. 存在 tools / skills / memory 持久化机制
7. 存在由上游输出触发下游的受控 pipeline
8. 至少跑通一条最小 agent 闭环
9. 仍未进入完全自治科研闭环

## 4. 角色验收

系统中至少应定义以下角色：

- `Reader Agent`
- `Insight Agent`
- `Dataset Agent`
- `Idea Generator Agent`
- `Planner Agent`
- `Coding Agent`
- `Evaluator Agent`
- `Writer Agent`
- `Picture Agent`

### 4.1 第一版角色实现验收

第一版必须完成以下最小实现或最小合并实现：

- `Reader Agent`
- `Insight Agent`
- `Dataset Agent`
- `Idea Generator Agent`
- `Planner Agent`
- `Coding+Evaluator Agent`
- `Writer+Picture Agent`

并满足：

- `Coding + Evaluator` 已合并
- `Writer + Picture` 已合并
- `Picture` 至少存在 mock 路径

## 5. 生命周期验收

系统中必须存在以下统一生命周期状态：

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

### 5.1 生命周期流转验收

至少应验证以下流转：

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
- `ready -> paused`
- `paused -> ready`

### 5.2 非法流转受控验收

至少应保证：

- 未满足输入的 agent 不可直接 `running`
- 未经过 `validating` 的输出不能直接 `succeeded`
- `failed` 状态不能未经重新调度直接视为成功

## 6. 执行模式验收

每个 agent 必须支持：

- `execution_mode`
- `model_provider`
- `model_name`
- `prompt_version`
- `skill_refs`
- `tool_refs`

### 6.1 模式集合验收

统一 execution mode 至少包括：

- `api`
- `codex_cli`
- `mock`

### 6.2 模式行为验收

至少应验证：

- 三种模式共用同一输入输出 contract
- `codex_cli` 可作为当前最小联调模式
- `mock` 可作为稳定 fallback
- `api` 具备接入位但不要求当前真实 key 联调完成

## 7. 输入输出契约验收

### 7.1 输入契约验收

每个 agent 输入至少应可查询或可追踪到：

- 输入版本
- 上游对象引用
- tool refs
- skill refs
- memory refs
- 结构化 payload

### 7.2 输出契约验收

每个 agent 输出至少应包含：

- `status`
- `summary`
- `structured_output`
- `validation_report`
- `repair_report`
- `artifact_refs`

### 7.3 输出质量验收

至少应验证：

- 输出为结构化 JSON 或等价结构化对象
- 输出可通过 schema 校验
- 输出可持久化
- 输出可被下游消费

## 8. Validation / Repair 验收

### 8.1 Validation 验收

至少应验证：

- 非法 JSON 能被识别
- 缺失字段能被识别
- 类型不匹配能被识别
- 语义不完整能被识别

### 8.2 Repair 验收

至少应验证：

- 输出校验失败后能进入 `repairing`
- repair 结果会重新进入 `validating`
- repair 成功后进入 `succeeded`
- repair 超过上限进入 `failed`

### 8.3 Repair 边界验收

必须证明：

- repair 不会伪造上游事实
- repair 不会绕过关键输入缺失
- repair 不会偷偷引入未注册工具

## 9. 并行流水线验收

### 9.1 触发机制验收

至少应验证：

- 上游输出通过校验后可触发下游
- 未通过校验的上游输出不能触发下游
- 下游可追溯其上游来源

### 9.2 并发策略验收

至少应验证：

- `Reader / Insight / Dataset / Idea` 支持相对高并发
- `Coding / Writer` 默认低并发
- 同一关键对象上的冲突写入受限

### 9.3 并行边界验收

必须保证：

- 并行不会造成同一 spec 的竞争覆盖
- 并行不会破坏日志与审计可回放性

## 10. Tools / Skills / Memory 验收

### 10.1 Tools

至少应验证：

- 工具可注册
- 工具有调用方式记录
- 工具有 schema 记录
- 工具有测试状态记录
- 未注册工具不能被 agent 复用

### 10.2 Skills

至少应验证：

- skill 可持久化
- skill 可版本化
- skill 可被 agent 引用

### 10.3 Memory

至少应验证：

- memory 可持久化
- memory 有来源
- memory 有 scope
- memory 不替代数据库主事实

## 11. 与阶段2对象集成验收

至少应验证：

- planner 结果可写入 `ExperimentSpec`
- agent 执行结果可写入 `ExperimentRun` / `RunLog`
- 汇总结果可写入 `ResultArchive`
- 需要对比时可复用 `ResultComparison`

## 12. 最小验收场景

### 场景 A：最小 planning 闭环

1. 输入 paper / insight / dataset asset
2. `Reader` 输出结构化阅读结果
3. `Insight` 输出结构化 insight
4. `Dataset` 输出结构化数据集约束
5. `Idea Generator` 输出候选 idea
6. `Planner` 输出结构化 spec
7. spec 成功落入 `ExperimentSpec`

### 场景 B：执行与校验闭环

1. 基于 planner spec 触发 `Coding+Evaluator`
2. 产生结构化执行结果
3. 输出进入 `validating`
4. 校验失败则进入 `repairing`
5. 修复成功后进入 `succeeded`
6. 结果落入 `ExperimentRun` / `RunLog`

### 场景 C：写作与归档闭环

1. `Writer+Picture` 消费前序结果
2. `Picture` 走 mock 输出
3. 生成结构化报告与归档摘要
4. 成果落入 `ResultArchive`

## 13. 不计入本阶段通过条件的内容

以下内容即使未完成，也不影响阶段3通过：

- 完全自治科研闭环
- 长时自主探索
- 无人值守的多轮自我迭代
- 高级多 agent 协商系统
- 真实图片生成型 `Picture Agent`
- 完整开放工具生态

## 14. 通过结论

当且仅当以下条件同时满足时，阶段3可视为通过：

- agent runtime 成立
- 统一 lifecycle 成立
- 三种 execution mode 成立
- 统一 contract 成立
- validation / repair 成立
- tools / skills / memory 持久化成立
- 最小 agent pipeline 成立
- 仍明确停留在受控边界内
