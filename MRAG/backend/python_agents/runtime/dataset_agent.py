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


DATASET_SCHEMA_REF = "schemas/dataset-output-v1.json"
JSON_BLOCK_PATTERN = re.compile(r"```(?:json)?\s*(.*?)```", re.IGNORECASE | re.DOTALL)


@dataclass
class DatasetRequest:
    research_direction: str = ""
    task_type: str = ""
    keywords: list[str] = field(default_factory=list)
    target_server_preference: str = ""
    dataset_constraints: dict[str, Any] = field(default_factory=dict)
    discovered_datasets: list[dict[str, Any]] = field(default_factory=list)
    server_context: dict[str, Any] = field(default_factory=dict)

    @classmethod
    def from_contract(cls, contract: AgentRuntimeInput) -> "DatasetRequest":
        metadata = dict(contract.metadata or {})
        keywords_raw = metadata.get("keywords", [])
        keywords: list[str]
        if isinstance(keywords_raw, list):
            keywords = [str(item).strip() for item in keywords_raw if str(item).strip()]
        else:
            keywords = [item.strip() for item in str(keywords_raw).split(",") if item.strip()]
        constraints = metadata.get("dataset_constraints", {})
        if not isinstance(constraints, dict):
            constraints = {"value": constraints}
        discovered = metadata.get("discovered_datasets", [])
        if not isinstance(discovered, list):
            discovered = []
        server_context = metadata.get("server_context", {})
        if not isinstance(server_context, dict):
            server_context = {}
        return cls(
            research_direction=str(metadata.get("research_direction", "")).strip(),
            task_type=str(metadata.get("task_type", "")).strip().lower(),
            keywords=keywords,
            target_server_preference=str(metadata.get("target_server_preference", "")).strip(),
            dataset_constraints=constraints,
            discovered_datasets=[item for item in discovered if isinstance(item, dict)],
            server_context=server_context,
        )

    def to_dict(self) -> dict[str, Any]:
        return {
            "research_direction": self.research_direction,
            "task_type": self.task_type,
            "keywords": list(self.keywords),
            "target_server_preference": self.target_server_preference,
            "dataset_constraints": dict(self.dataset_constraints),
            "discovered_dataset_count": len(self.discovered_datasets),
            "server_context": dict(self.server_context),
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


def _normalize_string_list(value: Any) -> list[str]:
    if value is None:
        return []
    if isinstance(value, list):
        out = []
        for item in value:
            text = str(item).strip()
            if text:
                out.append(text)
        return out
    if isinstance(value, str):
        stripped = value.strip()
        if not stripped:
            return []
        parsed = _extract_json_candidate(stripped)
        if isinstance(parsed, list):
            return _normalize_string_list(parsed)
        if "\n" in stripped:
            return [line.lstrip("-* ").strip() for line in stripped.splitlines() if line.strip()]
        return [stripped]
    return [str(value).strip()] if str(value).strip() else []


def _default_metrics(task_type: str) -> list[str]:
    mapping = {
        "retrieval": ["recall@10", "mrr", "ndcg@10"],
        "classification": ["accuracy", "macro_f1", "precision", "recall"],
        "generation": ["bleu", "rouge_l", "exact_match"],
        "multimodal": ["accuracy", "macro_f1", "recall@10"],
        "text": ["accuracy", "macro_f1"],
    }
    return mapping.get(task_type or "text", ["accuracy", "macro_f1"])


def _default_split_strategy(task_type: str) -> str:
    if task_type in {"retrieval", "ranking"}:
        return "query_document_train_dev_test"
    if task_type in {"generation"}:
        return "train_validation_test"
    return "stratified_train_validation_test"


def _choose_existing_dataset(request: DatasetRequest) -> dict[str, Any] | None:
    required_keywords = [item.lower() for item in request.keywords]
    preferred_server = request.target_server_preference.lower()
    best_item: dict[str, Any] | None = None
    best_score = -1
    for item in request.discovered_datasets:
        name = str(item.get("name", "")).lower()
        desc = str(item.get("description", "")).lower()
        modality = str(item.get("modality", item.get("detected_modality", ""))).lower()
        server_name = str(item.get("server_name", "")).lower()
        score = 0
        for keyword in required_keywords:
            if keyword and (keyword in name or keyword in desc):
                score += 2
        if request.task_type and request.task_type in modality:
            score += 1
        if preferred_server and preferred_server == server_name:
            score += 1
        if score > best_score:
            best_item = item
            best_score = score
    if best_score <= 0 and request.dataset_constraints.get("force_download"):
        return None
    return best_item if best_item is not None else (request.discovered_datasets[0] if request.discovered_datasets else None)


def _server_decision(request: DatasetRequest) -> dict[str, Any]:
    preferred = str(request.server_context.get("selected_server_name", request.target_server_preference)).strip()
    mode = str(request.server_context.get("decision_mode", "mock")).strip() or "mock"
    gpu_available = bool(request.server_context.get("gpu_available"))
    fallback_reason = str(request.server_context.get("fallback_reason", "")).strip()
    return {
        "selected_server_name": preferred or "mock_server",
        "decision_mode": mode,
        "gpu_available": gpu_available,
        "fallback_reason": fallback_reason,
    }


def _planned_dataset_location(contract: AgentRuntimeInput) -> str:
    workspace_dir = str(contract.workspace_dir or "").strip()
    if not workspace_dir:
        return ""
    workspace_path = Path(workspace_dir)
    if (
        workspace_path.name == contract.job_id
        and workspace_path.parent.name == "jobs"
        and workspace_path.parent.parent.name == "agents"
    ):
        workspace_root = workspace_path.parent.parent.parent
        return str(workspace_root / "datasets" / "downloads" / contract.job_id / "mock_dataset")
    return str(workspace_path / "mock_dataset")


def _ensure_dataset_location(
    contract: AgentRuntimeInput,
    payload: dict[str, Any],
    request: DatasetRequest,
) -> dict[str, Any]:
    location = str(payload.get("dataset_location", "")).strip()
    if location:
        payload["dataset_location"] = location
        return payload
    fetch_action = str(payload.get("fetch_action", "")).strip() or build_dataset_payload(request)["fetch_action"]
    if fetch_action != "register_existing":
        payload["dataset_location"] = _planned_dataset_location(contract)
    return payload


def build_dataset_payload(request: DatasetRequest) -> dict[str, Any]:
    existing = _choose_existing_dataset(request)
    server_info = _server_decision(request)
    task_type = request.task_type or "text"
    metrics = _default_metrics(task_type)
    split_strategy = _default_split_strategy(task_type)
    baseline_needed = True
    if existing is not None:
        fetch_action = "register_existing"
        dataset_location = str(existing.get("path", "")).strip()
        selected_dataset_ref = str(existing.get("id", "")).strip()
        notes_md = (
            f"Prefer registering existing dataset `{existing.get('name', 'dataset')}` from MRAG scan results. "
            f"Selected server context: {server_info['selected_server_name']}."
        )
    else:
        fetch_action = "mock_download"
        dataset_location = ""
        selected_dataset_ref = ""
        notes_md = (
            f"No sufficiently matched existing dataset was selected, so the agent plans a controlled mock download. "
            f"Server context: {server_info['selected_server_name']}."
        )
    eval_protocol = {
        "task_type": task_type,
        "metric_list": metrics,
        "evaluation_steps": [
            "Validate dataset availability and schema consistency.",
            "Prepare train/validation/test split according to split strategy.",
            "Run at least one baseline and capture metrics in structured form.",
            "Generate a reproducible evaluation report using the report template.",
        ],
        "data_split": {
            "strategy": split_strategy,
            "train_ratio": 0.8,
            "validation_ratio": 0.1,
            "test_ratio": 0.1,
        },
        "baseline_needed": baseline_needed,
        "report_template": {
            "sections": ["Dataset Summary", "Protocol", "Metrics", "Error Analysis", "Limitations"],
            "format": "markdown",
        },
    }
    metric_schema = {
        "primary_metric": metrics[0],
        "metric_list": metrics,
        "direction": "maximize",
    }
    return {
        "dataset_asset_ref": "",
        "dataset_location": dataset_location,
        "fetch_action": fetch_action,
        "selected_dataset_ref": selected_dataset_ref,
        "server_decision": server_info,
        "eval_protocol_json": eval_protocol,
        "metric_schema_json": metric_schema,
        "split_strategy": split_strategy,
        "notes_md": notes_md,
    }


def build_dataset_prompt(contract: AgentRuntimeInput, request: DatasetRequest) -> str:
    return (
        "You are MRAG Dataset Agent running in controlled mode.\n"
        "Do not inspect the workspace. Do not run shell commands. Do not browse.\n"
        "Return valid JSON only.\n"
        "The JSON must include: dataset_location, eval_protocol_json, metric_schema_json, split_strategy, notes_md.\n"
        "Always include dataset_asset_ref, fetch_action, selected_dataset_ref, and server_decision.\n"
        "Prefer registering an existing dataset when a strong match is already present.\n"
        "Prefer the shortest valid answer that satisfies the schema and current task.\n"
        "Do not add markdown fences.\n\n"
        f"job_id: {contract.job_id}\n"
        f"research_direction: {request.research_direction}\n"
        f"task_type: {request.task_type}\n"
        f"keywords: {json.dumps(request.keywords, ensure_ascii=False)}\n"
        f"target_server_preference: {request.target_server_preference}\n"
        f"dataset_constraints: {json.dumps(request.dataset_constraints, ensure_ascii=False)}\n"
        f"discovered_datasets: {json.dumps(request.discovered_datasets[:5], ensure_ascii=False)}\n"
        f"server_context: {json.dumps(request.server_context, ensure_ascii=False)}\n"
    )


def build_dataset_codex_schema() -> dict[str, Any]:
    string_array = {"type": "array", "items": {"type": "string"}}
    return {
        "type": "object",
        "required": [
            "dataset_asset_ref",
            "dataset_location",
            "fetch_action",
            "selected_dataset_ref",
            "server_decision",
            "eval_protocol_json",
            "metric_schema_json",
            "split_strategy",
            "notes_md",
        ],
        "properties": {
            "dataset_asset_ref": {"type": "string"},
            "dataset_location": {"type": "string"},
            "fetch_action": {"type": "string"},
            "selected_dataset_ref": {"type": "string"},
            "server_decision": {
                "type": "object",
                "required": ["selected_server_name", "decision_mode", "gpu_available", "fallback_reason"],
                "properties": {
                    "selected_server_name": {"type": "string"},
                    "decision_mode": {"type": "string"},
                    "gpu_available": {"type": "boolean"},
                    "fallback_reason": {"type": "string"},
                },
                "additionalProperties": False,
            },
            "eval_protocol_json": {
                "type": "object",
                "required": ["task_type", "metric_list", "evaluation_steps", "data_split", "baseline_needed", "report_template"],
                "properties": {
                    "task_type": {"type": "string"},
                    "metric_list": string_array,
                    "evaluation_steps": string_array,
                    "data_split": {
                        "type": "object",
                        "required": ["strategy", "train_ratio", "validation_ratio", "test_ratio"],
                        "properties": {
                            "strategy": {"type": "string"},
                            "train_ratio": {"type": "number"},
                            "validation_ratio": {"type": "number"},
                            "test_ratio": {"type": "number"},
                        },
                        "additionalProperties": False,
                    },
                    "baseline_needed": {"type": "boolean"},
                    "report_template": {
                        "type": "object",
                        "required": ["sections", "format"],
                        "properties": {
                            "sections": string_array,
                            "format": {"type": "string"},
                        },
                        "additionalProperties": False,
                    },
                },
                "additionalProperties": False,
            },
            "metric_schema_json": {
                "type": "object",
                "required": ["primary_metric", "metric_list", "direction"],
                "properties": {
                    "primary_metric": {"type": "string"},
                    "metric_list": string_array,
                    "direction": {"type": "string"},
                },
                "additionalProperties": False,
            },
            "split_strategy": {"type": "string"},
            "notes_md": {"type": "string"},
        },
        "additionalProperties": False,
    }


def extract_dataset_payload(payload: dict[str, Any], request: DatasetRequest) -> dict[str, Any]:
    result = build_dataset_payload(request)
    aliases = {
        "dataset_asset_ref": ["dataset_asset_ref", "datasetAssetRef"],
        "dataset_location": ["dataset_location", "datasetLocation", "path"],
        "fetch_action": ["fetch_action", "dataset_fetch_or_register", "fetchAction"],
        "selected_dataset_ref": ["selected_dataset_ref", "existing_dataset_ref", "dataset_ref"],
        "eval_protocol_json": ["eval_protocol_json", "eval_protocol", "evaluation_protocol", "protocol"],
        "metric_schema_json": ["metric_schema_json", "metric_schema", "metrics"],
        "split_strategy": ["split_strategy", "data_split_strategy"],
        "notes_md": ["notes_md", "notes", "note_md", "message"],
        "server_decision": ["server_decision"],
    }
    for target, field_aliases in aliases.items():
        for alias in field_aliases:
            if alias in payload and payload[alias] not in (None, ""):
                result[target] = payload[alias]
                break
    nested_candidates = []
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
    if not isinstance(result["eval_protocol_json"], dict):
        result["eval_protocol_json"] = build_dataset_payload(request)["eval_protocol_json"]
    if not isinstance(result["metric_schema_json"], dict):
        result["metric_schema_json"] = build_dataset_payload(request)["metric_schema_json"]
    result["split_strategy"] = str(result["split_strategy"]).strip() or _default_split_strategy(request.task_type or "text")
    result["notes_md"] = str(result["notes_md"]).strip() or build_dataset_payload(request)["notes_md"]
    if not isinstance(result["server_decision"], dict):
        result["server_decision"] = _server_decision(request)
    result["dataset_location"] = str(result["dataset_location"]).strip()
    result["dataset_asset_ref"] = str(result["dataset_asset_ref"]).strip()
    result["fetch_action"] = str(result["fetch_action"]).strip() or "register_existing"
    result["selected_dataset_ref"] = str(result["selected_dataset_ref"]).strip()
    return result


class DatasetMockExecutor(MockExecutor):
    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = DatasetRequest.from_contract(contract)
        dataset_payload = build_dataset_payload(request)
        normalized_payload = self.base_payload(contract, "mock")
        normalized_payload.update(
            {
                "summary": "Dataset mock produced controlled fetch/register decision and evaluation protocol.",
                "items": list(dataset_payload["eval_protocol_json"].get("metric_list", [])),
                "data": request.to_dict(),
                "metadata": {"agent_role": "dataset", "synthetic": True},
                **dataset_payload,
            }
        )
        normalized_payload = _ensure_dataset_location(contract, normalized_payload, request)
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[],
            warnings=["dataset mock executor used; results are deterministic placeholders."],
            error_message="",
        )


