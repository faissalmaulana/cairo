package objectstorage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"log/slog"

	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/faissalmaulana/cairo/internal/model"
	"github.com/faissalmaulana/cairo/internal/repository/metadata"
	"github.com/faissalmaulana/cairo/internal/service/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// discardLogger returns an slog.Logger that drops everything, keeping tests
// that exercise service errors quiet while satisfying the injected logger.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type MockBucketMetadataRepository struct {
	mock.Mock
}

func (mr *MockBucketMetadataRepository) CreateBucket(ctx context.Context, newBucket model.Bucket) (string, error) {
	args := mr.Mock.Called(ctx, newBucket)
	return args.String(0), args.Error(1)
}

func (mr *MockBucketMetadataRepository) GetBucket(ctx context.Context, name string, ownerID string) (model.Bucket, error) {
	args := mr.Mock.Called(ctx, name, ownerID)
	return args.Get(0).(model.Bucket), args.Error(1)
}

func (mr *MockBucketMetadataRepository) GetBucketByName(ctx context.Context, name string) (model.Bucket, error) {
	args := mr.Mock.Called(ctx, name)
	return args.Get(0).(model.Bucket), args.Error(1)
}

func (mr *MockBucketMetadataRepository) ListBuckets(ctx context.Context, ownerID string) ([]model.Bucket, error) {
	args := mr.Mock.Called(ctx, ownerID)
	return args.Get(0).([]model.Bucket), args.Error(1)
}

func (mr *MockBucketMetadataRepository) UpdateBucket(ctx context.Context, name string, ownerID string, update model.UpdateBucketInput) error {
	args := mr.Mock.Called(ctx, name, ownerID, update)
	return args.Error(0)
}

func (mr *MockBucketMetadataRepository) ReplaceBucket(ctx context.Context, bucket model.Bucket) error {
	args := mr.Mock.Called(ctx, bucket)
	return args.Error(0)
}

func (mr *MockBucketMetadataRepository) DeleteBucket(ctx context.Context, name string, ownerID string) error {
	args := mr.Mock.Called(ctx, name, ownerID)
	return args.Error(0)
}

type MockObjectMetadataRepository struct {
	mock.Mock
}

func (mor *MockObjectMetadataRepository) CreateObject(ctx context.Context, object model.Object) (string, error) {
	args := mor.Mock.Called(ctx, object)
	return args.String(0), args.Error(1)
}

func (mor *MockObjectMetadataRepository) GetObject(ctx context.Context, bucketID, ownerID, name string) (model.Object, error) {
	args := mor.Mock.Called(ctx, bucketID, ownerID, name)
	return args.Get(0).(model.Object), args.Error(1)
}

func (mor *MockObjectMetadataRepository) ListObjects(ctx context.Context, bucketID, ownerID string) ([]model.Object, error) {
	args := mor.Mock.Called(ctx, bucketID, ownerID)
	return args.Get(0).([]model.Object), args.Error(1)
}

func (mor *MockObjectMetadataRepository) DeleteObject(ctx context.Context, bucketID, ownerID, name string) error {
	args := mor.Mock.Called(ctx, bucketID, ownerID, name)
	return args.Error(0)
}

type MockChecksum struct {
	mock.Mock
}

func (mc *MockChecksum) Hash() helpers.Hash {
	args := mc.Mock.Called()
	return args.Get(0).(helpers.Hash)
}

func checksumOf(data string) string {
	h := helpers.GenerateSHA256()
	h.Write([]byte(data))
	return h.Sum()
}

func setupTest(t *testing.T) (*MockBucketMetadataRepository, *ObjectStorage) {
	m := new(MockBucketMetadataRepository)
	om := new(MockObjectMetadataRepository)
	cm := new(MockChecksum)
	return m, NewObjectStorage(m, om, disk.NewDisk(t.TempDir()), cm, discardLogger())
}

func setupObjectTest(t *testing.T) (*MockBucketMetadataRepository, *MockObjectMetadataRepository, *ObjectStorage) {
	m := new(MockBucketMetadataRepository)
	om := new(MockObjectMetadataRepository)
	cm := new(MockChecksum)
	return m, om, NewObjectStorage(m, om, disk.NewDisk(t.TempDir()), cm, discardLogger())
}

