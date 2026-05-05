# 阶段2规格说明

## 1. 文档目标

本文定义 MRAG 阶段2“实验自动化期”的最小可运行规格。

阶段2的核心目标不是扩展新的科研对象体系，而是在现有 MRAG 与阶段1资产模型之上，补齐实验执行基础设施与实验生命周期管理能力，形成：

- 可创建 experiment spec
- 可调度到可用服务器
- 可运行、可恢复、可审计
- 可查看运行日志与状态
- 可与 baseline / 历史结果进行最小对比

本文严格遵守以下边界：

- 优先复用 MRAG 已有 SSH / GPU / server / dataset scan 能力
- 优先复用阶段1已有 `DatasetAsset` / `Baseline` / `ResultArchive` / `Idea`
- 不进入阶段3/阶段4
- 不另起平行系统

## 2. 阶段2目标

阶段2目标如下：

### 2.1 执行基础设施

- 建立实验对象与运行对象
- 建立服务器心跳与资源快照机制
- 建立可解释的任务调度逻辑
- 建立统一 experiment spec 生成机制
- 建立远程实验执行入口与轮询入口

### 2.2 生命周期管理

- 让实验状态从草稿到归档形成闭环
- 保留每次运行的调度决策、日志、错误与产物
- 支持失败后重试或恢复
- 支持把完成结果沉淀为 `ResultArchive`

### 2.3 查看与比对

- 前端可查看实验列表、实验详情、运行日志、资源状态
- 前端可查看最小版本的结果对比

## 3. 非目标

以下内容不属于阶段2：

- 多 agent 自治科研 workflow
- 自动论文撰写、自动研究计划编排
- 复杂多阶段 DAG 工作流引擎
- 高级队列系统、消息总线、微服务化拆分
- 多租户权限系统
- 大规模资源编排平台化
- 阶段3/阶段4式策略优化与自治实验搜索

## 4. 复用原则

### 4.1 必须复用的现有能力

- `servers` 及其管理页面
- `SSHGateway`
- GPU 探测链
- 数据集路径校验与扫描结果
- `Dataset`
- `DatasetAsset`
- `Baseline`
- `ResultArchive`
- `Idea`

### 4.2 必须扩展而非重写的能力

- server 状态刷新扩展为 server heartbeat
- 手动 GPU 检查扩展为资源快照链
- dataset index task 的任务模式扩展为 experiment run 模式
- result archive 从手工归档扩展为实验完成后归档入口

### 4.3 可吸收但不可整套迁回的历史能力

可吸收：

- `Auto_v1` 的 experiment spec 文件协议
- `Auto_v1` 的状态机 / trace / artifact 组织方式
- `Auto_v1` 的 run 级日志与失败释放 server 逻辑

不可整套迁回：

- `Auto_v1` 的 phase0 workflow 主线
- `workflow_jobs` 作为阶段2中心对象
- mock experiment 主链路本身

## 5. 实验生命周期

阶段2实验生命周期主状态至少包含：

- `draft`
- `spec_ready`
- `queued`
- `scheduled`
- `running`
- `succeeded`
- `failed`
- `cancelled`
- `archived`

状态含义摘要：

- `draft`：实验草稿已建立，但 spec 还未冻结
- `spec_ready`：experiment spec 已生成并通过校验，可进入调度
- `queued`：进入待调度队列，等待资源可用
- `scheduled`：已选定服务器和资源，等待真正启动 run
- `running`：远程执行已开始，日志和状态持续采集
- `succeeded`：运行成功结束，产物与结果可归档
- `failed`：运行失败，保留错误信息，允许恢复或重试
- `cancelled`：由用户或系统取消
- `archived`：实验对象已进入历史归档态

更细的状态流转与约束见：

- [experiment-lifecycle.md](/D:/2/MRAG/docs/experiment-lifecycle.md)

## 6. 阶段2对象模型

阶段2最小对象模型至少包括：

- `Experiment`
- `ExperimentRun`
- `ExperimentSpec`
- `RunLog`
- `SchedulerDecision`
- `ServerHeartbeat`
- `GPUResourceSnapshot`
- `ResultComparison`

### 6.1 Experiment

定义：

- 表示一个实验定义对象
- 它承接阶段2的“研究意图 + 数据集资产 + baseline 参照 + spec”

最小字段建议：

