package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OIDCConfig holds OIDC provider configuration
type OIDCConfig struct {
	ProviderName string `yaml:"provider_name" json:"provider_name"`
	ProviderURL  string `yaml:"provider_url" json:"provider_url"`
	ClientID     string `yaml:"client_id" json:"client_id"`
	ClientSecret string `yaml:"client_secret" json:"-"`
	RedirectURI  string `yaml:"redirect_uri" json:"redirect_uri"`
	Scopes       string `yaml:"scopes" json:"scopes"`
	// Role mapping
	RolesClaim    string            `yaml:"roles_claim" json:"roles_claim"`
	AdminRoles    []string          `yaml:"admin_roles" json:"admin_roles"`
	UserRoles     []string          `yaml:"user_roles" json:"user_roles"`
	DefaultRole   string            `yaml:"default_role" json:"default_role"`
	GroupMappings map[string]string `yaml:"group_mappings" json:"group_mappings"`
}

// OIDCProvider handles OIDC authentication
type OIDCProvider struct {
	config        *OIDCConfig
	discovery     *OIDCDiscovery
	states        map[string]time.Time // state -> expiration
	mu            sync.RWMutex
	httpClient    *http.Client
	lastDiscovery time.Time
}

// OIDCDiscovery holds OIDC discovery document data
type OIDCDiscovery struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserinfoEndpoint      string   `json:"userinfo_endpoint"`
	JwksURI               string   `json:"jwks_uri"`
	ScopesSupported       []string `json:"scopes_supported"`
}

