package admin

import (
	"testing"

	"sakurairo-go/internal/config"
)

func TestNormalizeHeroOverlayOpacity(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		want     string
	}{
		{name: "keeps valid value", value: "0.45", fallback: "1", want: "0.45"},
		{name: "uses fallback for blank", value: "", fallback: "0.8", want: "0.8"},
		{name: "clamps below zero", value: "-0.2", fallback: "1", want: "0"},
		{name: "clamps above one", value: "1.4", fallback: "1", want: "1"},
		{name: "falls back to one for invalid input", value: "soft", fallback: "mist", want: "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeHeroOverlayOpacity(tt.value, tt.fallback); got != tt.want {
				t.Fatalf("normalizeHeroOverlayOpacity(%q, %q) = %q, want %q", tt.value, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestParseFocusCards(t *testing.T) {
	cards := parseFocusCards("Archive | /archives | /static/theme/content-image/d-1.jpg\nBroken line\nSearch | /search | /static/theme/content-image/d-2.jpg\nRemote | https://example.com | https://example.com/card.jpg")

	if len(cards) != 3 {
		t.Fatalf("len(parseFocusCards) = %d, want 3", len(cards))
	}
	if cards[0].Title != "Archive" || cards[0].URL != "/archives" || cards[0].Image != "/static/theme/content-image/d-1.jpg" {
		t.Fatalf("first focus card = %#v", cards[0])
	}
}

func TestNormalizeSiteSettingsKeepsConfigurableLicense(t *testing.T) {
	fallback := config.Site{
		Name:             "KoiMoe Diary",
		Description:      "soft",
		Author:           "Lilim",
		ThemeColor:       "#fb98c0",
		HeroImage:        "/hero.jpg",
		Avatar:           "/avatar.jpg",
		DefaultPostCover: "/cover.jpg",
		PostLicenseText:  "Default license",
		PostLicenseURL:   "https://example.com/license",
		FooterText:       "footer",
		FooterCredit:     "credit",
	}
	site := normalizeSiteSettings(config.Site{
		Name:            "KoiMoe Diary",
		Description:     "soft",
		Author:          "Lilim",
		PostLicenseURL:  "",
		SakuraEffects:   "yes",
		FooterText:      "footer",
		FooterCredit:    "credit",
		PostLicenseText: "",
	}, fallback)

	if site.PostLicenseText != fallback.PostLicenseText {
		t.Fatalf("PostLicenseText = %q, want fallback %q", site.PostLicenseText, fallback.PostLicenseText)
	}
	if site.PostLicenseURL != "" {
		t.Fatalf("PostLicenseURL = %q, want empty URL allowed", site.PostLicenseURL)
	}
	if site.SakuraEffects != "0" {
		t.Fatalf("SakuraEffects = %q, want normalized off", site.SakuraEffects)
	}
}
