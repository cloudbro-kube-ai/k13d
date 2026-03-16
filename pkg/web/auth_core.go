package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // admin, user, viewer
	Email        string    `json:"email,omitempty"`
	DisplayName  string    `json:"display_name,omitempty"`
	Source       string    `json:"source"` // local, ldap
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login,omitempty"`

	// Emergency locking (Teleport-inspired)
	Locked   bool      `json:"locked"`
	LockedAt time.Time `json:"locked_at,omitempty"`
	LockedBy string    `json:"locked_by,omitempty"`
}

// Session represents an authenticated session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	Source    string    `json:"source"` // local, ldap
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AuthManager handles authentication
type AuthManager struct {
	users          map[string]*User     // username -> User
	sessions       map[string]*Session  // session ID -> Session
	tokenSessions  map[string]*Session  // K8s token -> Session (cached)
	csrfTokens     map[string]time.Time // CSRF token -> expiration time
	mu             sync.RWMutex
	config         *AuthConfig
	ldapProvider   *LDAPProvider
	tokenValidator *K8sTokenValidator
	oidcProvider   *OIDCProvider
	jwtManager     *JWTManager          // JWT token manager (Teleport-inspired)
	roleValidator  func(string) bool    // Optional: validates custom role names
	bruteForce     *BruteForceProtector // IP-based brute-force protection
	accountLockout *AccountLockout      // Account-based lockout protection
	csrfDone       chan struct{}        // signals CSRF cleanup goroutine to stop
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled         bool          `yaml:"enabled" json:"enabled"`
	SessionDuration time.Duration `yaml:"session_duration" json:"session_duration"`
	DefaultAdmin    string        `yaml:"default_admin" json:"default_admin"`
	DefaultPassword string        `yaml:"default_password" json:"-"`
	LDAP            *LDAPConfig   `yaml:"ldap" json:"ldap"`
	OIDC            *OIDCConfig   `yaml:"oidc" json:"oidc"`
	// AuthMode: "token" (K8s RBAC token - default), "local" (username/password), "ldap", "oidc"
	AuthMode string `yaml:"auth_mode" json:"auth_mode"`
	// Quiet suppresses informational output (useful for tests)
	Quiet bool `yaml:"-" json:"-"`
}

// AuthOptions holds CLI authentication options
type AuthOptions struct {
	Mode            string // token, local, ldap
	Disabled        bool   // Disable authentication entirely
	DefaultAdmin    string // Default admin username for local mode
	DefaultPassword string // Default admin password for local mode
}

// NewAuthManager creates a new AuthManager
func NewAuthManager(cfg *AuthConfig) *AuthManager {
	am := &AuthManager{
		users:          make(map[string]*User),
		sessions:       make(map[string]*Session),
		tokenSessions:  make(map[string]*Session),
		csrfTokens:     make(map[string]time.Time),
		config:         cfg,
		jwtManager:     NewJWTManager(JWTConfig{}), // Auto-generates secret
		bruteForce:     NewBruteForceProtector(),
		accountLockout: NewAccountLockout(),
	}

	// Set default auth mode to "token" if not specified
	if cfg.AuthMode == "" {
		cfg.AuthMode = "token"
	}

	// Initialize K8s token validator for token auth mode
	if cfg.AuthMode == "token" || cfg.AuthMode == "" {
		validator, err := NewK8sTokenValidator()
		if err != nil {
			// Token validation may fail if kubeconfig is not available
			if !cfg.Quiet {
				fmt.Printf("  K8s token validator: Not available (%v)\n", err)
			}
		} else {
			am.tokenValidator = validator
			if !cfg.Quiet {
				fmt.Printf("  K8s token validator: Ready (using kubeconfig)\n")
			}
		}
	}

	// Initialize LDAP provider if configured
	if cfg.LDAP != nil && cfg.LDAP.Enabled {
		am.ldapProvider = NewLDAPProvider(cfg.LDAP)
	}

	// Initialize OIDC provider if configured
	if cfg.OIDC != nil && cfg.OIDC.ProviderURL != "" && cfg.OIDC.ClientID != "" {
		provider, err := NewOIDCProvider(cfg.OIDC)
		if err != nil {
			if !cfg.Quiet {
				fmt.Printf("  OIDC provider: Failed to initialize (%v)\n", err)
			}
		} else {
			am.oidcProvider = provider
			if !cfg.Quiet {
				fmt.Printf("  OIDC provider: Ready (%s)\n", cfg.OIDC.ProviderName)
			}
		}
	}

	// Start periodic CSRF token cleanup
	am.csrfDone = make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				am.CleanupExpiredCSRFTokens()
				am.cleanupExpiredSessions()
			case <-am.csrfDone:
				return
			}
		}
	}()

	// Create default admin user only for local auth mode
	if cfg.Enabled && cfg.AuthMode == "local" {
		adminUser := cfg.DefaultAdmin
		if adminUser == "" {
			adminUser = "admin"
		}
		adminPass := cfg.DefaultPassword
		if adminPass == "" {
			// Generate a secure random password instead of hardcoded default
			adminPass = generateSecurePassword(16)
			// Print password to stderr to avoid capture in structured log output
			fmt.Fprintf(os.Stderr, "  Admin password: %s\n", adminPass)
			fmt.Printf("  WARNING: Random admin password generated (see stderr). Change after first login.\n")
		}
		if err := am.createLocalUser(adminUser, adminPass, "admin"); err != nil {
			fmt.Printf("  Warning: failed to create admin user: %v\n", err)
		}
	}

	return am
}

