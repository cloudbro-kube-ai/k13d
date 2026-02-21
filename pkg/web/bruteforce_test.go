package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBruteForceProtector_RecordFailure_ProgressiveDelay(t *testing.T) {
	bp := NewBruteForceProtector()
	defer bp.Stop()

	ip := "192.168.1.100"

	// 1st failure: no delay
	delay := bp.RecordFailure(ip)
	if delay != 0 {
		t.Errorf("1st failure: expected 0 delay, got %v", delay)
	}

	// 2nd failure: 1s delay
	delay = bp.RecordFailure(ip)
	if delay != 1*time.Second {
		t.Errorf("2nd failure: expected 1s delay, got %v", delay)
	}

	// 3rd failure: 3s delay
	delay = bp.RecordFailure(ip)
	if delay != 3*time.Second {
		t.Errorf("3rd failure: expected 3s delay, got %v", delay)
	}

	// 4th failure: 5s delay
	delay = bp.RecordFailure(ip)
	if delay != 5*time.Second {
		t.Errorf("4th failure: expected 5s delay, got %v", delay)
	}

	// 5th failure: blocked (delay=0 means blocked response)
	delay = bp.RecordFailure(ip)
	if delay != 0 {
		t.Errorf("5th failure: expected 0 (blocked), got %v", delay)
	}

	if !bp.IsBlocked(ip) {
		t.Error("IP should be blocked after 5 failures")
	}
}

func TestBruteForceProtector_IsBlocked(t *testing.T) {
	bp := NewBruteForceProtector()
	defer bp.Stop()

	ip := "10.0.0.1"

	if bp.IsBlocked(ip) {
		t.Error("fresh IP should not be blocked")
	}

	// Fail 5 times
	for i := 0; i < 5; i++ {
		bp.RecordFailure(ip)
	}

	if !bp.IsBlocked(ip) {
		t.Error("IP should be blocked after 5 failures")
	}

	// Other IPs should not be affected
	if bp.IsBlocked("10.0.0.2") {
		t.Error("different IP should not be blocked")
	}
}

func TestBruteForceProtector_BlockExpiry(t *testing.T) {
	bp := NewBruteForceProtector()
	defer bp.Stop()

	// Use a very short block duration for testing
	bp.BlockDuration = 50 * time.Millisecond

	ip := "10.0.0.1"

	// Fail enough to trigger block
	for i := 0; i < 5; i++ {
		bp.RecordFailure(ip)
	}

	if !bp.IsBlocked(ip) {
		t.Error("IP should be blocked immediately")
	}

	// Wait for block to expire
	time.Sleep(100 * time.Millisecond)

	if bp.IsBlocked(ip) {
		t.Error("IP block should have expired")
	}
}

func TestBruteForceProtector_RecordSuccess_ResetsCounter(t *testing.T) {
	bp := NewBruteForceProtector()
	defer bp.Stop()

	ip := "172.16.0.1"

	// Fail 3 times
	for i := 0; i < 3; i++ {
		bp.RecordFailure(ip)
	}

	if bp.FailureCount(ip) != 3 {
		t.Errorf("expected 3 failures, got %d", bp.FailureCount(ip))
	}

	// Successful login resets counter
	bp.RecordSuccess(ip)

	if bp.FailureCount(ip) != 0 {
		t.Errorf("expected 0 failures after success, got %d", bp.FailureCount(ip))
	}

	if bp.IsBlocked(ip) {
		t.Error("IP should not be blocked after successful login")
	}
}

