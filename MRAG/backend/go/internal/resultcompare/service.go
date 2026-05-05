package resultcompare

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
	workspacepkg "mrag-platform/backend/go/internal/workspace"
)

type runStore interface {
	GetByID(context.Context, string) (*model.ExperimentRun, error)
	Update(context.Context, model.ExperimentRun) error
}

type experimentStore interface {
	GetByID(context.Context, string) (*model.Experiment, error)
}

type baselineReader interface {
	GetByID(context.Context, string) (*model.Baseline, error)
}

type archiveReader interface {
	GetByID(context.Context, string) (*model.ResultArchive, error)
	ListByDatasetAssetID(context.Context, string) ([]model.ResultArchive, error)
}

type comparisonStore interface {
	ListByExperimentID(context.Context, string) ([]model.ResultComparison, error)
	Create(context.Context, model.ResultComparison) error
}

type archiveWriter interface {
	Create(context.Context, model.ResultArchiveCreateRequest) (*model.ResultArchiveDetail, error)
}

type Service struct {
	runs          runStore
	experiments   experimentStore
	baselines     baselineReader
	archives      archiveReader
	comparisons   comparisonStore
	archiveWriter archiveWriter
	workspaceRoot string
}

func NewService(
	runs runStore,
	experiments experimentStore,
	baselines baselineReader,
	archives archiveReader,
	comparisons comparisonStore,
	archiveWriter archiveWriter,
	workspaceRoot string,
) *Service {
	if strings.TrimSpace(workspaceRoot) == "" {
		workspaceRoot = "workspace"
	}
	return &Service{
		runs:          runs,
		experiments:   experiments,
		baselines:     baselines,
		archives:      archives,
		comparisons:   comparisons,
		archiveWriter: archiveWriter,
		workspaceRoot: workspaceRoot,
	}
}

func (s *Service) ListByExperimentID(ctx context.Context, experimentID string) ([]model.ResultComparison, error) {
	return s.comparisons.ListByExperimentID(ctx, experimentID)
}

func (s *Service) CompareRun(ctx context.Context, runID string) (*model.RunCompareResult, error) {
	run, err := s.runs.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("run not found")
	}
	if run.RunStatus != "succeeded" {
		return nil, fmt.Errorf("run is not comparable")
	}

	exp, err := s.experiments.GetByID(ctx, run.ExperimentID)
	if err != nil {
		return nil, err
	}
	if exp == nil {
		return nil, fmt.Errorf("experiment not found")
	}

	runMetrics := extractMetricValues(run.ResultJSON)
	if len(runMetrics) == 0 {
		return nil, fmt.Errorf("run metrics not found")
	}

	archiveDetail, err := s.ensureResultArchive(ctx, run, exp, runMetrics)
	if err != nil {
		return nil, err
	}

	items := make([]model.ResultComparison, 0)
	judgments := make([]string, 0)

	if strings.TrimSpace(exp.BaselineID) != "" && s.baselines != nil {
		baseline, err := s.baselines.GetByID(ctx, exp.BaselineID)
		if err != nil {
			return nil, err
		}
		if baseline != nil {
			item, buildErr := s.buildBaselineComparison(ctx, *run, *exp, *baseline, runMetrics)
			if buildErr != nil {
				return nil, buildErr
			}
			items = append(items, item)
			judgments = append(judgments, readComparisonJudgment(item))
		}
	}

	if s.archives != nil {
		archives, err := s.archives.ListByDatasetAssetID(ctx, exp.DatasetAssetID)
		if err != nil {
			return nil, err
		}
		for _, archive := range archives {
			if archiveDetail != nil && archive.ID == archiveDetail.Archive.ID {
				continue
			}
			item, buildErr := s.buildArchiveComparison(ctx, *run, *exp, archive, runMetrics)
			if buildErr != nil {
				return nil, buildErr
			}
			items = append(items, item)
			judgments = append(judgments, readComparisonJudgment(item))
		}
	}

	workspaceDir := workspacepkg.New(s.workspaceRoot).ExperimentComparisonsDir(exp.ID)
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := s.writeComparisonWorkspace(item, workspaceDir); err != nil {
			return nil, err
		}
	}

	overall := summarizeOverall(judgments)
	run.ResultJSON = mergeResult(run.ResultJSON, map[string]interface{}{
		"result_archive_id":           archiveID(archiveDetail),
		"comparison_count":            len(items),
		"comparison_status":           "completed",
		"comparison_overall_judgment": overall,
	})
	run.UpdatedAt = time.Now()
	if err := s.runs.Update(ctx, *run); err != nil {
		return nil, err
	}

	return &model.RunCompareResult{
		Run:             *run,
		Comparisons:     items,
		ResultArchive:   archiveDetail,
		WorkspaceDir:    workspaceDir,
		OverallJudgment: overall,
	}, nil
}

