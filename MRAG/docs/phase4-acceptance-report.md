# 阶段4验收报告

## 验收范围

本次验收覆盖阶段4首期已经落地的能力，不新增功能、不修改实现，只对以下内容做整体验收、回归验证与证据化总结：

- Go 后端、Python runtime、前端构建与阶段3兼容回归
- 阶段4主链路：Dataset Profile -> Reader -> Idea -> Coding -> Writing
- VisDoM page-level retrieval 首期评测闭环
- `shenzhenvlab` 真实或半真实执行证据
- 阶段4前端展示与报告导出入口

本报告严格区分三类证据：

- 本次重新执行的验证
- 既有历史 smoke 证据
- 因环境或风险约束未重跑、只能引用的替代证据

## 环境说明

- 验收时间：2026-03-27
- 主机环境：Windows / PowerShell
- 仓库根目录：`D:\4\MRAG`
- Go：`go version go1.26.1 windows/amd64`
- Python：`Python 3.12.4`
- Node：`v20.12.0`
- npm：`10.5.0`
- PostgreSQL 容器：`mrag-postgres`
- 当前 Docker 数据库确认存在真实业务库：`mrag_platform`
- 当前 Docker 数据库中确认存在真实 `shenzhenvlab` 记录：
  - `id = srv_1774366607_14235076`
  - `host = shenzhen-vlab`
  - `ssh_port = 2266`
  - `username = bzli`
  - `auth_type = password`
  - `remote_root = /home/bzli`

## 执行的命令与测试

### 本次重新执行

1. Go 构建

```powershell
go build -buildvcs=false ./cmd/server
```

结果：通过

2. Go 阶段4专项测试

```powershell
go test -buildvcs=false ./internal/phase4workflow ./internal/handler ./internal/service ./internal/router ./internal/codingagent ./internal/ideaagent ./internal/writeragent ./internal/readeragent
```

结果：通过

3. 阶段4 workflow 主链路 smoke

```powershell
go test -buildvcs=false ./internal/handler -run TestPhase4WorkflowHandlerEndToEndSmoke -v
```

结果：通过  
说明：验证了 workflow create -> awaiting_selection -> select idea -> completed 的主链路。

4. Python 静态导入检查

```powershell
python -m compileall backend/python_agents backend/python_runners
```

结果：通过

5. Python runtime 单元测试

```powershell
python -m unittest discover -s backend/python_agents/runtime/tests -p "test_*.py"
```

结果：通过，`51` 个测试

6. Retrieval mainline / VisDoM 单元与集成测试

```powershell
python -m unittest discover -s backend/python_runners/retrieval_mainline/tests -p "test_*.py"
```

结果：通过，`7` 个测试

7. 前端基线验证

```powershell
npm.cmd run test:frontend:basic
```

结果：通过  
备注：存在 Vite chunk size warning，但不是失败。

8. 阶段3兼容回归

```powershell
Get-Content D:\4\MRAG\scripts\validate_stage3.sh -Raw | Invoke-Expression
```

结果：通过  
摘要文件：`D:\4\MRAG\workspace\validation\stage3\stage3_validation_summary.json`

### 本次只读核查

9. 数据库 migration 序列核查

- 已确认 migration 序列包含：
  - `019_phase4_foundation.sql`
  - `020_phase4_workflow.sql`
- 本次未手工改表
- backend 在阶段3回归中成功启动并完成 migration，说明当前 migration 链可回放至最新版本

10. 真实业务库只读核查

```powershell
docker exec mrag-postgres psql -U postgres -d mrag_platform -c "select id,name,host,ssh_port,username,remote_root from servers order by created_at desc limit 10;"
```

结果：通过  
说明：确认真实业务库中存在 `shenzhenvlab` 记录。

### 本次未成功重跑的真实验证

11. 本次曾尝试启动临时 backend 直接对真实 `mrag_platform` 做 `heartbeat/check-gpu` 探测，但首次临时探针脚本失败：

- 原因 1：临时 PowerShell 环境变量拼接命令错误
- 原因 2：首次使用了错误的 PostgreSQL 密码
- 处理：未继续扩展新的临时探针代码，避免在验收阶段引入额外自定义运行风险
- 替代证据：采用 Docker 业务库只读核查 + 既有 `phase4-remote-live` 历史真实 smoke 证据

## 每项结果

### 1. 全量回归检查

#### Go

- 通过：`go build` 成功
- 通过：phase4 相关 Go 包测试成功
- 通过：workflow e2e smoke 成功

#### Python runtime

- 通过：`compileall` 成功
- 通过：runtime 测试 `51` 个全部成功
- 通过：retrieval_mainline 测试 `7` 个全部成功

