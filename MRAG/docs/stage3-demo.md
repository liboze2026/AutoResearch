# Stage3 Demo

## 目标

本文用于演示阶段3“受控智能体期”的最小可运行闭环。

演示目标不是展示完全自治科研，而是展示：

- 受控 agent runtime 已经成立
- 结构化 contract、validator、repair 已经成立
- agent 可以通过上游结构化输出驱动下游
- 系统仍然严格复用阶段2实验基础设施，而不是新起一套科研 workflow

## 演示前准备

推荐准备如下：

- PostgreSQL 可用
- Go 后端可启动
- Vue 前端可启动
- `workspace/` 可读写
- 如果本机有 Codex CLI，可设置 `CODEX_CLI_BIN=codex`
- 如果没有 Codex CLI，也可以直接演示 fallback 到 `mock`

推荐先执行一次自动验收：

```powershell
Get-Content .\scripts\validate_stage3.sh -Raw | Invoke-Expression
```

## 推荐演示顺序

### 1. 查看 Agent 管理页面

前端页面：

- `/agents`
- `/agents/jobs`
- `/agents/catalog`
- `/agents/events`

演示重点：

- 阶段3已经有独立的 agent 管理入口
- 可以查看 agent、job、artifact、event、tool、skill
- 前端不是聊天页，而是受控节点的可审计管理页

### 2. 展示统一 runtime 能力

建议展示：

- `backend/python_agents/runtime`
- `runner.py`
- `contract.py`
- `executors.py`
- `schema_registry.py`

演示重点：

- `api / codex_cli / mock` 共用统一 contract
- 每次输出都必须过 validator
- validator 失败后会进入 repair
- 返回 Go 端之前必须再次校验

### 3. 展示最小端到端链路

推荐直接基于自动验收结果展示这条链路：

1. `Reader`
2. `Insight`
3. `Dataset`
4. `Idea Generator`
5. `Planner`
6. `Coding + Evaluator`
7. `Writer + Picture`

对应最小链路：

- `paper -> insight -> dataset -> idea -> plan -> coding/eval -> writer`

演示重点：

- 上游结果是结构化对象，不是自由文本聊天
- 下游消费的是上游对象引用与持久化产物
- pipeline 是受控编排，不是完全自治循环

### 4. 展示落盘产物

推荐展示以下产物：

- `workspace/papers/parsed/{paper_id}/parsed.md`
- `workspace/papers/insights/{paper_id}/summary.md`
- `workspace/datasets/{dataset_asset_id}/evalplan.json`
- `workspace/experiments/{experiment_id}/plan.json`
- `workspace/experiments/{experiment_id}/coding/`
- `workspace/drafts/{draft_id}/draft.json`

演示重点：

- 每个 agent 输出都可持久化
- 每个关键节点都有可追踪文件产物
- 所有产物都能回到阶段2对象体系中

### 5. 展示自动回退策略

推荐说明：

- `codex_cli` 不可用时会自动回退到 `mock`
- `shenzhenvlab` 不可用或无空闲 GPU 时，最小验收会自动回退到 mock 路径
- 回退过程会写入 warnings、job 状态、日志和 summary

演示重点：

- 阶段3重视“受控可运行”，不是盲目追求实时真实模型
- fallback 是设计能力，不是临时补丁

## 自动验收演示

阶段3推荐直接用自动验收脚本做演示主入口：

```powershell
Get-Content .\scripts\validate_stage3.sh -Raw | Invoke-Expression
```

验收脚本会自动验证：

- agent runtime 可启动
- executor（mock / codex_cli fallback）可工作
- schema validator / repair 可工作
- tool registry / skill registry / memory store 可工作
- Reader / Insight / Dataset / Idea / Planner / Coding-Evaluator / Writer 最小测试通过
- 前端 agent 页面可访问
- 最小端到端链路通过
- 最终输出 `PASS / FAIL`

## 当前演示边界

演示时需要明确说明：

- 当前仍是受控智能体，不是完全自治科研系统
- 当前 `Writer / Picture` 仍是最小版本
- 当前 `Coding` 仍限制在统一训练模板中
- 当前真实模型与真实抓取能力只保留受控扩展点

## 演示结论

阶段3当前已经可以稳定演示以下事实：

- MRAG 已从阶段2实验自动化迈入阶段3受控智能体期
- agent 已经有统一 runtime、统一 contract、统一验收口径
- 系统可以跑通最小科研链路，但仍保持严格边界
- 这是一套“可校验、可修复、可持久化、可回退”的受控系统，而不是完全自治科研闭环
