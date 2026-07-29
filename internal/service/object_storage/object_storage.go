package objectstorage

import (
	"context"
	"errors"

	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/faissalmaulana/cairo/internal/model"
	"github.com/faissalmaulana/cairo/internal/repository/metadata"
)

type CreateBucketInput struct {
	Name    string
	OwnerID string
}

type GetBucketInput struct {
	Name    string
	OwnerID string
}

var (
	ErrBucketAlreadyExists     = errors.New("bucket already exists")
	ErrBucketAlreadyOwnedByYou = errors.New("bucket already owned by you")
	ErrBucketNotFound          = errors.New("bucket not found")
	ErrInternal                = errors.New("internal error")
	ErrInvalidBucketName       = errors.New("invalid bucket name")
	ErrOwnerIDRequired         = errors.New("owner's ID is required")
)

type ObjectStorage struct {
	metadataDB metadata.MetadataRepository
}

func NewObjectStorage(metadata metadata.MetadataRepository) *ObjectStorage {
	return &ObjectStorage{
		metadataDB: metadata,
	}
}

func (oe *ObjectStorage) CreateBucket(ctx context.Context, newBucket CreateBucketInput) (string, error) {
	if err := helpers.ValidateOwnerID(newBucket.OwnerID); err != nil {
		return "", ErrOwnerIDRequired
	}

	if err := helpers.ValidateBucketName(newBucket.Name); err != nil {
		return "", ErrInvalidBucketName
	}

	bucketID, err := oe.metadataDB.CreateBucket(ctx, model.Bucket{
		Name:    newBucket.Name,
		OwnerID: newBucket.OwnerID,
	})

	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketAlreadyExists):
			return "", ErrBucketAlreadyExists
		case errors.Is(err, metadata.ErrBucketAlreadyOwnedByYou):
			return "", ErrBucketAlreadyOwnedByYou
		default:
			return "", ErrInternal
		}
	}

	return bucketID, nil
}

func (oe *ObjectStorage) GetBucket(ctx context.Context, input GetBucketInput) (*model.Bucket, error) {
	if err := helpers.ValidateOwnerID(input.OwnerID); err != nil {
		return nil, ErrOwnerIDRequired
	}

	bucket, err := oe.metadataDB.GetBucket(ctx, input.Name, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return nil, ErrBucketNotFound
		default:
			return nil, ErrInternal
		}
	}

	return &bucket, nil
}
