# 阶段1规格文档

## 1. 文档目的

本文档定义 MRAG 阶段1“科研资产管理期”的业务范围、系统边界、核心对象、模块划分、API 范围与页面范围。

阶段1以 MRAG 为主系统，在复用现有 SSH、GPU、服务器管理、数据集扫描能力的基础上，补齐论文、创新点、idea、科研数据集资产、baseline、结果归档的最小闭环。

本文档只定义规格，不进入业务代码实现。

## 2. 阶段1目标

阶段1目标如下：

- 在 MRAG 内构建最小可用的科研资产管理系统。
- 支持论文导入、论文解析、创新点抽取与 idea 池管理。
- 支持将 MRAG 已扫描和登记的数据集提升为科研数据集资产。
- 支持为科研数据集资产登记 baseline。
- 支持对实验输出、结论、指标、附件进行结果归档。
- 在前端提供最小页面，用于查看、创建、关联和验证上述科研对象。
- 所有新增功能必须挂接到 MRAG 当前目录结构、后端分层结构和前端页面结构中。

## 3. 非目标

以下内容不属于阶段1：

- 自动实验调度
- 真实训练任务下发
- GPU 作业编排
- 多 agent 自治
- 完整 workflow 编排系统
- 恢复 Auto_v1 的 phase0 orchestrator 主线
- 自动生成实验计划并自动执行
- 结果自动总结代理
- 多用户、权限、审批流
- 知识图谱、向量索引、推荐排序
- 重构 MRAG 为新架构或新系统

## 4. 阶段1范围说明

阶段1是“科研对象系统建设”阶段，而不是“自动科研执行”阶段。

因此：

- `ResultArchive` 先作为归档对象存在，用于记录已经发生的实验、分析或评测结果。
- `ResultArchive` 不代表系统已经具备自动实验能力。
- `DatasetAsset` 不是替代 MRAG 现有 `Dataset`，而是在现有数据集扫描与登记之上叠加科研资产语义。
- 论文解析、创新点抽取、idea 生成可以通过局部脚本或局部服务触发，但不形成统一自治工作流。

## 5. 核心科研对象

阶段1至少包含以下对象：

- `Paper`
- `PaperFile`
- `Insight`
- `Idea`
- `DatasetAsset`
- `Baseline`
- `ResultArchive`

对象关系详见 [research-asset-model.md](/D:/1/MRAG/docs/research-asset-model.md)。

## 6. 对象关系总览

核心关系如下：

- 一个 `Paper` 可以关联一个或多个 `PaperFile`。
- 一个 `Paper` 可以产生零个或多个 `Insight`。
- 一个 `Idea` 可以来源于一个或多个 `Insight`，也可以由人工创建。
- 一个 `DatasetAsset` 必须关联 MRAG 已存在的一个 `Dataset` 记录。
- 一个 `DatasetAsset` 可以拥有零个或多个 `Baseline`。
- 一个 `ResultArchive` 可以关联零个或一个 `Idea`。
- 一个 `ResultArchive` 必须关联一个 `DatasetAsset`。
- 一个 `ResultArchive` 可以关联零个或一个 `Baseline`。
- 一个 `ResultArchive` 可以关联零个或多个附件文件或归档产物。

## 7. 前后端模块划分

### 7.1 后端模块划分

后端继续沿用 MRAG 当前分层：

- `internal/model`
  - 定义阶段1新增对象、请求 DTO、响应 DTO
- `internal/repository`
  - 定义 `papers`、`paper_files`、`insights`、`ideas`、`dataset_assets`、`baselines`、`result_archives` 等持久化访问
- `internal/service`
  - 实现论文导入、论文解析、创新点抽取、idea 管理、科研数据集资产注册、baseline 管理、结果归档
- `internal/handler`
  - 暴露 HTTP API
- `internal/router`
  - 注册阶段1新增路由

复用模块：

- `server_service.go`
- `ssh_gateway.go`
- `dataset_service.go`
- `dataset_remote_runtime.go`
- 现有数据集、服务器、总览相关 router/handler/repository

### 7.2 前端模块划分

前端继续沿用 MRAG 当前结构：

- `src/api/modules`
  - 新增 `papers.ts`、`ideas.ts`、`baselines.ts`、`results.ts`
- `src/views`
  - 新增论文、idea、baseline、结果归档页面
  - 扩展现有数据集页面
- `src/types/domain.ts`
  - 扩展阶段1对象类型
- `src/router/index.ts`
  - 新增阶段1页面路由

## 8. API 范围

阶段1 API 以“对象管理 + 最小处理动作”为主。

### 8.1 Paper API

建议范围：

- `GET /api/v1/papers`
- `POST /api/v1/papers/import`
- `GET /api/v1/papers/:id`
- `POST /api/v1/papers/:id/parse`
- `GET /api/v1/papers/:id/files`

