from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

try:
    from .reader_phase4_sources import Phase4DatasetProfileSnapshot
except ImportError:  # pragma: no cover - supports direct script execution
    from reader_phase4_sources import Phase4DatasetProfileSnapshot


@dataclass
class ReaderContextSnapshot:
    task_definition: str = ""
    dataset_specific_challenges: list[str] = field(default_factory=list)
    relevant_methods_landscape: list[str] = field(default_factory=list)
    likely_strong_baselines: list[str] = field(default_factory=list)
    common_failure_points: list[str] = field(default_factory=list)
    evaluation_caveats: list[str] = field(default_factory=list)
    implementation_constraints: list[str] = field(default_factory=list)
    promising_research_directions: list[str] = field(default_factory=list)
    citation_metadata: list[dict[str, Any]] = field(default_factory=list)
    reading_summary: str = ""
    user_notes: str = ""

    @classmethod
    def from_payload(cls, payload: dict[str, Any] | None) -> "ReaderContextSnapshot":
        payload = dict(payload or {})
        return cls(
            task_definition=str(payload.get("task_definition", payload.get("taskDefinition", ""))).strip(),
            dataset_specific_challenges=_string_list(payload.get("dataset_specific_challenges", payload.get("datasetSpecificChallenges", []))),
            relevant_methods_landscape=_string_list(payload.get("relevant_methods_landscape", payload.get("relevantMethodsLandscape", []))),
            likely_strong_baselines=_string_list(payload.get("likely_strong_baselines", payload.get("likelyStrongBaselines", []))),
            common_failure_points=_string_list(payload.get("common_failure_points", payload.get("commonFailurePoints", []))),
            evaluation_caveats=_string_list(payload.get("evaluation_caveats", payload.get("evaluationCaveats", []))),
            implementation_constraints=_string_list(payload.get("implementation_constraints", payload.get("implementationConstraints", []))),
            promising_research_directions=_string_list(payload.get("promising_research_directions", payload.get("promisingResearchDirections", []))),
            citation_metadata=[item for item in payload.get("citation_metadata", payload.get("citationMetadata", [])) if isinstance(item, dict)],
            reading_summary=str(payload.get("reading_summary", payload.get("readingSummary", ""))).strip(),
            user_notes=str(payload.get("user_notes", payload.get("userNotes", ""))).strip(),
        )


@dataclass
class IdeaSeedSnapshot:
    title: str = ""
    problem_definition: str = ""
    core_method: str = ""
    differentiators: str = ""
    data_processing_needs: list[str] = field(default_factory=list)
    model_changes: list[str] = field(default_factory=list)
    training_plan: str = ""
    evaluation_metrics: list[str] = field(default_factory=list)
    risk_points: list[str] = field(default_factory=list)
    expected_gains: list[str] = field(default_factory=list)
    source_type: str = ""
    revision_of_id: str = ""

    @classmethod
    def from_payload(cls, payload: dict[str, Any] | None) -> "IdeaSeedSnapshot":
        payload = dict(payload or {})
        return cls(
            title=str(payload.get("title", "")).strip(),
            problem_definition=str(payload.get("problem_definition", payload.get("problemDefinition", ""))).strip(),
            core_method=str(payload.get("core_method", payload.get("coreMethod", ""))).strip(),
            differentiators=str(payload.get("differentiators", "")).strip(),
            data_processing_needs=_string_list(payload.get("data_processing_needs", payload.get("dataProcessingNeeds", []))),
            model_changes=_string_list(payload.get("model_changes", payload.get("modelChanges", []))),
            training_plan=str(payload.get("training_plan", payload.get("trainingPlan", ""))).strip(),
            evaluation_metrics=_string_list(payload.get("evaluation_metrics", payload.get("evaluationMetrics", []))),
            risk_points=_string_list(payload.get("risk_points", payload.get("riskPoints", []))),
            expected_gains=_string_list(payload.get("expected_gains", payload.get("expectedGains", []))),
            source_type=str(payload.get("source_type", payload.get("sourceType", ""))).strip(),
            revision_of_id=str(payload.get("revision_of_id", payload.get("revisionOfId", ""))).strip(),
        )


