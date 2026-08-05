package app

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/faissalmaulana/cairo/internal/handler"
	"github.com/faissalmaulana/cairo/internal/middleware"
	"github.com/gin-gonic/gin"
)

type Application struct {
	userHandler          *handler.UserHandler
	apiKeyHandler        *handler.ApiKeyHandler
	objectStorageHandler *handler.ObjectStorageHandler
	authMiddleware       *middleware.AuthMiddleware
	apiKeyMiddleware     *middleware.ApiKeyMiddleware
	addr                 string
	readHeaderTimeout    time.Duration
	readTimeout          time.Duration
	writeTimeout         time.Duration
	idleTimeout          time.Duration
	mode                 string
	health               handler.HealthChecker
	healthAddr           string
	logger               *slog.Logger
}

func New(
	userHandler *handler.UserHandler,
	apiKeyHandler *handler.ApiKeyHandler,
	objectStorageHandler *handler.ObjectStorageHandler,
	authMiddleware *middleware.AuthMiddleware,
	apiKeyMiddleware *middleware.ApiKeyMiddleware,
	addr string,
	readHeaderTimeout, readTimeout, writeTimeout, idleTimeout time.Duration,
	mode string,
	health handler.HealthChecker,
	healthAddr string,
	logger *slog.Logger,
) *Application {

	return &Application{
		userHandler:          userHandler,
		apiKeyHandler:        apiKeyHandler,
		objectStorageHandler: objectStorageHandler,
		authMiddleware:       authMiddleware,
		apiKeyMiddleware:     apiKeyMiddleware,
		addr:                 addr,
		readHeaderTimeout:    readHeaderTimeout,
		readTimeout:          readTimeout,
		writeTimeout:         writeTimeout,
		idleTimeout:          idleTimeout,
		mode:                 mode,
		health:               health,
		healthAddr:           healthAddr,
		logger:               logger,
	}
}

func (app *Application) Mux() http.Handler {
	gin.DisableConsoleColor()
	gin.SetMode(goodModeToGinMode(app.mode))

	router := gin.New()

	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.SlogMiddleware(app.logger))
	router.SetTrustedProxies(nil)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/signup", app.userHandler.SignUp)
		v1.POST("/signin", app.userHandler.SignIn)
		v1.POST("/refresh", app.userHandler.Refresh)
		v1.GET("/account", app.authMiddleware.CheckAuth, app.userHandler.Account)
		v1.POST("/account/logout", app.authMiddleware.CheckAuth, app.userHandler.Logout)

		apiKeys := v1.Group("/account/apikeys", app.authMiddleware.CheckAuth)
		{
			apiKeys.POST("", app.apiKeyHandler.Create)
			apiKeys.GET("", app.apiKeyHandler.List)
			apiKeys.DELETE("/:id", app.apiKeyHandler.Revoke)
		}

		accounts := v1.Group("/accounts/:account_id", app.apiKeyMiddleware.CheckApiKey, app.apiKeyMiddleware.RequireAccount)
		{
			accounts.GET("/buckets", app.objectStorageHandler.ListBuckets)
			accounts.POST("/buckets", app.objectStorageHandler.CreateBucket)
			accounts.GET("/buckets/:bucket_name", app.objectStorageHandler.GetBucket)
			accounts.PATCH("/buckets/:bucket_name/visibility", app.objectStorageHandler.SetBucketVisibility)
			accounts.DELETE("/buckets/:bucket_name", app.objectStorageHandler.DeleteBucket)

			accounts.GET("/buckets/:bucket_name/objects", app.objectStorageHandler.ListObjects)
			accounts.PUT("/buckets/:bucket_name/objects/*object_key", app.objectStorageHandler.UploadObject)
			accounts.GET("/buckets/:bucket_name/objects/*object_key", app.objectStorageHandler.GetObject)
			accounts.DELETE("/buckets/:bucket_name/objects/*object_key", app.objectStorageHandler.DeleteObject)
		}

		// Public object access: no account id, no api key. Access is granted
		// by the bucket's visibility alone; the file is served from the public
		// symlink namespace.
		publicObjects := v1.Group("/public/buckets/:bucket_name/objects")
		{
			publicObjects.GET("/*object_key", app.objectStorageHandler.GetPublicObject)
		}

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

func (app *Application) HealthMux() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", app.health)
	return mux
}

func (app *Application) Run() {
	srv := &http.Server{
		Addr:              app.addr,
		Handler:           app.Mux(),
		ReadHeaderTimeout: app.readHeaderTimeout,
		ReadTimeout:       app.readTimeout,
		WriteTimeout:      app.writeTimeout,
		IdleTimeout:       app.idleTimeout,
	}

	healthSrv := &http.Server{
		Addr:              app.healthAddr,
		Handler:           app.HealthMux(),
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
