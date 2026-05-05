from __future__ import annotations

import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

try:
    from .base import BaseAgent, BaseExecutor, BaseRepairer, BaseValidator
    from .coding_phase4_logic import (
        build_generation_plan,
        build_phase4_config,
        build_retry_plan,
        render_method_module,
        slugify,
    )
    from .contract import AgentArtifactManifestItem, AgentRepairAction, AgentToolUsage, AgentRuntimeInput, AgentRuntimeOutput
    from .executors import ApiExecutor, CodexCLIExecutor, MockExecutor
except ImportError:  # pragma: no cover
    from base import BaseAgent, BaseExecutor, BaseRepairer, BaseValidator
    from coding_phase4_logic import build_generation_plan, build_phase4_config, build_retry_plan, render_method_module, slugify
    from contract import AgentArtifactManifestItem, AgentRepairAction, AgentToolUsage, AgentRuntimeInput, AgentRuntimeOutput
    from executors import ApiExecutor, CodexCLIExecutor, MockExecutor


CODING_PHASE4_REQUIRED_FIELDS = (
    "protocol_version",
    "phase4_run_manifest",
    "phase4_config",
    "method_module",
    "retry_plan",
    "dataset_tool_assets",
    "evaluate_tool_assets",
    "entrypoints",
)


@dataclass
class CodingPhase4Request:
    run_manifest_id: str
    runner_mode: str
    max_retry_count: int
    user_notes: str
    dataset_profile: dict[str, Any]
    idea: dict[str, Any]
    reader_context: dict[str, Any]

    @classmethod
    def from_metadata(cls, metadata: dict[str, Any]) -> "CodingPhase4Request":
        max_retry_count = int(metadata.get("max_retry_count", 3) or 3)
        if max_retry_count <= 0:
            max_retry_count = 3
        return cls(
            run_manifest_id=str(metadata.get("run_manifest_id", "")).strip(),
            runner_mode=str(metadata.get("runner_mode", "")).strip() or "local_dummy",
            max_retry_count=max_retry_count,
            user_notes=str(metadata.get("user_notes", "")).strip(),
            dataset_profile=dict(metadata.get("dataset_profile") or {}),
            idea=dict(metadata.get("idea") or {}),
            reader_context=dict(metadata.get("reader_context") or {}),
        )


