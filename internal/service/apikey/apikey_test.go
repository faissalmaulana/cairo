package apikey_service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/faissalmaulana/cairo/internal/model"
	apikey_repository "github.com/faissalmaulana/cairo/internal/repository/apikey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockApiKeyRepository struct {
	mock.Mock
}

func (mr *MockApiKeyRepository) Create(ctx context.Context, newKey model.ApiKey) (string, error) {
	args := mr.Mock.Called(ctx, newKey)
	return args.String(0), args.Error(1)
}

func (mr *MockApiKeyRepository) GetByHash(ctx context.Context, keyHash string) (*model.ApiKey, error) {
	args := mr.Mock.Called(ctx, keyHash)
	return args.Get(0).(*model.ApiKey), args.Error(1)
}

func (mr *MockApiKeyRepository) ListByUser(ctx context.Context, userID string) ([]model.ApiKey, error) {
	args := mr.Mock.Called(ctx, userID)
	return args.Get(0).([]model.ApiKey), args.Error(1)
}

func (mr *MockApiKeyRepository) Revoke(ctx context.Context, id, userID string) error {
	args := mr.Mock.Called(ctx, id, userID)
	return args.Error(0)
}

func (mr *MockApiKeyRepository) TouchLastUsed(ctx context.Context, id string) error {
	args := mr.Mock.Called(ctx, id)
	return args.Error(0)
}

func TestGenerateKey(t *testing.T) {
	t.Run("generated key has cairo_ prefix", func(t *testing.T) {
		key, err := GenerateKey()
		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(key.Plain, keyPrefix), "expected prefix %q, got %q", keyPrefix, key.Plain)
		assert.Equal(t, 64, len(strings.TrimPrefix(key.Plain, keyPrefix)))
	})

	t.Run("key hash is the sha256 of the plaintext", func(t *testing.T) {
		key, err := GenerateKey()
		assert.NoError(t, err)
		assert.Equal(t, helpers.HashName(key.Plain), key.KeyHash)
	})

	t.Run("prefix is cairo_ plus first 6 random chars", func(t *testing.T) {
		key, err := GenerateKey()
		assert.NoError(t, err)
		random := strings.TrimPrefix(key.Plain, keyPrefix)
		assert.Equal(t, keyPrefix+random[:6], key.Prefix)
	})

	t.Run("two generated keys differ", func(t *testing.T) {
		k1, err := GenerateKey()
		assert.NoError(t, err)
		k2, err := GenerateKey()
		assert.NoError(t, err)
		assert.NotEqual(t, k1.Plain, k2.Plain)
	})
}

func TestCreate(t *testing.T) {
	t.Run("repo failure", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("Create", mock.Anything, mock.AnythingOfType("model.ApiKey")).
			Return("", errors.New("boom")).Once()

		svc := NewApiKeyService(mockRepo)
		_, err := svc.Create(context.Background(), "user-123")
		assert.ErrorIs(t, err, ErrApiKeyCreateFail)
	})

	t.Run("success returns plaintext key", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("Create", mock.Anything, mock.AnythingOfType("model.ApiKey")).
			Return("key-1", nil).Once()

		svc := NewApiKeyService(mockRepo)
		key, err := svc.Create(context.Background(), "user-123")

		assert.NoError(t, err)
		assert.Equal(t, "key-1", key.ID)
		assert.Equal(t, "user-123", key.UserID)
		assert.True(t, strings.HasPrefix(key.Plain, keyPrefix))
		assert.Equal(t, helpers.HashName(key.Plain), key.KeyHash)
		assert.False(t, key.CreatedAt.IsZero())
	})
}

