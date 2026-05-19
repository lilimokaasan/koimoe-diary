package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"sakurairo-go/internal/config"
)

type SettingsStore struct {
	db *sql.DB
}

func NewSettingsStore(db *sql.DB) *SettingsStore {
	return &SettingsStore{db: db}
}

func (s *SettingsStore) Init(defaults config.Site) error {
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS site_settings (
	setting_key VARCHAR(80) PRIMARY KEY,
	setting_value TEXT NOT NULL,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`); err != nil {
		return err
	}
	return s.ensureDefaults(defaults)
}

func (s *SettingsStore) Site(ctx context.Context, fallback config.Site) (config.Site, error) {
	values, err := s.values(ctx)
	if err != nil {
		return fallback, err
	}
	site := fallback
	if values["site_name"] != "" {
		site.Name = values["site_name"]
	}
	if values["site_description"] != "" {
		site.Description = values["site_description"]
	}
	if values["site_author"] != "" {
		site.Author = values["site_author"]
	}
	if values["site_notice"] != "" {
		site.Notice = values["site_notice"]
	}
	if values["theme_color"] != "" {
		site.ThemeColor = values["theme_color"]
	}
	if values["hero_image"] != "" {
		site.HeroImage = values["hero_image"]
	}
	if values["hero_overlay_opacity"] != "" {
		site.HeroOverlayOpacity = values["hero_overlay_opacity"]
	}
	if values["site_avatar"] != "" {
		site.Avatar = values["site_avatar"]
	}
	if values["default_post_cover"] != "" {
		site.DefaultPostCover = values["default_post_cover"]
	}
	if values["sakura_effects"] != "" {
		site.SakuraEffects = values["sakura_effects"]
	}
	if values["footer_text"] != "" {
		site.FooterText = values["footer_text"]
	}
	if values["footer_credit"] != "" {
		site.FooterCredit = values["footer_credit"]
	}
	if values["navigation"] != "" {
		var navigation []config.NavItem
		if err := json.Unmarshal([]byte(values["navigation"]), &navigation); err == nil && len(navigation) > 0 {
			site.Navigation = navigation
		}
	}
	if values["focus_cards"] != "" {
		var focusCards []config.FocusCard
		if err := json.Unmarshal([]byte(values["focus_cards"]), &focusCards); err == nil && len(focusCards) > 0 {
			site.FocusCards = focusCards
		}
	}
	return site, nil
}

func (s *SettingsStore) SaveSite(ctx context.Context, site config.Site) error {
	navigation, err := json.Marshal(site.Navigation)
	if err != nil {
		return err
	}
	focusCards, err := json.Marshal(site.FocusCards)
	if err != nil {
		return err
	}
	settings := map[string]string{
		"site_name":            site.Name,
		"site_description":     site.Description,
		"site_author":          site.Author,
		"site_notice":          site.Notice,
		"theme_color":          site.ThemeColor,
		"hero_image":           site.HeroImage,
		"hero_overlay_opacity": site.HeroOverlayOpacity,
		"site_avatar":          site.Avatar,
		"default_post_cover":   site.DefaultPostCover,
		"sakura_effects":       site.SakuraEffects,
		"footer_text":          site.FooterText,
		"footer_credit":        site.FooterCredit,
		"navigation":           string(navigation),
		"focus_cards":          string(focusCards),
	}
	for key, value := range settings {
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO site_settings (setting_key, setting_value)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *SettingsStore) ensureDefaults(defaults config.Site) error {
	navigation, err := json.Marshal(defaults.Navigation)
	if err != nil {
		return err
	}
	focusCards, err := json.Marshal(defaults.FocusCards)
	if err != nil {
		return err
	}
	settings := map[string]string{
		"site_name":            defaults.Name,
		"site_description":     defaults.Description,
		"site_author":          defaults.Author,
		"site_notice":          defaults.Notice,
		"theme_color":          defaults.ThemeColor,
		"hero_image":           defaults.HeroImage,
		"hero_overlay_opacity": defaults.HeroOverlayOpacity,
		"site_avatar":          defaults.Avatar,
		"default_post_cover":   defaults.DefaultPostCover,
		"sakura_effects":       defaults.SakuraEffects,
		"footer_text":          defaults.FooterText,
		"footer_credit":        defaults.FooterCredit,
		"navigation":           string(navigation),
		"focus_cards":          string(focusCards),
	}
	for key, value := range settings {
		if _, err := s.db.Exec(`
INSERT IGNORE INTO site_settings (setting_key, setting_value)
VALUES (?, ?)`, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *SettingsStore) values(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT setting_key, setting_value FROM site_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := map[string]string{}
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		values[key] = value
	}
	return values, rows.Err()
}
