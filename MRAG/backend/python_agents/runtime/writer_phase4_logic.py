from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass
class Phase4WriterRequest:
    run_manifest_id: str
    user_notes: str
    dataset_profile: dict[str, Any] = field(default_factory=dict)
    reader_context: dict[str, Any] = field(default_factory=dict)
    reader_sources: list[dict[str, Any]] = field(default_factory=list)
    selected_idea: dict[str, Any] = field(default_factory=dict)
    run_manifest: dict[str, Any] = field(default_factory=dict)
    metrics: dict[str, Any] = field(default_factory=dict)
    failure_summary: dict[str, Any] = field(default_factory=dict)
    artifact_summary: dict[str, Any] = field(default_factory=dict)
    coding_machine_report: dict[str, Any] = field(default_factory=dict)
    coding_human_report_excerpt: str = ""

    @classmethod
    def from_metadata(cls, metadata: dict[str, Any]) -> "Phase4WriterRequest":
        return cls(
            run_manifest_id=str(metadata.get("run_manifest_id", "")).strip(),
            user_notes=str(metadata.get("user_notes", "")).strip(),
            dataset_profile=dict(metadata.get("dataset_profile") or {}),
            reader_context=dict(metadata.get("reader_context") or {}),
            reader_sources=[dict(item) for item in list(metadata.get("reader_sources") or []) if isinstance(item, dict)],
            selected_idea=dict(metadata.get("selected_idea") or {}),
            run_manifest=dict(metadata.get("run_manifest") or {}),
            metrics=dict(metadata.get("metrics") or {}),
            failure_summary=dict(metadata.get("failure_summary") or {}),
            artifact_summary=dict(metadata.get("artifact_summary") or {}),
            coding_machine_report=dict(metadata.get("coding_machine_report") or {}),
            coding_human_report_excerpt=str(metadata.get("coding_human_report_excerpt", "")).strip(),
        )


def normalize_string(value: Any) -> str:
    return str(value or "").strip()


def normalize_string_list(value: Any) -> list[str]:
    if value is None:
        return []
    if isinstance(value, list):
        return [normalize_string(item) for item in value if normalize_string(item)]
    if isinstance(value, tuple):
        return [normalize_string(item) for item in value if normalize_string(item)]
    text = normalize_string(value)
    return [text] if text else []


def normalize_map(value: Any) -> dict[str, Any]:
    return dict(value or {}) if isinstance(value, dict) else {}


def _maybe_float(value: Any) -> float | None:
    try:
        return float(value)
    except (TypeError, ValueError):
        return None


def _citation_keys(source: dict[str, Any]) -> list[str]:
    metadata = normalize_map(source.get("metadata"))
    keys = []
    for field in ("doi", "DOI", "arxiv_id", "arxivId", "paper_id", "openalex_id"):
        text = normalize_string(metadata.get(field))
        if text:
            keys.append(text.lower())
    for field in ("sourceUrl", "source_url", "openAccessUrl", "open_access_url"):
        text = normalize_string(source.get(field) or metadata.get(field))
        if text:
            keys.append(text.lower())
    title = normalize_string(source.get("title") or metadata.get("title"))
    year = normalize_string(source.get("publicationYear") or metadata.get("publication_year"))
    if title:
        keys.append(f"title:{title.lower()}:{year}")
        keys.append(f"title:{title.lower()}")
    return [item for item in keys if item]


def _citation_ref(source: dict[str, Any], index: int) -> str:
    source_id = normalize_string(source.get("id"))
    if source_id:
        return source_id
    metadata = normalize_map(source.get("metadata"))
    for field in ("doi", "DOI", "arxiv_id", "arxivId", "paper_id", "openalex_id"):
        text = normalize_string(metadata.get(field))
        if text:
            return text
    source_url = normalize_string(source.get("sourceUrl") or source.get("source_url"))
    if source_url:
        return source_url
    title = normalize_string(source.get("title"))
    if title:
        return f"title:{title.lower()}"
    return f"citation:{index + 1}"


