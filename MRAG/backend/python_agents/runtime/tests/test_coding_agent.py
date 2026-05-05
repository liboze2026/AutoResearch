from __future__ import annotations

import json
import shutil
import sys
import unittest
import uuid
from pathlib import Path

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.contract import AgentInputRef, AgentRuntimeInput
from runtime.coding_agent import CODING_SCHEMA_REF, build_coding_agent, build_coding_codex_schema, build_coding_payload, CodingRequest

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"coding-test-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(execution_mode: str, workspace_dir: str) -> AgentRuntimeInput:
    exp_dir = Path(workspace_dir) / "experiments" / "exp_1"
    exp_dir.mkdir(parents=True, exist_ok=True)
    plan_path = exp_dir / "plan.json"
    plan_path.write_text(
        json.dumps(
            {
                "experiment_id": "exp_1",
                "idea_id": "idea_1",
                "dataset_asset_id": "dasset_1",
                "train_template_type": "text_finetune_v1",
                "resource_estimate": {"preferred_server_name": "shenzhenvlab"},
            },
            ensure_ascii=False,
        ),
        encoding="utf-8",
    )
    spec_path = exp_dir / "spec.json"
    spec_path.write_text(
        json.dumps(
            {
                "experiment_id": "exp_1",
                "dataset_ref": {"dataset_asset_id": "dasset_1"},
                "dataset_loader_ref": {"loader_id": "mrag.dataset_asset_loader.v1"},
                "train_template_type": "text_finetune_v1",
                "model_name": "mock/llama3.1-8b-instruct",
                "hyperparams": {"epochs": 3, "batch_size": 8, "gradient_accumulation": 4},
                "output_dir": str(exp_dir / "run_1" / "outputs"),
                "expected_metrics": {"primary": "accuracy"},
                "comparison_targets": [{"type": "baseline", "id": "baseline_1"}],
            },
            ensure_ascii=False,
        ),
        encoding="utf-8",
    )
    evalplan_path = Path(workspace_dir) / "datasets" / "dasset_1" / "evalplan.json"
    evalplan_path.parent.mkdir(parents=True, exist_ok=True)
    evalplan_path.write_text(
        json.dumps(
            {
                "dataset_asset_id": "dasset_1",
                "eval_protocol_json": {"metric_list": ["accuracy", "loss"]},
                "split_strategy": "train_validation_test",
            },
            ensure_ascii=False,
        ),
        encoding="utf-8",
    )
    return AgentRuntimeInput(
        job_id="coding-job-001",
        agent_type="coding",
        execution_mode=execution_mode,
        model_provider="codex",
        model_name="stage3-coding-test",
        prompt_version="v1",
        input_refs=[
            AgentInputRef(ref_type="experiment_plan", ref_path=str(plan_path)),
            AgentInputRef(ref_type="experiment_spec", ref_path=str(spec_path)),
            AgentInputRef(ref_type="dataset_eval_protocol", ref_path=str(evalplan_path)),
            AgentInputRef(ref_type="idea", ref_id="idea_1", metadata={"title": "Controlled Retrieval Controller"}),
        ],
        output_schema_ref=CODING_SCHEMA_REF,
        skill_refs=[],
        tool_refs=[],
        memory_refs=[],
        workspace_dir=workspace_dir,
        metadata={
            "experiment_id": "exp_1",
            "train_template_ref": "mock_train_template",
            "eval_protocol_ref": str(evalplan_path),
            "idea": {"title": "Controlled Retrieval Controller"},
        },
    )


class CodingAgentTests(unittest.TestCase):
    def test_coding_codex_schema_uses_string_patch_values(self) -> None:
        schema = build_coding_codex_schema()
        value_schema = schema["properties"]["code_patch_manifest"]["items"]["properties"]["value"]

        self.assertEqual(value_schema["type"], "string")

    def test_coding_payload_stringifies_patch_values(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            request = CodingRequest.from_contract(contract)

            payload = build_coding_payload(request)

            self.assertTrue(payload["code_patch_manifest"])
            self.assertTrue(all(isinstance(item["value"], str) for item in payload["code_patch_manifest"]))
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_mock_coding_generates_patch_manifest(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_coding_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertTrue(output.normalized_payload["code_patch_manifest"])
            self.assertTrue(output.normalized_payload["spec_overrides"])
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_coding_codex_cli_falls_back_to_mock(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("codex_cli", tmpdir)
            agent = build_coding_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.normalized_payload["execution_mode_requested"], "codex_cli")
            self.assertEqual(output.normalized_payload["execution_mode_used"], "mock")
            self.assertTrue(any("falling back to mock executor" in item.lower() for item in output.warnings))
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_coding_repair_handles_partial_payload(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_coding_agent(contract)
            output = agent.run(contract)
            output.normalized_payload = {
                "message": "Recovered coding output.",
                "result": (
                    "```json\n"
                    "{\"patch_manifest\": [{\"summary\": \"Recovered patch\"}], "
                    "\"metrics\": {\"primary_metric\": \"accuracy\"}, "
                    "\"evaluationSummaryMd\": \"Recovered evaluation summary.\"}\n```"
                ),
            }
            output.validation_status = "pending"
            output.repair_status = "pending"

            repaired = agent.repairer.repair(contract, output, ["invalid"])[0]
            errors = agent.validator.validate_output(contract, repaired)

            self.assertFalse(errors)
            self.assertEqual(repaired.normalized_payload["code_patch_manifest"][0]["summary"], "Recovered patch")
            self.assertEqual(repaired.normalized_payload["metrics_summary"]["primary_metric"], "accuracy")
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
