from __future__ import annotations

import json
import shutil
import subprocess
from pathlib import Path
from typing import Any

try:
    from .base import BaseExecutor
    from .config import APIExecutionConfig, CodexCLIConfig, RuntimeConfigLoader
    from .contract import AgentArtifactManifestItem, AgentRuntimeInput, AgentRuntimeOutput, AgentToolUsage
except ImportError:  # pragma: no cover - supports direct script execution
    from base import BaseExecutor
    from config import APIExecutionConfig, CodexCLIConfig, RuntimeConfigLoader
    from contract import AgentArtifactManifestItem, AgentRuntimeInput, AgentRuntimeOutput, AgentToolUsage


def _safe_json_loads(text: str) -> Any:
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        return None


def _write_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def _read_text(path: Path) -> str:
    if not path.exists():
        return ""
    return path.read_text(encoding="utf-8-sig")


def build_prompt(contract: AgentRuntimeInput) -> str:
    refs = [
        {
            "ref_type": item.ref_type,
            "ref_id": item.ref_id,
            "ref_path": item.ref_path,
            "ref_version": item.ref_version,
        }
        for item in contract.input_refs
    ]
    return (
        "You are a controlled MRAG stage3 runtime executor.\n"
        "Return a concise result for the current agent contract.\n"
        "Prefer JSON when possible.\n\n"
        f"job_id: {contract.job_id}\n"
        f"agent_type: {contract.agent_type}\n"
        f"execution_mode: {contract.execution_mode}\n"
        f"output_schema_ref: {contract.output_schema_ref}\n"
        f"model_provider: {contract.model_provider}\n"
        f"model_name: {contract.model_name}\n"
        f"prompt_version: {contract.prompt_version}\n"
        f"input_refs: {json.dumps(refs, ensure_ascii=False)}\n"
        f"skill_refs: {json.dumps(contract.skill_refs, ensure_ascii=False)}\n"
        f"tool_refs: {json.dumps(contract.tool_refs, ensure_ascii=False)}\n"
        f"memory_refs: {json.dumps(contract.memory_refs, ensure_ascii=False)}\n"
        f"metadata: {json.dumps(contract.metadata, ensure_ascii=False)}\n"
    )


def _artifact_items(items: list[tuple[str, str, Path, dict[str, Any]]]) -> list[AgentArtifactManifestItem]:
    manifests: list[AgentArtifactManifestItem] = []
    for artifact_type, name, path, metadata in items:
        if path.exists():
            manifests.append(
                AgentArtifactManifestItem(
                    artifact_type=artifact_type,
                    name=name,
                    file_path=str(path),
                    metadata=metadata,
                )
            )
    return manifests


class MockExecutor(BaseExecutor):
    name = "mock"

    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        return {"prompt_text": build_prompt(contract)}

    def execute(self, prepared_request: dict[str, Any], contract: AgentRuntimeInput) -> dict[str, Any]:
        return {
            "response_text": (
                f"stable mock response for agent={contract.agent_type}, "
                f"job={contract.job_id}, schema={contract.output_schema_ref}"
            ),
            "response_json": {
                "result": "stable_mock_response",
                "job_id": contract.job_id,
                "agent_type": contract.agent_type,
                "input_ref_count": len(contract.input_refs),
                "tool_ref_count": len(contract.tool_refs),
                "skill_ref_count": len(contract.skill_refs),
                "memory_ref_count": len(contract.memory_refs),
            },
        }

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        normalized_payload = self.base_payload(contract, "mock")
        normalized_payload.update(
            {
                "summary": f"Stable mock output for {contract.agent_type}.",
                "items": [],
                "data": collected_response["response_json"],
                "metadata": {"synthetic": True, "source": "mock_executor"},
                "response_text": collected_response["response_text"],
                "response_json": collected_response["response_json"],
            }
        )
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[
                AgentToolUsage(
                    tool_ref="mock_executor",
                    status="succeeded",
                    summary="Returned stable mock output.",
                    metadata={"executor": self.__class__.__name__},
                )
            ],
            warnings=["mock executor used; response is synthetic but stable."],
            error_message="",
        )