// OIDCTokenResponse represents the token response from OIDC provider
type OIDCTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// OIDCUserInfo represents user info from OIDC provider
type OIDCUserInfo struct {
	Sub           string   `json:"sub"`
	Name          string   `json:"name"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Picture       string   `json:"picture,omitempty"`
	Groups        []string `json:"groups,omitempty"`
	Roles         []string `json:"roles,omitempty"`
}

// NewOIDCProvider creates a new OIDC provider
func NewOIDCProvider(config *OIDCConfig) (*OIDCProvider, error) {
	if config == nil {
		return nil, fmt.Errorf("OIDC config is nil")
	}

	if config.ProviderURL == "" {
		return nil, fmt.Errorf("OIDC provider URL is required")
	}

	if config.ClientID == "" {
		return nil, fmt.Errorf("OIDC client ID is required")
	}

	provider := &OIDCProvider{
		config: config,
		states: make(map[string]time.Time),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Fetch discovery document
	if err := provider.fetchDiscovery(); err != nil {
		return nil, fmt.Errorf("failed to fetch OIDC discovery: %w", err)
	}

	// Set default scopes if not specified
	if config.Scopes == "" {
		config.Scopes = "openid email profile"
	}

	// Set default role if not specified
	if config.DefaultRole == "" {
		config.DefaultRole = "viewer"
	}

	return provider, nil
}

// fetchDiscovery fetches the OIDC discovery document
func (p *OIDCProvider) fetchDiscovery() error {
	// Build discovery URL
	discoveryURL := p.config.ProviderURL
	if !strings.HasSuffix(discoveryURL, "/.well-known/openid-configuration") {
		discoveryURL = strings.TrimSuffix(discoveryURL, "/") + "/.well-known/openid-configuration"
	}

	resp, err := p.httpClient.Get(discoveryURL)
	if err != nil {
		return fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discovery endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var discovery OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return fmt.Errorf("failed to decode discovery document: %w", err)
	}

	p.discovery = &discovery
	p.lastDiscovery = time.Now()
	return nil
}

// GetAuthorizationURL returns the URL to redirect user for authentication
func (p *OIDCProvider) GetAuthorizationURL(redirectURI string) (string, string, error) {
	if p.discovery == nil {
		if err := p.fetchDiscovery(); err != nil {
			return "", "", err
		}
	}

	// Generate state for CSRF protection
	state, err := generateRandomString(32)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state with expiration
	p.mu.Lock()
	p.states[state] = time.Now().Add(10 * time.Minute)
	// Clean up old states
	for s, exp := range p.states {
		if time.Now().After(exp) {
			delete(p.states, s)
		}
	}
	p.mu.Unlock()

	// Build authorization URL
	authURL, err := url.Parse(p.discovery.AuthorizationEndpoint)
	if err != nil {
		return "", "", fmt.Errorf("invalid authorization endpoint: %w", err)
	}

	params := url.Values{}
	params.Set("client_id", p.config.ClientID)
	params.Set("response_type", "code")
	params.Set("scope", p.config.Scopes)
	params.Set("redirect_uri", redirectURI)
	params.Set("state", state)

	authURL.RawQuery = params.Encode()
	return authURL.String(), state, nil
}

// ValidateState validates the state parameter from callback
func (p *OIDCProvider) ValidateState(state string) bool {
	p.mu.RLock()
	exp, exists := p.states[state]
	p.mu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(exp) {
		p.mu.Lock()
		delete(p.states, state)
		p.mu.Unlock()
		return false
	}

	// Remove state after validation (single use)
	p.mu.Lock()
	delete(p.states, state)
	p.mu.Unlock()

	return true
}

// ExchangeCode exchanges authorization code for tokens
func (p *OIDCProvider) ExchangeCode(ctx context.Context, code, redirectURI string) (*OIDCTokenResponse, error) {
	if p.discovery == nil {
		return nil, fmt.Errorf("OIDC discovery not available")
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", p.config.ClientID)
	data.Set("client_secret", p.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", p.discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp OIDCTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// GetUserInfo fetches user info using access token
func (p *OIDCProvider) GetUserInfo(ctx context.Context, accessToken string) (*OIDCUserInfo, error) {
	if p.discovery == nil || p.discovery.UserinfoEndpoint == "" {
		return nil, fmt.Errorf("userinfo endpoint not available")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", p.discovery.UserinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var userInfo OIDCUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo: %w", err)
	}

	return &userInfo, nil
}

// DetermineRole determines user role based on claims
func (p *OIDCProvider) DetermineRole(userInfo *OIDCUserInfo) string {
	// Check groups for role mapping
	for _, group := range userInfo.Groups {
		// Check admin roles
		for _, adminRole := range p.config.AdminRoles {
			if strings.EqualFold(group, adminRole) {
				return "admin"
			}
		}
		// Check user roles
		for _, userRole := range p.config.UserRoles {
			if strings.EqualFold(group, userRole) {
				return "user"
			}
		}
		// Check group mappings
		if role, ok := p.config.GroupMappings[group]; ok {
			return role
		}
	}

	// Check roles claim
	for _, role := range userInfo.Roles {
		for _, adminRole := range p.config.AdminRoles {
			if strings.EqualFold(role, adminRole) {
				return "admin"
			}
		}
		for _, userRole := range p.config.UserRoles {
			if strings.EqualFold(role, userRole) {
				return "user"
			}
		}
	}

	return p.config.DefaultRole
}

// generateRandomString generates a cryptographically secure random string
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// --- AuthManager OIDC methods ---

// HandleOIDCLogin initiates OIDC login flow
func (am *AuthManager) HandleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if am.oidcProvider == nil {
		http.Error(w, "OIDC not configured", http.StatusServiceUnavailable)
		return
	}

	// Determine redirect URI
	redirectURI := am.getOIDCRedirectURI(r)

	authURL, _, err := am.oidcProvider.GetAuthorizationURL(redirectURI)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate auth URL: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to OIDC provider
	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleOIDCCallback handles the OIDC callback
func (am *AuthManager) HandleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if am.oidcProvider == nil {
		http.Error(w, "OIDC not configured", http.StatusServiceUnavailable)
		return
	}

	// Check for error from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		http.Error(w, fmt.Sprintf("OIDC error: %s - %s", errParam, errDesc), http.StatusBadRequest)
		return
	}

	// Validate state
	state := r.URL.Query().Get("state")
	if !am.oidcProvider.ValidateState(state) {
		http.Error(w, "Invalid or expired state", http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Exchange code for tokens
	redirectURI := am.getOIDCRedirectURI(r)
	tokenResp, err := am.oidcProvider.ExchangeCode(r.Context(), code, redirectURI)
	if err != nil {
		http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Get user info
	userInfo, err := am.oidcProvider.GetUserInfo(r.Context(), tokenResp.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user info: %v", err), http.StatusInternalServerError)
		return
	}

	// Determine role
	role := am.oidcProvider.DetermineRole(userInfo)

	// Create or update user
	am.mu.Lock()
	user, exists := am.users[userInfo.Email]
	if !exists {
		user = &User{
			ID:          userInfo.Sub,
			Username:    userInfo.Email,
			Email:       userInfo.Email,
			DisplayName: userInfo.Name,
			Role:        role,
			Source:      "oidc",
			CreatedAt:   time.Now(),
		}
		am.users[userInfo.Email] = user
	} else {
		// Update existing user info
		user.DisplayName = userInfo.Name
		user.Role = role
		user.Source = "oidc"
	}
	user.LastLogin = time.Now()

	// Create session
	session := &Session{
		ID:        generateSessionID(),
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		Source:    "oidc",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(am.config.SessionDuration),
	}
	am.sessions[session.ID] = session
	am.mu.Unlock()

	// Set session cookie
	am.setSessionCookie(w, session)

	// Redirect to dashboard
	http.Redirect(w, r, "/", http.StatusFound)
}

// getOIDCRedirectURI constructs the OIDC redirect URI
func (am *AuthManager) getOIDCRedirectURI(r *http.Request) string {
	if am.oidcProvider != nil && am.oidcProvider.config.RedirectURI != "" {
		return am.oidcProvider.config.RedirectURI
	}

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/api/auth/oidc/callback", scheme, r.Host)
}

// setSessionCookie sets the session cookie with security flags
func (am *AuthManager) setSessionCookie(w http.ResponseWriter, session *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     "k13d_session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Set based on environment
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})
}

// HandleOIDCStatus returns OIDC configuration status
func (am *AuthManager) HandleOIDCStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"configured": am.oidcProvider != nil,
		"auth_mode":  am.config.AuthMode,
	}

	if am.oidcProvider != nil {
		status["provider_name"] = am.oidcProvider.config.ProviderName
		status["provider_url"] = am.oidcProvider.config.ProviderURL
		// Don't expose client secret
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}
