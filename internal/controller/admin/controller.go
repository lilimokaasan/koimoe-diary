package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
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
	settings *store.SettingsStore
	links    *store.LinkStore
	moments  *store.MomentStore
	renderer *view.Renderer
}

type PageData struct {
	Site          config.Site
	Title         string
	Description   string
	CanonicalURL  string
	MetaImage     string
	MetaType      string
	Error         string
	Message       string
	Posts         []models.Post
	Comments      []models.Comment
	Post          models.Post
	FriendLinks   []models.FriendLink
	FriendLink    models.FriendLink
	Moments       []models.Moment
	Moment        models.Moment
	Categories    []models.Category
	Category      models.Category
	Tags          []models.Tag
	Tag           models.Tag
	PreviousPost  models.Post
	NextPost      models.Post
	RecentPosts   []models.Post
	PostTotal     int
	CommentTotal  int
	Settings      config.Site
	Navigation    string
	FocusCards    string
	ContentHTML   string
	PostTags      string
	IsNew         bool
	Now           time.Time
	AdminLoggedIn bool
	ShowAdminNav  bool
}

func New(cfg *config.Config, posts *store.PostStore, settings *store.SettingsStore, links *store.LinkStore, moments *store.MomentStore, renderer *view.Renderer) *Controller {
	return &Controller{cfg: cfg, posts: posts, settings: settings, links: links, moments: moments, renderer: renderer}
}

func (c *Controller) Register(server *ghttp.Server) {
	server.BindHandler("GET:/admin", c.Dashboard)
	server.BindHandler("GET:/admin/login", c.Login)
	server.BindHandler("POST:/admin/login", c.LoginPost)
	server.BindHandler("POST:/admin/logout", c.Logout)
	server.BindHandler("GET:/admin/comments", c.Comments)
	server.BindHandler("POST:/admin/comments/{id}/status", c.UpdateCommentStatus)
	server.BindHandler("POST:/admin/comments/{id}/private", c.UpdateCommentPrivacy)
	server.BindHandler("POST:/admin/comments/{id}/delete", c.DeleteComment)
	server.BindHandler("GET:/admin/settings", c.Settings)
	server.BindHandler("POST:/admin/settings", c.SaveSettings)
	server.BindHandler("GET:/admin/links", c.Links)
	server.BindHandler("GET:/admin/links/new", c.NewLink)
	server.BindHandler("POST:/admin/links", c.SaveLink)
	server.BindHandler("GET:/admin/links/{id}/edit", c.EditLink)
	server.BindHandler("POST:/admin/links/{id}", c.SaveLink)
	server.BindHandler("POST:/admin/links/{id}/delete", c.DeleteLink)
	server.BindHandler("GET:/admin/moments", c.Moments)
	server.BindHandler("GET:/admin/moments/new", c.NewMoment)
	server.BindHandler("POST:/admin/moments", c.SaveMoment)
	server.BindHandler("GET:/admin/moments/{id}/edit", c.EditMoment)
	server.BindHandler("POST:/admin/moments/{id}", c.SaveMoment)
	server.BindHandler("POST:/admin/moments/{id}/delete", c.DeleteMoment)
	server.BindHandler("GET:/admin/taxonomy", c.Taxonomy)
	server.BindHandler("GET:/admin/categories/new", c.NewCategory)
	server.BindHandler("POST:/admin/categories", c.SaveCategory)
	server.BindHandler("GET:/admin/categories/{id}/edit", c.EditCategory)
	server.BindHandler("POST:/admin/categories/{id}", c.SaveCategory)
	server.BindHandler("POST:/admin/categories/{id}/delete", c.DeleteCategory)
	server.BindHandler("GET:/admin/tags/new", c.NewTag)
	server.BindHandler("POST:/admin/tags", c.SaveTag)
	server.BindHandler("GET:/admin/tags/{id}/edit", c.EditTag)
	server.BindHandler("POST:/admin/tags/{id}", c.SaveTag)
	server.BindHandler("POST:/admin/tags/{id}/delete", c.DeleteTag)
	server.BindHandler("GET:/admin/posts/new", c.NewPost)
	server.BindHandler("POST:/admin/posts", c.SavePost)
	server.BindHandler("GET:/admin/posts/{id}/edit", c.EditPost)
	server.BindHandler("GET:/admin/posts/{id}/preview", c.PreviewPost)
	server.BindHandler("POST:/admin/posts/{id}", c.SavePost)
}

