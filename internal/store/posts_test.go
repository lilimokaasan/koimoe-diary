package store

import (
	"testing"
	"time"
)

func TestNormalizeCommentStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{name: "approved", status: "approved", want: "approved"},
		{name: "hidden", status: "hidden", want: "hidden"},
		{name: "spam", status: "spam", want: "spam"},
		{name: "case and space", status: " Spam ", want: "spam"},
		{name: "unknown", status: "pending", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeCommentStatus(tt.status); got != tt.want {
				t.Fatalf("normalizeCommentStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestCommentStatusFilter(t *testing.T) {
	if got := commentStatusFilter("spam"); got != "spam" {
		t.Fatalf("commentStatusFilter(spam) = %q, want spam", got)
	}
	if got := commentStatusFilter("pending"); got != "" {
		t.Fatalf("commentStatusFilter(pending) = %q, want empty", got)
	}
}

func TestNormalizePostInputPublishedAt(t *testing.T) {
	future := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	input := normalizePostInput(PostInput{
		Title:       "Scheduled diary",
		Status:      "published",
		PublishedAt: future,
	})

	if input.PublishedAt.IsZero() {
		t.Fatal("PublishedAt is zero")
	}
	if !input.PublishedAt.Equal(future) {
		t.Fatalf("PublishedAt = %s, want %s", input.PublishedAt, future)
	}
}

func TestNormalizePostInputDefaultsPublishedAt(t *testing.T) {
	input := normalizePostInput(PostInput{Title: "Now"})
	if input.PublishedAt.IsZero() {
		t.Fatal("PublishedAt should default to now")
	}
}

func TestReadingMinutes(t *testing.T) {
	tests := []struct {
		name string
		html string
		want int
	}{
		{name: "empty", html: "", want: 1},
		{name: "short html", html: "<p>Hello &amp; welcome to KoiMoe.</p>", want: 1},
		{name: "long english", html: repeatWords("dream", 441), want: 3},
		{name: "cjk content", html: "<p>" + repeatRunes('萌', 441) + "</p>", want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readingMinutes(tt.html); got != tt.want {
				t.Fatalf("readingMinutes() = %d, want %d", got, tt.want)
			}
		})
	}
}

func repeatWords(word string, count int) string {
	out := ""
	for i := 0; i < count; i++ {
		if i > 0 {
			out += " "
		}
		out += word
	}
	return out
}

func repeatRunes(r rune, count int) string {
	out := make([]rune, count)
	for i := range out {
		out[i] = r
	}
	return string(out)
}
