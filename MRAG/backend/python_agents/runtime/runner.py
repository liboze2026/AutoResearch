#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

try:
    from .base import BaseAgent, BaseRepairer, BaseValidator
    from .coding_agent import build_coding_agent
    from .coding_phase4_agent import build_coding_phase4_agent
    from .contract import AgentArtifactManifestItem, AgentRuntimeInput
    from .dataset_agent import build_dataset_agent
    from .executors import ApiExecutor, CodexCLIExecutor, MockExecutor
    from .idea_agent import build_idea_agent
    from .idea_phase4_agent import build_idea_phase4_agent
    from .insight_agent import build_insight_agent
    from .planner_agent import build_planner_agent
    from .reader_agent import build_reader_agent
    from .reader_phase4_agent import build_reader_phase4_agent
    from .writer_agent import build_writer_agent
    from .writer_phase4_agent import build_writer_phase4_agent
except ImportError:  # pragma: no cover - supports direct script execution
    from base import BaseAgent, BaseRepairer, BaseValidator
    from coding_agent import build_coding_agent
    from coding_phase4_agent import build_coding_phase4_agent
    from contract import AgentArtifactManifestItem, AgentRuntimeInput
    from dataset_agent import build_dataset_agent
    from executors import ApiExecutor, CodexCLIExecutor, MockExecutor
    from idea_agent import build_idea_agent
    from idea_phase4_agent import build_idea_phase4_agent
    from insight_agent import build_insight_agent
    from planner_agent import build_planner_agent
    from reader_agent import build_reader_agent
    from reader_phase4_agent import build_reader_phase4_agent
    from writer_agent import build_writer_agent
    from writer_phase4_agent import build_writer_phase4_agent


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Stage3 agent runtime skeleton")
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=True)
    return parser.parse_args()


def load_contract(path: Path) -> AgentRuntimeInput:
    payload = json.loads(path.read_text(encoding="utf-8-sig"))
    return AgentRuntimeInput.from_dict(payload)


def select_executor(contract: AgentRuntimeInput):
    if contract.execution_mode == "api":
        return ApiExecutor()
    if contract.execution_mode == "codex_cli":
        return CodexCLIExecutor()
    return MockExecutor()


def build_agent(contract: AgentRuntimeInput):
    if contract.agent_type == "reader_phase4":
        return build_reader_phase4_agent(contract)
    if contract.agent_type == "reader":
        return build_reader_agent(contract)
    if contract.agent_type == "insight":
        return build_insight_agent(contract)
    if contract.agent_type == "dataset":
        return build_dataset_agent(contract)
    if contract.agent_type == "coding":
        return build_coding_agent(contract)
    if contract.agent_type == "coding_phase4":
        return build_coding_phase4_agent(contract)
    if contract.agent_type == "idea_phase4":
        return build_idea_phase4_agent(contract)
    if contract.agent_type == "idea_generator":
        return build_idea_agent(contract)
    if contract.agent_type == "planner":
        return build_planner_agent(contract)
    if contract.agent_type == "writer":
        return build_writer_agent(contract)
    if contract.agent_type == "writer_phase4":
        return build_writer_phase4_agent(contract)
    return BaseAgent(executor=select_executor(contract), validator=BaseValidator(), repairer=BaseRepairer())


def main() -> int:
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8", errors="replace")
    if hasattr(sys.stderr, "reconfigure"):
        sys.stderr.reconfigure(encoding="utf-8", errors="replace")

    args = parse_args()
    input_path = Path(args.input)
    output_path = Path(args.output)
    contract = load_contract(input_path)

    agent = build_agent(contract)
    result = agent.run(contract)
    result.artifact_manifest.extend(
        [
            AgentArtifactManifestItem(
                artifact_type="input_contract",
                name=input_path.name,
                file_path=str(input_path),
                metadata={"role": "runtime_input"},
            ),
            AgentArtifactManifestItem(
                artifact_type="output_contract",
                name=output_path.name,
                file_path=str(output_path),
                metadata={"role": "runtime_output"},
            ),
        ]
    )

    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(result.to_dict(), indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(json.dumps(result.to_dict(), ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
