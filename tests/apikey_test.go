package e2e

import (
	"net/http"
	"testing"

	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/stretchr/testify/require"
)

const (
	apiKeyBasePath = "/api/v1/account/apikeys"
)

// TODO: test againts enpoint that authenticated with api keys

func signupUser(t *testing.T, router http.Handler) handler.TokenResponse {
	t.Helper()

	return signUp(t, router, handler.SignUpRequest{
		Username: "e2eapiuser",
		Email:    "e2eapi@example.com",
		Password: "password123",
	})
}

func createApiKey(t *testing.T, router http.Handler, accessToken string) handler.CreateApiKeyResponse {
	t.Helper()

	w := doRequest(t, router, http.MethodPost, apiKeyBasePath, accessToken, nil)
	require.Equal(t, http.StatusCreated, w.Code)

	return okResponse[handler.CreateApiKeyResponse](t, w)
}

func TestCreateApiKey(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	tokens := signupUser(t, router)

	key := createApiKey(t, router, tokens.AccessToken)

	require.NotEmpty(t, key.ID)
	require.NotEmpty(t, key.Key)
	require.Contains(t, key.Key, "cairo_")
	require.NotEmpty(t, key.Prefix)
	require.NotEmpty(t, key.CreatedAt)
}

func TestListApiKeys(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	tokens := signupUser(t, router)

	created := createApiKey(t, router, tokens.AccessToken)

	w := doRequest(t, router, http.MethodGet, apiKeyBasePath, tokens.AccessToken, nil)
	require.Equal(t, http.StatusOK, w.Code)

	keys := okResponse[[]handler.ApiKeyResponse](t, w)

	var found bool
	for _, k := range keys {
		if k.ID != created.ID {
			continue
		}
		found = true
		require.Equal(t, created.Prefix, k.Prefix)
	}
	require.True(t, found, "created api key should appear in the list")
}

func TestRevokeApiKey(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)
	tokens := signupUser(t, router)

	// TODO: revoked key no longer authenticates

	t.Run("revoking unknown key returns not found", func(t *testing.T) {
		w := doRequest(t, router, http.MethodDelete, apiKeyBasePath+"/does-not-exist", tokens.AccessToken, nil)
		failResponse(t, w, http.StatusNotFound, "API_KEY_NOT_FOUND")
	})
}
