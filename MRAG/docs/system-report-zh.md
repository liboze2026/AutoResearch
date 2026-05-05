# MRAG 系统重构报告

## 1. 本次重构结论

本次重构已经将系统主线收敛为两条：

1. 数据集管理：围绕服务器目录扫描、路径校验、数据集登记、扫描摘要、索引任务同步。
2. 服务器管理：围绕服务器节点维护、SSH 连通性检查、空闲 GPU 检查、状态刷新、配置 JSON 管理。

已完成的核心删除与收口：

- 前端已删除 `/experiments`、`/results` 对应页面、路由、API 模块和概览依赖。
- Go 后端已删除 experiments/results 相关 handler、service、repository、router 注册。
- Python 实验服务目录已删除。
- Docker Compose 已去除 `python-service`。
- 数据库迁移新增 `003_remove_experiments_and_extend_servers.sql`：启动迁移时会 drop `experiments`、`experiment_configs`、`experiment_logs`、`experiment_results` 四张表，并扩展 `servers` 表字段。

## 2. 对需求 2 的检查结论

你的目标是：
“数据集管理的模式是扫描我设置的服务器的目录，检索数据集名称，大小，在前端做一定程度的展示。”

检查旧逻辑的结论：

- 旧版并不完全符合。
- 旧版主入口是“手动输入路径后导入”，不是“先扫描服务器目录，再选择候选数据集登记”。
- 旧版虽然有路径校验、扫描摘要、详情页，但流程中心仍是路径导入，不是服务器扫描发现。
- 旧版数据集页面还存在中文乱码，维护成本高。

本次调整后的状态：

- 数据集列表页新增“服务器扫描”入口。
- 可以选择服务器、设置扫描目录和深度。
- 后端新增 `/servers/:id/scan-datasets`，返回候选目录名称、路径、大小、文件数、目录数、最后修改时间和模态推断。
- 前端可从扫描结果直接“登记为数据集”。
- 登记前仍保留路径校验，避免脏数据进入系统。

所以现在的数据集主流程已经更接近你的目标，并且比旧版更规范。

## 3. 对需求 3 的检查结论

你的目标是：
“服务器管理应该在前端可操作，在前端可以设置 ip，端口，配置 config，在前端可以进行空闲显卡的检查，并且展示当前服务器状态。”

旧逻辑的结论：

- 旧版只支持查看服务器列表和做连接测试。
- 旧版不能在前端新增/编辑服务器。
- 旧版没有配置 JSON 编辑。
- 旧版没有 GPU 空闲检查接口。
- 旧版没有独立的状态刷新接口。

本次调整后的状态：

- 前端服务器页支持新增、编辑、删除服务器。
- 支持编辑 `host/ip`、`sshPort`、`username`、`authType`、`remoteRoot`、`taskWorkdir`、`config JSON`。
- 后端新增 `/servers/:id/check-gpu`。
- 后端新增 `/servers/:id/refresh-status`。
- 前端展示最近一次 SSH 探测结果、最近一次 GPU 检查结果、服务器在线状态、可用 GPU 数量。

因此需求 3 在当前仓库范围内已经完成主体落地。

## 4. 当前系统主逻辑

### 4.1 总体架构

- 前端：Vue 3 + Vite + Element Plus。
- 主后端：Go + Gin + PostgreSQL。
- 当前保留的主业务：数据集、服务器、系统设置、运行模式、总览统计。

### 4.2 数据集主流程

1. 在服务器管理中维护服务器信息。
2. 在数据集管理页选择服务器并扫描目录。
3. 前端展示候选目录列表。
4. 选择候选目录后登记为数据集。
5. 后端在创建数据集时执行路径校验和扫描摘要生成。
6. 数据集详情页展示扫描信息、预览项、索引任务。
7. 可发起索引构建，并同步索引任务状态。

### 4.3 服务器主流程

1. 前端新增或编辑服务器。
2. 服务器配置持久化到 `servers` 表。
3. 前端可触发 SSH 连通性测试。
4. 前端可触发 GPU 空闲检查。
5. 前端可触发服务器状态刷新。
6. 总览页和服务器页展示最新状态。

## 5. 当前仍需注意的点

