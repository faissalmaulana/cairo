package apikey_service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/faissalmaulana/cairo/internal/model"
	apikey_repository "github.com/faissalmaulana/cairo/internal/repository/apikey"
)

// lastUsedThrottle is the minimum interval between two persisted last_used_at
// updates for the same key, avoiding a write per request under concurrency.
const lastUsedThrottle = 5 * time.Minute

type ApiKeyService struct {
	repo apikey_repository.ApiKeyRepository

	lastUsedMu   sync.Mutex
	lastUsedSeen map[string]time.Time
}

func NewApiKeyService(repo apikey_repository.ApiKeyRepository) *ApiKeyService {
	return &ApiKeyService{
		repo:         repo,
		lastUsedSeen: make(map[string]time.Time),
	}
}

var (
	ErrInvalidApiKey    = errors.New("invalid api key")
	ErrApiKeyCreateFail = errors.New("can't create new api key")
	ErrApiKeyNotFound   = errors.New("api key not found")
)

const keyPrefix = "cairo_"

// GeneratedKey holds the plaintext key (shown once) alongside its persisted form.
type GeneratedKey struct {
	Plain string
	model.ApiKey
}

// GenerateKey returns a new "cairo_" prefixed key, its hash and its display prefix.
func GenerateKey() (GeneratedKey, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return GeneratedKey{}, err
	}

	random := hex.EncodeToString(raw)
	plain := keyPrefix + random

	return GeneratedKey{
		Plain: plain,
		ApiKey: model.ApiKey{
			KeyHash: helpers.HashName(plain),
			Prefix:  keyPrefix + random[:6],
		},
	}, nil
}

func (as *ApiKeyService) Create(ctx context.Context, userID string) (GeneratedKey, error) {
	key, err := GenerateKey()
	if err != nil {
		return GeneratedKey{}, err
	}
	key.UserID = userID
	key.CreatedAt = time.Now().UTC()

	id, err := as.repo.Create(ctx, key.ApiKey)
	if err != nil {
		return GeneratedKey{}, ErrApiKeyCreateFail
	}
	key.ID = id

	return key, nil
}

func (as *ApiKeyService) List(ctx context.Context, userID string) ([]model.ApiKey, error) {
	keys, err := as.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (as *ApiKeyService) Revoke(ctx context.Context, keyID, userID string) error {
	err := as.repo.Revoke(ctx, keyID, userID)
	if err != nil {
		if errors.Is(err, apikey_repository.ErrApiKeyNotFound) {
			return ErrApiKeyNotFound
		}
		return err
	}

	return nil
}

// touchLastUsed persists last_used_at at most once per lastUsedThrottle per key.
// The check-then-set is guarded by a mutex so concurrent requests on the same
// key don't duplicate writes; the remaining DB write is cheap and idempotent.
func (as *ApiKeyService) touchLastUsed(ctx context.Context, keyID string) error {
	now := time.Now()

	as.lastUsedMu.Lock()
	last, seen := as.lastUsedSeen[keyID]
	if seen && now.Sub(last) < lastUsedThrottle {
		as.lastUsedMu.Unlock()
		return nil
	}
	as.lastUsedMu.Unlock()

	if err := as.repo.TouchLastUsed(ctx, keyID); err != nil {
		return err
	}

	as.lastUsedMu.Lock()
	as.lastUsedSeen[keyID] = now
	as.lastUsedMu.Unlock()

	return nil
}

// Validate hashes the raw key and looks it up in the database, returning the
// stored key on success. Revoked keys no longer exist and are treated as invalid.
func (as *ApiKeyService) Validate(ctx context.Context, rawKey string) (*model.ApiKey, error) {
	key, err := as.repo.GetByHash(ctx, helpers.HashName(rawKey))
	if err != nil {
		if errors.Is(err, apikey_repository.ErrApiKeyNotFound) {
			return nil, ErrInvalidApiKey
		}
		return nil, err
	}

	if err := as.touchLastUsed(ctx, key.ID); err != nil {
		return nil, err
	}

	return key, nil
}
