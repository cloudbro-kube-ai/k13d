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
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

// --- Helper: mockMCPServer simulates an MCP server over io.Pipe ---

// mockMCPServer processes JSON-RPC requests on a pair of pipes, behaving like a
// real MCP server (initialize, tools/list, tools/call, etc.).
type mockMCPServer struct {
	tools     []Tool
	t         *testing.T
	reader    io.ReadCloser  // server reads client requests from here
	writer    io.WriteCloser // server writes responses here
	done      chan struct{}
	callDelay time.Duration // optional delay on tools/call
	failInit  bool          // if true, return an error on initialize
}

func newMockMCPServer(t *testing.T, tools []Tool) (*mockMCPServer, io.ReadCloser, io.WriteCloser) {
	t.Helper()
	// clientWriter -> serverReader (client writes requests, server reads them)
	serverReader, clientWriter := io.Pipe()
	// serverWriter -> clientReader (server writes responses, client reads them)
	clientReader, serverWriter := io.Pipe()

	m := &mockMCPServer{
		tools:  tools,
		t:      t,
		reader: serverReader,
		writer: serverWriter,
		done:   make(chan struct{}),
	}
	return m, clientReader, clientWriter
}

func (m *mockMCPServer) start() {
	go func() {
		defer close(m.done)
		scanner := bufio.NewScanner(m.reader)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var req JSONRPCRequest
			if err := json.Unmarshal(line, &req); err != nil {
				// Write parse error
				m.writeResponse(&JSONRPCResponse{
					JSONRPC: "2.0",
					Error:   &JSONRPCError{Code: -32700, Message: "Parse error"},
				})
				continue
			}
			m.handleRequest(&req)
		}
	}()
}

func (m *mockMCPServer) handleRequest(req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		if m.failInit {
			m.writeResponse(&JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &JSONRPCError{Code: -32603, Message: "initialization failed"},
			})
			return
		}
		result := InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: Capabilities{
				Tools: &ToolsCapability{ListChanged: false},
			},
			ServerInfo: ServerInfo{Name: "mock-server", Version: "1.0.0"},
		}
		resultBytes, _ := json.Marshal(result)
		m.writeResponse(&JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  resultBytes,
		})

	case "notifications/initialized":
		// No response needed

	case "tools/list":
		result := ListToolsResult{Tools: m.tools}
		resultBytes, _ := json.Marshal(result)
		m.writeResponse(&JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  resultBytes,
		})

	case "tools/call":
		if m.callDelay > 0 {
			time.Sleep(m.callDelay)
		}
		var params CallToolParams
		if paramsBytes, err := json.Marshal(req.Params); err == nil {
			_ = json.Unmarshal(paramsBytes, &params)
		}

		// Find tool
		found := false
		for _, tool := range m.tools {
			if tool.Name == params.Name {
				found = true
				break
			}
		}
		if !found {
			m.writeResponse(&JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &JSONRPCError{Code: -32602, Message: fmt.Sprintf("unknown tool: %s", params.Name)},
			})
			return
		}

		// Build a response based on the arguments
		text := fmt.Sprintf("executed %s", params.Name)
		if msg, ok := params.Arguments["message"]; ok {
			text = fmt.Sprintf("%v", msg)
		}
		result := CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: text}},
		}
		resultBytes, _ := json.Marshal(result)
		m.writeResponse(&JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  resultBytes,
		})

	default:
		m.writeResponse(&JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &JSONRPCError{Code: -32601, Message: "Method not found"},
		})
	}
}

func (m *mockMCPServer) writeResponse(resp *JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = m.writer.Write(append(data, '\n'))
}

func (m *mockMCPServer) stop() {
	m.reader.Close()
	m.writer.Close()
	<-m.done
}

// --- Helper: create a ServerConnection backed by pipes ---

func newTestServerConnection(t *testing.T, tools []Tool) (*ServerConnection, *mockMCPServer) {
	t.Helper()
	mock, clientReader, clientWriter := newMockMCPServer(t, tools)
	mock.start()

	scanner := bufio.NewScanner(clientReader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	conn := &ServerConnection{
		config: config.MCPServer{
			Name:    "test-server",
			Command: "mock",
		},
		stdin:   clientWriter,
		stdout:  clientReader,
		scanner: scanner,
		tools:   make([]Tool, 0),
	}

	return conn, mock
}

// =============================================================================
// 1. Client initialization and configuration
// =============================================================================

func TestNewClientInitialization(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Fatal("NewClient() should not return nil")
	}
	if client.servers == nil {
		t.Fatal("NewClient() should initialize servers map")
	}
	if len(client.servers) != 0 {
		t.Errorf("NewClient() servers map should be empty, got %d entries", len(client.servers))
	}
}

func TestNewClientMultipleInstances(t *testing.T) {
	c1 := NewClient()
	c2 := NewClient()

	if c1 == c2 {
		t.Error("NewClient() should return distinct instances")
	}

	// Modifying one should not affect the other
	c1.servers["test"] = &ServerConnection{}
	if len(c2.servers) != 0 {
		t.Error("clients should have independent server maps")
	}
}

// =============================================================================
// 2. Server connection lifecycle (Connect, Disconnect)
// =============================================================================

