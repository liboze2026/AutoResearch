from __future__ import annotations

import math
from collections import Counter
from time import perf_counter
from typing import Any

from methods.base import RetrievalMethod
from tools.page_retrieval_dataset import build_page_text, get_query_text, load_page_retrieval_dataset, tokenize_text


class PageLexicalRetrievalMethod(RetrievalMethod):
    def __init__(
        self,
        name: str,
        method_tags: list[str] | None = None,
        score_bias: float = 0.0,
        retrieval_notes: list[str] | None = None,
        top_k: int = 10,
        query_expansion_terms: list[str] | None = None,
        title_match_bonus: float = 0.25,
        ocr_match_bonus: float = 0.15,
        section_match_bonus: float = 0.12,
        exact_phrase_bonus: float = 0.10,
    ) -> None:
        super().__init__(
            name=name,
            method_tags=list(method_tags or []),
            score_bias=score_bias,
            retrieval_notes=list(retrieval_notes or []),
        )
        self.top_k = max(int(top_k), 1)
        self.query_expansion_terms = [str(item).strip().lower() for item in list(query_expansion_terms or []) if str(item).strip()]
        self.title_match_bonus = float(title_match_bonus)
        self.ocr_match_bonus = float(ocr_match_bonus)
        self.section_match_bonus = float(section_match_bonus)
        self.exact_phrase_bonus = float(exact_phrase_bonus)

    def run(self, manifest: Any, config: Any, dataset_asset: dict[str, Any]) -> dict[str, Any]:
        dataset = load_page_retrieval_dataset(dataset_asset)
        pages = dataset.pages
        queries = dataset.queries
        build_started = perf_counter()
        index = _build_index(pages)
        build_time_ms = round((perf_counter() - build_started) * 1000, 3)
        query_timings_ms: list[dict[str, Any]] = []
        failures: list[dict[str, Any]] = []
        predictions: list[dict[str, Any]] = []
        for query in queries:
            query_started = perf_counter()
            query_id = str(query.get("query_id", "")).strip()
            query_text = get_query_text(query)
            try:
                candidates = _rank_pages(
                    index,
                    pages,
                    query_text,
                    self.top_k,
                    self.score_bias,
                    self.query_expansion_terms,
                    self.title_match_bonus,
                    self.ocr_match_bonus,
                    self.section_match_bonus,
                    self.exact_phrase_bonus,
                )
                predictions.append(
                    {
                        "query_id": query_id,
                        "query": query_text,
                        "split": query.get("split", "unknown"),
                        "candidates": candidates,
                    }
                )
            except Exception as exc:  # pragma: no cover
                failures.append({"query_id": query_id, "error": str(exc)})
                predictions.append({"query_id": query_id, "query": query_text, "split": query.get("split", "unknown"), "candidates": []})
            query_timings_ms.append({"query_id": query_id, "latency_ms": round((perf_counter() - query_started) * 1000, 3)})
        return {
            "method_name": self.name,
            "method_tags": list(self.method_tags),
            "retrieval_notes": list(self.retrieval_notes),
            "granularity": "page",
            "index_stats": {
                "build_time_ms": build_time_ms,
                "document_count": len(pages),
                "vocabulary_size": len(index["idf"]),
                "average_document_length": round(index["average_length"], 3),
            },
            "runtime_stats": {
                "query_count": len(queries),
                "failure_count": len(failures),
                "max_gpu_memory_mb": 0,
            },
            "query_timings_ms": query_timings_ms,
            "predictions": predictions,
            "failures": failures,
        }


def _build_index(pages: list[dict[str, Any]]) -> dict[str, Any]:
    doc_tokens: dict[str, list[str]] = {}
    token_counts: dict[str, Counter[str]] = {}
    document_frequency: Counter[str] = Counter()
    total_tokens = 0
    for page in pages:
        page_id = str(page.get("page_id", "")).strip()
        tokens = tokenize_text(build_page_text(page))
        if not tokens:
            tokens = [page_id]
        doc_tokens[page_id] = tokens
        counts = Counter(tokens)
        token_counts[page_id] = counts
        total_tokens += len(tokens)
        document_frequency.update(counts.keys())
    document_count = max(len(pages), 1)
    idf = {
        token: math.log((document_count + 1) / (frequency + 1)) + 1.0
        for token, frequency in document_frequency.items()
    }
    return {
        "doc_tokens": doc_tokens,
        "token_counts": token_counts,
        "idf": idf,
        "average_length": total_tokens / document_count,
    }


def _rank_pages(
    index: dict[str, Any],
    pages: list[dict[str, Any]],
    query_text: str,
    top_k: int,
    score_bias: float,
    query_expansion_terms: list[str],
    title_match_bonus: float,
    ocr_match_bonus: float,
    section_match_bonus: float,
    exact_phrase_bonus: float,
) -> list[dict[str, Any]]:
    query_tokens = tokenize_text(query_text)
    for item in query_expansion_terms:
        query_tokens.extend(tokenize_text(item))
    if not query_tokens:
        query_tokens = ["empty"]
    query_token_set = set(query_tokens)
    lowered_query = query_text.lower()
    scored: list[tuple[float, dict[str, Any]]] = []
    average_length = max(float(index.get("average_length", 1.0)), 1.0)
    idf = dict(index.get("idf", {}))
    token_counts: dict[str, Counter[str]] = index.get("token_counts", {})
    for page in pages:
        page_id = str(page.get("page_id", "")).strip()
        counts = token_counts.get(page_id, Counter())
        doc_length = max(sum(counts.values()), 1)
        score = 0.0
        for token in query_tokens:
            tf = float(counts.get(token, 0))
            if tf <= 0:
                continue
            token_idf = idf.get(token, 1.0)
            score += token_idf * ((tf * 2.2) / (tf + 1.2 * (1.0 - 0.75 + 0.75 * doc_length / average_length)))
        title_tokens = set(tokenize_text(str(page.get("title", "") or "")))
        if title_tokens.intersection(query_token_set):
            score += title_match_bonus
        ocr_tokens = set(tokenize_text(str(page.get("ocr_text", "") or "")))
        if ocr_tokens.intersection(query_token_set):
            score += ocr_match_bonus
        section_tokens = set()
        for section_title in list(page.get("section_titles") or []):
            section_tokens.update(tokenize_text(str(section_title)))
        if section_tokens.intersection(query_token_set):
            score += section_match_bonus
        page_text = build_page_text(page).lower()
        if lowered_query and lowered_query in page_text:
            score += exact_phrase_bonus
        score += score_bias
        scored.append((round(score, 6), page))
    scored.sort(key=lambda item: (-item[0], str(item[1].get("page_id", ""))))
    candidates: list[dict[str, Any]] = []
    for rank, (score, page) in enumerate(scored[:top_k], start=1):
        candidates.append(
            {
                "page_id": page.get("page_id", ""),
                "score": round(score, 4),
                "rank": rank,
                "title": page.get("title", ""),
                "snippet": build_page_text(page)[:220],
            }
        )
    return candidates
