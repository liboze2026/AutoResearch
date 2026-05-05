from __future__ import annotations

import shutil
import sys
import unittest
import uuid
from pathlib import Path

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.contract import AgentRuntimeInput  # noqa: E402
from runtime.writer_phase4_agent import build_writer_phase4_agent  # noqa: E402

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"writer-phase4-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(execution_mode: str, workspace_dir: str) -> AgentRuntimeInput:
    return AgentRuntimeInput(
        job_id="ajob_phase4_writer_1",
        agent_type="writer_phase4",
        execution_mode=execution_mode,
        model_provider="phase4_writer",
        model_name="writer-phase4-default",
        prompt_version="v1",
        output_schema_ref="schemas/writer-phase4-output-v1.json",
        workspace_dir=workspace_dir,
        metadata={
            "run_manifest_id": "p4run_1",
            "user_notes": "Document both successful metrics and limitations.",
            "dataset_profile": {
                "id": "p4ds_visdom",
                "datasetName": "VisDoM",
                "taskType": "page_level_retrieval",
                "officialMetric": "recall@5",
                "sampleStatistics": {"pageCount": 5, "queryCount": 5},
                "fileStructureSnapshot": {"documents": 2, "queries": 5},
                "citation": "VisDoM dataset citation.",
            },
            "reader_context": {
                "id": "p4ctx_1",
                "summary": "Reader summary for page-level retrieval.",
                "taskDefinition": "Retrieve the correct page for each question.",
                "structuredContext": {
                    "relevant_methods_landscape": ["hybrid sparse+dense retrieval"],
                    "implementation_constraints": ["page-level first"],
                    "promising_research_directions": ["move to snippet-level retrieval later"],
                },
            },
            "reader_sources": [
                {
                    "id": "p4src_1",
                    "title": "VisDoM: Multi-Document QA with Visually Rich Elements Using Multimodal Retrieval-Augmented Generation",
                    "authors": ["Author A", "Author B"],
                    "venue": "ACL",
                    "publicationYear": 2025,
                    "sourceType": "conference",
                    "sourceUrl": "https://example.org/visdom-paper",
                    "metadata": {"doi": "10.1000/visdom.1"},
                },
                {
                    "id": "p4src_2",
                    "title": "Layout-aware document retrieval for complex pages",
                    "authors": ["Author C"],
                    "venue": "arXiv",
                    "publicationYear": 2024,
                    "sourceType": "arxiv",
                    "sourceUrl": "https://example.org/layout-paper",
                },
            ],
            "selected_idea": {
                "id": "p4idea_1",
                "title": "Layout-Aware Hard Negative Mining",
                "problemDefinition": "Improve retrieval recall on visually similar pages.",
                "coreMethod": "Weighted lexical scoring with layout and OCR cues.",
                "differentiators": "Adds layout-aware weighting to the baseline.",
                "riskPoints": ["OCR noise"],
                "expectedGains": ["better recall@5"],
            },
            "run_manifest": {
                "id": "p4run_1",
                "runnerMode": "local_dummy",
                "status": "succeeded",
                "retryCount": 1,
                "maxRetryCount": 3,
            },
            "metrics": {
                "primary_metric": "recall@5",
                "values": {"recall@1": 0.6, "recall@5": 0.8, "recall@10": 0.9, "mrr": 0.7, "ndcg@10": 0.76},
            },
            "artifact_summary": {"metrics_path": "/tmp/metrics.json", "human_report_path": "/tmp/report.md"},
            "failure_summary": {},
            "coding_machine_report": {"pipeline": "retrieval_mainline"},
            "coding_human_report_excerpt": "Coding run finished and produced stable fixture metrics.",
        },
    )


class WriterPhase4AgentTests(unittest.TestCase):
    def test_writer_phase4_builds_dual_layer_report(self) -> None:
        workspace = make_workspace()
        try:
            contract = build_contract("mock", workspace)
            result = build_writer_phase4_agent(contract).run(contract)
            self.assertEqual(result.status, "succeeded")
            self.assertEqual(result.validation_status, "succeeded")
            self.assertIn("machine_readable_report", result.normalized_payload)
            self.assertIn("human_readable_report_md", result.normalized_payload)
            self.assertTrue(result.normalized_payload["citation_refs"])
            self.assertTrue(result.artifact_manifest)
        finally:
            shutil.rmtree(workspace, ignore_errors=True)

    def test_writer_phase4_codex_cli_falls_back_with_trace(self) -> None:
        workspace = make_workspace()
        try:
            contract = build_contract("codex_cli", workspace)
            result = build_writer_phase4_agent(contract).run(contract)
            self.assertEqual(result.status, "succeeded")
            self.assertIn(result.normalized_payload["execution_mode_used"], {"mock", "codex_cli"})
            self.assertIn("generation_trace", result.normalized_payload)
            self.assertTrue(result.warnings)
        finally:
            shutil.rmtree(workspace, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