- `docs/runtime-modes-and-acceptance.md` 仍保留旧实验系统说明，属于历史文档，和当前代码不完全一致。
- `system_settings` 里仍保留 `experiment_output_root` 字段；现在前端文案已改成“任务输出路径”，但数据库字段名还未重命名，以避免迁移成本扩大。
- `backend/go/migrations/001_init.sql` 仍包含历史 experiments/results 建表语句；真正生效的删除逻辑在 `003_remove_experiments_and_extend_servers.sql`。这保证了升级兼容，但不是“从第一条迁移开始就没有历史痕迹”的最彻底形态。
- 前端构建已通过，但打包后仍有较大的 chunk 警告，属于性能优化项，不是功能错误。

## 6. 文件功能总览

说明：以下按“当前项目核心文件”列出；不包含 `node_modules/`、`dist/`、`server.exe` 等依赖或产物目录。

### 6.1 根目录

- `package.json`：前端依赖与构建脚本定义。
- `package-lock.json`：前端依赖锁文件。
- `tsconfig.json`：前端 TypeScript 编译配置。
- `tsconfig.node.json`：Vite/Node 侧 TypeScript 配置。
- `vite.config.ts`：Vite 构建配置。
- `index.html`：前端挂载入口 HTML。
- `.env.example`：前端环境变量示例。
- `docker-compose.yml`：本地开发/联调用的 PostgreSQL + Go 后端编排文件。

### 6.2 docs

- `docs/remote-dataset-contract.md`：远程数据集扫描、校验、索引任务的约定文档。
- `docs/runtime-modes-and-acceptance.md`：运行模式与验收说明，内容部分仍带历史实验系统描述。
- `docs/system-report-zh.md`：本次重构后新增的中文系统说明报告。

### 6.3 前端入口与全局

- `src/main.ts`：创建 Vue 应用，挂载 Pinia、Router、Element Plus。
- `src/App.vue`：顶层路由出口。
- `src/env.d.ts`：Vite 环境变量类型声明。
- `src/styles.css`：全局样式变量和页面基础样式。

### 6.4 前端布局/状态/组件

- `src/layouts/MainLayout.vue`：系统主布局，包含侧边栏、运行模式横幅和页面内容出口。
- `src/router/index.ts`：前端路由定义，仅保留总览、数据集、服务器、设置四块。
- `src/store/ui.ts`：侧边栏折叠等 UI 状态管理。
- `src/components/StatCard.vue`：总览统计卡片组件。
- `src/components/StatusTag.vue`：统一状态标签组件。
- `src/utils/mock.ts`：旧的简单 mock 工具，目前基本不参与主流程。
- `src/utils/format.ts`：日期和字节大小的公共格式化工具。

### 6.5 前端 API

- `src/api/http.ts`：统一 HTTP 请求封装，处理 API Envelope。
- `src/api/index.ts`：前端 API 模块聚合出口。
- `src/api/modules/datasets.ts`：数据集相关接口，包括列表、详情、路径校验、登记、索引、服务器扫描。
- `src/api/modules/servers.ts`：服务器相关接口，包括 CRUD、连接测试、状态刷新、GPU 检查。
- `src/api/modules/system.ts`：总览统计、运行模式、设置相关接口。

### 6.6 前端类型

- `src/types/domain.ts`：前后端共享的前端领域模型定义，已删除 experiments/results 类型，补充了服务器配置、GPU 检查、服务器扫描候选等模型。

### 6.7 前端页面

- `src/views/overview/OverviewPage.vue`：系统首页，展示数据集、扫描、索引、服务器在线情况和趋势图。
- `src/views/datasets/DatasetListPage.vue`：数据集列表页；新增服务器扫描入口、候选目录展示和登记流程。
- `src/views/datasets/DatasetDetailPage.vue`：数据集详情页，展示扫描摘要、预览项、索引任务和历史记录。
- `src/views/servers/ServerPage.vue`：服务器管理页，支持新增/编辑/删除、状态检查、SSH 连接测试、GPU 检查。
- `src/views/settings/SettingsPage.vue`：系统设置页，展示运行模式并维护路径、默认模型和展示偏好。

### 6.8 Go 后端入口与配置

