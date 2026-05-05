# MRAG Backend

`backend/` 是 `MRAG` 的后端工作区，负责 Go 控制面、Python agent/runtime、实验 runner 占位目录，以及与模板和脚本相关的后端入口。

## 目录说明

- `backend/go`
  - 主后端服务。
  - 负责 API、数据库访问、阶段2实验生命周期复用、阶段3受控 agent 控制面。
- `backend/python_agents`
  - Python agent 相关代码。
  - 其中 `runtime/` 是阶段3统一受控 runtime。
- `backend/python_runners`
  - 阶段2实验 runner 预留目录。
- `backend/python_templates`
  - 训练或执行模板目录。
- `backend/scripts`
  - 后端辅助脚本目录。

## 推荐入口

### Go 后端

```bash
cd backend/go
go run ./cmd/server
```

### Python agent runtime

重点目录：

- `backend/python_agents/runtime/contract.py`
- `backend/python_agents/runtime/executors.py`
- `backend/python_agents/runtime/runner.py`
- `backend/python_agents/runtime/schema_registry.py`

## 关键环境变量

### 基础后端配置

- `APP_PORT`
- `POSTGRES_DSN`
- `PYTHON_EXEC`
- `PYTHON_AGENTS_DIR`
- `PYTHON_TEMPLATES_DIR`
- `WORKSPACE_ROOT`

### 远程执行与资源探测

- `SSH_BINARY`
- `SSH_CLIENT_MODE`
- `SSH_DIAL_TIMEOUT_SEC`
- `SSH_COMMAND_TIMEOUT_SEC`
- `REMOTE_EXECUTION_MODE`
- `REMOTE_WORK_ROOT`
- `REMOTE_RUNNER_ENTRYPOINT`
- `REMOTE_DATASET_RUNNER_ENTRYPOINT`
- `DATASET_SCAN_MODE`
- `DATASET_INDEX_MODE`
- `OVERVIEW_STATS_MODE`

### 阶段3 agent runtime

- `AGENT_API_ENABLED`
- `AGENT_API_ALLOW_LIVE_EXECUTION`
- `AGENT_API_BASE_URL`
- `AGENT_API_KEY`
- `AGENT_RUNTIME_CONFIG_FILE`
- `CODEX_CLI_BIN`
- `CODEX_CLI_ARGS_JSON`
- `CODEX_CLI_TIMEOUT_SECONDS`
- `CODEX_CLI_OUTPUT_MODE`

## 本地开发

### 方式一：直接本地启动

```bash
cd backend/go
go run ./cmd/server
```

### 方式二：使用 Docker Compose

仓库根目录：

```bash
docker compose up -d postgres go-backend
```

默认映射：

- PostgreSQL: `localhost:15432`
- Go API: `localhost:8080`

## 模式说明

- `real`
  - 用于真实 SSH、真实数据集扫描、真实资源探测。
- `mock`
  - 用于本地开发、最小联调、自动验收和失败回退。
- `codex_cli`
  - 属于 Python runtime 的执行模式之一。
  - 当前最小测试优先验证的是 `codex_cli -> mock fallback` 的受控路径。

## 当前边界

- 当前仍以受控智能体系统为目标，不是完全自治科研系统。
- 当前真实模型调用和真实抓取能力只保留受控扩展点。
- 当前 `Coding` 仍受统一训练模板约束。
- 当前 `Writer / Picture` 仍是最小版本。

## 相关文档

- [主 README](../README.md)
- [阶段3运行时说明](./python_agents/runtime/README.md)
- [阶段3仓库维护计划](../docs/repo-maintenance-plan.md)
