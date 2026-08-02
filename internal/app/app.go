package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/faissalmaulana/cairo/internal/middleware"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type Application struct {
	userHandler    *handler.UserResource
	authMiddleware *middleware.AuthMiddleware
	sessionStore   sessions.Store
}

func New(usrHandler *handler.UserResource, authMiddleware *middleware.AuthMiddleware, sessStore sessions.Store) *Application {
	return &Application{
		userHandler:    usrHandler,
		authMiddleware: authMiddleware,
		sessionStore:   sessStore,
	}
}

func (app *Application) mux() http.Handler {
	gin.DisableConsoleColor()
	router := gin.Default()
	router.SetTrustedProxies(nil)

	router.Use(sessions.Sessions("authentication_session", app.sessionStore))

	router.POST("/signup", app.userHandler.SignUp)
	router.POST("/signin", app.userHandler.SignIn)

	router.GET("/account", app.authMiddleware.CheckAuth, app.userHandler.Account)
	router.GET("/logout", app.userHandler.Logout)

	return router
}

func (app *Application) Run() {

	srv := &http.Server{
		Addr:    "localhost:8080",
		Handler: app.mux(),
	}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no params) by default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Println("Server Shutdown:", err)
	}

	log.Println("Server exiting")
}
