package cluster

import (
	"testing"
)

func TestNewProvider_Existing(t *testing.T) {
	provider, err := NewProvider(ProviderExisting)
	if err != nil {
		t.Fatalf("NewProvider(ProviderExisting) failed: %v", err)
	}
	if provider == nil {
		t.Fatal("NewProvider returned nil provider")
	}
	if provider.Name() != "existing" {
		t.Errorf("Name() = %s, want existing", provider.Name())
	}
}

func TestNewProvider_Kind(t *testing.T) {
	provider, err := NewProvider(ProviderKind)
	if err != nil {
		t.Fatalf("NewProvider(ProviderKind) failed: %v", err)
	}
	if provider == nil {
		t.Fatal("NewProvider returned nil provider")
	}
	if provider.Name() != "kind" {
		t.Errorf("Name() = %s, want kind", provider.Name())
	}
}

func TestNewProvider_VCluster(t *testing.T) {
	provider, err := NewProvider(ProviderVCluster)
	if err != nil {
		t.Fatalf("NewProvider(ProviderVCluster) failed: %v", err)
	}
	if provider == nil {
		t.Fatal("NewProvider returned nil provider")
	}
	if provider.Name() != "vcluster" {
		t.Errorf("Name() = %s, want vcluster", provider.Name())
	}
}

func TestNewProvider_Unknown(t *testing.T) {
	_, err := NewProvider(ProviderType("unknown"))
	if err == nil {
		t.Error("NewProvider with unknown type should return error")
	}
}

func TestProviderOptions(t *testing.T) {
	config := &ProviderConfig{}

	// Test WithKindImage
	WithKindImage("kindest/node:v1.28.0")(config)
	if config.KindImage != "kindest/node:v1.28.0" {
		t.Errorf("KindImage = %s, want kindest/node:v1.28.0", config.KindImage)
	}

	// Test WithVClusterContext
	WithVClusterContext("host-context")(config)
	if config.VClusterContext != "host-context" {
		t.Errorf("VClusterContext = %s, want host-context", config.VClusterContext)
	}

	// Test WithVClusterKubeconfig
	WithVClusterKubeconfig("/path/to/kubeconfig")(config)
	if config.VClusterKubeconfig != "/path/to/kubeconfig" {
		t.Errorf("VClusterKubeconfig = %s, want /path/to/kubeconfig", config.VClusterKubeconfig)
	}

	// Test WithExistingKubeconfig
	WithExistingKubeconfig("/existing/kubeconfig")(config)
	if config.ExistingKubeconfig != "/existing/kubeconfig" {
		t.Errorf("ExistingKubeconfig = %s, want /existing/kubeconfig", config.ExistingKubeconfig)
	}

	// Test WithExistingContext
	WithExistingContext("existing-context")(config)
	if config.ExistingContext != "existing-context" {
		t.Errorf("ExistingContext = %s, want existing-context", config.ExistingContext)
	}

	// Test WithWorkDir
	WithWorkDir("/tmp/workdir")(config)
	if config.WorkDir != "/tmp/workdir" {
		t.Errorf("WorkDir = %s, want /tmp/workdir", config.WorkDir)
	}
}

func TestNewProvider_WithOptions(t *testing.T) {
	provider, err := NewProvider(ProviderExisting,
		WithExistingKubeconfig("/custom/kubeconfig"),
		WithExistingContext("custom-context"),
	)
	if err != nil {
		t.Fatalf("NewProvider with options failed: %v", err)
	}
	if provider == nil {
		t.Fatal("NewProvider returned nil provider")
	}
}

func TestProviderTypes(t *testing.T) {
	tests := []struct {
		providerType ProviderType
		expected     string
	}{
		{ProviderKind, "kind"},
		{ProviderVCluster, "vcluster"},
		{ProviderExisting, "existing"},
	}

	for _, tt := range tests {
		t.Run(string(tt.providerType), func(t *testing.T) {
			if string(tt.providerType) != tt.expected {
				t.Errorf("ProviderType = %s, want %s", tt.providerType, tt.expected)
			}
		})
	}
}
