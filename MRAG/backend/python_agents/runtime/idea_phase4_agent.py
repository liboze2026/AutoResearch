from __future__ import annotations

import json
from pathlib import Path
from typing import Any

try:
    from .base import BaseAgent, BaseExecutor, BaseRepairer, BaseValidator
    from .contract import AgentArtifactManifestItem, AgentRepairAction, AgentRuntimeInput, AgentRuntimeOutput
    from .idea_phase4_logic import (
        Phase4IdeaRequest,
        build_top_recommendations,
        generate_structured_ideas,
        score_and_rank_ideas,
    )
except ImportError:  # pragma: no cover - supports direct script execution
    from base import BaseAgent, BaseExecutor, BaseRepairer, BaseValidator
    from contract import AgentArtifactManifestItem, AgentRepairAction, AgentRuntimeInput, AgentRuntimeOutput
    from idea_phase4_logic import (
        Phase4IdeaRequest,
        build_top_recommendations,
        generate_structured_ideas,
        score_and_rank_ideas,
    )


IDEA_PHASE4_SCHEMA_REF = "schemas/idea-phase4-output-v1.json"
IDEA_REQUIRED_FIELDS = (
    "title",
    "problem_definition",
    "core_method",
    "differentiators",
    "data_processing_needs",
    "model_changes",
    "training_plan",
    "evaluation_metrics",
    "risk_points",
    "expected_gains",
    "score",
    "score_summary",
    "status",
    "source_type",
)


class IdeaPhase4Executor(BaseExecutor):
    name = "idea_phase4"

    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        request = Phase4IdeaRequest.from_metadata(contract.metadata)
        workspace_dir = Path(contract.workspace_dir or Path.cwd() / "workspace" / "agents" / "jobs" / contract.job_id)
        workspace_dir.mkdir(parents=True, exist_ok=True)
        return {
            "request": request,
            "workspace_dir": workspace_dir,
            "request_path": workspace_dir / "idea_phase4_request.json",
            "ideas_path": workspace_dir / "idea_phase4_ideas.json",
            "top_path": workspace_dir / "idea_phase4_top3.json",
        }

    def execute(self, prepared_request: dict[str, Any], contract: AgentRuntimeInput) -> dict[str, Any]:
        request: Phase4IdeaRequest = prepared_request["request"]
        ideas = score_and_rank_ideas(generate_structured_ideas(request), request)
        top_recommendations = build_top_recommendations(ideas)
        _write_json(prepared_request["request_path"], contract.metadata)
        _write_json(prepared_request["ideas_path"], [item.to_payload() for item in ideas])
        _write_json(prepared_request["top_path"], top_recommendations)
        return {
            "request": request,
            "ideas": ideas,
            "top_recommendations": top_recommendations,
        }

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        request: Phase4IdeaRequest = execution_result["request"]
        ideas_payload = [item.to_payload() for item in execution_result["ideas"]]
        top_recommendations = list(execution_result["top_recommendations"])
        normalized_payload = self.base_payload(contract, contract.execution_mode)
        normalized_payload.update(
            {
                "summary": f"Generated {len(ideas_payload)} structured phase4 ideas for {request.dataset_profile.dataset_name}.",
                "items": ideas_payload,
                "ideas": ideas_payload,
                "top_recommendations": top_recommendations,
                "generation_mode": request.generation_mode,
                "data": {
                    "dataset_profile_id": request.dataset_profile.id,
                    "reader_context_title": request.reader_context.task_definition,
                    "idea_count": len(ideas_payload),
                    "target_count": request.effective_target_count(),
                    "source_idea_id": request.source_idea_id,
                    "last_failure_run_id": request.last_failure_run_id,
                },
                "metadata": {
                    "dataset_name": request.dataset_profile.dataset_name,
                    "official_metric": request.dataset_profile.official_metric,
                    "generation_mode": request.generation_mode,
                    "failure_feedback": dict(request.failure_feedback),
                },
            }
        )
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload=normalized_payload,
            artifact_manifest=[
                AgentArtifactManifestItem("idea_phase4_request", prepared_request["request_path"].name, str(prepared_request["request_path"]), {"role": "idea_phase4_request"}),
                AgentArtifactManifestItem("idea_phase4_ideas", prepared_request["ideas_path"].name, str(prepared_request["ideas_path"]), {"role": "idea_phase4_ideas"}),
                AgentArtifactManifestItem("idea_phase4_top3", prepared_request["top_path"].name, str(prepared_request["top_path"]), {"role": "idea_phase4_top3"}),
            ],
            repair_actions=[],
            tool_usages=[],
            warnings=[],
            error_message="",
        )


