# 阶段1集成方案：基于 MRAG 增量建设科研资产管理系统

## 1. 目标与原则

本阶段以 **MRAG 作为主系统**，在不推翻现有架构的前提下，吸收 `Auto_v1` 阶段0中已经验证过的目录协议、对象模型和最小脚本接口，落地“科研资产管理系统”。

本方案遵守以下边界：

- 以 `MRAG` 为唯一主仓库与主运行入口。
- 复用 MRAG 已有的 SSH、GPU 检查、服务器管理、远程数据集扫描与登记能力。
- 不把 `Auto_v1` 整套 workflow / orchestrator / mock experiment 原样搬入 MRAG。
- 优先建设领域模型、数据库模型、Go API、Vue 页面。
- 阶段1只覆盖科研对象管理，不扩展到自动实验调度、自治 agent、多阶段工作流编排。

## 2. 当前项目扫描结论

### 2.1 MRAG 当前已有能力清单

根据当前仓库结构与代码，MRAG 已具备以下能力：

- 前端基础框架：Vue 3 + Vite + Element Plus，已有布局、路由、API 模块、领域类型定义。
- 后端基础框架：Go + Gin + PostgreSQL，目录结构稳定，采用 `handler/service/repository/model/router` 分层。
- 运行与部署：Docker Compose 启动 PostgreSQL + Go API。
- 服务器管理：
  - 服务器 CRUD
  - SSH 连通性检查
  - GPU 空闲检查
  - 服务器状态刷新
  - 服务器配置 JSON 持久化
- 数据集管理：
  - 服务器目录扫描
  - 数据集候选发现
  - 路径校验
  - 数据集登记
  - 扫描摘要与预览项入库
  - 数据集索引任务与状态同步
- 总览与运行模式：
  - Overview 统计
  - real/mock 运行模式配置与展示

已确认的关键复用代码位置：

- SSH 与远程执行：`backend/go/internal/service/ssh_gateway.go`
- 服务器能力：`backend/go/internal/service/server_service.go`
- 数据集扫描与登记：`backend/go/internal/service/dataset_service.go`
- 远程数据集运行时：`backend/go/internal/service/dataset_remote_runtime.go`
- 数据库中已有数据集扩展表：
  - `datasets`
  - `dataset_scan_records`
  - `dataset_preview_items`
  - `dataset_index_tasks`
  - `dataset_index_task_logs`

### 2.2 阶段0 Auto_v1 已有能力清单

根据 `Auto_v1` 当前结构，阶段0已经完成以下最小闭环能力：

- workspace 目录协议：
  - `workspace/papers/incoming`
  - `workspace/papers/parsed`
  - `workspace/papers/insights`
  - `workspace/ideas/pool`
  - `workspace/datasets`
  - `workspace/experiments`
  - `workspace/runs/<workflow_id>/...`
- 已验证的对象域：
  - `papers`
  - `paper_insights`
  - `ideas`
  - `datasets`
  - `baselines`
  - `experiments`
  - `artifacts`
- 最小 Python 脚本接口：
  - `ingest_papers.py`
  - `extract_insights.py`
  - `generate_ideas.py`
  - `plan_experiment.py`
  - `run_mock_experiment.py`
  - `summarize_result.py`
- 脚本接口共性：
  - 接收 CLI 参数
  - 读写 workspace
  - 输出单个 JSON 到 stdout
  - 失败时非零退出
- 最小样例资产：
  - mock papers
  - mock insights
  - mock ideas
  - mock datasets
  - mock experiments / results
- 已验证的阶段0编排：
  - Go orchestrator 调 Python mock agents
  - PostgreSQL 持久化
  - workspace 落盘
  - 验证脚本验收

### 2.3 功能重叠点分析

MRAG 与 Auto_v1 的重叠点主要在“基础资产底座”，但侧重点不同：

- 服务器资产：
  - MRAG：真实服务器管理、SSH、GPU、状态刷新，已经是可用实现。
  - Auto_v1：主要是 mock server 与调度演示。
  - 结论：**只复用 MRAG，不迁移 Auto_v1 server/scheduler 实现。**
- 数据集资产：
  - MRAG：真实扫描、路径校验、登记、预览、索引。
  - Auto_v1：dataset 目录协议、baseline 文档、最小对象模型。
  - 结论：**以 MRAG 数据集链路为主，吸收 Auto_v1 的 dataset/baseline 对象约定。**
