package store

import (
	"context"
	"database/sql"
	"fmt"
	stdhtml "html"
	"html/template"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"sakurairo-go/internal/commentrender"
	"sakurairo-go/internal/contentoutline"
	"sakurairo-go/internal/legacyshortcode"
	"sakurairo-go/internal/models"
)

type PostStore struct {
	db *sql.DB
}

type PostInput struct {
	ID           int64
	Slug         string
	Title        string
	Excerpt      string
	ContentHTML  string
	CoverImage   string
	Status       string
	IsPinned     bool
	CategoryName string
	Tags         []string
	PublishedAt  time.Time
}

type CommentStatusCounts struct {
	All      int
	Approved int
	Hidden   int
	Spam     int
}

type PostStatusCounts struct {
	All       int
	Published int
	Scheduled int
	Draft     int
	Private   int
}

type PageInput struct {
	ID          int64
	Slug        string
	Title       string
	Excerpt     string
	ContentHTML string
	CoverImage  string
	Status      string
}

type CategoryInput struct {
	ID          int64
	Slug        string
	Name        string
	Description string
	CoverImage  string
}

type TagInput struct {
	ID   int64
	Slug string
	Name string
}

func NewPostStore(db *sql.DB) *PostStore {
	return &PostStore{db: db}
}

func (s *PostStore) Init() error {
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS posts (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	slug VARCHAR(160) NOT NULL UNIQUE,
	title VARCHAR(255) NOT NULL,
	excerpt TEXT NOT NULL,
	content_html MEDIUMTEXT NOT NULL,
	cover_image VARCHAR(500) NOT NULL,
	status VARCHAR(20) NOT NULL DEFAULT 'published',
	is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
	views BIGINT NOT NULL DEFAULT 0,
	likes BIGINT NOT NULL DEFAULT 0,
	published_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX idx_posts_status_published (status, published_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS categories (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	slug VARCHAR(160) NOT NULL UNIQUE,
	name VARCHAR(120) NOT NULL,
	description TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS tags (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	slug VARCHAR(160) NOT NULL UNIQUE,
	name VARCHAR(120) NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS post_tags (
	post_id BIGINT NOT NULL,
	tag_id BIGINT NOT NULL,
	PRIMARY KEY (post_id, tag_id),
	INDEX idx_post_tags_tag (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS comments (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	post_id BIGINT NOT NULL,
	author VARCHAR(120) NOT NULL,
	email VARCHAR(255) NOT NULL,
	website VARCHAR(255) NOT NULL DEFAULT '',
	content TEXT NOT NULL,
	status VARCHAR(20) NOT NULL DEFAULT 'approved',
	parent_id BIGINT NOT NULL DEFAULT 0,
	is_private BOOLEAN NOT NULL DEFAULT FALSE,
	mail_notify BOOLEAN NOT NULL DEFAULT FALSE,
	ip VARCHAR(64) NOT NULL DEFAULT '',
	user_agent VARCHAR(255) NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	INDEX idx_comments_post_status_created (post_id, status, created_at),
	INDEX idx_comments_parent (parent_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS pages (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	slug VARCHAR(160) NOT NULL UNIQUE,
	title VARCHAR(255) NOT NULL,
	excerpt TEXT NOT NULL,
	content_html MEDIUMTEXT NOT NULL,
	cover_image VARCHAR(500) NOT NULL DEFAULT '',
	status VARCHAR(20) NOT NULL DEFAULT 'published',
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX idx_pages_status_updated (status, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return err
	}
	if err := s.ensurePostColumns(); err != nil {
		return err
	}
	if err := s.ensureCategoryColumns(); err != nil {
		return err
	}
	return s.ensureCommentColumns()
}

func (s *PostStore) SeedDemo() error {
	blogID, err := s.upsertCategory("blog", "Blog", "General blog posts.")
	if err != nil {
		return err
	}
	devID, err := s.upsertCategory("development", "Development", "Build notes and engineering logs.")
	if err != nil {
		return err
	}
	goTagID, err := s.upsertTag("go", "Go")
	if err != nil {
		return err
	}
	sakurairoTagID, err := s.upsertTag("sakurairo", "Sakurairo")
	if err != nil {
		return err
	}

	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return s.attachDemoTaxonomy(blogID, devID, goTagID, sakurairoTagID)
	}

	posts := []models.Post{
		{
			Slug:       "hello-sakurairo-go",
			Title:      "Hello, Sakurairo Go",
			Excerpt:    "The first demo post for the Go rewrite of the WordPress theme.",
			CoverImage: "/static/theme/content-image/d-1.jpg",
			Category:   models.Category{ID: blogID},
			ContentHTML: template.HTML(`<p>This first version keeps the core Sakurairo feeling: hero image, notice, feature cards, post list, archive, and search.</p>
<p>Next we can add comments, an admin editor, tags, categories, media management, and theme settings.</p>`),
			PublishedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			Slug:       "why-go",
			Title:      "Why rewrite it in Go",
			Excerpt:    "Go has a simple deployment model and works well as a single binary service.",
			CoverImage: "/static/theme/content-image/d-2.jpg",
			Category:   models.Category{ID: devID},
			ContentHTML: template.HTML(`<p>Compared with WordPress plus PHP-FPM, the Go app can run directly as a long-lived process managed by systemd.</p>
<p>Nginx handles HTTPS, static files, and reverse proxying. MySQL stores the content.</p>`),
			PublishedAt: time.Now().Add(-26 * time.Hour),
		},
	}

	for _, post := range posts {
		result, err := s.db.Exec(`
INSERT INTO posts (slug, title, excerpt, content_html, cover_image, category_id, published_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			post.Slug, post.Title, post.Excerpt, string(post.ContentHTML), post.CoverImage, post.Category.ID, post.PublishedAt,
		)
		if err != nil {
			return err
		}
		postID, _ := result.LastInsertId()
		if err := s.attachTags(postID, goTagID, sakurairoTagID); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostStore) ListPublished(ctx context.Context, limit int) ([]models.Post, error) {
	return s.ListPublishedPaged(ctx, 1, limit)
}

func (s *PostStore) ListRecent(ctx context.Context, limit int) ([]models.Post, error) {
	return s.ListPublishedPaged(ctx, 1, limit)
}

func (s *PostStore) DistinctCoverImages(ctx context.Context, limit int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT cover_image
FROM posts
WHERE status = 'published' AND cover_image <> ''
  AND published_at <= CURRENT_TIMESTAMP
GROUP BY cover_image
ORDER BY MAX(published_at) DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []string
	for rows.Next() {
		var image string
		if err := rows.Scan(&image); err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, rows.Err()
}

func (s *PostStore) ListAll(ctx context.Context, limit int) ([]models.Post, error) {
	return s.ListAllByStatus(ctx, "", limit)
}

func (s *PostStore) ListAllByStatus(ctx context.Context, status string, limit int) ([]models.Post, error) {
	status = postStatusFilter(status)
	where := ""
	args := []any{}
	switch status {
	case "published":
		where = "WHERE p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP"
	case "scheduled":
		where = "WHERE p.status = 'published' AND p.published_at > CURRENT_TIMESTAMP"
	case "draft", "private":
		where = "WHERE p.status = ?"
		args = append(args, status)
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.status, p.is_pinned,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
`+where+`
ORDER BY p.updated_at DESC
LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts, err := scanAdminPosts(rows)
	if err != nil {
		return nil, err
	}
	return posts, s.hydrateTags(ctx, posts)
}

func (s *PostStore) ListAllPages(ctx context.Context, limit int) ([]models.Page, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, slug, title, excerpt, content_html, cover_image, status, created_at, updated_at
FROM pages
ORDER BY updated_at DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPages(rows)
}

func (s *PostStore) ListPublishedPages(ctx context.Context, limit int) ([]models.Page, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, slug, title, excerpt, content_html, cover_image, status, created_at, updated_at
FROM pages
WHERE status = 'published'
ORDER BY updated_at DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPages(rows)
}

func (s *PostStore) ListPublishedPaged(ctx context.Context, page int, pageSize int) ([]models.Post, error) {
	page, pageSize = normalizePage(page, pageSize)
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.is_pinned,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published'
  AND p.published_at <= CURRENT_TIMESTAMP
ORDER BY p.is_pinned DESC, p.published_at DESC
LIMIT ? OFFSET ?`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, err
	}
	return posts, s.hydrateTags(ctx, posts)
}

func (s *PostStore) Search(ctx context.Context, query string, limit int) ([]models.Post, error) {
	return s.SearchPaged(ctx, query, 1, limit)
}

func (s *PostStore) SearchPaged(ctx context.Context, query string, page int, pageSize int) ([]models.Post, error) {
	query = normalizeSearchQuery(query)
	if query == "" {
		return s.ListPublishedPaged(ctx, page, pageSize)
	}
	page, pageSize = normalizePage(page, pageSize)
	like := likePattern(query)
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.is_pinned,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP AND (
	p.title LIKE ? ESCAPE '\\' OR
	p.excerpt LIKE ? ESCAPE '\\' OR
	p.content_html LIKE ? ESCAPE '\\' OR
	c.name LIKE ? ESCAPE '\\' OR
	EXISTS (
		SELECT 1
		FROM post_tags pt
		JOIN tags t ON t.id = pt.tag_id
		WHERE pt.post_id = p.id AND t.name LIKE ? ESCAPE '\\'
	)
)
ORDER BY p.is_pinned DESC, p.published_at DESC
LIMIT ? OFFSET ?`, like, like, like, like, like, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, err
	}
	return posts, s.hydrateTags(ctx, posts)
}

func (s *PostStore) SearchPages(ctx context.Context, query string, limit int) ([]models.Page, error) {
	query = normalizeSearchQuery(query)
	if query == "" {
		return nil, nil
	}
	like := likePattern(query)
	rows, err := s.db.QueryContext(ctx, `
SELECT id, slug, title, excerpt, content_html, cover_image, status, created_at, updated_at
FROM pages
WHERE status = 'published' AND (
	title LIKE ? ESCAPE '\\' OR
	excerpt LIKE ? ESCAPE '\\' OR
	content_html LIKE ? ESCAPE '\\'
)
ORDER BY updated_at DESC
LIMIT ?`, like, like, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPages(rows)
}

func (s *PostStore) ByCategory(ctx context.Context, slug string, page int, pageSize int) ([]models.Post, models.Category, error) {
	page, pageSize = normalizePage(page, pageSize)
	category, err := s.CategoryBySlug(ctx, slug)
	if err != nil {
		return nil, models.Category{}, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.is_pinned,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       c.id, c.slug, c.name, c.description
FROM posts p
JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP AND c.slug = ?
ORDER BY p.is_pinned DESC, p.published_at DESC
LIMIT ? OFFSET ?`, slug, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, models.Category{}, err
	}
	defer rows.Close()
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, models.Category{}, err
	}
	return posts, category, s.hydrateTags(ctx, posts)
}

func (s *PostStore) ByTag(ctx context.Context, slug string, page int, pageSize int) ([]models.Post, models.Tag, error) {
	page, pageSize = normalizePage(page, pageSize)
	tag, err := s.TagBySlug(ctx, slug)
	if err != nil {
		return nil, models.Tag{}, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.is_pinned,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
JOIN post_tags pt ON pt.post_id = p.id
JOIN tags t ON t.id = pt.tag_id
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP AND t.slug = ?
ORDER BY p.is_pinned DESC, p.published_at DESC
LIMIT ? OFFSET ?`, slug, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, models.Tag{}, err
	}
	defer rows.Close()
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, models.Tag{}, err
	}
	return posts, tag, s.hydrateTags(ctx, posts)
}

func (s *PostStore) CountPublished(ctx context.Context) (int, error) {
	return s.count(ctx, `SELECT COUNT(*) FROM posts WHERE status = 'published' AND published_at <= CURRENT_TIMESTAMP`)
}

func (s *PostStore) CountPostsByStatus(ctx context.Context) (PostStatusCounts, error) {
	var counts PostStatusCounts
	err := s.db.QueryRowContext(ctx, `
SELECT
	COUNT(*),
	SUM(CASE WHEN status = 'published' AND published_at <= CURRENT_TIMESTAMP THEN 1 ELSE 0 END),
	SUM(CASE WHEN status = 'published' AND published_at > CURRENT_TIMESTAMP THEN 1 ELSE 0 END),
	SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END),
	SUM(CASE WHEN status = 'private' THEN 1 ELSE 0 END)
FROM posts`).Scan(&counts.All, &counts.Published, &counts.Scheduled, &counts.Draft, &counts.Private)
	return counts, err
}

func (s *PostStore) CountSearch(ctx context.Context, query string) (int, error) {
	query = normalizeSearchQuery(query)
	if query == "" {
		return s.CountPublished(ctx)
	}
	like := likePattern(query)
	return s.count(ctx, `
SELECT COUNT(*)
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP AND (
	p.title LIKE ? ESCAPE '\\' OR
	p.excerpt LIKE ? ESCAPE '\\' OR
	p.content_html LIKE ? ESCAPE '\\' OR
	c.name LIKE ? ESCAPE '\\' OR
	EXISTS (
		SELECT 1
		FROM post_tags pt
		JOIN tags t ON t.id = pt.tag_id
		WHERE pt.post_id = p.id AND t.name LIKE ? ESCAPE '\\'
	)
)`, like, like, like, like, like)
}

func (s *PostStore) CountSearchPages(ctx context.Context, query string) (int, error) {
	query = normalizeSearchQuery(query)
	if query == "" {
		return 0, nil
	}
	like := likePattern(query)
	return s.count(ctx, `
SELECT COUNT(*)
FROM pages
WHERE status = 'published' AND (
	title LIKE ? ESCAPE '\\' OR
	excerpt LIKE ? ESCAPE '\\' OR
	content_html LIKE ? ESCAPE '\\'
)`, like, like, like)
}

func (s *PostStore) CountByCategory(ctx context.Context, slug string) (int, error) {
	return s.count(ctx, `
SELECT COUNT(*)
FROM posts p
JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP AND c.slug = ?`, slug)
}

func (s *PostStore) CountByTag(ctx context.Context, slug string) (int, error) {
	return s.count(ctx, `
SELECT COUNT(*)
FROM posts p
JOIN post_tags pt ON pt.post_id = p.id
JOIN tags t ON t.id = pt.tag_id
WHERE p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP AND t.slug = ?`, slug)
}

func (s *PostStore) ArchiveGroups(ctx context.Context) ([]models.ArchiveGroup, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.is_pinned,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP
ORDER BY p.published_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, err
	}
	groups := make([]models.ArchiveGroup, 0)
	indexByLabel := map[string]int{}
	for _, post := range posts {
		label := post.PublishedAt.Format("2006.01")
		index, ok := indexByLabel[label]
		if !ok {
			index = len(groups)
			indexByLabel[label] = index
			groups = append(groups, models.ArchiveGroup{
				Label: label,
				Year:  post.PublishedAt.Year(),
				Month: post.PublishedAt.Month(),
			})
		}
		groups[index].Posts = append(groups[index].Posts, post)
		groups[index].Count = len(groups[index].Posts)
	}
	return groups, nil
}

func (s *PostStore) BySlug(ctx context.Context, slug string) (models.Post, error) {
	return s.bySlug(ctx, slug, false)
}

func (s *PostStore) BySlugForAdmin(ctx context.Context, slug string) (models.Post, error) {
	return s.bySlug(ctx, slug, true)
}

func (s *PostStore) bySlug(ctx context.Context, slug string, includePrivate bool) (models.Post, error) {
	var post models.Post
	var content string
	statusClause := "AND p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP"
	if includePrivate {
		statusClause = "AND p.status IN ('published', 'private')"
	}
	err := s.db.QueryRowContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.status, p.is_pinned,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.slug = ? `+statusClause+`
LIMIT 1`, slug).Scan(
		&post.ID, &post.Slug, &post.Title, &post.Excerpt, &content, &post.CoverImage, &post.Status,
		&post.IsPinned,
		&post.CommentCount, &post.Views, &post.Likes, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
		&post.Category.ID, &post.Category.Slug, &post.Category.Name, &post.Category.Description,
	)
	post.ContentHTML = template.HTML(content)
	post.ReadingMinutes = readingMinutes(content)
	enhancePostContent(&post)
	if err == nil {
		posts := []models.Post{post}
		err = s.hydrateTags(ctx, posts)
		post = posts[0]
	}
	return post, err
}

func (s *PostStore) ByID(ctx context.Context, id int64) (models.Post, error) {
	var post models.Post
	var content string
	err := s.db.QueryRowContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.status, p.is_pinned,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.id = ?
LIMIT 1`, id).Scan(
		&post.ID, &post.Slug, &post.Title, &post.Excerpt, &content, &post.CoverImage, &post.Status,
		&post.IsPinned,
		&post.CommentCount, &post.Views, &post.Likes, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
		&post.Category.ID, &post.Category.Slug, &post.Category.Name, &post.Category.Description,
	)
	post.ContentHTML = template.HTML(content)
	post.ReadingMinutes = readingMinutes(content)
	enhancePostContent(&post)
	if err == nil {
		posts := []models.Post{post}
		err = s.hydrateTags(ctx, posts)
		post = posts[0]
	}
	return post, err
}

func (s *PostStore) PageBySlug(ctx context.Context, slug string) (models.Page, error) {
	return s.pageBySlug(ctx, slug, false)
}

func (s *PostStore) PageBySlugForAdmin(ctx context.Context, slug string) (models.Page, error) {
	return s.pageBySlug(ctx, slug, true)
}

func (s *PostStore) pageBySlug(ctx context.Context, slug string, includePrivate bool) (models.Page, error) {
	var page models.Page
	var content string
	statusClause := "AND status = 'published'"
	if includePrivate {
		statusClause = "AND status IN ('published', 'private')"
	}
	err := s.db.QueryRowContext(ctx, `
SELECT id, slug, title, excerpt, content_html, cover_image, status, created_at, updated_at
FROM pages
WHERE slug = ? `+statusClause+`
LIMIT 1`, slug).Scan(
		&page.ID, &page.Slug, &page.Title, &page.Excerpt, &content, &page.CoverImage, &page.Status, &page.CreatedAt, &page.UpdatedAt,
	)
	page.ContentHTML = template.HTML(content)
	enhancePageContent(&page)
	return page, err
}

func (s *PostStore) PageByID(ctx context.Context, id int64) (models.Page, error) {
	var page models.Page
	var content string
	err := s.db.QueryRowContext(ctx, `
SELECT id, slug, title, excerpt, content_html, cover_image, status, created_at, updated_at
FROM pages
WHERE id = ?
LIMIT 1`, id).Scan(
		&page.ID, &page.Slug, &page.Title, &page.Excerpt, &content, &page.CoverImage, &page.Status, &page.CreatedAt, &page.UpdatedAt,
	)
	page.ContentHTML = template.HTML(content)
	enhancePageContent(&page)
	return page, err
}

func (s *PostStore) AdjacentPublished(ctx context.Context, post models.Post) (models.Post, models.Post, error) {
	previous, err := s.adjacentPublished(ctx, post, `<`, `DESC`)
	if err != nil && err != sql.ErrNoRows {
		return models.Post{}, models.Post{}, err
	}
	next, nextErr := s.adjacentPublished(ctx, post, `>`, `ASC`)
	if nextErr != nil && nextErr != sql.ErrNoRows {
		return models.Post{}, models.Post{}, nextErr
	}
	return previous, next, nil
}

func (s *PostStore) RelatedPublished(ctx context.Context, post models.Post, limit int) ([]models.Post, error) {
	if limit <= 0 {
		limit = 3
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.is_pinned,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, ''),
       (CASE WHEN p.category_id = ? AND ? <> 0 THEN 2 ELSE 0 END + COUNT(DISTINCT shared.tag_id)) AS related_score
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
LEFT JOIN post_tags shared ON shared.post_id = p.id AND shared.tag_id IN (
	SELECT tag_id FROM post_tags WHERE post_id = ?
)
WHERE p.status = 'published'
  AND p.published_at <= CURRENT_TIMESTAMP
  AND p.id <> ?
GROUP BY p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.is_pinned,
         p.views, p.likes, p.published_at, p.created_at, p.updated_at,
         c.id, c.slug, c.name, c.description, p.category_id
ORDER BY related_score DESC, p.is_pinned DESC, p.published_at DESC
LIMIT ?`, post.Category.ID, post.Category.ID, post.ID, post.ID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var related models.Post
		var content string
		var score int
		if err := rows.Scan(
			&related.ID, &related.Slug, &related.Title, &related.Excerpt, &content, &related.CoverImage,
			&related.IsPinned,
			&related.CommentCount, &related.Views, &related.Likes, &related.PublishedAt, &related.CreatedAt, &related.UpdatedAt,
			&related.Category.ID, &related.Category.Slug, &related.Category.Name, &related.Category.Description,
			&score,
		); err != nil {
			return nil, err
		}
		related.ContentHTML = template.HTML(content)
		related.ReadingMinutes = readingMinutes(content)
		posts = append(posts, related)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return posts, s.hydrateTags(ctx, posts)
}

func (s *PostStore) CategoryBySlug(ctx context.Context, slug string) (models.Category, error) {
	var category models.Category
	err := s.db.QueryRowContext(ctx, `
SELECT id, slug, name, description, cover_image
FROM categories
WHERE slug = ?
LIMIT 1`, slug).Scan(&category.ID, &category.Slug, &category.Name, &category.Description, &category.CoverImage)
	return category, err
}

func (s *PostStore) TagBySlug(ctx context.Context, slug string) (models.Tag, error) {
	var tag models.Tag
	err := s.db.QueryRowContext(ctx, `
SELECT id, slug, name
FROM tags
WHERE slug = ?
LIMIT 1`, slug).Scan(&tag.ID, &tag.Slug, &tag.Name)
	return tag, err
}

func (s *PostStore) ListCategories(ctx context.Context) ([]models.Category, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id, c.slug, c.name, c.description, COUNT(p.id) AS post_count
FROM categories c
LEFT JOIN posts p ON p.category_id = c.id AND p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP
GROUP BY c.id, c.slug, c.name, c.description
HAVING post_count > 0
ORDER BY c.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(&category.ID, &category.Slug, &category.Name, &category.Description, &category.PostCount); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (s *PostStore) ListTags(ctx context.Context) ([]models.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT t.id, t.slug, t.name, COUNT(pt.post_id) AS post_count
FROM tags t
JOIN post_tags pt ON pt.tag_id = t.id
JOIN posts p ON p.id = pt.post_id AND p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP
GROUP BY t.id, t.slug, t.name
ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Slug, &tag.Name, &tag.PostCount); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *PostStore) ListCategoriesAdmin(ctx context.Context) ([]models.Category, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id, c.slug, c.name, c.description, c.cover_image, COUNT(p.id) AS post_count
FROM categories c
LEFT JOIN posts p ON p.category_id = c.id
GROUP BY c.id, c.slug, c.name, c.description, c.cover_image
ORDER BY c.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(&category.ID, &category.Slug, &category.Name, &category.Description, &category.CoverImage, &category.PostCount); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (s *PostStore) ListTagsAdmin(ctx context.Context) ([]models.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT t.id, t.slug, t.name, COUNT(pt.post_id) AS post_count
FROM tags t
LEFT JOIN post_tags pt ON pt.tag_id = t.id
GROUP BY t.id, t.slug, t.name
ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Slug, &tag.Name, &tag.PostCount); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *PostStore) CategoryByID(ctx context.Context, id int64) (models.Category, error) {
	var category models.Category
	err := s.db.QueryRowContext(ctx, `
SELECT c.id, c.slug, c.name, c.description, c.cover_image, COUNT(p.id) AS post_count
FROM categories c
LEFT JOIN posts p ON p.category_id = c.id
WHERE c.id = ?
GROUP BY c.id, c.slug, c.name, c.description, c.cover_image
LIMIT 1`, id).Scan(&category.ID, &category.Slug, &category.Name, &category.Description, &category.CoverImage, &category.PostCount)
	return category, err
}

func (s *PostStore) TagByID(ctx context.Context, id int64) (models.Tag, error) {
	var tag models.Tag
	err := s.db.QueryRowContext(ctx, `
SELECT t.id, t.slug, t.name, COUNT(pt.post_id) AS post_count
FROM tags t
LEFT JOIN post_tags pt ON pt.tag_id = t.id
WHERE t.id = ?
GROUP BY t.id, t.slug, t.name
LIMIT 1`, id).Scan(&tag.ID, &tag.Slug, &tag.Name, &tag.PostCount)
	return tag, err
}

func (s *PostStore) SaveCategory(ctx context.Context, input CategoryInput) (int64, error) {
	input = normalizeCategoryInput(input)
	if input.ID == 0 {
		result, err := s.db.ExecContext(ctx, `
INSERT INTO categories (slug, name, description, cover_image)
VALUES (?, ?, ?, ?)`, input.Slug, input.Name, input.Description, input.CoverImage)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE categories
SET slug = ?, name = ?, description = ?, cover_image = ?
WHERE id = ?`, input.Slug, input.Name, input.Description, input.CoverImage, input.ID)
	return input.ID, err
}

func (s *PostStore) SaveTag(ctx context.Context, input TagInput) (int64, error) {
	input = normalizeTagInput(input)
	if input.ID == 0 {
		result, err := s.db.ExecContext(ctx, `
INSERT INTO tags (slug, name)
VALUES (?, ?)`, input.Slug, input.Name)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE tags
SET slug = ?, name = ?
WHERE id = ?`, input.Slug, input.Name, input.ID)
	return input.ID, err
}

func (s *PostStore) DeleteCategory(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, `UPDATE posts SET category_id = NULL WHERE category_id = ?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id); err != nil {
		return err
	}
	err = tx.Commit()
	return err
}

func (s *PostStore) DeleteTag(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, `DELETE FROM post_tags WHERE tag_id = ?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, id); err != nil {
		return err
	}
	err = tx.Commit()
	return err
}

func (s *PostStore) SearchCategories(ctx context.Context, query string, limit int) ([]models.Category, error) {
	query = normalizeSearchQuery(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 6
	}
	like := likePattern(query)
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id, c.slug, c.name, c.description, COUNT(p.id) AS post_count
FROM categories c
LEFT JOIN posts p ON p.category_id = c.id AND p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP
WHERE c.name LIKE ? ESCAPE '\\' OR c.slug LIKE ? ESCAPE '\\' OR c.description LIKE ? ESCAPE '\\'
GROUP BY c.id, c.slug, c.name, c.description
HAVING post_count > 0
ORDER BY post_count DESC, c.name
LIMIT ?`, like, like, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(&category.ID, &category.Slug, &category.Name, &category.Description, &category.PostCount); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (s *PostStore) SearchTags(ctx context.Context, query string, limit int) ([]models.Tag, error) {
	query = normalizeSearchQuery(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 8
	}
	like := likePattern(query)
	rows, err := s.db.QueryContext(ctx, `
SELECT t.id, t.slug, t.name, COUNT(pt.post_id) AS post_count
FROM tags t
JOIN post_tags pt ON pt.tag_id = t.id
JOIN posts p ON p.id = pt.post_id AND p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP
WHERE t.name LIKE ? ESCAPE '\\' OR t.slug LIKE ? ESCAPE '\\'
GROUP BY t.id, t.slug, t.name
ORDER BY post_count DESC, t.name
LIMIT ?`, like, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Slug, &tag.Name, &tag.PostCount); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *PostStore) SearchIndex(ctx context.Context) (models.SearchIndex, error) {
	posts, err := s.ListPublished(ctx, 200)
	if err != nil {
		return models.SearchIndex{}, err
	}
	categories, err := s.ListCategories(ctx)
	if err != nil {
		return models.SearchIndex{}, err
	}
	tags, err := s.ListTags(ctx)
	if err != nil {
		return models.SearchIndex{}, err
	}
	pages, err := s.ListPublishedPages(ctx, 200)
	if err != nil {
		return models.SearchIndex{}, err
	}

	index := models.SearchIndex{
		GeneratedAt: time.Now(),
		Posts:       make([]models.SearchPostItem, 0, len(posts)),
		Pages:       make([]models.SearchPostItem, 0, len(pages)),
		Categories:  make([]models.SearchTaxonomyItem, 0, len(categories)),
		Tags:        make([]models.SearchTaxonomyItem, 0, len(tags)),
		Comments:    []models.SearchCommentItem{},
	}
	for _, post := range posts {
		tagNames := make([]string, 0, len(post.Tags))
		for _, tag := range post.Tags {
			tagNames = append(tagNames, tag.Name)
		}
		index.Posts = append(index.Posts, models.SearchPostItem{
			Title:        post.Title,
			URL:          "/post/" + post.Slug,
			Excerpt:      post.Excerpt,
			Content:      searchText(string(post.ContentHTML)),
			CoverImage:   post.CoverImage,
			Category:     post.Category.Name,
			Tags:         tagNames,
			CommentCount: post.CommentCount,
			Views:        post.Views,
			Likes:        post.Likes,
			PublishedAt:  post.PublishedAt,
		})
	}
	for _, page := range pages {
		index.Pages = append(index.Pages, models.SearchPostItem{
			Title:       page.Title,
			URL:         "/page/" + page.Slug,
			Excerpt:     page.Excerpt,
			Content:     searchText(string(page.ContentHTML)),
			CoverImage:  page.CoverImage,
			PublishedAt: page.UpdatedAt,
		})
	}
	for _, category := range categories {
		index.Categories = append(index.Categories, models.SearchTaxonomyItem{
			Name:      category.Name,
			URL:       "/category/" + category.Slug,
			PostCount: category.PostCount,
		})
	}
	for _, tag := range tags {
		index.Tags = append(index.Tags, models.SearchTaxonomyItem{
			Name:      tag.Name,
			URL:       "/tag/" + tag.Slug,
			PostCount: tag.PostCount,
		})
	}
	return index, nil
}

func (s *PostStore) CountComments(ctx context.Context) (int, error) {
	return s.count(ctx, `SELECT COUNT(*) FROM comments WHERE status = 'approved'`)
}

func (s *PostStore) CountCommentsByStatus(ctx context.Context) (CommentStatusCounts, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM comments GROUP BY status`)
	if err != nil {
		return CommentStatusCounts{}, err
	}
	defer rows.Close()

	var counts CommentStatusCounts
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return CommentStatusCounts{}, err
		}
		counts.All += count
		switch normalizeCommentStatus(status) {
		case "approved":
			counts.Approved += count
		case "hidden":
			counts.Hidden += count
		case "spam":
			counts.Spam += count
		}
	}
	return counts, rows.Err()
}

func (s *PostStore) IncrementViews(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE posts SET views = views + 1 WHERE id = ?`, id)
	return err
}

func (s *PostStore) IncrementLikes(ctx context.Context, id int64) (int64, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE posts SET likes = likes + 1 WHERE id = ? AND status = 'published' AND published_at <= CURRENT_TIMESTAMP`, id)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected == 0 {
		return 0, sql.ErrNoRows
	}
	var likes int64
	if err := s.db.QueryRowContext(ctx, `SELECT likes FROM posts WHERE id = ?`, id).Scan(&likes); err != nil {
		return 0, err
	}
	return likes, nil
}

func (s *PostStore) ListComments(ctx context.Context, postID int64) ([]models.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, post_id, parent_id, author, email, website, content, status, is_private, mail_notify, created_at
FROM comments
WHERE post_id = ? AND status = 'approved'
ORDER BY created_at ASC`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(
			&comment.ID, &comment.PostID, &comment.ParentID, &comment.Author, &comment.Email,
			&comment.Website, &comment.Content, &comment.Status, &comment.IsPrivate, &comment.MailNotify, &comment.CreatedAt,
		); err != nil {
			return nil, err
		}
		comment.ContentHTML = commentrender.HTML(comment.Content)
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return buildCommentTree(comments), nil
}

func (s *PostStore) ListAllComments(ctx context.Context, limit int) ([]models.Comment, error) {
	return s.ListAllCommentsByStatus(ctx, "", limit)
}

func (s *PostStore) ListAllCommentsByStatus(ctx context.Context, status string, limit int) ([]models.Comment, error) {
	status = commentStatusFilter(status)
	where := ""
	args := make([]any, 0, 2)
	if status != "" {
		where = "WHERE cm.status = ?"
		args = append(args, status)
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `
SELECT cm.id, cm.post_id, cm.parent_id, parent.author, p.title, p.slug, cm.author, cm.email, cm.website, cm.content, cm.status, cm.is_private, cm.mail_notify, cm.created_at
FROM comments cm
JOIN posts p ON p.id = cm.post_id
LEFT JOIN comments parent ON parent.id = cm.parent_id
`+where+`
ORDER BY cm.created_at DESC
LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		var parentAuthor sql.NullString
		if err := rows.Scan(
			&comment.ID, &comment.PostID, &comment.ParentID, &parentAuthor, &comment.PostTitle, &comment.PostSlug,
			&comment.Author, &comment.Email, &comment.Website, &comment.Content,
			&comment.Status, &comment.IsPrivate, &comment.MailNotify, &comment.CreatedAt,
		); err != nil {
			return nil, err
		}
		comment.ParentAuthor = parentAuthor.String
		comment.ContentHTML = commentrender.HTML(comment.Content)
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

func (s *PostStore) CommentByID(ctx context.Context, id int64) (models.Comment, error) {
	var comment models.Comment
	var parentAuthor sql.NullString
	err := s.db.QueryRowContext(ctx, `
SELECT cm.id, cm.post_id, cm.parent_id, parent.author, p.title, p.slug, cm.author, cm.email, cm.website, cm.content, cm.status, cm.is_private, cm.mail_notify, cm.created_at
FROM comments cm
JOIN posts p ON p.id = cm.post_id
LEFT JOIN comments parent ON parent.id = cm.parent_id
WHERE cm.id = ?`, id).Scan(
		&comment.ID, &comment.PostID, &comment.ParentID, &parentAuthor, &comment.PostTitle, &comment.PostSlug,
		&comment.Author, &comment.Email, &comment.Website, &comment.Content,
		&comment.Status, &comment.IsPrivate, &comment.MailNotify, &comment.CreatedAt,
	)
	comment.ParentAuthor = parentAuthor.String
	comment.ContentHTML = commentrender.HTML(comment.Content)
	return comment, err
}

func (s *PostStore) CreateComment(ctx context.Context, comment models.Comment, ip string, userAgent string) error {
	status := normalizeCommentStatus(comment.Status)
	if status == "" {
		status = "approved"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO comments (post_id, parent_id, author, email, website, content, status, is_private, mail_notify, ip, user_agent)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		comment.PostID, comment.ParentID, comment.Author, comment.Email, comment.Website, comment.Content, status, comment.IsPrivate, comment.MailNotify, ip, userAgent,
	)
	return err
}

func (s *PostStore) UpdateCommentStatus(ctx context.Context, id int64, status string) error {
	status = normalizeCommentStatus(status)
	if status == "" {
		status = "hidden"
	}
	_, err := s.db.ExecContext(ctx, `UPDATE comments SET status = ? WHERE id = ?`, status, id)
	return err
}

func (s *PostStore) UpdateCommentsStatus(ctx context.Context, ids []int64, status string) (int64, error) {
	status = normalizeCommentStatus(status)
	if status == "" {
		status = "hidden"
	}
	return s.execCommentBulk(ctx, `UPDATE comments SET status = ? WHERE id IN (%s)`, append([]any{status}, int64Args(ids)...), ids)
}

func postStatusFilter(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "published", "scheduled", "draft", "private":
		return strings.TrimSpace(strings.ToLower(status))
	default:
		return ""
	}
}

func normalizeCommentStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "approved":
		return "approved"
	case "hidden":
		return "hidden"
	case "spam":
		return "spam"
	default:
		return ""
	}
}