func (c *Controller) Settings(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_settings.tmpl", PageData{
		Site:     c.cfg.GetSite(),
		Title:    "Settings - " + c.cfg.GetSite().Name,
		Message:  r.GetQuery("saved").String(),
		Settings: c.cfg.GetSite(),
		Navigation: formatNavigation(
			c.cfg.GetSite().Navigation,
		),
		FocusCards: formatFocusCards(
			c.cfg.GetSite().FocusCards,
		),
		Now: time.Now(),
	})
}

func (c *Controller) SaveSettings(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	site := config.Site{
		Name:               strings.TrimSpace(r.GetForm("site_name").String()),
		Description:        strings.TrimSpace(r.GetForm("site_description").String()),
		Author:             strings.TrimSpace(r.GetForm("site_author").String()),
		Notice:             strings.TrimSpace(r.GetForm("site_notice").String()),
		ThemeColor:         strings.TrimSpace(r.GetForm("theme_color").String()),
		HeroImage:          strings.TrimSpace(r.GetForm("hero_image").String()),
		HeroOverlayOpacity: strings.TrimSpace(r.GetForm("hero_overlay_opacity").String()),
		Avatar:             strings.TrimSpace(r.GetForm("site_avatar").String()),
		DefaultPostCover:   strings.TrimSpace(r.GetForm("default_post_cover").String()),
		SakuraEffects:      strings.TrimSpace(r.GetForm("sakura_effects").String()),
		FooterText:         strings.TrimSpace(r.GetForm("footer_text").String()),
		FooterCredit:       strings.TrimSpace(r.GetForm("footer_credit").String()),
		Navigation:         parseNavigation(r.GetForm("navigation").String()),
		FocusCards:         parseFocusCards(r.GetForm("focus_cards").String()),
	}
	site = normalizeSiteSettings(site, c.cfg.GetSite())
	if uploadedAvatar, uploadErr := c.saveImageUpload(r, "avatar_upload", "Avatar"); uploadErr != "" {
		c.render(r, "admin_settings.tmpl", PageData{
			Site:       c.cfg.GetSite(),
			Title:      "Settings - " + c.cfg.GetSite().Name,
			Error:      uploadErr,
			Settings:   site,
			Navigation: formatNavigation(site.Navigation),
			FocusCards: formatFocusCards(site.FocusCards),
			Now:        time.Now(),
		})
		return
	} else if uploadedAvatar != "" {
		site.Avatar = uploadedAvatar
	}
	if err := c.settings.SaveSite(r.Context(), site); err != nil {
		c.error(r, err)
		return
	}
	c.cfg.SetSite(site)
	r.Response.RedirectTo("/admin/settings?saved=1", http.StatusSeeOther)
}

func (c *Controller) Login(r *ghttp.Request) {
	if c.isLoggedIn(r) {
		r.Response.RedirectTo("/admin", http.StatusSeeOther)
		return
	}
	c.render(r, "admin_login.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "Admin Login - " + c.cfg.GetSite().Name,
		Error: r.GetQuery("error").String(),
		Now:   time.Now(),
	})
}

func (c *Controller) LoginPost(r *ghttp.Request) {
	if c.cfg.AdminPassword == "" {
		c.render(r, "admin_login.tmpl", PageData{
			Site:  c.cfg.GetSite(),
			Title: "Admin Login - " + c.cfg.GetSite().Name,
			Error: "Admin password is not configured.",
			Now:   time.Now(),
		})
		return
	}
	username := strings.TrimSpace(r.GetForm("username").String())
	password := r.GetForm("password").String()
	if username != c.cfg.AdminUsername || subtle.ConstantTimeCompare([]byte(password), []byte(c.cfg.AdminPassword)) != 1 {
		c.render(r, "admin_login.tmpl", PageData{
			Site:  c.cfg.GetSite(),
			Title: "Admin Login - " + c.cfg.GetSite().Name,
			Error: "Invalid username or password.",
			Now:   time.Now(),
		})
		return
	}
	r.Cookie.SetCookie("sakurairo_admin", c.signToken(username, time.Now().Add(24*time.Hour)), "", "/", 24*time.Hour, ghttp.CookieOptions{
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	r.Response.RedirectTo("/admin", http.StatusSeeOther)
}

func (c *Controller) Logout(r *ghttp.Request) {
	c.clearAdminCookies(r)
	r.Response.RedirectTo("/admin/login", http.StatusSeeOther)
}

func (c *Controller) clearAdminCookies(r *ghttp.Request) {
	expiredAt := time.Unix(0, 0).UTC()
	for _, domain := range []string{"", "blog.koimoe.com", ".koimoe.com"} {
		for _, path := range []string{"/", "/admin"} {
			http.SetCookie(r.Response.Writer, &http.Cookie{
				Name:     "sakurairo_admin",
				Value:    "",
				Path:     path,
				Domain:   domain,
				Expires:  expiredAt,
				MaxAge:   -1,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}
	}
}

func (c *Controller) Dashboard(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	posts, err := c.posts.ListAll(r.Context(), 100)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_posts.tmpl", PageData{
		Site:    c.cfg.GetSite(),
		Title:   "Posts - " + c.cfg.GetSite().Name,
		Message: r.GetQuery("saved").String(),
		Posts:   posts,
		Now:     time.Now(),
	})
}

func (c *Controller) Comments(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	comments, err := c.posts.ListAllComments(r.Context(), 200)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_comments.tmpl", PageData{
		Site:     c.cfg.GetSite(),
		Title:    "Comments - " + c.cfg.GetSite().Name,
		Message:  r.GetQuery("saved").String(),
		Comments: comments,
		Now:      time.Now(),
	})
}