def citation_display_text(citation: dict[str, Any], index: int) -> str:
    authors = normalize_string_list(citation.get("authors"))
    title = normalize_string(citation.get("title")) or f"Untitled citation {index + 1}"
    venue = normalize_string(citation.get("venue"))
    year = normalize_string(citation.get("publication_year"))
    source_type = normalize_string(citation.get("source_type"))
    url = normalize_string(citation.get("open_access_url") or citation.get("source_url"))
    parts = []
    if authors:
        parts.append(", ".join(authors))
    parts.append(title)
    if venue:
        parts.append(venue)
    if year:
        parts.append(year)
    if source_type:
        parts.append(f"[{source_type}]")
    text = ". ".join([item for item in parts if item]).strip()
    if url:
        text = f"{text}. {url}"
    return f"[{index + 1}] {text}".strip()


def merge_citations(request: Phase4WriterRequest) -> tuple[list[dict[str, Any]], list[str], list[str]]:
    citations: list[dict[str, Any]] = []
    seen_keys: set[str] = set()
    citation_refs: list[str] = []
    reference_source_ids: list[str] = []

    dataset_citation = normalize_string(request.dataset_profile.get("citation"))
    if dataset_citation:
        dataset_ref = normalize_string(request.dataset_profile.get("id")) or "dataset_citation"
        citations.append(
            {
                "citation_ref": dataset_ref,
                "title": normalize_string(request.dataset_profile.get("datasetName")) or "Dataset Citation",
                "authors": [],
                "venue": "Dataset",
                "publication_year": "",
                "source_type": "dataset",
                "source_url": normalize_string(request.dataset_profile.get("serverPath")),
                "open_access_url": "",
                "quality_tier": "dataset",
                "display_text": f"[0] {dataset_citation}",
            }
        )
        citation_refs.append(dataset_ref)

    for index, source in enumerate(request.reader_sources):
        source_id = normalize_string(source.get("id"))
        keys = _citation_keys(source)
        if keys and any(item in seen_keys for item in keys):
            continue
        seen_keys.update(keys)
        citation = {
            "citation_ref": _citation_ref(source, index),
            "source_id": source_id,
            "title": normalize_string(source.get("title")),
            "authors": normalize_string_list(source.get("authors")),
            "venue": normalize_string(source.get("venue")),
            "publication_year": normalize_string(source.get("publicationYear") or source.get("publication_year")),
            "source_type": normalize_string(source.get("sourceType") or source.get("source_type")),
            "source_url": normalize_string(source.get("sourceUrl") or source.get("source_url")),
            "open_access_url": normalize_string(source.get("openAccessUrl") or source.get("open_access_url")),
            "quality_tier": normalize_string(source.get("qualityTier") or source.get("quality_tier")),
            "ranking_score": _maybe_float(source.get("rankingScore") or source.get("ranking_score")),
            "quality_score": _maybe_float(source.get("qualityScore") or source.get("quality_score")),
            "relevance_score": _maybe_float(source.get("relevanceScore") or source.get("relevance_score")),
            "citation_count": int(source.get("citationCount") or source.get("citation_count") or 0),
            "metadata": normalize_map(source.get("metadata")),
        }
        citation["display_text"] = citation_display_text(citation, len(citations))
        citations.append(citation)
        citation_refs.append(str(citation["citation_ref"]))
        if source_id:
            reference_source_ids.append(source_id)
    return citations, citation_refs, reference_source_ids


def build_report_title(request: Phase4WriterRequest) -> str:
    dataset_name = normalize_string(request.dataset_profile.get("datasetName")) or "Dataset"
    idea_title = normalize_string(request.selected_idea.get("title")) or "Selected Idea"
    return f"{dataset_name} Retrieval Experiment Report - {idea_title}"


