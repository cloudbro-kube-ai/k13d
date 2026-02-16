package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Second)

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !rl.Allow("test-client") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked
	if rl.Allow("test-client") {
		t.Error("4th request should be blocked")
	}

	// Wait for window to reset
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again after reset
	if !rl.Allow("test-client") {
		t.Error("Request after reset should be allowed")
	}
}

func TestRateLimiter_MultipleClients(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Second)

	// Client A gets 2 requests
	if !rl.Allow("client-a") {
		t.Error("Client A request 1 should be allowed")
	}
	if !rl.Allow("client-a") {
		t.Error("Client A request 2 should be allowed")
	}
	if rl.Allow("client-a") {
		t.Error("Client A request 3 should be blocked")
	}

	// Client B should still have full quota
	if !rl.Allow("client-b") {
		t.Error("Client B request 1 should be allowed")
	}
	if !rl.Allow("client-b") {
		t.Error("Client B request 2 should be allowed")
	}
	if rl.Allow("client-b") {
		t.Error("Client B request 3 should be blocked")
	}
}

func TestRateLimiter_GetRetryAfter(t *testing.T) {
	rl := NewRateLimiter(1, 1*time.Second)

	// Exhaust limit
	rl.Allow("test-client")

	// Check retry after
	retryAfter := rl.GetRetryAfter("test-client")
	if retryAfter <= 0 || retryAfter > 1*time.Second {
		t.Errorf("Retry after should be between 0 and 1 second, got %v", retryAfter)
	}
}

func TestRateLimitMiddleware_APIEndpoint(t *testing.T) {
	apiLimiter := NewRateLimiter(2, 1*time.Minute)
	authLimiter := NewRateLimiter(10, 1*time.Minute)

	handler := RateLimitMiddleware(apiLimiter, authLimiter)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}),
	)

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/k8s/pods", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d should succeed, got status %d", i+1, rr.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/api/k8s/pods", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request should be rate limited, got status %d", rr.Code)
	}

	// Check rate limit headers
	if rr.Header().Get("Retry-After") == "" {
		t.Error("Retry-After header should be set")
	}
	if rr.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("X-RateLimit-Limit header should be set")
	}
}

func TestRateLimitMiddleware_AuthEndpoint(t *testing.T) {
	apiLimiter := NewRateLimiter(100, 1*time.Minute)
	authLimiter := NewRateLimiter(2, 1*time.Minute)

	handler := RateLimitMiddleware(apiLimiter, authLimiter)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}),
	)

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/auth/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d should succeed, got status %d", i+1, rr.Code)
		}
	}

	// 3rd request should be rate limited (auth limit is 2)
	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request should be rate limited, got status %d", rr.Code)
	}
}

func TestRateLimitMiddleware_StaticFiles(t *testing.T) {
	apiLimiter := NewRateLimiter(1, 1*time.Minute)
	authLimiter := NewRateLimiter(1, 1*time.Minute)

	handler := RateLimitMiddleware(apiLimiter, authLimiter)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("static content"))
		}),
	)

	// Static files should not be rate limited
	// Make multiple requests to ensure no rate limiting
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/static/app.js", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Static file request %d should succeed, got status %d", i+1, rr.Code)
		}
	}
}

func TestRateLimitMiddleware_DifferentIPs(t *testing.T) {
	apiLimiter := NewRateLimiter(1, 1*time.Minute)
	authLimiter := NewRateLimiter(10, 1*time.Minute)

	handler := RateLimitMiddleware(apiLimiter, authLimiter)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	// Request from IP 1
	req1 := httptest.NewRequest("GET", "/api/k8s/pods", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Error("Request from IP1 should succeed")
	}

	// Second request from IP 1 should be rate limited
	req1b := httptest.NewRequest("GET", "/api/k8s/pods", nil)
	req1b.RemoteAddr = "192.168.1.1:12345"
	rr1b := httptest.NewRecorder()
	handler.ServeHTTP(rr1b, req1b)

	if rr1b.Code != http.StatusTooManyRequests {
		t.Error("Second request from IP1 should be rate limited")
	}

	// Request from IP 2 should succeed (different IP)
	req2 := httptest.NewRequest("GET", "/api/k8s/pods", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Error("Request from IP2 should succeed")
	}
}

func TestIsAuthEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/auth/login", true},
		{"/api/auth/logout", true},
		{"/api/auth/kubeconfig", true},
		{"/api/auth/oidc/login", true},
		{"/api/auth/oidc/callback", true},
		{"/api/k8s/pods", false},
		{"/api/chat/agentic", false},
		{"/static/app.js", false},
		{"/", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isAuthEndpoint(tt.path)
			if result != tt.expected {
				t.Errorf("isAuthEndpoint(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsAPIEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/k8s/pods", true},
		{"/api/chat/agentic", true},
		{"/api/health", true},
		{"/api/auth/login", false}, // Auth endpoints are excluded
		{"/api/auth/logout", false},
		{"/static/app.js", false},
		{"/", false},
		{"/api", false}, // Too short
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isAPIEndpoint(tt.path)
			if result != tt.expected {
				t.Errorf("isAPIEndpoint(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetClientIdentifier(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		username       string
		expectedPrefix string
	}{
		{
			name:           "Remote addr only",
			remoteAddr:     "192.168.1.1:12345",
			expectedPrefix: "192.168.1.1:12345",
		},
		{
			name:           "X-Forwarded-For",
			remoteAddr:     "10.0.0.1:12345",
			xForwardedFor:  "203.0.113.1",
			expectedPrefix: "203.0.113.1",
		},
		{
			name:           "X-Real-IP",
			remoteAddr:     "10.0.0.1:12345",
			xRealIP:        "203.0.113.2",
			expectedPrefix: "203.0.113.2",
		},
		{
			name:           "With username",
			remoteAddr:     "192.168.1.1:12345",
			username:       "admin",
			expectedPrefix: "192.168.1.1:12345:admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			if tt.username != "" {
				req.Header.Set("X-Username", tt.username)
			}

			identifier := getClientIdentifier(req)
			if identifier != tt.expectedPrefix {
				t.Errorf("getClientIdentifier() = %s, want %s", identifier, tt.expectedPrefix)
			}
		})
	}
}
