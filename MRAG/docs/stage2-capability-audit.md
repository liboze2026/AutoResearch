# 阶段2能力盘点

## 1. 盘点目标

本文用于盘点当前 `MRAG` 主系统、阶段0 `Auto_v1`、以及阶段1已落地模块中，哪些能力已经存在、哪些应直接复用、哪些应在原模块上扩展、哪些仍属于阶段2缺口。

本盘点严格遵守以下边界：

- 先盘点现有能力，再补齐缺口
- 优先复用 MRAG 已有 SSH / GPU / 服务器管理 / 数据集扫描能力
- 优先复用阶段1已有 `dataset asset` / `baseline` / `result archive` 对象模型
- 阶段2只聚焦“执行基础设施”和“实验生命周期管理”
- 不提前进入阶段3/阶段4
- 不另起平行系统

## 2. 结论摘要

当前系统已经具备阶段2最关键的几块底座，但还没有“实验自动化闭环”本身。

已经存在的、可直接作为阶段2底座的能力主要有：

- 服务器管理、SSH 连通性验证、远程命令执行底座
- 基于 `nvidia-smi` 的 GPU 资源探测
- 基于 SSH 的远程数据集路径校验与目录扫描
- 数据集扫描记录、预览、索引任务、任务日志这些“任务生命周期”骨架
- 阶段1的 `DatasetAsset` / `Baseline` / `ResultArchive` / `Idea` 对象模型与页面
- 阶段0 `Auto_v1` 中已经验证过的 experiment spec、workflow 状态推进、artifact 落盘、失败记录、run 级日志组织方式

阶段2当前缺失的不是底层连接能力，而是：

- `Experiment` / `ExperimentRun` 领域模型
- 实验调度器
- 实验状态机与恢复机制
- 统一训练模板与 spec 生成服务
- 真实远程训练执行链
- 运行日志采集与摘要
- 结果对比服务与前端实验页面

## 3. 盘点范围

本次重点检查了以下代码与文档：

- MRAG 服务器管理模块
- MRAG SSH 连接与远程执行模块
- MRAG GPU 探测模块
- MRAG 数据集扫描模块
- 阶段0 `Auto_v1` 的 mock workflow / mock experiment 模块
- 阶段1 `DatasetAsset` / `Baseline` / `ResultArchive` / `Idea` 模块

## 4. 已有能力清单

### 4.1 MRAG 已有服务器管理能力

已实现：

- 服务器 CRUD
- 服务器状态刷新
- SSH 连通性测试
- GPU 检查
- 服务器扫描远程数据集目录
- 前端服务器管理页

关键位置：

- `backend/go/internal/service/server_service.go`
- `backend/go/internal/service/ssh_gateway.go`
- `backend/go/internal/repository/server_repository.go`
- `backend/go/internal/handler/server_handler.go`
- `backend/go/internal/router/router.go`
- `src/views/servers/ServerPage.vue`
- `src/api/modules/servers.ts`

当前特征：

- 服务器信息已持久化到 `servers` 表
- 已有 `status`、`status_message`、`available_gpus`、`total_gpus`、`last_heartbeat`、`last_gpu_check_at`
- 前端能手动触发“状态刷新 / 连接测试 / GPU 检查”

结论：

- 这是阶段2服务器底座的主复用入口
- 不应重写服务器模型或另建新的 server registry

### 4.2 SSH 连接与远程命令执行能力

已实现：

- system ssh 执行
- `.ssh/config` alias 解析
- password 模式直连
- `ProxyCommand` / jump host 解析
- mock / real 双模式
- 非交互远程命令执行

关键位置：

- `backend/go/internal/service/ssh_gateway.go`

当前特征：

- `SSHGateway` 已提供 `Probe()` 与 `Exec()`
- `SystemSSHGateway.Exec()` 已能执行任意远程命令数组
- 当前虽然没有“通用远程命令 API”，但服务层已经具备完整执行能力

结论：

- “是否已有远程命令执行逻辑”：有，且是可复用的真实底座
- 阶段2不应重新造 SSH 执行器
- 后续实验调度、心跳探测、日志采集都应复用这条链

### 4.3 GPU 资源探测能力

已实现：

- 远程执行 `nvidia-smi`
- 解析 GPU 数量、显存使用、利用率
- 推断 GPU 是否可用
- 回写到 `servers` 表
- 前端展示最近一次 GPU 检查结果

关键位置：

- `backend/go/internal/service/server_service.go`
- `backend/go/internal/model/models.go`
- `src/views/servers/ServerPage.vue`

当前探测规则：

- 依赖 `nvidia-smi --query-gpu=index,name,memory.used,memory.total,utilization.gpu`
- 当前“可用”规则为：
  - `utilization < 10`
  - `memory.used < 1024MB`

结论：

