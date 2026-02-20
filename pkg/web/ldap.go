package web

import (
	"crypto/tls"
	"fmt"
	"strings"
	"sync"

	"github.com/go-ldap/ldap/v3"
)

// LDAPConfig holds LDAP configuration
type LDAPConfig struct {
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	Host              string   `yaml:"host" json:"host"`
	Port              int      `yaml:"port" json:"port"`
	UseTLS            bool     `yaml:"use_tls" json:"use_tls"`
	StartTLS          bool     `yaml:"start_tls" json:"start_tls"`
	InsecureSkipTLS   bool     `yaml:"insecure_skip_tls" json:"insecure_skip_tls"`
	BindDN            string   `yaml:"bind_dn" json:"bind_dn"`
	BindPassword      string   `yaml:"bind_password" json:"-"`
	BaseDN            string   `yaml:"base_dn" json:"base_dn"`
	UserSearchFilter  string   `yaml:"user_search_filter" json:"user_search_filter"` // e.g., "(uid=%s)" or "(sAMAccountName=%s)"
	UserSearchBase    string   `yaml:"user_search_base" json:"user_search_base"`
	GroupSearchBase   string   `yaml:"group_search_base" json:"group_search_base"`
	GroupSearchFilter string   `yaml:"group_search_filter" json:"group_search_filter"` // e.g., "(member=%s)"
	AdminGroups       []string `yaml:"admin_groups" json:"admin_groups"`               // Groups that grant admin role
	UserGroups        []string `yaml:"user_groups" json:"user_groups"`                 // Groups that grant user role
	ViewerGroups      []string `yaml:"viewer_groups" json:"viewer_groups"`             // Groups that grant viewer role
	UsernameAttr      string   `yaml:"username_attr" json:"username_attr"`             // e.g., "uid" or "sAMAccountName"
	EmailAttr         string   `yaml:"email_attr" json:"email_attr"`                   // e.g., "mail"
	DisplayNameAttr   string   `yaml:"display_name_attr" json:"display_name_attr"`     // e.g., "cn" or "displayName"
}

// LDAPProvider handles LDAP authentication
type LDAPProvider struct {
	config *LDAPConfig
	mu     sync.RWMutex
}

// LDAPUser represents a user retrieved from LDAP
type LDAPUser struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	DN          string   `json:"dn"`
	Groups      []string `json:"groups"`
	Role        string   `json:"role"`
}

// NewLDAPProvider creates a new LDAP provider
func NewLDAPProvider(cfg *LDAPConfig) *LDAPProvider {
	// Set defaults
	if cfg.UserSearchFilter == "" {
		cfg.UserSearchFilter = "(uid=%s)"
	}
	if cfg.UsernameAttr == "" {
		cfg.UsernameAttr = "uid"
	}
	if cfg.EmailAttr == "" {
		cfg.EmailAttr = "mail"
	}
	if cfg.DisplayNameAttr == "" {
		cfg.DisplayNameAttr = "cn"
	}
	if cfg.Port == 0 {
		if cfg.UseTLS {
			cfg.Port = 636
		} else {
			cfg.Port = 389
		}
	}

	return &LDAPProvider{
		config: cfg,
	}
}

// connect establishes a connection to the LDAP server
func (p *LDAPProvider) connect() (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	address := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)

	if p.config.UseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: p.config.InsecureSkipTLS,
		}
		conn, err = ldap.DialURL(fmt.Sprintf("ldaps://%s", address), ldap.DialWithTLSConfig(tlsConfig))
	} else {
		conn, err = ldap.DialURL(fmt.Sprintf("ldap://%s", address))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}

	// Start TLS if requested (for non-TLS connections)
	if p.config.StartTLS && !p.config.UseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: p.config.InsecureSkipTLS,
		}
		if err := conn.StartTLS(tlsConfig); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	return conn, nil
}

