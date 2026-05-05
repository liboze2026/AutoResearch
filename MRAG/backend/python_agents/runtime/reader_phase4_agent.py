from __future__ import annotations

import json
from pathlib import Path
from typing import Any

try:
    from .base import BaseAgent, BaseExecutor, BaseRepairer, BaseValidator
    from .contract import AgentArtifactManifestItem, AgentRepairAction, AgentRuntimeInput, AgentRuntimeOutput, AgentToolUsage
    from .reader_phase4_sources import (
        DEFAULT_FIXTURE_PATH,
        HTTPClient,
        Phase4ReaderRequest,
        build_reader_context,
        build_reading_summary,
        retrieve_research_sources,
        source_records_from_payload,
    )
except ImportError:  # pragma: no cover - supports direct script execution
    from base import BaseAgent, BaseExecutor, BaseRepairer, BaseValidator
    from contract import AgentArtifactManifestItem, AgentRepairAction, AgentRuntimeInput, AgentRuntimeOutput, AgentToolUsage
    from reader_phase4_sources import (
        DEFAULT_FIXTURE_PATH,
        HTTPClient,
        Phase4ReaderRequest,
        build_reader_context,
        build_reading_summary,
        retrieve_research_sources,
        source_records_from_payload,
    )


READER_PHASE4_SCHEMA_REF = "schemas/reader-phase4-output-v1.json"
READER_PHASE4_CONTEXT_FIELDS = (
    "task_definition",
    "dataset_specific_challenges",
    "relevant_methods_landscape",
    "likely_strong_baselines",
    "common_failure_points",
    "evaluation_caveats",
    "implementation_constraints",
    "promising_research_directions",
    "citation_metadata",
)


class ReaderPhase4Executor(BaseExecutor):
    name = "reader_phase4"

    def __init__(self, http_client: HTTPClient | None = None) -> None:
        self.http_client = http_client or HTTPClient()

    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        request = Phase4ReaderRequest.from_contract_metadata(contract.metadata, contract.execution_mode)
        workspace_dir = Path(contract.workspace_dir or Path.cwd() / "workspace" / "agents" / "jobs" / contract.job_id)
        workspace_dir.mkdir(parents=True, exist_ok=True)
        return {
            "request": request,
            "workspace_dir": workspace_dir,
            "queries_path": workspace_dir / "reader_phase4_queries.json",
            "sources_path": workspace_dir / "reader_phase4_sources.json",
            "context_path": workspace_dir / "reader_phase4_context.json",
        }

    def execute(self, prepared_request: dict[str, Any], contract: AgentRuntimeInput) -> dict[str, Any]:
        request: Phase4ReaderRequest = prepared_request["request"]
        sources, provider_statuses, search_queries, warnings, used_fixture = retrieve_research_sources(
            request,
            http_client=self.http_client,
            fixture_path=DEFAULT_FIXTURE_PATH,
        )
        reader_context = build_reader_context(request.dataset_profile, sources, request.user_notes)
        reading_summary = build_reading_summary(request.dataset_profile, sources, request.user_notes)
        _write_json(prepared_request["queries_path"], {"queries": search_queries, "request": request.to_payload()})
        _write_json(prepared_request["sources_path"], [item.to_payload() for item in sources])
        _write_json(prepared_request["context_path"], reader_context)
        return {
            "request": request,
            "sources": sources,
            "provider_statuses": provider_statuses,
            "queries": search_queries,
            "warnings": warnings,
            "used_fixture": used_fixture,
            "reading_summary": reading_summary,
            "reader_context": reader_context,
        }

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request: Phase4ReaderRequest = execution_result["request"]
        sources_payload = [item.to_payload() for item in execution_result["sources"]]
        reader_context = dict(execution_result["reader_context"])
        used_mode = "mock" if execution_result["used_fixture"] or request.resolved_search_mode() == "fixture" else "api"
        normalized_payload = self.base_payload(contract, used_mode)
        normalized_payload.update(
            {
                "summary": execution_result["reading_summary"],
                "reading_summary": execution_result["reading_summary"],
                "items": sources_payload,
                "sources": sources_payload,
                "reader_context": reader_context,
                "citation_metadata": list(reader_context.get("citation_metadata", [])),
                "data": {
                    "reader_request": request.to_payload(),
                    "search_queries": list(execution_result["queries"]),
                    "provider_statuses": list(execution_result["provider_statuses"]),
                    "source_count": len(sources_payload),
                },
                "metadata": {
                    "dataset_profile_id": request.dataset_profile.id,
                    "dataset_name": request.dataset_profile.dataset_name,
                    "search_mode_requested": request.search_mode,
                    "search_mode_used": "fixture" if execution_result["used_fixture"] else request.resolved_search_mode(),
                    "provider_statuses": list(execution_result["provider_statuses"]),
                    "used_fixture": execution_result["used_fixture"],
                },
            }
        )
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=[
                AgentArtifactManifestItem("reader_queries", prepared_request["queries_path"].name, str(prepared_request["queries_path"]), {"role": "reader_queries"}),
                AgentArtifactManifestItem("reader_sources", prepared_request["sources_path"].name, str(prepared_request["sources_path"]), {"role": "reader_sources"}),
                AgentArtifactManifestItem("reader_context", prepared_request["context_path"].name, str(prepared_request["context_path"]), {"role": "reader_context"}),
            ],
            repair_actions=[],
            tool_usages=[
                AgentToolUsage(
                    tool_ref=str(item.get("provider", "reader_provider")),
                    status=str(item.get("status", "unknown")),
                    summary=f"provider returned {int(item.get('result_count', 0) or 0)} result(s)",
                    metadata={"errors": list(item.get("errors", []))},
                )
                for item in execution_result["provider_statuses"]
            ],
            warnings=list(execution_result["warnings"]),
            error_message="",
        )


