package app

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"sakurairo-go/internal/buildinfo"
	"sakurairo-go/internal/config"
	"sakurairo-go/internal/controller/admin"
	"sakurairo-go/internal/controller/blog"
	"sakurairo-go/internal/mailer"
	"sakurairo-go/internal/store"
	"sakurairo-go/internal/view"
)

type App struct {
	cfg    *config.Config
	db     *sql.DB
	server *ghttp.Server
}

func New() (*App, error) {
	cfg := config.FromEnv()

	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.DBConnMaxMinutes) * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	postStore := store.NewPostStore(db)
	if err := postStore.Init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if cfg.SeedDemo {
		if err := postStore.SeedDemo(); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	settingsStore := store.NewSettingsStore(db)
	if err := settingsStore.Init(cfg.GetSite()); err != nil {
		_ = db.Close()
		return nil, err
	}
	linkStore := store.NewLinkStore(db)
	if err := linkStore.Init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	momentStore := store.NewMomentStore(db)
	if err := momentStore.Init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	mediaStore := store.NewMediaStore(db)
	if err := mediaStore.Init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	site, err := settingsStore.Site(context.Background(), cfg.GetSite())
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	cfg.SetSite(site)
	mailCfg, err := settingsStore.Mail(context.Background(), cfg.GetMail())
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	cfg.SetMail(mailCfg)

	renderer, err := view.NewDefaultRenderer()
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	server := g.Server()
	server.SetAddr(cfg.Addr)
	server.Use(requestLogger)
	server.AddStaticPath("/static", cfg.StaticDir)
	server.BindHandler("GET:/favicon.png", faviconHandler(cfg.StaticDir, "favicon.png", "image/png"))
	server.BindHandler("HEAD:/favicon.png", faviconHandler(cfg.StaticDir, "favicon.png", "image/png"))
	server.BindHandler("GET:/favicon.ico", faviconHandler(cfg.StaticDir, "favicon.ico", "image/x-icon"))
	server.BindHandler("HEAD:/favicon.ico", faviconHandler(cfg.StaticDir, "favicon.ico", "image/x-icon"))
	server.BindHandler("GET:/api/health", func(r *ghttp.Request) {
		r.Response.WriteJson(g.Map{
			"ok":         true,
			"name":       cfg.GetSite().Name,
			"build_info": buildinfo.Snapshot(),
		})
	})

	mailSender := mailer.NewDynamicSMTP(cfg.GetMail)
	admin.New(&cfg, postStore, settingsStore, linkStore, momentStore, mediaStore, mailSender, renderer).Register(server)
	blog.New(&cfg, postStore, linkStore, momentStore, mailSender, renderer).Register(server)

	return &App{cfg: &cfg, db: db, server: server}, nil
}

func (a *App) Run() {
	defer func() {
		if err := a.db.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	log.Printf("Sakurairo GoFrame listening on %s", a.cfg.Addr)
	a.server.Run()
}

func requestLogger(r *ghttp.Request) {
	if isQuietAssetPath(r.URL.Path) {
		r.Middleware.Next()
		return
	}

	started := time.Now()
	r.Middleware.Next()

	status := r.Response.Status
	if status == 0 {
		status = http.StatusOK
	}
	log.Printf(
		"request method=%s path=%s status=%d duration=%s ip=%s ua=%q",
		r.Method,
		r.URL.RequestURI(),
		status,
		time.Since(started).Round(time.Millisecond),
		r.GetClientIp(),
		r.UserAgent(),
	)
}

func isQuietAssetPath(path string) bool {
	return strings.HasPrefix(path, "/static/") || path == "/favicon.ico" || path == "/favicon.png"
}

func faviconHandler(staticDir string, name string, contentType string) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		r.Response.Header().Set("Content-Type", contentType)
		r.Response.Header().Set("Cache-Control", "public, max-age=604800")
		http.ServeFile(r.Response.ResponseWriter, r.Request, filepath.Join(staticDir, name))
	}
}
