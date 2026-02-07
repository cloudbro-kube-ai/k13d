package web

import (
	"fmt"

	"k8s.io/client-go/rest"
)

// ImpersonationTarget defines the K8s user/groups to impersonate for a role
type ImpersonationTarget struct {
	User   string   `yaml:"user" json:"user"`     // K8s username to impersonate
	Groups []string `yaml:"groups" json:"groups"` // K8s groups to impersonate
}

// ImpersonationConfig controls K8s impersonation behavior (Teleport-inspired)
type ImpersonationConfig struct {
	Enabled  bool                           `yaml:"enabled" json:"enabled"`   // Default: false (opt-in)
	Mappings map[string]ImpersonationTarget `yaml:"mappings" json:"mappings"` // role -> impersonation target
}

// DefaultImpersonationConfig returns the default impersonation configuration
func DefaultImpersonationConfig() *ImpersonationConfig {
	return &ImpersonationConfig{
		Enabled: false, // Disabled by default (opt-in)
		Mappings: map[string]ImpersonationTarget{
			"viewer": {
				User:   "k13d-viewer",
				Groups: []string{"k13d:viewers"},
			},
			"user": {
				User:   "k13d-user",
				Groups: []string{"k13d:editors"},
			},
			"admin": {
				User:   "k13d-admin",
				Groups: []string{"k13d:admins"},
			},
		},
	}
}

// GetImpersonatedConfig returns a copy of the base config with impersonation headers set
// based on the user's role. If impersonation is disabled or the role is not mapped,
// returns the original config unchanged.
func GetImpersonatedConfig(baseConfig *rest.Config, role string, cfg *ImpersonationConfig) *rest.Config {
	if cfg == nil || !cfg.Enabled {
		return baseConfig
	}

	target, exists := cfg.Mappings[role]
	if !exists {
		return baseConfig
	}

	// Create a copy to avoid mutating the original
	impersonatedConfig := rest.CopyConfig(baseConfig)
	impersonatedConfig.Impersonate = rest.ImpersonationConfig{
		UserName: target.User,
		Groups:   target.Groups,
	}

	return impersonatedConfig
}

// ValidateImpersonationConfig validates the impersonation configuration
func ValidateImpersonationConfig(cfg *ImpersonationConfig) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	if len(cfg.Mappings) == 0 {
		return fmt.Errorf("impersonation is enabled but no role mappings defined")
	}

	for role, target := range cfg.Mappings {
		if target.User == "" {
			return fmt.Errorf("impersonation target for role %q has no user", role)
		}
	}

	return nil
}
