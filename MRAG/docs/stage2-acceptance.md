# 阶段2验收标准

## 1. 文档目标

本文定义阶段2“实验自动化期”的验收标准。

阶段2验收关注的是：

- 是否形成实验执行闭环
- 是否复用了现有 MRAG 能力
- 是否具备可恢复、可追踪、可查看能力

## 2. 总体验收原则

阶段2通过验收，必须满足以下原则：

- 不另起平行系统
- 服务器、SSH、GPU、数据集扫描必须复用 MRAG 现有能力
- `DatasetAsset`、`Baseline`、`ResultArchive` 必须继续作为主资产对象
- 实验对象必须与阶段1资产对象打通
- 所有关键状态必须可查询
- 失败必须可重试或恢复

## 3. 核心验收目标

阶段2完成时，至少应满足：

1. 能创建 `Experiment`
2. 能生成 `ExperimentSpec`
3. 能基于 `Idea + DatasetAsset + Baseline` 发起实验
4. 能调度到可用服务器
5. 能记录运行状态
6. 能采集运行日志摘要和错误信息
7. 能在失败后重试或恢复
8. 能生成结果并进行最小对比
9. 能在前端查看实验列表、详情、日志与对比结果

## 4. 生命周期验收

### 4.1 状态集合必须存在

系统中必须存在并可查询以下实验生命周期状态：

- `draft`
- `spec_ready`
- `queued`
- `scheduled`
- `running`
- `succeeded`
- `failed`
- `cancelled`
- `archived`

### 4.2 状态流转必须可验证

至少应验证以下流转：

- `draft -> spec_ready`
- `spec_ready -> queued`
- `queued -> scheduled`
- `scheduled -> running`
- `running -> succeeded`
- `running -> failed`
- `queued|scheduled|running -> cancelled`
- `succeeded|failed|cancelled -> archived`

### 4.3 非法流转必须受控

至少应保证：

- 未生成 spec 的实验不能直接进入 `queued`
- 已 `archived` 的实验不能重新运行
- 已完成 run 不能再次进入 `running`

## 5. 对象模型验收

阶段2最小对象必须存在并可通过 API 或数据库确认：

- `Experiment`
- `ExperimentRun`
- `ExperimentSpec`
- `RunLog`
- `SchedulerDecision`
- `ServerHeartbeat`
- `GPUResourceSnapshot`
- `ResultComparison`

### 5.1 关联关系必须成立

至少应验证：

- `Experiment` 可关联 `Idea`
- `Experiment` 可关联 `DatasetAsset`
- `Experiment` 可关联 `Baseline`
- `ExperimentRun` 可关联 `Experiment`
- `ExperimentRun` 可关联 `Server`
- `ResultComparison` 可关联 `ExperimentRun`
- 成功 run 可沉淀到 `ResultArchive`

## 6. 服务器与资源验收

### 6.1 服务器复用验收

必须基于 MRAG 现有 `servers` 能力，而不是新建 server 系统。

至少应验证：

- 调度读取现有 `servers`
- 使用现有 SSH 能力执行远程任务
- 使用现有 GPU 探测能力作为调度输入

### 6.2 心跳验收

至少应验证：

- 可产生 `ServerHeartbeat`
- 可看到最近一次心跳状态与时间
- 心跳异常服务器不会被优先调度

### 6.3 GPU 快照验收

至少应验证：

- 可产生 `GPUResourceSnapshot`
- 调度决策可引用最近资源快照
- 前端可查看最近资源状态

## 7. 实验执行验收

### 7.1 Spec 验收

至少应验证：

- 可生成一份结构稳定的 `ExperimentSpec`
- spec 中包含 `Idea / DatasetAsset / Baseline / 资源需求 / 输出约定`
- spec 可追踪到具体对象版本或 ID

### 7.2 调度验收

至少应验证：

- 一个 `queued` 实验可被调度
- 系统生成 `SchedulerDecision`
- 决策中能看到目标 server 与选择理由

### 7.3 运行验收

至少应验证：

- 一个 `scheduled` run 可以进入 `running`
- `running` 期间有日志或事件更新
- 成功后进入 `succeeded`
- 失败后进入 `failed`

## 8. 日志与错误验收

至少应验证：

- 每个 `ExperimentRun` 至少有事件日志
- 失败 run 能看到错误信息
- 日志可在前端查看摘要
- 日志与 run 生命周期状态一致

## 9. 恢复与重试验收

至少应验证：

- `failed` run 可以发起重试
- 重试会生成新的 `ExperimentRun`
- 老 run 记录与日志仍保留
- 未知状态 run 可以触发同步或恢复动作

## 10. 结果沉淀与对比验收

### 10.1 ResultArchive 验收

至少应验证：

- 成功 run 可以生成或绑定一个 `ResultArchive`
- 归档后仍可追溯来源 experiment/run

### 10.2 ResultComparison 验收

至少应验证：

- 当前 run 能与一个 `Baseline` 做最小指标比较
- 当前 run 能与一个历史 `ResultArchive` 做最小指标比较
- 对比结果有结构化 `diff_json` 和摘要

## 11. 前端页面验收

阶段2至少应存在以下页面或等价区域：

- 实验列表页
- 实验详情页
- 运行日志区域或日志页
- 服务器资源页
- 结果对比页

### 11.1 实验列表页验收

至少应支持：

- 查看实验列表
- 查看状态
- 查看关联对象
- 进入详情

### 11.2 实验详情页验收

至少应支持：

- 查看 spec 摘要
- 查看 run 历史
- 查看最新状态
- 查看调度信息

### 11.3 日志区域验收

至少应支持：

- 查看最近日志
- 查看失败错误
- 区分事件日志与输出日志

### 11.4 服务器资源页验收

至少应支持：

- 查看心跳
- 查看 GPU 快照
- 查看可用资源摘要

### 11.5 结果对比页验收

至少应支持：

- baseline 对比
- 历史结果对比
- 指标差值展示

## 12. 最小验收场景

阶段2最小验收建议至少跑通以下场景：

### 场景 A：成功闭环

1. 创建实验草稿
2. 生成 spec
3. 提交调度
4. 成功运行
5. 生成结果归档
6. 生成 baseline 对比
7. 前端可查看全链路

### 场景 B：失败与重试

1. 创建实验
2. 调度并运行失败
3. 记录错误与日志
4. 发起重试
5. 新 run 成功完成

### 场景 C：资源视图

1. 刷新 server heartbeat
2. 刷新 GPU snapshot
3. 在资源页看到最近状态
4. 调度决策可引用这些数据

## 13. 不计入本阶段通过条件的内容

以下内容即使未完成，也不影响阶段2通过：

- 复杂自治 agent
- 多阶段研究编排
- 智能实验搜索
- 高级统计显著性分析
- 完整实时终端流式日志回放

## 14. 通过结论

当且仅当以下条件同时满足时，阶段2可视为通过：

- 生命周期闭环成立
- 对象模型完整可查
- 调度与运行真实打通
- 失败可恢复或重试
- 结果可归档与最小对比
- 前端可查看关键状态、日志和对比结果

