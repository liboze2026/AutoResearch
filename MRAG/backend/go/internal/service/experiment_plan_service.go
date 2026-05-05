package service

import (
	"encoding/json"
	"os"
	"path/filepath"

	"mrag-platform/backend/go/internal/model"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

func (s *ExperimentService) loadPlannerPlan(experimentID string) (*model.ExperimentPlanDocument, error) {
	paths := workspacepkg.New(s.workspaceRoot)
	planPath := filepath.Join(paths.ExperimentDir(experimentID), "plan.json")
	data, err := os.ReadFile(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var plan model.ExperimentPlanDocument
	if err = json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func applyPlannerPlanToSpec(spec map[string]interface{}, plan *model.ExperimentPlanDocument) map[string]interface{} {
	if plan == nil {
		return spec
	}
	out := cloneInterfaceMap(spec)
	if out == nil {
		out = map[string]interface{}{}
	}
	if plan.TrainTemplateType != "" {
		out["train_template_type"] = plan.TrainTemplateType
	}
	plannerAgentSection := map[string]interface{}{
		"resource_estimate": cloneInterfaceMap(plan.ResourceEstimate),
		"run_sequence":      cloneStringSlice(plan.RunSequence),
		"success_criteria":  cloneInterfaceMap(plan.SuccessCriteria),
		"fallback_plan":     cloneInterfaceMap(plan.FallbackPlan),
		"plan_path":         plan.PlanPath,
	}
	plannerExtensions := map[string]interface{}{
		"planner_agent": plannerAgentSection,
		"coding_agent":  "reserved",
	}
	if existing, ok := out["planner_extensions"].(map[string]interface{}); ok {
		for key, value := range existing {
			plannerExtensions[key] = value
		}
		plannerExtensions["planner_agent"] = plannerAgentSection
	}
	out["planner_extensions"] = plannerExtensions
	if plan.ExperimentPlanJSON != nil {
		out["planner_plan"] = cloneInterfaceMap(plan.ExperimentPlanJSON)
		if metrics, ok := plan.SuccessCriteria["required_metrics"].([]any); ok && len(metrics) > 0 {
			primary, _ := metrics[0].(string)
			secondary := make([]string, 0, len(metrics)-1)
			for _, item := range metrics[1:] {
				if text, ok := item.(string); ok && text != "" {
					secondary = append(secondary, text)
				}
			}
			out["expected_metrics"] = map[string]interface{}{
				"primary":   primary,
				"secondary": secondary,
			}
		}
	}
	return out
}

func cloneInterfaceMap(input map[string]any) map[string]interface{} {
	if input == nil {
		return nil
	}
	out := make(map[string]interface{}, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneStringSlice(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	out := make([]string, len(input))
	copy(out, input)
	return out
}
