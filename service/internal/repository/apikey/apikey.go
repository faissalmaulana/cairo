package apikey_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/faissalmaulana/cairo/internal/model"
	"github.com/faissalmaulana/cairo/internal/repository"
	"github.com/faissalmaulana/cairo/internal/variables"
	"github.com/google/uuid"
)

type ApiKeyRepository interface {
	Create(ctx context.Context, newKey model.ApiKey) (string, error)
	GetByKey(ctx context.Context, key string) (*model.ApiKey, error)
	ListByUser(ctx context.Context, userID string) ([]model.ApiKey, error)
	Revoke(ctx context.Context, id, userID string) error
	TouchLastUsed(ctx context.Context, id string) error
}

type SQLiteApiKeyRepository struct {
	db repository.DBTX
}

func NewSQLiteApiKeyRepository(db *sql.DB) *SQLiteApiKeyRepository {
	return &SQLiteApiKeyRepository{
		db: db,
	}
}

// WithTx returns a repository bound to the given transaction, so multiple
// repositories can participate in the same atomic unit of work.
func (ud *SQLiteApiKeyRepository) WithTx(tx *sql.Tx) *SQLiteApiKeyRepository {
	return &SQLiteApiKeyRepository{
		db: tx,
	}
}

var (
	ErrApiKeyNotFound = errors.New("api key not found")
)

func (ud *SQLiteApiKeyRepository) Create(ctx context.Context, newKey model.ApiKey) (string, error) {
	id := uuid.NewString()
	query := `INSERT INTO api_keys(id,user_id,key,createdAt) VALUES(?,?,?,?)`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	var createdAt int64
	if !newKey.CreatedAt.IsZero() {
		createdAt = newKey.CreatedAt.Unix()
	} else {
		createdAt = time.Now().Unix()
	}

	_, err := ud.db.ExecContext(
		queryctx,
		query,
		id,
		newKey.UserID,
		newKey.Key,
		createdAt,
	)

	return id, err
}

func (ud *SQLiteApiKeyRepository) GetByKey(ctx context.Context, key string) (*model.ApiKey, error) {
	query := `SELECT id,user_id,key,last_used_at,createdAt FROM api_keys WHERE key=?`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	var keyRow model.ApiKey
	var lastUsedAt sql.NullInt64
	var createdAt int64

	err := ud.db.QueryRowContext(queryctx, query, key).Scan(
		&keyRow.ID,
		&keyRow.UserID,
		&keyRow.Key,
		&lastUsedAt,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrApiKeyNotFound
	}
	if err != nil {
		return nil, err
	}

	if lastUsedAt.Valid {
		t := time.Unix(lastUsedAt.Int64, 0)
		keyRow.LastUsedAt = &t
	}
	keyRow.CreatedAt = time.Unix(createdAt, 0)

	return &keyRow, nil
}

func (ud *SQLiteApiKeyRepository) ListByUser(ctx context.Context, userID string) ([]model.ApiKey, error) {
	query := `SELECT id,user_id,key,last_used_at,createdAt FROM api_keys WHERE user_id=? ORDER BY createdAt DESC`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	rows, err := ud.db.QueryContext(queryctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]model.ApiKey, 0)
	for rows.Next() {
		var key model.ApiKey
		var lastUsedAt sql.NullInt64
		var createdAt int64

		if err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.Key,
			&lastUsedAt,
			&createdAt,
		); err != nil {
			return nil, err
		}

		if lastUsedAt.Valid {
			t := time.Unix(lastUsedAt.Int64, 0)
			key.LastUsedAt = &t
		}
		key.CreatedAt = time.Unix(createdAt, 0)
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

func (ud *SQLiteApiKeyRepository) Revoke(ctx context.Context, id, userID string) error {
	query := `DELETE FROM api_keys WHERE id=? AND user_id=?`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	result, err := ud.db.ExecContext(queryctx, query, id, userID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrApiKeyNotFound
	}

	return nil
}

func (ud *SQLiteApiKeyRepository) TouchLastUsed(ctx context.Context, id string) error {
	query := `UPDATE api_keys SET last_used_at=? WHERE id=?`

	queryctx, cancel := context.WithTimeout(ctx, variables.ContextTimeOut)
	defer cancel()

	_, err := ud.db.ExecContext(queryctx, query, time.Now().Unix(), id)

	return err
}
