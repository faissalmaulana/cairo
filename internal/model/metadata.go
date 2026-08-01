package model

import "time"

type BucketVisibility int

func (bv BucketVisibility) String() string {
	return []string{"private", "public"}[bv]
}

const (
	Private BucketVisibility = iota
	Public
)

type Bucket struct {
	ID        string
	Name      string
	OwnerID   string
	Visibilty BucketVisibility
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Object struct {
	Id string
	// object key
	BucketName string
	Key        string
	Size       int
	Sha256sum  string
}

type UpdateBucketInput struct {
	Visibilty *BucketVisibility
}
