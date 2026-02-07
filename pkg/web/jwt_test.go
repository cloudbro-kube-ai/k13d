package web

import (
	"strings"
	"testing"
	"time"
)

func TestJWT_GenerateAndValidate(t *testing.T) {
	jm := NewJWTManager(JWTConfig{
		Secret:        []byte("test-secret-key-for-testing-only"),
		TokenDuration: 1 * time.Hour,
		RefreshWindow: 15 * time.Minute,
	})

	claims := JWTClaims{
		Subject:   "user-123",
		Username:  "admin",
		Role:      "admin",
		SessionID: "session-456",
	}

	token, err := jm.GenerateToken(claims)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("Expected non-empty token")
	}

	// Validate
	validated, err := jm.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if validated.Subject != "user-123" {
		t.Errorf("Subject: got %q, want %q", validated.Subject, "user-123")
	}
	if validated.Username != "admin" {
		t.Errorf("Username: got %q, want %q", validated.Username, "admin")
	}
	if validated.Role != "admin" {
		t.Errorf("Role: got %q, want %q", validated.Role, "admin")
	}
	if validated.SessionID != "session-456" {
		t.Errorf("SessionID: got %q, want %q", validated.SessionID, "session-456")
	}
}

func TestJWT_ExpiredToken(t *testing.T) {
	jm := NewJWTManager(JWTConfig{
		Secret:        []byte("test-secret"),
		TokenDuration: 1 * time.Hour,
	})

	// Create an already-expired token
	claims := JWTClaims{
		Subject:   "user-123",
		Username:  "admin",
		Role:      "admin",
		IssuedAt:  time.Now().Add(-2 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
	}

	token, err := jm.GenerateToken(claims)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = jm.ValidateToken(token)
	if err == nil {
		t.Error("Expected error for expired token")
	}
}

func TestJWT_TamperedToken(t *testing.T) {
	jm := NewJWTManager(JWTConfig{
		Secret: []byte("test-secret"),
	})

	token, err := jm.GenerateToken(JWTClaims{
		Subject:  "user-123",
		Username: "admin",
		Role:     "admin",
	})
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// Tamper with the payload by replacing the claims section
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatal("Invalid token format")
	}

	// Replace the claims (middle part) with different data
	tamperedClaims := base64URLEncode([]byte(`{"sub":"hacker","username":"evil","role":"admin","iat":1,"exp":9999999999}`))
	tampered := parts[0] + "." + tamperedClaims + "." + parts[2]

	_, err = jm.ValidateToken(tampered)
	if err == nil {
		t.Error("Expected error for tampered token")
	}
}

func TestJWT_DifferentSecret(t *testing.T) {
	jm1 := NewJWTManager(JWTConfig{Secret: []byte("secret-1")})
	jm2 := NewJWTManager(JWTConfig{Secret: []byte("secret-2")})

	token, err := jm1.GenerateToken(JWTClaims{
		Subject:  "user-123",
		Username: "admin",
		Role:     "admin",
	})
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// Token signed with secret-1 should not validate with secret-2
	_, err = jm2.ValidateToken(token)
	if err == nil {
		t.Error("Expected error when validating with different secret")
	}
}

func TestJWT_RefreshWindow(t *testing.T) {
	jm := NewJWTManager(JWTConfig{
		Secret:        []byte("test-secret"),
		TokenDuration: 1 * time.Hour,
		RefreshWindow: 15 * time.Minute,
	})

	// Token that expires in 10 minutes (within refresh window)
	claims := JWTClaims{
		Subject:   "user-123",
		Username:  "admin",
		Role:      "admin",
		IssuedAt:  time.Now().Add(-50 * time.Minute).Unix(),
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	}

	token, _ := jm.GenerateToken(claims)

	if !jm.NeedsRefresh(token) {
		t.Error("Token within refresh window should need refresh")
	}

	// Refresh should return a new token
	newToken, err := jm.RefreshToken(token)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	if newToken == token {
		t.Error("Expected refreshed token to be different")
	}

	// New token should be valid
	newClaims, err := jm.ValidateToken(newToken)
	if err != nil {
		t.Fatalf("Refreshed token validation failed: %v", err)
	}

	if newClaims.Username != "admin" {
		t.Error("Refreshed token should preserve claims")
	}
}

func TestJWT_NotInRefreshWindow(t *testing.T) {
	jm := NewJWTManager(JWTConfig{
		Secret:        []byte("test-secret"),
		TokenDuration: 1 * time.Hour,
		RefreshWindow: 15 * time.Minute,
	})

	// Fresh token (not in refresh window yet)
	token, _ := jm.GenerateToken(JWTClaims{
		Subject:  "user-123",
		Username: "admin",
		Role:     "admin",
	})

	if jm.NeedsRefresh(token) {
		t.Error("Fresh token should not need refresh")
	}

	// RefreshToken should return the same token
	refreshed, err := jm.RefreshToken(token)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	if refreshed != token {
		t.Error("Token not in refresh window should be returned unchanged")
	}
}

func TestJWT_InvalidFormat(t *testing.T) {
	jm := NewJWTManager(JWTConfig{Secret: []byte("test-secret")})

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"no dots", "abcdef"},
		{"one dot", "abc.def"},
		{"too many dots", "a.b.c.d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jm.ValidateToken(tt.token)
			if err == nil {
				t.Error("Expected error for invalid token format")
			}
		})
	}
}

func TestJWT_AutoGenerateSecret(t *testing.T) {
	// Empty secret should auto-generate
	jm := NewJWTManager(JWTConfig{})

	token, err := jm.GenerateToken(JWTClaims{
		Subject:  "user-123",
		Username: "admin",
		Role:     "admin",
	})
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	// Should be able to validate with the same manager
	_, err = jm.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
}
