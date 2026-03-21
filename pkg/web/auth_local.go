package web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"golang.org/x/crypto/bcrypt"
)

func (am *AuthManager) createLocalUser(username, password, role string) error {
	if _, exists := am.users[username]; exists {
		return fmt.Errorf("user already exists: %s", username)
	}

	am.users[username] = &User{
		ID:           generateSessionID()[:16],
		Username:     username,
		PasswordHash: hashPassword(password),
		Role:         role,
		Source:       "local",
		CreatedAt:    time.Now(),
	}

	return nil
}

// hashPassword creates a bcrypt hash of the password
func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// Fallback should never happen with valid input
		return ""
	}
	return string(hash)
}

// checkPassword verifies a password against its hash.
// Supports bcrypt (preferred) and falls back to SHA256 for legacy hashes.
func checkPassword(password, hash string) bool {
	// Bcrypt hashes start with "$2a$", "$2b$", or "$2y$"
	if strings.HasPrefix(hash, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	// Legacy SHA256 fallback for existing users migrating from older versions
	h := sha256.Sum256([]byte(password))
	return hash == hex.EncodeToString(h[:])
}

// generateSessionID creates a random session ID
func generateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use timestamp + smaller random read as best effort
		fallback := make([]byte, 16)
		for i := range fallback {
			fallback[i] = byte(time.Now().UnixNano() >> (i * 4))
		}
		return base64.URLEncoding.EncodeToString(fallback)
	}
	return base64.URLEncoding.EncodeToString(b)
}

// CreateUser creates a new local user
func (am *AuthManager) CreateUser(username, password, role string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.users[username]; exists {
		return fmt.Errorf("user already exists: %s", username)
	}

	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}

	if len(username) < 3 || len(username) > 32 {
		return fmt.Errorf("username must be 3-32 characters")
	}
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, username); !matched {
		return fmt.Errorf("username must contain only alphanumeric characters, hyphens, and underscores")
	}

	am.users[username] = &User{
		ID:           generateSessionID()[:16],
		Username:     username,
		PasswordHash: hashPassword(password),
		Role:         role,
		Source:       "local",
		CreatedAt:    time.Now(),
	}

	return nil
}