class CodingPhase4Executor(BaseExecutor):
    name = "coding_phase4"

    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        request = CodingPhase4Request.from_metadata(contract.metadata)
        workspace_dir = Path(contract.workspace_dir or Path.cwd() / "workspace" / "agents" / "jobs" / contract.job_id)
        workspace_dir.mkdir(parents=True, exist_ok=True)
        return {
            "request": request,
            "workspace_dir": workspace_dir,
            "request_path": workspace_dir / "coding_phase4_request.json",
            "plan_path": workspace_dir / "coding_phase4_plan.json",
            "generation_trace_path": workspace_dir / "coding_phase4_generation_trace.json",
        }

    def execute(self, prepared_request: dict[str, Any], contract: AgentRuntimeInput) -> dict[str, Any]:
        request: CodingPhase4Request = prepared_request["request"]
        idea_title = str(request.idea.get("title", "")).strip()
        generation_backend = _run_generation_backend(contract)
        method_slug = slugify(idea_title or str(request.idea.get("coreMethod", "")).strip() or "generated_method")
        plan = build_generation_plan(
            request,
            execution_mode_used=str(generation_backend["execution_mode_used"] or contract.execution_mode or "mock"),
            response_text=str(generation_backend.get("response_text", "") or ""),
        )
        if not plan.method_slug:
            plan.method_slug = method_slug
        method_content = render_method_module(plan)
        config = build_phase4_config(request, plan)
        branch_name = plan.branch_name
        method_relative_path = plan.method_relative_path
        manifest_patch = {
            "protocol_version": "phase4-retrieval-mainline-v1",
            "runner_mode": request.runner_mode,
            "target_granularity": "page",
            "supports_method_branch": True,
            "supports_retry_repair": True,
        }
        plan_payload = {
            "method_slug": plan.method_slug,
            "branch_name": branch_name,
            "method_relative_path": method_relative_path,
            "config": config,
            "manifest_patch": manifest_patch,
            "generation_trace": generation_backend["trace"],
        }
        _write_json(prepared_request["request_path"], contract.metadata)
        _write_json(prepared_request["plan_path"], plan_payload)
        _write_json(prepared_request["generation_trace_path"], generation_backend["trace"])
        return {
            "request": request,
            "method_slug": plan.method_slug,
            "branch_name": branch_name,
            "method_relative_path": method_relative_path,
            "method_content": method_content,
            "config": config,
            "manifest_patch": manifest_patch,
            "generation_backend": generation_backend,
        }

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request: CodingPhase4Request = execution_result["request"]
        normalized_payload = self.base_payload(contract, contract.execution_mode)
        normalized_payload["workspace_dir"] = str(prepared_request["workspace_dir"])
        used_mode = str(execution_result["generation_backend"]["execution_mode_used"] or contract.execution_mode or "mock")
        normalized_payload["execution_mode_used"] = used_mode
        normalized_payload.update(
            {
                "summary": f"Prepared phase4 coding protocol and generated method module for {request.idea.get('title', '') or request.dataset_profile.get('datasetName', '')}.",
                "items": [
                    {
                        "type": "method_module",
                        "module_name": execution_result["method_slug"],
                        "relative_path": execution_result["method_relative_path"],
                    }
                ],
                "protocol_version": "phase4-retrieval-mainline-v1",
                "phase4_run_manifest": execution_result["manifest_patch"],
                "phase4_config": execution_result["config"],
                "method_module": {
                    "module_name": execution_result["method_slug"],
                    "relative_path": execution_result["method_relative_path"],
                    "branch_name": execution_result["branch_name"],
                    "summary": str(request.idea.get("coreMethod", "")).strip() or str(request.idea.get("title", "")).strip(),
                    "content": execution_result["method_content"],
                    "metadata": {
                        "idea_id": str(request.idea.get("id", "")).strip(),
                        "dataset_profile_id": str(request.dataset_profile.get("id", "")).strip(),
                    },
                },
                "retry_plan": execution_result["config"]["retry_policy"],
                "dataset_tool_assets": {
                    "dataset_profile_id": str(request.dataset_profile.get("id", "")).strip(),
                    "adapter_contract": "dataset-adapter-v1",
                },
                "evaluate_tool_assets": {
                    "primary_metric": str(request.dataset_profile.get("officialMetric", "")).strip() or "recall@5",
                    "tool": "evaluate_tool",
                },
                "entrypoints": {
                    "run": "run_entrypoint.py",
                    "eval": "eval_entrypoint.py",
                    "bootstrap": "bootstrap_env.sh",
                },
                "data": {
                    "run_manifest_id": request.run_manifest_id,
                    "runner_mode": request.runner_mode,
                    "method_branch": execution_result["branch_name"],
                },
                "metadata": {
                    "dataset_name": str(request.dataset_profile.get("datasetName", "")).strip(),
                    "idea_title": str(request.idea.get("title", "")).strip(),
                    "task_definition": str(request.reader_context.get("task_definition", "")).strip(),
                },
                "generation_trace": execution_result["generation_backend"]["trace"],
            }
        )
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=[
                AgentArtifactManifestItem("coding_phase4_request", prepared_request["request_path"].name, str(prepared_request["request_path"]), {"role": "coding_phase4_request"}),
                AgentArtifactManifestItem("coding_phase4_plan", prepared_request["plan_path"].name, str(prepared_request["plan_path"]), {"role": "coding_phase4_plan"}),
                AgentArtifactManifestItem("coding_phase4_generation_trace", prepared_request["generation_trace_path"].name, str(prepared_request["generation_trace_path"]), {"role": "coding_phase4_generation_trace"}),
                *execution_result["generation_backend"]["artifact_manifest"],
            ],
            repair_actions=[],
            tool_usages=[
                *execution_result["generation_backend"]["tool_usages"],
                AgentToolUsage("dataset_tool", "planned", "Dataset tool assets will be materialized during phase4 run execution.", {"contract": "dataset-adapter-v1"}),
                AgentToolUsage("evaluate_tool", "planned", "Evaluate tool assets will be materialized during phase4 eval execution.", {"primary_metric": str(request.dataset_profile.get("officialMetric", "")).strip() or "recall@5"}),
            ],
            warnings=list(execution_result["generation_backend"]["warnings"]),
            error_message="",
        )


class CodingPhase4Validator(BaseValidator):
    def validate_input(self, contract: AgentRuntimeInput) -> list[str]:
        errors = super().validate_input(contract)
        request = CodingPhase4Request.from_metadata(contract.metadata)
        if not request.run_manifest_id:
            errors.append("run_manifest_id is required")
        if not str(request.dataset_profile.get("id", "")).strip():
            errors.append("dataset_profile.id is required")
        if not str(request.dataset_profile.get("datasetName", "")).strip():
            errors.append("dataset_profile.datasetName is required")
        if not str(request.idea.get("id", "")).strip():
            errors.append("idea.id is required")
        if not str(request.idea.get("title", "")).strip() and not str(request.idea.get("coreMethod", "")).strip():
            errors.append("idea.title or idea.coreMethod is required")
        return errors

    def validate_payload(self, contract: AgentRuntimeInput, payload: dict[str, Any] | None) -> list[str]:
        errors = super().validate_payload(contract, payload)
        if payload is None:
            return errors
        for field_name in CODING_PHASE4_REQUIRED_FIELDS:
            if field_name not in payload:
                errors.append(f"normalized_payload.{field_name} is required")
        for object_name in ("phase4_run_manifest", "phase4_config", "method_module", "retry_plan", "dataset_tool_assets", "evaluate_tool_assets", "entrypoints"):
            if object_name in payload and not isinstance(payload.get(object_name), dict):
                errors.append(f"normalized_payload.{object_name} must be an object")
        return errors