func TestCreateBucket(t *testing.T) {
	t.Run("cannot create new bucket because owner's ID is not provided", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)
		newBucket := CreateBucketInput{
			Name:    "avatars",
			OwnerID: "",
		}

		_, err := objectStorage.CreateBucket(context.Background(), newBucket)
		assert.ErrorIs(t, err, ErrOwnerIDRequired)
		mockMetadata.AssertNotCalled(t, "GetBucketByName", mock.Anything, mock.Anything)
		mockMetadata.AssertNotCalled(t, "CreateBucket", mock.Anything, mock.Anything)
	})

	testCases := []struct {
		name  string
		input CreateBucketInput
	}{
		{
			name: "cannot create new bucket invalid name (min leng)",
			input: CreateBucketInput{
				Name:    "pr",
				OwnerID: "asdjsaodsaidhisadiisad",
			},
		},
		{
			name: "cannot create new bucket invalid name (max leng)",
			input: CreateBucketInput{
				Name:    "asdfghjklasd9asi-absduad82-2aishdiahsdiashdihiahd0-chaushcachiahcuahsuha",
				OwnerID: "asdjsaodsaidhisadiisad",
			},
		},
		{
			name: "cannot create new bucket invalid name (prefix with hypen)",
			input: CreateBucketInput{
				Name:    "-asdfghjklasd9asi-absduad82",
				OwnerID: "asdjsaodsaidhisadiisad",
			},
		},
		{
			name: "cannot create new bucket invalid name (suffix with hypen)",
			input: CreateBucketInput{
				Name:    "asdfghjklasd9asi-absduad82-",
				OwnerID: "asdjsaodsaidhisadiisad",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mockMetadata, objectStorage := setupTest(t)

			_, err := objectStorage.CreateBucket(context.Background(), tc.input)

			assert.ErrorIs(t, err, ErrInvalidBucketName)
			mockMetadata.AssertNotCalled(t, "GetBucketByName", mock.Anything, mock.Anything)
			mockMetadata.AssertNotCalled(t, "CreateBucket", mock.Anything, mock.Anything)
		})
	}

	t.Run("cannot create new bucket because already exist", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)
		newBucket := CreateBucketInput{
			Name:    "profile-users",
			OwnerID: "12345678",
		}

		mockMetadata.On("GetBucketByName", mock.Anything, newBucket.Name).
			Return(model.Bucket{Name: newBucket.Name, OwnerID: "different-owner"}, nil).Once()

		_, err := objectStorage.CreateBucket(context.Background(), newBucket)

		assert.ErrorIs(t, err, ErrBucketAlreadyExists)
		mockMetadata.AssertExpectations(t)
		mockMetadata.AssertNotCalled(t, "CreateBucket", mock.Anything, mock.Anything)
	})

	t.Run("cannot create new bucket because the user already own it", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)

		newBucket := CreateBucketInput{
			Name:    "profile-users",
			OwnerID: "12345678",
		}

		mockMetadata.On("GetBucketByName", mock.Anything, newBucket.Name).
			Return(model.Bucket{Name: newBucket.Name, OwnerID: newBucket.OwnerID}, nil).Once()

		_, err := objectStorage.CreateBucket(context.Background(), newBucket)

		assert.ErrorIs(t, err, ErrBucketAlreadyOwnedByYou)
		mockMetadata.AssertExpectations(t)
		mockMetadata.AssertNotCalled(t, "CreateBucket", mock.Anything, mock.Anything)
	})

	t.Run("cannot create new bucket, something went wrong with the metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)

		newBucket := CreateBucketInput{
			Name:    "profile-users",
			OwnerID: "12345678",
		}

		mockMetadata.On("GetBucketByName", mock.Anything, newBucket.Name).
			Return(model.Bucket{}, metadata.ErrBucketNotFound).Once()
		mockMetadata.On("CreateBucket", mock.Anything, model.Bucket{
			Name:    newBucket.Name,
			OwnerID: newBucket.OwnerID,
		}).Return("", assert.AnError).Once()

		_, err := objectStorage.CreateBucket(context.Background(), newBucket)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
	})

	t.Run("success create bucket", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)

		expctedBucketID := "asdijkacuubosj12"
		newBucket := CreateBucketInput{
			Name:    "avatars",
			OwnerID: "872371727",
		}

		mockMetadata.On("GetBucketByName", mock.Anything, newBucket.Name).
			Return(model.Bucket{}, metadata.ErrBucketNotFound).Once()
		mockMetadata.On("CreateBucket", mock.Anything, model.Bucket{
			Name:    newBucket.Name,
			OwnerID: newBucket.OwnerID,
		}).Return(expctedBucketID, nil).Once()

		bucketID, err := objectStorage.CreateBucket(context.Background(), newBucket)

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
		assert.Equal(t, expctedBucketID, bucketID)
	})
}

func TestGetBucket(t *testing.T) {
	t.Run("cannot get bucket because owner's ID is not provided", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)
		input := GetBucketInput{
			Name:    "avatars",
			OwnerID: "",
		}

		_, err := objectStorage.GetBucket(context.Background(), input)
		assert.ErrorIs(t, err, ErrOwnerIDRequired)
		mockMetadata.AssertNotCalled(t, "GetBucket", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot get bucket because not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)

		input := GetBucketInput{
			Name:    "not-found-bucket",
			OwnerID: "12345678",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(model.Bucket{}, metadata.ErrBucketNotFound).Once()

		_, err := objectStorage.GetBucket(context.Background(), input)

		assert.ErrorIs(t, err, ErrBucketNotFound)
		mockMetadata.AssertExpectations(t)
	})

	t.Run("cannot get bucket, something went wrong with the metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)

		input := GetBucketInput{
			Name:    "profile-users",
			OwnerID: "12345678",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(model.Bucket{}, metadata.ErrCannotGetBucket).Once()

		_, err := objectStorage.GetBucket(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
	})

	t.Run("success get bucket", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)

		expectedBucket := model.Bucket{
			ID:      "asdijkacuubosj12",
			Name:    "avatars",
			OwnerID: "872371727",
		}

		input := GetBucketInput{
			Name:    expectedBucket.Name,
			OwnerID: expectedBucket.OwnerID,
		}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(expectedBucket, nil).Once()

		bucket, err := objectStorage.GetBucket(context.Background(), input)

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
		assert.Equal(t, &expectedBucket, bucket)
	})
}

