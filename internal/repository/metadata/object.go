package metadata

import (
	"context"
	"errors"

	"github.com/faissalmaulana/cairo/internal/model"
)

type ObjectMetadataRepository interface {
	CreateObject(ctx context.Context, object model.Object) (string, error)
	GetObject(ctx context.Context, bucketID, ownerID, name string) (model.Object, error)
	ListObjects(ctx context.Context, bucketID, ownerID string) ([]model.Object, error)
	DeleteObject(ctx context.Context, bucketID, ownerID, name string) error
}

var (
	ErrObjectNotFound     = errors.New("object not found")
	ErrCannotCreateObject = errors.New("object can't create something went wrong")
	ErrCannotGetObject    = errors.New("object can't get something went wrong")
	ErrCannotDeleteObject = errors.New("object can't delete something went wrong")
)