class ApiExecutor(BaseExecutor):
    name = "api"

    def __init__(self, config_loader: RuntimeConfigLoader | None = None) -> None:
        self.config_loader = config_loader or RuntimeConfigLoader()

    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        config = self.config_loader.load_api_config()
        return {"config": config, "prompt_text": build_prompt(contract)}

    def execute(self, prepared_request: dict[str, Any], contract: AgentRuntimeInput) -> dict[str, Any]:
        config: APIExecutionConfig = prepared_request["config"]
        if not config.enabled:
            return {
                "ok": False,
                "reason": "api_execution_disabled",
                "message": "API execution is disabled. Configure AGENT_API_* via .env, runtime config file, or database config hook.",
            }
        if not config.base_url or not config.api_key:
            return {
                "ok": False,
                "reason": "api_configuration_missing",
                "message": "API execution is enabled but base URL or API key is missing.",
            }
        if not config.allow_live_execution:
            return {
                "ok": False,
                "reason": "api_live_execution_reserved",
                "message": "Live API execution remains reserved in this stage3 foundation build.",
            }
        return {
            "ok": False,
            "reason": "api_live_execution_not_implemented",
            "message": "API request structure is reserved, but live HTTP execution is not implemented in this task.",
        }

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        config: APIExecutionConfig = prepared_request["config"]
        normalized_payload = self.base_payload(contract, "api")
        normalized_payload.update(
            {
                "summary": execution_result["message"],
                "items": [],
                "data": {
                    "reason": execution_result["reason"],
                    "api_enabled": config.enabled,
                    "api_key_configured": bool(config.api_key),
                    "api_base_url_configured": bool(config.base_url),
                },
                "metadata": {"config_sources": config.config_sources},
                "api_enabled": config.enabled,
                "api_key_configured": bool(config.api_key),
                "api_base_url_configured": bool(config.base_url),
                "config_sources": config.config_sources,
            }
        )

        message = (
            f"{execution_result['message']} Supported configuration sources: "
            + ", ".join(config.config_sources)
        )
        return AgentRuntimeOutput(
            status="failed",
            normalized_payload=normalized_payload,
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[
                AgentToolUsage(
                    tool_ref="api_executor",
                    status="skipped",
                    summary=execution_result["reason"],
                    metadata={"config_sources": config.config_sources},
                )
            ],
            warnings=config.warnings,
            error_message=message,
        )


