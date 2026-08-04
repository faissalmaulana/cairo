package repository

import (
	"context"
	"database/sql"
)

// DBTX is implemented by both *sql.DB and *sql.Tx, letting repositories run
// either standalone or as part of a shared transaction.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}
