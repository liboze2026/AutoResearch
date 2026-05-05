from __future__ import annotations

import sys
import unittest
from pathlib import Path

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.reader_phase4_sources import (
    CrossrefProvider,
    OpenAlexProvider,
    Phase4DatasetProfileSnapshot,
    Phase4ReaderRequest,
    Phase4SourceRecord,
    ArxivProvider,
    build_reader_context,
    dedupe_source_records,
    sort_source_records,
)


class FakeHTTPClient:
    def __init__(self, json_payloads: dict[str, dict] | None = None, text_payloads: dict[str, str] | None = None) -> None:
        self.json_payloads = json_payloads or {}
        self.text_payloads = text_payloads or {}

    def get_json(self, url: str) -> dict:
        for token, payload in self.json_payloads.items():
            if token in url:
                return payload
        raise AssertionError(f"unexpected json url: {url}")

    def get_text(self, url: str) -> str:
        for token, payload in self.text_payloads.items():
            if token in url:
                return payload
        raise AssertionError(f"unexpected text url: {url}")


class ReaderPhase4SourcesTests(unittest.TestCase):
    def test_openalex_provider_parses_mock_response(self) -> None:
        provider = OpenAlexProvider(
            FakeHTTPClient(
                json_payloads={
                    "api.openalex.org/works": {
                        "results": [
                            {
                                "display_name": "Layout-aware Document Retrieval",
                                "publication_year": 2024,
                                "doi": "https://doi.org/10.1000/layout",
                                "cited_by_count": 88,
                                "ids": {"doi": "https://doi.org/10.1000/layout", "openalex": "https://openalex.org/W1"},
                                "primary_location": {"source": {"display_name": "CVPR"}},
                                "open_access": {"oa_url": "https://example.com/layout.pdf"},
                                "abstract_inverted_index": {"Layout": [0], "retrieval": [1], "paper": [2]},
                                "authorships": [{"author": {"display_name": "Alice"}}, {"author": {"display_name": "Bob"}}],
                            }
                        ]
                    }
                }
            )
        )

        items = provider.search("visdom retrieval", 3)

        self.assertEqual(len(items), 1)
        self.assertEqual(items[0].title, "Layout-aware Document Retrieval")
        self.assertEqual(items[0].venue, "CVPR")
        self.assertEqual(items[0].quality_tier, "top_venue")
        self.assertEqual(items[0].metadata["external_ids"]["openalex"], "https://openalex.org/W1")

    def test_crossref_and_arxiv_providers_parse_mock_response(self) -> None:
        client = FakeHTTPClient(
            json_payloads={
                "api.crossref.org/works": {
                    "message": {
                        "items": [
                            {
                                "title": ["Document Page Retrieval"],
                                "container-title": ["SIGIR"],
                                "abstract": "<jats:p>Late interaction for retrieval.</jats:p>",
                                "URL": "https://doi.org/10.1000/doc",
                                "DOI": "10.1000/doc",
                                "type": "proceedings-article",
                                "is-referenced-by-count": 15,
                                "published-online": {"date-parts": [[2023, 7, 1]]},
                                "author": [{"given": "Jane", "family": "Doe"}],
                            }
                        ]
                    }
                }
            },
            text_payloads={
                "export.arxiv.org/api/query": """<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:arxiv="http://arxiv.org/schemas/atom">
  <entry>
    <id>http://arxiv.org/abs/2401.12345v1</id>
    <updated>2024-01-10T00:00:00Z</updated>
    <published>2024-01-10T00:00:00Z</published>
    <title>VisDoM Retrieval with Hard Negatives</title>
    <summary>Hard negative mining for multimodal page retrieval.</summary>
    <author><name>Test Author</name></author>
    <link href="http://arxiv.org/pdf/2401.12345v1" rel="related" title="pdf" type="application/pdf"/>
  </entry>
</feed>"""
            },
        )

        crossref_items = CrossrefProvider(client).search("visdom retrieval", 2)
        arxiv_items = ArxivProvider(client).search("visdom retrieval", 2)

        self.assertEqual(crossref_items[0].quality_tier, "top_venue")
        self.assertEqual(crossref_items[0].publication_year, 2023)
        self.assertEqual(arxiv_items[0].quality_tier, "arxiv")
        self.assertEqual(arxiv_items[0].metadata["external_ids"]["arxiv"], "2401.12345v1")

    def test_dedupe_and_sort_prioritize_top_venue(self) -> None:
        duplicate_arxiv = Phase4SourceRecord(
            title="Layout Retrieval Paper",
            abstract="arxiv abstract",
            authors=["A"],
            venue="arXiv",
            publication_year=2024,
            source_type="arxiv",
            source_url="https://arxiv.org/abs/1234.5678",
            quality_tier="arxiv",
            quality_score=6.0,
            relevance_score=9.5,
            citation_count=2,
            metadata={"doi": "10.1000/layout", "external_ids": {"arxiv": "1234.5678"}, "provider_refs": ["arxiv"]},
        )
        top_venue = Phase4SourceRecord(
            title="Layout Retrieval Paper",
            abstract="conference abstract",
            authors=["B"],
            venue="CVPR",
            publication_year=2024,
            source_type="crossref",
            source_url="https://doi.org/10.1000/layout",
            quality_tier="top_venue",
            quality_score=9.2,
            relevance_score=8.0,
            citation_count=100,
            metadata={"doi": "10.1000/layout", "external_ids": {"doi": "10.1000/layout"}, "provider_refs": ["crossref"]},
        )
        extra = Phase4SourceRecord(
            title="General Dual Encoder Retrieval",
            venue="arXiv",
            publication_year=2023,
            source_type="arxiv",
            source_url="https://arxiv.org/abs/9999.0001",
            quality_tier="arxiv",
            quality_score=6.0,
            relevance_score=7.0,
            citation_count=0,
            metadata={"external_ids": {"arxiv": "9999.0001"}, "provider_refs": ["arxiv"]},
        )

        merged = dedupe_source_records([duplicate_arxiv, top_venue, extra])
        ranked = sort_source_records(merged)

        self.assertEqual(len(merged), 2)
        self.assertEqual(ranked[0].quality_tier, "top_venue")
        self.assertIn("crossref", ranked[0].metadata["provider_refs"])
        self.assertIn("arxiv", ranked[0].metadata["provider_refs"])

    def test_reader_context_contains_required_fields(self) -> None:
        dataset = Phase4DatasetProfileSnapshot(
            id="p4ds_visdom",
            dataset_name="VisDoM",
            task_type="retrieval",
            modality_composition=["image", "text"],
            official_metric="Recall@10",
            known_difficulties=["Fine-grained evidence often hides inside dense document pages."],
        )
        request = Phase4ReaderRequest(dataset_profile=dataset, user_notes="Focus on page-level retrieval.", execution_mode="mock")
        sources = sort_source_records(
            [
                Phase4SourceRecord(
                    title="LayoutLMv3 for Document Retrieval",
                    abstract="Layout-aware document retrieval with contrastive learning.",
                    venue="ACM MM",
                    publication_year=2022,
                    source_type="fixture",
                    source_url="https://arxiv.org/abs/2204.08387",
                    quality_tier="top_venue",
                    quality_score=9.2,
                    relevance_score=9.0,
                    citation_count=1200,
                    metadata={"provider_refs": ["fixture"], "external_ids": {"arxiv": "2204.08387"}},
                ),
                Phase4SourceRecord(
                    title="ColBERT for Page Retrieval",
                    abstract="Late interaction retrieval over document pages.",
                    venue="SIGIR",
                    publication_year=2020,
                    source_type="fixture",
                    source_url="https://arxiv.org/abs/2004.12832",
                    quality_tier="top_venue",
                    quality_score=9.0,
                    relevance_score=8.5,
                    citation_count=900,
                    metadata={"provider_refs": ["fixture"], "external_ids": {"arxiv": "2004.12832"}},
                ),
            ]
        )

        context = build_reader_context(request.dataset_profile, sources, request.user_notes)

        for field_name in (
            "task_definition",
            "dataset_specific_challenges",
            "relevant_methods_landscape",
            "likely_strong_baselines",
            "common_failure_points",
            "evaluation_caveats",
            "implementation_constraints",
            "promising_research_directions",
            "citation_metadata",
        ):
            self.assertIn(field_name, context)
        self.assertTrue(context["citation_metadata"])


if __name__ == "__main__":
    unittest.main()
