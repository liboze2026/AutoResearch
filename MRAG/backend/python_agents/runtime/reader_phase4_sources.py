from __future__ import annotations

import json
import re
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


DEFAULT_FIXTURE_PATH = Path(__file__).resolve().parent / "fixtures" / "reader_phase4_visdom.json"
ARXIV_NAMESPACE = {"atom": "http://www.w3.org/2005/Atom", "arxiv": "http://arxiv.org/schemas/atom"}
TOP_VENUE_HINTS = {
    "aaai",
    "acl",
    "acm mm",
    "cvpr",
    "eccv",
    "emnlp",
    "iccv",
    "iclr",
    "icml",
    "ijcai",
    "ijcv",
    "kdd",
    "naacl",
    "neurips",
    "nips",
    "sigir",
    "tpami",
    "transactions on pattern analysis and machine intelligence",
    "the web conference",
    "www",
}
LAYOUT_HINTS = ("layout", "layoutlm", "docformer", "document ai", "document understanding", "ocr")
DUAL_ENCODER_HINTS = ("clip", "dual encoder", "contrastive", "embedding")
LATE_INTERACTION_HINTS = ("colbert", "late interaction", "multi-vector", "rerank", "re-rank")
HARD_NEGATIVE_HINTS = ("hard negative", "negative mining", "hard-negative")


@dataclass
class Phase4DatasetProfileSnapshot:
    id: str = ""
    dataset_name: str = ""
    task_type: str = ""
    modality_composition: list[str] = field(default_factory=list)
    splits: list[dict[str, Any]] = field(default_factory=list)
    official_metric: str = ""
    official_baseline: str = ""
    known_difficulties: list[str] = field(default_factory=list)
    user_notes: str = ""
    metadata: dict[str, Any] = field(default_factory=dict)

    @classmethod
    def from_payload(cls, payload: dict[str, Any] | None) -> "Phase4DatasetProfileSnapshot":
        payload = dict(payload or {})
        raw_modalities = payload.get("modalityComposition", payload.get("modality_composition", []))
        raw_splits = payload.get("splits", [])
        raw_difficulties = payload.get("knownDifficulties", payload.get("known_difficulties", []))
        return cls(
            id=str(payload.get("id", "")).strip(),
            dataset_name=str(payload.get("datasetName", payload.get("dataset_name", ""))).strip(),
            task_type=str(payload.get("taskType", payload.get("task_type", ""))).strip().lower(),
            modality_composition=_string_list(raw_modalities),
            splits=[item for item in raw_splits if isinstance(item, dict)],
            official_metric=str(payload.get("officialMetric", payload.get("official_metric", ""))).strip(),
            official_baseline=str(payload.get("officialBaseline", payload.get("official_baseline", ""))).strip(),
            known_difficulties=_string_list(raw_difficulties),
            user_notes=str(payload.get("userNotes", payload.get("user_notes", ""))).strip(),
            metadata=dict(payload.get("metadata", {}) or {}),
        )

    def retrieval_task_label(self) -> str:
        if self.task_type == "retrieval":
            return "multimodal document retrieval"
        if self.task_type:
            return self.task_type
        return "multimodal document retrieval"


@dataclass
class Phase4ManualPaper:
    title: str = ""
    abstract: str = ""
    source_type: str = ""
    source_url: str = ""
    open_access_url: str = ""
    venue: str = ""
    year: int = 0
    authors: list[str] = field(default_factory=list)
    file_path: str = ""
    note: str = ""

    @classmethod
    def from_payload(cls, payload: dict[str, Any]) -> "Phase4ManualPaper":
        return cls(
            title=str(payload.get("title", "")).strip(),
            abstract=str(payload.get("abstract", "")).strip(),
            source_type=str(payload.get("sourceType", payload.get("source_type", payload.get("source", "manual")))).strip().lower(),
            source_url=str(payload.get("sourceUrl", payload.get("source_url", payload.get("url", "")))).strip(),
            open_access_url=str(payload.get("openAccessUrl", payload.get("open_access_url", ""))).strip(),
            venue=str(payload.get("venue", "")).strip(),
            year=_coerce_year(payload.get("year")),
            authors=_string_list(payload.get("authors", [])),
            file_path=str(payload.get("filePath", payload.get("file_path", ""))).strip(),
            note=str(payload.get("note", "")).strip(),
        )


