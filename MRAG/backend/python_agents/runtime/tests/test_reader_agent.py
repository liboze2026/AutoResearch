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
from runtime.reader_agent import READER_SCHEMA_REF, build_reader_agent

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"reader-test-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(execution_mode: str, workspace_dir: str) -> AgentRuntimeInput:
    return AgentRuntimeInput(
        job_id="reader-job-001",
        agent_type="reader",
        execution_mode=execution_mode,
        model_provider="codex",
        model_name="stage3-reader-test",
        prompt_version="v1",
        input_refs=[],
        output_schema_ref=READER_SCHEMA_REF,
        skill_refs=[],
        tool_refs=[],
        memory_refs=[],
        workspace_dir=workspace_dir,
        metadata={
            "research_direction": "multimodal retrieval",
            "keywords": ["retrieval", "reasoning"],
            "source_scope": "mixed",
            "time_range": {"year": 2026},
            "max_papers": 3,
        },
    )


class ReaderAgentTests(unittest.TestCase):
    def test_mock_reader_returns_candidate_papers(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = build_reader_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertEqual(len(output.normalized_payload["candidate_papers"]), 3)
            self.assertEqual(output.normalized_payload["items"], output.normalized_payload["candidate_papers"])
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_codex_cli_reader_falls_back_with_warning(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("codex_cli", tmpdir)
            agent = build_reader_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.normalized_payload["execution_mode_requested"], "codex_cli")
            self.assertEqual(output.normalized_payload["execution_mode_used"], "mock")
            self.assertTrue(output.normalized_payload["candidate_papers"])
            self.assertTrue(any("falling back to mock executor" in item.lower() for item in output.warnings))
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_manual_papers_are_normalized_into_candidate_papers(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            contract.metadata["manual_papers"] = [
                {
                    "title": "Manual Paper",
                    "abstract": "uploaded abstract",
                    "source": "manual_upload",
                    "year": 2025,
                    "url": "https://example.com/manual-paper",
                    "file_status": "uploaded",
                    "file_path": str(Path(tmpdir) / "manual-paper.pdf"),
                }
            ]
            agent = build_reader_agent(contract)

            repaired = agent.run(contract)

            self.assertEqual(repaired.validation_status, "succeeded")
            self.assertTrue(repaired.normalized_payload["candidate_papers"])
            self.assertEqual(repaired.normalized_payload["candidate_papers"][0]["file_status"], "uploaded")
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
