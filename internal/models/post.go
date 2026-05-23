package models

import (
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"time"
)

type Post struct {
	ID             int64
	Slug           string
	Title          string
	Excerpt        string
	ContentHTML    template.HTML
	Outline        []ContentHeading
	CoverImage     string
	Status         string
	Category       Category
	Tags           []Tag
	CommentCount   int64
	Views          int64
	Likes          int64
	ReadingMinutes int
	PublishedAt    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Page struct {
	ID          int64
	Slug        string
	Title       string
	Excerpt     string
	ContentHTML template.HTML
	Outline     []ContentHeading
	CoverImage  string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ContentHeading struct {
	ID    string
	Title string
	Level int
}

type Comment struct {
	ID           int64
	PostID       int64
	ParentID     int64
	PostTitle    string
	PostSlug     string
	ParentAuthor string
	Author       string
	Email        string
	Website      string
	Content      string
	ContentHTML  template.HTML
	Status       string
	IsPrivate    bool
	MailNotify   bool
	CreatedAt    time.Time
	Replies      []Comment
}

type Category struct {
	ID          int64
	Slug        string
	Name        string
	Description string
	CoverImage  string
	PostCount   int
}

type Tag struct {
	ID        int64
	Slug      string
	Name      string
	PostCount int
}

type MediaAsset struct {
	ID           int64
	Filename     string
	OriginalName string
	Title        string
	AltText      string
	Description  string
	MimeType     string
	SizeBytes    int64
	Width        int
	Height       int
	URL          string
	Storage      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (asset MediaAsset) Name() string {
	if asset.Title != "" {
		return asset.Title
	}
	if asset.OriginalName != "" {
		return asset.OriginalName
	}
	return asset.Filename
}

func (asset MediaAsset) SizeLabel() string {
	if asset.SizeBytes < 1024 {
		return strconv.FormatInt(asset.SizeBytes, 10) + " B"
	}
	units := []string{"KB", "MB", "GB"}
	value := float64(asset.SizeBytes)
	for _, unit := range units {
		value = value / 1024
		if value < 1024 {
			return strconv.FormatFloat(value, 'f', 1, 64) + " " + unit
		}
	}
	return strconv.FormatFloat(value, 'f', 1, 64) + " GB"
}

type ArchiveGroup struct {
	Label string
	Year  int
	Month time.Month
	Posts []Post
	Count int
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
