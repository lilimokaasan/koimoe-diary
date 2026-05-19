package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

type Site struct {
	Name               string
	Description        string
	Author             string
	Notice             string
	ThemeColor         string
	HeroImage          string
	HeroOverlayOpacity string
	Avatar             string
	DefaultPostCover   string
	SakuraEffects      string
	FooterText         string
	FooterCredit       string
	Navigation         []NavItem
	FocusCards         []FocusCard
}

type NavItem struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type FocusCard struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Image string `json:"image"`
}

type Config struct {
	siteMu           sync.RWMutex
	Addr             string
	DSN              string
	StaticDir        string
	SeedDemo         bool
	AdminUsername    string
	AdminPassword    string
	AdminSecret      string
	DBMaxOpenConns   int
	DBMaxIdleConns   int
	DBConnMaxMinutes int
	Site             Site
}

func (c *Config) GetSite() Site {
	c.siteMu.RLock()
	defer c.siteMu.RUnlock()
	return c.Site
}

func (c *Config) SetSite(site Site) {
	c.siteMu.Lock()
	defer c.siteMu.Unlock()
	c.Site = site
}

func FromEnv() Config {
	loadDotEnv(".env")

	return Config{
		Addr:             env("APP_ADDR", "127.0.0.1:8080"),
		DSN:              env("MYSQL_DSN", "sakurairo_app:change-me@tcp(127.0.0.1:3306)/sakurairo?charset=utf8mb4&parseTime=True&loc=Local"),
		StaticDir:        env("STATIC_DIR", "web/static"),
		SeedDemo:         env("SEED_DEMO", "1") != "0",
		AdminUsername:    env("ADMIN_USERNAME", "admin"),
		AdminPassword:    env("ADMIN_PASSWORD", ""),
		AdminSecret:      env("ADMIN_SECRET", ""),
		DBMaxOpenConns:   envInt("DB_MAX_OPEN_CONNS", 10),
		DBMaxIdleConns:   envInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxMinutes: envInt("DB_CONN_MAX_MINUTES", 30),
		Site: Site{
			Name:               env("SITE_NAME", "KoiMoe Diary"),
			Description:        env("SITE_DESCRIPTION", "恋と萌えの小さな場所"),
			Author:             env("SITE_AUTHOR", "莉莉姆"),
			Notice:             env("SITE_NOTICE", "A soft diary for tiny heartbeats, cute things, and everyday fragments."),
			ThemeColor:         env("THEME_COLOR", "#fb98c0"),
			HeroImage:          env("HERO_IMAGE", "/static/theme/screenshot.jpg"),
			HeroOverlayOpacity: env("HERO_OVERLAY_OPACITY", "1"),
			Avatar:             env("SITE_AVATAR", "/static/theme/content-image/d-1.jpg"),
			DefaultPostCover:   env("DEFAULT_POST_COVER", "/static/theme/content-image/d-1.jpg"),
			SakuraEffects:      env("SAKURA_EFFECTS", "0"),
			FooterText:         env("SITE_FOOTER_TEXT", "A soft diary for tiny heartbeats, cute things, and everyday fragments."),
			FooterCredit:       env("SITE_FOOTER_CREDIT", "A KoiMoe diary shaped with Sakurairo."),
			Navigation: []NavItem{
				{Label: "Home", URL: "/"},
				{Label: "Archives", URL: "/archives"},
				{Label: "Links", URL: "/links"},
				{Label: "Moments", URL: "/moments"},
				{Label: "Search", URL: "/search"},
				{Label: "Admin Login", URL: "/admin/login"},
			},
			FocusCards: []FocusCard{
				{Title: "Archive", URL: "/archives", Image: "/static/theme/content-image/d-1.jpg"},
				{Title: "Search", URL: "/search", Image: "/static/theme/content-image/d-2.jpg"},
				{Title: "KoiMoe Diary", URL: "/", Image: "/static/theme/content-image/d-3.jpg"},
			},
		},
	}
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}
