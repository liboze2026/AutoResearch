from __future__ import annotations

import json
import re
from typing import Any

try:
    from .contract import AgentRepairAction, AgentRuntimeInput
    from .schema_registry import SchemaDefinition, SchemaRegistry
except ImportError:  # pragma: no cover - supports direct script execution
    from contract import AgentRepairAction, AgentRuntimeInput
    from schema_registry import SchemaDefinition, SchemaRegistry


CODE_BLOCK_PATTERN = re.compile(r"```(?:json)?\s*(.*?)```", re.IGNORECASE | re.DOTALL)
JSON_OBJECT_PATTERN = re.compile(r"(\{.*\}|\[.*\])", re.DOTALL)


def summarize_value(value: Any, limit: int = 180) -> str:
    if isinstance(value, str):
        rendered = value
    else:
        try:
            rendered = json.dumps(value, ensure_ascii=False)
        except (TypeError, ValueError):
            rendered = repr(value)
    rendered = rendered.replace("\n", " ").strip()
    if len(rendered) <= limit:
        return rendered
    return rendered[: limit - 3] + "..."


class OutputNormalizer:
    def __init__(self, registry: SchemaRegistry | None = None) -> None:
        self.registry = registry or SchemaRegistry()

    def normalize_payload(
        self,
        contract: AgentRuntimeInput,
        payload: Any,
    ) -> tuple[dict[str, Any], list[AgentRepairAction]]:
        schema = self.registry.resolve(contract.output_schema_ref, contract.agent_type)
        normalized, actions = self._coerce_to_object(payload)
        actions.extend(self._inject_contract_fields(contract, normalized))
        actions.extend(self._apply_alias_mappings(normalized, schema))
        actions.extend(self._extract_embedded_json(normalized, schema))
        actions.extend(self._fix_array_object_shapes(normalized, schema))
        actions.extend(self._fill_defaults(contract, normalized, schema))
        return normalized, actions

    def _inject_contract_fields(self, contract: AgentRuntimeInput, payload: dict[str, Any]) -> list[AgentRepairAction]:
        field_values = {
            "job_id": contract.job_id,
            "agent_type": contract.agent_type,
            "execution_mode_requested": contract.execution_mode,
            "execution_mode_used": payload.get("execution_mode_used", contract.execution_mode),
            "model_provider": contract.model_provider,
            "model_name": contract.model_name,
            "prompt_version": contract.prompt_version,
            "output_schema_ref": contract.output_schema_ref,
            "workspace_dir": contract.workspace_dir,
        }
        actions: list[AgentRepairAction] = []
        for field_name, field_value in field_values.items():
            if field_name in payload and payload[field_name] not in (None, ""):
                continue
            payload[field_name] = field_value
            actions.append(
                AgentRepairAction(
                    action="fill_contract_field",
                    status="applied",
                    detail=f"Filled '{field_name}' from contract context.",
                    metadata={"field": field_name, "after": summarize_value(field_value)},
                )
            )
        return actions

    def _coerce_to_object(self, payload: Any) -> tuple[dict[str, Any], list[AgentRepairAction]]:
        if isinstance(payload, dict):
            return dict(payload), []

        if isinstance(payload, str):
            extracted = self._extract_json_candidate(payload)
            if isinstance(extracted, dict):
                return extracted, [
                    AgentRepairAction(
                        action="extract_json_root",
                        status="applied",
                        detail="Converted string payload into object payload.",
                        metadata={"before": summarize_value(payload), "after": summarize_value(extracted)},
                    )
                ]
            return {
                "summary": payload.strip(),
                "data": {},
                "items": [],
                "metadata": {},
            }, [
                AgentRepairAction(
                    action="wrap_string_payload",
                    status="applied",
                    detail="Wrapped string payload into normalized object payload.",
                    metadata={"before": summarize_value(payload), "after": summarize_value({"summary": payload.strip()})},
                )
            ]

        if isinstance(payload, list):
            wrapped = {"summary": "", "items": payload, "data": {}, "metadata": {}}
            return wrapped, [
                AgentRepairAction(
                    action="wrap_list_payload",
                    status="applied",
                    detail="Wrapped list payload into normalized object payload.",
                    metadata={"before": summarize_value(payload), "after": summarize_value(wrapped)},
                )
            ]

        wrapped = {"summary": "", "items": [], "data": {"value": payload}, "metadata": {}}
        return wrapped, [
            AgentRepairAction(
                action="wrap_scalar_payload",
                status="applied",
                detail="Wrapped scalar payload into normalized object payload.",
                metadata={"before": summarize_value(payload), "after": summarize_value(wrapped)},
            )
        ]

    def _apply_alias_mappings(self, payload: dict[str, Any], schema: SchemaDefinition) -> list[AgentRepairAction]:
        actions: list[AgentRepairAction] = []
        for field in schema.fields:
            if field.name in payload:
                continue
            for alias in field.aliases:
                if alias not in payload:
                    continue
                payload[field.name] = payload[alias]
                actions.append(
                    AgentRepairAction(
                        action="rename_field",
                        status="applied",
                        detail=f"Mapped field '{alias}' to '{field.name}'.",
                        metadata={"source_field": alias, "target_field": field.name, "before": summarize_value(payload[alias]), "after": summarize_value(payload[field.name])},
                    )
                )
                break
        return actions

    def _extract_embedded_json(self, payload: dict[str, Any], schema: SchemaDefinition) -> list[AgentRepairAction]:
        actions: list[AgentRepairAction] = []
        candidate_fields = ["data", "response_json", "result", "payload", "json", "object"]
        for field_name in candidate_fields:
            value = payload.get(field_name)
            if not isinstance(value, str):
                continue
            extracted = self._extract_json_candidate(value)
            if extracted is None:
                continue
            payload[field_name] = extracted
            actions.append(
                AgentRepairAction(
                    action="extract_markdown_json",
                    status="applied",
                    detail=f"Extracted structured JSON from '{field_name}'.",
                    metadata={"field": field_name, "before": summarize_value(value), "after": summarize_value(extracted)},
                )
            )
        return actions

    def _fix_array_object_shapes(self, payload: dict[str, Any], schema: SchemaDefinition) -> list[AgentRepairAction]:
        actions: list[AgentRepairAction] = []

        if "items" in payload and not isinstance(payload["items"], list):
            before = payload["items"]
            if isinstance(before, dict):
                payload["items"] = [before]
            elif before in (None, ""):
                payload["items"] = []
            else:
                payload["items"] = [before]
            actions.append(
                AgentRepairAction(
                    action="repair_array_shape",
                    status="applied",
                    detail="Coerced 'items' into array shape.",
                    metadata={"field": "items", "before": summarize_value(before), "after": summarize_value(payload["items"])},
                )
            )

        if "data" in payload and not isinstance(payload["data"], dict):
            before = payload["data"]
            if isinstance(before, list):
                payload["data"] = {"items": before}
                if "items" not in payload or not payload["items"]:
                    payload["items"] = before
            elif before in (None, ""):
                payload["data"] = {}
            else:
                payload["data"] = {"value": before}
            actions.append(
                AgentRepairAction(
                    action="repair_object_shape",
                    status="applied",
                    detail="Coerced 'data' into object shape.",
                    metadata={"field": "data", "before": summarize_value(before), "after": summarize_value(payload["data"])},
                )
            )

        if isinstance(payload.get("data"), dict) and (not payload.get("items")):
            data_items = payload["data"].get("items")
            if isinstance(data_items, list):
                payload["items"] = data_items
                actions.append(
                    AgentRepairAction(
                        action="sync_items_from_data",
                        status="applied",
                        detail="Copied array items from data.items into items.",
                        metadata={"after": summarize_value(payload["items"])},
                    )
                )
            elif data_items not in (None, ""):
                payload["items"] = [data_items]
                actions.append(
                    AgentRepairAction(
                        action="sync_items_from_data",
                        status="applied",
                        detail="Wrapped scalar data.items into items array.",
                        metadata={"after": summarize_value(payload["items"])},
                    )
                )

        if "metadata" in payload and not isinstance(payload["metadata"], dict):
            before = payload["metadata"]
            payload["metadata"] = {"value": before} if before not in (None, "") else {}
            actions.append(
                AgentRepairAction(
                    action="repair_metadata_shape",
                    status="applied",
                    detail="Coerced 'metadata' into object shape.",
                    metadata={"field": "metadata", "before": summarize_value(before), "after": summarize_value(payload["metadata"])},
                )
            )

        return actions

    def _fill_defaults(self, contract: AgentRuntimeInput, payload: dict[str, Any], schema: SchemaDefinition) -> list[AgentRepairAction]:
        actions: list[AgentRepairAction] = []
        for field in schema.fields:
            if field.name not in payload or payload[field.name] is None:
                default_value = field.default.copy() if isinstance(field.default, (list, dict)) else field.default
                payload[field.name] = default_value
                actions.append(
                    AgentRepairAction(
                        action="fill_default",
                        status="applied",
                        detail=f"Filled default value for '{field.name}'.",
                        metadata={"field": field.name, "after": summarize_value(default_value)},
                    )
                )

        if not isinstance(payload.get("summary"), str) or not payload.get("summary", "").strip():
            fallback = ""
            if isinstance(payload.get("data"), dict):
                for key in ("summary", "message", "title"):
                    value = payload["data"].get(key)
                    if isinstance(value, str) and value.strip():
                        fallback = value.strip()
                        break
            if not fallback and payload.get("items"):
                fallback = f"Recovered {len(payload['items'])} structured item(s)."
            payload["summary"] = fallback
            actions.append(
                AgentRepairAction(
                    action="repair_summary",
                    status="applied",
                    detail="Rebuilt missing summary field.",
                    metadata={"after": summarize_value(payload["summary"])},
                )
            )
        return actions

    def _extract_json_candidate(self, value: str) -> Any | None:
        text = value.strip()
        for pattern in (CODE_BLOCK_PATTERN, JSON_OBJECT_PATTERN):
            match = pattern.search(text)
            if not match:
                continue
            candidate = match.group(1).strip()
            try:
                return json.loads(candidate)
            except json.JSONDecodeError:
                continue
        try:
            return json.loads(text)
        except json.JSONDecodeError:
            return None
