package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db        *sql.DB
	startTime time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{
		db:        db,
		startTime: time.Now(),
	}
}

// ComponentHealth represents health status of a component
type ComponentHealth struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status     string                     `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Uptime     string                     `json:"uptime"`
	Components map[string]ComponentHealth `json:"components"`
}

// Health handles GET /health
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	components := make(map[string]ComponentHealth)

	// Check database
	dbHealth := h.checkDatabase()
	components["database"] = dbHealth
	if dbHealth.Status != "healthy" {
		status = "unhealthy"
	}

	// Check system resources
	sysHealth := h.checkSystem()
	components["system"] = sysHealth

	// Cache check (currently disabled)
	components["cache"] = ComponentHealth{
		Status:  "disabled",
		Message: "Cache is not enabled",
	}

	// OneDrive check (placeholder - would need account service)
	components["onedrive"] = ComponentHealth{
		Status:  "unknown",
		Message: "OneDrive health check not yet implemented",
	}

	// Calculate uptime
	uptime := time.Since(h.startTime)

	response := HealthResponse{
		Status:     status,
		Timestamp:  time.Now(),
		Uptime:     uptime.String(),
		Components: components,
	}

	w.Header().Set("Content-Type", "application/json")
	if status == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(response)
}

// checkDatabase checks database health
func (h *HealthHandler) checkDatabase() ComponentHealth {
	start := time.Now()
	err := h.db.Ping()
	latency := time.Since(start)

	if err != nil {
		return ComponentHealth{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}

	// Check connection pool stats
	stats := h.db.Stats()
	details := map[string]interface{}{
		"latency_ms":    latency.Milliseconds(),
		"open_conns":    stats.OpenConnections,
		"in_use":        stats.InUse,
		"idle":          stats.Idle,
		"wait_count":    stats.WaitCount,
		"wait_duration": stats.WaitDuration.String(),
	}

	health := "healthy"
	if stats.WaitCount > 100 {
		health = "degraded"
	}

	return ComponentHealth{
		Status:  health,
		Details: details,
	}
}

// checkSystem checks system resource health
func (h *HealthHandler) checkSystem() ComponentHealth {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	details := map[string]interface{}{
		"goroutines":     runtime.NumGoroutine(),
		"alloc_mb":       m.Alloc / 1024 / 1024,
		"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
		"sys_mb":         m.Sys / 1024 / 1024,
		"num_gc":         m.NumGC,
		"last_gc":        time.Unix(0, int64(m.LastGC)).Format(time.RFC3339),
	}

	status := "healthy"
	// Check if memory usage is too high (>1GB)
	if m.Alloc > 1024*1024*1024 {
		status = "degraded"
	}

	// Check if too many goroutines (>10000)
	if runtime.NumGoroutine() > 10000 {
		status = "degraded"
	}

	return ComponentHealth{
		Status:  status,
		Details: details,
	}
}

// Info handles GET /info
func (h *HealthHandler) Info(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"name":        "OneDrive Storage Middleware",
		"version":     "1.0.0",
		"api_version": "v1",
		"go_version":  runtime.Version(),
		"uptime":      time.Since(h.startTime).String(),
		"started_at":  h.startTime.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Ready handles GET /ready (Kubernetes readiness probe)
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// Check if server is ready to accept requests
	err := h.db.Ping()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "not ready",
			"message": "database connection not available",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
	})
}

// Live handles GET /live (Kubernetes liveness probe)
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - if we can respond, we're alive
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
	})
}
