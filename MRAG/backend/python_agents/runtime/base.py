from __future__ import annotations

from typing import Any

try:
    from .contract import AgentRepairAction, AgentRuntimeInput, AgentRuntimeOutput
    from .normalizer import OutputNormalizer
    from .schema_registry import SchemaRegistry
except ImportError:  # pragma: no cover - supports direct script execution
    from contract import AgentRepairAction, AgentRuntimeInput, AgentRuntimeOutput
    from normalizer import OutputNormalizer
    from schema_registry import SchemaRegistry


class BaseExecutor:
    name = "base"

    def prepare_request(self, contract: AgentRuntimeInput) -> dict[str, Any]:
        raise NotImplementedError

    def execute(self, prepared_request: dict[str, Any], contract: AgentRuntimeInput) -> dict[str, Any]:
        raise NotImplementedError

    def collect_response(
        self,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        contract: AgentRuntimeInput,
    ) -> dict[str, Any]:
        return execution_result

    def normalize_output(
        self,
        contract: AgentRuntimeInput,
        prepared_request: dict[str, Any],
        execution_result: dict[str, Any],
        collected_response: dict[str, Any],
    ) -> AgentRuntimeOutput:
        raise NotImplementedError

    def run(self, contract: AgentRuntimeInput) -> AgentRuntimeOutput:
        try:
            prepared_request = self.prepare_request(contract)
            execution_result = self.execute(prepared_request, contract)
            collected_response = self.collect_response(prepared_request, execution_result, contract)
            return self.normalize_output(contract, prepared_request, execution_result, collected_response)
        except Exception as exc:  # pragma: no cover - defensive guard
            return AgentRuntimeOutput(
                status="failed",
                normalized_payload=self.base_payload(contract, contract.execution_mode),
                artifact_manifest=[],
                repair_actions=[],
                tool_usages=[],
                warnings=[],
                error_message=f"{self.__class__.__name__} failed: {exc}",
            )

    def base_payload(self, contract: AgentRuntimeInput, used_mode: str) -> dict[str, Any]:
        return {
            "job_id": contract.job_id,
            "agent_type": contract.agent_type,
            "execution_mode_requested": contract.execution_mode,
            "execution_mode_used": used_mode,
            "model_provider": contract.model_provider,
            "model_name": contract.model_name,
            "prompt_version": contract.prompt_version,
            "output_schema_ref": contract.output_schema_ref,
            "workspace_dir": contract.workspace_dir,
            "executor": self.__class__.__name__,
        }


class BaseValidator:
    VALID_MODES = {"api", "codex_cli", "mock"}
    VALID_STATUSES = {
        "registered",
        "idle",
        "waiting_input",
        "ready",
        "running",
        "validating",
        "repairing",
        "succeeded",
        "failed",
        "paused",
    }

    def validate_input(self, contract: AgentRuntimeInput) -> list[str]:
        errors: list[str] = []
        if not contract.job_id:
            errors.append("job_id is required")
        if not contract.agent_type:
            errors.append("agent_type is required")
        if contract.execution_mode not in self.VALID_MODES:
            errors.append("execution_mode must be one of api, codex_cli, mock")
        if not contract.output_schema_ref:
            errors.append("output_schema_ref is required")
        return errors

    def __init__(self, registry: SchemaRegistry | None = None) -> None:
        self.registry = registry or SchemaRegistry()

    def validate_output(self, contract: AgentRuntimeInput, output: AgentRuntimeOutput) -> list[str]:
        errors: list[str] = []
        if output.status not in self.VALID_STATUSES:
            errors.append("status must be a valid lifecycle state")
        if output.normalized_payload is None:
            errors.append("normalized_payload is required")
        if output.artifact_manifest is None:
            errors.append("artifact_manifest is required")
        if output.repair_actions is None:
            errors.append("repair_actions is required")
        if output.tool_usages is None:
            errors.append("tool_usages is required")
        if output.warnings is None:
            errors.append("warnings is required")
        if output.validation_status not in {"pending", "validating", "succeeded", "failed"}:
            errors.append("validation_status must be one of pending, validating, succeeded, failed")
        if output.repair_status not in {"pending", "not_needed", "repairing", "succeeded", "failed"}:
            errors.append("repair_status must be one of pending, not_needed, repairing, succeeded, failed")
        if output.validation_errors is None:
            errors.append("validation_errors is required")
        errors.extend(self.validate_payload(contract, output.normalized_payload))
        return errors

    def validate_payload(self, contract: AgentRuntimeInput, payload: dict[str, Any] | None) -> list[str]:
        if payload is None:
            return ["normalized_payload is required"]

        schema = self.registry.resolve(contract.output_schema_ref, contract.agent_type)
        errors: list[str] = []
        for field in schema.fields:
            value = payload.get(field.name)
            if field.required and field.name not in payload:
                errors.append(f"normalized_payload.{field.name} is required")
                continue
            if field.required and value is None:
                errors.append(f"normalized_payload.{field.name} cannot be null")
                continue
            if value is None:
                continue
            if field.field_type == "string" and not isinstance(value, str):
                errors.append(f"normalized_payload.{field.name} must be a string")
            if field.field_type == "string" and isinstance(value, str) and field.required and value.strip() == "":
                errors.append(f"normalized_payload.{field.name} cannot be empty")
            if field.field_type == "array" and not isinstance(value, list):
                errors.append(f"normalized_payload.{field.name} must be an array")
            if field.field_type == "array" and isinstance(value, list) and not self.is_json_compatible(value):
                errors.append(f"normalized_payload.{field.name} must contain JSON-compatible values")
            if field.field_type == "object" and not isinstance(value, dict):
                errors.append(f"normalized_payload.{field.name} must be an object")
            if field.field_type == "object" and isinstance(value, dict) and not self.is_json_compatible(value):
                errors.append(f"normalized_payload.{field.name} must contain JSON-compatible values")
        return errors

    def is_json_compatible(self, value: Any) -> bool:
        if value is None or isinstance(value, (str, int, float, bool)):
            return True
        if isinstance(value, list):
            return all(self.is_json_compatible(item) for item in value)
        if isinstance(value, dict):
            return all(isinstance(key, str) and self.is_json_compatible(item) for key, item in value.items())
        return False


