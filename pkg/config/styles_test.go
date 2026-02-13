package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		// Exact match (no wildcard)
		{"production", "production", true},
		{"production", "staging", false},

		// Trailing wildcard
		{"prod-*", "prod-us-east", true},
		{"prod-*", "prod-", true},
		{"prod-*", "staging", false},
		{"prod-*", "prod", false},

		// Leading wildcard
		{"*-prod", "us-east-prod", true},
		{"*-prod", "-prod", true},
		{"*-prod", "prod", false},

		// Middle wildcard
		{"prod-*-east", "prod-us-east", true},
		{"prod-*-east", "prod--east", true},
		{"prod-*-east", "prod-east", false},

		// Multiple wildcards
		{"*-prod-*", "us-prod-east", true},
		{"*-prod-*", "eu-prod-west", true},
		{"*-prod-*", "prod-east", false},

		// Star-only matches everything
		{"*", "anything", true},
		{"*", "", true},

		// Empty pattern
		{"", "", true},
		{"", "notempty", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_vs_"+tt.name, func(t *testing.T) {
			got := matchGlob(tt.pattern, tt.name)
			if got != tt.want {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
			}
		})
	}
}

func TestGetSkinForContext_ExactMatch(t *testing.T) {
	cfg := &ContextSkinConfig{
		Mappings: map[string]string{
			"production":  "production",
			"staging":     "staging",
			"development": "development",
		},
	}

	if got := cfg.GetSkinForContext("production"); got != "production" {
		t.Errorf("GetSkinForContext('production') = %q, want %q", got, "production")
	}
	if got := cfg.GetSkinForContext("staging"); got != "staging" {
		t.Errorf("GetSkinForContext('staging') = %q, want %q", got, "staging")
	}
}

func TestGetSkinForContext_GlobMatch(t *testing.T) {
	cfg := &ContextSkinConfig{
		Mappings: map[string]string{
			"prod-*": "production",
			"stg-*":  "staging",
			"dev-*":  "development",
		},
	}

	if got := cfg.GetSkinForContext("prod-us-east"); got != "production" {
		t.Errorf("GetSkinForContext('prod-us-east') = %q, want %q", got, "production")
	}
	if got := cfg.GetSkinForContext("stg-eu-west"); got != "staging" {
		t.Errorf("GetSkinForContext('stg-eu-west') = %q, want %q", got, "staging")
	}
}

func TestGetSkinForContext_NoMatch(t *testing.T) {
	cfg := &ContextSkinConfig{
		Mappings: map[string]string{
			"production": "production",
		},
	}

	if got := cfg.GetSkinForContext("unknown-context"); got != "default" {
		t.Errorf("GetSkinForContext('unknown-context') = %q, want %q", got, "default")
	}
}

func TestGetSkinForContext_NilReceiver(t *testing.T) {
	var cfg *ContextSkinConfig
	if got := cfg.GetSkinForContext("anything"); got != "default" {
		t.Errorf("nil.GetSkinForContext() = %q, want %q", got, "default")
	}
}

func TestGetSkinForContext_NilMappings(t *testing.T) {
	cfg := &ContextSkinConfig{Mappings: nil}
	if got := cfg.GetSkinForContext("anything"); got != "default" {
		t.Errorf("GetSkinForContext() with nil mappings = %q, want %q", got, "default")
	}
}

func TestGetSkinForContext_ExactMatchTakesPrecedence(t *testing.T) {
	cfg := &ContextSkinConfig{
		Mappings: map[string]string{
			"prod-us-east": "custom-prod",
			"prod-*":       "production",
		},
	}

	// Exact match should win over glob
	if got := cfg.GetSkinForContext("prod-us-east"); got != "custom-prod" {
		t.Errorf("GetSkinForContext('prod-us-east') = %q, want %q (exact match precedence)", got, "custom-prod")
	}
	// Glob should still work for non-exact matches
	if got := cfg.GetSkinForContext("prod-eu-west"); got != "production" {
		t.Errorf("GetSkinForContext('prod-eu-west') = %q, want %q", got, "production")
	}
}

func TestLoadContextSkins_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgData := ContextSkinConfig{
		Mappings: map[string]string{
			"production": "production",
			"dev-*":      "development",
		},
	}
	data, err := yaml.Marshal(&cfgData)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "context-skins.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}

	// Override GetConfigDir to point to tmpDir
	origGetConfigDir := getConfigDirFunc
	getConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDirFunc = origGetConfigDir }()

	cfg, err := LoadContextSkins()
	if err != nil {
		t.Fatalf("LoadContextSkins() error = %v", err)
	}
	if len(cfg.Mappings) != 2 {
		t.Errorf("len(Mappings) = %d, want 2", len(cfg.Mappings))
	}
	if cfg.Mappings["production"] != "production" {
		t.Errorf("Mappings['production'] = %q, want %q", cfg.Mappings["production"], "production")
	}
}

