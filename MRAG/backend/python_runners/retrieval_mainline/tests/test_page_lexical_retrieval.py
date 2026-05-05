from __future__ import annotations

import tempfile
import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from methods.page_lexical_retrieval import PageLexicalRetrievalMethod
from protocol import PROTOCOL_VERSION, DatasetAdapterContract, ExperimentManifest, RunConfig
from tools.dataset_tool import materialize_dataset_assets
from tools.page_retrieval_dataset import DEFAULT_VISDOM_FIXTURE


class PageLexicalRetrievalMethodTest(unittest.TestCase):
    def test_method_produces_ranked_predictions_for_visdom_fixture(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            manifest = ExperimentManifest(
                protocol_version=PROTOCOL_VERSION,
                run_id="p4run_visdom_method",
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
            dataset_asset = materialize_dataset_assets(manifest, config)
            predictions = PageLexicalRetrievalMethod(name="page_lexical_retrieval", top_k=10).run(manifest, config, dataset_asset)
            self.assertEqual(predictions["granularity"], "page")
            self.assertEqual(predictions["runtime_stats"]["query_count"], 5)
            self.assertEqual(predictions["runtime_stats"]["failure_count"], 0)
            self.assertEqual(len(predictions["predictions"]), 5)
            first_candidates = {item["query_id"]: item["candidates"][0]["page_id"] for item in predictions["predictions"]}
            self.assertEqual(first_candidates["q_001"], "page_001")
            self.assertEqual(first_candidates["q_002"], "page_003")
            self.assertEqual(first_candidates["q_003"], "page_004")
            self.assertEqual(first_candidates["q_004"], "page_005")
            self.assertEqual(first_candidates["q_005"], "page_002")


if __name__ == "__main__":
    unittest.main()
