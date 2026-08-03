package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/faissalmaulana/cairo/internal/middleware"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type Application struct {
	userHandler       *handler.UserHandler
	authMiddleware    *middleware.AuthMiddleware
	sessionStore      sessions.Store
	addr              string
	readHeaderTimeout time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	mode              string
	health            handler.HealthChecker
	healthAddr        string
}

func New(
	userHandler *handler.UserHandler,
	authMiddleware *middleware.AuthMiddleware,
	sessStore sessions.Store,
	addr string,
	readHeaderTimeout, readTimeout, writeTimeout, idleTimeout time.Duration,
	mode string,
	health handler.HealthChecker,
	healthAddr string,
) *Application {

	return &Application{
		userHandler:       userHandler,
		authMiddleware:    authMiddleware,
		sessionStore:      sessStore,
		addr:              addr,
		readHeaderTimeout: readHeaderTimeout,
		readTimeout:       readTimeout,
		writeTimeout:      writeTimeout,
		idleTimeout:       idleTimeout,
		mode:              mode,
		health:            health,
		healthAddr:        healthAddr,
	}
}

func (app *Application) mux() http.Handler {
	gin.DisableConsoleColor()
	gin.SetMode(goodModeToGinMode(app.mode))

	router := gin.Default()
	router.SetTrustedProxies(nil)

	v1 := router.Group("/api/v1")
	{
		v1.Use(sessions.Sessions("authentication_session", app.sessionStore))
		v1.POST("/signup", app.userHandler.SignUp)
		v1.POST("/signin", app.userHandler.SignIn)
		v1.GET("/account", app.authMiddleware.CheckAuth, app.userHandler.Account)
		v1.GET("/logout", app.authMiddleware.CheckAuth, app.userHandler.Logout)
	}

	return router
}

// goodModeToGinMode maps the app's mode string to a gin mode constant,
// defaulting to debug for development/unknown values.
func goodModeToGinMode(mode string) string {
	switch strings.ToLower(mode) {
	case gin.ReleaseMode, "prod", "production":
		return gin.ReleaseMode
	case gin.TestMode:
		return gin.TestMode
	default: // dev, development, "" and anything else
		return gin.DebugMode
	}
}

func (app *Application) healthMux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", app.health)
	return mux
}

func (app *Application) Run() {
	srv := &http.Server{
		Addr:              app.addr,
		Handler:           app.mux(),
		ReadHeaderTimeout: app.readHeaderTimeout,
		ReadTimeout:       app.readTimeout,
		WriteTimeout:      app.writeTimeout,
		IdleTimeout:       app.idleTimeout,
	}

	healthSrv := &http.Server{
		Addr:              app.healthAddr,
		Handler:           app.healthMux(),
		ReadHeaderTimeout: app.readHeaderTimeout,
		IdleTimeout:       app.idleTimeout,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen %s: %v", srv.Addr, err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Printf("starting health server on %s", healthSrv.Addr)
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("health listen %s: %v", healthSrv.Addr, err)
		}
		log.Printf("health server on %s stopped", healthSrv.Addr)
	}()

	// Wait for interrupt signal to gracefully shutdown the servers.
	quit := make(chan os.Signal, 1)
	// kill (no params) by default sends syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdown := func(s *http.Server, name string) {
		log.Printf("shutting down %s", name)
		if err := s.Shutdown(ctx); err != nil {
			log.Printf("%s shutdown: %v", name, err)
		}
		log.Printf("%s closed", name)
	}
	shutdown(srv, "server")
	shutdown(healthSrv, "health server")

	wg.Wait()
	log.Println("Servers exiting")
}
