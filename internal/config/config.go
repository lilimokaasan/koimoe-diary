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
	PostLicenseText    string
	PostLicenseURL     string
	PostShare          string
	PostCopyNotice     string
	PostReward         string
	PostRewardText     string
	PostRewardAlipay   string
	PostRewardWechat   string
	SakuraEffects      string
	FooterText         string
	FooterCredit       string
	Navigation         []NavItem
	FocusCards         []FocusCard
	SocialLinks        []SocialLink
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

type SocialLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Icon  string `json:"icon"`
}

type Mail struct {
	Enabled    bool
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	FromName   string
	AdminEmail string
	TLSMode    string
}

type Config struct {
	siteMu           sync.RWMutex
	mailMu           sync.RWMutex
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
	Mail             Mail
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

func (c *Config) GetMail() Mail {
	c.mailMu.RLock()
	defer c.mailMu.RUnlock()
	return c.Mail
}

func (c *Config) SetMail(mail Mail) {
	c.mailMu.Lock()
	defer c.mailMu.Unlock()
	c.Mail = mail
}

func FromEnv() Config {
	loadDotEnv(".env")

	siteName := env("SITE_NAME", "KoiMoe Diary")
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
		Mail: Mail{
			Enabled:    env("MAIL_ENABLED", "0") == "1",
			Host:       env("SMTP_HOST", ""),
			Port:       envInt("SMTP_PORT", 587),
			Username:   env("SMTP_USERNAME", ""),
			Password:   env("SMTP_PASSWORD", ""),
			From:       env("SMTP_FROM", ""),
			FromName:   env("SMTP_FROM_NAME", siteName),
			AdminEmail: env("MAIL_ADMIN_EMAIL", env("SMTP_FROM", "")),
			TLSMode:    env("SMTP_TLS_MODE", "starttls"),
		},
		Site: Site{
			Name:               siteName,
			Description:        env("SITE_DESCRIPTION", "恋と萌えの小さな場所"),
			Author:             env("SITE_AUTHOR", "莉莉姆"),
			Notice:             env("SITE_NOTICE", "A soft diary for tiny heartbeats, cute things, and everyday fragments."),
			ThemeColor:         env("THEME_COLOR", "#fb98c0"),
			HeroImage:          env("HERO_IMAGE", "/static/theme/screenshot.jpg"),
			HeroOverlayOpacity: env("HERO_OVERLAY_OPACITY", "1"),
			Avatar:             env("SITE_AVATAR", "/static/curated-sakura-images/originals/sakura-branch-pastel.jpg"),
			DefaultPostCover:   env("DEFAULT_POST_COVER", "/static/curated-sakura-images/originals/fuji-pagoda-sakura-01.jpg"),
			PostLicenseText:    env("POST_LICENSE_TEXT", "Attribution-NonCommercial-ShareAlike 4.0 International"),
			PostLicenseURL:     env("POST_LICENSE_URL", "https://creativecommons.org/licenses/by-nc-sa/4.0/deed.zh"),
			PostShare:          env("POST_SHARE", "1"),
			PostCopyNotice:     env("POST_COPY_NOTICE", "1"),
			PostReward:         env("POST_REWARD", "0"),
			PostRewardText:     env("POST_REWARD_TEXT", "If this tiny fragment warmed your day, a small support is deeply appreciated."),
			PostRewardAlipay:   env("POST_REWARD_ALIPAY", ""),
			PostRewardWechat:   env("POST_REWARD_WECHAT", ""),
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
				{Title: "Archive", URL: "/archives", Image: "/static/curated-sakura-images/originals/fuji-pagoda-sakura-01.jpg"},
				{Title: "Search", URL: "/search", Image: "/static/curated-sakura-images/originals/sakura-branch-pastel.jpg"},
				{Title: "KoiMoe Diary", URL: "/", Image: "/static/curated-sakura-images/originals/white-castle-sakura.jpg"},
			},
			SocialLinks: []SocialLink{
				{Label: "Feed", URL: "/feed", Icon: "fa-rss"},
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
