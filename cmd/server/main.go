package main

import (
	"log"
	"os"
	"time"

	"github.com/faissalmaulana/cairo/internal/app"
	"github.com/faissalmaulana/cairo/internal/config"
	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/faissalmaulana/cairo/internal/middleware"
	"github.com/faissalmaulana/cairo/internal/repository/user"
	"github.com/faissalmaulana/cairo/internal/service/user"
	"github.com/gin-contrib/sessions/filesystem"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using default configuration")
	}

	db, err := config.OpenDB(config.DBConfig{
		DSN:             helpers.GetEnv("DB_DSN", "./database/foo.db"),
		MaxOpenConns:    helpers.GetEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    helpers.GetEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: helpers.GetEnvDuration("DB_CONN_MAX_LIFETIME", time.Hour),
		ConnMaxIdleTime: helpers.GetEnvDuration("DB_CONN_MAX_IDLE_TIME", time.Minute),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Verify the connection is alive
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	sessionPath := helpers.GetEnv("SESSION_PATH", "./database/sessions")
	if err := os.MkdirAll(sessionPath, 0o755); err != nil {
		log.Fatal(err)
	}

	userRepo := user_repository.NewSQLiteUserRepository(db)
	userSvc := user_service.NewUserService(userRepo)

	userHandler := handler.NewUserHandler(userSvc)
	authMiddleware := middleware.NewAuthMiddleware(userSvc)

	sessionStore := filesystem.NewStore(sessionPath, []byte(helpers.GetEnv("SESSION_SECRET", "hello,world")))
	app := app.New(
		userHandler,
		authMiddleware,
		sessionStore,
		helpers.GetEnv("SERVER_ADDR", "localhost:8080"),
		helpers.GetEnvDuration("SERVER_READ_HEADER_TIMEOUT", 5*time.Second),
		helpers.GetEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second),
		helpers.GetEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second),
		helpers.GetEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
	)
	app.Run()
}
