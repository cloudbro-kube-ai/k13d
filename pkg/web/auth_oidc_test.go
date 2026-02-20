package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockOIDCServer creates a mock OIDC provider server for testing
func mockOIDCServer() *httptest.Server {
	mux := http.NewServeMux()

	// Discovery endpoint
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		discovery := map[string]interface{}{
			"issuer":                 "https://mock-oidc.example.com",
			"authorization_endpoint": "https://mock-oidc.example.com/authorize",
			"token_endpoint":         "https://mock-oidc.example.com/token",
			"userinfo_endpoint":      "https://mock-oidc.example.com/userinfo",
			"jwks_uri":               "https://mock-oidc.example.com/.well-known/jwks.json",
			"scopes_supported":       []string{"openid", "email", "profile"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(discovery)
	})

	// Token endpoint
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		tokenResp := OIDCTokenResponse{
			AccessToken: "mock-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			IDToken:     "mock-id-token",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResp)
	})

	// Userinfo endpoint
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer mock-access-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userInfo := OIDCUserInfo{
			Sub:           "user-123",
			Name:          "Test User",
			Email:         "test@example.com",
			EmailVerified: true,
			Groups:        []string{"developers", "k8s-admins"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(userInfo)
	})

	return httptest.NewServer(mux)
}

func TestNewOIDCProvider(t *testing.T) {
	// Start mock server
	mockServer := mockOIDCServer()
	defer mockServer.Close()

	tests := []struct {
		name    string
		config  *OIDCConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "missing provider URL",
			config: &OIDCConfig{
				ClientID: "test-client",
			},
			wantErr: true,
		},
		{
			name: "missing client ID",
			config: &OIDCConfig{
				ProviderURL: mockServer.URL,
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &OIDCConfig{
				ProviderName: "Test Provider",
				ProviderURL:  mockServer.URL,
				ClientID:     "test-client",
				ClientSecret: "test-secret",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOIDCProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOIDCProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewOIDCProvider() returned nil provider")
			}
		})
	}
}

func TestOIDCProviderDiscovery(t *testing.T) {
	mockServer := mockOIDCServer()
	defer mockServer.Close()

	config := &OIDCConfig{
		ProviderName: "Test Provider",
		ProviderURL:  mockServer.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	}

	provider, err := NewOIDCProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.discovery == nil {
		t.Fatal("Discovery document not fetched")
	}

	if provider.discovery.Issuer != "https://mock-oidc.example.com" {
		t.Errorf("Unexpected issuer: %s", provider.discovery.Issuer)
	}
}

func TestOIDCProviderGetAuthorizationURL(t *testing.T) {
	mockServer := mockOIDCServer()
	defer mockServer.Close()

	config := &OIDCConfig{
		ProviderName: "Test Provider",
		ProviderURL:  mockServer.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Scopes:       "openid email profile",
	}

	provider, err := NewOIDCProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	redirectURI := "http://localhost:8080/api/auth/oidc/callback"
	authURL, state, err := provider.GetAuthorizationURL(redirectURI)
	if err != nil {
		t.Fatalf("GetAuthorizationURL failed: %v", err)
	}

	if authURL == "" {
		t.Error("Authorization URL is empty")
	}

	if state == "" {
		t.Error("State is empty")
	}

	// Verify state is stored
	provider.mu.RLock()
	_, exists := provider.states[state]
	provider.mu.RUnlock()
	if !exists {
		t.Error("State not stored in provider")
	}
}

func TestOIDCProviderValidateState(t *testing.T) {
	mockServer := mockOIDCServer()
	defer mockServer.Close()

	config := &OIDCConfig{
		ProviderName: "Test Provider",
		ProviderURL:  mockServer.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	}

	provider, err := NewOIDCProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Generate a state
	_, state, err := provider.GetAuthorizationURL("http://localhost/callback")
	if err != nil {
		t.Fatalf("GetAuthorizationURL failed: %v", err)
	}

	// First validation should succeed
	if !provider.ValidateState(state) {
		t.Error("First state validation should succeed")
	}

	// Second validation should fail (state is single-use)
	if provider.ValidateState(state) {
		t.Error("Second state validation should fail (single use)")
	}

	// Invalid state should fail
	if provider.ValidateState("invalid-state") {
		t.Error("Invalid state validation should fail")
	}

	// Expired state should fail
	provider.mu.Lock()
	provider.states["expired-state"] = time.Now().Add(-1 * time.Hour)
	provider.mu.Unlock()
	if provider.ValidateState("expired-state") {
		t.Error("Expired state validation should fail")
	}
}

