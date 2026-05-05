from __future__ import annotations

from typing import Any

from .base import RetrievalMethod


class DummyRetrievalMethod(RetrievalMethod):
    def run(self, manifest: Any, config: Any, dataset_asset: dict[str, Any]) -> dict[str, Any]:
        dataset_name = dataset_asset.get("dataset_name", "dataset")
        task_type = dataset_asset.get("task_type", "retrieval")
        note_bias = min(len(self.method_tags) * 0.01, 0.08)
        base_score = round(0.62 + self.score_bias + note_bias, 4)
        predictions = []
        queries = [
            f"{dataset_name} page-level retrieval baseline",
            f"{dataset_name} layout-aware hard negative analysis",
            f"{task_type} retrieval stress case",
        ]
        for index, query in enumerate(queries, start=1):
            candidates = []
            for rank in range(1, 4):
                score = round(max(base_score - (rank - 1) * 0.07 - index * 0.01, 0.05), 4)
                candidates.append(
                    {
                        "page_id": f"page_{index}_{rank}",
                        "score": score,
                        "evidence": f"{self.name} prioritizes page-level evidence with tags {', '.join(self.method_tags[:3]) or 'default'}.",
                    }
                )
            predictions.append({"query": query, "candidates": candidates})
        return {
            "method_name": self.name,
            "method_tags": list(self.method_tags),
            "retrieval_notes": list(self.retrieval_notes),
            "predictions": predictions,
        }
