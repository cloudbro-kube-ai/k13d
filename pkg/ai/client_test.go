package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: "http://localhost:8080",
		APIKey:   "test-key",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("expected client to be created")
	}

	if client.cfg != cfg {
		t.Error("client config doesn't match")
	}
}

func TestClient_IsReady(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.LLMConfig
		want bool
	}{
		{
			name: "nil client",
			cfg:  nil,
			want: false,
		},
		{
			name: "valid config",
			cfg: &config.LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				Endpoint: "http://localhost:8080",
				APIKey:   "test-key",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *Client
			if tt.cfg != nil {
				client, _ = NewClient(tt.cfg)
			}
			got := client.IsReady()
			if got != tt.want {
				t.Errorf("IsReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetModel(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.LLMConfig
		want string
	}{
		{
			name: "nil client",
			cfg:  nil,
			want: "",
		},
		{
			name: "valid config",
			cfg: &config.LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				Endpoint: "http://localhost:8080",
				APIKey:   "test-key",
			},
			want: "gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *Client
			if tt.cfg != nil {
				client, _ = NewClient(tt.cfg)
			}
			got := client.GetModel()
			if got != tt.want {
				t.Errorf("GetModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetProvider(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.LLMConfig
		want string
	}{
		{
			name: "nil client",
			cfg:  nil,
			want: "",
		},
		{
			name: "valid config",
			cfg: &config.LLMConfig{
				Provider: "openai",
				Model:    "gpt-4",
				Endpoint: "http://localhost:8080",
				APIKey:   "test-key",
			},
			want: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *Client
			if tt.cfg != nil {
				client, _ = NewClient(tt.cfg)
			}
			got := client.GetProvider()
			if got != tt.want {
				t.Errorf("GetProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_AskNonStreaming(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content-type")
		}

		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key authorization")
		}

		// Return mock response in OpenAI format
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"test-123","choices":[{"message":{"content":"Hello from AI"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}

	client, _ := NewClient(cfg)

	response, err := client.AskNonStreaming(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("AskNonStreaming() error = %v", err)
	}

	if response != "Hello from AI" {
		t.Errorf("AskNonStreaming() = %v, want 'Hello from AI'", response)
	}
}

func TestClient_AskNonStreaming_Error(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}

	client, _ := NewClient(cfg)

	_, err := client.AskNonStreaming(context.Background(), "Hello")
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestClient_Ask_Streaming(t *testing.T) {
	// Create a mock server for streaming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		// Send streaming response
		resp1 := `{"id":"test","choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`
		resp2 := `{"id":"test","choices":[{"delta":{"content":" World"},"finish_reason":null}]}`

		_, _ = w.Write([]byte("data: " + resp1 + "\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: " + resp2 + "\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}

	client, _ := NewClient(cfg)

	var result string
	err := client.Ask(context.Background(), "Hello", func(chunk string) {
		result += chunk
	})

	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}

	if result != "Hello World" {
		t.Errorf("Ask() streamed result = %v, want 'Hello World'", result)
	}
}

func TestClient_TestConnection_Success(t *testing.T) {
	// Create a mock server that returns OK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"test-123","choices":[{"message":{"content":"OK"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: server.URL,
		APIKey:   "test-key",
	}

	client, _ := NewClient(cfg)
	status := client.TestConnection(context.Background())

	if !status.Connected {
		t.Errorf("TestConnection() Connected = false, want true")
	}
	if status.Provider != "openai" {
		t.Errorf("TestConnection() Provider = %v, want openai", status.Provider)
	}
	if status.Model != "gpt-4" {
		t.Errorf("TestConnection() Model = %v, want gpt-4", status.Model)
	}
	if status.ResponseTime < 0 {
		t.Errorf("TestConnection() ResponseTime = %v, want >= 0", status.ResponseTime)
	}
	if status.Error != "" {
		t.Errorf("TestConnection() Error = %v, want empty", status.Error)
	}
}

func TestClient_TestConnection_Failure(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: server.URL,
		APIKey:   "invalid-key",
	}

	client, _ := NewClient(cfg)
	status := client.TestConnection(context.Background())

	if status.Connected {
		t.Errorf("TestConnection() Connected = true, want false")
	}
	if status.Error == "" {
		t.Error("TestConnection() Error should not be empty for failed connection")
	}
}

func TestClient_TestConnection_NilClient(t *testing.T) {
	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: "http://localhost:8080",
		APIKey:   "test-key",
	}

	client, _ := NewClient(cfg)
	// Simulate provider being nil
	client.provider = nil

	status := client.TestConnection(context.Background())

	if status.Connected {
		t.Errorf("TestConnection() Connected = true, want false for nil provider")
	}
	if status.Error == "" {
		t.Error("TestConnection() Error should not be empty for nil provider")
	}
}

func TestClient_GetEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.LLMConfig
		expected string
	}{
		{
			name: "custom endpoint",
			cfg: &config.LLMConfig{
				Provider: "openai",
				Endpoint: "https://custom.endpoint.com",
				APIKey:   "key",
			},
			expected: "https://custom.endpoint.com",
		},
		{
			name: "solar default",
			cfg: &config.LLMConfig{
				Provider: "solar",
				APIKey:   "key",
			},
			expected: "https://api.upstage.ai/v1",
		},
		{
			name: "openai default",
			cfg: &config.LLMConfig{
				Provider: "openai",
				APIKey:   "key",
			},
			expected: "https://api.openai.com/v1",
		},
		{
			name: "ollama default",
			cfg: &config.LLMConfig{
				Provider: "ollama",
				APIKey:   "key",
			},
			expected: "http://localhost:11434",
		},
		{
			name: "gemini default",
			cfg: &config.LLMConfig{
				Provider: "gemini",
				APIKey:   "key",
			},
			expected: "https://generativelanguage.googleapis.com/v1beta",
		},
		{
			name: "anthropic default",
			cfg: &config.LLMConfig{
				Provider: "anthropic",
				APIKey:   "key",
			},
			expected: "https://api.anthropic.com",
		},
		{
			name: "upstage default (alias for solar)",
			cfg: &config.LLMConfig{
				Provider: "upstage",
				APIKey:   "key",
			},
			expected: "https://api.upstage.ai/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg)
			if err != nil {
				t.Skipf("Skipping test (provider creation failed): %v", err)
			}

			got := client.GetEndpoint()
			if got != tt.expected {
				t.Errorf("GetEndpoint() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConnectionStatus_Fields(t *testing.T) {
	// Test ConnectionStatus struct
	status := &ConnectionStatus{
		Connected:    true,
		Provider:     "openai",
		Model:        "gpt-4",
		Endpoint:     "https://api.openai.com/v1",
		ResponseTime: 150,
		Message:      "Connected successfully",
	}

	if !status.Connected {
		t.Error("ConnectionStatus.Connected should be true")
	}
	if status.Provider != "openai" {
		t.Errorf("ConnectionStatus.Provider = %v, want openai", status.Provider)
	}
	if status.Model != "gpt-4" {
		t.Errorf("ConnectionStatus.Model = %v, want gpt-4", status.Model)
	}
	if status.ResponseTime != 150 {
		t.Errorf("ConnectionStatus.ResponseTime = %v, want 150", status.ResponseTime)
	}
}