func commentStatusFilter(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "approved", "hidden", "spam":
		return strings.TrimSpace(strings.ToLower(status))
	default:
		return ""
	}
}

func (s *PostStore) UpdateCommentPrivacy(ctx context.Context, id int64, isPrivate bool) error {
	_, err := s.db.ExecContext(ctx, `UPDATE comments SET is_private = ? WHERE id = ?`, isPrivate, id)
	return err
}

func (s *PostStore) UpdateCommentsPrivacy(ctx context.Context, ids []int64, isPrivate bool) (int64, error) {
	return s.execCommentBulk(ctx, `UPDATE comments SET is_private = ? WHERE id IN (%s)`, append([]any{isPrivate}, int64Args(ids)...), ids)
}

func (s *PostStore) DeleteComment(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, id)
	return err
}

func (s *PostStore) DeleteComments(ctx context.Context, ids []int64) (int64, error) {
	return s.execCommentBulk(ctx, `DELETE FROM comments WHERE id IN (%s)`, int64Args(ids), ids)
}

func (s *PostStore) execCommentBulk(ctx context.Context, query string, args []any, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := s.db.ExecContext(ctx, fmt.Sprintf(query, placeholders(len(ids))), args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func int64Args(values []int64) []any {
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	return args
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func buildCommentTree(comments []models.Comment) []models.Comment {
	byID := make(map[int64]*models.Comment, len(comments))
	for i := range comments {
		comments[i].Replies = nil
		byID[comments[i].ID] = &comments[i]
	}
	childIDs := make(map[int64]bool, len(comments))
	for i := range comments {
		comment := &comments[i]
		if comment.ParentID > 0 {
			if parent := byID[comment.ParentID]; parent != nil {
				comment.ParentAuthor = parent.Author
				parent.Replies = append(parent.Replies, *comment)
				childIDs[comment.ID] = true
			}
		}
	}
	var roots []models.Comment
	for i := range comments {
		if childIDs[comments[i].ID] {
			continue
		}
		roots = append(roots, comments[i])
	}
	return roots
}

func (s *PostStore) SavePost(ctx context.Context, input PostInput) (int64, error) {
	input = normalizePostInput(input)
	categoryID, err := s.upsertCategory(slugify(input.CategoryName), input.CategoryName, "")
	if err != nil {
		return 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	postID := input.ID
	if input.ID == 0 {
		result, execErr := tx.ExecContext(ctx, `
INSERT INTO posts (slug, title, excerpt, content_html, cover_image, category_id, status, is_pinned, published_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			input.Slug, input.Title, input.Excerpt, input.ContentHTML, input.CoverImage, categoryID, input.Status, input.IsPinned, input.PublishedAt,
		)
		if execErr != nil {
			err = execErr
			return 0, err
		}
		postID, err = result.LastInsertId()
		if err != nil {
			return 0, err
		}
	} else {
		_, err = tx.ExecContext(ctx, `
UPDATE posts
SET slug = ?, title = ?, excerpt = ?, content_html = ?, cover_image = ?, category_id = ?, status = ?, is_pinned = ?, published_at = ?
WHERE id = ?`,
			input.Slug, input.Title, input.Excerpt, input.ContentHTML, input.CoverImage, categoryID, input.Status, input.IsPinned, input.PublishedAt, input.ID,
		)
		if err != nil {
			return 0, err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM post_tags WHERE post_id = ?`, input.ID); err != nil {
			return 0, err
		}
	}

	for _, tagName := range input.Tags {
		tagName = strings.TrimSpace(tagName)
		if tagName == "" {
			continue
		}
		tagID, tagErr := s.upsertTag(slugify(tagName), tagName)
		if tagErr != nil {
			err = tagErr
			return 0, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT IGNORE INTO post_tags (post_id, tag_id) VALUES (?, ?)`, postID, tagID); err != nil {
			return 0, err
		}
	}

	err = tx.Commit()
	return postID, err
}

func (s *PostStore) SavePage(ctx context.Context, input PageInput) (int64, error) {
	input = normalizePageInput(input)
	now := time.Now()
	if input.ID == 0 {
		result, err := s.db.ExecContext(ctx, `
INSERT INTO pages (slug, title, excerpt, content_html, cover_image, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			input.Slug, input.Title, input.Excerpt, input.ContentHTML, input.CoverImage, input.Status, now, now,
		)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE pages
SET slug = ?, title = ?, excerpt = ?, content_html = ?, cover_image = ?, status = ?
WHERE id = ?`,
		input.Slug, input.Title, input.Excerpt, input.ContentHTML, input.CoverImage, input.Status, input.ID,
	)
	return input.ID, err
}

func scanPages(rows *sql.Rows) ([]models.Page, error) {
	var pages []models.Page
	for rows.Next() {
		var page models.Page
		var content string
		if err := rows.Scan(
			&page.ID, &page.Slug, &page.Title, &page.Excerpt, &content, &page.CoverImage, &page.Status, &page.CreatedAt, &page.UpdatedAt,
		); err != nil {
			return nil, err
		}
		page.ContentHTML = template.HTML(content)
		pages = append(pages, page)
	}
	return pages, rows.Err()
}

func scanAdminPosts(rows *sql.Rows) ([]models.Post, error) {
	var posts []models.Post
	for rows.Next() {
		var post models.Post
		var content string
		if err := rows.Scan(
			&post.ID, &post.Slug, &post.Title, &post.Excerpt, &content, &post.CoverImage, &post.Status,
			&post.IsPinned,
			&post.CommentCount, &post.Views, &post.Likes, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
			&post.Category.ID, &post.Category.Slug, &post.Category.Name, &post.Category.Description,
		); err != nil {
			return nil, err
		}
		post.ContentHTML = template.HTML(content)
		post.ReadingMinutes = readingMinutes(content)
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func scanPosts(rows *sql.Rows) ([]models.Post, error) {
	var posts []models.Post
	for rows.Next() {
		var post models.Post
		var content string
		if err := rows.Scan(
			&post.ID, &post.Slug, &post.Title, &post.Excerpt, &content, &post.CoverImage,
			&post.IsPinned,
			&post.CommentCount, &post.Views, &post.Likes, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
			&post.Category.ID, &post.Category.Slug, &post.Category.Name, &post.Category.Description,
		); err != nil {
			return nil, err
		}
		post.ContentHTML = template.HTML(content)
		post.ReadingMinutes = readingMinutes(content)
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (s *PostStore) adjacentPublished(ctx context.Context, post models.Post, comparator string, direction string) (models.Post, error) {
	var adjacent models.Post
	var content string
	query := `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.is_pinned,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND p.published_at <= CURRENT_TIMESTAMP AND p.published_at ` + comparator + ` ?
ORDER BY p.published_at ` + direction + `
LIMIT 1`
	err := s.db.QueryRowContext(ctx, query, post.PublishedAt).Scan(
		&adjacent.ID, &adjacent.Slug, &adjacent.Title, &adjacent.Excerpt, &content, &adjacent.CoverImage,
		&adjacent.IsPinned,
		&adjacent.CommentCount, &adjacent.Views, &adjacent.Likes, &adjacent.PublishedAt, &adjacent.CreatedAt, &adjacent.UpdatedAt,
		&adjacent.Category.ID, &adjacent.Category.Slug, &adjacent.Category.Name, &adjacent.Category.Description,
	)
	adjacent.ContentHTML = template.HTML(content)
	adjacent.ReadingMinutes = readingMinutes(content)
	if err == nil {
		posts := []models.Post{adjacent}
		err = s.hydrateTags(ctx, posts)
		adjacent = posts[0]
	}
	return adjacent, err
}

func PageInfo(page int, pageSize int, total int, basePath string, query string) models.PageInfo {
	page, pageSize = normalizePage(page, pageSize)
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	return models.PageInfo{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		BasePath:   basePath,
		Query:      query,
	}
}

func normalizePage(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func (s *PostStore) ensurePostColumns() error {
	if err := s.ensureColumn("posts", "category_id", `ALTER TABLE posts ADD COLUMN category_id BIGINT NULL AFTER cover_image`); err != nil {
		return err
	}
	if err := s.ensureColumn("posts", "likes", `ALTER TABLE posts ADD COLUMN likes BIGINT NOT NULL DEFAULT 0 AFTER views`); err != nil {
		return err
	}
	return s.ensureColumn("posts", "is_pinned", `ALTER TABLE posts ADD COLUMN is_pinned BOOLEAN NOT NULL DEFAULT FALSE AFTER status`)
}

func (s *PostStore) ensureCategoryColumns() error {
	return s.ensureColumn("categories", "cover_image", `ALTER TABLE categories ADD COLUMN cover_image VARCHAR(500) NOT NULL DEFAULT '' AFTER description`)
}

func (s *PostStore) ensureCommentColumns() error {
	if err := s.ensureColumn("comments", "parent_id", `ALTER TABLE comments ADD COLUMN parent_id BIGINT NOT NULL DEFAULT 0 AFTER status`); err != nil {
		return err
	}
	if err := s.ensureColumn("comments", "is_private", `ALTER TABLE comments ADD COLUMN is_private BOOLEAN NOT NULL DEFAULT FALSE AFTER status`); err != nil {
		return err
	}
	return s.ensureColumn("comments", "mail_notify", `ALTER TABLE comments ADD COLUMN mail_notify BOOLEAN NOT NULL DEFAULT FALSE AFTER is_private`)
}

func (s *PostStore) ensureColumn(table string, column string, alter string) error {
	var exists int
	if err := s.db.QueryRow(`
SELECT COUNT(*)
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`, table, column).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}
	_, err := s.db.Exec(alter)
	return err
}

func (s *PostStore) upsertCategory(slug string, name string, description string) (int64, error) {
	result, err := s.db.Exec(`
INSERT INTO categories (slug, name, description)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id), name = VALUES(name), description = VALUES(description)`, slug, name, description)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *PostStore) upsertTag(slug string, name string) (int64, error) {
	result, err := s.db.Exec(`
INSERT INTO tags (slug, name)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id), name = VALUES(name)`, slug, name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *PostStore) attachTags(postID int64, tagIDs ...int64) error {
	for _, tagID := range tagIDs {
		if _, err := s.db.Exec(`
INSERT IGNORE INTO post_tags (post_id, tag_id)
VALUES (?, ?)`, postID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostStore) attachDemoTaxonomy(blogID int64, devID int64, tagIDs ...int64) error {
	rows, err := s.db.Query(`SELECT id, slug FROM posts WHERE slug IN ('hello-sakurairo-go', 'why-go')`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		var slug string
		if err := rows.Scan(&postID, &slug); err != nil {
			return err
		}
		categoryID := blogID
		if slug == "why-go" {
			categoryID = devID
		}
		if _, err := s.db.Exec(`UPDATE posts SET category_id = COALESCE(category_id, ?) WHERE id = ?`, categoryID, postID); err != nil {
			return err
		}
		if err := s.attachTags(postID, tagIDs...); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *PostStore) hydrateTags(ctx context.Context, posts []models.Post) error {
	for i := range posts {
		rows, err := s.db.QueryContext(ctx, `
SELECT t.id, t.slug, t.name
FROM tags t
JOIN post_tags pt ON pt.tag_id = t.id
WHERE pt.post_id = ?
ORDER BY t.name`, posts[i].ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var tag models.Tag
			if err := rows.Scan(&tag.ID, &tag.Slug, &tag.Name); err != nil {
				_ = rows.Close()
				return err
			}
			posts[i].Tags = append(posts[i].Tags, tag)
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostStore) count(ctx context.Context, query string, args ...any) (int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

func enhancePostContent(post *models.Post) {
	post.ContentHTML = legacyshortcode.Apply(post.ContentHTML)
	post.ContentHTML, post.Outline = contentoutline.Apply(post.ContentHTML)
}

func enhancePageContent(page *models.Page) {
	page.ContentHTML = legacyshortcode.Apply(page.ContentHTML)
	page.ContentHTML, page.Outline = contentoutline.Apply(page.ContentHTML)
}

func normalizePostInput(input PostInput) PostInput {
	input.Title = strings.TrimSpace(input.Title)
	input.Slug = slugify(input.Slug)
	if input.Slug == "" {
		input.Slug = slugify(input.Title)
	}
	if input.Slug == "" {
		input.Slug = "post-" + time.Now().Format("20060102150405")
	}
	input.Excerpt = strings.TrimSpace(input.Excerpt)
	input.ContentHTML = strings.TrimSpace(input.ContentHTML)
	input.CoverImage = strings.TrimSpace(input.CoverImage)
	if input.CoverImage == "" {
		input.CoverImage = "/static/theme/content-image/d-1.jpg"
	}
	input.Status = strings.TrimSpace(input.Status)
	if input.Status != "draft" && input.Status != "private" {
		input.Status = "published"
	}
	if input.PublishedAt.IsZero() {
		input.PublishedAt = time.Now()
	}
	input.CategoryName = strings.TrimSpace(input.CategoryName)
	if input.CategoryName == "" {
		input.CategoryName = "Blog"
	}
	return input
}

func normalizePageInput(input PageInput) PageInput {
	input.Title = strings.TrimSpace(input.Title)
	input.Slug = slugify(input.Slug)
	if input.Slug == "" {
		input.Slug = slugify(input.Title)
	}
	if input.Slug == "" {
		if input.ID > 0 {
			input.Slug = "page-" + strconv.FormatInt(input.ID, 10)
		} else {
			input.Slug = "page-" + time.Now().Format("20060102150405")
		}
	}
	input.Excerpt = strings.TrimSpace(input.Excerpt)
	input.ContentHTML = strings.TrimSpace(input.ContentHTML)
	input.CoverImage = strings.TrimSpace(input.CoverImage)
	input.Status = strings.TrimSpace(input.Status)
	if input.Status != "draft" && input.Status != "private" {
		input.Status = "published"
	}
	return input
}

func normalizeCategoryInput(input CategoryInput) CategoryInput {
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Description = strings.TrimSpace(input.Description)
	input.CoverImage = strings.TrimSpace(input.CoverImage)
	if input.Slug == "" {
		input.Slug = slugify(input.Name)
	} else {
		input.Slug = slugify(input.Slug)
	}
	if input.Slug == "" {
		if input.ID > 0 {
			input.Slug = "category-" + strconv.FormatInt(input.ID, 10)
		} else {
			input.Slug = "category-" + time.Now().Format("20060102150405")
		}
	}
	return input
}

func normalizeTagInput(input TagInput) TagInput {
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	if input.Slug == "" {
		input.Slug = slugify(input.Name)
	} else {
		input.Slug = slugify(input.Slug)
	}
	if input.Slug == "" {
		if input.ID > 0 {
			input.Slug = "tag-" + strconv.FormatInt(input.ID, 10)
		} else {
			input.Slug = "tag-" + time.Now().Format("20060102150405")
		}
	}
	return input
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)
var htmlTag = regexp.MustCompile(`<[^>]+>`)
var whitespace = regexp.MustCompile(`\s+`)
var searchSpace = strings.NewReplacer("\u00a0", " ", "\u3000", " ")

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = nonSlug.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func searchText(value string) string {
	value = htmlTag.ReplaceAllString(value, " ")
	value = stdhtml.UnescapeString(value)
	value = searchSpace.Replace(value)
	value = whitespace.ReplaceAllString(value, " ")
	value = strings.TrimSpace(value)
	if len([]rune(value)) <= 600 {
		return value
	}
	runes := []rune(value)
	return string(runes[:600])
}

func readingMinutes(value string) int {
	value = htmlTag.ReplaceAllString(value, " ")
	value = stdhtml.UnescapeString(value)
	value = searchSpace.Replace(value)
	value = whitespace.ReplaceAllString(value, " ")
	value = strings.TrimSpace(value)
	if value == "" {
		return 1
	}

	words := 0
	cjk := 0
	inWord := false
	for _, r := range value {
		switch {
		case isCJKRune(r):
			cjk++
			inWord = false
		case isLatinWordRune(r):
			if !inWord {
				words++
				inWord = true
			}
		default:
			inWord = false
		}
	}

	weightedWords := words + int(math.Ceil(float64(cjk)/2.0))
	if weightedWords < 1 {
		weightedWords = 1
	}
	return max(1, int(math.Ceil(float64(weightedWords)/220.0)))
}

func isLatinWordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '\'' || r == '-'
}

func isCJKRune(r rune) bool {
	return (r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0x3040 && r <= 0x30FF) ||
		(r >= 0xAC00 && r <= 0xD7AF)
}

func normalizeSearchQuery(value string) string {
	value = stdhtml.UnescapeString(value)
	value = searchSpace.Replace(value)
	value = whitespace.ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func likePattern(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return "%" + value + "%"
}
