package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAuthManager(t *testing.T) {
	// Test with "local" auth mode (creates default admin user)
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
	})
	defer am.StopCleanup()

	if am == nil {
		t.Fatal("expected AuthManager to be created")
	}

	if len(am.users) != 1 {
		t.Errorf("expected 1 default user, got %d", len(am.users))
	}

	// Check default admin user exists
	user, exists := am.users["admin"]
	if !exists {
		t.Error("expected default admin user to exist")
	}

	if user.Role != "admin" {
		t.Errorf("expected admin role, got %s", user.Role)
	}
}

func TestNewAuthManager_TokenMode(t *testing.T) {
	// Test with "token" auth mode (no default admin user, K8s token auth)
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "token",
	})
	defer am.StopCleanup()

	if am == nil {
		t.Fatal("expected AuthManager to be created")
	}

	// Token mode should not create default users
	if len(am.users) != 0 {
		t.Errorf("expected 0 users in token mode, got %d", len(am.users))
	}

	// AuthMode should be set correctly
	if am.GetAuthMode() != "token" {
		t.Errorf("expected token auth mode, got %s", am.GetAuthMode())
	}
}

func TestNewAuthManager_DefaultMode(t *testing.T) {
	// Test with empty auth mode (should default to "token")
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
	})
	defer am.StopCleanup()

	if am == nil {
		t.Fatal("expected AuthManager to be created")
	}

	// Default mode should be "token"
	if am.GetAuthMode() != "token" {
		t.Errorf("expected default auth mode to be 'token', got %s", am.GetAuthMode())
	}
}

func TestAuthManager_Authenticate(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "testpass1234",
	})
	defer am.StopCleanup()

	tests := []struct {
		name      string
		username  string
		password  string
		wantError bool
	}{
		{"valid admin", "admin", "testpass1234", false},
		{"invalid password", "admin", "wrong", true},
		{"invalid user", "notexist", "testpass1234", true},
		{"empty username", "", "testpass1234", true},
		{"empty password", "admin", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := am.Authenticate(tt.username, tt.password)
			gotError := err != nil
			if gotError != tt.wantError {
				t.Errorf("Authenticate(%s, %s) error = %v, wantError %v", tt.username, tt.password, err, tt.wantError)
			}
			if !tt.wantError && session == nil {
				t.Error("expected session when authentication succeeds")
			}
		})
	}
}

func TestAuthManager_CreateUser(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
	})
	defer am.StopCleanup()

	// Create a new user
	err := am.CreateUser("testuser", "testpass1234", "user")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	// Try to create duplicate user
	err = am.CreateUser("testuser", "testpass1234", "user")
	if err == nil {
		t.Error("expected error when creating duplicate user")
	}

	// Authenticate with new user
	session, err := am.Authenticate("testuser", "testpass1234")
	if err != nil {
		t.Errorf("Authenticate() error = %v", err)
	}

	if session.Username != "testuser" {
		t.Errorf("expected username 'testuser', got %s", session.Username)
	}
}

func TestAuthManager_ValidateSession(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin",
	})
	defer am.StopCleanup()

	// Create a session
	session, _ := am.Authenticate("admin", "admin")

	// Validate session
	validated, err := am.ValidateSession(session.ID)
	if err != nil {
		t.Errorf("ValidateSession() error = %v", err)
	}

	if validated.Username != session.Username {
		t.Error("validated session doesn't match created session")
	}

	// Validate non-existent session
	_, err = am.ValidateSession("invalid-session-id")
	if err == nil {
		t.Error("expected error for invalid session")
	}
}

func TestAuthManager_InvalidateSession(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin",
	})
	defer am.StopCleanup()

	session, _ := am.Authenticate("admin", "admin")
	sessionID := session.ID

	// Verify session exists
	if _, err := am.ValidateSession(sessionID); err != nil {
		t.Fatal("session should exist before invalidation")
	}

	// Invalidate session
	am.InvalidateSession(sessionID)

	// Verify session is gone
	if _, err := am.ValidateSession(sessionID); err == nil {
		t.Error("session should not exist after invalidation")
	}
}

