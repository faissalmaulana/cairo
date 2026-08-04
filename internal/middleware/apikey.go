package middleware

import (
	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/faissalmaulana/cairo/internal/helpers"
	apikey_service "github.com/faissalmaulana/cairo/internal/service/apikey"
	"github.com/gin-gonic/gin"
)

type ApiKeyMiddleware struct {
	apiKeyService *apikey_service.ApiKeyService
}

func NewApiKeyMiddleware(apiKeyService *apikey_service.ApiKeyService) *ApiKeyMiddleware {
	return &ApiKeyMiddleware{
		apiKeyService: apiKeyService,
	}
}

// CheckApiKey authenticates a request using an API key sent in the
// "Authorization: Bearer <key>" header, resolving the owning user.
func (am *ApiKeyMiddleware) CheckApiKey(c *gin.Context) {
	rawKey, err := helpers.BearerToken(c.GetHeader("Authorization"))
	if err != nil {
		handler.FailError(c, handler.ErrApiKeyRequired)
		c.Abort()
		return
	}

	key, err := am.apiKeyService.Validate(c.Request.Context(), rawKey)
	if err != nil {
		handler.FailError(c, handler.ErrInvalidApiKey)
		c.Abort()
		return
	}

	c.Set(helpers.AuthUserIDKey, key.UserID)
	c.Set(helpers.ApiKeyIDKey, key.ID)
	c.Next()
}
