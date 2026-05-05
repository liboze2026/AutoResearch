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


IDEA_SCHEMA_REF = "schemas/idea-generator-output-v1.json"
JSON_BLOCK_PATTERN = re.compile(r"```(?:json)?\s*(.*?)```", re.IGNORECASE | re.DOTALL)


@dataclass
class InsightContext:
    insight_id: str = ""
    paper_id: str = ""
    paper_title: str = ""
    summary_md: str = ""
    contributions: list[str] = field(default_factory=list)
    novelty_points: list[str] = field(default_factory=list)


@dataclass
class DatasetContext:
    dataset_asset_id: str = ""
    name: str = ""
    task_type: str = ""
    evalplan_path: str = ""
    metric_list: list[str] = field(default_factory=list)
    split_strategy: str = ""


@dataclass
class IdeaRequest:
    paper_insight_refs: list[str] = field(default_factory=list)
    dataset_asset_refs: list[str] = field(default_factory=list)
    human_hints: list[str] = field(default_factory=list)
    manual_idea: dict[str, Any] = field(default_factory=dict)
    insights: list[InsightContext] = field(default_factory=list)
    datasets: list[DatasetContext] = field(default_factory=list)

    @classmethod
    def from_contract(cls, contract: AgentRuntimeInput) -> "IdeaRequest":
        metadata = dict(contract.metadata or {})
        request = cls(
            paper_insight_refs=_normalize_string_list(metadata.get("paper_insight_refs")),
            dataset_asset_refs=_normalize_string_list(metadata.get("dataset_asset_refs")),
            human_hints=_normalize_string_list(metadata.get("human_hints")),
            manual_idea=dict(metadata.get("manual_idea") or {}) if isinstance(metadata.get("manual_idea"), dict) else {},
            insights=_normalize_insights(metadata.get("paper_insights")),
            datasets=_normalize_datasets(metadata.get("dataset_assets")),
        )
        for ref in contract.input_refs:
            if ref.ref_type == "insight":
                if ref.ref_id and ref.ref_id not in request.paper_insight_refs:
                    request.paper_insight_refs.append(ref.ref_id)
                context = _load_insight_from_ref(ref.ref_id, ref.ref_path)
                if context is not None:
                    request.insights.append(context)
            if ref.ref_type == "dataset_asset":
                if ref.ref_id and ref.ref_id not in request.dataset_asset_refs:
                    request.dataset_asset_refs.append(ref.ref_id)
                context = _load_dataset_from_ref(ref.ref_id, ref.ref_path, ref.metadata or {})
                if context is not None:
                    request.datasets.append(context)
        request.insights = _dedupe_insights(request.insights)
        request.datasets = _dedupe_datasets(request.datasets)
        return request

    def to_dict(self) -> dict[str, Any]:
        return {
            "paper_insight_refs": list(self.paper_insight_refs),
            "dataset_asset_refs": list(self.dataset_asset_refs),
            "human_hints": list(self.human_hints),
            "manual_idea": dict(self.manual_idea),
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


def _safe_read_json(path: str) -> dict[str, Any]:
    candidate = Path(path)
    if not path or not candidate.exists() or not candidate.is_file():
        return {}
    try:
        parsed = json.loads(candidate.read_text(encoding="utf-8-sig"))
    except (OSError, json.JSONDecodeError):
        return {}
    return parsed if isinstance(parsed, dict) else {}


def _normalize_insights(value: Any) -> list[InsightContext]:
    if not isinstance(value, list):
        return []
    out: list[InsightContext] = []
    for item in value:
        if not isinstance(item, dict):
            continue
        out.append(
            InsightContext(
                insight_id=str(item.get("insight_id", "")).strip(),
                paper_id=str(item.get("paper_id", "")).strip(),
                paper_title=str(item.get("paper_title", "")).strip(),
                summary_md=str(item.get("summary_md", "")).strip(),
                contributions=_normalize_string_list(item.get("contributions_json")),
                novelty_points=_normalize_string_list(item.get("novelty_points")),
            )
        )
    return out


def _normalize_datasets(value: Any) -> list[DatasetContext]:
    if not isinstance(value, list):
        return []
    out: list[DatasetContext] = []
    for item in value:
        if not isinstance(item, dict):
            continue
        eval_protocol = item.get("eval_protocol_json", {})
        metric_list = _normalize_string_list(eval_protocol.get("metric_list") if isinstance(eval_protocol, dict) else item.get("metric_list"))
        out.append(
            DatasetContext(
                dataset_asset_id=str(item.get("dataset_asset_id", item.get("dataset_asset_ref", ""))).strip(),
                name=str(item.get("name", "")).strip(),
                task_type=str(item.get("task_type", "")).strip(),
                evalplan_path=str(item.get("evalplan_path", "")).strip(),
                metric_list=metric_list,
                split_strategy=str(item.get("split_strategy", "")).strip(),
            )
        )
    return out


def _load_insight_from_ref(insight_id: str, ref_path: str) -> InsightContext | None:
    payload = _safe_read_json(ref_path)
    if not payload:
        return None
    return InsightContext(
        insight_id=str(payload.get("insight_id", insight_id)).strip() or insight_id,
        paper_id=str(payload.get("paper_id", "")).strip(),
        paper_title=str(payload.get("paper_title", "")).strip(),
        summary_md=str(payload.get("summary_md", "")).strip(),
        contributions=_normalize_string_list(payload.get("contributions_json")),
        novelty_points=_normalize_string_list(payload.get("novelty_points")),
    )


def _load_dataset_from_ref(dataset_asset_id: str, ref_path: str, metadata: dict[str, Any]) -> DatasetContext | None:
    evalplan_path = str(metadata.get("evalplan_path", "")).strip()
    payload = _safe_read_json(evalplan_path)
    if not payload and ref_path and ref_path.endswith(".json"):
        payload = _safe_read_json(ref_path)
        if payload:
            evalplan_path = ref_path
    eval_protocol = payload.get("eval_protocol_json", {}) if isinstance(payload, dict) else {}
    metric_list = _normalize_string_list(eval_protocol.get("metric_list") if isinstance(eval_protocol, dict) else [])
    split_strategy = str(payload.get("split_strategy", "")).strip() if isinstance(payload, dict) else ""
    if not dataset_asset_id and not payload and not metadata:
        return None
    return DatasetContext(
        dataset_asset_id=dataset_asset_id or str(metadata.get("dataset_asset_id", "")).strip(),
        name=str(metadata.get("name", payload.get("dataset_name", ""))).strip() if isinstance(payload, dict) else str(metadata.get("name", "")).strip(),
        task_type=str(metadata.get("task_type", payload.get("task_type", ""))).strip() if isinstance(payload, dict) else str(metadata.get("task_type", "")).strip(),
        evalplan_path=evalplan_path,
        metric_list=metric_list,
        split_strategy=split_strategy,
    )


def _dedupe_insights(items: list[InsightContext]) -> list[InsightContext]:
    out: list[InsightContext] = []
    seen: set[tuple[str, str]] = set()
    for item in items:
        key = (item.insight_id, item.paper_id)
        if key in seen:
            continue
        seen.add(key)
        out.append(item)
    return out


def _dedupe_datasets(items: list[DatasetContext]) -> list[DatasetContext]:
    out: list[DatasetContext] = []
    seen: set[str] = set()
    for item in items:
        key = item.dataset_asset_id or item.evalplan_path
        if not key or key in seen:
            continue
        seen.add(key)
        out.append(item)
    return out


def _first_non_empty(*values: str) -> str:
    for value in values:
        trimmed = str(value).strip()
        if trimmed:
            return trimmed
    return ""


def _bound_priority(value: Any, default: int) -> int:
    try:
        numeric = int(value)
    except (TypeError, ValueError):
        return default
    return max(0, min(100, numeric))


def _bound_confidence(value: Any, default: float) -> float:
    try:
        numeric = float(value)
    except (TypeError, ValueError):
        return default
    return max(0.0, min(1.0, numeric))


def _default_research_direction(request: IdeaRequest) -> str:
    if request.manual_idea:
        return _first_non_empty(str(request.manual_idea.get("research_direction", "")).strip())
    for item in request.insights:
        if item.paper_title:
            return item.paper_title
    for item in request.human_hints:
        if item:
            return item
    return "controlled research idea"


def build_idea_payload(request: IdeaRequest) -> dict[str, Any]:
    if request.manual_idea:
        manual = request.manual_idea
        research_direction = _first_non_empty(str(manual.get("research_direction", "")).strip(), _default_research_direction(request))
        title = _first_non_empty(str(manual.get("title", "")).strip(), f"Standardized idea for {research_direction}")
        hint_block = "\n".join(f"- {item}" for item in request.human_hints[:3])
        description_md = _first_non_empty(
            str(manual.get("description_md", manual.get("descriptionMd", ""))).strip(),
            f"## Goal\nStandardize the manually provided idea around {research_direction}.\n\n## Notes\n{hint_block}".strip(),
        )
        innovation_type = _first_non_empty(str(manual.get("innovation_type", "")).strip(), "human_curated")
        expected_advantage = _first_non_empty(str(manual.get("expected_advantage", "")).strip(), "Keeps the idea pool structured and comparable.")
        risk_points = _normalize_string_list(manual.get("risk_points")) or ["Manual idea may still need experimental scoping."]
        priority = _bound_priority(manual.get("priority"), 82)
        confidence = _bound_confidence(manual.get("confidence"), 0.78)
    else:
        first_insight = request.insights[0] if request.insights else InsightContext()
        first_dataset = request.datasets[0] if request.datasets else DatasetContext()
        research_direction = _default_research_direction(request)
        insight_phrase = _first_non_empty(first_insight.novelty_points[0] if first_insight.novelty_points else "", first_insight.summary_md, research_direction)
        dataset_name = _first_non_empty(first_dataset.name, "available benchmark set")
        title = f"Use {dataset_name} to extend {research_direction}"
        description_md = (
            "## Hypothesis\n"
            f"Turn the insight `{insight_phrase}` into a focused experiment on `{dataset_name}`.\n\n"
            "## Plan\n"
            "- Reuse the dataset asset and its evaluation protocol.\n"
            "- Translate the paper insight into one controlled experimental claim.\n"
            "- Keep implementation scope small enough for stage3 experimentation.\n"
        )
        innovation_type = "insight_plus_dataset"
        expected_advantage = f"Should improve relevance or controllability on {dataset_name} with limited implementation overhead."
        risk_points = [
            "Insight-to-dataset mapping may be shallow without deeper literature review.",
            "Dataset protocol may not fully cover the claimed innovation.",
        ]
        priority = 76 if request.datasets else 68
        confidence = 0.71 if request.insights else 0.58
        research_direction = _default_research_direction(request)
    return {
        "title": title,
        "description_md": description_md,
        "research_direction": research_direction,
        "target_dataset_refs": [item.dataset_asset_id for item in request.datasets if item.dataset_asset_id],
        "dataset_eval_protocol_refs": [item.evalplan_path for item in request.datasets if item.evalplan_path],
        "innovation_type": innovation_type,
        "expected_advantage": expected_advantage,
        "risk_points": risk_points,
        "priority": priority,
        "confidence": confidence,
    }


def build_idea_prompt(contract: AgentRuntimeInput, request: IdeaRequest) -> str:
    return (
        "You are MRAG Idea Generator Agent running in controlled mode.\n"
        "Do not inspect the workspace. Do not run shell commands. Do not browse.\n"
        "Return valid JSON only.\n"
        "The JSON must include: title, description_md, research_direction, target_dataset_refs, "
        "dataset_eval_protocol_refs, innovation_type, expected_advantage, risk_points, priority, confidence.\n"
        "Prefer one concrete, low-scope idea that fits the current controlled experiment boundary.\n"
        "Prefer the shortest valid answer that satisfies the schema and current task.\n"
        "Do not add markdown fences.\n\n"
        f"job_id: {contract.job_id}\n"
        f"paper_insights: {json.dumps([item.__dict__ for item in request.insights], ensure_ascii=False)}\n"
        f"dataset_assets: {json.dumps([item.__dict__ for item in request.datasets], ensure_ascii=False)}\n"
        f"human_hints: {json.dumps(request.human_hints, ensure_ascii=False)}\n"
        f"manual_idea: {json.dumps(request.manual_idea, ensure_ascii=False)}\n"
    )


def build_idea_codex_schema() -> dict[str, Any]:
    string_array = {"type": "array", "items": {"type": "string"}}
    return {
        "type": "object",
        "required": [
            "title",
            "description_md",
            "research_direction",
            "target_dataset_refs",
            "dataset_eval_protocol_refs",
            "innovation_type",
            "expected_advantage",
            "risk_points",
            "priority",
            "confidence",
        ],
        "properties": {
            "title": {"type": "string"},
            "description_md": {"type": "string"},
            "research_direction": {"type": "string"},
            "target_dataset_refs": string_array,
            "dataset_eval_protocol_refs": string_array,
            "innovation_type": {"type": "string"},
            "expected_advantage": {"type": "string"},
            "risk_points": string_array,
            "priority": {"type": "number"},
            "confidence": {"type": "number"},
        },
        "additionalProperties": False,
    }


def extract_idea_payload(payload: dict[str, Any], request: IdeaRequest) -> dict[str, Any]:
    result = build_idea_payload(request)
    defaults = build_idea_payload(request)
    aliases = {
        "title": ["title"],
        "description_md": ["description_md", "descriptionMd", "description", "message"],
        "research_direction": ["research_direction", "researchDirection", "direction"],
        "target_dataset_refs": ["target_dataset_refs", "targetDatasets", "dataset_refs"],
        "dataset_eval_protocol_refs": ["dataset_eval_protocol_refs", "datasetEvalProtocolRefs", "evalplan_refs"],
        "innovation_type": ["innovation_type", "innovationType"],
        "expected_advantage": ["expected_advantage", "expectedAdvantage"],
        "risk_points": ["risk_points", "risks", "riskPoints"],
        "priority": ["priority"],
        "confidence": ["confidence"],
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
    result["title"] = _first_non_empty(str(result["title"]).strip(), defaults["title"])
    result["description_md"] = _first_non_empty(str(result["description_md"]).strip(), defaults["description_md"])
    result["research_direction"] = _first_non_empty(str(result["research_direction"]).strip(), defaults["research_direction"])
    result["target_dataset_refs"] = _normalize_string_list(result["target_dataset_refs"])
    result["dataset_eval_protocol_refs"] = _normalize_string_list(result["dataset_eval_protocol_refs"])
    result["innovation_type"] = _first_non_empty(str(result["innovation_type"]).strip(), defaults["innovation_type"])
    result["expected_advantage"] = _first_non_empty(str(result["expected_advantage"]).strip(), defaults["expected_advantage"])
    result["risk_points"] = _normalize_string_list(result["risk_points"]) or defaults["risk_points"]
    result["priority"] = _bound_priority(result["priority"], defaults["priority"])
    result["confidence"] = _bound_confidence(result["confidence"], defaults["confidence"])
    return result


class IdeaMockExecutor(MockExecutor):
    def normalize_output(self, contract: AgentRuntimeInput, prepared_request: dict[str, Any], execution_result: dict[str, Any], collected_response: dict[str, Any]) -> AgentRuntimeOutput:
        request = IdeaRequest.from_contract(contract)
        idea = build_idea_payload(request)
        normalized_payload = self.base_payload(contract, "mock")
        normalized_payload.update(
            {
                "summary": f"Idea mock produced structured idea {idea['title']}.",
                "items": list(idea["risk_points"]),
                "data": request.to_dict(),
                "metadata": {"agent_role": "idea_generator", "synthetic": True},
                **idea,
            }
        )
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[],
            warnings=["idea generator mock executor used; results are deterministic placeholders."],
            error_message="",
        )


class IdeaApiExecutor(ApiExecutor):
    def normalize_output(self, contract: AgentRuntimeInput, prepared_request: dict[str, Any], execution_result: dict[str, Any], collected_response: dict[str, Any]) -> AgentRuntimeOutput:
        output = super().normalize_output(contract, prepared_request, execution_result, collected_response)
        request = IdeaRequest.from_contract(contract)
        idea = extract_idea_payload(output.normalized_payload, request)
        output.normalized_payload.update(idea)
        output.normalized_payload["items"] = list(idea["risk_points"])
        output.normalized_payload["data"] = {
            **(output.normalized_payload.get("data") if isinstance(output.normalized_payload.get("data"), dict) else {}),
            **request.to_dict(),
        }
        return output


class IdeaCodexCLIExecutor(CodexCLIExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        prepared = super().prepare_request(contract)
        prompt_text = build_idea_prompt(contract, IdeaRequest.from_contract(contract))
        prepared["prompt_text"] = prompt_text
        prompt_path = Path(prepared["prompt_path"])
        prompt_path.parent.mkdir(parents=True, exist_ok=True)
        prompt_path.write_text(prompt_text, encoding="utf-8")
        schema_path = prompt_path.parent / "output_schema.json"
        schema_path.write_text(json.dumps(build_idea_codex_schema(), ensure_ascii=False, indent=2), encoding="utf-8")
        if "--output-schema" not in prepared["args"]:
            prepared["args"].extend(["--output-schema", str(schema_path)])
        return prepared

    def normalize_output(self, contract: AgentRuntimeInput, prepared_request: dict[str, Any], execution_result: dict[str, Any], collected_response: dict[str, Any]) -> AgentRuntimeOutput:
        output = super().normalize_output(contract, prepared_request, execution_result, collected_response)
        request = IdeaRequest.from_contract(contract)
        idea = extract_idea_payload(output.normalized_payload, request)
        output.normalized_payload.update(idea)
        output.normalized_payload["items"] = list(idea["risk_points"])
        output.normalized_payload["data"] = {
            **(output.normalized_payload.get("data") if isinstance(output.normalized_payload.get("data"), dict) else {}),
            **request.to_dict(),
        }
        return output


class IdeaValidator(BaseValidator):
    def validate_input(self, contract: AgentRuntimeInput) -> list[str]:
        errors = super().validate_input(contract)
        request = IdeaRequest.from_contract(contract)
        if not request.paper_insight_refs and not request.manual_idea:
            errors.append("paper_insight_refs is required unless manual_idea is provided")
        return errors

    def validate_payload(self, contract: AgentRuntimeInput, payload: dict[str, Any] | None) -> list[str]:
        errors = super().validate_payload(contract, payload)
        if payload is None:
            return errors
        for field_name in ("title", "description_md", "research_direction", "innovation_type", "expected_advantage"):
            value = payload.get(field_name)
            if not isinstance(value, str) or not value.strip():
                errors.append(f"normalized_payload.{field_name} must be a non-empty string")
        for field_name in ("target_dataset_refs", "dataset_eval_protocol_refs", "risk_points"):
            value = payload.get(field_name)
            if not isinstance(value, list):
                errors.append(f"normalized_payload.{field_name} must be an array")
                continue
            for index, item in enumerate(value):
                if not isinstance(item, str):
                    errors.append(f"normalized_payload.{field_name}[{index}] must be a string")
        priority = payload.get("priority")
        if not isinstance(priority, int):
            errors.append("normalized_payload.priority must be an integer")
        elif priority < 0 or priority > 100:
            errors.append("normalized_payload.priority must be between 0 and 100")
        confidence = payload.get("confidence")
        if not isinstance(confidence, (int, float)):
            errors.append("normalized_payload.confidence must be a number")
        elif float(confidence) < 0 or float(confidence) > 1:
            errors.append("normalized_payload.confidence must be between 0 and 1")
        return errors


class IdeaRepairer(BaseRepairer):
    def repair(self, contract: AgentRuntimeInput, output: AgentRuntimeOutput, errors: list[str]) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        repaired, actions = super().repair(contract, output, errors)
        request = IdeaRequest.from_contract(contract)
        idea = extract_idea_payload(repaired.normalized_payload, request)
        repaired.normalized_payload.update(idea)
        repaired.normalized_payload["items"] = list(idea["risk_points"])
        data = repaired.normalized_payload.get("data")
        if not isinstance(data, dict):
            data = {}
        data.update(request.to_dict())
        repaired.normalized_payload["data"] = data
        if not isinstance(repaired.normalized_payload.get("summary"), str) or not repaired.normalized_payload["summary"].strip():
            repaired.normalized_payload["summary"] = f"Idea generator repaired output for {idea['title']}."
        actions.append(
            AgentRepairAction(
                action="repair_idea_payload",
                status="applied",
                detail="Normalized Idea Generator structured payload.",
                metadata={"title": idea["title"]},
            )
        )
        return repaired, actions


class IdeaAgent(BaseAgent):
    pass


def build_idea_agent(contract: AgentRuntimeInput) -> IdeaAgent:
    if contract.execution_mode == "api":
        executor = IdeaApiExecutor()
    elif contract.execution_mode == "codex_cli":
        executor = IdeaCodexCLIExecutor()
    else:
        executor = IdeaMockExecutor()
    return IdeaAgent(executor=executor, validator=IdeaValidator(), repairer=IdeaRepairer())
