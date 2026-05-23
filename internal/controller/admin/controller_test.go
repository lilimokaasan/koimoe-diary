package admin

import (
	"strconv"
	"testing"
	"time"

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

func TestParseSocialLinks(t *testing.T) {
	links := parseSocialLinks("GitHub | https://github.com/example | fa-github\nMail | mailto:hello@example.com | envelope-o\nBad | javascript:alert(1) | fa-bug")

	if len(links) != 2 {
		t.Fatalf("len(parseSocialLinks) = %d, want 2", len(links))
	}
	if links[0].Label != "GitHub" || links[0].Icon != "fa-github" {
		t.Fatalf("first social link = %#v", links[0])
	}
	if links[1].Icon != "fa-envelope-o" {
		t.Fatalf("second social icon = %q, want fa-envelope-o", links[1].Icon)
	}
}

func TestParsePostPublishedAt(t *testing.T) {
	got, err := parsePostPublishedAt("2026-05-23T21:30")
	if err != nil {
		t.Fatalf("parsePostPublishedAt returned error: %v", err)
	}
	want := time.Date(2026, 5, 23, 21, 30, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Fatalf("parsePostPublishedAt = %s, want %s", got, want)
	}

	if got, err := parsePostPublishedAt(""); err != nil || !got.IsZero() {
		t.Fatalf("empty parse = %s, %v; want zero nil", got, err)
	}

	if _, err := parsePostPublishedAt("2026-05-23 21:30"); err == nil {
		t.Fatal("invalid datetime should return an error")
	}
}

func TestNormalizePostStatusFilter(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{status: "published", want: "published"},
		{status: "scheduled", want: "scheduled"},
		{status: " Draft ", want: "draft"},
		{status: "private", want: "private"},
		{status: "spam", want: ""},
	}

	for _, tt := range tests {
		if got := normalizePostStatusFilter(tt.status); got != tt.want {
			t.Fatalf("normalizePostStatusFilter(%q) = %q, want %q", tt.status, got, tt.want)
		}
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
		PostShare:        "1",
		PostCopyNotice:   "1",
		PostReward:       "0",
		PostRewardText:   "Support text",
		FooterText:       "footer",
		FooterCredit:     "credit",
		SocialLinks:      []config.SocialLink{{Label: "Feed", URL: "/feed", Icon: "fa-rss"}},
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
	if site.PostShare != "1" {
		t.Fatalf("PostShare = %q, want enabled by default", site.PostShare)
	}
	if site.PostCopyNotice != "1" {
		t.Fatalf("PostCopyNotice = %q, want enabled by default", site.PostCopyNotice)
	}
	if site.PostReward != "0" {
		t.Fatalf("PostReward = %q, want disabled without images", site.PostReward)
	}
	if site.PostRewardText != fallback.PostRewardText {
		t.Fatalf("PostRewardText = %q, want fallback %q", site.PostRewardText, fallback.PostRewardText)
	}
	if site.SakuraEffects != "0" {
		t.Fatalf("SakuraEffects = %q, want normalized off", site.SakuraEffects)
	}

	site = normalizeSiteSettings(config.Site{
		Name:             "KoiMoe Diary",
		PostShare:        "0",
		PostCopyNotice:   "0",
		PostReward:       "1",
		PostRewardText:   "Thanks",
		PostRewardAlipay: "/alipay.png",
	}, fallback)
	if site.PostShare != "0" {
		t.Fatalf("PostShare = %q, want disabled", site.PostShare)
	}
	if site.PostCopyNotice != "0" {
		t.Fatalf("PostCopyNotice = %q, want disabled", site.PostCopyNotice)
	}
	if site.PostReward != "1" {
		t.Fatalf("PostReward = %q, want enabled with image", site.PostReward)
	}
	if site.PostRewardText != "Thanks" {
		t.Fatalf("PostRewardText = %q, want custom text", site.PostRewardText)
	}
}

func TestNormalizeCommentIDs(t *testing.T) {
	values := []string{"1", "2", "bad", "2", "0", "-4", " 3 "}
	got := normalizeCommentIDs(values)
	want := []int64{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("len(normalizeCommentIDs) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalizeCommentIDs[%d] = %d, want %d", i, got[i], want[i])
		}
	}

	many := make([]string, 0, 105)
	for i := 1; i <= 105; i++ {
		many = append(many, strconv.Itoa(i))
	}
	got = normalizeCommentIDs(many)
	if len(got) != 100 {
		t.Fatalf("len(normalizeCommentIDs many) = %d, want 100", len(got))
	}
}
