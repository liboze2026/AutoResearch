from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

try:
    from .base import BaseAgent, BaseRepairer, BaseValidator
    from .contract import AgentRepairAction, AgentRuntimeInput, AgentRuntimeOutput
    from .executors import ApiExecutor, CodexCLIExecutor, MockExecutor
except ImportError:  # pragma: no cover - supports direct script execution
    from base import BaseAgent, BaseRepairer, BaseValidator
    from contract import AgentRepairAction, AgentRuntimeInput, AgentRuntimeOutput
    from executors import ApiExecutor, CodexCLIExecutor, MockExecutor


CODING_SCHEMA_REF = "schemas/coding-output-v1.json"
JSON_BLOCK_PATTERN = re.compile(r"```(?:json)?\s*(.*?)```", re.IGNORECASE | re.DOTALL)


@dataclass
class CodingRequest:
    experiment_id: str = ""
    idea: dict[str, Any] = field(default_factory=dict)
    experiment_plan: dict[str, Any] = field(default_factory=dict)
    experiment_spec: dict[str, Any] = field(default_factory=dict)
    train_template_ref: str = ""
    eval_protocol_ref: str = ""
    eval_protocol: dict[str, Any] = field(default_factory=dict)

    @classmethod
    def from_contract(cls, contract: AgentRuntimeInput) -> "CodingRequest":
        metadata = dict(contract.metadata or {})
        plan = _extract_plan(contract, metadata)
        spec = _extract_spec(contract, metadata)
        eval_protocol_ref, eval_protocol = _extract_eval_protocol(contract, metadata)
        idea = dict(metadata.get("idea", {})) if isinstance(metadata.get("idea"), dict) else {}
        for ref in contract.input_refs:
            if ref.ref_type == "idea" and isinstance(ref.metadata, dict):
                idea = {**ref.metadata, **idea}
        experiment_id = str(metadata.get("experiment_id", "")).strip()
        if not experiment_id:
            experiment_id = str(plan.get("experiment_id", spec.get("experiment_id", ""))).strip()
        return cls(
            experiment_id=experiment_id,
            idea=idea,
            experiment_plan=plan,
            experiment_spec=spec,
            train_template_ref=str(metadata.get("train_template_ref", "")).strip() or str(spec.get("train_template_type", "")).strip() or "mock_train_template",
            eval_protocol_ref=eval_protocol_ref,
            eval_protocol=eval_protocol,
        )

    def to_dict(self) -> dict[str, Any]:
        return {
            "experiment_id": self.experiment_id,
            "train_template_ref": self.train_template_ref,
            "eval_protocol_ref": self.eval_protocol_ref,
            "plan_has_selected_server": bool(self.experiment_plan.get("selected_server") or self.experiment_plan.get("selected_server_name")),
        }


def _extract_json_candidate(text: str) -> Any | None:
    stripped = text.strip()
    if not stripped:
        return None
    match = JSON_BLOCK_PATTERN.search(stripped)
    if match:
        stripped = match.group(1).strip()
    try:
        return json.loads(stripped)
    except json.JSONDecodeError:
        return None


def _load_json_file(path: str) -> dict[str, Any]:
    if not path:
        return {}
    try:
        with open(path, "r", encoding="utf-8") as fp:
            value = json.load(fp)
        if isinstance(value, dict):
            return value
    except OSError:
        return {}
    except json.JSONDecodeError:
        return {}
    return {}


def _extract_plan(contract: AgentRuntimeInput, metadata: dict[str, Any]) -> dict[str, Any]:
    if isinstance(metadata.get("experiment_plan"), dict):
        return dict(metadata["experiment_plan"])
    for ref in contract.input_refs:
        if ref.ref_type == "experiment_plan":
            value = _load_json_file(ref.ref_path)
            if value:
                return value
    return {}


def _extract_spec(contract: AgentRuntimeInput, metadata: dict[str, Any]) -> dict[str, Any]:
    if isinstance(metadata.get("experiment_spec"), dict):
        return dict(metadata["experiment_spec"])
    for ref in contract.input_refs:
        if ref.ref_type == "experiment_spec":
            value = _load_json_file(ref.ref_path)
            if value:
                return value
    return {}


def _extract_eval_protocol(contract: AgentRuntimeInput, metadata: dict[str, Any]) -> tuple[str, dict[str, Any]]:
    ref_value = str(metadata.get("eval_protocol_ref", "")).strip()
    if ref_value:
        return ref_value, _load_json_file(ref_value)
    for ref in contract.input_refs:
        if ref.ref_type == "dataset_eval_protocol":
            path = ref.ref_path or ref.ref_id
            return path, _load_json_file(path)
    return "", {}


