package blog

import (
	"context"
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
	links    *store.LinkStore
	renderer *view.Renderer
}

type PageData struct {
	Site             config.Site
	Title            string
	Description      string
	CanonicalURL     string
	MetaImage        string
	MetaType         string
	SectionTitle     string
	Query            string
	Posts            []models.Post
	Post             models.Post
	PreviousPost     models.Post
	NextPost         models.Post
	ArchiveGroups    []models.ArchiveGroup
	LinkCategories   []models.FriendLinkCategory
	FeaturedImages   []string
	Comments         []models.Comment
	CommentOK        bool
	CommentErr       string
	Category         models.Category
	Tag              models.Tag
	SearchCategories []models.Category
	SearchTags       []models.Tag
	RecentPosts      []models.Post
	Categories       []models.Category
	Tags             []models.Tag
	PostTotal        int
	CommentTotal     int
	Page             models.PageInfo
	Notice           string
	Now              time.Time
	AdminLoggedIn    bool
	ShowAdminNav     bool
	ErrorCode        string
	ErrorHeading     string
	ErrorMessage     string
	ErrorAction      string
	ErrorActionURL   string
}

func New(cfg *config.Config, posts *store.PostStore, links *store.LinkStore, renderer *view.Renderer) *Controller {
	return &Controller{cfg: cfg, posts: posts, links: links, renderer: renderer}
}

func (c *Controller) Register(server *ghttp.Server) {
	server.BindHandler("GET:/", c.Home)
	server.BindHandler("GET:/post/{slug}", c.Post)
	server.BindHandler("POST:/post/{slug}/comments", c.CreateComment)
	server.BindHandler("GET:/archive", c.Archive)
	server.BindHandler("GET:/archives", c.Archives)
	server.BindHandler("GET:/links", c.Links)
	server.BindHandler("GET:/feed", c.Feed)
	server.BindHandler("GET:/feed.xml", c.Feed)
	server.BindHandler("HEAD:/feed", c.Feed)
	server.BindHandler("HEAD:/feed.xml", c.Feed)
	server.BindHandler("GET:/sitemap.xml", c.Sitemap)
	server.BindHandler("HEAD:/sitemap.xml", c.Sitemap)
	server.BindHandler("GET:/category/{slug}", c.Category)
	server.BindHandler("GET:/tag/{slug}", c.Tag)
	server.BindHandler("GET:/search", c.Search)
	server.BindHandler("GET:/api/posts", c.APIPosts)
	server.BindHandler("GET:/api/posts/{slug}", c.APIPost)
	server.BindHandler("GET:/api/search-index", c.APISearchIndex)
	server.BindHandler("GET:/api/cache_search/json", c.APISearchIndex)
	server.BindHandler("GET:/cache_search/json", c.APISearchIndex)
	server.BindHandler("GET:/wp-json/sakura/v1/cache_search/json", c.APISearchIndex)
	server.BindHandler("GET:/api/random-cover", c.APIRandomCover)
	server.BindHandler("GET:/api/random-feature", c.APIRandomFeature)
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
		Site:           c.cfg.GetSite(),
		Title:          c.cfg.GetSite().Name,
		Description:    c.cfg.GetSite().Description,
		SectionTitle:   "Latest Posts",
		Posts:          posts,
		FeaturedImages: c.randomFeatureImages(r.Context(), 3),
		Page:           store.PageInfo(page, pageSize, total, "/", ""),
		Notice:         c.cfg.GetSite().Notice,
		Now:            time.Now(),
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

func (c *Controller) Links(r *ghttp.Request) {
	categories, err := c.links.ListPublic(r.Context())
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "links.tmpl", PageData{
		Site:           c.cfg.GetSite(),
		Title:          "Links - " + c.cfg.GetSite().Name,
		Description:    "Friendly little doors around KoiMoe Diary.",
		SectionTitle:   "Links",
		LinkCategories: categories,
		Now:            time.Now(),
	})
}

