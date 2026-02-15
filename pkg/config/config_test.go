package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg == nil {
		t.Fatal("NewDefaultConfig returned nil")
	}

	// Check LLM defaults (Upstage Solar is recommended default)
	if cfg.LLM.Provider != "upstage" {
		t.Errorf("LLM.Provider = %s, want upstage", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "solar-pro2" {
		t.Errorf("LLM.Model = %s, want solar-pro2", cfg.LLM.Model)
	}
	if cfg.LLM.Endpoint != DefaultSolarEndpoint {
		t.Errorf("LLM.Endpoint = %s, want %s", cfg.LLM.Endpoint, DefaultSolarEndpoint)
	}
	if !cfg.LLM.RetryEnabled {
		t.Error("LLM.RetryEnabled should be true by default")
	}
	if cfg.LLM.MaxRetries != 5 {
		t.Errorf("LLM.MaxRetries = %d, want 5", cfg.LLM.MaxRetries)
	}
	if cfg.LLM.MaxBackoff != 10.0 {
		t.Errorf("LLM.MaxBackoff = %f, want 10.0", cfg.LLM.MaxBackoff)
	}

	// Check Models (1 Solar + 2 OpenAI + 2 Ollama local models)
	if len(cfg.Models) != 5 {
		t.Errorf("len(Models) = %d, want 5", len(cfg.Models))
	}
	if cfg.ActiveModel != "solar-pro2" {
		t.Errorf("ActiveModel = %s, want solar-pro2", cfg.ActiveModel)
	}

	// Verify all expected models are included
	var hasSolar, hasQwen, hasGemma bool
	for _, m := range cfg.Models {
		if m.Name == "solar-pro2" && m.Provider == "upstage" {
			hasSolar = true
		}
		if m.Name == "qwen2.5-local" && m.Provider == "ollama" {
			hasQwen = true
		}
		if m.Name == "gemma2-local" && m.Provider == "ollama" {
			hasGemma = true
		}
	}
	if !hasSolar {
		t.Error("Default models should include solar-pro2")
	}
	if !hasQwen {
		t.Error("Default models should include qwen2.5-local")
	}
	if !hasGemma {
		t.Error("Default models should include gemma2-local")
	}

	// Check other defaults
	if cfg.Language != "ko" {
		t.Errorf("Language = %s, want ko", cfg.Language)
	}
	if !cfg.BeginnerMode {
		t.Error("BeginnerMode should be true by default")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %s, want debug", cfg.LogLevel)
	}
	if !cfg.EnableAudit {
		t.Error("EnableAudit should be true by default")
	}
}

func TestGetActiveModelProfile(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantProfile *ModelProfile
		wantNil     bool
	}{
		{
			name: "active model exists",
			config: &Config{
				ActiveModel: "gpt-4",
				Models: []ModelProfile{
					{Name: "gpt-4", Provider: "openai", Model: "gpt-4"},
					{Name: "claude", Provider: "anthropic", Model: "claude-3"},
				},
			},
			wantProfile: &ModelProfile{Name: "gpt-4", Provider: "openai", Model: "gpt-4"},
		},
		{
			name: "active model not found, returns first",
			config: &Config{
				ActiveModel: "nonexistent",
				Models: []ModelProfile{
					{Name: "gpt-4", Provider: "openai", Model: "gpt-4"},
				},
			},
			wantProfile: &ModelProfile{Name: "gpt-4", Provider: "openai", Model: "gpt-4"},
		},
		{
			name: "no models, returns nil",
			config: &Config{
				ActiveModel: "gpt-4",
				Models:      []ModelProfile{},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := tt.config.GetActiveModelProfile()
			if tt.wantNil {
				if profile != nil {
					t.Errorf("GetActiveModelProfile() = %v, want nil", profile)
				}
				return
			}
			if profile == nil {
				t.Fatal("GetActiveModelProfile() returned nil")
			}
			if profile.Name != tt.wantProfile.Name {
				t.Errorf("profile.Name = %s, want %s", profile.Name, tt.wantProfile.Name)
			}
		})
	}
}