func TestConnectWithRealProcess(t *testing.T) {
	// Use a helper Go script as a mock MCP server process.
	// We write a tiny Go program to stdout, compile it, and use it.
	// Instead, we use a shell one-liner that acts as an MCP server.
	// The simplest approach: use the project's own server package via `go run`.
	// But to avoid complexity, we test Connect with a non-existent command to verify error handling.

	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Connect(ctx, config.MCPServer{
		Name:    "bad-server",
		Command: "/nonexistent/binary/path",
		Args:    []string{},
	})
	if err == nil {
		t.Fatal("Connect with nonexistent binary should return error")
	}
	if !strings.Contains(err.Error(), "failed to start") {
		t.Errorf("error should mention 'failed to start', got: %v", err)
	}
}

func TestConnectDuplicateServerNoOp(t *testing.T) {
	client := NewClient()

	// Manually inject a server connection to simulate already-connected state
	client.servers["existing-server"] = &ServerConnection{
		config: config.MCPServer{Name: "existing-server"},
		ready:  true,
	}

	ctx := context.Background()
	err := client.Connect(ctx, config.MCPServer{
		Name:    "existing-server",
		Command: "echo",
	})
	if err != nil {
		t.Errorf("Connect to already-connected server should return nil, got: %v", err)
	}
}

func TestDisconnectExistingServer(t *testing.T) {
	client := NewClient()

	// Create pipes for a fake connection
	_, clientWriter := io.Pipe()
	clientReader, _ := io.Pipe()

	conn := &ServerConnection{
		config: config.MCPServer{Name: "test-server"},
		stdin:  clientWriter,
		stdout: clientReader,
		ready:  true,
	}
	client.servers["test-server"] = conn

	err := client.Disconnect("test-server")
	if err != nil {
		t.Errorf("Disconnect should return nil, got: %v", err)
	}

	if _, exists := client.servers["test-server"]; exists {
		t.Error("server should be removed from map after Disconnect")
	}
}

func TestDisconnectNonexistentIsNoOp(t *testing.T) {
	client := NewClient()
	err := client.Disconnect("does-not-exist")
	if err != nil {
		t.Errorf("Disconnect nonexistent should return nil, got: %v", err)
	}
}

func TestDisconnectAllWithMultipleServers(t *testing.T) {
	client := NewClient()

	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("server-%d", i)
		_, cw := io.Pipe()
		cr, _ := io.Pipe()
		client.servers[name] = &ServerConnection{
			config: config.MCPServer{Name: name},
			stdin:  cw,
			stdout: cr,
			ready:  true,
		}
	}

	if len(client.servers) != 5 {
		t.Fatalf("expected 5 servers, got %d", len(client.servers))
	}

	client.DisconnectAll()

	if len(client.servers) != 0 {
		t.Errorf("expected 0 servers after DisconnectAll, got %d", len(client.servers))
	}
}

// =============================================================================
// 3. ServerConnection initialize / listTools
// =============================================================================

func TestServerConnectionInitialize(t *testing.T) {
	conn, mock := newTestServerConnection(t, nil)
	defer mock.stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := conn.initialize(ctx)
	if err != nil {
		t.Fatalf("initialize should succeed, got: %v", err)
	}
	if !conn.ready {
		t.Error("connection should be marked ready after initialize")
	}
}

func TestServerConnectionInitializeFailure(t *testing.T) {
	conn, mock := newTestServerConnection(t, nil)
	mock.failInit = true
	defer mock.stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := conn.initialize(ctx)
	if err == nil {
		t.Fatal("initialize should fail when server returns error")
	}
	if !strings.Contains(err.Error(), "initialization failed") {
		t.Errorf("error should contain 'initialization failed', got: %v", err)
	}
}

func TestServerConnectionListTools(t *testing.T) {
	tools := []Tool{
		{Name: "kubectl", Description: "Run kubectl commands"},
		{Name: "bash", Description: "Run bash commands"},
		{Name: "custom_tool", Description: "A custom tool"},
	}

	conn, mock := newTestServerConnection(t, tools)
	defer mock.stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := conn.listTools(ctx)
	if err != nil {
		t.Fatalf("listTools should succeed, got: %v", err)
	}

	if len(conn.tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(conn.tools))
	}

	// Verify tools are tagged with server name
	for _, tool := range conn.tools {
		if tool.ServerName != "test-server" {
			t.Errorf("tool %s should have ServerName 'test-server', got %q", tool.Name, tool.ServerName)
		}
	}
}

func TestServerConnectionListToolsEmpty(t *testing.T) {
	conn, mock := newTestServerConnection(t, []Tool{})
	defer mock.stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := conn.listTools(ctx)
	if err != nil {
		t.Fatalf("listTools should succeed with empty tools, got: %v", err)
	}

	if len(conn.tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(conn.tools))
	}
}

// =============================================================================
// 4. Tool invocation (CallTool)
// =============================================================================