@dataclass
class Phase4ReaderRequest:
    dataset_profile: Phase4DatasetProfileSnapshot
    manual_papers: list[Phase4ManualPaper] = field(default_factory=list)
    user_notes: str = ""
    search_mode: str = "auto"
    max_papers: int = 10
    execution_mode: str = "mock"

    @classmethod
    def from_contract_metadata(cls, metadata: dict[str, Any], execution_mode: str) -> "Phase4ReaderRequest":
        metadata = dict(metadata or {})
        raw_papers = metadata.get("manual_papers", metadata.get("manualPapers", []))
        manual_papers = [
            Phase4ManualPaper.from_payload(item)
            for item in raw_papers
            if isinstance(item, dict)
        ]
        request = cls(
            dataset_profile=Phase4DatasetProfileSnapshot.from_payload(metadata.get("dataset_profile", metadata.get("datasetProfile"))),
            manual_papers=manual_papers,
            user_notes=str(metadata.get("user_notes", metadata.get("userNotes", ""))).strip(),
            search_mode=str(metadata.get("search_mode", metadata.get("searchMode", "auto"))).strip().lower() or "auto",
            max_papers=_normalize_max_papers(metadata.get("max_papers", metadata.get("maxPapers", 10))),
            execution_mode=str(execution_mode or "mock").strip().lower() or "mock",
        )
        if not request.dataset_profile.dataset_name:
            request.dataset_profile.dataset_name = "Unknown Dataset"
        return request

    def resolved_search_mode(self) -> str:
        if self.search_mode in {"fixture", "live"}:
            return self.search_mode
        if self.execution_mode == "mock":
            return "fixture"
        return "live"

    def to_payload(self) -> dict[str, Any]:
        return {
            "dataset_profile_id": self.dataset_profile.id,
            "dataset_name": self.dataset_profile.dataset_name,
            "task_type": self.dataset_profile.task_type,
            "modality_composition": list(self.dataset_profile.modality_composition),
            "known_difficulties": list(self.dataset_profile.known_difficulties),
            "search_mode_requested": self.search_mode,
            "search_mode_used": self.resolved_search_mode(),
            "execution_mode_requested": self.execution_mode,
            "manual_paper_count": len(self.manual_papers),
            "max_papers": self.max_papers,
        }


@dataclass
class Phase4SourceRecord:
    title: str
    abstract: str = ""
    authors: list[str] = field(default_factory=list)
    venue: str = ""
    publication_year: int = 0
    source_type: str = ""
    source_url: str = ""
    open_access_url: str = ""
    quality_tier: str = ""
    ranking_score: float = 0.0
    quality_score: float = 0.0
    relevance_score: float = 0.0
    citation_count: int = 0
    metadata: dict[str, Any] = field(default_factory=dict)

    def dedupe_keys(self) -> list[str]:
        keys: list[str] = []
        doi = str(self.metadata.get("doi", "")).strip().lower()
        if doi:
            keys.append(f"doi:{doi}")
        external_ids = dict(self.metadata.get("external_ids", {}) or {})
        arxiv_id = str(external_ids.get("arxiv", "")).strip().lower()
        if arxiv_id:
            keys.append(f"arxiv:{arxiv_id}")
        normalized_title = _normalize_title(self.title)
        if normalized_title:
            keys.append(f"title:{normalized_title}")
        source_url = self.source_url.strip().lower()
        if source_url:
            keys.append(f"url:{source_url}")
        return keys

    def sort_key(self) -> tuple[float, float, int, int, str]:
        return (
            _quality_rank(self.quality_tier),
            self.relevance_score,
            self.citation_count,
            self.publication_year,
            _normalize_title(self.title),
        )

    def to_payload(self) -> dict[str, Any]:
        return {
            "title": self.title,
            "abstract": self.abstract,
            "authors": list(self.authors),
            "venue": self.venue,
            "publication_year": self.publication_year,
            "source_type": self.source_type,
            "source_url": self.source_url,
            "open_access_url": self.open_access_url,
            "quality_tier": self.quality_tier,
            "ranking_score": round(self.ranking_score, 4),
            "quality_score": round(self.quality_score, 4),
            "relevance_score": round(self.relevance_score, 4),
            "citation_count": self.citation_count,
            "metadata": dict(self.metadata),
        }


class HTTPClient:
    def __init__(self, user_agent: str = "MRAG-Phase4-Reader/1.0", timeout_seconds: float = 10.0) -> None:
        self.user_agent = user_agent
        self.timeout_seconds = timeout_seconds

    def get_json(self, url: str) -> dict[str, Any]:
        with urllib.request.urlopen(self._request(url), timeout=self.timeout_seconds) as response:
            payload = response.read().decode("utf-8")
        loaded = json.loads(payload)
        return loaded if isinstance(loaded, dict) else {}

    def get_text(self, url: str) -> str:
        with urllib.request.urlopen(self._request(url), timeout=self.timeout_seconds) as response:
            return response.read().decode("utf-8")

    def _request(self, url: str) -> urllib.request.Request:
        return urllib.request.Request(url, headers={"User-Agent": self.user_agent})


class BaseSourceProvider:
    name = "base"

    def __init__(self, client: HTTPClient | None = None) -> None:
        self.client = client or HTTPClient()

    def search(self, query: str, limit: int) -> list[Phase4SourceRecord]:
        raise NotImplementedError


