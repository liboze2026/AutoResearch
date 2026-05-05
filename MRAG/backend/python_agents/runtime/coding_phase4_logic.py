from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Any


@dataclass
class CodingGenerationPlan:
    method_slug: str
    branch_name: str
    method_relative_path: str
    method_tags: list[str]
    retrieval_notes: list[str]
    query_expansion_terms: list[str]
    top_k: int
    score_bias: float
    title_match_bonus: float
    ocr_match_bonus: float
    section_match_bonus: float
    exact_phrase_bonus: float
    execution_mode_used: str
    generation_summary: str


def slugify(value: str) -> str:
    value = re.sub(r"[^a-zA-Z0-9]+", "_", value.strip().lower())
    value = re.sub(r"_+", "_", value).strip("_")
    return value or "generated_method"


def build_generation_plan(request: Any, execution_mode_used: str, response_text: str = "") -> CodingGenerationPlan:
    idea_title = str(request.idea.get("title", "")).strip()
    method_slug = slugify(idea_title or str(request.idea.get("coreMethod", "")).strip() or "generated_method")
    query_expansion_terms = build_query_expansion_terms(request)
    method_tags = build_method_tags(request)
    score_bias = min(0.02 * max(len(method_tags), 1), 0.10)
    title_bonus, ocr_bonus, section_bonus, exact_phrase_bonus = build_scoring_profile(request)
    notes = build_retrieval_notes(request, response_text)
    generation_summary = (
        f"Generated page-level retrieval method for {idea_title or request.dataset_profile.get('datasetName', '')} "
        f"with {len(query_expansion_terms)} query expansion terms and execution mode {execution_mode_used}."
    )
    return CodingGenerationPlan(
        method_slug=method_slug,
        branch_name=f"method/{method_slug}",
        method_relative_path=f"methods/generated/{method_slug}.py",
        method_tags=method_tags,
        retrieval_notes=notes,
        query_expansion_terms=query_expansion_terms,
        top_k=10,
        score_bias=round(score_bias, 2),
        title_match_bonus=title_bonus,
        ocr_match_bonus=ocr_bonus,
        section_match_bonus=section_bonus,
        exact_phrase_bonus=exact_phrase_bonus,
        execution_mode_used=execution_mode_used,
        generation_summary=generation_summary,
    )


def build_phase4_config(request: Any, plan: CodingGenerationPlan) -> dict[str, Any]:
    idea_title = str(request.idea.get("title", "")).strip()
    return {
        "protocol_version": "phase4-retrieval-mainline-v1",
        "method_name": plan.method_slug,
        "method_module_path": plan.method_relative_path,
        "runner_mode": request.runner_mode,
        "method_branch": plan.branch_name,
        "parameters": {
            "page_level_retrieval": True,
            "target_granularity": "page",
            "query_expansion_terms": list(plan.query_expansion_terms),
            "scoring_profile": {
                "title_match_bonus": plan.title_match_bonus,
                "ocr_match_bonus": plan.ocr_match_bonus,
                "section_match_bonus": plan.section_match_bonus,
                "exact_phrase_bonus": plan.exact_phrase_bonus,
                "score_bias": plan.score_bias,
            },
            "implementation_constraints": list(request.reader_context.get("implementation_constraints", []) or []),
        },
        "retry_policy": build_retry_plan(request.max_retry_count),
        "dataset_adapter": {
            "dataset_profile_id": str(request.dataset_profile.get("id", "")).strip(),
            "dataset_name": str(request.dataset_profile.get("datasetName", "")).strip(),
            "task_type": str(request.dataset_profile.get("taskType", "")).strip(),
            "server_path": str(request.dataset_profile.get("serverPath", "")).strip(),
            "official_metric": str(request.dataset_profile.get("officialMetric", "")).strip() or "recall@5",
            "splits": list(request.dataset_profile.get("splits", []) or []),
            "metadata": {
                "adapter_type": "visdom_page_retrieval" if looks_like_visdom_dataset(request.dataset_profile) else "generic_page_retrieval",
                "retrieval_granularity": "page",
                "known_difficulties": list(request.dataset_profile.get("knownDifficulties", []) or []),
                "file_structure_snapshot": dict(request.dataset_profile.get("fileStructureSnapshot") or {}),
                "sample_statistics": dict(request.dataset_profile.get("sampleStatistics") or {}),
                "user_notes": request.user_notes,
            },
        },
        "evaluate": {
            "primary_metric": str(request.dataset_profile.get("officialMetric", "")).strip() or "recall@5",
            "ranking_metrics": ["recall@1", "recall@5", "recall@10", "mrr", "ndcg@10"],
            "secondary_metrics": ["index_build_ms", "avg_query_latency_ms", "peak_gpu_memory_mb", "failure_rate"],
            "report_sections": ["metrics", "prediction_summary", "constraints", "repair_history"],
        },
        "notes": [
            f"idea_title={idea_title}",
            f"task_definition={str(request.reader_context.get('task_definition', '')).strip()}",
            f"generation_mode={plan.execution_mode_used}",
        ],
    }


