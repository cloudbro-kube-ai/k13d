package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ViewConfig represents per-resource view configuration (k9s views.yaml pattern)
type ViewConfig struct {
	Views map[string]ResourceViewConfig `yaml:"views"` // resource name -> view config
}

// ResourceViewConfig holds per-resource display settings
type ResourceViewConfig struct {
	SortColumn    string `yaml:"sortColumn"`    // Default sort column name (e.g., "AGE", "NAME")
	SortAscending bool   `yaml:"sortAscending"` // Sort direction
}

// DefaultViews returns the default view configuration
func DefaultViews() *ViewConfig {
	return &ViewConfig{
		Views: map[string]ResourceViewConfig{},
	}
}

// LoadViews loads view configuration from file
func LoadViews() (*ViewConfig, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return DefaultViews(), nil
	}

	viewsPath := filepath.Join(configDir, "views.yaml")
	data, err := os.ReadFile(viewsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultViews(), nil
		}
		return nil, err
	}

	var views ViewConfig
	if err := yaml.Unmarshal(data, &views); err != nil {
		return DefaultViews(), nil
	}

	if views.Views == nil {
		views.Views = map[string]ResourceViewConfig{}
	}

	return &views, nil
}

// SaveViews saves view configuration to file
func SaveViews(views *ViewConfig) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	viewsPath := filepath.Join(configDir, "views.yaml")
	data, err := yaml.Marshal(views)
	if err != nil {
		return err
	}

	return os.WriteFile(viewsPath, data, 0644)
}

// GetViewConfig returns view config for a resource, or nil if not configured
func (v *ViewConfig) GetViewConfig(resource string) *ResourceViewConfig {
	if v == nil || v.Views == nil {
		return nil
	}
	if cfg, ok := v.Views[resource]; ok {
		return &cfg
	}
	return nil
}
