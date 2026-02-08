package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AliasConfig represents configurable command aliases (k9s aliases.yaml pattern)
type AliasConfig struct {
	Aliases map[string]string `yaml:"aliases"` // alias -> full resource name (e.g., "pp" -> "pods")
}

// DefaultAliases returns the default alias configuration (empty - built-in aliases are in app.go)
func DefaultAliases() *AliasConfig {
	return &AliasConfig{
		Aliases: map[string]string{},
	}
}

// LoadAliases loads alias configuration from file
func LoadAliases() (*AliasConfig, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return DefaultAliases(), nil
	}

	aliasPath := filepath.Join(configDir, "aliases.yaml")
	data, err := os.ReadFile(aliasPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultAliases(), nil
		}
		return nil, err
	}

	var aliases AliasConfig
	if err := yaml.Unmarshal(data, &aliases); err != nil {
		return DefaultAliases(), nil
	}

	if aliases.Aliases == nil {
		aliases.Aliases = map[string]string{}
	}

	return &aliases, nil
}

// SaveAliases saves alias configuration to file
func SaveAliases(aliases *AliasConfig) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	aliasPath := filepath.Join(configDir, "aliases.yaml")
	data, err := yaml.Marshal(aliases)
	if err != nil {
		return err
	}

	return os.WriteFile(aliasPath, data, 0644)
}

// Resolve returns the full resource name for an alias, or the original input if not an alias
func (a *AliasConfig) Resolve(input string) string {
	if a == nil || a.Aliases == nil {
		return input
	}
	if resolved, ok := a.Aliases[input]; ok {
		return resolved
	}
	return input
}

// GetAll returns all configured aliases
func (a *AliasConfig) GetAll() map[string]string {
	if a == nil || a.Aliases == nil {
		return map[string]string{}
	}
	return a.Aliases
}