// Authenticate validates credentials and creates a session
func (am *AuthManager) Authenticate(username, password string) (*Session, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Try LDAP authentication first if enabled
	if am.ldapProvider != nil && am.ldapProvider.IsEnabled() {
		ldapUser, err := am.ldapProvider.Authenticate(username, password)
		if err == nil {
			// LDAP auth successful - create or update local user cache
			user, exists := am.users[username]
			if !exists {
				user = &User{
					ID:          generateSessionID()[:16],
					Username:    ldapUser.Username,
					Email:       ldapUser.Email,
					DisplayName: ldapUser.DisplayName,
					Role:        ldapUser.Role,
					Source:      "ldap",
					CreatedAt:   time.Now(),
				}
				am.users[username] = user
			} else {
				// Update user info from LDAP
				user.Email = ldapUser.Email
				user.DisplayName = ldapUser.DisplayName
				user.Role = ldapUser.Role
				user.Source = "ldap"
			}
			user.LastLogin = time.Now()

			// Create session
			session := &Session{
				ID:        generateSessionID(),
				UserID:    user.ID,
				Username:  user.Username,
				Role:      user.Role,
				Source:    "ldap",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(am.config.SessionDuration),
			}
			am.sessions[session.ID] = session
			return session, nil
		}
		// LDAP auth failed, fall through to local auth
	}

	// Try local authentication
	user, exists := am.users[username]
	if !exists {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Only allow local auth for local users
	if user.Source != "local" && user.Source != "" {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !checkPassword(password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Update last login
	user.LastLogin = time.Now()

	// Create session
	session := &Session{
		ID:        generateSessionID(),
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		Source:    "local",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(am.config.SessionDuration),
	}

	am.sessions[session.ID] = session

	return session, nil
}

func (am *AuthManager) ChangePassword(username, oldPassword, newPassword string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	user, exists := am.users[username]
	if !exists {
		return fmt.Errorf("user not found")
	}

	if !checkPassword(oldPassword, user.PasswordHash) {
		return fmt.Errorf("invalid current password")
	}

	user.PasswordHash = hashPassword(newPassword)
	return nil
}

// DeleteUser removes a user
func (am *AuthManager) DeleteUser(username string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.users[username]; !exists {
		return fmt.Errorf("user not found")
	}

	delete(am.users, username)
	return nil
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"` // K8s service account token
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	JWTToken  string    `json:"jwt_token,omitempty"` // JWT token (Teleport-inspired short-lived)
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
	AuthMode  string    `json:"auth_mode"`
}

// HandleLogin handles login requests
func (am *AuthManager) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	clientIP := ClientIP(r)

	// Check if the IP is blocked by brute-force protection
	if am.bruteForce.IsBlocked(clientIP) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Too many failed login attempts. Please try again later.",
		})
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	// Check account-based lockout (independent of IP rate limit)
	if req.Username != "" && am.accountLockout.IsLocked(req.Username) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "Account is locked. Please try again after 30 minutes.",
		})
		return
	}

	// Before creating new session, invalidate old session to prevent session fixation
	if cookie, err := r.Cookie("k13d_session"); err == nil && cookie.Value != "" {
		am.InvalidateSession(cookie.Value)
	}

	var session *Session
	var err error

	// Handle token-based login (K8s RBAC)
	if req.Token != "" {
		session, err = am.ValidateK8sToken(r.Context(), req.Token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid K8s token: " + err.Error(),
			})
			return
		}
		// Also store in sessions map for ValidateSession to find it
		am.mu.Lock()
		am.sessions[session.ID] = session
		am.mu.Unlock()
	} else {
		// Handle username/password login (local or LDAP)
		session, err = am.Authenticate(req.Username, req.Password)
		if err != nil {
			// Record account-level failure
			am.accountLockout.RecordFailure(req.Username)
			// Record IP-level failure and apply progressive delay
			delay := am.bruteForce.RecordFailure(clientIP)
			if delay > 0 {
				time.Sleep(delay)
			}
			WriteError(w, NewAPIError(ErrCodeUnauthorized, "Invalid credentials"))
			return
		}
	}

	// Login succeeded — clear failure counters
	am.bruteForce.RecordSuccess(clientIP)
	am.accountLockout.RecordSuccess(req.Username)

	// Set session cookie with security flags
	http.SetCookie(w, &http.Cookie{
		Name:     "k13d_session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil, // Set Secure flag if HTTPS
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})

	// Generate JWT token alongside session (Teleport-inspired)
	var jwtToken string
	if am.jwtManager != nil {
		jwtToken, _ = am.jwtManager.GenerateToken(JWTClaims{
			Subject:   session.UserID,
			Username:  session.Username,
			Role:      session.Role,
			SessionID: session.ID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LoginResponse{
		Token:     session.ID,
		JWTToken:  jwtToken,
		Username:  session.Username,
		Role:      session.Role,
		ExpiresAt: session.ExpiresAt,
		AuthMode:  session.Source,
	})
}

// HandleLogout handles logout requests
func (am *AuthManager) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session from cookie or header
	sessionID := ""
	if cookie, err := r.Cookie("k13d_session"); err == nil {
		sessionID = cookie.Value
	}
	// Also check Authorization header
	if sessionID == "" {
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				sessionID = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
	}

	if sessionID != "" {
		am.InvalidateSession(sessionID)
	}

	// Clear cookie with security flags
	http.SetCookie(w, &http.Cookie{
		Name:     "k13d_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "logged out"})
}

// UserRequest represents a user creation/update request
type UserRequest struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role"`
	Email    string `json:"email,omitempty"`
}

// HandleListUsers returns list of all users (admin only)
func (am *AuthManager) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	users := am.GetUsers()

	// Sanitize user data (remove sensitive fields)
	safeUsers := make([]map[string]interface{}, len(users))
	for i, u := range users {
		safeUsers[i] = map[string]interface{}{
			"id":           u.ID,
			"username":     u.Username,
			"role":         u.Role,
			"email":        u.Email,
			"display_name": u.DisplayName,
			"source":       u.Source,
			"created_at":   u.CreatedAt,
			"last_login":   u.LastLogin,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"users": safeUsers,
		"total": len(safeUsers),
	})
}