func TestListBuckets(t *testing.T) {
	t.Run("cannot list buckets because owner's ID is not provided", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)

		_, err := objectStorage.ListBuckets(context.Background(), "")
		assert.ErrorIs(t, err, ErrOwnerIDRequired)
		mockMetadata.AssertNotCalled(t, "ListBuckets", mock.Anything, mock.Anything)
	})

	t.Run("cannot list buckets, something went wrong with the metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)

		mockMetadata.On("ListBuckets", mock.Anything, "user-123").
			Return([]model.Bucket{}, assert.AnError).Once()

		_, err := objectStorage.ListBuckets(context.Background(), "user-123")

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
	})

	t.Run("success list buckets empty", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)

		mockMetadata.On("ListBuckets", mock.Anything, "user-123").
			Return([]model.Bucket{}, nil).Once()

		buckets, err := objectStorage.ListBuckets(context.Background(), "user-123")

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
		assert.Empty(t, buckets)
	})

	t.Run("success list buckets", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)

		expected := []model.Bucket{
			{ID: "id-1", Name: "avatars", OwnerID: "user-123"},
			{ID: "id-2", Name: "thumbnails", OwnerID: "user-123"},
		}

		mockMetadata.On("ListBuckets", mock.Anything, "user-123").
			Return(expected, nil).Once()

		buckets, err := objectStorage.ListBuckets(context.Background(), "user-123")

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
		assert.Equal(t, expected, buckets)
	})
}

func TestDeleteBucket(t *testing.T) {
	t.Run("cannot delete bucket because owner's ID is not provided", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)
		input := DeleteBucketInput{
			Name:    "avatars",
			OwnerID: "",
		}

		err := objectStorage.DeleteBucket(context.Background(), input)
		assert.ErrorIs(t, err, ErrOwnerIDRequired)
		mockMetadata.AssertNotCalled(t, "DeleteBucket", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot delete bucket because not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)
		input := DeleteBucketInput{
			Name:    "not-found-bucket",
			OwnerID: "user-123",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(model.Bucket{}, metadata.ErrBucketNotFound).Once()

		err := objectStorage.DeleteBucket(context.Background(), input)

		assert.ErrorIs(t, err, ErrBucketNotFound)
		mockMetadata.AssertExpectations(t)
		mockMetadata.AssertNotCalled(t, "DeleteBucket", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot delete bucket, something went wrong with the metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)
		input := DeleteBucketInput{
			Name:    "avatars",
			OwnerID: "user-123",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(model.Bucket{Name: "avatars", OwnerID: "user-123"}, nil).Once()
		mockMetadata.On("DeleteBucket", mock.Anything, input.Name, input.OwnerID).
			Return(assert.AnError).Once()

		err := objectStorage.DeleteBucket(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
	})

	t.Run("success delete bucket", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)
		input := DeleteBucketInput{
			Name:    "avatars",
			OwnerID: "user-123",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(model.Bucket{Name: "avatars", OwnerID: "user-123"}, nil).Once()
		mockMetadata.On("DeleteBucket", mock.Anything, input.Name, input.OwnerID).
			Return(nil).Once()

		err := objectStorage.DeleteBucket(context.Background(), input)

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
	})

	t.Run("success delete public bucket removes public symlink", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		mockMetadata := new(MockBucketMetadataRepository)
		objectStorage := NewObjectStorage(mockMetadata, new(MockObjectMetadataRepository), disk.NewDisk(entry), new(MockChecksum), discardLogger())

		bucket := model.Bucket{
			ID:         "bucket-1",
			Name:       "shared",
			OwnerID:    "user-123",
			Visibility: model.Public,
			BucketHash: helpers.HashName("bucket-1"),
		}
		publicDir := filepath.Join(entry, "public", bucket.BucketHash)
		require.NoError(t, disk.NewDisk(entry).Link(filepath.Join(bucket.OwnerID, bucket.BucketHash), bucket.BucketHash))

		input := DeleteBucketInput{Name: bucket.Name, OwnerID: bucket.OwnerID}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(bucket, nil).Once()
		mockMetadata.On("DeleteBucket", mock.Anything, input.Name, input.OwnerID).
			Return(nil).Once()

		err := objectStorage.DeleteBucket(context.Background(), input)

		assert.NoError(t, err)
		assert.NoFileExists(t, publicDir, "public symlink should be removed before deleting the bucket")
		mockMetadata.AssertExpectations(t)
	})

	t.Run("cannot delete public bucket when public symlink already missing", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		mockMetadata := new(MockBucketMetadataRepository)
		objectStorage := NewObjectStorage(mockMetadata, new(MockObjectMetadataRepository), disk.NewDisk(entry), new(MockChecksum), discardLogger())

		bucket := model.Bucket{
			ID:         "bucket-1",
			Name:       "shared",
			OwnerID:    "user-123",
			Visibility: model.Public,
			BucketHash: helpers.HashName("bucket-1"),
		}
		input := DeleteBucketInput{Name: bucket.Name, OwnerID: bucket.OwnerID}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(bucket, nil).Once()

		err := objectStorage.DeleteBucket(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertNotCalled(t, "DeleteBucket", mock.Anything, mock.Anything, mock.Anything)
		mockMetadata.AssertExpectations(t)
	})
}

