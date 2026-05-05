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


PLANNER_SCHEMA_REF = "schemas/planner-output-v1.json"
JSON_BLOCK_PATTERN = re.compile(r"```(?:json)?\s*(.*?)```", re.IGNORECASE | re.DOTALL)


@dataclass
class PlannerRequest:
    idea_id: str = ""
    dataset_asset_refs: list[str] = field(default_factory=list)
    eval_protocol_refs: list[str] = field(default_factory=list)
    server_resource_snapshots: list[dict[str, Any]] = field(default_factory=list)
    baseline_refs: list[str] = field(default_factory=list)
    idea: dict[str, Any] = field(default_factory=dict)
    datasets: list[dict[str, Any]] = field(default_factory=list)
    eval_plans: list[dict[str, Any]] = field(default_factory=list)
    baselines: list[dict[str, Any]] = field(default_factory=list)
    human_hints: list[str] = field(default_factory=list)

    @classmethod
    def from_contract(cls, contract: AgentRuntimeInput) -> "PlannerRequest":
        metadata = dict(contract.metadata or {})
        input_idea_id = ""
        input_datasets: list[dict[str, Any]] = []
        input_dataset_refs: list[str] = []
        input_eval_refs: list[str] = []
        input_baselines: list[dict[str, Any]] = []
        input_idea: dict[str, Any] = {}
        for ref in contract.input_refs:
            if ref.ref_type == "idea":
                if ref.ref_id and not input_idea_id:
                    input_idea_id = ref.ref_id
                if isinstance(ref.metadata, dict):
                    input_idea.update(ref.metadata)
            elif ref.ref_type == "dataset_asset":
                if ref.ref_id:
                    input_dataset_refs.append(ref.ref_id)
                item = {"dataset_asset_id": ref.ref_id}
                if isinstance(ref.metadata, dict):
                    item.update(ref.metadata)
                input_datasets.append(item)
            elif ref.ref_type == "dataset_eval_protocol":
                if ref.ref_path:
                    input_eval_refs.append(ref.ref_path)
                elif ref.ref_id:
                    input_eval_refs.append(ref.ref_id)
            elif ref.ref_type == "baseline":
                item = {"baseline_id": ref.ref_id}
                if isinstance(ref.metadata, dict):
                    item.update(ref.metadata)
                input_baselines.append(item)
        return cls(
            idea_id=str(metadata.get("idea_id", "")).strip() or input_idea_id,
            dataset_asset_refs=_normalize_string_list(metadata.get("dataset_asset_refs", [])) or _normalize_string_list(input_dataset_refs),
            eval_protocol_refs=_normalize_string_list(metadata.get("eval_protocol_refs", [])) or _normalize_string_list(input_eval_refs),
            server_resource_snapshots=_extract_server_snapshots(contract.input_refs, metadata),
            baseline_refs=_normalize_string_list(metadata.get("baseline_refs", [])),
            idea=(dict(metadata.get("idea", {})) if isinstance(metadata.get("idea"), dict) else {}) or input_idea,
            datasets=[item for item in metadata.get("dataset_assets", []) if isinstance(item, dict)] or input_datasets,
            eval_plans=[item for item in metadata.get("eval_plans", []) if isinstance(item, dict)],
            baselines=[item for item in metadata.get("baselines", []) if isinstance(item, dict)] or input_baselines,
            human_hints=_normalize_string_list(metadata.get("human_hints", [])),
        )

    def to_dict(self) -> dict[str, Any]:
        return {
            "idea_id": self.idea_id,
            "dataset_asset_refs": list(self.dataset_asset_refs),
            "eval_protocol_refs": list(self.eval_protocol_refs),
            "server_resource_snapshot_count": len(self.server_resource_snapshots),
            "baseline_refs": list(self.baseline_refs),
            "human_hints": list(self.human_hints),
        }


def _normalize_string_list(value: Any) -> list[str]:
    if value is None:
        return []
    if isinstance(value, list):
        return [str(item).strip() for item in value if str(item).strip()]
    text = str(value).strip()
    if not text:
        return []
    if "," in text:
        return [item.strip() for item in text.split(",") if item.strip()]
    return [text]


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


