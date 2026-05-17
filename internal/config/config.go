package config

import (
	"os"
	"strconv"
	"strings"
)

type Site struct {
	Name        string
	Description string
	Author      string
	Notice      string
	ThemeColor  string
	HeroImage   string
	Avatar      string
}

type Config struct {
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
			Name:        env("SITE_NAME", "Sakurairo Go"),
			Description: env("SITE_DESCRIPTION", "A lightweight GoFrame blog inspired by Sakurairo."),
			Author:      env("SITE_AUTHOR", "Codex"),
			Notice:      env("SITE_NOTICE", "The Sakurairo GoFrame rewrite has started."),
			ThemeColor:  env("THEME_COLOR", "#fe9600"),
			HeroImage:   env("HERO_IMAGE", "/static/theme/screenshot.jpg"),
			Avatar:      env("SITE_AVATAR", "/static/theme/content-image/d-1.jpg"),
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
