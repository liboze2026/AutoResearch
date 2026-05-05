from __future__ import annotations

from pathlib import Path
from typing import Any

from protocol import ExperimentManifest, RunConfig, dump_json
from tools.page_retrieval_dataset import build_page_retrieval_adapter


def materialize_dataset_assets(manifest: ExperimentManifest, config: RunConfig) -> dict[str, Any]:
    adapter = build_page_retrieval_adapter(config.dataset_adapter)
    payload = adapter.analyze()
    payload["dataset_profile_id"] = config.dataset_adapter.dataset_profile_id
    payload["dataset_name"] = config.dataset_adapter.dataset_name
    payload["task_type"] = config.dataset_adapter.task_type
    payload["official_metric"] = config.dataset_adapter.official_metric
    payload["run_id"] = manifest.run_id
    payload["contract"] = {
        "dataset_profile_id": config.dataset_adapter.dataset_profile_id,
        "dataset_name": config.dataset_adapter.dataset_name,
        "task_type": config.dataset_adapter.task_type,
        "server_path": config.dataset_adapter.server_path,
        "official_metric": config.dataset_adapter.official_metric,
        "splits": list(config.dataset_adapter.splits),
        "metadata": dict(config.dataset_adapter.metadata),
    }
    dump_json(manifest.dataset_tool_asset_path, payload)
    dump_json(
        manifest.dataset_adapter_path,
        {
            "contract_version": "dataset-adapter-v2",
            "adapter_type": payload["adapter_type"],
            "retrieval_granularity": payload["retrieval_granularity"],
            "dataset_adapter": payload["contract"],
            "adapter_asset": {
                "resolved_root": payload["resolved_root"],
                "paths": payload["paths"],
                "text_fields": payload["text_fields"],
                "query_fields": payload["query_fields"],
                "sample_statistics": payload["sample_statistics"],
            },
            "adapter_entrypoint": "tools.dataset_tool:materialize_dataset_assets",
        },
    )
    return payload


def ensure_layout(manifest: ExperimentManifest) -> None:
    for path in (
        manifest.run_dir,
        manifest.snapshot_dir,
        manifest.artifact_dir,
        manifest.logs_dir,
        str(Path(manifest.metrics_path).parent),
        str(Path(manifest.machine_report_path).parent),
        str(Path(manifest.human_report_path).parent),
    ):
        Path(path).mkdir(parents=True, exist_ok=True)
