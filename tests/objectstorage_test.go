package e2e

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/stretchr/testify/require"
)

const (
	storageUsername = "e2estorage"
	storageEmail    = "e2estorage@example.com"
	storagePassword = "password123"
)

type storageAuth struct {
	AccountID string
	APIKey    string
}

// setupStorageUser signs up a fresh user with default credentials, resolves
// its account id, and issues an api key authenticating object-storage requests
// for that account.
func setupStorageUser(t *testing.T, router http.Handler) storageAuth {
	t.Helper()

	return setupStorageUserWith(t, router, storageUsername, storageEmail)
}

// setupStorageUserWith is setupStorageUser with explicit credentials, so tests
// that need a second (distinct) account in the same database can do so.
func setupStorageUserWith(t *testing.T, router http.Handler, username, email string) storageAuth {
	t.Helper()

	tokens := signUp(t, router, handler.SignUpRequest{
		Username: username,
		Email:    email,
		Password: storagePassword,
	})
	require.NotEmpty(t, tokens.APIKey, "signup should return the auto-created api key")

	w := doRequest(t, router, http.MethodGet, "/api/v1/account", tokens.AccessToken, nil)
	require.Equal(t, http.StatusOK, w.Code)
	account := okResponse[handler.UserResponse](t, w)

	return storageAuth{AccountID: account.ID, APIKey: tokens.APIKey}
}

func bucketPath(accountID string) string {
	return fmt.Sprintf("/api/v1/accounts/%s/buckets", accountID)
}

func bucketNamePath(accountID, name string) string {
	return bucketPath(accountID) + "/" + name
}

func objectPath(accountID, bucketName, key string) string {
	return bucketNamePath(accountID, bucketName) + "/objects/" + key
}

// publicObjectPath is the unauthenticated route that serves an object from a
// public bucket, identified only by bucket name, with no account id or api key.
func publicObjectPath(bucketName, key string) string {
	return "/api/v1/public/buckets/" + bucketName + "/objects/" + key
}

// doUpload posts a file as multipart/form-data under the "file" field with the
// given filename, keeping the object key in the URL path.
func doUpload(t *testing.T, router http.Handler, method, path, apiKey, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req, err := http.NewRequest(method, path, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

type messageResponse struct {
	Message string `json:"message"`
}

func TestBucketAuthRequired(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	// No api key on the header.
	w := doRequest(t, router, http.MethodGet, bucketPath(auth.AccountID), "", nil)
	failResponse(t, w, http.StatusUnauthorized, "API_KEY_REQUIRED")
}

func TestCreateAndGetBucket(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	w := doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "my-bucket"})
	require.Equal(t, http.StatusCreated, w.Code)
	msg := okResponse[messageResponse](t, w)
	require.Equal(t, "bucket created", msg.Message)

	w = doRequest(t, router, http.MethodGet, bucketNamePath(auth.AccountID, "my-bucket"), auth.APIKey, nil)
	require.Equal(t, http.StatusOK, w.Code)
	bucket := okResponse[handler.BucketResponse](t, w)
	require.Equal(t, "my-bucket", bucket.Name)
	require.NotEmpty(t, bucket.ID)
	require.Equal(t, "private", bucket.Visibility)
	require.NotEmpty(t, bucket.CreatedAt)
}

func TestCreateBucketValidation(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	ok := doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "my-bucket"})
	require.Equal(t, http.StatusCreated, ok.Code)

	// Duplicate bucket name by the same owner.
	dup := doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "my-bucket"})
	failResponse(t, dup, http.StatusConflict, "BUCKET_EXISTS")

	// Invalid bucket name.
	invalid := doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "UPPERCASE"})
	failResponse(t, invalid, http.StatusBadRequest, "INVALID_BUCKET_NAME")
}

func TestListBuckets(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	t.Run("no buckets yet", func(t *testing.T) {
		w := doRequest(t, router, http.MethodGet, bucketPath(auth.AccountID), auth.APIKey, nil)
		require.Equal(t, http.StatusOK, w.Code)
		buckets := okResponse[[]handler.BucketResponse](t, w)
		require.Empty(t, buckets)
	})

	t.Run("after creating buckets", func(t *testing.T) {
		for _, name := range []string{"alpha", "beta"} {
			require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: name}).Code)
		}

		w := doRequest(t, router, http.MethodGet, bucketPath(auth.AccountID), auth.APIKey, nil)
		require.Equal(t, http.StatusOK, w.Code)
		buckets := okResponse[[]handler.BucketResponse](t, w)
		require.Len(t, buckets, 2)
	})
}