func (c *Controller) Feed(r *ghttp.Request) {
	posts, err := c.posts.ListPublished(r.Context(), 20)
	if err != nil {
		c.apiError(r, err)
		return
	}
	site := c.cfg.GetSite()
	baseURL := requestBaseURL(r)
	updated := time.Now()
	if len(posts) > 0 {
		updated = posts[0].PublishedAt
	}
	feed := atomFeed{
		XMLName:  xml.Name{Local: "feed"},
		XMLNS:    "http://www.w3.org/2005/Atom",
		Title:    site.Name,
		Subtitle: site.Description,
		ID:       baseURL + "/",
		Updated:  updated.Format(time.RFC3339),
		Links: []atomLink{
			{Href: baseURL + "/", Rel: "alternate", Type: "text/html"},
			{Href: baseURL + "/feed", Rel: "self", Type: "application/atom+xml"},
		},
		Author: atomAuthor{Name: site.Author},
	}
	for _, post := range posts {
		feed.Entries = append(feed.Entries, atomEntry{
			Title:   post.Title,
			ID:      baseURL + "/post/" + post.Slug,
			Updated: post.PublishedAt.Format(time.RFC3339),
			Links: []atomLink{{
				Href: baseURL + "/post/" + post.Slug,
				Rel:  "alternate",
				Type: "text/html",
			}},
			Summary: atomText{
				Type: "html",
				Text: post.Excerpt,
			},
		})
	}

	output, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		c.apiError(r, err)
		return
	}
	r.Response.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	r.Response.Write([]byte(xml.Header))
	r.Response.Write(output)
}

func (c *Controller) Sitemap(r *ghttp.Request) {
	ctx := r.Context()
	posts, err := c.posts.ListPublished(ctx, 500)
	if err != nil {
		c.apiError(r, err)
		return
	}
	categories, err := c.posts.ListCategories(ctx)
	if err != nil {
		c.apiError(r, err)
		return
	}
	tags, err := c.posts.ListTags(ctx)
	if err != nil {
		c.apiError(r, err)
		return
	}

	baseURL := requestBaseURL(r)
	now := time.Now()
	latest := now
	if len(posts) > 0 {
		latest = posts[0].PublishedAt
	}
	sitemap := sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs: []sitemapURL{
			{Loc: baseURL + "/", LastMod: latest.Format("2006-01-02"), ChangeFreq: "daily", Priority: "1.0"},
			{Loc: baseURL + "/archives", LastMod: latest.Format("2006-01-02"), ChangeFreq: "weekly", Priority: "0.7"},
			{Loc: baseURL + "/links", LastMod: now.Format("2006-01-02"), ChangeFreq: "monthly", Priority: "0.5"},
			{Loc: baseURL + "/search", LastMod: now.Format("2006-01-02"), ChangeFreq: "monthly", Priority: "0.3"},
		},
	}
	for _, post := range posts {
		sitemap.URLs = append(sitemap.URLs, sitemapURL{
			Loc:        baseURL + "/post/" + post.Slug,
			LastMod:    post.PublishedAt.Format("2006-01-02"),
			ChangeFreq: "monthly",
			Priority:   "0.8",
		})
	}
	for _, category := range categories {
		sitemap.URLs = append(sitemap.URLs, sitemapURL{
			Loc:        baseURL + "/category/" + category.Slug,
			LastMod:    latest.Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   "0.6",
		})
	}
	for _, tag := range tags {
		sitemap.URLs = append(sitemap.URLs, sitemapURL{
			Loc:        baseURL + "/tag/" + tag.Slug,
			LastMod:    latest.Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   "0.5",
		})
	}

	output, err := xml.MarshalIndent(sitemap, "", "  ")
	if err != nil {
		c.apiError(r, err)
		return
	}
	r.Response.Header().Set("Content-Type", "application/xml; charset=utf-8")
	r.Response.Write([]byte(xml.Header))
	r.Response.Write(output)
}