class FixtureProvider(BaseSourceProvider):
    name = "fixture"

    def __init__(self, fixture_path: Path | None = None) -> None:
        super().__init__(client=HTTPClient(timeout_seconds=1.0))
        self.fixture_path = fixture_path or DEFAULT_FIXTURE_PATH

    def search(self, query: str, limit: int) -> list[Phase4SourceRecord]:
        payload = json.loads(self.fixture_path.read_text(encoding="utf-8"))
        items = payload.get("sources", [])
        out: list[Phase4SourceRecord] = []
        for item in items[:limit]:
            if not isinstance(item, dict):
                continue
            record = Phase4SourceRecord(
                title=str(item.get("title", "")).strip(),
                abstract=str(item.get("abstract", "")).strip(),
                authors=_string_list(item.get("authors", [])),
                venue=str(item.get("venue", "")).strip(),
                publication_year=_coerce_year(item.get("publication_year", item.get("publicationYear"))),
                source_type=str(item.get("source_type", item.get("sourceType", "fixture"))).strip().lower() or "fixture",
                source_url=str(item.get("source_url", item.get("sourceUrl", ""))).strip(),
                open_access_url=str(item.get("open_access_url", item.get("openAccessUrl", ""))).strip(),
                quality_tier=str(item.get("quality_tier", item.get("qualityTier", "top_venue"))).strip() or "top_venue",
                ranking_score=float(item.get("ranking_score", item.get("rankingScore", 0.0)) or 0.0),
                quality_score=float(item.get("quality_score", item.get("qualityScore", 0.0)) or 0.0),
                relevance_score=float(item.get("relevance_score", item.get("relevanceScore", 0.0)) or 0.0),
                citation_count=int(item.get("citation_count", item.get("citationCount", 0)) or 0),
                metadata=dict(item.get("metadata", {}) or {}),
            )
            record.metadata.setdefault("provider_refs", []).append(self.name)
            record.metadata.setdefault("matched_queries", []).append(query)
            out.append(record)
        return out


class OpenAlexProvider(BaseSourceProvider):
    name = "openalex"

    def search(self, query: str, limit: int) -> list[Phase4SourceRecord]:
        url = (
            "https://api.openalex.org/works?"
            + urllib.parse.urlencode({"search": query, "per-page": str(limit), "mailto": "mrag-reader@example.com"})
        )
        payload = self.client.get_json(url)
        out: list[Phase4SourceRecord] = []
        for item in payload.get("results", [])[:limit]:
            title = str(item.get("display_name", "")).strip()
            if not title:
                continue
            venue = _string_nested(item, "primary_location", "source", "display_name")
            abstract = _reconstruct_openalex_abstract(item.get("abstract_inverted_index"))
            doi = str(item.get("doi", "")).strip()
            source_url = doi or str(item.get("id", "")).strip()
            metadata = {
                "doi": doi,
                "external_ids": {
                    key.lower(): str(value).strip()
                    for key, value in dict(item.get("ids", {}) or {}).items()
                    if str(value).strip()
                },
                "provider_refs": [self.name],
                "matched_queries": [query],
                "abstract": abstract,
            }
            quality_tier = _quality_tier(venue, "openalex")
            relevance = _estimate_relevance(query, title, abstract)
            record = Phase4SourceRecord(
                title=title,
                abstract=abstract,
                authors=_openalex_authors(item),
                venue=venue,
                publication_year=_coerce_year(item.get("publication_year")),
                source_type=self.name,
                source_url=source_url,
                open_access_url=_string_nested(item, "open_access", "oa_url"),
                quality_tier=quality_tier,
                quality_score=_quality_score(quality_tier),
                relevance_score=relevance,
                citation_count=int(item.get("cited_by_count", 0) or 0),
                metadata=metadata,
            )
            record.ranking_score = _ranking_score(record)
            out.append(record)
        return out


class CrossrefProvider(BaseSourceProvider):
    name = "crossref"

    def search(self, query: str, limit: int) -> list[Phase4SourceRecord]:
        url = (
            "https://api.crossref.org/works?"
            + urllib.parse.urlencode({"query.bibliographic": query, "rows": str(limit), "mailto": "mrag-reader@example.com"})
        )
        payload = self.client.get_json(url)
        items = dict(payload.get("message", {}) or {}).get("items", [])
        out: list[Phase4SourceRecord] = []
        for item in items[:limit]:
            if not isinstance(item, dict):
                continue
            titles = item.get("title", [])
            title = str(titles[0] if isinstance(titles, list) and titles else "").strip()
            if not title:
                continue
            venue_list = item.get("container-title", [])
            venue = str(venue_list[0] if isinstance(venue_list, list) and venue_list else "").strip()
            abstract = _strip_html_tags(str(item.get("abstract", "")).strip())
            doi = str(item.get("DOI", "")).strip()
            quality_tier = _quality_tier(venue, str(item.get("type", "crossref")).strip().lower())
            relevance = _estimate_relevance(query, title, abstract)
            record = Phase4SourceRecord(
                title=title,
                abstract=abstract,
                authors=_crossref_authors(item),
                venue=venue,
                publication_year=_crossref_year(item),
                source_type=self.name,
                source_url=str(item.get("URL", "")).strip() or (f"https://doi.org/{doi}" if doi else ""),
                open_access_url=_crossref_open_access_url(item),
                quality_tier=quality_tier,
                quality_score=_quality_score(quality_tier),
                relevance_score=relevance,
                citation_count=int(item.get("is-referenced-by-count", 0) or 0),
                metadata={
                    "doi": doi,
                    "external_ids": {"doi": doi} if doi else {},
                    "provider_refs": [self.name],
                    "matched_queries": [query],
                    "abstract": abstract,
                    "type": str(item.get("type", "")).strip(),
                },
            )
            record.ranking_score = _ranking_score(record)
            out.append(record)
        return out


