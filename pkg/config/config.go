package config

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM           LLMConfig           `yaml:"llm" json:"llm"`
	Models        []ModelProfile      `yaml:"models" json:"models"`               // Multiple LLM model profiles
	ActiveModel   string              `yaml:"active_model" json:"active_model"`   // Currently active model profile name
	MCP           MCPConfig           `yaml:"mcp" json:"mcp"`                     // MCP server configuration
	Storage       StorageConfig       `yaml:"storage" json:"storage"`             // Data storage configuration
	Prometheus    PrometheusConfig    `yaml:"prometheus" json:"prometheus"`       // Prometheus integration configuration
	Authorization AuthorizationConfig `yaml:"authorization" json:"authorization"` // RBAC authorization (Teleport-inspired)
	ReportPath    string              `yaml:"report_path" json:"report_path"`
	EnableAudit   bool                `yaml:"enable_audit" json:"enable_audit"`
	Language      string              `yaml:"language" json:"language"`
	BeginnerMode  bool                `yaml:"beginner_mode" json:"beginner_mode"`
	LogLevel      string              `yaml:"log_level" json:"log_level"`
}

// AuthorizationConfig holds RBAC authorization settings (Teleport-inspired)
type AuthorizationConfig struct {
	// DefaultTUIRole is the role used for TUI users (default: "admin" for backward compat)
	DefaultTUIRole string `yaml:"default_tui_role" json:"default_tui_role"`
	// AccessRequestTTL is the duration for approved access requests
	AccessRequestTTL string `yaml:"access_request_ttl" json:"access_request_ttl"`
	// RequireApprovalFor lists action categories requiring access request (e.g., ["dangerous"])
	RequireApprovalFor []string `yaml:"require_approval_for" json:"require_approval_for"`
	// CustomRoles defines additional role definitions beyond the built-in ones
	CustomRoles []RoleConfig `yaml:"roles" json:"roles"`
	// Impersonation controls K8s impersonation
	Impersonation ImpersonationConfigYAML `yaml:"impersonation" json:"impersonation"`
	// JWT configuration
	JWT JWTConfigYAML `yaml:"jwt" json:"jwt"`
	// ToolApproval controls AI tool execution approval policy
	ToolApproval ToolApprovalPolicy `yaml:"tool_approval" json:"tool_approval"`
}

// ToolApprovalPolicy controls which AI tool commands require user approval
type ToolApprovalPolicy struct {
	// AutoApproveReadOnly allows read-only commands without approval (default: true)
	AutoApproveReadOnly bool `yaml:"auto_approve_read_only" json:"auto_approve_read_only"`
	// RequireApprovalForWrite requires approval for write operations (default: true)
	RequireApprovalForWrite bool `yaml:"require_approval_for_write" json:"require_approval_for_write"`
	// RequireApprovalForUnknown requires approval for unknown/unrecognized commands (default: true)
	RequireApprovalForUnknown bool `yaml:"require_approval_for_unknown" json:"require_approval_for_unknown"`
	// BlockDangerous blocks dangerous commands entirely instead of allowing with approval (default: false)
	BlockDangerous bool `yaml:"block_dangerous" json:"block_dangerous"`
	// BlockedPatterns is a list of regex patterns that should be blocked entirely
	BlockedPatterns []string `yaml:"blocked_patterns" json:"blocked_patterns"`
	// ApprovalTimeoutSeconds is the timeout for waiting for user approval (default: 60)
	ApprovalTimeoutSeconds int `yaml:"approval_timeout_seconds" json:"approval_timeout_seconds"`
}

// DefaultToolApprovalPolicy returns the default tool approval policy
func DefaultToolApprovalPolicy() ToolApprovalPolicy {
	return ToolApprovalPolicy{
		AutoApproveReadOnly:       true,
		RequireApprovalForWrite:   true,
		RequireApprovalForUnknown: true,
		BlockDangerous:            false,
		BlockedPatterns:           []string{},
		ApprovalTimeoutSeconds:    60,
	}
}

