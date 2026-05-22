package blog

import (
	"strings"
	"testing"

	"sakurairo-go/internal/models"
)

func TestDetectCommentSpam(t *testing.T) {
	tests := []struct {
		name    string
		comment models.Comment
		want    string
	}{
		{
			name: "allows ordinary comment",
			comment: models.Comment{
				Author:  "Lilim",
				Email:   "lilim@example.com",
				Content: "This post feels soft and useful.",
			},
			want: "",
		},
		{
			name: "blocks too many links",
			comment: models.Comment{
				Author:  "Promo",
				Email:   "promo@example.com",
				Content: "see https://a.example and https://b.example and https://c.example",
			},
			want: "too many links",
		},
		{
			name: "blocks advertising keyword",
			comment: models.Comment{
				Author:  "SEO Team",
				Email:   "seo@example.com",
				Content: "Need backlink traffic now",
			},
			want: "advertising",
		},
		{
			name: "blocks repeated noise",
			comment: models.Comment{
				Author:  "Noise",
				Email:   "noise@example.com",
				Content: "so cute!!!!!!!!!!",
			},
			want: "repeated",
		},
		{
			name: "blocks contact advertising",
			comment: models.Comment{
				Author:  "Contact",
				Email:   "contact@example.com",
				Content: "chat on telegram and whatsapp",
			},
			want: "contact-info advertising",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectCommentSpam(tt.comment)
			if tt.want == "" && got != "" {
				t.Fatalf("detectCommentSpam() = %q, want empty", got)
			}
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Fatalf("detectCommentSpam() = %q, want substring %q", got, tt.want)
			}
		})
	}
}

func TestValidateCommentLeavesSpamForQuarantine(t *testing.T) {
	comment := models.Comment{
		Author:  "SEO Team",
		Email:   "seo@example.com",
		Content: "Need backlink traffic now",
	}
	if got := validateComment(comment); got != "" {
		t.Fatalf("validateComment() = %q, want empty so spam can be quarantined", got)
	}
}
