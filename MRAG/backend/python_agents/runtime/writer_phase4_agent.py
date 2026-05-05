from __future__ import annotations

import json
from pathlib import Path
from typing import Any

try:
    from .base import BaseAgent, BaseExecutor, BaseRepairer, BaseValidator
    from .contract import AgentArtifactManifestItem, AgentRepairAction, AgentToolUsage, AgentRuntimeInput, AgentRuntimeOutput
    from .executors import ApiExecutor, CodexCLIExecutor, MockExecutor
    from .writer_phase4_logic import (
        Phase4WriterRequest,
        build_machine_readable_report,
        merge_citations,
        render_human_readable_report,
    )
except ImportError:  # pragma: no cover
    from base import BaseAgent, BaseExecutor, BaseRepairer, BaseValidator
    from contract import AgentArtifactManifestItem, AgentRepairAction, AgentToolUsage, AgentRuntimeInput, AgentRuntimeOutput
    from executors import ApiExecutor, CodexCLIExecutor, MockExecutor
    from writer_phase4_logic import Phase4WriterRequest, build_machine_readable_report, merge_citations, render_human_readable_report


WRITER_PHASE4_REQUIRED_FIELDS = (
    "report_title",
    "machine_readable_report",
    "human_readable_report_md",
    "citation_refs",
    "reference_source_ids",
)


class WriterPhase4Executor(BaseExecutor):
    name = "writer_phase4"

    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        request = Phase4WriterRequest.from_metadata(contract.metadata)
        workspace_dir = Path(contract.workspace_dir or Path.cwd() / "workspace" / "agents" / "jobs" / contract.job_id)
        workspace_dir.mkdir(parents=True, exist_ok=True)
        return {
            "request": request,
            "workspace_dir": workspace_dir,
            "request_path": workspace_dir / "writer_phase4_request.json",
            "machine_report_path": workspace_dir / "writer_phase4_machine_report.json",
            "human_report_path": workspace_dir / "writer_phase4_human_report.md",
            "generation_trace_path": workspace_dir / "writer_phase4_generation_trace.json",
        }

    def execute(self, prepared_request: dict[str, Any], contract: AgentRuntimeInput) -> dict[str, Any]:
        request: Phase4WriterRequest = prepared_request["request"]
        generation_backend = _run_generation_backend(contract)
        machine_report = build_machine_readable_report(
            request,
            execution_mode_used=str(generation_backend["execution_mode_used"] or contract.execution_mode or "mock"),
            response_text=str(generation_backend.get("response_text", "") or ""),
        )
        human_report_md = render_human_readable_report(machine_report)
        _write_json(prepared_request["request_path"], contract.metadata)
        _write_json(prepared_request["machine_report_path"], machine_report)
        prepared_request["human_report_path"].write_text(human_report_md, encoding="utf-8")
        _write_json(prepared_request["generation_trace_path"], generation_backend["trace"])
        return {
            "request": request,
            "machine_report": machine_report,
            "human_report_md": human_report_md,
            "generation_backend": generation_backend,
        }

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        machine_report = execution_result["machine_report"]
        normalized_payload = self.base_payload(contract, contract.execution_mode)
        normalized_payload["workspace_dir"] = str(prepared_request["workspace_dir"])
        normalized_payload["execution_mode_used"] = execution_result["generation_backend"]["execution_mode_used"]
        normalized_payload.update(
            {
                "summary": f"Prepared phase4 structured report for run {machine_report['run_config']['run_manifest_id']}.",
                "items": [
                    machine_report["report_title"],
                    f"{len(machine_report['citations'])} citation(s)",
                ],
                "report_title": machine_report["report_title"],
                "machine_readable_report": machine_report,
                "human_readable_report_md": execution_result["human_report_md"],
                "citation_refs": machine_report["citation_refs"],
                "reference_source_ids": machine_report["reference_source_ids"],
                "data": {
                    "run_manifest_id": machine_report["run_config"]["run_manifest_id"],
                    "report_version": machine_report["report_version"],
                },
                "metadata": {
                    "dataset_name": machine_report["dataset"]["name"],
                    "idea_title": machine_report["idea"]["title"],
                    "run_status": machine_report["run_config"]["status"],
                },
                "generation_trace": execution_result["generation_backend"]["trace"],
            }
        )
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=[
                AgentArtifactManifestItem("writer_phase4_request", prepared_request["request_path"].name, str(prepared_request["request_path"]), {"role": "writer_phase4_request"}),
                AgentArtifactManifestItem("writer_phase4_machine_report", prepared_request["machine_report_path"].name, str(prepared_request["machine_report_path"]), {"role": "writer_phase4_machine_report"}),
                AgentArtifactManifestItem("writer_phase4_human_report", prepared_request["human_report_path"].name, str(prepared_request["human_report_path"]), {"role": "writer_phase4_human_report"}),
                AgentArtifactManifestItem("writer_phase4_generation_trace", prepared_request["generation_trace_path"].name, str(prepared_request["generation_trace_path"]), {"role": "writer_phase4_generation_trace"}),
                *execution_result["generation_backend"]["artifact_manifest"],
            ],
            repair_actions=[],
            tool_usages=[
                *execution_result["generation_backend"]["tool_usages"],
                AgentToolUsage("citation_merger", "succeeded", "Merged and deduplicated citations for the phase4 report.", {"citation_count": len(machine_report["citations"])}),
                AgentToolUsage("report_renderer", "succeeded", "Rendered machine-readable and human-readable experiment reports.", {"report_version": machine_report["report_version"]}),
            ],
            warnings=list(execution_result["generation_backend"]["warnings"]),
            error_message="",
        )