def _encode_patch_value(value: Any) -> str:
    if isinstance(value, str):
        return value.strip()
    try:
        return json.dumps(value, ensure_ascii=False, separators=(",", ":"))
    except TypeError:
        return str(value).strip()


def build_coding_payload(request: CodingRequest) -> dict[str, Any]:
    spec = dict(request.experiment_spec or {})
    plan = dict(request.experiment_plan or {})
    expected_metrics = spec.get("expected_metrics", {})
    primary_metric = "accuracy"
    if isinstance(expected_metrics, dict):
        primary_metric = str(expected_metrics.get("primary", "accuracy")).strip() or "accuracy"
    train_template_type = str(spec.get("train_template_type", request.train_template_ref)).strip() or "mock_train_template"
    hyperparams = dict(spec.get("hyperparams", {})) if isinstance(spec.get("hyperparams"), dict) else {}
    hyperparam_patch = {
        "epochs": min(int(hyperparams.get("epochs", 3) or 3), 1),
        "batch_size": min(int(hyperparams.get("batch_size", 8) or 8), 4),
        "gradient_accumulation": min(int(hyperparams.get("gradient_accumulation", 4) or 4), 2),
    }
    spec_overrides = {
        "model_name": str(spec.get("model_name", "mock/llama3.1-8b-instruct")).strip() or "mock/llama3.1-8b-instruct",
        "train_template_type": train_template_type,
        "hyperparams": hyperparam_patch,
        "planner_extensions": {
            "coding_agent": {
                "mode": "template_bound_v1",
                "generated_by": "coding_agent",
                "template_ref": request.train_template_ref or train_template_type,
            }
        },
    }
    patch_manifest = [
        {
            "patch_type": "spec_override",
            "target": "spec.hyperparams",
            "action": "merge",
            "value": _encode_patch_value(hyperparam_patch),
            "summary": "Shrink schedule for first controlled run.",
        },
        {
            "patch_type": "spec_override",
            "target": "spec.planner_extensions.coding_agent",
            "action": "set",
            "value": _encode_patch_value(spec_overrides["planner_extensions"]["coding_agent"]),
            "summary": "Record coding-agent bounded template mode.",
        },
        {
            "patch_type": "config_file",
            "target": "coding/spec_overrides.json",
            "action": "write",
            "value": _encode_patch_value(spec_overrides),
            "summary": "Persist template-safe overrides for audited execution.",
        },
    ]
    if plan:
        patch_manifest.append(
            {
                "patch_type": "notes",
                "target": "coding/template_patch_notes.md",
                "action": "write",
                "value": _encode_patch_value(
                    {
                        "selected_server": plan.get("selected_server", {}),
                        "template_ref": request.train_template_ref or train_template_type,
                    }
                ),
                "summary": "Persist plan-aware template notes.",
            }
        )
    return {
        "code_patch_manifest": patch_manifest,
        "spec_overrides": spec_overrides,
        "execution_result_ref": f"pending://coding/{request.experiment_id or 'unknown'}",
        "metrics_summary": {"primary_metric": primary_metric, "status": "pending"},
        "evaluation_summary_md": "Coding agent generated template-bound overrides. Evaluation will be executed by the stage2 run pipeline.",
    }


def build_coding_prompt(contract: AgentRuntimeInput, request: CodingRequest) -> str:
    return (
        "You are MRAG Coding Agent running in controlled mode.\n"
        "Do not inspect the workspace. Do not run shell commands. Do not browse.\n"
        "You must work strictly inside the existing train template system.\n"
        "Return valid JSON only.\n"
        "The JSON must include: code_patch_manifest, execution_result_ref, metrics_summary, evaluation_summary_md.\n"
        "Always include spec_overrides.\n"
        "In code_patch_manifest.value, use a compact JSON string or a short plain string.\n"
        "Do not propose a brand new project.\n"
        "Prefer the shortest valid answer that satisfies the schema and current task.\n"
        "Do not add markdown fences.\n\n"
        f"job_id: {contract.job_id}\n"
        f"experiment_id: {request.experiment_id}\n"
        f"idea: {json.dumps(request.idea, ensure_ascii=False)}\n"
        f"experiment_plan: {json.dumps(request.experiment_plan, ensure_ascii=False)}\n"
        f"experiment_spec: {json.dumps(request.experiment_spec, ensure_ascii=False)}\n"
        f"train_template_ref: {request.train_template_ref}\n"
        f"eval_protocol_ref: {request.eval_protocol_ref}\n"
        f"eval_protocol: {json.dumps(request.eval_protocol, ensure_ascii=False)}\n"
    )


