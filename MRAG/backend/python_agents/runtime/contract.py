from __future__ import annotations

from dataclasses import asdict, dataclass, field, fields
from typing import Any


@dataclass
class AgentInputRef:
    ref_type: str
    ref_id: str = ""
    ref_path: str = ""
    ref_version: str = ""
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass
class AgentArtifactManifestItem:
    artifact_type: str
    name: str
    file_path: str
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass
class AgentRepairAction:
    action: str
    status: str
    detail: str
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass
class AgentToolUsage:
    tool_ref: str
    status: str
    summary: str
    metadata: dict[str, Any] = field(default_factory=dict)


@dataclass
class AgentRuntimeInput:
    job_id: str
    agent_type: str
    execution_mode: str
    model_provider: str
    model_name: str
    prompt_version: str
    input_refs: list[AgentInputRef] = field(default_factory=list)
    output_schema_ref: str = ""
    skill_refs: list[str] = field(default_factory=list)
    tool_refs: list[str] = field(default_factory=list)
    memory_refs: list[str] = field(default_factory=list)
    workspace_dir: str = ""
    metadata: dict[str, Any] = field(default_factory=dict)

    @classmethod
    def from_dict(cls, payload: dict[str, Any]) -> "AgentRuntimeInput":
        allowed_ref_fields = {item.name for item in fields(AgentInputRef)}
        refs = [
            AgentInputRef(**{key: value for key, value in item.items() if key in allowed_ref_fields})
            for item in payload.get("input_refs", [])
            if isinstance(item, dict)
        ]
        return cls(
            job_id=str(payload.get("job_id", "")).strip(),
            agent_type=str(payload.get("agent_type", "")).strip(),
            execution_mode=str(payload.get("execution_mode", "")).strip(),
            model_provider=str(payload.get("model_provider", "")).strip(),
            model_name=str(payload.get("model_name", "")).strip(),
            prompt_version=str(payload.get("prompt_version", "")).strip(),
            input_refs=refs,
            output_schema_ref=str(payload.get("output_schema_ref", "")).strip(),
            skill_refs=[str(item).strip() for item in payload.get("skill_refs", []) if str(item).strip()],
            tool_refs=[str(item).strip() for item in payload.get("tool_refs", []) if str(item).strip()],
            memory_refs=[str(item).strip() for item in payload.get("memory_refs", []) if str(item).strip()],
            workspace_dir=str(payload.get("workspace_dir", "")).strip(),
            metadata=dict(payload.get("metadata", {}) or {}),
        )

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class AgentRuntimeOutput:
    status: str
    normalized_payload: dict[str, Any] = field(default_factory=dict)
    artifact_manifest: list[AgentArtifactManifestItem] = field(default_factory=list)
    repair_actions: list[AgentRepairAction] = field(default_factory=list)
    tool_usages: list[AgentToolUsage] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    validation_status: str = "pending"
    repair_status: str = "pending"
    validation_errors: list[str] = field(default_factory=list)
    error_message: str = ""

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)