#### 前端

- 通过：`typecheck + build` 成功
- 通过：phase4 workflow / dataset / ideas / reports 页面已进入构建产物
- 未执行浏览器级自动化交互测试：当前仓库没有现成组件/浏览器测试框架

#### 数据库变更

- 通过：migration 序列到 `020_phase4_workflow.sql`
- 通过：backend 在回归脚本里成功启动，说明 migration 未阻塞启动
- 未做额外手工 migration 回滚测试：本轮仅做非破坏式验收

#### 阶段3兼容性

- 通过：`validate_stage3.sh` 全量通过
- 说明：当前 validation DB 仍显示 `shenzhenvlab_probe = missing`，这是 stage3 validation 独立数据库的既有状态，不是阶段4引入问题

### 2. 阶段4主链路验证

#### Dataset Profile

- 通过：phase4 foundation 模型、服务、API、前端页面都已落地
- 证据：
  - 后端专项测试通过
  - workflow smoke 以 dataset profile 为入口推进
  - 前端页面：`src/views/datasets/DatasetDetailPage.vue`

#### Reader

- 通过：存在真实联网 Reader smoke，且本次 runtime 测试回归通过
- 关键证据：
  - `D:\4\MRAG\workspace\smoke\phase4-reader-live\output.json`
  - 输出显示：
    - `status = succeeded`
    - `execution_mode_used = api`
    - `used_fixture = false`
    - `provider_statuses = openalex/crossref/arxiv all succeeded`
- 结论：Reader 已具备按数据集找相关工作并生成结构化研究上下文的首期能力

#### Idea Pool

- 通过：存在批量 10 条 idea 生成证据，且 revision 候选链路存在
- 关键证据：
  - `D:\4\MRAG\workspace\smoke\phase4-idea\output.json`
  - `D:\4\MRAG\workspace\smoke\phase4-idea-revision\output.json`
- 输出确认：
  - 一次生成 `10` 条高细度结构化 idea
  - `top_recommendations` 存在
  - revision 候选 `3` 条
  - `revision_of_id` 与 `last_failure_run_id` 已建立

#### Coding Run

- 通过：存在本地主线生成执行证据 + 历史真实 `shenzhenvlab` 执行证据
- 本地主线关键证据：
  - `D:\4\MRAG\workspace\smoke\phase4-coding-step8-generated-run\artifacts\metrics.json`
- 历史真实远端关键证据：
  - `D:\4\MRAG\workspace\smoke\phase4-remote-live\backend-workspace\phase4\artifacts\p4run_1774519837_76f27ce8\metrics.json`
  - `D:\4\MRAG\workspace\smoke\phase4-remote-live\backend-workspace\phase4\runs\p4run_1774519837_76f27ce8\logs\driver.log`
  - `D:\4\MRAG\workspace\smoke\phase4-remote-live\backend-workspace\phase4\runs\p4run_1774519837_76f27ce8\logs\run.log`
- 说明：
  - 历史真实远端证据表明 phase4 coding 曾在 `shenzhenvlab` 上成功执行，且产生 logs / metrics / artifacts
  - 但本次未重新完成真实远端重跑

#### Writing Report

- 通过：machine-readable 报告已形成，human-readable 报告文件已形成
- 关键证据：
  - `D:\4\MRAG\workspace\smoke\phase4-writer-step9\workspace\writer_phase4_machine_report.json`
  - `D:\4\MRAG\workspace\smoke\phase4-writer-step9\workspace\writer_phase4_human_report.md`
- 结构化报告字段覆盖：
  - `dataset`
  - `task`
  - `reader_context_summary`
  - `citations`
  - `idea`
  - `implementation`
  - `run_config`
  - `metrics`
  - `error_summary`
  - `result_analysis`
  - `limitations`
  - `next_steps`

### 3. VisDoM 检索评测验证

#### 半真实 / fixture 指标

证据文件：`D:\4\MRAG\workspace\smoke\phase4-visdom-fixture\artifacts\metrics.json`

- `Recall@1 = 1.0`
- `Recall@5 = 1.0`
- `Recall@10 = 1.0`
- `MRR = 1.0`
- `nDCG@10 = 1.0`
- `failure_rate = 0.0`
- `retrieval_granularity = page`

结论：VisDoM page-level retrieval 的首期评测协议、adapter、baseline、metrics schema 已闭环。

#### 生成方法 smoke 指标

证据文件：`D:\4\MRAG\workspace\smoke\phase4-coding-step8-generated-run\artifacts\metrics.json`

- `Recall@1 = 1.0`
- `Recall@5 = 1.0`
- `Recall@10 = 1.0`
- `MRR = 1.0`
- `nDCG@10 = 1.0`
- `failure_rate = 0.0`
- `method_name = visdom_layout_aware_hard_negative_mining`

