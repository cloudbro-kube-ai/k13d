package cluster

import (
	"context"
	"fmt"
	"os"
)

// ExistingProvider uses an existing cluster (no provisioning)
type ExistingProvider struct {
	config *ProviderConfig
}

// NewExistingProvider creates a new existing cluster provider
func NewExistingProvider(config *ProviderConfig) *ExistingProvider {
	if config == nil {
		config = &ProviderConfig{}
	}
	return &ExistingProvider{config: config}
}

// Name returns the provider name
func (p *ExistingProvider) Name() string {
	return "existing"
}

// Exists always returns true for existing clusters
func (p *ExistingProvider) Exists(ctx context.Context, name string) (bool, error) {
	// Check if kubeconfig file exists
	kubeconfigPath := p.getKubeconfigPath()
	if _, err := os.Stat(kubeconfigPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Create is a no-op for existing clusters
func (p *ExistingProvider) Create(ctx context.Context, name string) error {
	// Verify the cluster is accessible
	exists, err := p.Exists(ctx, name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("kubeconfig not found at %s", p.getKubeconfigPath())
	}
	return nil
}

// Delete is a no-op for existing clusters
func (p *ExistingProvider) Delete(ctx context.Context, name string) error {
	// Don't delete existing clusters
	return nil
}

// GetKubeconfig returns the kubeconfig content
func (p *ExistingProvider) GetKubeconfig(ctx context.Context, name string) ([]byte, error) {
	kubeconfigPath := p.getKubeconfigPath()
	return os.ReadFile(kubeconfigPath)
}

// GetKubeconfigPath returns the path to the kubeconfig
func (p *ExistingProvider) GetKubeconfigPath(ctx context.Context, name string) (string, error) {
	return p.getKubeconfigPath(), nil
}

// getKubeconfigPath returns the kubeconfig path from config or default
func (p *ExistingProvider) getKubeconfigPath() string {
	if p.config.ExistingKubeconfig != "" {
		return p.config.ExistingKubeconfig
	}

	// Default to KUBECONFIG env or ~/.kube/config
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	homeDir, _ := os.UserHomeDir()
	return fmt.Sprintf("%s/.kube/config", homeDir)
}