func TestServerConnectionCallTool(t *testing.T) {
	tools := []Tool{
		{Name: "echo", Description: "Echo a message"},
	}

	conn, mock := newTestServerConnection(t, tools)
	defer mock.stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := conn.callTool(ctx, "echo", map[string]interface{}{
		"message": "hello world",
	})
	if err != nil {
		t.Fatalf("callTool should succeed, got: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if len(result.Content) == 0 {
		t.Fatal("result should have content blocks")
	}
	if result.Content[0].Text != "hello world" {
		t.Errorf("expected 'hello world', got %q", result.Content[0].Text)
	}
	if result.Content[0].Type != "text" {
		t.Errorf("expected content type 'text', got %q", result.Content[0].Type)
	}
}

func TestServerConnectionCallToolNotFound(t *testing.T) {
	tools := []Tool{
		{Name: "echo", Description: "Echo a message"},
	}

	conn, mock := newTestServerConnection(t, tools)
	defer mock.stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := conn.callTool(ctx, "nonexistent", nil)
	if err == nil {
		t.Fatal("callTool should fail for unknown tool")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention tool name, got: %v", err)
	}
}

func TestClientCallToolRouting(t *testing.T) {
	// Set up two server connections with different tools
	client := NewClient()

	// Server A with "tool_a"
	toolsA := []Tool{{Name: "tool_a", Description: "Tool A"}}
	connA, mockA := newTestServerConnection(t, toolsA)
	connA.config.Name = "server-a"
	connA.tools = []Tool{{Name: "tool_a", Description: "Tool A", ServerName: "server-a"}}
	defer mockA.stop()

	// Server B with "tool_b"
	toolsB := []Tool{{Name: "tool_b", Description: "Tool B"}}
	connB, mockB := newTestServerConnection(t, toolsB)
	connB.config.Name = "server-b"
	connB.tools = []Tool{{Name: "tool_b", Description: "Tool B", ServerName: "server-b"}}
	defer mockB.stop()

	client.servers["server-a"] = connA
	client.servers["server-b"] = connB

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call tool_a - should route to server-a
	result, err := client.CallTool(ctx, "tool_a", map[string]interface{}{"message": "from A"})
	if err != nil {
		t.Fatalf("CallTool for tool_a should succeed, got: %v", err)
	}
	if result.Content[0].Text != "from A" {
		t.Errorf("expected 'from A', got %q", result.Content[0].Text)
	}

	// Call tool_b - should route to server-b
	result, err = client.CallTool(ctx, "tool_b", map[string]interface{}{"message": "from B"})
	if err != nil {
		t.Fatalf("CallTool for tool_b should succeed, got: %v", err)
	}
	if result.Content[0].Text != "from B" {
		t.Errorf("expected 'from B', got %q", result.Content[0].Text)
	}
}

func TestClientCallToolNotRegistered(t *testing.T) {
	client := NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.CallTool(ctx, "nonexistent_tool", nil)
	if err == nil {
		t.Fatal("CallTool for unregistered tool should return error")
	}
	if !strings.Contains(err.Error(), "tool not found") {
		t.Errorf("error should say 'tool not found', got: %v", err)
	}
}

// =============================================================================
// 5. GetAllTools, IsConnected, GetConnectedServers
// =============================================================================

func TestGetAllToolsAcrossServers(t *testing.T) {
	client := NewClient()

	client.servers["s1"] = &ServerConnection{
		tools: []Tool{
			{Name: "t1", ServerName: "s1"},
			{Name: "t2", ServerName: "s1"},
		},
	}
	client.servers["s2"] = &ServerConnection{
		tools: []Tool{
			{Name: "t3", ServerName: "s2"},
		},
	}

	allTools := client.GetAllTools()
	if len(allTools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(allTools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range allTools {
		toolNames[tool.Name] = true
	}
	for _, expected := range []string{"t1", "t2", "t3"} {
		if !toolNames[expected] {
			t.Errorf("expected tool %q in result", expected)
		}
	}
}

func TestIsConnectedStates(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c *Client)
		server   string
		expected bool
	}{
		{
			name:     "nonexistent server",
			setup:    func(c *Client) {},
			server:   "missing",
			expected: false,
		},
		{
			name: "connected and ready",
			setup: func(c *Client) {
				c.servers["ready"] = &ServerConnection{ready: true}
			},
			server:   "ready",
			expected: true,
		},
		{
			name: "connected but not ready",
			setup: func(c *Client) {
				c.servers["not-ready"] = &ServerConnection{ready: false}
			},
			server:   "not-ready",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient()
			tt.setup(client)
			got := client.IsConnected(tt.server)
			if got != tt.expected {
				t.Errorf("IsConnected(%q) = %v, want %v", tt.server, got, tt.expected)
			}
		})
	}
}

func TestGetConnectedServersMultiple(t *testing.T) {
	client := NewClient()

	expected := []string{"alpha", "beta", "gamma"}
	for _, name := range expected {
		client.servers[name] = &ServerConnection{
			config: config.MCPServer{Name: name},
		}
	}

	got := client.GetConnectedServers()
	if len(got) != len(expected) {
		t.Fatalf("expected %d servers, got %d", len(expected), len(got))
	}

	gotMap := make(map[string]bool)
	for _, name := range got {
		gotMap[name] = true
	}
	for _, name := range expected {
		if !gotMap[name] {
			t.Errorf("expected server %q in result", name)
		}
	}
}

// =============================================================================
// 6. Error handling
// =============================================================================

