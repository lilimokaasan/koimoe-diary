package models

import "time"

type FriendLinkCategory struct {
	ID          int64
	Name        string
	Description string
	SortOrder   int
	Links       []FriendLink
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FriendLink struct {
	ID          int64
	CategoryID  int64
	Category    FriendLinkCategory
	Name        string
	URL         string
	Description string
	ImageURL    string
	SortOrder   int
	Visible     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
