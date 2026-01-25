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

// VClusterProvider implements cluster provisioning using vCluster
type VClusterProvider struct {
	config *ProviderConfig
}

// NewVClusterProvider creates a new vCluster provider
func NewVClusterProvider(config *ProviderConfig) *VClusterProvider {
	if config == nil {
		config = &ProviderConfig{}
	}
	return &VClusterProvider{config: config}
}

// Name returns the provider name
func (p *VClusterProvider) Name() string {
	return "vcluster"
}

// Exists checks if a vCluster exists
func (p *VClusterProvider) Exists(ctx context.Context, name string) (bool, error) {
	args := []string{"list", "--output", "json"}
	if p.config.VClusterContext != "" {
		args = append(args, "--context", p.config.VClusterContext)
	}
	if p.config.VClusterKubeconfig != "" {
		args = append(args, "--kubeconfig", p.config.VClusterKubeconfig)
	}

	cmd := exec.CommandContext(ctx, "vcluster", args...)
	output, err := cmd.Output()
	if err != nil {
		// If vcluster list fails, assume no clusters exist
		return false, nil
	}

	// Check if the name appears in the output
	return strings.Contains(string(output), name), nil
}

// Create creates a new vCluster
func (p *VClusterProvider) Create(ctx context.Context, name string) error {
	// Create namespace for vcluster
	namespace := fmt.Sprintf("vcluster-%s", name)
	if err := p.createNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	args := []string{"create", name, "--namespace", namespace, "--connect=false"}
	if p.config.VClusterContext != "" {
		args = append(args, "--context", p.config.VClusterContext)
	}
	if p.config.VClusterKubeconfig != "" {
		args = append(args, "--kubeconfig", p.config.VClusterKubeconfig)
	}

	cmd := exec.CommandContext(ctx, "vcluster", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create vcluster: %w, stderr: %s", err, stderr.String())
	}

	// Wait for vcluster to be ready
	if err := p.waitForVCluster(ctx, name, namespace); err != nil {
		return fmt.Errorf("vcluster created but not ready: %w", err)
	}

	return nil
}

// Delete deletes a vCluster
func (p *VClusterProvider) Delete(ctx context.Context, name string) error {
	namespace := fmt.Sprintf("vcluster-%s", name)

	args := []string{"delete", name, "--namespace", namespace}
	if p.config.VClusterContext != "" {
		args = append(args, "--context", p.config.VClusterContext)
	}
	if p.config.VClusterKubeconfig != "" {
		args = append(args, "--kubeconfig", p.config.VClusterKubeconfig)
	}

	cmd := exec.CommandContext(ctx, "vcluster", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete vcluster: %w, stderr: %s", err, stderr.String())
	}

	// Delete the namespace
	_ = p.deleteNamespace(ctx, namespace)

	return nil
}

// GetKubeconfig returns the kubeconfig for the vCluster
func (p *VClusterProvider) GetKubeconfig(ctx context.Context, name string) ([]byte, error) {
	namespace := fmt.Sprintf("vcluster-%s", name)

	// Create a temp file for kubeconfig
	workDir := p.config.WorkDir
	if workDir == "" {
		workDir = os.TempDir()
	}
	kubeconfigPath := filepath.Join(workDir, fmt.Sprintf("kubeconfig-vcluster-%s.yaml", name))

	args := []string{"connect", name,
		"--namespace", namespace,
		"--kube-config-context-name", name,
		"--update-current=false",
		"--kube-config", kubeconfigPath,
	}
	if p.config.VClusterContext != "" {
		args = append(args, "--context", p.config.VClusterContext)
	}
	if p.config.VClusterKubeconfig != "" {
		args = append(args, "--kubeconfig", p.config.VClusterKubeconfig)
	}

	cmd := exec.CommandContext(ctx, "vcluster", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get vcluster kubeconfig: %w, stderr: %s", err, stderr.String())
	}

	kubeconfig, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	// Cleanup temp file
	os.Remove(kubeconfigPath)

	return kubeconfig, nil
}

// GetKubeconfigPath returns the path to a kubeconfig file for the vCluster
func (p *VClusterProvider) GetKubeconfigPath(ctx context.Context, name string) (string, error) {
	kubeconfig, err := p.GetKubeconfig(ctx, name)
	if err != nil {
		return "", err
	}

	workDir := p.config.WorkDir
	if workDir == "" {
		workDir = os.TempDir()
	}

	kubeconfigPath := filepath.Join(workDir, fmt.Sprintf("kubeconfig-vcluster-%s.yaml", name))
	if err := os.WriteFile(kubeconfigPath, kubeconfig, 0600); err != nil {
		return "", fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	return kubeconfigPath, nil
}

// createNamespace creates a namespace in the host cluster
func (p *VClusterProvider) createNamespace(ctx context.Context, namespace string) error {
	args := []string{"create", "namespace", namespace}
	if p.config.VClusterKubeconfig != "" {
		args = append([]string{"--kubeconfig", p.config.VClusterKubeconfig}, args...)
	}
	if p.config.VClusterContext != "" {
		args = append([]string{"--context", p.config.VClusterContext}, args...)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	// Ignore error if namespace already exists
	_ = cmd.Run()
	return nil
}

// deleteNamespace deletes a namespace from the host cluster
func (p *VClusterProvider) deleteNamespace(ctx context.Context, namespace string) error {
	args := []string{"delete", "namespace", namespace, "--ignore-not-found"}
	if p.config.VClusterKubeconfig != "" {
		args = append([]string{"--kubeconfig", p.config.VClusterKubeconfig}, args...)
	}
	if p.config.VClusterContext != "" {
		args = append([]string{"--context", p.config.VClusterContext}, args...)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	return cmd.Run()
}

// waitForVCluster waits for vCluster to be ready
func (p *VClusterProvider) waitForVCluster(ctx context.Context, name, namespace string) error {
	deadline := time.Now().Add(5 * time.Minute)

	for time.Now().Before(deadline) {
		// Check if vcluster pod is running
		args := []string{"get", "pods", "-n", namespace, "-l", fmt.Sprintf("app=vcluster,release=%s", name),
			"-o", "jsonpath={.items[0].status.phase}"}
		if p.config.VClusterKubeconfig != "" {
			args = append([]string{"--kubeconfig", p.config.VClusterKubeconfig}, args...)
		}
		if p.config.VClusterContext != "" {
			args = append([]string{"--context", p.config.VClusterContext}, args...)
		}

		cmd := exec.CommandContext(ctx, "kubectl", args...)
		output, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(output)) == "Running" {
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for vcluster to be ready")
}

// IsVClusterInstalled checks if vcluster CLI is installed
func IsVClusterInstalled() bool {
	_, err := exec.LookPath("vcluster")
	return err == nil
}
