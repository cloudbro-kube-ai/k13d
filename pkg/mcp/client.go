package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

// Client manages MCP server connections
type Client struct {
	servers map[string]*ServerConnection
	mu      sync.RWMutex
}

// ServerConnection represents a connection to an MCP server
type ServerConnection struct {
	config  config.MCPServer
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	scanner *bufio.Scanner
	reqID   atomic.Int64
	mu      sync.Mutex
	tools   []Tool
	ready   bool
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	ServerName  string                 `json:"-"` // Which server provides this tool
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// InitializeResult represents the result of the initialize method
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

// Capabilities represents MCP server capabilities
type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// ToolsCapability represents tool capabilities
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerInfo contains server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ListToolsResult represents the result of tools/list
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// CallToolParams represents parameters for tools/call
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult represents the result of tools/call
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a content block in tool result
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// NewClient creates a new MCP client
func NewClient() *Client {
	return &Client{
		servers: make(map[string]*ServerConnection),
	}
}

// Connect starts an MCP server and establishes connection
func (c *Client) Connect(ctx context.Context, serverCfg config.MCPServer) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already connected
	if _, exists := c.servers[serverCfg.Name]; exists {
		return nil
	}

	conn, err := c.startServer(ctx, serverCfg)
	if err != nil {
		return fmt.Errorf("failed to start MCP server %s: %w", serverCfg.Name, err)
	}

	// Initialize the connection
	if err := conn.initialize(ctx); err != nil {
		conn.Close()
		return fmt.Errorf("failed to initialize MCP server %s: %w", serverCfg.Name, err)
	}

	// List available tools
	if err := conn.listTools(ctx); err != nil {
		conn.Close()
		return fmt.Errorf("failed to list tools from MCP server %s: %w", serverCfg.Name, err)
	}

	c.servers[serverCfg.Name] = conn
	return nil
}

// startServer starts the MCP server process
func (c *Client) startServer(ctx context.Context, serverCfg config.MCPServer) (*ServerConnection, error) {
	cmd := exec.CommandContext(ctx, serverCfg.Command, serverCfg.Args...)

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range serverCfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	// Set scanner buffer to 10MB to handle large MCP responses
	scanner.Buffer(make([]byte, 0, 10*1024*1024), 10*1024*1024)

	conn := &ServerConnection{
		config:  serverCfg,
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		scanner: scanner,
		tools:   make([]Tool, 0),
	}

	return conn, nil
}

// Disconnect stops an MCP server connection
func (c *Client) Disconnect(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, exists := c.servers[name]
	if !exists {
		return nil
	}

	err := conn.Close()
	delete(c.servers, name)
	return err
}

// DisconnectAll stops all MCP server connections
func (c *Client) DisconnectAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for name, conn := range c.servers {
		conn.Close()
		delete(c.servers, name)
	}
}

// GetAllTools returns all tools from all connected servers
func (c *Client) GetAllTools() []Tool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var allTools []Tool
	for _, conn := range c.servers {
		allTools = append(allTools, conn.tools...)
	}
	return allTools
}

// CallTool executes a tool on the appropriate server
func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*CallToolResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Find which server has this tool
	for _, conn := range c.servers {
		for _, tool := range conn.tools {
			if tool.Name == toolName {
				return conn.callTool(ctx, toolName, args)
			}
		}
	}

	return nil, fmt.Errorf("tool not found: %s", toolName)
}

// IsConnected checks if a server is connected
func (c *Client) IsConnected(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conn, exists := c.servers[name]
	return exists && conn.ready
}

// GetConnectedServers returns list of connected server names
func (c *Client) GetConnectedServers() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	names := make([]string, 0, len(c.servers))
	for name := range c.servers {
		names = append(names, name)
	}
	return names
}

// ServerConnection methods