func TestSetBucketVisibility(t *testing.T) {
	t.Run("cannot set visibility because owner's ID is not provided", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)
		input := SetBucketVisibilityInput{
			Name:       "avatars",
			OwnerID:    "",
			Visibility: model.Public,
		}

		err := objectStorage.SetBucketVisibility(context.Background(), input)
		assert.ErrorIs(t, err, ErrOwnerIDRequired)
		mockMetadata.AssertNotCalled(t, "GetBucket", mock.Anything, mock.Anything, mock.Anything)
		mockMetadata.AssertNotCalled(t, "UpdateBucket", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot set visibility because bucket not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)
		input := SetBucketVisibilityInput{
			Name:       "not-found-bucket",
			OwnerID:    "user-123",
			Visibility: model.Public,
		}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(model.Bucket{}, metadata.ErrBucketNotFound).Once()

		err := objectStorage.SetBucketVisibility(context.Background(), input)

		assert.ErrorIs(t, err, ErrBucketNotFound)
		mockMetadata.AssertExpectations(t)
		mockMetadata.AssertNotCalled(t, "UpdateBucket", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot set visibility, something went wrong with the metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)
		input := SetBucketVisibilityInput{
			Name:       "avatars",
			OwnerID:    "user-123",
			Visibility: model.Public,
		}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(model.Bucket{Name: "avatars", OwnerID: "user-123"}, nil).Once()
		mockMetadata.On("UpdateBucket", mock.Anything, input.Name, input.OwnerID, mock.Anything).
			Return(assert.AnError).Once()

		err := objectStorage.SetBucketVisibility(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
	})

	t.Run("success set bucket visibility to public", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest(t)
		input := SetBucketVisibilityInput{
			Name:       "avatars",
			OwnerID:    "user-123",
			Visibility: model.Public,
		}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(model.Bucket{Name: "avatars", OwnerID: "user-123"}, nil).Once()
		mockMetadata.On("UpdateBucket", mock.Anything, input.Name, input.OwnerID, model.UpdateBucketInput{
			Visibility: &input.Visibility,
		}).Return(nil).Once()

		err := objectStorage.SetBucketVisibility(context.Background(), input)

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
	})

	t.Run("success set bucket visibility to public creates a disk symlink", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}

		mockMetadata := new(MockBucketMetadataRepository)
		objectStorage := NewObjectStorage(mockMetadata, new(MockObjectMetadataRepository), disk.NewDisk(entry), new(MockChecksum), discardLogger())
		input := SetBucketVisibilityInput{
			Name:       "avatars",
			OwnerID:    "user-123",
			Visibility: model.Public,
		}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(bucket, nil).Once()
		mockMetadata.On("UpdateBucket", mock.Anything, input.Name, input.OwnerID, model.UpdateBucketInput{
			Visibility: &input.Visibility,
		}).Return(nil).Once()

		err := objectStorage.SetBucketVisibility(context.Background(), input)

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)

		link := filepath.Join(entry, "public", helpers.HashName(bucket.ID))
		info, err := os.Lstat(link)
		require.NoError(t, err)
		assert.True(t, info.Mode()&os.ModeSymlink != 0, "public bucket should be symlinked")
	})

	t.Run("success set bucket visibility to private removes the disk symlink", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}

		mockMetadata := new(MockBucketMetadataRepository)
		objectStorage := NewObjectStorage(mockMetadata, new(MockObjectMetadataRepository), disk.NewDisk(entry), new(MockChecksum), discardLogger())
		public := model.Public
		private := model.Private

		mockMetadata.On("GetBucket", mock.Anything, "avatars", "user-123").
			Return(bucket, nil).Twice()
		mockMetadata.On("UpdateBucket", mock.Anything, "avatars", "user-123", model.UpdateBucketInput{
			Visibility: &public,
		}).Return(nil).Once()
		mockMetadata.On("UpdateBucket", mock.Anything, "avatars", "user-123", model.UpdateBucketInput{
			Visibility: &private,
		}).Return(nil).Once()

		require.NoError(t, objectStorage.SetBucketVisibility(context.Background(), SetBucketVisibilityInput{
			Name: "avatars", OwnerID: "user-123", Visibility: public,
		}))
		require.NoError(t, objectStorage.SetBucketVisibility(context.Background(), SetBucketVisibilityInput{
			Name: "avatars", OwnerID: "user-123", Visibility: private,
		}))

		link := filepath.Join(entry, "public", helpers.HashName(bucket.ID))
		_, err := os.Lstat(link)
		assert.True(t, os.IsNotExist(err), "public symlink should be removed")
		mockMetadata.AssertExpectations(t)
	})
}

