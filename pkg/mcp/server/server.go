package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

// Server implements an MCP server that exposes k13d tools via stdio
type Server struct {
	stdin   io.Reader
	stdout  io.Writer
	scanner *bufio.Scanner
	mu      sync.Mutex
	tools   map[string]*Tool
	running atomic.Bool

	// Server info
	name    string
	version string
}

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Handler     ToolHandler            `json:"-"`
}

// ToolHandler is a function that executes a tool
type ToolHandler func(ctx context.Context, args map[string]interface{}) (string, error)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Protocol types
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ListToolsResult struct {
	Tools []ToolDefinition `json:"tools"`
}

type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// New creates a new MCP server
func New(name, version string) *Server {
	return &Server{
		stdin:   os.Stdin,
		stdout:  os.Stdout,
		tools:   make(map[string]*Tool),
		name:    name,
		version: version,
	}
}

// NewWithIO creates a new MCP server with custom IO (for testing)
func NewWithIO(name, version string, stdin io.Reader, stdout io.Writer) *Server {
	return &Server{
		stdin:   stdin,
		stdout:  stdout,
		tools:   make(map[string]*Tool),
		name:    name,
		version: version,
	}
}

// RegisterTool adds a tool to the server
func (s *Server) RegisterTool(tool *Tool) {
	s.tools[tool.Name] = tool
}

// Run starts the MCP server and processes requests
func (s *Server) Run(ctx context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		return fmt.Errorf("server already running")
	}
	defer s.running.Store(false)

	s.scanner = bufio.NewScanner(s.stdin)
	// Increase buffer size for large messages
	s.scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !s.scanner.Scan() {
			if err := s.scanner.Err(); err != nil {
				return fmt.Errorf("scanner error: %w", err)
			}
			return nil // EOF
		}

		line := s.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		s.handleRequest(ctx, &req)
	}
}

// handleRequest processes a single JSON-RPC request
func (s *Server) handleRequest(ctx context.Context, req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "notifications/initialized":
		// No response needed for notifications
	case "tools/list":
		s.handleListTools(req)
	case "tools/call":
		s.handleCallTool(ctx, req)
	case "ping":
		s.sendResult(req.ID, map[string]interface{}{})
	default:
		s.sendError(req.ID, -32601, "Method not found", req.Method)
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req *JSONRPCRequest) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: Capabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
		},
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
	}
	s.sendResult(req.ID, result)
}

// handleListTools handles the tools/list request
func (s *Server) handleListTools(req *JSONRPCRequest) {
	tools := make([]ToolDefinition, 0, len(s.tools))
	for _, t := range s.tools {
		tools = append(tools, ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	s.sendResult(req.ID, ListToolsResult{Tools: tools})
}

// handleCallTool handles the tools/call request
func (s *Server) handleCallTool(ctx context.Context, req *JSONRPCRequest) {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	tool, ok := s.tools[params.Name]
	if !ok {
		s.sendError(req.ID, -32602, "Unknown tool", params.Name)
		return
	}

	output, err := tool.Handler(ctx, params.Arguments)

	result := CallToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: output},
		},
	}

	if err != nil {
		result.IsError = true
		if output == "" {
			result.Content[0].Text = err.Error()
		}
	}

	s.sendResult(req.ID, result)
}

// sendResult sends a successful response
func (s *Server) sendResult(id json.RawMessage, result interface{}) {
	s.send(&JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

// sendError sends an error response
func (s *Server) sendError(id json.RawMessage, code int, message string, data interface{}) {
	s.send(&JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	})
}

// send writes a response to stdout
func (s *Server) send(resp *JSONRPCResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	s.stdout.Write(append(data, '\n'))
}
