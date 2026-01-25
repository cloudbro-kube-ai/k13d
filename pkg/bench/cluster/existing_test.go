package cluster

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewExistingProvider(t *testing.T) {
	// Test with nil config
	p := NewExistingProvider(nil)
	if p == nil {
		t.Fatal("NewExistingProvider returned nil")
	}
	if p.config == nil {
		t.Error("config should not be nil")
	}

	// Test with config
	config := &ProviderConfig{
		ExistingKubeconfig: "/custom/path",
	}
	p = NewExistingProvider(config)
	if p.config.ExistingKubeconfig != "/custom/path" {
		t.Errorf("ExistingKubeconfig = %s, want /custom/path", p.config.ExistingKubeconfig)
	}
}

func TestExistingProvider_Name(t *testing.T) {
	p := NewExistingProvider(nil)
	if p.Name() != "existing" {
		t.Errorf("Name() = %s, want existing", p.Name())
	}
}

func TestExistingProvider_Exists(t *testing.T) {
	ctx := context.Background()

	// Create a temp kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	if err := os.WriteFile(kubeconfigPath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create temp kubeconfig: %v", err)
	}

	// Test with existing file
	p := NewExistingProvider(&ProviderConfig{
		ExistingKubeconfig: kubeconfigPath,
	})
	exists, err := p.Exists(ctx, "test-cluster")
	if err != nil {
		t.Fatalf("Exists() failed: %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}

	// Test with non-existing file
	p = NewExistingProvider(&ProviderConfig{
		ExistingKubeconfig: "/nonexistent/kubeconfig",
	})
	exists, err = p.Exists(ctx, "test-cluster")
	if err != nil {
		t.Fatalf("Exists() failed: %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false")
	}
}

func TestExistingProvider_Create(t *testing.T) {
	ctx := context.Background()

	// Create a temp kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	if err := os.WriteFile(kubeconfigPath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create temp kubeconfig: %v", err)
	}

	// Test with existing kubeconfig
	p := NewExistingProvider(&ProviderConfig{
		ExistingKubeconfig: kubeconfigPath,
	})
	err := p.Create(ctx, "test-cluster")
	if err != nil {
		t.Errorf("Create() failed: %v", err)
	}

	// Test with non-existing kubeconfig
	p = NewExistingProvider(&ProviderConfig{
		ExistingKubeconfig: "/nonexistent/kubeconfig",
	})
	err = p.Create(ctx, "test-cluster")
	if err == nil {
		t.Error("Create() should fail for non-existent kubeconfig")
	}
}

func TestExistingProvider_Delete(t *testing.T) {
	ctx := context.Background()
	p := NewExistingProvider(nil)

	// Delete should be a no-op
	err := p.Delete(ctx, "test-cluster")
	if err != nil {
		t.Errorf("Delete() failed: %v", err)
	}
}

func TestExistingProvider_GetKubeconfig(t *testing.T) {
	ctx := context.Background()

	// Create a temp kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	content := []byte("apiVersion: v1\nkind: Config")
	if err := os.WriteFile(kubeconfigPath, content, 0600); err != nil {
		t.Fatalf("Failed to create temp kubeconfig: %v", err)
	}

	p := NewExistingProvider(&ProviderConfig{
		ExistingKubeconfig: kubeconfigPath,
	})

	kubeconfig, err := p.GetKubeconfig(ctx, "test-cluster")
	if err != nil {
		t.Fatalf("GetKubeconfig() failed: %v", err)
	}
	if string(kubeconfig) != string(content) {
		t.Errorf("GetKubeconfig() = %s, want %s", kubeconfig, content)
	}
}

func TestExistingProvider_GetKubeconfigPath(t *testing.T) {
	ctx := context.Background()

	// Create a temp kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	if err := os.WriteFile(kubeconfigPath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create temp kubeconfig: %v", err)
	}

	p := NewExistingProvider(&ProviderConfig{
		ExistingKubeconfig: kubeconfigPath,
	})

	path, err := p.GetKubeconfigPath(ctx, "test-cluster")
	if err != nil {
		t.Fatalf("GetKubeconfigPath() failed: %v", err)
	}
	if path != kubeconfigPath {
		t.Errorf("GetKubeconfigPath() = %s, want %s", path, kubeconfigPath)
	}
}

func TestExistingProvider_getKubeconfigPath_default(t *testing.T) {
	// Test with empty config (should use default)
	p := NewExistingProvider(&ProviderConfig{})

	// Save and clear KUBECONFIG env var
	oldKubeconfig := os.Getenv("KUBECONFIG")
	os.Unsetenv("KUBECONFIG")
	defer func() {
		if oldKubeconfig != "" {
			os.Setenv("KUBECONFIG", oldKubeconfig)
		}
	}()

	path := p.getKubeconfigPath()
	homeDir, _ := os.UserHomeDir()
	expectedDefault := filepath.Join(homeDir, ".kube", "config")
	if path != expectedDefault {
		t.Errorf("getKubeconfigPath() = %s, want %s", path, expectedDefault)
	}
}

func TestExistingProvider_getKubeconfigPath_env(t *testing.T) {
	// Test with KUBECONFIG env var
	p := NewExistingProvider(&ProviderConfig{})

	// Set KUBECONFIG env var
	oldKubeconfig := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", "/env/kubeconfig")
	defer func() {
		if oldKubeconfig != "" {
			os.Setenv("KUBECONFIG", oldKubeconfig)
		} else {
			os.Unsetenv("KUBECONFIG")
		}
	}()

	path := p.getKubeconfigPath()
	if path != "/env/kubeconfig" {
		t.Errorf("getKubeconfigPath() = %s, want /env/kubeconfig", path)
	}
}
