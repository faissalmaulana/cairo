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
	ID string
	// object key
	BucketID string
	Key      string
	// path is the combination of bucketID and key as filepath
	Path      string
	Size      int
	Sha256sum string
}

type UpdateBucketInput struct {
	Visibilty *BucketVisibility
}