func (c *Controller) UpdateCommentStatus(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	status := r.GetForm("status").String()
	if err := c.posts.UpdateCommentStatus(r.Context(), id, status); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/comments?saved=1", http.StatusSeeOther)
}

func (c *Controller) UpdateCommentPrivacy(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	isPrivate := r.GetForm("is_private").Bool()
	if err := c.posts.UpdateCommentPrivacy(r.Context(), id, isPrivate); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/comments?saved=1", http.StatusSeeOther)
}

func (c *Controller) DeleteComment(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	if err := c.posts.DeleteComment(r.Context(), id); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/comments?saved=1", http.StatusSeeOther)
}

func (c *Controller) Links(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	links, err := c.links.ListAdmin(r.Context())
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_links.tmpl", PageData{
		Site:        c.cfg.GetSite(),
		Title:       "Links - " + c.cfg.GetSite().Name,
		Message:     r.GetQuery("saved").String(),
		FriendLinks: links,
		Now:         time.Now(),
	})
}

func (c *Controller) NewLink(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_link_form.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "New Link - " + c.cfg.GetSite().Name,
		FriendLink: models.FriendLink{
			Category:  models.FriendLinkCategory{Name: "Friends"},
			Visible:   true,
			SortOrder: 0,
		},
		IsNew: true,
		Now:   time.Now(),
	})
}

func (c *Controller) EditLink(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	link, err := c.links.ByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_link_form.tmpl", PageData{
		Site:       c.cfg.GetSite(),
		Title:      "Edit Link - " + c.cfg.GetSite().Name,
		Message:    r.GetQuery("saved").String(),
		FriendLink: link,
		Now:        time.Now(),
	})
}

func (c *Controller) SaveLink(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	input := store.FriendLinkInput{
		ID:           id,
		CategoryName: r.GetForm("category").String(),
		Name:         r.GetForm("name").String(),
		URL:          r.GetForm("url").String(),
		Description:  r.GetForm("description").String(),
		ImageURL:     r.GetForm("image_url").String(),
		SortOrder:    r.GetForm("sort_order").Int(),
		Visible:      r.GetForm("visible").String() == "1",
	}
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.URL) == "" {
		c.render(r, "admin_link_form.tmpl", PageData{
			Site:       c.cfg.GetSite(),
			Title:      "Link Form - " + c.cfg.GetSite().Name,
			Error:      "Name and URL are required.",
			FriendLink: friendLinkFromInput(input),
			IsNew:      id == 0,
			Now:        time.Now(),
		})
		return
	}
	linkID, err := c.links.SaveLink(r.Context(), input)
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/links/"+strconv.FormatInt(linkID, 10)+"/edit?saved=1", http.StatusSeeOther)
}

func (c *Controller) DeleteLink(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	if err := c.links.DeleteLink(r.Context(), id); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/links?saved=1", http.StatusSeeOther)
}

func (c *Controller) Moments(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	moments, err := c.moments.ListAdmin(r.Context(), 200)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_moments.tmpl", PageData{
		Site:    c.cfg.GetSite(),
		Title:   "Moments - " + c.cfg.GetSite().Name,
		Message: r.GetQuery("saved").String(),
		Moments: moments,
		Now:     time.Now(),
	})
}

