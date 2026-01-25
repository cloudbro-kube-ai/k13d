package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ToolType represents the type of tool
type ToolType string

const (
	ToolTypeKubectl ToolType = "kubectl"
	ToolTypeBash    ToolType = "bash"
	ToolTypeRead    ToolType = "read_file"
	ToolTypeWrite   ToolType = "write_file"
	ToolTypeMCP     ToolType = "mcp" // MCP server provided tool
)

// Tool represents an MCP-compatible tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	Type        ToolType               `json:"-"`
	ServerName  string                 `json:"-"` // For MCP tools: which server provides this
}

// ToolCall represents a tool invocation request from the LLM
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolCallFunc `json:"function"`
}

// ToolCallFunc represents the function part of a tool call
type ToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error"`
}

// KubectlArgs represents arguments for kubectl tool
type KubectlArgs struct {
	Command   string `json:"command"`
	Namespace string `json:"namespace,omitempty"`
}

// BashArgs represents arguments for bash tool
type BashArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"` // seconds
}

// MCPToolExecutor interface for executing MCP tools
type MCPToolExecutor interface {
	CallTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error)
}

// Registry holds all available tools
type Registry struct {
	tools       map[string]*Tool
	executor    *Executor
	mcpExecutor MCPToolExecutor
}

// NewRegistry creates a new tool registry with default tools
func NewRegistry() *Registry {
	r := &Registry{
		tools:    make(map[string]*Tool),
		executor: NewExecutor(),
	}
	r.registerDefaultTools()
	return r
}

// registerDefaultTools registers the default MCP tools
func (r *Registry) registerDefaultTools() {
	// Kubectl tool - primary tool for Kubernetes operations
	r.Register(&Tool{
		Name:        "kubectl",
		Description: "Execute kubectl commands to manage Kubernetes resources. Use this for all Kubernetes operations like get, describe, create, apply, delete, scale, logs, exec, etc.",
		Type:        ToolTypeKubectl,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The kubectl command to execute (without 'kubectl' prefix). Examples: 'get pods -n default', 'describe deployment nginx', 'logs pod/nginx -f'",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "Optional namespace override. If not specified, uses the namespace from the command or current context.",
				},
			},
			"required": []string{"command"},
		},
	})

	// Bash tool - for general shell commands
	r.Register(&Tool{
		Name:        "bash",
		Description: "Execute bash shell commands. Use for non-kubectl operations like file operations, curl, jq, etc. Be cautious with destructive commands.",
		Type:        ToolTypeBash,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The bash command to execute",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Timeout in seconds (default: 30)",
				},
			},
			"required": []string{"command"},
		},
	})
}

// Register adds a tool to the registry
func (r *Registry) Register(tool *Tool) {
	r.tools[tool.Name] = tool
}

// Unregister removes a tool from the registry
func (r *Registry) Unregister(name string) {
	delete(r.tools, name)
}

// SetMCPExecutor sets the MCP tool executor
func (r *Registry) SetMCPExecutor(executor MCPToolExecutor) {
	r.mcpExecutor = executor
}

// RegisterMCPTool registers an MCP-provided tool
func (r *Registry) RegisterMCPTool(name, description, serverName string, inputSchema map[string]interface{}) {
	r.tools[name] = &Tool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
		Type:        ToolTypeMCP,
		ServerName:  serverName,
	}
}

// UnregisterMCPTools removes all tools from a specific MCP server
func (r *Registry) UnregisterMCPTools(serverName string) {
	for name, tool := range r.tools {
		if tool.Type == ToolTypeMCP && tool.ServerName == serverName {
			delete(r.tools, name)
		}
	}
}

// GetMCPTools returns all MCP-provided tools
func (r *Registry) GetMCPTools() []*Tool {
	var mcpTools []*Tool
	for _, tool := range r.tools {
		if tool.Type == ToolTypeMCP {
			mcpTools = append(mcpTools, tool)
		}
	}
	return mcpTools
}

// Get returns a tool by name
func (r *Registry) Get(name string) (*Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools
func (r *Registry) List() []*Tool {
	tools := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// ToOpenAIFormat returns tools in OpenAI function calling format
func (r *Registry) ToOpenAIFormat() []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.InputSchema,
			},
		})
	}
	return result
}

