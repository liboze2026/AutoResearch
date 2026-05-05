from __future__ import annotations

import math
from pathlib import Path
from statistics import mean
from typing import Any

from protocol import ExperimentManifest, MetricsPayload, RunConfig, dump_json, load_json
from tools.page_retrieval_dataset import build_qrels_lookup, load_page_retrieval_dataset


PRIMARY_METRICS = ("recall@1", "recall@5", "recall@10", "mrr", "ndcg@10")
SECONDARY_METRICS = ("index_build_ms", "avg_query_latency_ms", "peak_gpu_memory_mb", "failure_rate")


def evaluate_predictions(manifest: ExperimentManifest, config: RunConfig, predictions: dict[str, Any]) -> MetricsPayload:
    dataset_asset = load_json(manifest.dataset_tool_asset_path)
    dataset = load_page_retrieval_dataset(dataset_asset)
    qrels_lookup = build_qrels_lookup(dataset.qrels)
    ranked_predictions = _ranked_predictions(predictions)
    values = {
        "recall@1": round(_recall_at_k(ranked_predictions, qrels_lookup, 1), 4),
        "recall@5": round(_recall_at_k(ranked_predictions, qrels_lookup, 5), 4),
        "recall@10": round(_recall_at_k(ranked_predictions, qrels_lookup, 10), 4),
        "mrr": round(_mean_reciprocal_rank(ranked_predictions, qrels_lookup), 4),
        "ndcg@10": round(_mean_ndcg_at_k(ranked_predictions, qrels_lookup, 10), 4),
        "index_build_ms": round(float(predictions.get("index_stats", {}).get("build_time_ms", 0.0) or 0.0), 4),
        "avg_query_latency_ms": round(_average_query_latency(predictions), 4),
        "peak_gpu_memory_mb": float(predictions.get("runtime_stats", {}).get("max_gpu_memory_mb", 0.0) or 0.0),
        "failure_rate": round(_failure_rate(predictions, len(dataset.queries)), 4),
        "query_count": len(dataset.queries),
        "page_count": len(dataset.pages),
    }
    primary_metric = str(config.evaluate.get("primary_metric", "")).strip() or (config.dataset_adapter.official_metric or "recall@5").strip() or "recall@5"
    metrics = MetricsPayload(
        protocol_version=manifest.protocol_version,
        run_id=manifest.run_id,
        primary_metric=primary_metric,
        values=values,
        status="succeeded" if values["failure_rate"] < 1.0 else "failed",
        retrieval_summary={
            "method_name": predictions.get("method_name", config.method_name),
            "method_branch": config.method_branch,
            "prediction_count": len(ranked_predictions),
            "adapter_type": dataset_asset.get("adapter_type", ""),
            "dataset_family": dataset_asset.get("dataset_family", ""),
            "retrieval_granularity": dataset_asset.get("retrieval_granularity", "page"),
        },
        metadata={
            "runner_mode": config.runner_mode,
            "primary_metrics": list(PRIMARY_METRICS),
            "secondary_metrics": list(SECONDARY_METRICS),
        },
    )
    metrics.validate()
    return metrics


