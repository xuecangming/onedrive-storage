package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         string
}

// DefaultCORSConfig returns default CORS configuration
// In production, set CORS_ALLOWED_ORIGINS environment variable to restrict origins
func DefaultCORSConfig() *CORSConfig {
	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	allowedOrigins := []string{"*"}
	if origins != "" {
		allowedOrigins = strings.Split(origins, ",")
		for i := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
		}
	}

	return &CORSConfig{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
		MaxAge:         "86400",
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
// Uses DefaultCORSConfig which respects CORS_ALLOWED_ORIGINS env var
func CORSMiddleware(next http.Handler) http.Handler {
	return CORSMiddlewareWithConfig(DefaultCORSConfig())(next)
}

// CORSMiddlewareWithConfig creates a CORS middleware with custom configuration
func CORSMiddlewareWithConfig(config *CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowedOrigin := ""

			// Check if origin is allowed
			for _, o := range config.AllowedOrigins {
				if o == "*" || o == origin {
					allowedOrigin = o
					if o == "*" {
						allowedOrigin = "*"
					} else {
						allowedOrigin = origin
					}
					break
				}
			}

			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				w.Header().Set("Access-Control-Max-Age", config.MaxAge)

				// Add Vary header for proper caching when not using wildcard
				if allowedOrigin != "*" {
					w.Header().Add("Vary", "Origin")
				}
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf(
			"%s %s %d %s",
			r.Method,
			r.RequestURI,
			wrapped.statusCode,
			time.Since(start),
		)
	})
}

// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				// Print stack trace
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				log.Printf("Stack trace:\n%s", buf[:n])
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	requests map[string]*clientLimiter
	mu       sync.RWMutex
	rate     int           // requests per interval
	interval time.Duration // time interval
	cleanup  time.Duration // cleanup interval for old entries
}

type clientLimiter struct {
	tokens    int
	lastCheck time.Time
}

// NewRateLimiter creates a new rate limiter
// rate: number of requests allowed per interval
// interval: time period for the rate limit
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientLimiter),
		rate:     rate,
		interval: interval,
		cleanup:  5 * time.Minute,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// cleanupLoop periodically removes old entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, limiter := range rl.requests {
			if now.Sub(limiter.lastCheck) > rl.cleanup {
				delete(rl.requests, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request from the given key is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	limiter, exists := rl.requests[key]

	if !exists {
		rl.requests[key] = &clientLimiter{
			tokens:    rl.rate - 1,
			lastCheck: now,
		}
		return true
	}

	// Replenish tokens based on time elapsed
	elapsed := now.Sub(limiter.lastCheck)
	tokensToAdd := int(elapsed / rl.interval) * rl.rate
	limiter.tokens = min(rl.rate, limiter.tokens+tokensToAdd)
	limiter.lastCheck = now

	if limiter.tokens > 0 {
		limiter.tokens--
		return true
	}

	return false
}

// RateLimitMiddleware creates a rate limiting middleware
// Limits requests per IP address
func RateLimitMiddleware(rate int, interval time.Duration) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(rate, interval)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			clientIP := getClientIP(r)

			if !limiter.Allow(clientIP) {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", interval.Seconds()))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// Remove port number if present
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx]
	}
	return addr
}