@dataclass
class Phase4IdeaRequest:
    dataset_profile: Phase4DatasetProfileSnapshot
    reader_context: ReaderContextSnapshot
    user_notes: str = ""
    manual_idea: IdeaSeedSnapshot = field(default_factory=IdeaSeedSnapshot)
    generation_mode: str = "new"
    target_count: int = 10
    source_idea: IdeaSeedSnapshot = field(default_factory=IdeaSeedSnapshot)
    source_idea_id: str = ""
    failure_feedback: dict[str, Any] = field(default_factory=dict)
    last_failure_run_id: str = ""

    @classmethod
    def from_metadata(cls, metadata: dict[str, Any] | None) -> "Phase4IdeaRequest":
        metadata = dict(metadata or {})
        request = cls(
            dataset_profile=Phase4DatasetProfileSnapshot.from_payload(metadata.get("dataset_profile", metadata.get("datasetProfile"))),
            reader_context=ReaderContextSnapshot.from_payload(metadata.get("reader_context", metadata.get("readerContext"))),
            user_notes=str(metadata.get("user_notes", metadata.get("userNotes", ""))).strip(),
            manual_idea=IdeaSeedSnapshot.from_payload(metadata.get("manual_idea", metadata.get("manualIdea"))),
            generation_mode=str(metadata.get("generation_mode", metadata.get("generationMode", "new"))).strip().lower() or "new",
            target_count=_normalize_target_count(metadata.get("target_count", metadata.get("targetCount", 10)), 10),
            source_idea=IdeaSeedSnapshot.from_payload(metadata.get("source_idea", metadata.get("sourceIdea"))),
            source_idea_id=str(metadata.get("source_idea_id", metadata.get("sourceIdeaId", ""))).strip(),
            failure_feedback=dict(metadata.get("failure_feedback", metadata.get("failureFeedback", {})) or {}),
            last_failure_run_id=str(metadata.get("last_failure_run_id", metadata.get("lastFailureRunId", ""))).strip(),
        )
        if not request.dataset_profile.dataset_name:
            request.dataset_profile.dataset_name = "Unknown Dataset"
        return request

    def effective_target_count(self) -> int:
        if self.generation_mode == "revision":
            return _normalize_target_count(self.target_count, 3)
        return _normalize_target_count(self.target_count, 10)


@dataclass
class IdeaCandidate:
    title: str
    problem_definition: str
    core_method: str
    differentiators: str
    data_processing_needs: list[str]
    model_changes: list[str]
    training_plan: str
    evaluation_metrics: list[str]
    risk_points: list[str]
    expected_gains: list[str]
    score: dict[str, float] = field(default_factory=dict)
    score_summary: dict[str, Any] = field(default_factory=dict)
    status: str = "scored"
    source_type: str = "idea_phase4_generated"
    revision_of_id: str = ""
    failure_feedback: dict[str, Any] = field(default_factory=dict)
    last_failure_run_id: str = ""

    def to_payload(self) -> dict[str, Any]:
        return {
            "title": self.title,
            "problem_definition": self.problem_definition,
            "core_method": self.core_method,
            "differentiators": self.differentiators,
            "data_processing_needs": list(self.data_processing_needs),
            "model_changes": list(self.model_changes),
            "training_plan": self.training_plan,
            "evaluation_metrics": list(self.evaluation_metrics),
            "risk_points": list(self.risk_points),
            "expected_gains": list(self.expected_gains),
            "score": dict(self.score),
            "score_summary": dict(self.score_summary),
            "status": self.status,
            "source_type": self.source_type,
            "revision_of_id": self.revision_of_id,
            "failure_feedback": dict(self.failure_feedback),
            "last_failure_run_id": self.last_failure_run_id,
        }


