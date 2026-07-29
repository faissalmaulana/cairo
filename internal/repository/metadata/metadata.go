package metadata

import (
	"context"
	"errors"

	"github.com/faissalmaulana/cairo/internal/model"
)

type MetadataRepository interface {
	CreateBucket(ctx context.Context, newBucket model.Bucket) (string, error)
	GetBucket(ctx context.Context, name string, ownerID string) (model.Bucket, error)
}

type CreateBucketInput struct {
	Name    string
	OwnerID string
}

var (
	ErrBucketAlreadyExists     = errors.New("bucket already exists")
	ErrBucketAlreadyOwnedByYou = errors.New("bucket already owned by you")
	ErrCannotCreateBucket      = errors.New("bucket can't create something went wrong")
	ErrBucketNotFound          = errors.New("bucket not found")
	ErrCannotGetBucket         = errors.New("bucket can't get something went wrong")
)
