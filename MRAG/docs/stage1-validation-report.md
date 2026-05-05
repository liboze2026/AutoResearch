# Stage1 Validation Report

## 目标

本报告用于阶段1交付验收，范围只包括阶段1科研资产管理闭环，不包含阶段2自动实验、调度增强或 agent 自治。

## 验收范围

后端与前端应能共同完成以下最小闭环：

- 论文导入、解析、创新点抽取。
- Idea 池管理，包括手工创建与从论文生成。
- 将 MRAG 已扫描数据集注册为 dataset asset。
- 为 dataset asset 创建 baseline。
- 为 dataset asset 创建 result archive。
- 关键页面可访问并执行最小操作。

## 已验证能力

### 后端

- Go backend 可编译：`go build ./...`
- 阶段1服务测试已覆盖：
  - `paper_service_test.go`
  - `idea_service_test.go`
  - `dataset_asset_service_test.go`
  - `baseline_service_test.go`
  - `result_archive_service_test.go`
- 阶段1数据库迁移已补齐到：
  - `008_stage1_dataset_asset_fields.sql`
  - `009_stage1_baseline_source_type.sql`
  - `010_stage1_result_archive_note.sql`

### 前端

- Vue3 前端可编译：`npm run build`
- 前端基础测试方式：
  - `npm run typecheck`
  - `npm run build`
- 关键科研资产页面已接入现有路由与布局。

### API 闭环

已补齐并联调通过的阶段1接口包括：

- Papers
  - `POST /api/papers/import`
  - `GET /api/papers`
  - `GET /api/papers/:id`
  - `POST /api/papers/:id/parse`
  - `POST /api/papers/:id/extract-insights`
  - `GET /api/papers/:id/insights`
- Ideas
  - `GET /api/ideas`
  - `POST /api/ideas`
  - `GET /api/ideas/:id`
  - `PATCH /api/ideas/:id`
  - `POST /api/ideas/generate-from-paper/:paperId`
- Dataset Assets
  - `GET /api/dataset-assets`
  - `POST /api/dataset-assets/register-from-scan`
  - `POST /api/dataset-assets`
  - `GET /api/dataset-assets/:id`
- Baselines
  - `GET /api/baselines`
  - `POST /api/baselines`
  - `GET /api/baselines/:id`
  - `PATCH /api/baselines/:id`
- Result Archives
  - `GET /api/result-archives`
  - `POST /api/result-archives`
  - `GET /api/result-archives/:id`
  - `PATCH /api/result-archives/:id`

## 关键验证点

### 论文资产

- 能导入 `workspace/papers/incoming` 中已有论文文件。
- 导入后会触发最小 mock 解析。
- 解析后可继续执行 insight extraction。

### Idea 池

- 能手工创建 idea。
- 能从论文 insight 生成 2-3 个 deterministic mock idea。
- 能在 workspace 中看到对应 idea 目录与元数据。

### 数据集资产

- 不重写 MRAG 扫描逻辑。
- 使用既有 `datasets` / `dataset_scan_records` 作为事实来源。
- 通过 `dataset_asset_sources` 建立关联。

### Baseline

- baseline 必须归属于一个 dataset asset。
- 能保存 metric schema、result json 和 note。
- 当前支持 `manual / result_archive / mixed` 来源类型。

### Result Archive

- result archive 是归档对象，不是实验执行对象。
- 支持 `dataset_asset_id` 和可选 `idea_id`。
- 能保存 `result.md`、`metrics.json`、`note.md` 和附件。

## 阻碍项修复记录

阶段1验收收尾时已修复：

- `dataset_repository.go` 中 `GetScanRecordByID` 接入时的语法错误。
- 前端 HTTP client 不支持 `PATCH` 与 form 上传的问题。
- `dataset_assets` 缺少 `local_or_remote_path / readme / loader / schema` 字段。
- `baselines` 缺少 `source_type` 字段。
- `result_archives` 缺少 `note_md` 字段。
- 前端未接入科研资产路由与页面的问题。

## 已知限制

- 论文解析与 insight extraction 仍为可控 mock，不是复杂 NLP。
- 结果归档当前前端用文本方式登记附件，不是完整 multipart 上传。
- 前端页面以列表、详情抽屉、简单表单为主，没有复杂筛选、分页和批量操作。
- Docker 镜像重建可能受本地镜像源配置影响，验收脚本优先复用已有 compose 服务。

## 结论

阶段1的最小科研资产管理闭环已具备验收条件，可用于：

- 论文资产管理
- innovation insight 管理
- idea 池管理
- dataset asset 管理
- baseline 管理
- result archive 管理

不建议在本阶段继续扩展自动实验、资源调度增强或 agent 自治能力。