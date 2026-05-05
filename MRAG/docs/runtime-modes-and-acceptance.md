Runtime Modes And Acceptance
运行模式与验收说明

This document explains how the platform switches between `mock` and `real`, what each mode means today, and which page flows are already closed-loop. 
本文说明平台如何在 `mock` 与 `real` 模式之间切换、各模式当前的含义，以及哪些页面流程已经形成闭环。

## 1. Unified Runtime Switches

## 1. 统一运行时开关

The platform now uses one environment variable per capability:
平台现在为每项能力使用一个独立的环境变量：

* `REMOTE_EXECUTION_MODE=mock|real`

* `REMOTE_EXECUTION_MODE=mock|real`（远程执行模式）

* `DATASET_SCAN_MODE=mock|real`

* `DATASET_SCAN_MODE=mock|real`（数据集扫描模式）

* `DATASET_INDEX_MODE=mock|real`

* `DATASET_INDEX_MODE=mock|real`（数据集索引模式）

* `OVERVIEW_STATS_MODE=mock|real`

* `OVERVIEW_STATS_MODE=mock|real`（总览统计模式）

Supplementary switch:
补充开关：

* `SSH_CLIENT_MODE=mock|real`
* `SSH_CLIENT_MODE=mock|real`

`SSH_CLIENT_MODE` controls server connection testing and SSH command diagnostics.
`SSH_CLIENT_MODE` 控制服务器连接测试和 SSH 命令诊断行为。

## 2. Quick Start Presets

## 2. 快速启动配置

### 2.1 Full Mock Startup

### 2.1 全 Mock 启动

Use when you want a stable demo without depending on local directories or remote servers.
用于无需依赖本地目录或远程服务器的稳定演示场景。

```env
SSH_CLIENT_MODE=mock
REMOTE_EXECUTION_MODE=mock
DATASET_SCAN_MODE=mock
DATASET_INDEX_MODE=mock
OVERVIEW_STATS_MODE=mock
```

Result:
结果：

* server connection tests return demo results

* 服务器连接测试返回演示结果

* remote experiments use the mock remote adapter

* 远程实验使用 mock 远程适配器

* dataset validation and scan return demo summaries

* 数据集校验和扫描返回演示摘要

* dataset index tasks use demo state transitions

* 数据集索引任务使用演示状态流转

* overview uses backend demo stats

* 总览页面使用后端模拟统计数据

### 2.2 Full Real Startup

### 2.2 全 Real 启动

Use when you want the platform to run real mainline chains.
用于运行真实业务主链路。

```env
SSH_CLIENT_MODE=real
REMOTE_EXECUTION_MODE=real
DATASET_SCAN_MODE=real
DATASET_INDEX_MODE=real
OVERVIEW_STATS_MODE=real
```

Result:
结果：

* server connection tests use the system `ssh` client

* 服务器连接测试使用系统 `ssh` 客户端

* remote experiments are submitted over SSH

* 远程实验通过 SSH 提交执行

* local datasets are scanned by the Go backend

* 本地数据集由 Go 后端扫描

* remote datasets are scanned over SSH

* 远程数据集通过 SSH 扫描

* index tasks create real task records and invoke local/remote adapters

* 索引任务会创建真实任务记录并调用本地/远程适配器

* overview reads database aggregates

* 总览页面读取数据库聚合数据

### 2.3 Mixed Startup Examples

### 2.3 混合启动示例

#### Only overview uses real stats

#### 仅总览使用真实数据

```env
REMOTE_EXECUTION_MODE=mock
DATASET_SCAN_MODE=mock
DATASET_INDEX_MODE=mock
OVERVIEW_STATS_MODE=real
```

#### Only datasets use real chains

#### 仅数据集使用真实链路

```env
REMOTE_EXECUTION_MODE=mock
DATASET_SCAN_MODE=real
DATASET_INDEX_MODE=real
OVERVIEW_STATS_MODE=mock
```

#### Remote experiments real, datasets still demo

#### 远程实验真实，数据集仍为演示

```env
REMOTE_EXECUTION_MODE=real
DATASET_SCAN_MODE=mock
DATASET_INDEX_MODE=mock
OVERVIEW_STATS_MODE=real
SSH_CLIENT_MODE=real
```

## 3. Required Environment Variables

## 3. 必需环境变量

Core variables:
核心变量：

* `APP_PORT`

* `APP_PORT`

* `POSTGRES_DSN`

* `POSTGRES_DSN`

* `PYTHON_SERVICE_URL`

* `PYTHON_SERVICE_URL`

* `SSH_BINARY`

* `SSH_BINARY`

* `SSH_CLIENT_MODE`

* `SSH_CLIENT_MODE`

* `SSH_DIAL_TIMEOUT_SEC`

* `SSH_DIAL_TIMEOUT_SEC`

* `SSH_COMMAND_TIMEOUT_SEC`

* `SSH_COMMAND_TIMEOUT_SEC`

* `REMOTE_EXECUTION_MODE`

* `REMOTE_EXECUTION_MODE`