### 8.2 Insight API

建议范围：

- `GET /api/v1/papers/:id/insights`
- `POST /api/v1/papers/:id/extract-insights`
- `POST /api/v1/insights/:id/promote-to-idea`

### 8.3 Idea API

建议范围：

- `GET /api/v1/ideas`
- `POST /api/v1/ideas`
- `GET /api/v1/ideas/:id`
- `PUT /api/v1/ideas/:id`
- `POST /api/v1/ideas/generate`

### 8.4 DatasetAsset API

建议范围：

- `GET /api/v1/dataset-assets`
- `POST /api/v1/dataset-assets`
- `GET /api/v1/dataset-assets/:id`
- `PUT /api/v1/dataset-assets/:id`

说明：

- `DatasetAsset` 创建时必须引用 MRAG 已有 `datasets.id`。
- 不允许绕过 MRAG 扫描链路直接创建“脱离数据集表”的科研数据集资产。

### 8.5 Baseline API

建议范围：

- `GET /api/v1/dataset-assets/:id/baselines`
- `POST /api/v1/dataset-assets/:id/baselines`
- `GET /api/v1/baselines/:id`
- `PUT /api/v1/baselines/:id`

### 8.6 ResultArchive API

建议范围：

- `GET /api/v1/result-archives`
- `POST /api/v1/result-archives`
- `GET /api/v1/result-archives/:id`
- `PUT /api/v1/result-archives/:id`

说明：

- `ResultArchive` 是归档对象，不是实验任务对象。
- 创建 `ResultArchive` 不触发训练、调度、远程执行或 workflow。
- `ResultArchive` 只记录已发生结果的元数据、结论、指标、关联对象与附件路径。

## 9. 页面范围

阶段1前端至少包括以下页面：

- 论文列表页
- 论文详情页
- idea 池页
- 数据集页
- baseline 页
- 结果归档页

### 9.1 论文列表页

用途：

- 查看论文列表
- 发起论文导入
- 查看解析状态
- 进入论文详情页

### 9.2 论文详情页

用途：

- 查看论文基础信息与关联文件
- 查看解析结果
- 查看已抽取的 insights
- 基于 insight 生成或创建 idea

### 9.3 idea 池页

用途：

- 查看 idea 列表
- 管理 idea 状态
- 查看来源论文/insight
- 进入后续结果归档关联

### 9.4 数据集页

用途：

- 继续承载 MRAG 现有数据集展示能力
- 明确某个数据集是否已注册为 `DatasetAsset`
- 查看与该数据集关联的 baseline 与结果归档

### 9.5 baseline 页

用途：

- 查看某个科研数据集资产下的 baseline 列表
- 新建与编辑 baseline
- 查看关键指标、说明和附件

### 9.6 结果归档页

用途：

- 查看结果归档列表
- 创建结果归档
- 关联数据集资产、baseline、idea
- 查看指标、摘要、路径与附件

## 10. DatasetAsset 与 MRAG Dataset 的关系

阶段1必须明确以下约束：

- `DatasetAsset` 是科研资产层对象。
- MRAG 当前 `datasets` 表是数据源发现和扫描登记层对象。
- 一个 `DatasetAsset` 必须引用一个已经存在于 MRAG `datasets` 表中的数据集。
- `DatasetAsset` 不负责扫描服务器目录，也不负责替代现有 dataset preview / scan / index 能力。
- 数据集扫描、路径校验、索引构建继续由 MRAG 现有数据集链路负责。
- 阶段1只是在其上增加科研资产语义，例如：研究任务描述、推荐用途、标签、baseline、归档结果关系。

## 11. ResultArchive 的阶段1定位

阶段1必须明确以下约束：

- `ResultArchive` 是归档对象，不是实验自动化对象。
- `ResultArchive` 可以记录：
  - 结果标题
  - 结果摘要
  - 指标 JSON
  - 结论
  - 附件路径
  - 来源说明
  - 关联的 `DatasetAsset`
  - 可选关联的 `Baseline`
  - 可选关联的 `Idea`
- `ResultArchive` 不负责：
  - 调度服务器
  - 调用 GPU
  - 触发训练
  - 管理实验状态机
  - 驱动自动总结

## 12. 工作目录范围

阶段1允许新增研究资产工作目录，例如：

```text
workspace/research-assets/
  papers/
    incoming/
    parsed/
    insights/
  ideas/
    pool/
  datasets/
    <dataset_asset_id>/
      baselines/
  results/
    <result_archive_id>/
```

该目录用于保留可审计产物，但不是阶段1的主状态源。主状态源仍然是 PostgreSQL。

## 13. 规格完成标准

当以下条件满足时，可认为阶段1规格定义完成：

- 核心对象定义完整
- 对象关系明确
- 模块边界明确
- API 范围明确
- 页面范围明确
- 验收标准可独立执行
- 与集成方案保持一致
