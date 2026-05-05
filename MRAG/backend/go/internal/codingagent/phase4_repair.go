package codingagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
)

type phase4RunAttemptError struct {
	Attempt int
	Stage   string
	Err     error
}

func (e *phase4RunAttemptError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return fmt.Sprintf("attempt %d failed at %s: %v", e.Attempt, e.Stage, e.Err)
}

func (e *phase4RunAttemptError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (s *Phase4Service) executePhase4ManagedRun(ctx context.Context, job *model.AgentJob, runManifest *model.Phase4RunManifest, datasetProfile *model.Phase4DatasetProfile, idea *model.Phase4Idea, readerContext *model.Phase4ReaderContext, artifactPaths map[string]any, snapshotID string, payload model.Phase4CodingRuntimePayload) error {
	startedAt := time.Now()
	if _, err := s.phase4.UpdateRunManifestStatus(ctx, runManifest.ID, model.Phase4RunManifestStatusUpdateRequest{
		Status:    model.Phase4RunStatusRunning,
		StartedAt: &startedAt,
	}); err != nil {
		return err
	}

	maxRetries := runManifest.MaxRetryCount
	if maxRetries < 0 {
		maxRetries = 0
	}
	attemptLimit := maxRetries + 1
	failures := make([]map[string]any, 0, attemptLimit)

	for attempt := 1; attempt <= attemptLimit; attempt++ {
		if attempt > 1 {
			if _, err := s.phase4.UpdateRunManifestStatus(ctx, runManifest.ID, model.Phase4RunManifestStatusUpdateRequest{
				Status:     model.Phase4RunStatusRunning,
				RetryCount: intPtr(attempt - 1),
			}); err != nil {
				return err
			}
			repairEntry, err := s.applyPhase4Repair(artifactPaths, payload, idea, failures[len(failures)-1], attempt-1)
			if err != nil {
				failures = append(failures, s.buildPhase4FailureFeedback(runManifest, artifactPaths, attempt, maxRetries, "repair", err, failures))
				break
			}
			appendPhase4RepairLog(stringValue(artifactPaths["repair_log_path"]), repairEntry)
		}

		var (
			metricsRaw     map[string]any
			metricsSummary map[string]any
			err            error
		)
		if isPhase4ShenzhenlabMode(runManifest.RunnerMode) {
			metricsRaw, metricsSummary, err = s.runPhase4RemoteAttempt(ctx, runManifest, datasetProfile, artifactPaths, attempt)
		} else {
			metricsRaw, metricsSummary, err = s.runPhase4LocalAttempt(ctx, runManifest, artifactPaths, attempt)
		}
		if err == nil {
			return s.finalizePhase4Success(ctx, job, runManifest, datasetProfile, idea, readerContext, artifactPaths, snapshotID, metricsRaw, metricsSummary, attempt-1, failures)
		}
		failures = append(failures, s.buildPhase4FailureFeedback(runManifest, artifactPaths, attempt, maxRetries, phase4AttemptStage(err), err, failures))
		appendPhase4RepairLog(stringValue(artifactPaths["repair_log_path"]), map[string]any{
			"attempt":      attempt,
			"status":       "failed",
			"stage":        phase4AttemptStage(err),
			"error":        err.Error(),
			"retry_count":  max(0, attempt-1),
			"recorded_at":  time.Now().Format(time.RFC3339Nano),
			"driver_log":   stringValue(artifactPaths["driver_log_path"]),
			"artifact_dir": stringValue(artifactPaths["artifact_dir"]),
		})
		if _, updateErr := s.phase4.UpdateRunManifest(ctx, runManifest.ID, model.Phase4RunManifestUpdateRequest{
			ArtifactPaths:   &artifactPaths,
			LogsPath:        stringPtr(stringValue(artifactPaths["logs_dir"])),
			MetricsPath:     stringPtr(stringValue(artifactPaths["metrics_path"])),
			FailureFeedback: &failures[len(failures)-1],
			RetryCount:      intPtr(max(0, attempt-1)),
		}); updateErr != nil {
			return updateErr
		}
	}
	return s.finalizePhase4TestFailure(ctx, job, runManifest, idea, artifactPaths, snapshotID, failures)
}

func (s *Phase4Service) runPhase4LocalAttempt(ctx context.Context, runManifest *model.Phase4RunManifest, artifactPaths map[string]any, attempt int) (map[string]any, map[string]any, error) {
	if err := clearPhase4AttemptOutputs(artifactPaths); err != nil {
		return nil, nil, &phase4RunAttemptError{Attempt: attempt, Stage: "clear_outputs", Err: err}
	}
	driverLogPath := stringValue(artifactPaths["driver_log_path"])
	manifestPath := stringValue(artifactPaths["manifest_path"])
	snapshotDir := stringValue(artifactPaths["snapshot_dir"])
	runOutput, runErr := s.executeEntrypoint(ctx, snapshotDir, "run_entrypoint.py", manifestPath)
	appendPhase4DriverLog(driverLogPath, fmt.Sprintf("run_entrypoint_attempt_%d", attempt), runOutput, runErr)
	if runErr != nil {
		return nil, nil, &phase4RunAttemptError{Attempt: attempt, Stage: "run_entrypoint", Err: runErr}
	}
	evalOutput, evalErr := s.executeEntrypoint(ctx, snapshotDir, "eval_entrypoint.py", manifestPath)
	appendPhase4DriverLog(driverLogPath, fmt.Sprintf("eval_entrypoint_attempt_%d", attempt), evalOutput, evalErr)
	if evalErr != nil {
		return nil, nil, &phase4RunAttemptError{Attempt: attempt, Stage: "eval_entrypoint", Err: evalErr}
	}
	metricsRaw := readJSON(stringValue(artifactPaths["metrics_path"]))
	return metricsRaw, summarizePhase4Metrics(metricsRaw), nil
}

func (s *Phase4Service) runPhase4RemoteAttempt(ctx context.Context, runManifest *model.Phase4RunManifest, datasetProfile *model.Phase4DatasetProfile, artifactPaths map[string]any, attempt int) (map[string]any, map[string]any, error) {
	if err := clearPhase4AttemptOutputs(artifactPaths); err != nil {
		return nil, nil, &phase4RunAttemptError{Attempt: attempt, Stage: "clear_outputs", Err: err}
	}
	if s.servers == nil || s.gpuSelector == nil || s.remoteExecutor == nil {
		return nil, nil, &phase4RunAttemptError{Attempt: attempt, Stage: "remote_configure", Err: fmt.Errorf("phase4 shenzhenvlab execution is not configured")}
	}
	serverNode, err := s.resolvePhase4ShenzhenlabServer(ctx, strings.TrimSpace(runManifest.ServerID))
	if err != nil {
		return nil, nil, &phase4RunAttemptError{Attempt: attempt, Stage: "resolve_server", Err: err}
	}
	selection, probe, err := s.gpuSelector.SelectGPU(ctx, serverNode)
	if err != nil {
		return nil, nil, &phase4RunAttemptError{Attempt: attempt, Stage: "select_gpu", Err: err}
	}
	remotePaths, err := buildPhase4RemotePaths(runManifest.ID, selection.GPUIndex, s.phase4RemoteWorkRoot)
	if err != nil {
		return nil, nil, &phase4RunAttemptError{Attempt: attempt, Stage: "remote_paths", Err: err}
	}
	if err = s.materializePhase4RemoteFiles(runManifest, datasetProfile, artifactPaths, remotePaths, *selection); err != nil {
		return nil, nil, &phase4RunAttemptError{Attempt: attempt, Stage: "materialize_remote_files", Err: err}
	}
	artifactPaths = s.attachPhase4RemoteArtifactPaths(artifactPaths, remotePaths, *selection, probe)
	serverID := serverNode.ID
	gpu := fmt.Sprintf("%d", selection.GPUIndex)
	if _, err = s.phase4.UpdateRunManifest(ctx, runManifest.ID, model.Phase4RunManifestUpdateRequest{
		ServerID:      &serverID,
		GPU:           &gpu,
		ArtifactPaths: &artifactPaths,
		LogsPath:      stringPtr(stringValue(artifactPaths["logs_dir"])),
		MetricsPath:   stringPtr(stringValue(artifactPaths["metrics_path"])),
	}); err != nil {
		return nil, nil, &phase4RunAttemptError{Attempt: attempt, Stage: "update_manifest_remote", Err: err}
	}
	if _, err = s.remoteExecutor.Execute(ctx, phase4RemoteRunRequest{
		Server:         serverNode,
		RunManifest:    runManifest,
		DatasetProfile: datasetProfile,
		ArtifactPaths:  artifactPaths,
		RemotePaths:    remotePaths,
		GPUSelection:   selection,
		CommandTimeout: s.commandTimeout,
	}); err != nil {
		return nil, nil, &phase4RunAttemptError{Attempt: attempt, Stage: "shenzhenvlab_remote_run", Err: err}
	}
	metricsRaw := readJSON(stringValue(artifactPaths["metrics_path"]))
	return metricsRaw, summarizePhase4Metrics(metricsRaw), nil
}

func (s *Phase4Service) finalizePhase4Success(ctx context.Context, job *model.AgentJob, runManifest *model.Phase4RunManifest, datasetProfile *model.Phase4DatasetProfile, idea *model.Phase4Idea, readerContext *model.Phase4ReaderContext, artifactPaths map[string]any, snapshotID string, metricsRaw map[string]any, metricsSummary map[string]any, retryCount int, failures []map[string]any) error {
	payload := updatePhase4CodingJobPayload(job.NormalizedPayload, runManifest.ID, snapshotID, artifactPaths, metricsSummary, stringValue(artifactPaths["human_report_path"]))
	payload["repair_history"] = failures
	payload["retry_count"] = retryCount
	job.NormalizedPayload = payload
	job.UpdatedAt = time.Now()
	if err := s.jobUpdates.Update(ctx, *job); err != nil {
		return err
	}
	if err := s.persistPhase4Artifacts(ctx, job.ID, artifactPaths); err != nil {
		return err
	}
	finishedAt := time.Now()
	if _, err := s.phase4.UpdateRunManifest(ctx, runManifest.ID, model.Phase4RunManifestUpdateRequest{
		ArtifactPaths: &artifactPaths,
		LogsPath:      stringPtr(stringValue(artifactPaths["logs_dir"])),
		MetricsPath:   stringPtr(stringValue(artifactPaths["metrics_path"])),
		RetryCount:    intPtr(retryCount),
		FailureFeedback: mapPtr(map[string]any{
			"status":         "succeeded",
			"retry_count":    retryCount,
			"attempts_used":  retryCount + 1,
			"repair_history": failures,
		}),
	}); err != nil {
		return err
	}
	if _, err := s.phase4.UpdateRunManifestStatus(ctx, runManifest.ID, model.Phase4RunManifestStatusUpdateRequest{
		Status:     model.Phase4RunStatusSucceeded,
		RetryCount: intPtr(retryCount),
		FinishedAt: &finishedAt,
	}); err != nil {
		return err
	}
	if idea != nil && strings.TrimSpace(idea.Status) == model.Phase4IdeaStatusSelected {
		_, _ = s.phase4.UpdateIdeaStatus(ctx, idea.ID, model.Phase4IdeaStatusUpdateRequest{Status: model.Phase4IdeaStatusImplemented})
	}
	appendPhase4RepairLog(stringValue(artifactPaths["repair_log_path"]), map[string]any{
		"status":        "succeeded",
		"retry_count":   retryCount,
		"attempts_used": retryCount + 1,
		"recorded_at":   time.Now().Format(time.RFC3339Nano),
	})
	return s.publishPhase4RunReady(ctx, *runManifest, *datasetProfile, *idea, *readerContext, artifactPaths, metricsRaw)
}

func (s *Phase4Service) finalizePhase4TestFailure(ctx context.Context, job *model.AgentJob, runManifest *model.Phase4RunManifest, idea *model.Phase4Idea, artifactPaths map[string]any, snapshotID string, failures []map[string]any) error {
	finalFeedback := map[string]any{
		"status":          model.Phase4RunStatusTestFailed,
		"attempts_used":   len(failures),
		"max_retry_count": runManifest.MaxRetryCount,
		"failure_history": failures,
		"logs_path":       stringValue(artifactPaths["logs_dir"]),
		"repair_log_path": stringValue(artifactPaths["repair_log_path"]),
	}
	if len(failures) > 0 {
		for key, value := range failures[len(failures)-1] {
			if _, exists := finalFeedback[key]; !exists {
				finalFeedback[key] = value
			}
		}
	}
	if failureFeedbackPath := strings.TrimSpace(stringValue(artifactPaths["failure_feedback_path"])); failureFeedbackPath != "" {
		if err := writeJSON(failureFeedbackPath, finalFeedback); err != nil {
			return err
		}
	}
	if idea != nil {
		_, _ = s.phase4.UpdateIdeaStatus(ctx, idea.ID, model.Phase4IdeaStatusUpdateRequest{
			Status:           idea.Status,
			FailureFeedback:  finalFeedback,
			LastFailureRunID: runManifest.ID,
		})
	}
	if s.ideaRevisions != nil && idea != nil {
		if revisionResult, err := s.ideaRevisions.GenerateRevisionCandidates(ctx, idea.ID, model.Phase4IdeaRevisionGenerateRequest{
			FailureFeedback:  finalFeedback,
			LastFailureRunID: runManifest.ID,
			TargetCount:      3,
			ExecutionMode:    firstNonEmpty(job.ExecutionMode, "api"),
			ModelProvider:    "phase4_idea",
			ModelName:        "idea-phase4-default",
			PromptVersion:    "v1",
		}); err == nil && revisionResult != nil {
			if revisionResult.Job != nil {
				artifactPaths["revision_job_id"] = revisionResult.Job.ID
			}
			revisionIDs := make([]string, 0, len(revisionResult.Ideas))
			topIDs := make([]string, 0, len(revisionResult.TopRecommendations))
			for _, item := range revisionResult.Ideas {
				revisionIDs = append(revisionIDs, item.ID)
			}
			for _, item := range revisionResult.TopRecommendations {
				topIDs = append(topIDs, item.ID)
			}
			artifactPaths["revision_idea_ids"] = revisionIDs
			artifactPaths["revision_top_idea_ids"] = topIDs
		} else if err != nil {
			job.Warnings = append(job.Warnings, "phase4 idea revision generation failed: "+err.Error())
		}
	}
	job.NormalizedPayload = updatePhase4CodingJobPayload(job.NormalizedPayload, runManifest.ID, snapshotID, artifactPaths, map[string]any{"status": model.Phase4RunStatusTestFailed}, stringValue(artifactPaths["human_report_path"]))
	job.NormalizedPayload["failure_feedback"] = finalFeedback
	job.NormalizedPayload["repair_history"] = failures
	job.Warnings = append(job.Warnings, fmt.Sprintf("phase4 coding reached test_failed after %d attempt(s)", len(failures)))
	job.UpdatedAt = time.Now()
	if err := s.jobUpdates.Update(ctx, *job); err != nil {
		return err
	}
	if err := s.persistPhase4Artifacts(ctx, job.ID, artifactPaths); err != nil {
		return err
	}
	finishedAt := time.Now()
	if _, err := s.phase4.UpdateRunManifest(ctx, runManifest.ID, model.Phase4RunManifestUpdateRequest{
		ArtifactPaths:   &artifactPaths,
		LogsPath:        stringPtr(stringValue(artifactPaths["logs_dir"])),
		MetricsPath:     stringPtr(stringValue(artifactPaths["metrics_path"])),
		FailureFeedback: &finalFeedback,
		RetryCount:      intPtr(max(0, len(failures)-1)),
	}); err != nil {
		return err
	}
	if _, err := s.phase4.UpdateRunManifestStatus(ctx, runManifest.ID, model.Phase4RunManifestStatusUpdateRequest{
		Status:          model.Phase4RunStatusTestFailed,
		RetryCount:      intPtr(max(0, len(failures)-1)),
		FailureFeedback: finalFeedback,
		FinishedAt:      &finishedAt,
	}); err != nil {
		return err
	}
	appendPhase4RepairLog(stringValue(artifactPaths["repair_log_path"]), map[string]any{
		"status":        model.Phase4RunStatusTestFailed,
		"attempts_used": len(failures),
		"recorded_at":   time.Now().Format(time.RFC3339Nano),
	})
	return nil
}

func (s *Phase4Service) applyPhase4Repair(artifactPaths map[string]any, payload model.Phase4CodingRuntimePayload, idea *model.Phase4Idea, lastFailure map[string]any, repairRound int) (map[string]any, error) {
	methodModulePath := strings.TrimSpace(stringValue(artifactPaths["method_module_path"]))
	configPath := strings.TrimSpace(stringValue(artifactPaths["config_path"]))
	snapshotDir := strings.TrimSpace(stringValue(artifactPaths["snapshot_dir"]))
	if methodModulePath == "" || configPath == "" || snapshotDir == "" {
		return nil, fmt.Errorf("phase4 repair requires method module path, config path, and snapshot dir")
	}
	methodSlug := firstNonEmpty(strings.TrimSpace(payload.MethodModule.ModuleName), strings.TrimSuffix(filepath.Base(methodModulePath), filepath.Ext(methodModulePath)), "generated_method")
	reason := firstNonEmpty(stringValue(lastFailure["error"]), stringValue(lastFailure["stage"]), "phase4_repair")
	strategy := "runtime_error_first"
	switch repairRound {
	case 1:
		strategy = "runtime_error_first"
	case 2:
		strategy = "small_code_or_param_adjustment"
	default:
		strategy = "fallback_to_previous_stable_snapshot"
		if err := s.rollbackPhase4Snapshot(artifactPaths); err != nil {
			return nil, err
		}
	}
	content := buildPhase4FallbackMethodModule(methodSlug, idea, reason, strategy, repairRound)
	if err := os.WriteFile(methodModulePath, []byte(content), 0o644); err != nil {
		return nil, err
	}
	if err := applyPhase4RepairConfig(configPath, repairRound, reason); err != nil {
		return nil, err
	}
	if repairRound >= 3 {
		relativeMethodPath := filepath.ToSlash(relPathWithin(snapshotDir, methodModulePath))
		configPayload := readJSON(configPath)
		configPayload["method_module_path"] = relativeMethodPath
		if err := writeJSON(configPath, configPayload); err != nil {
			return nil, err
		}
	}
	return map[string]any{
		"repair_round": repairRound,
		"strategy":     strategy,
		"reason":       reason,
		"method_path":  methodModulePath,
		"config_path":  configPath,
		"recorded_at":  time.Now().Format(time.RFC3339Nano),
	}, nil
}

func (s *Phase4Service) rollbackPhase4Snapshot(artifactPaths map[string]any) error {
	sourceRoot := strings.TrimSpace(stringValue(artifactPaths["source_root"]))
	snapshotDir := strings.TrimSpace(stringValue(artifactPaths["snapshot_dir"]))
	methodModulePath := strings.TrimSpace(stringValue(artifactPaths["method_module_path"]))
	if sourceRoot == "" || snapshotDir == "" {
		return fmt.Errorf("phase4 snapshot rollback requires source_root and snapshot_dir")
	}
	if err := os.RemoveAll(snapshotDir); err != nil {
		return err
	}
	if err := copyDirectory(sourceRoot, snapshotDir); err != nil {
		return err
	}
	if methodModulePath != "" {
		if err := os.MkdirAll(filepath.Dir(methodModulePath), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func buildPhase4FallbackMethodModule(methodSlug string, idea *model.Phase4Idea, reason string, strategy string, repairRound int) string {
	title := methodSlug
	if idea != nil && strings.TrimSpace(idea.Title) != "" {
		title = strings.TrimSpace(idea.Title)
	}
	notes := []string{
		fmt.Sprintf("title=%s", title),
		fmt.Sprintf("repair_strategy=%s", strategy),
		fmt.Sprintf("repair_round=%d", repairRound),
		fmt.Sprintf("fallback_reason=%s", reason),
	}
	renderedNotes := make([]string, 0, len(notes))
	for _, item := range notes {
		renderedNotes = append(renderedNotes, fmt.Sprintf("%q", item))
	}
	return strings.Join([]string{
		"from methods.page_lexical_retrieval import PageLexicalRetrievalMethod",
		"",
		"def build_method():",
		fmt.Sprintf("    return PageLexicalRetrievalMethod(name=%q, method_tags=['repair', 'page', 'lexical'], score_bias=0.01, retrieval_notes=[%s], top_k=10, query_expansion_terms=[], title_match_bonus=0.25, ocr_match_bonus=0.15, section_match_bonus=0.12, exact_phrase_bonus=0.10)", methodSlug, strings.Join(renderedNotes, ", ")),
		"",
	}, "\n")
}

func applyPhase4RepairConfig(configPath string, repairRound int, reason string) error {
	configPayload := readJSON(configPath)
	parameters := mapValue(configPayload["parameters"])
	scoringProfile := mapValue(parameters["scoring_profile"])
	if repairRound >= 2 {
		parameters["query_expansion_terms"] = []string{}
	}
	if len(scoringProfile) == 0 {
		scoringProfile = map[string]any{}
	}
	scoringProfile["title_match_bonus"] = 0.25
	scoringProfile["ocr_match_bonus"] = 0.15
	scoringProfile["section_match_bonus"] = 0.12
	scoringProfile["exact_phrase_bonus"] = 0.10
	scoringProfile["repair_reason"] = reason
	parameters["scoring_profile"] = scoringProfile
	parameters["repair_round"] = repairRound
	configPayload["parameters"] = parameters
	notes := phase4CodingStringSlice(configPayload["notes"])
	notes = append(notes, fmt.Sprintf("repair_round=%d", repairRound), fmt.Sprintf("repair_reason=%s", reason))
	configPayload["notes"] = notes
	return writeJSON(configPath, configPayload)
}

func clearPhase4AttemptOutputs(artifactPaths map[string]any) error {
	for _, key := range []string{
		"metrics_path",
		"machine_report_path",
		"human_report_path",
		"dataset_tool_asset_path",
		"dataset_adapter_path",
		"evaluate_tool_asset_path",
		"eval_summary_path",
		"predictions_path",
		"run_log_path",
		"bootstrap_stdout_path",
		"bootstrap_stderr_path",
		"runtime_stdout_path",
		"runtime_stderr_path",
		"failure_feedback_path",
	} {
		target := strings.TrimSpace(stringValue(artifactPaths[key]))
		if target == "" {
			continue
		}
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func summarizePhase4Metrics(metricsRaw map[string]any) map[string]any {
	metricsSummary := map[string]any{}
	if values := mapValue(metricsRaw["values"]); len(values) > 0 {
		metricsSummary = values
		if primary := stringValue(metricsRaw["primary_metric"]); primary != "" {
			metricsSummary["primary_metric"] = primary
		}
	}
	if len(metricsSummary) == 0 {
		metricsSummary["status"] = "missing"
	}
	return metricsSummary
}

func (s *Phase4Service) buildPhase4FailureFeedback(runManifest *model.Phase4RunManifest, artifactPaths map[string]any, attempt int, maxRetries int, stage string, execErr error, history []map[string]any) map[string]any {
	feedback := map[string]any{
		"run_manifest_id": runManifest.ID,
		"stage":           stage,
		"error":           execErr.Error(),
		"attempt":         attempt,
		"max_retry_count": maxRetries,
		"logs_path":       stringValue(artifactPaths["logs_dir"]),
		"driver_log_path": stringValue(artifactPaths["driver_log_path"]),
		"method_module":   stringValue(artifactPaths["method_module_path"]),
		"config_path":     stringValue(artifactPaths["config_path"]),
		"recorded_at":     time.Now().Format(time.RFC3339Nano),
	}
	if len(history) > 0 {
		feedback["previous_failures"] = history
	}
	return feedback
}

func appendPhase4RepairLog(path string, payload map[string]any) {
	if strings.TrimSpace(path) == "" || len(payload) == 0 {
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = file.WriteString(string(raw) + "\n")
}

func phase4AttemptStage(err error) string {
	var attemptErr *phase4RunAttemptError
	if errors.As(err, &attemptErr) && attemptErr != nil && strings.TrimSpace(attemptErr.Stage) != "" {
		return attemptErr.Stage
	}
	return "run_attempt"
}

func intPtr(value int) *int {
	return &value
}

func mapPtr(value map[string]any) *map[string]any {
	return &value
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
