package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"mrag-platform/backend/go/internal/model"
)

type AgentJobRepository struct {
	db *sql.DB
}

func NewAgentJobRepository(db *sql.DB) *AgentJobRepository {
	return &AgentJobRepository{db: db}
}

func (r *AgentJobRepository) GetByID(ctx context.Context, id string) (*model.AgentJob, error) {
	item, err := scanAgentJob(r.db.QueryRowContext(ctx, `SELECT id,agent_type,execution_mode,model_provider,model_name,prompt_version,input_refs,output_schema_ref,skill_refs,tool_refs,memory_refs,workspace_dir,metadata,trigger_event_id,dedup_key,retry_count,max_retries,concurrency_limit,status,normalized_payload,artifact_manifest,repair_actions,tool_usages,warnings,validation_status,repair_status,validation_errors,error_message,started_at,completed_at,created_at,updated_at FROM agent_jobs WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AgentJobRepository) Create(ctx context.Context, item model.AgentJob) error {
	inputRefsRaw, _ := json.Marshal(item.InputRefs)
	skillRefsRaw, _ := json.Marshal(item.SkillRefs)
	toolRefsRaw, _ := json.Marshal(item.ToolRefs)
	memoryRefsRaw, _ := json.Marshal(item.MemoryRefs)
	metadataRaw, _ := json.Marshal(item.Metadata)
	normalizedPayloadRaw, _ := json.Marshal(item.NormalizedPayload)
	artifactManifestRaw, _ := json.Marshal(item.ArtifactManifest)
	repairActionsRaw, _ := json.Marshal(item.RepairActions)
	toolUsagesRaw, _ := json.Marshal(item.ToolUsages)
	warningsRaw, _ := json.Marshal(item.Warnings)
	validationErrorsRaw, _ := json.Marshal(item.ValidationErrors)
	_, err := r.db.ExecContext(ctx, `INSERT INTO agent_jobs (id,agent_type,execution_mode,model_provider,model_name,prompt_version,input_refs,output_schema_ref,skill_refs,tool_refs,memory_refs,workspace_dir,metadata,trigger_event_id,dedup_key,retry_count,max_retries,concurrency_limit,status,normalized_payload,artifact_manifest,repair_actions,tool_usages,warnings,validation_status,repair_status,validation_errors,error_message,started_at,completed_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32)`,
		item.ID, item.AgentType, item.ExecutionMode, item.ModelProvider, item.ModelName, item.PromptVersion, inputRefsRaw, item.OutputSchemaRef, skillRefsRaw, toolRefsRaw, memoryRefsRaw, item.WorkspaceDir, metadataRaw, item.TriggerEventID, item.DedupKey, item.RetryCount, item.MaxRetries, item.ConcurrencyLimit, item.Status, normalizedPayloadRaw, artifactManifestRaw, repairActionsRaw, toolUsagesRaw, warningsRaw, item.ValidationStatus, item.RepairStatus, validationErrorsRaw, item.ErrorMessage, item.StartedAt, item.CompletedAt, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *AgentJobRepository) Update(ctx context.Context, item model.AgentJob) error {
	inputRefsRaw, _ := json.Marshal(item.InputRefs)
	skillRefsRaw, _ := json.Marshal(item.SkillRefs)
	toolRefsRaw, _ := json.Marshal(item.ToolRefs)
	memoryRefsRaw, _ := json.Marshal(item.MemoryRefs)
	metadataRaw, _ := json.Marshal(item.Metadata)
	normalizedPayloadRaw, _ := json.Marshal(item.NormalizedPayload)
	artifactManifestRaw, _ := json.Marshal(item.ArtifactManifest)
	repairActionsRaw, _ := json.Marshal(item.RepairActions)
	toolUsagesRaw, _ := json.Marshal(item.ToolUsages)
	warningsRaw, _ := json.Marshal(item.Warnings)
	validationErrorsRaw, _ := json.Marshal(item.ValidationErrors)
	_, err := r.db.ExecContext(ctx, `UPDATE agent_jobs SET agent_type=$2,execution_mode=$3,model_provider=$4,model_name=$5,prompt_version=$6,input_refs=$7,output_schema_ref=$8,skill_refs=$9,tool_refs=$10,memory_refs=$11,workspace_dir=$12,metadata=$13,trigger_event_id=$14,dedup_key=$15,retry_count=$16,max_retries=$17,concurrency_limit=$18,status=$19,normalized_payload=$20,artifact_manifest=$21,repair_actions=$22,tool_usages=$23,warnings=$24,validation_status=$25,repair_status=$26,validation_errors=$27,error_message=$28,started_at=$29,completed_at=$30,updated_at=$31 WHERE id=$1`,
		item.ID, item.AgentType, item.ExecutionMode, item.ModelProvider, item.ModelName, item.PromptVersion, inputRefsRaw, item.OutputSchemaRef, skillRefsRaw, toolRefsRaw, memoryRefsRaw, item.WorkspaceDir, metadataRaw, item.TriggerEventID, item.DedupKey, item.RetryCount, item.MaxRetries, item.ConcurrencyLimit, item.Status, normalizedPayloadRaw, artifactManifestRaw, repairActionsRaw, toolUsagesRaw, warningsRaw, item.ValidationStatus, item.RepairStatus, validationErrorsRaw, item.ErrorMessage, item.StartedAt, item.CompletedAt, item.UpdatedAt,
	)
	return err
}

func (r *AgentJobRepository) FindByDedupKey(ctx context.Context, dedupKey string) (*model.AgentJob, error) {
	item, err := scanAgentJob(r.db.QueryRowContext(ctx, `SELECT id,agent_type,execution_mode,model_provider,model_name,prompt_version,input_refs,output_schema_ref,skill_refs,tool_refs,memory_refs,workspace_dir,metadata,trigger_event_id,dedup_key,retry_count,max_retries,concurrency_limit,status,normalized_payload,artifact_manifest,repair_actions,tool_usages,warnings,validation_status,repair_status,validation_errors,error_message,started_at,completed_at,created_at,updated_at FROM agent_jobs WHERE dedup_key=$1 ORDER BY created_at DESC LIMIT 1`, dedupKey))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AgentJobRepository) List(ctx context.Context, limit int) ([]model.AgentJob, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,agent_type,execution_mode,model_provider,model_name,prompt_version,input_refs,output_schema_ref,skill_refs,tool_refs,memory_refs,workspace_dir,metadata,trigger_event_id,dedup_key,retry_count,max_retries,concurrency_limit,status,normalized_payload,artifact_manifest,repair_actions,tool_usages,warnings,validation_status,repair_status,validation_errors,error_message,started_at,completed_at,created_at,updated_at FROM agent_jobs ORDER BY created_at DESC, id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.AgentJob, 0)
	for rows.Next() {
		item, scanErr := scanAgentJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *AgentJobRepository) LatestByAgentType(ctx context.Context, agentType string) (*model.AgentJob, error) {
	item, err := scanAgentJob(r.db.QueryRowContext(ctx, `SELECT id,agent_type,execution_mode,model_provider,model_name,prompt_version,input_refs,output_schema_ref,skill_refs,tool_refs,memory_refs,workspace_dir,metadata,trigger_event_id,dedup_key,retry_count,max_retries,concurrency_limit,status,normalized_payload,artifact_manifest,repair_actions,tool_usages,warnings,validation_status,repair_status,validation_errors,error_message,started_at,completed_at,created_at,updated_at FROM agent_jobs WHERE agent_type=$1 ORDER BY created_at DESC, id DESC LIMIT 1`, agentType))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AgentJobRepository) ListByStatus(ctx context.Context, status string, limit int) ([]model.AgentJob, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,agent_type,execution_mode,model_provider,model_name,prompt_version,input_refs,output_schema_ref,skill_refs,tool_refs,memory_refs,workspace_dir,metadata,trigger_event_id,dedup_key,retry_count,max_retries,concurrency_limit,status,normalized_payload,artifact_manifest,repair_actions,tool_usages,warnings,validation_status,repair_status,validation_errors,error_message,started_at,completed_at,created_at,updated_at FROM agent_jobs WHERE status=$1 ORDER BY created_at ASC, id ASC LIMIT $2`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.AgentJob, 0)
	for rows.Next() {
		item, scanErr := scanAgentJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *AgentJobRepository) CountActiveByAgentType(ctx context.Context, agentType string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM agent_jobs WHERE agent_type=$1 AND status IN ('running','validating','repairing')`, agentType).Scan(&count)
	return count, err
}

func (r *AgentJobRepository) CountByAgentType(ctx context.Context, agentType string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM agent_jobs WHERE agent_type=$1`, agentType).Scan(&count)
	return count, err
}

func scanAgentJob(scanner researchAssetScanner) (model.AgentJob, error) {
	var item model.AgentJob
	var inputRefsRaw []byte
	var skillRefsRaw []byte
	var toolRefsRaw []byte
	var memoryRefsRaw []byte
	var metadataRaw []byte
	var normalizedPayloadRaw []byte
	var artifactManifestRaw []byte
	var repairActionsRaw []byte
	var toolUsagesRaw []byte
	var warningsRaw []byte
	var validationErrorsRaw []byte
	err := scanner.Scan(&item.ID, &item.AgentType, &item.ExecutionMode, &item.ModelProvider, &item.ModelName, &item.PromptVersion, &inputRefsRaw, &item.OutputSchemaRef, &skillRefsRaw, &toolRefsRaw, &memoryRefsRaw, &item.WorkspaceDir, &metadataRaw, &item.TriggerEventID, &item.DedupKey, &item.RetryCount, &item.MaxRetries, &item.ConcurrencyLimit, &item.Status, &normalizedPayloadRaw, &artifactManifestRaw, &repairActionsRaw, &toolUsagesRaw, &warningsRaw, &item.ValidationStatus, &item.RepairStatus, &validationErrorsRaw, &item.ErrorMessage, &item.StartedAt, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.InputRefs = []model.AgentInputRef{}
	item.SkillRefs = []string{}
	item.ToolRefs = []string{}
	item.MemoryRefs = []string{}
	item.Metadata = map[string]any{}
	item.NormalizedPayload = map[string]any{}
	item.ArtifactManifest = []model.AgentArtifactManifestItem{}
	item.RepairActions = []model.AgentRepairAction{}
	item.ToolUsages = []model.AgentToolUsage{}
	item.Warnings = []string{}
	item.ValidationErrors = []string{}
	decodeJSON(inputRefsRaw, &item.InputRefs)
	decodeJSON(skillRefsRaw, &item.SkillRefs)
	decodeJSON(toolRefsRaw, &item.ToolRefs)
	decodeJSON(memoryRefsRaw, &item.MemoryRefs)
	decodeJSON(metadataRaw, &item.Metadata)
	decodeJSON(normalizedPayloadRaw, &item.NormalizedPayload)
	decodeJSON(artifactManifestRaw, &item.ArtifactManifest)
	decodeJSON(repairActionsRaw, &item.RepairActions)
	decodeJSON(toolUsagesRaw, &item.ToolUsages)
	decodeJSON(warningsRaw, &item.Warnings)
	decodeJSON(validationErrorsRaw, &item.ValidationErrors)
	return item, nil
}

type AgentSchemaRepository struct {
	db *sql.DB
}

func NewAgentSchemaRepository(db *sql.DB) *AgentSchemaRepository {
	return &AgentSchemaRepository{db: db}
}

func (r *AgentSchemaRepository) Create(ctx context.Context, item model.AgentSchema) error {
	jsonSchemaRaw, _ := json.Marshal(item.JSONSchema)
	_, err := r.db.ExecContext(ctx, `INSERT INTO agent_schemas (id,schema_name,version,agent_type,schema_ref,json_schema,python_schema_ref,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		item.ID, item.SchemaName, item.Version, item.AgentType, item.SchemaRef, jsonSchemaRaw, item.PythonSchemaRef, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *AgentSchemaRepository) GetByID(ctx context.Context, id string) (*model.AgentSchema, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,schema_name,version,agent_type,schema_ref,json_schema,python_schema_ref,created_at,updated_at FROM agent_schemas WHERE id=$1`, id)
	item, err := scanAgentSchema(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AgentSchemaRepository) List(ctx context.Context) ([]model.AgentSchema, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,schema_name,version,agent_type,schema_ref,json_schema,python_schema_ref,created_at,updated_at FROM agent_schemas ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.AgentSchema, 0)
	for rows.Next() {
		item, scanErr := scanAgentSchema(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func scanAgentSchema(scanner researchAssetScanner) (model.AgentSchema, error) {
	var item model.AgentSchema
	var jsonSchemaRaw []byte
	err := scanner.Scan(&item.ID, &item.SchemaName, &item.Version, &item.AgentType, &item.SchemaRef, &jsonSchemaRaw, &item.PythonSchemaRef, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.JSONSchema = map[string]any{}
	decodeJSON(jsonSchemaRaw, &item.JSONSchema)
	return item, nil
}

type AgentArtifactRepository struct {
	db *sql.DB
}

func NewAgentArtifactRepository(db *sql.DB) *AgentArtifactRepository {
	return &AgentArtifactRepository{db: db}
}

func (r *AgentArtifactRepository) Create(ctx context.Context, item model.AgentArtifact) error {
	metadataRaw, _ := json.Marshal(item.MetadataJSON)
	_, err := r.db.ExecContext(ctx, `INSERT INTO agent_artifacts (id,job_id,artifact_type,name,file_path,checksum,metadata_json,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		item.ID, item.JobID, item.ArtifactType, item.Name, item.FilePath, item.Checksum, metadataRaw, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *AgentArtifactRepository) ListByJobID(ctx context.Context, jobID string) ([]model.AgentArtifact, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,job_id,artifact_type,name,file_path,checksum,metadata_json,created_at,updated_at FROM agent_artifacts WHERE job_id=$1 ORDER BY created_at ASC, id ASC`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.AgentArtifact, 0)
	for rows.Next() {
		var item model.AgentArtifact
		var metadataRaw []byte
		if err = rows.Scan(&item.ID, &item.JobID, &item.ArtifactType, &item.Name, &item.FilePath, &item.Checksum, &metadataRaw, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.MetadataJSON = map[string]any{}
		decodeJSON(metadataRaw, &item.MetadataJSON)
		items = append(items, item)
	}
	return items, nil
}

type AgentJobTriggerRepository struct {
	db *sql.DB
}

func NewAgentJobTriggerRepository(db *sql.DB) *AgentJobTriggerRepository {
	return &AgentJobTriggerRepository{db: db}
}

func (r *AgentJobTriggerRepository) Create(ctx context.Context, item model.AgentJobTrigger) error {
	metadataRaw, _ := json.Marshal(item.Metadata)
	_, err := r.db.ExecContext(ctx, `INSERT INTO agent_job_triggers (id,job_id,trigger_type,status,metadata,error_message,requested_at,started_at,completed_at,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		item.ID, item.JobID, item.TriggerType, item.Status, metadataRaw, item.ErrorMessage, item.RequestedAt, item.StartedAt, item.CompletedAt, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *AgentJobTriggerRepository) Update(ctx context.Context, item model.AgentJobTrigger) error {
	metadataRaw, _ := json.Marshal(item.Metadata)
	_, err := r.db.ExecContext(ctx, `UPDATE agent_job_triggers SET trigger_type=$2,status=$3,metadata=$4,error_message=$5,requested_at=$6,started_at=$7,completed_at=$8,updated_at=$9 WHERE id=$1`,
		item.ID, item.TriggerType, item.Status, metadataRaw, item.ErrorMessage, item.RequestedAt, item.StartedAt, item.CompletedAt, item.UpdatedAt,
	)
	return err
}
