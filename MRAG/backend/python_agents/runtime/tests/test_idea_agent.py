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
from runtime.idea_agent import IDEA_SCHEMA_REF, build_idea_agent

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"idea-test-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(execution_mode: str, workspace_dir: str) -> AgentRuntimeInput:
    insight_dir = Path(workspace_dir) / "papers" / "insights" / "paper_1"
    insight_dir.mkdir(parents=True, exist_ok=True)
    insight_path = insight_dir / "insight_agent_output.json"
    insight_path.write_text(
        json.dumps(
            {
                "paper_id": "paper_1",
                "paper_title": "Controlled Retrieval Paper",
                "insight_id": "pinsight_1",
                "summary_md": "This paper improves retrieval reliability.",
                "contributions_json": ["Controlled retrieval orchestration."],
                "novelty_points": ["Turns retrieval into a controlled pipeline."],
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
                "dataset_location": "/data/retrieval",
                "eval_protocol_json": {"metric_list": ["mrr", "ndcg@10"]},
                "split_strategy": "query_document_train_dev_test",
            },
            ensure_ascii=False,
        ),
        encoding="utf-8",
    )
    return AgentRuntimeInput(
        job_id="idea-job-001",
        agent_type="idea_generator",
        execution_mode=execution_mode,
        model_provider="codex",
        model_name="stage3-idea-test",
        prompt_version="v1",
        input_refs=[
            AgentInputRef(ref_type="insight", ref_id="pinsight_1", ref_path=str(insight_path)),
            AgentInputRef(
                ref_type="dataset_asset",
                ref_id="dasset_1",
                ref_path=str(evalplan_path),
                metadata={"evalplan_path": str(evalplan_path), "name": "Retrieval Benchmark", "task_type": "retrieval"},
            ),
        ],
        output_schema_ref=IDEA_SCHEMA_REF,
        skill_refs=[],
        tool_refs=[],
        memory_refs=[],
        workspace_dir=workspace_dir,
        metadata={
            "paper_insight_refs": ["pinsight_1"],
            "dataset_asset_refs": ["dasset_1"],
            "human_hints": ["Focus on retrieval robustness."],
        },
    )


class IdeaAgentTests(unittest.TestCase):
    def test_generate_idea_from_insight_and_dataset(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_idea_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertTrue(output.normalized_payload["title"])
            self.assertEqual(output.normalized_payload["target_dataset_refs"], ["dasset_1"])
            self.assertTrue(output.normalized_payload["dataset_eval_protocol_refs"])
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_manual_idea_is_standardized(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            contract.metadata["manual_idea"] = {
                "title": "Human Seed Idea",
                "description_md": "Build a smaller retrieval controller.",
                "research_direction": "retrieval control",
                "innovation_type": "human_curated",
                "expected_advantage": "Faster iteration.",
                "risk_points": ["May be too incremental."],
                "priority": 88,
                "confidence": 0.81,
            }
            agent = build_idea_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.normalized_payload["title"], "Human Seed Idea")
            self.assertEqual(output.normalized_payload["innovation_type"], "human_curated")
            self.assertEqual(output.normalized_payload["priority"], 88)
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_idea_repair_handles_partial_payload(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_idea_agent(contract)
            output = agent.run(contract)
            output.normalized_payload = {
                "message": "Recovered idea output",
                "result": (
                    "```json\n"
                    "{\"title\": \"Recovered Idea\", \"description\": \"Recovered description\", "
                    "\"direction\": \"retrieval\", \"dataset_refs\": [\"dasset_1\"], "
                    "\"evalplan_refs\": [\"/tmp/evalplan.json\"], \"innovationType\": \"method_adaptation\", "
                    "\"expectedAdvantage\": \"Higher robustness\", \"risks\": [\"Needs validation\"], "
                    "\"priority\": 77, \"confidence\": 0.66}\n```"
                ),
            }
            output.validation_status = "pending"
            output.repair_status = "pending"

            repaired = agent.repairer.repair(contract, output, ["invalid"])[0]
            errors = agent.validator.validate_output(contract, repaired)

            self.assertFalse(errors)
            self.assertEqual(repaired.normalized_payload["title"], "Recovered Idea")
            self.assertEqual(repaired.normalized_payload["research_direction"], "retrieval")
            self.assertEqual(repaired.normalized_payload["priority"], 77)
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
