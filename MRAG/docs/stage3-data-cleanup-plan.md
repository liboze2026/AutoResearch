# 阶段3数据清洗方案

## 1. 目标与边界

本文只定义阶段3开始前的数据清洗方案，不执行任何删除动作。

本方案遵守以下硬约束：

- 可以清理阶段0/1/2产生的演示型测试数据
- 真实服务器配置不能误删
- `servers` 表中必须保留 `shenzhenvlab`
- 如存在 mock server，可保留作为 fallback
- 如果 `shenzhenvlab` 当前 GPU 空闲，可用于阶段3最小测试；否则自动回退 mock server

## 2. 清洗原则

### 2.1 只清理“可明确识别的演示数据”

允许清理的数据必须同时满足以下至少一类特征：

- ID 或标题明显带 `demo` / `stage1_demo` / `stage2_demo` / `validation`
- `source_type` 明确是 `seed`、`mock`、`demo`
- 由阶段验证脚本生成，且只用于页面/API 验收
- workspace 中与数据库演示记录一一对应的派生文件

### 2.2 真实数据默认不删

以下情况默认视为真实或疑似真实，禁止直接删除：

- 来自真实服务器扫描的 dataset / dataset asset
- 引用了真实远程路径的数据资产
- 引用了 `shenzhenvlab` 的服务器、数据集、实验、结果
- 无法从命名、来源、时间、路径判定为 demo 的记录

### 2.3 先标记、再备份、后删除

真正执行清洗时，必须按顺序进行：

1. 生成候选清单
2. 核对与 `shenzhenvlab` 的引用关系
3. 导出待删记录快照
4. 先删除 demo 派生数据，再删除 demo 主记录
5. 删除后做引用完整性验证

当前阶段仅停留在第 1 步和第 2 步的方案设计。

## 3. 建议保留的数据

### 3.1 必须保留：`servers.name = shenzhenvlab`

保留要求：

- 不论其当前是否在线，都不能删除该服务器记录
- 不得清空其 SSH、路径、认证配置
- 清洗脚本必须把它加入硬编码保护名单

建议保护条件：

- `id = <query result of shenzhenvlab>`
- 或 `name = 'shenzhenvlab'`
- 两者任一命中都禁止删除

### 3.2 建议保留：真实远程数据资产

从当前 workspace 可见，至少存在与真实服务器相关的资产痕迹，例如：

- `workspace/datasets/dasset_1774356289_6a6d9fb1/README.md` 指向 `Server: shenzhenvlab`
- 路径 `/home/bzli/lbz/data/VisDoM-main` 明显属于真实远程目录

因此以下类型默认保留：

- 由真实 server scan 注册的 `dataset_assets`
- 其关联 `baselines`
- 其关联 `result_archives`
- 其关联 `experiments / experiment_runs / result_comparisons`

### 3.3 可保留：mock server 记录

若当前库中存在 mock server，建议保留至少一条，作为阶段3 fallback 节点。

保留理由：

- 阶段3最小测试当前优先走 `codex_cli` / mock
- 当 `shenzhenvlab` 不空闲或不可达时，需要安全回退路径

## 4. 建议清理的数据类别

### 4.1 阶段1 migration seed 数据

可清理候选：

- `paper_stage1_demo_001`
- `pinsight_stage1_demo_001`
- `idea_stage1_demo_001`
- `dasset_stage1_demo_001`
- `baseline_stage1_demo_001`
- `rarch_stage1_demo_001`
- `ds_stage1_demo_nlp`

判断依据：

- 来自 `backend/go/migrations/006_stage1_research_assets_seed.sql`
- 目的明确是 stage1 demo / validation

清理前提：

- 若后续还有阶段1回归验收需求，可选择不删，只标记为 demo
- 若这些记录仍被阶段2 demo 记录引用，必须先处理阶段2 demo 记录

### 4.2 阶段2 migration seed 数据