func TestUploadObject(t *testing.T) {
	t.Run("cannot upload object because owner's ID is not provided", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		input := UploadObjectInput{
			BucketName: "avatars",
			OwnerID:    "",
			Name:       "haaland.png",
			Content:    strings.NewReader("data"),
		}

		_, err := objectStorage.UploadObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrOwnerIDRequired)
		mockMetadata.AssertNotCalled(t, "GetBucket", mock.Anything, mock.Anything, mock.Anything)
		mockObjectMetadata.AssertNotCalled(t, "CreateObject", mock.Anything, mock.Anything)
	})

	t.Run("cannot upload object because bucket not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		input := UploadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
			Content:    strings.NewReader("data"),
		}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(model.Bucket{}, metadata.ErrBucketNotFound).Once()

		_, err := objectStorage.UploadObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrBucketNotFound)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "CreateObject", mock.Anything, mock.Anything)
	})

	t.Run("cannot upload object, something went wrong with the bucket metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		input := UploadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
			Content:    strings.NewReader("data"),
		}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(model.Bucket{}, assert.AnError).Once()

		_, err := objectStorage.UploadObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "CreateObject", mock.Anything, mock.Anything)
	})

	t.Run("cannot upload object because disk write failed", func(t *testing.T) {
		t.Parallel()
		entry := filepath.Join(t.TempDir(), "entry")
		require.NoError(t, os.WriteFile(entry, []byte("x"), 0o644))

		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		mockChecksum := new(MockChecksum)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), mockChecksum, discardLogger())

		input := UploadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
			Content:    strings.NewReader("data"),
		}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}, nil).Once()
		mockChecksum.On("Hash").Return(helpers.GenerateSHA256()).Once()

		_, err := objectStorage.UploadObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "CreateObject", mock.Anything, mock.Anything)
	})

	t.Run("cannot upload object because object metadata commit failed", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		mockChecksum := new(MockChecksum)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), mockChecksum, discardLogger())

		input := UploadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
			Content:    strings.NewReader("HELLO,WORLD"),
		}

		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}
		hashedBucketID := helpers.HashName(bucket.ID)
		hashedKey := helpers.HashName(input.Name)

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(bucket, nil).Once()
		mockChecksum.On("Hash").Return(helpers.GenerateSHA256()).Once()
		mockObjectMetadata.On("CreateObject", mock.Anything, model.Object{
			BucketID:    bucket.ID,
			Key:         input.Name,
			Path:        filepath.Join(hashedBucketID, hashedKey),
			Size:        len("HELLO,WORLD"),
			Sha256sum:   checksumOf("HELLO,WORLD"),
			ContentType: "image/png",
		}).Return("", assert.AnError).Once()

		_, err := objectStorage.UploadObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
	})

	t.Run("success upload object", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		mockChecksum := new(MockChecksum)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), mockChecksum, discardLogger())

		input := UploadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
			Content:    strings.NewReader("HELLO,WORLD"),
		}

		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}
		hashedBucketID := helpers.HashName(bucket.ID)
		hashedKey := helpers.HashName(input.Name)
		expectedObjectID := "object-id-123"

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(bucket, nil).Once()
		mockChecksum.On("Hash").Return(helpers.GenerateSHA256()).Once()

		mockObjectMetadata.On("CreateObject", mock.Anything, model.Object{
			BucketID:    bucket.ID,
			Key:         input.Name,
			Path:        filepath.Join(hashedBucketID, hashedKey),
			Size:        len("HELLO,WORLD"),
			Sha256sum:   checksumOf("HELLO,WORLD"),
			ContentType: "image/png",
		}).Return(expectedObjectID, nil).Once()

		objectID, err := objectStorage.UploadObject(context.Background(), input)

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
		assert.Equal(t, expectedObjectID, objectID)

		got, err := os.ReadFile(filepath.Join(entry, input.OwnerID, hashedBucketID, hashedKey))
		require.NoError(t, err)
		assert.Equal(t, "HELLO,WORLD", string(got))
	})
}

