from __future__ import annotations

import sys
import unittest
from pathlib import Path

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.idea_phase4_logic import (  # noqa: E402
    Phase4IdeaRequest,
    build_top_recommendations,
    classify_failure_feedback,
    generate_structured_ideas,
    score_and_rank_ideas,
)


def build_metadata() -> dict:
    return {
        "dataset_profile": {
            "id": "p4ds_visdom",
            "datasetName": "VisDoM",
            "taskType": "retrieval",
            "modalityComposition": ["image", "text"],
            "officialMetric": "Recall@10",
            "knownDifficulties": ["OCR noise", "layout ambiguity", "template overlap"],
        },
        "reader_context": {
            "task_definition": "Improve page-level retrieval recall for visually rich documents.",
            "dataset_specific_challenges": ["OCR noise on dense pages", "Boilerplate-heavy layouts"],
            "relevant_methods_landscape": ["late interaction reranking", "layout-aware retrievers"],
            "likely_strong_baselines": ["BM25 over OCR", "dual-encoder page retriever"],
            "common_failure_points": ["Near-duplicate pages confuse the retriever"],
            "evaluation_caveats": ["Recall@10 is the primary official metric"],
            "implementation_constraints": ["Keep the first release page-level only"],
            "promising_research_directions": ["hard negative mining", "query-conditioned chunking"],
            "citation_metadata": [{"title": "VisDoM: Multi-Document QA with Visually Rich Elements Using Multimodal Retrieval-Augmented Generation"}],
        },
        "user_notes": "Focus on engineering-feasible page retrieval improvements.",
    }


class IdeaPhase4LogicTests(unittest.TestCase):
    def test_generate_ten_structured_ideas_and_rank_top3(self) -> None:
        request = Phase4IdeaRequest.from_metadata(build_metadata())

        ideas = score_and_rank_ideas(generate_structured_ideas(request), request)
        top3 = build_top_recommendations(ideas)

        self.assertEqual(len(ideas), 10)
        self.assertEqual(len(top3), 3)
        self.assertGreaterEqual(top3[0]["overallScore"], top3[1]["overallScore"])
        self.assertGreaterEqual(top3[1]["overallScore"], top3[2]["overallScore"])
        first = ideas[0].to_payload()
        for field_name in (
            "problem_definition",
            "core_method",
            "differentiators",
            "data_processing_needs",
            "model_changes",
            "training_plan",
            "evaluation_metrics",
            "risk_points",
            "expected_gains",
            "score",
            "score_summary",
        ):
            self.assertIn(field_name, first)
        self.assertTrue(ideas[0].score_summary["recommended"])

    def test_revision_generation_preserves_lineage_and_failure_feedback(self) -> None:
        metadata = build_metadata()
        metadata.update(
            {
                "generation_mode": "revision",
                "target_count": 3,
                "source_idea_id": "p4idea_src_001",
                "source_idea": {
                    "title": "VisDoM: Layout-Aware Hard Negative Mining",
                    "problem_definition": "Improve page-level retrieval recall.",
                    "core_method": "Use layout-aware hard negatives.",
                    "differentiators": "Targets template confusion.",
                    "data_processing_needs": ["Persist page neighborhoods"],
                    "model_changes": ["Add layout token projection"],
                    "training_plan": "Curriculum over hard negatives.",
                    "evaluation_metrics": ["Recall@10"],
                    "risk_points": ["Potential instability"],
                    "expected_gains": ["Higher Recall@10"],
                },
                "failure_feedback": {"error": "CUDA OOM on multi-scale renders"},
                "last_failure_run_id": "p4run_fail_001",
            }
        )
        request = Phase4IdeaRequest.from_metadata(metadata)

        ideas = score_and_rank_ideas(generate_structured_ideas(request), request)

        self.assertEqual(len(ideas), 3)
        self.assertEqual(classify_failure_feedback(request.failure_feedback), "resource")
        for item in ideas:
            payload = item.to_payload()
            self.assertEqual(payload["revision_of_id"], "p4idea_src_001")
            self.assertEqual(payload["last_failure_run_id"], "p4run_fail_001")
            self.assertEqual(payload["failure_feedback"]["error"], "CUDA OOM on multi-scale renders")


if __name__ == "__main__":
    unittest.main()
