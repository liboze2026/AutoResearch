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


READER_SCHEMA_REF = "schemas/reader-output-v1.json"
READER_SOURCE_SCOPES = {"arxiv", "conference", "journal", "mixed"}
JSON_BLOCK_PATTERN = re.compile(r"```(?:json)?\s*(.*?)```", re.IGNORECASE | re.DOTALL)


@dataclass
class ReaderManualPaper:
    title: str = ""
    abstract: str = ""
    source: str = ""
    year: int = 0
    url: str = ""
    file_status: str = ""
    file_path: str = ""

    @classmethod
    def from_dict(cls, payload: dict[str, Any]) -> "ReaderManualPaper":
        return cls(
            title=str(payload.get("title", "")).strip(),
            abstract=str(payload.get("abstract", "")).strip(),
            source=str(payload.get("source", "")).strip(),
            year=_coerce_year(payload.get("year")),
            url=str(payload.get("url", "")).strip(),
            file_status=str(payload.get("file_status", payload.get("fileStatus", ""))).strip(),
            file_path=str(payload.get("file_path", payload.get("filePath", ""))).strip(),
        )


@dataclass
class ReaderRequest:
    research_direction: str = ""
    keywords: list[str] = field(default_factory=list)
    source_scope: str = "mixed"
    time_range: dict[str, Any] = field(default_factory=dict)
    max_papers: int = 5
    manual_papers: list[ReaderManualPaper] = field(default_factory=list)

    @classmethod
    def from_contract(cls, contract: AgentRuntimeInput) -> "ReaderRequest":
        metadata = dict(contract.metadata or {})
        raw_keywords = metadata.get("keywords", [])
        keywords: list[str]
        if isinstance(raw_keywords, list):
            keywords = [str(item).strip() for item in raw_keywords if str(item).strip()]
        elif isinstance(raw_keywords, str):
            keywords = [item.strip() for item in raw_keywords.split(",") if item.strip()]
        else:
            keywords = []

        raw_scope = str(metadata.get("source_scope", "mixed")).strip().lower() or "mixed"
        source_scope = raw_scope if raw_scope in READER_SOURCE_SCOPES else "mixed"

        raw_time_range = metadata.get("time_range", {})
        time_range = raw_time_range if isinstance(raw_time_range, dict) else {"value": raw_time_range}

        raw_manual = metadata.get("manual_papers", [])
        if not isinstance(raw_manual, list):
            raw_manual = []
        manual_papers = [
            ReaderManualPaper.from_dict(item)
            for item in raw_manual
            if isinstance(item, dict)
        ]

        for ref in contract.input_refs:
            if ref.ref_type not in {"paper_file", "manual_paper", "paper_upload"}:
                continue
            manual_papers.append(
                ReaderManualPaper(
                    title=str((ref.metadata or {}).get("title", "")).strip(),
                    abstract=str((ref.metadata or {}).get("abstract", "")).strip(),
                    source=str((ref.metadata or {}).get("source", "manual_upload")).strip(),
                    year=_coerce_year((ref.metadata or {}).get("year")),
                    url=str((ref.metadata or {}).get("url", "")).strip(),
                    file_status=str((ref.metadata or {}).get("file_status", "uploaded")).strip() or "uploaded",
                    file_path=ref.ref_path,
                )
            )

        return cls(
            research_direction=str(metadata.get("research_direction", "")).strip(),
            keywords=keywords,
            source_scope=source_scope,
            time_range=time_range,
            max_papers=_normalize_max_papers(metadata.get("max_papers", 5)),
            manual_papers=manual_papers,
        )

    def to_dict(self) -> dict[str, Any]:
        return {
            "research_direction": self.research_direction,
            "keywords": list(self.keywords),
            "source_scope": self.source_scope,
            "time_range": dict(self.time_range),
            "max_papers": self.max_papers,
            "manual_paper_count": len(self.manual_papers),
        }


def _normalize_max_papers(value: Any) -> int:
    try:
        parsed = int(value)
    except (TypeError, ValueError):
        parsed = 5
    if parsed <= 0:
        return 1
    if parsed > 20:
        return 20
    return parsed


def _coerce_year(value: Any) -> int:
    try:
        year = int(value)
    except (TypeError, ValueError):
        return 0
    if year < 1900 or year > 2100:
        return 0
    return year


def _slugify(value: str) -> str:
    cleaned = re.sub(r"[^a-zA-Z0-9]+", "-", value.strip().lower()).strip("-")
    return cleaned or "paper"


