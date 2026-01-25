package web

import (
	"testing"
)

func TestNewLDAPProvider(t *testing.T) {
	cfg := &LDAPConfig{
		Enabled: true,
		Host:    "ldap.example.com",
		Port:    389,
	}

	provider := NewLDAPProvider(cfg)
	if provider == nil {
		t.Fatal("expected LDAPProvider to be created")
	}

	if !provider.IsEnabled() {
		t.Error("expected provider to be enabled")
	}
}

func TestNewLDAPProvider_Defaults(t *testing.T) {
	cfg := &LDAPConfig{
		Enabled: true,
		Host:    "ldap.example.com",
	}

	provider := NewLDAPProvider(cfg)

	// Check defaults are set
	if cfg.Port != 389 {
		t.Errorf("expected default port 389, got %d", cfg.Port)
	}

	if cfg.UserSearchFilter != "(uid=%s)" {
		t.Errorf("expected default user search filter, got %s", cfg.UserSearchFilter)
	}

	if cfg.UsernameAttr != "uid" {
		t.Errorf("expected default username attr 'uid', got %s", cfg.UsernameAttr)
	}

	if cfg.EmailAttr != "mail" {
		t.Errorf("expected default email attr 'mail', got %s", cfg.EmailAttr)
	}

	if cfg.DisplayNameAttr != "cn" {
		t.Errorf("expected default display name attr 'cn', got %s", cfg.DisplayNameAttr)
	}

	if !provider.IsEnabled() {
		t.Error("expected provider to be enabled")
	}
}

func TestNewLDAPProvider_TLSDefaults(t *testing.T) {
	cfg := &LDAPConfig{
		Enabled: true,
		Host:    "ldap.example.com",
		UseTLS:  true,
	}

	NewLDAPProvider(cfg)

	if cfg.Port != 636 {
		t.Errorf("expected TLS default port 636, got %d", cfg.Port)
	}
}

func TestLDAPProvider_IsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config *LDAPConfig
		want   bool
	}{
		{
			name: "enabled",
			config: &LDAPConfig{
				Enabled: true,
				Host:    "ldap.example.com",
			},
			want: true,
		},
		{
			name: "disabled",
			config: &LDAPConfig{
				Enabled: false,
				Host:    "ldap.example.com",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewLDAPProvider(tt.config)
			if got := provider.IsEnabled(); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLDAPProvider_GetConfig(t *testing.T) {
	cfg := &LDAPConfig{
		Enabled:      true,
		Host:         "ldap.example.com",
		Port:         389,
		BaseDN:       "dc=example,dc=com",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
		AdminGroups:  []string{"admins"},
		UserGroups:   []string{"users"},
	}

	provider := NewLDAPProvider(cfg)
	retrievedConfig := provider.GetConfig()

	if retrievedConfig.Host != cfg.Host {
		t.Errorf("expected host %s, got %s", cfg.Host, retrievedConfig.Host)
	}

	if retrievedConfig.Port != cfg.Port {
		t.Errorf("expected port %d, got %d", cfg.Port, retrievedConfig.Port)
	}

	if retrievedConfig.BaseDN != cfg.BaseDN {
		t.Errorf("expected base DN %s, got %s", cfg.BaseDN, retrievedConfig.BaseDN)
	}

	// Password should be empty in returned config
	if retrievedConfig.BindPassword != "" {
		t.Error("expected bind password to be empty in returned config")
	}
}

func TestLDAPProvider_DetermineRole(t *testing.T) {
	cfg := &LDAPConfig{
		Enabled:      true,
		Host:         "ldap.example.com",
		AdminGroups:  []string{"k8s-admins", "cluster-admins"},
		UserGroups:   []string{"k8s-users", "developers"},
		ViewerGroups: []string{"k8s-viewers", "readonly"},
	}

	provider := NewLDAPProvider(cfg)

	tests := []struct {
		name     string
		groups   []string
		wantRole string
	}{
		{"admin group", []string{"k8s-admins", "other"}, "admin"},
		{"admin group case insensitive", []string{"K8S-ADMINS"}, "admin"},
		{"user group", []string{"developers", "other"}, "user"},
		{"viewer group", []string{"readonly"}, "viewer"},
		{"admin takes priority", []string{"k8s-admins", "developers", "readonly"}, "admin"},
		{"user takes priority over viewer", []string{"developers", "readonly"}, "user"},
		{"no matching group defaults to viewer", []string{"unknown-group"}, "viewer"},
		{"empty groups defaults to viewer", []string{}, "viewer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.determineRole(tt.groups)
			if got != tt.wantRole {
				t.Errorf("determineRole(%v) = %s, want %s", tt.groups, got, tt.wantRole)
			}
		})
	}
}

func TestLDAPProvider_TestConnection_NotEnabled(t *testing.T) {
	cfg := &LDAPConfig{
		Enabled: false,
		Host:    "ldap.example.com",
	}

	provider := NewLDAPProvider(cfg)
	err := provider.TestConnection()

	if err == nil {
		t.Error("expected error when LDAP is not enabled")
	}
}

func TestLDAPProvider_Authenticate_NotEnabled(t *testing.T) {
	cfg := &LDAPConfig{
		Enabled: false,
		Host:    "ldap.example.com",
	}

	provider := NewLDAPProvider(cfg)
	_, err := provider.Authenticate("user", "pass")

	if err == nil {
		t.Error("expected error when LDAP is not enabled")
	}
}

func TestAuthManager_IsLDAPEnabled(t *testing.T) {
	tests := []struct {
		name       string
		authConfig *AuthConfig
		want       bool
	}{
		{
			name: "LDAP enabled",
			authConfig: &AuthConfig{
				Enabled: true,
				Quiet:   true,
				LDAP: &LDAPConfig{
					Enabled: true,
					Host:    "ldap.example.com",
				},
			},
			want: true,
		},
		{
			name: "LDAP disabled",
			authConfig: &AuthConfig{
				Enabled: true,
				Quiet:   true,
				LDAP: &LDAPConfig{
					Enabled: false,
					Host:    "ldap.example.com",
				},
			},
			want: false,
		},
		{
			name: "LDAP config nil",
			authConfig: &AuthConfig{
				Enabled: true,
				Quiet:   true,
				LDAP:    nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := NewAuthManager(tt.authConfig)
			if got := am.IsLDAPEnabled(); got != tt.want {
				t.Errorf("IsLDAPEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthManager_GetLDAPConfig(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled: true,
		Quiet:   true,
		LDAP: &LDAPConfig{
			Enabled: true,
			Host:    "ldap.example.com",
			Port:    389,
		},
	}

	am := NewAuthManager(authConfig)
	ldapConfig := am.GetLDAPConfig()

	if ldapConfig == nil {
		t.Fatal("expected LDAP config to be returned")
	}

	if ldapConfig.Host != "ldap.example.com" {
		t.Errorf("expected host 'ldap.example.com', got %s", ldapConfig.Host)
	}
}

func TestAuthManager_GetLDAPConfig_Nil(t *testing.T) {
	authConfig := &AuthConfig{
		Enabled: true,
		Quiet:   true,
		LDAP:    nil,
	}

	am := NewAuthManager(authConfig)
	ldapConfig := am.GetLDAPConfig()

	if ldapConfig != nil {
		t.Error("expected nil LDAP config when LDAP not configured")
	}
}
