package objectstorage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
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
	ErrChecksumMismatch        = errors.New("checksum mismatch")
)

type ObjectStorage struct {
	bucketDB metadata.BucketMetadataRepository
	objectDB metadata.ObjectMetadataRepository
	disk     *disk.Disk
	checksum helpers.CheckSummer
	logger   *slog.Logger
}

func NewObjectStorage(
	metadata metadata.BucketMetadataRepository,
	objectMetadata metadata.ObjectMetadataRepository,
	disk *disk.Disk,
	checksum helpers.CheckSummer,
	logger *slog.Logger,
) *ObjectStorage {
	return &ObjectStorage{
		bucketDB: metadata,
		objectDB: objectMetadata,
		disk:     disk,
		checksum: checksum,
		logger:   logger,
	}
}

// logError logs the underlying cause of an error that is about to be returned
// to callers as a user-friendly sentinel (e.g. ErrInternal). It prefers the
// request-scoped logger so the line carries the request_id.
func (oe *ObjectStorage) logError(ctx context.Context, err error) {
	helpers.LoggerFromContext(ctx, oe.logger).Error("internal_error", "err_msg", err)
}

func (oe *ObjectStorage) CreateBucket(ctx context.Context, newBucket CreateBucketInput) (string, error) {
	if err := helpers.ValidateOwnerID(newBucket.OwnerID); err != nil {
		return "", ErrOwnerIDRequired
	}

	if err := helpers.ValidateBucketName(newBucket.Name); err != nil {
		return "", ErrInvalidBucketName
	}

	existing, err := oe.bucketDB.GetBucketByName(ctx, newBucket.Name)
	if err != nil && !errors.Is(err, metadata.ErrBucketNotFound) {
		oe.logError(ctx, err)
		return "", ErrInternal
	}
	if err == nil {
		if existing.OwnerID == newBucket.OwnerID {
			return "", ErrBucketAlreadyOwnedByYou
		}
		return "", ErrBucketAlreadyExists
	}

	bucketID, err := oe.bucketDB.CreateBucket(ctx, model.Bucket{
		Name:      newBucket.Name,
		OwnerID:   newBucket.OwnerID,
		Visibilty: model.Private,
	})
	if err != nil {
		oe.logError(ctx, err)
		return "", ErrInternal
	}

	return bucketID, nil
}

func (oe *ObjectStorage) GetBucket(ctx context.Context, input GetBucketInput) (*model.Bucket, error) {
	if err := helpers.ValidateOwnerID(input.OwnerID); err != nil {
		return nil, ErrOwnerIDRequired
	}

	bucket, err := oe.bucketDB.GetBucket(ctx, input.Name, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return nil, ErrBucketNotFound
		default:
			oe.logError(ctx, err)
			return nil, ErrInternal
		}
	}

	return &bucket, nil
}

func (oe *ObjectStorage) ListBuckets(ctx context.Context, ownerID string) ([]model.Bucket, error) {
	if err := helpers.ValidateOwnerID(ownerID); err != nil {
		return nil, ErrOwnerIDRequired
	}

	buckets, err := oe.bucketDB.ListBuckets(ctx, ownerID)
	if err != nil {
		oe.logError(ctx, err)
		return nil, ErrInternal
	}

	return buckets, nil
}

func (oe *ObjectStorage) DeleteBucket(ctx context.Context, input DeleteBucketInput) error {
	if err := helpers.ValidateOwnerID(input.OwnerID); err != nil {
		return ErrOwnerIDRequired
	}

	_, err := oe.bucketDB.GetBucket(ctx, input.Name, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return ErrBucketNotFound
		default:
			oe.logError(ctx, err)
			return ErrInternal
		}
	}

	err = oe.bucketDB.DeleteBucket(ctx, input.Name, input.OwnerID)
	if err != nil {
		oe.logError(ctx, err)
		return ErrInternal
	}

	return nil
}