class CodingPhase4Repairer(BaseRepairer):
    def repair(
        self,
        contract: AgentRuntimeInput,
        output: AgentRuntimeOutput,
        errors: list[str],
    ) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        repaired, actions = super().repair(contract, output, errors)
        executor = CodingPhase4Executor()
        prepared_request = executor.prepare_request(contract)
        execution_result = executor.execute(prepared_request, contract)
        rebuilt = executor.normalize_output(contract, prepared_request, execution_result, execution_result)
        repaired.normalized_payload = rebuilt.normalized_payload
        repaired.artifact_manifest = rebuilt.artifact_manifest
        repaired.tool_usages = rebuilt.tool_usages
        actions.append(
            AgentRepairAction(
                action="rebuild_phase4_coding_payload",
                status="applied",
                detail="Rebuilt phase4 coding payload, method module, and protocol assets from request metadata.",
                metadata={"run_manifest_id": CodingPhase4Request.from_metadata(contract.metadata).run_manifest_id},
            )
        )
        return repaired, actions


class CodingPhase4Agent(BaseAgent):
    pass


def build_coding_phase4_agent(contract: AgentRuntimeInput) -> CodingPhase4Agent:
    return CodingPhase4Agent(
        executor=CodingPhase4Executor(),
        validator=CodingPhase4Validator(),
        repairer=CodingPhase4Repairer(),
    )


def _write_json(path: Path, payload: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

def _run_generation_backend(contract: AgentRuntimeInput) -> dict[str, Any]:
    trace = {
        "execution_mode_requested": contract.execution_mode,
        "execution_mode_used": contract.execution_mode or "mock",
        "backend_executor": "",
        "status": "skipped",
        "warnings": [],
        "response_preview": "",
    }
    if contract.execution_mode == "mock":
        trace["backend_executor"] = "deterministic_mock"
        trace["status"] = "succeeded"
        return {
            "execution_mode_used": "mock",
            "trace": trace,
            "artifact_manifest": [],
            "tool_usages": [
                AgentToolUsage(
                    "coding_generation_backend",
                    "succeeded",
                    "Used deterministic mock generation for phase4 coding.",
                    {"backend_executor": "deterministic_mock"},
                )
            ],
            "warnings": [],
            "response_text": "",
        }

    delegate_contract = AgentRuntimeInput(
        job_id=contract.job_id,
        agent_type=contract.agent_type,
        execution_mode=contract.execution_mode,
        model_provider=contract.model_provider,
        model_name=contract.model_name,
        prompt_version=contract.prompt_version,
        input_refs=list(contract.input_refs),
        output_schema_ref="schemas/generic-agent-output-v1.json",
        skill_refs=list(contract.skill_refs),
        tool_refs=list(contract.tool_refs),
        memory_refs=list(contract.memory_refs),
        workspace_dir=contract.workspace_dir,
        metadata={
            **dict(contract.metadata or {}),
            "coding_instruction": (
                "Return compact guidance for a retrieval method module. "
                "Prefer JSON keys: retrieval_strategy, query_expansion_terms, scoring_hints, repair_priority."
            ),
        },
    )
    delegate = CodexCLIExecutor() if contract.execution_mode == "codex_cli" else ApiExecutor()
    delegate_output = delegate.run(delegate_contract)
    response_text = str(delegate_output.normalized_payload.get("response_text", "") or delegate_output.error_message or "").strip()
    used_mode = str(delegate_output.normalized_payload.get("execution_mode_used", "") or contract.execution_mode or "mock")
    warnings = list(delegate_output.warnings)
    tool_status = "succeeded" if delegate_output.status == "succeeded" else "fallback"
    if delegate_output.status != "succeeded":
        warnings.append(
            f"{contract.execution_mode} generation backend did not return a usable coding draft; deterministic fallback was applied."
        )
        used_mode = "mock"
        tool_status = "fallback"
    trace.update(
        {
            "execution_mode_used": used_mode,
            "backend_executor": delegate_output.normalized_payload.get("executor", delegate.__class__.__name__),
            "status": delegate_output.status,
            "warnings": warnings,
            "response_preview": response_text[:240],
        }
    )
    return {
        "execution_mode_used": used_mode,
        "trace": trace,
        "artifact_manifest": list(delegate_output.artifact_manifest),
        "tool_usages": [
            AgentToolUsage(
                "coding_generation_backend",
                tool_status,
                "Used runtime generation backend for phase4 coding planning.",
                {
                    "backend_executor": trace["backend_executor"],
                    "execution_mode_requested": contract.execution_mode,
                    "execution_mode_used": used_mode,
                },
            ),
            *list(delegate_output.tool_usages),
        ],
        "warnings": warnings,
        "response_text": response_text,
    }

