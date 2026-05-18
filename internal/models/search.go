package models

import "time"

type SearchIndex struct {
	GeneratedAt time.Time            `json:"generated_at"`
	Posts       []SearchPostItem     `json:"posts"`
	Categories  []SearchTaxonomyItem `json:"categories"`
	Tags        []SearchTaxonomyItem `json:"tags"`
}

type SearchPostItem struct {
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	Excerpt      string    `json:"excerpt"`
	Content      string    `json:"content"`
	CoverImage   string    `json:"cover_image"`
	Category     string    `json:"category"`
	Tags         []string  `json:"tags"`
	CommentCount int64     `json:"comment_count"`
	Views        int64     `json:"views"`
	Likes        int64     `json:"likes"`
	PublishedAt  time.Time `json:"published_at"`
}

type SearchTaxonomyItem struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	PostCount int    `json:"post_count"`
}
