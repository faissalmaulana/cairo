package main

import (
	"log"
	"os"
	"time"

	"github.com/faissalmaulana/cairo/internal/app"
	"github.com/faissalmaulana/cairo/internal/config"
	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/faissalmaulana/cairo/internal/middleware"
	"github.com/faissalmaulana/cairo/internal/repository/user"
	"github.com/faissalmaulana/cairo/internal/service/user"
	"github.com/gin-contrib/sessions/filesystem"
)

func main() {
	db, err := config.OpenDB(config.DBConfig{
		DSN:             "./database/foo.db",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Verify the connection is alive
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll("./database/sessions", 0o755); err != nil {
		log.Fatal(err)
	}

	userRepo := user_repository.NewSQLiteUserRepository(db)
	userSvc := user_service.NewUserService(userRepo)

	userHandler := handler.NewUserHandler(userSvc)
	authMiddleware := middleware.NewAuthMiddleware(userSvc)

	sessionStore := filesystem.NewStore("./database/sessions", []byte("hello,world"))
	app := app.New(userHandler, authMiddleware, sessionStore)
	app.Run()
}