可清理候选：

- `exp_stage2_demo_001`
- `espec_stage2_demo_001`
- `erun_stage2_demo_001`
- `sdec_stage2_demo_001`
- `shb_stage2_demo_001`
- `gsnap_stage2_demo_001`
- `rcmp_stage2_demo_001`

判断依据：

- 来自 `backend/go/migrations/012_stage2_experiment_seed.sql`
- 明确用于 stage2 demo

清理顺序建议：

1. `result_comparisons`
2. `scheduler_decisions`
3. `run_logs`
4. `experiment_runs`
5. `experiment_specs`
6. `experiments`
7. `server_heartbeats` / `gpu_resource_snapshots`

### 4.3 workspace 中明显的阶段1演示论文数据

可清理候选：

- `workspace/papers/incoming/stage1_flow_demo.md`
- 由其派生的 `workspace/papers/parsed/...`
- 由其派生的 `workspace/papers/insights/...`

判断依据：

- 文件名明确为 `stage1_flow_demo`
- 内容明显用于 deterministic stage1 演示

注意：

- 删除前应确认数据库中没有非 demo `paper` 记录引用这些路径

### 4.4 阶段验证生成的临时 idea / experiment / result archive

从当前 workspace 能看到多批时间戳式记录，如：

- `idea_1774414979_a9b14976`
- `exp_1774414991_08f98cea`
- `archive_1774414999_a04ca89f`

其中部分内容明确写有：

- `Stage2 Validation Idea`
- `Stage2 Validation Dataset`
- `Stage2 Validation Archive`

因此建议把以下特征作为“验证型数据候选”：

- 标题中含 `Validation`
- note/readme/summary 中含 `stage2 validation`
- spec 中使用 `mock/...` 模型名
- output/result 中是模板化 mock_train_template 产物

注意：

- 这类数据不能只靠 ID 时间戳判断
- 必须结合标题、summary、source_type、workspace 内容交叉判断

## 5. 禁止清理的数据类别

### 5.1 真实服务器配置

禁止删除：

- `servers.name = shenzhenvlab`
- 任何手工录入、非 demo 命名、且具有真实远程主机配置的 server 记录

### 5.2 真实远程扫描登记的数据集

禁止删除：

- 来自真实 server scan 的 dataset
- 来自真实 dataset scan record 注册的 dataset asset
- 任何 `local_or_remote_path` 指向真实远程目录的科研资产

### 5.3 无法明确判定为 demo 的对象

禁止删除：

- source 信息缺失
- 标题不含 demo/validation
- 但引用真实 server、真实 path、真实 baseline 的对象

规则：

- 有疑问时一律保留

## 6. `servers` 表的安全策略

### 6.1 必保规则

执行清洗时必须先查询：

- `SELECT id, name, host, status, available_gpus, total_gpus FROM servers ORDER BY created_at;`

然后应用以下保护规则：

- `name = 'shenzhenvlab'` 的记录强制保留
- 若存在多条名称相同记录，全部转人工确认，不自动删
- 若存在 mock server，允许保留至少一条

### 6.2 阶段3测试使用策略

执行阶段3最小测试前，建议判断：

1. `shenzhenvlab` 是否在线
2. 最近 heartbeat 是否新鲜
3. 最近 GPU snapshot 是否显示存在空闲 GPU

若以上条件满足：

- 可选择 `shenzhenvlab` 执行最小真实测试

否则：

- 自动回退 mock server
- 或直接使用 mock / codex_cli execution mode

## 7. 推荐的清洗判定规则

建议未来实现清洗脚本时使用“白名单保护 + 黑名单候选 + 依赖检查”三层规则。

### 7.1 白名单保护

强制保护：

- `servers.name = 'shenzhenvlab'`
- 所有关联 `shenzhenvlab` 的关键科研资产
- 所有真实远程路径资产

### 7.2 黑名单候选