func TestGetBucketNotFound(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	w := doRequest(t, router, http.MethodGet, bucketNamePath(auth.AccountID, "missing"), auth.APIKey, nil)
	failResponse(t, w, http.StatusNotFound, "BUCKET_NOT_FOUND")
}

func TestBucketVisibility(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "public-ify"}).Code)

	set := doRequest(t, router, http.MethodPatch, bucketNamePath(auth.AccountID, "public-ify")+"/visibility", auth.APIKey, handler.SetBucketVisibilityRequest{SetToPublic: true})
	require.Equal(t, http.StatusOK, set.Code)
	msg := okResponse[messageResponse](t, set)
	require.Equal(t, "bucket visibility updated", msg.Message)

	w := doRequest(t, router, http.MethodGet, bucketNamePath(auth.AccountID, "public-ify"), auth.APIKey, nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "public", okResponse[handler.BucketResponse](t, w).Visibility)

	setPrivate := doRequest(t, router, http.MethodPatch, bucketNamePath(auth.AccountID, "public-ify")+"/visibility", auth.APIKey, handler.SetBucketVisibilityRequest{SetToPublic: false})
	require.Equal(t, http.StatusOK, setPrivate.Code)

	w = doRequest(t, router, http.MethodGet, bucketNamePath(auth.AccountID, "public-ify"), auth.APIKey, nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "private", okResponse[handler.BucketResponse](t, w).Visibility)
}

func TestDeleteBucket(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "to-delete"}).Code)

	del := doRequest(t, router, http.MethodDelete, bucketNamePath(auth.AccountID, "to-delete"), auth.APIKey, nil)
	require.Equal(t, http.StatusOK, del.Code)
	msg := okResponse[messageResponse](t, del)
	require.Equal(t, "bucket deleted", msg.Message)

	// Deleted bucket is no longer retrievable.
	w := doRequest(t, router, http.MethodGet, bucketNamePath(auth.AccountID, "to-delete"), auth.APIKey, nil)
	failResponse(t, w, http.StatusNotFound, "BUCKET_NOT_FOUND")
}

func TestDeletePublicBucketRemovesPublicAccess(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "public-delete"}).Code)
	require.Equal(t, http.StatusOK, doRequest(t, router, http.MethodPatch, bucketNamePath(auth.AccountID, "public-delete")+"/visibility", auth.APIKey, handler.SetBucketVisibilityRequest{SetToPublic: true}).Code)

	content := []byte("visible before delete")
	require.Equal(t, http.StatusCreated, doUpload(t, router, http.MethodPut, objectPath(auth.AccountID, "public-delete", "share.txt"), auth.APIKey, "share.txt", content).Code)

	// The object is reachable in the public namespace before deletion.
	w := doRequest(t, router, http.MethodGet, publicObjectPath("public-delete", "share.txt"), "", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, content, w.Body.Bytes())

	// Deleting the bucket must revoke public access.
	require.Equal(t, http.StatusOK, doRequest(t, router, http.MethodDelete, bucketNamePath(auth.AccountID, "public-delete"), auth.APIKey, nil).Code)

	// Gone from both the authenticated and the public namespace.
	w = doRequest(t, router, http.MethodGet, bucketNamePath(auth.AccountID, "public-delete"), auth.APIKey, nil)
	failResponse(t, w, http.StatusNotFound, "BUCKET_NOT_FOUND")
	w = doRequest(t, router, http.MethodGet, publicObjectPath("public-delete", "share.txt"), "", nil)
	failResponse(t, w, http.StatusNotFound, "BUCKET_NOT_FOUND")
}

