package repository

import (
	"context"
	"database/sql"

	"mrag-platform/backend/go/internal/model"
)

type IdeaRepository struct {
	db *sql.DB
}

func NewIdeaRepository(db *sql.DB) *IdeaRepository {
	return &IdeaRepository{db: db}
}

func (r *IdeaRepository) List(ctx context.Context) ([]model.Idea, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,title,description_md,status,weight,source_type,priority,confidence,created_at,updated_at FROM ideas ORDER BY priority DESC, weight DESC, updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Idea, 0)
	for rows.Next() {
		item, scanErr := scanIdea(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *IdeaRepository) GetByID(ctx context.Context, id string) (*model.Idea, error) {
	item, err := scanIdea(r.db.QueryRowContext(ctx, `SELECT id,title,description_md,status,weight,source_type,priority,confidence,created_at,updated_at FROM ideas WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *IdeaRepository) Create(ctx context.Context, idea model.Idea) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO ideas (id,title,description_md,status,weight,source_type,priority,confidence,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		idea.ID, idea.Title, idea.DescriptionMD, idea.Status, idea.Weight, idea.SourceType, idea.Priority, idea.Confidence, idea.CreatedAt, idea.UpdatedAt,
	)
	return err
}

func (r *IdeaRepository) Update(ctx context.Context, idea model.Idea) error {
	_, err := r.db.ExecContext(ctx, `UPDATE ideas SET title=$2,description_md=$3,status=$4,weight=$5,source_type=$6,priority=$7,confidence=$8,updated_at=$9 WHERE id=$1`,
		idea.ID, idea.Title, idea.DescriptionMD, idea.Status, idea.Weight, idea.SourceType, idea.Priority, idea.Confidence, idea.UpdatedAt,
	)
	return err
}

func (r *IdeaRepository) AddSource(ctx context.Context, source model.IdeaSource) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO idea_sources (idea_id,paper_id,paper_insight_id,source_note,created_at,updated_at) VALUES ($1,NULLIF($2,''),NULLIF($3,''),$4,$5,$6)`,
		source.IdeaID, source.PaperID, source.PaperInsightID, source.SourceNote, source.CreatedAt, source.UpdatedAt,
	)
	return err
}

func (r *IdeaRepository) ListSources(ctx context.Context, ideaID string) ([]model.IdeaSource, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT s.id,s.idea_id,COALESCE(s.paper_id,''),COALESCE(s.paper_insight_id,''),s.source_note,COALESCE(p.title,''),s.created_at,s.updated_at FROM idea_sources s LEFT JOIN papers p ON p.id = s.paper_id WHERE s.idea_id=$1 ORDER BY s.id ASC`, ideaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.IdeaSource, 0)
	for rows.Next() {
		var item model.IdeaSource
		if err = rows.Scan(&item.ID, &item.IdeaID, &item.PaperID, &item.PaperInsightID, &item.SourceNote, &item.PaperTitle, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func scanIdea(scanner researchAssetScanner) (model.Idea, error) {
	var item model.Idea
	err := scanner.Scan(&item.ID, &item.Title, &item.DescriptionMD, &item.Status, &item.Weight, &item.SourceType, &item.Priority, &item.Confidence, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}
