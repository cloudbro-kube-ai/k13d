package web

import (
	"testing"

	"k8s.io/client-go/rest"
)

func TestImpersonation_DisabledFallback(t *testing.T) {
	baseConfig := &rest.Config{
		Host: "https://k8s.example.com",
	}

	// When disabled, should return original config
	cfg := &ImpersonationConfig{Enabled: false}
	result := GetImpersonatedConfig(baseConfig, "admin", cfg)

	if result != baseConfig {
		t.Error("Expected original config when impersonation is disabled")
	}

	// Nil config should also return original
	result = GetImpersonatedConfig(baseConfig, "admin", nil)
	if result != baseConfig {
		t.Error("Expected original config when config is nil")
	}
}

func TestImpersonation_RoleMapping(t *testing.T) {
	baseConfig := &rest.Config{
		Host: "https://k8s.example.com",
	}

	cfg := DefaultImpersonationConfig()
	cfg.Enabled = true

	tests := []struct {
		role           string
		expectedUser   string
		expectedGroups []string
	}{
		{"viewer", "k13d-viewer", []string{"k13d:viewers"}},
		{"user", "k13d-user", []string{"k13d:editors"}},
		{"admin", "k13d-admin", []string{"k13d:admins"}},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := GetImpersonatedConfig(baseConfig, tt.role, cfg)

			if result == baseConfig {
				t.Error("Expected a copy, not the original config")
			}

			if result.Impersonate.UserName != tt.expectedUser {
				t.Errorf("Impersonate user: got %q, want %q",
					result.Impersonate.UserName, tt.expectedUser)
			}

			if len(result.Impersonate.Groups) != len(tt.expectedGroups) {
				t.Errorf("Impersonate groups: got %v, want %v",
					result.Impersonate.Groups, tt.expectedGroups)
			} else {
				for i, g := range tt.expectedGroups {
					if result.Impersonate.Groups[i] != g {
						t.Errorf("Group[%d]: got %q, want %q",
							i, result.Impersonate.Groups[i], g)
					}
				}
			}

			// Original config should not be modified
			if baseConfig.Impersonate.UserName != "" {
				t.Error("Original config should not be modified")
			}
		})
	}
}

func TestImpersonation_UnknownRole(t *testing.T) {
	baseConfig := &rest.Config{
		Host: "https://k8s.example.com",
	}

	cfg := DefaultImpersonationConfig()
	cfg.Enabled = true

	result := GetImpersonatedConfig(baseConfig, "unknown-role", cfg)
	if result != baseConfig {
		t.Error("Expected original config for unknown role")
	}
}

func TestImpersonation_Validation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ImpersonationConfig
		wantErr bool
	}{
		{"nil config", nil, false},
		{"disabled", &ImpersonationConfig{Enabled: false}, false},
		{"valid enabled", DefaultImpersonationConfig(), false},
		{"enabled no mappings", &ImpersonationConfig{Enabled: true, Mappings: map[string]ImpersonationTarget{}}, true},
		{"empty user", &ImpersonationConfig{
			Enabled:  true,
			Mappings: map[string]ImpersonationTarget{"viewer": {User: "", Groups: []string{"g"}}},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cfg != nil && !tt.cfg.Enabled && tt.name == "valid enabled" {
				tt.cfg.Enabled = true
			}
			err := ValidateImpersonationConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateImpersonationConfig: err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestImpersonation_PreservesHostConfig(t *testing.T) {
	baseConfig := &rest.Config{
		Host:        "https://k8s.example.com",
		BearerToken: "original-token",
	}

	cfg := DefaultImpersonationConfig()
	cfg.Enabled = true

	result := GetImpersonatedConfig(baseConfig, "admin", cfg)

	if result.Host != baseConfig.Host {
		t.Errorf("Host should be preserved: got %q, want %q", result.Host, baseConfig.Host)
	}
	if result.BearerToken != baseConfig.BearerToken {
		t.Errorf("BearerToken should be preserved: got %q, want %q",
			result.BearerToken, baseConfig.BearerToken)
	}
}