func (c *Controller) NewMoment(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_moment_form.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "New Moment - " + c.cfg.GetSite().Name,
		Moment: models.Moment{
			Author: c.cfg.GetSite().Author,
			Status: "published",
		},
		IsNew: true,
		Now:   time.Now(),
	})
}

func (c *Controller) EditMoment(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	moment, err := c.moments.ByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_moment_form.tmpl", PageData{
		Site:    c.cfg.GetSite(),
		Title:   "Edit Moment - " + c.cfg.GetSite().Name,
		Message: r.GetQuery("saved").String(),
		Moment:  moment,
		Now:     time.Now(),
	})
}

func (c *Controller) SaveMoment(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	input := store.MomentInput{
		ID:      id,
		Content: r.GetForm("content").String(),
		Author:  r.GetForm("author").String(),
		Status:  r.GetForm("status").String(),
	}
	if strings.TrimSpace(input.Author) == "" {
		input.Author = c.cfg.GetSite().Author
	}
	if strings.TrimSpace(input.Content) == "" {
		c.render(r, "admin_moment_form.tmpl", PageData{
			Site:   c.cfg.GetSite(),
			Title:  "Moment Form - " + c.cfg.GetSite().Name,
			Error:  "Content is required.",
			Moment: momentFromInput(input),
			IsNew:  id == 0,
			Now:    time.Now(),
		})
		return
	}
	momentID, err := c.moments.Save(r.Context(), input)
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/moments/"+strconv.FormatInt(momentID, 10)+"/edit?saved=1", http.StatusSeeOther)
}

func (c *Controller) DeleteMoment(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	if err := c.moments.Delete(r.Context(), id); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/moments?saved=1", http.StatusSeeOther)
}

func (c *Controller) Taxonomy(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	categories, err := c.posts.ListCategoriesAdmin(r.Context())
	if err != nil {
		c.error(r, err)
		return
	}
	tags, err := c.posts.ListTagsAdmin(r.Context())
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_taxonomy.tmpl", PageData{
		Site:       c.cfg.GetSite(),
		Title:      "Taxonomy - " + c.cfg.GetSite().Name,
		Message:    r.GetQuery("saved").String(),
		Categories: categories,
		Tags:       tags,
		Now:        time.Now(),
	})
}

func (c *Controller) NewCategory(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_category_form.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "New Category - " + c.cfg.GetSite().Name,
		IsNew: true,
		Now:   time.Now(),
	})
}

func (c *Controller) EditCategory(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	category, err := c.posts.CategoryByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_category_form.tmpl", PageData{
		Site:     c.cfg.GetSite(),
		Title:    "Edit Category - " + c.cfg.GetSite().Name,
		Message:  r.GetQuery("saved").String(),
		Category: category,
		Now:      time.Now(),
	})
}

func (c *Controller) SaveCategory(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	input := store.CategoryInput{
		ID:          id,
		Name:        r.GetForm("name").String(),
		Slug:        r.GetForm("slug").String(),
		Description: r.GetForm("description").String(),
		CoverImage:  r.GetForm("cover_image").String(),
	}
	if strings.TrimSpace(input.Name) == "" {
		c.render(r, "admin_category_form.tmpl", PageData{
			Site:     c.cfg.GetSite(),
			Title:    "Category Form - " + c.cfg.GetSite().Name,
			Error:    "Name is required.",
			Category: categoryFromInput(input),
			IsNew:    id == 0,
			Now:      time.Now(),
		})
		return
	}
	categoryID, err := c.posts.SaveCategory(r.Context(), input)
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/categories/"+strconv.FormatInt(categoryID, 10)+"/edit?saved=1", http.StatusSeeOther)
}

func (c *Controller) DeleteCategory(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	if err := c.posts.DeleteCategory(r.Context(), id); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/taxonomy?saved=1", http.StatusSeeOther)
}

func (c *Controller) NewTag(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_tag_form.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "New Tag - " + c.cfg.GetSite().Name,
		IsNew: true,
		Now:   time.Now(),
	})
}

func (c *Controller) EditTag(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	tag, err := c.posts.TagByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_tag_form.tmpl", PageData{
		Site:    c.cfg.GetSite(),
		Title:   "Edit Tag - " + c.cfg.GetSite().Name,
		Message: r.GetQuery("saved").String(),
		Tag:     tag,
		Now:     time.Now(),
	})
}