def _extract_server_snapshots(input_refs: list[Any], metadata: dict[str, Any]) -> list[dict[str, Any]]:
    out: list[dict[str, Any]] = []
    metadata_snapshots = metadata.get("server_resource_snapshots", [])
    if isinstance(metadata_snapshots, list):
        for item in metadata_snapshots:
            if isinstance(item, dict):
                out.append(dict(item))
    for ref in input_refs:
        if getattr(ref, "ref_type", "") != "server_resource_snapshot":
            continue
        if isinstance(getattr(ref, "metadata", None), dict):
            merged = dict(ref.metadata)
            if getattr(ref, "ref_id", "") and "server_id" not in merged:
                merged["server_id"] = ref.ref_id
            out.append(merged)
    return out


def _pick_primary_dataset(request: PlannerRequest) -> dict[str, Any]:
    if request.datasets:
        return dict(request.datasets[0])
    if request.dataset_asset_refs:
        return {"dataset_asset_id": request.dataset_asset_refs[0], "task_type": "text", "name": request.dataset_asset_refs[0]}
    return {"dataset_asset_id": "", "task_type": "text", "name": "unknown dataset"}


def _pick_primary_eval_plan(request: PlannerRequest) -> dict[str, Any]:
    if request.eval_plans:
        return dict(request.eval_plans[0])
    if request.eval_protocol_refs:
        return {"eval_protocol_ref": request.eval_protocol_refs[0], "metric_list": ["accuracy"], "task_type": "text"}
    return {"metric_list": ["accuracy"], "task_type": "text"}


def _pick_primary_baseline(request: PlannerRequest) -> dict[str, Any]:
    if request.baselines:
        return dict(request.baselines[0])
    if request.baseline_refs:
        return {"baseline_id": request.baseline_refs[0], "name": request.baseline_refs[0]}
    return {}


def _choose_server(request: PlannerRequest) -> dict[str, Any]:
    candidates = [item for item in request.server_resource_snapshots if isinstance(item, dict)]
    if not candidates:
        return {
            "server_id": "",
            "server_name": "mock_server",
            "status": "unknown",
            "best_free_mem_mb": 0,
            "best_utilization": 100,
            "selection_reason": "no_server_snapshot_available",
        }
    candidates.sort(
        key=lambda item: (
            str(item.get("status", "")) == "online",
            int(item.get("best_free_mem_mb", 0) or 0),
            -int(item.get("best_utilization", 100) or 100),
        ),
        reverse=True,
    )
    chosen = dict(candidates[0])
    chosen["selection_reason"] = "highest_available_gpu_capacity"
    return chosen


def _select_train_template(task_type: str, idea: dict[str, Any]) -> str:
    title = str(idea.get("title", "")).lower()
    if "lora" in title:
        return "lora_sft_v1"
    task_type = str(task_type).strip().lower()
    if task_type in {"text", "qa", "rag"}:
        return "text_finetune_v1"
    return "generic_train_v1"