func TestList(t *testing.T) {
	t.Run("repo failure", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("ListByUser", mock.Anything, "user-123").
			Return([]model.ApiKey{}, errors.New("boom")).Once()

		svc := NewApiKeyService(mockRepo)
		_, err := svc.List(context.Background(), "user-123")
		assert.Error(t, err)
	})

	t.Run("success returns keys", func(t *testing.T) {
		expected := []model.ApiKey{
			{ID: "key-1", UserID: "user-123"},
			{ID: "key-2", UserID: "user-123"},
		}
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("ListByUser", mock.Anything, "user-123").
			Return(expected, nil).Once()

		svc := NewApiKeyService(mockRepo)
		keys, err := svc.List(context.Background(), "user-123")

		assert.NoError(t, err)
		assert.Equal(t, expected, keys)
	})
}

func TestRevoke(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("Revoke", mock.Anything, "key-1", "user-123").
			Return(apikey_repository.ErrApiKeyNotFound).Once()

		svc := NewApiKeyService(mockRepo)
		err := svc.Revoke(context.Background(), "key-1", "user-123")
		assert.ErrorIs(t, err, ErrApiKeyNotFound)
	})

	t.Run("repo failure", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("Revoke", mock.Anything, "key-1", "user-123").
			Return(errors.New("boom")).Once()

		svc := NewApiKeyService(mockRepo)
		err := svc.Revoke(context.Background(), "key-1", "user-123")
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("Revoke", mock.Anything, "key-1", "user-123").
			Return(nil).Once()

		svc := NewApiKeyService(mockRepo)
		assert.NoError(t, svc.Revoke(context.Background(), "key-1", "user-123"))
	})
}

func TestValidate(t *testing.T) {
	t.Run("key not found", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).
			Return(&model.ApiKey{}, apikey_repository.ErrApiKeyNotFound).Once()

		svc := NewApiKeyService(mockRepo)
		_, err := svc.Validate(context.Background(), "cairo_invalid")
		assert.ErrorIs(t, err, ErrInvalidApiKey)
	})

	t.Run("revoked key rejected (hard deleted)", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).
			Return(&model.ApiKey{}, apikey_repository.ErrApiKeyNotFound).Once()

		svc := NewApiKeyService(mockRepo)
		_, err := svc.Validate(context.Background(), "cairo_revoked")
		assert.ErrorIs(t, err, ErrInvalidApiKey)
	})

	t.Run("repo failure", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).
			Return(&model.ApiKey{}, errors.New("boom")).Once()

		svc := NewApiKeyService(mockRepo)
		_, err := svc.Validate(context.Background(), "cairo_any")
		assert.Error(t, err)
	})

	t.Run("touch last used failure", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).
			Return(&model.ApiKey{ID: "key-1", UserID: "user-123"}, nil).Once()
		mockRepo.On("TouchLastUsed", mock.Anything, "key-1").
			Return(errors.New("boom")).Once()

		svc := NewApiKeyService(mockRepo)
		_, err := svc.Validate(context.Background(), "cairo_any")
		assert.Error(t, err)
	})

	t.Run("success returns key and bumps last used", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).
			Return(&model.ApiKey{ID: "key-1", UserID: "user-123"}, nil).Once()
		mockRepo.On("TouchLastUsed", mock.Anything, "key-1").
			Return(nil).Once()

		svc := NewApiKeyService(mockRepo)
		key, err := svc.Validate(context.Background(), "cairo_valid")

		assert.NoError(t, err)
		assert.Equal(t, "user-123", key.UserID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repeated validates within throttle window write once", func(t *testing.T) {
		mockRepo := new(MockApiKeyRepository)
		mockRepo.On("GetByHash", mock.Anything, mock.AnythingOfType("string")).
			Return(&model.ApiKey{ID: "key-1", UserID: "user-123"}, nil).Twice()
		mockRepo.On("TouchLastUsed", mock.Anything, "key-1").
			Return(nil).Once()

		svc := NewApiKeyService(mockRepo)
		_, err := svc.Validate(context.Background(), "cairo_valid")
		assert.NoError(t, err)
		_, err = svc.Validate(context.Background(), "cairo_valid")
		assert.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})
}