class BaseRepairer:
    def __init__(self, normalizer: OutputNormalizer | None = None) -> None:
        self.normalizer = normalizer or OutputNormalizer()

    def repair(
        self,
        contract: AgentRuntimeInput,
        output: AgentRuntimeOutput,
        errors: list[str],
    ) -> tuple[AgentRuntimeOutput, list[AgentRepairAction]]:
        actions: list[AgentRepairAction] = []
        repaired = output

        if repaired.normalized_payload is None:
            repaired.normalized_payload = {}
            actions.append(
                AgentRepairAction(
                    action="fill_normalized_payload",
                    status="applied",
                    detail="normalized_payload was missing and has been initialized",
                )
            )
        if repaired.artifact_manifest is None:
            repaired.artifact_manifest = []
            actions.append(
                AgentRepairAction(
                    action="fill_artifact_manifest",
                    status="applied",
                    detail="artifact_manifest was missing and has been initialized",
                )
            )
        if repaired.repair_actions is None:
            repaired.repair_actions = []
        if repaired.tool_usages is None:
            repaired.tool_usages = []
            actions.append(
                AgentRepairAction(
                    action="fill_tool_usages",
                    status="applied",
                    detail="tool_usages was missing and has been initialized",
                )
            )
        if repaired.warnings is None:
            repaired.warnings = []
            actions.append(
                AgentRepairAction(
                    action="fill_warnings",
                    status="applied",
                    detail="warnings was missing and has been initialized",
                )
            )
        if not repaired.status:
            repaired.status = "failed"
            actions.append(
                AgentRepairAction(
                    action="default_status",
                    status="applied",
                    detail="status was missing and has been set to failed",
                    metadata={"errors": errors},
                )
            )
        normalized_payload, payload_actions = self.normalizer.normalize_payload(contract, repaired.normalized_payload)
        repaired.normalized_payload = normalized_payload
        actions.extend(payload_actions)
        return repaired, actions


class BaseAgent:
    def __init__(self, executor: BaseExecutor, validator: BaseValidator, repairer: BaseRepairer) -> None:
        self.executor = executor
        self.validator = validator
        self.repairer = repairer

    def run(self, contract: AgentRuntimeInput) -> AgentRuntimeOutput:
        input_errors = self.validator.validate_input(contract)
        if input_errors:
            return AgentRuntimeOutput(
                status="failed",
                normalized_payload={},
                artifact_manifest=[],
                repair_actions=[],
                tool_usages=[],
                warnings=[],
                validation_status="failed",
                repair_status="failed",
                validation_errors=input_errors,
                error_message="; ".join(input_errors),
            )

        output = self.executor.run(contract)
        output.validation_status = "validating"
        output.repair_status = "pending"
        output_errors = self.validator.validate_output(contract, output)
        if not output_errors:
            output.validation_status = "succeeded"
            output.repair_status = "not_needed"
            output.validation_errors = []
            return output

        output.validation_status = "failed"
        output.validation_errors = output_errors
        output.repair_status = "repairing"
        repaired, actions = self.repairer.repair(contract, output, output_errors)
        repaired.repair_actions.extend(actions)
        final_errors = self.validator.validate_output(contract, repaired)
        if final_errors:
            repaired.validation_status = "failed"
            repaired.repair_status = "failed"
            repaired.validation_errors = final_errors
            repaired.status = "failed"
            repaired.error_message = "; ".join(final_errors)
            return repaired
        repaired.validation_status = "succeeded"
        repaired.repair_status = "succeeded"
        repaired.validation_errors = []
        return repaired
