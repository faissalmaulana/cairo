package helpers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"log/slog"
	"regexp"
	"strings"
)

var bucketNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// AuthUserIDKey is the gin context key holding the authenticated user's id.
const AuthUserIDKey = "auth_user_id"

// ApiKeyIDKey is the gin context key holding the authenticated api key's id.
const ApiKeyIDKey = "api_key_id"

// RequestLoggerKey is the request-context key holding a request-scoped
// *slog.Logger, injected by middleware.RequestIDMiddleware/SlogMiddleware so
// service-level logs carry the same request_id as the access log.
const RequestLoggerKey = "request_logger"

// LoggerFromContext returns the request-scoped logger when the context carries
// one, otherwise the caller's fallback logger. This lets services enrich their
// logs with request_id without being tightly coupled to the HTTP layer.
func LoggerFromContext(ctx context.Context, fallback *slog.Logger) *slog.Logger {
	if l, ok := ctx.Value(RequestLoggerKey).(*slog.Logger); ok && l != nil {
		return l
	}
	return fallback
}

func ValidateOwnerID(ownerID string) error {
	if len(ownerID) == 0 {
		return errors.New("ownerID is empty")
	}
	return nil
}

func ValidateBucketName(name string) error {
	if !bucketNameRegex.MatchString(name) || (len(name) < 3 || len(name) > 63) {
		return errors.New("invalid bucket name")
	}

	return nil
}

type CheckSummer interface {
	Hash() Hash
}

type Hash interface {
	io.Writer
	Sum() string
}

func GenerateSHA256() Hash {
	return &sha256Hash{hasher: sha256.New()}
}

type sha256Factory struct{}

func (sha256Factory) Hash() Hash { return GenerateSHA256() }

// NewSha256Factory returns a CheckSummer that yields a fresh sha256 Hash per
// call. Object storage expects a one-shot hash per use, so it must be given a
// factory rather than a single Hash instance.
func NewSha256Factory() CheckSummer {
	return sha256Factory{}
}

type sha256Hash struct {
	hasher hash.Hash
}

func (h *sha256Hash) Write(p []byte) (int, error) {
	return h.hasher.Write(p)
}

func (h *sha256Hash) Sum() string {
	return hex.EncodeToString(h.hasher.Sum(nil))
}

// HashName returns a 16-character hex digest (first 8 bytes of sha256) of
// input, suitable for use as a directory or filename on disk while keeping
// paths short. 64 bits of entropy keeps collisions negligible at scale.
func HashName(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:8])
}

// BearerToken extracts the token value from an "Authorization: Bearer <token>"
// header. It returns an error if the header is missing or malformed.
func BearerToken(header string) (string, error) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization header")
	}
	return parts[1], nil
}