// SetRoleValidator sets a function that validates custom role names
func (am *AuthManager) SetRoleValidator(fn func(string) bool) {
	am.roleValidator = fn
}

// GetAuthMode returns the current authentication mode
func (am *AuthManager) GetAuthMode() string {
	return am.config.AuthMode
}

// ValidateSession checks if a session ID is valid and returns the session
func (am *AuthManager) ValidateSession(sessionID string) (*Session, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	session, ok := am.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	if time.Now().After(session.ExpiresAt) {
		delete(am.sessions, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// InvalidateSession removes a session
func (am *AuthManager) InvalidateSession(sessionID string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.sessions, sessionID)
}

// GetUsers returns all local users
func (am *AuthManager) GetUsers() []*User {
	am.mu.RLock()
	defer am.mu.RUnlock()

	users := make([]*User, 0, len(am.users))
	for _, user := range am.users {
		users = append(users, user)
	}
	return users
}

// AuthMiddleware is a middleware that validates authentication
func (am *AuthManager) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !am.config.Enabled {
			// Set admin role headers so downstream handlers (RBAC, feature gates)
			// treat the anonymous user as admin when auth is disabled.
			r.Header.Set("X-User-ID", "anonymous")
			r.Header.Set("X-Username", "anonymous")
			r.Header.Set("X-User-Role", "admin")
			next(w, r)
			return
		}

		// Check for session cookie, Authorization header, or query parameter
		sessionID := ""
		token := ""

		// Try cookie first
		if cookie, err := r.Cookie("k13d_session"); err == nil {
			sessionID = cookie.Value
		}

		// Try Authorization header
		if sessionID == "" {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				token = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		// Try query parameter (for WebSocket connections that cannot set headers)
		if sessionID == "" && token == "" {
			if qToken := r.URL.Query().Get("token"); qToken != "" {
				token = qToken
			}
		}

		// For token auth mode, try K8s token validation first
		if am.config.AuthMode == "token" && token != "" {
			session, err := am.ValidateK8sToken(r.Context(), token)
			if err == nil {
				r.Header.Set("X-User-ID", session.UserID)
				r.Header.Set("X-Username", session.Username)
				r.Header.Set("X-User-Role", session.Role)
				next(w, r)
				return
			}
			// Fall through to session validation
			sessionID = token
		}

		if sessionID == "" && token != "" {
			sessionID = token
		}

		if sessionID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Try JWT validation first (Teleport-inspired short-lived tokens)
		if am.jwtManager != nil {
			claims, jwtErr := am.jwtManager.ValidateToken(sessionID)
			if jwtErr == nil {
				// JWT is valid - check if user is locked
				if am.IsUserLocked(claims.Username) {
					http.Error(w, "Account locked", http.StatusForbidden)
					return
				}

				r.Header.Set("X-User-ID", claims.Subject)
				r.Header.Set("X-Username", claims.Username)
				r.Header.Set("X-User-Role", claims.Role)

				// Auto-refresh if near expiry
				if am.jwtManager.NeedsRefresh(sessionID) {
					newToken, refreshErr := am.jwtManager.RefreshToken(sessionID)
					if refreshErr == nil && newToken != sessionID {
						w.Header().Set("X-Refreshed-Token", newToken)
					}
				}

				next(w, r)
				return
			}
			// JWT validation failed, fall through to opaque session
		}

		session, err := am.ValidateSession(sessionID)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Check if user is locked (Teleport-inspired emergency locking)
		if am.IsUserLocked(session.Username) {
			am.InvalidateSession(sessionID)
			http.Error(w, "Account locked", http.StatusForbidden)
			return
		}

		// Add session info to request context
		r.Header.Set("X-User-ID", session.UserID)
		r.Header.Set("X-Username", session.Username)
		r.Header.Set("X-User-Role", session.Role)

		next(w, r)
	}
}

// AdminMiddleware is a middleware that requires admin role
func (am *AuthManager) AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := r.Header.Get("X-User-Role")
		if role != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// HandleAuthStatus returns the current authentication status
func (am *AuthManager) HandleAuthStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Determine runtime environment
	environment := "unknown"
	kubeconfigUser := ""
	if am.tokenValidator != nil {
		environment = string(am.tokenValidator.GetEnvironment())
		kubeconfigUser = am.tokenValidator.GetKubeconfigUser()
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"auth_enabled":     am.config.Enabled,
		"auth_mode":        am.config.AuthMode,
		"ldap_enabled":     am.IsLDAPEnabled(),
		"oidc_configured":  am.oidcProvider != nil,
		"token_available":  am.tokenValidator != nil,
		"session_duration": am.config.SessionDuration.String(),
		"total_users":      len(am.users),
		"active_sessions":  len(am.sessions),
		"environment":      environment,
		"kubeconfig_user":  kubeconfigUser,
	})
}