def build_planner_payload(request: PlannerRequest) -> dict[str, Any]:
    idea = request.idea or {}
    dataset = _pick_primary_dataset(request)
    eval_plan = _pick_primary_eval_plan(request)
    baseline = _pick_primary_baseline(request)
    selected_server = _choose_server(request)
    task_type = str(dataset.get("task_type", eval_plan.get("task_type", "text"))).strip().lower() or "text"
    train_template_type = _select_train_template(task_type, idea)
    metrics = _normalize_string_list(eval_plan.get("metric_list", ["accuracy"]))
    if not metrics and isinstance(eval_plan.get("eval_protocol_json"), dict):
        metrics = _normalize_string_list(eval_plan["eval_protocol_json"].get("metric_list", []))
    if not metrics:
        metrics = ["accuracy"]
    dataset_asset_id = str(dataset.get("dataset_asset_id", dataset.get("id", ""))).strip()
    baseline_id = str(baseline.get("baseline_id", baseline.get("id", ""))).strip()
    eval_ref = str(eval_plan.get("eval_protocol_ref", eval_plan.get("evalplan_path", ""))).strip()
    resource_estimate = {
        "gpu_count": 1,
        "min_free_mem_mb": max(int(selected_server.get("best_free_mem_mb", 0) or 0), 8192 if task_type in {"text", "qa", "rag"} else 12288),
        "estimated_hours": 4 if task_type in {"text", "qa", "rag"} else 6,
        "preferred_server_id": str(selected_server.get("server_id", "")).strip(),
        "preferred_server_name": str(selected_server.get("server_name", "mock_server")).strip() or "mock_server",
    }
    run_sequence = [
        "validate_inputs",
        "generate_experiment_spec",
        "queue_experiment",
        "schedule_run",
        "collect_logs",
        "compare_results",
    ]
    success_criteria = {
        "required_metrics": metrics,
        "minimum_relative_gain_vs_baseline": 0.02 if baseline_id else 0.0,
        "must_generate_result_archive": True,
        "must_record_comparison": True,
    }
    fallback_plan = {
        "fallback_server_name": "mock_server" if resource_estimate["preferred_server_name"] == "shenzhenvlab" else resource_estimate["preferred_server_name"],
        "fallback_template_type": "generic_train_v1",
        "fallback_actions": [
            "reduce_batch_size",
            "switch_to_shorter_schedule",
            "reuse_existing_baseline",
        ],
    }
    experiment_plan_json = {
        "idea_id": request.idea_id,
        "dataset_asset_id": dataset_asset_id,
        "baseline_id": baseline_id,
        "eval_protocol_ref": eval_ref,
        "selected_server": selected_server,
        "spec_seed": {
            "train_template_type": train_template_type,
            "expected_metrics": metrics,
            "output_subdir": "outputs",
        },
        "planner_notes": {
            "innovation_type": str(idea.get("innovation_type", "")).strip(),
            "expected_advantage": str(idea.get("expected_advantage", "")).strip(),
            "risk_points": list(idea.get("risk_points", [])) if isinstance(idea.get("risk_points"), list) else [],
            "human_hints": list(request.human_hints),
        },
    }
    return {
        "experiment_plan_json": experiment_plan_json,
        "train_template_type": train_template_type,
        "resource_estimate": resource_estimate,
        "run_sequence": run_sequence,
        "success_criteria": success_criteria,
        "fallback_plan": fallback_plan,
    }


def build_planner_prompt(contract: AgentRuntimeInput, request: PlannerRequest) -> str:
    return (
        "You are MRAG Planner Agent running in controlled mode.\n"
        "Do not inspect the workspace. Do not run shell commands. Do not browse.\n"
        "Return valid JSON only.\n"
        "The JSON must include: experiment_plan_json, train_template_type, resource_estimate, run_sequence, success_criteria, fallback_plan.\n"
        "Keep the plan minimal and executable inside the existing stage2 experiment lifecycle.\n"
        "Prefer the shortest valid answer that satisfies the schema and current task.\n"
        "Do not add markdown fences.\n\n"
        f"job_id: {contract.job_id}\n"
        f"idea_id: {request.idea_id}\n"
        f"idea: {json.dumps(request.idea, ensure_ascii=False)}\n"
        f"dataset_asset_refs: {json.dumps(request.dataset_asset_refs, ensure_ascii=False)}\n"
        f"eval_protocol_refs: {json.dumps(request.eval_protocol_refs, ensure_ascii=False)}\n"
        f"baseline_refs: {json.dumps(request.baseline_refs, ensure_ascii=False)}\n"
        f"server_resource_snapshots: {json.dumps(request.server_resource_snapshots[:5], ensure_ascii=False)}\n"
        f"human_hints: {json.dumps(request.human_hints, ensure_ascii=False)}\n"
    )


