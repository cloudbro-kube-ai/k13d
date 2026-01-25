package config

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PluginConfig represents a plugin definition
type PluginConfig struct {
	ShortCut    string   `yaml:"shortCut"`    // Key to trigger (e.g., "p")
	Description string   `yaml:"description"` // Plugin description
	Scopes      []string `yaml:"scopes"`      // Resource types this applies to
	Command     string   `yaml:"command"`     // Command to execute
	Args        []string `yaml:"args"`        // Command arguments
	Background  bool     `yaml:"background"`  // Run in background
	Confirm     bool     `yaml:"confirm"`     // Require confirmation
}

// PluginsFile represents the plugins.yaml file structure
type PluginsFile struct {
	Plugins map[string]PluginConfig `yaml:"plugins"`
}

// DefaultPlugins returns the default plugin configuration
func DefaultPlugins() *PluginsFile {
	return &PluginsFile{
		Plugins: map[string]PluginConfig{
			"dive": {
				ShortCut:    "Ctrl-D",
				Description: "Dive into container image layers",
				Scopes:      []string{"pods"},
				Command:     "dive",
				Args:        []string{"$IMAGE"},
				Background:  false,
				Confirm:     false,
			},
			"debug": {
				ShortCut:    "Shift-D",
				Description: "Debug pod with ephemeral container",
				Scopes:      []string{"pods"},
				Command:     "kubectl",
				Args:        []string{"debug", "-n", "$NAMESPACE", "$NAME", "-it", "--image=busybox"},
				Background:  false,
				Confirm:     true,
			},
		},
	}
}

// LoadPlugins loads plugin configuration from file
func LoadPlugins() (*PluginsFile, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return DefaultPlugins(), nil
	}

	pluginPath := filepath.Join(configDir, "plugins.yaml")
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultPlugins(), nil
		}
		return nil, err
	}

	var plugins PluginsFile
	if err := yaml.Unmarshal(data, &plugins); err != nil {
		return DefaultPlugins(), nil
	}

	return &plugins, nil
}

// SavePlugins saves plugin configuration to file
func SavePlugins(plugins *PluginsFile) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	pluginPath := filepath.Join(configDir, "plugins.yaml")
	data, err := yaml.Marshal(plugins)
	if err != nil {
		return err
	}

	return os.WriteFile(pluginPath, data, 0644)
}

// GetPluginsForScope returns plugins applicable to a given resource type
func (p *PluginsFile) GetPluginsForScope(resourceType string) map[string]PluginConfig {
	result := make(map[string]PluginConfig)
	for name, plugin := range p.Plugins {
		for _, scope := range plugin.Scopes {
			if scope == resourceType || scope == "*" {
				result[name] = plugin
				break
			}
		}
	}
	return result
}

// PluginContext holds the runtime context for plugin execution
type PluginContext struct {
	Namespace   string
	Name        string
	Context     string
	Image       string
	Labels      map[string]string
	Annotations map[string]string
}

// ExpandArgs expands variables in plugin arguments
func (p *PluginConfig) ExpandArgs(ctx *PluginContext) []string {
	expanded := make([]string, len(p.Args))
	for i, arg := range p.Args {
		switch arg {
		case "$NAMESPACE":
			expanded[i] = ctx.Namespace
		case "$NAME":
			expanded[i] = ctx.Name
		case "$CONTEXT":
			expanded[i] = ctx.Context
		case "$IMAGE":
			expanded[i] = ctx.Image
		default:
			// Handle $LABELS.key and $ANNOTATIONS.key
			if strings.HasPrefix(arg, "$LABELS.") {
				key := arg[8:]
				if val, ok := ctx.Labels[key]; ok {
					expanded[i] = val
				} else {
					expanded[i] = ""
				}
			} else if strings.HasPrefix(arg, "$ANNOTATIONS.") {
				key := arg[13:]
				if val, ok := ctx.Annotations[key]; ok {
					expanded[i] = val
				} else {
					expanded[i] = ""
				}
			} else {
				expanded[i] = arg
			}
		}
	}
	return expanded
}

// Execute runs the plugin command
func (p *PluginConfig) Execute(ctx context.Context, pCtx *PluginContext) error {
	args := p.ExpandArgs(pCtx)

	// Check if command exists
	path, err := exec.LookPath(p.Command)
	if err != nil {
		return fmt.Errorf("command %q not found: %w", p.Command, err)
	}

	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if p.Background {
		return cmd.Start()
	}
	return cmd.Run()
}

// Validate checks if the plugin configuration is valid
func (p *PluginConfig) Validate() error {
	if p.ShortCut == "" {
		return fmt.Errorf("shortCut is required")
	}
	if p.Command == "" {
		return fmt.Errorf("command is required")
	}
	if len(p.Scopes) == 0 {
		return fmt.Errorf("at least one scope is required")
	}
	return nil
}

// GetAvailableVariables returns the list of supported plugin variables
func GetAvailableVariables() []string {
	return []string{
		"$NAMESPACE - Resource namespace",
		"$NAME - Resource name",
		"$CONTEXT - Current Kubernetes context",
		"$IMAGE - Container image (for pods)",
		"$LABELS.key - Label value by key",
		"$ANNOTATIONS.key - Annotation value by key",
	}
}
