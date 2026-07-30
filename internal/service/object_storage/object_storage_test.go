package objectstorage

import (
	"context"
	"testing"

	"github.com/faissalmaulana/cairo/internal/model"
	"github.com/faissalmaulana/cairo/internal/repository/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMetadataRepository struct {
	mock.Mock
}

func (mr *MockMetadataRepository) CreateBucket(ctx context.Context, newBucket model.Bucket) (string, error) {
	args := mr.Mock.Called(ctx, newBucket)
	return args.String(0), args.Error(1)
}

func (mr *MockMetadataRepository) GetBucket(ctx context.Context, name string, ownerID string) (model.Bucket, error) {
	args := mr.Mock.Called(ctx, name, ownerID)
	return args.Get(0).(model.Bucket), args.Error(1)
}

func (mr *MockMetadataRepository) GetBucketByName(ctx context.Context, name string) (model.Bucket, error) {
	args := mr.Mock.Called(ctx, name)
	return args.Get(0).(model.Bucket), args.Error(1)
}

func (mr *MockMetadataRepository) ListBuckets(ctx context.Context, ownerID string) ([]model.Bucket, error) {
	args := mr.Mock.Called(ctx, ownerID)
	return args.Get(0).([]model.Bucket), args.Error(1)
}

func (mr *MockMetadataRepository) UpdateBucket(ctx context.Context, name string, ownerID string, update model.UpdateBucketInput) error {
	args := mr.Mock.Called(ctx, name, ownerID, update)
	return args.Error(0)
}

func (mr *MockMetadataRepository) ReplaceBucket(ctx context.Context, bucket model.Bucket) error {
	args := mr.Mock.Called(ctx, bucket)
	return args.Error(0)
}

func (mr *MockMetadataRepository) DeleteBucket(ctx context.Context, name string, ownerID string) error {
	args := mr.Mock.Called(ctx, name, ownerID)
	return args.Error(0)
}

func setupTest() (*MockMetadataRepository, *ObjectStorage) {
	m := new(MockMetadataRepository)
	return m, NewObjectStorage(m)
}

func TestCreateBucket(t *testing.T) {
	t.Run("cannot create new bucket because owner's ID is not provided", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest()
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
			mockMetadata, objectStorage := setupTest()

			_, err := objectStorage.CreateBucket(context.Background(), tc.input)

			assert.ErrorIs(t, err, ErrInvalidBucketName)
			mockMetadata.AssertNotCalled(t, "GetBucketByName", mock.Anything, mock.Anything)
			mockMetadata.AssertNotCalled(t, "CreateBucket", mock.Anything, mock.Anything)
		})
	}

	t.Run("cannot create new bucket because already exist", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest()
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
		mockMetadata, objectStorage := setupTest()

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
		mockMetadata, objectStorage := setupTest()

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
		mockMetadata, objectStorage := setupTest()

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
		mockMetadata, objectStorage := setupTest()
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
		mockMetadata, objectStorage := setupTest()

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
		mockMetadata, objectStorage := setupTest()

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
		mockMetadata, objectStorage := setupTest()

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
		mockMetadata, objectStorage := setupTest()

		_, err := objectStorage.ListBuckets(context.Background(), "")
		assert.ErrorIs(t, err, ErrOwnerIDRequired)
		mockMetadata.AssertNotCalled(t, "ListBuckets", mock.Anything, mock.Anything)
	})

	t.Run("cannot list buckets, something went wrong with the metadata repository method", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest()

		mockMetadata.On("ListBuckets", mock.Anything, "user-123").
			Return([]model.Bucket{}, assert.AnError).Once()

		_, err := objectStorage.ListBuckets(context.Background(), "user-123")

		assert.ErrorIs(t, err, ErrInternal)
		mockMetadata.AssertExpectations(t)
	})

	t.Run("success list buckets empty", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest()

		mockMetadata.On("ListBuckets", mock.Anything, "user-123").
			Return([]model.Bucket{}, nil).Once()

		buckets, err := objectStorage.ListBuckets(context.Background(), "user-123")

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
		assert.Empty(t, buckets)
	})

	t.Run("success list buckets", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest()

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
		mockMetadata, objectStorage := setupTest()
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
		mockMetadata, objectStorage := setupTest()
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
		mockMetadata, objectStorage := setupTest()
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
		mockMetadata, objectStorage := setupTest()
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
}

func TestSetBucketVisibility(t *testing.T) {
	t.Run("cannot set visibility because owner's ID is not provided", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest()
		input := SetBucketVisibilityInput{
			Name:      "avatars",
			OwnerID:   "",
			Visibilty: model.Public,
		}

		err := objectStorage.SetBucketVisibility(context.Background(), input)
		assert.ErrorIs(t, err, ErrOwnerIDRequired)
		mockMetadata.AssertNotCalled(t, "GetBucket", mock.Anything, mock.Anything, mock.Anything)
		mockMetadata.AssertNotCalled(t, "UpdateBucket", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cannot set visibility because bucket not found", func(t *testing.T) {
		t.Parallel()
		mockMetadata, objectStorage := setupTest()
		input := SetBucketVisibilityInput{
			Name:      "not-found-bucket",
			OwnerID:   "user-123",
			Visibilty: model.Public,
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
		mockMetadata, objectStorage := setupTest()
		input := SetBucketVisibilityInput{
			Name:      "avatars",
			OwnerID:   "user-123",
			Visibilty: model.Public,
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
		mockMetadata, objectStorage := setupTest()
		input := SetBucketVisibilityInput{
			Name:      "avatars",
			OwnerID:   "user-123",
			Visibilty: model.Public,
		}

		mockMetadata.On("GetBucket", mock.Anything, input.Name, input.OwnerID).
			Return(model.Bucket{Name: "avatars", OwnerID: "user-123"}, nil).Once()
		mockMetadata.On("UpdateBucket", mock.Anything, input.Name, input.OwnerID, model.UpdateBucketInput{
			Visibilty: &input.Visibilty,
		}).Return(nil).Once()

		err := objectStorage.SetBucketVisibility(context.Background(), input)

		assert.NoError(t, err)
		mockMetadata.AssertExpectations(t)
	})
}