- “是否已有 GPU 空闲显存探测逻辑”：有
- 但当前是“手动触发式检查”，不是后台持续探测
- 阶段2应扩展为调度输入与周期采样，而不是重写 GPU 检测脚本

### 4.4 数据集路径校验与扫描能力

已实现：

- 本地 / 远程数据集路径校验
- 远程目录扫描
- 文件类型统计
- 层级摘要
- 预览项生成
- 扫描记录入库
- 前端扫描结果登记为数据集

关键位置：

- `backend/go/internal/service/dataset_service.go`
- `backend/go/internal/service/dataset_remote_runtime.go`
- `backend/go/internal/service/dataset_adapters.go`
- `backend/go/internal/model/models.go`
- `src/views/datasets/DatasetListPage.vue`
- `src/views/datasets/DatasetDetailPage.vue`

当前特征：

- 远程扫描通过 SSH 运行 shell/python 逻辑
- 扫描后生成 `DatasetScanRecord`、`DatasetPreviewItem`
- 远程 runner contract 已定义在 `docs/remote-dataset-contract.md`

结论：

- “是否已有数据集路径 / 扫描结果可复用”：有，而且复用价值很高
- 阶段2实验 spec 生成时，应直接引用：
  - 已登记 `Dataset`
  - 已注册 `DatasetAsset`
  - 最近一次扫描摘要 / 路径 / server 信息

### 4.5 数据集任务生命周期骨架

已实现：

- `DatasetIndexTask`
- `DatasetIndexTaskLog`
- task create / sync
- `building` / `completed` / `failed`
- 远程任务路径记录
- 错误信息与日志记录

关键位置：

- `backend/go/internal/service/dataset_service.go`
- `backend/go/internal/service/dataset_remote_runtime.go`
- `backend/go/internal/model/models.go`

结论：

- 这不是实验系统，但已经是阶段2非常重要的“任务状态机样板”
- `ExperimentRun` 可以复用它的设计思路：
  - 任务表
  - task log 表
  - start / sync
  - statusPath / resultPath / logPath
  - `responsePayload`

### 4.6 阶段1科研资产对象模型

已实现：

- `Paper`
- `PaperInsight`
- `Idea`
- `DatasetAsset`
- `Baseline`
- `ResultArchive`

关键位置：

- `backend/go/internal/model/research_asset_models.go`
- `backend/go/internal/service/dataset_asset_service.go`
- `backend/go/internal/service/baseline_service.go`
- `backend/go/internal/service/result_archive_service.go`
- `backend/go/migrations/005_stage1_research_assets.sql`
- `src/views/research/DatasetAssetPage.vue`
- `src/views/research/BaselinePage.vue`
- `src/views/research/ResultArchivePage.vue`

当前特征：

- `DatasetAsset` 已明确依附于 MRAG 的已扫描 `Dataset`
- `Baseline` 已有 `metric_schema_json`、`result_json`
- `ResultArchive` 已有 `metric_json`、`idea_id`、`baseline_id`、`server_id`
- 这些对象已具备 workspace 落盘能力

结论：

- 阶段2必须直接复用这些对象，不允许再造平行的 dataset/baseline/result 体系
- 阶段2实验的输入与输出应围绕这些对象展开

### 4.7 阶段0 Auto_v1 的 workflow / mock experiment 经验

已实现：

- phase0 workflow 状态推进
- `workflow_jobs`
- `experiments`
- `artifacts`
- experiment spec 生成
- 服务器选择
- 运行日志落盘
- 错误记录
- run 目录组织

关键位置：

- `Auto_v1/internal/workflow/phase0.go`
- `Auto_v1/internal/repository/server_repository.go`
- `Auto_v1/internal/repository/experiment_repository.go`
- `Auto_v1/python_agents/plan_experiment.py`
- `Auto_v1/python_agents/run_mock_experiment.py`
- `Auto_v1/migrations/001_init.sql`

当前可借鉴点：

- 状态机与 trace 记录
- `workspace/runs/<workflow_id>/...` 这种 run 级目录组织
- experiment spec 文件协议
- artifact 可审计写法
- 失败时释放 server busy 标记
- 调度规则与日志追加方式

结论：

- 阶段2不能把 `Auto_v1` 整套直接搬回 MRAG
- 但应该复用它已经验证过的“实验生命周期组织方式”

## 5. 特别检查结论

### 5.1 是否已有服务器心跳逻辑

结论：

- 只有“心跳字段”，没有“自动心跳机制”

已存在：

- `servers.last_heartbeat`
- `UpdateStatus()` 会在状态刷新时更新 `last_heartbeat`

未发现：

- 后台定时心跳任务
- server agent 主动上报
- 调度前自动健康轮询
- 心跳超时剔除逻辑

判断：

- 当前属于“手动刷新式状态更新时间”，不能视为完整服务器心跳系统

