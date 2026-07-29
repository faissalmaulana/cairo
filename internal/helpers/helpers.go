package helpers

import (
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
