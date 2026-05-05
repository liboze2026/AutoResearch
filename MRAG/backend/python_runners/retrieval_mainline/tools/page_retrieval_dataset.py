from __future__ import annotations

import json
from collections import Counter, defaultdict
from dataclasses import dataclass
from pathlib import Path
from statistics import mean
from typing import Any

from protocol import DatasetAdapterContract


DEFAULT_VISDOM_FIXTURE = Path(__file__).resolve().parents[1] / "tests" / "fixtures" / "visdom_like_dataset"
PAGE_FILE_CANDIDATES = (
    "pages.jsonl",
    "page_corpus.jsonl",
    "documents.jsonl",
    "corpus.jsonl",
    "pages.json",
    "documents.json",
)
QUERY_FILE_CANDIDATES = (
    "queries.jsonl",
    "questions.jsonl",
    "query_set.jsonl",
    "queries.json",
)
QREL_FILE_CANDIDATES = (
    "qrels.jsonl",
    "relevance.jsonl",
    "annotations.jsonl",
    "qrels.json",
)
DEFAULT_PAGE_TEXT_FIELDS = ("title", "text", "ocr_text", "caption", "section_titles")
DEFAULT_QUERY_TEXT_FIELDS = ("query", "question", "text")


@dataclass
class PageRetrievalDataset:
    pages: list[dict[str, Any]]
    queries: list[dict[str, Any]]
    qrels: list[dict[str, Any]]
    paths: dict[str, str]


class BasePageRetrievalAdapter:
    adapter_type = "generic_page_retrieval"
    dataset_family = "visual_document_retrieval"
    retrieval_granularity = "page"

    def __init__(self, contract: DatasetAdapterContract) -> None:
        self.contract = contract
        self.metadata = dict(contract.metadata or {})
        self.root = self._resolve_root()

    def analyze(self) -> dict[str, Any]:
        dataset = self.load_dataset()
        snapshot = build_file_structure_snapshot(self.root)
        statistics = build_sample_statistics(dataset.pages, dataset.queries, dataset.qrels)
        payload = {
            "dataset_profile_id": self.contract.dataset_profile_id,
            "dataset_name": self.contract.dataset_name,
            "dataset_family": self.dataset_family,
            "task_type": self.contract.task_type,
            "official_metric": self.contract.official_metric,
            "run_id": self.metadata.get("run_id", ""),
            "adapter_type": self.adapter_type,
            "retrieval_granularity": self.retrieval_granularity,
            "resolved_root": str(self.root),
            "paths": dict(dataset.paths),
            "text_fields": list(DEFAULT_PAGE_TEXT_FIELDS),
            "query_fields": list(DEFAULT_QUERY_TEXT_FIELDS),
            "splits": list(self.contract.splits),
            "metadata": dict(self.metadata),
            "file_structure_snapshot": snapshot,
            "sample_statistics": statistics,
            "sample_preview": {
                "pages": dataset.pages[:2],
                "queries": dataset.queries[:2],
                "qrels": dataset.qrels[:4],
            },
        }
        return payload

    def load_dataset(self) -> PageRetrievalDataset:
        pages_path = self._resolve_data_file("pages_path", PAGE_FILE_CANDIDATES)
        queries_path = self._resolve_data_file("queries_path", QUERY_FILE_CANDIDATES)
        qrels_path = self._resolve_data_file("qrels_path", QREL_FILE_CANDIDATES)
        pages = normalize_pages(load_records(pages_path, "pages"))
        queries = normalize_queries(load_records(queries_path, "queries"))
        qrels = normalize_qrels(load_records(qrels_path, "qrels"))
        return PageRetrievalDataset(
            pages=pages,
            queries=queries,
            qrels=qrels,
            paths={
                "pages_path": str(pages_path),
                "queries_path": str(queries_path),
                "qrels_path": str(qrels_path),
            },
        )

    def _resolve_root(self) -> Path:
        candidates = [
            self.metadata.get("resolved_local_path"),
            self.metadata.get("fixture_path"),
            self.contract.server_path,
        ]
        for candidate in candidates:
            if not candidate:
                continue
            path = Path(str(candidate))
            if path.exists():
                return path
        if looks_like_visdom(self.contract.dataset_name, self.contract.server_path) and DEFAULT_VISDOM_FIXTURE.exists():
            return DEFAULT_VISDOM_FIXTURE
        return Path(self.contract.server_path)

    def _resolve_data_file(self, metadata_key: str, candidates: tuple[str, ...]) -> Path:
        explicit = str(self.metadata.get(metadata_key, "")).strip()
        if explicit:
            explicit_path = Path(explicit)
            if explicit_path.exists():
                return explicit_path
            relative_path = self.root / explicit
            if relative_path.exists():
                return relative_path
        for candidate in candidates:
            for path in sorted(self.root.rglob(candidate)):
                if path.is_file():
                    return path
        raise FileNotFoundError(f"could not resolve {metadata_key} under {self.root}")