func TestBruteForceProtector_FailureCount(t *testing.T) {
	bp := NewBruteForceProtector()
	defer bp.Stop()

	ip := "192.168.0.10"

	if bp.FailureCount(ip) != 0 {
		t.Error("unknown IP should have 0 failures")
	}

	bp.RecordFailure(ip)
	bp.RecordFailure(ip)

	if bp.FailureCount(ip) != 2 {
		t.Errorf("expected 2 failures, got %d", bp.FailureCount(ip))
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xRealIP    string
		remoteAddr string
		want       string
	}{
		{
			name:       "X-Forwarded-For with single IP",
			xff:        "203.0.113.50",
			remoteAddr: "127.0.0.1:1234",
			want:       "203.0.113.50",
		},
		{
			name:       "X-Forwarded-For with multiple IPs",
			xff:        "203.0.113.50, 70.41.3.18, 150.172.238.178",
			remoteAddr: "127.0.0.1:1234",
			want:       "203.0.113.50",
		},
		{
			name:       "X-Real-IP",
			xRealIP:    "198.51.100.42",
			remoteAddr: "127.0.0.1:1234",
			want:       "198.51.100.42",
		},
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.1.1:54321",
			want:       "192.168.1.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			want:       "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For takes priority over X-Real-IP",
			xff:        "10.0.0.1",
			xRealIP:    "10.0.0.2",
			remoteAddr: "127.0.0.1:1234",
			want:       "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xRealIP != "" {
				r.Header.Set("X-Real-IP", tt.xRealIP)
			}

			got := ClientIP(r)
			if got != tt.want {
				t.Errorf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

// disableDelays sets all progressive delays to zero for fast testing.
func disableDelays(am *AuthManager) {
	am.bruteForce.Delays = []time.Duration{0, 0, 0, 0, 0}
}

func TestHandleLogin_BruteForceBlocking(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "correctpassword",
	})
	disableDelays(am)

	// Simulate 5 failed logins from the same IP
	for i := 0; i < 5; i++ {
		body, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "wrongpassword",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.0.0.99:12345"
		w := httptest.NewRecorder()

		am.HandleLogin(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("attempt %d: expected 401, got %d", i+1, w.Code)
		}
	}

	// 6th attempt should be blocked (429)
	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "wrongpassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.99:12345"
	w := httptest.NewRecorder()

	am.HandleLogin(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("blocked attempt: expected 429, got %d", w.Code)
	}

	// Even a correct password from blocked IP should be rejected
	body, _ = json.Marshal(map[string]string{
		"username": "admin",
		"password": "correctpassword",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.99:12345"
	w = httptest.NewRecorder()

	am.HandleLogin(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("blocked IP with correct pass: expected 429, got %d", w.Code)
	}
}

func TestHandleLogin_DifferentIPsIndependent(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "correctpassword",
	})
	disableDelays(am)

	// Fail from IP1
	for i := 0; i < 4; i++ {
		body, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "wrong",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		am.HandleLogin(w, req)
	}

	// IP2 should still be able to login
	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "correctpassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.2:12345"
	w := httptest.NewRecorder()

	am.HandleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("different IP should succeed: expected 200, got %d", w.Code)
	}
}

func TestHandleLogin_SuccessResetsCounter(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "correctpassword",
	})
	disableDelays(am)

	clientIP := "10.0.0.50"

	// Fail 3 times
	for i := 0; i < 3; i++ {
		body, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "wrong",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = clientIP + ":12345"
		w := httptest.NewRecorder()
		am.HandleLogin(w, req)
	}

	if am.bruteForce.FailureCount(clientIP) != 3 {
		t.Errorf("expected 3 failures, got %d", am.bruteForce.FailureCount(clientIP))
	}

	// Successful login
	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "correctpassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = clientIP + ":12345"
	w := httptest.NewRecorder()

	am.HandleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("correct login: expected 200, got %d", w.Code)
	}

	// Counter should be reset
	if am.bruteForce.FailureCount(clientIP) != 0 {
		t.Errorf("expected 0 failures after success, got %d", am.bruteForce.FailureCount(clientIP))
	}
}

func TestHandleLogin_XForwardedFor(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "correctpassword",
	})
	disableDelays(am)

	realIP := "203.0.113.50"

	// Fail from a real IP behind nginx
	for i := 0; i < 5; i++ {
		body, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "wrong",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", realIP+", 10.0.0.1")
		req.RemoteAddr = "10.0.0.1:12345" // nginx proxy IP
		w := httptest.NewRecorder()
		am.HandleLogin(w, req)
	}

	// The real client IP should be blocked, not the proxy
	if !am.bruteForce.IsBlocked(realIP) {
		t.Error("real client IP behind proxy should be blocked")
	}

	// Proxy IP should not be blocked
	if am.bruteForce.IsBlocked("10.0.0.1") {
		t.Error("proxy IP should not be blocked")
	}

	// Next attempt with same X-Forwarded-For should be blocked (429)
	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "correctpassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", realIP)
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()

	am.HandleLogin(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("blocked XFF IP: expected 429, got %d", w.Code)
	}
}