- 论文 / insight / idea：
  - MRAG：当前没有完整对象域。
  - Auto_v1：已经有最小对象模型、workspace 协议、脚本接口、样例数据。
  - 结论：**这是阶段1的主要迁移来源。**
- 实验与结果：
  - MRAG：历史上已删除 experiments/results 主系统模块。
  - Auto_v1：保留了 experiment/result 的最小闭环。
  - 结论：**阶段1只吸收“结果归档”对象能力，不恢复 Auto_v1 的自动实验 orchestrator。**

## 3. 复用策略

### 3.1 必须直接复用的 MRAG 模块

以下模块作为阶段1底座，禁止重复实现：

- 服务器表与服务器 API
- SSH 网关与真实 SSH 调用链路
- GPU 检查能力
- 服务器状态刷新
- 服务器目录扫描与候选数据集发现
- 数据集路径校验
- 数据集登记与扫描摘要
- 前端基础布局、路由、API 封装、类型管理方式
- Go 后端分层结构与 Docker 运行方式

### 3.2 建议复用的 MRAG 承载方式

阶段1新增科研对象时，应延续 MRAG 现有目录与代码风格：

- 后端继续沿用：
  - `internal/model`
  - `internal/repository`
  - `internal/service`
  - `internal/handler`
  - `internal/router`
- 前端继续沿用：
  - `src/api/modules`
  - `src/views`
  - `src/types/domain.ts`
  - `src/router/index.ts`

### 3.3 可从阶段0吸收的“协议”而非“整套实现”

建议吸收以下内容，但以“协议迁入、按 MRAG 风格重写或轻改”为主：

- workspace 目录语义
- 论文导入与解析后的中间对象划分
- insight / idea 的对象边界
- baseline 的最小元数据结构
- 结果归档的 artifact 思路
- Python 脚本统一 CLI + JSON stdout 契约
- `stable_id`、`metadata.json`、`summary.md` 这类可审计产物习惯

## 4. 迁移策略

### 4.1 迁移原则

- 不直接把 `Auto_v1/internal/workflow/phase0.go` 搬入 MRAG。
- 不恢复 `workflow_jobs` 作为阶段1主线中心。
- 不把 `plan_experiment.py`、`run_mock_experiment.py`、`summarize_result.py` 作为阶段1核心执行链。
- 迁移时优先抽取：
  - 对象模型
  - 文件协议
  - 解析脚本接口
  - 验证样例
- 尽量避免跨语言强耦合；阶段1中的 Python 仅保留在“论文解析/创新点抽取”这类局部任务。

### 4.2 建议迁移的阶段0目录协议

建议把 `Auto_v1/workspace` 的以下目录语义迁入 MRAG，但不要要求 MRAG 完全复制原目录树：

- `workspace/papers/incoming`
  - 迁入用途：论文原始导入区或导入任务输入区
- `workspace/papers/parsed`
  - 迁入用途：论文解析中间产物区
- `workspace/papers/insights`
  - 迁入用途：创新点/洞察产物区
- `workspace/ideas/pool`
  - 迁入用途：idea 池文件化快照区
- `workspace/datasets/<dataset_key>/baseline.*`
  - 迁入用途：baseline 附件或文件资产约定
- `workspace/experiments` 与 `workspace/runs/*/results`
  - 迁入用途：仅保留为“结果归档”参考，不迁入 workflow 编排

建议在 MRAG 中把这些目录语义重新收口为统一资产目录，例如：

```text
MRAG/workspace/
  research-assets/
    papers/
      incoming/
      parsed/
      insights/
    ideas/
      pool/
    datasets/
      <dataset_id>/
        baselines/
    results/
      <result_id>/
```

### 4.3 建议迁移的阶段0 脚本接口

建议迁移或重构的脚本接口：

- `ingest_papers.py`
  - 价值：定义了“原始论文 -> 规范化中间表示”的最小接口
  - 建议：重命名/重构为更聚焦阶段1的 paper parser
- `extract_insights.py`
  - 价值：定义了“解析结果 -> insight”接口
  - 建议：保留为创新点抽取脚本原型
- `generate_ideas.py`
  - 价值：定义了“insight -> idea”接口
  - 建议：保留其输入输出契约，改造成面向 MRAG 资产对象的生成器

不建议现在迁移为核心链路的脚本：

- `plan_experiment.py`
- `run_mock_experiment.py`
- `summarize_result.py`

这些脚本可作为后续阶段参考，但不应进入阶段1主系统主路径。

### 4.4 建议迁移的阶段0 对象模型

建议迁入 MRAG 的对象模型：

