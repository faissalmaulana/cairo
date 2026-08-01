package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"regexp"
)

var bucketNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

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

type sha256Hash struct {
	hasher hash.Hash
}

func (h *sha256Hash) Write(p []byte) (int, error) {
	return h.hasher.Write(p)
}

func (h *sha256Hash) Sum() string {
	return hex.EncodeToString(h.hasher.Sum(nil))
}

// HashName returns the hex-encoded sha256 digest of input, suitable for use
// as a directory or filename on disk.
func HashName(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