class VisDoMPageRetrievalAdapter(BasePageRetrievalAdapter):
    adapter_type = "visdom_page_retrieval"
    dataset_family = "visdom"


def build_page_retrieval_adapter(contract: DatasetAdapterContract) -> BasePageRetrievalAdapter:
    adapter_type = str((contract.metadata or {}).get("adapter_type", "")).strip().lower()
    if adapter_type == "visdom_page_retrieval" or looks_like_visdom(contract.dataset_name, contract.server_path):
        return VisDoMPageRetrievalAdapter(contract)
    return BasePageRetrievalAdapter(contract)


def load_page_retrieval_dataset(dataset_asset: dict[str, Any]) -> PageRetrievalDataset:
    paths = dict(dataset_asset.get("paths") or {})
    pages_path = Path(str(paths.get("pages_path", "")).strip())
    queries_path = Path(str(paths.get("queries_path", "")).strip())
    qrels_path = Path(str(paths.get("qrels_path", "")).strip())
    if not pages_path.exists():
        raise FileNotFoundError(f"pages_path does not exist: {pages_path}")
    if not queries_path.exists():
        raise FileNotFoundError(f"queries_path does not exist: {queries_path}")
    if not qrels_path.exists():
        raise FileNotFoundError(f"qrels_path does not exist: {qrels_path}")
    return PageRetrievalDataset(
        pages=normalize_pages(load_records(pages_path, "pages")),
        queries=normalize_queries(load_records(queries_path, "queries")),
        qrels=normalize_qrels(load_records(qrels_path, "qrels")),
        paths={
            "pages_path": str(pages_path),
            "queries_path": str(queries_path),
            "qrels_path": str(qrels_path),
        },
    )


def build_file_structure_snapshot(root: Path, max_preview: int = 16) -> dict[str, Any]:
    file_count = 0
    directory_count = 0
    files_by_extension: Counter[str] = Counter()
    preview_paths: list[str] = []
    top_level_entries: list[dict[str, Any]] = []
    max_depth = 0
    if not root.exists():
        return {
            "root_path": str(root),
            "exists": False,
            "file_count": 0,
            "directory_count": 0,
            "files_by_extension": {},
            "top_level_entries": [],
            "sample_paths": [],
            "max_depth": 0,
        }
    for item in sorted(root.iterdir(), key=lambda value: value.name.lower()):
        top_level_entries.append({"name": item.name, "type": "directory" if item.is_dir() else "file"})
    for path in sorted(root.rglob("*")):
        depth = len(path.relative_to(root).parts)
        max_depth = max(max_depth, depth)
        if path.is_dir():
            directory_count += 1
            continue
        file_count += 1
        files_by_extension[(path.suffix.lower() or "<no_ext>")] += 1
        if len(preview_paths) < max_preview:
            preview_paths.append(path.relative_to(root).as_posix())
    return {
        "root_path": str(root),
        "exists": True,
        "file_count": file_count,
        "directory_count": directory_count,
        "files_by_extension": dict(sorted(files_by_extension.items())),
        "top_level_entries": top_level_entries[:8],
        "sample_paths": preview_paths,
        "max_depth": max_depth,
    }


def build_sample_statistics(pages: list[dict[str, Any]], queries: list[dict[str, Any]], qrels: list[dict[str, Any]]) -> dict[str, Any]:
    page_lengths = [len(build_page_text(page)) for page in pages]
    query_lengths = [len(get_query_text(query)) for query in queries]
    page_split_counts: Counter[str] = Counter(str(page.get("split", "unknown") or "unknown") for page in pages)
    query_split_counts: Counter[str] = Counter(str(query.get("split", "unknown") or "unknown") for query in queries)
    qrels_by_query: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for qrel in qrels:
        qrels_by_query[str(qrel.get("query_id", "")).strip()].append(qrel)
    positive_counts = [len(items) for items in qrels_by_query.values()]
    return {
        "page_count": len(pages),
        "query_count": len(queries),
        "qrel_count": len(qrels),
        "page_split_counts": dict(sorted(page_split_counts.items())),
        "query_split_counts": dict(sorted(query_split_counts.items())),
        "average_page_text_length": round(mean(page_lengths), 2) if page_lengths else 0.0,
        "average_query_length": round(mean(query_lengths), 2) if query_lengths else 0.0,
        "average_positive_pages_per_query": round(mean(positive_counts), 2) if positive_counts else 0.0,
        "max_positive_pages_per_query": max(positive_counts) if positive_counts else 0,
        "unique_relevant_page_count": len({str(qrel.get("page_id", "")).strip() for qrel in qrels if str(qrel.get("page_id", "")).strip()}),
    }