- `backend/README.md`：后端说明文档。
- `backend/go/go.mod`：Go 模块依赖定义。
- `backend/go/go.sum`：Go 依赖锁文件。
- `backend/go/.env.example`：Go 后端环境变量示例。
- `backend/go/Dockerfile`：Go 后端镜像构建文件。
- `backend/go/cmd/server/main.go`：Go 服务启动入口，初始化数据库、迁移、仓储、服务和路由。
- `backend/go/internal/config/config.go`：读取 Go 后端环境变量配置。

### 6.9 Go 后端模型

- `backend/go/internal/model/models.go`：后端领域模型与 API DTO，已移除 experiments/results 模型，补充服务器扩展结构。

### 6.10 Go 后端 handler

- `backend/go/internal/handler/dataset_handler.go`：数据集 HTTP 接口层。
- `backend/go/internal/handler/server_handler.go`：服务器 HTTP 接口层，包含新增的刷新状态、GPU 检查、扫描目录接口。
- `backend/go/internal/handler/settings_handler.go`：系统设置接口层。
- `backend/go/internal/handler/overview_handler.go`：总览统计接口层。
- `backend/go/internal/handler/runtime_handler.go`：运行模式接口层。

### 6.11 Go 后端 router 与基础工具

- `backend/go/internal/router/router.go`：Gin 路由注册。
- `backend/go/internal/pkg/db/postgres.go`：PostgreSQL 连接与迁移执行。
- `backend/go/internal/pkg/httpx/bind.go`：请求绑定工具。
- `backend/go/internal/pkg/httpx/id.go`：统一 ID 生成工具。
- `backend/go/internal/pkg/httpx/response.go`：统一响应封装。
- `backend/go/internal/pkg/httpx/time.go`：时间相关工具。

### 6.12 Go 后端 repository

- `backend/go/internal/repository/dataset_repository.go`：数据集、扫描记录、预览项、索引任务的数据库读写。
- `backend/go/internal/repository/server_repository.go`：服务器节点数据库读写，已支持 `config`、状态消息、GPU 信息字段。
- `backend/go/internal/repository/settings_repository.go`：系统设置读写。
- `backend/go/internal/repository/overview_repository.go`：总览统计聚合查询，现聚焦数据集与服务器维度。

### 6.13 Go 后端 service

- `backend/go/internal/service/dataset_service.go`：数据集主业务服务，负责校验、扫描、登记、索引任务、状态同步。
- `backend/go/internal/service/dataset_adapters.go`：数据集扫描/索引适配器定义与解析器。
- `backend/go/internal/service/dataset_runtime.go`：数据集运行时公共结构和工具函数。
- `backend/go/internal/service/dataset_local_runtime.go`：本地目录扫描与本地索引任务模拟实现。
- `backend/go/internal/service/dataset_remote_runtime.go`：远程目录扫描、校验、索引任务的 SSH 实现。
- `backend/go/internal/service/server_service.go`：服务器业务服务，负责 CRUD、SSH 探测、GPU 检查、目录扫描。
- `backend/go/internal/service/ssh_gateway.go`：SSH 网关，包含 real/mock 两种执行方式。
- `backend/go/internal/service/overview_service.go`：总览统计服务。
- `backend/go/internal/service/overview_stats_adapter.go`：总览统计的 real/mock 适配器。
- `backend/go/internal/service/runtime_profile_service.go`：运行模式说明生成。
- `backend/go/internal/service/settings_service.go`：系统设置服务。
- `backend/go/internal/service/helpers.go`：服务层通用字符串辅助函数。

### 6.14 数据库迁移

- `backend/go/migrations/001_init.sql`：历史初始化迁移，包含旧实验表定义。
- `backend/go/migrations/002_dataset_realization.sql`：数据集扫描、预览、索引任务等落地迁移。
- `backend/go/migrations/003_remove_experiments_and_extend_servers.sql`：本次新增迁移，删除实验/结果表，并扩展服务器字段。

## 7. 建议的下一步

如果你要继续精简到更彻底的“纯数据集/服务器平台”形态，下一步最值得做的是：

1. 重写 `docs/runtime-modes-and-acceptance.md`，清除历史实验说明。
2. 在后续迁移中把 `experiment_output_root` 正式重命名为更中性的 `task_output_root`。
3. 为服务器扫描和 GPU 检查补一组后端单元测试/集成测试。
4. 按页面拆分 ECharts 和 Element Plus 相关 chunk，继续减小前端首屏包体积。
