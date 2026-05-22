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

type MediaAssetInput struct {
	Filename     string
	OriginalName string
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
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS media_assets (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	filename VARCHAR(255) NOT NULL,
	original_name VARCHAR(255) NOT NULL,
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	return err
}

func (s *MediaStore) List(ctx context.Context, limit int) ([]models.MediaAsset, error) {
	if limit <= 0 {
		limit = 240
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, filename, original_name, mime_type, size_bytes, width, height, url, storage, created_at, updated_at
FROM media_assets
ORDER BY created_at DESC, id DESC
LIMIT ?`, limit)
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

func (s *MediaStore) UpsertLocal(ctx context.Context, input MediaAssetInput) (int64, error) {
	input = normalizeMediaAssetInput(input)
	result, err := s.db.ExecContext(ctx, `
INSERT INTO media_assets (filename, original_name, mime_type, size_bytes, width, height, url, storage, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

func normalizeMediaAssetInput(input MediaAssetInput) MediaAssetInput {
	input.Filename = strings.TrimSpace(input.Filename)
	input.OriginalName = strings.TrimSpace(input.OriginalName)
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
