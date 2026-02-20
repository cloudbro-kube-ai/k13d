package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTClaims represents the claims in a k13d JWT token
type JWTClaims struct {
	Subject   string `json:"sub"`      // User ID
	Username  string `json:"username"` // Username
	Role      string `json:"role"`     // User role
	SessionID string `json:"sid"`      // Associated session ID
	IssuedAt  int64  `json:"iat"`      // Issued at (unix timestamp)
	ExpiresAt int64  `json:"exp"`      // Expires at (unix timestamp)
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret        []byte        // HMAC-SHA256 signing secret
	TokenDuration time.Duration // Token lifetime (default: 1h)
	RefreshWindow time.Duration // Window before expiry to auto-refresh (default: 15m)
}

// JWTManager handles JWT generation and validation (stdlib only, no external deps)
type JWTManager struct {
	config JWTConfig
}

// NewJWTManager creates a new JWT manager.
// NOTE: If no secret is provided, a random secret is generated on each server start.
// This means all JWT tokens are invalidated on server restart, which is the intended
// behavior for short-lived tokens. For persistent JWT validation across restarts,
// provide a stable secret via JWTConfig.Secret.
func NewJWTManager(cfg JWTConfig) *JWTManager {
	if len(cfg.Secret) == 0 {
		// Generate a random 256-bit secret
		secret := make([]byte, 32)
		_, _ = rand.Read(secret)
		cfg.Secret = secret
	}

	if cfg.TokenDuration == 0 {
		cfg.TokenDuration = 1 * time.Hour
	}
	if cfg.RefreshWindow == 0 {
		cfg.RefreshWindow = 15 * time.Minute
	}

	return &JWTManager{config: cfg}
}

// GenerateToken creates a signed JWT token from claims
func (j *JWTManager) GenerateToken(claims JWTClaims) (string, error) {
	// Set timestamps if not set
	now := time.Now()
	if claims.IssuedAt == 0 {
		claims.IssuedAt = now.Unix()
	}
	if claims.ExpiresAt == 0 {
		claims.ExpiresAt = now.Add(j.config.TokenDuration).Unix()
	}

	// Create header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	// Encode header and claims
	headerEncoded := base64URLEncode(headerJSON)
	claimsEncoded := base64URLEncode(claimsJSON)

	// Create signature
	signingInput := headerEncoded + "." + claimsEncoded
	signature := j.sign([]byte(signingInput))
	signatureEncoded := base64URLEncode(signature)

	return signingInput + "." + signatureEncoded, nil
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Verify signature
	signingInput := parts[0] + "." + parts[1]
	signatureBytes, err := base64URLDecode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}

	expectedSignature := j.sign([]byte(signingInput))
	if !hmac.Equal(signatureBytes, expectedSignature) {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode claims
	claimsBytes, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid claims encoding: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	// Check expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// RefreshToken creates a new token if the current one is within the refresh window
func (j *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Check if within refresh window
	expiresAt := time.Unix(claims.ExpiresAt, 0)
	timeUntilExpiry := time.Until(expiresAt)

	if timeUntilExpiry > j.config.RefreshWindow {
		// Not within refresh window yet, return original
		return tokenString, nil
	}

	// Create new token with refreshed timestamps
	newClaims := JWTClaims{
		Subject:   claims.Subject,
		Username:  claims.Username,
		Role:      claims.Role,
		SessionID: claims.SessionID,
	}

	return j.GenerateToken(newClaims)
}

// NeedsRefresh checks if a token is within the refresh window
func (j *JWTManager) NeedsRefresh(tokenString string) bool {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return false
	}

	expiresAt := time.Unix(claims.ExpiresAt, 0)
	return time.Until(expiresAt) <= j.config.RefreshWindow
}

// sign creates an HMAC-SHA256 signature
func (j *JWTManager) sign(data []byte) []byte {
	mac := hmac.New(sha256.New, j.config.Secret)
	mac.Write(data)
	return mac.Sum(nil)
}

// base64URLEncode encodes data to base64url (RFC 7515)
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// base64URLDecode decodes base64url data (RFC 7515)
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
