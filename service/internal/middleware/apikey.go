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

// RequireAccount ties the :account_id path param to the user authenticated by
// the api key. Without it, callers could target another user's namespace (e.g.
// with buckets.owner_id referencing users(id), a bogus account id surfaces as
// a foreign-key 500 instead of a clean rejection).
func (am *ApiKeyMiddleware) RequireAccount(c *gin.Context) {
	if c.GetString(helpers.AuthUserIDKey) != c.Param("account_id") {
		handler.FailError(c, handler.ErrAccountMismatch)
		c.Abort()
		return
	}
	c.Next()
}
