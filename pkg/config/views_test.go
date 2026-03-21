package config

import (
	"testing"
)

func TestDefaultViews(t *testing.T) {
	v := DefaultViews()
	if v == nil {
		t.Fatal("DefaultViews() returned nil")
	}
	if v.Views == nil {
		t.Fatal("DefaultViews().Views is nil")
	}
	if len(v.Views) != 0 {
		t.Errorf("DefaultViews() has %d views, want 0", len(v.Views))
	}
}

func TestGetViewConfig(t *testing.T) {
	v := &ViewConfig{
		Views: map[string]ResourceViewConfig{
			"pods": {
				SortColumn:    "AGE",
				SortAscending: false,
			},
			"deployments": {
				SortColumn:    "NAME",
				SortAscending: true,
			},
		},
	}

	t.Run("existing resource", func(t *testing.T) {
		cfg := v.GetViewConfig("pods")
		if cfg == nil {
			t.Fatal("GetViewConfig(pods) returned nil")
		}
		if cfg.SortColumn != "AGE" {
			t.Errorf("SortColumn = %q, want %q", cfg.SortColumn, "AGE")
		}
		if cfg.SortAscending {
			t.Error("SortAscending = true, want false")
		}
	})

	t.Run("another existing resource", func(t *testing.T) {
		cfg := v.GetViewConfig("deployments")
		if cfg == nil {
			t.Fatal("GetViewConfig(deployments) returned nil")
		}
		if cfg.SortColumn != "NAME" {
			t.Errorf("SortColumn = %q, want %q", cfg.SortColumn, "NAME")
		}
		if !cfg.SortAscending {
			t.Error("SortAscending = false, want true")
		}
	})

	t.Run("non-existing resource", func(t *testing.T) {
		cfg := v.GetViewConfig("services")
		if cfg != nil {
			t.Errorf("GetViewConfig(services) = %v, want nil", cfg)
		}
	})
}

func TestGetViewConfig_NilSafety(t *testing.T) {
	// nil ViewConfig
	var v *ViewConfig
	cfg := v.GetViewConfig("pods")
	if cfg != nil {
		t.Errorf("nil.GetViewConfig() = %v, want nil", cfg)
	}

	// nil Views map
	v2 := &ViewConfig{Views: nil}
	cfg2 := v2.GetViewConfig("pods")
	if cfg2 != nil {
		t.Errorf("nilMap.GetViewConfig() = %v, want nil", cfg2)
	}
}

func TestResourceViewConfig_Defaults(t *testing.T) {
	// Zero-value struct should have sensible defaults
	var cfg ResourceViewConfig
	if cfg.SortColumn != "" {
		t.Errorf("zero SortColumn = %q, want empty", cfg.SortColumn)
	}
	if cfg.SortAscending {
		t.Error("zero SortAscending = true, want false")
	}
}