def write_evaluation_assets(manifest: ExperimentManifest, metrics: MetricsPayload) -> None:
    dump_json(manifest.metrics_path, metrics.to_payload())
    dump_json(
        manifest.evaluate_tool_asset_path,
        {
            "tool": "evaluate_tool",
            "primary_metric": metrics.primary_metric,
            "values": dict(metrics.values),
            "status": metrics.status,
            "retrieval_summary": dict(metrics.retrieval_summary),
        },
    )
    dump_json(
        manifest.machine_report_path,
        {
            "run_id": manifest.run_id,
            "status": metrics.status,
            "metrics": metrics.to_payload(),
        },
    )
    human_report = "\n".join(
        [
            "# Phase4 Retrieval Evaluation",
            "",
            f"- Run ID: `{manifest.run_id}`",
            f"- Primary Metric: `{metrics.primary_metric}`",
            f"- Recall@1: `{metrics.values.get('recall@1', 0)}`",
            f"- Recall@5: `{metrics.values.get('recall@5', 0)}`",
            f"- Recall@10: `{metrics.values.get('recall@10', 0)}`",
            f"- MRR: `{metrics.values.get('mrr', 0)}`",
            f"- nDCG@10: `{metrics.values.get('ndcg@10', 0)}`",
            f"- Index Build Time (ms): `{metrics.values.get('index_build_ms', 0)}`",
            f"- Avg Query Latency (ms): `{metrics.values.get('avg_query_latency_ms', 0)}`",
            f"- Failure Rate: `{metrics.values.get('failure_rate', 0)}`",
        ]
    )
    Path(manifest.human_report_path).write_text(human_report + "\n", encoding="utf-8")
    Path(manifest.eval_summary_path).write_text(
        (
            "# Evaluation Summary\n\n"
            f"- Method: `{metrics.retrieval_summary.get('method_name', '')}`\n"
            f"- Recall@1: `{metrics.values.get('recall@1', 0)}`\n"
            f"- Recall@5: `{metrics.values.get('recall@5', 0)}`\n"
            f"- Recall@10: `{metrics.values.get('recall@10', 0)}`\n"
            f"- MRR: `{metrics.values.get('mrr', 0)}`\n"
            f"- nDCG@10: `{metrics.values.get('ndcg@10', 0)}`\n"
        ),
        encoding="utf-8",
    )


def _ranked_predictions(predictions: dict[str, Any]) -> list[dict[str, Any]]:
    items = list(predictions.get("predictions") or [])
    normalized: list[dict[str, Any]] = []
    for item in items:
        query_id = str(item.get("query_id", "")).strip()
        candidates = list(item.get("candidates") or [])
        normalized.append({"query_id": query_id, "candidates": candidates})
    return normalized


def _recall_at_k(predictions: list[dict[str, Any]], qrels_lookup: dict[str, dict[str, float]], k: int) -> float:
    if not predictions:
        return 0.0
    hits = 0
    for item in predictions:
        relevant = {page_id for page_id, relevance in qrels_lookup.get(item["query_id"], {}).items() if relevance > 0}
        retrieved = [str(candidate.get("page_id", "")).strip() for candidate in item["candidates"][:k]]
        if relevant.intersection(retrieved):
            hits += 1
    return hits / max(len(predictions), 1)


def _mean_reciprocal_rank(predictions: list[dict[str, Any]], qrels_lookup: dict[str, dict[str, float]]) -> float:
    reciprocal_ranks: list[float] = []
    for item in predictions:
        relevant = {page_id for page_id, relevance in qrels_lookup.get(item["query_id"], {}).items() if relevance > 0}
        reciprocal = 0.0
        for rank, candidate in enumerate(item["candidates"], start=1):
            page_id = str(candidate.get("page_id", "")).strip()
            if page_id in relevant:
                reciprocal = 1.0 / rank
                break
        reciprocal_ranks.append(reciprocal)
    return sum(reciprocal_ranks) / max(len(reciprocal_ranks), 1)


def _mean_ndcg_at_k(predictions: list[dict[str, Any]], qrels_lookup: dict[str, dict[str, float]], k: int) -> float:
    scores: list[float] = []
    for item in predictions:
        relevance_lookup = qrels_lookup.get(item["query_id"], {})
        dcg = 0.0
        for rank, candidate in enumerate(item["candidates"][:k], start=1):
            relevance = float(relevance_lookup.get(str(candidate.get("page_id", "")).strip(), 0.0))
            if relevance <= 0:
                continue
            dcg += (2**relevance - 1.0) / math.log2(rank + 1)
        ideal_relevances = sorted((float(value) for value in relevance_lookup.values()), reverse=True)[:k]
        idcg = 0.0
        for rank, relevance in enumerate(ideal_relevances, start=1):
            if relevance <= 0:
                continue
            idcg += (2**relevance - 1.0) / math.log2(rank + 1)
        scores.append(dcg / idcg if idcg > 0 else 0.0)
    return sum(scores) / max(len(scores), 1)


def _average_query_latency(predictions: dict[str, Any]) -> float:
    timings = list(predictions.get("query_timings_ms") or [])
    latencies = [float(item.get("latency_ms", 0.0) or 0.0) for item in timings]
    return mean(latencies) if latencies else 0.0


def _failure_rate(predictions: dict[str, Any], total_queries: int) -> float:
    failures = list(predictions.get("failures") or [])
    return len(failures) / max(total_queries, 1)
