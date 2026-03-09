package mcp

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.servers == nil {
		t.Fatal("servers map not initialized")
	}
}

func TestClientGetAllToolsEmpty(t *testing.T) {
	client := NewClient()
	tools := client.GetAllTools()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

func TestClientIsConnected(t *testing.T) {
	client := NewClient()
	if client.IsConnected("nonexistent") {
		t.Error("expected IsConnected to return false for nonexistent server")
	}
}

func TestClientGetConnectedServers(t *testing.T) {
	client := NewClient()
	servers := client.GetConnectedServers()
	if len(servers) != 0 {
		t.Errorf("expected 0 connected servers, got %d", len(servers))
	}
}

func TestClientDisconnectNonexistent(t *testing.T) {
	client := NewClient()
	err := client.Disconnect("nonexistent")
	if err != nil {
		t.Errorf("Disconnect of nonexistent server should return nil, got %v", err)
	}
}

func TestClientDisconnectAll(t *testing.T) {
	client := NewClient()
	// Should not panic on empty client
	client.DisconnectAll()
	if len(client.servers) != 0 {
		t.Error("expected empty servers after DisconnectAll")
	}
}

func TestClientCallToolNotFound(t *testing.T) {
	client := NewClient()
	_, err := client.CallTool(context.TODO(), "nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

func TestMCPToolExecutorAdapter(t *testing.T) {
	client := NewClient()
	adapter := NewMCPToolExecutor(client)
	if adapter == nil {
		t.Fatal("NewMCPToolExecutor returned nil")
	}
	if adapter.client != client {
		t.Error("adapter client mismatch")
	}
}