func (s *Service) ensureResultArchive(ctx context.Context, run *model.ExperimentRun, exp *model.Experiment, runMetrics map[string]float64) (*model.ResultArchiveDetail, error) {
	if s.archiveWriter == nil {
		return nil, nil
	}
	if archiveIDValue := strings.TrimSpace(readNestedString(run.ResultJSON, "result_archive_id")); archiveIDValue != "" && s.archives != nil {
		existing, err := s.archives.GetByID(ctx, archiveIDValue)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return &model.ResultArchiveDetail{Archive: *existing}, nil
		}
	}

	summary := s.buildArchiveSummary(run, exp, runMetrics)
	archive, err := s.archiveWriter.Create(ctx, model.ResultArchiveCreateRequest{
		Title:          fmt.Sprintf("%s | %s", firstNonEmpty(strings.TrimSpace(exp.Title), "Experiment Result"), run.ID),
		DatasetAssetID: exp.DatasetAssetID,
		BaselineID:     exp.BaselineID,
		IdeaID:         exp.IdeaID,
		ServerID:       run.AssignedServerID,
		SummaryMD:      summary,
		MetricJSON:     flatMetricJSON(runMetrics),
		Status:         "archived",
		NoteMD:         fmt.Sprintf("Auto archived from experiment `%s` run `%s`.", exp.ID, run.ID),
	})
	if err != nil {
		return nil, err
	}
	return archive, nil
}

func (s *Service) buildBaselineComparison(ctx context.Context, run model.ExperimentRun, exp model.Experiment, baseline model.Baseline, runMetrics map[string]float64) (model.ResultComparison, error) {
	targetMetrics := extractMetricValues(map[string]interface{}{"metrics": baseline.ResultJSON})
	comparisonJSON, summary := buildComparisonPayload("baseline", baseline.ID, baseline.Name, runMetrics, targetMetrics)
	item := model.ResultComparison{
		ID:             httpx.NewID("cmp"),
		ExperimentID:   exp.ID,
		RunID:          run.ID,
		BaselineID:     baseline.ID,
		ComparisonJSON: comparisonJSON,
		SummaryMD:      summary,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.comparisons.Create(ctx, item); err != nil {
		return item, err
	}
	return item, nil
}

func (s *Service) buildArchiveComparison(ctx context.Context, run model.ExperimentRun, exp model.Experiment, archive model.ResultArchive, runMetrics map[string]float64) (model.ResultComparison, error) {
	comparisonJSON, summary := buildComparisonPayload("result_archive", archive.ID, archive.Title, runMetrics, extractMetricValues(map[string]interface{}{"metrics": archive.MetricJSON}))
	item := model.ResultComparison{
		ID:                    httpx.NewID("cmp"),
		ExperimentID:          exp.ID,
		RunID:                 run.ID,
		TargetResultArchiveID: archive.ID,
		ComparisonJSON:        comparisonJSON,
		SummaryMD:             summary,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	if err := s.comparisons.Create(ctx, item); err != nil {
		return item, err
	}
	return item, nil
}

func (s *Service) buildArchiveSummary(run *model.ExperimentRun, exp *model.Experiment, runMetrics map[string]float64) string {
	builder := strings.Builder{}
	builder.WriteString("# Auto Result Archive\n\n")
	builder.WriteString(fmt.Sprintf("- Experiment: `%s`\n", exp.ID))
	builder.WriteString(fmt.Sprintf("- Run: `%s`\n", run.ID))
	if strings.TrimSpace(run.AssignedServerID) != "" {
		builder.WriteString(fmt.Sprintf("- Server: `%s`\n", run.AssignedServerID))
	}
	builder.WriteString("- Metrics:\n")
	keys := sortedMetricKeys(runMetrics)
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("  - `%s`: %.4f\n", key, runMetrics[key]))
	}
	return builder.String()
}