### 5.2 是否已有 GPU 空闲显存探测逻辑

结论：

- 有，且已能真实执行

已存在：

- `server_service.CheckGPU()`
- 基于 `nvidia-smi` 的远程探测
- 可用 GPU 数量与每张卡状态解析

缺口：

- 无周期探测
- 无探测历史
- 无调度期资源占用锁
- 当前可用判断规则较简单，尚未纳入实验级资源模型

### 5.3 是否已有远程命令执行逻辑

结论：

- 有

已存在：

- `SSHGateway.Exec()`
- real/mock 两套实现
- ssh config alias / password / proxy command 支持

缺口：

- 还没有实验执行专用 runner 协议在主系统中落地
- 没有实验作业 start / poll / cancel / retry 的上层封装

### 5.4 是否已有数据集路径 / 扫描结果可复用

结论：

- 有，而且应优先复用

已存在：

- `Dataset.path`
- `Dataset.serverId / serverName`
- `DatasetScanRecord`
- `DatasetPreviewItem`
- `DatasetAsset.existingDatasetRef`

阶段2价值：

- experiment spec 不需要重复扫描数据集
- 可直接引用已知路径、模态、文件规模、server 归属、index 状态

## 6. 可直接复用模块

以下模块建议“直接复用，不改主身份”：

### 6.1 必须直接复用

- 服务器表与服务器 API
- `SSHGateway`
- `ServerService.CheckGPU`
- `ServerService.ScanDatasets`
- `DatasetService` 的远程路径校验与扫描链
- `Dataset` / `DatasetScanRecord` / `DatasetPreviewItem`
- `DatasetAsset`
- `Baseline`
- `ResultArchive`
- `Idea`

### 6.2 可直接复用的页面与接口模式

- 服务器页的“手动触发检查 + 结果展示”交互模式
- 数据集详情页的“任务状态 + 日志 + 远程路径”展示模式
- 研究资产页的“对象详情 + supporting files”模式

## 7. 需要扩展而非重写的模块

### 7.1 服务器模块

应扩展：

- 在 `servers` 基础上增加心跳策略、调度标签、资源占用信息、最近调度时间

不应重写：

- server 表
- server CRUD
- ssh 连接方式

### 7.2 GPU 探测模块

应扩展：

- 周期采样
- 资源占用快照
- 调度前二次校验
- 更细粒度显存 / 进程信息

不应重写：

- 基于 SSH + `nvidia-smi` 的现有探测链

### 7.3 数据集链路

应扩展：

- 让 experiment spec 直接消费 dataset / dataset asset 元信息
- 在实验执行中引用 dataset index / preview / scan 摘要

不应重写：

- 数据集扫描器
- 数据集登记流程
- 远程数据集 contract

### 7.4 DatasetIndexTask 任务模型

应扩展：

- 抽象出 experiment run task 的共性

不应重写：

- 已有的 start / sync / log / statusPath / resultPath 设计思路

### 7.5 阶段1科研资产模型

应扩展：

- `ResultArchive` 从“手工归档”接入“实验完成后归档”
- `Baseline` 接入自动对比
- `Idea` / `DatasetAsset` 接入 experiment spec 生成

不应重写：

- `DatasetAsset` / `Baseline` / `ResultArchive` 表与页面主身份

### 7.6 Auto_v1 phase0 经验

应扩展式吸收：

- spec 文件协议
- 状态机与 trace
- run 目录与 artifact 组织
- server 调度释放逻辑

不应直接迁回：

- `workflow_jobs` 为主线中心
- 原 phase0 mock orchestrator
- 原 mock experiment Python 主链路

## 8. 阶段2缺失模块

以下是阶段2为了形成闭环仍需新建的模块：

### 8.1 领域模型与表

缺失：

- `experiments`
- `experiment_runs` 或等价 run 表
- `experiment_run_logs`
- `experiment_artifacts`
- `experiment_retry_records` 或恢复记录
- `experiment_comparisons`

说明：

- MRAG 当前历史 `experiments` 表已被迁移删除
- 阶段2需要重新以“实验执行对象”身份正式建模

### 8.2 experiment spec 生成服务

缺失：

- 基于 `Idea + DatasetAsset + Baseline + Dataset` 的 spec 生成器
- spec schema
- spec 校验器
- 统一训练模板映射

### 8.3 调度器

缺失：

- 按 server / GPU 可用性选择执行节点
- 资源占用锁
- 并发控制
- 排队策略
- 失败回收

### 8.4 真实实验执行链

缺失：

- 远程实验 runner contract
- 启动训练命令
- 轮询运行状态
- 停止 / 取消
- 恢复 / retry

说明：

- `REMOTE_RUNNER_ENTRYPOINT` 仍只是配置入口
- 当前主系统并没有 experiment handler / service / router / page

### 8.5 日志与审计