// Execute runs a tool call and returns the result
func (r *Registry) Execute(ctx context.Context, call *ToolCall) *ToolResult {
	tool, ok := r.tools[call.Function.Name]
	if !ok {
		return &ToolResult{
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("Unknown tool: %s", call.Function.Name),
			IsError:    true,
		}
	}

	var result string
	var err error

	// Handle MCP tools specially
	if tool.Type == ToolTypeMCP {
		if r.mcpExecutor == nil {
			return &ToolResult{
				ToolCallID: call.ID,
				Content:    "MCP executor not configured",
				IsError:    true,
			}
		}

		// Parse arguments JSON
		var args map[string]interface{}
		if call.Function.Arguments != "" {
			if parseErr := json.Unmarshal([]byte(call.Function.Arguments), &args); parseErr != nil {
				return &ToolResult{
					ToolCallID: call.ID,
					Content:    fmt.Sprintf("Failed to parse arguments: %v", parseErr),
					IsError:    true,
				}
			}
		}

		result, err = r.mcpExecutor.CallTool(ctx, call.Function.Name, args)
	} else {
		result, err = r.executor.Execute(ctx, tool, call.Function.Arguments)
	}

	if err != nil {
		return &ToolResult{
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("Error executing %s: %v", call.Function.Name, err),
			IsError:    true,
		}
	}

	return &ToolResult{
		ToolCallID: call.ID,
		Content:    result,
		IsError:    false,
	}
}

// Executor handles actual tool execution
type Executor struct {
	kubectlPath string
	bashPath    string
	timeout     time.Duration
}

// NewExecutor creates a new tool executor
func NewExecutor() *Executor {
	return &Executor{
		kubectlPath: "kubectl",
		bashPath:    "/bin/bash",
		timeout:     30 * time.Second,
	}
}

// Execute runs a tool with the given arguments
func (e *Executor) Execute(ctx context.Context, tool *Tool, argsJSON string) (string, error) {
	switch tool.Type {
	case ToolTypeKubectl:
		return e.executeKubectl(ctx, argsJSON)
	case ToolTypeBash:
		return e.executeBash(ctx, argsJSON)
	case ToolTypeMCP:
		// MCP tools are handled separately through the registry's MCPExecutor
		return "", fmt.Errorf("MCP tools must be executed through Registry.Execute")
	default:
		return "", fmt.Errorf("unsupported tool type: %s", tool.Type)
	}
}

// executeKubectl runs a kubectl command
func (e *Executor) executeKubectl(ctx context.Context, argsJSON string) (string, error) {
	var args KubectlArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid kubectl arguments: %w", err)
	}

	// Build the kubectl command
	cmdStr := args.Command
	if !strings.HasPrefix(cmdStr, "kubectl") {
		cmdStr = "kubectl " + cmdStr
	}

	// Add namespace if specified and not already in command
	if args.Namespace != "" && !strings.Contains(cmdStr, "-n ") && !strings.Contains(cmdStr, "--namespace") {
		cmdStr = strings.Replace(cmdStr, "kubectl ", fmt.Sprintf("kubectl -n %s ", args.Namespace), 1)
	}

	return e.runCommand(ctx, cmdStr, e.timeout)
}

// executeBash runs a bash command
func (e *Executor) executeBash(ctx context.Context, argsJSON string) (string, error) {
	var args BashArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid bash arguments: %w", err)
	}

	timeout := e.timeout
	if args.Timeout > 0 {
		timeout = time.Duration(args.Timeout) * time.Second
	}

	return e.runCommand(ctx, args.Command, timeout)
}

// runCommand executes a shell command with timeout
func (e *Executor) runCommand(ctx context.Context, cmdStr string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.bashPath, "-c", cmdStr)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after %v", timeout)
	}

	if err != nil {
		if output == "" {
			return "", err
		}
		// Return output even on error (often contains useful info)
		return output, nil
	}

	return output, nil
}

// ParseToolCalls extracts tool calls from OpenAI response
func ParseToolCalls(data []byte) ([]ToolCall, error) {
	var response struct {
		Choices []struct {
			Message struct {
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
			Delta struct {
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"delta"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	var calls []ToolCall
	for _, choice := range response.Choices {
		calls = append(calls, choice.Message.ToolCalls...)
		calls = append(calls, choice.Delta.ToolCalls...)
	}

	return calls, nil
}