func (s *Service) writeComparisonWorkspace(item model.ResultComparison, dir string) error {
	raw, err := json.MarshalIndent(item.ComparisonJSON, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, item.ID+".json"), raw, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, item.ID+".md"), []byte(ensureTrailingLine(item.SummaryMD)), 0o644)
}

func buildComparisonPayload(targetType string, targetID string, targetName string, runMetrics map[string]float64, targetMetrics map[string]float64) (map[string]interface{}, string) {
	diffItems := make([]map[string]interface{}, 0)
	betterCount := 0
	worseCount := 0
	equalCount := 0
	keys := unionMetricKeys(runMetrics, targetMetrics)
	for _, key := range keys {
		candidate, candidateOK := runMetrics[key]
		target, targetOK := targetMetrics[key]
		if !candidateOK || !targetOK {
			diffItems = append(diffItems, map[string]interface{}{"metric": key, "status": "missing_target_metric"})
			continue
		}
		higherBetter := metricHigherIsBetter(key)
		diff := candidate - target
		judgment := compareMetric(candidate, target, higherBetter)
		switch judgment {
		case "较优":
			betterCount++
		case "较差":
			worseCount++
		default:
			equalCount++
		}
		diffItems = append(diffItems, map[string]interface{}{
			"metric":           key,
			"candidate_value":  round4(candidate),
			"target_value":     round4(target),
			"diff":             round4(diff),
			"higher_is_better": higherBetter,
			"judgment":         judgment,
		})
	}

	overall := "持平"
	switch {
	case betterCount > 0 && worseCount == 0:
		overall = "较优"
	case worseCount > 0 && betterCount == 0:
		overall = "较差"
	case betterCount > worseCount:
		overall = "较优"
	case worseCount > betterCount:
		overall = "较差"
	}

	payload := map[string]interface{}{
		"target_type":       targetType,
		"target_id":         targetID,
		"target_name":       targetName,
		"metric_diffs":      diffItems,
		"judgment":          overall,
		"better_count":      betterCount,
		"equal_count":       equalCount,
		"worse_count":       worseCount,
		"candidate_metrics": flatMetricJSON(runMetrics),
		"target_metrics":    flatMetricJSON(targetMetrics),
	}
	summary := buildComparisonSummaryMarkdown(targetType, targetName, overall, diffItems)
	return payload, summary
}

func buildComparisonSummaryMarkdown(targetType string, targetName string, overall string, diffs []map[string]interface{}) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("# Comparison vs %s\n\n", firstNonEmpty(strings.TrimSpace(targetName), targetType)))
	builder.WriteString(fmt.Sprintf("- Target type: `%s`\n", targetType))
	builder.WriteString(fmt.Sprintf("- Overall judgment: `%s`\n", overall))
	builder.WriteString("- Metric details:\n")
	for _, item := range diffs {
		metric, _ := item["metric"].(string)
		judgment, _ := item["judgment"].(string)
		status, _ := item["status"].(string)
		if status != "" {
			builder.WriteString(fmt.Sprintf("  - `%s`: %s\n", metric, status))
			continue
		}
		builder.WriteString(fmt.Sprintf("  - `%s`: %.4f vs %.4f, %s\n", metric, readFloat(item["candidate_value"]), readFloat(item["target_value"]), judgment))
	}
	return builder.String()
}