class WriterPhase4Validator(BaseValidator):
    def validate_input(self, contract: AgentRuntimeInput) -> list[str]:
        errors = super().validate_input(contract)
        request = Phase4WriterRequest.from_metadata(contract.metadata)
        if not request.run_manifest_id:
            errors.append("run_manifest_id is required")
        if not str(request.run_manifest.get("id", "")).strip() and not request.run_manifest_id:
            errors.append("run_manifest.id is required")
        if not str(request.dataset_profile.get("id", "")).strip():
            errors.append("dataset_profile.id is required")
        if not str(request.selected_idea.get("id", "")).strip():
            errors.append("selected_idea.id is required")
        return errors

    def validate_payload(self, contract: AgentRuntimeInput, payload: dict[str, Any] | None) -> list[str]:
        errors = super().validate_payload(contract, payload)
        if payload is None:
            return errors
        for field_name in WRITER_PHASE4_REQUIRED_FIELDS:
            if field_name not in payload:
                errors.append(f"normalized_payload.{field_name} is required")
        if "machine_readable_report" in payload and not isinstance(payload.get("machine_readable_report"), dict):
            errors.append("normalized_payload.machine_readable_report must be an object")
        if "citation_refs" in payload and not isinstance(payload.get("citation_refs"), list):
            errors.append("normalized_payload.citation_refs must be an array")
        if "reference_source_ids" in payload and not isinstance(payload.get("reference_source_ids"), list):
            errors.append("normalized_payload.reference_source_ids must be an array")
        return errors


class WriterPhase4Repairer(BaseRepairer):
    def repair(
        self,
        contract: AgentRuntimeInput,
        output: AgentRuntimeOutput,
        errors: list[str],
    ) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        repaired, actions = super().repair(contract, output, errors)
        executor = WriterPhase4Executor()
        prepared_request = executor.prepare_request(contract)
        execution_result = executor.execute(prepared_request, contract)
        rebuilt = executor.normalize_output(contract, prepared_request, execution_result, execution_result)
        repaired.normalized_payload = rebuilt.normalized_payload
        repaired.artifact_manifest = rebuilt.artifact_manifest
        repaired.tool_usages = rebuilt.tool_usages
        repaired.warnings = rebuilt.warnings
        actions.append(
            AgentRepairAction(
                action="rebuild_phase4_writer_report",
                status="applied",
                detail="Rebuilt the phase4 structured report from persisted run metadata and citations.",
                metadata={"run_manifest_id": Phase4WriterRequest.from_metadata(contract.metadata).run_manifest_id},
            )
        )
        return repaired, actions


class WriterPhase4Agent(BaseAgent):
    pass


def build_writer_phase4_agent(contract: AgentRuntimeInput) -> WriterPhase4Agent:
    return WriterPhase4Agent(
        executor=WriterPhase4Executor(),
        validator=WriterPhase4Validator(),
        repairer=WriterPhase4Repairer(),
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
                    "writing_generation_backend",
                    "succeeded",
                    "Used deterministic mock generation for phase4 writing.",
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
        metadata=dict(contract.metadata or {}),
    )
    if contract.execution_mode == "codex_cli":
        executor: BaseExecutor = CodexCLIExecutor()
    else:
        executor = ApiExecutor()
    result = executor.run(delegate_contract)
    trace["backend_executor"] = executor.__class__.__name__
    trace["status"] = result.status
    trace["warnings"] = list(result.warnings)
    trace["response_preview"] = str(result.normalized_payload.get("summary", "") or result.error_message or "")[:240]
    if result.status == "succeeded":
        return {
            "execution_mode_used": contract.execution_mode,
            "trace": trace,
            "artifact_manifest": result.artifact_manifest,
            "tool_usages": [
                AgentToolUsage(
                    "writing_generation_backend",
                    "succeeded",
                    "Used runtime generation backend for phase4 writing.",
                    {"backend_executor": executor.__class__.__name__, "execution_mode_used": contract.execution_mode},
                ),
                *result.tool_usages,
            ],
            "warnings": list(result.warnings),
            "response_text": str(result.normalized_payload.get("summary", "")),
        }
    fallback = MockExecutor().run(
        AgentRuntimeInput(
            job_id=contract.job_id,
            agent_type=contract.agent_type,
            execution_mode="mock",
            model_provider=contract.model_provider,
            model_name=contract.model_name,
            prompt_version=contract.prompt_version,
            input_refs=list(contract.input_refs),
            output_schema_ref="schemas/generic-agent-output-v1.json",
            skill_refs=list(contract.skill_refs),
            tool_refs=list(contract.tool_refs),
            memory_refs=list(contract.memory_refs),
            workspace_dir=contract.workspace_dir,
            metadata=dict(contract.metadata or {}),
        )
    )
    warnings = list(result.warnings)
    if result.error_message:
        warnings.append(f"{executor.__class__.__name__} execution failed: {result.error_message}. Falling back to mock executor.")
    trace["execution_mode_used"] = "mock"
    trace["warnings"] = warnings
    return {
        "execution_mode_used": "mock",
        "trace": trace,
        "artifact_manifest": result.artifact_manifest + fallback.artifact_manifest,
        "tool_usages": [
            AgentToolUsage(
                "writing_generation_backend",
                "succeeded",
                "Used runtime generation backend for phase4 writing with fallback.",
                {"backend_executor": executor.__class__.__name__, "execution_mode_requested": contract.execution_mode, "execution_mode_used": "mock"},
            ),
            *result.tool_usages,
            *fallback.tool_usages,
        ],
        "warnings": warnings + list(fallback.warnings),
        "response_text": str(result.normalized_payload.get("summary", "") or result.error_message or ""),
    }
