package e2e

import (
	"errors"
	"net/http"
	"testing"

	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/stretchr/testify/require"
)

func TestSignUp(t *testing.T) {
	router := setupEnv(t)

	w := doRequest(t, router, http.MethodPost, "/api/v1/signup", "", handler.SignUpRequest{
		Username: "e2euser",
		Email:    "e2euser@example.com",
		Password: "password123",
	})

	var response handler.Response

	decodeBodyResponse(t, w, &response)

	// check status from response header status code and the body
	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, true, response.Success)

	require.Nil(t, response.Error)

	data, ok := response.Data.(map[string]any)
	if !ok {
		require.Error(t, errors.New("type assertion fail"))
	}

	// check data body
	require.NotEmpty(t, data["access_token"])
	require.NotEmpty(t, data["refresh_token"])
	require.NotEmpty(t, data["token_type"])
	require.NotEmpty(t, data["expires_in"])
	require.NotEmpty(t, data["refresh_expires_in"])
}

func TestSignUpValidation(t *testing.T) {
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
			require.Equal(t, http.StatusBadRequest, w.Code)

			var response handler.Response

			decodeBodyResponse(t, w, &response)
			require.Equal(t, false, response.Success)
			require.NotNil(t, response.Error)
			require.Nil(t, response.Data)

			require.Equal(t, "BAD_REQUEST", response.Error.Code)
		})
	}
}

func TestSignUpDuplicateEmail(t *testing.T) {
	router := setupEnv(t)

	body := handler.SignUpRequest{
		Username: "e2euser",
		Email:    "dup@example.com",
		Password: "password123",
	}
	w := doRequest(t, router, http.MethodPost, "/api/v1/signup", "", body)
	require.Equal(t, http.StatusCreated, w.Code)

	w = doRequest(t, router, http.MethodPost, "/api/v1/signup", "", body)
	require.Equal(t, http.StatusConflict, w.Code)

	var response handler.Response

	decodeBodyResponse(t, w, &response)
	require.Equal(t, false, response.Success)
	require.NotNil(t, response.Error)
	require.Nil(t, response.Data)

	require.Equal(t, "EMAIL_EXISTS", response.Error.Code)
}

func TestSignIn(t *testing.T) {
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

	var response handler.Response

	decodeBodyResponse(t, w, &response)

	// check status from response header status code and the body
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, true, response.Success)

	require.Nil(t, response.Error)
	data, ok := response.Data.(map[string]any)
	if !ok {
		require.Error(t, errors.New("type assertion fail"))
	}

	// check data body
	require.NotEmpty(t, data["access_token"])
	require.NotEmpty(t, data["refresh_token"])
	require.NotEmpty(t, data["token_type"])
	require.NotEmpty(t, data["expires_in"])
	require.NotEmpty(t, data["refresh_expires_in"])
}

func TestSignInInvalidCredentials(t *testing.T) {
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

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var response handler.Response

	decodeBodyResponse(t, w, &response)
	require.Equal(t, false, response.Success)
	require.NotNil(t, response.Error)
	require.Nil(t, response.Data)

	require.Equal(t, "INVALID", response.Error.Code)
}

func TestAccount(t *testing.T) {
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

		var response handler.Response

		decodeBodyResponse(t, w, &response)

		// check status from response header status code and the body
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, true, response.Success)

		require.Nil(t, response.Error)
		data, ok := response.Data.(map[string]any)
		if !ok {
			require.Error(t, errors.New("type assertion fail"))
		}

		// check data body
		require.NotEmpty(t, data["id"])
		require.NotEmpty(t, data["username"])
		require.NotEmpty(t, data["email"])
	})

	t.Run("fail request is unauthorized", func(t *testing.T) {
		w := doRequest(t, router, http.MethodGet, "/api/v1/account", "", nil)

		var response handler.Response

		decodeBodyResponse(t, w, &response)

		// check status from response header status code and the body
		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Equal(t, false, response.Success)

		require.Nil(t, response.Data)
		require.NotNil(t, response.Error)

		require.Equal(t, "TOKEN_REQUIRED", response.Error.Code)
	})
}

func TestRefresh(t *testing.T) {
	router := setupEnv(t)

	body := handler.SignUpRequest{
		Username: "e2euser",
		Email:    "e2euser@example.com",
		Password: "password123",
	}

	tokens := signUp(t, router, body)

	w := doRequest(t, router, http.MethodPost, "/api/v1/refresh", tokens.RefreshToken, nil)
	require.Equal(t, http.StatusOK, w.Code)

	var response handler.Response

	decodeBodyResponse(t, w, &response)

	// check status from response header status code and the body
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, true, response.Success)

	require.Nil(t, response.Error)
	data, ok := response.Data.(map[string]any)
	if !ok {
		require.Error(t, errors.New("type assertion fail"))
	}

	// check data body
	require.NotEmpty(t, data["access_token"])
	require.NotEmpty(t, data["refresh_token"])
	require.NotEmpty(t, data["token_type"])
	require.NotEmpty(t, data["expires_in"])
	require.NotEmpty(t, data["refresh_expires_in"])
}

func TestRefreshTokenIsSingleUse(t *testing.T) {
	router := setupEnv(t)

	body := handler.SignUpRequest{
		Username: "e2euser",
		Email:    "e2euser@example.com",
		Password: "password123",
	}

	tokens := signUp(t, router, body)

	doRequest(t, router, http.MethodPost, "/api/v1/refresh", tokens.RefreshToken, nil)

	// the old refresh token should invalid. because got rotated
	w := doRequest(t, router, http.MethodPost, "/api/v1/refresh", tokens.RefreshToken, nil)

	var response handler.Response

	decodeBodyResponse(t, w, &response)

	// check status from response header status code and the body
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Equal(t, false, response.Success)

	require.Nil(t, response.Data)

	require.NotNil(t, response.Error)
	require.Equal(t, "REFRESH_FAILED", response.Error.Code)
}

func TestLogout(t *testing.T) {
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
	require.Equal(t, http.StatusUnauthorized, w.Code)

	var responseGetAccount handler.Response

	decodeBodyResponse(t, w, &responseGetAccount)

	// check status from response header status code and the body
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Equal(t, false, responseGetAccount.Success)
	require.Nil(t, responseGetAccount.Data)
	require.NotNil(t, responseGetAccount.Error)

	require.Equal(t, "TOKEN_REVOKED", responseGetAccount.Error.Code)

	w = doRequest(t, router, http.MethodPost, "/api/v1/refresh", tokens.RefreshToken, nil)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	var responseRefresh handler.Response

	decodeBodyResponse(t, w, &responseRefresh)

	// check status from response header status code and the body
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Equal(t, false, responseRefresh.Success)
	require.Nil(t, responseRefresh.Data)
	require.NotNil(t, responseRefresh.Error)

	require.Equal(t, "REFRESH_FAILED", responseRefresh.Error.Code)
}
