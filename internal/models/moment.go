package models

import "time"

type Moment struct {
	ID        int64
	Content   string
	Author    string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
