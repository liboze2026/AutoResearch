package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"mrag-platform/backend/go/internal/model"
)

type AgentEventRepository struct {
	db *sql.DB
}

func NewAgentEventRepository(db *sql.DB) *AgentEventRepository {
	return &AgentEventRepository{db: db}
}

func (r *AgentEventRepository) Create(ctx context.Context, item model.AgentEvent) error {
	inputRefsRaw, _ := json.Marshal(item.InputRefs)
	payloadRaw, _ := json.Marshal(item.Payload)
	triggeredJobIDsRaw, _ := json.Marshal(item.TriggeredJobIDs)
	_, err := r.db.ExecContext(ctx, `INSERT INTO agent_events (id,event_type,source_ref,input_refs,payload,status,triggered_job_ids,error_message,created_at,processed_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		item.ID, item.EventType, item.SourceRef, inputRefsRaw, payloadRaw, item.Status, triggeredJobIDsRaw, item.ErrorMessage, item.CreatedAt, item.ProcessedAt, item.UpdatedAt,
	)
	return err
}

func (r *AgentEventRepository) Update(ctx context.Context, item model.AgentEvent) error {
	inputRefsRaw, _ := json.Marshal(item.InputRefs)
	payloadRaw, _ := json.Marshal(item.Payload)
	triggeredJobIDsRaw, _ := json.Marshal(item.TriggeredJobIDs)
	_, err := r.db.ExecContext(ctx, `UPDATE agent_events SET event_type=$2,source_ref=$3,input_refs=$4,payload=$5,status=$6,triggered_job_ids=$7,error_message=$8,processed_at=$9,updated_at=$10 WHERE id=$1`,
		item.ID, item.EventType, item.SourceRef, inputRefsRaw, payloadRaw, item.Status, triggeredJobIDsRaw, item.ErrorMessage, item.ProcessedAt, item.UpdatedAt,
	)
	return err
}

func (r *AgentEventRepository) GetByID(ctx context.Context, id string) (*model.AgentEvent, error) {
	item, err := scanAgentEvent(r.db.QueryRowContext(ctx, `SELECT id,event_type,source_ref,input_refs,payload,status,triggered_job_ids,error_message,created_at,processed_at,updated_at FROM agent_events WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AgentEventRepository) List(ctx context.Context, limit int) ([]model.AgentEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,event_type,source_ref,input_refs,payload,status,triggered_job_ids,error_message,created_at,processed_at,updated_at FROM agent_events ORDER BY created_at DESC, id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.AgentEvent, 0)
	for rows.Next() {
		item, scanErr := scanAgentEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

type AgentSubscriptionRepository struct {
	db *sql.DB
}

func NewAgentSubscriptionRepository(db *sql.DB) *AgentSubscriptionRepository {
	return &AgentSubscriptionRepository{db: db}
}

func (r *AgentSubscriptionRepository) Create(ctx context.Context, item model.AgentSubscription) error {
	skillRefsRaw, _ := json.Marshal(item.SkillRefs)
	toolRefsRaw, _ := json.Marshal(item.ToolRefs)
	memoryRefsRaw, _ := json.Marshal(item.MemoryRefs)
	triggerRuleRaw, _ := json.Marshal(item.TriggerRule)
	_, err := r.db.ExecContext(ctx, `INSERT INTO agent_subscriptions (id,name,event_type,agent_type,enabled,execution_mode,model_provider,model_name,prompt_version,output_schema_ref,skill_refs,tool_refs,memory_refs,trigger_rule,max_retries,concurrency_limit,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		item.ID, item.Name, item.EventType, item.AgentType, item.Enabled, item.ExecutionMode, item.ModelProvider, item.ModelName, item.PromptVersion, item.OutputSchemaRef, skillRefsRaw, toolRefsRaw, memoryRefsRaw, triggerRuleRaw, item.MaxRetries, item.ConcurrencyLimit, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *AgentSubscriptionRepository) List(ctx context.Context) ([]model.AgentSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,event_type,agent_type,enabled,execution_mode,model_provider,model_name,prompt_version,output_schema_ref,skill_refs,tool_refs,memory_refs,trigger_rule,max_retries,concurrency_limit,created_at,updated_at FROM agent_subscriptions ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.AgentSubscription, 0)
	for rows.Next() {
		item, scanErr := scanAgentSubscription(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *AgentSubscriptionRepository) ListByEventType(ctx context.Context, eventType string) ([]model.AgentSubscription, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,event_type,agent_type,enabled,execution_mode,model_provider,model_name,prompt_version,output_schema_ref,skill_refs,tool_refs,memory_refs,trigger_rule,max_retries,concurrency_limit,created_at,updated_at FROM agent_subscriptions WHERE enabled=TRUE AND event_type=$1 ORDER BY created_at ASC, id ASC`, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.AgentSubscription, 0)
	for rows.Next() {
		item, scanErr := scanAgentSubscription(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func scanAgentEvent(scanner researchAssetScanner) (model.AgentEvent, error) {
	var item model.AgentEvent
	var inputRefsRaw []byte
	var payloadRaw []byte
	var triggeredJobIDsRaw []byte
	err := scanner.Scan(&item.ID, &item.EventType, &item.SourceRef, &inputRefsRaw, &payloadRaw, &item.Status, &triggeredJobIDsRaw, &item.ErrorMessage, &item.CreatedAt, &item.ProcessedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.InputRefs = []model.AgentInputRef{}
	item.Payload = map[string]any{}
	item.TriggeredJobIDs = []string{}
	decodeJSON(inputRefsRaw, &item.InputRefs)
	decodeJSON(payloadRaw, &item.Payload)
	decodeJSON(triggeredJobIDsRaw, &item.TriggeredJobIDs)
	return item, nil
}

func scanAgentSubscription(scanner researchAssetScanner) (model.AgentSubscription, error) {
	var item model.AgentSubscription
	var skillRefsRaw []byte
	var toolRefsRaw []byte
	var memoryRefsRaw []byte
	var triggerRuleRaw []byte
	err := scanner.Scan(&item.ID, &item.Name, &item.EventType, &item.AgentType, &item.Enabled, &item.ExecutionMode, &item.ModelProvider, &item.ModelName, &item.PromptVersion, &item.OutputSchemaRef, &skillRefsRaw, &toolRefsRaw, &memoryRefsRaw, &triggerRuleRaw, &item.MaxRetries, &item.ConcurrencyLimit, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.SkillRefs = []string{}
	item.ToolRefs = []string{}
	item.MemoryRefs = []string{}
	item.TriggerRule = map[string]any{}
	decodeJSON(skillRefsRaw, &item.SkillRefs)
	decodeJSON(toolRefsRaw, &item.ToolRefs)
	decodeJSON(memoryRefsRaw, &item.MemoryRefs)
	decodeJSON(triggerRuleRaw, &item.TriggerRule)
	return item, nil
}
