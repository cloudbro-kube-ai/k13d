package mcp

import (
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func TestGetAvailableProfiles(t *testing.T) {
	profiles := GetAvailableProfiles()

	if len(profiles) == 0 {
		t.Fatal("Expected at least one profile")
	}

	expectedProfiles := []string{"k8s", "docker", "shell", "github", "aws", "argocd", "thinking", "fullstack"}
	for _, expected := range expectedProfiles {
		found := false
		for _, p := range profiles {
			if p.ID == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected profile %s not found", expected)
		}
	}
}

func TestGetProfileByID(t *testing.T) {
	tests := []struct {
		id    string
		found bool
	}{
		{"k8s", true},
		{"docker", true},
		{"fullstack", true},
		{"nonexistent", false},
	}

	for _, test := range tests {
		profile := GetProfileByID(test.id)
		if test.found && profile == nil {
			t.Errorf("Expected profile %s to be found", test.id)
		}
		if !test.found && profile != nil {
			t.Errorf("Expected profile %s not to be found", test.id)
		}
	}
}

func TestIsProfileInstalled(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			Servers: []config.MCPServer{
				{
					Name:    "kubernetes",
					Command: "npx",
					Args:    []string{"-y", "@anthropic/mcp-server-kubernetes"},
					Enabled: false,
				},
			},
		},
	}

	// k8s profile should be installed (has kubernetes server)
	if !IsProfileInstalled(cfg, "k8s") {
		t.Error("Expected k8s profile to be installed")
	}

	// docker profile should not be installed (no docker server)
	if IsProfileInstalled(cfg, "docker") {
		t.Error("Expected docker profile not to be installed")
	}

	// fullstack should not be installed (missing servers)
	if IsProfileInstalled(cfg, "fullstack") {
		t.Error("Expected fullstack profile not to be installed")
	}
}

func TestInstallProfile(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			Servers: []config.MCPServer{},
		},
	}

	err := InstallProfile(cfg, "docker")
	if err != nil {
		t.Fatalf("Failed to install docker profile: %v", err)
	}

	// Verify servers were added
	found := false
	for _, srv := range cfg.MCP.Servers {
		if srv.Name == "docker" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected docker server to be installed")
	}
}

func TestInstallProfile_NotFound(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			Servers: []config.MCPServer{},
		},
	}

	err := InstallProfile(cfg, "nonexistent")
	if err != ErrProfileNotFound {
		t.Errorf("Expected ErrProfileNotFound, got %v", err)
	}
}

func TestInstallProfile_FullStack(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			Servers: []config.MCPServer{},
		},
	}

	err := InstallProfile(cfg, "fullstack")
	if err != nil {
		t.Fatalf("Failed to install fullstack profile: %v", err)
	}

	// Should have multiple servers
	if len(cfg.MCP.Servers) < 3 {
		t.Errorf("Expected at least 3 servers, got %d", len(cfg.MCP.Servers))
	}

	// Verify specific servers
	expectedServers := []string{"kubernetes", "docker", "shell", "github"}
	for _, expected := range expectedServers {
		found := false
		for _, srv := range cfg.MCP.Servers {
			if srv.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected server %s to be installed", expected)
		}
	}
}

func TestUninstallProfile(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			Servers: []config.MCPServer{
				{
					Name:    "kubernetes",
					Command: "npx",
					Args:    []string{"-y", "@anthropic/mcp-server-kubernetes"},
					Enabled: false,
				},
				{
					Name:    "docker",
					Command: "npx",
					Args:    []string{"-y", "@modelcontextprotocol/server-docker"},
					Enabled: false,
				},
			},
		},
	}

	err := UninstallProfile(cfg, "k8s")
	if err != nil {
		t.Fatalf("Failed to uninstall k8s profile: %v", err)
	}

	// Verify kubernetes server was removed
	for _, srv := range cfg.MCP.Servers {
		if srv.Name == "kubernetes" {
			t.Error("Expected kubernetes server to be removed")
		}
	}

	// Verify docker server is still there (not part of k8s profile)
	found := false
	for _, srv := range cfg.MCP.Servers {
		if srv.Name == "docker" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected docker server to remain")
	}
}

func TestUninstallProfile_NotFound(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCPConfig{
			Servers: []config.MCPServer{},
		},
	}

	err := UninstallProfile(cfg, "nonexistent")
	if err != ErrProfileNotFound {
		t.Errorf("Expected ErrProfileNotFound, got %v", err)
	}
}

func TestProfileServersConfigured(t *testing.T) {
	profiles := GetAvailableProfiles()

	for _, profile := range profiles {
		if len(profile.Servers) == 0 {
			t.Errorf("Profile %s has no servers configured", profile.ID)
		}

		for _, server := range profile.Servers {
			if server.Name == "" {
				t.Errorf("Profile %s has server with empty name", profile.ID)
			}
			if server.Command == "" {
				t.Errorf("Profile %s server %s has empty command", profile.ID, server.Name)
			}
			if len(server.Args) == 0 {
				t.Errorf("Profile %s server %s has empty args", profile.ID, server.Name)
			}
		}
	}
}

func TestProfileMetadata(t *testing.T) {
	profiles := GetAvailableProfiles()

	for _, profile := range profiles {
		if profile.ID == "" {
			t.Error("Profile has empty ID")
		}
		if profile.Name == "" {
			t.Errorf("Profile %s has empty name", profile.ID)
		}
		if profile.Description == "" {
			t.Errorf("Profile %s has empty description", profile.ID)
		}
		if profile.Category == "" {
			t.Errorf("Profile %s has empty category", profile.ID)
		}
		if len(profile.Tags) == 0 {
			t.Errorf("Profile %s has no tags", profile.ID)
		}
	}
}