BASE_TEMPLATES: list[dict[str, Any]] = [
    {
        "key": "layout_hard_negative",
        "title": "Layout-Aware Hard Negative Mining",
        "focus": "Use layout-aware hard negatives so visually similar but semantically wrong pages stop dominating first-stage recall.",
        "core": "Add layout-aware hard negative mining inside page retrieval training and mine negatives from same-document or same-template pages.",
        "data": [
            "Build page-pair mining lists from document ids and layout similarity.",
            "Cache OCR confidence and page structure metadata for negative sampling.",
        ],
        "model": [
            "Keep the current dual-encoder backbone.",
            "Add a lightweight layout-aware projection or layout token fusion head.",
        ],
        "training": "Warm start from the strongest page-level retriever, then introduce progressively harder same-template negatives over two curriculum stages.",
        "gain": [
            "Better recall on visually similar distractor pages.",
            "Improved robustness against template overlap and dense layout noise.",
        ],
        "risk": ["Aggressive hard negative mining can destabilize early training."],
    },
    {
        "key": "two_stage_rerank",
        "title": "Two-Stage Page Recall with Late Interaction Reranking",
        "focus": "Keep efficient first-stage page recall but rerank only the shortlist with region-sensitive late interaction.",
        "core": "Train a two-stage retriever where a fast page encoder proposes candidates and a late-interaction reranker refines top-k pages.",
        "data": [
            "Create stable top-k candidate caches per split for reranker supervision.",
            "Record page regions or OCR spans needed for reranking inputs.",
        ],
        "model": [
            "Retain the existing first-stage retriever.",
            "Add a ColBERT-style late-interaction module over page tokens or OCR spans.",
        ],
        "training": "Freeze or slowly update the first-stage encoder, then train the reranker on shortlist pages with recall-oriented loss and hard negatives.",
        "gain": [
            "Higher top-k precision without paying cross-encoder cost on the full index.",
            "Better handling of pages with small but critical evidence regions.",
        ],
        "risk": ["Reranker latency may hide gains if shortlist size is not controlled."],
    },
    {
        "key": "query_conditioned_chunking",
        "title": "Query-Conditioned Page Chunking",
        "focus": "Represent each page by a small set of query-sensitive chunks instead of a single global page embedding.",
        "core": "Generate query-conditioned page chunks or region proposals so page retrieval remains page-level but the representation highlights evidence-bearing areas.",
        "data": [
            "Precompute page chunk candidates aligned with OCR spans or layout blocks.",
            "Store chunk-to-page mappings for evaluation and debugging.",
        ],
        "model": [
            "Add a query-conditioned chunk scoring layer before page aggregation.",
            "Aggregate top scoring chunks back to page-level retrieval scores.",
        ],
        "training": "Train chunk scoring jointly with page-level supervision using chunk-drop regularization to avoid overfitting to small regions.",
        "gain": [
            "Better recall when the answer signal lives in a small page region.",
            "Cleaner path toward later snippet-level retrieval extension.",
        ],
        "risk": ["Chunk generation quality can bottleneck the whole method."],
    },
    {
        "key": "ocr_confidence_fusion",
        "title": "OCR-Confidence-Aware Text-Image Fusion",
        "focus": "Downweight noisy OCR while preserving layout and image cues for page retrieval.",
        "core": "Fuse OCR confidence, text embeddings, and page visual embeddings so low-quality OCR does not overwhelm retrieval scores.",
        "data": [
            "Expose OCR confidence statistics in the dataset adapter.",
            "Normalize text coverage and OCR quality per page.",
        ],
        "model": [
            "Add confidence-aware gating between OCR text and visual features.",
            "Condition page fusion weights on OCR quality signals.",
        ],
        "training": "Use confidence-aware dropout on noisy text spans and train with mixed clean/noisy OCR batches.",
        "gain": [
            "More stable retrieval when OCR is sparse or corrupted.",
            "Lower reliance on brittle text-only signals.",
        ],
        "risk": ["OCR quality metadata may be inconsistent across splits."],
    },
    {
        "key": "cross_page_neighborhood",
        "title": "Cross-Page Neighborhood Negative Sampling",
        "focus": "Treat nearby pages and same-document neighbors as structured negatives instead of random global negatives.",
        "core": "Model document-level page neighborhoods so retrieval learns to separate truly relevant pages from adjacent distractors within the same document.",
        "data": [
            "Index page order, document membership, and neighboring page ids.",
            "Create neighbor-aware negative sampling tables in the dataset profile.",
        ],
        "model": [
            "Keep the backbone unchanged.",
            "Inject neighborhood-aware sampling weights and a page-distance regularizer.",
        ],
        "training": "Alternate between global negatives and same-document neighborhood negatives with ratio scheduling by epoch.",
        "gain": [
            "Improved discrimination among near-duplicate pages from the same document.",
            "Minimal model surgery compared with stronger baselines.",
        ],
        "risk": ["Neighbor negatives may be too hard early in training and slow convergence."],
    },
    {
        "key": "section_token_fusion",
        "title": "Section-Token Late Fusion Retriever",
        "focus": "Build page embeddings from a small number of section tokens aligned with layout regions instead of one global page vector.",
        "core": "Represent each page by section tokens from titles, tables, and dense text blocks, then late-fuse them at scoring time.",
        "data": [
            "Split each page into stable layout sections.",
            "Export section token masks for training and evaluation.",
        ],
        "model": [
            "Add section-token pooling on top of the page encoder.",
            "Fuse section-token scores with page-level retrieval scores.",
        ],
        "training": "Train section-token pooling jointly with page retrieval, then ablate fusion weights on the validation split.",
        "gain": [
            "Better handling of heterogeneous page layouts and long pages.",
            "Interpretable debugging through section-level scores.",
        ],
        "risk": ["Section segmentation quality may vary across document templates."],
    },
    {
        "key": "metric_aligned_distillation",
        "title": "Metric-Aligned Recall Distillation",
        "focus": "Distill a stronger but slower teacher into a first-stage retriever optimized directly for the dataset recall metric.",
        "core": "Use a teacher reranker or ensemble to create metric-aligned soft targets, then distill them into a lightweight page retriever.",
        "data": [
            "Persist teacher scores for train and validation queries.",
            "Track metric-aligned target distributions in artifacts for regression checks.",
        ],
        "model": [
            "Keep a small retriever student.",
            "Use teacher-guided distillation loss aligned with Recall@k or MRR-style ranking.",
        ],
        "training": "Generate teacher scores offline, then train the student with a mix of contrastive loss and metric-aligned distillation loss.",
        "gain": [
            "Potentially better recall without paying teacher cost at inference.",
            "Cleaner compatibility with the unified retrieval mainline.",
        ],
        "risk": ["Teacher quality caps the student and adds pipeline complexity."],
    },
    {
        "key": "template_debias",
        "title": "Template-Debias Contrastive Retrieval",
        "focus": "Penalize retrieval shortcuts caused by repeated document templates and boilerplate regions.",
        "core": "Introduce template-debias regularization so the retriever cannot win by matching headers, footers, and repeated layouts alone.",
        "data": [
            "Cluster repeated templates or boilerplate-heavy pages.",
            "Annotate template groups in dataset statistics for training-time balancing.",
        ],
        "model": [
            "Keep the base retriever.",
            "Add a template-adversarial or group-balanced loss term.",
        ],
        "training": "Balance batches across template groups and penalize overconfident template-only matches on hard negatives.",
        "gain": [
            "Better generalization across unseen document templates.",
            "Lower false positives on boilerplate-heavy collections.",
        ],
        "risk": ["Template clustering noise can weaken the debias signal."],
    },
    {
        "key": "multi_scale_page_pyramid",
        "title": "Multi-Scale Page Pyramid Retrieval",
        "focus": "Encode each page at multiple image scales so fine evidence and global layout are both visible to retrieval.",
        "core": "Build a multi-scale page pyramid and learn scale-aware aggregation for first-stage retrieval embeddings.",
        "data": [
            "Cache page renders at multiple resolutions under the dataset adapter.",
            "Track per-scale memory and preprocessing costs for reproducibility.",
        ],
        "model": [
            "Share the base encoder across scales with lightweight aggregation.",
            "Learn scale weights conditioned on query type or OCR density.",
        ],
        "training": "Train with mixed-scale augmentation and enforce consistent page ranking across scales.",
        "gain": [
            "Higher sensitivity to small visual evidence without losing page-level context.",
            "Potential gains on tables, charts, and dense evidence blocks.",
        ],
        "risk": ["Multi-scale inputs can push GPU memory and batch time beyond budget."],
    },
    {
        "key": "hybrid_uncertainty_gate",
        "title": "Hybrid Lexical-Dense Retrieval with Uncertainty Gating",
        "focus": "Blend lexical and dense retrieval only when the dense model is uncertain, keeping the system simple and robust.",
        "core": "Combine a dense page retriever with a lexical/OCR baseline using an uncertainty gate estimated from embedding confidence and OCR coverage.",
        "data": [
            "Store lexical baseline scores alongside dense retrieval outputs.",
            "Compute OCR coverage and confidence features per page.",
        ],
        "model": [
            "Keep both lexical and dense retrievers intact.",
            "Add a lightweight uncertainty gate for score fusion.",
        ],
        "training": "Fit the uncertainty gate on validation-like folds after training the dense retriever, then freeze fusion for final evaluation.",
        "gain": [
            "Strong fallback behavior on OCR-heavy or noisy queries.",
            "High engineering practicality with limited new model risk.",
        ],
        "risk": ["Fusion gains may saturate if the dense retriever is already strong."],
    },
]


