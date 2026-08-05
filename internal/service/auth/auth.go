package auth_service

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/faissalmaulana/cairo/internal/model"
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
// fails, the transaction rolls back and no user persists. The plaintext key is
// returned exactly once so the caller can hand it to the user.
func (as *AuthService) SignUp(ctx context.Context, nu user_service.SignUpInput) (string, model.ApiKey, error) {
	newUser, err := user_service.PrepareSignUp(nu)
	if err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", model.ApiKey{}, err
	}

	tx, err := as.db.BeginTx(ctx, nil)
	if err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", model.ApiKey{}, err
	}
	defer tx.Rollback()

	userID, err := as.userRepo.WithTx(tx).Create(ctx, *newUser)
	if err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", model.ApiKey{}, err
	}

	key, err := apikey_service.GenerateKey()
	if err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", model.ApiKey{}, err
	}
	key.UserID = userID
	key.CreatedAt = time.Now().UTC()

	keyID, err := as.apiKeyRepo.WithTx(tx).Create(ctx, key)
	if err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", model.ApiKey{}, err
	}
	key.ID = keyID

	if err := tx.Commit(); err != nil {
		helpers.LoggerFromContext(ctx, as.logger).Error("internal_error", "err_msg", err)
		return "", model.ApiKey{}, err
	}

	return userID, key, nil
}