class IdeaPhase4Validator(BaseValidator):
    def validate_input(self, contract: AgentRuntimeInput) -> list[str]:
        errors = super().validate_input(contract)
        request = Phase4IdeaRequest.from_metadata(contract.metadata)
        if not request.dataset_profile.dataset_name:
            errors.append("dataset_profile.datasetName is required")
        if not request.reader_context.task_definition:
            errors.append("reader_context.task_definition is required")
        if request.generation_mode == "revision" and not request.source_idea_id and not request.source_idea.title:
            errors.append("source_idea_id or source_idea.title is required in revision mode")
        return errors

    def validate_payload(self, contract: AgentRuntimeInput, payload: dict[str, Any] | None) -> list[str]:
        errors = super().validate_payload(contract, payload)
        if payload is None:
            return errors
        ideas = payload.get("ideas")
        if not isinstance(ideas, list) or not ideas:
            errors.append("normalized_payload.ideas must be a non-empty array")
        else:
            for index, item in enumerate(ideas):
                if not isinstance(item, dict):
                    errors.append(f"normalized_payload.ideas[{index}] must be an object")
                    continue
                for field_name in IDEA_REQUIRED_FIELDS:
                    if field_name not in item:
                        errors.append(f"normalized_payload.ideas[{index}].{field_name} is required")
                for field_name in ("title", "problem_definition", "core_method", "differentiators", "training_plan", "status", "source_type"):
                    if not isinstance(item.get(field_name), str) or not str(item.get(field_name)).strip():
                        errors.append(f"normalized_payload.ideas[{index}].{field_name} must be a non-empty string")
                for field_name in ("data_processing_needs", "model_changes", "evaluation_metrics", "risk_points", "expected_gains"):
                    if not isinstance(item.get(field_name), list):
                        errors.append(f"normalized_payload.ideas[{index}].{field_name} must be an array")
                if not isinstance(item.get("score"), dict):
                    errors.append(f"normalized_payload.ideas[{index}].score must be an object")
                if not isinstance(item.get("score_summary"), dict):
                    errors.append(f"normalized_payload.ideas[{index}].score_summary must be an object")
        top = payload.get("top_recommendations")
        if not isinstance(top, list):
            errors.append("normalized_payload.top_recommendations must be an array")
        else:
            for index, item in enumerate(top):
                if not isinstance(item, dict):
                    errors.append(f"normalized_payload.top_recommendations[{index}] must be an object")
                    continue
                for field_name in ("title", "overallScore", "rank", "recommendationReason", "score"):
                    if field_name not in item:
                        errors.append(f"normalized_payload.top_recommendations[{index}].{field_name} is required")
        if not isinstance(payload.get("generation_mode"), str) or not payload.get("generation_mode", "").strip():
            errors.append("normalized_payload.generation_mode must be a non-empty string")
        return errors


class IdeaPhase4Repairer(BaseRepairer):
    def repair(
        self,
        contract: AgentRuntimeInput,
        output: AgentRuntimeOutput,
        errors: list[str],
    ) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        repaired, actions = super().repair(contract, output, errors)
        request = Phase4IdeaRequest.from_metadata(contract.metadata)
        ideas = score_and_rank_ideas(generate_structured_ideas(request), request)
        ideas_payload = [item.to_payload() for item in ideas]
        repaired.normalized_payload["summary"] = f"Generated {len(ideas_payload)} structured phase4 ideas for {request.dataset_profile.dataset_name}."
        repaired.normalized_payload["items"] = ideas_payload
        repaired.normalized_payload["ideas"] = ideas_payload
        repaired.normalized_payload["top_recommendations"] = build_top_recommendations(ideas)
        repaired.normalized_payload["generation_mode"] = request.generation_mode
        data = repaired.normalized_payload.get("data")
        if not isinstance(data, dict):
            data = {}
        data.update(
            {
                "dataset_profile_id": request.dataset_profile.id,
                "idea_count": len(ideas_payload),
                "target_count": request.effective_target_count(),
                "source_idea_id": request.source_idea_id,
                "last_failure_run_id": request.last_failure_run_id,
            }
        )
        repaired.normalized_payload["data"] = data
        metadata = repaired.normalized_payload.get("metadata")
        if not isinstance(metadata, dict):
            metadata = {}
        metadata.update(
            {
                "dataset_name": request.dataset_profile.dataset_name,
                "generation_mode": request.generation_mode,
                "failure_feedback": dict(request.failure_feedback),
            }
        )
        repaired.normalized_payload["metadata"] = metadata
        actions.append(
            AgentRepairAction(
                action="repair_phase4_ideas",
                status="applied",
                detail="Rebuilt structured phase4 ideas, scores, and recommendations from request metadata.",
                metadata={"idea_count": len(ideas_payload)},
            )
        )
        return repaired, actions


class IdeaPhase4Agent(BaseAgent):
    pass


def build_idea_phase4_agent(contract: AgentRuntimeInput) -> IdeaPhase4Agent:
    return IdeaPhase4Agent(
        executor=IdeaPhase4Executor(),
        validator=IdeaPhase4Validator(),
        repairer=IdeaPhase4Repairer(),
    )


def _write_json(path: Path, payload: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