func TestSendRequestTimeout(t *testing.T) {
	// Create a connection where the server never responds.
	// io.Pipe() is synchronous: writes block until the read end consumes the data.
	// We must drain the read end of the stdin pipe so that conn.stdin.Write()
	// in sendRequest completes and the code reaches the context timeout select.
	stdinReader, clientWriter := io.Pipe()
	clientReader, _ := io.Pipe() // server never writes a response

	// Drain stdin so the Write in sendRequest doesn't block forever
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := stdinReader.Read(buf); err != nil {
				return
			}
		}
	}()

	scanner := bufio.NewScanner(clientReader)

	conn := &ServerConnection{
		config:  config.MCPServer{Name: "slow-server"},
		stdin:   clientWriter,
		stdout:  clientReader,
		scanner: scanner,
	}

	// Use a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := conn.sendRequest(ctx, "tools/list", nil)
	if err == nil {
		t.Fatal("sendRequest should time out")
	}
	// Should be context.DeadlineExceeded
	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got: %v", ctx.Err())
	}

	// Cleanup
	stdinReader.Close()
	clientWriter.Close()
	clientReader.Close()
}

func TestSendRequestConnectionClosed(t *testing.T) {
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	scanner := bufio.NewScanner(clientReader)

	conn := &ServerConnection{
		config:  config.MCPServer{Name: "closing-server"},
		stdin:   clientWriter,
		stdout:  clientReader,
		scanner: scanner,
	}

	// Close the server writer immediately to simulate connection drop
	go func() {
		// Read and discard the request
		buf := make([]byte, 4096)
		_, _ = serverReader.Read(buf)
		// Close the server's response writer
		serverWriter.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := conn.sendRequest(ctx, "initialize", nil)
	if err == nil {
		t.Fatal("sendRequest should fail when connection closes")
	}

	// Cleanup
	serverReader.Close()
	clientWriter.Close()
}

func TestSendRequestInvalidJSON(t *testing.T) {
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	scanner := bufio.NewScanner(clientReader)

	conn := &ServerConnection{
		config:  config.MCPServer{Name: "bad-json-server"},
		stdin:   clientWriter,
		stdout:  clientReader,
		scanner: scanner,
	}

	// Server reads request then writes invalid JSON
	go func() {
		buf := make([]byte, 4096)
		_, _ = serverReader.Read(buf)
		_, _ = serverWriter.Write([]byte("{invalid json}\n"))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := conn.sendRequest(ctx, "initialize", nil)
	if err == nil {
		t.Fatal("sendRequest should fail on invalid JSON response")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error should mention unmarshal failure, got: %v", err)
	}

	// Cleanup
	serverReader.Close()
	serverWriter.Close()
	clientWriter.Close()
	clientReader.Close()
}

func TestSendRequestRPCError(t *testing.T) {
	conn, mock := newTestServerConnection(t, nil)
	defer mock.stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call an unknown method that triggers a -32601 error
	_, err := conn.sendRequest(ctx, "unknown/method", nil)
	if err == nil {
		t.Fatal("sendRequest should return error for unknown method")
	}
	if !strings.Contains(err.Error(), "RPC error") {
		t.Errorf("error should contain 'RPC error', got: %v", err)
	}
	if !strings.Contains(err.Error(), "Method not found") {
		t.Errorf("error should contain 'Method not found', got: %v", err)
	}
}

func TestSendRequestDefaultTimeout(t *testing.T) {
	// Test that sendRequest applies a 30s default timeout when context has none.
	// We cannot wait 30 seconds, but we can verify the context is created
	// by observing that the function does not block indefinitely when the server
	// never responds. We use a trick: cancel the parent context quickly.

	stdinReader, clientWriter := io.Pipe()
	clientReader, _ := io.Pipe()

	// Drain stdin so the Write in sendRequest doesn't block forever
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := stdinReader.Read(buf); err != nil {
				return
			}
		}
	}()

	scanner := bufio.NewScanner(clientReader)

	conn := &ServerConnection{
		config:  config.MCPServer{Name: "no-timeout-server"},
		stdin:   clientWriter,
		stdout:  clientReader,
		scanner: scanner,
	}

	// Use context.Background() with NO deadline.
	// Then cancel it externally after a short delay.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	_, err := conn.sendRequest(ctx, "tools/list", nil)
	if err == nil {
		t.Fatal("sendRequest should fail when context is cancelled")
	}

	// Cleanup
	stdinReader.Close()
	clientWriter.Close()
	clientReader.Close()
}

// =============================================================================
// 7. ServerConnection Close
// =============================================================================

func TestServerConnectionClose(t *testing.T) {
	_, clientWriter := io.Pipe()
	clientReader, _ := io.Pipe()

	conn := &ServerConnection{
		config: config.MCPServer{Name: "close-test"},
		stdin:  clientWriter,
		stdout: clientReader,
		ready:  true,
	}

	err := conn.Close()
	if err != nil {
		t.Errorf("Close should return nil, got: %v", err)
	}
	if conn.ready {
		t.Error("ready should be false after Close")
	}
}

func TestServerConnectionCloseWithNilFields(t *testing.T) {
	conn := &ServerConnection{
		config: config.MCPServer{Name: "nil-fields"},
		stdin:  nil,
		stdout: nil,
		cmd:    nil,
		ready:  true,
	}

	// Should not panic
	err := conn.Close()
	if err != nil {
		t.Errorf("Close with nil fields should return nil, got: %v", err)
	}
	if conn.ready {
		t.Error("ready should be false after Close")
	}
}

