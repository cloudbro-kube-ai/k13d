package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/providers"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/tools"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
)

// Client wraps an LLM provider with additional functionality
type Client struct {
	cfg          *config.LLMConfig
	provider     providers.Provider
	toolRegistry *tools.Registry
}

// NewClient creates a new AI client using the provider factory
func NewClient(cfg *config.LLMConfig) (*Client, error) {
	providerCfg := &providers.ProviderConfig{
		Provider:        cfg.Provider,
		Model:           cfg.Model,
		Endpoint:        cfg.Endpoint,
		APIKey:          cfg.APIKey,
		Region:          cfg.Region,
		AzureDeployment: cfg.AzureDeployment,
		SkipTLSVerify:   cfg.SkipTLSVerify,
		ReasoningEffort: cfg.ReasoningEffort,
	}

	factory := providers.GetFactory()
	provider, err := factory.Create(providerCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Wrap with retry logic if configured
	if cfg.RetryEnabled {
		retryCfg := &providers.RetryConfig{
			MaxAttempts: cfg.MaxRetries,
			MaxBackoff:  cfg.MaxBackoff,
			JitterRatio: 0.1,
		}
		if retryCfg.MaxAttempts == 0 {
			retryCfg.MaxAttempts = 5
		}
		if retryCfg.MaxBackoff == 0 {
			retryCfg.MaxBackoff = 10.0
		}
		provider = providers.CreateWithRetry(provider, retryCfg)
	}

	return &Client{
		cfg:          cfg,
		provider:     provider,
		toolRegistry: tools.NewRegistry(),
	}, nil
}

// Ask sends a prompt to the AI provider and streams the response via callback
func (c *Client) Ask(ctx context.Context, prompt string, callback func(string)) error {
	if c.provider == nil {
		return fmt.Errorf("AI provider not initialized")
	}
	return c.provider.Ask(ctx, prompt, callback)
}

// AskNonStreaming sends a prompt and returns the full response
func (c *Client) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	if c.provider == nil {
		return "", fmt.Errorf("AI provider not initialized")
	}
	return c.provider.AskNonStreaming(ctx, prompt)
}

