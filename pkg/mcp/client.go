package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/log"
)

// Client manages MCP server connections
type Client struct {
	servers     map[string]*ServerConnection
	mu          sync.RWMutex
	OnReconnect func(serverName string) // called after successful reconnect; used to re-register tools
}

// ServerConnection represents a connection to an MCP server
type ServerConnection struct {
	config    config.MCPServer
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	scanner   *bufio.Scanner
	stderrBuf *bytes.Buffer // captures MCP server stderr for debugging crashes
	reqID     atomic.Int64
	mu        sync.Mutex // protects stdin writes and lifecycle (Close)
	tools     []Tool
	ready     bool

	// A single readLoop goroutine owns the scanner for the connection's
	// lifetime and routes responses to per-request channels by ID. This
	// avoids the per-request reader goroutines that raced on the scanner
	// and consumed each other's responses after a timeout.
	readOnce  sync.Once
	pendingMu sync.Mutex
	pending   map[int64]chan *JSONRPCResponse
	readErr   error // set by readLoop when the connection dies
}

// ensureReader lazily starts the single readLoop goroutine for this
// connection. Called on the first request so that directly-constructed
// connections (e.g. in tests) work without extra wiring.
func (conn *ServerConnection) ensureReader() {
	conn.readOnce.Do(func() {
		conn.pendingMu.Lock()
		if conn.pending == nil {
			conn.pending = make(map[int64]chan *JSONRPCResponse)
		}
		conn.pendingMu.Unlock()
		go conn.readLoop()
	})
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	ServerName  string                 `json:"-"` // Which server provides this tool
}

// JSONRPCRequest represents a JSON-RPC 2.0 request (must include id)
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC 2.0 notification (no id field).
// Per the spec, notifications are one-way messages that must NOT include an id.
type JSONRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
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
	Arguments map[string]interface{} `json:"arguments"`
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

	// Set environment variables: inherit system env, then overlay non-empty
	// config values. Empty config values are skipped so that system-level
	// env vars (e.g. tokens set in .zshrc) are not accidentally overridden.
	cmd.Env = os.Environ()
	for k, v := range serverCfg.Env {
		if v != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
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

	stderrBuf := &bytes.Buffer{}
	cmd.Stderr = stderrBuf

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	// Set scanner buffer to 10MB to handle large MCP responses
	scanner.Buffer(make([]byte, 0, 10*1024*1024), 10*1024*1024)

	conn := &ServerConnection{
		config:    serverCfg,
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		scanner:   scanner,
		stderrBuf: stderrBuf,
		tools:     make([]Tool, 0),
		pending:   make(map[int64]chan *JSONRPCResponse),
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

// isConnectionError returns true if the error indicates a broken connection (e.g. broken pipe)
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "connection closed") || // readLoop's error when the server process exits
		strings.Contains(s, "file already closed") || // write to closed stdin pipe
		errors.Is(err, io.ErrClosedPipe)
}

// CallTool executes a tool on the appropriate server
func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (*CallToolResult, error) {
	result, err := c.callToolOnce(ctx, toolName, args)
	if err == nil {
		return result, nil
	}

	// On connection error, try to reconnect and retry once
	if isConnectionError(err) {
		c.mu.Lock()
		var serverName string
		var serverCfg config.MCPServer
		for name, conn := range c.servers {
			for _, tool := range conn.tools {
				if tool.Name == toolName {
					serverName = name
					serverCfg = conn.config
					break
				}
			}
			if serverName != "" {
				break
			}
		}
		if serverName != "" {
			if conn, exists := c.servers[serverName]; exists {
				conn.Close()
				delete(c.servers, serverName)
			}
		}
		c.mu.Unlock()

		if serverName != "" {
			if reconnectErr := c.Connect(ctx, serverCfg); reconnectErr == nil {
				if c.OnReconnect != nil {
					c.OnReconnect(serverName)
				}
				return c.callToolOnce(ctx, toolName, args)
			}
		}
	}

	return nil, err
}