class ArxivProvider(BaseSourceProvider):
    name = "arxiv"

    def search(self, query: str, limit: int) -> list[Phase4SourceRecord]:
        arxiv_query = "all:" + " AND ".join(_escape_arxiv_term(item) for item in query.split() if item.strip())
        url = (
            "https://export.arxiv.org/api/query?"
            + urllib.parse.urlencode({"search_query": arxiv_query, "start": "0", "max_results": str(limit)})
        )
        payload = self.client.get_text(url)
        root = ET.fromstring(payload)
        out: list[Phase4SourceRecord] = []
        for entry in root.findall("atom:entry", ARXIV_NAMESPACE)[:limit]:
            title = _normalize_whitespace(entry.findtext("atom:title", "", ARXIV_NAMESPACE))
            if not title:
                continue
            abstract = _normalize_whitespace(entry.findtext("atom:summary", "", ARXIV_NAMESPACE))
            source_url = _normalize_whitespace(entry.findtext("atom:id", "", ARXIV_NAMESPACE))
            arxiv_id = source_url.rsplit("/", 1)[-1] if source_url else ""
            doi = _normalize_whitespace(entry.findtext("arxiv:doi", "", ARXIV_NAMESPACE))
            record = Phase4SourceRecord(
                title=title,
                abstract=abstract,
                authors=[
                    _normalize_whitespace(author.findtext("atom:name", "", ARXIV_NAMESPACE))
                    for author in entry.findall("atom:author", ARXIV_NAMESPACE)
                    if _normalize_whitespace(author.findtext("atom:name", "", ARXIV_NAMESPACE))
                ],
                venue="arXiv",
                publication_year=_coerce_year(_normalize_whitespace(entry.findtext("atom:published", "", ARXIV_NAMESPACE))[:4]),
                source_type=self.name,
                source_url=source_url,
                open_access_url=_arxiv_pdf_url(entry, source_url),
                quality_tier="arxiv",
                quality_score=_quality_score("arxiv"),
                relevance_score=_estimate_relevance(query, title, abstract),
                citation_count=0,
                metadata={
                    "doi": doi,
                    "external_ids": {"arxiv": arxiv_id, **({"doi": doi} if doi else {})},
                    "provider_refs": [self.name],
                    "matched_queries": [query],
                    "abstract": abstract,
                },
            )
            record.ranking_score = _ranking_score(record)
            out.append(record)
        return out


def build_search_queries(request: Phase4ReaderRequest) -> list[str]:
    dataset = request.dataset_profile
    dataset_name = dataset.dataset_name or "visual document retrieval"
    task_label = dataset.retrieval_task_label()
    raw: list[str] = [
        f"{dataset_name} {task_label}",
        f"{dataset_name} page retrieval multimodal document",
        "visual document retrieval page level retrieval",
        "layout aware multimodal document retrieval",
        "document page retrieval hard negative late interaction",
    ]
    if _looks_like_visdom(dataset_name, dataset.user_notes, request.user_notes):
        raw.insert(1, "VisDoM visual document retrieval page-level retrieval")
        raw.insert(2, "VisDoM multimodal document retrieval")
    for paper in request.manual_papers[:2]:
        if paper.title:
            raw.append(paper.title)
    return _unique_strings(raw)[:4]


def retrieve_research_sources(
    request: Phase4ReaderRequest,
    *,
    http_client: HTTPClient | None = None,
    live_providers: list[BaseSourceProvider] | None = None,
    fixture_path: Path | None = None,
) -> tuple[list[Phase4SourceRecord], list[dict[str, Any]], list[str], list[str], bool]:
    search_queries = build_search_queries(request)
    warnings: list[str] = []
    provider_statuses: list[dict[str, Any]] = []
    collected: list[Phase4SourceRecord] = manual_papers_to_records(request.manual_papers, request)
    used_fixture = False

    if request.resolved_search_mode() == "live":
        providers = live_providers or [
            OpenAlexProvider(http_client),
            CrossrefProvider(http_client),
            ArxivProvider(http_client),
        ]
        per_provider_limit = max(2, min(6, request.max_papers))
        for provider in providers:
            provider_total = 0
            provider_errors: list[str] = []
            for query in search_queries[:3]:
                try:
                    items = provider.search(query, per_provider_limit)
                    provider_total += len(items)
                    collected.extend(items)
                except Exception as exc:  # pragma: no cover - defensive in live mode
                    provider_errors.append(str(exc))
            status = "succeeded" if provider_total > 0 else "failed"
            provider_statuses.append(
                {
                    "provider": provider.name,
                    "status": status,
                    "result_count": provider_total,
                    "errors": provider_errors,
                }
            )
            if provider_errors:
                warnings.append(f"{provider.name} lookup failed: {'; '.join(provider_errors)}")

    deduped = sort_source_records(dedupe_source_records(collected))[: request.max_papers]
    non_fixture_count = sum(1 for item in deduped if item.source_type != "fixture")
    if request.resolved_search_mode() == "live" and non_fixture_count == 0:
        fixture_provider = FixtureProvider(fixture_path)
        deduped = sort_source_records(
            dedupe_source_records(deduped + fixture_provider.search(search_queries[0] if search_queries else "fixture", request.max_papers))
        )[: request.max_papers]
        used_fixture = True
        provider_statuses.append(
            {
                "provider": fixture_provider.name,
                "status": "fallback",
                "result_count": len(deduped),
                "errors": [],
            }
        )
        warnings.append("live reader providers returned no usable sources; fixture fallback was used.")
    elif request.resolved_search_mode() == "fixture":
        fixture_provider = FixtureProvider(fixture_path)
        deduped = sort_source_records(
            dedupe_source_records(deduped + fixture_provider.search(search_queries[0] if search_queries else "fixture", request.max_papers))
        )[: request.max_papers]
        used_fixture = True
        provider_statuses.append(
            {
                "provider": fixture_provider.name,
                "status": "succeeded",
                "result_count": len(deduped),
                "errors": [],
            }
        )

    return deduped, provider_statuses, search_queries, _unique_strings(warnings), used_fixture