class ReaderPhase4Validator(BaseValidator):
    def validate_payload(self, contract: AgentRuntimeInput, payload: dict[str, Any] | None) -> list[str]:
        errors = super().validate_payload(contract, payload)
        if payload is None:
            return errors
        sources = payload.get("sources")
        if not isinstance(sources, list) or not sources:
            errors.append("normalized_payload.sources must be a non-empty array")
        else:
            for index, item in enumerate(sources):
                if not isinstance(item, dict):
                    errors.append(f"normalized_payload.sources[{index}] must be an object")
                    continue
                for field_name in ("title", "abstract", "venue", "source_type", "source_url", "quality_tier"):
                    if field_name not in item or not isinstance(item.get(field_name), str):
                        errors.append(f"normalized_payload.sources[{index}].{field_name} must be a string")
                if not isinstance(item.get("authors"), list):
                    errors.append(f"normalized_payload.sources[{index}].authors must be an array")
                if not isinstance(item.get("publication_year"), int):
                    errors.append(f"normalized_payload.sources[{index}].publication_year must be an integer")
                if not isinstance(item.get("metadata"), dict):
                    errors.append(f"normalized_payload.sources[{index}].metadata must be an object")
        context = payload.get("reader_context")
        if not isinstance(context, dict):
            errors.append("normalized_payload.reader_context must be an object")
            return errors
        for field_name in READER_PHASE4_CONTEXT_FIELDS:
            if field_name not in context:
                errors.append(f"normalized_payload.reader_context.{field_name} is required")
                continue
            value = context.get(field_name)
            if field_name == "task_definition" and not isinstance(value, str):
                errors.append("normalized_payload.reader_context.task_definition must be a string")
            if field_name != "task_definition" and field_name != "citation_metadata" and not isinstance(value, list):
                errors.append(f"normalized_payload.reader_context.{field_name} must be an array")
            if field_name == "citation_metadata" and not isinstance(value, list):
                errors.append("normalized_payload.reader_context.citation_metadata must be an array")
        if not isinstance(payload.get("reading_summary"), str):
            errors.append("normalized_payload.reading_summary must be a string")
        if not isinstance(payload.get("citation_metadata"), list):
            errors.append("normalized_payload.citation_metadata must be an array")
        return errors


class ReaderPhase4Repairer(BaseRepairer):
    def repair(
        self,
        contract: AgentRuntimeInput,
        output: AgentRuntimeOutput,
        errors: list[str],
    ) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        repaired, actions = super().repair(contract, output, errors)
        request = Phase4ReaderRequest.from_contract_metadata(contract.metadata, contract.execution_mode)
        sources = source_records_from_payload(repaired.normalized_payload.get("sources") or repaired.normalized_payload.get("items"))
        if not sources:
            sources, provider_statuses, queries, warnings, used_fixture = retrieve_research_sources(
                request,
                live_providers=[],
                fixture_path=DEFAULT_FIXTURE_PATH,
            )
            repaired.normalized_payload.setdefault("metadata", {})
            repaired.normalized_payload["metadata"]["repair_fixture_used"] = used_fixture
            repaired.normalized_payload.setdefault("data", {})
            repaired.normalized_payload["data"]["provider_statuses"] = provider_statuses
            repaired.normalized_payload["data"]["search_queries"] = queries
            repaired.warnings.extend(warnings)
            actions.append(
                AgentRepairAction(
                    action="repair_reader_sources",
                    status="applied",
                    detail="Reader phase4 sources were rebuilt from fixture-backed providers.",
                    metadata={"source_count": len(sources)},
                )
            )
        context = repaired.normalized_payload.get("reader_context")
        if not isinstance(context, dict) or any(field_name not in context for field_name in READER_PHASE4_CONTEXT_FIELDS):
            context = build_reader_context(request.dataset_profile, sources, request.user_notes)
            actions.append(
                AgentRepairAction(
                    action="repair_reader_context",
                    status="applied",
                    detail="Reader phase4 structured context was rebuilt.",
                    metadata={"dataset_name": request.dataset_profile.dataset_name},
                )
            )
        reading_summary = build_reading_summary(request.dataset_profile, sources, request.user_notes)
        sources_payload = [item.to_payload() for item in sources]
        repaired.normalized_payload["summary"] = reading_summary
        repaired.normalized_payload["reading_summary"] = reading_summary
        repaired.normalized_payload["items"] = sources_payload
        repaired.normalized_payload["sources"] = sources_payload
        repaired.normalized_payload["reader_context"] = context
        repaired.normalized_payload["citation_metadata"] = list(context.get("citation_metadata", []))
        repaired.normalized_payload.setdefault("data", {})
        repaired.normalized_payload["data"]["source_count"] = len(sources_payload)
        return repaired, actions


class ReaderPhase4Agent(BaseAgent):
    pass


def build_reader_phase4_agent(contract: AgentRuntimeInput, http_client: HTTPClient | None = None) -> ReaderPhase4Agent:
    return ReaderPhase4Agent(
        executor=ReaderPhase4Executor(http_client=http_client),
        validator=ReaderPhase4Validator(),
        repairer=ReaderPhase4Repairer(),
    )


def _write_json(path: Path, payload: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
