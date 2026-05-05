package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"mrag-platform/backend/go/internal/model"
)

type PaperRepository struct {
	db *sql.DB
}

func NewPaperRepository(db *sql.DB) *PaperRepository {
	return &PaperRepository{db: db}
}

func (r *PaperRepository) List(ctx context.Context) ([]model.Paper, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,title,abstract,authors,venue,year,status,source_type,source_url,parse_mode,parse_error,parser_note,created_at,updated_at FROM papers ORDER BY updated_at DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Paper, 0)
	for rows.Next() {
		item, scanErr := scanPaper(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *PaperRepository) GetByID(ctx context.Context, id string) (*model.Paper, error) {
	item, err := scanPaper(r.db.QueryRowContext(ctx, `SELECT id,title,abstract,authors,venue,year,status,source_type,source_url,parse_mode,parse_error,parser_note,created_at,updated_at FROM papers WHERE id=$1`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PaperRepository) Create(ctx context.Context, paper model.Paper) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO papers (id,title,abstract,authors,venue,year,status,source_type,source_url,parse_mode,parse_error,parser_note,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		paper.ID, paper.Title, paper.Abstract, paper.Authors, paper.Venue, paper.Year, paper.Status, paper.SourceType, paper.SourceURL, paper.ParseMode, paper.ParseError, paper.ParserNote, paper.CreatedAt, paper.UpdatedAt,
	)
	return err
}

func (r *PaperRepository) UpdatePaperMetadata(ctx context.Context, paper model.Paper) error {
	_, err := r.db.ExecContext(ctx, `UPDATE papers SET title=$2,abstract=$3,authors=$4,venue=$5,year=$6,status=$7,source_url=$8,parse_mode=$9,parse_error=$10,parser_note=$11,updated_at=$12,source_type=$13 WHERE id=$1`,
		paper.ID, paper.Title, paper.Abstract, paper.Authors, paper.Venue, paper.Year, paper.Status, paper.SourceURL, paper.ParseMode, paper.ParseError, paper.ParserNote, paper.UpdatedAt, paper.SourceType,
	)
	return err
}

func (r *PaperRepository) AddFile(ctx context.Context, file model.PaperFile) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO paper_files (id,paper_id,file_path,file_type,checksum,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		file.ID, file.PaperID, file.FilePath, file.FileType, file.Checksum, file.CreatedAt, file.UpdatedAt,
	)
	return err
}

func (r *PaperRepository) ListFiles(ctx context.Context, paperID string) ([]model.PaperFile, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,paper_id,file_path,file_type,checksum,created_at,updated_at FROM paper_files WHERE paper_id=$1 ORDER BY created_at ASC`, paperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.PaperFile, 0)
	for rows.Next() {
		var item model.PaperFile
		if err = rows.Scan(&item.ID, &item.PaperID, &item.FilePath, &item.FileType, &item.Checksum, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *PaperRepository) UpsertInsight(ctx context.Context, insight model.PaperInsight) error {
	contribRaw, _ := json.Marshal(insight.ContributionsJSON)
	methodsRaw, _ := json.Marshal(insight.MethodsJSON)
	limitationsRaw, _ := json.Marshal(insight.LimitationsJSON)
	noveltyRaw, _ := json.Marshal(insight.NoveltyPointsJSON)
	_, err := r.db.ExecContext(ctx, `INSERT INTO paper_insights (id,paper_id,summary_md,contributions_json,methods_json,limitations_json,novelty_points_json,extract_status,extract_error,created_at,updated_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	ON CONFLICT (paper_id) DO UPDATE SET
		summary_md=EXCLUDED.summary_md,
		contributions_json=EXCLUDED.contributions_json,
		methods_json=EXCLUDED.methods_json,
		limitations_json=EXCLUDED.limitations_json,
		novelty_points_json=EXCLUDED.novelty_points_json,
		extract_status=EXCLUDED.extract_status,
		extract_error=EXCLUDED.extract_error,
		updated_at=EXCLUDED.updated_at`,
		insight.ID, insight.PaperID, insight.SummaryMD, contribRaw, methodsRaw, limitationsRaw, noveltyRaw, insight.ExtractStatus, insight.ExtractError, insight.CreatedAt, insight.UpdatedAt,
	)
	return err
}

func (r *PaperRepository) ListInsightsByPaper(ctx context.Context, paperID string) ([]model.PaperInsight, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,paper_id,summary_md,contributions_json,methods_json,limitations_json,novelty_points_json,extract_status,extract_error,created_at,updated_at FROM paper_insights WHERE paper_id=$1 ORDER BY created_at DESC`, paperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.PaperInsight, 0)
	for rows.Next() {
		item, scanErr := scanPaperInsight(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, nil
}

func scanPaper(scanner researchAssetScanner) (model.Paper, error) {
	var item model.Paper
	err := scanner.Scan(&item.ID, &item.Title, &item.Abstract, &item.Authors, &item.Venue, &item.Year, &item.Status, &item.SourceType, &item.SourceURL, &item.ParseMode, &item.ParseError, &item.ParserNote, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanPaperInsight(scanner researchAssetScanner) (model.PaperInsight, error) {
	var item model.PaperInsight
	var contribRaw []byte
	var methodsRaw []byte
	var limitationsRaw []byte
	var noveltyRaw []byte
	err := scanner.Scan(&item.ID, &item.PaperID, &item.SummaryMD, &contribRaw, &methodsRaw, &limitationsRaw, &noveltyRaw, &item.ExtractStatus, &item.ExtractError, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	decodeJSON(contribRaw, &item.ContributionsJSON)
	decodeJSON(methodsRaw, &item.MethodsJSON)
	decodeJSON(limitationsRaw, &item.LimitationsJSON)
	decodeJSON(noveltyRaw, &item.NoveltyPointsJSON)
	return item, nil
}