// callToolOnce executes a tool on the appropriate server (no retry)
func (c *Client) callToolOnce(ctx context.Context, toolName string, args map[string]interface{}) (*CallToolResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

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

// readLoop is the single owner of the connection's scanner. It runs for the
// lifetime of the connection, skipping notifications (messages with "method"
// but no "id" — e.g. tools/list_changed from kubernetes-mcp-server) and
// dispatching responses to the pending request channels by ID. When the
// connection dies it fails all pending requests so callers never hang.
func (conn *ServerConnection) readLoop() {
	var readErr error

	for conn.scanner.Scan() {
		line := conn.scanner.Bytes()

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			// Malformed line is a protocol violation — treat as fatal.
			readErr = fmt.Errorf("failed to unmarshal: %w", err)
			break
		}
		if _, hasMethod := raw["method"]; hasMethod {
			if _, hasID := raw["id"]; !hasID {
				continue // notification — skip
			}
		}

		var resp JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			readErr = fmt.Errorf("failed to unmarshal response: %w", err)
			break
		}

		conn.pendingMu.Lock()
		ch, ok := conn.pending[resp.ID]
		if ok {
			delete(conn.pending, resp.ID)
		}
		conn.pendingMu.Unlock()
		if ok {
			ch <- &resp // buffered, never blocks
		}
	}

	// Connection closed or scanner error — fail all pending requests.
	if readErr == nil && conn.scanner.Err() != nil {
		readErr = fmt.Errorf("scanner error: %w", conn.scanner.Err())
	}
	if readErr == nil {
		msg := "connection closed"
		if conn.stderrBuf != nil && conn.stderrBuf.Len() > 0 {
			stderr := strings.TrimSpace(conn.stderrBuf.String())
			msg = fmt.Sprintf("connection closed (MCP server stderr: %s)", stderr)
			// Always print to stderr so user sees it in terminal
			fmt.Fprintf(os.Stderr, "[k13d] MCP server %s exited unexpectedly. stderr:\n%s\n", conn.config.Name, stderr)
			log.Debugf("MCP server %s exited, stderr: %s", conn.config.Name, stderr)
		} else {
			fmt.Fprintf(os.Stderr, "[k13d] MCP server %s connection closed (no stderr output)\n", conn.config.Name)
		}
		readErr = fmt.Errorf("%s", msg)
	}

	conn.pendingMu.Lock()
	conn.readErr = readErr
	for id, ch := range conn.pending {
		delete(conn.pending, id)
		close(ch) // receiving from closed channel signals connection failure
	}
	conn.pendingMu.Unlock()
}

// sendRequest sends a JSON-RPC request and waits for the matching response
// routed by readLoop. Safe for concurrent use.
func (conn *ServerConnection) sendRequest(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	conn.ensureReader()

	// Ensure context has a timeout so callers never wait forever
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

	// Register the pending request before writing so the response can't be
	// missed even if the server replies immediately.
	respChan := make(chan *JSONRPCResponse, 1)
	conn.pendingMu.Lock()
	if conn.readErr != nil {
		err := conn.readErr
		conn.pendingMu.Unlock()
		return nil, err
	}
	conn.pending[reqID] = respChan
	conn.pendingMu.Unlock()

	unregister := func() {
		conn.pendingMu.Lock()
		delete(conn.pending, reqID)
		conn.pendingMu.Unlock()
	}

	// Write request with newline delimiter (serialize writes via conn.mu)
	conn.mu.Lock()
	_, err = conn.stdin.Write(append(reqBytes, '\n'))
	conn.mu.Unlock()
	if err != nil {
		unregister()
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	select {
	case <-ctx.Done():
		unregister()
		return nil, ctx.Err()
	case resp, ok := <-respChan:
		if !ok {
			// readLoop closed the channel: connection died
			conn.pendingMu.Lock()
			err := conn.readErr
			conn.pendingMu.Unlock()
			if err == nil {
				err = fmt.Errorf("connection closed")
			}
			return nil, err
		}
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

	// Send initialized notification (conn.mu serializes stdin writes)
	conn.mu.Lock()
	notif := JSONRPCNotification{
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
	// Some MCP servers crash when "arguments" is missing/null for no-arg tools.
	// Always send an object ({} at minimum).
	if args == nil {
		args = map[string]interface{}{}
	}

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
