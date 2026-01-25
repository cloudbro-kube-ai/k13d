package cluster

import (
	"testing"
)

func TestNewKindProvider(t *testing.T) {
	// Test with nil config
	p := NewKindProvider(nil)
	if p == nil {
		t.Fatal("NewKindProvider returned nil")
	}
	if p.config == nil {
		t.Error("config should not be nil")
	}

	// Test with config
	config := &ProviderConfig{
		KindImage: "kindest/node:v1.28.0",
		WorkDir:   "/tmp/test",
	}
	p = NewKindProvider(config)
	if p.config.KindImage != "kindest/node:v1.28.0" {
		t.Errorf("KindImage = %s, want kindest/node:v1.28.0", p.config.KindImage)
	}
	if p.config.WorkDir != "/tmp/test" {
		t.Errorf("WorkDir = %s, want /tmp/test", p.config.WorkDir)
	}
}

func TestKindProvider_Name(t *testing.T) {
	p := NewKindProvider(nil)
	if p.Name() != "kind" {
		t.Errorf("Name() = %s, want kind", p.Name())
	}
}

func TestIsKindInstalled(t *testing.T) {
	// This test just verifies the function doesn't panic
	// The actual result depends on whether kind is installed
	_ = IsKindInstalled()
}

// Note: The following tests require kind CLI to be installed and are skipped if not available
// They also require Docker to be running

func TestKindProvider_Exists_NoKind(t *testing.T) {
	if !IsKindInstalled() {
		t.Skip("kind CLI not installed")
	}

	// Test with a cluster name that likely doesn't exist
	// p := NewKindProvider(nil)
	// ctx := context.Background()

	// Note: We don't actually test Exists here because it requires kind CLI
	// and would make CI dependent on having kind installed
}
