package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"mrag-platform/backend/go/internal/model"
)

type ToolRegistryRepository struct {
	db *sql.DB
}

func NewToolRegistryRepository(db *sql.DB) *ToolRegistryRepository {
	return &ToolRegistryRepository{db: db}
}

func (r *ToolRegistryRepository) Create(ctx context.Context, item model.ToolDefinition) error {
	inputSchemaRaw, _ := json.Marshal(item.InputSchema)
	outputSchemaRaw, _ := json.Marshal(item.OutputSchema)
	_, err := r.db.ExecContext(ctx, `INSERT INTO tool_registry (tool_id,name,owner_agent_type,path,description,usage_md,input_schema,output_schema,test_status,version,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		item.ToolID, item.Name, item.OwnerAgentType, item.Path, item.Description, item.UsageMD, inputSchemaRaw, outputSchemaRaw, item.TestStatus, item.Version, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *ToolRegistryRepository) List(ctx context.Context) ([]model.ToolDefinition, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tool_id,name,owner_agent_type,path,description,usage_md,input_schema,output_schema,test_status,version,created_at,updated_at FROM tool_registry ORDER BY created_at ASC, tool_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.ToolDefinition, 0)
	for rows.Next() {
		item, scanErr := scanToolDefinition(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

type SkillRegistryRepository struct {
	db *sql.DB
}

func NewSkillRegistryRepository(db *sql.DB) *SkillRegistryRepository {
	return &SkillRegistryRepository{db: db}
}

func (r *SkillRegistryRepository) Create(ctx context.Context, item model.SkillDefinition) error {
	dependenciesRaw, _ := json.Marshal(item.Dependencies)
	_, err := r.db.ExecContext(ctx, `INSERT INTO skill_registry (skill_id,name,description,skill_dir,entrypoint,dependencies,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		item.SkillID, item.Name, item.Description, item.SkillDir, item.Entrypoint, dependenciesRaw, item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *SkillRegistryRepository) List(ctx context.Context) ([]model.SkillDefinition, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT skill_id,name,description,skill_dir,entrypoint,dependencies,created_at,updated_at FROM skill_registry ORDER BY created_at ASC, skill_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.SkillDefinition, 0)
	for rows.Next() {
		item, scanErr := scanSkillDefinition(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

type AgentMemoryRepository struct {
	db *sql.DB
}

func NewAgentMemoryRepository(db *sql.DB) *AgentMemoryRepository {
	return &AgentMemoryRepository{db: db}
}

func (r *AgentMemoryRepository) Upsert(ctx context.Context, item model.AgentMemoryRecord) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO agent_memory (id,agent_type,memory_key,content_md,source_ref,updated_at,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (agent_type, memory_key) DO UPDATE SET content_md=EXCLUDED.content_md, source_ref=EXCLUDED.source_ref, updated_at=EXCLUDED.updated_at`,
		item.ID, item.AgentType, item.MemoryKey, item.ContentMD, item.SourceRef, item.UpdatedAt, item.CreatedAt,
	)
	return err
}

func (r *AgentMemoryRepository) ListByAgentType(ctx context.Context, agentType string) ([]model.AgentMemoryRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,agent_type,memory_key,content_md,source_ref,updated_at,created_at FROM agent_memory WHERE agent_type=$1 ORDER BY updated_at DESC, memory_key ASC`, agentType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.AgentMemoryRecord, 0)
	for rows.Next() {
		var item model.AgentMemoryRecord
		if err = rows.Scan(&item.ID, &item.AgentType, &item.MemoryKey, &item.ContentMD, &item.SourceRef, &item.UpdatedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func scanToolDefinition(scanner researchAssetScanner) (model.ToolDefinition, error) {
	var item model.ToolDefinition
	var inputSchemaRaw []byte
	var outputSchemaRaw []byte
	err := scanner.Scan(&item.ToolID, &item.Name, &item.OwnerAgentType, &item.Path, &item.Description, &item.UsageMD, &inputSchemaRaw, &outputSchemaRaw, &item.TestStatus, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.InputSchema = map[string]any{}
	item.OutputSchema = map[string]any{}
	decodeJSON(inputSchemaRaw, &item.InputSchema)
	decodeJSON(outputSchemaRaw, &item.OutputSchema)
	return item, nil
}

func scanSkillDefinition(scanner researchAssetScanner) (model.SkillDefinition, error) {
	var item model.SkillDefinition
	var dependenciesRaw []byte
	err := scanner.Scan(&item.SkillID, &item.Name, &item.Description, &item.SkillDir, &item.Entrypoint, &dependenciesRaw, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.Dependencies = []string{}
	decodeJSON(dependenciesRaw, &item.Dependencies)
	return item, nil
}
