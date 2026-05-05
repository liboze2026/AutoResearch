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


INSIGHT_SCHEMA_REF = "schemas/insight-output-v1.json"
VALID_FOCUS = {"method", "contribution", "limitation", "novelty"}
JSON_BLOCK_PATTERN = re.compile(r"```(?:json)?\s*(.*?)```", re.IGNORECASE | re.DOTALL)


@dataclass
class InsightRequest:
    paper_id: str = ""
    parsed_content_ref: str = ""
    focus: str = ""
    parsed_text: str = ""

    @classmethod
    def from_contract(cls, contract: AgentRuntimeInput) -> "InsightRequest":
        metadata = dict(contract.metadata or {})
        paper_id = str(metadata.get("paper_id", "")).strip()
        parsed_content_ref = str(metadata.get("parsed_content_ref", "")).strip()
        focus = str(metadata.get("focus", "")).strip().lower()
        if focus not in VALID_FOCUS:
            focus = ""
        for ref in contract.input_refs:
            if not paper_id and ref.ref_type == "paper" and ref.ref_id:
                paper_id = ref.ref_id
            if not parsed_content_ref and ref.ref_type == "parsed_content":
                parsed_content_ref = ref.ref_path
                if not paper_id and ref.ref_id:
                    paper_id = ref.ref_id
        parsed_text = ""
        if parsed_content_ref:
            path = Path(parsed_content_ref)
            if path.exists():
                parsed_text = path.read_text(encoding="utf-8-sig")
        return cls(
            paper_id=paper_id,
            parsed_content_ref=parsed_content_ref,
            focus=focus,
            parsed_text=parsed_text,
        )

    def to_dict(self) -> dict[str, Any]:
        return {
            "paper_id": self.paper_id,
            "parsed_content_ref": self.parsed_content_ref,
            "focus": self.focus,
        }


def extract_title(parsed_text: str, fallback: str) -> str:
    for line in parsed_text.splitlines():
        candidate = line.strip().lstrip("\ufeff")
        if candidate.startswith("#"):
            return candidate.lstrip("# ").strip()
        if len(candidate) >= 12:
            return candidate
    return fallback or "Untitled Paper"


def build_summary(title: str, focus: str) -> str:
    if focus == "method":
        return f"{title} focuses on a controlled method pipeline with reproducible parsing, validation, and research asset persistence."
    if focus == "contribution":
        return f"{title} contributes a staged research workflow that separates paper ingestion, structured insight extraction, and downstream automation."
    if focus == "limitation":
        return f"{title} still depends on deterministic mock extraction in early stages, so its insights emphasize control and traceability over deep semantic understanding."
    if focus == "novelty":
        return f"{title} is novel in how it turns paper processing into a controlled, schema-driven asset flow instead of an unconstrained chat workflow."
    return f"{title} is summarized as a controlled research paper asset whose main value is making ingestion and insight extraction auditable and reusable."


def build_contributions(title: str, focus: str) -> list[str]:
    items = [
        f"Frames {title} as a reusable research asset instead of a transient document.",
        "Separates ingestion, parsing, schema validation, and downstream automation into controlled stages.",
    ]
    if focus == "contribution":
        items.append("Emphasizes explicit contract-driven outputs for later research agents.")
    return items


def build_methods(focus: str) -> list[str]:
    items = [
        "Use parsed markdown as the stable intermediate representation.",
        "Use schema-first validation and repair before persisting insights.",
    ]
    if focus == "method":
        items.append("Prefer deterministic workspace artifacts to preserve auditability.")
    return items


def build_limitations(focus: str) -> list[str]:
    items = [
        "Current insight generation can still rely on mock or fallback execution paths.",
        "Novelty and contribution extraction remain shallow when no real model call is enabled.",
    ]
    if focus == "limitation":
        items.append("Limitations become more visible when parsed input is sparse or only metadata-derived.")
    return items


def build_novelty_points(title: str, focus: str) -> list[str]:
    items = [
        f"{title} treats insight extraction as a controlled node inside a larger research pipeline.",
        "The output is designed to be persisted, validated, repaired, and consumed by downstream agents.",
    ]
    if focus == "novelty":
        items.append("Novelty is framed as orchestration discipline rather than just model cleverness.")
    return items