// RoleConfig defines a custom RBAC role in config
type RoleConfig struct {
	Name  string       `yaml:"name" json:"name"`
	Allow []RuleConfig `yaml:"allow" json:"allow"`
	Deny  []RuleConfig `yaml:"deny" json:"deny"`
}

// RuleConfig defines a permission rule in config
type RuleConfig struct {
	Resources  []string `yaml:"resources" json:"resources"`
	Actions    []string `yaml:"actions" json:"actions"`
	Namespaces []string `yaml:"namespaces" json:"namespaces"`
}

// ImpersonationConfigYAML is the YAML-friendly impersonation config
type ImpersonationConfigYAML struct {
	Enabled  bool                            `yaml:"enabled" json:"enabled"`
	Mappings map[string]ImpersonationMapping `yaml:"mappings" json:"mappings"`
}

// ImpersonationMapping defines how a role maps to K8s impersonation
type ImpersonationMapping struct {
	User   string   `yaml:"user" json:"user"`
	Groups []string `yaml:"groups" json:"groups"`
}

// JWTConfigYAML is the YAML-friendly JWT config
type JWTConfigYAML struct {
	Secret        string `yaml:"secret" json:"-"`                      // HMAC secret (never exposed in JSON)
	TokenDuration string `yaml:"token_duration" json:"token_duration"` // e.g., "1h"
	RefreshWindow string `yaml:"refresh_window" json:"refresh_window"` // e.g., "15m"
}

// PrometheusConfig holds Prometheus integration settings
type PrometheusConfig struct {
	// ExposeMetrics enables the /metrics endpoint for Prometheus scraping
	ExposeMetrics bool `yaml:"expose_metrics" json:"expose_metrics"`
	// ExternalURL is the URL of an external Prometheus server for querying
	ExternalURL string `yaml:"external_url" json:"external_url"`
	// Username for basic auth to external Prometheus
	Username string `yaml:"username" json:"username"`
	// Password for basic auth to external Prometheus
	Password string `yaml:"password" json:"password"`
	// CollectK8sMetrics enables Kubernetes metrics collection
	CollectK8sMetrics bool `yaml:"collect_k8s_metrics" json:"collect_k8s_metrics"`
	// CollectionInterval in seconds (default: 60)
	CollectionInterval int `yaml:"collection_interval" json:"collection_interval"`
}