结论：Coding Agent 生成的方法模块已能接入 retrieval_mainline，并产出完整五项主指标。

#### 历史真实远端 run 指标

证据文件：`D:\4\MRAG\workspace\smoke\phase4-remote-live\backend-workspace\phase4\artifacts\p4run_1774519837_76f27ce8\metrics.json`

- `Recall@5 = 0.66`
- `query_count = 3`
- `runner_mode = shenzhenvlab`

结论：真实远端 smoke 证明了 phase4 run 可以在 `shenzhenvlab` 上执行并回收指标，但这条历史真实 run **未覆盖完整五项主指标**，当前只明确记录了 `Recall@5`。

### 4. 真实或半真实 shenzhenvlab 路径验证

#### 本次直接验证到的事实

- Docker 真实业务库中存在 `shenzhenvlab` server 记录
- 记录的 `remote_root = /home/bzli`
- 阶段4配置默认 `PHASE4_REMOTE_WORK_ROOT = /home/bzli/mrag`
- 历史真实远端 smoke 产物存在：
  - 远端 run id：`p4run_1774519837_76f27ce8`
  - 本地已回收：
    - `metrics.json`
    - `report.md`
    - `machine_report.json`
    - `driver.log`
    - `run.log`

#### 本次未完成的真实探测

- 未能在本轮验收中重新完成 `heartbeat/check-gpu` 的真实 API 探测
- 因此本次没有拿到新的“当前空闲 GPU 数量”直接证据

#### 替代证据

- 历史真实 smoke：
  - `driver.log` 显示：
    - `run_id = p4run_1774519837_76f27ce8`
    - `selected_gpu = 0`
    - `completed`
- 历史真实产物目录：
  - `D:\4\MRAG\workspace\smoke\phase4-remote-live\backend-workspace\phase4\artifacts\p4run_1774519837_76f27ce8`

#### 后续真实重跑命令

若要补齐“本次验收时刻”的真实远端证据，建议后续在已知安全窗口内执行：

```powershell
POST /api/v2/phase4/coding/run
runnerMode = shenzhenvlab
serverId = srv_1774366607_14235076
```

并同时执行：

```powershell
POST /api/v1/servers/srv_1774366607_14235076/heartbeat
POST /api/v1/servers/srv_1774366607_14235076/check-gpu
```

### 5. 报告导出与前端展示验证

#### 前端展示

- 通过：`npm.cmd run test:frontend:basic`
- 通过：构建产物中已包含：
  - `DatasetDetailPage`
  - `IdeaPoolPage`
  - `ResultArchivePage`
  - `Phase4WorkflowListPage`
  - `Phase4WorkflowDetailPage`

#### 导出入口

代码证据：

- `src/views/research/ResultArchivePage.vue`
  - 存在 `Export JSON`
  - 存在 `Export MD`
  - 调用 `downloadTextFile(...)`
- `src/views/phase4/Phase4WorkflowDetailPage.vue`
  - 存在 `Export JSON`
  - 存在 `Export MD`
  - 存在 `Retry Failed Stage`
  - 存在 `Select Revision`

#### 报告内容填充

- `machine-readable`：非空，字段齐全
- `human-readable`：非空，但发现文本编码问题，见下文“未通过项”

## 通过项

1. Go / Python / 前端 / 阶段3兼容回归全部通过
2. 数据集驱动的 phase4 workflow 主链路已有可执行后端 smoke 证据
3. Reader 已有真实联网检索与结构化上下文输出证据
4. Idea 已能批量生成 10 条高细度 idea，并能产出 revision 候选
5. Coding 已能生成 method module、接入 retrieval_mainline、执行并产出 logs / metrics / artifacts
6. Writing 已能生成 machine-readable 结构化实验报告
7. VisDoM page-level retrieval 的五项主指标在 fixture / local generated run 上已闭环
8. 阶段3未被本轮阶段4改造破坏

## 未通过项

1. **本轮验收没有重新完成一次真实 `shenzhenvlab` coding 重跑**
   - 当前只有历史真实远端 smoke 证据和本次 Docker 业务库只读核查
   - 因此“本次时刻的真实远端可用性”证据不完整

2. **历史真实 `shenzhenvlab` metrics 未覆盖完整五项主指标**
   - 现有真实远端 `metrics.json` 只明确记录了 `Recall@5`
   - `Recall@1 / Recall@10 / MRR / nDCG@10` 在真实远端 smoke 中没有完整体现