func TestLoadContextSkins_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	origGetConfigDir := getConfigDirFunc
	getConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDirFunc = origGetConfigDir }()

	cfg, err := LoadContextSkins()
	if err != nil {
		t.Fatalf("LoadContextSkins() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadContextSkins() returned nil")
	}
	if len(cfg.Mappings) != 0 {
		t.Errorf("len(Mappings) = %d, want 0", len(cfg.Mappings))
	}
}

func TestLoadContextSkins_MalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "context-skins.yaml"), []byte("{{{invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	origGetConfigDir := getConfigDirFunc
	getConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDirFunc = origGetConfigDir }()

	cfg, err := LoadContextSkins()
	if err != nil {
		t.Fatalf("LoadContextSkins() should not return error for malformed YAML, got %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadContextSkins() returned nil")
	}
	if len(cfg.Mappings) != 0 {
		t.Errorf("len(Mappings) = %d, want 0 for malformed YAML", len(cfg.Mappings))
	}
}

func TestBuiltInContextSkins(t *testing.T) {
	skins := BuiltInContextSkins()

	// Should have exactly 3 built-in skins
	if len(skins) != 3 {
		t.Fatalf("len(BuiltInContextSkins()) = %d, want 3", len(skins))
	}

	for _, name := range []string{"production", "staging", "development"} {
		skin, ok := skins[name]
		if !ok {
			t.Errorf("BuiltInContextSkins() missing %q", name)
			continue
		}
		if skin == nil {
			t.Errorf("BuiltInContextSkins()[%q] is nil", name)
		}
	}
}

func TestBuiltInContextSkins_Colors(t *testing.T) {
	skins := BuiltInContextSkins()

	// Production should have red borders
	prod := skins["production"]
	if prod.K13s.Frame.BorderColor != "#ff5555" {
		t.Errorf("production BorderColor = %q, want #ff5555", prod.K13s.Frame.BorderColor)
	}
	if prod.K13s.StatusBar.BgColor != "#ff5555" {
		t.Errorf("production StatusBar.BgColor = %q, want #ff5555", prod.K13s.StatusBar.BgColor)
	}

	// Staging should have orange/yellow borders
	stg := skins["staging"]
	if stg.K13s.Frame.BorderColor != "#ffb86c" {
		t.Errorf("staging BorderColor = %q, want #ffb86c", stg.K13s.Frame.BorderColor)
	}

	// Development should have green borders
	dev := skins["development"]
	if dev.K13s.Frame.BorderColor != "#50fa7b" {
		t.Errorf("development BorderColor = %q, want #50fa7b", dev.K13s.Frame.BorderColor)
	}
}

func TestBuiltInContextSkins_PreserveDefaults(t *testing.T) {
	skins := BuiltInContextSkins()
	defaults := DefaultStyles()

	// Built-in skins should preserve non-overridden values from DefaultStyles
	for name, skin := range skins {
		// Body should remain default
		if skin.K13s.Body != defaults.K13s.Body {
			t.Errorf("%s skin Body = %+v, want default %+v", name, skin.K13s.Body, defaults.K13s.Body)
		}
		// Table styles should remain default
		if skin.K13s.Views.Table != defaults.K13s.Views.Table {
			t.Errorf("%s skin Table styles differ from defaults", name)
		}
	}
}

func TestLoadStylesForContext(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a context-skins.yaml that maps "my-prod" to "production"
	cfgData := ContextSkinConfig{
		Mappings: map[string]string{
			"my-prod": "production",
			"my-stg":  "staging",
		},
	}
	data, err := yaml.Marshal(&cfgData)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "context-skins.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}

	origGetConfigDir := getConfigDirFunc
	getConfigDirFunc = func() (string, error) { return tmpDir, nil }
	defer func() { getConfigDirFunc = origGetConfigDir }()

	// Test built-in skin resolution
	styles, err := LoadStylesForContext("my-prod")
	if err != nil {
		t.Fatalf("LoadStylesForContext() error = %v", err)
	}
	if styles.K13s.Frame.BorderColor != "#ff5555" {
		t.Errorf("production context BorderColor = %q, want #ff5555", styles.K13s.Frame.BorderColor)
	}

	// Test unmapped context returns defaults
	styles, err = LoadStylesForContext("unknown-context")
	if err != nil {
		t.Fatalf("LoadStylesForContext() error = %v", err)
	}
	defaults := DefaultStyles()
	if styles.K13s.Frame.BorderColor != defaults.K13s.Frame.BorderColor {
		t.Errorf("unmapped context BorderColor = %q, want default %q",
			styles.K13s.Frame.BorderColor, defaults.K13s.Frame.BorderColor)
	}
}
