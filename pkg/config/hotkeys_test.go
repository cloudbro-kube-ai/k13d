package config

import (
	"testing"
)

func TestDefaultHotkeys(t *testing.T) {
	hk := DefaultHotkeys()
	if hk == nil {
		t.Fatal("DefaultHotkeys() returned nil")
	}
	if len(hk.Hotkeys) == 0 {
		t.Error("DefaultHotkeys() returned empty hotkeys map")
	}
	// Verify known default hotkey
	stern, ok := hk.Hotkeys["stern-logs"]
	if !ok {
		t.Fatal("missing default hotkey: stern-logs")
	}
	if stern.ShortCut != "Shift-L" {
		t.Errorf("stern-logs shortcut = %q, want %q", stern.ShortCut, "Shift-L")
	}
	if stern.Command != "stern" {
		t.Errorf("stern-logs command = %q, want %q", stern.Command, "stern")
	}
}

func TestGetHotkeysForScope(t *testing.T) {
	hk := &HotkeysFile{
		Hotkeys: map[string]HotkeyConfig{
			"pod-action": {
				ShortCut: "Shift-P",
				Scopes:   []string{"pods"},
				Command:  "echo",
			},
			"global-action": {
				ShortCut: "Ctrl-G",
				Scopes:   []string{"*"},
				Command:  "echo",
			},
			"svc-action": {
				ShortCut: "Shift-S",
				Scopes:   []string{"services"},
				Command:  "echo",
			},
		},
	}

	tests := []struct {
		name     string
		scope    string
		wantKeys []string
	}{
		{"pods scope", "pods", []string{"pod-action", "global-action"}},
		{"services scope", "services", []string{"svc-action", "global-action"}},
		{"no match", "configmaps", []string{"global-action"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hk.GetHotkeysForScope(tt.scope)
			if len(result) != len(tt.wantKeys) {
				t.Errorf("GetHotkeysForScope(%q) returned %d results, want %d", tt.scope, len(result), len(tt.wantKeys))
			}
			for _, key := range tt.wantKeys {
				if _, ok := result[key]; !ok {
					t.Errorf("GetHotkeysForScope(%q) missing key %q", tt.scope, key)
				}
			}
		})
	}
}

func TestHotkeyExpandArgs(t *testing.T) {
	hk := HotkeyConfig{
		Args: []string{"-n", "$NAMESPACE", "$NAME", "--context", "$CONTEXT", "literal"},
	}

	expanded := hk.ExpandArgs("kube-system", "my-pod", "prod-cluster")

	expected := []string{"-n", "kube-system", "my-pod", "--context", "prod-cluster", "literal"}
	if len(expanded) != len(expected) {
		t.Fatalf("len(expanded) = %d, want %d", len(expanded), len(expected))
	}
	for i, v := range expanded {
		if v != expected[i] {
			t.Errorf("expanded[%d] = %q, want %q", i, v, expected[i])
		}
	}
}

func TestParseShortcut(t *testing.T) {
	tests := []struct {
		input     string
		wantCtrl  bool
		wantShift bool
		wantAlt   bool
		wantKey   string
	}{
		{"Ctrl-K", true, false, false, "K"},
		{"Shift-L", false, true, false, "L"},
		{"Alt-X", false, false, true, "X"},
		{"Ctrl-Shift-D", true, true, false, "D"},
		{"P", false, false, false, "P"},
		{"Ctrl-Alt-Delete", true, false, true, "Delete"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ctrl, shift, alt, key := ParseShortcut(tt.input)
			if ctrl != tt.wantCtrl {
				t.Errorf("ctrl = %v, want %v", ctrl, tt.wantCtrl)
			}
			if shift != tt.wantShift {
				t.Errorf("shift = %v, want %v", shift, tt.wantShift)
			}
			if alt != tt.wantAlt {
				t.Errorf("alt = %v, want %v", alt, tt.wantAlt)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestHotkeyConfigYAML(t *testing.T) {
	hk := DefaultHotkeys()

	// Verify YAML round-trip by checking structure
	for name, hotkey := range hk.Hotkeys {
		if hotkey.ShortCut == "" {
			t.Errorf("hotkey %q has empty shortcut", name)
		}
		if hotkey.Command == "" {
			t.Errorf("hotkey %q has empty command", name)
		}
		if len(hotkey.Scopes) == 0 {
			t.Errorf("hotkey %q has empty scopes", name)
		}
	}
}
