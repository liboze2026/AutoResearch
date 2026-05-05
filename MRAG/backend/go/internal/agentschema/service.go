package agentschema

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/pkg/httpx"
)

type schemaStore interface {
	Create(context.Context, model.AgentSchema) error
	GetByID(context.Context, string) (*model.AgentSchema, error)
	List(context.Context) ([]model.AgentSchema, error)
}

type Service struct {
	store schemaStore
}

func NewService(store schemaStore) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, req model.AgentSchemaCreateRequest) (*model.AgentSchema, error) {
	if err := validateCreate(req); err != nil {
		return nil, err
	}
	now := time.Now()
	item := model.AgentSchema{
		ID:              httpx.NewID("asch"),
		SchemaName:      strings.TrimSpace(req.SchemaName),
		Version:         strings.TrimSpace(req.Version),
		AgentType:       strings.TrimSpace(req.AgentType),
		SchemaRef:       strings.TrimSpace(req.SchemaRef),
		JSONSchema:      ensureMap(req.JSONSchema),
		PythonSchemaRef: strings.TrimSpace(req.PythonSchemaRef),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.store.Create(ctx, item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*model.AgentSchema, error) {
	return s.store.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]model.AgentSchema, error) {
	return s.store.List(ctx)
}

func validateCreate(req model.AgentSchemaCreateRequest) error {
	if strings.TrimSpace(req.SchemaName) == "" {
		return fmt.Errorf("schema_name is required")
	}
	if strings.TrimSpace(req.Version) == "" {
		return fmt.Errorf("version is required")
	}
	if strings.TrimSpace(req.AgentType) == "" {
		return fmt.Errorf("agent_type is required")
	}
	if strings.TrimSpace(req.SchemaRef) == "" {
		return fmt.Errorf("schema_ref is required")
	}
	if len(req.JSONSchema) == 0 && strings.TrimSpace(req.PythonSchemaRef) == "" {
		return fmt.Errorf("json_schema or python_schema_ref is required")
	}
	return nil
}

func ensureMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	copyValue := make(map[string]any, len(value))
	for key, item := range value {
		copyValue[key] = item
	}
	return copyValue
}
