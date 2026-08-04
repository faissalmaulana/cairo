package e2e

import (
	"net/http"
	"testing"

	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/stretchr/testify/require"
)

func TestSignUp(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)

	w := doRequest(t, router, http.MethodPost, "/api/v1/signup", "", handler.SignUpRequest{
		Username: "e2euser",
		Email:    "e2euser@example.com",
		Password: "password123",
	})
	require.Equal(t, http.StatusCreated, w.Code)

	tokens := okResponse[handler.TokenResponse](t, w)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
	require.NotEmpty(t, tokens.TokenType)
	require.NotEmpty(t, tokens.ExpiresIn)
	require.NotEmpty(t, tokens.RefreshExpiresIn)
}

func TestSignUpValidation(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)

	cases := []struct {
		name string
		body handler.SignUpRequest
	}{
		{
			name: "short password",
			body: handler.SignUpRequest{
				Username: "e2euser",
				Email:    "e2euser@example.com",
				Password: "short",
			},
		},
		{
			name: "invalid email",
			body: handler.SignUpRequest{
				Username: "e2euser",
				Email:    "not-an-email",
				Password: "password123",
			},
		},
		{
			name: "missing username",
			body: handler.SignUpRequest{
				Email:    "e2euser@example.com",
				Password: "password123",
				// Username is omitted, so it defaults to the zero value
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(t, router, http.MethodPost, "/api/v1/signup", "", tc.body)
			failResponse(t, w, http.StatusBadRequest, "BAD_REQUEST")
		})
	}
}

func TestSignUpDuplicateEmail(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)

	body := handler.SignUpRequest{
		Username: "e2euser",
		Email:    "dup@example.com",
		Password: "password123",
	}
	w := doRequest(t, router, http.MethodPost, "/api/v1/signup", "", body)
	require.Equal(t, http.StatusCreated, w.Code)

	w = doRequest(t, router, http.MethodPost, "/api/v1/signup", "", body)
	failResponse(t, w, http.StatusConflict, "EMAIL_EXISTS")
}

func TestSignIn(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)

	body := handler.SignUpRequest{
		Username: "e2euser",
		Email:    "e2euser@example.com",
		Password: "password123",
	}
	signUp(t, router, body)

	w := doRequest(t, router, http.MethodPost, "/api/v1/signin", "", handler.SignInRequest{
		Email:    body.Email,
		Password: body.Password,
	})
	require.Equal(t, http.StatusOK, w.Code)

	tokens := okResponse[handler.TokenResponse](t, w)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
	require.NotEmpty(t, tokens.TokenType)
	require.NotEmpty(t, tokens.ExpiresIn)
	require.NotEmpty(t, tokens.RefreshExpiresIn)
}

func TestSignInInvalidCredentials(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)

	body := handler.SignUpRequest{
		Username: "e2euser",
		Email:    "e2euser@example.com",
		Password: "password123",
	}
	signUp(t, router, body)

	w := doRequest(t, router, http.MethodPost, "/api/v1/signin", "", handler.SignInRequest{
		Email:    body.Email,
		Password: "wrongpassword123",
	})
	failResponse(t, w, http.StatusUnauthorized, "INVALID")
}

func TestAccount(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)

	t.Run("success getting user's account", func(t *testing.T) {
		body := handler.SignUpRequest{
			Username: "e2euser",
			Email:    "e2euser@example.com",
			Password: "password123",
		}
		tokens := signUp(t, router, body)

		w := doRequest(t, router, http.MethodGet, "/api/v1/account", tokens.AccessToken, nil)
		require.Equal(t, http.StatusOK, w.Code)

		account := okResponse[handler.UserResponse](t, w)
		require.NotEmpty(t, account.ID)
		require.NotEmpty(t, account.Username)
		require.NotEmpty(t, account.Email)
	})

	t.Run("fail request is unauthorized", func(t *testing.T) {
		w := doRequest(t, router, http.MethodGet, "/api/v1/account", "", nil)
		failResponse(t, w, http.StatusUnauthorized, "TOKEN_REQUIRED")
	})
}

func TestRefresh(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)

	body := handler.SignUpRequest{
		Username: "e2euser",
		Email:    "e2euser@example.com",
		Password: "password123",
	}
	tokens := signUp(t, router, body)

	w := doRequest(t, router, http.MethodPost, "/api/v1/refresh", tokens.RefreshToken, nil)
	require.Equal(t, http.StatusOK, w.Code)

	newTokens := okResponse[handler.TokenResponse](t, w)
	require.NotEmpty(t, newTokens.AccessToken)
	require.NotEmpty(t, newTokens.RefreshToken)
	require.NotEmpty(t, newTokens.TokenType)
	require.NotEmpty(t, newTokens.ExpiresIn)
	require.NotEmpty(t, newTokens.RefreshExpiresIn)
}

func TestRefreshTokenIsSingleUse(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)

	body := handler.SignUpRequest{
		Username: "e2euser",
		Email:    "e2euser@example.com",
		Password: "password123",
	}
	tokens := signUp(t, router, body)

	doRequest(t, router, http.MethodPost, "/api/v1/refresh", tokens.RefreshToken, nil)

	// the old refresh token should be invalid because it got rotated
	w := doRequest(t, router, http.MethodPost, "/api/v1/refresh", tokens.RefreshToken, nil)
	failResponse(t, w, http.StatusUnauthorized, "REFRESH_FAILED")
}

func TestLogout(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)

	body := handler.SignUpRequest{
		Username: "e2euser",
		Email:    "e2euser@example.com",
		Password: "password123",
	}
	tokens := signUp(t, router, body)

	w := doRequest(t, router, http.MethodPost, "/api/v1/account/logout", tokens.AccessToken, map[string]any{
		"refresh_token": tokens.RefreshToken,
	})
	require.Equal(t, http.StatusOK, w.Code)

	w = doRequest(t, router, http.MethodGet, "/api/v1/account", tokens.AccessToken, nil)
	failResponse(t, w, http.StatusUnauthorized, "TOKEN_REVOKED")

	w = doRequest(t, router, http.MethodPost, "/api/v1/refresh", tokens.RefreshToken, nil)
	failResponse(t, w, http.StatusUnauthorized, "REFRESH_FAILED")
}
