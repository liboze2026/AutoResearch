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
from runtime.writer_agent import WRITER_SCHEMA_REF, build_writer_agent

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"writer-test-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(execution_mode: str, workspace_dir: str) -> AgentRuntimeInput:
    template_path = Path(workspace_dir) / "templates" / "paper_template.md"
    template_path.parent.mkdir(parents=True, exist_ok=True)
    template_path.write_text("# Abstract\n# Introduction\n# Method\n# Experiments\n# Conclusion\n", encoding="utf-8")
    return AgentRuntimeInput(
        job_id="writer-job-001",
        agent_type="writer",
        execution_mode=execution_mode,
        model_provider="codex",
        model_name="stage3-writer-test",
        prompt_version="v1",
        input_refs=[
            AgentInputRef(ref_type="paper_template", ref_path=str(template_path)),
            AgentInputRef(ref_type="idea", ref_id="idea_1", metadata={"title": "Controlled Drafting", "research_direction": "controlled writing"}),
            AgentInputRef(
                ref_type="experiment_result",
                ref_id="run_1",
                metadata={"metrics": {"primary_metric": "accuracy", "values": {"accuracy": 0.88, "loss": 0.12}}},
            ),
            AgentInputRef(ref_type="comparison", ref_id="cmp_1", metadata={"summary_md": "Accuracy is higher than baseline."}),
            AgentInputRef(ref_type="citation", ref_id="paper_1", metadata={"citation_text": "Author et al. Controlled Drafting, 2026."}),
        ],
        output_schema_ref=WRITER_SCHEMA_REF,
        skill_refs=[],
        tool_refs=[],
        memory_refs=[],
        workspace_dir=workspace_dir,
        metadata={
            "paper_template_ref": str(template_path),
            "idea_refs": ["idea_1"],
            "experiment_result_refs": ["run_1"],
            "comparison_refs": ["cmp_1"],
            "citation_refs": ["paper_1"],
            "ideas": [{"title": "Controlled Drafting", "research_direction": "controlled writing", "expected_advantage": "improve traceability"}],
            "experiment_results": [{"run_id": "run_1", "metrics": {"primary_metric": "accuracy", "values": {"accuracy": 0.88, "loss": 0.12}}}],
            "comparisons": [{"comparison_id": "cmp_1", "summary_md": "Accuracy is higher than baseline."}],
            "citations": [{"citation_ref": "paper_1", "citation_text": "Author et al. Controlled Drafting, 2026."}],
        },
    )


class WriterAgentTests(unittest.TestCase):
    def test_mock_writer_generates_structured_draft(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_writer_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertTrue(output.normalized_payload["title"])
            self.assertTrue(output.normalized_payload["figure_plan"])
            self.assertEqual(output.normalized_payload["metadata"]["picture_mode"], "mock")
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_writer_codex_cli_falls_back_to_mock(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("codex_cli", tmpdir)
            agent = build_writer_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.normalized_payload["execution_mode_requested"], "codex_cli")
            self.assertEqual(output.normalized_payload["execution_mode_used"], "mock")
            self.assertTrue(any("falling back to mock executor" in item.lower() for item in output.warnings))
            self.assertTrue(any("picture agent remains mock" in item.lower() for item in output.warnings))
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_writer_repair_handles_partial_payload(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_writer_agent(contract)
            output = agent.run(contract)
            output.normalized_payload = {
                "message": "Recovered draft.",
                "result": (
                    "```json\n"
                    "{\"title\": \"Recovered Title\", "
                    "\"abstract\": \"Recovered abstract.\", "
                    "\"introduction\": \"Recovered intro.\", "
                    "\"method\": \"Recovered method.\", "
                    "\"experiments\": \"Recovered experiments.\", "
                    "\"conclusion\": \"Recovered conclusion.\", "
                    "\"references\": [\"Recovered ref\"], "
                    "\"figures\": [{\"title\": \"Recovered figure\"}]}\n```"
                ),
            }
            output.validation_status = "pending"
            output.repair_status = "pending"

            repaired = agent.repairer.repair(contract, output, ["invalid"])[0]
            errors = agent.validator.validate_output(contract, repaired)

            self.assertFalse(errors)
            self.assertEqual(repaired.normalized_payload["title"], "Recovered Title")
            self.assertEqual(repaired.normalized_payload["references_stub"][0], "Recovered ref")
            self.assertEqual(repaired.normalized_payload["figure_plan"][0]["title"], "Recovered figure")
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
