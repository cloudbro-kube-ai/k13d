package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func useDarwinConfigPathsForTest(t *testing.T) (homeDir string, legacyBase string) {
	t.Helper()

	homeDir = filepath.Join(t.TempDir(), "home")
	legacyBase = filepath.Join(t.TempDir(), "legacy-config-home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(homeDir) error = %v", err)
	}
	if err := os.MkdirAll(legacyBase, 0o755); err != nil {
		t.Fatalf("MkdirAll(legacyBase) error = %v", err)
	}

	t.Setenv("HOME", homeDir)
	t.Setenv("K13D_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	oldUserHomeDirFunc := userHomeDirFunc
	oldXDGConfigHomeFn := xdgConfigHomeFn
	oldRuntimeGOOS := runtimeGOOS

	userHomeDirFunc = func() (string, error) { return homeDir, nil }
	xdgConfigHomeFn = func() string { return legacyBase }
	runtimeGOOS = "darwin"

	t.Cleanup(func() {
		userHomeDirFunc = oldUserHomeDirFunc
		xdgConfigHomeFn = oldXDGConfigHomeFn
		runtimeGOOS = oldRuntimeGOOS
	})

	return homeDir, legacyBase
}

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
	var hasSolar, hasGPTOSS, hasGemma bool
	for _, m := range cfg.Models {
		if m.Name == "solar-pro2" && m.Provider == "upstage" {
			hasSolar = true
		}
		if m.Name == "gpt-oss-local" && m.Provider == "ollama" && m.Model == DefaultOllamaModel {
			hasGPTOSS = true
		}
		if m.Name == "gemma2-local" && m.Provider == "ollama" {
			hasGemma = true
		}
	}
	if !hasSolar {
		t.Error("Default models should include solar-pro2")
	}
	if !hasGPTOSS {
		t.Error("Default models should include gpt-oss-local")
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
	if cfg.Authorization.ToolApproval.AutoApproveReadOnly {
		t.Error("ToolApproval.AutoApproveReadOnly should be false by default")
	}
	if !cfg.Authorization.ToolApproval.RequireApprovalForWrite {
		t.Error("ToolApproval.RequireApprovalForWrite should be true by default")
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
			{Name: "claude", Provider: "anthropic", Model: "claude-3", APIKey: "test-key", SkipTLSVerify: true},
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
	if !cfg.LLM.SkipTLSVerify {
		t.Error("LLM.SkipTLSVerify should be true after switching to claude profile")
	}

	// Test switching back resets SkipTLSVerify
	if !cfg.SetActiveModel("gpt-4") {
		t.Error("SetActiveModel('gpt-4') returned false")
	}
	if cfg.LLM.SkipTLSVerify {
		t.Error("LLM.SkipTLSVerify should be false after switching to gpt-4 profile")
	}

	// Test switching to non-existent model
	if cfg.SetActiveModel("nonexistent") {
		t.Error("SetActiveModel('nonexistent') should return false")
	}
	if cfg.ActiveModel != "gpt-4" {
		t.Errorf("ActiveModel should remain gpt-4, got %s", cfg.ActiveModel)
	}
}

func TestSyncActiveModelProfileFromLLM(t *testing.T) {
	cfg := &Config{
		ActiveModel: "gpt-4o",
		LLM: LLMConfig{
			Provider:        "openai",
			Model:           "gpt-4o",
			Endpoint:        "https://api.openai.com/v1",
			APIKey:          "sk-test",
			Region:          "global",
			AzureDeployment: "ignored",
			SkipTLSVerify:   true,
		},
		Models: []ModelProfile{
			{Name: "gpt-4o", Provider: "openai", Model: "gpt-4"},
			{Name: "other", Provider: "anthropic", Model: "claude-sonnet-4"},
		},
	}

	if !cfg.SyncActiveModelProfileFromLLM() {
		t.Fatal("SyncActiveModelProfileFromLLM() = false, want true")
	}

	active := cfg.Models[0]
	if active.Model != "gpt-4o" {
		t.Errorf("active.Model = %s, want gpt-4o", active.Model)
	}
	if active.Endpoint != "https://api.openai.com/v1" {
		t.Errorf("active.Endpoint = %s, want OpenAI endpoint", active.Endpoint)
	}
	if active.APIKey != "sk-test" {
		t.Errorf("active.APIKey = %s, want sk-test", active.APIKey)
	}
	if !active.SkipTLSVerify {
		t.Error("active.SkipTLSVerify should be true")
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
	homeDir, _ := useDarwinConfigPathsForTest(t)

	got := GetConfigPath()
	want := filepath.Join(homeDir, ".config", "k13d", "config.yaml")
	if got != want {
		t.Errorf("GetConfigPath() = %s, want %s", got, want)
	}
}

func TestGetConfigPathUsesEnvOverride(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "custom-config.yaml")
	t.Setenv("K13D_CONFIG", customPath)

	if got := GetConfigPath(); got != customPath {
		t.Errorf("GetConfigPath() = %s, want %s", got, customPath)
	}
}

func TestGetConfigPathUsesXDGConfigHomeOverride(t *testing.T) {
	homeDir, _ := useDarwinConfigPathsForTest(t)
	xdgHome := filepath.Join(t.TempDir(), "xdg-home")
	t.Setenv("XDG_CONFIG_HOME", xdgHome)
	userHomeDirFunc = func() (string, error) { return homeDir, nil }

	got := GetConfigPath()
	want := filepath.Join(xdgHome, "k13d", "config.yaml")
	if got != want {
		t.Errorf("GetConfigPath() = %s, want %s", got, want)
	}
}

func TestGetRuntimeSourceInfo_ReportsEnvAndFileStatus(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "custom-config.yaml")
	if err := os.WriteFile(customPath, []byte("llm:\n  provider: openai\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("K13D_CONFIG", customPath)
	t.Setenv("K13D_LLM_PROVIDER", "anthropic")
	t.Setenv("K13D_LLM_MODEL", "claude-sonnet")
	t.Setenv("K13D_LLM_API_KEY", "redacted")
	t.Setenv("K13D_DEFAULT_ROLE", "viewer")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg-home"))

	info := GetRuntimeSourceInfo()
	if info.ConfigPath != customPath {
		t.Fatalf("ConfigPath = %s, want %s", info.ConfigPath, customPath)
	}
	if info.ConfigPathSource != "K13D_CONFIG" {
		t.Fatalf("ConfigPathSource = %s, want K13D_CONFIG", info.ConfigPathSource)
	}
	if !info.ConfigFileExists {
		t.Fatal("ConfigFileExists = false, want true")
	}
	if info.XDGConfigHome == "" {
		t.Fatal("XDGConfigHome should be reported when set")
	}

	wantOverrides := []string{
		"K13D_LLM_PROVIDER",
		"K13D_LLM_MODEL",
		"K13D_LLM_API_KEY",
		"K13D_DEFAULT_ROLE",
	}
	if len(info.EnvOverrides) != len(wantOverrides) {
		t.Fatalf("len(EnvOverrides) = %d, want %d (%v)", len(info.EnvOverrides), len(wantOverrides), info.EnvOverrides)
	}
	for i, want := range wantOverrides {
		if info.EnvOverrides[i] != want {
			t.Fatalf("EnvOverrides[%d] = %s, want %s", i, info.EnvOverrides[i], want)
		}
	}
	wantLLMOverrides := []string{
		"K13D_LLM_PROVIDER",
		"K13D_LLM_MODEL",
		"K13D_LLM_API_KEY",
	}
	if len(info.LLMEnvOverrides) != len(wantLLMOverrides) {
		t.Fatalf("len(LLMEnvOverrides) = %d, want %d (%v)", len(info.LLMEnvOverrides), len(wantLLMOverrides), info.LLMEnvOverrides)
	}
	for i, want := range wantLLMOverrides {
		if info.LLMEnvOverrides[i] != want {
			t.Fatalf("LLMEnvOverrides[%d] = %s, want %s", i, info.LLMEnvOverrides[i], want)
		}
	}
}

func TestGetRuntimeSourceInfo_MissingConfigFile(t *testing.T) {
	customPath := filepath.Join(t.TempDir(), "missing-config.yaml")
	t.Setenv("K13D_CONFIG", customPath)
	t.Setenv("K13D_LLM_PROVIDER", "")
	t.Setenv("K13D_LLM_MODEL", "")
	t.Setenv("K13D_LLM_ENDPOINT", "")
	t.Setenv("K13D_LLM_API_KEY", "")
	t.Setenv("K13D_JWT_SECRET", "")
	t.Setenv("K13D_DEFAULT_ROLE", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	info := GetRuntimeSourceInfo()
	if info.ConfigFileExists {
		t.Fatal("ConfigFileExists = true, want false")
	}
	if len(info.EnvOverrides) != 0 {
		t.Fatalf("EnvOverrides = %v, want empty", info.EnvOverrides)
	}
	if len(info.LLMEnvOverrides) != 0 {
		t.Fatalf("LLMEnvOverrides = %v, want empty", info.LLMEnvOverrides)
	}
}

func TestGetRuntimeSourceInfo_UsesMacOSDotConfigSource(t *testing.T) {
	homeDir, _ := useDarwinConfigPathsForTest(t)

	info := GetRuntimeSourceInfo()
	wantPath := filepath.Join(homeDir, ".config", "k13d", "config.yaml")
	if info.ConfigPath != wantPath {
		t.Fatalf("ConfigPath = %s, want %s", info.ConfigPath, wantPath)
	}
	if info.ConfigPathSource != "macOS default (~/.config)" {
		t.Fatalf("ConfigPathSource = %s, want macOS default (~/.config)", info.ConfigPathSource)
	}
}

func TestGetConfigDir(t *testing.T) {
	homeDir, _ := useDarwinConfigPathsForTest(t)

	dir, err := GetConfigDir()
	if err != nil {
		t.Errorf("GetConfigDir() error = %v", err)
	}
	want := filepath.Join(homeDir, ".config", "k13d")
	if dir != want {
		t.Errorf("GetConfigDir() = %s, want %s", dir, want)
	}
}

func TestLoadConfigMigratesLegacyMacOSConfigPath(t *testing.T) {
	homeDir, legacyBase := useDarwinConfigPathsForTest(t)
	legacyPath := filepath.Join(legacyBase, "k13d", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("llm:\n  provider: anthropic\n  model: claude-sonnet-4\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.LLM.Provider != "anthropic" {
		t.Fatalf("LLM.Provider = %s, want anthropic", cfg.LLM.Provider)
	}

	newPath := filepath.Join(homeDir, ".config", "k13d", "config.yaml")
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("ReadFile(newPath) error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("migrated config file is empty")
	}
}

func TestLoadAliasesFallsBackToLegacyMacOSConfigDir(t *testing.T) {
	_, legacyBase := useDarwinConfigPathsForTest(t)
	legacyPath := filepath.Join(legacyBase, "k13d", "aliases.yaml")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("aliases:\n  pp: pods\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	aliases, err := LoadAliases()
	if err != nil {
		t.Fatalf("LoadAliases() error = %v", err)
	}
	if got := aliases.Resolve("pp"); got != "pods" {
		t.Fatalf("aliases.Resolve(pp) = %s, want pods", got)
	}
}

func TestConfigSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "config.yaml")
	t.Setenv("K13D_CONFIG", tmpPath)

	cfg := NewDefaultConfig()
	cfg.LLM.Provider = "test-provider"
	cfg.LLM.Model = "test-model"
	cfg.LLM.APIKey = "test-api-key"
	cfg.Language = "ko"

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loadedCfg == nil {
		t.Fatal("LoadConfig() returned nil")
	}

	if loadedCfg.LLM.Provider != "test-provider" {
		t.Errorf("LoadConfig().LLM.Provider = %s, want test-provider", loadedCfg.LLM.Provider)
	}
	if loadedCfg.LLM.Model != "test-model" {
		t.Errorf("LoadConfig().LLM.Model = %s, want test-model", loadedCfg.LLM.Model)
	}
	if loadedCfg.LLM.APIKey != "test-api-key" {
		t.Errorf("LoadConfig().LLM.APIKey = %s, want test-api-key", loadedCfg.LLM.APIKey)
	}
	if loadedCfg.Language != "ko" {
		t.Errorf("LoadConfig().Language = %s, want ko", loadedCfg.Language)
	}
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
	t.Setenv("K13D_CONFIG", filepath.Join(t.TempDir(), "missing-config.yaml"))

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

func TestSaveCreatesMissingConfigDirAndFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "nested", "k13d", "config.yaml")
	t.Setenv("K13D_CONFIG", configPath)

	cfg := NewDefaultConfig()
	cfg.LLM.Provider = "ollama"
	cfg.LLM.Model = DefaultOllamaModel
	cfg.LLM.Endpoint = DefaultOllamaEndpoint

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("config file mode = %o, want 0600", info.Mode().Perm())
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected saved config file to be non-empty")
	}
}

func TestLoadConfigExpandsEnvPlaceholders(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("K13D_CONFIG", configPath)
	t.Setenv("TEST_K13D_API_KEY", "expanded-secret")
	t.Setenv("TEST_K13D_MODEL", "gpt-4o")

	data := []byte(`
llm:
  provider: openai
  model: ${TEST_K13D_MODEL}
  api_key: ${TEST_K13D_API_KEY}
models:
  - name: openai-prod
    provider: openai
    model: ${TEST_K13D_MODEL}
    api_key: ${TEST_K13D_API_KEY}
active_model: openai-prod
`)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.LLM.Model != "gpt-4o" {
		t.Errorf("cfg.LLM.Model = %s, want gpt-4o", cfg.LLM.Model)
	}
	if cfg.LLM.APIKey != "expanded-secret" {
		t.Errorf("cfg.LLM.APIKey = %s, want expanded-secret", cfg.LLM.APIKey)
	}
	if len(cfg.Models) != 1 || cfg.Models[0].APIKey != "expanded-secret" {
		t.Fatalf("cfg.Models[0].APIKey = %q, want expanded-secret", cfg.Models[0].APIKey)
	}
}

func TestModelProfileYAMLRoundTrip(t *testing.T) {
	// Test that ModelProfile with all fields survives YAML marshal/unmarshal
	original := Config{
		ActiveModel: "ollama-local",
		LLM: LLMConfig{
			Provider:      "ollama",
			Model:         DefaultOllamaModel,
			Endpoint:      "https://ollama.internal:11434",
			SkipTLSVerify: true,
			Temperature:   0.7,
			MaxTokens:     4096,
			MaxIterations: 10,
		},
		Models: []ModelProfile{
			{
				Name:          "ollama-local",
				Provider:      "ollama",
				Model:         DefaultOllamaModel,
				Endpoint:      "https://ollama.internal:11434",
				SkipTLSVerify: true,
				Description:   "Local Ollama with self-signed cert",
			},
			{
				Name:     "openai-prod",
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "sk-test-key",
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}

	// Unmarshal back
	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	// Verify all profiles survived
	if len(loaded.Models) != 2 {
		t.Fatalf("len(Models) = %d, want 2", len(loaded.Models))
	}

	// Verify SkipTLSVerify survived round-trip
	if !loaded.Models[0].SkipTLSVerify {
		t.Error("Models[0].SkipTLSVerify should be true after YAML round-trip")
	}
	if loaded.Models[1].SkipTLSVerify {
		t.Error("Models[1].SkipTLSVerify should be false after YAML round-trip")
	}

	// Verify ActiveModel and LLM survived
	if loaded.ActiveModel != "ollama-local" {
		t.Errorf("ActiveModel = %s, want ollama-local", loaded.ActiveModel)
	}
	if loaded.LLM.SkipTLSVerify != true {
		t.Error("LLM.SkipTLSVerify should survive YAML round-trip")
	}
}

func TestSetActiveModelAppliesAllFields(t *testing.T) {
	// Verify that switching between profiles correctly applies ALL fields,
	// including that switching from a profile with SkipTLSVerify=true to one
	// with SkipTLSVerify=false properly resets the value.
	cfg := &Config{
		ActiveModel: "default",
		Models: []ModelProfile{
			{
				Name:          "internal-ollama",
				Provider:      "ollama",
				Model:         "llama3.2",
				Endpoint:      "https://internal.example.com:11434",
				SkipTLSVerify: true,
				Region:        "",
			},
			{
				Name:            "azure-prod",
				Provider:        "azopenai",
				Model:           "gpt-4",
				Endpoint:        "https://myazure.openai.azure.com",
				APIKey:          "az-key",
				AzureDeployment: "prod-gpt4",
				Region:          "eastus",
				SkipTLSVerify:   false,
			},
			{
				Name:     "openai-simple",
				Provider: "openai",
				Model:    "gpt-4o",
				APIKey:   "sk-key",
			},
		},
		LLM: LLMConfig{},
	}

	// Switch to internal Ollama with TLS skip
	cfg.SetActiveModel("internal-ollama")
	if cfg.LLM.Provider != "ollama" {
		t.Errorf("Provider = %s, want ollama", cfg.LLM.Provider)
	}
	if cfg.LLM.Endpoint != "https://internal.example.com:11434" {
		t.Errorf("Endpoint = %s, want internal endpoint", cfg.LLM.Endpoint)
	}
	if !cfg.LLM.SkipTLSVerify {
		t.Error("SkipTLSVerify should be true for internal-ollama")
	}

	// Switch to Azure — SkipTLSVerify must reset, Azure fields must apply
	cfg.SetActiveModel("azure-prod")
	if cfg.LLM.Provider != "azopenai" {
		t.Errorf("Provider = %s, want azopenai", cfg.LLM.Provider)
	}
	if cfg.LLM.AzureDeployment != "prod-gpt4" {
		t.Errorf("AzureDeployment = %s, want prod-gpt4", cfg.LLM.AzureDeployment)
	}
	if cfg.LLM.Region != "eastus" {
		t.Errorf("Region = %s, want eastus", cfg.LLM.Region)
	}
	if cfg.LLM.SkipTLSVerify {
		t.Error("SkipTLSVerify should be false for azure-prod")
	}
	if cfg.LLM.APIKey != "az-key" {
		t.Errorf("APIKey = %s, want az-key", cfg.LLM.APIKey)
	}

	// Switch to simple OpenAI — stale fields from Azure must be cleared
	cfg.SetActiveModel("openai-simple")
	if cfg.LLM.Provider != "openai" {
		t.Errorf("Provider = %s, want openai", cfg.LLM.Provider)
	}
	if cfg.LLM.AzureDeployment != "" {
		t.Errorf("AzureDeployment = %s, want empty (cleared from previous)", cfg.LLM.AzureDeployment)
	}
	if cfg.LLM.Region != "" {
		t.Errorf("Region = %s, want empty (cleared from previous)", cfg.LLM.Region)
	}
	if cfg.LLM.Endpoint != "" {
		t.Errorf("Endpoint = %s, want empty (cleared from previous)", cfg.LLM.Endpoint)
	}
}

func TestConfigSaveLoadRoundTrip(t *testing.T) {
	// Test that saving and loading config preserves model profiles
	tmpDir, err := os.MkdirTemp("", "k13d-config-roundtrip")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpPath := filepath.Join(tmpDir, "config.yaml")

	cfg := NewDefaultConfig()
	cfg.ActiveModel = "custom-ollama"
	cfg.Models = append(cfg.Models, ModelProfile{
		Name:          "custom-ollama",
		Provider:      "ollama",
		Model:         "mistral:7b",
		Endpoint:      "https://ml.internal:11434",
		SkipTLSVerify: true,
		Description:   "Custom Ollama with TLS skip",
	})
	cfg.SetActiveModel("custom-ollama")

	// Marshal and write
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Read back and unmarshal
	readData, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var loaded Config
	if err := yaml.Unmarshal(readData, &loaded); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	// Verify model profiles persisted
	if loaded.ActiveModel != "custom-ollama" {
		t.Errorf("ActiveModel = %s, want custom-ollama", loaded.ActiveModel)
	}
	if loaded.LLM.Provider != "ollama" {
		t.Errorf("LLM.Provider = %s, want ollama", loaded.LLM.Provider)
	}
	if loaded.LLM.SkipTLSVerify != true {
		t.Error("LLM.SkipTLSVerify should be true after save/load")
	}

	// Find the custom profile
	found := false
	for _, m := range loaded.Models {
		if m.Name == "custom-ollama" {
			found = true
			if m.Endpoint != "https://ml.internal:11434" {
				t.Errorf("Profile endpoint = %s, want https://ml.internal:11434", m.Endpoint)
			}
			if !m.SkipTLSVerify {
				t.Error("Profile SkipTLSVerify should be true")
			}
		}
	}
	if !found {
		t.Error("custom-ollama profile not found in loaded config")
	}

	// Verify switching model on loaded config works
	loaded.SetActiveModel("solar-pro2")
	if loaded.LLM.Provider != "upstage" {
		t.Errorf("After switch, LLM.Provider = %s, want upstage", loaded.LLM.Provider)
	}
	if loaded.LLM.SkipTLSVerify {
		t.Error("After switch to solar-pro2, SkipTLSVerify should be false")
	}
}
