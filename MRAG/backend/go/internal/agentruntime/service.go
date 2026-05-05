package agentruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mrag-platform/backend/go/internal/model"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type Service struct {
	pythonExec      string
	pythonAgentsDir string
	workspaceRoot   string
}

func NewService(pythonExec string, pythonAgentsDir string, workspaceRoot string) *Service {
	if strings.TrimSpace(pythonExec) == "" {
		pythonExec = "python"
	}
	if strings.TrimSpace(pythonAgentsDir) == "" {
		pythonAgentsDir = filepath.Join("..", "python_agents")
	}
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{pythonExec: pythonExec, pythonAgentsDir: pythonAgentsDir, workspaceRoot: workspaceRoot}
}

func (s *Service) Execute(ctx context.Context, input model.AgentRuntimeInput) (*model.AgentRuntimeOutput, error) {
	if err := validateInput(input); err != nil {
		return nil, err
	}
	workspaceDir := strings.TrimSpace(input.WorkspaceDir)
	if workspaceDir == "" {
		workspaceDir = workspacepkg.New(s.workspaceRoot).AgentJobDir(input.JobID)
		input.WorkspaceDir = workspaceDir
	}
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return nil, err
	}

	inputPath := filepath.Join(workspaceDir, "input.json")
	outputPath := filepath.Join(workspaceDir, "output.json")
	raw, err := json.MarshalIndent(input, "", "  ")
	if err != nil {
		return nil, err
	}
	if err = os.WriteFile(inputPath, raw, 0o644); err != nil {
		return nil, err
	}

	runnerPath, err := filepath.Abs(filepath.Join(s.pythonAgentsDir, "runtime", "runner.py"))
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, s.pythonExec, runnerPath, "--input", inputPath, "--output", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("agent runtime execution failed: %s", strings.TrimSpace(string(output)))
	}

	outputRaw, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, err
	}
	var result model.AgentRuntimeOutput
	if err = json.Unmarshal(outputRaw, &result); err != nil {
		return nil, err
	}
	if err = validateOutput(result); err != nil {
		return nil, err
	}
	return &result, nil
}

func validateInput(input model.AgentRuntimeInput) error {
	if strings.TrimSpace(input.JobID) == "" {
		return fmt.Errorf("job_id is required")
	}
	if strings.TrimSpace(input.AgentType) == "" {
		return fmt.Errorf("agent_type is required")
	}
	switch strings.TrimSpace(input.ExecutionMode) {
	case "api", "codex_cli", "mock":
	default:
		return fmt.Errorf("execution_mode must be one of api, codex_cli, mock")
	}
	if strings.TrimSpace(input.OutputSchemaRef) == "" {
		return fmt.Errorf("output_schema_ref is required")
	}
	return nil
}

func validateOutput(output model.AgentRuntimeOutput) error {
	if strings.TrimSpace(output.Status) == "" {
		return fmt.Errorf("runtime output status is required")
	}
	if output.NormalizedPayload == nil {
		return fmt.Errorf("runtime output normalized_payload is required")
	}
	if output.ArtifactManifest == nil {
		return fmt.Errorf("runtime output artifact_manifest is required")
	}
	if output.RepairActions == nil {
		return fmt.Errorf("runtime output repair_actions is required")
	}
	if output.ToolUsages == nil {
		return fmt.Errorf("runtime output tool_usages is required")
	}
	if output.Warnings == nil {
		return fmt.Errorf("runtime output warnings is required")
	}
	if strings.TrimSpace(output.ValidationStatus) == "" {
		return fmt.Errorf("runtime output validation_status is required")
	}
	if strings.TrimSpace(output.RepairStatus) == "" {
		return fmt.Errorf("runtime output repair_status is required")
	}
	if output.ValidationErrors == nil {
		return fmt.Errorf("runtime output validation_errors is required")
	}
	return nil
}
