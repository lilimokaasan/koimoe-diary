package models

import (
	"fmt"
	"html/template"
	"net/url"
	"time"
)

type Post struct {
	ID           int64
	Slug         string
	Title        string
	Excerpt      string
	ContentHTML  template.HTML
	CoverImage   string
	Status       string
	Category     Category
	Tags         []Tag
	CommentCount int64
	Views        int64
	PublishedAt  time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Comment struct {
	ID        int64
	PostID    int64
	PostTitle string
	PostSlug  string
	Author    string
	Email     string
	Website   string
	Content   string
	Status    string
	CreatedAt time.Time
}

type Category struct {
	ID          int64
	Slug        string
	Name        string
	Description string
	PostCount   int
}

type Tag struct {
	ID        int64
	Slug      string
	Name      string
	PostCount int
}

type PageInfo struct {
	Page       int
	PageSize   int
	Total      int
	TotalPages int
	BasePath   string
	Query      string
}

func (p PageInfo) HasPrev() bool {
	return p.Page > 1
}

func (p PageInfo) HasNext() bool {
	return p.Page < p.TotalPages
}

func (p PageInfo) PrevPage() int {
	if !p.HasPrev() {
		return 1
	}
	return p.Page - 1
}

func (p PageInfo) NextPage() int {
	if !p.HasNext() {
		return p.TotalPages
	}
	return p.Page + 1
}

func (p PageInfo) URL(page int) string {
	values := url.Values{}
	if page > 1 {
		values.Set("page", fmt.Sprint(page))
	}
	if p.Query != "" {
		values.Set("q", p.Query)
	}
	if encoded := values.Encode(); encoded != "" {
		return p.BasePath + "?" + encoded
	}
	return p.BasePath
}
