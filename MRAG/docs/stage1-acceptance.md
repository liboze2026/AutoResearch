# 阶段1验收标准

## 1. 文档目的

本文档定义 MRAG 阶段1“科研资产管理期”的验收范围、验收方法和通过标准。

阶段1验收以“对象、API、页面、关联关系、最小可验证性”为核心，不以自动实验、自动调度或自治工作流为验收目标。

## 2. 验收总原则

阶段1验收必须满足以下原则：

- 以 MRAG 为主系统完成增量建设。
- 不重复实现 MRAG 已有 SSH / GPU / 数据集扫描能力。
- 所有阶段1对象都能通过 API 和页面进行最小验证。
- 对象之间的关系必须能被创建、查看和校验。
- `DatasetAsset` 必须挂接 MRAG 已有 `Dataset`。
- `ResultArchive` 必须以归档对象验收，而不是以实验自动化对象验收。

## 3. 验收对象范围

本阶段至少验收以下对象：

- `Paper`
- `PaperFile`
- `Insight`
- `Idea`
- `DatasetAsset`
- `Baseline`
- `ResultArchive`

## 4. 功能验收项

### 4.1 Paper 验收

通过标准：

- 可以导入论文并创建 `Paper` 记录。
- 导入后至少能关联一个 `PaperFile`。
- 可以查看论文列表。
- 可以查看论文详情。
- 可以触发解析并看到解析状态或解析结果。

最小验证：

- `GET /api/v1/papers` 返回数据。
- `POST /api/v1/papers/import` 能创建记录。
- 论文详情页可展示标题、来源、文件、解析状态。

### 4.2 Insight 验收

通过标准：

- 可以基于指定 `Paper` 生成一个或多个 `Insight`。
- 可以在论文详情页看到关联 insights。
- 可以通过 API 读取论文 insights。

最小验证：

- `POST /api/v1/papers/:id/extract-insights` 成功。
- `GET /api/v1/papers/:id/insights` 返回非空或明确空结果。

### 4.3 Idea 验收

通过标准：

- 可以手工创建 `Idea`。
- 可以基于 insight 生成 `Idea`。
- 可以查看 idea 池列表。
- 可以查看 idea 与来源论文/insight 的关联。

最小验证：

- `GET /api/v1/ideas` 可用。
- `POST /api/v1/ideas` 可用。
- `POST /api/v1/ideas/generate` 或 `POST /api/v1/insights/:id/promote-to-idea` 至少一条链路可用。
- idea 池页可展示列表和来源信息。

### 4.4 DatasetAsset 验收

通过标准：

- 可以基于 MRAG 已有 `Dataset` 创建 `DatasetAsset`。
- 可以查看 `DatasetAsset` 详情。
- 可以在数据集页面看到其科研资产注册状态。

强制约束：

- 不允许创建未关联 MRAG `Dataset` 的 `DatasetAsset`。

最小验证：

- `POST /api/v1/dataset-assets` 时传入已存在 `dataset_id` 成功。
- 传入不存在 `dataset_id` 失败并返回明确错误。
- 数据集页或数据集详情页能看到“已注册/未注册为科研资产”。

### 4.5 Baseline 验收

通过标准：

- 可以为某个 `DatasetAsset` 创建 baseline。
- 可以查看 baseline 列表。
- 可以查看 baseline 的指标与说明。

最小验证：

- `POST /api/v1/dataset-assets/:id/baselines` 可用。
- `GET /api/v1/dataset-assets/:id/baselines` 可用。
- baseline 页可展示列表和核心指标。

### 4.6 ResultArchive 验收

通过标准：

- 可以创建 `ResultArchive`。
- `ResultArchive` 必须能关联 `DatasetAsset`。
- `ResultArchive` 可以可选关联 `Baseline` 与 `Idea`。
- 可以查看结果归档列表与详情。
- 可以查看摘要、结论、指标与附件路径。

强制约束：

- 创建 `ResultArchive` 不触发训练、调度、GPU 检查、远程执行或 workflow。
- `ResultArchive` 仅按归档对象验收。

