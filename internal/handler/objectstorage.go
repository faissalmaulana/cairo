package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/faissalmaulana/cairo/internal/model"
	objectstorage "github.com/faissalmaulana/cairo/internal/service/object_storage"
	"github.com/gin-gonic/gin"
)

type ObjectStorageHandler struct {
	objectStorage *objectstorage.ObjectStorage
}

func NewObjectStorageHandler(objectStorage *objectstorage.ObjectStorage) *ObjectStorageHandler {
	return &ObjectStorageHandler{
		objectStorage: objectStorage,
	}
}

type BucketResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type ObjectResponse struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Size      int    `json:"size"`
	Sha256sum string `json:"sha256sum"`
}

func toBucketResponse(bucket model.Bucket) BucketResponse {
	return BucketResponse{
		ID:         bucket.ID,
		Name:       bucket.Name,
		Visibility: bucket.Visibility.String(),
		CreatedAt:  bucket.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  bucket.UpdatedAt.Format(time.RFC3339),
	}
}

func toObjectResponse(obj model.Object) ObjectResponse {
	return ObjectResponse{
		ID:        obj.ID,
		Key:       obj.Key,
		Size:      obj.Size,
		Sha256sum: obj.Sha256sum,
	}
}

// objectKey returns the object key from the catch-all path param, stripping the
// leading slash so keys such as "dir/file.txt" survive URL encoding intact.
func objectKey(c *gin.Context) string {
	return strings.TrimPrefix(c.Param("object_key"), "/")
}

// accountID returns the authenticated user id resolved from the api key row in
// the database, set by ApiKeyMiddleware.CheckApiKey. The :account_id path param
// has already been matched against it by RequireAccount; handlers always act on
// the database-backed id rather than re-trusting the URL.
func accountID(c *gin.Context) string {
	return c.GetString(helpers.AuthUserIDKey)
}

func (oh *ObjectStorageHandler) handleObjectStorageError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, objectstorage.ErrOwnerIDRequired):
		FailError(c, ErrOwnerIDRequired)
	case errors.Is(err, objectstorage.ErrInvalidBucketName):
		FailError(c, ErrInvalidBucketName)
	case errors.Is(err, objectstorage.ErrBucketAlreadyOwnedByYou), errors.Is(err, objectstorage.ErrBucketAlreadyExists):
		FailError(c, ErrBucketAlreadyExists)
	case errors.Is(err, objectstorage.ErrBucketNotFound):
		FailError(c, ErrBucketNotFound)
	case errors.Is(err, objectstorage.ErrObjectNotFound):
		FailError(c, ErrObjectNotFound)
	case errors.Is(err, objectstorage.ErrUnauthorized):
		FailError(c, ErrBucketForbidden)
	case errors.Is(err, objectstorage.ErrBucketNotEmpty):
		FailError(c, ErrBucketNotEmpty)
	case errors.Is(err, objectstorage.ErrChecksumMismatch):
		FailError(c, ErrChecksumMismatch)
	default:
		FailError(c, ErrInternalServer)
	}
}

func (oh *ObjectStorageHandler) ListBuckets(c *gin.Context) {
	buckets, err := oh.objectStorage.ListBuckets(c.Request.Context(), accountID(c))
	if err != nil {
		oh.handleObjectStorageError(c, err)
		return
	}

	resp := make([]BucketResponse, 0, len(buckets))
	for _, bucket := range buckets {
		resp = append(resp, toBucketResponse(bucket))
	}

	OK(c, http.StatusOK, resp)
}

type CreateBucketRequest struct {
	Name string `json:"name" binding:"required"`
}

func (oh *ObjectStorageHandler) CreateBucket(c *gin.Context) {
	var req CreateBucketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailError(c, ErrValidation(err))
		return
	}

	_, err := oh.objectStorage.CreateBucket(c.Request.Context(), objectstorage.CreateBucketInput{
		Name:    req.Name,
		OwnerID: accountID(c),
	})
	if err != nil {
		oh.handleObjectStorageError(c, err)
		return
	}

	OK(c, http.StatusCreated, gin.H{"message": "bucket created"})
}

func (oh *ObjectStorageHandler) GetBucket(c *gin.Context) {
	bucket, err := oh.objectStorage.GetBucket(c.Request.Context(), objectstorage.GetBucketInput{
		Name:    c.Param("bucket_name"),
		OwnerID: accountID(c),
	})
	if err != nil {
		oh.handleObjectStorageError(c, err)
		return
	}

	OK(c, http.StatusOK, toBucketResponse(*bucket))
}

