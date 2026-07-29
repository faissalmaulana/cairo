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

type DeleteBucketInput struct {
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

	existing, err := oe.metadataDB.GetBucketByName(ctx, newBucket.Name)
	if err != nil && !errors.Is(err, metadata.ErrBucketNotFound) {
		return "", ErrInternal
	}
	if err == nil {
		if existing.OwnerID == newBucket.OwnerID {
			return "", ErrBucketAlreadyOwnedByYou
		}
		return "", ErrBucketAlreadyExists
	}

	bucketID, err := oe.metadataDB.CreateBucket(ctx, model.Bucket{
		Name:    newBucket.Name,
		OwnerID: newBucket.OwnerID,
	})
	if err != nil {
		return "", ErrInternal
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

func (oe *ObjectStorage) ListBuckets(ctx context.Context, ownerID string) ([]model.Bucket, error) {
	if err := helpers.ValidateOwnerID(ownerID); err != nil {
		return nil, ErrOwnerIDRequired
	}

	buckets, err := oe.metadataDB.ListBuckets(ctx, ownerID)
	if err != nil {
		return nil, ErrInternal
	}

	return buckets, nil
}

func (oe *ObjectStorage) DeleteBucket(ctx context.Context, input DeleteBucketInput) error {
	if err := helpers.ValidateOwnerID(input.OwnerID); err != nil {
		return ErrOwnerIDRequired
	}

	_, err := oe.metadataDB.GetBucket(ctx, input.Name, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return ErrBucketNotFound
		default:
			return ErrInternal
		}
	}

	err = oe.metadataDB.DeleteBucket(ctx, input.Name, input.OwnerID)
	if err != nil {
		return ErrInternal
	}

	return nil
}
