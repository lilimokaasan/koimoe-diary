package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"sakurairo-go/internal/models"
)

type MomentStore struct {
	db *sql.DB
}

type MomentInput struct {
	ID      int64
	Content string
	Author  string
	Status  string
}

func NewMomentStore(db *sql.DB) *MomentStore {
	return &MomentStore{db: db}
}

func (s *MomentStore) Init() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS moments (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	content TEXT NOT NULL,
	author_name VARCHAR(120) NOT NULL,
	status VARCHAR(20) NOT NULL DEFAULT 'published',
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX idx_moments_public (status, created_at),
	INDEX idx_moments_updated (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`)
	return err
}

func (s *MomentStore) ListPublished(ctx context.Context, limit int) ([]models.Moment, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, content, author_name, status, created_at, updated_at
FROM moments
WHERE status = 'published'
ORDER BY created_at DESC, id DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMoments(rows)
}

func (s *MomentStore) ListAdmin(ctx context.Context, limit int) ([]models.Moment, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, content, author_name, status, created_at, updated_at
FROM moments
ORDER BY created_at DESC, id DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMoments(rows)
}

func (s *MomentStore) ByID(ctx context.Context, id int64) (models.Moment, error) {
	var moment models.Moment
	err := s.db.QueryRowContext(ctx, `
SELECT id, content, author_name, status, created_at, updated_at
FROM moments
WHERE id = ?`, id).Scan(
		&moment.ID, &moment.Content, &moment.Author, &moment.Status, &moment.CreatedAt, &moment.UpdatedAt,
	)
	return moment, err
}

func (s *MomentStore) Save(ctx context.Context, input MomentInput) (int64, error) {
	input = normalizeMomentInput(input)
	if input.ID == 0 {
		result, err := s.db.ExecContext(ctx, `
INSERT INTO moments (content, author_name, status, created_at)
VALUES (?, ?, ?, ?)`,
			input.Content, input.Author, input.Status, time.Now(),
		)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE moments
SET content = ?, author_name = ?, status = ?
WHERE id = ?`,
		input.Content, input.Author, input.Status, input.ID,
	)
	return input.ID, err
}

func (s *MomentStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM moments WHERE id = ?`, id)
	return err
}

func normalizeMomentInput(input MomentInput) MomentInput {
	input.Content = strings.TrimSpace(input.Content)
	input.Author = strings.TrimSpace(input.Author)
	input.Status = strings.TrimSpace(input.Status)
	if input.Status != "draft" {
		input.Status = "published"
	}
	return input
}

func scanMoments(rows *sql.Rows) ([]models.Moment, error) {
	var moments []models.Moment
	for rows.Next() {
		var moment models.Moment
		if err := rows.Scan(&moment.ID, &moment.Content, &moment.Author, &moment.Status, &moment.CreatedAt, &moment.UpdatedAt); err != nil {
			return nil, err
		}
		moments = append(moments, moment)
	}
	return moments, rows.Err()
}
