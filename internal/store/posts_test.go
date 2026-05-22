package store

import "testing"

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