- `papers`
- `paper_insights`
- `ideas`
- `baselines`
- `artifacts` 的“结果归档/附件索引”思路

需要调整后再迁入的对象：

- `datasets`
  - 原因：MRAG 已经有更成熟的数据集对象与扫描链路
  - 处理：不迁移旧表，改为在 MRAG `datasets` 表基础上扩展科研资产字段
- `experiments`
  - 原因：MRAG 已明确删除该主系统能力
  - 处理：阶段1不恢复实验执行对象，只新建“结果归档”对象
- `workflow_jobs`
  - 原因：阶段1不以 workflow 编排为核心
  - 处理：不作为主模型迁入

## 5. 新建模块清单

阶段1建议新增以下模块，全部挂接在 MRAG 现有结构下。

### 5.1 后端领域对象

建议新增表或等价对象：

- `papers`
  - 论文主表，记录导入来源、标题、摘要、作者、年份、原始文件路径、解析状态
- `paper_parse_jobs` 或 `paper_parse_records`
  - 记录论文解析执行结果、错误信息、原始 payload、产物路径
- `paper_insights`
  - 记录论文创新点/洞察，支持多条 insight
- `ideas`
  - 记录 idea 池对象，支持状态、来源论文、来源 insight、人工编辑
- `dataset_baselines`
  - 记录某科研数据集下的 baseline 定义、指标与备注
- `research_results`
  - 记录结果归档对象，关联数据集 / baseline / idea / 服务器 / 路径
- `research_artifacts`
  - 记录 markdown/json/附件等资产路径，统一归档索引

说明：

- 如果希望最大程度减少数据库复杂度，可以不单独建 `research_artifacts`，而先把路径字段挂在 `paper_parse_records`、`dataset_baselines`、`research_results` 上。
- 但建议至少保留统一 artifact 设计思路，避免后续再次拆表。

### 5.2 后端代码模块

建议新增：

- `internal/handler/paper_handler.go`
- `internal/handler/idea_handler.go`
- `internal/handler/research_result_handler.go`
- `internal/service/paper_service.go`
- `internal/service/paper_parser_service.go`
- `internal/service/idea_service.go`
- `internal/service/baseline_service.go`
- `internal/service/research_result_service.go`
- `internal/repository/paper_repository.go`
- `internal/repository/idea_repository.go`
- `internal/repository/baseline_repository.go`
- `internal/repository/research_result_repository.go`

建议扩展现有：

- `internal/model/models.go`
- `internal/router/router.go`
- `internal/service/dataset_service.go`
- `internal/repository/dataset_repository.go`

### 5.3 前端页面模块

建议新增页面：

- `src/views/papers/PaperListPage.vue`
- `src/views/papers/PaperDetailPage.vue`
- `src/views/ideas/IdeaPoolPage.vue`
- `src/views/baselines/BaselinePage.vue`
- `src/views/results/ResultArchivePage.vue`

建议扩展页面：

- `src/views/datasets/DatasetDetailPage.vue`
  - 增加科研资产视角：注册状态、baseline 列表、关联结果
- `src/views/datasets/DatasetListPage.vue`
  - 增加“注册为科研数据集资产”动作或状态展示

建议新增 API 模块：

- `src/api/modules/papers.ts`
- `src/api/modules/ideas.ts`
- `src/api/modules/baselines.ts`
- `src/api/modules/results.ts`

## 6. 推荐目录结构调整方案

目标是最小侵入式合并，不重做仓库布局。

### 6.1 MRAG 代码目录保持不变

继续保留：

```text
MRAG/
  backend/go/internal/{model,repository,service,handler,router}
  src/{api,views,types,router,components}
  docs/
```

### 6.2 在 MRAG 内新增研究资产工作目录

建议新增：

```text
MRAG/
  workspace/
    research-assets/
      papers/
        incoming/
        parsed/
        insights/
      ideas/
        pool/
      datasets/
        <dataset_id>/
          baselines/
      results/
        <result_id>/
```

设计原因：

- 保留阶段0的 workspace 协议优势
- 不污染 MRAG 现有 `backend/`、`src/` 主代码结构
- 给论文解析、创新点抽取、结果归档保留可审计文件面
- 为后续验证脚本提供稳定路径

### 6.3 Python 辅助脚本建议位置

不建议把 `Auto_v1/python_agents` 原封不动迁进 MRAG 根目录。

建议改造为：

```text
MRAG/
  backend/python/
    paper_parser/
    insight_extractor/
    idea_generator/
```

或：

