from __future__ import annotations

from pathlib import Path
from typing import Any

from method_registry import load_method
from protocol import dump_json, load_config, load_json, load_manifest
from tools.dataset_tool import ensure_layout, materialize_dataset_assets


def run_mainline(manifest_path: str) -> dict[str, Any]:
    manifest = load_manifest(manifest_path)
    config = load_config(manifest.config_path)
    ensure_layout(manifest)
    dataset_asset = materialize_dataset_assets(manifest, config)
    method = load_method(config.method_module_path)
    predictions = method.run(manifest, config, dataset_asset)
    dump_json(manifest.predictions_path, predictions)
    summary = {
        "run_id": manifest.run_id,
        "method_name": predictions.get("method_name", config.method_name),
        "prediction_count": len(predictions.get("predictions", [])),
        "dataset_profile_id": manifest.dataset_profile_id,
        "adapter_type": dataset_asset.get("adapter_type", ""),
        "page_count": dataset_asset.get("sample_statistics", {}).get("page_count", 0),
        "query_count": dataset_asset.get("sample_statistics", {}).get("query_count", 0),
    }
    dump_json(Path(manifest.artifact_dir) / "run_summary.json", summary)
    Path(manifest.logs_dir, "run.log").write_text(
        (
            "[retrieval_mainline] run started\n"
            f"[retrieval_mainline] method={summary['method_name']}\n"
            f"[retrieval_mainline] adapter={summary['adapter_type']}\n"
            f"[retrieval_mainline] page_count={summary['page_count']}\n"
            f"[retrieval_mainline] query_count={summary['query_count']}\n"
            f"[retrieval_mainline] prediction_count={summary['prediction_count']}\n"
        ),
        encoding="utf-8",
    )
    return {"manifest": manifest.to_payload(), "config": config.to_payload(), "predictions": predictions, "summary": summary}


def load_predictions(path: str) -> dict[str, Any]:
    return load_json(path)