func (c *Controller) SaveTag(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	input := store.TagInput{
		ID:   id,
		Name: r.GetForm("name").String(),
		Slug: r.GetForm("slug").String(),
	}
	if strings.TrimSpace(input.Name) == "" {
		c.render(r, "admin_tag_form.tmpl", PageData{
			Site:  c.cfg.GetSite(),
			Title: "Tag Form - " + c.cfg.GetSite().Name,
			Error: "Name is required.",
			Tag:   tagFromInput(input),
			IsNew: id == 0,
			Now:   time.Now(),
		})
		return
	}
	tagID, err := c.posts.SaveTag(r.Context(), input)
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/tags/"+strconv.FormatInt(tagID, 10)+"/edit?saved=1", http.StatusSeeOther)
}

func (c *Controller) DeleteTag(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	if err := c.posts.DeleteTag(r.Context(), id); err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/taxonomy?saved=1", http.StatusSeeOther)
}

func (c *Controller) NewPost(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	c.render(r, "admin_post_form.tmpl", PageData{
		Site:  c.cfg.GetSite(),
		Title: "New Post - " + c.cfg.GetSite().Name,
		Post: models.Post{
			Status:     "published",
			CoverImage: "/static/theme/content-image/d-1.jpg",
			Category:   models.Category{Name: "Blog"},
		},
		IsNew: true,
		Now:   time.Now(),
	})
}

func (c *Controller) EditPost(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	post, err := c.posts.ByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	data := c.formData(post, "Edit Post - "+c.cfg.GetSite().Name, "")
	data.Message = r.GetQuery("saved").String()
	c.render(r, "admin_post_form.tmpl", data)
}

func (c *Controller) PreviewPost(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	post, err := c.posts.ByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		r.Response.WriteStatus(404, "Not Found")
		return
	}
	if err != nil {
		c.error(r, err)
		return
	}
	data, err := c.previewData(r, post)
	if err != nil {
		c.error(r, err)
		return
	}
	c.render(r, "admin_post_preview.tmpl", data)
}

func (c *Controller) SavePost(r *ghttp.Request) {
	if !c.requireLogin(r) {
		return
	}
	id := r.GetRouter("id").Int64()
	input := store.PostInput{
		ID:           id,
		Slug:         r.GetForm("slug").String(),
		Title:        r.GetForm("title").String(),
		Excerpt:      r.GetForm("excerpt").String(),
		ContentHTML:  r.GetForm("content_html").String(),
		CoverImage:   r.GetForm("cover_image").String(),
		Status:       r.GetForm("status").String(),
		CategoryName: r.GetForm("category").String(),
		Tags:         splitTags(r.GetForm("tags").String()),
	}
	if uploadedCover, uploadErr := c.saveCoverUpload(r); uploadErr != "" {
		post := postFromInput(input)
		c.render(r, "admin_post_form.tmpl", c.formData(post, "Post Form - "+c.cfg.GetSite().Name, uploadErr))
		return
	} else if uploadedCover != "" {
		input.CoverImage = uploadedCover
	}
	if strings.TrimSpace(input.CoverImage) == "" {
		input.CoverImage = c.cfg.GetSite().DefaultPostCover
	}
	if input.Title == "" || input.ContentHTML == "" {
		post := postFromInput(input)
		c.render(r, "admin_post_form.tmpl", c.formData(post, "Post Form - "+c.cfg.GetSite().Name, "Title and content are required."))
		return
	}
	postID, err := c.posts.SavePost(r.Context(), input)
	if err != nil {
		c.error(r, err)
		return
	}
	r.Response.RedirectTo("/admin/posts/"+strconv.FormatInt(postID, 10)+"/edit?saved=1", http.StatusSeeOther)
}

func (c *Controller) requireLogin(r *ghttp.Request) bool {
	if c.isLoggedIn(r) {
		return true
	}
	r.Response.RedirectTo("/admin/login", http.StatusSeeOther)
	return false
}

func (c *Controller) isLoggedIn(r *ghttp.Request) bool {
	cookie := r.Cookie.Get("sakurairo_admin")
	if cookie == nil {
		return false
	}
	username, ok := c.verifyToken(cookie.String())
	return ok && username == c.cfg.AdminUsername
}

func (c *Controller) signToken(username string, expires time.Time) string {
	payload := username + ":" + strconv.FormatInt(expires.Unix(), 10)
	signature := c.signature(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + ":" + signature))
}