def manual_papers_to_records(items: list[Phase4ManualPaper], request: Phase4ReaderRequest) -> list[Phase4SourceRecord]:
    out: list[Phase4SourceRecord] = []
    for item in items:
        if not item.title:
            continue
        quality_tier = _quality_tier(item.venue, item.source_type or "manual")
        record = Phase4SourceRecord(
            title=item.title,
            abstract=item.abstract,
            authors=_unique_strings(item.authors),
            venue=item.venue,
            publication_year=item.year,
            source_type=item.source_type or "manual",
            source_url=item.source_url,
            open_access_url=item.open_access_url,
            quality_tier=quality_tier,
            quality_score=_quality_score(quality_tier),
            relevance_score=_estimate_relevance(request.dataset_profile.dataset_name, item.title, item.abstract),
            citation_count=0,
            metadata={
                "provider_refs": ["manual"],
                "matched_queries": build_search_queries(request)[:1],
                "abstract": item.abstract,
                "note": item.note,
                "file_path": item.file_path,
            },
        )
        record.ranking_score = _ranking_score(record)
        out.append(record)
    return out


def dedupe_source_records(items: list[Phase4SourceRecord]) -> list[Phase4SourceRecord]:
    deduped: dict[str, Phase4SourceRecord] = {}
    for item in items:
        keys = item.dedupe_keys() or [f"title:{_normalize_title(item.title)}"]
        match_key = next((key for key in keys if key in deduped), keys[0])
        if match_key not in deduped:
            deduped[match_key] = _clone_record(item)
            continue
        deduped[match_key] = _merge_records(deduped[match_key], item)
    return list(deduped.values())


def sort_source_records(items: list[Phase4SourceRecord]) -> list[Phase4SourceRecord]:
    for item in items:
        if item.quality_score <= 0:
            item.quality_score = _quality_score(item.quality_tier)
        item.ranking_score = _ranking_score(item)
    return sorted(items, key=lambda item: item.sort_key(), reverse=True)


def build_reader_context(
    dataset: Phase4DatasetProfileSnapshot,
    sources: list[Phase4SourceRecord],
    user_notes: str = "",
) -> dict[str, Any]:
    method_hints = _collect_method_hints(sources)
    challenges = _unique_strings(dataset.known_difficulties + _default_dataset_challenges(dataset))
    baselines = _build_baselines(method_hints)
    common_failures = _build_failure_points(dataset, method_hints)
    evaluation_caveats = _build_evaluation_caveats(dataset)
    implementation_constraints = _build_implementation_constraints(dataset)
    directions = _build_research_directions(method_hints, challenges)
    citation_metadata = [
        {
            "title": item.title,
            "venue": item.venue,
            "year": item.publication_year,
            "source_url": item.source_url,
            "quality_tier": item.quality_tier,
            "source_type": item.source_type,
            "external_ids": dict(item.metadata.get("external_ids", {}) or {}),
            "doi": str(item.metadata.get("doi", "")).strip(),
        }
        for item in sources[: min(10, len(sources))]
    ]
    task_definition = (
        f"For {dataset.dataset_name}, prioritize {dataset.retrieval_task_label()} "
        f"that retrieves the correct document page early and robustly under visual-document noise."
    )
    if not dataset.dataset_name:
        task_definition = "Prioritize page-level multimodal document retrieval with strong recall under visually rich document conditions."
    methods_landscape = _build_methods_landscape(method_hints)
    reading_summary = build_reading_summary(dataset, sources, user_notes)
    return {
        "task_definition": task_definition,
        "dataset_specific_challenges": challenges,
        "relevant_methods_landscape": methods_landscape,
        "likely_strong_baselines": baselines,
        "common_failure_points": common_failures,
        "evaluation_caveats": evaluation_caveats,
        "implementation_constraints": implementation_constraints,
        "promising_research_directions": directions,
        "citation_metadata": citation_metadata,
        "reading_summary": reading_summary,
        "user_notes": user_notes.strip(),
    }


def build_reading_summary(dataset: Phase4DatasetProfileSnapshot, sources: list[Phase4SourceRecord], user_notes: str = "") -> str:
    top_titles = ", ".join(item.title for item in sources[:3])
    summary = (
        f"Reader synthesized {len(sources)} source(s) for {dataset.dataset_name or 'the target dataset'}. "
        f"The strongest signals cluster around layout-aware document encoders, efficient first-stage retrieval, "
        f"and harder negative sampling for page-level recall."
    )
    if top_titles:
        summary += f" Representative citations include {top_titles}."
    if user_notes.strip():
        summary += f" User notes were considered: {user_notes.strip()}."
    return summary


