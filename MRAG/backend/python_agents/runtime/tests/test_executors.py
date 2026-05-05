from __future__ import annotations

import sys
import unittest
from pathlib import Path
import shutil
import uuid

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.base import BaseAgent, BaseRepairer, BaseValidator
from runtime.config import RuntimeConfigLoader
from runtime.contract import AgentRuntimeInput
from runtime.executors import ApiExecutor, CodexCLIExecutor, MockExecutor

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"test-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(execution_mode: str, workspace_dir: str) -> AgentRuntimeInput:
    return AgentRuntimeInput(
        job_id="job-test-001",
        agent_type="reader",
        execution_mode=execution_mode,
        model_provider="codex",
        model_name="stage3-test-model",
        prompt_version="v1",
        input_refs=[],
        output_schema_ref="schemas/test-output-v1.json",
        skill_refs=["skills/test"],
        tool_refs=["tools/test"],
        memory_refs=["memory/test"],
        workspace_dir=workspace_dir,
        metadata={"source": "unit-test"},
    )


class ExecutorTests(unittest.TestCase):
    def test_mock_executor_returns_stable_output(self) -> None:
        tmpdir = make_workspace()
        try:
            contract = build_contract("mock", tmpdir)
            agent = BaseAgent(MockExecutor(), BaseValidator(), BaseRepairer())

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.normalized_payload["execution_mode_used"], "mock")
            self.assertEqual(output.normalized_payload["response_json"]["result"], "stable_mock_response")
            self.assertEqual(output.tool_usages[0].tool_ref, "mock_executor")
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_codex_cli_missing_binary_falls_back_to_mock(self) -> None:
        tmpdir = make_workspace()
        try:
            loader = RuntimeConfigLoader(project_root=Path(tmpdir), environ={"CODEX_CLI_BIN": "definitely-missing-codex-cli"})
            contract = build_contract("codex_cli", tmpdir)
            agent = BaseAgent(CodexCLIExecutor(config_loader=loader), BaseValidator(), BaseRepairer())

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.normalized_payload["execution_mode_requested"], "codex_cli")
            self.assertEqual(output.normalized_payload["execution_mode_used"], "mock")
            self.assertIn("fallback", output.tool_usages[0].status)
            self.assertTrue(any("falling back to mock executor" in item.lower() for item in output.warnings))
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_codex_cli_prepare_request_supports_prompt_text_and_project_root_placeholders(self) -> None:
        tmpdir = make_workspace()
        try:
            loader = RuntimeConfigLoader(
                project_root=Path(tmpdir),
                environ={
                    "CODEX_CLI_BIN": "codex",
                    "CODEX_CLI_ARGS_JSON": '["exec","-C","{project_root}","--output-last-message","{output_file}","{prompt_text}"]',
                },
            )
            contract = build_contract("codex_cli", tmpdir)
            executor = CodexCLIExecutor(config_loader=loader)

            prepared = executor.prepare_request(contract)
            expected_output = str(Path(tmpdir) / "codex_cli" / "response.txt")

            self.assertEqual(prepared["args"][0], "exec")
            self.assertEqual(prepared["args"][2], tmpdir)
            self.assertEqual(prepared["args"][4], expected_output)
            self.assertIn("job_id: job-test-001", prepared["args"][5])
            self.assertTrue(prepared["uses_prompt_text_arg"])
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)

    def test_api_executor_reports_missing_configuration_sources(self) -> None:
        tmpdir = make_workspace()
        try:
            loader = RuntimeConfigLoader(project_root=Path(tmpdir), environ={})
            contract = build_contract("api", tmpdir)
            agent = BaseAgent(ApiExecutor(config_loader=loader), BaseValidator(), BaseRepairer())

            output = agent.run(contract)

            self.assertEqual(output.status, "failed")
            self.assertIn(".env", output.error_message)
            self.assertIn("runtime config file", output.error_message)
            self.assertIn("database config table hook", output.error_message)
        finally:
            shutil.rmtree(tmpdir, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