// HandleCreateUser creates a new user (admin only)
func (am *AuthManager) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	if req.Username == "" || req.Password == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "Username and password are required"))
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	// Validate role: accept built-in roles; custom roles validated via roleValidator if set
	validBuiltIn := req.Role == "admin" || req.Role == "user" || req.Role == "viewer"
	if !validBuiltIn && (am.roleValidator == nil || !am.roleValidator(req.Role)) {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid role. Must be admin, user, viewer, or a valid custom role"))
		return
	}

	if err := am.CreateUser(req.Username, req.Password, req.Role); err != nil {
		WriteError(w, NewAPIError(ErrCodeConflict, err.Error()))
		return
	}

	// Update email if provided
	if req.Email != "" {
		am.mu.Lock()
		if user, exists := am.users[req.Username]; exists {
			user.Email = req.Email
		}
		am.mu.Unlock()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":   "created",
		"username": req.Username,
	})
}

// HandleUpdateUser updates an existing user (admin only)
func (am *AuthManager) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		writeMethodNotAllowed(w)
		return
	}

	// Get username from URL path
	username := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	if username == "" || strings.ContainsAny(username, "/\\..") {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid username"))
		return
	}

	var req UserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	user, exists := am.users[username]
	if !exists {
		WriteError(w, NewAPIError(ErrCodeNotFound, "User not found"))
		return
	}

	// Only allow updating local users
	if user.Source != "local" && user.Source != "" {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Cannot update non-local user"))
		return
	}

	// Update fields
	if req.Role != "" {
		validBuiltIn := req.Role == "admin" || req.Role == "user" || req.Role == "viewer"
		if !validBuiltIn && (am.roleValidator == nil || !am.roleValidator(req.Role)) {
			WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid role"))
			return
		}
		user.Role = req.Role
	}

	if req.Email != "" {
		user.Email = req.Email
	}

	if req.Password != "" {
		user.PasswordHash = hashPassword(req.Password)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":   "updated",
		"username": username,
	})
}

// HandleDeleteUser deletes a user (admin only)
func (am *AuthManager) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeMethodNotAllowed(w)
		return
	}

	// Get username from URL path
	username := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")
	if username == "" || strings.ContainsAny(username, "/\\..") {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid username"))
		return
	}

	// Prevent deleting the current user
	currentUser := r.Header.Get("X-Username")
	if username == currentUser {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Cannot delete your own account"))
		return
	}

	if err := am.DeleteUser(username); err != nil {
		WriteError(w, NewAPIError(ErrCodeNotFound, err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":   "deleted",
		"username": username,
	})
}

