package web

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// BruteForceProtector tracks failed login attempts per IP and applies
// progressive delays and temporary bans.
type BruteForceProtector struct {
	mu sync.Mutex

	// failedAttempts tracks consecutive failures per IP.
	failedAttempts map[string]*loginAttempt

	// Configuration (exported for testing).
	MaxFailures   int             // consecutive failures before blocking (default: 5)
	BlockDuration time.Duration   // how long an IP stays blocked (default: 15min)
	Delays        []time.Duration // progressive delay per failure count (index = failure #)

	// done signals the cleanup goroutine to stop.
	done chan struct{}
}

// loginAttempt records failure state for a single IP.
type loginAttempt struct {
	Count     int
	LastFail  time.Time
	BlockedAt time.Time // zero value means not blocked
}

// defaultDelays maps failure count → sleep duration.
// Index 0 is unused; index 1 = first failure, etc.
var defaultDelays = []time.Duration{
	0,               // 0 failures (unused)
	0,               // 1st failure: no delay
	1 * time.Second, // 2nd failure
	3 * time.Second, // 3rd failure
	5 * time.Second, // 4th failure
}

// NewBruteForceProtector creates a protector with sensible defaults.
func NewBruteForceProtector() *BruteForceProtector {
	bp := &BruteForceProtector{
		failedAttempts: make(map[string]*loginAttempt),
		MaxFailures:    5,
		BlockDuration:  15 * time.Minute,
		Delays:         defaultDelays,
		done:           make(chan struct{}),
	}
	go bp.cleanupLoop()
	return bp
}

// Stop signals the cleanup goroutine to exit.
func (bp *BruteForceProtector) Stop() {
	close(bp.done)
}

// IsBlocked returns true if the IP is currently blocked.
func (bp *BruteForceProtector) IsBlocked(ip string) bool {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	attempt, exists := bp.failedAttempts[ip]
	if !exists {
		return false
	}

	if !attempt.BlockedAt.IsZero() {
		if time.Since(attempt.BlockedAt) < bp.BlockDuration {
			return true
		}
		// Block expired — clear record.
		delete(bp.failedAttempts, ip)
		return false
	}

	return false
}

// RecordFailure increments the failure counter for an IP.
// Returns the delay the caller should apply before responding.
func (bp *BruteForceProtector) RecordFailure(ip string) time.Duration {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	attempt, exists := bp.failedAttempts[ip]
	if !exists {
		attempt = &loginAttempt{}
		bp.failedAttempts[ip] = attempt
	}

	attempt.Count++
	attempt.LastFail = time.Now()

	if attempt.Count >= bp.MaxFailures {
		attempt.BlockedAt = time.Now()
		fmt.Printf("[SECURITY] IP %s blocked for %v after %d consecutive failed login attempts\n",
			ip, bp.BlockDuration, attempt.Count)
		return 0 // no delay needed — caller should return blocked response
	}

	// Progressive delay
	idx := attempt.Count
	if idx >= len(bp.Delays) {
		idx = len(bp.Delays) - 1
	}
	return bp.Delays[idx]
}

// RecordSuccess resets the failure counter for an IP on successful login.
func (bp *BruteForceProtector) RecordSuccess(ip string) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	delete(bp.failedAttempts, ip)
}

// FailureCount returns the current consecutive failure count for an IP (for testing/monitoring).
func (bp *BruteForceProtector) FailureCount(ip string) int {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	if attempt, exists := bp.failedAttempts[ip]; exists {
		return attempt.Count
	}
	return 0
}

// cleanupLoop periodically removes expired block entries.
func (bp *BruteForceProtector) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bp.mu.Lock()
			now := time.Now()
			for ip, attempt := range bp.failedAttempts {
				// Remove expired blocks
				if !attempt.BlockedAt.IsZero() && now.Sub(attempt.BlockedAt) > bp.BlockDuration {
					delete(bp.failedAttempts, ip)
					continue
				}
				// Remove stale entries (no activity for 2× block duration)
				if now.Sub(attempt.LastFail) > bp.BlockDuration*2 {
					delete(bp.failedAttempts, ip)
				}
			}
			bp.mu.Unlock()
		case <-bp.done:
			return
		}
	}
}

// ClientIP extracts the real client IP from a request, respecting
// X-Forwarded-For and X-Real-IP headers (for reverse proxy setups like nginx).
func ClientIP(r *http.Request) string {
	// X-Forwarded-For may contain multiple IPs: "client, proxy1, proxy2"
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// The first IP is the original client.
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to RemoteAddr (strip port).
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