def build_mock_output(request: InsightRequest) -> dict[str, Any]:
    title = extract_title(request.parsed_text, request.paper_id or "Untitled Paper")
    summary_md = build_summary(title, request.focus)
    contributions = build_contributions(title, request.focus)
    methods = build_methods(request.focus)
    limitations = build_limitations(request.focus)
    novelty_points = build_novelty_points(title, request.focus)
    return {
        "summary_md": summary_md,
        "contributions_json": contributions,
        "methods_json": methods,
        "limitations_json": limitations,
        "novelty_points": novelty_points,
    }


def build_insight_prompt(contract: AgentRuntimeInput, request: InsightRequest) -> str:
    preview = request.parsed_text[:4000]
    return (
        "You are MRAG Insight Agent running in controlled mode.\n"
        "Do not inspect the workspace. Do not run shell commands. Do not browse.\n"
        "Return valid JSON only.\n"
        "The JSON must contain: summary_md, contributions_json, methods_json, limitations_json, novelty_points.\n"
        "All *_json fields should be arrays of concise strings.\n"
        "Prefer the shortest valid answer that satisfies the schema and current task.\n"
        "Do not add markdown fences.\n\n"
        f"job_id: {contract.job_id}\n"
        f"paper_id: {request.paper_id}\n"
        f"parsed_content_ref: {request.parsed_content_ref}\n"
        f"focus: {request.focus}\n"
        f"parsed_preview: {json.dumps(preview, ensure_ascii=False)}\n"
    )


def build_insight_codex_schema() -> dict[str, Any]:
    string_array = {
        "type": "array",
        "items": {"type": "string"},
    }
    return {
        "type": "object",
        "required": [
            "summary_md",
            "contributions_json",
            "methods_json",
            "limitations_json",
            "novelty_points",
        ],
        "properties": {
            "summary_md": {"type": "string"},
            "contributions_json": string_array,
            "methods_json": string_array,
            "limitations_json": string_array,
            "novelty_points": string_array,
        },
        "additionalProperties": False,
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


def extract_insight_payload(payload: dict[str, Any], request: InsightRequest) -> dict[str, Any]:
    result = {
        "summary_md": "",
        "contributions_json": [],
        "methods_json": [],
        "limitations_json": [],
        "novelty_points": [],
    }
    direct_fields = {
        "summary_md": ["summary_md", "summaryMd", "message"],
        "contributions_json": ["contributions_json", "contributions", "contribution_points"],
        "methods_json": ["methods_json", "methods", "method_points"],
        "limitations_json": ["limitations_json", "limitations", "limitation_points"],
        "novelty_points": ["novelty_points", "novelty", "novelty_points_json"],
    }
    for target, aliases in direct_fields.items():
        for alias in aliases:
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
        for target, aliases in direct_fields.items():
            if result[target] not in (None, "", []):
                continue
            for alias in aliases:
                if alias in nested and nested[alias] not in (None, ""):
                    result[target] = nested[alias]
                    break

    mock_defaults = build_mock_output(request)
    summary_md = str(result["summary_md"]).strip() or mock_defaults["summary_md"]
    contributions = _normalize_string_list(result["contributions_json"]) or mock_defaults["contributions_json"]
    methods = _normalize_string_list(result["methods_json"]) or mock_defaults["methods_json"]
    limitations = _normalize_string_list(result["limitations_json"]) or mock_defaults["limitations_json"]
    novelty_points = _normalize_string_list(result["novelty_points"]) or mock_defaults["novelty_points"]
    return {
        "summary_md": summary_md,
        "contributions_json": contributions,
        "methods_json": methods,
        "limitations_json": limitations,
        "novelty_points": novelty_points,
    }


class InsightMockExecutor(MockExecutor):
    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = InsightRequest.from_contract(contract)
        insight = build_mock_output(request)
        normalized_payload = self.base_payload(contract, "mock")
        normalized_payload.update(
            {
                "summary": f"Insight mock produced structured insight for paper {request.paper_id or 'unknown'}.",
                "items": list(insight["novelty_points"]),
                "data": {
                    "paper_id": request.paper_id,
                    "parsed_content_ref": request.parsed_content_ref,
                    "focus": request.focus,
                },
                "metadata": {"agent_role": "insight", "synthetic": True},
                **insight,
            }
        )
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[],
            warnings=["insight mock executor used; results are deterministic placeholders."],
            error_message="",
        )


