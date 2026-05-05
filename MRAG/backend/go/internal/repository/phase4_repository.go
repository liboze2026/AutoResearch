package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"mrag-platform/backend/go/internal/model"
)

type Phase4Repository struct {
	db *sql.DB
}

func NewPhase4Repository(db *sql.DB) *Phase4Repository {
	return &Phase4Repository{db: db}
}

func (r *Phase4Repository) ListDatasetProfiles(ctx context.Context, taskType string, status string) ([]model.Phase4DatasetProfile, error) {
	query := `SELECT p.id,p.dataset_name,p.task_type,p.modality_composition_json,p.splits_json,p.label_schema_json,p.file_structure_snapshot_json,p.sample_statistics_json,p.official_metric,p.official_baseline,p.license,p.citation,p.known_difficulties_json,p.user_notes,p.metadata_json,p.source_mode,COALESCE(p.server_id,''),COALESCE(s.name,''),p.server_path,p.status,p.created_at,p.updated_at
	FROM phase4_dataset_profiles p
	LEFT JOIN servers s ON s.id = p.server_id
	WHERE 1=1`
	args := make([]any, 0, 2)
	index := 1
	if strings.TrimSpace(taskType) != "" {
		query += fmt.Sprintf(" AND p.task_type=$%d", index)
		args = append(args, strings.TrimSpace(strings.ToLower(taskType)))
		index++
	}
	if strings.TrimSpace(status) != "" {
		query += fmt.Sprintf(" AND p.status=$%d", index)
		args = append(args, strings.TrimSpace(strings.ToLower(status)))
		index++
	}
	query += " ORDER BY p.updated_at DESC, p.created_at DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Phase4DatasetProfile, 0)
	for rows.Next() {
		item, scanErr := scanPhase4DatasetProfile(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Phase4Repository) GetDatasetProfileByID(ctx context.Context, id string) (*model.Phase4DatasetProfile, error) {
	item, err := scanPhase4DatasetProfile(r.db.QueryRowContext(ctx, `SELECT p.id,p.dataset_name,p.task_type,p.modality_composition_json,p.splits_json,p.label_schema_json,p.file_structure_snapshot_json,p.sample_statistics_json,p.official_metric,p.official_baseline,p.license,p.citation,p.known_difficulties_json,p.user_notes,p.metadata_json,p.source_mode,COALESCE(p.server_id,''),COALESCE(s.name,''),p.server_path,p.status,p.created_at,p.updated_at
	FROM phase4_dataset_profiles p
	LEFT JOIN servers s ON s.id = p.server_id
	WHERE p.id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Phase4Repository) CreateDatasetProfile(ctx context.Context, item model.Phase4DatasetProfile) error {
	modalityRaw, _ := json.Marshal(item.ModalityComposition)
	splitsRaw, _ := json.Marshal(item.Splits)
	labelSchemaRaw, _ := json.Marshal(item.LabelSchema)
	fileSnapshotRaw, _ := json.Marshal(item.FileStructureSnapshot)
	sampleStatsRaw, _ := json.Marshal(item.SampleStatistics)
	difficultiesRaw, _ := json.Marshal(item.KnownDifficulties)
	metadataRaw, _ := json.Marshal(item.Metadata)
	_, err := r.db.ExecContext(ctx, `INSERT INTO phase4_dataset_profiles (id,dataset_name,task_type,modality_composition_json,splits_json,label_schema_json,file_structure_snapshot_json,sample_statistics_json,official_metric,official_baseline,license,citation,known_difficulties_json,user_notes,metadata_json,source_mode,server_id,server_path,status,created_at,updated_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,NULLIF($17,''),$18,$19,$20,$21)`,
		item.ID, item.DatasetName, item.TaskType, modalityRaw, splitsRaw, labelSchemaRaw, fileSnapshotRaw, sampleStatsRaw, item.OfficialMetric, item.OfficialBaseline, item.License, item.Citation, difficultiesRaw, item.UserNotes, metadataRaw, item.SourceMode, item.ServerID, item.ServerPath, item.Status, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) UpdateDatasetProfile(ctx context.Context, item model.Phase4DatasetProfile) error {
	modalityRaw, _ := json.Marshal(item.ModalityComposition)
	splitsRaw, _ := json.Marshal(item.Splits)
	labelSchemaRaw, _ := json.Marshal(item.LabelSchema)
	fileSnapshotRaw, _ := json.Marshal(item.FileStructureSnapshot)
	sampleStatsRaw, _ := json.Marshal(item.SampleStatistics)
	difficultiesRaw, _ := json.Marshal(item.KnownDifficulties)
	metadataRaw, _ := json.Marshal(item.Metadata)
	_, err := r.db.ExecContext(ctx, `UPDATE phase4_dataset_profiles SET dataset_name=$2,task_type=$3,modality_composition_json=$4,splits_json=$5,label_schema_json=$6,file_structure_snapshot_json=$7,sample_statistics_json=$8,official_metric=$9,official_baseline=$10,license=$11,citation=$12,known_difficulties_json=$13,user_notes=$14,metadata_json=$15,source_mode=$16,server_id=NULLIF($17,''),server_path=$18,status=$19,updated_at=$20 WHERE id=$1`,
		item.ID, item.DatasetName, item.TaskType, modalityRaw, splitsRaw, labelSchemaRaw, fileSnapshotRaw, sampleStatsRaw, item.OfficialMetric, item.OfficialBaseline, item.License, item.Citation, difficultiesRaw, item.UserNotes, metadataRaw, item.SourceMode, item.ServerID, item.ServerPath, item.Status, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) DeleteDatasetProfile(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM phase4_dataset_profiles WHERE id=$1`, id)
	return err
}

func (r *Phase4Repository) ListReaderSources(ctx context.Context, datasetProfileID string) ([]model.Phase4ReaderSource, error) {
	query := `SELECT id,COALESCE(dataset_profile_id,''),title,authors_json,venue,publication_year,source_type,source_url,open_access_url,quality_tier,ranking_score,quality_score,relevance_score,citation_count,metadata_json,created_at,updated_at
	FROM phase4_reader_sources`
	args := make([]any, 0, 1)
	if strings.TrimSpace(datasetProfileID) != "" {
		query += ` WHERE dataset_profile_id=$1`
		args = append(args, datasetProfileID)
	}
	query += ` ORDER BY updated_at DESC, created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Phase4ReaderSource, 0)
	for rows.Next() {
		item, scanErr := scanPhase4ReaderSource(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Phase4Repository) GetReaderSourceByID(ctx context.Context, id string) (*model.Phase4ReaderSource, error) {
	item, err := scanPhase4ReaderSource(r.db.QueryRowContext(ctx, `SELECT id,COALESCE(dataset_profile_id,''),title,authors_json,venue,publication_year,source_type,source_url,open_access_url,quality_tier,ranking_score,quality_score,relevance_score,citation_count,metadata_json,created_at,updated_at
	FROM phase4_reader_sources WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Phase4Repository) CreateReaderSource(ctx context.Context, item model.Phase4ReaderSource) error {
	authorsRaw, _ := json.Marshal(item.Authors)
	metadataRaw, _ := json.Marshal(item.Metadata)
	_, err := r.db.ExecContext(ctx, `INSERT INTO phase4_reader_sources (id,dataset_profile_id,title,authors_json,venue,publication_year,source_type,source_url,open_access_url,quality_tier,ranking_score,quality_score,relevance_score,citation_count,metadata_json,created_at,updated_at)
	VALUES ($1,NULLIF($2,''),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		item.ID, item.DatasetProfileID, item.Title, authorsRaw, item.Venue, item.PublicationYear, item.SourceType, item.SourceURL, item.OpenAccessURL, item.QualityTier, item.RankingScore, item.QualityScore, item.RelevanceScore, item.CitationCount, metadataRaw, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) UpdateReaderSource(ctx context.Context, item model.Phase4ReaderSource) error {
	authorsRaw, _ := json.Marshal(item.Authors)
	metadataRaw, _ := json.Marshal(item.Metadata)
	_, err := r.db.ExecContext(ctx, `UPDATE phase4_reader_sources SET title=$2,authors_json=$3,venue=$4,publication_year=$5,source_type=$6,source_url=$7,open_access_url=$8,quality_tier=$9,ranking_score=$10,quality_score=$11,relevance_score=$12,citation_count=$13,metadata_json=$14,updated_at=$15 WHERE id=$1`,
		item.ID, item.Title, authorsRaw, item.Venue, item.PublicationYear, item.SourceType, item.SourceURL, item.OpenAccessURL, item.QualityTier, item.RankingScore, item.QualityScore, item.RelevanceScore, item.CitationCount, metadataRaw, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) ListReaderContexts(ctx context.Context, datasetProfileID string) ([]model.Phase4ReaderContext, error) {
	query := `SELECT id,COALESCE(dataset_profile_id,''),title,summary,task_definition,related_work_json,retrieval_focus_json,ranking_notes,source_ids_json,structured_context_json,status,created_at,updated_at
	FROM phase4_reader_contexts`
	args := make([]any, 0, 1)
	if strings.TrimSpace(datasetProfileID) != "" {
		query += ` WHERE dataset_profile_id=$1`
		args = append(args, datasetProfileID)
	}
	query += ` ORDER BY updated_at DESC, created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Phase4ReaderContext, 0)
	for rows.Next() {
		item, scanErr := scanPhase4ReaderContext(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Phase4Repository) GetReaderContextByID(ctx context.Context, id string) (*model.Phase4ReaderContext, error) {
	item, err := scanPhase4ReaderContext(r.db.QueryRowContext(ctx, `SELECT id,COALESCE(dataset_profile_id,''),title,summary,task_definition,related_work_json,retrieval_focus_json,ranking_notes,source_ids_json,structured_context_json,status,created_at,updated_at
	FROM phase4_reader_contexts WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Phase4Repository) CreateReaderContext(ctx context.Context, item model.Phase4ReaderContext) error {
	relatedWorkRaw, _ := json.Marshal(item.RelatedWork)
	retrievalFocusRaw, _ := json.Marshal(item.RetrievalFocus)
	sourceIDsRaw, _ := json.Marshal(item.SourceIDs)
	structuredRaw, _ := json.Marshal(item.StructuredContext)
	_, err := r.db.ExecContext(ctx, `INSERT INTO phase4_reader_contexts (id,dataset_profile_id,title,summary,task_definition,related_work_json,retrieval_focus_json,ranking_notes,source_ids_json,structured_context_json,status,created_at,updated_at)
	VALUES ($1,NULLIF($2,''),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		item.ID, item.DatasetProfileID, item.Title, item.Summary, item.TaskDefinition, relatedWorkRaw, retrievalFocusRaw, item.RankingNotes, sourceIDsRaw, structuredRaw, item.Status, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) UpdateReaderContext(ctx context.Context, item model.Phase4ReaderContext) error {
	relatedWorkRaw, _ := json.Marshal(item.RelatedWork)
	retrievalFocusRaw, _ := json.Marshal(item.RetrievalFocus)
	sourceIDsRaw, _ := json.Marshal(item.SourceIDs)
	structuredRaw, _ := json.Marshal(item.StructuredContext)
	_, err := r.db.ExecContext(ctx, `UPDATE phase4_reader_contexts SET title=$2,summary=$3,task_definition=$4,related_work_json=$5,retrieval_focus_json=$6,ranking_notes=$7,source_ids_json=$8,structured_context_json=$9,status=$10,updated_at=$11 WHERE id=$1`,
		item.ID, item.Title, item.Summary, item.TaskDefinition, relatedWorkRaw, retrievalFocusRaw, item.RankingNotes, sourceIDsRaw, structuredRaw, item.Status, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) ListIdeas(ctx context.Context, datasetProfileID string, status string) ([]model.Phase4Idea, error) {
	query := `SELECT id,COALESCE(dataset_profile_id,''),COALESCE(reader_context_id,''),title,problem_definition,core_method,differentiators,data_processing_needs_json,model_changes_json,training_plan,evaluation_metrics_json,risk_points_json,expected_gains_json,score_json,score_summary_json,status,source_type,COALESCE(revision_of_id,''),COALESCE(lineage_root_id,''),failure_feedback_json,last_failure_run_id,created_at,updated_at
	FROM phase4_ideas WHERE 1=1`
	args := make([]any, 0, 2)
	index := 1
	if strings.TrimSpace(datasetProfileID) != "" {
		query += fmt.Sprintf(" AND dataset_profile_id=$%d", index)
		args = append(args, datasetProfileID)
		index++
	}
	if strings.TrimSpace(status) != "" {
		query += fmt.Sprintf(" AND status=$%d", index)
		args = append(args, strings.TrimSpace(strings.ToLower(status)))
		index++
	}
	query += ` ORDER BY updated_at DESC, created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Phase4Idea, 0)
	for rows.Next() {
		item, scanErr := scanPhase4Idea(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Phase4Repository) GetIdeaByID(ctx context.Context, id string) (*model.Phase4Idea, error) {
	item, err := scanPhase4Idea(r.db.QueryRowContext(ctx, `SELECT id,COALESCE(dataset_profile_id,''),COALESCE(reader_context_id,''),title,problem_definition,core_method,differentiators,data_processing_needs_json,model_changes_json,training_plan,evaluation_metrics_json,risk_points_json,expected_gains_json,score_json,score_summary_json,status,source_type,COALESCE(revision_of_id,''),COALESCE(lineage_root_id,''),failure_feedback_json,last_failure_run_id,created_at,updated_at
	FROM phase4_ideas WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Phase4Repository) CreateIdea(ctx context.Context, item model.Phase4Idea) error {
	dataProcessingRaw, _ := json.Marshal(item.DataProcessingNeeds)
	modelChangesRaw, _ := json.Marshal(item.ModelChanges)
	evalMetricsRaw, _ := json.Marshal(item.EvaluationMetrics)
	risksRaw, _ := json.Marshal(item.RiskPoints)
	expectedGainsRaw, _ := json.Marshal(item.ExpectedGains)
	scoreRaw, _ := json.Marshal(item.Score)
	scoreSummaryRaw, _ := json.Marshal(item.ScoreSummary)
	failureRaw, _ := json.Marshal(item.FailureFeedback)
	_, err := r.db.ExecContext(ctx, `INSERT INTO phase4_ideas (id,dataset_profile_id,reader_context_id,title,problem_definition,core_method,differentiators,data_processing_needs_json,model_changes_json,training_plan,evaluation_metrics_json,risk_points_json,expected_gains_json,score_json,score_summary_json,status,source_type,revision_of_id,lineage_root_id,failure_feedback_json,last_failure_run_id,created_at,updated_at)
	VALUES ($1,NULLIF($2,''),NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NULLIF($18,''),NULLIF($19,''),$20,$21,$22,$23)`,
		item.ID, item.DatasetProfileID, item.ReaderContextID, item.Title, item.ProblemDefinition, item.CoreMethod, item.Differentiators, dataProcessingRaw, modelChangesRaw, item.TrainingPlan, evalMetricsRaw, risksRaw, expectedGainsRaw, scoreRaw, scoreSummaryRaw, item.Status, item.SourceType, item.RevisionOfID, item.LineageRootID, failureRaw, item.LastFailureRunID, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) UpdateIdea(ctx context.Context, item model.Phase4Idea) error {
	dataProcessingRaw, _ := json.Marshal(item.DataProcessingNeeds)
	modelChangesRaw, _ := json.Marshal(item.ModelChanges)
	evalMetricsRaw, _ := json.Marshal(item.EvaluationMetrics)
	risksRaw, _ := json.Marshal(item.RiskPoints)
	expectedGainsRaw, _ := json.Marshal(item.ExpectedGains)
	scoreRaw, _ := json.Marshal(item.Score)
	scoreSummaryRaw, _ := json.Marshal(item.ScoreSummary)
	failureRaw, _ := json.Marshal(item.FailureFeedback)
	_, err := r.db.ExecContext(ctx, `UPDATE phase4_ideas SET dataset_profile_id=NULLIF($2,''),reader_context_id=NULLIF($3,''),title=$4,problem_definition=$5,core_method=$6,differentiators=$7,data_processing_needs_json=$8,model_changes_json=$9,training_plan=$10,evaluation_metrics_json=$11,risk_points_json=$12,expected_gains_json=$13,score_json=$14,score_summary_json=$15,status=$16,source_type=$17,revision_of_id=NULLIF($18,''),lineage_root_id=NULLIF($19,''),failure_feedback_json=$20,last_failure_run_id=$21,updated_at=$22 WHERE id=$1`,
		item.ID, item.DatasetProfileID, item.ReaderContextID, item.Title, item.ProblemDefinition, item.CoreMethod, item.Differentiators, dataProcessingRaw, modelChangesRaw, item.TrainingPlan, evalMetricsRaw, risksRaw, expectedGainsRaw, scoreRaw, scoreSummaryRaw, item.Status, item.SourceType, item.RevisionOfID, item.LineageRootID, failureRaw, item.LastFailureRunID, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) DeleteIdea(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM phase4_ideas WHERE id=$1`, id)
	return err
}

func (r *Phase4Repository) ListRunManifests(ctx context.Context, datasetProfileID string, ideaID string, status string) ([]model.Phase4RunManifest, error) {
	query := `SELECT id,dataset_profile_id,idea_id,COALESCE(reader_context_id,''),code_snapshot_id,runner_mode,COALESCE(server_id,''),gpu,status,retry_count,max_retry_count,artifact_paths_json,logs_path,metrics_path,failure_feedback_json,created_at,updated_at,started_at,finished_at
	FROM phase4_run_manifests WHERE 1=1`
	args := make([]any, 0, 3)
	index := 1
	if strings.TrimSpace(datasetProfileID) != "" {
		query += fmt.Sprintf(" AND dataset_profile_id=$%d", index)
		args = append(args, datasetProfileID)
		index++
	}
	if strings.TrimSpace(ideaID) != "" {
		query += fmt.Sprintf(" AND idea_id=$%d", index)
		args = append(args, ideaID)
		index++
	}
	if strings.TrimSpace(status) != "" {
		query += fmt.Sprintf(" AND status=$%d", index)
		args = append(args, strings.TrimSpace(strings.ToLower(status)))
		index++
	}
	query += ` ORDER BY updated_at DESC, created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Phase4RunManifest, 0)
	for rows.Next() {
		item, scanErr := scanPhase4RunManifest(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Phase4Repository) GetRunManifestByID(ctx context.Context, id string) (*model.Phase4RunManifest, error) {
	item, err := scanPhase4RunManifest(r.db.QueryRowContext(ctx, `SELECT id,dataset_profile_id,idea_id,COALESCE(reader_context_id,''),code_snapshot_id,runner_mode,COALESCE(server_id,''),gpu,status,retry_count,max_retry_count,artifact_paths_json,logs_path,metrics_path,failure_feedback_json,created_at,updated_at,started_at,finished_at
	FROM phase4_run_manifests WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Phase4Repository) CreateRunManifest(ctx context.Context, item model.Phase4RunManifest) error {
	artifactRaw, _ := json.Marshal(item.ArtifactPaths)
	failureRaw, _ := json.Marshal(item.FailureFeedback)
	_, err := r.db.ExecContext(ctx, `INSERT INTO phase4_run_manifests (id,dataset_profile_id,idea_id,reader_context_id,code_snapshot_id,runner_mode,server_id,gpu,status,retry_count,max_retry_count,artifact_paths_json,logs_path,metrics_path,failure_feedback_json,created_at,updated_at,started_at,finished_at)
	VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,NULLIF($7,''),$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		item.ID, item.DatasetProfileID, item.IdeaID, item.ReaderContextID, item.CodeSnapshotID, item.RunnerMode, item.ServerID, item.GPU, item.Status, item.RetryCount, item.MaxRetryCount, artifactRaw, item.LogsPath, item.MetricsPath, failureRaw, item.CreatedAt, item.UpdatedAt, item.StartedAt, item.FinishedAt,
	)
	return err
}

func (r *Phase4Repository) UpdateRunManifest(ctx context.Context, item model.Phase4RunManifest) error {
	artifactRaw, _ := json.Marshal(item.ArtifactPaths)
	failureRaw, _ := json.Marshal(item.FailureFeedback)
	_, err := r.db.ExecContext(ctx, `UPDATE phase4_run_manifests SET dataset_profile_id=$2,idea_id=$3,reader_context_id=NULLIF($4,''),code_snapshot_id=$5,runner_mode=$6,server_id=NULLIF($7,''),gpu=$8,status=$9,retry_count=$10,max_retry_count=$11,artifact_paths_json=$12,logs_path=$13,metrics_path=$14,failure_feedback_json=$15,updated_at=$16,started_at=$17,finished_at=$18 WHERE id=$1`,
		item.ID, item.DatasetProfileID, item.IdeaID, item.ReaderContextID, item.CodeSnapshotID, item.RunnerMode, item.ServerID, item.GPU, item.Status, item.RetryCount, item.MaxRetryCount, artifactRaw, item.LogsPath, item.MetricsPath, failureRaw, item.UpdatedAt, item.StartedAt, item.FinishedAt,
	)
	return err
}

func (r *Phase4Repository) ListStructuredReports(ctx context.Context, runManifestID string) ([]model.Phase4StructuredReportRecord, error) {
	query := `SELECT id,run_manifest_id,COALESCE(dataset_profile_id,''),COALESCE(idea_id,''),COALESCE(reader_context_id,''),title,machine_readable_report_json,human_readable_report_md,citation_refs_json,reference_source_ids_json,status,created_at,updated_at
	FROM phase4_structured_reports`
	args := make([]any, 0, 1)
	if strings.TrimSpace(runManifestID) != "" {
		query += ` WHERE run_manifest_id=$1`
		args = append(args, runManifestID)
	}
	query += ` ORDER BY updated_at DESC, created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Phase4StructuredReportRecord, 0)
	for rows.Next() {
		item, scanErr := scanPhase4StructuredReport(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Phase4Repository) GetStructuredReportByID(ctx context.Context, id string) (*model.Phase4StructuredReportRecord, error) {
	item, err := scanPhase4StructuredReport(r.db.QueryRowContext(ctx, `SELECT id,run_manifest_id,COALESCE(dataset_profile_id,''),COALESCE(idea_id,''),COALESCE(reader_context_id,''),title,machine_readable_report_json,human_readable_report_md,citation_refs_json,reference_source_ids_json,status,created_at,updated_at
	FROM phase4_structured_reports WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Phase4Repository) CreateStructuredReport(ctx context.Context, item model.Phase4StructuredReportRecord) error {
	machineRaw, _ := json.Marshal(item.MachineReadableReport)
	citationRaw, _ := json.Marshal(item.CitationRefs)
	sourceRaw, _ := json.Marshal(item.ReferenceSourceIDs)
	_, err := r.db.ExecContext(ctx, `INSERT INTO phase4_structured_reports (id,run_manifest_id,dataset_profile_id,idea_id,reader_context_id,title,machine_readable_report_json,human_readable_report_md,citation_refs_json,reference_source_ids_json,status,created_at,updated_at)
	VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),NULLIF($5,''),$6,$7,$8,$9,$10,$11,$12,$13)`,
		item.ID, item.RunManifestID, item.DatasetProfileID, item.IdeaID, item.ReaderContextID, item.Title, machineRaw, item.HumanReadableReportMD, citationRaw, sourceRaw, item.Status, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) UpdateStructuredReport(ctx context.Context, item model.Phase4StructuredReportRecord) error {
	machineRaw, _ := json.Marshal(item.MachineReadableReport)
	citationRaw, _ := json.Marshal(item.CitationRefs)
	sourceRaw, _ := json.Marshal(item.ReferenceSourceIDs)
	_, err := r.db.ExecContext(ctx, `UPDATE phase4_structured_reports SET title=$2,machine_readable_report_json=$3,human_readable_report_md=$4,citation_refs_json=$5,reference_source_ids_json=$6,status=$7,updated_at=$8 WHERE id=$1`,
		item.ID, item.Title, machineRaw, item.HumanReadableReportMD, citationRaw, sourceRaw, item.Status, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) ListWorkflows(ctx context.Context, datasetProfileID string, status string) ([]model.Phase4Workflow, error) {
	query := `SELECT id,dataset_profile_id,COALESCE(reader_context_id,''),COALESCE(selected_idea_id,''),COALESCE(current_run_manifest_id,''),COALESCE(latest_report_id,''),COALESCE(latest_reader_job_id,''),COALESCE(latest_idea_job_id,''),COALESCE(latest_coding_job_id,''),COALESCE(latest_writer_job_id,''),status,next_action,last_error,manual_inputs_json,metadata_json,created_at,updated_at
	FROM phase4_workflows WHERE 1=1`
	args := make([]any, 0, 2)
	index := 1
	if strings.TrimSpace(datasetProfileID) != "" {
		query += fmt.Sprintf(" AND dataset_profile_id=$%d", index)
		args = append(args, strings.TrimSpace(datasetProfileID))
		index++
	}
	if strings.TrimSpace(status) != "" {
		query += fmt.Sprintf(" AND status=$%d", index)
		args = append(args, strings.TrimSpace(strings.ToLower(status)))
		index++
	}
	query += ` ORDER BY updated_at DESC, created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Phase4Workflow, 0)
	for rows.Next() {
		item, scanErr := scanPhase4Workflow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Phase4Repository) GetWorkflowByID(ctx context.Context, id string) (*model.Phase4Workflow, error) {
	item, err := scanPhase4Workflow(r.db.QueryRowContext(ctx, `SELECT id,dataset_profile_id,COALESCE(reader_context_id,''),COALESCE(selected_idea_id,''),COALESCE(current_run_manifest_id,''),COALESCE(latest_report_id,''),COALESCE(latest_reader_job_id,''),COALESCE(latest_idea_job_id,''),COALESCE(latest_coding_job_id,''),COALESCE(latest_writer_job_id,''),status,next_action,last_error,manual_inputs_json,metadata_json,created_at,updated_at
	FROM phase4_workflows WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Phase4Repository) CreateWorkflow(ctx context.Context, item model.Phase4Workflow) error {
	manualInputsRaw, _ := json.Marshal(item.ManualInputs)
	metadataRaw, _ := json.Marshal(item.Metadata)
	_, err := r.db.ExecContext(ctx, `INSERT INTO phase4_workflows (id,dataset_profile_id,reader_context_id,selected_idea_id,current_run_manifest_id,latest_report_id,latest_reader_job_id,latest_idea_job_id,latest_coding_job_id,latest_writer_job_id,status,next_action,last_error,manual_inputs_json,metadata_json,created_at,updated_at)
	VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),NULLIF($9,''),NULLIF($10,''),$11,$12,$13,$14,$15,$16,$17)`,
		item.ID, item.DatasetProfileID, item.ReaderContextID, item.SelectedIdeaID, item.CurrentRunManifestID, item.LatestReportID, item.LatestReaderJobID, item.LatestIdeaJobID, item.LatestCodingJobID, item.LatestWriterJobID, item.Status, item.NextAction, item.LastError, manualInputsRaw, metadataRaw, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) UpdateWorkflow(ctx context.Context, item model.Phase4Workflow) error {
	manualInputsRaw, _ := json.Marshal(item.ManualInputs)
	metadataRaw, _ := json.Marshal(item.Metadata)
	_, err := r.db.ExecContext(ctx, `UPDATE phase4_workflows SET dataset_profile_id=$2,reader_context_id=NULLIF($3,''),selected_idea_id=NULLIF($4,''),current_run_manifest_id=NULLIF($5,''),latest_report_id=NULLIF($6,''),latest_reader_job_id=NULLIF($7,''),latest_idea_job_id=NULLIF($8,''),latest_coding_job_id=NULLIF($9,''),latest_writer_job_id=NULLIF($10,''),status=$11,next_action=$12,last_error=$13,manual_inputs_json=$14,metadata_json=$15,updated_at=$16 WHERE id=$1`,
		item.ID, item.DatasetProfileID, item.ReaderContextID, item.SelectedIdeaID, item.CurrentRunManifestID, item.LatestReportID, item.LatestReaderJobID, item.LatestIdeaJobID, item.LatestCodingJobID, item.LatestWriterJobID, item.Status, item.NextAction, item.LastError, manualInputsRaw, metadataRaw, item.UpdatedAt,
	)
	return err
}

func (r *Phase4Repository) ListWorkflowActions(ctx context.Context, workflowID string) ([]model.Phase4WorkflowAction, error) {
	query := `SELECT id,workflow_id,stage,action_type,actor_type,status,COALESCE(job_id,''),COALESCE(run_manifest_id,''),COALESCE(report_id,''),payload_json,error_message,created_at,updated_at
	FROM phase4_workflow_actions`
	args := make([]any, 0, 1)
	if strings.TrimSpace(workflowID) != "" {
		query += ` WHERE workflow_id=$1`
		args = append(args, strings.TrimSpace(workflowID))
	}
	query += ` ORDER BY created_at ASC, updated_at ASC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.Phase4WorkflowAction, 0)
	for rows.Next() {
		item, scanErr := scanPhase4WorkflowAction(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *Phase4Repository) CreateWorkflowAction(ctx context.Context, item model.Phase4WorkflowAction) error {
	payloadRaw, _ := json.Marshal(item.Payload)
	_, err := r.db.ExecContext(ctx, `INSERT INTO phase4_workflow_actions (id,workflow_id,stage,action_type,actor_type,status,job_id,run_manifest_id,report_id,payload_json,error_message,created_at,updated_at)
	VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''),NULLIF($8,''),NULLIF($9,''),$10,$11,$12,$13)`,
		item.ID, item.WorkflowID, item.Stage, item.ActionType, item.ActorType, item.Status, item.JobID, item.RunManifestID, item.ReportID, payloadRaw, item.ErrorMessage, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func scanPhase4DatasetProfile(scanner researchAssetScanner) (model.Phase4DatasetProfile, error) {
	var item model.Phase4DatasetProfile
	var modalityRaw []byte
	var splitsRaw []byte
	var labelSchemaRaw []byte
	var fileSnapshotRaw []byte
	var sampleStatsRaw []byte
	var difficultiesRaw []byte
	var metadataRaw []byte
	if err := scanner.Scan(&item.ID, &item.DatasetName, &item.TaskType, &modalityRaw, &splitsRaw, &labelSchemaRaw, &fileSnapshotRaw, &sampleStatsRaw, &item.OfficialMetric, &item.OfficialBaseline, &item.License, &item.Citation, &difficultiesRaw, &item.UserNotes, &metadataRaw, &item.SourceMode, &item.ServerID, &item.ServerName, &item.ServerPath, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	item.ModalityComposition = []string{}
	item.Splits = []model.Phase4DatasetSplit{}
	item.LabelSchema = map[string]any{}
	item.FileStructureSnapshot = map[string]any{}
	item.SampleStatistics = map[string]any{}
	item.KnownDifficulties = []string{}
	item.Metadata = map[string]any{}
	decodeJSON(modalityRaw, &item.ModalityComposition)
	decodeJSON(splitsRaw, &item.Splits)
	decodeJSON(labelSchemaRaw, &item.LabelSchema)
	decodeJSON(fileSnapshotRaw, &item.FileStructureSnapshot)
	decodeJSON(sampleStatsRaw, &item.SampleStatistics)
	decodeJSON(difficultiesRaw, &item.KnownDifficulties)
	decodeJSON(metadataRaw, &item.Metadata)
	return item, nil
}

func scanPhase4ReaderSource(scanner researchAssetScanner) (model.Phase4ReaderSource, error) {
	var item model.Phase4ReaderSource
	var authorsRaw []byte
	var metadataRaw []byte
	if err := scanner.Scan(&item.ID, &item.DatasetProfileID, &item.Title, &authorsRaw, &item.Venue, &item.PublicationYear, &item.SourceType, &item.SourceURL, &item.OpenAccessURL, &item.QualityTier, &item.RankingScore, &item.QualityScore, &item.RelevanceScore, &item.CitationCount, &metadataRaw, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	item.Authors = []string{}
	item.Metadata = map[string]any{}
	decodeJSON(authorsRaw, &item.Authors)
	decodeJSON(metadataRaw, &item.Metadata)
	return item, nil
}

func scanPhase4ReaderContext(scanner researchAssetScanner) (model.Phase4ReaderContext, error) {
	var item model.Phase4ReaderContext
	var relatedWorkRaw []byte
	var retrievalFocusRaw []byte
	var sourceIDsRaw []byte
	var structuredRaw []byte
	if err := scanner.Scan(&item.ID, &item.DatasetProfileID, &item.Title, &item.Summary, &item.TaskDefinition, &relatedWorkRaw, &retrievalFocusRaw, &item.RankingNotes, &sourceIDsRaw, &structuredRaw, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	item.RelatedWork = []string{}
	item.RetrievalFocus = []string{}
	item.SourceIDs = []string{}
	item.StructuredContext = map[string]any{}
	decodeJSON(relatedWorkRaw, &item.RelatedWork)
	decodeJSON(retrievalFocusRaw, &item.RetrievalFocus)
	decodeJSON(sourceIDsRaw, &item.SourceIDs)
	decodeJSON(structuredRaw, &item.StructuredContext)
	return item, nil
}

func scanPhase4Idea(scanner researchAssetScanner) (model.Phase4Idea, error) {
	var item model.Phase4Idea
	var dataProcessingRaw []byte
	var modelChangesRaw []byte
	var evalMetricsRaw []byte
	var risksRaw []byte
	var expectedGainsRaw []byte
	var scoreRaw []byte
	var scoreSummaryRaw []byte
	var failureRaw []byte
	if err := scanner.Scan(&item.ID, &item.DatasetProfileID, &item.ReaderContextID, &item.Title, &item.ProblemDefinition, &item.CoreMethod, &item.Differentiators, &dataProcessingRaw, &modelChangesRaw, &item.TrainingPlan, &evalMetricsRaw, &risksRaw, &expectedGainsRaw, &scoreRaw, &scoreSummaryRaw, &item.Status, &item.SourceType, &item.RevisionOfID, &item.LineageRootID, &failureRaw, &item.LastFailureRunID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	item.DataProcessingNeeds = []string{}
	item.ModelChanges = []string{}
	item.EvaluationMetrics = []string{}
	item.RiskPoints = []string{}
	item.ExpectedGains = []string{}
	item.ScoreSummary = map[string]any{}
	item.FailureFeedback = map[string]any{}
	decodeJSON(dataProcessingRaw, &item.DataProcessingNeeds)
	decodeJSON(modelChangesRaw, &item.ModelChanges)
	decodeJSON(evalMetricsRaw, &item.EvaluationMetrics)
	decodeJSON(risksRaw, &item.RiskPoints)
	decodeJSON(expectedGainsRaw, &item.ExpectedGains)
	decodeJSON(scoreRaw, &item.Score)
	decodeJSON(scoreSummaryRaw, &item.ScoreSummary)
	decodeJSON(failureRaw, &item.FailureFeedback)
	return item, nil
}

func scanPhase4RunManifest(scanner researchAssetScanner) (model.Phase4RunManifest, error) {
	var item model.Phase4RunManifest
	var artifactRaw []byte
	var failureRaw []byte
	if err := scanner.Scan(&item.ID, &item.DatasetProfileID, &item.IdeaID, &item.ReaderContextID, &item.CodeSnapshotID, &item.RunnerMode, &item.ServerID, &item.GPU, &item.Status, &item.RetryCount, &item.MaxRetryCount, &artifactRaw, &item.LogsPath, &item.MetricsPath, &failureRaw, &item.CreatedAt, &item.UpdatedAt, &item.StartedAt, &item.FinishedAt); err != nil {
		return item, err
	}
	item.ArtifactPaths = map[string]any{}
	item.FailureFeedback = map[string]any{}
	decodeJSON(artifactRaw, &item.ArtifactPaths)
	decodeJSON(failureRaw, &item.FailureFeedback)
	return item, nil
}

func scanPhase4StructuredReport(scanner researchAssetScanner) (model.Phase4StructuredReportRecord, error) {
	var item model.Phase4StructuredReportRecord
	var machineRaw []byte
	var citationRaw []byte
	var sourceRaw []byte
	if err := scanner.Scan(&item.ID, &item.RunManifestID, &item.DatasetProfileID, &item.IdeaID, &item.ReaderContextID, &item.Title, &machineRaw, &item.HumanReadableReportMD, &citationRaw, &sourceRaw, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	item.MachineReadableReport = map[string]any{}
	item.CitationRefs = []string{}
	item.ReferenceSourceIDs = []string{}
	decodeJSON(machineRaw, &item.MachineReadableReport)
	decodeJSON(citationRaw, &item.CitationRefs)
	decodeJSON(sourceRaw, &item.ReferenceSourceIDs)
	return item, nil
}

func scanPhase4Workflow(scanner researchAssetScanner) (model.Phase4Workflow, error) {
	var item model.Phase4Workflow
	var manualInputsRaw []byte
	var metadataRaw []byte
	if err := scanner.Scan(&item.ID, &item.DatasetProfileID, &item.ReaderContextID, &item.SelectedIdeaID, &item.CurrentRunManifestID, &item.LatestReportID, &item.LatestReaderJobID, &item.LatestIdeaJobID, &item.LatestCodingJobID, &item.LatestWriterJobID, &item.Status, &item.NextAction, &item.LastError, &manualInputsRaw, &metadataRaw, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	item.ManualInputs = map[string]any{}
	item.Metadata = map[string]any{}
	decodeJSON(manualInputsRaw, &item.ManualInputs)
	decodeJSON(metadataRaw, &item.Metadata)
	return item, nil
}

func scanPhase4WorkflowAction(scanner researchAssetScanner) (model.Phase4WorkflowAction, error) {
	var item model.Phase4WorkflowAction
	var payloadRaw []byte
	if err := scanner.Scan(&item.ID, &item.WorkflowID, &item.Stage, &item.ActionType, &item.ActorType, &item.Status, &item.JobID, &item.RunManifestID, &item.ReportID, &payloadRaw, &item.ErrorMessage, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	item.Payload = map[string]any{}
	decodeJSON(payloadRaw, &item.Payload)
	return item, nil
}