func (c *Controller) Search(r *ghttp.Request) {
	q := strings.TrimSpace(r.GetQuery("q").String())
	page := currentPage(r)
	pageSize := 10
	var posts []models.Post
	var searchCategories []models.Category
	var searchTags []models.Tag
	total := 0
	if q != "" {
		var err error
		posts, err = c.posts.SearchPaged(r.Context(), q, page, pageSize)
		if err != nil {
			c.error(r, err)
			return
		}
		total, err = c.posts.CountSearch(r.Context(), q)
		if err != nil {
			c.error(r, err)
			return
		}
		searchCategories, err = c.posts.SearchCategories(r.Context(), q, 6)
		if err != nil {
			c.error(r, err)
			return
		}
		searchTags, err = c.posts.SearchTags(r.Context(), q, 8)
		if err != nil {
			c.error(r, err)
			return
		}
	}
	c.render(r, "search.tmpl", PageData{
		Site:             c.cfg.GetSite(),
		Title:            "Search - " + c.cfg.GetSite().Name,
		Description:      "Search posts, categories, and tags from " + c.cfg.GetSite().Name + ".",
		SectionTitle:     "Search",
		Query:            q,
		Posts:            posts,
		SearchCategories: searchCategories,
		SearchTags:       searchTags,
		Page:             store.PageInfo(page, pageSize, total, "/search", q),
		Now:              time.Now(),
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

func (c *Controller) APISearchIndex(r *ghttp.Request) {
	index, err := c.posts.SearchIndex(r.Context())
	if err != nil {
		c.apiError(r, err)
		return
	}
	r.Response.Header().Set("Cache-Control", "public, max-age=60")
	r.Response.WriteJson(index)
}

func (c *Controller) APIRandomCover(r *ghttp.Request) {
	images := c.coverImagePool()
	c.writeRandomImage(r, "cover", images)
}

func (c *Controller) APIRandomFeature(r *ghttp.Request) {
	images := c.featureImagePool(r.Context())
	c.writeRandomImage(r, "feature", images)
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
	c.renderStatus(r, http.StatusNotFound, "error.tmpl", PageData{
		Site:           c.cfg.GetSite(),
		Title:          "Not Found - " + c.cfg.GetSite().Name,
		Description:    "This page is resting somewhere outside KoiMoe Diary.",
		ErrorCode:      "404",
		ErrorHeading:   "The page slipped behind the petals.",
		ErrorMessage:   "This little place does not exist yet. It may have moved, or it may still be waiting to be written.",
		ErrorAction:    "Back to KoiMoe Diary",
		ErrorActionURL: "/",
		Now:            time.Now(),
	})
}

func (c *Controller) error(r *ghttp.Request, err error) {
	log.Println(err)
	c.renderStatus(r, http.StatusInternalServerError, "error.tmpl", PageData{
		Site:           c.cfg.GetSite(),
		Title:          "Something went softly wrong - " + c.cfg.GetSite().Name,
		Description:    "KoiMoe Diary could not finish this request.",
		ErrorCode:      "500",
		ErrorHeading:   "The diary lost its thread for a moment.",
		ErrorMessage:   "Something went wrong while opening this page. The server kept the details in its logs, so the public page can stay calm.",
		ErrorAction:    "Return home",
		ErrorActionURL: "/",
		Now:            time.Now(),
	})
}

func (c *Controller) apiError(r *ghttp.Request, err error) {
	log.Println(err)
	r.Response.WriteStatus(500, "Internal Server Error")
}

func (c *Controller) render(r *ghttp.Request, name string, data PageData) {
	c.withMeta(r, &data)
	c.withUserEntry(r, &data)
	c.withSidebar(r.Context(), &data)
	c.renderer.HTML(r, name, data)
}

func (c *Controller) renderStatus(r *ghttp.Request, status int, name string, data PageData) {
	c.withMeta(r, &data)
	c.withUserEntry(r, &data)
	c.withSidebar(r.Context(), &data)
	c.renderer.HTMLStatus(r, status, name, data)
}

func (c *Controller) withMeta(r *ghttp.Request, data *PageData) {
	site := c.cfg.GetSite()
	baseURL := requestBaseURL(r)
	path := r.URL.Path
	if path == "" {
		path = "/"
	}
	data.CanonicalURL = baseURL + path
	if strings.TrimSpace(data.Description) == "" {
		data.Description = site.Description
	}
	image := site.HeroImage
	if data.Post.ID > 0 && strings.TrimSpace(data.Post.CoverImage) != "" {
		image = data.Post.CoverImage
	} else if strings.TrimSpace(image) == "" {
		image = site.Avatar
	}
	data.MetaImage = absoluteURL(baseURL, image)
	if data.Post.ID > 0 {
		data.MetaType = "article"
	} else {
		data.MetaType = "website"
	}
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

func (c *Controller) withUserEntry(r *ghttp.Request, data *PageData) {
	data.AdminLoggedIn = c.isAdminLoggedIn(r)
	data.ShowAdminNav = true
	for _, item := range data.Site.Navigation {
		url := strings.TrimSpace(item.URL)
		if url == "/admin" || url == "/admin/login" {
			data.ShowAdminNav = false
			return
		}
	}
}

func (c *Controller) isAdminLoggedIn(r *ghttp.Request) bool {
	cookie := r.Cookie.Get("sakurairo_admin")
	if cookie == nil {
		return false
	}
	username, ok := c.verifyAdminToken(cookie.String())
	return ok && username == c.cfg.AdminUsername
}

func (c *Controller) verifyAdminToken(token string) (string, bool) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", false
	}
	parts := strings.Split(string(data), ":")
	if len(parts) != 3 {
		return "", false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", false
	}
	payload := parts[0] + ":" + parts[1]
	expected := c.adminSignature(payload)
	if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expected)) != 1 {
		return "", false
	}
	return parts[0], true
}

