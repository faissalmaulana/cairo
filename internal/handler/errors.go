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
	ErrInvalidCredentials = NewError(http.StatusUnauthorized, "INVALID", "email or password incorrect")
	ErrRequiredSignIn     = NewError(http.StatusUnauthorized, "INVALID", "required sign in")
	ErrInternalServer     = NewError(http.StatusInternalServerError, "SERVER_ERROR", "something went wrong")
	ErrSignUpFailed       = NewError(http.StatusInternalServerError, "SERVER_ERROR", "something went wrong during sign up")
	ErrLogoutFailed       = NewError(http.StatusInternalServerError, "SERVER_ERROR", "something went wrong during logout")
	ErrEmailAlreadyExists = NewError(http.StatusConflict, "EMAIL_EXISTS", "email is not available")
	ErrTokenRequired      = NewError(http.StatusUnauthorized, "TOKEN_REQUIRED", "token is required")
	ErrInvalidToken       = NewError(http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
	ErrTokenExpired       = NewError(http.StatusUnauthorized, "TOKEN_EXPIRED", "token expired")
	ErrTokenRevoked       = NewError(http.StatusUnauthorized, "TOKEN_REVOKED", "token revoked")
	ErrRefreshFailed      = NewError(http.StatusUnauthorized, "REFRESH_FAILED", "refresh token is invalid or revoked")
	ErrAuthHeaderRequired = NewError(http.StatusUnauthorized, "AUTH_HEADER_REQUIRED", "authorization header is required")
)