def build_result_analysis(request: Phase4WriterRequest, primary_metric: str, metric_values: dict[str, Any]) -> dict[str, Any]:
    observations: list[str] = []
    primary_value = metric_values.get(primary_metric)
    if primary_value is not None:
        observations.append(f"Primary metric {primary_metric} = {primary_value}.")
    for key in ("recall@1", "recall@5", "recall@10", "mrr", "ndcg@10"):
        if key in metric_values and key != primary_metric:
            observations.append(f"{key} = {metric_values[key]}.")
    if normalize_string(request.run_manifest.get("status")) == "test_failed":
        observations.append("The run ended in test_failed after the configured repair loop.")
    if not observations:
        observations.append("Metric outputs were limited; inspect artifacts and logs for detailed behavior.")
    summary = observations[0]
    return {
        "primary_metric": primary_metric,
        "primary_metric_value": primary_value,
        "summary": summary,
        "observations": observations,
    }


def build_limitations(request: Phase4WriterRequest) -> list[str]:
    limitations = []
    limitations.extend(normalize_string_list(request.selected_idea.get("riskPoints")))
    limitations.extend(normalize_string_list(normalize_map(request.reader_context.get("structuredContext")).get("implementation_constraints")))
    if normalize_string(request.run_manifest.get("status")) == "test_failed":
        final_error = normalize_string(normalize_map(request.failure_summary).get("final_error"))
        if final_error:
            limitations.append(f"Run ended with test_failed: {final_error}")
        else:
            limitations.append("Run ended with test_failed and requires a revised idea candidate.")
    if not limitations:
        limitations.append("Current report focuses on page-level retrieval; snippet-level evidence retrieval is not yet covered.")
    return unique_strings(limitations)


def build_next_steps(request: Phase4WriterRequest) -> list[str]:
    structured_context = normalize_map(request.reader_context.get("structuredContext"))
    next_steps = normalize_string_list(structured_context.get("promising_research_directions"))
    if not next_steps:
        next_steps = [
            "Promote the current page-level retrieval baseline to a stronger hybrid or dense retriever.",
            "Add evidence/snippet-level retrieval after page-level metrics stabilize.",
            "Use failure feedback and report limitations to seed the next idea revision cycle.",
        ]
    if normalize_string(request.run_manifest.get("status")) == "test_failed":
        next_steps.insert(0, "Prioritize the generated revision candidates before changing the high-level research goal.")
    return unique_strings(next_steps)


