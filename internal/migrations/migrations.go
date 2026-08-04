package migrations

import (
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedMigrations embed.FS

// Up runs all pending migrations against the given database.
func Up(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)
	goose.SetDialect("sqlite3")

	return goose.Up(db, ".")
}

func Down(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)
	goose.SetDialect("sqlite3")

	return goose.Up(db, ".")
}
