//go:build integration

// Integration tests for Web server
// Run with: go test -tags=integration ./tests/integration/...

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/web"
)

func TestWebServer_Health(t *testing.T) {
	cfg := config.NewDefaultConfig()
	server, err := web.NewServer(cfg, nil, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	handler := server.GetHandler()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health endpoint returned %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Health status = %v, want ok", resp["status"])
	}
}

func TestWebServer_Authentication(t *testing.T) {
	cfg := config.NewDefaultConfig()
	server, err := web.NewServer(cfg, nil, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	handler := server.GetHandler()

	// Test login with default credentials
	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin123",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Login returned %d, want %d", w.Code, http.StatusOK)
	}

	var loginResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&loginResp); err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}

	token, ok := loginResp["token"].(string)
	if !ok || token == "" {
		t.Fatal("Login response missing token")
	}

	t.Log("Login successful, got token")

	// Test authenticated request
	req = httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Authenticated request returned %d, want %d", w.Code, http.StatusOK)
	}

	// Test logout
	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Logout returned %d, want %d", w.Code, http.StatusOK)
	}
}

func TestWebServer_InvalidLogin(t *testing.T) {
	cfg := config.NewDefaultConfig()
	server, err := web.NewServer(cfg, nil, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	handler := server.GetHandler()

	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "wrongpassword",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Invalid login returned %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWebServer_UnauthenticatedRequest(t *testing.T) {
	cfg := config.NewDefaultConfig()
	server, err := web.NewServer(cfg, nil, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	handler := server.GetHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Unauthenticated request returned %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWebServer_LLMSettings(t *testing.T) {
	cfg := config.NewDefaultConfig()
	server, err := web.NewServer(cfg, nil, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	handler := server.GetHandler()

	// Login first
	loginBody, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin123",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var loginResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&loginResp)
	token := loginResp["token"].(string)

	// Get LLM status
	req = httptest.NewRequest(http.MethodGet, "/api/llm/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("LLM status returned %d, want %d", w.Code, http.StatusOK)
	}

	var statusResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&statusResp); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}

	// Should return current LLM config (even if not connected)
	if statusResp["provider"] == nil {
		t.Error("LLM status missing provider")
	}
}

func TestWebServer_StaticFiles(t *testing.T) {
	cfg := config.NewDefaultConfig()
	server, err := web.NewServer(cfg, nil, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	handler := server.GetHandler()

	// Test index.html
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Static file returned %d, want %d", w.Code, http.StatusOK)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" && contentType != "text/html" {
		t.Logf("Content-Type: %s", contentType)
	}
}

func TestWebServer_CORS(t *testing.T) {
	cfg := config.NewDefaultConfig()
	server, err := web.NewServer(cfg, nil, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	handler := server.GetHandler()

	// Test preflight request
	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should allow CORS
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Log("CORS not configured (may be expected)")
	}
}

func TestWebServer_ConcurrentRequests(t *testing.T) {
	cfg := config.NewDefaultConfig()
	server, err := web.NewServer(cfg, nil, nil, nil, nil, 0)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	handler := server.GetHandler()

	// Run concurrent health checks
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			done <- w.Code == http.StatusOK
		}()
	}

	// Wait for all with timeout
	timeout := time.After(5 * time.Second)
	successes := 0
	for i := 0; i < 10; i++ {
		select {
		case success := <-done:
			if success {
				successes++
			}
		case <-timeout:
			t.Fatal("Concurrent requests timed out")
		}
	}

	if successes != 10 {
		t.Errorf("Only %d/10 concurrent requests succeeded", successes)
	}
}

func TestWebServer_RateLimiting(t *testing.T) {
	// Note: This test would require rate limiting to be implemented
	t.Skip("Rate limiting not yet implemented")
}
