package store

import (
	"context"
	"database/sql"
	"strings"

	"sakurairo-go/internal/models"
)

type LinkStore struct {
	db *sql.DB
}

type FriendLinkInput struct {
	ID           int64
	CategoryName string
	Name         string
	URL          string
	Description  string
	ImageURL     string
	SortOrder    int
	Visible      bool
}

func NewLinkStore(db *sql.DB) *LinkStore {
	return &LinkStore{db: db}
}

func (s *LinkStore) Init() error {
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS friend_link_categories (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	slug VARCHAR(160) NOT NULL UNIQUE,
	name VARCHAR(120) NOT NULL,
	description TEXT NOT NULL,
	sort_order INT NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX idx_friend_link_categories_sort (sort_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS friend_links (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	category_id BIGINT NOT NULL,
	name VARCHAR(160) NOT NULL,
	url VARCHAR(500) NOT NULL,
	description TEXT NOT NULL,
	image_url VARCHAR(500) NOT NULL DEFAULT '',
	sort_order INT NOT NULL DEFAULT 0,
	visible TINYINT(1) NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX idx_friend_links_public (visible, sort_order, id),
	INDEX idx_friend_links_category (category_id, sort_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return err
	}
	_, err := s.ensureCategory(context.Background(), "Friends", "")
	return err
}

func (s *LinkStore) ListPublic(ctx context.Context) ([]models.FriendLinkCategory, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id, c.name, c.description, c.sort_order, c.created_at, c.updated_at,
       l.id, l.category_id, l.name, l.url, l.description, l.image_url, l.sort_order, l.visible, l.created_at, l.updated_at
FROM friend_link_categories c
JOIN friend_links l ON l.category_id = c.id AND l.visible = 1
ORDER BY c.sort_order ASC, c.id ASC, l.sort_order ASC, l.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanGroupedLinks(rows)
}

func (s *LinkStore) ListCategories(ctx context.Context) ([]models.FriendLinkCategory, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, description, sort_order, created_at, updated_at
FROM friend_link_categories
ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.FriendLinkCategory
	for rows.Next() {
		var category models.FriendLinkCategory
		if err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.SortOrder, &category.CreatedAt, &category.UpdatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (s *LinkStore) ListAdmin(ctx context.Context) ([]models.FriendLink, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT l.id, l.category_id, l.name, l.url, l.description, l.image_url, l.sort_order, l.visible, l.created_at, l.updated_at,
       c.id, c.name, c.description, c.sort_order, c.created_at, c.updated_at
FROM friend_links l
JOIN friend_link_categories c ON c.id = l.category_id
ORDER BY c.sort_order ASC, c.id ASC, l.sort_order ASC, l.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []models.FriendLink
	for rows.Next() {
		link, err := scanLinkWithCategory(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, rows.Err()
}

func (s *LinkStore) ByID(ctx context.Context, id int64) (models.FriendLink, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT l.id, l.category_id, l.name, l.url, l.description, l.image_url, l.sort_order, l.visible, l.created_at, l.updated_at,
       c.id, c.name, c.description, c.sort_order, c.created_at, c.updated_at
FROM friend_links l
JOIN friend_link_categories c ON c.id = l.category_id
WHERE l.id = ?`, id)
	return scanLinkWithCategory(row)
}

func (s *LinkStore) SaveLink(ctx context.Context, input FriendLinkInput) (int64, error) {
	input = normalizeFriendLinkInput(input)
	categoryID, err := s.ensureCategory(ctx, input.CategoryName, "")
	if err != nil {
		return 0, err
	}
	if input.ID == 0 {
		result, err := s.db.ExecContext(ctx, `
INSERT INTO friend_links (category_id, name, url, description, image_url, sort_order, visible)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			categoryID, input.Name, input.URL, input.Description, input.ImageURL, input.SortOrder, input.Visible,
		)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE friend_links
SET category_id = ?, name = ?, url = ?, description = ?, image_url = ?, sort_order = ?, visible = ?
WHERE id = ?`,
		categoryID, input.Name, input.URL, input.Description, input.ImageURL, input.SortOrder, input.Visible, input.ID,
	)
	return input.ID, err
}

func (s *LinkStore) DeleteLink(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM friend_links WHERE id = ?`, id)
	return err
}

func (s *LinkStore) ensureCategory(ctx context.Context, name string, description string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Friends"
	}
	result, err := s.db.ExecContext(ctx, `
INSERT INTO friend_link_categories (slug, name, description)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id), name = VALUES(name), description = VALUES(description)`,
		slugify(name), name, description,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func normalizeFriendLinkInput(input FriendLinkInput) FriendLinkInput {
	input.CategoryName = strings.TrimSpace(input.CategoryName)
	input.Name = strings.TrimSpace(input.Name)
	input.URL = strings.TrimSpace(input.URL)
	input.Description = strings.TrimSpace(input.Description)
	input.ImageURL = strings.TrimSpace(input.ImageURL)
	if input.CategoryName == "" {
		input.CategoryName = "Friends"
	}
	return input
}

type linkScanner interface {
	Scan(dest ...any) error
}

func scanLinkWithCategory(row linkScanner) (models.FriendLink, error) {
	var link models.FriendLink
	var visible bool
	err := row.Scan(
		&link.ID, &link.CategoryID, &link.Name, &link.URL, &link.Description, &link.ImageURL, &link.SortOrder, &visible, &link.CreatedAt, &link.UpdatedAt,
		&link.Category.ID, &link.Category.Name, &link.Category.Description, &link.Category.SortOrder, &link.Category.CreatedAt, &link.Category.UpdatedAt,
	)
	link.Visible = visible
	return link, err
}

func scanGroupedLinks(rows *sql.Rows) ([]models.FriendLinkCategory, error) {
	categories := make([]models.FriendLinkCategory, 0)
	indexByID := make(map[int64]int)
	for rows.Next() {
		var category models.FriendLinkCategory
		var link models.FriendLink
		var visible bool
		if err := rows.Scan(
			&category.ID, &category.Name, &category.Description, &category.SortOrder, &category.CreatedAt, &category.UpdatedAt,
			&link.ID, &link.CategoryID, &link.Name, &link.URL, &link.Description, &link.ImageURL, &link.SortOrder, &visible, &link.CreatedAt, &link.UpdatedAt,
		); err != nil {
			return nil, err
		}
		link.Visible = visible
		link.Category = category
		idx, ok := indexByID[category.ID]
		if !ok {
			idx = len(categories)
			indexByID[category.ID] = idx
			categories = append(categories, category)
		}
		categories[idx].Links = append(categories[idx].Links, link)
	}
	return categories, rows.Err()
}