def build_machine_readable_report(request: Phase4WriterRequest, execution_mode_used: str, response_text: str = "") -> dict[str, Any]:
    citations, citation_refs, reference_source_ids = merge_citations(request)
    metrics = normalize_map(request.metrics)
    metric_values = normalize_map(metrics.get("values"))
    primary_metric = normalize_string(metrics.get("primary_metric")) or normalize_string(request.dataset_profile.get("officialMetric")) or "recall@5"
    structured_context = normalize_map(request.reader_context.get("structuredContext"))
    report_title = build_report_title(request)
    return {
        "report_version": "phase4-experiment-report-v1",
        "report_title": report_title,
        "dataset": {
            "id": normalize_string(request.dataset_profile.get("id")),
            "name": normalize_string(request.dataset_profile.get("datasetName")),
            "task_type": normalize_string(request.dataset_profile.get("taskType")),
            "modality_composition": normalize_string_list(request.dataset_profile.get("modalityComposition")),
            "official_metric": normalize_string(request.dataset_profile.get("officialMetric")),
            "official_baseline": normalize_string(request.dataset_profile.get("officialBaseline")),
            "known_difficulties": normalize_string_list(request.dataset_profile.get("knownDifficulties")),
            "sample_statistics": normalize_map(request.dataset_profile.get("sampleStatistics")),
            "file_structure_snapshot": normalize_map(request.dataset_profile.get("fileStructureSnapshot")),
        },
        "task": {
            "definition": normalize_string(request.reader_context.get("taskDefinition")) or normalize_string(structured_context.get("task_definition")),
            "retrieval_focus": normalize_string_list(request.reader_context.get("retrievalFocus")) or normalize_string_list(structured_context.get("retrieval_focus")),
            "granularity": "page-level retrieval",
        },
        "reader_context_summary": {
            "summary": normalize_string(request.reader_context.get("summary")),
            "task_definition": normalize_string(request.reader_context.get("taskDefinition")) or normalize_string(structured_context.get("task_definition")),
            "dataset_specific_challenges": normalize_string_list(structured_context.get("dataset_specific_challenges")),
            "relevant_methods_landscape": normalize_string_list(structured_context.get("relevant_methods_landscape")),
            "likely_strong_baselines": normalize_string_list(structured_context.get("likely_strong_baselines")),
            "common_failure_points": normalize_string_list(structured_context.get("common_failure_points")),
            "evaluation_caveats": normalize_string_list(structured_context.get("evaluation_caveats")),
            "implementation_constraints": normalize_string_list(structured_context.get("implementation_constraints")),
            "promising_research_directions": normalize_string_list(structured_context.get("promising_research_directions")),
        },
        "citations": citations,
        "idea": {
            "id": normalize_string(request.selected_idea.get("id")),
            "title": normalize_string(request.selected_idea.get("title")),
            "problem_definition": normalize_string(request.selected_idea.get("problemDefinition")),
            "core_method": normalize_string(request.selected_idea.get("coreMethod")),
            "differentiators": normalize_string(request.selected_idea.get("differentiators")),
            "data_processing_needs": normalize_string_list(request.selected_idea.get("dataProcessingNeeds")),
            "model_changes": normalize_string_list(request.selected_idea.get("modelChanges")),
            "training_plan": normalize_string(request.selected_idea.get("trainingPlan")),
            "evaluation_metrics": normalize_string_list(request.selected_idea.get("evaluationMetrics")),
            "risk_points": normalize_string_list(request.selected_idea.get("riskPoints")),
            "expected_gains": normalize_string_list(request.selected_idea.get("expectedGains")),
            "score_summary": normalize_map(request.selected_idea.get("scoreSummary")),
        },
        "implementation": {
            "runner_mode": normalize_string(request.run_manifest.get("runnerMode")),
            "code_snapshot_id": normalize_string(request.run_manifest.get("codeSnapshotId")),
            "artifact_summary": normalize_map(request.artifact_summary),
            "coding_machine_report": normalize_map(request.coding_machine_report),
            "coding_human_report_excerpt": request.coding_human_report_excerpt,
        },
        "run_config": {
            "run_manifest_id": normalize_string(request.run_manifest.get("id")) or request.run_manifest_id,
            "status": normalize_string(request.run_manifest.get("status")),
            "server_id": normalize_string(request.run_manifest.get("serverId")),
            "gpu": normalize_string(request.run_manifest.get("gpu")),
            "retry_count": request.run_manifest.get("retryCount"),
            "max_retry_count": request.run_manifest.get("maxRetryCount"),
            "started_at": normalize_string(request.run_manifest.get("startedAt")),
            "finished_at": normalize_string(request.run_manifest.get("finishedAt")),
        },
        "metrics": {
            "primary_metric": primary_metric,
            "values": metric_values,
            "raw": metrics,
        },
        "error_summary": normalize_map(request.failure_summary),
        "result_analysis": build_result_analysis(request, primary_metric, metric_values),
        "limitations": build_limitations(request),
        "next_steps": build_next_steps(request),
        "citation_refs": citation_refs,
        "reference_source_ids": reference_source_ids,
        "user_notes": request.user_notes,
        "generation_metadata": {
            "execution_mode_used": execution_mode_used,
            "response_preview": normalize_string(response_text)[:240],
        },
    }