// StorageConfig holds data persistence configuration
type StorageConfig struct {
	// Database settings
	DBType     string `yaml:"db_type" json:"db_type"`         // sqlite, postgres, mariadb, mysql
	DBPath     string `yaml:"db_path" json:"db_path"`         // Path for SQLite file (default: ~/.config/k13d/audit.db)
	DBHost     string `yaml:"db_host" json:"db_host"`         // Database host (for postgres/mysql)
	DBPort     int    `yaml:"db_port" json:"db_port"`         // Database port
	DBName     string `yaml:"db_name" json:"db_name"`         // Database name
	DBUser     string `yaml:"db_user" json:"db_user"`         // Database username
	DBPassword string `yaml:"db_password" json:"db_password"` // Database password
	DBSSLMode  string `yaml:"db_ssl_mode" json:"db_ssl_mode"` // SSL mode (for postgres)

	// Data persistence options
	PersistAuditLogs     bool `yaml:"persist_audit_logs" json:"persist_audit_logs"`         // Store audit logs in DB (default: true)
	PersistLLMUsage      bool `yaml:"persist_llm_usage" json:"persist_llm_usage"`           // Store LLM token usage (default: true)
	PersistSecurityScans bool `yaml:"persist_security_scans" json:"persist_security_scans"` // Store security scan results (default: true)
	PersistMetrics       bool `yaml:"persist_metrics" json:"persist_metrics"`               // Store cluster metrics history (default: true)
	PersistSessions      bool `yaml:"persist_sessions" json:"persist_sessions"`             // Store AI conversation sessions (default: true)

	// File-based logging
	EnableAuditFile bool   `yaml:"enable_audit_file" json:"enable_audit_file"` // Write audit logs to text file (default: false)
	AuditFilePath   string `yaml:"audit_file_path" json:"audit_file_path"`     // Path for audit log file

	// Data retention
	AuditRetentionDays    int `yaml:"audit_retention_days" json:"audit_retention_days"`         // Days to keep audit logs (0 = forever)
	MetricsRetentionDays  int `yaml:"metrics_retention_days" json:"metrics_retention_days"`     // Days to keep metrics (default: 30)
	LLMUsageRetentionDays int `yaml:"llm_usage_retention_days" json:"llm_usage_retention_days"` // Days to keep LLM usage (default: 90)
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

// DefaultDBPath returns the default SQLite database path
func DefaultDBPath() string {
	return filepath.Join(xdg.ConfigHome, "k13d", "audit.db")
}

// DefaultAuditFilePath returns the default audit log file path
func DefaultAuditFilePath() string {
	return filepath.Join(xdg.ConfigHome, "k13d", "audit.log")
}

// DefaultSessionsPath returns the default sessions directory
func DefaultSessionsPath() string {
	return filepath.Join(xdg.DataHome, "k13d", "sessions")
}

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
		Storage: StorageConfig{
			DBType:                "sqlite",
			DBPath:                "", // Empty means use default: ~/.config/k13d/audit.db
			PersistAuditLogs:      true,
			PersistLLMUsage:       true,
			PersistSecurityScans:  true,
			PersistMetrics:        true,
			PersistSessions:       true,
			EnableAuditFile:       false,
			AuditFilePath:         "", // Empty means use default: ~/.config/k13d/audit.log
			AuditRetentionDays:    0,  // 0 = keep forever
			MetricsRetentionDays:  30,
			LLMUsageRetentionDays: 90,
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
		Prometheus: PrometheusConfig{
			ExposeMetrics:      false,
			CollectK8sMetrics:  true,
			CollectionInterval: 60,
		},
		Authorization: AuthorizationConfig{
			DefaultTUIRole:     "admin", // Full access by default (backward compatible)
			AccessRequestTTL:   "30m",
			RequireApprovalFor: []string{},
			Impersonation: ImpersonationConfigYAML{
				Enabled: false,
			},
			JWT: JWTConfigYAML{
				TokenDuration: "1h",
				RefreshWindow: "15m",
			},
			ToolApproval: DefaultToolApprovalPolicy(),
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

// GetEffectiveDBPath returns the effective database path
func (c *Config) GetEffectiveDBPath() string {
	if c.Storage.DBPath != "" {
		return c.Storage.DBPath
	}
	return DefaultDBPath()
}

// GetEffectiveAuditFilePath returns the effective audit file path
func (c *Config) GetEffectiveAuditFilePath() string {
	if c.Storage.AuditFilePath != "" {
		return c.Storage.AuditFilePath
	}
	return DefaultAuditFilePath()
}

// IsPersistenceEnabled returns true if any persistence is enabled
func (c *Config) IsPersistenceEnabled() bool {
	return c.EnableAudit && (c.Storage.PersistAuditLogs ||
		c.Storage.PersistLLMUsage ||
		c.Storage.PersistSecurityScans ||
		c.Storage.PersistMetrics)
}

func LoadConfig() (*Config, error) {
	path := GetConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := NewDefaultConfig()
		applyEnvOverrides(cfg)
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		cfg := NewDefaultConfig()
		applyEnvOverrides(cfg)
		return cfg, nil // Fail gracefully to defaults
	}

	cfg := NewDefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		cfg = NewDefaultConfig()
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

// applyEnvOverrides applies K13D_* environment variable overrides.
// Environment variables take precedence over config file values.
// This enables configuration via Docker/K8s environment without a config file.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("K13D_LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("K13D_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("K13D_LLM_ENDPOINT"); v != "" {
		cfg.LLM.Endpoint = v
	}
	if v := os.Getenv("K13D_LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("K13D_JWT_SECRET"); v != "" {
		cfg.Authorization.JWT.Secret = v
	}
	if v := os.Getenv("K13D_DEFAULT_ROLE"); v != "" {
		cfg.Authorization.DefaultTUIRole = v
	}
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