def source_records_from_payload(value: Any) -> list[Phase4SourceRecord]:
    if not isinstance(value, list):
        return []
    out: list[Phase4SourceRecord] = []
    for item in value:
        if not isinstance(item, dict):
            continue
        record = Phase4SourceRecord(
            title=str(item.get("title", "")).strip(),
            abstract=str(item.get("abstract", "")).strip(),
            authors=_string_list(item.get("authors", [])),
            venue=str(item.get("venue", "")).strip(),
            publication_year=_coerce_year(item.get("publication_year", item.get("publicationYear"))),
            source_type=str(item.get("source_type", item.get("sourceType", ""))).strip().lower(),
            source_url=str(item.get("source_url", item.get("sourceUrl", ""))).strip(),
            open_access_url=str(item.get("open_access_url", item.get("openAccessUrl", ""))).strip(),
            quality_tier=str(item.get("quality_tier", item.get("qualityTier", ""))).strip(),
            ranking_score=float(item.get("ranking_score", item.get("rankingScore", 0.0)) or 0.0),
            quality_score=float(item.get("quality_score", item.get("qualityScore", 0.0)) or 0.0),
            relevance_score=float(item.get("relevance_score", item.get("relevanceScore", 0.0)) or 0.0),
            citation_count=int(item.get("citation_count", item.get("citationCount", 0)) or 0),
            metadata=dict(item.get("metadata", {}) or {}),
        )
        if record.title:
            out.append(record)
    return out


def _normalize_max_papers(value: Any) -> int:
    try:
        parsed = int(value)
    except (TypeError, ValueError):
        parsed = 10
    if parsed <= 0:
        return 1
    if parsed > 20:
        return 20
    return parsed


def _coerce_year(value: Any) -> int:
    if isinstance(value, str) and len(value) >= 4:
        value = value[:4]
    try:
        parsed = int(value)
    except (TypeError, ValueError):
        return 0
    if parsed < 1900 or parsed > 2100:
        return 0
    return parsed


def _string_list(value: Any) -> list[str]:
    if isinstance(value, list):
        return _unique_strings(str(item).strip() for item in value if str(item).strip())
    if isinstance(value, str) and value.strip():
        return [value.strip()]
    return []


def _unique_strings(items: Any) -> list[str]:
    out: list[str] = []
    seen: set[str] = set()
    for item in items:
        text = str(item).strip()
        if not text:
            continue
        key = text.lower()
        if key in seen:
            continue
        seen.add(key)
        out.append(text)
    return out


def _normalize_title(value: str) -> str:
    return re.sub(r"[^a-z0-9]+", " ", value.strip().lower()).strip()


def _quality_tier(venue: str, source_type: str) -> str:
    source_type = source_type.strip().lower()
    normalized_venue = venue.strip().lower()
    if any(hint in normalized_venue for hint in TOP_VENUE_HINTS):
        return "top_venue"
    if source_type == "arxiv":
        return "arxiv"
    if source_type in {"openalex", "crossref"} and venue.strip():
        return "peer_reviewed"
    if source_type == "manual" and venue.strip():
        return "peer_reviewed"
    if source_type == "fixture":
        return "top_venue"
    return "open_metadata"


def _quality_rank(tier: str) -> float:
    return {
        "top_venue": 4.0,
        "peer_reviewed": 3.0,
        "manual": 2.8,
        "arxiv": 2.0,
        "open_metadata": 1.0,
    }.get(tier.strip().lower(), 1.0)


def _quality_score(tier: str) -> float:
    return {
        "top_venue": 9.2,
        "peer_reviewed": 7.8,
        "manual": 6.5,
        "arxiv": 6.0,
        "open_metadata": 4.5,
    }.get(tier.strip().lower(), 4.5)


def _ranking_score(item: Phase4SourceRecord) -> float:
    citation_bonus = min(float(item.citation_count), 3000.0) / 300.0
    year_bonus = max(min(item.publication_year, 2030) - 2018, 0) * 0.08
    return round(_quality_rank(item.quality_tier) * 20.0 + item.relevance_score * 4.0 + citation_bonus + year_bonus, 4)


def _estimate_relevance(query: str, title: str, abstract: str) -> float:
    haystack = f"{title} {abstract}".lower()
    tokens = [item for item in _normalize_title(query).split() if len(item) > 2]
    if not tokens:
        return 5.0
    overlap = sum(1 for token in tokens if token in haystack)
    dense_bonus = 1.5 if any(token in haystack for token in ("retrieval", "document", "layout", "page")) else 0.0
    return min(10.0, round((overlap / max(len(tokens), 1)) * 8.5 + dense_bonus, 4))


def _clone_record(item: Phase4SourceRecord) -> Phase4SourceRecord:
    return Phase4SourceRecord(
        title=item.title,
        abstract=item.abstract,
        authors=list(item.authors),
        venue=item.venue,
        publication_year=item.publication_year,
        source_type=item.source_type,
        source_url=item.source_url,
        open_access_url=item.open_access_url,
        quality_tier=item.quality_tier,
        ranking_score=item.ranking_score,
        quality_score=item.quality_score,
        relevance_score=item.relevance_score,
        citation_count=item.citation_count,
        metadata=dict(item.metadata),
    )


