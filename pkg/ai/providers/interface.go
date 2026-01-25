package providers

import (
	"context"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Name returns the provider name (e.g., "openai", "gemini", "ollama")
	Name() string

	// Ask sends a prompt and streams the response via callback
	Ask(ctx context.Context, prompt string, callback func(string)) error

	// AskNonStreaming sends a prompt and returns the full response
	AskNonStreaming(ctx context.Context, prompt string) (string, error)

	// IsReady returns true if the provider is configured and ready
	IsReady() bool

	// GetModel returns the current model name
	GetModel() string

	// ListModels returns available models for this provider (optional)
	ListModels(ctx context.Context) ([]string, error)
}

// ToolProvider extends Provider with tool/function calling support
type ToolProvider interface {
	Provider

	// AskWithTools sends a prompt with tools and handles tool calls
	// The toolCallback is called for each tool call, allowing the caller to execute tools
	// Returns the final response after all tool calls are resolved
	AskWithTools(ctx context.Context, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) error
}

// ToolDefinition represents a tool that can be called by the LLM
type ToolDefinition struct {
	Type     string      `json:"type"` // "function"
	Function FunctionDef `json:"function"`
}

// FunctionDef defines a function that can be called
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool invocation from the LLM
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall contains the function name and arguments
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolResult represents the result of executing a tool
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"-"`
}

// ToolCallback is called when the LLM wants to execute a tool
// It should execute the tool and return the result
type ToolCallback func(call ToolCall) ToolResult

// ChatMessage represents a message in a conversation
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ProviderConfig holds common configuration for all providers
type ProviderConfig struct {
	Provider        string `yaml:"provider" json:"provider"`
	Model           string `yaml:"model" json:"model"`
	Endpoint        string `yaml:"endpoint" json:"endpoint"`
	APIKey          string `yaml:"api_key" json:"api_key"`
	Region          string `yaml:"region" json:"region"`                     // For AWS Bedrock
	AzureDeployment string `yaml:"azure_deployment" json:"azure_deployment"` // For Azure OpenAI
	SkipTLSVerify   bool   `yaml:"skip_tls_verify" json:"skip_tls_verify"`
	ReasoningEffort string `yaml:"reasoning_effort" json:"reasoning_effort"` // For Solar Pro2: "minimal" or "high"
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts int     `yaml:"max_attempts" json:"max_attempts"`
	MaxBackoff  float64 `yaml:"max_backoff" json:"max_backoff"`   // seconds
	JitterRatio float64 `yaml:"jitter_ratio" json:"jitter_ratio"` // 0.0 - 1.0
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 5,
		MaxBackoff:  10.0,
		JitterRatio: 0.1,
	}
}
