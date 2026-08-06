package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocsPagePubliclyServed(t *testing.T) {
	t.Parallel()

	router := setupEnv(t)

	w := doRequest(t, router, http.MethodGet, "/api/v1/docs", "", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	require.Contains(t, w.Body.String(), "Cairo - API Documentation")

	w2 := doRequest(t, router, http.MethodGet, "/api/v1/docs/../", "", nil)
	require.NotEqual(t, http.StatusOK, w2.Code, "path traversal must not reach other routes")
	require.NotContains(t, w2.Body.String(), "Cairo - API Documentation")
}
