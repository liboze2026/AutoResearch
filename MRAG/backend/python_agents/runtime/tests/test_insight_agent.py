from __future__ import annotations

import shutil
import sys
import unittest
import uuid
from pathlib import Path

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.contract import AgentInputRef, AgentRuntimeInput
from runtime.insight_agent import INSIGHT_SCHEMA_REF, build_insight_agent

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"insight-test-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(execution_mode: str, workspace_dir: str) -> AgentRuntimeInput:
    parsed_dir = Path(workspace_dir) / "papers" / "parsed" / "paper_1"
    parsed_dir.mkdir(parents=True, exist_ok=True)
    parsed_path = parsed_dir / "parsed.md"
    parsed_path.write_text("# Demo Insight Paper\n\nThis paper discusses structured pipelines.\n", encoding="utf-8")
    return AgentRuntimeInput(
        job_id="insight-job-001",
        agent_type="insight",
        execution_mode=execution_mode,
        model_provider="codex",
        model_name="stage3-insight-test",
        prompt_version="v1",
        input_refs=[
            AgentInputRef(ref_type="paper", ref_id="paper_1"),
            AgentInputRef(ref_type="parsed_content", ref_id="paper_1", ref_path=str(parsed_path)),
        ],
        output_schema_ref=INSIGHT_SCHEMA_REF,
        skill_refs=[],
        tool_refs=[],
        memory_refs=[],
        workspace_dir=workspace_dir,
        metadata={"paper_id": "paper_1", "parsed_content_ref": str(parsed_path), "focus": "novelty"},
    )


class InsightAgentTests(unittest.TestCase):
    def test_mock_insight_returns_structured_output(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_insight_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertTrue(output.normalized_payload["summary_md"])
            self.assertTrue(output.normalized_payload["novelty_points"])
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_codex_cli_insight_falls_back_with_warning(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("codex_cli", tmpdir)
            agent = build_insight_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.normalized_payload["execution_mode_requested"], "codex_cli")
            self.assertEqual(output.normalized_payload["execution_mode_used"], "mock")
            self.assertTrue(any("falling back to mock executor" in item.lower() for item in output.warnings))
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_insight_repair_handles_partial_payload(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_insight_agent(contract)
            output = agent.run(contract)
            output.normalized_payload = {
                "summary": "Recovered insight summary.",
                "result": "```json\n{\"contributions\": [\"Recovered contribution\"], \"methods\": \"Recovered method\", \"limitations\": [\"Recovered limitation\"], \"novelty_points\": \"Recovered novelty\"}\n```",
            }
            output.validation_status = "pending"
            output.repair_status = "pending"

            repaired = agent.repairer.repair(contract, output, ["invalid"])[0]
            errors = agent.validator.validate_output(contract, repaired)

            self.assertFalse(errors)
            self.assertEqual(repaired.normalized_payload["contributions_json"][0], "Recovered contribution")
            self.assertEqual(repaired.normalized_payload["methods_json"][0], "Recovered method")
            self.assertEqual(repaired.normalized_payload["novelty_points"][0], "Recovered novelty")
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