- `id`
- `title`
- `status`
- `idea_id`
- `dataset_asset_id`
- `baseline_id`
- `latest_run_id`
- `latest_spec_id`
- `summary_md`
- `owner_note_md`
- `created_at`
- `updated_at`
- `archived_at`

说明：

- `Experiment` 是生命周期主对象
- 一个 `Experiment` 可以有多个 `ExperimentRun`
- 一个 `Experiment` 必须关联一个 `DatasetAsset`
- 一个 `Experiment` 可选关联一个 `Idea`
- 一个 `Experiment` 可选关联一个主 `Baseline`

### 6.2 ExperimentRun

定义：

- 表示某个实验的一次实际运行实例
- 是调度、执行、恢复、日志和产物的主体

最小字段建议：

- `id`
- `experiment_id`
- `status`
- `run_number`
- `server_id`
- `spec_id`
- `scheduler_decision_id`
- `remote_job_id`
- `log_path`
- `status_path`
- `result_path`
- `error_message`
- `started_at`
- `finished_at`
- `created_at`
- `updated_at`

说明：

- 一次失败重试应生成新的 `ExperimentRun`
- run 是审计与恢复的最小单位

### 6.3 ExperimentSpec

定义：

- 表示一份冻结的实验执行规格

最小字段建议：

- `id`
- `experiment_id`
- `spec_version`
- `spec_json`
- `spec_hash`
- `generator_version`
- `validation_status`
- `created_at`

最小内容要求：

- 引用的 `idea`
- 引用的 `dataset asset`
- 引用的 `baseline`
- 训练模板类型
- 目标指标
- 资源需求
- 输出路径约定
- 恢复策略

说明：

- spec 必须是可追踪、可审计、可复现的
- 阶段2不要求复杂 planner，只要求稳定生成与冻结

### 6.4 RunLog

定义：

- 表示某次 `ExperimentRun` 的事件日志或输出日志条目

最小字段建议：

- `id`
- `run_id`
- `log_type`
- `level`
- `message`
- `source`
- `created_at`

`log_type` 最小建议值：

- `event`
- `stdout`
- `stderr`
- `system`

说明：

- 阶段2先做日志摘要和关键事件，不要求完整实时终端回放

### 6.5 SchedulerDecision

定义：

- 表示一次调度决策的记录

最小字段建议：

- `id`
- `run_id`
- `selected_server_id`
- `decision_reason`
- `candidate_snapshot_json`
- `required_resource_json`
- `decision_status`
- `created_at`

说明：

- 用于追踪为什么选中某台服务器
- 也是调试调度错误的重要依据

### 6.6 ServerHeartbeat

定义：

- 表示某台服务器的一次心跳记录

最小字段建议：

- `id`
- `server_id`
- `heartbeat_status`
- `message`
- `checked_at`

说明：

- 阶段2不依赖 agent 主动上报
- 最小版本可由后端定时或调度前探测生成

### 6.7 GPUResourceSnapshot

定义：

- 表示某台服务器某次 GPU 资源快照

最小字段建议：

- `id`
- `server_id`
- `available_gpu_count`
- `total_gpu_count`
- `snapshot_json`
- `checked_at`

说明：

- 最小版本可直接复用当前 `CheckGPU` 输出结构
- 后续调度、失败分析、资源视图都依赖它

### 6.8 ResultComparison

定义：

- 表示实验结果与 baseline 或历史结果的对比记录

最小字段建议：

- `id`
- `experiment_id`
- `run_id`
- `comparison_type`
- `baseline_id`
- `target_result_archive_id`
- `summary_md`
- `diff_json`
- `status`
- `created_at`

`comparison_type` 最小建议值：

- `baseline`
- `historical_result`

说明：

- 阶段2先做最小版本
- 先只比较结构化指标，不做复杂图表分析

## 7. 对象关系

### 7.1 Experiment 与已有对象关系

- `Experiment.idea_id -> ideas.id`
- `Experiment.dataset_asset_id -> dataset_assets.id`
- `Experiment.baseline_id -> baselines.id`

说明：

- `Idea` 代表实验动机或研究问题来源
- `DatasetAsset` 是实验输入数据语义主对象
- `Baseline` 是实验默认比较参照

### 7.2 Experiment 与 ResultArchive 的关系

- `Experiment` 不直接等于 `ResultArchive`
- `ResultArchive` 是实验完成后的沉淀对象

最小策略：

- `ExperimentRun` 成功后，可以生成或更新一个 `ResultArchive`
- `ResultArchive` 继续作为长期历史记录存在
- `Experiment` 保持执行生命周期语义

