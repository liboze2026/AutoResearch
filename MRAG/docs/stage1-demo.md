# Stage1 Demo

## 演示目标

本演示用于展示阶段1在 MRAG 主系统中的最小科研资产管理闭环。

## 演示前准备

- 启动 PostgreSQL 与 Go backend。
- 启动 Vue3 前端。
- 确认 `workspace/` 已挂载并可读写。

推荐直接执行：

```bash
bash scripts/validate_stage1.sh
```

## 建议演示顺序

### 1. 论文资产

进入前端页面：

- `/papers`

演示动作：

- 导入一篇论文。
- 观察论文状态从 `imported` 到 `parsed`。
- 触发“抽取创新点”。
- 进入论文详情页查看：
  - 基础信息
  - 文件列表
  - insight summary
  - contributions / methods / limitations

### 2. Idea 池

进入前端页面：

- `/ideas`

演示动作：

- 手工新增一个 idea。
- 选择一篇已完成 insight extraction 的论文，点击“从论文生成”。
- 打开 idea 详情，查看来源记录。

### 3. 数据集资产

进入前端页面：

- `/dataset-assets`

演示动作：

- 查看已有 dataset asset。
- 从 MRAG 已扫描数据集里选择一条记录，注册为 dataset asset。
- 打开详情，确认：
  - 路径
  - README
  - loader note
  - schema note
  - source linkage

### 4. Baseline

进入前端页面：

- `/baselines`

演示动作：

- 选择一个 dataset asset。
- 创建 baseline，录入：
  - metric schema
  - result json
  - note
- 打开详情，查看 baseline 内容。

### 5. Result Archive

进入前端页面：

- `/result-archives`

演示动作：

- 创建一个 result archive。
- 关联 dataset asset，可选关联 idea。
- 录入：
  - summary
  - metric json
  - note
  - figure / table 文本附件
- 打开详情，查看归档文件记录。

## 页面清单

- `/papers`
- `/papers/:id`
- `/ideas`
- `/dataset-assets`
- `/baselines`
- `/result-archives`

## 演示时推荐强调的边界

- 论文解析当前是 mock，但输入输出契约已稳定。
- dataset asset 是在 MRAG 既有扫描能力之上增加科研资产层，不重写扫描。
- baseline 是参考资产，不做自动优劣判断。
- result archive 是归档资产，不是实验执行对象。

## 演示后可展示的 workspace 产物

- `workspace/papers/parsed/<paper_id>/`
- `workspace/papers/insights/<paper_id>/`
- `workspace/ideas/pool/<idea_id>/`
- `workspace/datasets/<dataset_asset_id>/`
- `workspace/datasets/<dataset_asset_id>/baselines/<baseline_id>/`
- `workspace/results/<archive_id>/`

## 验收重点

- 所有对象都能在前端看到。
- 所有对象都能通过 API 创建或触发。
- workspace 中有对应落盘产物。
- 数据对象关系正确：
  - paper -> insight
  - insight -> idea
  - dataset -> dataset asset
  - dataset asset -> baseline
  - dataset asset / idea -> result archive