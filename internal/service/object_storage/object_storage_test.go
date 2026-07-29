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

func setupTest() (*MockMetadataRepository, *ObjectStorage) {
	m := new(MockMetadataRepository)
	return m, NewObjectStorage(m)
}

func TestCreateBucket(t *testing.T) {
	t.Run("cannot create new bucket because owner's ID is not provided", func(t *testing.T) {
		mockMetadata, objectStorage := setupTest()
		newBucket := CreateBucketInput{
			Name:    "avatars",
			OwnerID: "",
		}

		_, err := objectStorage.CreateBucket(context.Background(), newBucket)
		assert.ErrorIs(t, err, ErrNewBucketOwnerEmpty)
		mockMetadata.AssertNotCalled(t, "CreateBucket", mock.Anything, mock.Anything)
	})

	testCases := []struct {
		name    string
		input   CreateBucketInput
		got     string
		isError bool
	}{
		{
			name: "cannot create new bucket invalid name (min leng)",
			input: CreateBucketInput{
				Name:    "pr",
				OwnerID: "asdjsaodsaidhisadiisad",
			},
			isError: true,
		},
		{
			name: "cannot create new bucket invalid name (max leng)",
			input: CreateBucketInput{
				Name:    "asdfghjklasd9asi-absduad82-2aishdiahsdiashdihiahd0-chaushcachiahcuahsuha",
				OwnerID: "asdjsaodsaidhisadiisad",
			},
			isError: true,
		},
		{
			name: "cannot create new bucket invalid name (prefix with hypen)",
			input: CreateBucketInput{
				Name:    "-asdfghjklasd9asi-absduad82",
				OwnerID: "asdjsaodsaidhisadiisad",
			},
			isError: true,
		},
		{
			name: "cannot create new bucket invalid name (suffix with hypen)",
			input: CreateBucketInput{
				Name:    "asdfghjklasd9asi-absduad82-",
				OwnerID: "asdjsaodsaidhisadiisad",
			},
			isError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockMetadata, objectStorage := setupTest()

			_, err := objectStorage.CreateBucket(context.Background(), tc.input)

			assert.ErrorIs(t, err, ErrInvalidBucketName)
			mockMetadata.AssertNotCalled(t, "CreateBucket", mock.Anything, mock.Anything)
		})
	}

	t.Run("cannot create new bucket because already exist", func(t *testing.T) {
		mockMetadata, objectStorage := setupTest()
		// somewhere in the storage the bucket exist
		newBucket := CreateBucketInput{
			Name:    "profile-users",
			OwnerID: "12345678",
		}

		mockMetadata.On("CreateBucket", mock.Anything, model.Bucket{
			Name:    newBucket.Name,
			OwnerID: newBucket.OwnerID,
		}).Return("", metadata.ErrBucketAlreadyExists).Once()

		_, err := objectStorage.CreateBucket(context.Background(), newBucket)

		mockMetadata.AssertExpectations(t)
		assert.ErrorIs(t, err, metadata.ErrBucketAlreadyExists)
	})

	t.Run("cannot create new bucket because the user already own it", func(t *testing.T) {
		mockMetadata, objectStorage := setupTest()

		newBucket := CreateBucketInput{
			Name:    "profile-users",
			OwnerID: "12345678",
		}

		mockMetadata.On("CreateBucket", mock.Anything, model.Bucket{
			Name:    newBucket.Name,
			OwnerID: newBucket.OwnerID,
		}).Return("", metadata.ErrBucketAlreadyOwnedByYou).Once()

		_, err := objectStorage.CreateBucket(context.Background(), newBucket)

		mockMetadata.AssertExpectations(t)
		assert.ErrorIs(t, err, metadata.ErrBucketAlreadyOwnedByYou)
	})

	t.Run("cannot create new bucket, something went wrong with the metadata repository method", func(t *testing.T) {
		mockMetadata, objectStorage := setupTest()

		newBucket := CreateBucketInput{
			Name:    "profile-users",
			OwnerID: "12345678",
		}

		mockMetadata.On("CreateBucket", mock.Anything, model.Bucket{
			Name:    newBucket.Name,
			OwnerID: newBucket.OwnerID,
		}).Return("", metadata.ErrCannotCreateBucket).Once()

		_, err := objectStorage.CreateBucket(context.Background(), newBucket)

		mockMetadata.AssertExpectations(t)
		assert.ErrorIs(t, err, metadata.ErrCannotCreateBucket)
	})

	t.Run("success create bucket", func(t *testing.T) {
		mockMetadata, objectStorage := setupTest()

		expctedBucketID := "asdijkacuubosj12"
		newBucket := CreateBucketInput{
			Name:    "avatars",
			OwnerID: "872371727",
		}

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