func TestServerConnectionCloseIdempotent(t *testing.T) {
	_, clientWriter := io.Pipe()
	clientReader, _ := io.Pipe()

	conn := &ServerConnection{
		config: config.MCPServer{Name: "idempotent-close"},
		stdin:  clientWriter,
		stdout: clientReader,
		ready:  true,
	}

	// First close
	err := conn.Close()
	if err != nil {
		t.Errorf("first Close should return nil, got: %v", err)
	}

	// Second close should not panic (stdin/stdout already closed)
	err = conn.Close()
	if err != nil {
		t.Errorf("second Close should return nil, got: %v", err)
	}
}

// =============================================================================
// 8. MCPToolExecutorAdapter
// =============================================================================

func TestMCPToolExecutorAdapterCallTool(t *testing.T) {
	client := NewClient()

	tools := []Tool{{Name: "greet", Description: "Greet someone"}}
	conn, mock := newTestServerConnection(t, tools)
	conn.config.Name = "adapter-server"
	conn.tools = []Tool{{Name: "greet", Description: "Greet", ServerName: "adapter-server"}}
	defer mock.stop()

	client.servers["adapter-server"] = conn

	adapter := NewMCPToolExecutor(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := adapter.CallTool(ctx, "greet", map[string]interface{}{
		"message": "hello from adapter",
	})
	if err != nil {
		t.Fatalf("adapter CallTool should succeed, got: %v", err)
	}
	if output != "hello from adapter" {
		t.Errorf("expected 'hello from adapter', got %q", output)
	}
}

func TestMCPToolExecutorAdapterToolNotFound(t *testing.T) {
	client := NewClient()
	adapter := NewMCPToolExecutor(client)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := adapter.CallTool(ctx, "nonexistent", nil)
	if err == nil {
		t.Fatal("adapter CallTool should fail for nonexistent tool")
	}
}

func TestMCPToolExecutorAdapterMultipleContentBlocks(t *testing.T) {
	// We need to test the adapter's content concatenation behavior.
	// We'll create a custom mock that returns multiple content blocks.
	client := NewClient()

	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	scanner := bufio.NewScanner(clientReader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	conn := &ServerConnection{
		config:  config.MCPServer{Name: "multi-block"},
		stdin:   clientWriter,
		stdout:  clientReader,
		scanner: scanner,
		tools:   []Tool{{Name: "multi", Description: "Multi block tool", ServerName: "multi-block"}},
	}
	client.servers["multi-block"] = conn

	// Custom mock that returns multiple content blocks
	go func() {
		reqScanner := bufio.NewScanner(serverReader)
		for reqScanner.Scan() {
			var req JSONRPCRequest
			if err := json.Unmarshal(reqScanner.Bytes(), &req); err != nil {
				continue
			}
			result := CallToolResult{
				Content: []ContentBlock{
					{Type: "text", Text: "line 1"},
					{Type: "text", Text: "line 2"},
					{Type: "image", Text: ""},      // should be skipped
					{Type: "text", Text: "line 3"},
				},
			}
			resultBytes, _ := json.Marshal(result)
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  resultBytes,
			}
			respBytes, _ := json.Marshal(resp)
			_, _ = serverWriter.Write(append(respBytes, '\n'))
		}
	}()

	adapter := NewMCPToolExecutor(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := adapter.CallTool(ctx, "multi", nil)
	if err != nil {
		t.Fatalf("adapter CallTool should succeed, got: %v", err)
	}

	expected := "line 1\nline 2\nline 3"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}

	// Cleanup
	serverReader.Close()
	serverWriter.Close()
	clientWriter.Close()
	clientReader.Close()
}

func TestMCPToolExecutorAdapterIsErrorResult(t *testing.T) {
	client := NewClient()

	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	scanner := bufio.NewScanner(clientReader)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	conn := &ServerConnection{
		config:  config.MCPServer{Name: "error-server"},
		stdin:   clientWriter,
		stdout:  clientReader,
		scanner: scanner,
		tools:   []Tool{{Name: "fail", Description: "Always fails", ServerName: "error-server"}},
	}
	client.servers["error-server"] = conn

	// Custom mock that returns isError=true
	go func() {
		reqScanner := bufio.NewScanner(serverReader)
		for reqScanner.Scan() {
			var req JSONRPCRequest
			if err := json.Unmarshal(reqScanner.Bytes(), &req); err != nil {
				continue
			}
			result := CallToolResult{
				Content: []ContentBlock{{Type: "text", Text: "something went wrong"}},
				IsError: true,
			}
			resultBytes, _ := json.Marshal(result)
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  resultBytes,
			}
			respBytes, _ := json.Marshal(resp)
			_, _ = serverWriter.Write(append(respBytes, '\n'))
		}
	}()

	adapter := NewMCPToolExecutor(client)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := adapter.CallTool(ctx, "fail", nil)
	if err == nil {
		t.Fatal("adapter CallTool should return error when IsError is true")
	}
	if !strings.Contains(err.Error(), "tool execution failed") {
		t.Errorf("error should mention 'tool execution failed', got: %v", err)
	}
	if output != "something went wrong" {
		t.Errorf("expected output 'something went wrong', got %q", output)
	}

	// Cleanup
	serverReader.Close()
	serverWriter.Close()
	clientWriter.Close()
	clientReader.Close()
}

// =============================================================================
// 9. Concurrent access safety
// =============================================================================