type SetBucketVisibilityRequest struct {
	SetToPublic bool `json:"set_to_public"`
}

func (oh *ObjectStorageHandler) SetBucketVisibility(c *gin.Context) {
	var req SetBucketVisibilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		FailError(c, ErrValidation(err))
		return
	}

	visibility := model.Private
	if req.SetToPublic {
		visibility = model.Public
	}

	err := oh.objectStorage.SetBucketVisibility(c.Request.Context(), objectstorage.SetBucketVisibilityInput{
		Name:       c.Param("bucket_name"),
		OwnerID:    accountID(c),
		Visibility: visibility,
	})
	if err != nil {
		oh.handleObjectStorageError(c, err)
		return
	}

	OK(c, http.StatusOK, gin.H{"message": "bucket visibility updated"})
}

func (oh *ObjectStorageHandler) DeleteBucket(c *gin.Context) {
	err := oh.objectStorage.DeleteBucket(c.Request.Context(), objectstorage.DeleteBucketInput{
		Name:    c.Param("bucket_name"),
		OwnerID: accountID(c),
	})
	if err != nil {
		oh.handleObjectStorageError(c, err)
		return
	}

	OK(c, http.StatusOK, gin.H{"message": "bucket deleted"})
}

func (oh *ObjectStorageHandler) ListObjects(c *gin.Context) {
	objects, err := oh.objectStorage.ListObjects(c.Request.Context(), c.Param("bucket_name"), accountID(c))
	if err != nil {
		oh.handleObjectStorageError(c, err)
		return
	}

	resp := make([]ObjectResponse, 0, len(objects))
	for _, obj := range objects {
		resp = append(resp, toObjectResponse(obj))
	}

	OK(c, http.StatusOK, resp)
}

func (oh *ObjectStorageHandler) UploadObject(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		FailError(c, ErrValidation(errors.New("file field is required")))
		return
	}

	src, err := file.Open()
	if err != nil {
		FailError(c, ErrValidation(err))
		return
	}
	defer src.Close()

	_, err = oh.objectStorage.UploadObject(c.Request.Context(), objectstorage.UploadObjectInput{
		BucketName: c.Param("bucket_name"),
		OwnerID:    accountID(c),
		Name:       objectKey(c),
		Content:    src,
	})
	if err != nil {
		oh.handleObjectStorageError(c, err)
		return
	}

	OK(c, http.StatusCreated, gin.H{"message": "object uploaded"})
}

func (oh *ObjectStorageHandler) GetObject(c *gin.Context) {
	rc, object, err := oh.objectStorage.GetObject(c.Request.Context(), objectstorage.DownloadObjectInput{
		BucketName: c.Param("bucket_name"),
		OwnerID:    accountID(c),
		Name:       objectKey(c),
	})
	if err != nil {
		oh.handleObjectStorageError(c, err)
		return
	}

	oh.streamObject(c, rc, object)
}

// GetPublicObject serves an object from a public bucket without requiring an
// account id or api key. Access is granted purely by the bucket's visibility.
func (oh *ObjectStorageHandler) GetPublicObject(c *gin.Context) {
	rc, object, err := oh.objectStorage.GetPublicObject(c.Request.Context(), c.Param("bucket_name"), objectKey(c))
	if err != nil {
		oh.handleObjectStorageError(c, err)
		return
	}

	oh.streamObject(c, rc, object)
}

// streamObject writes a verified object stream directly to the response
// without buffering the whole body in the handler. The content type comes
// from the stored object metadata; content length from the persisted size.
func (oh *ObjectStorageHandler) streamObject(c *gin.Context, rc io.ReadCloser, object model.Object) {
	defer rc.Close()

	header := c.Writer.Header()
	header.Set("Content-Type", object.ContentType)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Content-Length", strconv.Itoa(object.Size))

	if _, err := io.Copy(c.Writer, rc); err != nil {
		FailError(c, ErrInternalServer)
		return
	}
}

func (oh *ObjectStorageHandler) DeleteObject(c *gin.Context) {
	err := oh.objectStorage.DeleteObject(c.Request.Context(), objectstorage.DeleteObjectInput{
		BucketName: c.Param("bucket_name"),
		OwnerID:    accountID(c),
		Name:       objectKey(c),
	})
	if err != nil {
		oh.handleObjectStorageError(c, err)
		return
	}

	OK(c, http.StatusOK, gin.H{"message": "object deleted"})
}
