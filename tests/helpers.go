package e2e

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/stretchr/testify/require"
)

// envelope[T] decodes the shared success/error response but keeps a typed
// Data field, so callers get concrete structs instead of map[string]any.
// handler.Response can't be reused here because its Data is declared any.
type envelope[T any] struct {
	Success bool               `json:"success"`
	Data    T                  `json:"data"`
	Error   *handler.ErrorInfo `json:"error,omitempty"`
}

// okResponse asserts a success response and returns the typed Data payload.
// The caller is expected to assert the HTTP status separately with doRequest.
func okResponse[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()

	var env envelope[T]
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	require.True(t, env.Success, "expected success envelope")
	require.Nil(t, env.Error)

	return env.Data
}

// failResponse asserts the expected HTTP status + error code and that no data
// is present in the response body.
func failResponse(t *testing.T, w *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	require.Equal(t, wantStatus, w.Code)

	var env envelope[any]
	require.NoError(t, json.NewDecoder(w.Body).Decode(&env))
	require.False(t, env.Success)
	require.Nil(t, env.Data)
	require.NotNil(t, env.Error)
	require.Equal(t, wantCode, env.Error.Code)
}