缺失：

- 运行日志采集
- stderr/stdout 摘要
- 关键事件日志
- 审计链路视图

### 8.6 结果对比

缺失：

- 与 baseline 自动对比
- 与历史 result archive 对比
- 指标对齐与 diff 计算
- 前端对比视图

### 8.7 前端实验页面

缺失：

- 实验列表页
- 实验详情页
- 运行状态页
- 日志摘要
- 对比结果展示

## 9. 不建议现在做的模块

以下内容不建议在当前阶段2盘点后立即推进：

- 阶段3/阶段4式自治 agent 编排
- 多 agent 自动科研 workflow
- 复杂推荐、知识图谱、自动论文综述
- 结果自动写论文
- 多租户权限系统
- 大规模队列 / 消息中间件化重构
- 为了实验而重做一套独立 server / dataset / result 系统

## 10. 复用优先级列表

阶段2建议按以下优先级复用现有代码继续开发：

### P0：必须优先复用

1. `MRAG` 的服务器管理与 `SSHGateway`
2. `MRAG` 的 GPU 探测链
3. `MRAG` 的远程数据集路径校验与扫描结果
4. 阶段1 `DatasetAsset`
5. 阶段1 `Baseline`
6. 阶段1 `ResultArchive`

原因：

- 这些模块已经是当前主系统真源
- 直接决定阶段2输入、调度基础和输出归档

### P1：应优先沿用设计

1. `DatasetIndexTask` 的任务表 / 日志 / start-sync 模式
2. `Auto_v1` 的 experiment spec 文件协议
3. `Auto_v1` 的 run 级 artifact / log 组织方式

原因：

- 它们已经验证了“可恢复、可审计、可查看”的组织方式
- 比从零发明实验 run 生命周期更稳

### P2：按需吸收

1. `Auto_v1` 的 server 选择规则
2. `Auto_v1` 的 workflow trace 事件结构
3. `Auto_v1` 的 mock runner 输入输出契约

原因：

- 可作为阶段2实现参考
- 但不应原样迁回成为主系统

## 11. 对阶段2设计的直接建议

基于本次盘点，阶段2应采用以下路线：

### 11.1 输入侧

实验创建输入应基于：

- `Idea`
- `DatasetAsset`
- `Baseline`
- 其背后的 `Dataset` 扫描信息

### 11.2 基础设施侧

调度与执行应复用：

- `servers`
- `SSHGateway`
- GPU 探测链
- 远程 runner contract 思路

### 11.3 生命周期侧

实验运行态应参考：

- `DatasetIndexTask` 的任务模型
- `Auto_v1` 的状态机 / artifact / run log 设计

### 11.4 输出侧

实验完成后应沉淀到：

- `ResultArchive`
- 并与 `Baseline` / 历史结果建立对比关系

## 12. 最终分类输出

### 12.1 已有模块

- 服务器管理
- SSH 远程连接与命令执行
- GPU 资源探测
- 远程数据集扫描
- 数据集扫描记录 / 预览 / 索引任务
- `DatasetAsset`
- `Baseline`
- `ResultArchive`
- `Idea`
- `Auto_v1` phase0 的 spec / workflow / artifact 经验

### 12.2 应复用模块

- `ServerService`
- `SSHGateway`
- `DatasetService` 远程路径与扫描链
- `Dataset` / `DatasetScanRecord`
- `DatasetAsset`
- `Baseline`
- `ResultArchive`
- `DatasetIndexTask` 模型

### 12.3 待扩展模块

- server 状态刷新 -> 自动心跳
- GPU 检查 -> 周期采样与调度输入
- dataset task 模型 -> experiment run 模型
- `ResultArchive` -> 自动归档入口
- `Baseline` -> 自动比较入口
- `Auto_v1` spec / trace / run 目录协议 -> MRAG 原生实验执行链

### 12.4 待新建模块

- `Experiment`
- `ExperimentRun`
- 调度器
- experiment spec 生成器
- 统一训练模板
- 远程实验 runner 协议
- 运行日志采集
- 失败恢复 / retry
- baseline / 历史结果对比
- 实验列表 / 详情 / 日志 / 对比前端页面

### 12.5 下一步建议

下一步最小动作建议是：

1. 先做“阶段2对象与状态机设计”，明确 `Experiment` / `ExperimentRun` / `ExperimentArtifact` / `ExperimentComparison` 的最小表结构
2. 设计 experiment spec schema，并明确它如何引用 `Idea` / `DatasetAsset` / `Baseline`
3. 以 MRAG 现有 `servers + ssh + gpu probe + dataset scan` 为底座，设计实验调度最小闭环
4. 明确实验运行日志、错误、重试、归档分别落到哪些表和 workspace 路径

在这之后，再进入阶段2的最小数据库迁移与 API 设计，而不是先写训练业务代码。
