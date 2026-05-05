from __future__ import annotations

import json
import os
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Mapping


def _parse_bool(value: Any, default: bool = False) -> bool:
    if isinstance(value, bool):
        return value
    if value is None:
        return default
    return str(value).strip().lower() in {"1", "true", "yes", "on"}


def _parse_int(value: Any, default: int) -> int:
    try:
        return int(value)
    except (TypeError, ValueError):
        return default


def _parse_json_list(value: Any) -> list[str]:
    if isinstance(value, list):
        return [str(item) for item in value]
    if not value:
        return []
    try:
        parsed = json.loads(str(value))
    except json.JSONDecodeError:
        return []
    if not isinstance(parsed, list):
        return []
    return [str(item) for item in parsed]


@dataclass
class APIExecutionConfig:
    enabled: bool = False
    allow_live_execution: bool = False
    base_url: str = ""
    api_key: str = ""
    timeout_seconds: int = 60
    config_sources: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)


@dataclass
class CodexCLIConfig:
    command: str = "codex"
    args: list[str] = field(default_factory=list)
    timeout_seconds: int = 60
    output_mode: str = "stdout"
    config_sources: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)


class RuntimeConfigLoader:
    def __init__(self, project_root: Path | None = None, environ: Mapping[str, str] | None = None) -> None:
        self.project_root = project_root or Path(__file__).resolve().parents[3]
        self.environ = dict(environ or os.environ)

    def describe_api_sources(self) -> list[str]:
        return [
            ".env / environment variables (AGENT_API_*)",
            "runtime config file (AGENT_RUNTIME_CONFIG_FILE or backend/python_agents/runtime/config.local.json)",
            "database config table hook (reserved extension point)",
        ]

    def describe_codex_sources(self) -> list[str]:
        return [
            ".env / environment variables (CODEX_CLI_*)",
            "runtime config file (AGENT_RUNTIME_CONFIG_FILE or backend/python_agents/runtime/config.local.json)",
            "database config table hook (reserved extension point)",
        ]

    def load_api_config(self) -> APIExecutionConfig:
        env_data = self._load_env_values()
        file_data = self._load_file_config().get("api", {})
        db_data = self._load_database_config().get("api", {})

        enabled = _parse_bool(env_data.get("AGENT_API_ENABLED", db_data.get("enabled", file_data.get("enabled"))), False)
        allow_live = _parse_bool(
            env_data.get("AGENT_API_ALLOW_LIVE_EXECUTION", db_data.get("allow_live_execution", file_data.get("allow_live_execution"))),
            False,
        )
        base_url = str(env_data.get("AGENT_API_BASE_URL", db_data.get("base_url", file_data.get("base_url", "")))).strip()
        api_key = str(env_data.get("AGENT_API_KEY", db_data.get("api_key", file_data.get("api_key", "")))).strip()
        timeout_seconds = _parse_int(
            env_data.get("AGENT_API_TIMEOUT_SECONDS", db_data.get("timeout_seconds", file_data.get("timeout_seconds", 60))),
            60,
        )

        warnings: list[str] = []
        if not enabled:
            warnings.append("API execution is disabled by default in stage3 foundation.")
        if enabled and not base_url:
            warnings.append("API execution is enabled but AGENT_API_BASE_URL is missing.")
        if enabled and not api_key:
            warnings.append("API execution is enabled but AGENT_API_KEY is missing.")
        if enabled and base_url and api_key and not allow_live:
            warnings.append("Live API execution remains reserved until explicitly enabled.")

        return APIExecutionConfig(
            enabled=enabled,
            allow_live_execution=allow_live,
            base_url=base_url,
            api_key=api_key,
            timeout_seconds=timeout_seconds,
            config_sources=self.describe_api_sources(),
            warnings=warnings,
        )

    def load_codex_cli_config(self) -> CodexCLIConfig:
        env_data = self._load_env_values()
        file_data = self._load_file_config().get("codex_cli", {})
        db_data = self._load_database_config().get("codex_cli", {})

        command = str(env_data.get("CODEX_CLI_BIN", db_data.get("command", file_data.get("command", "codex")))).strip() or "codex"
        args = _parse_json_list(env_data.get("CODEX_CLI_ARGS_JSON"))
        if not args:
            args = _parse_json_list(db_data.get("args"))
        if not args:
            args = _parse_json_list(file_data.get("args"))
        timeout_seconds = _parse_int(
            env_data.get("CODEX_CLI_TIMEOUT_SECONDS", db_data.get("timeout_seconds", file_data.get("timeout_seconds", 60))),
            60,
        )
        output_mode = str(env_data.get("CODEX_CLI_OUTPUT_MODE", db_data.get("output_mode", file_data.get("output_mode", "stdout")))).strip() or "stdout"
        if output_mode not in {"stdout", "file"}:
            output_mode = "stdout"

        return CodexCLIConfig(
            command=command,
            args=args,
            timeout_seconds=timeout_seconds,
            output_mode=output_mode,
            config_sources=self.describe_codex_sources(),
            warnings=[],
        )

    def _load_env_values(self) -> dict[str, Any]:
        values = self._load_dotenv_file()
        values.update(self.environ)
        return values

    def _load_dotenv_file(self) -> dict[str, str]:
        env_path = self.project_root / ".env"
        if not env_path.exists():
            return {}

        result: dict[str, str] = {}
        for raw_line in env_path.read_text(encoding="utf-8-sig").splitlines():
            line = raw_line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            result[key.strip()] = value.strip().strip('"').strip("'")
        return result

    def _load_file_config(self) -> dict[str, Any]:
        configured = self.environ.get("AGENT_RUNTIME_CONFIG_FILE", "").strip()
        config_path = Path(configured) if configured else self.project_root / "backend" / "python_agents" / "runtime" / "config.local.json"
        if not config_path.exists():
            return {}
        try:
            payload = json.loads(config_path.read_text(encoding="utf-8-sig"))
        except json.JSONDecodeError:
            return {}
        return payload if isinstance(payload, dict) else {}

    def _load_database_config(self) -> dict[str, Any]:
        raw = self.environ.get("AGENT_RUNTIME_DB_CONFIG_JSON", "").strip()
        if not raw:
            return {}
        try:
            payload = json.loads(raw)
        except json.JSONDecodeError:
            return {}
        return payload if isinstance(payload, dict) else {}