自动纳入候选：

- `*_stage1_demo_*`
- `*_stage2_demo_*`
- 标题或描述包含 `demo`
- 标题或描述包含 `validation`
- `source_type in ('seed', 'mock', 'demo')`

### 7.3 依赖检查

任何候选对象在真正删除前，都必须检查：

- 是否被非 demo 记录引用
- 是否引用或被引用 `shenzhenvlab`
- 是否关联真实远程路径
- 是否关联最近仍在使用的 dataset asset / baseline / archive

## 8. 建议的实际删除顺序

当前不执行，只提供未来顺序建议。

### 8.1 先删派生对象

- `result_comparisons`
- `scheduler_decisions`
- `run_logs`
- `archive_files`
- workspace 派生文件

### 8.2 再删执行对象

- `experiment_runs`
- `experiment_specs`
- `experiments`

### 8.3 再删科研资产 demo 主对象

- `result_archives`
- `baselines`
- `idea_sources`
- `ideas`
- `paper_insights`
- `paper_files`
- `papers`
- `dataset_asset_sources`
- `dataset_assets`
- `datasets`

### 8.4 最后处理监控类派生数据

- `server_heartbeats`
- `gpu_resource_snapshots`

注意：

- `servers` 表最后处理，而且默认只删 mock/demo server
- `shenzhenvlab` 永不进入删除候选

## 9. workspace 清理建议

未来若执行 workspace 清理，建议遵循：

- 只删除与 demo 数据库记录可一一映射的目录
- 先删 `workspace/experiments/<demo_exp_id>`
- 再删 `workspace/results/<demo_archive_id>`
- 再删 `workspace/ideas/pool/<demo_idea_id>`
- 最后删 `workspace/papers/*` 下的 demo 派生文件

禁止直接整库清空：

- 不允许直接清空 `workspace/datasets`
- 不允许直接清空 `workspace/results`
- 不允许直接清空 `workspace/experiments`

因为其中已经出现真实服务器相关资产痕迹。

## 10. 当前建议的清洗清单

### 10.1 明确可作为清洗候选

- 阶段1 seed：`paper_stage1_demo_001`、`idea_stage1_demo_001`、`dasset_stage1_demo_001`、`baseline_stage1_demo_001`、`rarch_stage1_demo_001`
- 阶段2 seed：`exp_stage2_demo_001`、`espec_stage2_demo_001`、`erun_stage2_demo_001`、`sdec_stage2_demo_001`、`shb_stage2_demo_001`、`gsnap_stage2_demo_001`、`rcmp_stage2_demo_001`
- workspace demo paper：`workspace/papers/incoming/stage1_flow_demo.md` 及其派生解析/insight 文件
- 标题或内容明确带 `Stage2 Validation` 的临时验证实验链条

### 10.2 明确不能删

- `servers.name = shenzhenvlab`
- 与 `shenzhenvlab` 关联的真实远程 dataset asset
- 任何真实远程路径资产
- 无法明确判定为 demo 的记录

## 11. 执行前检查清单

未来真正执行清洗前，必须先完成以下检查：

1. 导出 `servers` 全表快照
2. 单独确认 `shenzhenvlab` 的 `id / host / auth / remote_root / task_workdir`
3. 导出所有引用 `shenzhenvlab` 的 dataset / dataset asset / experiment / archive
4. 导出所有 demo 候选记录及其引用关系
5. 导出对应 workspace 目录列表
6. 在事务或 dry-run 模式下生成“将删除条数”报告

## 12. 方案结论

阶段3开始前，建议清掉明确的 stage1/stage2 demo seed 和 validation 派生数据，但保留真实服务器与真实远程资产，尤其必须保护 `shenzhenvlab`。

真正执行时，应采用：

- 保护白名单
- demo 候选识别
- 依赖检查
- dry-run 预览
- 分阶段删除

当前阶段不执行任何破坏性操作。
