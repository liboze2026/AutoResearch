package traintemplate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type PreparedRun struct {
	TemplateName string `json:"templateName"`
	RunDir       string `json:"runDir"`
	SpecPath     string `json:"specPath"`
	RunnerPath   string `json:"runnerPath"`
	OutputDir    string `json:"outputDir"`
}

type Service struct {
	templateRoot  string
	workspaceRoot string
}

func NewService(templateRoot string, workspaceRoot string) *Service {
	if strings.TrimSpace(templateRoot) == "" {
		templateRoot = filepath.Join("..", "python_templates")
	}
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{templateRoot: templateRoot, workspaceRoot: workspaceRoot}
}

func (s *Service) PrepareRunDir(_ context.Context, experimentID string, runNumber int, spec map[string]interface{}) (*PreparedRun, error) {
	if strings.TrimSpace(experimentID) == "" {
		return nil, fmt.Errorf("experimentID is required")
	}
	if runNumber <= 0 {
		return nil, fmt.Errorf("runNumber must be positive")
	}
	templateName, err := s.SelectTemplate(spec)
	if err != nil {
		return nil, err
	}
	if err := s.ValidateSpec(templateName, spec); err != nil {
		return nil, err
	}

	paths := workspacepkg.New(s.workspaceRoot)
	runDir := filepath.Join(paths.ExperimentDir(experimentID), fmt.Sprintf("run_%d", runNumber))
	outputDir := filepath.Join(runDir, "outputs")
	templateDir := filepath.Join(s.templateRoot, templateName)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	if err := copyDir(templateDir, runDir); err != nil {
		return nil, err
	}

	specCopy := cloneSpec(spec)
	specCopy["output_dir"] = outputDir
	specPath := filepath.Join(runDir, "spec.json")
	raw, err := json.MarshalIndent(specCopy, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(specPath, raw, 0o644); err != nil {
		return nil, err
	}

	return &PreparedRun{
		TemplateName: templateName,
		RunDir:       runDir,
		SpecPath:     specPath,
		RunnerPath:   filepath.Join(runDir, "runner.py"),
		OutputDir:    outputDir,
	}, nil
}

func (s *Service) SelectTemplate(spec map[string]interface{}) (string, error) {
	templateType := strings.TrimSpace(asString(spec["train_template_type"]))
	switch templateType {
	case "", "mock_train_template", "generic_train_v1", "text_finetune_v1", "lora_sft_v1":
		return "mock_train_template", nil
	default:
		return "", fmt.Errorf("unsupported train template type: %s", templateType)
	}
}

func (s *Service) ValidateSpec(templateName string, spec map[string]interface{}) error {
	schemaPath := filepath.Join(s.templateRoot, templateName, "config_schema.json")
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}
	var schema struct {
		RequiredFields []string `json:"required_fields"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return err
	}
	for _, field := range schema.RequiredFields {
		if _, ok := spec[field]; !ok {
			return fmt.Errorf("spec missing required field: %s", field)
		}
	}
	return nil
}

func cloneSpec(spec map[string]interface{}) map[string]interface{} {
	copyValue := make(map[string]interface{}, len(spec))
	for key, value := range spec {
		copyValue[key] = value
	}
	return copyValue
}

func copyDir(src string, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func asString(value interface{}) string {
	if raw, ok := value.(string); ok {
		return raw
	}
	return ""
}
