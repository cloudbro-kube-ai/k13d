package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

// ToolNameInstruction is prepended when MCP tools are present.
// MCP tools (pods_list, namespaces_list, etc.) have specific names that LLMs may guess incorrectly.
// Not used for kubectl/bash only - those names are standard and unlikely to be hallucinated.
const ToolNameInstruction = "CRITICAL: Use ONLY the exact tool names from the function schema. Never invent, guess, or abbreviate tool names (e.g. do not use pod_list if the schema says pods_list).\n\n"

const kubectlPathEnvVar = "K13D_KUBECTL_PATH"

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
	mu          sync.RWMutex
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
		Description: "Executes a kubectl command against the user's Kubernetes cluster. Use this tool only when you need to query or modify the state of the user's Kubernetes cluster.\n\nIMPORTANT: Interactive commands are not supported in this environment. This includes:\n- kubectl exec with -it or -ti flags (use non-interactive exec instead)\n- kubectl edit (use kubectl get -o yaml, kubectl patch, or kubectl apply instead)\n- kubectl port-forward (use alternative methods like NodePort or LoadBalancer)\n- kubectl attach (prefer logs or a targeted non-interactive exec command instead)\n\nFor interactive operations, please use these non-interactive alternatives:\n- Instead of 'kubectl edit', use 'kubectl get -o yaml' to view, 'kubectl patch' for targeted changes, or 'kubectl apply' to apply full changes\n- Instead of 'kubectl exec -it', use 'kubectl exec' with a specific command\n- Instead of 'kubectl port-forward', use service types like NodePort or LoadBalancer",
		Type:        ToolTypeKubectl,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The complete kubectl command to execute. Prefer to use heredoc syntax for multi-line commands. Please include the kubectl prefix as well.\n\nIMPORTANT: Do not use interactive commands. Instead:\n- Use 'kubectl get -o yaml', 'kubectl patch', or 'kubectl apply' instead of 'kubectl edit'\n- Use 'kubectl exec' with specific commands instead of 'kubectl exec -it'\n- Use service types like NodePort or LoadBalancer instead of 'kubectl port-forward'\n\nExamples:\nuser: what pods are running in the cluster?\nassistant: kubectl get pods\n\nuser: what is the status of the pod my-pod?\nassistant: kubectl get pod my-pod -o jsonpath='{.status.phase}'\n\nuser: I need to edit the pod configuration\nassistant: # Option 1: Using patch for targeted changes\nkubectl patch pod my-pod --patch '{\"spec\":{\"containers\":[{\"name\":\"main\",\"image\":\"new-image\"}]}}'\n\n# Option 2: Using get and apply for full changes\nkubectl get pod my-pod -o yaml > pod.yaml\n# Edit pod.yaml locally\nkubectl apply -f pod.yaml\n\nuser: I need to execute a command in the pod\nassistant: kubectl exec my-pod -- /bin/sh -c \"your command here\"",
				},
				"namespace": map[string]interface{}{
					"type":        "string",
					"description": "Optional namespace override. If not specified, uses the namespace from the command or current context.",
				},
				"modifies_resource": map[string]interface{}{
					"type":        "string",
					"description": "Whether the command modifies a Kubernetes resource. Allowed values: yes, no, unknown.",
					"enum":        []string{"yes", "no", "unknown"},
				},
			},
			"required": []string{"command"},
		},
	})

	// Bash tool - for general shell commands
	r.Register(&Tool{
		Name:        "bash",
		Description: "Executes a bash command. Use this tool only when you need to execute a shell command and the dedicated kubectl tool or another explicitly available tool cannot do the job.",
		Type:        ToolTypeBash,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The bash command to execute. Do not use bash for kubectl or helm operations that belong in the kubectl tool.",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Timeout in seconds (default: 30)",
				},
				"modifies_resource": map[string]interface{}{
					"type":        "string",
					"description": "Whether the command modifies a Kubernetes resource. Allowed values: yes, no, unknown.",
					"enum":        []string{"yes", "no", "unknown"},
				},
			},
			"required": []string{"command"},
		},
	})
}

// Register adds a tool to the registry
func (r *Registry) Register(tool *Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name] = tool
}

// Unregister removes a tool from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// SetMCPExecutor sets the MCP tool executor
func (r *Registry) SetMCPExecutor(executor MCPToolExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mcpExecutor = executor
}

// RegisterMCPTool registers an MCP-provided tool
func (r *Registry) RegisterMCPTool(name, description, serverName string, inputSchema map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, tool := range r.tools {
		if tool.Type == ToolTypeMCP && tool.ServerName == serverName {
			delete(r.tools, name)
		}
	}
}

// GetMCPTools returns all MCP-provided tools
func (r *Registry) GetMCPTools() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var mcpTools []*Tool
	for _, tool := range r.tools {
		if tool.Type == ToolTypeMCP {
			mcpTools = append(mcpTools, tool)
		}
	}
	sort.Slice(mcpTools, func(i, j int) bool {
		return mcpTools[i].Name < mcpTools[j].Name
	})
	return mcpTools
}

// Get returns a tool by name
func (r *Registry) Get(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools
func (r *Registry) List() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tools := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})
	return tools
}