func (oe *ObjectStorage) SetBucketVisibility(ctx context.Context, input SetBucketVisibilityInput) error {
	if err := helpers.ValidateOwnerID(input.OwnerID); err != nil {
		return ErrOwnerIDRequired
	}

	_, err := oe.bucketDB.GetBucket(ctx, input.Name, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return ErrBucketNotFound
		default:
			oe.logError(ctx, err)
			return ErrInternal
		}
	}

	err = oe.bucketDB.UpdateBucket(ctx, input.Name, input.OwnerID, model.UpdateBucketInput{
		Visibilty: &input.Visibilty,
	})
	if err != nil {
		oe.logError(ctx, err)
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

	bucket, err := oe.bucketDB.GetBucket(ctx, input.BucketName, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return "", ErrBucketNotFound
		default:
			oe.logError(ctx, err)
			return "", ErrInternal
		}
	}

	hash := oe.checksum.Hash()
	cr := &countingReader{src: io.TeeReader(input.Content, hash)}
	hashedBucketID := helpers.HashName(bucket.ID)
	hashedKey := helpers.HashName(input.Name)

	if code, err := oe.disk.Write(disk.DataInput{
		Src:          cr,
		Filename:     hashedKey,
		Directory:    input.OwnerID,
		Subdirectory: hashedBucketID,
	}); err != nil || code != 0 {
		oe.logError(ctx, err)
		return "", ErrInternal
	}

	checksum := hash.Sum()

	objectID, err := oe.objectDB.CreateObject(ctx, model.Object{
		BucketID:  bucket.ID,
		Key:       input.Name,
		Path:      filepath.Join(hashedBucketID, hashedKey),
		Size:      int(cr.n),
		Sha256sum: checksum,
	})

	if err != nil {
		oe.logError(ctx, err)
		return "", ErrInternal
	}

	return objectID, nil
}

func (oe *ObjectStorage) GetObject(ctx context.Context, input DownloadObjectInput) (io.ReadCloser, error) {
	buck, err := oe.bucketDB.GetBucketByName(ctx, input.BucketName)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return nil, ErrBucketNotFound
		default:
			oe.logError(ctx, err)
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
			oe.logError(ctx, err)
			return nil, ErrInternal
		}
	}
	defer rc.Close()

	hash := oe.checksum.Hash()
	var buf bytes.Buffer
	if _, err := io.Copy(io.MultiWriter(hash, &buf), rc); err != nil {
		oe.logError(ctx, err)
		return nil, ErrInternal
	}
	if hash.Sum() != objectMetadata.Sha256sum {
		return nil, ErrChecksumMismatch
	}

	return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func (oe *ObjectStorage) ListObjects(ctx context.Context, bucketName, ownerID string) ([]model.Object, error) {
	if err := helpers.ValidateOwnerID(ownerID); err != nil {
		return nil, ErrOwnerIDRequired
	}

	bucket, err := oe.bucketDB.GetBucket(ctx, bucketName, ownerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return nil, ErrBucketNotFound
		default:
			oe.logError(ctx, err)
			return nil, ErrInternal
		}
	}

	objects, err := oe.objectDB.ListObjects(ctx, bucket.ID, ownerID)
	if err != nil {
		oe.logError(ctx, err)
		return nil, ErrInternal
	}

	return objects, nil
}

func (oe *ObjectStorage) DeleteObject(ctx context.Context, input DeleteObjectInput) error {
	if err := helpers.ValidateOwnerID(input.OwnerID); err != nil {
		return ErrOwnerIDRequired
	}

	bucket, err := oe.bucketDB.GetBucket(ctx, input.BucketName, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return ErrBucketNotFound
		default:
			oe.logError(ctx, err)
			return ErrInternal
		}
	}

	object, err := oe.objectDB.GetObject(ctx, bucket.ID, input.OwnerID, input.Name)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrObjectNotFound):
			return ErrObjectNotFound
		default:
			oe.logError(ctx, err)
			return ErrInternal
		}
	}

	if err := oe.disk.Delete(input.OwnerID, object.Path); err != nil {
		oe.logError(ctx, err)
		return ErrInternal
	}

	if err := oe.objectDB.DeleteObject(ctx, bucket.ID, input.OwnerID, input.Name); err != nil {
		oe.logError(ctx, err)
		return ErrInternal
	}

	return nil
}