// ConnectionStatus represents the detailed status of an LLM connection test
type ConnectionStatus struct {
	Connected    bool   `json:"connected"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Endpoint     string `json:"endpoint"`
	ResponseTime int64  `json:"response_time_ms"`
	Error        string `json:"error,omitempty"`
	Message      string `json:"message,omitempty"`
}

// CheckStatus verifies the AI provider is responding
func (c *Client) CheckStatus(ctx context.Context) error {
	_, err := c.AskNonStreaming(ctx, "ping")
	return err
}

// TestConnection performs a detailed connection test and returns status information
func (c *Client) TestConnection(ctx context.Context) *ConnectionStatus {
	status := &ConnectionStatus{
		Connected: false,
		Provider:  c.GetProvider(),
		Model:     c.GetModel(),
		Endpoint:  c.cfg.Endpoint,
	}

	if c.provider == nil {
		status.Error = "AI provider not initialized"
		return status
	}

	if !c.IsReady() {
		status.Error = "AI provider not ready - check API key and endpoint configuration"
		return status
	}

	// Time the request
	start := time.Now()
	_, err := c.AskNonStreaming(ctx, "Say 'OK' if you can hear me.")
	elapsed := time.Since(start)

	status.ResponseTime = elapsed.Milliseconds()

	if err != nil {
		status.Error = err.Error()
		// Provide helpful error messages
		if status.Provider == "openai" && status.Endpoint == "" {
			status.Message = "Using default OpenAI endpoint. Check your API key."
		} else if status.Provider == "ollama" {
			status.Message = "Ensure Ollama is running at " + status.Endpoint
		}
		return status
	}

	status.Connected = true
	status.Message = fmt.Sprintf("Successfully connected to %s (%s)", status.Provider, status.Model)
	return status
}

// GetEndpoint returns the configured endpoint (or default for the provider)
func (c *Client) GetEndpoint() string {
	if c.cfg.Endpoint != "" {
		return c.cfg.Endpoint
	}
	// Return default endpoints based on provider
	switch c.cfg.Provider {
	case "solar":
		return "https://api.upstage.ai/v1"
	case "openai":
		return "https://api.openai.com/v1"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta"
	case "ollama":
		return "http://localhost:11434"
	case "anthropic":
		return "https://api.anthropic.com"
	default:
		return ""
	}
}

// ListModels returns available models from the provider
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	if c.provider == nil {
		return nil, fmt.Errorf("AI provider not initialized")
	}
	return c.provider.ListModels(ctx)
}

// IsReady returns true if the client is configured and ready to use
func (c *Client) IsReady() bool {
	if c == nil || c.provider == nil {
		return false
	}
	return c.provider.IsReady()
}

// GetModel returns the current model name
func (c *Client) GetModel() string {
	if c == nil || c.provider == nil {
		return ""
	}
	return c.provider.GetModel()
}

// GetProvider returns the current provider name
func (c *Client) GetProvider() string {
	if c == nil || c.provider == nil {
		return ""
	}
	return c.provider.Name()
}

// GetAvailableProviders returns a list of available provider names
func GetAvailableProviders() string {
	return providers.GetFactory().ListProviders()
}

// ToolExecutionCallback is called when a tool is executed
// It receives the tool name, command, and result
type ToolExecutionCallback func(toolName string, command string, result string, isError bool)

// AskWithTools sends a prompt with tool calling support (agentic mode)
// The toolApprovalCallback is called before executing each tool for user approval
// Returns error if provider doesn't support tool calling
func (c *Client) AskWithTools(ctx context.Context, prompt string, callback func(string), toolApprovalCallback func(toolName string, args string) bool) error {
	return c.AskWithToolsAndExecution(ctx, prompt, callback, toolApprovalCallback, nil)
}

// AskWithToolsAndExecution is like AskWithTools but also provides tool execution feedback
func (c *Client) AskWithToolsAndExecution(ctx context.Context, prompt string, callback func(string), toolApprovalCallback func(toolName string, args string) bool, toolExecutionCallback ToolExecutionCallback) error {
	if c.provider == nil {
		return fmt.Errorf("AI provider not initialized")
	}

	// Check if provider supports tool calling
	toolProvider, ok := c.provider.(providers.ToolProvider)
	if !ok {
		// Fallback to regular Ask if tool calling not supported
		return c.provider.Ask(ctx, prompt, callback)
	}

	// Convert tool registry to OpenAI format
	toolDefs := make([]providers.ToolDefinition, 0)
	for _, tool := range c.toolRegistry.List() {
		toolDefs = append(toolDefs, providers.ToolDefinition{
			Type: "function",
			Function: providers.FunctionDef{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	// Tool callback that requests approval before execution
	toolCallback := func(call providers.ToolCall) providers.ToolResult {
		// Extract command from arguments
		var args map[string]interface{}
		json.Unmarshal([]byte(call.Function.Arguments), &args)
		command := ""
		if cmd, ok := args["command"].(string); ok {
			command = cmd
		}

		// Request approval if callback provided
		if toolApprovalCallback != nil {
			if !toolApprovalCallback(call.Function.Name, call.Function.Arguments) {
				if toolExecutionCallback != nil {
					toolExecutionCallback(call.Function.Name, command, "Tool execution cancelled by user", true)
				}
				return providers.ToolResult{
					ToolCallID: call.ID,
					Content:    "Tool execution cancelled by user",
					IsError:    true,
				}
			}
		}

		// Convert to tools.ToolCall and execute
		toolCall := &tools.ToolCall{
			ID:   call.ID,
			Type: call.Type,
			Function: tools.ToolCallFunc{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
			},
		}

		result := c.toolRegistry.Execute(ctx, toolCall)

		// Notify about tool execution result
		if toolExecutionCallback != nil {
			toolExecutionCallback(call.Function.Name, command, result.Content, result.IsError)
		}

		return providers.ToolResult{
			ToolCallID: result.ToolCallID,
			Content:    result.Content,
			IsError:    result.IsError,
		}
	}

	return toolProvider.AskWithTools(ctx, prompt, toolDefs, callback, toolCallback)
}

// SupportsTools returns true if the current provider supports tool calling
func (c *Client) SupportsTools() bool {
	if c.provider == nil {
		return false
	}
	_, ok := c.provider.(providers.ToolProvider)
	return ok
}

// GetToolRegistry returns the tool registry for external configuration
func (c *Client) GetToolRegistry() *tools.Registry {
	return c.toolRegistry
}
