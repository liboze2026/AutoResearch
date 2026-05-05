# 阶段2工程结构接入说明

## 1. 文档目标

本文说明阶段2在当前 `MRAG + 阶段1` 项目上的工程结构接入方式。

本次接入只做目录和模块骨架，不实现完整业务逻辑，目标是为后续阶段2开发提供稳定落点，同时不破坏阶段1现有 API 和页面。

## 2. 结构接入原则

- 复用当前主后端目录：`backend/go/internal`
- 不新建平行 Go backend 主线
- 复用现有 `service/repository/handler/router/model` 分层
- Python 侧只补 runner/template 目录，不接入复杂逻辑
- workspace 侧先补 experiment 与 logs 目录，作为阶段2 supporting artifacts 入口

## 3. 本次新增目录

### 3.1 Go 后端阶段2骨架目录

说明：用户给出的建议路径为 `backend/internal/...`，结合当前仓库实际结构，本次接入统一落在现有主后端目录 `backend/go/internal/...`。

新增：

- `backend/go/internal/experiment`
- `backend/go/internal/experimentrun`
- `backend/go/internal/scheduler`
- `backend/go/internal/heartbeat`
- `backend/go/internal/gpuresource`
- `backend/go/internal/runner`
- `backend/go/internal/traintemplate`
- `backend/go/internal/resultcompare`
- `backend/go/internal/logcollector`
- `backend/go/internal/recovery`

### 3.2 Python 侧骨架目录

新增：

- `backend/python_templates`
- `backend/python_runners`

### 3.3 Workspace 目录

新增：

- `workspace/experiments`
- `workspace/logs`

## 4. 新增目录树

```text
MRAG/
  backend/
    go/
      internal/
        experiment/
          doc.go
        experimentrun/
          doc.go
        scheduler/
          doc.go
        heartbeat/
          doc.go
        gpuresource/
          doc.go
        runner/
          doc.go
        traintemplate/
          doc.go
        resultcompare/
          doc.go
        logcollector/
          doc.go
        recovery/
          doc.go
    python_templates/
      README.md
    python_runners/
      README.md
  workspace/
    experiments/
      README.md
    logs/
      README.md
```

## 5. 新增模块职责摘要

### 5.1 纯新增骨架模块

以下模块是新增目录，占位阶段2的明确责任边界：

- `experiment`
  - 实验主对象相关骨架
- `experimentrun`
  - 实验运行对象相关骨架
- `scheduler`
  - 调度决策骨架
- `heartbeat`
  - 服务器心跳骨架
- `gpuresource`
  - GPU 资源快照骨架
- `runner`
  - 远程执行 runner 合约骨架
- `traintemplate`
  - 训练模板映射骨架
- `resultcompare`
  - 结果比较骨架
- `logcollector`
  - 日志采集骨架
- `recovery`
  - 恢复与重试骨架

### 5.2 对已有模块的扩展预留

以下不是平行实现，而是为后续扩展现有模块预留落点：

- `scheduler`
  - 未来扩展现有 server / ssh / gpu probe 能力
- `heartbeat`
  - 未来扩展现有 server status refresh
- `gpuresource`
  - 未来扩展现有 GPU 检查输出
- `runner`
  - 未来复用现有 `SSHGateway`
- `resultcompare`
  - 未来复用现有 `Baseline` / `ResultArchive`
- `logcollector`
  - 未来为 `ExperimentRun` 采集事件日志和输出摘要
- `recovery`
  - 未来配合 run 状态机实现恢复和重试

## 6. 与现有模块的关系

### 6.1 直接复用的已有模块

后续阶段2实现时，以下模块继续保持主身份：

- `backend/go/internal/service/server_service.go`
- `backend/go/internal/service/ssh_gateway.go`
- `backend/go/internal/service/dataset_service.go`
- `backend/go/internal/service/dataset_remote_runtime.go`
- `backend/go/internal/service/dataset_asset_service.go`
- `backend/go/internal/service/baseline_service.go`
- `backend/go/internal/service/result_archive_service.go`
- `backend/go/internal/model/models.go`
- `backend/go/internal/model/research_asset_models.go`

### 6.2 本次未修改的稳定区域

本次没有改动：

- 阶段1已有 API 路由
- 阶段1已有页面
- 后端启动入口
- 现有数据库迁移
- 前端 API 模块

## 7. 为什么这样接入

这样设计的原因是：

- 当前 MRAG 的真实主后端在 `backend/go/internal`，直接在这里加阶段2骨架最稳
- 如果按字面另建 `backend/internal/...`，会形成第二套 Go 内部模块路径，后续很容易演变成平行系统
- 先用 `doc.go` 明确包边界，后续实现时可以渐进接入，不需要先写空逻辑或假逻辑
- Python 与 workspace 目录先就位，便于后续接 experiment spec、runner 和日志文件面

## 8. 当前接入结果

本次结构接入后，项目状态应保持：

- 阶段1后端仍可启动
- 阶段1前端页面不受影响
- 阶段2已有明确目录落点
- 后续实现可逐步接入，而不是一次性重构

## 9. 下一步建议

结构接入之后，下一步最小动作建议是：

1. 在现有 `model` 与 `repository` 层新增阶段2对象的最小结构定义
2. 先接 `Experiment / ExperimentRun / ExperimentSpec` 三个主对象
3. 再接 `SchedulerDecision / RunLog / ResultComparison`
4. 最后再把 heartbeat、gpu snapshot、runner、recovery 串起来

在此之前，不需要提前实现复杂训练逻辑。