def build_retry_plan(max_retry_count: int) -> dict[str, Any]:
    if max_retry_count <= 0:
        max_retry_count = 3
    return {
        "max_retries": max_retry_count,
        "max_attempts": max_retry_count + 1,
        "repair_priority": [
            "runtime_error_first",
            "small_code_or_param_adjustment",
            "fallback_to_previous_stable_snapshot",
        ],
        "hooks": {
            "runtime_error_first": "rewrite_method_module_to_safe_page_baseline",
            "small_code_or_param_adjustment": "simplify_scoring_profile_and_query_expansion",
            "fallback_to_previous_stable_snapshot": "restore_snapshot_and_reapply_safe_method_module",
        },
    }


def render_method_module(plan: CodingGenerationPlan) -> str:
    rendered_tags = ", ".join(repr(item) for item in plan.method_tags)
    rendered_notes = ", ".join(repr(item) for item in plan.retrieval_notes)
    rendered_terms = ", ".join(repr(item) for item in plan.query_expansion_terms)
    return (
        "from methods.page_lexical_retrieval import PageLexicalRetrievalMethod\n\n"
        f"QUERY_EXPANSION_TERMS = [{rendered_terms}]\n"
        f"RETRIEVAL_NOTES = [{rendered_notes}]\n\n"
        "def build_method():\n"
        f"    return PageLexicalRetrievalMethod(name={plan.method_slug!r}, method_tags=[{rendered_tags}], "
        f"score_bias={plan.score_bias:.2f}, retrieval_notes=RETRIEVAL_NOTES, top_k={plan.top_k}, "
        f"query_expansion_terms=QUERY_EXPANSION_TERMS, title_match_bonus={plan.title_match_bonus:.2f}, "
        f"ocr_match_bonus={plan.ocr_match_bonus:.2f}, section_match_bonus={plan.section_match_bonus:.2f}, "
        f"exact_phrase_bonus={plan.exact_phrase_bonus:.2f})\n"
    )


def render_safe_fallback_module(method_slug: str, reason: str, repair_stage: str) -> str:
    notes = [f"fallback_reason={reason}", f"repair_stage={repair_stage}", "controlled_page_level_retrieval"]
    rendered_notes = ", ".join(repr(item) for item in notes)
    return (
        "from methods.page_lexical_retrieval import PageLexicalRetrievalMethod\n\n"
        "def build_method():\n"
        f"    return PageLexicalRetrievalMethod(name={method_slug!r}, method_tags=['fallback', 'page', 'lexical'], "
        f"score_bias=0.01, retrieval_notes=[{rendered_notes}], top_k=10, query_expansion_terms=[], "
        "title_match_bonus=0.25, ocr_match_bonus=0.15, section_match_bonus=0.12, exact_phrase_bonus=0.10)\n"
    )