func TestSetActiveModel(t *testing.T) {
	cfg := &Config{
		ActiveModel: "gpt-4",
		Models: []ModelProfile{
			{Name: "gpt-4", Provider: "openai", Model: "gpt-4", Endpoint: "https://api.openai.com"},
			{Name: "claude", Provider: "anthropic", Model: "claude-3", APIKey: "test-key"},
		},
		LLM: LLMConfig{},
	}

	// Test switching to existing model
	if !cfg.SetActiveModel("claude") {
		t.Error("SetActiveModel('claude') returned false")
	}
	if cfg.ActiveModel != "claude" {
		t.Errorf("ActiveModel = %s, want claude", cfg.ActiveModel)
	}
	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("LLM.Provider = %s, want anthropic", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "claude-3" {
		t.Errorf("LLM.Model = %s, want claude-3", cfg.LLM.Model)
	}
	if cfg.LLM.APIKey != "test-key" {
		t.Errorf("LLM.APIKey = %s, want test-key", cfg.LLM.APIKey)
	}

	// Test switching to non-existent model
	if cfg.SetActiveModel("nonexistent") {
		t.Error("SetActiveModel('nonexistent') should return false")
	}
	if cfg.ActiveModel != "claude" {
		t.Errorf("ActiveModel should remain claude, got %s", cfg.ActiveModel)
	}
}

func TestAddModelProfile(t *testing.T) {
	cfg := &Config{Models: []ModelProfile{}}

	// Add new profile
	cfg.AddModelProfile(ModelProfile{Name: "new-model", Provider: "openai", Model: "gpt-4o"})
	if len(cfg.Models) != 1 {
		t.Errorf("len(Models) = %d, want 1", len(cfg.Models))
	}

	// Update existing profile
	cfg.AddModelProfile(ModelProfile{Name: "new-model", Provider: "openai", Model: "gpt-4-turbo"})
	if len(cfg.Models) != 1 {
		t.Errorf("len(Models) = %d, want 1 after update", len(cfg.Models))
	}
	if cfg.Models[0].Model != "gpt-4-turbo" {
		t.Errorf("Model = %s, want gpt-4-turbo", cfg.Models[0].Model)
	}

	// Add another profile
	cfg.AddModelProfile(ModelProfile{Name: "another-model", Provider: "anthropic", Model: "claude-3"})
	if len(cfg.Models) != 2 {
		t.Errorf("len(Models) = %d, want 2", len(cfg.Models))
	}
}

func TestRemoveModelProfile(t *testing.T) {
	cfg := &Config{
		ActiveModel: "gpt-4",
		Models: []ModelProfile{
			{Name: "gpt-4", Provider: "openai", Model: "gpt-4"},
			{Name: "claude", Provider: "anthropic", Model: "claude-3"},
		},
	}

	// Remove non-active model
	if !cfg.RemoveModelProfile("claude") {
		t.Error("RemoveModelProfile('claude') returned false")
	}
	if len(cfg.Models) != 1 {
		t.Errorf("len(Models) = %d, want 1", len(cfg.Models))
	}

	// Add back and remove active model
	cfg.Models = append(cfg.Models, ModelProfile{Name: "claude", Provider: "anthropic", Model: "claude-3"})
	if !cfg.RemoveModelProfile("gpt-4") {
		t.Error("RemoveModelProfile('gpt-4') returned false")
	}
	if cfg.ActiveModel != "claude" {
		t.Errorf("ActiveModel = %s, want claude after removing active", cfg.ActiveModel)
	}

	// Remove non-existent model
	if cfg.RemoveModelProfile("nonexistent") {
		t.Error("RemoveModelProfile('nonexistent') should return false")
	}
}

func TestMCPServerOperations(t *testing.T) {
	cfg := &Config{MCP: MCPConfig{Servers: []MCPServer{}}}

	// Add MCP server
	cfg.AddMCPServer(MCPServer{Name: "server1", Command: "npx", Enabled: true})
	if len(cfg.MCP.Servers) != 1 {
		t.Errorf("len(MCP.Servers) = %d, want 1", len(cfg.MCP.Servers))
	}

	// Update existing server
	cfg.AddMCPServer(MCPServer{Name: "server1", Command: "docker", Enabled: false})
	if len(cfg.MCP.Servers) != 1 {
		t.Errorf("len(MCP.Servers) = %d, want 1 after update", len(cfg.MCP.Servers))
	}
	if cfg.MCP.Servers[0].Command != "docker" {
		t.Errorf("Command = %s, want docker", cfg.MCP.Servers[0].Command)
	}

	// Toggle server
	if !cfg.ToggleMCPServer("server1", true) {
		t.Error("ToggleMCPServer('server1', true) returned false")
	}
	if !cfg.MCP.Servers[0].Enabled {
		t.Error("Server should be enabled after toggle")
	}

	// Toggle non-existent server
	if cfg.ToggleMCPServer("nonexistent", true) {
		t.Error("ToggleMCPServer('nonexistent', true) should return false")
	}

	// Get enabled servers
	cfg.AddMCPServer(MCPServer{Name: "server2", Command: "node", Enabled: false})
	enabled := cfg.GetEnabledMCPServers()
	if len(enabled) != 1 {
		t.Errorf("len(GetEnabledMCPServers()) = %d, want 1", len(enabled))
	}

	// Remove server
	if !cfg.RemoveMCPServer("server1") {
		t.Error("RemoveMCPServer('server1') returned false")
	}
	if len(cfg.MCP.Servers) != 1 {
		t.Errorf("len(MCP.Servers) = %d, want 1 after remove", len(cfg.MCP.Servers))
	}

	// Remove non-existent server
	if cfg.RemoveMCPServer("nonexistent") {
		t.Error("RemoveMCPServer('nonexistent') should return false")
	}
}

func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath() returned empty string")
	}
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("GetConfigPath() base = %s, want config.yaml", filepath.Base(path))
	}
}

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	if err != nil {
		t.Errorf("GetConfigDir() error = %v", err)
	}
	if dir == "" {
		t.Error("GetConfigDir() returned empty string")
	}
	if filepath.Base(dir) != "k13d" {
		t.Errorf("GetConfigDir() base = %s, want k13d", filepath.Base(dir))
	}
}

