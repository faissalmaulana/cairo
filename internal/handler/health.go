package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type HealthChecker interface {
	http.Handler
}

type DependenciesHealth struct {
	DB *sql.DB
}

func (dh *DependenciesHealth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]any{}
	healthy := true

	if err := dh.DB.PingContext(ctx); err != nil {
		checks["database"] = map[string]any{"status": "unhealthy", "error": err.Error()}
		healthy = false
	} else {
		checks["database"] = map[string]any{"status": "healthy"}
	}

	status := http.StatusOK
	statusLabel := "healthy"
	if !healthy {
		status = http.StatusServiceUnavailable
		statusLabel = "unhealthy"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": statusLabel,
		"checks": checks,
	})
}