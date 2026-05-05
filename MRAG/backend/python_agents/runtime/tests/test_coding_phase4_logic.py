from __future__ import annotations

import shutil
import sys
import tempfile
import unittest
from pathlib import Path

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
RETRIEVAL_MAINLINE_ROOT = Path(__file__).resolve().parents[3] / "python_runners" / "retrieval_mainline"
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))
if str(RETRIEVAL_MAINLINE_ROOT) not in sys.path:
    sys.path.insert(0, str(RETRIEVAL_MAINLINE_ROOT))

from runtime.coding_phase4_agent import CodingPhase4Request  # noqa: E402
from runtime.coding_phase4_logic import build_generation_plan, build_phase4_config, render_method_module  # noqa: E402
from method_registry import load_method  # noqa: E402


def build_request() -> CodingPhase4Request:
    return CodingPhase4Request(
        run_manifest_id="p4run_demo",
        runner_mode="local_dummy",
        max_retry_count=3,
        user_notes="Prefer layout-aware and OCR-robust retrieval.",
        dataset_profile={
            "id": "p4ds_visdom",
            "datasetName": "VisDoM",
            "taskType": "visual_document_retrieval",
            "serverPath": "/home/bzli/mrag/datasets/visdom",
            "officialMetric": "recall@5",
            "knownDifficulties": ["OCR noise", "layout ambiguity"],
            "fileStructureSnapshot": {"file_count": 42},
            "sampleStatistics": {"page_count": 100, "query_count": 12},
        },
        idea={
            "id": "p4idea_demo",
            "title": "Layout-Aware OCR-Robust Page Retrieval",
            "problemDefinition": "Improve page-level retrieval recall.",
            "coreMethod": "Blend layout-aware scoring with OCR-aware lexical matching.",
            "trainingPlan": "Start with retrieval_mainline, then add harder negatives.",
            "modelChanges": ["layout scoring", "ocr robust query expansion"],
            "dataProcessingNeeds": ["page regions", "ocr normalization"],
            "evaluationMetrics": ["recall@1", "mrr"],
        },
        reader_context={
            "task_definition": "Page-level retrieval for visually rich documents.",
            "implementation_constraints": ["page-level first"],
            "relevant_methods_landscape": ["layout-aware retrievers", "hybrid retrieval"],
            "likely_strong_baselines": ["dual encoder", "lexical retrieval"],
            "promising_research_directions": ["hard negative mining", "ocr normalization"],
        },
    )


class CodingPhase4LogicTest(unittest.TestCase):
    def test_method_module_can_be_loaded_via_registry(self) -> None:
        request = build_request()
        plan = build_generation_plan(request, execution_mode_used="mock", response_text="")
        module_content = render_method_module(plan)
        with tempfile.TemporaryDirectory() as temp_dir:
            module_path = Path(temp_dir) / "generated_method.py"
            module_path.write_text(module_content, encoding="utf-8")
            method = load_method(str(module_path))
        self.assertEqual(method.name, plan.method_slug)
        self.assertEqual(method.top_k, 10)
        self.assertTrue(method.query_expansion_terms)
        self.assertGreater(method.title_match_bonus, 0.25)
        self.assertGreater(method.ocr_match_bonus, 0.15)

    def test_phase4_config_contains_repair_and_scoring_contract(self) -> None:
        request = build_request()
        plan = build_generation_plan(request, execution_mode_used="codex_cli", response_text="prefer OCR-robust lexical reranking")
        config = build_phase4_config(request, plan)
        self.assertEqual(config["method_name"], plan.method_slug)
        self.assertEqual(config["method_module_path"], plan.method_relative_path)
        self.assertEqual(config["retry_policy"]["max_retries"], 3)
        self.assertEqual(config["retry_policy"]["max_attempts"], 4)
        self.assertIn("fallback_to_previous_stable_snapshot", config["retry_policy"]["repair_priority"])
        self.assertIn("query_expansion_terms", config["parameters"])
        self.assertIn("scoring_profile", config["parameters"])
        self.assertEqual(config["dataset_adapter"]["metadata"]["retrieval_granularity"], "page")


if __name__ == "__main__":
    unittest.main()