func TestGetPrivatePrObject(t *testing.T) {
	t.Run("cannot get object because bucket not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		input := DownloadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(model.Bucket{}, metadata.ErrBucketNotFound).Once()

		_, _, err := objectStorage.GetObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrBucketNotFound)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "GetObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot get object, something went wrong with the bucket metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		input := DownloadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(model.Bucket{}, assert.AnError).Once()

		_, _, err := objectStorage.GetObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "GetObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot get object from private bucket because owner's ID is not provided", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		input := DownloadObjectInput{
			BucketName: "avatars",
			OwnerID:    "",
			Name:       "haaland.png",
		}

		_, _, err := objectStorage.GetObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrOwnerIDRequired)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "GetBucket", mock.Anything, mock.Anything, mock.Anything)
		mockObjectMetadata.AssertNotCalled(t, "GetObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot get object because object metadata not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		input := DownloadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "missing.png",
		}

		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123", Visibility: model.Private}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, bucket.OwnerID, input.Name).
			Return(model.Object{}, metadata.ErrObjectNotFound).Once()

		_, _, err := objectStorage.GetObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrObjectNotFound)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
	})

	t.Run("cannot get object because file not found on disk", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), new(MockChecksum), discardLogger())

		input := DownloadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123", Visibility: model.Private}
		object := model.Object{ID: "object-1", BucketID: bucket.ID, Key: input.Name, Path: "some/hash"}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, bucket.OwnerID, input.Name).
			Return(object, nil).Once()

		_, _, err := objectStorage.GetObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrObjectNotFound)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
	})

	t.Run("cannot get object, something went wrong reading from disk", func(t *testing.T) {
		t.Parallel()
		entry := filepath.Join(t.TempDir(), "entry")
		require.NoError(t, os.WriteFile(entry, []byte("x"), 0o644))

		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), new(MockChecksum), discardLogger())

		input := DownloadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123", Visibility: model.Private}
		object := model.Object{ID: "object-1", BucketID: bucket.ID, Key: input.Name, Path: "some/hash"}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, bucket.OwnerID, input.Name).
			Return(object, nil).Once()

		_, _, err := objectStorage.GetObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
	})

	t.Run("success get object from private bucket", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123", Visibility: model.Private}
		object := model.Object{ID: "object-1", BucketID: bucket.ID, Key: "haaland.png", Path: "some/hash",
			Sha256sum: checksumOf("HELLO,WORLD")}
		file := filepath.Join(entry, bucket.OwnerID, object.Path)
		require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
		require.NoError(t, os.WriteFile(file, []byte("HELLO,WORLD"), 0o644))

		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		mockChecksum := new(MockChecksum)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), mockChecksum, discardLogger())

		input := DownloadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, bucket.OwnerID, input.Name).
			Return(object, nil).Once()
		mockChecksum.On("Hash").Return(helpers.GenerateSHA256()).Once()

		rc, _, err := objectStorage.GetObject(context.Background(), input)
		require.NoError(t, err)
		defer rc.Close()

		got, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, "HELLO,WORLD", string(got))
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
		mockChecksum.AssertExpectations(t)
	})

	t.Run("cannot get object because checksum mismatch", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123", Visibility: model.Private}
		object := model.Object{ID: "object-1", BucketID: bucket.ID, Key: "haaland.png", Path: "some/hash",
			Sha256sum: "0000000000000000000000000000000000000000000000000000000000000000"}
		file := filepath.Join(entry, bucket.OwnerID, object.Path)
		require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
		require.NoError(t, os.WriteFile(file, []byte("HELLO,WORLD"), 0o644))

		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		mockChecksum := new(MockChecksum)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), mockChecksum, discardLogger())

		input := DownloadObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, bucket.OwnerID, input.Name).
			Return(object, nil).Once()
		mockChecksum.On("Hash").Return(helpers.GenerateSHA256()).Once()

		_, _, err := objectStorage.GetObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrChecksumMismatch)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
		mockChecksum.AssertExpectations(t)
	})
}

