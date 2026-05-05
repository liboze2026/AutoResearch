from __future__ import annotations

import shutil
import sys
import uuid
import unittest
from pathlib import Path

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.base import BaseAgent, BaseExecutor, BaseRepairer, BaseValidator
from runtime.contract import AgentRuntimeInput, AgentRuntimeOutput
from runtime.reader_agent import build_reader_agent

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"validation-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(workspace_dir: str) -> AgentRuntimeInput:
    return AgentRuntimeInput(
        job_id="job-validation-001",
        agent_type="reader",
        execution_mode="mock",
        model_provider="codex",
        model_name="validation-model",
        prompt_version="v1",
        input_refs=[],
        output_schema_ref="schemas/reader-output-v1.json",
        skill_refs=[],
        tool_refs=[],
        memory_refs=[],
        workspace_dir=workspace_dir,
        metadata={"source": "validation-test"},
    )


class ValidExecutor(BaseExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict:
        return {}

    def execute(self, prepared_request: dict, contract: AgentRuntimeInput) -> dict:
        return {}

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict,
        execution_result: dict,
        collected_response: dict,
    ) -> AgentRuntimeOutput:
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload={
                "job_id": contract.job_id,
                "agent_type": contract.agent_type,
                "execution_mode_requested": contract.execution_mode,
                "execution_mode_used": contract.execution_mode,
                "model_provider": contract.model_provider,
                "model_name": contract.model_name,
                "prompt_version": contract.prompt_version,
                "output_schema_ref": contract.output_schema_ref,
                "workspace_dir": contract.workspace_dir,
                "summary": "Valid structured output.",
                "items": [{"id": "a"}],
                "candidate_papers": [
                    {
                        "title": "Valid Reader Paper",
                        "abstract": "Valid abstract",
                        "source": "arxiv",
                        "year": 2026,
                        "url": "https://arxiv.org/abs/2603.0001",
                        "file_status": "metadata_only",
                    }
                ],
                "data": {"score": 1.0},
                "metadata": {"source": "valid-executor"},
            },
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[],
            warnings=[],
            error_message="",
        )


class RepairableExecutor(BaseExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict:
        return {}

    def execute(self, prepared_request: dict, contract: AgentRuntimeInput) -> dict:
        return {}

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict,
        execution_result: dict,
        collected_response: dict,
    ) -> AgentRuntimeOutput:
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload={
                "job_id": contract.job_id,
                "agent_type": contract.agent_type,
                "execution_mode_requested": contract.execution_mode,
                "execution_mode_used": contract.execution_mode,
                "model_provider": contract.model_provider,
                "model_name": contract.model_name,
                "prompt_version": contract.prompt_version,
                "output_schema_ref": contract.output_schema_ref,
                "workspace_dir": contract.workspace_dir,
                "message": "Recovered from incomplete upstream format.",
                "result": "```json\n{\"candidate_papers\": [{\"title\": \"Recovered Paper\", \"abstract\": \"Recovered abstract\", \"source\": \"conference\", \"year\": 2025, \"url\": \"https://example.com/recovered\", \"file_status\": \"metadata_only\"}], \"score\": 0.82}\n```",
                "metadata": "legacy-meta",
            },
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[],
            warnings=[],
            error_message="",
        )


class BrokenExecutor(BaseExecutor):
    def prepare_request(self, contract: AgentRuntimeInput) -> dict:
        return {}

    def execute(self, prepared_request: dict, contract: AgentRuntimeInput) -> dict:
        return {}

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict,
        execution_result: dict,
        collected_response: dict,
    ) -> AgentRuntimeOutput:
        return AgentRuntimeOutput(
            status="succeeded",
            normalized_payload={"summary": 123, "items": object(), "data": object()},
            artifact_manifest=[],
            repair_actions=[],
            tool_usages=[],
            warnings=[],
            error_message="",
        )


class ValidationTests(unittest.TestCase):
    def test_valid_output_passes_without_repair(self) -> None:
        workspace = make_workspace()
        try:
            contract = build_contract(workspace)
            output = BaseAgent(ValidExecutor(), BaseValidator(), BaseRepairer()).run(contract)
            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertEqual(output.repair_status, "not_needed")
            self.assertEqual(output.validation_errors, [])
        finally:
            shutil.rmtree(workspace, ignore_errors=True)

    def test_missing_fields_are_repaired(self) -> None:
        workspace = make_workspace()
        try:
            contract = build_contract(workspace)
            agent = build_reader_agent(contract)
            agent.executor = RepairableExecutor()
            output = agent.run(contract)
            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertEqual(output.repair_status, "succeeded")
            self.assertEqual(output.normalized_payload["summary"], "Recovered from incomplete upstream format.")
            self.assertEqual(output.normalized_payload["items"][0]["title"], "Recovered Paper")
            self.assertEqual(output.normalized_payload["candidate_papers"][0]["title"], "Recovered Paper")
            self.assertEqual(output.normalized_payload["data"]["score"], 0.82)
            self.assertIsInstance(output.normalized_payload["metadata"], dict)
            self.assertTrue(any(item.action == "rename_field" for item in output.repair_actions))
            self.assertTrue(any(item.action == "extract_markdown_json" for item in output.repair_actions))
        finally:
            shutil.rmtree(workspace, ignore_errors=True)

    def test_unrepairable_output_fails_with_reason(self) -> None:
        workspace = make_workspace()
        try:
            contract = build_contract(workspace)
            agent = build_reader_agent(contract)
            agent.executor = BrokenExecutor()
            output = agent.run(contract)
            self.assertEqual(output.status, "failed")
            self.assertEqual(output.validation_status, "failed")
            self.assertEqual(output.repair_status, "failed")
            self.assertTrue(output.validation_errors)
            self.assertIn("normalized_payload.data", output.error_message)
        finally:
            shutil.rmtree(workspace, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