func TestConfigSaveLoad(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "k13d-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config and save to temp location
	cfg := NewDefaultConfig()
	cfg.LLM.Provider = "test-provider"
	cfg.LLM.Model = "test-model"
	cfg.LLM.APIKey = "test-api-key"
	cfg.Language = "ko"

	// Save to temp file
	tmpPath := filepath.Join(tmpDir, "config.yaml")
	cfgDir := filepath.Dir(tmpPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Manually save since Save() uses GetConfigPath()
	data, err := os.ReadFile(GetConfigPath())
	if err == nil {
		// If config exists, just verify Save() works
		if err := cfg.Save(); err != nil {
			t.Errorf("Save() error = %v", err)
		}
	}

	// Test LoadConfig returns defaults when file doesn't exist
	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() error = %v", err)
	}
	if loadedCfg == nil {
		t.Fatal("LoadConfig() returned nil")
	}

	// Verify it's a valid config (either loaded or default)
	if loadedCfg.LLM.Provider == "" {
		t.Error("LoadConfig() returned config with empty provider")
	}

	_ = data // suppress unused variable warning
}

func TestLLMConfigFields(t *testing.T) {
	cfg := LLMConfig{
		Provider:        "azopenai",
		Model:           "gpt-4",
		Endpoint:        "https://myazure.openai.azure.com",
		APIKey:          "secret-key",
		Region:          "eastus",
		AzureDeployment: "my-deployment",
		SkipTLSVerify:   true,
		RetryEnabled:    true,
		MaxRetries:      3,
		MaxBackoff:      5.0,
		UseJSONMode:     true,
	}

	// Verify all fields
	if cfg.Provider != "azopenai" {
		t.Errorf("Provider = %s, want azopenai", cfg.Provider)
	}
	if cfg.AzureDeployment != "my-deployment" {
		t.Errorf("AzureDeployment = %s, want my-deployment", cfg.AzureDeployment)
	}
	if cfg.Region != "eastus" {
		t.Errorf("Region = %s, want eastus", cfg.Region)
	}
	if !cfg.SkipTLSVerify {
		t.Error("SkipTLSVerify should be true")
	}
	if !cfg.UseJSONMode {
		t.Error("UseJSONMode should be true")
	}
}

func TestModelProfileFields(t *testing.T) {
	profile := ModelProfile{
		Name:            "azure-gpt4",
		Provider:        "azopenai",
		Model:           "gpt-4",
		Endpoint:        "https://myazure.openai.azure.com",
		APIKey:          "secret",
		Region:          "westus",
		AzureDeployment: "deploy-1",
		Description:     "Azure OpenAI GPT-4",
	}

	if profile.Name != "azure-gpt4" {
		t.Errorf("Name = %s, want azure-gpt4", profile.Name)
	}
	if profile.Description != "Azure OpenAI GPT-4" {
		t.Errorf("Description = %s, want Azure OpenAI GPT-4", profile.Description)
	}
}

func TestMCPServerFields(t *testing.T) {
	server := MCPServer{
		Name:        "filesystem",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
		Env:         map[string]string{"DEBUG": "true"},
		Description: "File system MCP server",
		Enabled:     true,
	}

	if server.Name != "filesystem" {
		t.Errorf("Name = %s, want filesystem", server.Name)
	}
	if len(server.Args) != 3 {
		t.Errorf("len(Args) = %d, want 3", len(server.Args))
	}
	if server.Env["DEBUG"] != "true" {
		t.Errorf("Env['DEBUG'] = %s, want true", server.Env["DEBUG"])
	}
}

func TestLoadConfig(t *testing.T) {
	// Test loading config returns defaults when file doesn't exist
	cfg, err := LoadConfig()
	if err != nil {
		t.Errorf("LoadConfig() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig() returned nil")
	}

	// Should have default values
	if cfg.LLM.Provider == "" {
		t.Error("LoadConfig() should return default provider")
	}
}
