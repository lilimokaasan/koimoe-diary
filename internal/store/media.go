package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"sakurairo-go/internal/models"
)

type MediaStore struct {
	db *sql.DB
}

type MediaListOptions struct {
	Query string
	Limit int
}

type MediaAssetInput struct {
	ID           int64
	Filename     string
	OriginalName string
	Title        string
	AltText      string
	Description  string
	MimeType     string
	SizeBytes    int64
	Width        int
	Height       int
	URL          string
	Storage      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewMediaStore(db *sql.DB) *MediaStore {
	return &MediaStore{db: db}
}

func (s *MediaStore) Init() error {
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS media_assets (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	filename VARCHAR(255) NOT NULL,
	original_name VARCHAR(255) NOT NULL,
	title VARCHAR(255) NOT NULL DEFAULT '',
	alt_text VARCHAR(255) NOT NULL DEFAULT '',
	description TEXT NOT NULL,
	mime_type VARCHAR(120) NOT NULL,
	size_bytes BIGINT NOT NULL DEFAULT 0,
	width INT NOT NULL DEFAULT 0,
	height INT NOT NULL DEFAULT 0,
	url VARCHAR(500) NOT NULL UNIQUE,
	storage VARCHAR(40) NOT NULL DEFAULT 'local',
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX idx_media_assets_created (created_at, id),
	INDEX idx_media_assets_storage (storage, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return err
	}
	if err := s.ensureColumn("title", `ALTER TABLE media_assets ADD COLUMN title VARCHAR(255) NOT NULL DEFAULT '' AFTER original_name`); err != nil {
		return err
	}
	if err := s.ensureColumn("alt_text", `ALTER TABLE media_assets ADD COLUMN alt_text VARCHAR(255) NOT NULL DEFAULT '' AFTER title`); err != nil {
		return err
	}
	return s.ensureColumn("description", `ALTER TABLE media_assets ADD COLUMN description TEXT NOT NULL AFTER alt_text`)
}

func (s *MediaStore) List(ctx context.Context, limit int) ([]models.MediaAsset, error) {
	return s.ListWithOptions(ctx, MediaListOptions{Limit: limit})
}

func (s *MediaStore) ListWithOptions(ctx context.Context, options MediaListOptions) ([]models.MediaAsset, error) {
	options.Query = strings.TrimSpace(options.Query)
	if options.Limit <= 0 {
		options.Limit = 240
	}
	like := "%" + options.Query + "%"
	rows, err := s.db.QueryContext(ctx, `
SELECT id, filename, original_name, title, alt_text, description, mime_type, size_bytes, width, height, url, storage, created_at, updated_at
FROM media_assets
WHERE (? = '' OR filename LIKE ? OR original_name LIKE ? OR title LIKE ? OR alt_text LIKE ? OR description LIKE ? OR url LIKE ?)
ORDER BY created_at DESC, id DESC
LIMIT ?`, options.Query, like, like, like, like, like, like, options.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMediaAssets(rows)
}

func (s *MediaStore) Count(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_assets`).Scan(&count)
	return count, err
}

func (s *MediaStore) ByID(ctx context.Context, id int64) (models.MediaAsset, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, filename, original_name, title, alt_text, description, mime_type, size_bytes, width, height, url, storage, created_at, updated_at
FROM media_assets
WHERE id = ?`, id)
	var asset models.MediaAsset
	err := row.Scan(
		&asset.ID,
		&asset.Filename,
		&asset.OriginalName,
		&asset.Title,
		&asset.AltText,
		&asset.Description,
		&asset.MimeType,
		&asset.SizeBytes,
		&asset.Width,
		&asset.Height,
		&asset.URL,
		&asset.Storage,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)
	return asset, err
}

func (s *MediaStore) UpsertLocal(ctx context.Context, input MediaAssetInput) (int64, error) {
	input = normalizeMediaAssetInput(input)
	result, err := s.db.ExecContext(ctx, `
INSERT INTO media_assets (filename, original_name, title, alt_text, description, mime_type, size_bytes, width, height, url, storage, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	id = LAST_INSERT_ID(id),
	filename = VALUES(filename),
	original_name = VALUES(original_name),
	mime_type = VALUES(mime_type),
	size_bytes = VALUES(size_bytes),
	width = VALUES(width),
	height = VALUES(height),
	storage = VALUES(storage),
	updated_at = VALUES(updated_at)`,
		input.Filename,
		input.OriginalName,
		input.Title,
		input.AltText,
		input.Description,
		input.MimeType,
		input.SizeBytes,
		input.Width,
		input.Height,
		input.URL,
		input.Storage,
		input.CreatedAt,
		input.UpdatedAt,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *MediaStore) UpdateDetails(ctx context.Context, input MediaAssetInput) error {
	input.Title = strings.TrimSpace(input.Title)
	input.AltText = strings.TrimSpace(input.AltText)
	input.Description = strings.TrimSpace(input.Description)
	_, err := s.db.ExecContext(ctx, `
UPDATE media_assets
SET title = ?, alt_text = ?, description = ?
WHERE id = ?`, input.Title, input.AltText, input.Description, input.ID)
	return err
}

func (s *MediaStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM media_assets WHERE id = ?`, id)
	return err
}

func (s *MediaStore) ensureColumn(column string, alter string) error {
	var exists int
	if err := s.db.QueryRow(`
SELECT COUNT(*)
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'media_assets' AND COLUMN_NAME = ?`, column).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}
	_, err := s.db.Exec(alter)
	return err
}

func normalizeMediaAssetInput(input MediaAssetInput) MediaAssetInput {
	input.Filename = strings.TrimSpace(input.Filename)
	input.OriginalName = strings.TrimSpace(input.OriginalName)
	input.Title = strings.TrimSpace(input.Title)
	input.AltText = strings.TrimSpace(input.AltText)
	input.Description = strings.TrimSpace(input.Description)
	input.MimeType = strings.TrimSpace(input.MimeType)
	input.URL = strings.TrimSpace(input.URL)
	input.Storage = strings.TrimSpace(input.Storage)
	if input.OriginalName == "" {
		input.OriginalName = input.Filename
	}
	if input.MimeType == "" {
		input.MimeType = "application/octet-stream"
	}
	if input.Storage == "" {
		input.Storage = "local"
	}
	if input.CreatedAt.IsZero() {
		input.CreatedAt = time.Now()
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = input.CreatedAt
	}
	return input
}

func scanMediaAssets(rows *sql.Rows) ([]models.MediaAsset, error) {
	var assets []models.MediaAsset
	for rows.Next() {
		var asset models.MediaAsset
		if err := rows.Scan(
			&asset.ID,
			&asset.Filename,
			&asset.OriginalName,
			&asset.Title,
			&asset.AltText,
			&asset.Description,
			&asset.MimeType,
			&asset.SizeBytes,
			&asset.Width,
			&asset.Height,
			&asset.URL,
			&asset.Storage,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}
