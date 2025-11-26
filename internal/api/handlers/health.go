package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db *sql.DB
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Health handles GET /health
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	components := map[string]string{
		"database": "healthy",
		"cache":    "healthy",
		"onedrive": "healthy",
	}

	// Check database
	if err := h.db.Ping(); err != nil {
		status = "unhealthy"
		components["database"] = "unhealthy"
	}

	response := map[string]interface{}{
		"status":     status,
		"timestamp":  nil,
		"components": components,
	}

	w.Header().Set("Content-Type", "application/json")
	if status == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(response)
}

// Info handles GET /info
func (h *HealthHandler) Info(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"name":        "OneDrive Storage Middleware",
		"version":     "1.0.0",
		"api_version": "v1",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