REVISION_TEMPLATES: dict[str, list[dict[str, Any]]] = {
    "resource": [
        {
            "modifier": "Memory-Capped Retriever Revision",
            "core_shift": "Constrain image resolution, freeze more of the backbone, and add cached page embeddings to avoid GPU or wall-time failures.",
            "training_shift": "Train in two passes: first cache page features, then fine-tune only the lightweight retrieval head.",
            "data_shift": ["Materialize page caches and reproducible feature shards before full training."],
            "model_shift": ["Freeze the encoder backbone and train only the projection or reranking head."],
            "risk_shift": ["May trade off some peak recall for a stable training budget."],
            "gain_shift": ["Much lower OOM risk and faster repair loops on busy GPUs."],
        },
        {
            "modifier": "Shortlist-First Efficiency Revision",
            "core_shift": "Move expensive scoring behind a small shortlist so the original method still survives under a tighter runtime envelope.",
            "training_shift": "Train a cheap candidate generator first, then distill expensive scoring onto the shortlist.",
            "data_shift": ["Persist shortlist caches to keep repeated experiments deterministic."],
            "model_shift": ["Replace full-index expensive scoring with shortlist-only reranking."],
            "risk_shift": ["Shortlist recall may bottleneck final performance if too small."],
            "gain_shift": ["Preserves core method intent while lowering compute cost sharply."],
        },
    ],
    "data": [
        {
            "modifier": "Adapter-First Robustness Revision",
            "core_shift": "Reduce moving parts by stabilizing dataset adapter assumptions before adding method-specific complexity.",
            "training_shift": "Start with a schema-validated baseline run, then re-enable the original method pieces incrementally.",
            "data_shift": ["Simplify page schema, path handling, and split assertions with explicit adapter checks."],
            "model_shift": ["Temporarily narrow the model delta to the smallest reproducible retrieval head."],
            "risk_shift": ["Innovation depth is reduced until adapter issues are removed."],
            "gain_shift": ["Higher chance of getting a valid end-to-end run and clearer debugging signals."],
        }
    ],
    "quality": [
        {
            "modifier": "Harder-Negative Recall Revision",
            "core_shift": "Keep the original direction but strengthen supervision where recall failed, especially on near-duplicate pages.",
            "training_shift": "Increase hard-negative ratio gradually and monitor Recall@k after each curriculum stage.",
            "data_shift": ["Mine failure-focused hard negatives from the failed run traces."],
            "model_shift": ["Add a lightweight reranking or calibration layer only if the baseline retriever saturates."],
            "risk_shift": ["Additional hard negatives can create unstable loss if introduced too early."],
            "gain_shift": ["Targets the observed failure mode instead of replacing the whole idea."],
        },
        {
            "modifier": "Template-Debias Quality Revision",
            "core_shift": "Directly suppress shortcut matches that likely caused poor retrieval quality in the failed run.",
            "training_shift": "Rebalance batches around failure-heavy template groups and repeated layouts.",
            "data_shift": ["Tag repeated templates from failed queries and promote them during sampling."],
            "model_shift": ["Add a small template-debias regularizer rather than a full new backbone."],
            "risk_shift": ["Template grouping quality matters for the revision to help."],
            "gain_shift": ["Improves robustness on the exact failure slices surfaced by the failed run."],
        },
    ],
    "stability": [
        {
            "modifier": "Curriculum-Stabilized Revision",
            "core_shift": "Keep the original architecture but spread difficulty across a staged training schedule to avoid collapse.",
            "training_shift": "Move from easy negatives to hard negatives and unlock modules gradually after validation gates.",
            "data_shift": ["Track curriculum stage boundaries and failure slices as explicit artifacts."],
            "model_shift": ["Delay the noisiest auxiliary losses until the base retriever is stable."],
            "risk_shift": ["Longer schedule can increase experiment turnaround time."],
            "gain_shift": ["Reduces train instability without discarding the idea."],
        }
    ],
}


