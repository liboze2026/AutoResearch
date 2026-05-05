package traintemplate

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPrepareRunDirAndRunnerExecution(t *testing.T) {
	workspaceRoot := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	templateRoot := filepath.Clean(filepath.Join(wd, "..", "..", "..", "python_templates"))
	svc := NewService(templateRoot, workspaceRoot)

	spec := map[string]interface{}{
		"dataset_ref": map[string]interface{}{
			"dataset_asset_id": "dasset_demo_001",
			"name":             "Demo Dataset",
		},
		"dataset_loader_ref": map[string]interface{}{
			"loader_id": "mrag.dataset_asset_loader.v1",
		},
		"train_template_type": "mock_train_template",
		"model_name":          "mock/llama3.1-8b-instruct",
		"hyperparams": map[string]interface{}{
			"epochs":     3,
			"batch_size": 8,
		},
		"output_dir": "/tmp/ignored-by-go-service",
		"expected_metrics": map[string]interface{}{
			"primary": "accuracy",
		},
		"comparison_targets": []map[string]interface{}{
			{"type": "baseline", "id": "baseline_demo_001"},
		},
	}

	prepared, err := svc.PrepareRunDir(context.Background(), "exp_demo_001", 1, spec)
	if err != nil {
		t.Fatalf("PrepareRunDir returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(prepared.RunDir, "train.py")); err != nil {
		t.Fatalf("expected train.py in run dir: %v", err)
	}
	raw, err := os.ReadFile(prepared.SpecPath)
	if err != nil {
		t.Fatalf("ReadFile spec.json returned error: %v", err)
	}
	var writtenSpec map[string]interface{}
	if err := json.Unmarshal(raw, &writtenSpec); err != nil {
		t.Fatalf("unmarshal spec.json: %v", err)
	}
	if writtenSpec["model_name"] != "mock/llama3.1-8b-instruct" {
		t.Fatalf("unexpected model_name: %v", writtenSpec["model_name"])
	}

	pythonExe := "python"
	cmd := exec.Command(pythonExe, prepared.RunnerPath, "--spec", prepared.SpecPath, "--output-dir", prepared.OutputDir)
	cmd.Dir = prepared.RunDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("runner execution failed: %v\n%s", err, string(output))
	}

	for _, name := range []string{"metrics.json", "result.md", "stdout.log", "stderr.log"} {
		if _, err := os.Stat(filepath.Join(prepared.OutputDir, name)); err != nil {
			t.Fatalf("expected artifact %s: %v", name, err)
		}
	}

	metricsRaw, err := os.ReadFile(filepath.Join(prepared.OutputDir, "metrics.json"))
	if err != nil {
		t.Fatalf("ReadFile metrics.json returned error: %v", err)
	}
	var metrics map[string]interface{}
	if err := json.Unmarshal(metricsRaw, &metrics); err != nil {
		t.Fatalf("unmarshal metrics.json: %v", err)
	}
	if metrics["primary_metric"] != "accuracy" {
		t.Fatalf("expected accuracy metric, got %v", metrics["primary_metric"])
	}
}