func TestOIDCProviderDetermineRole(t *testing.T) {
	config := &OIDCConfig{
		ProviderName: "Test Provider",
		ProviderURL:  "https://example.com",
		ClientID:     "test-client",
		AdminRoles:   []string{"k8s-admins", "cluster-admins"},
		UserRoles:    []string{"developers", "users"},
		DefaultRole:  "viewer",
		GroupMappings: map[string]string{
			"ops-team": "admin",
			"qa-team":  "user",
		},
	}

	provider := &OIDCProvider{config: config}

	tests := []struct {
		name     string
		userInfo *OIDCUserInfo
		wantRole string
	}{
		{
			name: "admin via groups",
			userInfo: &OIDCUserInfo{
				Groups: []string{"k8s-admins"},
			},
			wantRole: "admin",
		},
		{
			name: "user via groups",
			userInfo: &OIDCUserInfo{
				Groups: []string{"developers"},
			},
			wantRole: "user",
		},
		{
			name: "admin via group mapping",
			userInfo: &OIDCUserInfo{
				Groups: []string{"ops-team"},
			},
			wantRole: "admin",
		},
		{
			name: "user via group mapping",
			userInfo: &OIDCUserInfo{
				Groups: []string{"qa-team"},
			},
			wantRole: "user",
		},
		{
			name: "admin via roles claim",
			userInfo: &OIDCUserInfo{
				Roles: []string{"cluster-admins"},
			},
			wantRole: "admin",
		},
		{
			name: "default role when no match",
			userInfo: &OIDCUserInfo{
				Groups: []string{"random-group"},
			},
			wantRole: "viewer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := provider.DetermineRole(tt.userInfo)
			if role != tt.wantRole {
				t.Errorf("DetermineRole() = %s, want %s", role, tt.wantRole)
			}
		})
	}
}

func TestOIDCHandleLogin(t *testing.T) {
	mockServer := mockOIDCServer()
	defer mockServer.Close()

	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "oidc",
		Quiet:           true,
		OIDC: &OIDCConfig{
			ProviderName: "Test Provider",
			ProviderURL:  mockServer.URL,
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		},
	}

	authManager := NewAuthManager(authConfig)

	// Test OIDC login redirect
	req := httptest.NewRequest("GET", "/api/auth/oidc/login", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()

	authManager.HandleOIDCLogin(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected redirect status 302, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Expected Location header for redirect")
	}
}

func TestOIDCHandleStatus(t *testing.T) {
	mockServer := mockOIDCServer()
	defer mockServer.Close()

	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "oidc",
		Quiet:           true,
		OIDC: &OIDCConfig{
			ProviderName: "Test Provider",
			ProviderURL:  mockServer.URL,
			ClientID:     "test-client",
			ClientSecret: "test-secret",
		},
	}

	authManager := NewAuthManager(authConfig)

	req := httptest.NewRequest("GET", "/api/auth/oidc/status", nil)
	w := httptest.NewRecorder()

	authManager.HandleOIDCStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var status map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if status["configured"] != true {
		t.Error("Expected OIDC to be configured")
	}

	if status["provider_name"] != "Test Provider" {
		t.Errorf("Unexpected provider name: %v", status["provider_name"])
	}
}

func TestOIDCNotConfigured(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		Quiet:           true,
	}

	authManager := NewAuthManager(authConfig)

	// Test OIDC login when not configured
	req := httptest.NewRequest("GET", "/api/auth/oidc/login", nil)
	w := httptest.NewRecorder()

	authManager.HandleOIDCLogin(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 when OIDC not configured, got %d", w.Code)
	}
}

func TestGenerateRandomString(t *testing.T) {
	// Test that it generates different strings
	s1, err := generateRandomString(32)
	if err != nil {
		t.Fatalf("generateRandomString failed: %v", err)
	}

	s2, err := generateRandomString(32)
	if err != nil {
		t.Fatalf("generateRandomString failed: %v", err)
	}

	if s1 == s2 {
		t.Error("generateRandomString should generate different strings")
	}

	if len(s1) != 32 {
		t.Errorf("Expected string length 32, got %d", len(s1))
	}
}
