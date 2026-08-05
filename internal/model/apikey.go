package model

import "time"

type ApiKey struct {
	ID         string
	UserID     string
	Key        string
	LastUsedAt *time.Time
	CreatedAt  time.Time
}
