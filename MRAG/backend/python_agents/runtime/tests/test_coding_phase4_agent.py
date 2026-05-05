from __future__ import annotations

import unittest
import sys
from pathlib import Path
import shutil
import uuid

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.coding_phase4_agent import build_coding_phase4_agent  # noqa: E402
from runtime.contract import AgentRuntimeInput  # noqa: E402

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"coding-phase4-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


class CodingPhase4AgentTest(unittest.TestCase):
    def test_agent_builds_protocol_payload(self) -> None:
        workspace = make_workspace()
        contract = AgentRuntimeInput(
            job_id="ajob_phase4_coding_1",
            agent_type="coding_phase4",
            execution_mode="mock",
            model_provider="phase4_coding",
            model_name="coding-phase4-default",
            prompt_version="v1",
            output_schema_ref="schemas/coding-phase4-output-v1.json",
            workspace_dir=workspace,
            metadata={
                "run_manifest_id": "p4run_demo",
                "runner_mode": "local_dummy",
                "max_retry_count": 3,
                "dataset_profile": {
                    "id": "p4ds_demo",
                    "datasetName": "VisDoM",
                    "taskType": "multimodal_retrieval",
                    "officialMetric": "recall@5",
                },
                "idea": {
                    "id": "p4idea_demo",
                    "title": "Layout-Aware Hard Negative Mining",
                    "coreMethod": "Add layout-aware hard negative sampling."
                },
                "reader_context": {
                    "task_definition": "Page-level retrieval for visually rich documents.",
                    "implementation_constraints": ["page-level first"]
                }
            },
        )
        try:
            result = build_coding_phase4_agent(contract).run(contract)
            self.assertEqual(result.status, "succeeded")
            self.assertEqual(result.validation_status, "succeeded")
            self.assertIn("phase4_config", result.normalized_payload)
            self.assertIn("method_module", result.normalized_payload)
            self.assertEqual(result.normalized_payload["protocol_version"], "phase4-retrieval-mainline-v1")
            self.assertEqual(result.normalized_payload["phase4_config"]["dataset_adapter"]["metadata"]["retrieval_granularity"], "page")
            self.assertIn("recall@10", result.normalized_payload["phase4_config"]["evaluate"]["ranking_metrics"])
            self.assertIn("PageLexicalRetrievalMethod", result.normalized_payload["method_module"]["content"])
            self.assertIn("generation_trace", result.normalized_payload)
            self.assertEqual(result.normalized_payload["generation_trace"]["execution_mode_used"], "mock")
        finally:
            shutil.rmtree(workspace, ignore_errors=True)

    def test_codex_cli_mode_falls_back_but_still_generates_auditable_payload(self) -> None:
        workspace = make_workspace()
        contract = AgentRuntimeInput(
            job_id="ajob_phase4_coding_2",
            agent_type="coding_phase4",
            execution_mode="codex_cli",
            model_provider="phase4_coding",
            model_name="coding-phase4-default",
            prompt_version="v1",
            output_schema_ref="schemas/coding-phase4-output-v1.json",
            workspace_dir=workspace,
            metadata={
                "run_manifest_id": "p4run_demo",
                "runner_mode": "local_dummy",
                "max_retry_count": 2,
                "dataset_profile": {
                    "id": "p4ds_demo",
                    "datasetName": "VisDoM",
                    "taskType": "multimodal_retrieval",
                    "officialMetric": "recall@5",
                },
                "idea": {
                    "id": "p4idea_demo",
                    "title": "Hybrid Retrieval with OCR Expansion",
                    "coreMethod": "Use OCR-aware lexical expansion."
                },
                "reader_context": {
                    "task_definition": "Page-level retrieval for visually rich documents.",
                    "implementation_constraints": ["page-level first"]
                }
            },
        )
        try:
            result = build_coding_phase4_agent(contract).run(contract)
            self.assertEqual(result.status, "succeeded")
            self.assertEqual(result.validation_status, "succeeded")
            self.assertIn(result.normalized_payload["execution_mode_used"], {"mock", "codex_cli"})
            self.assertIn("generation_trace", result.normalized_payload)
            self.assertTrue(result.artifact_manifest)
            self.assertTrue(result.tool_usages)
        finally:
            shutil.rmtree(workspace, ignore_errors=True)

    def test_api_mode_falls_back_but_preserves_generation_trace(self) -> None:
        workspace = make_workspace()
        contract = AgentRuntimeInput(
            job_id="ajob_phase4_coding_3",
            agent_type="coding_phase4",
            execution_mode="api",
            model_provider="phase4_coding",
            model_name="coding-phase4-default",
            prompt_version="v1",
            output_schema_ref="schemas/coding-phase4-output-v1.json",
            workspace_dir=workspace,
            metadata={
                "run_manifest_id": "p4run_demo",
                "runner_mode": "local_dummy",
                "max_retry_count": 1,
                "dataset_profile": {
                    "id": "p4ds_demo",
                    "datasetName": "VisDoM",
                    "taskType": "multimodal_retrieval",
                    "officialMetric": "recall@5",
                },
                "idea": {
                    "id": "p4idea_demo",
                    "title": "Adaptive OCR Title Weighting",
                    "coreMethod": "Adjust lexical scoring using OCR and title confidence."
                },
                "reader_context": {
                    "task_definition": "Page-level retrieval for visually rich documents.",
                    "implementation_constraints": ["page-level first"]
                }
            },
        )
        try:
            result = build_coding_phase4_agent(contract).run(contract)
            self.assertEqual(result.status, "succeeded")
            self.assertEqual(result.validation_status, "succeeded")
            self.assertIn(result.normalized_payload["execution_mode_used"], {"mock", "api"})
            self.assertIn("generation_trace", result.normalized_payload)
            self.assertTrue(result.tool_usages)
            self.assertTrue(result.warnings)
        finally:
            shutil.rmtree(workspace, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