def build_coding_codex_schema() -> dict[str, Any]:
    string_array = {"type": "array", "items": {"type": "string"}}
    return {
        "type": "object",
        "required": [
            "code_patch_manifest",
            "spec_overrides",
            "execution_result_ref",
            "metrics_summary",
            "evaluation_summary_md",
        ],
        "properties": {
            "code_patch_manifest": {
                "type": "array",
                "items": {
                    "type": "object",
                    "required": ["patch_type", "target", "action", "value", "summary"],
                    "properties": {
                        "patch_type": {"type": "string"},
                        "target": {"type": "string"},
                        "action": {"type": "string"},
                        "value": {"type": "string"},
                        "summary": {"type": "string"},
                    },
                    "additionalProperties": False,
                },
            },
            "spec_overrides": {
                "type": "object",
                "required": ["model_name", "train_template_type", "hyperparams", "planner_extensions"],
                "properties": {
                    "model_name": {"type": "string"},
                    "train_template_type": {"type": "string"},
                    "hyperparams": {
                        "type": "object",
                        "required": ["epochs", "batch_size", "gradient_accumulation"],
                        "properties": {
                            "epochs": {"type": "integer"},
                            "batch_size": {"type": "integer"},
                            "gradient_accumulation": {"type": "integer"},
                        },
                        "additionalProperties": False,
                    },
                    "planner_extensions": {
                        "type": "object",
                        "required": ["coding_agent"],
                        "properties": {
                            "coding_agent": {
                                "type": "object",
                                "required": ["mode", "generated_by", "template_ref"],
                                "properties": {
                                    "mode": {"type": "string"},
                                    "generated_by": {"type": "string"},
                                    "template_ref": {"type": "string"},
                                },
                                "additionalProperties": False,
                            }
                        },
                        "additionalProperties": False,
                    },
                },
                "additionalProperties": False,
            },
            "execution_result_ref": {"type": "string"},
            "metrics_summary": {
                "type": "object",
                "required": ["primary_metric", "status"],
                "properties": {
                    "primary_metric": {"type": "string"},
                    "status": {"type": "string"},
                },
                "additionalProperties": False,
            },
            "evaluation_summary_md": {"type": "string"},
        },
        "additionalProperties": False,
    }


def extract_coding_payload(payload: dict[str, Any], request: CodingRequest) -> dict[str, Any]:
    result = build_coding_payload(request)
    aliases = {
        "code_patch_manifest": ["code_patch_manifest", "patch_manifest", "patches", "codePatchManifest"],
        "execution_result_ref": ["execution_result_ref", "executionResultRef", "run_ref"],
        "metrics_summary": ["metrics_summary", "metricsSummary", "metrics"],
        "evaluation_summary_md": ["evaluation_summary_md", "evaluationSummaryMd", "evaluation_summary", "summary_md"],
        "spec_overrides": ["spec_overrides", "specOverrides"],
    }
    for target, field_aliases in aliases.items():
        for alias in field_aliases:
            if alias in payload and payload[alias] not in (None, ""):
                result[target] = payload[alias]
                break
    nested_candidates: list[dict[str, Any]] = []
    for key in ("data", "result", "response_json", "payload", "json", "object"):
        nested = payload.get(key)
        if isinstance(nested, dict):
            nested_candidates.append(nested)
        elif isinstance(nested, str):
            parsed = _extract_json_candidate(nested)
            if isinstance(parsed, dict):
                nested_candidates.append(parsed)
    response_text = payload.get("response_text", "")
    if isinstance(response_text, str):
        parsed = _extract_json_candidate(response_text)
        if isinstance(parsed, dict):
            nested_candidates.append(parsed)
    for nested in nested_candidates:
        for target, field_aliases in aliases.items():
            for alias in field_aliases:
                if alias in nested and nested[alias] not in (None, ""):
                    result[target] = nested[alias]
                    break
    if not isinstance(result["code_patch_manifest"], list):
        result["code_patch_manifest"] = build_coding_payload(request)["code_patch_manifest"]
    if not isinstance(result["metrics_summary"], dict):
        result["metrics_summary"] = build_coding_payload(request)["metrics_summary"]
    if not isinstance(result["spec_overrides"], dict):
        result["spec_overrides"] = build_coding_payload(request)["spec_overrides"]
    result["execution_result_ref"] = str(result["execution_result_ref"]).strip() or build_coding_payload(request)["execution_result_ref"]
    result["evaluation_summary_md"] = str(result["evaluation_summary_md"]).strip() or build_coding_payload(request)["evaluation_summary_md"]
    return result