func TestGetPublicObject(t *testing.T) {
	t.Run("cannot get object because bucket not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)

		mockMetadata.On("GetBucketByName", mock.Anything, "avatars").
			Return(model.Bucket{}, metadata.ErrBucketNotFound).Once()

		_, _, err := objectStorage.GetPublicObject(context.Background(), "avatars", "haaland.png")

		assert.ErrorIs(t, err, ErrBucketNotFound)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "GetObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot get object, something went wrong with the bucket metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)

		mockMetadata.On("GetBucketByName", mock.Anything, "avatars").
			Return(model.Bucket{}, assert.AnError).Once()

		_, _, err := objectStorage.GetPublicObject(context.Background(), "avatars", "haaland.png")

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "GetObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot get object from private bucket", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123", Visibility: model.Private}

		mockMetadata.On("GetBucketByName", mock.Anything, "avatars").
			Return(bucket, nil).Once()

		_, _, err := objectStorage.GetPublicObject(context.Background(), "avatars", "haaland.png")

		assert.ErrorIs(t, err, ErrUnauthorized)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "GetObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot get object because object metadata not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123", Visibility: model.Public}

		mockMetadata.On("GetBucketByName", mock.Anything, "avatars").
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, bucket.OwnerID, "missing.png").
			Return(model.Object{}, metadata.ErrObjectNotFound).Once()

		_, _, err := objectStorage.GetPublicObject(context.Background(), "avatars", "missing.png")

		assert.ErrorIs(t, err, ErrObjectNotFound)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
	})

	t.Run("cannot get object because file not found on disk", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), new(MockChecksum), discardLogger())

		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123", Visibility: model.Public}
		object := model.Object{ID: "object-1", BucketID: bucket.ID, Key: "haaland.png", Path: "some/hash"}

		mockMetadata.On("GetBucketByName", mock.Anything, "avatars").
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, bucket.OwnerID, "haaland.png").
			Return(object, nil).Once()

		_, _, err := objectStorage.GetPublicObject(context.Background(), "avatars", "haaland.png")

		assert.ErrorIs(t, err, ErrObjectNotFound)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
	})

	t.Run("cannot get object, something went wrong reading from disk", func(t *testing.T) {
		t.Parallel()
		entry := filepath.Join(t.TempDir(), "entry")
		require.NoError(t, os.WriteFile(entry, []byte("x"), 0o644))

		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), new(MockChecksum), discardLogger())

		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123", Visibility: model.Public}
		object := model.Object{ID: "object-1", BucketID: bucket.ID, Key: "haaland.png", Path: "some/hash"}

		mockMetadata.On("GetBucketByName", mock.Anything, "avatars").
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, bucket.OwnerID, "haaland.png").
			Return(object, nil).Once()

		_, _, err := objectStorage.GetPublicObject(context.Background(), "avatars", "haaland.png")

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
	})

	t.Run("success get object from public bucket", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123", Visibility: model.Public}
		object := model.Object{ID: "object-1", BucketID: bucket.ID, Key: "haaland.png", Path: "some/hash",
			Sha256sum: checksumOf("HELLO,WORLD")}
		file := filepath.Join(entry, "public", object.Path)
		require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
		require.NoError(t, os.WriteFile(file, []byte("HELLO,WORLD"), 0o644))

		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		mockChecksum := new(MockChecksum)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), mockChecksum, discardLogger())

		mockMetadata.On("GetBucketByName", mock.Anything, "avatars").
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, bucket.OwnerID, "haaland.png").
			Return(object, nil).Once()
		mockChecksum.On("Hash").Return(helpers.GenerateSHA256()).Once()

		rc, _, err := objectStorage.GetPublicObject(context.Background(), "avatars", "haaland.png")
		require.NoError(t, err)
		defer rc.Close()

		got, err := io.ReadAll(rc)
		require.NoError(t, err)
		assert.Equal(t, "HELLO,WORLD", string(got))
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
		mockChecksum.AssertExpectations(t)
	})

	t.Run("cannot get object because checksum mismatch", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123", Visibility: model.Public}
		object := model.Object{ID: "object-1", BucketID: bucket.ID, Key: "haaland.png", Path: "some/hash",
			Sha256sum: "0000000000000000000000000000000000000000000000000000000000000000"}
		file := filepath.Join(entry, "public", object.Path)
		require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
		require.NoError(t, os.WriteFile(file, []byte("HELLO,WORLD"), 0o644))

		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		mockChecksum := new(MockChecksum)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), mockChecksum, discardLogger())

		mockMetadata.On("GetBucketByName", mock.Anything, "avatars").
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, bucket.OwnerID, "haaland.png").
			Return(object, nil).Once()
		mockChecksum.On("Hash").Return(helpers.GenerateSHA256()).Once()

		_, _, err := objectStorage.GetPublicObject(context.Background(), "avatars", "haaland.png")

		assert.ErrorIs(t, err, ErrChecksumMismatch)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
		mockChecksum.AssertExpectations(t)
	})
}

func TestListObjects(t *testing.T) {
	t.Run("cannot list objects because owner's ID is not provided", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)

		_, err := objectStorage.ListObjects(context.Background(), "avatars", "")

		assert.ErrorIs(t, err, ErrOwnerIDRequired)
		mockMetadata.AssertNotCalled(t, "GetBucket", mock.Anything, mock.Anything, mock.Anything)
		mockObjectMetadata.AssertNotCalled(t, "ListObjects", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot list objects because bucket not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)

		mockMetadata.On("GetBucket", mock.Anything, "avatars", "user-123").
			Return(model.Bucket{}, metadata.ErrBucketNotFound).Once()

		_, err := objectStorage.ListObjects(context.Background(), "avatars", "user-123")

		assert.ErrorIs(t, err, ErrBucketNotFound)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "ListObjects", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot list objects, something went wrong with the bucket metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)

		mockMetadata.On("GetBucket", mock.Anything, "avatars", "user-123").
			Return(model.Bucket{}, assert.AnError).Once()

		_, err := objectStorage.ListObjects(context.Background(), "avatars", "user-123")

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "ListObjects", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot list objects, something went wrong with the object metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}

		mockMetadata.On("GetBucket", mock.Anything, bucket.Name, bucket.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("ListObjects", mock.Anything, bucket.ID, bucket.OwnerID).
			Return([]model.Object{}, assert.AnError).Once()

		_, err := objectStorage.ListObjects(context.Background(), bucket.Name, bucket.OwnerID)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
	})

	t.Run("success list objects empty", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}

		mockMetadata.On("GetBucket", mock.Anything, bucket.Name, bucket.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("ListObjects", mock.Anything, bucket.ID, bucket.OwnerID).
			Return([]model.Object{}, nil).Once()

		objects, err := objectStorage.ListObjects(context.Background(), bucket.Name, bucket.OwnerID)

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
		assert.Empty(t, objects)
	})

	t.Run("success list objects", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}

		expected := []model.Object{
			{ID: "id-1", BucketID: bucket.ID, Key: "haaland.png", Path: "hash/1", Size: 11},
			{ID: "id-2", BucketID: bucket.ID, Key: "debruyne.png", Path: "hash/2", Size: 10},
		}

		mockMetadata.On("GetBucket", mock.Anything, bucket.Name, bucket.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("ListObjects", mock.Anything, bucket.ID, bucket.OwnerID).
			Return(expected, nil).Once()

		objects, err := objectStorage.ListObjects(context.Background(), bucket.Name, bucket.OwnerID)

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
		assert.Equal(t, expected, objects)
	})
}