func summarizeOverall(judgments []string) string {
	if len(judgments) == 0 {
		return "无可用对比对象"
	}
	allBetter := true
	hasWorse := false
	hasBetter := false
	for _, item := range judgments {
		switch item {
		case "较优":
			hasBetter = true
		case "较差":
			allBetter = false
			hasWorse = true
		default:
			allBetter = false
		}
	}
	switch {
	case allBetter && hasBetter:
		return "最优"
	case hasWorse && !hasBetter:
		return "较差"
	case hasBetter:
		return "较优"
	default:
		return "持平"
	}
}

func readComparisonJudgment(item model.ResultComparison) string {
	value, _ := item.ComparisonJSON["judgment"].(string)
	return value
}

func archiveID(detail *model.ResultArchiveDetail) string {
	if detail == nil {
		return ""
	}
	return detail.Archive.ID
}

func extractMetricValues(root map[string]interface{}) map[string]float64 {
	values := map[string]float64{}
	if root == nil {
		return values
	}
	if metricsRaw, ok := root["metrics"].(map[string]interface{}); ok {
		if nested, ok := metricsRaw["values"].(map[string]interface{}); ok {
			for key, value := range nested {
				if floatValue, ok := toFloat(value); ok {
					values[key] = floatValue
				}
			}
			if len(values) > 0 {
				return values
			}
		}
		for key, value := range metricsRaw {
			if key == "primary_metric" || key == "values" {
				continue
			}
			if floatValue, ok := toFloat(value); ok {
				values[key] = floatValue
			}
		}
		if len(values) > 0 {
			return values
		}
	}
	if nested, ok := root["values"].(map[string]interface{}); ok {
		for key, value := range nested {
			if floatValue, ok := toFloat(value); ok {
				values[key] = floatValue
			}
		}
	}
	for key, value := range root {
		if floatValue, ok := toFloat(value); ok {
			values[key] = floatValue
		}
	}
	return values
}

func unionMetricKeys(left map[string]float64, right map[string]float64) []string {
	set := make(map[string]struct{}, len(left)+len(right))
	for key := range left {
		set[key] = struct{}{}
	}
	for key := range right {
		set[key] = struct{}{}
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedMetricKeys(values map[string]float64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func metricHigherIsBetter(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.Contains(lower, "loss"), strings.Contains(lower, "latency"), strings.Contains(lower, "time"), strings.Contains(lower, "error"):
		return false
	default:
		return true
	}
}

func compareMetric(candidate float64, target float64, higherBetter bool) string {
	const epsilon = 1e-9
	if math.Abs(candidate-target) <= epsilon {
		return "持平"
	}
	if higherBetter {
		if candidate > target {
			return "较优"
		}
		return "较差"
	}
	if candidate < target {
		return "较优"
	}
	return "较差"
}

func flatMetricJSON(values map[string]float64) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = round4(value)
	}
	return out
}

func toFloat(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func readFloat(value interface{}) float64 {
	floatValue, _ := toFloat(value)
	return floatValue
}

func readNestedString(root map[string]interface{}, key string) string {
	if root == nil {
		return ""
	}
	raw, _ := root[key].(string)
	return raw
}

func ensureTrailingLine(value string) string {
	if strings.HasSuffix(value, "\n") {
		return value
	}
	return value + "\n"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mergeResult(existing map[string]interface{}, updates map[string]interface{}) map[string]interface{} {
	if existing == nil {
		existing = map[string]interface{}{}
	}
	for key, value := range updates {
		existing[key] = value
	}
	return existing
}
