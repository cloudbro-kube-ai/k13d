package cluster

import (
	"testing"
)

func TestNewVClusterProvider(t *testing.T) {
	// Test with nil config
	p := NewVClusterProvider(nil)
	if p == nil {
		t.Fatal("NewVClusterProvider returned nil")
	}
	if p.config == nil {
		t.Error("config should not be nil")
	}

	// Test with config
	config := &ProviderConfig{
		VClusterContext:    "host-context",
		VClusterKubeconfig: "/path/to/kubeconfig",
		WorkDir:            "/tmp/test",
	}
	p = NewVClusterProvider(config)
	if p.config.VClusterContext != "host-context" {
		t.Errorf("VClusterContext = %s, want host-context", p.config.VClusterContext)
	}
	if p.config.VClusterKubeconfig != "/path/to/kubeconfig" {
		t.Errorf("VClusterKubeconfig = %s, want /path/to/kubeconfig", p.config.VClusterKubeconfig)
	}
	if p.config.WorkDir != "/tmp/test" {
		t.Errorf("WorkDir = %s, want /tmp/test", p.config.WorkDir)
	}
}

func TestVClusterProvider_Name(t *testing.T) {
	p := NewVClusterProvider(nil)
	if p.Name() != "vcluster" {
		t.Errorf("Name() = %s, want vcluster", p.Name())
	}
}

func TestIsVClusterInstalled(t *testing.T) {
	// This test just verifies the function doesn't panic
	// The actual result depends on whether vcluster is installed
	_ = IsVClusterInstalled()
}

// Note: The following tests require vcluster CLI to be installed
// and are skipped if not available

func TestVClusterProvider_Exists_NoVCluster(t *testing.T) {
	if !IsVClusterInstalled() {
		t.Skip("vcluster CLI not installed")
	}

	// Note: We don't actually test Exists here because it requires vcluster CLI
	// and would make CI dependent on having vcluster installed
}
