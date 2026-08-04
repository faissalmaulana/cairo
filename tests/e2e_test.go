package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/faissalmaulana/cairo/internal/app"
	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/faissalmaulana/cairo/internal/middleware"
	"github.com/faissalmaulana/cairo/internal/migrations"
	apikey_repository "github.com/faissalmaulana/cairo/internal/repository/apikey"
	user_repository "github.com/faissalmaulana/cairo/internal/repository/user"
	apikey_service "github.com/faissalmaulana/cairo/internal/service/apikey"
	auth_service "github.com/faissalmaulana/cairo/internal/service/auth"
	token_service "github.com/faissalmaulana/cairo/internal/service/token"
	user_service "github.com/faissalmaulana/cairo/internal/service/user"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "modernc.org/sqlite"
)

func startRedis(t *testing.T, ctx context.Context) string {
	t.Helper()

	redisC, err := testcontainers.Run(
		ctx,
		"redis:alpine",
		testcontainers.WithExposedPorts("6379/tcp"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("6379/tcp"),
			wait.ForLog("Ready to accept connections"),
		),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, redisC)

	endpoint, err := redisC.Endpoint(ctx, "")
	require.NoError(t, err)

	return endpoint
}

// setupEnv boots redis (testcontainers), a fresh sqlite test database migrated
// with goose.
func setupEnv(t *testing.T) http.Handler {
	t.Helper()

	ctx := context.Background()
	endpoint := startRedis(t, ctx)

	rdb := redis.NewClient(&redis.Options{Addr: endpoint})
	t.Cleanup(func() { rdb.Close() })

	dsn := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
	})

	require.NoError(t, migrations.Up(db))

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	userRepo := user_repository.NewSQLiteUserRepository(db)
	userSvc := user_service.NewUserService(userRepo, logger)

	apiKeyRepo := apikey_repository.NewSQLiteApiKeyRepository(db)
	apiKeySvc := apikey_service.NewApiKeyService(apiKeyRepo, logger)

	authSvc := auth_service.NewAuthService(db, userRepo, apiKeyRepo, logger)

	tokenSvc := token_service.NewTokenService(
		"e2e-secret",
		5*time.Minute,
		168*time.Hour,
		rdb,
		logger,
	)

	userHandler := handler.NewUserHandler(userSvc, tokenSvc, authSvc)
	apiKeyHandler := handler.NewApiKeyHandler(apiKeySvc)
	authMiddleware := middleware.NewAuthMiddleware(tokenSvc)
	apiKeyMiddleware := middleware.NewApiKeyMiddleware(apiKeySvc)

	application := app.New(
		userHandler,
		apiKeyHandler,
		authMiddleware,
		apiKeyMiddleware,
		"",
		0, 0, 0, 0,
		"test",
		&handler.DependenciesHealth{DB: db, Redis: rdb},
		"",
		logger,
	)

	checkHealthDependency(t, application.HealthMux())
	return application.Mux()

}

func checkHealthDependency(t *testing.T, router http.Handler) {
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/healthz", nil)
	assert.NoError(t, err)

	router.ServeHTTP(w, req)

	var resp map[string]any

	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "healthy", resp["status"])

}

// doRequest performs a request against the router, setting the JSON body (if
// any) and an optional bearer token. It returns the recorder for assertions.
func doRequest(t *testing.T, router http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, path, rdr)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// signUp registers a fresh user and returns the token response for
// authenticating subsequent requests.
func signUp(t *testing.T, router http.Handler, body handler.SignUpRequest) handler.TokenResponse {
	t.Helper()

	w := doRequest(t, router, http.MethodPost, "/api/v1/signup", "", body)

	require.Equal(t, http.StatusCreated, w.Code)

	return okResponse[handler.TokenResponse](t, w)
}