def generate_structured_ideas(request: Phase4IdeaRequest) -> list[IdeaCandidate]:
    if request.generation_mode == "revision":
        return _generate_revision_candidates(request)
    return _generate_batch_candidates(request)


def score_and_rank_ideas(items: list[IdeaCandidate], request: Phase4IdeaRequest) -> list[IdeaCandidate]:
    ranked: list[IdeaCandidate] = []
    for candidate in items:
        candidate.score = score_idea_candidate(candidate, request)
        candidate.score_summary = build_score_summary(candidate, request)
        ranked.append(candidate)
    ranked.sort(
        key=lambda item: (
            float(item.score_summary.get("overallScore", 0.0)),
            float(item.score.get("datasetFit", 0.0)),
            float(item.score.get("expectedGain", 0.0)),
            float(item.score.get("feasibility", 0.0)),
        ),
        reverse=True,
    )
    for index, candidate in enumerate(ranked):
        candidate.score_summary["rank"] = index + 1
        candidate.score_summary["recommended"] = index < 3
        candidate.score_summary["recommendationTier"] = "top3" if index < 3 else "candidate"
        candidate.score_summary["recommendationReason"] = recommendation_reason(candidate, index)
    return ranked


def build_top_recommendations(items: list[IdeaCandidate]) -> list[dict[str, Any]]:
    out: list[dict[str, Any]] = []
    for candidate in items[:3]:
        out.append(
            {
                "title": candidate.title,
                "overallScore": candidate.score_summary.get("overallScore", 0.0),
                "rank": candidate.score_summary.get("rank", 0),
                "recommendationReason": candidate.score_summary.get("recommendationReason", ""),
                "score": dict(candidate.score),
            }
        )
    return out