class InsightApiExecutor(ApiExecutor):
    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        output = super().normalize_output(contract, prepared_request, execution_result, collected_response)
        request = InsightRequest.from_contract(contract)
        insight = extract_insight_payload(output.normalized_payload, request)
        output.normalized_payload.update(insight)
        output.normalized_payload["items"] = list(insight["novelty_points"])
        output.normalized_payload["data"] = {
            **(output.normalized_payload.get("data") if isinstance(output.normalized_payload.get("data"), dict) else {}),
            **request.to_dict(),
        }
        return output


class InsightCodexCLIExecutor(CodexCLIExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        prepared = super().prepare_request(contract)
        prompt_text = build_insight_prompt(contract, InsightRequest.from_contract(contract))
        prepared["prompt_text"] = prompt_text
        prompt_path = Path(prepared["prompt_path"])
        prompt_path.parent.mkdir(parents=True, exist_ok=True)
        prompt_path.write_text(prompt_text, encoding="utf-8")
        schema_path = prompt_path.parent / "output_schema.json"
        schema_path.write_text(json.dumps(build_insight_codex_schema(), ensure_ascii=False, indent=2), encoding="utf-8")
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
        request = InsightRequest.from_contract(contract)
        insight = extract_insight_payload(output.normalized_payload, request)
        output.normalized_payload.update(insight)
        output.normalized_payload["items"] = list(insight["novelty_points"])
        output.normalized_payload["data"] = {
            **(output.normalized_payload.get("data") if isinstance(output.normalized_payload.get("data"), dict) else {}),
            **request.to_dict(),
        }
        return output


class InsightValidator(BaseValidator):
    def validate_input(self, contract: AgentRuntimeInput) -> list[str]:
        errors = super().validate_input(contract)
        request = InsightRequest.from_contract(contract)
        if not request.paper_id:
            errors.append("paper_id is required")
        if not request.parsed_content_ref:
            errors.append("parsed_content_ref is required")
        return errors

    def validate_payload(self, contract: AgentRuntimeInput, payload: dict[str, Any] | None) -> list[str]:
        errors = super().validate_payload(contract, payload)
        if payload is None:
            return errors
        if not isinstance(payload.get("summary_md"), str) or not payload.get("summary_md", "").strip():
            errors.append("normalized_payload.summary_md must be a non-empty string")
        for field_name in ("contributions_json", "methods_json", "limitations_json", "novelty_points"):
            value = payload.get(field_name)
            if not isinstance(value, list):
                errors.append(f"normalized_payload.{field_name} must be an array")
                continue
            for index, item in enumerate(value):
                if not isinstance(item, str):
                    errors.append(f"normalized_payload.{field_name}[{index}] must be a string")
        return errors


class InsightRepairer(BaseRepairer):
    def repair(
        self,
        contract: AgentRuntimeInput,
        output: AgentRuntimeOutput,
        errors: list[str],
    ) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        repaired, actions = super().repair(contract, output, errors)
        request = InsightRequest.from_contract(contract)
        insight = extract_insight_payload(repaired.normalized_payload, request)
        repaired.normalized_payload.update(insight)
        repaired.normalized_payload["items"] = list(insight["novelty_points"])
        data = repaired.normalized_payload.get("data")
        if not isinstance(data, dict):
            data = {}
        data.update(request.to_dict())
        repaired.normalized_payload["data"] = data
        if not isinstance(repaired.normalized_payload.get("summary"), str) or not repaired.normalized_payload["summary"].strip():
            repaired.normalized_payload["summary"] = f"Insight repaired output for paper {request.paper_id or 'unknown'}."
        actions.append(
            AgentRepairAction(
                action="repair_insight_payload",
                status="applied",
                detail="Normalized Insight Agent structured payload.",
                metadata={"paper_id": request.paper_id},
            )
        )
        return repaired, actions


class InsightAgent(BaseAgent):
    pass


def build_insight_agent(contract: AgentRuntimeInput) -> InsightAgent:
    if contract.execution_mode == "api":
        executor = InsightApiExecutor()
    elif contract.execution_mode == "codex_cli":
        executor = InsightCodexCLIExecutor()
    else:
        executor = InsightMockExecutor()
    return InsightAgent(executor=executor, validator=InsightValidator(), repairer=InsightRepairer())