func TestAuthManager_HandleLogin(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin",
	})
	defer am.StopCleanup()

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
	}{
		{"successful login", "admin", "admin", http.StatusOK},
		{"invalid credentials", "admin", "wrong", http.StatusUnauthorized},
		{"missing user", "notexist", "password", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{
				"username": tt.username,
				"password": tt.password,
			})

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			am.HandleLogin(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("HandleLogin() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}

				if _, exists := resp["token"]; !exists {
					t.Error("expected token in successful login response")
				}
			}
		})
	}
}

func TestAuthManager_HandleLogout(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin",
	})
	defer am.StopCleanup()

	// Create a session first
	session, _ := am.Authenticate("admin", "admin")

	// Logout request
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "k13d_session",
		Value: session.ID,
	})
	w := httptest.NewRecorder()

	am.HandleLogout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleLogout() status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify session is invalidated
	if _, err := am.ValidateSession(session.ID); err == nil {
		t.Error("session should be invalidated after logout")
	}
}

func TestAuthMiddleware(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin",
	})
	defer am.StopCleanup()

	// Create a test handler
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Create session for authenticated tests
	session, _ := am.Authenticate("admin", "admin")

	tests := []struct {
		name           string
		token          string
		useCookie      bool
		expectedStatus int
		handlerCalled  bool
	}{
		{"valid token header", session.ID, false, http.StatusOK, true},
		{"valid token cookie", session.ID, true, http.StatusOK, true},
		{"invalid token", "invalid-token", false, http.StatusUnauthorized, false},
		{"no token", "", false, http.StatusUnauthorized, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled = false

			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			if tt.token != "" {
				if tt.useCookie {
					req.AddCookie(&http.Cookie{
						Name:  "k13d_session",
						Value: tt.token,
					})
				} else {
					req.Header.Set("Authorization", "Bearer "+tt.token)
				}
			}
			w := httptest.NewRecorder()

			middleware := am.AuthMiddleware(testHandler)
			middleware.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("AuthMiddleware() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if handlerCalled != tt.handlerCalled {
				t.Errorf("handler called = %v, want %v", handlerCalled, tt.handlerCalled)
			}
		})
	}
}

func TestAuthMiddleware_Disabled(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         false,
		SessionDuration: time.Hour,
	})
	defer am.StopCleanup()

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	w := httptest.NewRecorder()

	middleware := am.AuthMiddleware(testHandler)
	middleware.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("handler should be called when auth is disabled")
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 when auth disabled, got %d", w.Code)
	}
}

func TestAuthManager_ChangePassword(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "oldpass",
	})
	defer am.StopCleanup()

	// Change password with wrong old password
	err := am.ChangePassword("admin", "wrongold", "newpass")
	if err == nil {
		t.Error("expected error when old password is wrong")
	}

	// Change password with correct old password
	err = am.ChangePassword("admin", "oldpass", "newpass")
	if err != nil {
		t.Errorf("ChangePassword() error = %v", err)
	}

	// Verify new password works
	_, err = am.Authenticate("admin", "newpass")
	if err != nil {
		t.Error("expected to authenticate with new password")
	}

	// Verify old password doesn't work
	_, err = am.Authenticate("admin", "oldpass")
	if err == nil {
		t.Error("expected old password to fail")
	}
}

func TestAuthManager_DeleteUser(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
	})
	defer am.StopCleanup()

	// Create a user to delete
	if err := am.CreateUser("testuser", "testpass1234", "user"); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	// Delete user
	err := am.DeleteUser("testuser")
	if err != nil {
		t.Errorf("DeleteUser() error = %v", err)
	}

	// Try to authenticate with deleted user
	_, err = am.Authenticate("testuser", "testpass1234")
	if err == nil {
		t.Error("expected authentication to fail for deleted user")
	}

	// Try to delete non-existent user
	err = am.DeleteUser("notexist")
	if err == nil {
		t.Error("expected error when deleting non-existent user")
	}
}

func TestAuthManager_GetUsers(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
	})
	defer am.StopCleanup()

	// Create additional users
	if err := am.CreateUser("user1", "password1234", "user"); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if err := am.CreateUser("user2", "password2abcd", "viewer"); err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	users := am.GetUsers()
	if len(users) != 3 { // admin + user1 + user2
		t.Errorf("expected 3 users, got %d", len(users))
	}
}

