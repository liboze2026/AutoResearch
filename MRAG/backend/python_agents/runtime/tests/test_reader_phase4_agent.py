from __future__ import annotations

import shutil
import sys
import unittest
import uuid
from pathlib import Path

PYTHON_AGENTS_ROOT = Path(__file__).resolve().parents[2]
if str(PYTHON_AGENTS_ROOT) not in sys.path:
    sys.path.insert(0, str(PYTHON_AGENTS_ROOT))

from runtime.contract import AgentRuntimeInput
from runtime.reader_phase4_agent import READER_PHASE4_SCHEMA_REF, build_reader_phase4_agent
from runtime.reader_phase4_sources import HTTPClient

TEST_ROOT = Path(__file__).resolve().parents[4] / "workspace" / "python-runtime-tests"
TEST_ROOT.mkdir(parents=True, exist_ok=True)


def make_workspace() -> str:
    workspace = TEST_ROOT / f"reader-phase4-{uuid.uuid4().hex}"
    workspace.mkdir(parents=True, exist_ok=True)
    return str(workspace)


def build_contract(execution_mode: str, workspace_dir: str) -> AgentRuntimeInput:
    return AgentRuntimeInput(
        job_id="reader-phase4-job-001",
        agent_type="reader_phase4",
        execution_mode=execution_mode,
        model_provider="codex",
        model_name="phase4-reader-test",
        prompt_version="v1",
        input_refs=[],
        output_schema_ref=READER_PHASE4_SCHEMA_REF,
        skill_refs=[],
        tool_refs=[],
        memory_refs=[],
        workspace_dir=workspace_dir,
        metadata={
            "dataset_profile": {
                "id": "p4ds_visdom",
                "datasetName": "VisDoM",
                "taskType": "retrieval",
                "modalityComposition": ["image", "text"],
                "officialMetric": "Recall@10",
                "knownDifficulties": ["Visual pages contain dense layout and OCR noise."],
                "userNotes": "Focus on page-level retrieval.",
            },
            "user_notes": "Prioritize page-level recall.",
            "max_papers": 4,
        },
    )


class FakeLiveHTTPClient(HTTPClient):
    def __init__(self) -> None:
        super().__init__(timeout_seconds=1.0)

    def get_json(self, url: str) -> dict:
        if "api.openalex.org/works" in url:
            return {
                "results": [
                    {
                        "display_name": "Layout-aware Retrieval for Visual Documents",
                        "publication_year": 2024,
                        "doi": "https://doi.org/10.1000/visualdocs",
                        "cited_by_count": 44,
                        "ids": {"doi": "https://doi.org/10.1000/visualdocs", "openalex": "https://openalex.org/W99"},
                        "primary_location": {"source": {"display_name": "CVPR"}},
                        "open_access": {"oa_url": "https://example.com/visualdocs.pdf"},
                        "abstract_inverted_index": {"Layout": [0], "aware": [1], "retrieval": [2]},
                        "authorships": [{"author": {"display_name": "Alice"}}],
                    }
                ]
            }
        if "api.crossref.org/works" in url:
            return {"message": {"items": []}}
        raise AssertionError(f"unexpected url {url}")

    def get_text(self, url: str) -> str:
        if "export.arxiv.org/api/query" not in url:
            raise AssertionError(f"unexpected url {url}")
        return """<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xmlns:arxiv="http://arxiv.org/schemas/atom">
  <entry>
    <id>http://arxiv.org/abs/2401.88888v1</id>
    <updated>2024-01-10T00:00:00Z</updated>
    <published>2024-01-10T00:00:00Z</published>
    <title>Late Interaction for Visual Document Retrieval</title>
    <summary>Late interaction ranking for page retrieval.</summary>
    <author><name>Test Author</name></author>
    <link href="http://arxiv.org/pdf/2401.88888v1" rel="related" title="pdf" type="application/pdf"/>
  </entry>
</feed>"""


class ReaderPhase4AgentTests(unittest.TestCase):
    def test_mock_reader_uses_fixture_and_returns_structured_context(self) -> None:
        workspace = make_workspace()
        try:
            contract = build_contract("mock", workspace)
            agent = build_reader_phase4_agent(contract)

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertEqual(output.normalized_payload["execution_mode_used"], "mock")
            self.assertTrue(output.normalized_payload["sources"])
            self.assertIn("reader_context", output.normalized_payload)
            self.assertTrue(output.normalized_payload["reader_context"]["citation_metadata"])
            self.assertTrue(output.artifact_manifest)
        finally:
            shutil.rmtree(workspace, ignore_errors=True)

    def test_live_reader_uses_mockable_http_provider(self) -> None:
        workspace = make_workspace()
        try:
            contract = build_contract("api", workspace)
            agent = build_reader_phase4_agent(contract, http_client=FakeLiveHTTPClient())

            output = agent.run(contract)

            self.assertEqual(output.status, "succeeded")
            self.assertEqual(output.validation_status, "succeeded")
            self.assertEqual(output.normalized_payload["execution_mode_used"], "api")
            self.assertEqual(output.normalized_payload["metadata"]["used_fixture"], False)
            self.assertTrue(any(item["source_type"] == "openalex" for item in output.normalized_payload["sources"]))
            self.assertTrue(output.normalized_payload["reader_context"]["likely_strong_baselines"])
        finally:
            shutil.rmtree(workspace, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