def score_idea_candidate(candidate: IdeaCandidate, request: Phase4IdeaRequest) -> dict[str, float]:
    text_blob = " ".join(
        [
            candidate.problem_definition,
            candidate.core_method,
            candidate.differentiators,
            " ".join(candidate.data_processing_needs),
            " ".join(candidate.model_changes),
            candidate.training_plan,
            " ".join(candidate.evaluation_metrics),
            " ".join(candidate.expected_gains),
        ]
    ).lower()
    dataset_name = request.dataset_profile.dataset_name.lower()
    reader_context = request.reader_context
    dataset_fit = _fit_score(text_blob, dataset_name, reader_context)
    novelty = _bounded_score(
        5.8
        + 1.2 * _contains_any(text_blob, ("query-conditioned", "template-debias", "multi-scale", "distill", "late interaction"))
        + 1.0 * _contains_any(text_blob, ("layout-aware", "hard negative", "uncertainty gate"))
        + 0.5 * _contains_any(request.user_notes.lower(), ("novel", "new", "different")),
    )
    feasibility = _bounded_score(
        8.8
        - 0.7 * len(candidate.model_changes)
        - 0.5 * _contains_any(text_blob, ("multi-scale", "teacher", "distill", "rerank"))
        - 0.6 * _contains_any(text_blob, ("query-conditioned chunk", "region-sensitive"))
    )
    expected_gain = _bounded_score(
        6.9
        + 1.1 * _contains_any(text_blob, ("hard negative", "late interaction", "query-conditioned", "template-debias"))
        + 0.8 * _contains_any(text_blob, ("recall", "page-level", "evidence"))
    )
    compute_cost = _bounded_score(
        3.2
        + 1.5 * _contains_any(text_blob, ("multi-scale", "teacher", "rerank"))
        + 1.1 * _contains_any(text_blob, ("late interaction", "distill"))
        + 0.8 * max(len(candidate.model_changes) - 2, 0),
    )
    failure_risk = _bounded_score(
        3.0
        + 0.9 * _contains_any(text_blob, ("multi-scale", "chunk", "teacher"))
        + 0.8 * _contains_any(text_blob, ("hard negative", "curriculum"))
        + 0.7 * _contains_any(" ".join(candidate.risk_points).lower(), ("unstable", "oom", "adapter", "latency")),
    )
    reproducibility = _bounded_score(
        8.8
        - 0.7 * _contains_any(text_blob, ("teacher", "query-conditioned", "multi-scale"))
        - 0.5 * max(len(candidate.data_processing_needs) - 2, 0),
    )
    return {
        "novelty": round(novelty, 3),
        "datasetFit": round(dataset_fit, 3),
        "feasibility": round(feasibility, 3),
        "expectedGain": round(expected_gain, 3),
        "computeCost": round(compute_cost, 3),
        "failureRisk": round(failure_risk, 3),
        "reproducibility": round(reproducibility, 3),
    }


def build_score_summary(candidate: IdeaCandidate, request: Phase4IdeaRequest) -> dict[str, Any]:
    score = candidate.score
    overall = (
        0.22 * score["novelty"]
        + 0.24 * score["datasetFit"]
        + 0.18 * score["feasibility"]
        + 0.20 * score["expectedGain"]
        + 0.16 * score["reproducibility"]
        - 0.12 * score["computeCost"]
        - 0.15 * score["failureRisk"]
    )
    return {
        "overallScore": round(max(0.0, min(10.0, overall)), 4),
        "generationMode": request.generation_mode,
        "sourceSignals": {
            "datasetName": request.dataset_profile.dataset_name,
            "metric": request.dataset_profile.official_metric,
            "taskDefinition": request.reader_context.task_definition,
            "failureClass": classify_failure_feedback(request.failure_feedback),
        },
    }


def recommendation_reason(candidate: IdeaCandidate, index: int) -> str:
    score = candidate.score
    if index == 0:
        return f"Best balance of dataset fit ({score['datasetFit']:.1f}), expected gain ({score['expectedGain']:.1f}), and feasible engineering scope."
    if score["feasibility"] >= 8.5:
        return "High feasibility with limited model surgery, making it a strong early implementation candidate."
    if score["novelty"] >= 8.2:
        return "High novelty while still staying close enough to the dataset constraints to be worth a controlled run."
    return "Recommended because it stays aligned with the Reader context and offers a clear experimental path for Coding."


