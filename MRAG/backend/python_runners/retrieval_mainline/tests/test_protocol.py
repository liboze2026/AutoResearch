from __future__ import annotations

import tempfile
import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from protocol import (
    PROTOCOL_VERSION,
    DatasetAdapterContract,
    ExperimentManifest,
    MetricsPayload,
    RunConfig,
    dump_json,
    load_config,
    load_manifest,
    load_metrics,
)


class ProtocolTest(unittest.TestCase):
    def test_manifest_roundtrip(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            manifest_path = root / "manifest.json"
            config_path = root / "config.json"
            manifest = ExperimentManifest(
                protocol_version=PROTOCOL_VERSION,
                run_id="p4run_demo",
                dataset_profile_id="p4ds_demo",
                idea_id="p4idea_demo",
                reader_context_id="p4ctx_demo",
                run_dir=str(root / "run"),
                snapshot_dir=str(root / "run" / "snapshot"),
                artifact_dir=str(root / "artifacts"),
                logs_dir=str(root / "run" / "logs"),
                config_path=str(config_path),
                metrics_path=str(root / "artifacts" / "metrics.json"),
                predictions_path=str(root / "artifacts" / "predictions.json"),
                machine_report_path=str(root / "artifacts" / "machine_report.json"),
                human_report_path=str(root / "artifacts" / "report.md"),
                dataset_tool_asset_path=str(root / "artifacts" / "dataset_tool.json"),
                dataset_adapter_path=str(root / "artifacts" / "dataset_adapter.json"),
                evaluate_tool_asset_path=str(root / "artifacts" / "evaluate_tool.json"),
                eval_summary_path=str(root / "artifacts" / "eval_summary.md"),
                bootstrap_script_path=str(root / "run" / "snapshot" / "bootstrap_env.sh"),
            )
            config = RunConfig(
                protocol_version=PROTOCOL_VERSION,
                method_name="dummy_retrieval",
                method_module_path="",
                runner_mode="local_dummy",
                method_branch="method/dummy_retrieval",
                dataset_adapter=DatasetAdapterContract(
                    dataset_profile_id="p4ds_demo",
                    dataset_name="VisDoM",
                    task_type="multimodal_retrieval",
                    server_path="/data/visdom",
                    official_metric="recall@5",
                ),
            )
            dump_json(manifest_path, manifest.to_payload())
            dump_json(config_path, config.to_payload())
            self.assertEqual(load_manifest(manifest_path).run_id, "p4run_demo")
            self.assertEqual(load_config(config_path).dataset_adapter.dataset_name, "VisDoM")

    def test_metrics_roundtrip(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            metrics_path = Path(temp_dir) / "metrics.json"
            metrics = MetricsPayload(
                protocol_version=PROTOCOL_VERSION,
                run_id="p4run_demo",
                primary_metric="recall@5",
                values={"recall@5": 0.77},
            )
            dump_json(metrics_path, metrics.to_payload())
            self.assertEqual(load_metrics(metrics_path).values["recall@5"], 0.77)


if __name__ == "__main__":
    unittest.main()
