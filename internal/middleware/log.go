package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/faissalmaulana/cairo/internal/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// SlogMiddleware logs one request summary line per request and injects a
// request-scoped logger into the request context. Services that log through
// helpers.LoggerFromContext pick up request_id/method/path so their error
// lines correlate with the summary line.
func SlogMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		requestID, _ := c.Get("request_id")

		reqLogger := logger.With(
			slog.String("request_id", requestID.(string)),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)

		ctx := context.WithValue(c.Request.Context(), helpers.RequestLoggerKey, reqLogger)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		reqLogger.Info("request",
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
		)
	}
}
