package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/faissalmaulana/cairo/internal/model"
	apikey_service "github.com/faissalmaulana/cairo/internal/service/apikey"
	"github.com/gin-gonic/gin"
)

type ApiKeyHandler struct {
	apiKeyService *apikey_service.ApiKeyService
}

func NewApiKeyHandler(apiKeyService *apikey_service.ApiKeyService) *ApiKeyHandler {
	return &ApiKeyHandler{
		apiKeyService: apiKeyService,
	}
}

type CreateApiKeyResponse struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	CreatedAt string `json:"created_at"`
}

func (ah *ApiKeyHandler) Create(c *gin.Context) {
	authUserID := c.GetString(helpers.AuthUserIDKey)
	if authUserID == "" {
		FailError(c, ErrTokenRequired)
		return
	}

	key, err := ah.apiKeyService.Create(c.Request.Context(), authUserID)
	if err != nil {
		FailError(c, ErrApiKeyCreateFailed)
		return
	}

	OK(c, http.StatusCreated, CreateApiKeyResponse{
		ID:        key.ID,
		Key:       key.Key,
		CreatedAt: key.CreatedAt.Format(time.RFC3339),
	})
}

type ApiKeyResponse struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	LastUsed  string `json:"last_used,omitempty"`
	CreatedAt string `json:"created_at"`
}

func toApiKeyResponse(key model.ApiKey) ApiKeyResponse {
	resp := ApiKeyResponse{
		ID:        key.ID,
		Key:       key.Key,
		CreatedAt: key.CreatedAt.Format(time.RFC3339),
	}
	if lu := key.LastUsedAt; lu != nil {
		resp.LastUsed = lu.Format(time.RFC3339)
	}
	return resp
}

func (ah *ApiKeyHandler) List(c *gin.Context) {
	authUserID := c.GetString(helpers.AuthUserIDKey)
	if authUserID == "" {
		FailError(c, ErrTokenRequired)
		return
	}

	keys, err := ah.apiKeyService.List(c.Request.Context(), authUserID)
	if err != nil {
		FailError(c, ErrInternalServer)
		return
	}

	resp := make([]ApiKeyResponse, 0, len(keys))
	for _, key := range keys {
		resp = append(resp, toApiKeyResponse(key))
	}

	OK(c, http.StatusOK, resp)
}

func (ah *ApiKeyHandler) Revoke(c *gin.Context) {
	authUserID := c.GetString(helpers.AuthUserIDKey)
	if authUserID == "" {
		FailError(c, ErrTokenRequired)
		return
	}

	err := ah.apiKeyService.Revoke(c.Request.Context(), c.Param("id"), authUserID)
	if err != nil {
		switch {
		case errors.Is(err, apikey_service.ErrApiKeyNotFound):
			FailError(c, ErrApiKeyNotFound)
		default:
			FailError(c, ErrInternalServer)
		}
		return
	}

	OK(c, http.StatusOK, gin.H{"message": "api key revoked"})
}
