from __future__ import annotations

import shutil
import sys
import unittest
import uuid
from pathlib import Path

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.contract import AgentRuntimeInput
from runtime.dataset_agent import DATASET_SCHEMA_REF, build_dataset_agent

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"dataset-test-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(execution_mode: str, workspace_dir: str) -> AgentRuntimeInput:
    return AgentRuntimeInput(
        job_id="dataset-job-001",
        agent_type="dataset",
        execution_mode=execution_mode,
        model_provider="codex",
        model_name="stage3-dataset-test",
        prompt_version="v1",
        input_refs=[],
        output_schema_ref=DATASET_SCHEMA_REF,
        skill_refs=[],
        tool_refs=[],
        memory_refs=[],
        workspace_dir=workspace_dir,
        metadata={
            "research_direction": "multimodal retrieval",
            "task_type": "retrieval",
            "keywords": ["retrieval", "benchmark"],
            "target_server_preference": "shenzhenvlab",
            "dataset_constraints": {"max_size_gb": 20},
            "discovered_datasets": [
                {
                    "id": "ds_existing_1",
                    "name": "Multimodal Retrieval Benchmark",
                    "description": "A benchmark for retrieval experiments.",
                    "path": "/data/retrieval/benchmark",
                    "server_name": "shenzhenvlab",
                    "modality": "retrieval",
                }
            ],
            "server_context": {
                "selected_server_name": "shenzhenvlab",
                "decision_mode": "real",
                "gpu_available": True,
                "fallback_reason": "",
            },
        },
    )


class DatasetAgentTests(unittest.TestCase):
    def test_mock_dataset_returns_eval_protocol(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_dataset_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertEqual(output.normalized_payload["fetch_action"], "register_existing")
            self.assertEqual(output.normalized_payload["selected_dataset_ref"], "ds_existing_1")
            self.assertEqual(output.normalized_payload["eval_protocol_json"]["task_type"], "retrieval")
            self.assertTrue(output.normalized_payload["eval_protocol_json"]["metric_list"])
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_codex_cli_dataset_falls_back_with_warning(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("codex_cli", tmpdir)
            agent = build_dataset_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.normalized_payload["execution_mode_requested"], "codex_cli")
            self.assertEqual(output.normalized_payload["execution_mode_used"], "mock")
            self.assertEqual(output.normalized_payload["dataset_location"], "/data/retrieval/benchmark")
            self.assertTrue(any("falling back to mock executor" in item.lower() for item in output.warnings))
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_dataset_without_discovered_assets_repairs_dataset_location(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("codex_cli", tmpdir)
            contract.metadata["discovered_datasets"] = []
            agent = build_dataset_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertEqual(output.repair_status, "not_needed")
            self.assertEqual(output.normalized_payload["fetch_action"], "mock_download")
            self.assertTrue(output.normalized_payload["dataset_location"].endswith("mock_dataset"))
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_dataset_repair_handles_partial_payload(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_dataset_agent(contract)
            output = agent.run(contract)
            output.normalized_payload = {
                "message": "Recovered dataset plan.",
                "result": (
                    "```json\n"
                    "{\"datasetLocation\": \"/data/mock-dataset\", \"metrics\": {\"primary_metric\": \"accuracy\"}, "
                    "\"eval_protocol\": {\"task_type\": \"classification\", \"metric_list\": [\"accuracy\"], "
                    "\"evaluation_steps\": [\"step1\"], \"data_split\": {\"strategy\": \"train_test\"}, "
                    "\"baseline_needed\": true, \"report_template\": {\"format\": \"markdown\"}}, "
                    "\"data_split_strategy\": \"train_test\"}\n```"
                ),
            }
            output.validation_status = "pending"
            output.repair_status = "pending"

            repaired = agent.repairer.repair(contract, output, ["invalid"])[0]
            errors = agent.validator.validate_output(contract, repaired)

            self.assertFalse(errors)
            self.assertEqual(repaired.normalized_payload["dataset_location"], "/data/mock-dataset")
            self.assertEqual(repaired.normalized_payload["split_strategy"], "train_test")
            self.assertEqual(repaired.normalized_payload["metric_schema_json"]["primary_metric"], "accuracy")
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
