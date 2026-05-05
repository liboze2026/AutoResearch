# Stage2 Validation Report

## Scope

- Validation date: 2026-03-25
- Validator: Codex
- Backend entrypoint: `docker compose run ... go-backend`
- Frontend entrypoint: `npm run dev -- --host 127.0.0.1 --port 4173`
- Validation script: `bash ./scripts/validate_stage2.sh`

## Validation Result

阶段2自动验收已通过。

本次实际通过的检查范围包括：

- 后端启动正常
- 前端启动正常
- 服务器 heartbeat 可记录
- GPU snapshot 可记录
- experiment 可创建
- experiment spec 可生成
- run 可排队
- run 可调度
- run 可启动
- 日志可查询
- 模拟失败可重试
- comparison 可生成
- 前端关键路由可访问

## Actual PASS Output

```text
PASS: stage2 validation passed
- steps:
  - validate_stage2_01_boot.sh
  - validate_stage2_02_assets.sh
  - validate_stage2_03_lifecycle.sh
  - validate_stage2_04_recovery_frontend.sh
```

## Split Validation Steps

### 1. validate_stage2_01_boot.sh

验证：

- PostgreSQL 启动
- 后端阶段2相关包测试
- 前端基础测试
- 后端容器启动
- 前端 dev server 启动

### 2. validate_stage2_02_assets.sh

验证：

- 创建 idea
- 创建 dataset asset
- 创建 baseline
- 创建 result archive
- 创建 server
- heartbeat 入库
- GPU snapshot 入库

### 3. validate_stage2_03_lifecycle.sh

验证：

- 创建 experiment
- 生成 experiment spec
- queue run
- schedule run
- start run
- 查询 logs
- 生成 comparison

### 4. validate_stage2_04_recovery_frontend.sh

验证：

- 构造失败 run
- 查询 recovery 信息
- retry 生成新 run
- 前端关键路由返回 200

## Key Fixes Completed During Validation

为使阶段2验收真正通过，这次收尾修复了这些阻碍项：

- 修复了验证脚本在 WSL 下的 POST / JSON 解析问题
- 修复了 mock GPU probe 返回非法 JSON 导致 `gpu-snapshot` 失败的问题
- 补齐了统一训练模板目录在容器中的挂载与配置
- 修复了 recovery 验证脚本的失败 run ID 幂等性问题
- 将总验收脚本收束为分段脚本编排，便于定位问题和复跑

## Created Objects From Final PASS Run

- server_id: `srv_1774414985_df475f4e`
- experiment_id: `exp_1774414991_08f98cea`
- run_id: `run_1774414994_10d34582`
- retry_run_id: `run_1774415014_6a893ccb`
- result_archive_id: `archive_1774414999_a04ca89f`
- comparison_count: `4`

## Conclusion

按阶段2当前定义，系统已经达到“可运行、可恢复、可审计、可查看”的最小验收标准，可以视为阶段2完成。

同时也要明确：

- 当前统一训练模板仍是最小版本
- 当前恢复机制仍是最小重试版本
- 当前结果对比还不是自动科研决策器
