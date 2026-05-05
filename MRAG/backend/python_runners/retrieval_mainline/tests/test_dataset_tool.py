from __future__ import annotations

import tempfile
import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from protocol import PROTOCOL_VERSION, DatasetAdapterContract, ExperimentManifest, RunConfig, load_json
from tools.dataset_tool import materialize_dataset_assets
from tools.page_retrieval_dataset import (
    DEFAULT_VISDOM_FIXTURE,
    VisDoMPageRetrievalAdapter,
    build_page_retrieval_adapter,
)


class DatasetToolTest(unittest.TestCase):
    def test_visdom_adapter_analyzes_fixture(self) -> None:
        contract = DatasetAdapterContract(
            dataset_profile_id="p4ds_visdom",
            dataset_name="VisDoM",
            task_type="visual_document_retrieval",
            server_path=str(DEFAULT_VISDOM_FIXTURE),
            official_metric="recall@5",
            metadata={"fixture_path": str(DEFAULT_VISDOM_FIXTURE)},
        )
        adapter = build_page_retrieval_adapter(contract)
        self.assertIsInstance(adapter, VisDoMPageRetrievalAdapter)
        payload = adapter.analyze()
        self.assertEqual(payload["adapter_type"], "visdom_page_retrieval")
        self.assertEqual(payload["retrieval_granularity"], "page")
        self.assertEqual(payload["sample_statistics"]["page_count"], 5)
        self.assertEqual(payload["sample_statistics"]["query_count"], 5)
        self.assertEqual(payload["sample_statistics"]["qrel_count"], 5)
        self.assertEqual(payload["sample_statistics"]["unique_relevant_page_count"], 5)
        self.assertTrue(payload["file_structure_snapshot"]["exists"])
        self.assertIn("metadata/pages.jsonl", payload["file_structure_snapshot"]["sample_paths"])

    def test_materialize_dataset_assets_writes_reusable_contract(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            manifest = ExperimentManifest(
                protocol_version=PROTOCOL_VERSION,
                run_id="p4run_visdom_assets",
                dataset_profile_id="p4ds_visdom",
                idea_id="p4idea_demo",
                reader_context_id="p4ctx_demo",
                run_dir=str(root / "run"),
                snapshot_dir=str(root / "snapshot"),
                artifact_dir=str(root / "artifacts"),
                logs_dir=str(root / "run" / "logs"),
                config_path=str(root / "run" / "config.json"),
                metrics_path=str(root / "artifacts" / "metrics.json"),
                predictions_path=str(root / "artifacts" / "predictions.json"),
                machine_report_path=str(root / "artifacts" / "machine_report.json"),
                human_report_path=str(root / "artifacts" / "report.md"),
                dataset_tool_asset_path=str(root / "artifacts" / "dataset_tool.json"),
                dataset_adapter_path=str(root / "artifacts" / "dataset_adapter.json"),
                evaluate_tool_asset_path=str(root / "artifacts" / "evaluate_tool.json"),
                eval_summary_path=str(root / "artifacts" / "eval_summary.md"),
                bootstrap_script_path=str(root / "snapshot" / "bootstrap_env.sh"),
            )
            config = RunConfig(
                protocol_version=PROTOCOL_VERSION,
                method_name="page_lexical_retrieval",
                method_module_path="",
                runner_mode="local_dummy",
                method_branch="method/page_lexical_retrieval",
                dataset_adapter=DatasetAdapterContract(
                    dataset_profile_id="p4ds_visdom",
                    dataset_name="VisDoM",
                    task_type="visual_document_retrieval",
                    server_path=str(DEFAULT_VISDOM_FIXTURE),
                    official_metric="recall@5",
                    metadata={"fixture_path": str(DEFAULT_VISDOM_FIXTURE)},
                ),
            )
            payload = materialize_dataset_assets(manifest, config)
            adapter_contract = load_json(manifest.dataset_adapter_path)
            self.assertEqual(payload["sample_statistics"]["page_count"], 5)
            self.assertEqual(adapter_contract["contract_version"], "dataset-adapter-v2")
            self.assertEqual(adapter_contract["adapter_type"], "visdom_page_retrieval")
            self.assertEqual(adapter_contract["retrieval_granularity"], "page")
            self.assertEqual(adapter_contract["adapter_asset"]["sample_statistics"]["query_count"], 5)


if __name__ == "__main__":
    unittest.main()
