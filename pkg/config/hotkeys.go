package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// HotkeyConfig represents a custom hotkey binding
type HotkeyConfig struct {
	ShortCut    string   `yaml:"shortCut"`    // Key combination (e.g., "Shift-L", "Ctrl-K")
	Description string   `yaml:"description"` // Human-readable description
	Scopes      []string `yaml:"scopes"`      // Resource types this applies to (e.g., ["pods", "deployments"])
	Command     string   `yaml:"command"`     // External command to run
	Args        []string `yaml:"args"`        // Command arguments (supports $NAMESPACE, $NAME, $CONTEXT)
	Dangerous   bool     `yaml:"dangerous"`   // Requires confirmation before execution
}

// HotkeysFile represents the hotkeys.yaml file structure
type HotkeysFile struct {
	Hotkeys map[string]HotkeyConfig `yaml:"hotkeys"`
}

// DefaultHotkeys returns the default hotkey configuration
func DefaultHotkeys() *HotkeysFile {
	return &HotkeysFile{
		Hotkeys: map[string]HotkeyConfig{
			"stern-logs": {
				ShortCut:    "Shift-L",
				Description: "Stern multi-pod logs",
				Scopes:      []string{"pods", "deployments"},
				Command:     "stern",
				Args:        []string{"-n", "$NAMESPACE", "$NAME"},
				Dangerous:   false,
			},
			"port-forward-8080": {
				ShortCut:    "Ctrl-P",
				Description: "Port forward to 8080",
				Scopes:      []string{"pods", "services"},
				Command:     "kubectl",
				Args:        []string{"port-forward", "-n", "$NAMESPACE", "$NAME", "8080:8080"},
				Dangerous:   false,
			},
		},
	}
}

// LoadHotkeys loads hotkey configuration from file
func LoadHotkeys() (*HotkeysFile, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return DefaultHotkeys(), nil
	}

	hotkeyPath := filepath.Join(configDir, "hotkeys.yaml")
	data, err := os.ReadFile(hotkeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultHotkeys(), nil
		}
		return nil, err
	}

	var hotkeys HotkeysFile
	if err := yaml.Unmarshal(data, &hotkeys); err != nil {
		return DefaultHotkeys(), nil
	}

	return &hotkeys, nil
}

// SaveHotkeys saves hotkey configuration to file
func SaveHotkeys(hotkeys *HotkeysFile) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	hotkeyPath := filepath.Join(configDir, "hotkeys.yaml")
	data, err := yaml.Marshal(hotkeys)
	if err != nil {
		return err
	}

	return os.WriteFile(hotkeyPath, data, 0644)
}

// GetHotkeysForScope returns hotkeys applicable to a given resource type
func (h *HotkeysFile) GetHotkeysForScope(resourceType string) map[string]HotkeyConfig {
	result := make(map[string]HotkeyConfig)
	for name, hk := range h.Hotkeys {
		for _, scope := range hk.Scopes {
			if scope == resourceType || scope == "*" {
				result[name] = hk
				break
			}
		}
	}
	return result
}

// ExpandArgs expands variables in command arguments
func (h *HotkeyConfig) ExpandArgs(namespace, name, context string) []string {
	expanded := make([]string, len(h.Args))
	for i, arg := range h.Args {
		switch arg {
		case "$NAMESPACE":
			expanded[i] = namespace
		case "$NAME":
			expanded[i] = name
		case "$CONTEXT":
			expanded[i] = context
		default:
			expanded[i] = arg
		}
	}
	return expanded
}

// ParseShortcut parses a shortcut string into modifiers and key
// Returns: hasCtrl, hasShift, hasAlt, key rune
func ParseShortcut(shortcut string) (bool, bool, bool, string) {
	hasCtrl := false
	hasShift := false
	hasAlt := false
	key := shortcut

	// Parse modifiers
	for {
		if len(key) > 5 && key[:5] == "Ctrl-" {
			hasCtrl = true
			key = key[5:]
		} else if len(key) > 6 && key[:6] == "Shift-" {
			hasShift = true
			key = key[6:]
		} else if len(key) > 4 && key[:4] == "Alt-" {
			hasAlt = true
			key = key[4:]
		} else {
			break
		}
	}

	return hasCtrl, hasShift, hasAlt, key
}
