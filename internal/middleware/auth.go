package middleware

import (
	"errors"

	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/faissalmaulana/cairo/internal/helpers"
	token_service "github.com/faissalmaulana/cairo/internal/service/token"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	tokenService *token_service.TokenService
}

func NewAuthMiddleware(tokenService *token_service.TokenService) *AuthMiddleware {
	return &AuthMiddleware{
		tokenService: tokenService,
	}
}

func (am *AuthMiddleware) CheckAuth(c *gin.Context) {
	rawToken, err := helpers.BearerToken(c.GetHeader("Authorization"))
	if err != nil {
		handler.FailError(c, handler.ErrTokenRequired)
		c.Abort()
		return
	}

	jti, err := am.tokenService.ExtractJTI(rawToken)
	if err != nil {
		handler.FailError(c, handler.ErrInvalidToken)
		c.Abort()
		return
	}

	revoked, err := am.tokenService.IsRevoked(c.Request.Context(), jti)
	if err != nil {
		handler.FailError(c, handler.ErrInternalServer)
		c.Abort()
		return
	}
	if revoked {
		handler.FailError(c, handler.ErrTokenRevoked)
		c.Abort()
		return
	}

	claims, err := am.tokenService.ParseAccessToken(rawToken)
	if err != nil {
		switch {
		case errors.Is(err, token_service.ErrExpiredToken):
			handler.FailError(c, handler.ErrTokenExpired)
		default:
			handler.FailError(c, handler.ErrInvalidToken)
		}
		c.Abort()
		return
	}

	c.Set(helpers.AuthUserIDKey, claims.Subject)
	c.Next()
}
