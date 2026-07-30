package model

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
}
