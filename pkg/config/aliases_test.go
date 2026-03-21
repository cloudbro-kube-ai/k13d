package config

import (
	"testing"
)

func TestDefaultAliases(t *testing.T) {
	a := DefaultAliases()
	if a == nil {
		t.Fatal("DefaultAliases() returned nil")
	}
	if a.Aliases == nil {
		t.Fatal("DefaultAliases().Aliases is nil")
	}
	// Default aliases should be empty (built-in aliases are in app.go)
	if len(a.Aliases) != 0 {
		t.Errorf("DefaultAliases() has %d aliases, want 0", len(a.Aliases))
	}
}

func TestAliasResolve(t *testing.T) {
	a := &AliasConfig{
		Aliases: map[string]string{
			"pp":  "pods",
			"dp":  "deployments",
			"svc": "services",
		},
	}

	tests := []struct {
		input string
		want  string
	}{
		{"pp", "pods"},
		{"dp", "deployments"},
		{"svc", "services"},
		{"pods", "pods"},       // not an alias, returns as-is
		{"unknown", "unknown"}, // not an alias, returns as-is
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := a.Resolve(tt.input)
			if got != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAliasResolve_NilSafety(t *testing.T) {
	// nil AliasConfig
	var a *AliasConfig
	if got := a.Resolve("pods"); got != "pods" {
		t.Errorf("nil.Resolve() = %q, want %q", got, "pods")
	}

	// nil Aliases map
	a2 := &AliasConfig{Aliases: nil}
	if got := a2.Resolve("pods"); got != "pods" {
		t.Errorf("nilMap.Resolve() = %q, want %q", got, "pods")
	}
}

func TestAliasGetAll(t *testing.T) {
	a := &AliasConfig{
		Aliases: map[string]string{
			"pp": "pods",
			"dp": "deployments",
		},
	}

	all := a.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll() returned %d, want 2", len(all))
	}
	if all["pp"] != "pods" {
		t.Errorf("GetAll()[pp] = %q, want %q", all["pp"], "pods")
	}
}

func TestAliasGetAll_NilSafety(t *testing.T) {
	var a *AliasConfig
	all := a.GetAll()
	if len(all) != 0 {
		t.Errorf("nil.GetAll() returned %d, want 0", len(all))
	}

	a2 := &AliasConfig{Aliases: nil}
	all2 := a2.GetAll()
	if len(all2) != 0 {
		t.Errorf("nilMap.GetAll() returned %d, want 0", len(all2))
	}
}