// Close shuts down the server connection
func (conn *ServerConnection) Close() error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.ready = false

	if conn.stdin != nil {
		conn.stdin.Close()
	}
	if conn.stdout != nil {
		conn.stdout.Close()
	}
	if conn.cmd != nil && conn.cmd.Process != nil {
		_ = conn.cmd.Process.Kill()
		_ = conn.cmd.Wait()
	}
	return nil
}

// sendRequest sends a JSON-RPC request and waits for response
func (conn *ServerConnection) sendRequest(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	// Ensure context has a timeout to prevent goroutine leaks
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	reqID := conn.reqID.Add(1)
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      reqID,
		Method:  method,
		Params:  params,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Write request with newline delimiter
	if _, err := conn.stdin.Write(append(reqBytes, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response — goroutine will eventually unblock when connection closes.
	// Channels are buffered so the goroutine can write and exit even if select returns first.
	responseChan := make(chan *JSONRPCResponse, 1)
	errChan := make(chan error, 1)

	go func() {
		if conn.scanner.Scan() {
			var resp JSONRPCResponse
			if err := json.Unmarshal(conn.scanner.Bytes(), &resp); err != nil {
				errChan <- fmt.Errorf("failed to unmarshal response: %w", err)
				return
			}
			responseChan <- &resp
		} else {
			if err := conn.scanner.Err(); err != nil {
				errChan <- fmt.Errorf("scanner error: %w", err)
			} else {
				errChan <- fmt.Errorf("connection closed")
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errChan:
		return nil, err
	case resp := <-responseChan:
		if resp.Error != nil {
			return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp, nil
	}
}

// initialize performs the MCP initialization handshake
func (conn *ServerConnection) initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "k13d",
			"version": "1.0.0",
		},
	}

	resp, err := conn.sendRequest(ctx, "initialize", params)
	if err != nil {
		return err
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse initialize result: %w", err)
	}

	// Send initialized notification (acquire lock since sendRequest released it)
	conn.mu.Lock()
	notif := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	notifBytes, err := json.Marshal(notif)
	if err != nil {
		conn.mu.Unlock()
		return fmt.Errorf("failed to marshal notification: %w", err)
	}
	if _, err := conn.stdin.Write(append(notifBytes, '\n')); err != nil {
		conn.mu.Unlock()
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}
	conn.ready = true
	conn.mu.Unlock()
	return nil
}

// listTools retrieves available tools from the server
func (conn *ServerConnection) listTools(ctx context.Context) error {
	resp, err := conn.sendRequest(ctx, "tools/list", nil)
	if err != nil {
		return err
	}

	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse tools list: %w", err)
	}

	// Tag tools with server name
	for i := range result.Tools {
		result.Tools[i].ServerName = conn.config.Name
	}

	conn.tools = result.Tools
	return nil
}

// callTool executes a tool on this server
func (conn *ServerConnection) callTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error) {
	params := CallToolParams{
		Name:      name,
		Arguments: args,
	}

	resp, err := conn.sendRequest(ctx, "tools/call", params)
	if err != nil {
		return nil, err
	}

	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tool result: %w", err)
	}

	return &result, nil
}

// MCPToolExecutorAdapter adapts the MCP Client to the tools.MCPToolExecutor interface
type MCPToolExecutorAdapter struct {
	client *Client
}

// NewMCPToolExecutor creates an adapter that implements tools.MCPToolExecutor
func NewMCPToolExecutor(client *Client) *MCPToolExecutorAdapter {
	return &MCPToolExecutorAdapter{client: client}
}

// CallTool implements tools.MCPToolExecutor
func (a *MCPToolExecutorAdapter) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	result, err := a.client.CallTool(ctx, toolName, args)
	if err != nil {
		return "", err
	}

	// Extract text content from result
	var output strings.Builder
	for _, content := range result.Content {
		if content.Type == "text" && content.Text != "" {
			if output.Len() > 0 {
				output.WriteString("\n")
			}
			output.WriteString(content.Text)
		}
	}

	if result.IsError {
		return output.String(), fmt.Errorf("tool execution failed: %s", output.String())
	}

	return output.String(), nil
}