def _merge_records(current: Phase4SourceRecord, incoming: Phase4SourceRecord) -> Phase4SourceRecord:
    preferred = incoming if incoming.sort_key() > current.sort_key() else current
    merged = _clone_record(preferred)
    merged.abstract = preferred.abstract or current.abstract or incoming.abstract
    merged.authors = _unique_strings(list(current.authors) + list(incoming.authors))
    merged.open_access_url = preferred.open_access_url or current.open_access_url or incoming.open_access_url
    merged.source_url = preferred.source_url or current.source_url or incoming.source_url
    merged.citation_count = max(current.citation_count, incoming.citation_count)
    merged.quality_score = max(current.quality_score, incoming.quality_score)
    merged.relevance_score = max(current.relevance_score, incoming.relevance_score)
    merged.ranking_score = max(current.ranking_score, incoming.ranking_score)
    merged.metadata = _merge_metadata(current.metadata, incoming.metadata)
    merged.quality_tier = preferred.quality_tier or current.quality_tier or incoming.quality_tier
    return merged


def _merge_metadata(current: dict[str, Any], incoming: dict[str, Any]) -> dict[str, Any]:
    merged = dict(current or {})
    for key, value in dict(incoming or {}).items():
        if key in {"provider_refs", "matched_queries", "keywords"}:
            merged[key] = _unique_strings(list(merged.get(key, [])) + list(value if isinstance(value, list) else [value]))
            continue
        if key == "external_ids":
            current_ids = dict(merged.get("external_ids", {}) or {})
            for id_key, id_value in dict(value or {}).items():
                if str(id_value).strip():
                    current_ids[str(id_key).strip().lower()] = str(id_value).strip()
            merged["external_ids"] = current_ids
            continue
        if key == "doi":
            if not str(merged.get("doi", "")).strip() and str(value).strip():
                merged[key] = str(value).strip()
            continue
        if key == "abstract":
            if not str(merged.get("abstract", "")).strip() and str(value).strip():
                merged[key] = str(value).strip()
            continue
        if key not in merged or merged[key] in ("", None, [], {}):
            merged[key] = value
    return merged


def _reconstruct_openalex_abstract(payload: Any) -> str:
    if not isinstance(payload, dict):
        return ""
    max_position = -1
    for positions in payload.values():
        if isinstance(positions, list):
            max_position = max(max_position, max((int(item) for item in positions if isinstance(item, int)), default=-1))
    if max_position < 0:
        return ""
    words = [""] * (max_position + 1)
    for word, positions in payload.items():
        if not isinstance(word, str) or not isinstance(positions, list):
            continue
        for position in positions:
            if isinstance(position, int) and 0 <= position < len(words):
                words[position] = word
    return _normalize_whitespace(" ".join(item for item in words if item))


def _openalex_authors(item: dict[str, Any]) -> list[str]:
    out: list[str] = []
    for author_item in item.get("authorships", []):
        if not isinstance(author_item, dict):
            continue
        display_name = _string_nested(author_item, "author", "display_name")
        if display_name:
            out.append(display_name)
    return _unique_strings(out)


def _crossref_authors(item: dict[str, Any]) -> list[str]:
    out: list[str] = []
    for author_item in item.get("author", []):
        if not isinstance(author_item, dict):
            continue
        name = " ".join(
            part for part in [str(author_item.get("given", "")).strip(), str(author_item.get("family", "")).strip()] if part
        ).strip()
        if name:
            out.append(name)
    return _unique_strings(out)


def _crossref_year(item: dict[str, Any]) -> int:
    for key in ("published-print", "published-online", "issued", "created"):
        date_parts = dict(item.get(key, {}) or {}).get("date-parts", [])
        if isinstance(date_parts, list) and date_parts and isinstance(date_parts[0], list) and date_parts[0]:
            return _coerce_year(date_parts[0][0])
    return 0


def _crossref_open_access_url(item: dict[str, Any]) -> str:
    for link in item.get("link", []):
        if not isinstance(link, dict):
            continue
        candidate = str(link.get("URL", "")).strip()
        if candidate:
            return candidate
    return ""


def _arxiv_pdf_url(entry: ET.Element, source_url: str) -> str:
    for link in entry.findall("atom:link", ARXIV_NAMESPACE):
        title = str(link.attrib.get("title", "")).strip().lower()
        href = str(link.attrib.get("href", "")).strip()
        if title == "pdf" and href:
            return href
    if source_url:
        return source_url.replace("/abs/", "/pdf/") + ".pdf"
    return ""


def _escape_arxiv_term(value: str) -> str:
    return re.sub(r"[^a-zA-Z0-9]+", "", value)


def _string_nested(payload: dict[str, Any], *keys: str) -> str:
    current: Any = payload
    for key in keys:
        if not isinstance(current, dict):
            return ""
        current = current.get(key)
    return str(current or "").strip()


def _normalize_whitespace(value: str) -> str:
    return re.sub(r"\s+", " ", value or "").strip()