def build_method_tags(request: Any) -> list[str]:
    raw = (
        list(request.idea.get("modelChanges", []) or [])
        + list(request.idea.get("dataProcessingNeeds", []) or [])
        + list(request.idea.get("evaluationMetrics", []) or [])
    )
    normalized = unique_phrases(raw)
    if not normalized:
        normalized = ["page-level", "retrieval", "baseline"]
    return normalized[:6]


def build_query_expansion_terms(request: Any) -> list[str]:
    phrases = []
    phrases.extend(extract_phrases(str(request.idea.get("title", ""))))
    phrases.extend(extract_phrases(str(request.idea.get("coreMethod", ""))))
    phrases.extend(extract_phrases(str(request.idea.get("problemDefinition", ""))))
    phrases.extend(flatten_text_values(request.reader_context.get("promising_research_directions")))
    phrases.extend(flatten_text_values(request.reader_context.get("relevant_methods_landscape")))
    phrases.extend(flatten_text_values(request.reader_context.get("likely_strong_baselines")))
    phrases.extend(flatten_text_values(request.dataset_profile.get("knownDifficulties")))
    return unique_phrases(phrases)[:8]


def build_scoring_profile(request: Any) -> tuple[float, float, float, float]:
    joined = " ".join(
        [
            str(request.idea.get("title", "")).lower(),
            str(request.idea.get("coreMethod", "")).lower(),
            " ".join(str(item).lower() for item in request.dataset_profile.get("knownDifficulties", []) or []),
            " ".join(str(item).lower() for item in request.reader_context.get("implementation_constraints", []) or []),
        ]
    )
    title_bonus = 0.25
    ocr_bonus = 0.15
    section_bonus = 0.12
    exact_phrase_bonus = 0.10
    if "layout" in joined or "section" in joined:
        title_bonus += 0.10
        section_bonus += 0.08
    if "ocr" in joined or "visual" in joined:
        ocr_bonus += 0.08
    if "hard negative" in joined or "contrastive" in joined:
        exact_phrase_bonus += 0.05
    return (round(title_bonus, 2), round(ocr_bonus, 2), round(section_bonus, 2), round(exact_phrase_bonus, 2))


def build_retrieval_notes(request: Any, response_text: str) -> list[str]:
    notes = [
        str(request.idea.get("title", "")).strip(),
        str(request.idea.get("coreMethod", "")).strip(),
        str(request.idea.get("trainingPlan", "")).strip(),
        str(request.reader_context.get("task_definition", "")).strip(),
        str(request.user_notes or "").strip(),
    ]
    if response_text.strip():
        notes.append(f"generator_trace={response_text.strip()[:180]}")
    return [item for item in notes if item]


def extract_phrases(value: str) -> list[str]:
    value = re.sub(r"[^a-zA-Z0-9\\-_/ ]+", " ", value or "")
    tokens = [item.strip().lower() for item in value.split() if item.strip()]
    candidates = []
    for item in tokens:
        if len(item) < 4:
            continue
        candidates.append(item)
    return candidates


def flatten_text_values(value: Any) -> list[str]:
    if isinstance(value, list):
        out: list[str] = []
        for item in value:
            text = str(item).strip()
            if text:
                out.extend(extract_phrases(text))
        return out
    text = str(value or "").strip()
    if not text:
        return []
    return extract_phrases(text)


def unique_phrases(items: list[str]) -> list[str]:
    out: list[str] = []
    seen: set[str] = set()
    for item in items:
        text = str(item or "").strip().lower()
        if not text or text in seen:
            continue
        seen.add(text)
        out.append(text)
    return out


def looks_like_visdom_dataset(dataset_profile: dict[str, Any]) -> bool:
    joined = " ".join(
        [
            str(dataset_profile.get("datasetName", "")).strip().lower(),
            str(dataset_profile.get("serverPath", "")).strip().lower(),
            str(dataset_profile.get("taskType", "")).strip().lower(),
        ]
    )
    return "visdom" in joined or "visual document" in joined