class CodexCLIExecutor(BaseExecutor):
    name = "codex_cli"

    def __init__(
        self,
        config_loader: RuntimeConfigLoader | None = None,
        fallback_executor: BaseExecutor | None = None,
    ) -> None:
        self.config_loader = config_loader or RuntimeConfigLoader()
        self.fallback_executor = fallback_executor or MockExecutor()

    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        config = self.config_loader.load_codex_cli_config()
        project_root = self.config_loader.project_root
        workspace_dir = Path(contract.workspace_dir or Path.cwd() / "workspace" / "agents" / "jobs" / contract.job_id)
        codex_dir = workspace_dir / "codex_cli"
        prompt_path = codex_dir / "prompt.txt"
        output_path = codex_dir / "response.txt"
        stdout_path = codex_dir / "stdout.txt"
        stderr_path = codex_dir / "stderr.txt"
        prompt_text = build_prompt(contract)
        _write_text(prompt_path, prompt_text)

        raw_args = config.args
        args = [
            item.format(
                prompt_file=str(prompt_path),
                output_file=str(output_path),
                workspace_dir=str(workspace_dir),
                project_root=str(project_root),
                agent_type=contract.agent_type,
                job_id=contract.job_id,
                model_name=contract.model_name,
                prompt_text=prompt_text,
            )
            for item in raw_args
        ]
        uses_prompt_file = any("{prompt_file}" in item for item in raw_args)
        uses_prompt_text_arg = any("{prompt_text}" in item for item in raw_args)
        uses_output_file = any("{output_file}" in item for item in raw_args) or config.output_mode == "file"

        return {
            "config": config,
            "project_root": project_root,
            "workspace_dir": workspace_dir,
            "command": config.command,
            "args": args,
            "prompt_text": prompt_text,
            "prompt_path": prompt_path,
            "output_path": output_path,
            "stdout_path": stdout_path,
            "stderr_path": stderr_path,
            "uses_prompt_file": uses_prompt_file,
            "uses_prompt_text_arg": uses_prompt_text_arg,
            "uses_output_file": uses_output_file,
        }

    def execute(self, prepared_request: dict[str, Any], contract: AgentRuntimeInput) -> dict[str, Any]:
        config: CodexCLIConfig = prepared_request["config"]
        command = prepared_request["command"]
        if shutil.which(command) is None:
            return {
                "ok": False,
                "reason": "codex_cli_not_found",
                "message": f"Codex CLI command '{command}' was not found. Falling back to mock executor.",
            }

        cmd = [command, *prepared_request["args"]]
        stdin_payload = None if prepared_request["uses_prompt_file"] or prepared_request["uses_prompt_text_arg"] else prepared_request["prompt_text"]
        try:
            completed = subprocess.run(
                cmd,
                input=stdin_payload,
                text=True,
                capture_output=True,
                encoding="utf-8",
                errors="replace",
                timeout=config.timeout_seconds,
                cwd=str(prepared_request["workspace_dir"]),
            )
        except subprocess.TimeoutExpired as exc:
            _write_text(prepared_request["stdout_path"], exc.stdout or "")
            _write_text(prepared_request["stderr_path"], exc.stderr or "")
            return {
                "ok": False,
                "reason": "codex_cli_timeout",
                "message": f"Codex CLI timed out after {config.timeout_seconds} seconds. Falling back to mock executor.",
            }
        except OSError as exc:
            return {
                "ok": False,
                "reason": "codex_cli_os_error",
                "message": f"Codex CLI execution failed: {exc}. Falling back to mock executor.",
            }

        _write_text(prepared_request["stdout_path"], completed.stdout)
        _write_text(prepared_request["stderr_path"], completed.stderr)

        if completed.returncode != 0:
            return {
                "ok": False,
                "reason": "codex_cli_nonzero_exit",
                "message": f"Codex CLI exited with code {completed.returncode}. Falling back to mock executor.",
                "returncode": completed.returncode,
            }

        return {
            "ok": True,
            "returncode": completed.returncode,
            "stdout": completed.stdout,
            "stderr": completed.stderr,
        }

    def collect_response(
        self,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        contract: AgentRuntimeInput,
    ) -> dict[str, Any]:
        if not execution_result.get("ok"):
            fallback_output = self.fallback_executor.run(contract)
            return {
                "fallback_output": fallback_output,
                "fallback_reason": execution_result["reason"],
                "fallback_message": execution_result["message"],
            }

        output_path: Path = prepared_request["output_path"]
        response_text = _read_text(output_path) if prepared_request["uses_output_file"] else execution_result.get("stdout", "")
        if not response_text:
            response_text = execution_result.get("stdout", "")

        return {
            "response_text": response_text,
            "response_json": _safe_json_loads(response_text),
        }

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        config: CodexCLIConfig = prepared_request["config"]
        artifact_manifest = _artifact_items(
            [
                ("codex_prompt", prepared_request["prompt_path"].name, prepared_request["prompt_path"], {"role": "codex_cli_prompt"}),
                ("codex_stdout", prepared_request["stdout_path"].name, prepared_request["stdout_path"], {"role": "codex_cli_stdout"}),
                ("codex_stderr", prepared_request["stderr_path"].name, prepared_request["stderr_path"], {"role": "codex_cli_stderr"}),
                ("codex_output", prepared_request["output_path"].name, prepared_request["output_path"], {"role": "codex_cli_output"}),
            ]
        )

        if "fallback_output" in collected_response:
            fallback_output: AgentRuntimeOutput = collected_response["fallback_output"]
            normalized_payload = dict(fallback_output.normalized_payload)
            normalized_payload.update(
                {
                    "execution_mode_requested": "codex_cli",
                    "execution_mode_used": "mock",
                    "executor": self.__class__.__name__,
                    "summary": normalized_payload.get("summary", f"Fallback mock output for {contract.agent_type}."),
                    "items": normalized_payload.get("items", []),
                    "data": normalized_payload.get("data", {}),
                    "metadata": {
                        **(normalized_payload.get("metadata") if isinstance(normalized_payload.get("metadata"), dict) else {}),
                        "fallback": True,
                        "config_sources": config.config_sources,
                    },
                    "codex_cli_command": prepared_request["command"],
                    "codex_cli_args": prepared_request["args"],
                    "codex_cli_available": False,
                    "fallback_reason": collected_response["fallback_reason"],
                }
            )
            warnings = list(fallback_output.warnings)
            warnings.append(collected_response["fallback_message"])
            warnings.extend(config.warnings)
            return AgentRuntimeOutput(
                status=fallback_output.status,
                normalized_payload=normalized_payload,
                artifact_manifest=artifact_manifest + fallback_output.artifact_manifest,
                repair_actions=fallback_output.repair_actions,
                tool_usages=[
                    AgentToolUsage(
                        tool_ref="codex_cli",
                        status="fallback",
                        summary=collected_response["fallback_reason"],
                        metadata={
                            "command": prepared_request["command"],
                            "args": prepared_request["args"],
                            "config_sources": config.config_sources,
                        },
                    ),
                    *fallback_output.tool_usages,
                ],
                warnings=warnings,
                error_message=fallback_output.error_message,
            )

        normalized_payload = self.base_payload(contract, "codex_cli")
        normalized_payload.update(
            {
                "summary": collected_response["response_text"].strip() or f"Codex CLI output for {contract.agent_type}.",
                "items": [],
                "data": collected_response["response_json"] if isinstance(collected_response["response_json"], dict) else {},
                "metadata": {
                    "codex_cli_output_mode": config.output_mode,
                    "config_sources": config.config_sources,
                },
                "response_text": collected_response["response_text"],
                "response_json": collected_response["response_json"],
                "codex_cli_command": prepared_request["command"],
                "codex_cli_args": prepared_request["args"],
                "codex_cli_output_mode": config.output_mode,
            }
        )
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=artifact_manifest,
            repair_actions=[],
            tool_usages=[
                AgentToolUsage(
                    tool_ref="codex_cli",
                    status="succeeded",
                    summary="Executed Codex CLI successfully.",
                    metadata={
                        "command": prepared_request["command"],
                        "args": prepared_request["args"],
                        "config_sources": config.config_sources,
                    },
                )
            ],
            warnings=config.warnings,
            error_message="",
        )