// HandleCurrentUser returns the current user info
func (am *AuthManager) HandleCurrentUser(w http.ResponseWriter, r *http.Request) {
	if !am.config.Enabled {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"username":     "anonymous",
			"role":         "admin",
			"auth_enabled": false,
			"ldap_enabled": false,
			"auth_mode":    "none",
		})
		return
	}

	username := r.Header.Get("X-Username")
	role := r.Header.Get("X-User-Role")

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"username":        username,
		"role":            role,
		"auth_enabled":    true,
		"ldap_enabled":    am.IsLDAPEnabled(),
		"auth_mode":       am.config.AuthMode,
		"token_available": am.tokenValidator != nil,
	})
}

// GenerateCSRFToken generates a new CSRF token
func (am *AuthManager) GenerateCSRFToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)

	am.mu.Lock()
	defer am.mu.Unlock()
	am.csrfTokens[token] = time.Now().Add(1 * time.Hour)
	return token
}

// ValidateCSRFToken validates a CSRF token
func (am *AuthManager) ValidateCSRFToken(token string) bool {
	if token == "" {
		return false
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	expiry, ok := am.csrfTokens[token]
	if !ok {
		return false
	}

	if time.Now().After(expiry) {
		delete(am.csrfTokens, token)
		return false
	}

	return true
}

// CleanupExpiredCSRFTokens removes expired CSRF tokens
func (am *AuthManager) CleanupExpiredCSRFTokens() {
	am.mu.Lock()
	defer am.mu.Unlock()
	now := time.Now()
	for token, expiry := range am.csrfTokens {
		if now.After(expiry) {
			delete(am.csrfTokens, token)
		}
	}
}

// StopCleanup stops the cleanup goroutine
func (am *AuthManager) StopCleanup() {
	close(am.csrfDone)
}

// cleanupExpiredSessions removes expired sessions and token sessions
func (am *AuthManager) cleanupExpiredSessions() {
	am.mu.Lock()
	defer am.mu.Unlock()
	now := time.Now()
	for id, session := range am.sessions {
		if now.After(session.ExpiresAt) {
			delete(am.sessions, id)
		}
	}
	for token, session := range am.tokenSessions {
		if now.After(session.ExpiresAt) {
			delete(am.tokenSessions, token)
		}
	}
}

// HandleCSRFToken returns a new CSRF token
func (am *AuthManager) HandleCSRFToken(w http.ResponseWriter, r *http.Request) {
	token := am.GenerateCSRFToken()
	if token == "" {
		http.Error(w, "Failed to generate CSRF token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"csrf_token": token})
}

// CSRFMiddleware handles CSRF protection
func (am *AuthManager) CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF check for safe methods
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF check for login/logout/OIDC endpoints (no session exists yet or OAuth flow)
		if r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/logout" ||
			r.URL.Path == "/api/auth/kubeconfig" ||
			strings.HasPrefix(r.URL.Path, "/api/auth/oidc/") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF check for API endpoints that use Bearer token auth.
		// This is intentional: Bearer token authentication is not vulnerable to CSRF
		// because browsers do not automatically attach Authorization headers to
		// cross-origin requests (unlike cookies). API clients using Bearer tokens
		// are therefore exempt from CSRF validation.
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}

		// Validate CSRF token from header
		csrfToken := r.Header.Get("X-CSRF-Token")
		if csrfToken == "" {
			csrfToken = r.Header.Get("X-Csrf-Token") // Case-insensitive fallback
		}

		if !am.ValidateCSRFToken(csrfToken) {
			http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