class CodingMockExecutor(MockExecutor):
    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = CodingRequest.from_contract(contract)
        coding_payload = build_coding_payload(request)
        normalized_payload = self.base_payload(contract, "mock")
        normalized_payload.update(
            {
                "summary": "Coding mock produced a template-bound patch manifest.",
                "items": [item["summary"] for item in coding_payload["code_patch_manifest"]],
                "data": request.to_dict(),
                "metadata": {"agent_role": "coding", "synthetic": True},
                **coding_payload,
            }
        )
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[],
            warnings=[],
            validation_status="pending",
            repair_status="pending",
            validation_errors=[],
            error_message="",
        )


class CodingApiExecutor(ApiExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        request = CodingRequest.from_contract(contract)
        prepared = super().prepare_request(contract)
        prepared["prompt"] = build_coding_prompt(contract, request)
        prepared["coding_request"] = request.to_dict()
        return prepared

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = CodingRequest.from_contract(contract)
        normalized_payload = self.base_payload(contract, "api")
        payload = extract_coding_payload(collected_response, request)
        normalized_payload.update(
            {
                "summary": "Coding API executor normalized a template-bound response.",
                "items": [item.get("summary", "") for item in payload["code_patch_manifest"] if isinstance(item, dict)],
                "data": request.to_dict(),
                "metadata": {"agent_role": "coding", "executor_response": collected_response},
                **payload,
            }
        )
        return AgentRuntimeOutput(
            status="succeeded" if not execution_result.get("error") else "failed",
            normalized_payload=normalized_payload,
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[],
            warnings=[],
            validation_status="pending",
            repair_status="pending",
            validation_errors=[],
            error_message=str(execution_result.get("error", "")).strip(),
        )


class CodingCodexCLIExecutor(CodexCLIExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        request = CodingRequest.from_contract(contract)
        prepared = super().prepare_request(contract)
        prompt_text = build_coding_prompt(contract, request)
        prepared["prompt_text"] = prompt_text
        prompt_path = Path(prepared["prompt_path"])
        prompt_path.parent.mkdir(parents=True, exist_ok=True)
        prompt_path.write_text(prompt_text, encoding="utf-8")
        schema_path = prompt_path.parent / "output_schema.json"
        schema_path.write_text(json.dumps(build_coding_codex_schema(), ensure_ascii=False, indent=2), encoding="utf-8")
        if "--output-schema" not in prepared["args"]:
            prepared["args"].extend(["--output-schema", str(schema_path)])
        prepared["coding_request"] = request.to_dict()
        return prepared

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = CodingRequest.from_contract(contract)
        output = super().normalize_output(contract, prepared_request, execution_result, collected_response)
        payload = extract_coding_payload(output.normalized_payload, request)
        output.normalized_payload.update(
            {
                "summary": "Coding Codex CLI executor normalized a template-bound response.",
                "items": [item.get("summary", "") for item in payload["code_patch_manifest"] if isinstance(item, dict)],
                "data": request.to_dict(),
                "metadata": {
                    "agent_role": "coding",
                    "stdout": execution_result.get("stdout", ""),
                    "stderr": execution_result.get("stderr", ""),
                },
                **payload,
            }
        )
        return output


class CodingValidator(BaseValidator):
    pass


class CodingRepairer(BaseRepairer):
    def repair(
        self,
        contract: AgentRuntimeInput,
        output: AgentRuntimeOutput,
        errors: list[str],
    ) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        repaired, actions = super().repair(contract, output, errors)
        request = CodingRequest.from_contract(contract)
        payload = extract_coding_payload(repaired.normalized_payload, request)
        repaired.normalized_payload.update(payload)
        return repaired, actions


class CodingAgent(BaseAgent):
    pass


def build_coding_agent(contract: AgentRuntimeInput) -> CodingAgent:
    if contract.execution_mode == "api":
        executor = CodingApiExecutor()
    elif contract.execution_mode == "codex_cli":
        executor = CodingCodexCLIExecutor()
    else:
        executor = CodingMockExecutor()
    return CodingAgent(executor=executor, validator=CodingValidator(), repairer=CodingRepairer())