func TestBucketOwnershipIsolation(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	owner := setupStorageUser(t, router)
	intruder := setupStorageUserWith(t, router, "e2eintruder", "e2eintruder@example.com")

	require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(owner.AccountID), owner.APIKey, handler.CreateBucketRequest{Name: "guarded"}).Code)

	// The intruder scoping its request to its own account id cannot see or
	// delete the owner's bucket.
	get := doRequest(t, router, http.MethodGet, bucketNamePath(intruder.AccountID, "guarded"), intruder.APIKey, nil)
	failResponse(t, get, http.StatusNotFound, "BUCKET_NOT_FOUND")

	del := doRequest(t, router, http.MethodDelete, bucketNamePath(intruder.AccountID, "guarded"), intruder.APIKey, nil)
	failResponse(t, del, http.StatusNotFound, "BUCKET_NOT_FOUND")
}

func TestUploadAndDownloadObject(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "files"}).Code)

	content := []byte("hello cairo object storage")
	up := doUpload(t, router, http.MethodPut, objectPath(auth.AccountID, "files", "note.txt"), auth.APIKey, "note.txt", content)
	require.Equal(t, http.StatusCreated, up.Code)
	msg := okResponse[messageResponse](t, up)
	require.Equal(t, "object uploaded", msg.Message)

	down := doRequest(t, router, http.MethodGet, objectPath(auth.AccountID, "files", "note.txt"), auth.APIKey, nil)
	require.Equal(t, http.StatusOK, down.Code)
	require.Equal(t, "application/octet-stream", down.Header().Get("Content-Type"))
	require.Equal(t, content, down.Body.Bytes())
}

func TestUploadObjectWithNestedKey(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "nested"}).Code)

	// Slashes in the key must survive URL encoding intact (catch-all param).
	content := []byte("nested file bytes")
	up := doUpload(t, router, http.MethodPut, objectPath(auth.AccountID, "nested", "dir/sub/file.txt"), auth.APIKey, "file.txt", content)
	require.Equal(t, http.StatusCreated, up.Code)

	down := doRequest(t, router, http.MethodGet, objectPath(auth.AccountID, "nested", "dir/sub/file.txt"), auth.APIKey, nil)
	require.Equal(t, http.StatusOK, down.Code)
	require.Equal(t, content, down.Body.Bytes())
}

func TestListObjects(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "content"}).Code)

	contentA := []byte("aaa")
	contentB := []byte("bbbbb")
	require.Equal(t, http.StatusCreated, doUpload(t, router, http.MethodPut, objectPath(auth.AccountID, "content", "a.txt"), auth.APIKey, "a.txt", contentA).Code)
	require.Equal(t, http.StatusCreated, doUpload(t, router, http.MethodPut, objectPath(auth.AccountID, "content", "b.txt"), auth.APIKey, "b.txt", contentB).Code)

	w := doRequest(t, router, http.MethodGet, bucketNamePath(auth.AccountID, "content")+"/objects", auth.APIKey, nil)
	require.Equal(t, http.StatusOK, w.Code)
	objects := okResponse[[]handler.ObjectResponse](t, w)
	require.Len(t, objects, 2)

	got := map[string]handler.ObjectResponse{}
	for _, obj := range objects {
		got[obj.Key] = obj
	}

	a := got["a.txt"]
	require.Equal(t, len(contentA), a.Size)
	require.Equal(t, checksumOfBytes(contentA), a.Sha256sum)

	b := got["b.txt"]
	require.Equal(t, len(contentB), b.Size)
	require.Equal(t, checksumOfBytes(contentB), b.Sha256sum)
}

func TestDeleteObject(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "trash"}).Code)

	require.Equal(t, http.StatusCreated, doUpload(t, router, http.MethodPut, objectPath(auth.AccountID, "trash", "junk.txt"), auth.APIKey, "junk.txt", []byte("junk")).Code)

	del := doRequest(t, router, http.MethodDelete, objectPath(auth.AccountID, "trash", "junk.txt"), auth.APIKey, nil)
	require.Equal(t, http.StatusOK, del.Code)
	msg := okResponse[messageResponse](t, del)
	require.Equal(t, "object deleted", msg.Message)

	// No longer retrievable.
	down := doRequest(t, router, http.MethodGet, objectPath(auth.AccountID, "trash", "junk.txt"), auth.APIKey, nil)
	failResponse(t, down, http.StatusNotFound, "OBJECT_NOT_FOUND")
}

