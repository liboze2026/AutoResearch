from __future__ import annotations

import json
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Any


PROTOCOL_VERSION = "phase4-retrieval-mainline-v1"


@dataclass
class DatasetAdapterContract:
    dataset_profile_id: str
    dataset_name: str
    task_type: str
    server_path: str
    official_metric: str
    splits: list[dict[str, Any]] = field(default_factory=list)
    metadata: dict[str, Any] = field(default_factory=dict)

    def validate(self) -> None:
        if not self.dataset_profile_id.strip():
            raise ValueError("dataset_adapter.dataset_profile_id is required")
        if not self.dataset_name.strip():
            raise ValueError("dataset_adapter.dataset_name is required")
        if not self.task_type.strip():
            raise ValueError("dataset_adapter.task_type is required")


@dataclass
class RunConfig:
    protocol_version: str
    method_name: str
    method_module_path: str
    runner_mode: str
    method_branch: str
    parameters: dict[str, Any] = field(default_factory=dict)
    retry_policy: dict[str, Any] = field(default_factory=dict)
    dataset_adapter: DatasetAdapterContract | None = None
    evaluate: dict[str, Any] = field(default_factory=dict)
    notes: list[str] = field(default_factory=list)

    def validate(self) -> None:
        if self.protocol_version.strip() != PROTOCOL_VERSION:
            raise ValueError("config.protocol_version is invalid")
        if not self.method_name.strip():
            raise ValueError("config.method_name is required")
        if not self.runner_mode.strip():
            raise ValueError("config.runner_mode is required")
        if self.dataset_adapter is None:
            raise ValueError("config.dataset_adapter is required")
        self.dataset_adapter.validate()

    def to_payload(self) -> dict[str, Any]:
        payload = asdict(self)
        if self.dataset_adapter is not None:
            payload["dataset_adapter"] = asdict(self.dataset_adapter)
        return payload


@dataclass
class ExperimentManifest:
    protocol_version: str
    run_id: str
    dataset_profile_id: str
    idea_id: str
    reader_context_id: str
    run_dir: str
    snapshot_dir: str
    artifact_dir: str
    logs_dir: str
    config_path: str
    metrics_path: str
    predictions_path: str
    machine_report_path: str
    human_report_path: str
    dataset_tool_asset_path: str
    dataset_adapter_path: str
    evaluate_tool_asset_path: str
    eval_summary_path: str
    bootstrap_script_path: str
    metadata: dict[str, Any] = field(default_factory=dict)

    def validate(self) -> None:
        if self.protocol_version.strip() != PROTOCOL_VERSION:
            raise ValueError("manifest.protocol_version is invalid")
        for field_name in (
            "run_id",
            "dataset_profile_id",
            "idea_id",
            "run_dir",
            "snapshot_dir",
            "artifact_dir",
            "logs_dir",
            "config_path",
            "metrics_path",
            "predictions_path",
            "machine_report_path",
            "human_report_path",
            "dataset_tool_asset_path",
            "dataset_adapter_path",
            "evaluate_tool_asset_path",
            "eval_summary_path",
            "bootstrap_script_path",
        ):
            if not str(getattr(self, field_name, "")).strip():
                raise ValueError(f"manifest.{field_name} is required")

    def to_payload(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class MetricsPayload:
    protocol_version: str
    run_id: str
    primary_metric: str
    values: dict[str, Any] = field(default_factory=dict)
    status: str = "succeeded"
    retrieval_summary: dict[str, Any] = field(default_factory=dict)
    metadata: dict[str, Any] = field(default_factory=dict)

    def validate(self) -> None:
        if self.protocol_version.strip() != PROTOCOL_VERSION:
            raise ValueError("metrics.protocol_version is invalid")
        if not self.run_id.strip():
            raise ValueError("metrics.run_id is required")
        if not self.primary_metric.strip():
            raise ValueError("metrics.primary_metric is required")
        if not isinstance(self.values, dict):
            raise ValueError("metrics.values must be an object")

    def to_payload(self) -> dict[str, Any]:
        return asdict(self)


def load_json(path: str | Path) -> dict[str, Any]:
    return json.loads(Path(path).read_text(encoding="utf-8"))


def dump_json(path: str | Path, payload: Any) -> None:
    output_path = Path(path)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def load_manifest(path: str | Path) -> ExperimentManifest:
    payload = load_json(path)
    manifest = ExperimentManifest(**payload)
    manifest.validate()
    return manifest


def load_config(path: str | Path) -> RunConfig:
    payload = load_json(path)
    dataset_adapter = DatasetAdapterContract(**dict(payload.get("dataset_adapter") or {}))
    payload["dataset_adapter"] = dataset_adapter
    config = RunConfig(**payload)
    config.validate()
    return config


def load_metrics(path: str | Path) -> MetricsPayload:
    payload = load_json(path)
    metrics = MetricsPayload(**payload)
    metrics.validate()
    return metrics