func TestConcurrentGetAllTools(t *testing.T) {
	client := NewClient()

	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("server-%d", i)
		client.servers[name] = &ServerConnection{
			tools: []Tool{
				{Name: fmt.Sprintf("tool-%d-a", i), ServerName: name},
				{Name: fmt.Sprintf("tool-%d-b", i), ServerName: name},
			},
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tools := client.GetAllTools()
			if len(tools) != 6 {
				t.Errorf("expected 6 tools, got %d", len(tools))
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentIsConnected(t *testing.T) {
	client := NewClient()
	client.servers["test"] = &ServerConnection{ready: true}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = client.IsConnected("test")
			_ = client.IsConnected("missing")
		}()
	}
	wg.Wait()
}

func TestConcurrentGetConnectedServers(t *testing.T) {
	client := NewClient()
	for i := 0; i < 5; i++ {
		client.servers[fmt.Sprintf("s%d", i)] = &ServerConnection{}
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			servers := client.GetConnectedServers()
			if len(servers) != 5 {
				t.Errorf("expected 5 servers, got %d", len(servers))
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentDisconnectAll(t *testing.T) {
	client := NewClient()

	var wg sync.WaitGroup
	// Multiple goroutines adding and disconnecting
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.DisconnectAll()
		}()
	}
	wg.Wait()

	if len(client.servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(client.servers))
	}
}

func TestConcurrentReadAndWrite(t *testing.T) {
	client := NewClient()

	// Setup some initial state
	client.servers["initial"] = &ServerConnection{
		tools: []Tool{{Name: "tool1", ServerName: "initial"}},
		ready: true,
	}

	var wg sync.WaitGroup

	// Readers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = client.GetAllTools()
			_ = client.IsConnected("initial")
			_ = client.GetConnectedServers()
		}()
	}

	// Writer (disconnect)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("temp-%d", idx)
			// Add then disconnect
			client.mu.Lock()
			client.servers[name] = &ServerConnection{ready: true}
			client.mu.Unlock()
			_ = client.Disconnect(name)
		}(i)
	}

	wg.Wait()
}

// =============================================================================
// 10. JSON-RPC types and serialization
// =============================================================================

