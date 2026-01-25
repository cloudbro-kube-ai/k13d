package config

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM          LLMConfig      `yaml:"llm" json:"llm"`
	Models       []ModelProfile `yaml:"models" json:"models"`             // Multiple LLM model profiles
	ActiveModel  string         `yaml:"active_model" json:"active_model"` // Currently active model profile name
	MCP          MCPConfig      `yaml:"mcp" json:"mcp"`                   // MCP server configuration
	ReportPath   string         `yaml:"report_path" json:"report_path"`
	EnableAudit  bool           `yaml:"enable_audit" json:"enable_audit"`
	Language     string         `yaml:"language" json:"language"`
	BeginnerMode bool           `yaml:"beginner_mode" json:"beginner_mode"`
	LogLevel     string         `yaml:"log_level" json:"log_level"`
}

type LLMConfig struct {
	Provider        string  `yaml:"provider" json:"provider"`
	Model           string  `yaml:"model" json:"model"`
	Endpoint        string  `yaml:"endpoint" json:"endpoint"`
	APIKey          string  `yaml:"api_key" json:"api_key"`
	Region          string  `yaml:"region" json:"region"`                     // For AWS Bedrock
	AzureDeployment string  `yaml:"azure_deployment" json:"azure_deployment"` // For Azure OpenAI
	SkipTLSVerify   bool    `yaml:"skip_tls_verify" json:"skip_tls_verify"`
	RetryEnabled    bool    `yaml:"retry_enabled" json:"retry_enabled"`
	MaxRetries      int     `yaml:"max_retries" json:"max_retries"`
	MaxBackoff      float64 `yaml:"max_backoff" json:"max_backoff"`           // seconds
	UseJSONMode     bool    `yaml:"use_json_mode" json:"use_json_mode"`       // Fallback for models without tool calling
	ReasoningEffort string  `yaml:"reasoning_effort" json:"reasoning_effort"` // For Solar Pro2: "minimal" (default) or "high"
}

// ModelProfile represents a saved LLM model configuration
type ModelProfile struct {
	Name            string `yaml:"name" json:"name"`                   // Profile name (e.g., "gpt-4-turbo", "claude-3")
	Provider        string `yaml:"provider" json:"provider"`           // Provider type
	Model           string `yaml:"model" json:"model"`                 // Model identifier
	Endpoint        string `yaml:"endpoint" json:"endpoint,omitempty"` // Custom endpoint
	APIKey          string `yaml:"api_key" json:"api_key,omitempty"`   // API key (masked in UI)
	Region          string `yaml:"region" json:"region,omitempty"`     // For AWS Bedrock
	AzureDeployment string `yaml:"azure_deployment" json:"azure_deployment,omitempty"`
	Description     string `yaml:"description" json:"description,omitempty"` // User description
}

// MCPConfig holds MCP server configurations
type MCPConfig struct {
	Servers []MCPServer `yaml:"servers" json:"servers"`
}

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Name        string            `yaml:"name" json:"name"`         // Server identifier
	Command     string            `yaml:"command" json:"command"`   // Executable command (e.g., "npx", "docker")
	Args        []string          `yaml:"args" json:"args"`         // Command arguments
	Env         map[string]string `yaml:"env" json:"env,omitempty"` // Environment variables
	Description string            `yaml:"description" json:"description,omitempty"`
	Enabled     bool              `yaml:"enabled" json:"enabled"` // Whether this server is active
}

func GetConfigPath() string {
	return filepath.Join(xdg.ConfigHome, "k13d", "config.yaml")
}

// GetConfigDir returns the k13d configuration directory
func GetConfigDir() (string, error) {
	dir := filepath.Join(xdg.ConfigHome, "k13d")
	return dir, nil
}

// DefaultOllamaEndpoint is the default Ollama server endpoint
const DefaultOllamaEndpoint = "http://localhost:11434"

// DefaultOllamaModel is the recommended model for low-spec environments (2 cores, 8GB RAM)
// qwen2.5:3b provides excellent multilingual support (Korean included) with tool calling
const DefaultOllamaModel = "qwen2.5:3b"

// DefaultSolarEndpoint is the default Upstage Solar API endpoint
const DefaultSolarEndpoint = "https://api.upstage.ai/v1"

// DefaultSolarModel is the recommended Solar model
const DefaultSolarModel = "solar-pro2"

func NewDefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Provider:     "solar",
			Model:        DefaultSolarModel,
			Endpoint:     DefaultSolarEndpoint,
			RetryEnabled: true,
			MaxRetries:   5,
			MaxBackoff:   10.0,
		},
		Models: []ModelProfile{
			{
				Name:        "solar-pro2",
				Provider:    "solar",
				Model:       DefaultSolarModel,
				Endpoint:    DefaultSolarEndpoint,
				Description: "Upstage Solar Pro2 (Recommended - Best quality/cost balance)",
			},
			{
				Name:        "gpt-4",
				Provider:    "openai",
				Model:       "gpt-4",
				Description: "OpenAI GPT-4",
			},
			{
				Name:        "gpt-4o",
				Provider:    "openai",
				Model:       "gpt-4o",
				Description: "OpenAI GPT-4o (Faster)",
			},
			{
				Name:        "qwen2.5-local",
				Provider:    "ollama",
				Model:       DefaultOllamaModel,
				Endpoint:    DefaultOllamaEndpoint,
				Description: "Qwen2.5 3B (Local/Offline, Korean supported, low-spec friendly)",
			},
			{
				Name:        "gemma2-local",
				Provider:    "ollama",
				Model:       "gemma2:2b",
				Endpoint:    DefaultOllamaEndpoint,
				Description: "Gemma2 2B (Local/Offline, fastest, minimal resources)",
			},
		},
		ActiveModel: "solar-pro2",
		MCP: MCPConfig{
			Servers: []MCPServer{},
		},
		Language:     "ko",
		BeginnerMode: true,
		LogLevel:     "debug",
		ReportPath:   "report.md",
		EnableAudit:  true,
	}
}

// GetActiveModelProfile returns the currently active model profile
func (c *Config) GetActiveModelProfile() *ModelProfile {
	for i := range c.Models {
		if c.Models[i].Name == c.ActiveModel {
			return &c.Models[i]
		}
	}
	// Return first model if active not found
	if len(c.Models) > 0 {
		return &c.Models[0]
	}
	return nil
}

// SetActiveModel switches to a different model profile and updates LLM config
func (c *Config) SetActiveModel(name string) bool {
	for _, m := range c.Models {
		if m.Name == name {
			c.ActiveModel = name
			c.LLM.Provider = m.Provider
			c.LLM.Model = m.Model
			c.LLM.Endpoint = m.Endpoint
			c.LLM.APIKey = m.APIKey
			c.LLM.Region = m.Region
			c.LLM.AzureDeployment = m.AzureDeployment
			return true
		}
	}
	return false
}

// AddModelProfile adds a new model profile
func (c *Config) AddModelProfile(profile ModelProfile) {
	// Check if name already exists, update if so
	for i, m := range c.Models {
		if m.Name == profile.Name {
			c.Models[i] = profile
			return
		}
	}
	c.Models = append(c.Models, profile)
}

// RemoveModelProfile removes a model profile by name
func (c *Config) RemoveModelProfile(name string) bool {
	for i, m := range c.Models {
		if m.Name == name {
			c.Models = append(c.Models[:i], c.Models[i+1:]...)
			// If removed active model, switch to first available
			if c.ActiveModel == name && len(c.Models) > 0 {
				c.SetActiveModel(c.Models[0].Name)
			}
			return true
		}
	}
	return false
}

// GetEnabledMCPServers returns only enabled MCP servers
func (c *Config) GetEnabledMCPServers() []MCPServer {
	var enabled []MCPServer
	for _, s := range c.MCP.Servers {
		if s.Enabled {
			enabled = append(enabled, s)
		}
	}
	return enabled
}

// AddMCPServer adds a new MCP server configuration
func (c *Config) AddMCPServer(server MCPServer) {
	// Check if name already exists, update if so
	for i, s := range c.MCP.Servers {
		if s.Name == server.Name {
			c.MCP.Servers[i] = server
			return
		}
	}
	c.MCP.Servers = append(c.MCP.Servers, server)
}

// RemoveMCPServer removes an MCP server by name
func (c *Config) RemoveMCPServer(name string) bool {
	for i, s := range c.MCP.Servers {
		if s.Name == name {
			c.MCP.Servers = append(c.MCP.Servers[:i], c.MCP.Servers[i+1:]...)
			return true
		}
	}
	return false
}

// ToggleMCPServer enables or disables an MCP server
func (c *Config) ToggleMCPServer(name string, enabled bool) bool {
	for i, s := range c.MCP.Servers {
		if s.Name == name {
			c.MCP.Servers[i].Enabled = enabled
			return true
		}
	}
	return false
}

func LoadConfig() (*Config, error) {
	path := GetConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return NewDefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return NewDefaultConfig(), nil // Fail gracefully to defaults
	}

	cfg := NewDefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return NewDefaultConfig(), nil // Fail gracefully to defaults
	}

	return cfg, nil
}

func (c *Config) Save() error {
	path := GetConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