* `REMOTE_WORK_ROOT`

* `REMOTE_WORK_ROOT`

* `REMOTE_RUNNER_ENTRYPOINT`

* `REMOTE_RUNNER_ENTRYPOINT`

* `REMOTE_DATASET_RUNNER_ENTRYPOINT`

* `REMOTE_DATASET_RUNNER_ENTRYPOINT`

* `DATASET_SCAN_MODE`

* `DATASET_SCAN_MODE`

* `DATASET_INDEX_MODE`

* `DATASET_INDEX_MODE`

* `DATASET_PREVIEW_LIMIT`

* `DATASET_PREVIEW_LIMIT`

* `OVERVIEW_STATS_MODE`

* `OVERVIEW_STATS_MODE`

* `OVERVIEW_TREND_DAYS`

* `OVERVIEW_TREND_DAYS`

Reference example file:
参考示例文件：

* `backend/go/.env.example`
* `backend/go/.env.example`

## 4. Remote Server Requirements

## 4. 远程服务器要求

The remote server side should satisfy:
远程服务器需满足：

* SSH access from the local machine

* 本地机器可以通过 SSH 访问

* preferred support for `.ssh/config` alias-based login

* 建议支持 `.ssh/config` 别名登录

* working root: `/home/bzli/lbz`

* 工作目录：`/home/bzli/lbz`

* optional remote experiment runner: `./bin/mrag-remote-runner`

* 可选远程实验执行器

* optional remote dataset runner: `./bin/mrag-dataset-runner`

* 可选远程数据集执行器

Preferred server record setup:
推荐服务器配置：

* `authType=ssh_config`

* `authType=ssh_config`

* `host=<your SSH Host alias>`

* `host=<你的 SSH 别名>`

This lets the system `ssh` client reuse configuration.
这样系统 `ssh` 可复用配置。

## 5. What Is Real Today

## 5. 当前已实现的真实能力

### 5.1 Already Real Orchestration

### 5.1 已真实打通的流程

* frontend pages now use real backend APIs

* 前端已使用真实后端 API

* server connection test can perform real SSH login verification

* 支持真实 SSH 登录验证

* `runMode=remote` no longer falls back

* 远程模式不再回退本地

* dataset import validates real paths

* 数据集导入校验真实路径

* scan reads real file trees

* 扫描真实文件系统

* results persisted in PostgreSQL

* 结果写入 PostgreSQL

### 5.2 Real Chain, Placeholder Business Logic

### 5.2 真实链路 + 占位算法

* remote experiment still depends on Python implementation

* 远程实验仍依赖 Python 实现

* dataset index build needs remote runner

* 数据集索引依赖远程执行器

## 6. What Is Still Mock By Design

## 6. 仍为 Mock 的部分

When mode is `mock`:
当模式为 `mock` 时：

* experiment responses

* 实验响应为模拟

* dataset scan summaries

* 数据扫描为模拟

* index state transitions

* 索引状态为模拟

* overview statistics

* 总览数据为模拟

## 7. Mock Cleanup Summary

## 7. Mock 清理总结

### 7.1 Mock Kept

### 7.1 保留的 Mock

* SSH mock gateway

* SSH 模拟网关

* mock remote adapter

* 模拟远程适配器

### 7.2 Mock Removed

### 7.2 已移除

* old frontend mock files

* 前端旧 mock 文件

* fake overview stats

* 假统计数据

## 8. Acceptance Checklist

## 8. 验收清单

### 8.1 Overview Page

### 8.1 总览页

Closed loop:
闭环：

* stats from backend
* 数据来自后端

Remaining placeholder:
剩余占位：

* mock numbers are demo
* mock 数据为演示

### 8.2 Dataset Page

### 8.2 数据集页

Closed loop:
闭环：

* import → scan → preview → index
* 导入 → 扫描 → 预览 → 索引

Remaining placeholder:
占位：

* index build still placeholder
* 索引构建仍为占位

### 8.3 Experiment Page

### 8.3 实验页

Closed loop:
闭环：

* create → start → stop → sync
* 创建 → 启动 → 停止 → 同步

Remaining placeholder:
占位：

* algorithm logic still external
* 算法仍依赖外部实现

### 8.4 Server Page

### 8.4 服务器页

Closed loop:
闭环：

* test → update status
* 测试 → 更新状态

### 8.5 Settings Page

### 8.5 设置页

Closed loop:
闭环：

* settings stored in DB
* 设置写入数据库

## 9. Verification Steps

## 9. 验证步骤

1. Start backend

2. 启动后端

3. Check header mode

4. 检查模式

5. Verify settings

6. 验证设置

7. Check overview

8. 查看总览

9. Test dataset flow

10. 测试数据集流程

11. Build index

12. 构建索引

13. Run experiments

14. 运行实验

15. Test server connection

16. 测试服务器连接

## 10. Related Contract Docs

## 10. 相关文档

* `docs/remote-python-contract.md`

* `docs/remote-python-contract.md`

* `docs/remote-dataset-contract.md`

* `docs/remote-dataset-contract.md`

