package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestCORSMiddleware(t *testing.T) {
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("sets CORS headers with wildcard origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("Access-Control-Allow-Origin = %v, want '*'", w.Header().Get("Access-Control-Allow-Origin"))
		}
		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Access-Control-Allow-Methods header is missing")
		}
		if w.Header().Get("Access-Control-Allow-Headers") == "" {
			t.Error("Access-Control-Allow-Headers header is missing")
		}
	})

	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("status code = %v, want %v", w.Code, http.StatusNoContent)
		}
	})

	t.Run("passes through regular requests", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

func TestCORSMiddlewareWithConfig(t *testing.T) {
	t.Run("restricts to specific origins", func(t *testing.T) {
		config := &CORSConfig{
			AllowedOrigins: []string{"http://allowed.com"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Content-Type"},
			MaxAge:         "3600",
		}

		handler := CORSMiddlewareWithConfig(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Allowed origin
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://allowed.com")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "http://allowed.com" {
			t.Errorf("Access-Control-Allow-Origin = %v, want 'http://allowed.com'", w.Header().Get("Access-Control-Allow-Origin"))
		}

		// Disallowed origin
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("Origin", "http://notallowed.com")
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)

		if w2.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("Access-Control-Allow-Origin should be empty for disallowed origin, got %v", w2.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("adds Vary header for specific origins", func(t *testing.T) {
		config := &CORSConfig{
			AllowedOrigins: []string{"http://allowed.com"},
			AllowedMethods: []string{"GET"},
			AllowedHeaders: []string{"Content-Type"},
			MaxAge:         "3600",
		}

		handler := CORSMiddlewareWithConfig(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://allowed.com")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Header().Get("Vary") != "Origin" {
			t.Errorf("Vary = %v, want 'Origin'", w.Header().Get("Vary"))
		}
	})
}

func TestDefaultCORSConfig(t *testing.T) {
	t.Run("uses wildcard by default", func(t *testing.T) {
		os.Unsetenv("CORS_ALLOWED_ORIGINS")
		config := DefaultCORSConfig()

		if len(config.AllowedOrigins) != 1 || config.AllowedOrigins[0] != "*" {
			t.Errorf("AllowedOrigins = %v, want ['*']", config.AllowedOrigins)
		}
	})

	t.Run("uses env var when set", func(t *testing.T) {
		os.Setenv("CORS_ALLOWED_ORIGINS", "http://example.com, http://another.com")
		defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

		config := DefaultCORSConfig()

		if len(config.AllowedOrigins) != 2 {
			t.Errorf("AllowedOrigins length = %v, want 2", len(config.AllowedOrigins))
		}
		if config.AllowedOrigins[0] != "http://example.com" {
			t.Errorf("AllowedOrigins[0] = %v, want 'http://example.com'", config.AllowedOrigins[0])
		}
	})
}

func TestRateLimiter(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		limiter := NewRateLimiter(5, time.Second)

		for i := 0; i < 5; i++ {
			if !limiter.Allow("client1") {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		limiter := NewRateLimiter(3, time.Second)

		// Exhaust the limit
		for i := 0; i < 3; i++ {
			limiter.Allow("client1")
		}

		// This should be blocked
		if limiter.Allow("client1") {
			t.Error("Request should be blocked after exceeding limit")
		}
	})

	t.Run("isolates different clients", func(t *testing.T) {
		limiter := NewRateLimiter(2, time.Second)

		// Exhaust limit for client1
		limiter.Allow("client1")
		limiter.Allow("client1")

		// client2 should still be allowed
		if !limiter.Allow("client2") {
			t.Error("Different client should have separate limit")
		}
	})

	t.Run("replenishes tokens over time", func(t *testing.T) {
		limiter := NewRateLimiter(1, 50*time.Millisecond)

		// Use the token
		limiter.Allow("client1")

		// Should be blocked
		if limiter.Allow("client1") {
			t.Error("Should be blocked immediately after using token")
		}

		// Wait for replenishment
		time.Sleep(60 * time.Millisecond)

		// Should be allowed again
		if !limiter.Allow("client1") {
			t.Error("Should be allowed after token replenishment")
		}
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	handler := RateLimitMiddleware(2, time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("allows requests within limit", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Request %d: status = %v, want %v", i+1, w.Code, http.StatusOK)
			}
		}
	})

	t.Run("returns 429 when rate limited", func(t *testing.T) {
		// Create a fresh handler for this test
		testHandler := RateLimitMiddleware(1, time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// First request should succeed
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "10.0.0.1:12345"
		w1 := httptest.NewRecorder()
		testHandler.ServeHTTP(w1, req1)

		// Second request should be rate limited
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "10.0.0.1:12345"
		w2 := httptest.NewRecorder()
		testHandler.ServeHTTP(w2, req2)

		if w2.Code != http.StatusTooManyRequests {
			t.Errorf("status = %v, want %v", w2.Code, http.StatusTooManyRequests)
		}

		if w2.Header().Get("Retry-After") == "" {
			t.Error("Retry-After header is missing")
		}
	})
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "from RemoteAddr",
			remoteAddr: "192.168.1.1:12345",
			headers:    nil,
			expected:   "192.168.1.1",
		},
		{
			name:       "from X-Forwarded-For",
			remoteAddr: "127.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2"},
			expected:   "10.0.0.1",
		},
		{
			name:       "from X-Real-IP",
			remoteAddr: "127.0.0.1:12345",
			headers:    map[string]string{"X-Real-IP": "172.16.0.1"},
			expected:   "172.16.0.1",
		},
		{
			name:       "X-Forwarded-For takes precedence",
			remoteAddr: "127.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1", "X-Real-IP": "172.16.0.1"},
			expected:   "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := getClientIP(req)
			if result != tt.expected {
				t.Errorf("getClientIP() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLoggingMiddleware(t *testing.T) {
	called := false
	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusCreated {
		t.Errorf("status code = %v, want %v", w.Code, http.StatusCreated)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		handler := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		// Should not panic
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status code = %v, want %v", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("passes through when no panic", func(t *testing.T) {
		handler := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status code = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("statusCode = %v, want %v", rw.statusCode, http.StatusNotFound)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("underlying status code = %v, want %v", w.Code, http.StatusNotFound)
	}
}
