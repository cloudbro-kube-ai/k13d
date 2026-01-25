// Package cluster provides cluster provisioning for benchmarks
package cluster

import (
	"context"
	"fmt"
)

// Provider interface for cluster provisioning
type Provider interface {
	// Name returns the provider name
	Name() string

	// Exists checks if a cluster with the given name exists
	Exists(ctx context.Context, name string) (bool, error)

	// Create creates a new cluster with the given name
	Create(ctx context.Context, name string) error

	// Delete deletes the cluster with the given name
	Delete(ctx context.Context, name string) error

	// GetKubeconfig returns the kubeconfig for the cluster
	GetKubeconfig(ctx context.Context, name string) ([]byte, error)

	// GetKubeconfigPath returns the path to the kubeconfig file
	GetKubeconfigPath(ctx context.Context, name string) (string, error)
}

// ProviderType represents the type of cluster provider
type ProviderType string

const (
	ProviderKind     ProviderType = "kind"
	ProviderVCluster ProviderType = "vcluster"
	ProviderExisting ProviderType = "existing"
)

// NewProvider creates a new cluster provider based on the type
func NewProvider(providerType ProviderType, opts ...ProviderOption) (Provider, error) {
	config := &ProviderConfig{}
	for _, opt := range opts {
		opt(config)
	}

	switch providerType {
	case ProviderKind:
		return NewKindProvider(config), nil
	case ProviderVCluster:
		return NewVClusterProvider(config), nil
	case ProviderExisting:
		return NewExistingProvider(config), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// ProviderConfig holds configuration for providers
type ProviderConfig struct {
	// Kind-specific settings
	KindImage string // Kind node image

	// VCluster-specific settings
	VClusterContext    string // Host context for vCluster
	VClusterKubeconfig string // Host kubeconfig for vCluster

	// Existing cluster settings
	ExistingKubeconfig string // Path to existing kubeconfig
	ExistingContext    string // Context to use

	// General settings
	WorkDir string // Working directory for temp files
}

// ProviderOption is a functional option for providers
type ProviderOption func(*ProviderConfig)

// WithKindImage sets the Kind node image
func WithKindImage(image string) ProviderOption {
	return func(c *ProviderConfig) {
		c.KindImage = image
	}
}

// WithVClusterContext sets the host context for vCluster
func WithVClusterContext(context string) ProviderOption {
	return func(c *ProviderConfig) {
		c.VClusterContext = context
	}
}

// WithVClusterKubeconfig sets the host kubeconfig for vCluster
func WithVClusterKubeconfig(kubeconfig string) ProviderOption {
	return func(c *ProviderConfig) {
		c.VClusterKubeconfig = kubeconfig
	}
}

// WithExistingKubeconfig sets the existing kubeconfig path
func WithExistingKubeconfig(kubeconfig string) ProviderOption {
	return func(c *ProviderConfig) {
		c.ExistingKubeconfig = kubeconfig
	}
}

// WithExistingContext sets the context to use
func WithExistingContext(context string) ProviderOption {
	return func(c *ProviderConfig) {
		c.ExistingContext = context
	}
}

// WithWorkDir sets the working directory
func WithWorkDir(dir string) ProviderOption {
	return func(c *ProviderConfig) {
		c.WorkDir = dir
	}
}