func TestObjectErrors(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "errors"}).Code)

	t.Run("uploading to a missing bucket", func(t *testing.T) {
		w := doUpload(t, router, http.MethodPut, objectPath(auth.AccountID, "nope", "f.txt"), auth.APIKey, "f.txt", []byte("x"))
		failResponse(t, w, http.StatusNotFound, "BUCKET_NOT_FOUND")
	})

	t.Run("downloading a missing object", func(t *testing.T) {
		w := doRequest(t, router, http.MethodGet, objectPath(auth.AccountID, "errors", "missing.txt"), auth.APIKey, nil)
		failResponse(t, w, http.StatusNotFound, "OBJECT_NOT_FOUND")
	})

	t.Run("listing objects in a missing bucket", func(t *testing.T) {
		w := doRequest(t, router, http.MethodGet, bucketNamePath(auth.AccountID, "nope")+"/objects", auth.APIKey, nil)
		failResponse(t, w, http.StatusNotFound, "BUCKET_NOT_FOUND")
	})
}

func TestPublicObjectAccess(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "public-assets"}).Code)
	require.Equal(t, http.StatusOK, doRequest(t, router, http.MethodPatch, bucketNamePath(auth.AccountID, "public-assets")+"/visibility", auth.APIKey, handler.SetBucketVisibilityRequest{SetToPublic: true}).Code)

	content := []byte("publicly readable bytes")
	require.Equal(t, http.StatusCreated, doUpload(t, router, http.MethodPut, objectPath(auth.AccountID, "public-assets", "readme.txt"), auth.APIKey, "readme.txt", content).Code)

	// Public buckets are served with no account id and no api key.
	w := doRequest(t, router, http.MethodGet, publicObjectPath("public-assets", "readme.txt"), "", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
	require.Equal(t, content, w.Body.Bytes())

	// Nested keys survive on the public route too.
	nested := []byte("nested public bytes")
	require.Equal(t, http.StatusCreated, doUpload(t, router, http.MethodPut, objectPath(auth.AccountID, "public-assets", "dir/sub/file.txt"), auth.APIKey, "file.txt", nested).Code)
	w = doRequest(t, router, http.MethodGet, publicObjectPath("public-assets", "dir/sub/file.txt"), "", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, nested, w.Body.Bytes())

	// A missing object on a public bucket is a normal not-found.
	w = doRequest(t, router, http.MethodGet, publicObjectPath("public-assets", "missing.txt"), "", nil)
	failResponse(t, w, http.StatusNotFound, "OBJECT_NOT_FOUND")

	// An unknown public bucket is a not-found.
	w = doRequest(t, router, http.MethodGet, publicObjectPath("does-not-exist", "x.txt"), "", nil)
	failResponse(t, w, http.StatusNotFound, "BUCKET_NOT_FOUND")
}

func TestPrivateObjectNotPubliclyAccessible(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	auth := setupStorageUser(t, router)

	// The bucket is left private by default.
	require.Equal(t, http.StatusCreated, doRequest(t, router, http.MethodPost, bucketPath(auth.AccountID), auth.APIKey, handler.CreateBucketRequest{Name: "guarded"}).Code)
	require.Equal(t, http.StatusCreated, doUpload(t, router, http.MethodPut, objectPath(auth.AccountID, "guarded", "secret.txt"), auth.APIKey, "secret.txt", []byte("secret")).Code)

	// Even though the object exists, the public route must refuse to serve it.
	w := doRequest(t, router, http.MethodGet, publicObjectPath("guarded", "secret.txt"), "", nil)
	failResponse(t, w, http.StatusForbidden, "FORBIDDEN")

	// The owner can still fetch it through the authenticated route.
	w = doRequest(t, router, http.MethodGet, objectPath(auth.AccountID, "guarded", "secret.txt"), auth.APIKey, nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []byte("secret"), w.Body.Bytes())
}

// checksumOfBytes returns the hex sha256 of data, mirroring how the server
// computes object checksums.
func checksumOfBytes(data []byte) string {
	h := helpers.GenerateSHA256()
	h.Write(data)
	return h.Sum()
}
