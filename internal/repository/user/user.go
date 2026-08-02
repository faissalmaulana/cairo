package user_repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/faissalmaulana/cairo/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, newUsr model.User) error
	GetPasswordByEmail(ctx context.Context, email string) (string, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

type SQLiteUserRepository struct {
	db *sql.DB
}

func NewSQLiteUserRepository(db *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{
		db: db,
	}
}

func (ud *SQLiteUserRepository) Create(ctx context.Context, newUsr model.User) error {
	query := `INSERT INTO users(username,email,password,createdAt) VALUES(?,?,?,?)`

	queryctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := ud.db.ExecContext(
		queryctx,
		query,
		newUsr.Username,
		newUsr.Email,
		newUsr.Password,
		time.Now(),
	)

	return err
}
func (ud *SQLiteUserRepository) GetPasswordByEmail(ctx context.Context, email string) (string, error) {

	query := `SELECT password FROM users WHERE email=?`

	queryctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var resultRow string
	err := ud.db.QueryRowContext(queryctx, query, email).Scan(&resultRow)

	return resultRow, err
}

func (ud *SQLiteUserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email=?) `

	queryctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var exist bool
	err := ud.db.QueryRowContext(queryctx, query, email).Scan(&exist)

	return exist, err

}

func (ud *SQLiteUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id,username,email FROM users WHERE email = ?`

	queryctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var user model.User

	err := ud.db.QueryRowContext(queryctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
	)

	defer cancel()

	return &user, err

}