// Authenticate validates user credentials against LDAP
func (p *LDAPProvider) Authenticate(username, password string) (*LDAPUser, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.config.Enabled {
		return nil, fmt.Errorf("LDAP authentication is disabled")
	}

	// Connect to LDAP
	conn, err := p.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Bind with service account to search for user
	if p.config.BindDN != "" {
		if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			return nil, fmt.Errorf("failed to bind with service account: %w", err)
		}
	}

	// Search for user
	searchBase := p.config.UserSearchBase
	if searchBase == "" {
		searchBase = p.config.BaseDN
	}

	searchFilter := fmt.Sprintf(p.config.UserSearchFilter, ldap.EscapeFilter(username))
	searchReq := ldap.NewSearchRequest(
		searchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, 0, false,
		searchFilter,
		[]string{p.config.UsernameAttr, p.config.EmailAttr, p.config.DisplayNameAttr, "dn"},
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to search for user: %w", err)
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	if len(result.Entries) > 1 {
		return nil, fmt.Errorf("multiple users found")
	}

	userEntry := result.Entries[0]
	userDN := userEntry.DN

	// Bind as the user to validate password
	if err := conn.Bind(userDN, password); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Re-bind as service account to search for groups
	if p.config.BindDN != "" {
		if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			return nil, fmt.Errorf("failed to re-bind with service account: %w", err)
		}
	}

	// Get user groups
	groups, err := p.getUserGroups(conn, userDN)
	if err != nil {
		// Log error but don't fail authentication
		groups = []string{}
	}

	// Determine role based on group membership
	role := p.determineRole(groups)

	ldapUser := &LDAPUser{
		Username:    userEntry.GetAttributeValue(p.config.UsernameAttr),
		Email:       userEntry.GetAttributeValue(p.config.EmailAttr),
		DisplayName: userEntry.GetAttributeValue(p.config.DisplayNameAttr),
		DN:          userDN,
		Groups:      groups,
		Role:        role,
	}

	return ldapUser, nil
}

// getUserGroups retrieves the groups a user belongs to
func (p *LDAPProvider) getUserGroups(conn *ldap.Conn, userDN string) ([]string, error) {
	if p.config.GroupSearchBase == "" {
		return []string{}, nil
	}

	searchFilter := p.config.GroupSearchFilter
	if searchFilter == "" {
		searchFilter = "(member=%s)"
	}
	searchFilter = fmt.Sprintf(searchFilter, ldap.EscapeFilter(userDN))

	searchReq := ldap.NewSearchRequest(
		p.config.GroupSearchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		searchFilter,
		[]string{"cn"},
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to search for groups: %w", err)
	}

	groups := make([]string, 0, len(result.Entries))
	for _, entry := range result.Entries {
		cn := entry.GetAttributeValue("cn")
		if cn != "" {
			groups = append(groups, cn)
		}
	}

	return groups, nil
}

// determineRole determines the user's role based on group membership
func (p *LDAPProvider) determineRole(groups []string) string {
	groupSet := make(map[string]bool)
	for _, g := range groups {
		groupSet[strings.ToLower(g)] = true
	}

	// Check admin groups first (highest priority)
	for _, adminGroup := range p.config.AdminGroups {
		if groupSet[strings.ToLower(adminGroup)] {
			return "admin"
		}
	}

	// Check user groups
	for _, userGroup := range p.config.UserGroups {
		if groupSet[strings.ToLower(userGroup)] {
			return "user"
		}
	}

	// Check viewer groups
	for _, viewerGroup := range p.config.ViewerGroups {
		if groupSet[strings.ToLower(viewerGroup)] {
			return "viewer"
		}
	}

	// Default to viewer if no specific group matched but user authenticated
	return "viewer"
}

// TestConnection tests the LDAP connection
func (p *LDAPProvider) TestConnection() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.config.Enabled {
		return fmt.Errorf("LDAP is not enabled")
	}

	conn, err := p.connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Try to bind with service account
	if p.config.BindDN != "" {
		if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
			return fmt.Errorf("failed to bind with service account: %w", err)
		}
	}

	return nil
}

// IsEnabled returns whether LDAP is enabled
func (p *LDAPProvider) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config != nil && p.config.Enabled
}

// GetConfig returns a copy of the LDAP config (without sensitive data)
func (p *LDAPProvider) GetConfig() *LDAPConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.config == nil {
		return nil
	}

	// Return copy without password
	return &LDAPConfig{
		Enabled:           p.config.Enabled,
		Host:              p.config.Host,
		Port:              p.config.Port,
		UseTLS:            p.config.UseTLS,
		StartTLS:          p.config.StartTLS,
		InsecureSkipTLS:   p.config.InsecureSkipTLS,
		BindDN:            p.config.BindDN,
		BaseDN:            p.config.BaseDN,
		UserSearchFilter:  p.config.UserSearchFilter,
		UserSearchBase:    p.config.UserSearchBase,
		GroupSearchBase:   p.config.GroupSearchBase,
		GroupSearchFilter: p.config.GroupSearchFilter,
		AdminGroups:       p.config.AdminGroups,
		UserGroups:        p.config.UserGroups,
		ViewerGroups:      p.config.ViewerGroups,
		UsernameAttr:      p.config.UsernameAttr,
		EmailAttr:         p.config.EmailAttr,
		DisplayNameAttr:   p.config.DisplayNameAttr,
	}
}
