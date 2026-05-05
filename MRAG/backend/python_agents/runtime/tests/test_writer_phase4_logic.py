from __future__ import annotations

import unittest
import sys
from pathlib import Path

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.writer_phase4_logic import Phase4WriterRequest, build_machine_readable_report, merge_citations, render_human_readable_report  # noqa: E402


def build_request() -> Phase4WriterRequest:
    return Phase4WriterRequest(
        run_manifest_id="p4run_writer_1",
        user_notes="Focus on traceability.",
        dataset_profile={
            "id": "p4ds_visdom",
            "datasetName": "VisDoM",
            "taskType": "page_level_retrieval",
            "officialMetric": "recall@5",
            "officialBaseline": "page lexical baseline",
            "knownDifficulties": ["layout-heavy pages"],
            "sampleStatistics": {"pageCount": 5, "queryCount": 5},
            "fileStructureSnapshot": {"documents": 2, "queries": 5},
            "citation": "VisDoM dataset citation.",
        },
        reader_context={
            "id": "p4ctx_visdom",
            "summary": "Reader identified layout-heavy retrieval challenges and hybrid retrieval opportunities.",
            "taskDefinition": "Retrieve the correct page for each question in visually rich documents.",
            "retrievalFocus": ["page-level retrieval"],
            "structuredContext": {
                "dataset_specific_challenges": ["tables", "captions"],
                "relevant_methods_landscape": ["hybrid lexical retrieval", "layout-aware dense retrieval"],
                "likely_strong_baselines": ["bm25", "hybrid sparse+dense"],
                "common_failure_points": ["ocr noise"],
                "evaluation_caveats": ["page-level only"],
                "implementation_constraints": ["page-level first"],
                "promising_research_directions": ["snippet-level retrieval after page-level stabilization"],
            },
        },
        reader_sources=[
            {
                "id": "p4src_1",
                "title": "VisDoM: Multi-Document QA with Visually Rich Elements Using Multimodal Retrieval-Augmented Generation",
                "authors": ["Author A", "Author B"],
                "venue": "ACL",
                "publicationYear": 2025,
                "sourceType": "conference",
                "sourceUrl": "https://example.org/visdom-paper",
                "qualityTier": "top_conference",
                "metadata": {"doi": "10.1000/visdom.1"},
            },
            {
                "id": "p4src_2",
                "title": "VisDoM: Multi-Document QA with Visually Rich Elements Using Multimodal Retrieval-Augmented Generation",
                "authors": ["Author A", "Author B"],
                "venue": "ACL",
                "publicationYear": 2025,
                "sourceType": "conference",
                "sourceUrl": "https://example.org/visdom-paper-duplicate",
                "qualityTier": "top_conference",
                "metadata": {"doi": "10.1000/visdom.1"},
            },
        ],
        selected_idea={
            "id": "p4idea_1",
            "title": "Layout-Aware Hard Negative Mining",
            "problemDefinition": "Improve recall on visually similar pages.",
            "coreMethod": "Weighted lexical retrieval with layout-aware signals.",
            "differentiators": "Adds title and OCR weighting to page ranking.",
            "dataProcessingNeeds": ["ocr text", "title metadata"],
            "modelChanges": ["weighted scoring profile"],
            "trainingPlan": "Start from retrieval_mainline and add a stronger reranker later.",
            "evaluationMetrics": ["recall@1", "recall@5", "recall@10", "mrr", "ndcg@10"],
            "riskPoints": ["ocr noise"],
            "expectedGains": ["better page recall"],
            "scoreSummary": {"overall": 0.86},
        },
        run_manifest={
            "id": "p4run_writer_1",
            "runnerMode": "local_dummy",
            "status": "succeeded",
            "retryCount": 1,
            "maxRetryCount": 3,
        },
        metrics={
            "primary_metric": "recall@5",
            "values": {"recall@1": 0.6, "recall@5": 0.8, "recall@10": 0.9, "mrr": 0.7, "ndcg@10": 0.76},
        },
        artifact_summary={"metrics_path": "/tmp/metrics.json", "human_report_path": "/tmp/report.md"},
        coding_machine_report={"pipeline": "retrieval_mainline"},
        coding_human_report_excerpt="Baseline retrieval completed with stable metrics.",
    )


class WriterPhase4LogicTests(unittest.TestCase):
    def test_merge_citations_deduplicates_reader_sources(self) -> None:
        request = build_request()
        citations, refs, source_ids = merge_citations(request)
        self.assertEqual(len([item for item in citations if item["source_type"] != "dataset"]), 1)
        self.assertIn("p4src_1", source_ids)
        self.assertGreaterEqual(len(refs), 2)

    def test_machine_report_contains_required_sections(self) -> None:
        request = build_request()
        report = build_machine_readable_report(request, execution_mode_used="mock")
        for key in (
            "dataset",
            "task",
            "reader_context_summary",
            "citations",
            "idea",
            "implementation",
            "run_config",
            "metrics",
            "error_summary",
            "result_analysis",
            "limitations",
            "next_steps",
        ):
            self.assertIn(key, report)
        self.assertEqual(report["dataset"]["name"], "VisDoM")
        self.assertEqual(report["metrics"]["primary_metric"], "recall@5")

    def test_render_human_report_contains_required_headings(self) -> None:
        request = build_request()
        report = build_machine_readable_report(request, execution_mode_used="mock")
        markdown = render_human_readable_report(report)
        for heading in (
            "## 数据集与任务",
            "## 相关工作",
            "## Idea 说明",
            "## 实现方法",
            "## 实验设置",
            "## 结果与分析",
            "## 局限与下一步",
            "## 参考文献",
        ):
            self.assertIn(heading, markdown)


if __name__ == "__main__":
    unittest.main()
