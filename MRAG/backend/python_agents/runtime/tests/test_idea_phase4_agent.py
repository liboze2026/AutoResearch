from __future__ import annotations

import shutil
import sys
import unittest
import uuid
from pathlib import Path

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.contract import AgentRuntimeInput  # noqa: E402
from runtime.idea_phase4_agent import IDEA_PHASE4_SCHEMA_REF, build_idea_phase4_agent  # noqa: E402

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"idea-phase4-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(workspace_dir: str, generation_mode: str = "new") -> AgentRuntimeInput:
    metadata = {
        "dataset_profile": {
            "id": "p4ds_visdom",
            "datasetName": "VisDoM",
            "taskType": "retrieval",
            "modalityComposition": ["image", "text"],
            "officialMetric": "Recall@10",
            "knownDifficulties": ["OCR noise", "layout ambiguity"],
        },
        "reader_context": {
            "task_definition": "Improve page-level retrieval recall for visually rich documents.",
            "dataset_specific_challenges": ["OCR noise on dense pages", "near-duplicate templates"],
            "relevant_methods_landscape": ["late interaction reranking", "layout-aware retrievers"],
            "likely_strong_baselines": ["dual-encoder page retriever"],
            "common_failure_points": ["Near-duplicate pages cause confusion"],
            "evaluation_caveats": ["Recall@10 is the official target"],
            "implementation_constraints": ["Keep the first phase page-level only"],
            "promising_research_directions": ["hard negative mining", "query-conditioned chunking"],
            "citation_metadata": [{"title": "VisDoM"}],
        },
        "user_notes": "Prefer concrete engineering plans.",
        "generation_mode": generation_mode,
    }
    if generation_mode == "revision":
        metadata.update(
            {
                "source_idea_id": "p4idea_src_001",
                "source_idea": {
                    "title": "VisDoM: Layout-Aware Hard Negative Mining",
                    "problem_definition": "Improve page retrieval recall.",
                    "core_method": "Use layout-aware hard negatives.",
                    "differentiators": "Targets template overlap.",
                    "data_processing_needs": ["Persist page neighborhoods"],
                    "model_changes": ["Add layout token fusion"],
                    "training_plan": "Curriculum learning.",
                    "evaluation_metrics": ["Recall@10"],
                    "risk_points": ["Potential instability"],
                    "expected_gains": ["Higher Recall@10"],
                },
                "failure_feedback": {"error": "low recall on near-duplicate pages"},
                "last_failure_run_id": "p4run_fail_001",
                "target_count": 3,
            }
        )
    return AgentRuntimeInput(
        job_id="idea-phase4-job-001",
        agent_type="idea_phase4",
        execution_mode="api",
        model_provider="codex",
        model_name="phase4-idea-test",
        prompt_version="v1",
        input_refs=[],
        output_schema_ref=IDEA_PHASE4_SCHEMA_REF,
        skill_refs=[],
        tool_refs=[],
        memory_refs=[],
        workspace_dir=workspace_dir,
        metadata=metadata,
    )


class IdeaPhase4AgentTests(unittest.TestCase):
    def test_new_generation_returns_ranked_ideas(self) -> None:
        workspace = make_workspace()
        try:
            contract = build_contract(workspace)
            agent = build_idea_phase4_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertEqual(output.normalized_payload["generation_mode"], "new")
            self.assertEqual(len(output.normalized_payload["ideas"]), 10)
            self.assertEqual(len(output.normalized_payload["top_recommendations"]), 3)
            self.assertTrue(output.artifact_manifest)
        finally:
            shutil.rmtree(workspace, ignore_errors=True)

    def test_revision_generation_returns_revision_candidates(self) -> None:
        workspace = make_workspace()
        try:
            contract = build_contract(workspace, generation_mode="revision")
            agent = build_idea_phase4_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertEqual(output.normalized_payload["generation_mode"], "revision")
            self.assertEqual(len(output.normalized_payload["ideas"]), 3)
            self.assertEqual(output.normalized_payload["ideas"][0]["revision_of_id"], "p4idea_src_001")
        finally:
            shutil.rmtree(workspace, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
