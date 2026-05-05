# Stage3 Agent Runtime

This directory contains the stage3 controlled-agent runtime skeleton.

Current scope:

- unified agent input contract
- unified agent output contract
- BaseAgent / BaseExecutor / BaseValidator / BaseRepairer
- execution adapters for `api` / `codex_cli` / `mock`
- a runtime runner that validates the contract and normalizes every executor result
- schema registry + schema validator + output normalizer + repair layer

Executor notes:

- `MockExecutor` returns deterministic synthetic output for stable testing.
- `CodexCLIExecutor` checks whether the Codex CLI command exists, records stdout/stderr, supports timeout, and falls back to `MockExecutor` with a warning when unavailable.
- `ApiExecutor` reserves `.env`, runtime config file, and database-config hook extension points without hardcoding any API key. Live API calls remain intentionally disabled in this foundation task.
- every runtime result is validated, optionally repaired, then validated again before it is returned to Go.

Non-goals for this layer:

- concrete Reader / Insight / Planner / Coding agent business logic
- model-specific prompting logic
- uncontrolled free-form chat behavior