func (c *Controller) verifyToken(token string) (string, bool) {
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
	expected := c.signature(payload)
	if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expected)) != 1 {
		return "", false
	}
	return parts[0], true
}

func (c *Controller) signature(payload string) string {
	secret := c.cfg.AdminSecret
	if secret == "" {
		secret = c.cfg.AdminPassword
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func (c *Controller) formData(post models.Post, title string, errText string) PageData {
	tags := make([]string, 0, len(post.Tags))
	for _, tag := range post.Tags {
		tags = append(tags, tag.Name)
	}
	return PageData{
		Site:        c.cfg.GetSite(),
		Title:       title,
		Error:       errText,
		Message:     "",
		Post:        post,
		ContentHTML: string(post.ContentHTML),
		PostTags:    strings.Join(tags, ", "),
		Now:         time.Now(),
	}
}

func (c *Controller) previewData(r *ghttp.Request, post models.Post) (PageData, error) {
	recentPosts, err := c.posts.ListRecent(r.Context(), 5)
	if err != nil {
		return PageData{}, err
	}
	categories, err := c.posts.ListCategories(r.Context())
	if err != nil {
		return PageData{}, err
	}
	tags, err := c.posts.ListTags(r.Context())
	if err != nil {
		return PageData{}, err
	}
	postTotal, err := c.posts.CountPublished(r.Context())
	if err != nil {
		return PageData{}, err
	}
	commentTotal, err := c.posts.CountComments(r.Context())
	if err != nil {
		return PageData{}, err
	}
	return PageData{
		Site:          c.cfg.GetSite(),
		Title:         "Preview: " + post.Title + " - " + c.cfg.GetSite().Name,
		Description:   post.Excerpt,
		CanonicalURL:  "/admin/posts/" + strconv.FormatInt(post.ID, 10) + "/preview",
		MetaImage:     post.CoverImage,
		MetaType:      "article",
		Post:          post,
		RecentPosts:   recentPosts,
		Categories:    categories,
		Tags:          tags,
		PostTotal:     postTotal,
		CommentTotal:  commentTotal,
		Now:           time.Now(),
		AdminLoggedIn: true,
		ShowAdminNav:  false,
	}, nil
}

func (c *Controller) saveCoverUpload(r *ghttp.Request) (string, string) {
	return c.saveImageUpload(r, "cover_upload", "Cover image")
}

func (c *Controller) saveImageUpload(r *ghttp.Request, field string, label string) (string, string) {
	file := r.GetUploadFile(field)
	if file == nil || file.Filename == "" {
		return "", ""
	}
	if file.Size > 5*1024*1024 {
		return "", label + " must be smaller than 5 MB."
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		return "", label + " must be jpg, png, gif, or webp."
	}
	month := time.Now().Format("2006/01")
	targetDir := filepath.Join(c.cfg.StaticDir, "uploads", month)
	filename, err := file.Save(targetDir, true)
	if err != nil {
		log.Printf("save image upload: %v", err)
		return "", "Could not save " + strings.ToLower(label) + "."
	}
	return "/static/uploads/" + month + "/" + filename, ""
}

func postFromInput(input store.PostInput) models.Post {
	tags := make([]models.Tag, 0, len(input.Tags))
	for _, tag := range input.Tags {
		tags = append(tags, models.Tag{Name: tag})
	}
	return models.Post{
		ID:          input.ID,
		Slug:        input.Slug,
		Title:       input.Title,
		Excerpt:     input.Excerpt,
		CoverImage:  input.CoverImage,
		Status:      input.Status,
		Category:    models.Category{Name: input.CategoryName},
		Tags:        tags,
		ContentHTML: template.HTML(input.ContentHTML),
	}
}

func friendLinkFromInput(input store.FriendLinkInput) models.FriendLink {
	return models.FriendLink{
		ID:          input.ID,
		Name:        strings.TrimSpace(input.Name),
		URL:         strings.TrimSpace(input.URL),
		Description: strings.TrimSpace(input.Description),
		ImageURL:    strings.TrimSpace(input.ImageURL),
		SortOrder:   input.SortOrder,
		Visible:     input.Visible,
		Category: models.FriendLinkCategory{
			Name: strings.TrimSpace(input.CategoryName),
		},
	}
}

func momentFromInput(input store.MomentInput) models.Moment {
	status := strings.TrimSpace(input.Status)
	if status != "draft" {
		status = "published"
	}
	return models.Moment{
		ID:      input.ID,
		Content: strings.TrimSpace(input.Content),
		Author:  strings.TrimSpace(input.Author),
		Status:  status,
	}
}

func categoryFromInput(input store.CategoryInput) models.Category {
	return models.Category{
		ID:          input.ID,
		Slug:        strings.TrimSpace(input.Slug),
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		CoverImage:  strings.TrimSpace(input.CoverImage),
	}
}

func tagFromInput(input store.TagInput) models.Tag {
	return models.Tag{
		ID:   input.ID,
		Slug: strings.TrimSpace(input.Slug),
		Name: strings.TrimSpace(input.Name),
	}
}

func (c *Controller) render(r *ghttp.Request, name string, data PageData) {
	c.renderer.HTML(r, name, data)
}

func (c *Controller) error(r *ghttp.Request, err error) {
	log.Println(err)
	r.Response.WriteStatus(500, "Internal Server Error")
}

func splitTags(value string) []string {
	value = strings.ReplaceAll(value, "\uFF0C", ",")
	parts := strings.Split(value, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			tags = append(tags, part)
		}
	}
	return tags
}

func normalizeSiteSettings(site config.Site, fallback config.Site) config.Site {
	if site.Name == "" {
		site.Name = fallback.Name
	}
	if site.Description == "" {
		site.Description = fallback.Description
	}
	if site.Author == "" {
		site.Author = fallback.Author
	}
	if site.ThemeColor == "" {
		site.ThemeColor = fallback.ThemeColor
	}
	if site.HeroImage == "" {
		site.HeroImage = fallback.HeroImage
	}
	site.HeroOverlayOpacity = normalizeHeroOverlayOpacity(site.HeroOverlayOpacity, fallback.HeroOverlayOpacity)
	if site.Avatar == "" {
		site.Avatar = fallback.Avatar
	}
	if site.DefaultPostCover == "" {
		site.DefaultPostCover = fallback.DefaultPostCover
	}
	if site.SakuraEffects != "1" {
		site.SakuraEffects = "0"
	}
	if site.FooterText == "" {
		site.FooterText = fallback.FooterText
	}
	if site.FooterCredit == "" {
		site.FooterCredit = fallback.FooterCredit
	}
	if len(site.Navigation) == 0 {
		site.Navigation = fallback.Navigation
	}
	if len(site.FocusCards) == 0 {
		site.FocusCards = fallback.FocusCards
	}
	return site
}

func normalizeHeroOverlayOpacity(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	opacity, err := strconv.ParseFloat(value, 64)
	if err != nil {
		opacity, err = strconv.ParseFloat(fallback, 64)
		if err != nil {
			return "1"
		}
	}
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	return strconv.FormatFloat(opacity, 'f', -1, 64)
}

func parseNavigation(value string) []config.NavItem {
	var items []config.NavItem
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		label, url, ok := strings.Cut(line, "|")
		if !ok {
			continue
		}
		label = strings.TrimSpace(label)
		url = strings.TrimSpace(url)
		if label == "" || !isAllowedNavURL(url) {
			continue
		}
		items = append(items, config.NavItem{Label: label, URL: url})
	}
	return items
}

func formatNavigation(items []config.NavItem) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		if item.Label == "" || item.URL == "" {
			continue
		}
		lines = append(lines, item.Label+" | "+item.URL)
	}
	return strings.Join(lines, "\n")
}

func parseFocusCards(value string) []config.FocusCard {
	var cards []config.FocusCard
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		title := strings.TrimSpace(parts[0])
		url := strings.TrimSpace(parts[1])
		image := strings.TrimSpace(strings.Join(parts[2:], "|"))
		if title == "" || !isAllowedNavURL(url) || image == "" {
			continue
		}
		cards = append(cards, config.FocusCard{Title: title, URL: url, Image: image})
		if len(cards) == 3 {
			break
		}
	}
	return cards
}

func formatFocusCards(cards []config.FocusCard) string {
	lines := make([]string, 0, len(cards))
	for _, card := range cards {
		if card.Title == "" || card.URL == "" || card.Image == "" {
			continue
		}
		lines = append(lines, card.Title+" | "+card.URL+" | "+card.Image)
	}
	return strings.Join(lines, "\n")
}

func isAllowedNavURL(url string) bool {
	return strings.HasPrefix(url, "/") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")
}