def build_planner_codex_schema() -> dict[str, Any]:
    string_array = {"type": "array", "items": {"type": "string"}}
    return {
        "type": "object",
        "required": [
            "experiment_plan_json",
            "train_template_type",
            "resource_estimate",
            "run_sequence",
            "success_criteria",
            "fallback_plan",
        ],
        "properties": {
            "experiment_plan_json": {
                "type": "object",
                "required": ["idea_id", "dataset_asset_id", "baseline_id", "eval_protocol_ref", "selected_server", "spec_seed", "planner_notes"],
                "properties": {
                    "idea_id": {"type": "string"},
                    "dataset_asset_id": {"type": "string"},
                    "baseline_id": {"type": "string"},
                    "eval_protocol_ref": {"type": "string"},
                    "selected_server": {
                        "type": "object",
                        "required": ["server_id", "server_name", "status", "best_free_mem_mb", "best_utilization", "selection_reason"],
                        "properties": {
                            "server_id": {"type": "string"},
                            "server_name": {"type": "string"},
                            "status": {"type": "string"},
                            "best_free_mem_mb": {"type": "integer"},
                            "best_utilization": {"type": "integer"},
                            "selection_reason": {"type": "string"},
                        },
                        "additionalProperties": False,
                    },
                    "spec_seed": {
                        "type": "object",
                        "required": ["train_template_type", "expected_metrics", "output_subdir"],
                        "properties": {
                            "train_template_type": {"type": "string"},
                            "expected_metrics": string_array,
                            "output_subdir": {"type": "string"},
                        },
                        "additionalProperties": False,
                    },
                    "planner_notes": {
                        "type": "object",
                        "required": ["innovation_type", "expected_advantage", "risk_points", "human_hints"],
                        "properties": {
                            "innovation_type": {"type": "string"},
                            "expected_advantage": {"type": "string"},
                            "risk_points": string_array,
                            "human_hints": string_array,
                        },
                        "additionalProperties": False,
                    },
                },
                "additionalProperties": False,
            },
            "train_template_type": {"type": "string"},
            "resource_estimate": {
                "type": "object",
                "required": ["gpu_count", "min_free_mem_mb", "estimated_hours", "preferred_server_id", "preferred_server_name"],
                "properties": {
                    "gpu_count": {"type": "integer"},
                    "min_free_mem_mb": {"type": "integer"},
                    "estimated_hours": {"type": "integer"},
                    "preferred_server_id": {"type": "string"},
                    "preferred_server_name": {"type": "string"},
                },
                "additionalProperties": False,
            },
            "run_sequence": string_array,
            "success_criteria": {
                "type": "object",
                "required": ["required_metrics", "minimum_relative_gain_vs_baseline", "must_generate_result_archive", "must_record_comparison"],
                "properties": {
                    "required_metrics": string_array,
                    "minimum_relative_gain_vs_baseline": {"type": "number"},
                    "must_generate_result_archive": {"type": "boolean"},
                    "must_record_comparison": {"type": "boolean"},
                },
                "additionalProperties": False,
            },
            "fallback_plan": {
                "type": "object",
                "required": ["fallback_server_name", "fallback_template_type", "fallback_actions"],
                "properties": {
                    "fallback_server_name": {"type": "string"},
                    "fallback_template_type": {"type": "string"},
                    "fallback_actions": string_array,
                },
                "additionalProperties": False,
            },
        },
        "additionalProperties": False,
    }


