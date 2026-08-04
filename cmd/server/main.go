package main

import (
	"io"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/faissalmaulana/cairo/internal/app"
	"github.com/faissalmaulana/cairo/internal/config"
	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/faissalmaulana/cairo/internal/middleware"
	apikey_repository "github.com/faissalmaulana/cairo/internal/repository/apikey"
	user_repository "github.com/faissalmaulana/cairo/internal/repository/user"
	apikey_service "github.com/faissalmaulana/cairo/internal/service/apikey"
	auth_service "github.com/faissalmaulana/cairo/internal/service/auth"
	token_service "github.com/faissalmaulana/cairo/internal/service/token"
	user_service "github.com/faissalmaulana/cairo/internal/service/user"
	"github.com/joho/godotenv"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using default configuration")
	}

	db, err := config.OpenDB(config.DBConfig{
		DSN:             helpers.GetEnv("DB_DSN", ""),
		MaxOpenConns:    helpers.GetEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    helpers.GetEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: helpers.GetEnvDuration("DB_CONN_MAX_LIFETIME", time.Hour),
		ConnMaxIdleTime: helpers.GetEnvDuration("DB_CONN_MAX_IDLE_TIME", time.Minute),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rdb, err := config.NewRedis(config.RedisConfig{
		Addr:     helpers.GetEnv("REDIS_ADDR", ""),
		Password: helpers.GetEnv("REDIS_PASSWORD", ""),
		DB:       helpers.GetEnvInt("REDIS_DB", 0),
		Protocol: helpers.GetEnvInt("REDIS_PROTOCOL", 3),
	})
	if err != nil {
		log.Fatal(err)
	}

	userRepo := user_repository.NewSQLiteUserRepository(db)

	filelogger := &lumberjack.Logger{
		Filename:   helpers.GetEnv("LOG_LOCATION", "./log/cairo.log"),
		MaxSize:    helpers.GetEnvInt("LOG_MAX_SIZE", 50), // megabytes
		MaxBackups: helpers.GetEnvInt("LOG_MAX_BACKUPS", 3),
		Compress:   true,
	}

	// Combine stdout and Lumberjack file writer
	multiWriter := io.MultiWriter(os.Stdout, filelogger)
	logger := slog.New(slog.NewJSONHandler(multiWriter, nil))

	userSvc := user_service.NewUserService(userRepo, logger)

	apiKeyRepo := apikey_repository.NewSQLiteApiKeyRepository(db)
	apiKeySvc := apikey_service.NewApiKeyService(apiKeyRepo, logger)

	tokenSvc := token_service.NewTokenService(
		helpers.GetEnv("JWT_SECRET", ""),
		helpers.GetEnvDuration("JWT_ACCESS_TTL", 5*time.Minute),
		helpers.GetEnvDuration("JWT_REFRESH_TTL", 168*time.Hour),
		rdb,
		logger,
	)

	authSvc := auth_service.NewAuthService(db, userRepo, apiKeyRepo, logger)

	userHandler := handler.NewUserHandler(userSvc, tokenSvc, authSvc)
	authMiddleware := middleware.NewAuthMiddleware(tokenSvc)

	apiKeyHandler := handler.NewApiKeyHandler(apiKeySvc)
	apiKeyMiddleware := middleware.NewApiKeyMiddleware(apiKeySvc)

	app := app.New(
		userHandler,
		apiKeyHandler,
		authMiddleware,
		apiKeyMiddleware,
		helpers.GetEnv("SERVER_ADDR", "localhost:8080"),
		helpers.GetEnvDuration("SERVER_READ_HEADER_TIMEOUT", 5*time.Second),
		helpers.GetEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
		helpers.GetEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		helpers.GetEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		helpers.GetEnv("SERVER_MODE", "development"),
		&handler.DependenciesHealth{DB: db, Redis: rdb},
		helpers.GetEnv("HEALTH_ADDR", "localhost:8081"),
		logger,
	)
	app.Run()
}
