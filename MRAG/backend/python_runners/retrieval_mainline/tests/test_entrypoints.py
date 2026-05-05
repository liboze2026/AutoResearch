from __future__ import annotations

import os
import tempfile
import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from eval_entrypoint import main as eval_main
from protocol import PROTOCOL_VERSION, DatasetAdapterContract, ExperimentManifest, RunConfig, dump_json, load_json
from run_entrypoint import main as run_main
from tools.page_retrieval_dataset import DEFAULT_VISDOM_FIXTURE


class EntrypointSmokeTest(unittest.TestCase):
    def test_run_and_eval_entrypoints(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            snapshot_dir = root / "snapshot"
            snapshot_dir.mkdir(parents=True, exist_ok=True)
            method_path = root / "generated_method.py"
            method_path.write_text(
                "from methods.page_lexical_retrieval import PageLexicalRetrievalMethod\n\n"
                "def build_method():\n"
                "    return PageLexicalRetrievalMethod(name='generated_page_lexical', method_tags=['layout', 'lexical'], top_k=10)\n",
                encoding="utf-8",
            )
            config_path = root / "run" / "config.json"
            manifest_path = root / "run" / "manifest.json"
            config = RunConfig(
                protocol_version=PROTOCOL_VERSION,
                method_name="generated_page_lexical",
                method_module_path=str(method_path),
                runner_mode="local_dummy",
                method_branch="method/generated_page_lexical",
                dataset_adapter=DatasetAdapterContract(
                    dataset_profile_id="p4ds_demo",
                    dataset_name="VisDoM",
                    task_type="visual_document_retrieval",
                    server_path=str(DEFAULT_VISDOM_FIXTURE),
                    official_metric="recall@5",
                    metadata={
                        "fixture_path": str(DEFAULT_VISDOM_FIXTURE),
                        "adapter_type": "visdom_page_retrieval",
                        "retrieval_granularity": "page",
                    },
                ),
                evaluate={"primary_metric": "recall@5"},
            )
            manifest = ExperimentManifest(
                protocol_version=PROTOCOL_VERSION,
                run_id="p4run_smoke",
                dataset_profile_id="p4ds_demo",
                idea_id="p4idea_demo",
                reader_context_id="p4ctx_demo",
                run_dir=str(root / "run"),
                snapshot_dir=str(snapshot_dir),
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
                bootstrap_script_path=str(snapshot_dir / "bootstrap_env.sh"),
            )
            dump_json(config_path, config.to_payload())
            dump_json(manifest_path, manifest.to_payload())
            old_argv = os.sys.argv
            try:
                os.sys.argv = ["run_entrypoint.py", "--manifest", str(manifest_path)]
                self.assertEqual(run_main(), 0)
                os.sys.argv = ["eval_entrypoint.py", "--manifest", str(manifest_path)]
                self.assertEqual(eval_main(), 0)
            finally:
                os.sys.argv = old_argv
            metrics = load_json(manifest.metrics_path)
            self.assertIn("recall@1", metrics["values"])
            self.assertIn("recall@5", metrics["values"])
            self.assertIn("recall@10", metrics["values"])
            self.assertIn("mrr", metrics["values"])
            self.assertIn("ndcg@10", metrics["values"])
            self.assertIn("avg_query_latency_ms", metrics["values"])
            self.assertTrue(Path(manifest.human_report_path).exists())
            self.assertTrue(Path(manifest.logs_dir, "run.log").exists())


if __name__ == "__main__":
    unittest.main()