func TestAuthManager_HandleLogin_WithToken(t *testing.T) {
	// Test token login request (will fail since we don't have K8s cluster)
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "token",
	})
	defer am.StopCleanup()

	body, _ := json.Marshal(map[string]string{
		"token": "fake-k8s-token",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	am.HandleLogin(w, req)

	// Should fail because K8s token validator is not available in tests
	if w.Code != http.StatusUnauthorized {
		t.Errorf("HandleLogin() with token status = %d, want %d (unauthorized because no K8s cluster)", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthManager_CleanupExpiredSessions(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
		DefaultAdmin:    "admin",
		DefaultPassword: "admin",
	})
	defer am.StopCleanup()

	// Create a session
	session, err := am.Authenticate("admin", "admin")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Verify session exists
	if _, err := am.ValidateSession(session.ID); err != nil {
		t.Fatal("session should exist before cleanup")
	}

	// Manually expire the session
	am.mu.Lock()
	am.sessions[session.ID].ExpiresAt = time.Now().Add(-time.Hour)
	am.mu.Unlock()

	// Run cleanup
	am.cleanupExpiredSessions()

	// Session should be removed
	am.mu.RLock()
	_, exists := am.sessions[session.ID]
	am.mu.RUnlock()
	if exists {
		t.Error("expired session should be cleaned up")
	}
}

func TestAuthManager_CleanupExpiredCSRFTokens(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
	})
	defer am.StopCleanup()

	// Generate a CSRF token
	token := am.GenerateCSRFToken()
	if token == "" {
		t.Fatal("expected CSRF token to be generated")
	}

	// Token should be valid
	if !am.ValidateCSRFToken(token) {
		t.Error("CSRF token should be valid initially")
	}

	// Manually expire the token
	am.mu.Lock()
	am.csrfTokens[token] = time.Now().Add(-time.Hour)
	am.mu.Unlock()

	// Run cleanup
	am.CleanupExpiredCSRFTokens()

	// Token should be removed
	if am.ValidateCSRFToken(token) {
		t.Error("expired CSRF token should be cleaned up")
	}
}

func TestAuthManager_HandleUpdateUser_PathTraversal(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
	})
	defer am.StopCleanup()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{"path traversal with ..", "/api/admin/users/../admin", http.StatusBadRequest},
		{"path traversal with /", "/api/admin/users/foo/bar", http.StatusBadRequest},
		{"empty username", "/api/admin/users/", http.StatusBadRequest},
		{"valid username", "/api/admin/users/testuser", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(UserRequest{Role: "user"})
			req := httptest.NewRequest(http.MethodPut, tt.path, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			am.HandleUpdateUser(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("HandleUpdateUser(%s) status = %d, want %d", tt.path, w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestAuthManager_HandleDeleteUser_PathTraversal(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
	})
	defer am.StopCleanup()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{"path traversal with ..", "/api/admin/users/../admin", http.StatusBadRequest},
		{"path traversal with /", "/api/admin/users/foo/bar", http.StatusBadRequest},
		{"empty username", "/api/admin/users/", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, tt.path, nil)
			req.Header.Set("X-Username", "admin")
			w := httptest.NewRecorder()

			am.HandleDeleteUser(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("HandleDeleteUser(%s) status = %d, want %d", tt.path, w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestAuthManager_StopCleanup(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
	})

	// StopCleanup should not panic when called
	am.StopCleanup()

	// Calling StopCleanup again should not panic (channel already closed)
	// This is a no-op since the goroutine already exited
}

func TestAuthManager_CreateUser_Validation(t *testing.T) {
	am := NewAuthManager(&AuthConfig{
		Quiet:           true,
		Enabled:         true,
		SessionDuration: time.Hour,
		AuthMode:        "local",
	})
	defer am.StopCleanup()

	tests := []struct {
		name     string
		username string
		password string
		role     string
		wantErr  bool
	}{
		{"valid user", "testuser", "password1234", "user", false},
		{"short password", "testuser2", "short", "user", true},
		{"short username", "ab", "password1234", "user", true},
		{"long username", "a23456789012345678901234567890123", "password1234", "user", true},
		{"invalid chars in username", "test@user", "password1234", "user", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := am.CreateUser(tt.username, tt.password, tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser(%s) error = %v, wantErr %v", tt.username, err, tt.wantErr)
			}
		})
	}
}