func (c *Controller) adminSignature(payload string) string {
	secret := c.cfg.AdminSecret
	if secret == "" {
		secret = c.cfg.AdminPassword
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func (c *Controller) coverImagePool() []string {
	site := c.cfg.GetSite()
	images := []string{
		site.HeroImage,
		site.Avatar,
		"/static/theme/screenshot.jpg",
		"/static/theme/content-image/d-1.jpg",
		"/static/theme/content-image/d-2.jpg",
		"/static/theme/content-image/d-3.jpg",
		"/static/theme/content-image/d-4.jpg",
	}
	images = append(images, c.curatedImages("originals")...)
	images = append(images, c.curatedImages("square")...)
	return compactImageURLs(images)
}

func (c *Controller) featureImagePool(ctx context.Context) []string {
	images, err := c.posts.DistinctCoverImages(ctx, 80)
	if err != nil {
		log.Printf("load feature images: %v", err)
	}
	images = append(images, c.curatedImages("square")...)
	if len(images) == 0 {
		images = c.coverImagePool()
	}
	return compactImageURLs(images)
}

func (c *Controller) randomFeatureImages(ctx context.Context, count int) []string {
	images := c.featureImagePool(ctx)
	if count <= 0 {
		return nil
	}
	if len(images) < count {
		images = compactImageURLs(append(images, c.coverImagePool()...))
	}
	if len(images) == 0 {
		images = []string{
			"/static/theme/content-image/d-1.jpg",
			"/static/theme/content-image/d-2.jpg",
			"/static/theme/content-image/d-3.jpg",
		}
	}
	result := make([]string, 0, count)
	for len(result) < count && len(images) > 0 {
		index := randomIndex(len(images))
		result = append(result, images[index])
		images = append(images[:index], images[index+1:]...)
	}
	for len(result) < count {
		result = append(result, result[len(result)%len(result)])
	}
	return result
}

func (c *Controller) curatedImages(section string) []string {
	dir := filepath.Join(c.cfg.StaticDir, "curated-sakura-images", section)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	images := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isImageFile(entry.Name()) {
			continue
		}
		images = append(images, "/static/curated-sakura-images/"+section+"/"+entry.Name())
	}
	return images
}

func (c *Controller) writeRandomImage(r *ghttp.Request, kind string, images []string) {
	images = compactImageURLs(images)
	if len(images) == 0 {
		r.Response.WriteStatus(404, "No images")
		return
	}
	image := images[randomIndex(len(images))]
	r.Response.Header().Set("Cache-Control", "no-store")
	if r.GetQuery("format").String() == "json" {
		r.Response.WriteJson(map[string]any{
			"kind":  kind,
			"url":   image,
			"count": len(images),
		})
		return
	}
	r.Response.RedirectTo(image, http.StatusFound)
}

func compactImageURLs(images []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(images))
	for _, image := range images {
		image = strings.TrimSpace(image)
		if image == "" || seen[image] {
			continue
		}
		seen[image] = true
		result = append(result, image)
	}
	return result
}

func isImageFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return true
	default:
		return false
	}
}

func randomIndex(length int) int {
	if length <= 1 {
		return 0
	}
	n, err := crand.Int(crand.Reader, big.NewInt(int64(length)))
	if err != nil {
		return int(time.Now().UnixNano() % int64(length))
	}
	return int(n.Int64())
}

func requestBaseURL(r *ghttp.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	if host == "" {
		host = "blog.koimoe.com"
	}
	return strings.TrimRight(scheme+"://"+host, "/")
}

func absoluteURL(baseURL string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return strings.TrimRight(baseURL, "/") + value
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

type atomFeed struct {
	XMLName  xml.Name    `xml:"feed"`
	XMLNS    string      `xml:"xmlns,attr"`
	Title    string      `xml:"title"`
	Subtitle string      `xml:"subtitle,omitempty"`
	ID       string      `xml:"id"`
	Updated  string      `xml:"updated"`
	Links    []atomLink  `xml:"link"`
	Author   atomAuthor  `xml:"author"`
	Entries  []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	ID      string     `xml:"id"`
	Updated string     `xml:"updated"`
	Links   []atomLink `xml:"link"`
	Summary atomText   `xml:"summary"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomText struct {
	Type string `xml:"type,attr,omitempty"`
	Text string `xml:",chardata"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}