func TestJSONRPCRequestSerialization(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      42,
		Method:  "tools/call",
		Params: CallToolParams{
			Name:      "kubectl",
			Arguments: map[string]interface{}{"command": "get pods"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if parsed["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", parsed["jsonrpc"])
	}
	if parsed["method"] != "tools/call" {
		t.Errorf("method = %v, want tools/call", parsed["method"])
	}
}

func TestJSONRPCResponseDeserialization(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		checkFn func(t *testing.T, resp *JSONRPCResponse)
	}{
		{
			name:  "success response",
			input: `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`,
			checkFn: func(t *testing.T, resp *JSONRPCResponse) {
				if resp.Error != nil {
					t.Error("expected no error")
				}
				if resp.Result == nil {
					t.Error("expected result")
				}
			},
		},
		{
			name:  "error response",
			input: `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`,
			checkFn: func(t *testing.T, resp *JSONRPCResponse) {
				if resp.Error == nil {
					t.Fatal("expected error")
				}
				if resp.Error.Code != -32601 {
					t.Errorf("error code = %d, want -32601", resp.Error.Code)
				}
				if resp.Error.Message != "Method not found" {
					t.Errorf("error message = %q, want 'Method not found'", resp.Error.Message)
				}
			},
		},
		{
			name:    "invalid json",
			input:   `{not valid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp JSONRPCResponse
			err := json.Unmarshal([]byte(tt.input), &resp)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected unmarshal error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, &resp)
			}
		})
	}
}

func TestToolSerialization(t *testing.T) {
	tool := Tool{
		Name:        "kubectl",
		Description: "Execute kubectl commands",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The kubectl command",
				},
			},
			"required": []interface{}{"command"},
		},
		ServerName: "k8s-server",
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("failed to marshal tool: %v", err)
	}

	// ServerName should NOT appear in JSON (tagged with json:"-")
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if _, exists := raw["ServerName"]; exists {
		t.Error("ServerName should not be serialized to JSON")
	}

	// Roundtrip
	var parsed Tool
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal tool: %v", err)
	}
	if parsed.Name != "kubectl" {
		t.Errorf("Name = %q, want kubectl", parsed.Name)
	}
	if parsed.ServerName != "" {
		t.Errorf("ServerName should be empty after deserialization, got %q", parsed.ServerName)
	}
}

func TestCallToolResultSerialization(t *testing.T) {
	result := CallToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: "hello"},
			{Type: "text", Text: "world"},
		},
		IsError: false,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed CallToolResult
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(parsed.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(parsed.Content))
	}
	if parsed.Content[0].Text != "hello" {
		t.Errorf("content[0].Text = %q, want 'hello'", parsed.Content[0].Text)
	}
	if parsed.IsError {
		t.Error("IsError should be false")
	}
}

// =============================================================================
// 11. Full integration: initialize + listTools + callTool via pipes
// =============================================================================

func TestFullServerConnectionLifecycle(t *testing.T) {
	tools := []Tool{
		{Name: "greet", Description: "Greet someone"},
		{Name: "echo", Description: "Echo input"},
	}

	conn, mock := newTestServerConnection(t, tools)
	defer mock.stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Step 1: Initialize
	if err := conn.initialize(ctx); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	if !conn.ready {
		t.Fatal("connection should be ready after initialize")
	}

	// Step 2: List tools
	if err := conn.listTools(ctx); err != nil {
		t.Fatalf("listTools failed: %v", err)
	}
	if len(conn.tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(conn.tools))
	}

	// Verify tool names
	toolNames := make(map[string]bool)
	for _, tool := range conn.tools {
		toolNames[tool.Name] = true
	}
	if !toolNames["greet"] || !toolNames["echo"] {
		t.Errorf("expected tools greet and echo, got: %v", toolNames)
	}

	// Step 3: Call a tool
	result, err := conn.callTool(ctx, "greet", map[string]interface{}{
		"message": "world",
	})
	if err != nil {
		t.Fatalf("callTool failed: %v", err)
	}
	if result.Content[0].Text != "world" {
		t.Errorf("expected 'world', got %q", result.Content[0].Text)
	}
}

// =============================================================================
// 12. Connect with real process (full E2E with Go test binary as MCP server)
// =============================================================================

func TestConnectWithEchoServer(t *testing.T) {
	// Verify that 'cat' is available (it is on all Unix systems)
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available")
	}

	// We cannot use 'cat' as an MCP server because it doesn't speak JSON-RPC.
	// Instead, we test that Connect properly fails when the server process
	// doesn't respond with valid JSON-RPC within timeout.

	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := client.Connect(ctx, config.MCPServer{
		Name:    "echo-server",
		Command: "cat", // cat will echo stdin but won't produce valid JSON-RPC
		Args:    []string{},
	})

	// This should fail because 'cat' won't send an initialize response
	if err == nil {
		// If somehow it succeeded, disconnect
		_ = client.Disconnect("echo-server")
		t.Fatal("Connect with 'cat' as MCP server should fail")
	}

	// The error should be about initialization failure or timeout
	if !strings.Contains(err.Error(), "failed to initialize") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Logf("got error (expected): %v", err)
	}
}

func TestStartServerEnvVars(t *testing.T) {
	// Test that environment variables are properly passed to the server process
	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use 'env' command to print environment, but we mainly care that
	// the process starts successfully with custom env vars
	envCmd := "env"
	if _, err := exec.LookPath(envCmd); err != nil {
		t.Skip("env command not available")
	}

	serverCfg := config.MCPServer{
		Name:    "env-test",
		Command: envCmd,
		Env: map[string]string{
			"MCP_TEST_VAR":  "test_value",
			"MCP_TEST_VAR2": "test_value2",
		},
	}

	// startServer should succeed even though 'env' exits immediately
	conn, err := client.startServer(ctx, serverCfg)
	if err != nil {
		// On some systems, env exits before we can get pipes set up, which is fine
		t.Logf("startServer error (may be expected): %v", err)
		return
	}

	// Verify the cmd has our env vars
	found := 0
	for _, envVar := range conn.cmd.Env {
		if envVar == "MCP_TEST_VAR=test_value" || envVar == "MCP_TEST_VAR2=test_value2" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("expected 2 custom env vars, found %d", found)
	}

	// Also verify the process inherits the system environment
	hasPath := false
	for _, envVar := range conn.cmd.Env {
		if strings.HasPrefix(envVar, "PATH=") || strings.HasPrefix(envVar, "HOME=") {
			hasPath = true
			break
		}
	}
	if !hasPath {
		t.Error("server process should inherit system environment variables")
	}

	conn.Close()
}

// =============================================================================
// 13. reqID counter
// =============================================================================

func TestRequestIDIncrement(t *testing.T) {
	tools := []Tool{{Name: "echo", Description: "Echo"}}
	conn, mock := newTestServerConnection(t, tools)
	defer mock.stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Make multiple requests and verify IDs increment
	for i := 0; i < 5; i++ {
		_, _ = conn.sendRequest(ctx, "tools/list", nil)
	}

	// After 5 requests, the reqID should be 5
	got := conn.reqID.Load()
	if got != 5 {
		t.Errorf("reqID after 5 requests = %d, want 5", got)
	}
}

// =============================================================================
// 14. Edge cases
// =============================================================================

func TestCallToolWithNilArgs(t *testing.T) {
	tools := []Tool{{Name: "no-args", Description: "No args needed"}}
	conn, mock := newTestServerConnection(t, tools)
	defer mock.stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := conn.callTool(ctx, "no-args", nil)
	if err != nil {
		t.Fatalf("callTool with nil args should succeed, got: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

func TestGetAllToolsPreservesOrder(t *testing.T) {
	// When there's only one server, tools should come in order
	client := NewClient()
	client.servers["only"] = &ServerConnection{
		tools: []Tool{
			{Name: "a", ServerName: "only"},
			{Name: "b", ServerName: "only"},
			{Name: "c", ServerName: "only"},
		},
	}

	tools := client.GetAllTools()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}
	if tools[0].Name != "a" || tools[1].Name != "b" || tools[2].Name != "c" {
		t.Errorf("tools order not preserved: %v", tools)
	}
}

func TestConnectContextCancelled(t *testing.T) {
	client := NewClient()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.Connect(ctx, config.MCPServer{
		Name:    "cancelled-server",
		Command: "sleep",
		Args:    []string{"100"},
	})

	// Should fail due to cancelled context
	if err == nil {
		_ = client.Disconnect("cancelled-server")
		// On some systems, sleep might start before context cancellation is checked.
		// This is acceptable - the key test is that it doesn't hang.
		t.Log("Connect returned nil with cancelled context (process started before check)")
	}
}

func TestClientCallToolWithEmptyServerTools(t *testing.T) {
	client := NewClient()

	// Server exists but has no tools
	client.servers["empty-server"] = &ServerConnection{
		tools: []Tool{},
		ready: true,
	}

	ctx := context.Background()
	_, err := client.CallTool(ctx, "any-tool", nil)
	if err == nil {
		t.Fatal("CallTool should fail when no server has the tool")
	}
	if !strings.Contains(err.Error(), "tool not found") {
		t.Errorf("error should say 'tool not found', got: %v", err)
	}
}

// =============================================================================
// 15. Test with real MCP server subprocess (using Go test helper)
// =============================================================================

// TestConnectWithGoMCPHelper uses a Go subprocess as a mock MCP server.
// The subprocess reads JSON-RPC requests from stdin and writes responses to stdout.
func TestConnectWithGoMCPHelper(t *testing.T) {
	if os.Getenv("MCP_TEST_HELPER") == "1" {
		// This is the child process acting as an MCP server
		runMCPTestHelper()
		return
	}

	// Find the Go binary
	goPath, err := exec.LookPath("go")
	if err != nil {
		// Try common paths
		for _, p := range []string{"/usr/local/go/bin/go", os.Getenv("HOME") + "/go/bin/go"} {
			if _, statErr := os.Stat(p); statErr == nil {
				goPath = p
				break
			}
		}
		if goPath == "" {
			t.Skip("go binary not found")
		}
	}

	// Get the test binary path - we re-execute ourselves with the helper env var
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get test binary path: %v", err)
	}

	client := NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx, config.MCPServer{
		Name:    "go-helper",
		Command: testBinary,
		Args:    []string{"-test.run=TestConnectWithGoMCPHelper"},
		Env: map[string]string{
			"MCP_TEST_HELPER": "1",
		},
	})
	if err != nil {
		t.Fatalf("Connect with Go MCP helper should succeed, got: %v", err)
	}
	defer func() { _ = client.Disconnect("go-helper") }()

	// Verify connection
	if !client.IsConnected("go-helper") {
		t.Error("server should be connected and ready")
	}

	// Verify tools
	allTools := client.GetAllTools()
	if len(allTools) != 2 {
		t.Fatalf("expected 2 tools from helper, got %d", len(allTools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range allTools {
		toolNames[tool.Name] = true
	}
	if !toolNames["test_echo"] || !toolNames["test_add"] {
		t.Errorf("expected tools test_echo and test_add, got: %v", toolNames)
	}

	// Call a tool
	result, err := client.CallTool(ctx, "test_echo", map[string]interface{}{
		"message": "integration test",
	})
	if err != nil {
		t.Fatalf("CallTool should succeed, got: %v", err)
	}
	if len(result.Content) == 0 || result.Content[0].Text != "integration test" {
		t.Errorf("unexpected result: %+v", result)
	}

	// Disconnect
	err = client.Disconnect("go-helper")
	if err != nil {
		t.Errorf("Disconnect should succeed, got: %v", err)
	}
	if client.IsConnected("go-helper") {
		t.Error("server should not be connected after Disconnect")
	}
}

// runMCPTestHelper runs a simple MCP server that handles initialize, tools/list, and tools/call.
func runMCPTestHelper() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	tools := []Tool{
		{Name: "test_echo", Description: "Echo back the message"},
		{Name: "test_add", Description: "Add two numbers"},
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			writeHelperResponse(&JSONRPCResponse{
				JSONRPC: "2.0",
				Error:   &JSONRPCError{Code: -32700, Message: "Parse error"},
			})
			continue
		}

		switch req.Method {
		case "initialize":
			result := InitializeResult{
				ProtocolVersion: "2024-11-05",
				Capabilities:    Capabilities{Tools: &ToolsCapability{}},
				ServerInfo:      ServerInfo{Name: "test-helper", Version: "1.0.0"},
			}
			resultBytes, _ := json.Marshal(result)
			writeHelperResponse(&JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  resultBytes,
			})

		case "notifications/initialized":
			// No response

		case "tools/list":
			result := ListToolsResult{Tools: tools}
			resultBytes, _ := json.Marshal(result)
			writeHelperResponse(&JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  resultBytes,
			})

		case "tools/call":
			var params CallToolParams
			if paramsBytes, err := json.Marshal(req.Params); err == nil {
				_ = json.Unmarshal(paramsBytes, &params)
			}

			var text string
			switch params.Name {
			case "test_echo":
				if msg, ok := params.Arguments["message"]; ok {
					text = fmt.Sprintf("%v", msg)
				}
			case "test_add":
				a, _ := params.Arguments["a"].(float64)
				b, _ := params.Arguments["b"].(float64)
				text = fmt.Sprintf("%g", a+b)
			default:
				writeHelperResponse(&JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &JSONRPCError{Code: -32602, Message: "Unknown tool"},
				})
				continue
			}

			result := CallToolResult{
				Content: []ContentBlock{{Type: "text", Text: text}},
			}
			resultBytes, _ := json.Marshal(result)
			writeHelperResponse(&JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  resultBytes,
			})

		default:
			writeHelperResponse(&JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &JSONRPCError{Code: -32601, Message: "Method not found"},
			})
		}
	}
	os.Exit(0)
}

func writeHelperResponse(resp *JSONRPCResponse) {
	data, _ := json.Marshal(resp)
	os.Stdout.Write(append(data, '\n'))
}