def _strip_html_tags(value: str) -> str:
    return _normalize_whitespace(re.sub(r"<[^>]+>", " ", value))


def _looks_like_visdom(*values: str) -> bool:
    joined = " ".join(value.lower() for value in values if value)
    return "visdom" in joined or "visual document" in joined


def _collect_method_hints(items: list[Phase4SourceRecord]) -> set[str]:
    haystack = " ".join(f"{item.title} {item.abstract}".lower() for item in items)
    hints: set[str] = set()
    if any(token in haystack for token in LAYOUT_HINTS):
        hints.add("layout")
    if any(token in haystack for token in DUAL_ENCODER_HINTS):
        hints.add("dual_encoder")
    if any(token in haystack for token in LATE_INTERACTION_HINTS):
        hints.add("late_interaction")
    if any(token in haystack for token in HARD_NEGATIVE_HINTS):
        hints.add("hard_negative")
    if not hints:
        hints.update({"layout", "dual_encoder"})
    return hints


def _default_dataset_challenges(dataset: Phase4DatasetProfileSnapshot) -> list[str]:
    out: list[str] = []
    modalities = " ".join(dataset.modality_composition).lower()
    if "image" in modalities or "vision" in modalities or "document" in modalities or not modalities:
        out.append("OCR noise, dense layouts, and visually similar pages can distort retrieval ranking.")
    if dataset.task_type == "retrieval":
        out.append("Page-level labels can hide finer evidence mismatch inside long visual documents.")
        out.append("Hard negatives often come from nearby pages or templates within the same collection.")
    if dataset.official_metric:
        out.append(f"Optimization should align with the official metric {dataset.official_metric} instead of generic accuracy alone.")
    return out


def _build_methods_landscape(method_hints: set[str]) -> list[str]:
    out: list[str] = []
    if "layout" in method_hints:
        out.append("Layout-aware document encoders remain strong representation baselines for visually rich pages.")
    if "dual_encoder" in method_hints:
        out.append("Dual-encoder contrastive retrieval is still the most practical first-stage recall backbone.")
    if "late_interaction" in method_hints:
        out.append("Late-interaction retrieval improves difficult ranking cases without a full cross-encoder cost.")
    if "hard_negative" in method_hints:
        out.append("Hard-negative mining repeatedly appears when recall is limited by near-duplicate pages.")
    return _unique_strings(out)


def _build_baselines(method_hints: set[str]) -> list[str]:
    out = ["OCR-text BM25 over page transcriptions."]
    if "dual_encoder" in method_hints:
        out.append("CLIP-style dual-encoder retrieval with query text and page image embeddings.")
    if "layout" in method_hints:
        out.append("Layout-aware document encoder plus a contrastive retrieval head.")
    if "late_interaction" in method_hints:
        out.append("Late-interaction reranking on top of a dense or lexical shortlist.")
    return _unique_strings(out)


def _build_failure_points(dataset: Phase4DatasetProfileSnapshot, method_hints: set[str]) -> list[str]:
    out = [
        "Retriever overweights repeated headers or boilerplate instead of evidence-bearing regions.",
        "Negative sampling is too easy, so recall collapses on near-duplicate pages.",
    ]
    if "layout" in method_hints:
        out.append("Layout features dominate while semantic text cues remain underused for dense evidence pages.")
    if "dual_encoder" in method_hints:
        out.append("Simple global embeddings miss small evidence spans hidden inside cluttered document pages.")
    out.extend(dataset.known_difficulties[:2])
    return _unique_strings(out)


def _build_evaluation_caveats(dataset: Phase4DatasetProfileSnapshot) -> list[str]:
    out = [
        "Page-level retrieval metrics should be interpreted before any snippet-level extension.",
        "Train/validation/test splits should be checked for document overlap and template leakage.",
    ]
    if dataset.official_metric:
        out.insert(0, f"Official evaluation should preserve the dataset metric definition for {dataset.official_metric}.")
    return _unique_strings(out)


def _build_implementation_constraints(dataset: Phase4DatasetProfileSnapshot) -> list[str]:
    out = [
        "Need stable page ids, split definitions, and reproducible indexing to compare retrieval recall fairly.",
        "Image resolution, OCR preprocessing, and batching policy can materially change retrieval quality.",
    ]
    modalities = " ".join(dataset.modality_composition).lower()
    if "text" in modalities and ("image" in modalities or "vision" in modalities or not modalities):
        out.append("Image-text batching and OCR fusion must fit GPU memory without changing page ordering semantics.")
    return _unique_strings(out)


def _build_research_directions(method_hints: set[str], challenges: list[str]) -> list[str]:
    out: list[str] = []
    if "layout" in method_hints:
        out.append("Layout-aware hard-negative mining for visually similar but semantically incorrect pages.")
    if "late_interaction" in method_hints:
        out.append("Two-stage retrieval with efficient page recall followed by region-sensitive reranking.")
    if "dual_encoder" in method_hints:
        out.append("Query-conditioned page representations that keep efficient first-stage retrieval.")
    if not out:
        out.append("Dataset-specific hard-negative mining for visually rich page retrieval.")
    if any("evidence" in item.lower() for item in challenges):
        out.append("Evidence-aware page chunking that improves page-level recall before snippet expansion.")
    return _unique_strings(out)
