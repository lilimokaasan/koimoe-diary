package blog

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"

	"sakurairo-go/internal/config"
	"sakurairo-go/internal/models"
	"sakurairo-go/internal/store"
	"sakurairo-go/internal/view"
)

type Controller struct {
	cfg      *config.Config
	posts    *store.PostStore
	renderer *view.Renderer
}

type PageData struct {
	Site          config.Site
	Title         string
	Description   string
	SectionTitle  string
	Query         string
	Posts         []models.Post
	Post          models.Post
	PreviousPost  models.Post
	NextPost      models.Post
	ArchiveGroups []models.ArchiveGroup
	Comments      []models.Comment
	CommentOK     bool
	CommentErr    string
	Category      models.Category
	Tag           models.Tag
	RecentPosts   []models.Post
	Categories    []models.Category
	Tags          []models.Tag
	PostTotal     int
	CommentTotal  int
	Page          models.PageInfo
	Notice        string
	Now           time.Time
}

func New(cfg *config.Config, posts *store.PostStore, renderer *view.Renderer) *Controller {
	return &Controller{cfg: cfg, posts: posts, renderer: renderer}
}

func (c *Controller) Register(server *ghttp.Server) {
	server.BindHandler("GET:/", c.Home)
	server.BindHandler("GET:/post/{slug}", c.Post)
	server.BindHandler("POST:/post/{slug}/comments", c.CreateComment)
	server.BindHandler("GET:/archive", c.Archive)
	server.BindHandler("GET:/archives", c.Archives)
	server.BindHandler("GET:/category/{slug}", c.Category)
	server.BindHandler("GET:/tag/{slug}", c.Tag)
	server.BindHandler("GET:/search", c.Search)
	server.BindHandler("GET:/api/posts", c.APIPosts)
	server.BindHandler("GET:/api/posts/{slug}", c.APIPost)
	server.BindHandler("POST:/api/posts/{id}/like", c.LikePost)
	server.BindStatusHandler(404, c.NotFound)
}

func (c *Controller) Home(r *ghttp.Request) {
	page := currentPage(r)
	pageSize := 10
	posts, err := c.posts.ListPublishedPaged(r.Context(), page, pageSize)
	if err != nil {
		c.error(r, err)
		return
	}
	total, err := c.posts.CountPublished(r.Context())
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "home.tmpl", PageData{
		Site:         c.cfg.GetSite(),
		Title:        c.cfg.GetSite().Name,
		Description:  c.cfg.GetSite().Description,
		SectionTitle: "Latest Posts",
		Posts:        posts,
		Page:         store.PageInfo(page, pageSize, total, "/", ""),
		Notice:       c.cfg.GetSite().Notice,
		Now:          time.Now(),
	})
}