// HandleResetPassword resets a user's password (admin only)
func (am *AuthManager) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req struct {
		Username    string `json:"username"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	if req.Username == "" || req.NewPassword == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "Username and new password are required"))
		return
	}

	if len(req.NewPassword) < 12 {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Password must be at least 12 characters"))
		return
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	user, exists := am.users[req.Username]
	if !exists {
		WriteError(w, NewAPIError(ErrCodeNotFound, "User not found"))
		return
	}

	if user.Source != "local" && user.Source != "" {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Cannot reset password for non-local user"))
		return
	}

	user.PasswordHash = hashPassword(req.NewPassword)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":   "password_reset",
		"username": req.Username,
	})
}

func generateSecurePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a less secure but still random password
		return fmt.Sprintf("k13d-%d", time.Now().UnixNano())
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// LockUser locks a user account immediately and invalidates all sessions (Teleport-inspired)
func (am *AuthManager) LockUser(username, lockedBy string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	user, exists := am.users[username]
	if !exists {
		return fmt.Errorf("user not found: %s", username)
	}

	user.Locked = true
	user.LockedAt = time.Now()
	user.LockedBy = lockedBy

	// Invalidate all sessions for this user
	for id, session := range am.sessions {
		if session.Username == username {
			delete(am.sessions, id)
		}
	}

	// Persist lock to database
	if db.DB != nil {
		_, err := db.DB.Exec(
			"INSERT OR REPLACE INTO user_locks (username, locked_at, locked_by, reason) VALUES (?, ?, ?, ?)",
			username, user.LockedAt, lockedBy, "emergency lock")
		if err != nil {
			fmt.Printf("Warning: failed to persist user lock: %v\n", err)
		}
	}

	// Record audit event
	_ = db.RecordAudit(db.AuditEntry{
		User:       lockedBy,
		Action:     "lock_user",
		Resource:   "user/" + username,
		Details:    fmt.Sprintf("User %s locked by %s", username, lockedBy),
		ActionType: db.ActionTypeUserLocked,
		Source:     "web",
		Success:    true,
	})

	return nil
}

// UnlockUser unlocks a previously locked user account
func (am *AuthManager) UnlockUser(username string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	user, exists := am.users[username]
	if !exists {
		return fmt.Errorf("user not found: %s", username)
	}

	user.Locked = false
	user.LockedAt = time.Time{}
	user.LockedBy = ""

	// Remove from database
	if db.DB != nil {
		_, _ = db.DB.Exec("DELETE FROM user_locks WHERE username = ?", username)
	}

	// Record audit event
	_ = db.RecordAudit(db.AuditEntry{
		User:       username,
		Action:     "unlock_user",
		Resource:   "user/" + username,
		Details:    fmt.Sprintf("User %s unlocked", username),
		ActionType: db.ActionTypeUserUnlocked,
		Source:     "web",
		Success:    true,
	})

	return nil
}

// IsUserLocked checks if a user is locked (fast path: in-memory, fallback: DB)
func (am *AuthManager) IsUserLocked(username string) bool {
	am.mu.RLock()
	user, exists := am.users[username]
	am.mu.RUnlock()

	if exists {
		return user.Locked
	}

	// Fallback: check database for users not yet loaded in memory
	if db.DB != nil {
		var count int
		err := db.DB.QueryRow("SELECT COUNT(*) FROM user_locks WHERE username = ?", username).Scan(&count)
		if err == nil && count > 0 {
			return true
		}
	}

	return false
}

// LoadUserLocks loads persisted user locks from the database on startup
func (am *AuthManager) LoadUserLocks() {
	if db.DB == nil {
		return
	}

	rows, err := db.DB.Query("SELECT username, locked_at, locked_by FROM user_locks")
	if err != nil {
		return
	}
	defer rows.Close()

	am.mu.Lock()
	defer am.mu.Unlock()

	for rows.Next() {
		var username, lockedBy string
		var lockedAt time.Time
		if err := rows.Scan(&username, &lockedAt, &lockedBy); err != nil {
			continue
		}
		if user, exists := am.users[username]; exists {
			user.Locked = true
			user.LockedAt = lockedAt
			user.LockedBy = lockedBy
		}
	}
}

// HandleLockUser handles POST /api/admin/lock - locks a user account (admin only)
func (am *AuthManager) HandleLockUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	if req.Username == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "Username is required"))
		return
	}

	// Prevent self-lock
	currentUser := r.Header.Get("X-Username")
	if req.Username == currentUser {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Cannot lock your own account"))
		return
	}

	if err := am.LockUser(req.Username, currentUser); err != nil {
		WriteError(w, NewAPIError(ErrCodeNotFound, err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":   "locked",
		"username": req.Username,
	})
}

// HandleUnlockUser handles POST /api/admin/unlock - unlocks a user account (admin only)
func (am *AuthManager) HandleUnlockUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	if req.Username == "" {
		WriteError(w, NewAPIError(ErrCodeValidation, "Username is required"))
		return
	}

	if err := am.UnlockUser(req.Username); err != nil {
		WriteError(w, NewAPIError(ErrCodeNotFound, err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":   "unlocked",
		"username": req.Username,
	})
}
