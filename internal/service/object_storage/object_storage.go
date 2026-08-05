package objectstorage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
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
	Name       string
	OwnerID    string
	Visibility model.BucketVisibility
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
		Name:       newBucket.Name,
		OwnerID:    newBucket.OwnerID,
		Visibility: model.Private,
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

	bucket, err := oe.bucketDB.GetBucket(ctx, input.Name, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return ErrBucketNotFound
		default:
			oe.logError(ctx, err)
			return ErrInternal
		}
	}

	// A public bucket has a symlink in the public namespace; remove it before
	// deleting the bucket so the public path cannot dangle or point at a
	// directory whose objects are being torn down.
	if bucket.Visibility == model.Public {
		if err := oe.unlinkBucket(ctx, bucket); err != nil {
			return err
		}
	}

	// Objects no longer cascade from buckets, so their rows must be removed
	// before the bucket row itself.
	if err := oe.objectDB.DeleteObjectsByBucket(ctx, bucket.ID); err != nil {
		oe.logError(ctx, err)
		return ErrInternal
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

	bucket, err := oe.bucketDB.GetBucket(ctx, input.Name, input.OwnerID)
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
		Visibility: &input.Visibility,
	})
	if err != nil {
		oe.logError(ctx, err)
		return ErrInternal
	}

	if input.Visibility == model.Public {
		if err := oe.linkBucket(ctx, bucket); err != nil {
			return err
		}
		return nil
	}

	if err := oe.unlinkBucket(ctx, bucket); err != nil {
		return err
	}

	return nil
}

// bucketHash returns the persisted bucket hash, falling back to deriving it
// from the bucket's ID for rows created before the hash was stored.
func bucketHash(bucket model.Bucket) string {
	if bucket.BucketHash != "" {
		return bucket.BucketHash
	}
	return helpers.HashName(bucket.ID)
}

// lookupContentType derives a MIME type from the object key's extension, falling
// back to application/octet-stream when it cannot be determined.
func lookupContentType(key string) string {
	if ct := mime.TypeByExtension(filepath.Ext(key)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// linkBucket exposes the bucket's directory through the public namespace under
// its bare bucket hash, so objects can be read publicly without revealing the
// owner's account directory.
func (oe *ObjectStorage) linkBucket(ctx context.Context, bucket model.Bucket) error {
	if err := oe.disk.Link(filepath.Join(bucket.OwnerID, bucketHash(bucket)), bucketHash(bucket)); err != nil {
		oe.logError(ctx, err)
		return ErrInternal
	}
	return nil
}

// unlinkBucket removes the bucket's directory from the public namespace.
func (oe *ObjectStorage) unlinkBucket(ctx context.Context, bucket model.Bucket) error {
	if err := oe.disk.Unlink(bucketHash(bucket)); err != nil {
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
	hashedBucketID := bucketHash(bucket)
	hashedKey := helpers.HashName(input.Name)
	contentType := lookupContentType(input.Name)

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
		BucketID:    bucket.ID,
		Key:         input.Name,
		Path:        filepath.Join(hashedBucketID, hashedKey),
		Size:        int(cr.n),
		Sha256sum:   checksum,
		ContentType: contentType,
	})

	if err != nil {
		oe.logError(ctx, err)
		return "", ErrInternal
	}

	return objectID, nil
}

// This for getting private object
func (oe *ObjectStorage) GetObject(ctx context.Context, input DownloadObjectInput) (io.ReadCloser, model.Object, error) {
	if err := helpers.ValidateOwnerID(input.OwnerID); err != nil {
		return nil, model.Object{}, ErrOwnerIDRequired
	}

	buck, err := oe.bucketDB.GetBucket(ctx, input.BucketName, input.OwnerID)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			return nil, model.Object{}, ErrBucketNotFound
		default:
			oe.logError(ctx, err)
			return nil, model.Object{}, ErrInternal
		}
	}

	objectMetadata, err := oe.objectDB.GetObject(ctx, buck.ID, input.Name)
	if err != nil {
		return nil, model.Object{}, ErrObjectNotFound
	}

	rc, err := oe.disk.Read(buck.OwnerID, objectMetadata.Path)
	if err != nil {
		switch {
		case errors.Is(err, disk.ErrFileNotFound):
			return nil, model.Object{}, ErrObjectNotFound
		default:
			oe.logError(ctx, err)
			return nil, model.Object{}, ErrInternal
		}
	}

	verified, err := oe.verifyAndBuffer(ctx, rc, objectMetadata)
	return verified, objectMetadata, err
}

// GetPublicObject returns an object from a public bucket without requiring an
// owner's account id. Access is granted purely by the bucket's visibility; the
// file is read through the public symlink namespace so the owner's directory is
// never used to locate it. The verified bytes are streamed back together with
// the stored object metadata (including its persisted content type).
func (oe *ObjectStorage) GetPublicObject(ctx context.Context, bucketName, name string) (io.ReadCloser, model.Object, error) {
	bucket, err := oe.bucketDB.GetBucketByName(ctx, bucketName)
	if err != nil {
		switch {
		case errors.Is(err, metadata.ErrBucketNotFound):
			fmt.Println("HELLOOOO 3", err)
			return nil, model.Object{}, ErrBucketNotFound
		default:
			oe.logError(ctx, err)
			return nil, model.Object{}, ErrInternal
		}
	}

	if bucket.Visibility != model.Public {
		return nil, model.Object{}, ErrUnauthorized
	}

	objectMetadata, err := oe.objectDB.GetObject(ctx, bucket.ID, name)
	if err != nil {
		fmt.Println("HELLOOOO 2", err)
		return nil, model.Object{}, ErrObjectNotFound
	}

	rc, err := oe.disk.Read("public", objectMetadata.Path)
	if err != nil {
		switch {
		case errors.Is(err, disk.ErrFileNotFound), errors.Is(err, disk.ErrDirectoryNotFound):
			fmt.Println("HELLOOOO FROM DISSKKKK", err)
			return nil, model.Object{}, ErrObjectNotFound
		default:
			oe.logError(ctx, err)
			return nil, model.Object{}, ErrInternal
		}
	}

	verified, err := oe.verifyAndBuffer(ctx, rc, objectMetadata)
	return verified, objectMetadata, err
}

// verifyAndBuffer reads an object stream in full, recomputing and comparing its
// sha256 against the stored checksum, and returns the verified content buffered
// in memory. Callers are expected to close the returned stream.
func (oe *ObjectStorage) verifyAndBuffer(ctx context.Context, rc io.ReadCloser, objectMetadata model.Object) (io.ReadCloser, error) {
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

	objects, err := oe.objectDB.ListObjects(ctx, bucket.ID)
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

	object, err := oe.objectDB.GetObject(ctx, bucket.ID, input.Name)
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

	if err := oe.objectDB.DeleteObject(ctx, bucket.ID, input.Name); err != nil {
		oe.logError(ctx, err)
		return ErrInternal
	}

	return nil
}