def extract_planner_payload(payload: dict[str, Any], request: PlannerRequest) -> dict[str, Any]:
    result = build_planner_payload(request)
    aliases = {
        "experiment_plan_json": ["experiment_plan_json", "experimentPlanJson", "plan", "plan_json"],
        "train_template_type": ["train_template_type", "trainTemplateType", "template_type"],
        "resource_estimate": ["resource_estimate", "resourceEstimate"],
        "run_sequence": ["run_sequence", "runSequence", "steps"],
        "success_criteria": ["success_criteria", "successCriteria"],
        "fallback_plan": ["fallback_plan", "fallbackPlan"],
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
    if not isinstance(result["experiment_plan_json"], dict):
        result["experiment_plan_json"] = build_planner_payload(request)["experiment_plan_json"]
    if not isinstance(result["resource_estimate"], dict):
        result["resource_estimate"] = build_planner_payload(request)["resource_estimate"]
    if not isinstance(result["success_criteria"], dict):
        result["success_criteria"] = build_planner_payload(request)["success_criteria"]
    if not isinstance(result["fallback_plan"], dict):
        result["fallback_plan"] = build_planner_payload(request)["fallback_plan"]
    result["run_sequence"] = _normalize_string_list(result["run_sequence"])
    if not result["run_sequence"]:
        result["run_sequence"] = build_planner_payload(request)["run_sequence"]
    result["train_template_type"] = str(result["train_template_type"]).strip() or build_planner_payload(request)["train_template_type"]
    return result


class PlannerMockExecutor(MockExecutor):
    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = PlannerRequest.from_contract(contract)
        planner_payload = build_planner_payload(request)
        normalized_payload = self.base_payload(contract, "mock")
        normalized_payload.update(
            {
                "summary": "Planner mock produced a controlled experiment plan.",
                "items": list(planner_payload["run_sequence"]),
                "data": request.to_dict(),
                "metadata": {"agent_role": "planner", "synthetic": True},
                **planner_payload,
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


class PlannerApiExecutor(ApiExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        request = PlannerRequest.from_contract(contract)
        prepared = super().prepare_request(contract)
        prepared["prompt"] = build_planner_prompt(contract, request)
        prepared["planner_request"] = request.to_dict()
        return prepared

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = PlannerRequest.from_contract(contract)
        normalized_payload = self.base_payload(contract, "api")
        payload = extract_planner_payload(collected_response, request)
        normalized_payload.update(
            {
                "summary": "Planner API executor normalized a planner response.",
                "items": list(payload["run_sequence"]),
                "data": request.to_dict(),
                "metadata": {"agent_role": "planner", "executor_response": collected_response},
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


class PlannerCodexCLIExecutor(CodexCLIExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        request = PlannerRequest.from_contract(contract)
        prepared = super().prepare_request(contract)
        prompt_text = build_planner_prompt(contract, request)
        prepared["prompt_text"] = prompt_text
        prompt_path = Path(prepared["prompt_path"])
        prompt_path.parent.mkdir(parents=True, exist_ok=True)
        prompt_path.write_text(prompt_text, encoding="utf-8")
        schema_path = prompt_path.parent / "output_schema.json"
        schema_path.write_text(json.dumps(build_planner_codex_schema(), ensure_ascii=False, indent=2), encoding="utf-8")
        if "--output-schema" not in prepared["args"]:
            prepared["args"].extend(["--output-schema", str(schema_path)])
        prepared["planner_request"] = request.to_dict()
        return prepared

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = PlannerRequest.from_contract(contract)
        output = super().normalize_output(contract, prepared_request, execution_result, collected_response)
        payload = extract_planner_payload(output.normalized_payload, request)
        output.normalized_payload.update(
            {
                "summary": "Planner Codex CLI executor normalized a planner response.",
                "items": list(payload["run_sequence"]),
                "data": request.to_dict(),
                "metadata": {
                    "agent_role": "planner",
                    "stdout": execution_result.get("stdout", ""),
                    "stderr": execution_result.get("stderr", ""),
                },
                **payload,
            }
        )
        return output


class PlannerValidator(BaseValidator):
    pass


class PlannerRepairer(BaseRepairer):
    def repair(
        self,
        contract: AgentRuntimeInput,
        output: AgentRuntimeOutput,
        errors: list[str],
    ) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        repaired, actions = super().repair(contract, output, errors)
        request = PlannerRequest.from_contract(contract)
        payload = extract_planner_payload(repaired.normalized_payload, request)
        repaired.normalized_payload.update(payload)
        return repaired, actions


class PlannerAgent(BaseAgent):
    pass


def build_planner_agent(contract: AgentRuntimeInput) -> PlannerAgent:
    if contract.execution_mode == "api":
        executor = PlannerApiExecutor()
    elif contract.execution_mode == "codex_cli":
        executor = PlannerCodexCLIExecutor()
    else:
        executor = PlannerMockExecutor()
    return PlannerAgent(executor=executor, validator=PlannerValidator(), repairer=PlannerRepairer())
