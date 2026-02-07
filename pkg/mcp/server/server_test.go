package server

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	s := New("test-server", "1.0.0")
	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.name != "test-server" {
		t.Errorf("name = %q, want %q", s.name, "test-server")
	}
	if s.version != "1.0.0" {
		t.Errorf("version = %q, want %q", s.version, "1.0.0")
	}
	if s.tools == nil {
		t.Error("tools map is nil")
	}
}

func TestNewWithIO(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}

	s := NewWithIO("test", "1.0", stdin, stdout)
	if s == nil {
		t.Fatal("NewWithIO() returned nil")
	}
	if s.stdin != stdin {
		t.Error("stdin not set correctly")
	}
	if s.stdout != stdout {
		t.Error("stdout not set correctly")
	}
}

func TestRegisterTool(t *testing.T) {
	s := New("test", "1.0")

	tool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"arg1": map[string]interface{}{"type": "string"},
			},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			return "success", nil
		},
	}

	s.RegisterTool(tool)

	if _, ok := s.tools["test_tool"]; !ok {
		t.Error("tool not registered")
	}
}

func TestHandleInitialize(t *testing.T) {
	stdin := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test"}}}
`)
	stdout := &bytes.Buffer{}

	s := NewWithIO("k13d", "0.6.1", stdin, stdout)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_ = s.Run(ctx)

	output := stdout.String()
	if !strings.Contains(output, "k13d") {
		t.Errorf("response should contain server name, got: %s", output)
	}
	if !strings.Contains(output, "2024-11-05") {
		t.Errorf("response should contain protocol version, got: %s", output)
	}
}

func TestHandleListTools(t *testing.T) {
	stdin := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}
`)
	stdout := &bytes.Buffer{}

	s := NewWithIO("test", "1.0", stdin, stdout)
	s.RegisterTool(&Tool{
		Name:        "kubectl",
		Description: "Run kubectl commands",
		InputSchema: map[string]interface{}{"type": "object"},
		Handler:     func(ctx context.Context, args map[string]interface{}) (string, error) { return "", nil },
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_ = s.Run(ctx)

	output := stdout.String()
	if !strings.Contains(output, "kubectl") {
		t.Errorf("response should contain tool name, got: %s", output)
	}
}

func TestHandleCallTool(t *testing.T) {
	stdin := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hello"}}}
`)
	stdout := &bytes.Buffer{}

	s := NewWithIO("test", "1.0", stdin, stdout)
	s.RegisterTool(&Tool{
		Name:        "echo",
		Description: "Echo text",
		InputSchema: map[string]interface{}{"type": "object"},
		Handler: func(ctx context.Context, args map[string]interface{}) (string, error) {
			if text, ok := args["text"].(string); ok {
				return text, nil
			}
			return "", nil
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_ = s.Run(ctx)

	output := stdout.String()
	if !strings.Contains(output, "hello") {
		t.Errorf("response should contain echoed text, got: %s", output)
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	stdin := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"unknown/method"}
`)
	stdout := &bytes.Buffer{}

	s := NewWithIO("test", "1.0", stdin, stdout)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_ = s.Run(ctx)

	output := stdout.String()
	if !strings.Contains(output, "Method not found") {
		t.Errorf("response should contain error, got: %s", output)
	}
}

func TestHandleInvalidJSON(t *testing.T) {
	stdin := strings.NewReader(`{invalid json}
`)
	stdout := &bytes.Buffer{}

	s := NewWithIO("test", "1.0", stdin, stdout)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_ = s.Run(ctx)

	output := stdout.String()
	if !strings.Contains(output, "Parse error") {
		t.Errorf("response should contain parse error, got: %s", output)
	}
}

func TestHandlePing(t *testing.T) {
	stdin := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}
`)
	stdout := &bytes.Buffer{}

	s := NewWithIO("test", "1.0", stdin, stdout)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_ = s.Run(ctx)

	output := stdout.String()

	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("ping should not return error: %v", resp.Error)
	}
}

func TestServerAlreadyRunning(t *testing.T) {
	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	s := NewWithIO("test", "1.0", stdin, stdout)

	// Simulate running state
	s.running.Store(true)

	ctx := context.Background()
	err := s.Run(ctx)

	if err == nil {
		t.Error("expected error when server already running")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error should mention already running, got: %v", err)
	}
}

func TestCallUnknownTool(t *testing.T) {
	stdin := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nonexistent"}}
`)
	stdout := &bytes.Buffer{}

	s := NewWithIO("test", "1.0", stdin, stdout)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_ = s.Run(ctx)

	output := stdout.String()
	if !strings.Contains(output, "Unknown tool") {
		t.Errorf("response should contain unknown tool error, got: %s", output)
	}
}