// ToOpenAIFormat returns tools in OpenAI function calling format
func (r *Registry) ToOpenAIFormat() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]map[string]interface{}, 0, len(r.tools))
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		tool := r.tools[name]
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
	r.mu.RLock()
	tool, ok := r.tools[call.Function.Name]
	mcpExec := r.mcpExecutor
	r.mu.RUnlock()

	if !ok {
		return &ToolResult{
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("Unknown tool: %s. Use only the exact tool names from the function schema. Retry with the correct name.", call.Function.Name),
			IsError:    true,
		}
	}

	var result string
	var err error

	// Handle MCP tools specially
	if tool.Type == ToolTypeMCP {
		if mcpExec == nil {
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

		result, err = mcpExec.CallTool(ctx, call.Function.Name, args)
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
	kubectlErr  error
	bashPath    string
	timeout     time.Duration
}

// NewExecutor creates a new tool executor
func NewExecutor() *Executor {
	kubectlPath, kubectlErr := resolveKubectlPath()
	return &Executor{
		kubectlPath: kubectlPath,
		kubectlErr:  kubectlErr,
		bashPath:    "/bin/bash",
		timeout:     30 * time.Second,
	}
}

func resolveKubectlPath() (string, error) {
	return resolveKubectlPathWith(strings.TrimSpace(os.Getenv(kubectlPathEnvVar)), exec.LookPath)
}

func resolveKubectlPathWith(override string, lookPath func(string) (string, error)) (string, error) {
	if override != "" {
		resolved, err := lookPath(override)
		if err != nil {
			return "", fmt.Errorf("%s is set to %q but that binary is not executable: %w", kubectlPathEnvVar, override, err)
		}
		return resolved, nil
	}

	candidates := []string{
		"kubectl",
		"microk8s.kubectl",
		"/usr/local/bin/kubectl",
		"/usr/bin/kubectl",
		"/bin/kubectl",
		"/snap/bin/kubectl",
		"/snap/bin/microk8s.kubectl",
		"/opt/homebrew/bin/kubectl",
		"/usr/local/bin/microk8s.kubectl",
	}
	for _, candidate := range candidates {
		if resolved, err := lookPath(candidate); err == nil {
			return resolved, nil
		}
	}

	return "", fmt.Errorf("kubectl binary not found in PATH or common locations; install kubectl or set %s to its absolute path", kubectlPathEnvVar)
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

// executeKubectl runs a kubectl command using explicit argument arrays (no shell injection)
func (e *Executor) executeKubectl(ctx context.Context, argsJSON string) (string, error) {
	var args KubectlArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid kubectl arguments: %w", err)
	}

	if err := ValidateKubectlToolCommand(args.Command); err != nil {
		return "", err
	}

	// Parse the command into individual arguments
	cmdStr := args.Command
	// Strip "kubectl" prefix if present
	if strings.HasPrefix(cmdStr, "kubectl ") {
		cmdStr = strings.TrimPrefix(cmdStr, "kubectl ")
	} else if cmdStr == "kubectl" {
		cmdStr = ""
	}
	cmdStr = strings.TrimSpace(cmdStr)

	cmdArgs := strings.Fields(cmdStr)

	// Add namespace if specified and not already in command
	if args.Namespace != "" {
		hasNamespace := false
		for _, arg := range cmdArgs {
			if arg == "-n" || arg == "--namespace" || strings.HasPrefix(arg, "-n=") || strings.HasPrefix(arg, "--namespace=") {
				hasNamespace = true
				break
			}
		}
		if !hasNamespace {
			cmdArgs = append([]string{"-n", args.Namespace}, cmdArgs...)
		}
	}

	return e.runKubectlCommand(ctx, cmdArgs, e.timeout)
}

// executeBash runs a bash command
func (e *Executor) executeBash(ctx context.Context, argsJSON string) (string, error) {
	var args BashArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid bash arguments: %w", err)
	}

	if err := ValidateBashToolCommand(args.Command); err != nil {
		return "", err
	}

	timeout := e.timeout
	if args.Timeout > 0 {
		timeout = time.Duration(args.Timeout) * time.Second
	}

	return e.runCommand(ctx, args.Command, timeout)
}

// runKubectlCommand executes kubectl with explicit arguments (no shell interpolation)
func (e *Executor) runKubectlCommand(ctx context.Context, args []string, timeout time.Duration) (string, error) {
	if e.kubectlErr != nil {
		return "", e.kubectlErr
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.kubectlPath, args...)

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

// ValidateKubectlToolCommand rejects kubectl operations that require an interactive terminal.
func ValidateKubectlToolCommand(command string) error {
	normalized := strings.TrimSpace(command)
	if normalized == "" {
		return fmt.Errorf("kubectl command is required")
	}
	if !strings.HasPrefix(normalized, "kubectl ") {
		normalized = "kubectl " + normalized
	}

	switch {
	case strings.Contains(normalized, "kubectl edit "):
		return fmt.Errorf("interactive kubectl edit cannot be approved in unattended agent mode; use kubectl get -o yaml, kubectl patch, or kubectl apply instead")
	case strings.Contains(normalized, "kubectl port-forward "):
		return fmt.Errorf("kubectl port-forward cannot be approved in unattended agent mode; use a Service or ingress-based alternative instead")
	case strings.Contains(normalized, "kubectl attach "):
		return fmt.Errorf("kubectl attach cannot be approved in unattended agent mode; use logs or a targeted non-interactive exec command instead")
	case strings.Contains(normalized, " exec -it "), strings.Contains(normalized, " exec -ti "):
		return fmt.Errorf("interactive kubectl exec cannot be approved in unattended agent mode; run a non-interactive kubectl exec command instead")
	default:
		return nil
	}
}

// ValidateBashToolCommand rejects bash invocations that should be handled through the kubectl tool.
func ValidateBashToolCommand(command string) error {
	normalized := strings.TrimSpace(command)
	if normalized == "" {
		return fmt.Errorf("bash command is required")
	}

	lowered := strings.ToLower(normalized)
	if strings.Contains(lowered, "kubectl ") || strings.Contains(lowered, "helm ") {
		return fmt.Errorf("bash-wrapped Kubernetes or Helm operations cannot be approved; use the dedicated kubectl tool instead")
	}

	return nil
}
