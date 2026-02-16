package web

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int           // requests per window
	window   time.Duration // time window
	cleanup  time.Duration // cleanup interval
	done     chan struct{}  // signals cleanup goroutine to stop
}

// visitor tracks rate limit state for a single IP/user
type visitor struct {
	tokens    int
	lastReset time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// limit: maximum requests per window
// window: time window for rate limiting (e.g., 1 minute)
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
		cleanup:  5 * time.Minute, // Clean up stale entries every 5 minutes
		done:     make(chan struct{}),
	}

	// Start background cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Stop signals the cleanup goroutine to exit
func (rl *RateLimiter) Stop() {
	close(rl.done)
}

// Allow checks if a request from the given identifier should be allowed
func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.Lock()
	v, exists := rl.visitors[identifier]
	if !exists {
		v = &visitor{
			tokens:    rl.limit,
			lastReset: time.Now(),
		}
		rl.visitors[identifier] = v
	}
	rl.mu.Unlock()

	v.mu.Lock()
	defer v.mu.Unlock()

	// Reset tokens if window has passed
	now := time.Now()
	if now.Sub(v.lastReset) > rl.window {
		v.tokens = rl.limit
		v.lastReset = now
	}

	// Check if request is allowed
	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

// GetRetryAfter returns the time until the rate limit resets
func (rl *RateLimiter) GetRetryAfter(identifier string) time.Duration {
	rl.mu.Lock()
	v, exists := rl.visitors[identifier]
	rl.mu.Unlock()

	if !exists {
		return 0
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	elapsed := time.Since(v.lastReset)
	remaining := rl.window - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// cleanupLoop periodically removes stale visitor entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for id, v := range rl.visitors {
				v.mu.Lock()
				if now.Sub(v.lastReset) > rl.window*2 {
					delete(rl.visitors, id)
				}
				v.mu.Unlock()
			}
			rl.mu.Unlock()
		case <-rl.done:
			return
		}
	}
}

// RateLimitMiddleware creates a rate limiting middleware
// Different limits for different endpoint categories
func RateLimitMiddleware(apiLimiter, authLimiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client identifier (IP + username if authenticated)
			identifier := getClientIdentifier(r)

			// Select appropriate rate limiter based on endpoint
			var limiter *RateLimiter
			var limitType string

			if isAuthEndpoint(r.URL.Path) {
				limiter = authLimiter
				limitType = "auth"
			} else if isAPIEndpoint(r.URL.Path) {
				limiter = apiLimiter
				limitType = "api"
			} else {
				// No rate limiting for static files
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			if !limiter.Allow(identifier) {
				retryAfter := limiter.GetRetryAfter(identifier)
				w.Header().Set("Retry-After", formatRetryAfter(retryAfter))
				w.Header().Set("X-RateLimit-Limit", formatIntValue(limiter.limit))
				w.Header().Set("X-RateLimit-Reset", formatTimestamp(time.Now().Add(retryAfter)))

				WriteError(w, NewAPIErrorWithSuggestion(
					ErrCodeRateLimited,
					"Too many requests",
					"You have exceeded the rate limit for "+limitType+" endpoints. Please wait before trying again.",
				))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIdentifier returns a unique identifier for rate limiting
// Uses IP address + username if authenticated
func getClientIdentifier(r *http.Request) string {
	// Get IP from X-Forwarded-For or RemoteAddr
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}

	// Add username if authenticated
	username := r.Header.Get("X-Username")
	if username != "" {
		return ip + ":" + username
	}

	return ip
}

// isAuthEndpoint checks if the path is an authentication endpoint
func isAuthEndpoint(path string) bool {
	authPaths := []string{
		"/api/auth/login",
		"/api/auth/logout",
		"/api/auth/kubeconfig",
		"/api/auth/oidc/login",
		"/api/auth/oidc/callback",
	}

	for _, ap := range authPaths {
		if path == ap {
			return true
		}
	}
	return false
}

// isAPIEndpoint checks if the path is an API endpoint
func isAPIEndpoint(path string) bool {
	// All /api/ paths except auth are API endpoints
	return len(path) > 5 && path[:5] == "/api/" && !isAuthEndpoint(path)
}

// formatRetryAfter formats a duration in seconds for Retry-After header
func formatRetryAfter(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return formatIntValue(seconds)
}

// formatIntValue converts an int to string using stdlib
func formatIntValue(n int) string {
	return fmt.Sprintf("%d", n)
}

// formatTimestamp formats a time as Unix timestamp string
func formatTimestamp(t time.Time) string {
	return fmt.Sprintf("%d", t.Unix())
}