def _default_year(request: ReaderRequest) -> int:
    for key in ("end_year", "year", "to_year"):
        year = _coerce_year(request.time_range.get(key))
        if year:
            return year
    return 2026


def _scope_sequence(scope: str) -> list[str]:
    if scope == "arxiv":
        return ["arxiv"]
    if scope == "conference":
        return ["conference"]
    if scope == "journal":
        return ["journal"]
    return ["arxiv", "conference", "journal"]


def _candidate_template(title: str, abstract: str, source: str, year: int, url: str, file_status: str, file_path: str = "") -> dict[str, Any]:
    item = {
        "title": title.strip(),
        "abstract": abstract.strip(),
        "source": source.strip(),
        "year": year,
        "url": url.strip(),
        "file_status": file_status.strip() or "metadata_only",
    }
    if file_path.strip():
        item["file_path"] = file_path.strip()
    return item


def build_stable_reader_candidates(request: ReaderRequest) -> list[dict[str, Any]]:
    candidates: list[dict[str, Any]] = []

    for item in request.manual_papers:
        title = item.title or "Manual Uploaded Paper"
        source = item.source or "manual_upload"
        year = item.year or _default_year(request)
        file_status = item.file_status or ("uploaded" if item.file_path else "metadata_only")
        url = item.url or ""
        abstract = item.abstract or f"Manual paper normalized into Reader output for {title}."
        candidates.append(
            _candidate_template(
                title=title,
                abstract=abstract,
                source=source,
                year=year,
                url=url,
                file_status=file_status,
                file_path=item.file_path,
            )
        )

    remaining = max(request.max_papers - len(candidates), 0)
    if remaining == 0:
        return candidates[: request.max_papers]

    direction = request.research_direction or "General Research"
    keywords = request.keywords or ["foundation model"]
    year = _default_year(request)
    scopes = _scope_sequence(request.source_scope)
    for index in range(remaining):
        keyword = keywords[index % len(keywords)]
        source = scopes[index % len(scopes)]
        title = f"{direction}: {keyword.title()} Study {index + 1}"
        if source == "arxiv":
            url = f"https://arxiv.org/abs/2603.{index + 101:04d}"
        elif source == "conference":
            url = f"https://openreview.net/forum?id=reader-{_slugify(keyword)}-{index + 1}"
        else:
            url = f"https://doi.org/10.1000/reader.{_slugify(keyword)}.{index + 1}"
        abstract = (
            f"Controlled Reader mock candidate for {direction} with keyword '{keyword}'. "
            f"This placeholder preserves the stage3 contract and can be replaced by real retrieval later."
        )
        candidates.append(
            _candidate_template(
                title=title,
                abstract=abstract,
                source=source,
                year=year,
                url=url,
                file_status="metadata_only",
            )
        )
    return candidates[: request.max_papers]


def build_reader_prompt(contract: AgentRuntimeInput, request: ReaderRequest) -> str:
    return (
        "You are MRAG Reader Agent running in controlled mode.\n"
        "Do not inspect the workspace. Do not run shell commands. Do not browse.\n"
        "Return valid JSON only.\n"
        "The JSON must contain 'candidate_papers' as an array.\n"
        "Each candidate paper must include: title, abstract, source, year, url, file_status.\n"
        "Always include file_path as a string; use an empty string when unknown.\n"
        "Prefer the shortest valid answer that satisfies the schema and current task.\n"
        "Do not add markdown fences.\n\n"
        f"job_id: {contract.job_id}\n"
        f"research_direction: {request.research_direction}\n"
        f"keywords: {json.dumps(request.keywords, ensure_ascii=False)}\n"
        f"source_scope: {request.source_scope}\n"
        f"time_range: {json.dumps(request.time_range, ensure_ascii=False)}\n"
        f"max_papers: {request.max_papers}\n"
        f"manual_paper_count: {len(request.manual_papers)}\n"
    )