// --- Account Lockout Tests ---

func TestAccountLockout_LockAfterMaxFailures(t *testing.T) {
	al := NewAccountLockout()
	defer al.Stop()

	username := "testuser"

	for i := 0; i < 9; i++ {
		locked := al.RecordFailure(username)
		if locked {
			t.Errorf("should not lock on failure %d", i+1)
		}
	}

	if al.IsLocked(username) {
		t.Error("should not be locked before 10th failure")
	}

	// 10th failure triggers lock
	locked := al.RecordFailure(username)
	if !locked {
		t.Error("10th failure should trigger lock")
	}

	if !al.IsLocked(username) {
		t.Error("account should be locked after 10 failures")
	}
}

func TestAccountLockout_LockExpiry(t *testing.T) {
	al := NewAccountLockout()
	defer al.Stop()

	al.LockDuration = 50 * time.Millisecond

	username := "expiryuser"
	for i := 0; i < 10; i++ {
		al.RecordFailure(username)
	}

	if !al.IsLocked(username) {
		t.Error("should be locked immediately")
	}

	time.Sleep(100 * time.Millisecond)

	if al.IsLocked(username) {
		t.Error("lock should have expired")
	}
}

func TestAccountLockout_SuccessResets(t *testing.T) {
	al := NewAccountLockout()
	defer al.Stop()

	username := "resetuser"
	for i := 0; i < 5; i++ {
		al.RecordFailure(username)
	}

	if al.FailureCount(username) != 5 {
		t.Errorf("expected 5 failures, got %d", al.FailureCount(username))
	}

	al.RecordSuccess(username)

	if al.FailureCount(username) != 0 {
		t.Errorf("expected 0 after success, got %d", al.FailureCount(username))
	}
}

func TestAccountLockout_IndependentAccounts(t *testing.T) {
	al := NewAccountLockout()
	defer al.Stop()

	for i := 0; i < 10; i++ {
		al.RecordFailure("user1")
	}

	if !al.IsLocked("user1") {
		t.Error("user1 should be locked")
	}

	if al.IsLocked("user2") {
		t.Error("user2 should not be affected by user1's lockout")
	}
}

func TestHandleLogin_AccountLockout(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "correctpassword",
	})
	disableDelays(am)

	// Simulate 10 failed logins for the same account from different IPs
	for i := 0; i < 10; i++ {
		body, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "wrongpassword",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = fmt.Sprintf("10.0.%d.1:12345", i) // different IPs
		w := httptest.NewRecorder()

		am.HandleLogin(w, req)
	}

	// 11th attempt should be blocked by account lockout (429)
	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "correctpassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.99.99.99:12345" // fresh IP — proves it's account-level
	w := httptest.NewRecorder()

	am.HandleLogin(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("locked account: expected 429, got %d", w.Code)
	}

	// Verify the response message
	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp["error"], "계정이 잠겼습니다") {
		t.Errorf("expected Korean lockout message, got: %s", resp["error"])
	}
}

func TestHandleLogin_AccountLockout_IndependentOfIPBlock(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "correctpassword",
	})
	disableDelays(am)

	// Create a second user
	_ = am.CreateUser("user2", "password2!", "user")

	// Lock the "admin" account via failures from various IPs
	for i := 0; i < 10; i++ {
		body, _ := json.Marshal(map[string]string{
			"username": "admin",
			"password": "wrong",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = fmt.Sprintf("10.0.%d.1:12345", i)
		w := httptest.NewRecorder()
		am.HandleLogin(w, req)
	}

	// user2 should still be able to login even though admin is locked
	body, _ := json.Marshal(map[string]string{
		"username": "user2",
		"password": "password2!",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.99.0.1:12345"
	w := httptest.NewRecorder()

	am.HandleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("user2 should succeed while admin is locked: expected 200, got %d", w.Code)
	}
}