3. **human-readable 报告存在编码/乱码问题**
   - 文件：`D:\4\MRAG\workspace\smoke\phase4-writer-step9\workspace\writer_phase4_human_report.md`
   - 现象：中文标题出现乱码，如 `鏁版嵁闆嗕笌浠诲姟`
   - 影响：可读实验报告的展示质量和可直接复用性下降

4. **前端关键交互没有浏览器级自动化测试**
   - 当前仓库没有现成组件/浏览器测试框架
   - 本轮只能以 `typecheck + build + 后端 workflow smoke` 作为替代证据

## 风险项

1. 真实远端 `shenzhenvlab` 执行链路虽然有历史成功证据，但本轮没有完成重新探测，存在环境时变风险
2. human-readable 报告编码问题如果出现在前端展示，会直接影响阶段4“报告收口器”的用户感知
3. 真实远端指标 schema 目前与 fixture/local run 不完全一致，说明真实 smoke 还未完全对齐五项主指标
4. 前端当前缺少浏览器级 smoke/回归测试，后续 UI 迭代回归风险较高

## 对阶段3影响评估

- 本次阶段3全量回归通过，说明阶段4首期实现没有破坏阶段3主链路
- 当前确认仍可用的阶段3能力包括：
  - Reader / Insight / Dataset / Idea / Planner / Coding / Writer 旧链路
  - tool registry / skill registry / memory store / agent admin
  - 阶段3前端主要入口
- 当前 `stage3_validation_summary.json` 已重新生成，说明阶段4代码未造成阶段3回归失败

## 关键路径与日志摘要

### 阶段3兼容

- 摘要：`D:\4\MRAG\workspace\validation\stage3\stage3_validation_summary.json`

### Reader

- 输出：`D:\4\MRAG\workspace\smoke\phase4-reader-live\output.json`
- 摘要：`execution_mode_used = api`，`used_fixture = false`，5 条来源，OpenAlex/Crossref/arXiv 全成功

### Idea

- 批量生成：`D:\4\MRAG\workspace\smoke\phase4-idea\output.json`
- 修正版候选：`D:\4\MRAG\workspace\smoke\phase4-idea-revision\output.json`

### Coding

- 生成方法 run：`D:\4\MRAG\workspace\smoke\phase4-coding-step8-generated-run\artifacts\metrics.json`
- 历史真实远端：
  - `D:\4\MRAG\workspace\smoke\phase4-remote-live\backend-workspace\phase4\artifacts\p4run_1774519837_76f27ce8\metrics.json`
  - `D:\4\MRAG\workspace\smoke\phase4-remote-live\backend-workspace\phase4\runs\p4run_1774519837_76f27ce8\logs\driver.log`

### Writing

- machine-readable：
  - `D:\4\MRAG\workspace\smoke\phase4-writer-step9\workspace\writer_phase4_machine_report.json`
- human-readable：
  - `D:\4\MRAG\workspace\smoke\phase4-writer-step9\workspace\writer_phase4_human_report.md`

### VisDoM metrics

- fixture：
  - `D:\4\MRAG\workspace\smoke\phase4-visdom-fixture\artifacts\metrics.json`
- generated run：
  - `D:\4\MRAG\workspace\smoke\phase4-coding-step8-generated-run\artifacts\metrics.json`

## 最终结论

**结论：阶段4首期当前达到“有条件通过”，尚未达到“完全无保留通过”。**

理由如下：

- 阶段4主链路的核心能力已经形成闭环，且有大量本次回归结果与既有 smoke 证据支撑：
  - dataset profile
  - Reader 结构化研究上下文
  - Idea 批量生成与 revision
  - Coding logs / metrics / artifacts
  - Writing 结构化报告
- 阶段3兼容回归通过，说明阶段4没有明显破坏旧系统
- 但仍存在三项影响“首期完全可交付”的真实缺口：
  1. 本轮没有重新完成 `shenzhenvlab` 真实 coding 重跑
  2. 真实远端 metrics 还未完整覆盖五项主指标
  3. human-readable 报告存在编码乱码问题

因此，本报告给出的正式判断是：

- **主链路工程实现：通过**
- **阶段3兼容性：通过**
- **首期 VisDoM 评测协议与基线：通过**
- **真实远端执行能力：有历史证据，但本轮验收证据不完整**
- **报告收口：machine-readable 通过，human-readable 存在缺陷**
- **阶段4首期总体：有条件通过**

## 下一步建议

1. 先修复 human-readable 报告编码问题，再重新导出一份 phase4 报告
2. 以 `srv_1774366607_14235076` 为目标，在安全时段补一次新的真实 `shenzhenvlab` phase4 coding + writing run
3. 让真实远端 run 的 `metrics.json` 完整对齐五项主指标
4. 为 phase4 前端主路径补最小浏览器级 smoke 测试
