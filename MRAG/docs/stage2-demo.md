# Stage2 Demo

## 目标

本演示用于展示阶段2“实验自动化期”的最小可运行闭环，不进入阶段3的智能规划或复杂科研决策。

## 演示前准备

- 启动 PostgreSQL 和 Go 后端
- 启动 Vue3 前端
- 确认 `workspace/` 可读写
- 推荐先运行：

```bash
bash ./scripts/validate_stage2.sh
```

## 推荐演示顺序

### 1. 查看服务器资源

页面：

- `/servers`

演示动作：

- 触发 heartbeat
- 触发 GPU snapshot
- 查看最近 heartbeat 列表
- 查看 GPU 快照列表和空闲显存

演示重点：

- 阶段2没有重写服务器系统
- 直接复用 MRAG 现有服务器、SSH、GPU 探测能力

### 2. 创建实验

页面：

- `/experiments`

演示动作：

- 新建一个 experiment
- 选择 `dataset asset`
- 可选选择 `idea`
- 可选选择 `baseline`
- 查看列表中的 `title / dataset / idea / status / priority`

演示重点：

- `Experiment` 是执行对象
- `DatasetAsset / Idea / Baseline` 继续复用阶段1资产模型

### 3. 生成 Experiment Spec

页面：

- `/experiments/:id`

演示动作：

- 点击 `Generate Spec`
- 查看 spec 内容
- 展示 `workspace/experiments/{experiment_id}/spec.json`

演示重点：

- spec 是统一、结构化、可审计的
- 当前为规则生成，不依赖复杂 agent

### 4. Queue 与 Schedule

页面：

- `/experiments/:id`

演示动作：

- 点击 `Queue`
- 点击 `Schedule`
- 查看 run 状态从 `queued` 到 `scheduled`
- 查看 assigned server 和 scheduler decision

演示重点：

- 调度器只读 heartbeat / GPU snapshot
- 不重新探测，不重复开发已有资源模块

### 5. Start Run 与日志查看

页面：

- `/experiments/:id`

演示动作：

- 点击 `Start Run`
- 查看状态从 `preparing -> running -> succeeded`
- 查看日志 tail
- 查看 `started_at / ended_at / assigned server / error message`

演示重点：

- 当前统一训练模板仍是最小版本
- 当前实际稳定模板是 `mock_train_template`
- 日志采集为最小闭环，不是复杂实时流式系统

### 6. 结果归档与结果对比

页面：

- `/experiments/:id/comparisons`

演示动作：

- 查看 baseline comparison
- 查看同 dataset asset 下历史 result archive comparison
- 查看 `summary_md`

演示重点：

- 成功 run 会联动阶段1 `result_archive`
- `result_comparison` 只做最小结构化对比
- 当前结果对比还不是自动科研决策器

### 7. 失败恢复与最小重试

页面：

- `/experiments/:id`

演示动作：

- 选择一个失败 run
- 查看 recovery 信息
- 触发 retry
- 查看新 run 重新进入 `queued`

演示重点：

- 当前恢复机制仍是最小重试版本
- 不是 checkpoint resume，不做跨服务器断点续跑

## 推荐展示的 workspace 产物

- `workspace/experiments/{experiment_id}/spec.json`
- `workspace/experiments/{experiment_id}/run_{n}/`
- `workspace/experiments/{experiment_id}/run_{n}/outputs/metrics.json`
- `workspace/experiments/{experiment_id}/run_{n}/outputs/result.md`
- `workspace/experiments/{experiment_id}/comparisons/`

## 演示结论

阶段2当前已经形成最小实验自动化闭环：

- 可创建 experiment
- 可生成 spec
- 可基于服务器资源调度
- 可启动统一模板 run
- 可记录日志、错误和结果
- 可失败重试
- 可回流 result archive 并生成 comparison

但阶段2仍有清晰边界：

- 不做复杂真实训练平台
- 不做复杂恢复机制
- 不做自动科研决策
