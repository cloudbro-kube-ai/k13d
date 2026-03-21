package config

import (
	"testing"
)

func TestDefaultPlugins(t *testing.T) {
	p := DefaultPlugins()
	if p == nil {
		t.Fatal("DefaultPlugins() returned nil")
	}
	if len(p.Plugins) == 0 {
		t.Error("DefaultPlugins() returned empty plugins map")
	}
	// Verify known default plugin
	dive, ok := p.Plugins["dive"]
	if !ok {
		t.Fatal("missing default plugin: dive")
	}
	if dive.Command != "dive" {
		t.Errorf("dive command = %q, want %q", dive.Command, "dive")
	}
}

func TestGetPluginsForScope(t *testing.T) {
	p := &PluginsFile{
		Plugins: map[string]PluginConfig{
			"pod-only": {
				ShortCut: "P",
				Scopes:   []string{"pods"},
				Command:  "echo",
			},
			"global": {
				ShortCut: "G",
				Scopes:   []string{"*"},
				Command:  "echo",
			},
			"multi-scope": {
				ShortCut: "M",
				Scopes:   []string{"pods", "deployments"},
				Command:  "echo",
			},
		},
	}

	tests := []struct {
		name  string
		scope string
		want  int
	}{
		{"pods gets all matching", "pods", 3},
		{"deployments", "deployments", 2},
		{"services only global", "services", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.GetPluginsForScope(tt.scope)
			if len(result) != tt.want {
				t.Errorf("GetPluginsForScope(%q) returned %d, want %d", tt.scope, len(result), tt.want)
			}
		})
	}
}

func TestPluginExpandArgs(t *testing.T) {
	p := PluginConfig{
		Args: []string{"debug", "-n", "$NAMESPACE", "$NAME", "--image=$IMAGE"},
	}

	ctx := &PluginContext{
		Namespace: "default",
		Name:      "my-pod",
		Image:     "busybox",
	}

	expanded := p.ExpandArgs(ctx)

	// $NAMESPACE and $NAME are whole-token variables, $IMAGE too
	if expanded[2] != "default" {
		t.Errorf("expanded[2] = %q, want %q", expanded[2], "default")
	}
	if expanded[3] != "my-pod" {
		t.Errorf("expanded[3] = %q, want %q", expanded[3], "my-pod")
	}
	// "--image=$IMAGE" is NOT a whole-token variable, so it stays as-is
	if expanded[4] != "--image=$IMAGE" {
		t.Errorf("expanded[4] = %q, want %q (inline vars not expanded)", expanded[4], "--image=$IMAGE")
	}
}

func TestPluginExpandArgs_Labels(t *testing.T) {
	p := PluginConfig{
		Args: []string{"$LABELS.app", "$ANNOTATIONS.version"},
	}

	ctx := &PluginContext{
		Labels:      map[string]string{"app": "nginx"},
		Annotations: map[string]string{"version": "1.0"},
	}

	expanded := p.ExpandArgs(ctx)
	if expanded[0] != "nginx" {
		t.Errorf("label expansion = %q, want %q", expanded[0], "nginx")
	}
	if expanded[1] != "1.0" {
		t.Errorf("annotation expansion = %q, want %q", expanded[1], "1.0")
	}
}

func TestPluginExpandArgs_MissingLabels(t *testing.T) {
	p := PluginConfig{
		Args: []string{"$LABELS.nonexistent", "$ANNOTATIONS.missing"},
	}

	ctx := &PluginContext{
		Labels:      map[string]string{},
		Annotations: map[string]string{},
	}

	expanded := p.ExpandArgs(ctx)
	if expanded[0] != "" {
		t.Errorf("missing label = %q, want empty", expanded[0])
	}
	if expanded[1] != "" {
		t.Errorf("missing annotation = %q, want empty", expanded[1])
	}
}

func TestPluginValidate(t *testing.T) {
	tests := []struct {
		name    string
		plugin  PluginConfig
		wantErr bool
	}{
		{
			name: "valid plugin",
			plugin: PluginConfig{
				ShortCut: "P",
				Command:  "echo",
				Scopes:   []string{"pods"},
			},
			wantErr: false,
		},
		{
			name: "missing shortcut",
			plugin: PluginConfig{
				Command: "echo",
				Scopes:  []string{"pods"},
			},
			wantErr: true,
		},
		{
			name: "missing command",
			plugin: PluginConfig{
				ShortCut: "P",
				Scopes:   []string{"pods"},
			},
			wantErr: true,
		},
		{
			name: "missing scopes",
			plugin: PluginConfig{
				ShortCut: "P",
				Command:  "echo",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plugin.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainsShellMetachar(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"safe-string", false},
		{"hello world", false},
		{"pipe|injection", true},
		{"semi;colon", true},
		{"back`tick", true},
		{"dollar$var", true},
		{"paren(open", true},
		{"newline\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := containsShellMetachar(tt.input)
			if got != tt.want {
				t.Errorf("containsShellMetachar(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetAvailableVariables(t *testing.T) {
	vars := GetAvailableVariables()
	if len(vars) == 0 {
		t.Error("GetAvailableVariables() returned empty list")
	}
	// Should contain at least $NAMESPACE and $NAME
	found := 0
	for _, v := range vars {
		if v == "$NAMESPACE - Resource namespace" || v == "$NAME - Resource name" {
			found++
		}
	}
	if found < 2 {
		t.Errorf("expected at least $NAMESPACE and $NAME, found %d", found)
	}
}
