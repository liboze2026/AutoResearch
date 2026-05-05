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
from runtime.planner_agent import PLANNER_SCHEMA_REF, build_planner_agent

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"planner-test-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(execution_mode: str, workspace_dir: str) -> AgentRuntimeInput:
    idea_dir = Path(workspace_dir) / "ideas" / "pool" / "idea_1"
    idea_dir.mkdir(parents=True, exist_ok=True)
    idea_path = idea_dir / "structured_idea.json"
    idea_path.write_text(
        json.dumps(
            {
                "title": "Controlled Retrieval Controller",
                "description_md": "Generate a protocol-aware retrieval controller.",
                "research_direction": "multimodal retrieval",
                "target_dataset_refs": ["dasset_1"],
                "dataset_eval_protocol_refs": [str(Path(workspace_dir) / "datasets" / "dasset_1" / "evalplan.json")],
                "innovation_type": "methodology",
                "expected_advantage": "Better reproducibility.",
                "risk_points": ["May be too incremental."],
                "priority": 80,
                "confidence": 0.72,
            },
            ensure_ascii=False,
        ),
        encoding="utf-8",
    )
    dataset_dir = Path(workspace_dir) / "datasets" / "dasset_1"
    dataset_dir.mkdir(parents=True, exist_ok=True)
    evalplan_path = dataset_dir / "evalplan.json"
    evalplan_path.write_text(
        json.dumps(
            {
                "dataset_asset_id": "dasset_1",
                "task_type": "text",
                "eval_protocol_json": {"metric_list": ["accuracy", "loss"]},
                "metric_schema_json": {"primary_metric": "accuracy"},
                "split_strategy": "train_validation_test",
                "baseline_id": "baseline_1",
            },
            ensure_ascii=False,
        ),
        encoding="utf-8",
    )
    return AgentRuntimeInput(
        job_id="planner-job-001",
        agent_type="planner",
        execution_mode=execution_mode,
        model_provider="codex",
        model_name="stage3-planner-test",
        prompt_version="v1",
        input_refs=[
            AgentInputRef(ref_type="idea", ref_id="idea_1", ref_path=str(idea_path)),
            AgentInputRef(
                ref_type="dataset_asset",
                ref_id="dasset_1",
                ref_path=str(evalplan_path),
                metadata={"task_type": "text", "name": "Planner Dataset", "evalplan_path": str(evalplan_path)},
            ),
            AgentInputRef(
                ref_type="server_resource_snapshot",
                ref_id="srv_1",
                metadata={
                    "server_id": "srv_1",
                    "server_name": "shenzhenvlab",
                    "status": "online",
                    "best_free_mem_mb": 32768,
                    "best_utilization": 12,
                },
            ),
        ],
        output_schema_ref=PLANNER_SCHEMA_REF,
        skill_refs=[],
        tool_refs=[],
        memory_refs=[],
        workspace_dir=workspace_dir,
        metadata={
            "idea_id": "idea_1",
            "dataset_asset_refs": ["dasset_1"],
            "eval_protocol_refs": [str(evalplan_path)],
            "server_resource_snapshot_refs": ["srv_1"],
            "baseline_refs": ["baseline_1"],
            "idea": {
                "title": "Controlled Retrieval Controller",
                "research_direction": "multimodal retrieval",
                "innovation_type": "methodology",
                "expected_advantage": "Better reproducibility.",
                "risk_points": ["May be too incremental."],
            },
            "dataset_assets": [
                {
                    "dataset_asset_id": "dasset_1",
                    "name": "Planner Dataset",
                    "task_type": "text",
                }
            ],
            "eval_plans": [
                {
                    "eval_protocol_ref": str(evalplan_path),
                    "metric_list": ["accuracy", "loss"],
                    "task_type": "text",
                }
            ],
            "baselines": [{"baseline_id": "baseline_1", "name": "BM25"}],
            "human_hints": ["Prefer the smaller template first."],
        },
    )


class PlannerAgentTests(unittest.TestCase):
    def test_generate_plan_from_idea_and_dataset(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_planner_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertTrue(output.normalized_payload["experiment_plan_json"])
            self.assertTrue(output.normalized_payload["resource_estimate"])
            self.assertTrue(output.normalized_payload["run_sequence"])
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_planner_codex_cli_falls_back_to_mock(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("codex_cli", tmpdir)
            agent = build_planner_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.normalized_payload["execution_mode_requested"], "codex_cli")
            self.assertEqual(output.normalized_payload["execution_mode_used"], "mock")
            self.assertTrue(any("falling back to mock executor" in item.lower() for item in output.warnings))
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_planner_repair_handles_partial_payload(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_planner_agent(contract)
            output = agent.run(contract)
            output.normalized_payload = {
                "message": "Recovered planner output.",
                "result": (
                    "```json\n"
                    "{\"plan\": {\"dataset_asset_id\": \"dasset_1\", \"idea_id\": \"idea_1\"}, "
                    "\"template_type\": \"text_finetune_v1\", "
                    "\"resourceEstimate\": {\"gpu_count\": 1, \"min_free_mem_mb\": 16384}, "
                    "\"steps\": [\"generate_experiment_spec\", \"queue_experiment\"], "
                    "\"successCriteria\": {\"required_metrics\": [\"accuracy\"]}, "
                    "\"fallbackPlan\": {\"fallback_template_type\": \"generic_train_v1\"}}\n```"
                ),
            }
            output.validation_status = "pending"
            output.repair_status = "pending"

            repaired = agent.repairer.repair(contract, output, ["invalid"])[0]
            errors = agent.validator.validate_output(contract, repaired)

            self.assertFalse(errors)
            self.assertEqual(repaired.normalized_payload["train_template_type"], "text_finetune_v1")
            self.assertEqual(repaired.normalized_payload["resource_estimate"]["gpu_count"], 1)
            self.assertEqual(repaired.normalized_payload["run_sequence"][0], "generate_experiment_spec")
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
