package app

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"sakurairo-go/internal/config"
	"sakurairo-go/internal/controller/admin"
	"sakurairo-go/internal/controller/blog"
	"sakurairo-go/internal/store"
	"sakurairo-go/internal/view"
)

type App struct {
	cfg    config.Config
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

	renderer, err := view.NewDefaultRenderer()
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	server := g.Server()
	server.SetAddr(cfg.Addr)
	server.AddStaticPath("/static", cfg.StaticDir)
	server.BindHandler("GET:/api/health", func(r *ghttp.Request) {
		r.Response.WriteJson(g.Map{
			"ok":   true,
			"name": cfg.Site.Name,
		})
	})

	admin.New(cfg, postStore, renderer).Register(server)
	blog.New(cfg, postStore, renderer).Register(server)

	return &App{cfg: cfg, db: db, server: server}, nil
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
