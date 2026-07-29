package objectstorage

import (
	"context"
	"errors"
	"regexp"

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
	/**
	 * Bucket names can only contain lowercase letters (a-z), numbers (0-9), and hyphens (-).
	 * Bucket names cannot begin or end with a hyphen.
	 * Bucket names can only be between 3-63 characters in length.
	 */
	bucketNameRegex        = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	ErrInvalidBucketName   = errors.New("invalid bucket name")
	ErrNewBucketOwnerEmpty = errors.New("owner's ID is required")
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
	if len(newBucket.OwnerID) == 0 {
		return "", ErrNewBucketOwnerEmpty
	}

	if !bucketNameRegex.MatchString(newBucket.Name) || (len(newBucket.Name) < 3 || len(newBucket.Name) > 63) {
		return "", ErrInvalidBucketName
	}

	bucketID, err := oe.metadataDB.CreateBucket(ctx, model.Bucket{
		Name:    newBucket.Name,
		OwnerID: newBucket.OwnerID,
	})

	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketAlreadyExists):
			return "", metadata.ErrBucketAlreadyExists
		case errors.Is(err, metadata.ErrBucketAlreadyOwnedByYou):
			return "", metadata.ErrBucketAlreadyOwnedByYou
		default:
			return "", metadata.ErrCannotCreateBucket
		}
	}

	return bucketID, nil
}

func (oe *ObjectStorage) GetBucket(ctx context.Context, input GetBucketInput) (*model.Bucket, error) {
	if len(input.OwnerID) == 0 {
		return nil, ErrNewBucketOwnerEmpty
	}

	bucket, err := oe.metadataDB.GetBucket(ctx, input.Name, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return nil, metadata.ErrBucketNotFound
		default:
			return nil, metadata.ErrCannotGetBucket
		}
	}

	return &bucket, nil
}