class DatasetApiExecutor(ApiExecutor):
    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        output = super().normalize_output(contract, prepared_request, execution_result, collected_response)
        request = DatasetRequest.from_contract(contract)
        dataset_payload = extract_dataset_payload(output.normalized_payload, request)
        output.normalized_payload.update(dataset_payload)
        output.normalized_payload = _ensure_dataset_location(contract, output.normalized_payload, request)
        output.normalized_payload["items"] = list(dataset_payload["eval_protocol_json"].get("metric_list", []))
        output.normalized_payload["data"] = {
            **(output.normalized_payload.get("data") if isinstance(output.normalized_payload.get("data"), dict) else {}),
            **request.to_dict(),
        }
        return output


class DatasetCodexCLIExecutor(CodexCLIExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        prepared = super().prepare_request(contract)
        prompt_text = build_dataset_prompt(contract, DatasetRequest.from_contract(contract))
        prepared["prompt_text"] = prompt_text
        with open(prepared["prompt_path"], "w", encoding="utf-8") as handle:
            handle.write(prompt_text)
        schema_path = Path(prepared["prompt_path"]).parent / "output_schema.json"
        schema_path.write_text(json.dumps(build_dataset_codex_schema(), ensure_ascii=False, indent=2), encoding="utf-8")
        if "--output-schema" not in prepared["args"]:
            prepared["args"].extend(["--output-schema", str(schema_path)])
        return prepared

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        output = super().normalize_output(contract, prepared_request, execution_result, collected_response)
        request = DatasetRequest.from_contract(contract)
        dataset_payload = extract_dataset_payload(output.normalized_payload, request)
        output.normalized_payload.update(dataset_payload)
        output.normalized_payload = _ensure_dataset_location(contract, output.normalized_payload, request)
        output.normalized_payload["items"] = list(dataset_payload["eval_protocol_json"].get("metric_list", []))
        output.normalized_payload["data"] = {
            **(output.normalized_payload.get("data") if isinstance(output.normalized_payload.get("data"), dict) else {}),
            **request.to_dict(),
        }
        return output


class DatasetValidator(BaseValidator):
    def validate_input(self, contract: AgentRuntimeInput) -> list[str]:
        errors = super().validate_input(contract)
        request = DatasetRequest.from_contract(contract)
        if not request.research_direction:
            errors.append("research_direction is required")
        if not request.task_type:
            errors.append("task_type is required")
        return errors

    def validate_payload(self, contract: AgentRuntimeInput, payload: dict[str, Any] | None) -> list[str]:
        errors = [
            item
            for item in super().validate_payload(contract, payload)
            if item != "normalized_payload.dataset_asset_ref cannot be empty"
        ]
        if payload is None:
            return errors
        if not isinstance(payload.get("dataset_asset_ref"), str):
            errors.append("normalized_payload.dataset_asset_ref must be a string")
        if not isinstance(payload.get("dataset_location"), str):
            errors.append("normalized_payload.dataset_location must be a string")
        if not isinstance(payload.get("eval_protocol_json"), dict):
            errors.append("normalized_payload.eval_protocol_json must be an object")
        if not isinstance(payload.get("metric_schema_json"), dict):
            errors.append("normalized_payload.metric_schema_json must be an object")
        if not isinstance(payload.get("split_strategy"), str) or not payload.get("split_strategy", "").strip():
            errors.append("normalized_payload.split_strategy must be a non-empty string")
        if not isinstance(payload.get("notes_md"), str) or not payload.get("notes_md", "").strip():
            errors.append("normalized_payload.notes_md must be a non-empty string")
        protocol = payload.get("eval_protocol_json", {})
        if isinstance(protocol, dict):
            if not isinstance(protocol.get("task_type"), str):
                errors.append("normalized_payload.eval_protocol_json.task_type must be a string")
            if not isinstance(protocol.get("metric_list"), list):
                errors.append("normalized_payload.eval_protocol_json.metric_list must be an array")
            if not isinstance(protocol.get("evaluation_steps"), list):
                errors.append("normalized_payload.eval_protocol_json.evaluation_steps must be an array")
            if not isinstance(protocol.get("data_split"), dict):
                errors.append("normalized_payload.eval_protocol_json.data_split must be an object")
            if not isinstance(protocol.get("baseline_needed"), bool):
                errors.append("normalized_payload.eval_protocol_json.baseline_needed must be a boolean")
            if not isinstance(protocol.get("report_template"), dict):
                errors.append("normalized_payload.eval_protocol_json.report_template must be an object")
        return errors


class DatasetRepairer(BaseRepairer):
    def repair(
        self,
        contract: AgentRuntimeInput,
        output: AgentRuntimeOutput,
        errors: list[str],
    ) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        repaired, actions = super().repair(contract, output, errors)
        request = DatasetRequest.from_contract(contract)
        dataset_payload = extract_dataset_payload(repaired.normalized_payload, request)
        repaired.normalized_payload.update(dataset_payload)
        repaired.normalized_payload = _ensure_dataset_location(contract, repaired.normalized_payload, request)
        repaired.normalized_payload["items"] = list(dataset_payload["eval_protocol_json"].get("metric_list", []))
        data = repaired.normalized_payload.get("data")
        if not isinstance(data, dict):
            data = {}
        data.update(request.to_dict())
        repaired.normalized_payload["data"] = data
        if not isinstance(repaired.normalized_payload.get("summary"), str) or not repaired.normalized_payload["summary"].strip():
            repaired.normalized_payload["summary"] = "Dataset agent repaired structured fetch/register output."
        actions.append(
            AgentRepairAction(
                action="repair_dataset_payload",
                status="applied",
                detail="Normalized Dataset Agent fetch/register and eval protocol payload.",
                metadata={"task_type": request.task_type},
            )
        )
        return repaired, actions


class DatasetAgent(BaseAgent):
    pass


def build_dataset_agent(contract: AgentRuntimeInput) -> DatasetAgent:
    if contract.execution_mode == "api":
        executor = DatasetApiExecutor()
    elif contract.execution_mode == "codex_cli":
        executor = DatasetCodexCLIExecutor()
    else:
        executor = DatasetMockExecutor()
    return DatasetAgent(executor=executor, validator=DatasetValidator(), repairer=DatasetRepairer())
