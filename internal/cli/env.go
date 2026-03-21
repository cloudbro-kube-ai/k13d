// Package cli contains shared startup logic for k13d entry points.
package cli

import (
	"os"
	"strconv"
)

// EnvDefault returns the environment variable value if set, otherwise the default.
func EnvDefault(envKey, defaultVal string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return defaultVal
}

// EnvBoolDefault returns the environment variable parsed as bool, or the default.
func EnvBoolDefault(envKey string, defaultVal bool) bool {
	v := os.Getenv(envKey)
	if v == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return parsed
}

// EnvIntDefault returns the environment variable parsed as int, or the default.
func EnvIntDefault(envKey string, defaultVal int) int {
	v := os.Getenv(envKey)
	if v == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return parsed
}