最小验证：

- `POST /api/v1/result-archives` 可用。
- `GET /api/v1/result-archives` 可用。
- 结果归档页可展示关联数据集资产、baseline、idea、指标和摘要。

## 5. 页面验收项

阶段1至少验收以下页面。

### 5.1 论文列表页

通过标准：

- 可打开。
- 可展示论文列表。
- 可触发导入。
- 可跳转详情页。

### 5.2 论文详情页

通过标准：

- 可展示论文基础信息。
- 可展示关联文件。
- 可展示解析结果与 insights。
- 可支持基于 insight 生成或关联 idea。

### 5.3 idea 池页

通过标准：

- 可展示 idea 列表。
- 可展示状态。
- 可展示来源信息。
- 可编辑最小字段。

### 5.4 数据集页

通过标准：

- 继续保留 MRAG 原有数据集展示能力。
- 可显示科研数据集资产注册状态。
- 可进入 baseline 或结果归档相关信息入口。

### 5.5 baseline 页

通过标准：

- 可展示 baseline 列表。
- 可新建和编辑 baseline。
- 可展示关键指标和说明。

### 5.6 结果归档页

通过标准：

- 可展示结果归档列表。
- 可创建结果归档。
- 可查看结果归档详情。
- 可看到关联对象与指标摘要。

## 6. 对象关系验收项

阶段1必须通过以下关系验收：

- `Paper` 与 `PaperFile` 的一对多关系可验证。
- `Paper` 与 `Insight` 的一对多关系可验证。
- `Insight` 与 `Idea` 的来源关系可验证。
- `DatasetAsset` 与 MRAG `Dataset` 的关联可验证。
- `DatasetAsset` 与 `Baseline` 的一对多关系可验证。
- `DatasetAsset` 与 `ResultArchive` 的一对多关系可验证。
- `ResultArchive` 与 `Baseline` / `Idea` 的可选关联可验证。

## 7. API 验收标准

阶段1 API 验收通过需满足：

- 所有新增 API 路由可以访问。
- 成功请求返回统一响应结构。
- 失败请求返回明确错误信息。
- 关键对象的创建、读取、更新链路可走通。
- 关键关联的非法输入能被拦截，例如无效 `dataset_id`。

## 8. 数据与产物验收标准

阶段1数据验收通过需满足：

- 核心对象成功入库。
- 关键关系成功入库。
- 论文解析、insight 抽取、结果归档等必要产物有稳定路径可检查。
- PostgreSQL 作为主状态源。
- 如使用 workspace 目录，仅作为可审计产物区，不替代数据库真源。

## 9. 复用约束验收

阶段1必须满足以下复用约束：

- 数据集扫描继续复用 MRAG 现有数据集链路。
- SSH 调用继续复用 MRAG 现有 SSH 网关。
- GPU 检查继续复用 MRAG 现有能力。
- 不新增平行的 server 管理系统。
- 不新增平行的 dataset 扫描系统。

## 10. 不通过条件

出现以下任一情况，阶段1视为未通过：

- 新系统绕开 MRAG，形成平行主系统。
- 重复实现 SSH / GPU / 数据集扫描基础能力。
- `DatasetAsset` 不依赖 MRAG 已有 `Dataset`。
- `ResultArchive` 被实现为实验自动执行对象。
- 页面缺失，无法看到论文、idea、数据集、baseline、结果归档。
- API 只有文档没有可验证实现。
- 对象能创建但关系不可验证。

## 11. 最终通过标准

当以下条件全部满足时，阶段1可判定通过：

1. 前端能看到：论文列表、论文详情、idea 池、数据集、baseline、结果归档。
2. 能导入论文并完成解析入库。
3. 能基于论文生成 insights 并入库。
4. 能管理 idea 池。
5. 能将 MRAG 已扫描数据集注册为 `DatasetAsset`。
6. 能为 `DatasetAsset` 登记 baseline。
7. 能创建和查看 `ResultArchive`。
8. API 和页面均可验证通过。
9. 所有实现仍挂接在 MRAG 当前项目结构中。
