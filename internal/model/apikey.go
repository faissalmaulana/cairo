package model

import "time"

type ApiKey struct {
	ID         string
	UserID     string
	KeyHash    string `json:"-"`
	Prefix     string
	LastUsedAt *time.Time
	CreatedAt  time.Time
}