def build_reader_codex_schema(request: ReaderRequest) -> dict[str, Any]:
    item_schema = {
        "type": "object",
        "required": ["title", "abstract", "source", "year", "url", "file_status", "file_path"],
        "properties": {
            "title": {"type": "string"},
            "abstract": {"type": "string"},
            "source": {"type": "string"},
            "year": {"type": "integer"},
            "url": {"type": "string"},
            "file_status": {"type": "string"},
            "file_path": {"type": "string"},
        },
        "additionalProperties": False,
    }
    max_items = request.max_papers if request.max_papers > 0 else 1
    return {
        "type": "object",
        "required": ["candidate_papers"],
        "properties": {
            "candidate_papers": {
                "type": "array",
                "minItems": 1,
                "maxItems": max_items,
                "items": item_schema,
            }
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


def _normalize_candidate_item(value: Any, request: ReaderRequest) -> dict[str, Any] | None:
    if isinstance(value, str):
        return _candidate_template(
            title=value,
            abstract="",
            source=request.source_scope or "mixed",
            year=_default_year(request),
            url="",
            file_status="metadata_only",
        )
    if not isinstance(value, dict):
        return None

    def first(keys: list[str], default: Any = "") -> Any:
        for key in keys:
            if key in value and value[key] not in (None, ""):
                return value[key]
        return default

    file_path = str(first(["file_path", "filePath", "path", "local_path", "ref_path"], "")).strip()
    file_status = str(first(["file_status", "fileStatus", "status"], "")).strip()
    if not file_status:
        file_status = "uploaded" if file_path else "metadata_only"
    return _candidate_template(
        title=str(first(["title", "paper_title", "name"], "Untitled Candidate")).strip(),
        abstract=str(first(["abstract", "summary", "paper_abstract", "description"], "")).strip(),
        source=str(first(["source", "venue", "origin", "source_type"], request.source_scope or "mixed")).strip(),
        year=_coerce_year(first(["year", "published_year", "date_year"], _default_year(request))) or _default_year(request),
        url=str(first(["url", "paper_url", "source_url", "link"], "")).strip(),
        file_status=file_status,
        file_path=file_path,
    )


def extract_candidate_papers(payload: dict[str, Any], request: ReaderRequest) -> list[dict[str, Any]]:
    candidates: list[Any] = []
    if isinstance(payload.get("candidate_papers"), list):
        candidates = payload["candidate_papers"]
    elif isinstance(payload.get("items"), list):
        candidates = payload["items"]
    elif isinstance(payload.get("data"), dict):
        data = payload["data"]
        if isinstance(data.get("candidate_papers"), list):
            candidates = data["candidate_papers"]
        elif isinstance(data.get("papers"), list):
            candidates = data["papers"]
        elif isinstance(data.get("response_json"), dict):
            response_json = data["response_json"]
            if isinstance(response_json.get("candidate_papers"), list):
                candidates = response_json["candidate_papers"]
    if not candidates:
        for field_name in ("result", "response_json", "payload", "json", "object"):
            nested = payload.get(field_name)
            if isinstance(nested, dict) and isinstance(nested.get("candidate_papers"), list):
                candidates = nested["candidate_papers"]
                break
            if isinstance(nested, list):
                candidates = nested
                break
    if not candidates:
        response_json = payload.get("response_json")
        if isinstance(response_json, dict) and isinstance(response_json.get("candidate_papers"), list):
            candidates = response_json["candidate_papers"]
    if not candidates:
        response_text = payload.get("response_text", "")
        if isinstance(response_text, str):
            parsed = _extract_json_candidate(response_text)
            if isinstance(parsed, dict) and isinstance(parsed.get("candidate_papers"), list):
                candidates = parsed["candidate_papers"]
            elif isinstance(parsed, list):
                candidates = parsed
    normalized = [_normalize_candidate_item(item, request) for item in candidates]
    return [item for item in normalized if item]


def repair_reader_payload(contract: AgentRuntimeInput, payload: dict[str, Any]) -> list[dict[str, Any]]:
    request = ReaderRequest.from_contract(contract)
    candidate_papers = extract_candidate_papers(payload, request)
    if not candidate_papers:
        candidate_papers = build_stable_reader_candidates(request)
    return candidate_papers[: request.max_papers]


class ReaderMockExecutor(MockExecutor):
    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request = ReaderRequest.from_contract(contract)
        candidate_papers = build_stable_reader_candidates(request)
        normalized_payload = self.base_payload(contract, "mock")
        normalized_payload.update(
            {
                "summary": f"Reader mock produced {len(candidate_papers)} candidate paper(s).",
                "items": candidate_papers,
                "candidate_papers": candidate_papers,
                "data": {
                    "reader_request": request.to_dict(),
                    "candidate_count": len(candidate_papers),
                    "response_json": {
                        "candidate_papers": candidate_papers,
                        "mode": "reader_mock",
                    },
                },
                "metadata": {
                    "synthetic": True,
                    "agent_role": "reader",
                },
            }
        )
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[],
            warnings=["reader mock executor used; results are stable placeholders."],
            error_message="",
        )


class ReaderApiExecutor(ApiExecutor):
    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        output = super().normalize_output(contract, prepared_request, execution_result, collected_response)
        request = ReaderRequest.from_contract(contract)
        output.normalized_payload["candidate_papers"] = extract_candidate_papers(output.normalized_payload, request)
        output.normalized_payload["items"] = list(output.normalized_payload["candidate_papers"])
        output.normalized_payload["data"] = {
            **(output.normalized_payload.get("data") if isinstance(output.normalized_payload.get("data"), dict) else {}),
            "reader_request": request.to_dict(),
        }
        return output


class ReaderCodexCLIExecutor(CodexCLIExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        prepared = super().prepare_request(contract)
        request = ReaderRequest.from_contract(contract)
        prompt_text = build_reader_prompt(contract, request)
        prepared["prompt_text"] = prompt_text
        prompt_path = Path(prepared["prompt_path"])
        prompt_path.parent.mkdir(parents=True, exist_ok=True)
        prompt_path.write_text(prompt_text, encoding="utf-8")
        schema_path = prompt_path.parent / "output_schema.json"
        schema_path.write_text(json.dumps(build_reader_codex_schema(request), ensure_ascii=False, indent=2), encoding="utf-8")
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
        request = ReaderRequest.from_contract(contract)
        candidate_papers = extract_candidate_papers(output.normalized_payload, request)
        if candidate_papers:
            output.normalized_payload["candidate_papers"] = candidate_papers[: request.max_papers]
            output.normalized_payload["items"] = list(output.normalized_payload["candidate_papers"])
            output.normalized_payload["summary"] = (
                output.normalized_payload.get("summary", "").strip()
                or f"Reader codex_cli produced {len(output.normalized_payload['candidate_papers'])} candidate paper(s)."
            )
        output.normalized_payload["data"] = {
            **(output.normalized_payload.get("data") if isinstance(output.normalized_payload.get("data"), dict) else {}),
            "reader_request": request.to_dict(),
        }
        return output


class ReaderValidator(BaseValidator):
    def validate_payload(self, contract: AgentRuntimeInput, payload: dict[str, Any] | None) -> list[str]:
        errors = super().validate_payload(contract, payload)
        if payload is None:
            return errors
        candidate_papers = payload.get("candidate_papers")
        if not isinstance(candidate_papers, list):
            errors.append("normalized_payload.candidate_papers must be an array")
            return errors
        for index, item in enumerate(candidate_papers):
            if not isinstance(item, dict):
                errors.append(f"normalized_payload.candidate_papers[{index}] must be an object")
                continue
            for field_name in ("title", "abstract", "source", "url", "file_status"):
                value = item.get(field_name)
                if not isinstance(value, str):
                    errors.append(f"normalized_payload.candidate_papers[{index}].{field_name} must be a string")
                elif field_name in {"title", "source", "file_status"} and not value.strip():
                    errors.append(f"normalized_payload.candidate_papers[{index}].{field_name} cannot be empty")
            if not isinstance(item.get("year"), int):
                errors.append(f"normalized_payload.candidate_papers[{index}].year must be an integer")
        return errors


class ReaderRepairer(BaseRepairer):
    def repair(
        self,
        contract: AgentRuntimeInput,
        output: AgentRuntimeOutput,
        errors: list[str],
    ) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        repaired, actions = super().repair(contract, output, errors)
        candidate_papers = repair_reader_payload(contract, repaired.normalized_payload)
        repaired.normalized_payload["candidate_papers"] = candidate_papers
        repaired.normalized_payload["items"] = list(candidate_papers)
        data = repaired.normalized_payload.get("data")
        if not isinstance(data, dict):
            data = {}
        data["reader_request"] = ReaderRequest.from_contract(contract).to_dict()
        data["candidate_count"] = len(candidate_papers)
        repaired.normalized_payload["data"] = data
        if not isinstance(repaired.normalized_payload.get("summary"), str) or not repaired.normalized_payload["summary"].strip():
            repaired.normalized_payload["summary"] = f"Reader repaired {len(candidate_papers)} candidate paper(s)."
        actions.append(
            AgentRepairAction(
                action="repair_candidate_papers",
                status="applied",
                detail="Normalized Reader candidate_papers payload.",
                metadata={"candidate_count": len(candidate_papers)},
            )
        )
        return repaired, actions


class ReaderAgent(BaseAgent):
    pass


def build_reader_agent(contract: AgentRuntimeInput) -> ReaderAgent:
    if contract.execution_mode == "api":
        executor = ReaderApiExecutor()
    elif contract.execution_mode == "codex_cli":
        executor = ReaderCodexCLIExecutor()
    else:
        executor = ReaderMockExecutor()
    return ReaderAgent(executor=executor, validator=ReaderValidator(), repairer=ReaderRepairer())