def classify_failure_feedback(feedback: dict[str, Any]) -> str:
    text = " ".join(f"{key} {value}" for key, value in dict(feedback or {}).items()).lower()
    if any(token in text for token in ("oom", "cuda", "gpu", "memory", "timeout", "slow")):
        return "resource"
    if any(token in text for token in ("adapter", "schema", "split", "path", "dataset", "dataloader")):
        return "data"
    if any(token in text for token in ("low recall", "metric", "retrieval quality", "poor recall", "top-k")):
        return "quality"
    if any(token in text for token in ("nan", "unstable", "collapse", "diverge", "exploding")):
        return "stability"
    return "quality"


def _generate_batch_candidates(request: Phase4IdeaRequest) -> list[IdeaCandidate]:
    target_count = request.effective_target_count()
    citations = _citation_titles(request.reader_context.citation_metadata)
    metric = _first_non_empty(request.dataset_profile.official_metric, "Recall@10")
    challenges = request.reader_context.dataset_specific_challenges or request.dataset_profile.known_difficulties
    outputs: list[IdeaCandidate] = []
    for template in BASE_TEMPLATES[:target_count]:
        outputs.append(
            IdeaCandidate(
                title=f"{request.dataset_profile.dataset_name}: {template['title']}",
                problem_definition=_build_problem_definition(request, template["focus"]),
                core_method=_build_core_method(template["core"]),
                differentiators=_build_differentiators(request, template["focus"], citations),
                data_processing_needs=_merge_unique(template["data"], [f"Preserve split-consistent page ids for {request.dataset_profile.dataset_name}.", *challenges[:1]]),
                model_changes=_merge_unique(template["model"], ["Keep the retrieval mainline centered on page-level ranking before any snippet extension."]),
                training_plan=_build_training_plan(template["training"]),
                evaluation_metrics=_merge_unique([metric, "Recall@1", "Recall@5", "MRR"], request.reader_context.evaluation_caveats[:1]),
                risk_points=_merge_unique(template["risk"], request.reader_context.common_failure_points[:1]),
                expected_gains=_merge_unique(template["gain"], [f"Should improve page-level retrieval quality on {request.dataset_profile.dataset_name}."]),
                status="scored",
                source_type="idea_phase4_generated",
            )
        )
    return outputs


def _generate_revision_candidates(request: Phase4IdeaRequest) -> list[IdeaCandidate]:
    failure_class = classify_failure_feedback(request.failure_feedback)
    templates = list(REVISION_TEMPLATES.get(failure_class, REVISION_TEMPLATES["quality"]))
    if len(templates) < request.effective_target_count():
        for fallback_key, fallback_items in REVISION_TEMPLATES.items():
            if fallback_key == failure_class:
                continue
            for fallback in fallback_items:
                templates.append(fallback)
                if len(templates) >= request.effective_target_count():
                    break
            if len(templates) >= request.effective_target_count():
                break
    source = request.source_idea
    metric = _first_non_empty(request.dataset_profile.official_metric, "Recall@10")
    outputs: list[IdeaCandidate] = []
    for template in templates[: request.effective_target_count()]:
        title = _first_non_empty(source.title, request.manual_idea.title, f"{request.dataset_profile.dataset_name} revision")
        outputs.append(
            IdeaCandidate(
                title=f"{title} - {template['modifier']}",
                problem_definition=_first_non_empty(source.problem_definition, request.reader_context.task_definition, f"Repair the failed retrieval idea for {request.dataset_profile.dataset_name} without discarding its core direction."),
                core_method=_merge_sentence(source.core_method, template["core_shift"]),
                differentiators=_merge_sentence(_first_non_empty(source.differentiators, "Revision candidate targeted at the observed failed run."), f"This revision directly addresses failure class `{failure_class}` rather than replacing the original research direction."),
                data_processing_needs=_merge_unique(source.data_processing_needs, template["data_shift"]),
                model_changes=_merge_unique(source.model_changes, template["model_shift"]),
                training_plan=_merge_sentence(_first_non_empty(source.training_plan, "Stabilize the failed run before widening scope."), template["training_shift"]),
                evaluation_metrics=_merge_unique(source.evaluation_metrics or [metric, "Recall@10"], ["Repair success rate", "Same-split reproducibility"]),
                risk_points=_merge_unique(source.risk_points, template["risk_shift"]),
                expected_gains=_merge_unique(source.expected_gains, template["gain_shift"]),
                status="scored",
                source_type="revision_candidate",
                revision_of_id=_first_non_empty(request.source_idea_id, source.revision_of_id),
                failure_feedback=dict(request.failure_feedback),
                last_failure_run_id=request.last_failure_run_id,
            )
        )
    return outputs


