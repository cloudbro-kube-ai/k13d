package cli

import (
	"testing"
)

func TestEnvDefault(t *testing.T) {
	tests := []struct {
		name       string
		envKey     string
		envVal     string
		defaultVal string
		want       string
	}{
		{"env set", "TEST_CLI_ENV_1", "custom", "default", "custom"},
		{"env empty", "TEST_CLI_ENV_2", "", "default", "default"},
		{"env not set", "TEST_CLI_ENV_NONEXISTENT_XYZ", "", "fallback", "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			got := EnvDefault(tt.envKey, tt.defaultVal)
			if got != tt.want {
				t.Errorf("EnvDefault(%q, %q) = %q, want %q", tt.envKey, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestEnvBoolDefault(t *testing.T) {
	tests := []struct {
		name       string
		envKey     string
		envVal     string
		defaultVal bool
		want       bool
	}{
		{"true string", "TEST_CLI_BOOL_1", "true", false, true},
		{"false string", "TEST_CLI_BOOL_2", "false", true, false},
		{"1", "TEST_CLI_BOOL_3", "1", false, true},
		{"0", "TEST_CLI_BOOL_4", "0", true, false},
		{"invalid", "TEST_CLI_BOOL_5", "notabool", true, true},
		{"empty uses default", "TEST_CLI_BOOL_6", "", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			got := EnvBoolDefault(tt.envKey, tt.defaultVal)
			if got != tt.want {
				t.Errorf("EnvBoolDefault(%q, %v) = %v, want %v", tt.envKey, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestEnvIntDefault(t *testing.T) {
	tests := []struct {
		name       string
		envKey     string
		envVal     string
		defaultVal int
		want       int
	}{
		{"valid int", "TEST_CLI_INT_1", "8080", 3000, 8080},
		{"zero", "TEST_CLI_INT_2", "0", 3000, 0},
		{"negative", "TEST_CLI_INT_3", "-1", 3000, -1},
		{"invalid", "TEST_CLI_INT_4", "notanint", 3000, 3000},
		{"empty uses default", "TEST_CLI_INT_5", "", 9090, 9090},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}
			got := EnvIntDefault(tt.envKey, tt.defaultVal)
			if got != tt.want {
				t.Errorf("EnvIntDefault(%q, %d) = %d, want %d", tt.envKey, tt.defaultVal, got, tt.want)
			}
		})
	}
}