### 7.3 ResultComparison 的最小版本

最小版本只要求：

- 比较当前 run 的指标结果与一个 baseline 的 `result_json`
- 或比较当前 run 与一个历史 `ResultArchive.metric_json`
- 生成结构化 `diff_json`
- 生成简要 `summary_md`

最小 `diff_json` 结构建议：

```json
{
  "primaryMetric": "accuracy",
  "current": 0.86,
  "target": 0.83,
  "delta": 0.03,
  "direction": "improved",
  "otherMetrics": []
}
```

阶段2不要求：

- 自动图表生成
- 多目标复杂统计显著性分析
- 跨多次 run 智能归因

## 8. 后端模块范围

阶段2后端范围至少包括以下模块：

### 8.1 新建或恢复的核心模块

- `experiment_handler`
- `experiment_service`
- `experiment_repository`
- `scheduler_service`
- `run_log_service`
- `comparison_service`

### 8.2 在现有模块上扩展

- `server_service`
- `server_repository`
- `result_archive_service`
- `baseline_service`
- `dataset_asset_service`

### 8.3 数据库范围

建议新增最小表：

- `experiments`
- `experiment_specs`
- `experiment_runs`
- `run_logs`
- `scheduler_decisions`
- `server_heartbeats`
- `gpu_resource_snapshots`
- `result_comparisons`

### 8.4 Workspace 范围

建议扩展支持：

```text
workspace/
  experiments/
    <experiment_id>/
      specs/
      runs/
        <run_id>/
          logs/
          artifacts/
          result/
```

说明：

- PostgreSQL 仍是真源
- workspace 作为 supporting artifacts 与审计文件面

## 9. 前端页面范围

阶段2前端范围至少包括：

### 9.1 实验列表页

需要支持：

- 查看实验列表
- 按状态筛选
- 查看关联 `Idea / DatasetAsset / Baseline`
- 查看最新 run 状态
- 发起新实验或进入详情

### 9.2 实验详情页

需要支持：

- 查看 experiment 基本信息
- 查看当前 spec
- 查看关联对象
- 查看 run 历史
- 查看最近调度决策
- 发起重试 / 取消 / 归档

### 9.3 运行日志页或日志区域

需要支持：

- 查看关键事件日志
- 查看 stdout/stderr 摘要
- 查看错误信息
- 查看最近更新时间

### 9.4 服务器资源页

可以扩展现有 `ServerPage`

需要支持：

- 服务器心跳状态
- 最近一次 GPU 快照
- 可用 GPU 数量
- 最近调度占用情况

### 9.5 结果对比页

需要支持：

- 当前 run 与 baseline 对比
- 当前 run 与历史结果对比
- 指标差异摘要
- 最小 diff 表格或卡片

## 10. 最小执行链

阶段2最小闭环建议为：

1. 基于 `Idea + DatasetAsset + Baseline` 创建 `Experiment`
2. 生成 `ExperimentSpec`
3. 状态进入 `spec_ready`
4. 提交进入 `queued`
5. 调度器选择 server，写入 `SchedulerDecision`
6. 生成 `ExperimentRun`，状态进入 `scheduled`
7. 远程执行启动，状态进入 `running`
8. 采集关键日志与状态
9. 成功则进入 `succeeded`，失败则进入 `failed`
10. 成功结果沉淀到 `ResultArchive`
11. 生成最小 `ResultComparison`
12. 可选归档 `Experiment`

## 11. 可恢复性要求

阶段2最小恢复要求：

- run 启动失败必须保留错误信息
- 远程任务状态未知时必须可再次同步
- 失败重试必须生成新的 `ExperimentRun`
- 已有日志、spec、调度决策不可丢失

阶段2不要求：

- 训练过程中的 checkpoint 自动接续到算法内部细节
- 跨机器迁移恢复

## 12. 可测试性要求

所有新增能力必须可测试、可恢复、可追踪。

最小测试范围应覆盖：

- spec 生成与校验
- 状态流转
- 调度决策
- 运行失败与重试
- comparison 生成
- result archive 沉淀

## 13. 文档关系

本阶段2规格文档配套文档如下：

- [stage2-acceptance.md](/D:/2/MRAG/docs/stage2-acceptance.md)
- [experiment-lifecycle.md](/D:/2/MRAG/docs/experiment-lifecycle.md)

