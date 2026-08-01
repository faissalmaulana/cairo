package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

func GenerateSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// HashName returns the hex-encoded sha256 digest of input, suitable for use
// as a directory or filename on disk.
func HashName(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
