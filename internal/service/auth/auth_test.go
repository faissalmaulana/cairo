package auth_service

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	apikey_repository "github.com/faissalmaulana/cairo/internal/repository/apikey"
	user_repository "github.com/faissalmaulana/cairo/internal/repository/user"
	user_service "github.com/faissalmaulana/cairo/internal/service/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", t.TempDir()+"/test.db")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users(
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			createdAt int
		);
		CREATE TABLE IF NOT EXISTS api_keys(
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			key_hash TEXT NOT NULL UNIQUE,
			prefix TEXT NOT NULL,
			last_used_at int,
			createdAt int
		);
	`)
	require.NoError(t, err)

	return db
}

func signUpInput() user_service.SignUpInput {
	return user_service.SignUpInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "supersecret",
	}
}

func TestSignUp(t *testing.T) {
	t.Run("success creates user and its first api key", func(t *testing.T) {
		db := setupDB(t)
		svc := NewAuthService(db,
			user_repository.NewSQLiteUserRepository(db),
			apikey_repository.NewSQLiteApiKeyRepository(db),
		)

		userID, err := svc.SignUp(context.Background(), signUpInput())

		require.NoError(t, err)
		assert.NotEmpty(t, userID)

		var userCount int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM users WHERE id=?`, userID).Scan(&userCount))
		assert.Equal(t, 1, userCount)

		var keyID, keyHash, prefix string
		var userIDInKey string
		err = db.QueryRow(`SELECT id, user_id, key_hash, prefix FROM api_keys WHERE user_id=?`, userID).
			Scan(&keyID, &userIDInKey, &keyHash, &prefix)
		require.NoError(t, err)
		assert.Equal(t, userID, userIDInKey)
		assert.True(t, strings.HasPrefix(prefix, "cairo_"))
		assert.NotEmpty(t, keyHash)
	})

	t.Run("api key insert failure rolls back the user", func(t *testing.T) {
		db := setupDB(t)
		_, err := db.Exec(`DROP TABLE api_keys`)
		require.NoError(t, err)

		svc := NewAuthService(db,
			user_repository.NewSQLiteUserRepository(db),
			apikey_repository.NewSQLiteApiKeyRepository(db),
		)

		_, err = svc.SignUp(context.Background(), signUpInput())
		assert.Error(t, err)

		var userCount int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount))
		assert.Equal(t, 0, userCount)
	})
}