func (c *Controller) Post(r *ghttp.Request) {
	slug := r.GetRouter("slug").String()
	if slug == "" {
		c.NotFound(r)
		return
	}

	post, err := c.posts.BySlug(r.Context(), slug)
	if errors.Is(err, sql.ErrNoRows) {
		c.NotFound(r)
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	if err := c.posts.IncrementViews(r.Context(), post.ID); err != nil {
		log.Printf("increment views: %v", err)
	}
	comments, err := c.posts.ListComments(r.Context(), post.ID)
	if err != nil {
		c.error(r, err)
		return
	}
	previousPost, nextPost, err := c.posts.AdjacentPublished(r.Context(), post)
	if err != nil {
		log.Printf("load adjacent posts: %v", err)
	}

	c.render(r, "post.tmpl", PageData{
		Site:         c.cfg.GetSite(),
		Title:        post.Title + " - " + c.cfg.GetSite().Name,
		Description:  post.Excerpt,
		Post:         post,
		PreviousPost: previousPost,
		NextPost:     nextPost,
		Comments:     comments,
		CommentOK:    r.GetQuery("comment").String() == "ok",
		Now:          time.Now(),
	})
}

func (c *Controller) CreateComment(r *ghttp.Request) {
	slug := r.GetRouter("slug").String()
	post, err := c.posts.BySlug(r.Context(), slug)
	if errors.Is(err, sql.ErrNoRows) {
		c.NotFound(r)
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}

	if strings.TrimSpace(r.GetForm("homepage").String()) != "" {
		r.Response.RedirectTo("/post/"+slug, http.StatusSeeOther)
		return
	}

	comment := models.Comment{
		PostID:  post.ID,
		Author:  strings.TrimSpace(r.GetForm("author").String()),
		Email:   strings.TrimSpace(r.GetForm("email").String()),
		Website: strings.TrimSpace(r.GetForm("website").String()),
		Content: strings.TrimSpace(r.GetForm("content").String()),
	}
	if errText := validateComment(comment); errText != "" {
		comments, listErr := c.posts.ListComments(r.Context(), post.ID)
		if listErr != nil {
			c.error(r, listErr)
			return
		}
		c.render(r, "post.tmpl", PageData{
			Site:        c.cfg.GetSite(),
			Title:       post.Title + " - " + c.cfg.GetSite().Name,
			Description: post.Excerpt,
			Post:        post,
			Comments:    comments,
			CommentErr:  errText,
			Now:         time.Now(),
		})
		return
	}

	if err := c.posts.CreateComment(r.Context(), comment, r.GetClientIp(), r.UserAgent()); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/post/"+slug+"?comment=ok#comments", http.StatusSeeOther)
}

func (c *Controller) Archive(r *ghttp.Request) {
	page := currentPage(r)
	pageSize := 10
	posts, err := c.posts.ListPublishedPaged(r.Context(), page, pageSize)
	if err != nil {
		c.error(r, err)
		return
	}
	total, err := c.posts.CountPublished(r.Context())
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "archive.tmpl", PageData{
		Site:         c.cfg.GetSite(),
		Title:        "Archive - " + c.cfg.GetSite().Name,
		Description:  c.cfg.GetSite().Description,
		SectionTitle: "Archive",
		Posts:        posts,
		Page:         store.PageInfo(page, pageSize, total, "/archive", ""),
		Now:          time.Now(),
	})
}

func (c *Controller) Archives(r *ghttp.Request) {
	groups, err := c.posts.ArchiveGroups(r.Context())
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "archives.tmpl", PageData{
		Site:          c.cfg.GetSite(),
		Title:         "Archives - " + c.cfg.GetSite().Name,
		Description:   c.cfg.GetSite().Description,
		SectionTitle:  "Archives",
		ArchiveGroups: groups,
		Now:           time.Now(),
	})
}

func (c *Controller) Search(r *ghttp.Request) {
	q := r.GetQuery("q").String()
	page := currentPage(r)
	pageSize := 10
	posts, err := c.posts.SearchPaged(r.Context(), q, page, pageSize)
	if err != nil {
		c.error(r, err)
		return
	}
	total, err := c.posts.CountSearch(r.Context(), q)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "search.tmpl", PageData{
		Site:         c.cfg.GetSite(),
		Title:        "Search - " + c.cfg.GetSite().Name,
		Description:  c.cfg.GetSite().Description,
		SectionTitle: "Search",
		Query:        q,
		Posts:        posts,
		Page:         store.PageInfo(page, pageSize, total, "/search", q),
		Now:          time.Now(),
	})
}

func (c *Controller) Category(r *ghttp.Request) {
	slug := r.GetRouter("slug").String()
	page := currentPage(r)
	pageSize := 10
	posts, category, err := c.posts.ByCategory(r.Context(), slug, page, pageSize)
	if errors.Is(err, sql.ErrNoRows) {
		c.NotFound(r)
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	total, err := c.posts.CountByCategory(r.Context(), slug)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "archive.tmpl", PageData{
		Site:         c.cfg.GetSite(),
		Title:        category.Name + " - " + c.cfg.GetSite().Name,
		Description:  category.Description,
		SectionTitle: "Category: " + category.Name,
		Category:     category,
		Posts:        posts,
		Page:         store.PageInfo(page, pageSize, total, "/category/"+slug, ""),
		Now:          time.Now(),
	})
}

