package auth_service

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/faissalmaulana/cairo/internal/helpers"
	apikey_repository "github.com/faissalmaulana/cairo/internal/repository/apikey"
	user_repository "github.com/faissalmaulana/cairo/internal/repository/user"
	apikey_service "github.com/faissalmaulana/cairo/internal/service/apikey"
	user_service "github.com/faissalmaulana/cairo/internal/service/user"
)

type AuthService struct {
	db         *sql.DB
	userRepo   *user_repository.SQLiteUserRepository
	apiKeyRepo *apikey_repository.SQLiteApiKeyRepository
	logger     *slog.Logger
}

func NewAuthService(
	db *sql.DB,
	userRepo *user_repository.SQLiteUserRepository,
	apiKeyRepo *apikey_repository.SQLiteApiKeyRepository,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		db:         db,
		userRepo:   userRepo,
		apiKeyRepo: apiKeyRepo,
		logger:     logger,
	}
}

// SignUp creates a new user and its first api key atomically. If either insert
// fails, the transaction rolls back and no user persists.
func (as *AuthService) SignUp(ctx context.Context, nu user_service.SignUpInput) (string, error) {
	newUser, err := user_service.PrepareSignUp(nu)
	if err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", err
	}

	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", err
	}
	defer tx.Rollback()

	userID, err := as.userRepo.WithTx(tx).Create(ctx, *newUser)
	if err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", err
	}

	key, err := apikey_service.GenerateKey()
	if err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", err
	}
	key.UserID = userID

	if _, err := as.apiKeyRepo.WithTx(tx).Create(ctx, key.ApiKey); err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", err
	}

	if err := tx.Commit(); err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", err
	}

	return userID, nil
}
