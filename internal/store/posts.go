package store

import (
	"context"
	"database/sql"
	stdhtml "html"
	"html/template"
	"math"
	"regexp"
	"strings"
	"time"

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
	CategoryName string
	Tags         []string
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
	views BIGINT NOT NULL DEFAULT 0,
	likes BIGINT NOT NULL DEFAULT 0,
	published_at DATETIME NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX idx_posts_status_published (status, published_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`); err != nil {
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS tags (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	slug VARCHAR(160) NOT NULL UNIQUE,
	name VARCHAR(120) NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS post_tags (
	post_id BIGINT NOT NULL,
	tag_id BIGINT NOT NULL,
	PRIMARY KEY (post_id, tag_id),
	INDEX idx_post_tags_tag (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`); err != nil {
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
	ip VARCHAR(64) NOT NULL DEFAULT '',
	user_agent VARCHAR(255) NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	INDEX idx_comments_post_status_created (post_id, status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`); err != nil {
		return err
	}
	return s.ensurePostColumns()
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

func (s *PostStore) ListAll(ctx context.Context, limit int) ([]models.Post, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.status,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
ORDER BY p.updated_at DESC
LIMIT ?`, limit)
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

func (s *PostStore) ListPublishedPaged(ctx context.Context, page int, pageSize int) ([]models.Post, error) {
	page, pageSize = normalizePage(page, pageSize)
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published'
ORDER BY p.published_at DESC
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
	if query == "" {
		return s.ListPublishedPaged(ctx, page, pageSize)
	}
	page, pageSize = normalizePage(page, pageSize)
	like := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND (p.title LIKE ? OR p.excerpt LIKE ? OR p.content_html LIKE ?)
ORDER BY p.published_at DESC
LIMIT ? OFFSET ?`, like, like, like, pageSize, (page-1)*pageSize)
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

func (s *PostStore) ByCategory(ctx context.Context, slug string, page int, pageSize int) ([]models.Post, models.Category, error) {
	page, pageSize = normalizePage(page, pageSize)
	category, err := s.CategoryBySlug(ctx, slug)
	if err != nil {
		return nil, models.Category{}, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       c.id, c.slug, c.name, c.description
FROM posts p
JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND c.slug = ?
ORDER BY p.published_at DESC
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
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
JOIN post_tags pt ON pt.post_id = p.id
JOIN tags t ON t.id = pt.tag_id
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND t.slug = ?
ORDER BY p.published_at DESC
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
	return s.count(ctx, `SELECT COUNT(*) FROM posts WHERE status = 'published'`)
}

func (s *PostStore) CountSearch(ctx context.Context, query string) (int, error) {
	if query == "" {
		return s.CountPublished(ctx)
	}
	like := "%" + query + "%"
	return s.count(ctx, `SELECT COUNT(*) FROM posts WHERE status = 'published' AND (title LIKE ? OR excerpt LIKE ? OR content_html LIKE ?)`, like, like, like)
}

func (s *PostStore) CountByCategory(ctx context.Context, slug string) (int, error) {
	return s.count(ctx, `
SELECT COUNT(*)
FROM posts p
JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND c.slug = ?`, slug)
}

func (s *PostStore) CountByTag(ctx context.Context, slug string) (int, error) {
	return s.count(ctx, `
SELECT COUNT(*)
FROM posts p
JOIN post_tags pt ON pt.post_id = p.id
JOIN tags t ON t.id = pt.tag_id
WHERE p.status = 'published' AND t.slug = ?`, slug)
}

func (s *PostStore) ArchiveGroups(ctx context.Context) ([]models.ArchiveGroup, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published'
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
	var post models.Post
	var content string
	err := s.db.QueryRowContext(ctx, `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.slug = ? AND p.status = 'published'
LIMIT 1`, slug).Scan(
		&post.ID, &post.Slug, &post.Title, &post.Excerpt, &content, &post.CoverImage,
		&post.CommentCount, &post.Views, &post.Likes, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
		&post.Category.ID, &post.Category.Slug, &post.Category.Name, &post.Category.Description,
	)
	post.ContentHTML = template.HTML(content)
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
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image, p.status,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.id = ?
LIMIT 1`, id).Scan(
		&post.ID, &post.Slug, &post.Title, &post.Excerpt, &content, &post.CoverImage, &post.Status,
		&post.CommentCount, &post.Views, &post.Likes, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
		&post.Category.ID, &post.Category.Slug, &post.Category.Name, &post.Category.Description,
	)
	post.ContentHTML = template.HTML(content)
	if err == nil {
		posts := []models.Post{post}
		err = s.hydrateTags(ctx, posts)
		post = posts[0]
	}
	return post, err
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

func (s *PostStore) CategoryBySlug(ctx context.Context, slug string) (models.Category, error) {
	var category models.Category
	err := s.db.QueryRowContext(ctx, `
SELECT id, slug, name, description
FROM categories
WHERE slug = ?
LIMIT 1`, slug).Scan(&category.ID, &category.Slug, &category.Name, &category.Description)
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
LEFT JOIN posts p ON p.category_id = c.id AND p.status = 'published'
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
JOIN posts p ON p.id = pt.post_id AND p.status = 'published'
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

	index := models.SearchIndex{
		GeneratedAt: time.Now(),
		Posts:       make([]models.SearchPostItem, 0, len(posts)),
		Categories:  make([]models.SearchTaxonomyItem, 0, len(categories)),
		Tags:        make([]models.SearchTaxonomyItem, 0, len(tags)),
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

func (s *PostStore) IncrementViews(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE posts SET views = views + 1 WHERE id = ?`, id)
	return err
}

func (s *PostStore) IncrementLikes(ctx context.Context, id int64) (int64, error) {
	result, err := s.db.ExecContext(ctx, `UPDATE posts SET likes = likes + 1 WHERE id = ? AND status = 'published'`, id)
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
SELECT id, post_id, author, email, website, content, status, created_at
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
			&comment.ID, &comment.PostID, &comment.Author, &comment.Email,
			&comment.Website, &comment.Content, &comment.Status, &comment.CreatedAt,
		); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

func (s *PostStore) ListAllComments(ctx context.Context, limit int) ([]models.Comment, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT cm.id, cm.post_id, p.title, p.slug, cm.author, cm.email, cm.website, cm.content, cm.status, cm.created_at
FROM comments cm
JOIN posts p ON p.id = cm.post_id
ORDER BY cm.created_at DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(
			&comment.ID, &comment.PostID, &comment.PostTitle, &comment.PostSlug,
			&comment.Author, &comment.Email, &comment.Website, &comment.Content,
			&comment.Status, &comment.CreatedAt,
		); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

func (s *PostStore) CreateComment(ctx context.Context, comment models.Comment, ip string, userAgent string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO comments (post_id, author, email, website, content, status, ip, user_agent)
VALUES (?, ?, ?, ?, ?, 'approved', ?, ?)`,
		comment.PostID, comment.Author, comment.Email, comment.Website, comment.Content, ip, userAgent,
	)
	return err
}

func (s *PostStore) UpdateCommentStatus(ctx context.Context, id int64, status string) error {
	if status != "approved" {
		status = "hidden"
	}
	_, err := s.db.ExecContext(ctx, `UPDATE comments SET status = ? WHERE id = ?`, status, id)
	return err
}

func (s *PostStore) DeleteComment(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM comments WHERE id = ?`, id)
	return err
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

	now := time.Now()
	postID := input.ID
	if input.ID == 0 {
		result, execErr := tx.ExecContext(ctx, `
INSERT INTO posts (slug, title, excerpt, content_html, cover_image, category_id, status, published_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			input.Slug, input.Title, input.Excerpt, input.ContentHTML, input.CoverImage, categoryID, input.Status, now,
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
SET slug = ?, title = ?, excerpt = ?, content_html = ?, cover_image = ?, category_id = ?, status = ?
WHERE id = ?`,
			input.Slug, input.Title, input.Excerpt, input.ContentHTML, input.CoverImage, categoryID, input.Status, input.ID,
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

func scanAdminPosts(rows *sql.Rows) ([]models.Post, error) {
	var posts []models.Post
	for rows.Next() {
		var post models.Post
		var content string
		if err := rows.Scan(
			&post.ID, &post.Slug, &post.Title, &post.Excerpt, &content, &post.CoverImage, &post.Status,
			&post.CommentCount, &post.Views, &post.Likes, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
			&post.Category.ID, &post.Category.Slug, &post.Category.Name, &post.Category.Description,
		); err != nil {
			return nil, err
		}
		post.ContentHTML = template.HTML(content)
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
			&post.CommentCount, &post.Views, &post.Likes, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt,
			&post.Category.ID, &post.Category.Slug, &post.Category.Name, &post.Category.Description,
		); err != nil {
			return nil, err
		}
		post.ContentHTML = template.HTML(content)
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (s *PostStore) adjacentPublished(ctx context.Context, post models.Post, comparator string, direction string) (models.Post, error) {
	var adjacent models.Post
	var content string
	query := `
SELECT p.id, p.slug, p.title, p.excerpt, p.content_html, p.cover_image,
       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.status = 'approved') AS comment_count,
       p.views, p.likes, p.published_at, p.created_at, p.updated_at,
       COALESCE(c.id, 0), COALESCE(c.slug, ''), COALESCE(c.name, ''), COALESCE(c.description, '')
FROM posts p
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.status = 'published' AND p.published_at ` + comparator + ` ?
ORDER BY p.published_at ` + direction + `
LIMIT 1`
	err := s.db.QueryRowContext(ctx, query, post.PublishedAt).Scan(
		&adjacent.ID, &adjacent.Slug, &adjacent.Title, &adjacent.Excerpt, &content, &adjacent.CoverImage,
		&adjacent.CommentCount, &adjacent.Views, &adjacent.Likes, &adjacent.PublishedAt, &adjacent.CreatedAt, &adjacent.UpdatedAt,
		&adjacent.Category.ID, &adjacent.Category.Slug, &adjacent.Category.Name, &adjacent.Category.Description,
	)
	adjacent.ContentHTML = template.HTML(content)
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
	return s.ensureColumn("posts", "likes", `ALTER TABLE posts ADD COLUMN likes BIGINT NOT NULL DEFAULT 0 AFTER views`)
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
	if input.Status != "draft" {
		input.Status = "published"
	}
	input.CategoryName = strings.TrimSpace(input.CategoryName)
	if input.CategoryName == "" {
		input.CategoryName = "Blog"
	}
	return input
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)
var htmlTag = regexp.MustCompile(`<[^>]+>`)
var whitespace = regexp.MustCompile(`\s+`)

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = nonSlug.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func searchText(value string) string {
	value = htmlTag.ReplaceAllString(value, " ")
	value = stdhtml.UnescapeString(value)
	value = whitespace.ReplaceAllString(value, " ")
	value = strings.TrimSpace(value)
	if len([]rune(value)) <= 600 {
		return value
	}
	runes := []rune(value)
	return string(runes[:600])
}
