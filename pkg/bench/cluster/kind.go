package cluster

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// KindProvider implements cluster provisioning using Kind
type KindProvider struct {
	config *ProviderConfig
}

// NewKindProvider creates a new Kind provider
func NewKindProvider(config *ProviderConfig) *KindProvider {
	if config == nil {
		config = &ProviderConfig{}
	}
	return &KindProvider{config: config}
}

// Name returns the provider name
func (p *KindProvider) Name() string {
	return "kind"
}

// Exists checks if a Kind cluster exists
func (p *KindProvider) Exists(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "kind", "get", "clusters")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list kind clusters: %w", err)
	}

	clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, cluster := range clusters {
		if strings.TrimSpace(cluster) == name {
			return true, nil
		}
	}
	return false, nil
}

// Create creates a new Kind cluster
func (p *KindProvider) Create(ctx context.Context, name string) error {
	// Retry logic for cluster creation
	maxRetries := 3
	retryDelay := 5 * time.Second

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			// Delete any partial cluster from previous attempt
			_ = p.Delete(ctx, name)
			time.Sleep(retryDelay)
		}

		args := []string{"create", "cluster", "--name", name, "--wait", "5m"}
		if p.config.KindImage != "" {
			args = append(args, "--image", p.config.KindImage)
		}

		cmd := exec.CommandContext(ctx, "kind", args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf("failed to create kind cluster (attempt %d/%d): %w, stderr: %s",
				i+1, maxRetries, err, stderr.String())
			continue
		}

		// Verify cluster is ready
		if err := p.waitForCluster(ctx, name); err != nil {
			lastErr = fmt.Errorf("cluster created but not ready: %w", err)
			continue
		}

		return nil
	}

	return lastErr
}

// Delete deletes a Kind cluster
func (p *KindProvider) Delete(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "kind", "delete", "cluster", "--name", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete kind cluster: %w, stderr: %s", err, stderr.String())
	}
	return nil
}

// GetKubeconfig returns the kubeconfig for the Kind cluster
func (p *KindProvider) GetKubeconfig(ctx context.Context, name string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "kind", "get", "kubeconfig", "--name", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	return output, nil
}

// GetKubeconfigPath returns the path to a kubeconfig file for the cluster
func (p *KindProvider) GetKubeconfigPath(ctx context.Context, name string) (string, error) {
	kubeconfig, err := p.GetKubeconfig(ctx, name)
	if err != nil {
		return "", err
	}

	// Write to temp file
	workDir := p.config.WorkDir
	if workDir == "" {
		workDir = os.TempDir()
	}

	kubeconfigPath := filepath.Join(workDir, fmt.Sprintf("kubeconfig-%s.yaml", name))
	if err := os.WriteFile(kubeconfigPath, kubeconfig, 0600); err != nil {
		return "", fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	return kubeconfigPath, nil
}

// waitForCluster waits for the cluster to be ready
func (p *KindProvider) waitForCluster(ctx context.Context, name string) error {
	kubeconfigPath, err := p.GetKubeconfigPath(ctx, name)
	if err != nil {
		return err
	}
	defer os.Remove(kubeconfigPath)

	// Wait for nodes to be ready
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigPath,
			"get", "nodes", "-o", "jsonpath={.items[*].status.conditions[?(@.type=='Ready')].status}")
		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), "True") {
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for cluster nodes to be ready")
}

// IsKindInstalled checks if kind CLI is installed
func IsKindInstalled() bool {
	_, err := exec.LookPath("kind")
	return err == nil
}
