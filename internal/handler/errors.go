package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Error struct {
	Status  int
	Code    string
	Message string
}

func NewError(status int, code, message string) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
	}
}

func FailError(c *gin.Context, err *Error) {
	Fail(c, err.Status, err.Code, err.Message)
}

func ErrValidation(err error) *Error {
	message := "invalid body request"
	if err != nil {
		message = err.Error()
	}
	return NewError(http.StatusBadRequest, "BAD_REQUEST", message)
}

var (
	ErrInvalidCredentials  = NewError(http.StatusUnauthorized, "INVALID", "email or password incorrect")
	ErrRequiredSignIn      = NewError(http.StatusUnauthorized, "INVALID", "required sign in")
	ErrInternalServer      = NewError(http.StatusInternalServerError, "SERVER_ERROR", "something went wrong")
	ErrSignUpFailed        = NewError(http.StatusInternalServerError, "SERVER_ERROR", "something went wrong during sign up")
	ErrLogoutFailed        = NewError(http.StatusInternalServerError, "SERVER_ERROR", "something went wrong during logout")
	ErrEmailAlreadyExists  = NewError(http.StatusConflict, "EMAIL_EXISTS", "email is not available")
	ErrTokenRequired       = NewError(http.StatusUnauthorized, "TOKEN_REQUIRED", "token is required")
	ErrInvalidToken        = NewError(http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
	ErrTokenExpired        = NewError(http.StatusUnauthorized, "TOKEN_EXPIRED", "token expired")
	ErrTokenRevoked        = NewError(http.StatusUnauthorized, "TOKEN_REVOKED", "token revoked")
	ErrRefreshFailed       = NewError(http.StatusUnauthorized, "REFRESH_FAILED", "refresh token is invalid or revoked")
	ErrAuthHeaderRequired  = NewError(http.StatusUnauthorized, "AUTH_HEADER_REQUIRED", "authorization header is required")
	ErrApiKeyRequired      = NewError(http.StatusUnauthorized, "API_KEY_REQUIRED", "api key is required")
	ErrInvalidApiKey       = NewError(http.StatusUnauthorized, "INVALID_API_KEY", "invalid api key")
	ErrApiKeyCreateFailed  = NewError(http.StatusInternalServerError, "SERVER_ERROR", "something went wrong during api key creation")
	ErrApiKeyNotFound      = NewError(http.StatusNotFound, "API_KEY_NOT_FOUND", "api key not found")
	ErrOwnerIDRequired     = NewError(http.StatusBadRequest, "OWNER_REQUIRED", "owner id is required")
	ErrInvalidBucketName   = NewError(http.StatusBadRequest, "INVALID_BUCKET_NAME", "invalid bucket name")
	ErrBucketAlreadyExists = NewError(http.StatusConflict, "BUCKET_EXISTS", "bucket already exists")
	ErrBucketNotFound      = NewError(http.StatusNotFound, "BUCKET_NOT_FOUND", "bucket not found")
	ErrObjectNotFound      = NewError(http.StatusNotFound, "OBJECT_NOT_FOUND", "object not found")
	ErrBucketForbidden     = NewError(http.StatusForbidden, "FORBIDDEN", "not allowed to access this bucket")
	ErrChecksumMismatch    = NewError(http.StatusConflict, "CHECKSUM_MISMATCH", "checksum mismatch")
)