func TestDeleteObject(t *testing.T) {
	t.Run("cannot delete object because owner's ID is not provided", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		input := DeleteObjectInput{
			BucketName: "avatars",
			OwnerID:    "",
			Name:       "haaland.png",
		}

		err := objectStorage.DeleteObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrOwnerIDRequired)
		mockMetadata.AssertNotCalled(t, "GetBucket", mock.Anything, mock.Anything, mock.Anything)
		mockObjectMetadata.AssertNotCalled(t, "GetObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot delete object because bucket not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		input := DeleteObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(model.Bucket{}, metadata.ErrBucketNotFound).Once()

		err := objectStorage.DeleteObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrBucketNotFound)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "GetObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot delete object, something went wrong with the bucket metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		input := DeleteObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(model.Bucket{}, assert.AnError).Once()

		err := objectStorage.DeleteObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "GetObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot delete object because object not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, mockObjectMetadata, objectStorage := setupObjectTest(t)
		input := DeleteObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, input.OwnerID, input.Name).
			Return(model.Object{}, metadata.ErrObjectNotFound).Once()

		err := objectStorage.DeleteObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrObjectNotFound)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "DeleteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot delete object, something went wrong with the object metadata repository method", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}
		object := model.Object{ID: "object-1", BucketID: bucket.ID, Key: "haaland.png", Path: "some/hash"}
		file := filepath.Join(entry, bucket.OwnerID, object.Path)
		require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
		require.NoError(t, os.WriteFile(file, []byte("data"), 0o644))

		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), new(MockChecksum), discardLogger())

		input := DeleteObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, input.OwnerID, input.Name).
			Return(object, nil).Once()
		mockObjectMetadata.On("DeleteObject", mock.Anything, bucket.ID, input.OwnerID, input.Name).
			Return(assert.AnError).Once()

		err := objectStorage.DeleteObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
	})

	t.Run("cannot delete object because disk delete failed", func(t *testing.T) {
		t.Parallel()
		entry := filepath.Join(t.TempDir(), "entry")
		require.NoError(t, os.WriteFile(entry, []byte("x"), 0o644))

		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), new(MockChecksum), discardLogger())

		input := DeleteObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}
		object := model.Object{ID: "object-1", BucketID: bucket.ID, Key: input.Name, Path: "some/hash"}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, input.OwnerID, input.Name).
			Return(object, nil).Once()

		err := objectStorage.DeleteObject(context.Background(), input)

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertNotCalled(t, "DeleteObject", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("success delete object", func(t *testing.T) {
		t.Parallel()
		entry := t.TempDir()
		bucket := model.Bucket{ID: "bucket-1", Name: "avatars", OwnerID: "user-123"}
		object := model.Object{ID: "object-1", BucketID: bucket.ID, Key: "haaland.png", Path: "some/hash"}
		file := filepath.Join(entry, bucket.OwnerID, object.Path)
		require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
		require.NoError(t, os.WriteFile(file, []byte("data"), 0o644))

		mockMetadata := new(MockBucketMetadataRepository)
		mockObjectMetadata := new(MockObjectMetadataRepository)
		objectStorage := NewObjectStorage(mockMetadata, mockObjectMetadata, disk.NewDisk(entry), new(MockChecksum), discardLogger())

		input := DeleteObjectInput{
			BucketName: "avatars",
			OwnerID:    "user-123",
			Name:       "haaland.png",
		}

		mockMetadata.On("GetBucket", mock.Anything, input.BucketName, input.OwnerID).
			Return(bucket, nil).Once()
		mockObjectMetadata.On("GetObject", mock.Anything, bucket.ID, input.OwnerID, input.Name).
			Return(object, nil).Once()
		mockObjectMetadata.On("DeleteObject", mock.Anything, bucket.ID, input.OwnerID, input.Name).
			Return(nil).Once()

		err := objectStorage.DeleteObject(context.Background(), input)

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
		mockObjectMetadata.AssertExpectations(t)
		_, err = os.Stat(file)
		assert.True(t, os.IsNotExist(err))
	})
}