func (c *Controller) Tag(r *ghttp.Request) {
	slug := r.GetRouter("slug").String()
	page := currentPage(r)
	pageSize := 10
	posts, tag, err := c.posts.ByTag(r.Context(), slug, page, pageSize)
	if errors.Is(err, sql.ErrNoRows) {
		c.NotFound(r)
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	total, err := c.posts.CountByTag(r.Context(), slug)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "archive.tmpl", PageData{
		Site:         c.cfg.GetSite(),
		Title:        tag.Name + " - " + c.cfg.GetSite().Name,
		Description:  c.cfg.GetSite().Description,
		SectionTitle: "Tag: " + tag.Name,
		Tag:          tag,
		Posts:        posts,
		Page:         store.PageInfo(page, pageSize, total, "/tag/"+slug, ""),
		Now:          time.Now(),
	})
}

func (c *Controller) APIPosts(r *ghttp.Request) {
	posts, err := c.posts.ListPublished(r.Context(), 50)
	if err != nil {
		c.apiError(r, err)
		return
	}
	r.Response.WriteJson(posts)
}

func (c *Controller) APIPost(r *ghttp.Request) {
	slug := r.GetRouter("slug").String()
	post, err := c.posts.BySlug(r.Context(), slug)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.apiError(r, err)
		return
	}
	r.Response.WriteJson(post)
}

func (c *Controller) LikePost(r *ghttp.Request) {
	id := r.GetRouter("id").Int64()
	if id <= 0 {
		r.Response.WriteStatus(400, "Bad Request")
		return
	}
	likes, err := c.posts.IncrementLikes(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.apiError(r, err)
		return
	}
	r.Response.WriteJson(map[string]any{
		"ok":    true,
		"likes": likes,
	})
}

func (c *Controller) NotFound(r *ghttp.Request) {
	r.Response.WriteHeader(404)
	c.render(r, "404.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "Not Found - " + c.cfg.GetSite().Name,
		Now:   time.Now(),
	})
}

func (c *Controller) error(r *ghttp.Request, err error) {
	log.Println(err)
	r.Response.WriteStatus(500, "Internal Server Error")
}

func (c *Controller) apiError(r *ghttp.Request, err error) {
	log.Println(err)
	r.Response.WriteStatus(500, "Internal Server Error")
}

func (c *Controller) render(r *ghttp.Request, name string, data PageData) {
	c.withSidebar(r.Context(), &data)
	c.renderer.HTML(r, name, data)
}

func (c *Controller) withSidebar(ctx context.Context, data *PageData) {
	var err error
	data.RecentPosts, err = c.posts.ListRecent(ctx, 5)
	if err != nil {
		log.Printf("load recent posts: %v", err)
	}
	data.Categories, err = c.posts.ListCategories(ctx)
	if err != nil {
		log.Printf("load categories: %v", err)
	}
	data.Tags, err = c.posts.ListTags(ctx)
	if err != nil {
		log.Printf("load tags: %v", err)
	}
	data.PostTotal, err = c.posts.CountPublished(ctx)
	if err != nil {
		log.Printf("count posts: %v", err)
	}
	data.CommentTotal, err = c.posts.CountComments(ctx)
	if err != nil {
		log.Printf("count comments: %v", err)
	}
}

func currentPage(r *ghttp.Request) int {
	page := r.GetQuery("page").Int()
	if page < 1 {
		return 1
	}
	return page
}

func validateComment(comment models.Comment) string {
	switch {
	case comment.Author == "":
		return "Name is required."
	case comment.Email == "":
		return "Email is required."
	case !strings.Contains(comment.Email, "@"):
		return "Email looks invalid."
	case comment.Content == "":
		return "Comment content is required."
	case len([]rune(comment.Author)) > 80:
		return "Name is too long."
	case len([]rune(comment.Content)) > 2000:
		return "Comment is too long."
	default:
		return ""
	}
}