def _fit_score(text_blob: str, dataset_name: str, reader_context: ReaderContextSnapshot) -> float:
    score = 6.0
    if dataset_name and dataset_name in text_blob:
        score += 1.2
    if _contains_any(text_blob, ("page-level", "page retrieval", "visdom", "document retrieval")):
        score += 1.3
    if _contains_any(text_blob, ("ocr", "layout", "visual", "page")):
        score += 0.7
    if _contains_any(" ".join(reader_context.dataset_specific_challenges).lower(), ("ocr", "layout", "template", "dense")) and _contains_any(text_blob, ("ocr", "layout", "template", "chunk")):
        score += 0.8
    if _contains_any(" ".join(reader_context.promising_research_directions).lower(), ("rerank", "hard negative", "page", "chunk")) and _contains_any(text_blob, ("rerank", "hard negative", "page", "chunk")):
        score += 0.6
    return _bounded_score(score)


def _build_problem_definition(request: Phase4IdeaRequest, focus: str) -> str:
    task_definition = _first_non_empty(
        request.reader_context.task_definition,
        f"Improve page-level retrieval recall on {request.dataset_profile.dataset_name}.",
    )
    challenge = _first_non_empty(
        *(request.reader_context.dataset_specific_challenges or request.dataset_profile.known_difficulties),
        "The dataset contains visually rich pages with layout and OCR ambiguity.",
    )
    return f"{task_definition} The concrete problem is to {focus.lower()} while accounting for {challenge.lower()}."


def _build_core_method(core: str) -> str:
    core = core.strip()
    if not core.endswith("."):
        core += "."
    return core


def _build_differentiators(request: Phase4IdeaRequest, focus: str, citations: list[str]) -> str:
    citation_hint = ""
    if citations:
        citation_hint = f" It stays grounded in signals seen in {', '.join(citations[:2])} but tightens them around {request.dataset_profile.dataset_name}."
    manual_hint = ""
    if request.manual_idea.title:
        manual_hint = f" It also incorporates the user-provided direction `{request.manual_idea.title}` where compatible."
    return f"This idea is explicitly optimized for phase4 page-level retrieval and emphasizes {focus.lower()} over generic multimodal RAG heuristics.{citation_hint}{manual_hint}".strip()


def _build_training_plan(training: str) -> str:
    training = training.strip()
    if training.endswith("."):
        return training
    return training + "."


def _citation_titles(items: list[dict[str, Any]]) -> list[str]:
    titles: list[str] = []
    for item in items:
        if not isinstance(item, dict):
            continue
        title = _first_non_empty(str(item.get("title", "")).strip(), str(item.get("paper_title", "")).strip())
        if title:
            titles.append(title)
    return _merge_unique(titles, [])


def _merge_sentence(left: str, right: str) -> str:
    left = left.strip()
    right = right.strip()
    if not left:
        return right
    if not right:
        return left
    if not left.endswith("."):
        left += "."
    return f"{left} {right}".strip()


def _contains_any(text: str, needles: tuple[str, ...]) -> float:
    text = (text or "").lower()
    return 1.0 if any(needle in text for needle in needles) else 0.0


def _bounded_score(value: float) -> float:
    return max(0.0, min(10.0, round(float(value), 4)))


def _merge_unique(primary: list[str], secondary: list[str]) -> list[str]:
    items: list[str] = []
    seen: set[str] = set()
    for source in (primary or [], secondary or []):
        for item in source if isinstance(source, list) else []:
            text = str(item).strip()
            if not text:
                continue
            key = text.lower()
            if key in seen:
                continue
            seen.add(key)
            items.append(text)
    return items


def _normalize_target_count(value: Any, default: int) -> int:
    try:
        numeric = int(value)
    except (TypeError, ValueError):
        numeric = default
    if default >= 10:
        return max(1, min(10, numeric))
    return max(1, min(5, numeric))


def _string_list(value: Any) -> list[str]:
    if value is None:
        return []
    if isinstance(value, list):
        out: list[str] = []
        seen: set[str] = set()
        for item in value:
            text = str(item).strip()
            if not text:
                continue
            key = text.lower()
            if key in seen:
                continue
            seen.add(key)
            out.append(text)
        return out
    text = str(value).strip()
    return [text] if text else []


def _first_non_empty(*values: str) -> str:
    for value in values:
        text = str(value).strip()
        if text:
            return text
    return ""