def render_human_readable_report(machine_report: dict[str, Any]) -> str:
    dataset = normalize_map(machine_report.get("dataset"))
    task = normalize_map(machine_report.get("task"))
    reader_context = normalize_map(machine_report.get("reader_context_summary"))
    idea = normalize_map(machine_report.get("idea"))
    implementation = normalize_map(machine_report.get("implementation"))
    run_config = normalize_map(machine_report.get("run_config"))
    metrics = normalize_map(machine_report.get("metrics"))
    result_analysis = normalize_map(machine_report.get("result_analysis"))
    error_summary = normalize_map(machine_report.get("error_summary"))
    citations = list(machine_report.get("citations") or [])

    metric_lines = []
    for key, value in normalize_map(metrics.get("values")).items():
        metric_lines.append(f"- {key}: {value}")
    if not metric_lines:
        metric_lines.append("- Metrics were not available in the current run artifacts.")

    limitation_lines = [f"- {item}" for item in normalize_string_list(machine_report.get("limitations"))]
    next_step_lines = [f"- {item}" for item in normalize_string_list(machine_report.get("next_steps"))]

    related_work_lines = []
    for item in citations[:5]:
        display_text = normalize_string(normalize_map(item).get("display_text"))
        if display_text:
            related_work_lines.append(f"- {display_text}")
    if not related_work_lines:
        related_work_lines.append("- Reader did not supply citation metadata; add structured references in the next run.")

    analysis_lines = [f"- {item}" for item in normalize_string_list(result_analysis.get("observations"))]
    if error_summary:
        final_error = normalize_string(error_summary.get("final_error"))
        if final_error:
            analysis_lines.append(f"- Failure summary: {final_error}")

    return "\n".join(
        [
            f"# {normalize_string(machine_report.get('report_title')) or 'Phase4 Experiment Report'}",
            "",
            "## 数据集与任务",
            f"- Dataset: {normalize_string(dataset.get('name'))}",
            f"- Task: {normalize_string(task.get('definition'))}",
            f"- Retrieval focus: {', '.join(normalize_string_list(task.get('retrieval_focus'))) or 'page-level retrieval'}",
            f"- Official metric: {normalize_string(dataset.get('official_metric'))}",
            "",
            "## 相关工作",
            *related_work_lines,
            "",
            "## Idea 说明",
            f"- Title: {normalize_string(idea.get('title'))}",
            f"- Problem definition: {normalize_string(idea.get('problem_definition'))}",
            f"- Core method: {normalize_string(idea.get('core_method'))}",
            f"- Differentiators: {normalize_string(idea.get('differentiators'))}",
            "",
            "## 实现方法",
            f"- Runner mode: {normalize_string(run_config.get('runner_mode'))}",
            f"- Code snapshot: {normalize_string(implementation.get('coding_machine_report', {}).get('code_snapshot_id') or implementation.get('code_snapshot_id'))}",
            f"- Artifact summary keys: {', '.join(sorted(normalize_map(implementation.get('artifact_summary')).keys())) or 'N/A'}",
            f"- Coding report excerpt: {normalize_string(implementation.get('coding_human_report_excerpt')) or 'N/A'}",
            "",
            "## 实验设置",
            f"- Run manifest: {normalize_string(run_config.get('run_manifest_id'))}",
            f"- Status: {normalize_string(run_config.get('status'))}",
            f"- Server/GPU: {normalize_string(run_config.get('server_id')) or 'local'} / {normalize_string(run_config.get('gpu')) or 'N/A'}",
            f"- Retry count: {run_config.get('retry_count')} / {run_config.get('max_retry_count')}",
            f"- Implementation constraints: {', '.join(normalize_string_list(reader_context.get('implementation_constraints'))) or 'N/A'}",
            "",
            "## 结果与分析",
            *metric_lines,
            *analysis_lines,
            "",
            "## 局限与下一步",
            *limitation_lines,
            *next_step_lines,
            "",
            "## 参考文献",
            *[f"- {normalize_string(normalize_map(item).get('display_text'))}" for item in citations if normalize_string(normalize_map(item).get("display_text"))],
            "",
        ]
    ).strip() + "\n"


def unique_strings(items: list[str]) -> list[str]:
    out: list[str] = []
    seen: set[str] = set()
    for item in items:
        text = normalize_string(item)
        key = text.lower()
        if not text or key in seen:
            continue
        seen.add(key)
        out.append(text)
    return out
