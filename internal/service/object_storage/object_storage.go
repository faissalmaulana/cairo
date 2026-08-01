package objectstorage

import (
	"context"
	"errors"
	"io"
	"path/filepath"

	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/faissalmaulana/cairo/internal/model"
	"github.com/faissalmaulana/cairo/internal/repository/metadata"
	"github.com/faissalmaulana/cairo/internal/service/disk"
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

type SetBucketVisibilityInput struct {
	Name      string
	OwnerID   string
	Visibilty model.BucketVisibility
}

type UploadObjectInput struct {
	BucketName string
	OwnerID    string
	Name       string
	Content    io.Reader
}

type DownloadObjectInput struct {
	BucketName string
	OwnerID    string
	Name       string
}

type DeleteObjectInput struct {
	BucketName string
	OwnerID    string
	Name       string
}

var (
	ErrBucketAlreadyExists     = errors.New("bucket already exists")
	ErrBucketAlreadyOwnedByYou = errors.New("bucket already owned by you")
	ErrBucketNotFound          = errors.New("bucket not found")
	ErrInternal                = errors.New("internal error")
	ErrInvalidBucketName       = errors.New("invalid bucket name")
	ErrObjectNotFound          = errors.New("object not found")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrOwnerIDRequired         = errors.New("owner's ID is required")
)

type ObjectStorage struct {
	metadataDB metadata.MetadataRepository
	objectDB   metadata.ObjectMetadataRepository
	disk       *disk.Disk
}

func NewObjectStorage(metadata metadata.MetadataRepository, objectMetadata metadata.ObjectMetadataRepository, disk *disk.Disk) *ObjectStorage {
	return &ObjectStorage{
		metadataDB: metadata,
		objectDB:   objectMetadata,
		disk:       disk,
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
		Name:      newBucket.Name,
		OwnerID:   newBucket.OwnerID,
		Visibilty: model.Private,
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

func (oe *ObjectStorage) SetBucketVisibility(ctx context.Context, input SetBucketVisibilityInput) error {
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

	err = oe.metadataDB.UpdateBucket(ctx, input.Name, input.OwnerID, model.UpdateBucketInput{
		Visibilty: &input.Visibilty,
	})
	if err != nil {
		return ErrInternal
	}

	return nil
}

type countingReader struct {
	src io.Reader
	n   int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.src.Read(p)
	cr.n += int64(n)
	return n, err
}

func (oe *ObjectStorage) UploadObject(ctx context.Context, input UploadObjectInput) (string, error) {
	if err := helpers.ValidateOwnerID(input.OwnerID); err != nil {
		return "", ErrOwnerIDRequired
	}

	bucket, err := oe.metadataDB.GetBucket(ctx, input.BucketName, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return "", ErrBucketNotFound
		default:
			return "", ErrInternal
		}
	}

	cr := &countingReader{src: input.Content}
	hashedBucketID := helpers.HashName(bucket.ID)
	hashedKey := helpers.HashName(input.Name)

	if code, err := oe.disk.Write(disk.DataInput{
		Src:          cr,
		Filename:     hashedKey,
		Directory:    input.OwnerID,
		Subdirectory: hashedBucketID,
	}); err != nil || code != 0 {
		return "", ErrInternal
	}

	objectID, err := oe.objectDB.CreateObject(ctx, model.Object{
		BucketID: bucket.ID,
		Key:      input.Name,
		Path:     filepath.Join(hashedBucketID, hashedKey),
		Size:     int(cr.n),
	})
	if err != nil {
		return "", ErrInternal
	}

	return objectID, nil
}

func (oe *ObjectStorage) DownloadObject(ctx context.Context, input DownloadObjectInput) (io.ReadCloser, error) {
	buck, err := oe.metadataDB.GetBucketByName(ctx, input.BucketName)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return nil, ErrBucketNotFound
		default:
			return nil, ErrInternal
		}
	}

	if buck.Visibilty == model.Private {
		if err := helpers.ValidateOwnerID(input.OwnerID); err != nil {
			return nil, ErrOwnerIDRequired
		}
		if buck.OwnerID != input.OwnerID {
			return nil, ErrUnauthorized
		}
	}

	objectMetadata, err := oe.objectDB.GetObject(ctx, buck.ID, buck.OwnerID, input.Name)
	if err != nil {
		return nil, ErrObjectNotFound
	}

	rc, err := oe.disk.Read(buck.OwnerID, objectMetadata.Path)
	if err != nil {
		switch {
		case errors.Is(err, disk.ErrFileNotFound):
			return nil, ErrObjectNotFound
		default:
			return nil, ErrInternal
		}
	}

	return rc, nil
}

func (oe *ObjectStorage) ListObjects(ctx context.Context, bucketName, ownerID string) ([]model.Object, error) {
	if err := helpers.ValidateOwnerID(ownerID); err != nil {
		return nil, ErrOwnerIDRequired
	}

	bucket, err := oe.metadataDB.GetBucket(ctx, bucketName, ownerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return nil, ErrBucketNotFound
		default:
			return nil, ErrInternal
		}
	}

	objects, err := oe.objectDB.ListObjects(ctx, bucket.ID, ownerID)
	if err != nil {
		return nil, ErrInternal
	}

	return objects, nil
}

func (oe *ObjectStorage) DeleteObject(ctx context.Context, input DeleteObjectInput) error {
	if err := helpers.ValidateOwnerID(input.OwnerID); err != nil {
		return ErrOwnerIDRequired
	}

	bucket, err := oe.metadataDB.GetBucket(ctx, input.BucketName, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return ErrBucketNotFound
		default:
			return ErrInternal
		}
	}

	object, err := oe.objectDB.GetObject(ctx, bucket.ID, input.OwnerID, input.Name)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrObjectNotFound):
			return ErrObjectNotFound
		default:
			return ErrInternal
		}
	}

	if err := oe.disk.Delete(input.OwnerID, object.Path); err != nil {
		return ErrInternal
	}

	if err := oe.objectDB.DeleteObject(ctx, bucket.ID, input.OwnerID, input.Name); err != nil {
		return ErrInternal
	}

	return nil
}
