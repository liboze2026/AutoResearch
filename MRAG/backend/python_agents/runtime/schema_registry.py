from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass
class SchemaField:
    name: str
    field_type: str
    required: bool = False
    default: Any = None
    aliases: list[str] = field(default_factory=list)


@dataclass
class SchemaDefinition:
    schema_name: str
    version: str
    agent_type: str
    schema_ref: str
    json_schema: dict[str, Any]
    python_schema_ref: str
    fields: list[SchemaField]


class SchemaRegistry:
    def __init__(self) -> None:
        self._generic = SchemaDefinition(
            schema_name="generic_agent_output",
            version="v1",
            agent_type="*",
            schema_ref="schemas/generic-agent-output-v1.json",
            json_schema={
                "type": "object",
                "required": [
                    "job_id",
                    "agent_type",
                    "execution_mode_requested",
                    "execution_mode_used",
                    "model_provider",
                    "model_name",
                    "prompt_version",
                    "output_schema_ref",
                    "workspace_dir",
                    "summary",
                    "items",
                    "data",
                    "metadata",
                ],
                "properties": {
                    "job_id": {"type": "string"},
                    "agent_type": {"type": "string"},
                    "execution_mode_requested": {"type": "string"},
                    "execution_mode_used": {"type": "string"},
                    "model_provider": {"type": "string"},
                    "model_name": {"type": "string"},
                    "prompt_version": {"type": "string"},
                    "output_schema_ref": {"type": "string"},
                    "workspace_dir": {"type": "string"},
                    "summary": {"type": "string"},
                    "items": {"type": "array"},
                    "data": {"type": "object"},
                    "metadata": {"type": "object"},
                },
            },
            python_schema_ref="runtime.schema_registry:generic_agent_output_v1",
            fields=[
                SchemaField("job_id", "string", required=True),
                SchemaField("agent_type", "string", required=True),
                SchemaField("execution_mode_requested", "string", required=True),
                SchemaField("execution_mode_used", "string", required=True),
                SchemaField("model_provider", "string", required=True, default=""),
                SchemaField("model_name", "string", required=True, default=""),
                SchemaField("prompt_version", "string", required=True, default=""),
                SchemaField("output_schema_ref", "string", required=True),
                SchemaField("workspace_dir", "string", required=True, default=""),
                SchemaField("summary", "string", required=True, default="", aliases=["message", "response_text", "text", "final_text", "content"]),
                SchemaField("items", "array", required=True, default=[], aliases=["results", "entries", "highlights", "bullets", "list"]),
                SchemaField("data", "object", required=True, default={}, aliases=["payload", "response_json", "result", "json", "object"]),
                SchemaField("metadata", "object", required=True, default={}, aliases=["meta", "extra"]),
            ],
        )
        self._reader = SchemaDefinition(
            schema_name="reader_output",
            version="v1",
            agent_type="reader",
            schema_ref="schemas/reader-output-v1.json",
            json_schema={
                **self._generic.json_schema,
                "required": list(self._generic.json_schema["required"]) + ["candidate_papers"],
                "properties": {
                    **self._generic.json_schema["properties"],
                    "candidate_papers": {"type": "array"},
                },
            },
            python_schema_ref="runtime.reader_agent:ReaderValidator",
            fields=[
                *[
                    SchemaField(
                        name=item.name,
                        field_type=item.field_type,
                        required=item.required,
                        default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                        aliases=list(item.aliases),
                    )
                    for item in self._generic.fields
                ],
                SchemaField("candidate_papers", "array", required=True, default=[], aliases=["candidatePapers", "papers"]),
            ],
        )
        self._reader_phase4 = SchemaDefinition(
            schema_name="reader_phase4_output",
            version="v1",
            agent_type="reader_phase4",
            schema_ref="schemas/reader-phase4-output-v1.json",
            json_schema={
                **self._generic.json_schema,
                "required": list(self._generic.json_schema["required"]) + [
                    "reading_summary",
                    "sources",
                    "reader_context",
                    "citation_metadata",
                ],
                "properties": {
                    **self._generic.json_schema["properties"],
                    "reading_summary": {"type": "string"},
                    "sources": {"type": "array"},
                    "reader_context": {"type": "object"},
                    "citation_metadata": {"type": "array"},
                },
            },
            python_schema_ref="runtime.reader_phase4_agent:ReaderPhase4Validator",
            fields=[
                *[
                    SchemaField(
                        name=item.name,
                        field_type=item.field_type,
                        required=item.required,
                        default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                        aliases=list(item.aliases),
                    )
                    for item in self._generic.fields
                ],
                SchemaField("reading_summary", "string", required=True, default="", aliases=["readingSummary"]),
                SchemaField("sources", "array", required=True, default=[], aliases=["reader_sources", "papers"]),
                SchemaField("reader_context", "object", required=True, default={}, aliases=["readerContext", "context"]),
                SchemaField("citation_metadata", "array", required=True, default=[], aliases=["citationMetadata"]),
            ],
        )
        self._insight = SchemaDefinition(
            schema_name="insight_output",
            version="v1",
            agent_type="insight",
            schema_ref="schemas/insight-output-v1.json",
            json_schema={
                **self._generic.json_schema,
                "required": list(self._generic.json_schema["required"]) + [
                    "summary_md",
                    "contributions_json",
                    "methods_json",
                    "limitations_json",
                    "novelty_points",
                ],
                "properties": {
                    **self._generic.json_schema["properties"],
                    "summary_md": {"type": "string"},
                    "contributions_json": {"type": "array"},
                    "methods_json": {"type": "array"},
                    "limitations_json": {"type": "array"},
                    "novelty_points": {"type": "array"},
                },
            },
            python_schema_ref="runtime.insight_agent:InsightValidator",
            fields=[
                *[
                    SchemaField(
                        name=item.name,
                        field_type=item.field_type,
                        required=item.required,
                        default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                        aliases=list(item.aliases),
                    )
                    for item in self._generic.fields
                ],
                SchemaField("summary_md", "string", required=True, default="", aliases=["summaryMd", "summary", "message"]),
                SchemaField("contributions_json", "array", required=True, default=[], aliases=["contributions", "contribution_points"]),
                SchemaField("methods_json", "array", required=True, default=[], aliases=["methods", "method_points"]),
                SchemaField("limitations_json", "array", required=True, default=[], aliases=["limitations", "limitation_points"]),
                SchemaField("novelty_points", "array", required=True, default=[], aliases=["novelty", "novelty_points_json"]),
            ],
        )
        self._dataset = SchemaDefinition(
            schema_name="dataset_output",
            version="v1",
            agent_type="dataset",
            schema_ref="schemas/dataset-output-v1.json",
            json_schema={
                **self._generic.json_schema,
                "required": list(self._generic.json_schema["required"]) + [
                    "dataset_asset_ref",
                    "dataset_location",
                    "eval_protocol_json",
                    "metric_schema_json",
                    "split_strategy",
                    "notes_md",
                ],
                "properties": {
                    **self._generic.json_schema["properties"],
                    "dataset_asset_ref": {"type": "string"},
                    "dataset_location": {"type": "string"},
                    "eval_protocol_json": {"type": "object"},
                    "metric_schema_json": {"type": "object"},
                    "split_strategy": {"type": "string"},
                    "notes_md": {"type": "string"},
                },
            },
            python_schema_ref="runtime.dataset_agent:DatasetValidator",
            fields=[
                *[
                    SchemaField(
                        name=item.name,
                        field_type=item.field_type,
                        required=item.required,
                        default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                        aliases=list(item.aliases),
                    )
                    for item in self._generic.fields
                ],
                SchemaField("dataset_asset_ref", "string", required=True, default="", aliases=["datasetAssetRef"]),
                SchemaField("dataset_location", "string", required=True, default="", aliases=["datasetLocation", "path"]),
                SchemaField("eval_protocol_json", "object", required=True, default={}, aliases=["eval_protocol", "evaluation_protocol", "protocol"]),
                SchemaField("metric_schema_json", "object", required=True, default={}, aliases=["metric_schema", "metrics"]),
                SchemaField("split_strategy", "string", required=True, default="", aliases=["data_split_strategy"]),
                SchemaField("notes_md", "string", required=True, default="", aliases=["notes", "note_md", "message"]),
            ],
        )
        self._idea_phase4 = SchemaDefinition(
            schema_name="idea_phase4_output",
            version="v1",
            agent_type="idea_phase4",
            schema_ref="schemas/idea-phase4-output-v1.json",
            json_schema={
                **self._generic.json_schema,
                "required": list(self._generic.json_schema["required"]) + [
                    "ideas",
                    "top_recommendations",
                    "generation_mode",
                ],
                "properties": {
                    **self._generic.json_schema["properties"],
                    "ideas": {"type": "array"},
                    "top_recommendations": {"type": "array"},
                    "generation_mode": {"type": "string"},
                },
            },
            python_schema_ref="runtime.idea_phase4_agent:IdeaPhase4Validator",
            fields=[
                *[
                    SchemaField(
                        name=item.name,
                        field_type=item.field_type,
                        required=item.required,
                        default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                        aliases=list(item.aliases),
                    )
                    for item in self._generic.fields
                ],
                SchemaField("ideas", "array", required=True, default=[], aliases=["idea_pool", "candidates"]),
                SchemaField("top_recommendations", "array", required=True, default=[], aliases=["topRecommendations", "recommended_top3"]),
                SchemaField("generation_mode", "string", required=True, default="new", aliases=["generationMode"]),
            ],
        )
        self._idea = SchemaDefinition(
            schema_name="idea_generator_output",
            version="v1",
            agent_type="idea_generator",
            schema_ref="schemas/idea-generator-output-v1.json",
            json_schema={
                **self._generic.json_schema,
                "required": list(self._generic.json_schema["required"]) + [
                    "title",
                    "description_md",
                    "research_direction",
                    "target_dataset_refs",
                    "dataset_eval_protocol_refs",
                    "innovation_type",
                    "expected_advantage",
                    "risk_points",
                    "priority",
                    "confidence",
                ],
                "properties": {
                    **self._generic.json_schema["properties"],
                    "title": {"type": "string"},
                    "description_md": {"type": "string"},
                    "research_direction": {"type": "string"},
                    "target_dataset_refs": {"type": "array"},
                    "dataset_eval_protocol_refs": {"type": "array"},
                    "innovation_type": {"type": "string"},
                    "expected_advantage": {"type": "string"},
                    "risk_points": {"type": "array"},
                    "priority": {"type": "number"},
                    "confidence": {"type": "number"},
                },
            },
            python_schema_ref="runtime.idea_agent:IdeaValidator",
            fields=[
                *[
                    SchemaField(
                        name=item.name,
                        field_type=item.field_type,
                        required=item.required,
                        default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                        aliases=list(item.aliases),
                    )
                    for item in self._generic.fields
                ],
                SchemaField("title", "string", required=True, default=""),
                SchemaField("description_md", "string", required=True, default="", aliases=["descriptionMd", "description"]),
                SchemaField("research_direction", "string", required=True, default="", aliases=["researchDirection", "direction"]),
                SchemaField("target_dataset_refs", "array", required=True, default=[], aliases=["targetDatasets", "dataset_refs"]),
                SchemaField("dataset_eval_protocol_refs", "array", required=True, default=[], aliases=["datasetEvalProtocolRefs", "evalplan_refs"]),
                SchemaField("innovation_type", "string", required=True, default="", aliases=["innovationType"]),
                SchemaField("expected_advantage", "string", required=True, default="", aliases=["expectedAdvantage"]),
                SchemaField("risk_points", "array", required=True, default=[], aliases=["riskPoints", "risks"]),
            ],
        )
        self._planner = SchemaDefinition(
            schema_name="planner_output",
            version="v1",
            agent_type="planner",
            schema_ref="schemas/planner-output-v1.json",
            json_schema={
                **self._generic.json_schema,
                "required": list(self._generic.json_schema["required"]) + [
                    "experiment_plan_json",
                    "train_template_type",
                    "resource_estimate",
                    "run_sequence",
                    "success_criteria",
                    "fallback_plan",
                ],
                "properties": {
                    **self._generic.json_schema["properties"],
                    "experiment_plan_json": {"type": "object"},
                    "train_template_type": {"type": "string"},
                    "resource_estimate": {"type": "object"},
                    "run_sequence": {"type": "array"},
                    "success_criteria": {"type": "object"},
                    "fallback_plan": {"type": "object"},
                },
            },
            python_schema_ref="runtime.planner_agent:PlannerValidator",
            fields=[
                *[
                    SchemaField(
                        name=item.name,
                        field_type=item.field_type,
                        required=item.required,
                        default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                        aliases=list(item.aliases),
                    )
                    for item in self._generic.fields
                ],
                SchemaField("experiment_plan_json", "object", required=True, default={}, aliases=["experimentPlanJson", "plan", "plan_json"]),
                SchemaField("train_template_type", "string", required=True, default="", aliases=["trainTemplateType", "template_type"]),
                SchemaField("resource_estimate", "object", required=True, default={}, aliases=["resourceEstimate"]),
                SchemaField("run_sequence", "array", required=True, default=[], aliases=["runSequence", "steps"]),
                SchemaField("success_criteria", "object", required=True, default={}, aliases=["successCriteria"]),
                SchemaField("fallback_plan", "object", required=True, default={}, aliases=["fallbackPlan"]),
            ],
        )
        self._coding = SchemaDefinition(
            schema_name="coding_output",
            version="v1",
            agent_type="coding",
            schema_ref="schemas/coding-output-v1.json",
            json_schema={
                **self._generic.json_schema,
                "required": list(self._generic.json_schema["required"]) + [
                    "code_patch_manifest",
                    "execution_result_ref",
                    "metrics_summary",
                    "evaluation_summary_md",
                ],
                "properties": {
                    **self._generic.json_schema["properties"],
                    "code_patch_manifest": {"type": "array"},
                    "execution_result_ref": {"type": "string"},
                    "metrics_summary": {"type": "object"},
                    "evaluation_summary_md": {"type": "string"},
                    "spec_overrides": {"type": "object"},
                },
            },
            python_schema_ref="runtime.coding_agent:CodingValidator",
            fields=[
                *[
                    SchemaField(
                        name=item.name,
                        field_type=item.field_type,
                        required=item.required,
                        default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                        aliases=list(item.aliases),
                    )
                    for item in self._generic.fields
                ],
                SchemaField("code_patch_manifest", "array", required=True, default=[], aliases=["patch_manifest", "patches", "codePatchManifest"]),
                SchemaField("execution_result_ref", "string", required=True, default="", aliases=["executionResultRef", "run_ref"]),
                SchemaField("metrics_summary", "object", required=True, default={}, aliases=["metricsSummary", "metrics"]),
                SchemaField("evaluation_summary_md", "string", required=True, default="", aliases=["evaluationSummaryMd", "evaluation_summary", "summary_md"]),
                SchemaField("spec_overrides", "object", required=False, default={}, aliases=["specOverrides"]),
            ],
        )
        self._coding_phase4 = SchemaDefinition(
            schema_name="coding_phase4_output",
            version="v1",
            agent_type="coding_phase4",
            schema_ref="schemas/coding-phase4-output-v1.json",
            json_schema={
                **self._generic.json_schema,
                "required": list(self._generic.json_schema["required"]) + [
                    "protocol_version",
                    "phase4_run_manifest",
                    "phase4_config",
                    "method_module",
                    "retry_plan",
                    "dataset_tool_assets",
                    "evaluate_tool_assets",
                    "entrypoints",
                ],
                "properties": {
                    **self._generic.json_schema["properties"],
                    "protocol_version": {"type": "string"},
                    "phase4_run_manifest": {"type": "object"},
                    "phase4_config": {"type": "object"},
                    "method_module": {"type": "object"},
                    "retry_plan": {"type": "object"},
                    "dataset_tool_assets": {"type": "object"},
                    "evaluate_tool_assets": {"type": "object"},
                    "entrypoints": {"type": "object"},
                },
            },
            python_schema_ref="runtime.coding_phase4_agent:CodingPhase4Validator",
            fields=[
                *[
                    SchemaField(
                        name=item.name,
                        field_type=item.field_type,
                        required=item.required,
                        default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                        aliases=list(item.aliases),
                    )
                    for item in self._generic.fields
                ],
                SchemaField("protocol_version", "string", required=True, default="phase4-retrieval-mainline-v1"),
                SchemaField("phase4_run_manifest", "object", required=True, default={}),
                SchemaField("phase4_config", "object", required=True, default={}),
                SchemaField("method_module", "object", required=True, default={}),
                SchemaField("retry_plan", "object", required=True, default={}),
                SchemaField("dataset_tool_assets", "object", required=True, default={}),
                SchemaField("evaluate_tool_assets", "object", required=True, default={}),
                SchemaField("entrypoints", "object", required=True, default={}),
            ],
        )
        self._writer = SchemaDefinition(
            schema_name="writer_output",
            version="v1",
            agent_type="writer",
            schema_ref="schemas/writer-output-v1.json",
            json_schema={
                **self._generic.json_schema,
                "required": list(self._generic.json_schema["required"]) + [
                    "title",
                    "abstract",
                    "introduction",
                    "method",
                    "experiments",
                    "conclusion",
                    "references_stub",
                    "figure_plan",
                ],
                "properties": {
                    **self._generic.json_schema["properties"],
                    "title": {"type": "string"},
                    "abstract": {"type": "string"},
                    "introduction": {"type": "string"},
                    "method": {"type": "string"},
                    "experiments": {"type": "string"},
                    "conclusion": {"type": "string"},
                    "references_stub": {"type": "array"},
                    "figure_plan": {"type": "array"},
                },
            },
            python_schema_ref="runtime.writer_agent:WriterValidator",
            fields=[
                *[
                    SchemaField(
                        name=item.name,
                        field_type=item.field_type,
                        required=item.required,
                        default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                        aliases=list(item.aliases),
                    )
                    for item in self._generic.fields
                ],
                SchemaField("title", "string", required=True, default=""),
                SchemaField("abstract", "string", required=True, default=""),
                SchemaField("introduction", "string", required=True, default="", aliases=["intro"]),
                SchemaField("method", "string", required=True, default="", aliases=["methods"]),
                SchemaField("experiments", "string", required=True, default="", aliases=["experiment_section"]),
                SchemaField("conclusion", "string", required=True, default=""),
                SchemaField("references_stub", "array", required=True, default=[], aliases=["referencesStub", "references"]),
                SchemaField("figure_plan", "array", required=True, default=[], aliases=["figurePlan", "figures"]),
            ],
        )
        self._writer_phase4 = SchemaDefinition(
            schema_name="writer_phase4_output",
            version="v1",
            agent_type="writer_phase4",
            schema_ref="schemas/writer-phase4-output-v1.json",
            json_schema={
                **self._generic.json_schema,
                "required": list(self._generic.json_schema["required"]) + [
                    "report_title",
                    "machine_readable_report",
                    "human_readable_report_md",
                    "citation_refs",
                    "reference_source_ids",
                ],
                "properties": {
                    **self._generic.json_schema["properties"],
                    "report_title": {"type": "string"},
                    "machine_readable_report": {"type": "object"},
                    "human_readable_report_md": {"type": "string"},
                    "citation_refs": {"type": "array"},
                    "reference_source_ids": {"type": "array"},
                },
            },
            python_schema_ref="runtime.writer_phase4_agent:WriterPhase4Validator",
            fields=[
                *[
                    SchemaField(
                        name=item.name,
                        field_type=item.field_type,
                        required=item.required,
                        default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                        aliases=list(item.aliases),
                    )
                    for item in self._generic.fields
                ],
                SchemaField("report_title", "string", required=True, default=""),
                SchemaField("machine_readable_report", "object", required=True, default={}),
                SchemaField("human_readable_report_md", "string", required=True, default="", aliases=["humanReadableReportMd"]),
                SchemaField("citation_refs", "array", required=True, default=[], aliases=["citationRefs"]),
                SchemaField("reference_source_ids", "array", required=True, default=[], aliases=["referenceSourceIds"]),
            ],
        )

    def resolve(self, schema_ref: str, agent_type: str) -> SchemaDefinition:
        schema_ref = (schema_ref or "").strip()
        agent_type = (agent_type or "").strip()
        if not schema_ref:
            raise ValueError("output_schema_ref is required")

        if schema_ref == self._reader_phase4.schema_ref or agent_type == "reader_phase4":
            return self._clone(self._reader_phase4, schema_ref or self._reader_phase4.schema_ref, agent_type or "reader_phase4")
        if schema_ref == self._reader.schema_ref or agent_type == "reader":
            return self._clone(self._reader, schema_ref or self._reader.schema_ref, agent_type or "reader")
        if schema_ref == self._insight.schema_ref or agent_type == "insight":
            return self._clone(self._insight, schema_ref or self._insight.schema_ref, agent_type or "insight")
        if schema_ref == self._dataset.schema_ref or agent_type == "dataset":
            return self._clone(self._dataset, schema_ref or self._dataset.schema_ref, agent_type or "dataset")
        if schema_ref == self._idea_phase4.schema_ref or agent_type == "idea_phase4":
            return self._clone(self._idea_phase4, schema_ref or self._idea_phase4.schema_ref, agent_type or "idea_phase4")
        if schema_ref == self._idea.schema_ref or agent_type == "idea_generator":
            return self._clone(self._idea, schema_ref or self._idea.schema_ref, agent_type or "idea_generator")
        if schema_ref == self._planner.schema_ref or agent_type == "planner":
            return self._clone(self._planner, schema_ref or self._planner.schema_ref, agent_type or "planner")
        if schema_ref == self._coding.schema_ref or agent_type == "coding":
            return self._clone(self._coding, schema_ref or self._coding.schema_ref, agent_type or "coding")
        if schema_ref == self._coding_phase4.schema_ref or agent_type == "coding_phase4":
            return self._clone(self._coding_phase4, schema_ref or self._coding_phase4.schema_ref, agent_type or "coding_phase4")
        if schema_ref == self._writer_phase4.schema_ref or agent_type == "writer_phase4":
            return self._clone(self._writer_phase4, schema_ref or self._writer_phase4.schema_ref, agent_type or "writer_phase4")
        if schema_ref == self._writer.schema_ref or agent_type == "writer":
            return self._clone(self._writer, schema_ref or self._writer.schema_ref, agent_type or "writer")
        entry = self._clone_generic(schema_ref, agent_type or "*")
        return entry

    def _clone(self, source: SchemaDefinition, schema_ref: str, agent_type: str) -> SchemaDefinition:
        return SchemaDefinition(
            schema_name=source.schema_name,
            version=source.version,
            agent_type=agent_type,
            schema_ref=schema_ref,
            json_schema=dict(source.json_schema),
            python_schema_ref=source.python_schema_ref,
            fields=[
                SchemaField(
                    name=item.name,
                    field_type=item.field_type,
                    required=item.required,
                    default=item.default if not isinstance(item.default, (list, dict)) else item.default.copy(),
                    aliases=list(item.aliases),
                )
                for item in source.fields
            ],
        )

    def _clone_generic(self, schema_ref: str, agent_type: str) -> SchemaDefinition:
        return self._clone(self._generic, schema_ref, agent_type)
