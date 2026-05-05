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


WRITER_SCHEMA_REF = "schemas/writer-output-v1.json"
JSON_BLOCK_PATTERN = re.compile(r"```(?:json)?\s*(.*?)```", re.IGNORECASE | re.DOTALL)


@dataclass
class WriterRequest:
    paper_template_ref: str = ""
    template_text: str = ""
    idea_refs: list[str] = field(default_factory=list)
    experiment_result_refs: list[str] = field(default_factory=list)
    comparison_refs: list[str] = field(default_factory=list)
    citation_refs: list[str] = field(default_factory=list)
    ideas: list[dict[str, Any]] = field(default_factory=list)
    experiment_results: list[dict[str, Any]] = field(default_factory=list)
    comparisons: list[dict[str, Any]] = field(default_factory=list)
    citations: list[dict[str, Any]] = field(default_factory=list)

    @classmethod
    def from_contract(cls, contract: AgentRuntimeInput) -> "WriterRequest":
        metadata = dict(contract.metadata or {})
        template_ref = str(metadata.get("paper_template_ref", "")).strip()
        if not template_ref:
            for ref in contract.input_refs:
                if ref.ref_type == "paper_template":
                    template_ref = ref.ref_path or ref.ref_id
                    break
        return cls(
            paper_template_ref=template_ref,
            template_text=_load_text(template_ref),
            idea_refs=_normalize_string_list(metadata.get("idea_refs", [])),
            experiment_result_refs=_normalize_string_list(metadata.get("experiment_result_refs", [])),
            comparison_refs=_normalize_string_list(metadata.get("comparison_refs", [])),
            citation_refs=_normalize_string_list(metadata.get("citation_refs", [])),
            ideas=_extract_ref_maps(contract.input_refs, "idea", metadata.get("ideas", [])),
            experiment_results=_extract_ref_maps(contract.input_refs, "experiment_result", metadata.get("experiment_results", [])),
            comparisons=_extract_ref_maps(contract.input_refs, "comparison", metadata.get("comparisons", [])),
            citations=_extract_ref_maps(contract.input_refs, "citation", metadata.get("citations", [])),
        )

    def to_dict(self) -> dict[str, Any]:
        return {
            "paper_template_ref": self.paper_template_ref,
            "idea_refs": list(self.idea_refs),
            "experiment_result_refs": list(self.experiment_result_refs),
            "comparison_refs": list(self.comparison_refs),
            "citation_refs": list(self.citation_refs),
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


def _load_text(path: str) -> str:
    path = str(path or "").strip()
    if not path:
        return ""
    try:
        return Path(path).read_text(encoding="utf-8-sig")
    except OSError:
        return ""


def _extract_ref_maps(input_refs: list[Any], ref_type: str, metadata_items: Any) -> list[dict[str, Any]]:
    out: list[dict[str, Any]] = []
    if isinstance(metadata_items, list):
        for item in metadata_items:
            if isinstance(item, dict):
                out.append(dict(item))
    for ref in input_refs:
        if getattr(ref, "ref_type", "") != ref_type:
            continue
        item: dict[str, Any] = {}
        if getattr(ref, "ref_id", ""):
            item[f"{ref_type}_ref"] = ref.ref_id
        if getattr(ref, "ref_path", ""):
            item[f"{ref_type}_path"] = ref.ref_path
        if isinstance(getattr(ref, "metadata", None), dict):
            item.update(ref.metadata)
        if item:
            out.append(item)
    return out


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


def _pick_title(request: WriterRequest) -> str:
    if request.ideas:
        title = str(request.ideas[0].get("title", "")).strip()
        if title:
            return title
    return "Controlled Research Draft"


def _pick_metric_lines(request: WriterRequest) -> list[str]:
    if not request.experiment_results:
        return ["The first controlled run is prepared to report accuracy and loss under the shared template."]
    metrics = request.experiment_results[0].get("metrics", {})
    if not isinstance(metrics, dict):
        return ["The first controlled run completed under the shared template."]
    values = metrics.get("values", {})
    if isinstance(values, dict) and values:
        return [f"{key}: {value}" for key, value in values.items()]
    lines: list[str] = []
    for key, value in metrics.items():
        if key in {"primary_metric", "values"}:
            continue
        lines.append(f"{key}: {value}")
    return lines or ["Metrics are available in the linked experiment result."]


def _build_picture_plan(request: WriterRequest) -> list[dict[str, Any]]:
    figure_sources = list(request.experiment_result_refs[:1]) + list(request.comparison_refs[:1])
    return [
        {
            "figure_id": "fig_1",
            "figure_type": "workflow_mock",
            "title": "Controlled Stage3 Pipeline Overview",
            "description": "Mock flowchart showing idea -> plan -> coding/evaluation -> writer.",
            "source_refs": [item for item in figure_sources if item],
            "placeholder_notes": [
                "Picture agent is mock in v1.",
                "Replace with a rendered pipeline diagram in a later stage.",
            ],
        },
        {
            "figure_id": "fig_2",
            "figure_type": "comparison_mock",
            "title": "Baseline vs Controlled Run Comparison",
            "description": "Mock comparison chart placeholder summarizing primary metrics against baseline references.",
            "source_refs": list(request.comparison_refs[:2]),
            "placeholder_notes": [
                "Use bar chart or table rendering later.",
                "Current output is a controlled description only.",
            ],
        },
    ]


def build_writer_payload(request: WriterRequest) -> dict[str, Any]:
    title = _pick_title(request)
    metric_lines = _pick_metric_lines(request)
    primary_idea = request.ideas[0] if request.ideas else {}
    expected_advantage = str(primary_idea.get("expected_advantage", "improve controllability and reproducibility")).strip()
    research_direction = str(primary_idea.get("research_direction", "controlled experimentation")).strip()
    abstract = (
        f"This draft studies {research_direction} under a controlled MRAG stage3 pipeline. "
        f"We reuse structured idea, experiment planning, template-bounded coding, and stage2 evaluation to produce a traceable first result. "
        f"The current draft emphasizes reproducibility, clear provenance, and a stable writing contract."
    )
    introduction = (
        "Recent research workflows often rely on loosely coupled notes and ad hoc scripting, which makes iterative experimentation difficult to audit. "
        "Our stage3 pipeline frames each step as a controlled agent node with schema validation, repair, and persistent artifacts. "
        f"This draft focuses on how that controlled design supports {research_direction} while keeping the writing process tied to concrete upstream assets."
    )
    method = (
        "We use a unified runtime and contract layer so all intermediate assets remain structured. "
        "The planner constrains execution to shared templates, the coding/evaluator stage runs through the existing stage2 experiment lifecycle, "
        "and the writer aggregates idea, run, comparison, and citation references into a stable manuscript scaffold. "
        f"The expected benefit is to {expected_advantage} without leaving the audited template surface."
    )
    experiments = (
        "We report the first controlled experiment using the shared training template and the existing stage2 evaluation path.\n\n"
        + "\n".join(f"- {line}" for line in metric_lines)
        + "\n\n"
        + "Comparisons are sourced from the structured comparison artifacts linked into this draft."
    )
    conclusion = (
        "The first-stage manuscript shows that a controlled agent pipeline can turn structured idea and run artifacts into a stable paper draft. "
        "The current version prioritizes completeness, traceability, and contract stability over polished language or final figures."
    )
    references_stub = []
    for citation in request.citations:
        text = str(citation.get("citation_text", citation.get("title", citation.get("citation_ref", "")))).strip()
        if text:
            references_stub.append(text)
    if not references_stub:
        references_stub = [f"[Ref {idx + 1}] Placeholder citation from {ref}" for idx, ref in enumerate(request.citation_refs[:5])]
    if not references_stub:
        references_stub = ["[Ref 1] Placeholder citation to be filled from controlled paper assets."]
    return {
        "title": title,
        "abstract": abstract,
        "introduction": introduction,
        "method": method,
        "experiments": experiments,
        "conclusion": conclusion,
        "references_stub": references_stub,
        "figure_plan": _build_picture_plan(request),
    }


def build_writer_prompt(contract: AgentRuntimeInput, request: WriterRequest) -> str:
    template_excerpt = request.template_text[:1200]
    return (
        "You are MRAG Writer Agent running in controlled mode.\n"
        "Do not inspect the workspace. Do not run shell commands. Do not browse.\n"
        "Picture generation is mock-only in v1.\n"
        "Return valid JSON only.\n"
        "The JSON must include: title, abstract, introduction, method, experiments, conclusion, references_stub, figure_plan.\n"
        "Prefer the shortest valid answer that satisfies the schema and current task.\n"
        "Do not add markdown fences.\n\n"
        f"job_id: {contract.job_id}\n"
        f"paper_template_ref: {request.paper_template_ref}\n"
        f"template_excerpt: {json.dumps(template_excerpt, ensure_ascii=False)}\n"
        f"ideas: {json.dumps(request.ideas, ensure_ascii=False)}\n"
        f"experiment_results: {json.dumps(request.experiment_results, ensure_ascii=False)}\n"
        f"comparisons: {json.dumps(request.comparisons, ensure_ascii=False)}\n"
        f"citations: {json.dumps(request.citations, ensure_ascii=False)}\n"
    )


def build_writer_codex_schema() -> dict[str, Any]:
    string_array = {"type": "array", "items": {"type": "string"}}
    return {
        "type": "object",
        "required": [
            "title",
            "abstract",
            "introduction",
            "method",
            "experiments",
            "conclusion",
            "references_stub",
            "figure_plan",
        ],
        "properties": {
            "title": {"type": "string"},
            "abstract": {"type": "string"},
            "introduction": {"type": "string"},
            "method": {"type": "string"},
            "experiments": {"type": "string"},
            "conclusion": {"type": "string"},
            "references_stub": string_array,
            "figure_plan": {
                "type": "array",
                "items": {
                    "type": "object",
                    "required": ["figure_id", "figure_type", "title", "description", "source_refs", "placeholder_notes"],
                    "properties": {
                        "figure_id": {"type": "string"},
                        "figure_type": {"type": "string"},
                        "title": {"type": "string"},
                        "description": {"type": "string"},
                        "source_refs": string_array,
                        "placeholder_notes": string_array,
                    },
                    "additionalProperties": False,
                },
            },
        },
        "additionalProperties": False,
    }


def extract_writer_payload(payload: dict[str, Any], request: WriterRequest) -> dict[str, Any]:
    result = build_writer_payload(request)
    aliases = {
        "title": ["title"],
        "abstract": ["abstract"],
        "introduction": ["introduction", "intro"],
        "method": ["method", "methods"],
        "experiments": ["experiments", "experiment_section"],
        "conclusion": ["conclusion"],
        "references_stub": ["references_stub", "referencesStub", "references"],
        "figure_plan": ["figure_plan", "figurePlan", "figures"],
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
    result["title"] = str(result["title"]).strip() or build_writer_payload(request)["title"]
    for field in ("abstract", "introduction", "method", "experiments", "conclusion"):
        result[field] = str(result[field]).strip() or build_writer_payload(request)[field]
    if not isinstance(result["references_stub"], list):
        result["references_stub"] = build_writer_payload(request)["references_stub"]
    if not isinstance(result["figure_plan"], list):
        result["figure_plan"] = build_writer_payload(request)["figure_plan"]
    return result


class WriterMockExecutor(MockExecutor):
    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = WriterRequest.from_contract(contract)
        writer_payload = build_writer_payload(request)
        normalized_payload = self.base_payload(contract, "mock")
        normalized_payload.update(
            {
                "summary": "Writer mock produced a structured draft scaffold with mock picture placeholders.",
                "items": [writer_payload["title"], f"{len(writer_payload['figure_plan'])} figure placeholder(s)"],
                "data": request.to_dict(),
                "metadata": {"agent_role": "writer", "picture_mode": "mock", "synthetic": True},
                **writer_payload,
            }
        )
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[],
            warnings=["picture agent remains mock in v1"],
            validation_status="pending",
            repair_status="pending",
            validation_errors=[],
            error_message="",
        )


class WriterApiExecutor(ApiExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        request = WriterRequest.from_contract(contract)
        prepared = super().prepare_request(contract)
        prepared["prompt"] = build_writer_prompt(contract, request)
        prepared["writer_request"] = request.to_dict()
        return prepared

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = WriterRequest.from_contract(contract)
        normalized_payload = self.base_payload(contract, "api")
        payload = extract_writer_payload(collected_response, request)
        normalized_payload.update(
            {
                "summary": "Writer API executor normalized a structured draft response.",
                "items": [payload["title"], f"{len(payload['figure_plan'])} figure plan item(s)"],
                "data": request.to_dict(),
                "metadata": {"agent_role": "writer", "picture_mode": "mock", "executor_response": collected_response},
                **payload,
            }
        )
        return AgentRuntimeOutput(
            status="succeeded" if not execution_result.get("error") else "failed",
            normalized_payload=normalized_payload,
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[],
            warnings=["picture agent remains mock in v1"],
            validation_status="pending",
            repair_status="pending",
            validation_errors=[],
            error_message=str(execution_result.get("error", "")).strip(),
        )


class WriterCodexCLIExecutor(CodexCLIExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        request = WriterRequest.from_contract(contract)
        prepared = super().prepare_request(contract)
        prompt_text = build_writer_prompt(contract, request)
        prepared["prompt_text"] = prompt_text
        prompt_path = Path(prepared["prompt_path"])
        prompt_path.parent.mkdir(parents=True, exist_ok=True)
        prompt_path.write_text(prompt_text, encoding="utf-8")
        schema_path = prompt_path.parent / "output_schema.json"
        schema_path.write_text(json.dumps(build_writer_codex_schema(), ensure_ascii=False, indent=2), encoding="utf-8")
        if "--output-schema" not in prepared["args"]:
            prepared["args"].extend(["--output-schema", str(schema_path)])
        prepared["writer_request"] = request.to_dict()
        return prepared

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = WriterRequest.from_contract(contract)
        output = super().normalize_output(contract, prepared_request, execution_result, collected_response)
        payload = extract_writer_payload(output.normalized_payload, request)
        warnings = list(output.warnings)
        warnings.append("picture agent remains mock in v1")
        output.warnings = warnings
        output.normalized_payload.update(
            {
                "summary": "Writer Codex CLI executor normalized a structured draft response.",
                "items": [payload["title"], f"{len(payload['figure_plan'])} figure plan item(s)"],
                "data": request.to_dict(),
                "metadata": {
                    "agent_role": "writer",
                    "picture_mode": "mock",
                    "stdout": execution_result.get("stdout", ""),
                    "stderr": execution_result.get("stderr", ""),
                },
                **payload,
            }
        )
        return output


class WriterValidator(BaseValidator):
    pass


class WriterRepairer(BaseRepairer):
    def repair(
        self,
        contract: AgentRuntimeInput,
        output: AgentRuntimeOutput,
        errors: list[str],
    ) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        repaired, actions = super().repair(contract, output, errors)
        request = WriterRequest.from_contract(contract)
        payload = extract_writer_payload(repaired.normalized_payload, request)
        repaired.normalized_payload.update(payload)
        return repaired, actions


class WriterAgent(BaseAgent):
    pass


def build_writer_agent(contract: AgentRuntimeInput) -> WriterAgent:
    if contract.execution_mode == "api":
        executor = WriterApiExecutor()
    elif contract.execution_mode == "codex_cli":
        executor = WriterCodexCLIExecutor()
    else:
        executor = WriterMockExecutor()
    return WriterAgent(executor=executor, validator=WriterValidator(), repairer=WriterRepairer())