```text
MRAG/
  tools/research/
    parse_paper.py
    extract_insights.py
    generate_ideas.py
```

建议原则：

- 名称与阶段1业务对象对齐
- 去掉 “agent” 和 “phase0” 的叙事
- 保留统一 CLI 契约

## 7. 阶段1开发边界

### 7.1 阶段1应该做的内容

- 论文导入
- 论文解析入库
- 创新点抽取入库
- idea 池管理
- 将 MRAG 已扫描的数据集登记为科研数据集资产
- baseline 登记
- 结果归档
- 对应 API 与前端最小页面
- 最小验证脚本或接口验收

### 7.2 不建议现在做的内容

以下内容明确不属于阶段1，避免范围膨胀：

- 自动实验执行
- GPU 作业调度编排
- 恢复 Auto_v1 的完整 phase0 workflow
- 多 agent 自治
- 实验 planner 自动生成
- 训练任务下发与远程运行
- 结果自动总结 agent
- 复杂权限、多用户、协同编辑
- 向量索引、知识图谱、推荐排序等扩展能力
- 重构 MRAG 为微服务或消息队列架构

## 8. 最小侵入式合并建议

### 8.1 数据模型层面

- 以 MRAG 现有 `datasets`、`servers` 为主，不改主身份。
- 对数据集只补充“科研资产属性”，不另起一套平行 dataset 体系。
- 新增 `papers / insights / ideas / baselines / results` 等表，形成研究资产层。
- “结果归档”替代“实验执行”，避免把已删除的 experiments 主流程重新引入。

### 8.2 服务层面

- 论文解析、insight 抽取、idea 生成作为独立服务调用，不引入统一 orchestrator。
- 一次导入论文可以是同步接口，也可以是单次后台任务，但不建立 phase0 式状态机主线。
- 数据集注册直接复用现有扫描结果与服务器关系，不重写扫描器。

### 8.3 前端层面

- 在现有左侧导航基础上新增“论文”“Ideas”“Baselines”“结果归档”页面入口。
- 数据集页面继续作为数据集资产主入口，只补充科研属性，不新造平行页面体系。
- 论文详情页作为 insight 和 idea 的上游入口。

### 8.4 文档与验证层面

- 复用阶段0“可审计产物 + 验证脚本”思路。
- 但验收对象变成：
  - API 可调通
  - 页面可见
  - 入库可验证
  - workspace 产物可检查
- 不再以“完整 workflow 跑通”作为阶段1验收中心。

## 9. 推荐实施顺序

为降低侵入性，建议按以下顺序推进：

1. 先补研究资产数据库模型与 Go model/repository。
2. 再补 papers / insights / ideas / baselines / results API。
3. 再把 dataset 页面接上“科研资产注册”和 baseline/结果关联。
4. 最后补最小前端页面与验证脚本。

## 10. 结论清单

### 10.1 复用模块

- MRAG 服务器管理模块
- MRAG SSH 网关
- MRAG GPU 检查能力
- MRAG 服务器状态刷新能力
- MRAG 数据集扫描、校验、登记、预览、索引链路
- MRAG Vue 前端框架、路由、API 封装、页面布局
- MRAG Go 后端分层结构与 Docker 运行方式

### 10.2 待迁移模块

- Auto_v1 的 workspace 目录协议语义
- Auto_v1 的 `papers / paper_insights / ideas / baselines` 对象模型
- Auto_v1 的论文解析、创新点抽取、idea 生成脚本接口
- Auto_v1 的 mock papers / mock insights / mock ideas / mock datasets 作为阶段1测试样例
- Auto_v1 的 artifact 可审计思路与最小验证脚本思路

### 10.3 待新建模块

- MRAG 中的 `papers` 领域模型、表、API、前端页面
- MRAG 中的 `paper_insights` 领域模型、表、API
- MRAG 中的 `ideas` 领域模型、表、API、idea 池页面
- MRAG 中的 `dataset_baselines` 领域模型、表、API、页面
- MRAG 中的 `research_results` 领域模型、表、API、结果归档页面
- MRAG 中的研究资产 workspace 目录
- MRAG 中的论文解析 / insight 抽取 / idea 生成辅助脚本

### 10.4 下一步建议

下一步最小动作建议是：

1. 先产出阶段1的领域模型草案与数据库表设计清单。
2. 明确 `datasets` 表上哪些字段继续复用、哪些科研属性需要补充。
3. 明确 papers / insights / ideas / baselines / results 的最小 API 列表。
4. 再进入数据库迁移与接口实现，不要先写复杂页面。