def load_records(path: Path, collection_key: str) -> list[dict[str, Any]]:
    if path.suffix.lower() == ".jsonl":
        items: list[dict[str, Any]] = []
        for raw_line in path.read_text(encoding="utf-8-sig").splitlines():
            line = raw_line.strip()
            if not line:
                continue
            payload = json.loads(line)
            if isinstance(payload, dict):
                items.append(payload)
        return items
    payload = json.loads(path.read_text(encoding="utf-8-sig"))
    if isinstance(payload, list):
        return [item for item in payload if isinstance(item, dict)]
    if isinstance(payload, dict):
        collection = payload.get(collection_key)
        if isinstance(collection, list):
            return [item for item in collection if isinstance(item, dict)]
        for value in payload.values():
            if isinstance(value, list) and value and isinstance(value[0], dict):
                return [item for item in value if isinstance(item, dict)]
    raise ValueError(f"unsupported dataset record format in {path}")


def normalize_pages(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    pages: list[dict[str, Any]] = []
    for index, item in enumerate(items, start=1):
        page_id = first_text(item, "page_id", "pageId", "doc_id", "document_id", fallback=f"page_{index:04d}")
        pages.append(
            {
                "page_id": page_id,
                "split": first_text(item, "split", fallback="unknown"),
                "title": first_text(item, "title", "page_title"),
                "text": first_text(item, "text", "page_text", "content"),
                "ocr_text": first_text(item, "ocr_text", "ocr", "ocrText"),
                "section_titles": list(item.get("section_titles") or item.get("sectionTitles") or []),
                "metadata": {key: value for key, value in item.items() if key not in {"page_id", "pageId", "doc_id", "document_id", "split", "title", "page_title", "text", "page_text", "content", "ocr_text", "ocr", "ocrText", "section_titles", "sectionTitles"}},
            }
        )
    return pages


def normalize_queries(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    queries: list[dict[str, Any]] = []
    for index, item in enumerate(items, start=1):
        query_id = first_text(item, "query_id", "queryId", "qid", fallback=f"query_{index:04d}")
        queries.append(
            {
                "query_id": query_id,
                "split": first_text(item, "split", fallback="unknown"),
                "query": first_text(item, "query", "question", "text"),
                "metadata": {key: value for key, value in item.items() if key not in {"query_id", "queryId", "qid", "split", "query", "question", "text"}},
            }
        )
    return queries


def normalize_qrels(items: list[dict[str, Any]]) -> list[dict[str, Any]]:
    qrels: list[dict[str, Any]] = []
    for item in items:
        query_id = first_text(item, "query_id", "queryId", "qid")
        page_id = first_text(item, "page_id", "pageId", "doc_id", "document_id")
        if not query_id or not page_id:
            continue
        relevance = item.get("relevance", item.get("label", item.get("score", 1)))
        try:
            relevance_value = float(relevance)
        except (TypeError, ValueError):
            relevance_value = 1.0
        qrels.append({"query_id": query_id, "page_id": page_id, "relevance": relevance_value})
    return qrels


def build_page_text(page: dict[str, Any], text_fields: tuple[str, ...] = DEFAULT_PAGE_TEXT_FIELDS) -> str:
    values: list[str] = []
    for field_name in text_fields:
        value = page.get(field_name)
        if isinstance(value, list):
            values.extend(str(item).strip() for item in value if str(item).strip())
            continue
        text = str(value or "").strip()
        if text:
            values.append(text)
    if not values and page.get("metadata"):
        values.append(json.dumps(page["metadata"], ensure_ascii=False))
    return " ".join(values).strip()


def get_query_text(query: dict[str, Any], text_fields: tuple[str, ...] = DEFAULT_QUERY_TEXT_FIELDS) -> str:
    for field_name in text_fields:
        text = str(query.get(field_name, "") or "").strip()
        if text:
            return text
    return json.dumps(query.get("metadata", {}), ensure_ascii=False)


def build_qrels_lookup(qrels: list[dict[str, Any]]) -> dict[str, dict[str, float]]:
    lookup: dict[str, dict[str, float]] = defaultdict(dict)
    for qrel in qrels:
        query_id = str(qrel.get("query_id", "")).strip()
        page_id = str(qrel.get("page_id", "")).strip()
        if not query_id or not page_id:
            continue
        lookup[query_id][page_id] = float(qrel.get("relevance", 0.0) or 0.0)
    return dict(lookup)


def tokenize_text(value: str) -> list[str]:
    tokens: list[str] = []
    current: list[str] = []
    for char in value.lower():
        if char.isalnum():
            current.append(char)
            continue
        if current:
            tokens.append("".join(current))
            current = []
    if current:
        tokens.append("".join(current))
    return tokens


def looks_like_visdom(*values: str) -> bool:
    joined = " ".join(str(value).lower() for value in values if value)
    return "visdom" in joined or "visual document" in joined


def first_text(payload: dict[str, Any], *keys: str, fallback: str = "") -> str:
    for key in keys:
        value = str(payload.get(key, "") or "").strip()
        if value:
            return value
    return fallback
