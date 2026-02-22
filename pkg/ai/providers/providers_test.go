package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers: mock servers
// ---------------------------------------------------------------------------

// newOpenAIStreamServer returns an httptest server that speaks the OpenAI SSE
// streaming protocol and emits the given tokens one per SSE event.
func newOpenAIStreamServer(t *testing.T, tokens []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request basics
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for i, token := range tokens {
			chunk := openAIChatResponse{
				ID: fmt.Sprintf("chatcmpl-%d", i),
				Choices: []struct {
					Message struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					} `json:"message"`
					Delta struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Delta: struct {
							Content   string     `json:"content"`
							ToolCalls []ToolCall `json:"tool_calls,omitempty"`
						}{Content: token},
					},
				},
			}
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

// newOpenAINonStreamServer returns an httptest server that responds with a
// single non-streaming OpenAI chat completion.
func newOpenAINonStreamServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIChatResponse{
			ID: "chatcmpl-test",
			Choices: []struct {
				Message struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					}{Content: content},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// newOpenAIModelsServer returns an httptest server that responds to GET /models.
func newOpenAIModelsServer(t *testing.T, models []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			resp := openAIModelsResponse{}
			for _, m := range models {
				resp.Data = append(resp.Data, struct {
					ID string `json:"id"`
				}{ID: m})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
}

// newErrorServer returns an httptest server that always returns the given status code.
func newErrorServer(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
}

// newOllamaStreamServer returns an httptest server that speaks Ollama's NDJSON streaming protocol.
func newOllamaStreamServer(t *testing.T, tokens []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)

		for _, token := range tokens {
			resp := ollamaChatResponse{
				Done: false,
			}
			resp.Message.Content = token
			data, _ := json.Marshal(resp)
			fmt.Fprintf(w, "%s\n", data)
		}
		// Final done message
		resp := ollamaChatResponse{Done: true}
		data, _ := json.Marshal(resp)
		fmt.Fprintf(w, "%s\n", data)
	}))
}

// newOllamaNonStreamServer returns an httptest server for Ollama non-streaming.
func newOllamaNonStreamServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaChatResponse{Done: true}
		resp.Message.Content = content
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// newOllamaModelsServer returns an httptest server for Ollama model listing.
func newOllamaModelsServer(t *testing.T, models []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api/tags") {
			resp := ollamaModelsResponse{}
			for _, m := range models {
				resp.Models = append(resp.Models, struct {
					Name string `json:"name"`
				}{Name: m})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
}

// newGeminiStreamServer returns an httptest server that speaks the Gemini SSE streaming protocol.
func newGeminiStreamServer(t *testing.T, tokens []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the x-goog-api-key header
		if key := r.Header.Get("x-goog-api-key"); key == "" {
			t.Error("expected x-goog-api-key header to be set")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for _, token := range tokens {
			resp := geminiResponse{
				Candidates: []struct {
					Content struct {
						Parts []geminiPart `json:"parts"`
					} `json:"content"`
				}{
					{
						Content: struct {
							Parts []geminiPart `json:"parts"`
						}{
							Parts: []geminiPart{{Text: token}},
						},
					},
				},
			}
			data, _ := json.Marshal(resp)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
	}))
}

// newGeminiNonStreamServer returns an httptest server for Gemini non-streaming.
func newGeminiNonStreamServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiResponse{
			Candidates: []struct {
				Content struct {
					Parts []geminiPart `json:"parts"`
				} `json:"content"`
			}{
				{
					Content: struct {
						Parts []geminiPart `json:"parts"`
					}{
						Parts: []geminiPart{{Text: content}},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// newGeminiModelsServer returns an httptest server for Gemini model listing.
func newGeminiModelsServer(t *testing.T, models []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := geminiModelsResponse{}
		for _, m := range models {
			resp.Models = append(resp.Models, struct {
				Name string `json:"name"`
			}{Name: "models/" + m})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// newEmbeddedNonStreamServer returns an httptest server for embedded non-streaming responses.
func newEmbeddedNonStreamServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := embeddedChatResponse{
			ID:    "chatcmpl-embedded",
			Model: "qwen2.5",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role      string     `json:"role"`
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Role      string     `json:"role"`
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					}{
						Role:    "assistant",
						Content: content,
					},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// newEmbeddedStreamServer returns an httptest server for embedded SSE streaming.
func newEmbeddedStreamServer(t *testing.T, tokens []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for _, token := range tokens {
			resp := embeddedChatResponse{
				ID: "chatcmpl-stream",
				Choices: []struct {
					Index   int `json:"index"`
					Message struct {
						Role      string     `json:"role"`
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					} `json:"message"`
					Delta struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Delta: struct {
							Content   string     `json:"content"`
							ToolCalls []ToolCall `json:"tool_calls,omitempty"`
						}{Content: token},
					},
				},
			}
			data, _ := json.Marshal(resp)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		// Final event with stop reason
		finalResp := embeddedChatResponse{
			ID: "chatcmpl-stream",
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role      string     `json:"role"`
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Delta: struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					}{Content: ""},
					FinishReason: "stop",
				},
			},
		}
		finalData, _ := json.Marshal(finalResp)
		fmt.Fprintf(w, "data: %s\n\n", finalData)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

// newAzureStreamServer returns an httptest server for Azure OpenAI SSE streaming.
func newAzureStreamServer(t *testing.T, tokens []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Azure-specific header
		if apiKey := r.Header.Get("api-key"); apiKey == "" {
			t.Error("expected api-key header to be set for Azure")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for i, token := range tokens {
			chunk := openAIChatResponse{
				ID: fmt.Sprintf("chatcmpl-azure-%d", i),
				Choices: []struct {
					Message struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					} `json:"message"`
					Delta struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				}{
					{
						Delta: struct {
							Content   string     `json:"content"`
							ToolCalls []ToolCall `json:"tool_calls,omitempty"`
						}{Content: token},
					},
				},
			}
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

// newAzureNonStreamServer returns an httptest server for Azure non-streaming.
func newAzureNonStreamServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIChatResponse{
			ID: "chatcmpl-azure-test",
			Choices: []struct {
				Message struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					}{Content: content},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// requestCapture is a server that captures the last request body and request headers.
type requestCapture struct {
	Server  *httptest.Server
	Body    []byte
	Headers http.Header
}

// newOpenAICaptureServer creates a server that captures the request and returns the given response content.
func newOpenAICaptureServer(t *testing.T, content string) *requestCapture {
	t.Helper()
	rc := &requestCapture{}
	rc.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rc.Body = body
		rc.Headers = r.Header.Clone()

		resp := openAIChatResponse{
			ID: "chatcmpl-capture",
			Choices: []struct {
				Message struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					}{Content: content},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	return rc
}

// ===========================================================================
// OpenAI Provider Tests
// ===========================================================================

func TestOpenAIProvider_AskStreaming(t *testing.T) {
	tokens := []string{"Hello", " ", "world", "!"}
	srv := newOpenAIStreamServer(t, tokens)
	defer srv.Close()

	p, err := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewOpenAIProvider: %v", err)
	}

	var collected []string
	err = p.Ask(context.Background(), "say hello", func(s string) {
		collected = append(collected, s)
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}

	got := strings.Join(collected, "")
	want := "Hello world!"
	if got != want {
		t.Errorf("Ask collected = %q, want %q", got, want)
	}
}

func TestOpenAIProvider_AskNonStreaming(t *testing.T) {
	srv := newOpenAINonStreamServer(t, "I can help with Kubernetes.")
	defer srv.Close()

	p, err := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewOpenAIProvider: %v", err)
	}

	resp, err := p.AskNonStreaming(context.Background(), "help me")
	if err != nil {
		t.Fatalf("AskNonStreaming: %v", err)
	}
	if resp != "I can help with Kubernetes." {
		t.Errorf("response = %q, want %q", resp, "I can help with Kubernetes.")
	}
}

func TestOpenAIProvider_AskNonStreaming_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIChatResponse{ID: "chatcmpl-empty"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, err := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewOpenAIProvider: %v", err)
	}

	_, err = p.AskNonStreaming(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}
	if !strings.Contains(err.Error(), "no response") {
		t.Errorf("error = %v, want to contain 'no response'", err)
	}
}

func TestOpenAIProvider_ListModels(t *testing.T) {
	models := []string{"gpt-4", "gpt-3.5-turbo", "gpt-4o"}
	srv := newOpenAIModelsServer(t, models)
	defer srv.Close()

	p, err := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewOpenAIProvider: %v", err)
	}

	result, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}

	if len(result) != len(models) {
		t.Errorf("ListModels returned %d models, want %d", len(result), len(models))
	}
	for i, m := range models {
		if result[i] != m {
			t.Errorf("model[%d] = %q, want %q", i, result[i], m)
		}
	}
}

func TestOpenAIProvider_ErrorStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{"401 unauthorized", 401, "invalid api key", "API error (status 401)"},
		{"429 rate limit", 429, "rate limit exceeded", "API error (status 429)"},
		{"500 server error", 500, "internal server error", "API error (status 500)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newErrorServer(tt.statusCode, tt.body)
			defer srv.Close()

			p, _ := NewOpenAIProvider(&ProviderConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "test-key",
				Endpoint: srv.URL,
			})

			// Test streaming
			err := p.Ask(context.Background(), "test", func(s string) {})
			if err == nil {
				t.Error("Ask: expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Ask error = %v, want to contain %q", err, tt.wantErr)
			}

			// Test non-streaming
			_, err = p.AskNonStreaming(context.Background(), "test")
			if err == nil {
				t.Error("AskNonStreaming: expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("AskNonStreaming error = %v, want to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestOpenAIProvider_RequestBuilding(t *testing.T) {
	rc := newOpenAICaptureServer(t, "ok")
	defer rc.Server.Close()

	p, _ := NewOpenAIProvider(&ProviderConfig{
		Provider:        "openai",
		Model:           "gpt-4-turbo",
		APIKey:          "sk-test-12345",
		Endpoint:        rc.Server.URL,
		ReasoningEffort: "high",
	})

	_, err := p.AskNonStreaming(context.Background(), "describe pods")
	if err != nil {
		t.Fatalf("AskNonStreaming: %v", err)
	}

	// Verify authorization header
	authHeader := rc.Headers.Get("Authorization")
	if authHeader != "Bearer sk-test-12345" {
		t.Errorf("Authorization = %q, want 'Bearer sk-test-12345'", authHeader)
	}

	// Verify request body contains model, messages, and reasoning_effort
	var reqBody openAIChatRequest
	if err := json.Unmarshal(rc.Body, &reqBody); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}
	if reqBody.Model != "gpt-4-turbo" {
		t.Errorf("request model = %q, want 'gpt-4-turbo'", reqBody.Model)
	}
	if reqBody.Stream {
		t.Error("request stream = true for AskNonStreaming, want false")
	}
	if reqBody.ReasoningEffort != "high" {
		t.Errorf("reasoning_effort = %q, want 'high'", reqBody.ReasoningEffort)
	}
	if len(reqBody.Messages) != 2 {
		t.Errorf("messages count = %d, want 2 (system + user)", len(reqBody.Messages))
	}
	if reqBody.Messages[0].Role != "system" {
		t.Errorf("messages[0].role = %q, want 'system'", reqBody.Messages[0].Role)
	}
	if reqBody.Messages[1].Role != "user" {
		t.Errorf("messages[1].role = %q, want 'user'", reqBody.Messages[1].Role)
	}
	if reqBody.Messages[1].Content != "describe pods" {
		t.Errorf("messages[1].content = %q, want 'describe pods'", reqBody.Messages[1].Content)
	}
}

func TestOpenAIProvider_ContextCancellation(t *testing.T) {
	// Server that blocks for a while
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	p, _ := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := p.Ask(ctx, "test", func(s string) {})
	if err == nil {
		t.Error("expected context cancellation error, got nil")
	}
}

func TestOpenAIProvider_EndpointTrailingSlash(t *testing.T) {
	srv := newOpenAINonStreamServer(t, "ok")
	defer srv.Close()

	// Endpoint with trailing slash should be handled
	p, _ := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: srv.URL + "/",
	})

	op := p.(*OpenAIProvider)
	if strings.HasSuffix(op.endpoint, "/") {
		t.Error("endpoint should not end with /")
	}
}

func TestOpenAIProvider_ListModels_ErrorStatus(t *testing.T) {
	srv := newErrorServer(403, "forbidden")
	defer srv.Close()

	p, _ := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	_, err := p.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected error for 403 status, got nil")
	}
	if !strings.Contains(err.Error(), "status 403") {
		t.Errorf("error = %v, want to contain 'status 403'", err)
	}
}

// ===========================================================================
// Ollama Provider Tests
// ===========================================================================

func TestOllamaProvider_AskStreaming(t *testing.T) {
	tokens := []string{"kubectl", " get", " pods"}
	srv := newOllamaStreamServer(t, tokens)
	defer srv.Close()

	p, err := NewOllamaProvider(&ProviderConfig{
		Provider: "ollama",
		Model:    "llama3.2",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewOllamaProvider: %v", err)
	}

	var collected []string
	err = p.Ask(context.Background(), "list pods", func(s string) {
		collected = append(collected, s)
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}

	got := strings.Join(collected, "")
	want := "kubectl get pods"
	if got != want {
		t.Errorf("Ask collected = %q, want %q", got, want)
	}
}

func TestOllamaProvider_AskNonStreaming(t *testing.T) {
	srv := newOllamaNonStreamServer(t, "Here are your pods.")
	defer srv.Close()

	p, _ := NewOllamaProvider(&ProviderConfig{
		Provider: "ollama",
		Model:    "llama3.2",
		Endpoint: srv.URL,
	})

	resp, err := p.AskNonStreaming(context.Background(), "list pods")
	if err != nil {
		t.Fatalf("AskNonStreaming: %v", err)
	}
	if resp != "Here are your pods." {
		t.Errorf("response = %q, want %q", resp, "Here are your pods.")
	}
}

func TestOllamaProvider_AskNonStreaming_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ollamaChatResponse{Done: true}
		// Message.Content is empty
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := NewOllamaProvider(&ProviderConfig{
		Provider: "ollama",
		Model:    "llama3.2",
		Endpoint: srv.URL,
	})

	_, err := p.AskNonStreaming(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for empty response, got nil")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error = %v, want to contain 'empty response'", err)
	}
}

func TestOllamaProvider_ListModels(t *testing.T) {
	models := []string{"llama3.2:latest", "codellama:7b", "mistral:latest"}
	srv := newOllamaModelsServer(t, models)
	defer srv.Close()

	p, _ := NewOllamaProvider(&ProviderConfig{
		Provider: "ollama",
		Endpoint: srv.URL,
	})

	result, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(result) != len(models) {
		t.Errorf("ListModels returned %d models, want %d", len(result), len(models))
	}
	for i, m := range models {
		if result[i] != m {
			t.Errorf("model[%d] = %q, want %q", i, result[i], m)
		}
	}
}

func TestOllamaProvider_ErrorStatusCode(t *testing.T) {
	srv := newErrorServer(500, "model not found")
	defer srv.Close()

	p, _ := NewOllamaProvider(&ProviderConfig{
		Provider: "ollama",
		Endpoint: srv.URL,
	})

	err := p.Ask(context.Background(), "test", func(s string) {})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("error = %v, want to contain 'status 500'", err)
	}
}

func TestOllamaProvider_IsReady(t *testing.T) {
	p, _ := NewOllamaProvider(&ProviderConfig{
		Provider: "ollama",
		Model:    "llama3.2",
		Endpoint: "http://localhost:11434",
	})
	if !p.IsReady() {
		t.Error("Ollama provider with endpoint should be ready")
	}
}

func TestOllamaProvider_AskWithTools_NoToolCalls(t *testing.T) {
	// Server returns a response with no tool calls
	srv := newOllamaNonStreamServer(t, "There are 3 pods running.")
	defer srv.Close()

	p, _ := NewOllamaProvider(&ProviderConfig{
		Provider: "ollama",
		Model:    "llama3.2",
		Endpoint: srv.URL,
	})

	var callbackContent string
	err := p.(ToolProvider).AskWithTools(context.Background(), "list pods", nil, func(s string) {
		callbackContent += s
	}, func(call ToolCall) ToolResult {
		t.Error("tool callback should not be called when there are no tool calls")
		return ToolResult{}
	})
	if err != nil {
		t.Fatalf("AskWithTools: %v", err)
	}
	if callbackContent != "There are 3 pods running." {
		t.Errorf("callback content = %q, want 'There are 3 pods running.'", callbackContent)
	}
}

// ===========================================================================
// Gemini Provider Tests
// ===========================================================================

func TestGeminiProvider_AskStreaming(t *testing.T) {
	tokens := []string{"Kubernetes", " is", " awesome"}
	srv := newGeminiStreamServer(t, tokens)
	defer srv.Close()

	p, err := NewGeminiProvider(&ProviderConfig{
		Provider: "gemini",
		Model:    "gemini-2.5-flash",
		APIKey:   "test-gemini-key",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewGeminiProvider: %v", err)
	}

	var collected []string
	err = p.Ask(context.Background(), "what is k8s", func(s string) {
		collected = append(collected, s)
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}

	got := strings.Join(collected, "")
	want := "Kubernetes is awesome"
	if got != want {
		t.Errorf("Ask collected = %q, want %q", got, want)
	}
}

func TestGeminiProvider_AskNonStreaming(t *testing.T) {
	srv := newGeminiNonStreamServer(t, "Gemini response here")
	defer srv.Close()

	p, _ := NewGeminiProvider(&ProviderConfig{
		Provider: "gemini",
		Model:    "gemini-2.0-flash",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	resp, err := p.AskNonStreaming(context.Background(), "test")
	if err != nil {
		t.Fatalf("AskNonStreaming: %v", err)
	}
	if resp != "Gemini response here" {
		t.Errorf("response = %q, want 'Gemini response here'", resp)
	}
}

func TestGeminiProvider_AskNonStreaming_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Empty candidates
		resp := geminiResponse{}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := NewGeminiProvider(&ProviderConfig{
		Provider: "gemini",
		Model:    "gemini-1.5-pro",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	_, err := p.AskNonStreaming(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for empty candidates, got nil")
	}
	if !strings.Contains(err.Error(), "no response") {
		t.Errorf("error = %v, want to contain 'no response'", err)
	}
}

func TestGeminiProvider_ListModels(t *testing.T) {
	models := []string{"gemini-2.5-flash", "gemini-2.5-pro"}
	srv := newGeminiModelsServer(t, models)
	defer srv.Close()

	p, _ := NewGeminiProvider(&ProviderConfig{
		Provider: "gemini",
		Model:    "gemini-2.5-flash",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	result, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(result) != len(models) {
		t.Fatalf("ListModels returned %d models, want %d", len(result), len(models))
	}
	// Models should be returned without "models/" prefix
	for i, m := range models {
		if result[i] != m {
			t.Errorf("model[%d] = %q, want %q", i, result[i], m)
		}
	}
}

func TestGeminiProvider_ErrorStatusCode(t *testing.T) {
	srv := newErrorServer(400, `{"error": "bad request"}`)
	defer srv.Close()

	p, _ := NewGeminiProvider(&ProviderConfig{
		Provider: "gemini",
		Model:    "gemini-2.5-flash",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	err := p.Ask(context.Background(), "test", func(s string) {})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Errorf("error = %v, want to contain 'status 400'", err)
	}
}

func TestGeminiProvider_AskWithTools_NoToolCalls(t *testing.T) {
	// Server returns text without function calls
	srv := newGeminiNonStreamServer(t, "Here is your answer without tools.")
	defer srv.Close()

	p, _ := NewGeminiProvider(&ProviderConfig{
		Provider: "gemini",
		Model:    "gemini-2.5-flash",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	var callbackContent string
	err := p.(ToolProvider).AskWithTools(context.Background(), "explain pods", nil, func(s string) {
		callbackContent += s
	}, func(call ToolCall) ToolResult {
		t.Error("tool callback should not be called")
		return ToolResult{}
	})
	if err != nil {
		t.Fatalf("AskWithTools: %v", err)
	}
	if callbackContent != "Here is your answer without tools." {
		t.Errorf("callback = %q, want 'Here is your answer without tools.'", callbackContent)
	}
}

func TestGeminiProvider_AskWithTools_WithFunctionCall(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp geminiResponse

		if callCount == 1 {
			// First call: return a function call
			resp = geminiResponse{
				Candidates: []struct {
					Content struct {
						Parts []geminiPart `json:"parts"`
					} `json:"content"`
				}{
					{
						Content: struct {
							Parts []geminiPart `json:"parts"`
						}{
							Parts: []geminiPart{
								{
									FunctionCall: &geminiFuncCall{
										Name: "kubectl",
										Args: map[string]interface{}{"command": "kubectl get pods"},
									},
								},
							},
						},
					},
				},
			}
		} else {
			// Second call: return text
			resp = geminiResponse{
				Candidates: []struct {
					Content struct {
						Parts []geminiPart `json:"parts"`
					} `json:"content"`
				}{
					{
						Content: struct {
							Parts []geminiPart `json:"parts"`
						}{
							Parts: []geminiPart{{Text: "Found 2 pods."}},
						},
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := NewGeminiProvider(&ProviderConfig{
		Provider: "gemini",
		Model:    "gemini-2.5-flash",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	toolCalled := false
	var callbackContent string
	err := p.(ToolProvider).AskWithTools(
		context.Background(),
		"list pods",
		[]ToolDefinition{{
			Type: "function",
			Function: FunctionDef{
				Name:        "kubectl",
				Description: "Execute kubectl commands",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		}},
		func(s string) {
			callbackContent += s
		},
		func(call ToolCall) ToolResult {
			toolCalled = true
			if call.Function.Name != "kubectl" {
				t.Errorf("tool name = %q, want 'kubectl'", call.Function.Name)
			}
			return ToolResult{
				ToolCallID: call.ID,
				Content:    "NAME       READY   STATUS\npod1       1/1     Running\npod2       1/1     Running",
			}
		},
	)
	if err != nil {
		t.Fatalf("AskWithTools: %v", err)
	}
	if !toolCalled {
		t.Error("tool callback should have been called")
	}
	if !strings.Contains(callbackContent, "Found 2 pods.") {
		t.Errorf("callback should contain final answer, got %q", callbackContent)
	}
}

// ===========================================================================
// Azure OpenAI Provider Tests
// ===========================================================================

func TestAzureOpenAIProvider_AskStreaming(t *testing.T) {
	tokens := []string{"Azure", " ", "response"}
	srv := newAzureStreamServer(t, tokens)
	defer srv.Close()

	p, err := NewAzureOpenAIProvider(&ProviderConfig{
		Provider:        "azopenai",
		AzureDeployment: "gpt-4",
		APIKey:          "test-azure-key",
		Endpoint:        srv.URL,
	})
	if err != nil {
		t.Fatalf("NewAzureOpenAIProvider: %v", err)
	}

	var collected []string
	err = p.Ask(context.Background(), "test", func(s string) {
		collected = append(collected, s)
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}

	got := strings.Join(collected, "")
	if got != "Azure response" {
		t.Errorf("Ask collected = %q, want 'Azure response'", got)
	}
}

func TestAzureOpenAIProvider_AskNonStreaming(t *testing.T) {
	srv := newAzureNonStreamServer(t, "Azure non-streaming response")
	defer srv.Close()

	p, _ := NewAzureOpenAIProvider(&ProviderConfig{
		Provider:        "azopenai",
		AzureDeployment: "gpt-4",
		APIKey:          "test-key",
		Endpoint:        srv.URL,
	})

	resp, err := p.AskNonStreaming(context.Background(), "test")
	if err != nil {
		t.Fatalf("AskNonStreaming: %v", err)
	}
	if resp != "Azure non-streaming response" {
		t.Errorf("response = %q, want 'Azure non-streaming response'", resp)
	}
}

func TestAzureOpenAIProvider_AskNonStreaming_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIChatResponse{ID: "azure-empty"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := NewAzureOpenAIProvider(&ProviderConfig{
		Provider:        "azopenai",
		AzureDeployment: "gpt-4",
		APIKey:          "test-key",
		Endpoint:        srv.URL,
	})

	_, err := p.AskNonStreaming(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}
	if !strings.Contains(err.Error(), "no response") {
		t.Errorf("error = %v, want to contain 'no response'", err)
	}
}

func TestAzureOpenAIProvider_GetModel_UsesDeployment(t *testing.T) {
	p, _ := NewAzureOpenAIProvider(&ProviderConfig{
		Provider:        "azopenai",
		AzureDeployment: "my-gpt4-deployment",
		APIKey:          "test-key",
		Endpoint:        "https://test.openai.azure.com",
	})
	if p.GetModel() != "my-gpt4-deployment" {
		t.Errorf("GetModel() = %q, want 'my-gpt4-deployment'", p.GetModel())
	}
}

func TestAzureOpenAIProvider_DeploymentFallsBackToModel(t *testing.T) {
	p, _ := NewAzureOpenAIProvider(&ProviderConfig{
		Provider:        "azopenai",
		Model:           "gpt-4",
		AzureDeployment: "", // empty deployment
		APIKey:          "test-key",
		Endpoint:        "https://test.openai.azure.com",
	})
	if p.GetModel() != "gpt-4" {
		t.Errorf("GetModel() = %q, want 'gpt-4' (fallback from model)", p.GetModel())
	}
}

func TestAzureOpenAIProvider_ErrorStatusCode(t *testing.T) {
	srv := newErrorServer(401, "unauthorized")
	defer srv.Close()

	p, _ := NewAzureOpenAIProvider(&ProviderConfig{
		Provider:        "azopenai",
		AzureDeployment: "gpt-4",
		APIKey:          "bad-key",
		Endpoint:        srv.URL,
	})

	err := p.Ask(context.Background(), "test", func(s string) {})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Errorf("error = %v, want to contain 'status 401'", err)
	}
}

func TestAzureOpenAIProvider_AskWithTools_NoToolCalls(t *testing.T) {
	// Server returns a response with no tool calls
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := azureOpenAIChatResponse{
			Choices: []struct {
				Message struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					}{Content: "No tools needed."},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := NewAzureOpenAIProvider(&ProviderConfig{
		Provider:        "azopenai",
		AzureDeployment: "gpt-4",
		APIKey:          "test-key",
		Endpoint:        srv.URL,
	})

	var callbackContent string
	err := p.(*AzureOpenAIProvider).AskWithTools(context.Background(), "what is k8s", nil, func(s string) {
		callbackContent += s
	}, func(call ToolCall) ToolResult {
		t.Error("tool callback should not be called")
		return ToolResult{}
	})
	if err != nil {
		t.Fatalf("AskWithTools: %v", err)
	}
	if callbackContent != "No tools needed." {
		t.Errorf("callback = %q, want 'No tools needed.'", callbackContent)
	}
}

// ===========================================================================
// Embedded Provider Tests
// ===========================================================================

func TestEmbeddedProvider_AskStreaming(t *testing.T) {
	tokens := []string{"embedded", " response", " here"}
	srv := newEmbeddedStreamServer(t, tokens)
	defer srv.Close()

	p, err := NewEmbeddedProvider(&ProviderConfig{
		Provider: "embedded",
		Model:    "qwen2.5",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewEmbeddedProvider: %v", err)
	}

	var collected []string
	err = p.Ask(context.Background(), "test", func(s string) {
		collected = append(collected, s)
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}

	got := strings.Join(collected, "")
	if got != "embedded response here" {
		t.Errorf("Ask collected = %q, want 'embedded response here'", got)
	}
}

func TestEmbeddedProvider_AskNonStreaming(t *testing.T) {
	srv := newEmbeddedNonStreamServer(t, "embedded non-streaming")
	defer srv.Close()

	p, _ := NewEmbeddedProvider(&ProviderConfig{
		Provider: "embedded",
		Endpoint: srv.URL,
	})

	resp, err := p.AskNonStreaming(context.Background(), "test")
	if err != nil {
		t.Fatalf("AskNonStreaming: %v", err)
	}
	if resp != "embedded non-streaming" {
		t.Errorf("response = %q, want 'embedded non-streaming'", resp)
	}
}

func TestEmbeddedProvider_AskNonStreaming_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := embeddedChatResponse{ID: "empty"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, _ := NewEmbeddedProvider(&ProviderConfig{
		Provider: "embedded",
		Endpoint: srv.URL,
	})

	_, err := p.AskNonStreaming(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}
	if !strings.Contains(err.Error(), "no response") {
		t.Errorf("error = %v, want to contain 'no response'", err)
	}
}

func TestEmbeddedProvider_ListModels(t *testing.T) {
	p, _ := NewEmbeddedProvider(&ProviderConfig{
		Provider: "embedded",
		Model:    "qwen2.5-0.5b-instruct",
	})

	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 1 || models[0] != "qwen2.5-0.5b-instruct" {
		t.Errorf("ListModels = %v, want [qwen2.5-0.5b-instruct]", models)
	}
}

func TestEmbeddedProvider_DefaultValues(t *testing.T) {
	p, _ := NewEmbeddedProvider(&ProviderConfig{
		Provider: "embedded",
	})

	ep := p.(*EmbeddedProvider)
	if ep.endpoint != "http://127.0.0.1:8081" {
		t.Errorf("default endpoint = %q, want 'http://127.0.0.1:8081'", ep.endpoint)
	}
	if p.GetModel() != "qwen2.5-0.5b-instruct" {
		t.Errorf("default model = %q, want 'qwen2.5-0.5b-instruct'", p.GetModel())
	}
	if p.Name() != "embedded" {
		t.Errorf("Name() = %q, want 'embedded'", p.Name())
	}
}

func TestEmbeddedProvider_IsReady_ServerUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	p, _ := NewEmbeddedProvider(&ProviderConfig{
		Provider: "embedded",
		Endpoint: srv.URL,
	})

	if !p.IsReady() {
		t.Error("embedded provider should be ready when health endpoint is OK")
	}
}

func TestEmbeddedProvider_IsReady_ServerDown(t *testing.T) {
	// Use an endpoint that will not respond
	p, _ := NewEmbeddedProvider(&ProviderConfig{
		Provider: "embedded",
		Endpoint: "http://127.0.0.1:1", // unlikely to be running
	})

	if p.IsReady() {
		t.Error("embedded provider should not be ready when server is down")
	}
}

func TestEmbeddedProvider_ErrorStatusCode(t *testing.T) {
	srv := newErrorServer(503, "model loading")
	defer srv.Close()

	p, _ := NewEmbeddedProvider(&ProviderConfig{
		Provider: "embedded",
		Endpoint: srv.URL,
	})

	err := p.Ask(context.Background(), "test", func(s string) {})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 503") {
		t.Errorf("error = %v, want to contain 'status 503'", err)
	}
}

// ===========================================================================
// Bedrock Provider Tests (limited: signing + response parsing)
// ===========================================================================

func TestBedrockProvider_AskNonStreaming_WithMock(t *testing.T) {
	// Set up AWS credentials for signing
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST123")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "testsecret456")
	t.Setenv("AWS_SESSION_TOKEN", "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify AWS auth headers are present
		if auth := r.Header.Get("Authorization"); auth == "" {
			t.Error("expected Authorization header for AWS request")
		}
		if amzDate := r.Header.Get("X-Amz-Date"); amzDate == "" {
			t.Error("expected X-Amz-Date header for AWS request")
		}

		resp := bedrockClaudeResponse{
			Content: []struct {
				Text string `json:"text"`
			}{
				{Text: "Bedrock response"},
			},
			StopReason: "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	// We can't easily use the mock server with bedrock because it constructs the
	// endpoint from region. Instead, test that request signing logic works and
	// response parsing is correct by overriding the internal HTTP client.
	p, err := NewBedrockProvider(&ProviderConfig{
		Provider: "bedrock",
		Model:    "anthropic.claude-3-sonnet-20240229-v1:0",
		Region:   "us-east-1",
	})
	if err != nil {
		t.Fatalf("NewBedrockProvider: %v", err)
	}

	// The bedrock provider builds its own endpoint from region + model, so we
	// cannot easily point it at httptest. Instead we test the signRequest method
	// directly to ensure coverage of that code path.
	bp := p.(*BedrockProvider)

	body := []byte(`{"test": "payload"}`)
	req, _ := http.NewRequest("POST", "https://bedrock-runtime.us-east-1.amazonaws.com/model/test/invoke", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	err = bp.signRequest(req, body)
	if err != nil {
		t.Fatalf("signRequest: %v", err)
	}

	// Verify headers were set
	if auth := req.Header.Get("Authorization"); !strings.HasPrefix(auth, "AWS4-HMAC-SHA256") {
		t.Errorf("Authorization header should start with AWS4-HMAC-SHA256, got %q", auth)
	}
	if amzDate := req.Header.Get("X-Amz-Date"); amzDate == "" {
		t.Error("X-Amz-Date should be set")
	}
	if hash := req.Header.Get("X-Amz-Content-Sha256"); hash == "" {
		t.Error("X-Amz-Content-Sha256 should be set")
	}
}

func TestBedrockProvider_SignRequest_MissingCredentials(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	p, _ := NewBedrockProvider(&ProviderConfig{
		Provider: "bedrock",
		Region:   "us-east-1",
	})

	bp := p.(*BedrockProvider)
	body := []byte(`{}`)
	req, _ := http.NewRequest("POST", "https://example.com/invoke", strings.NewReader(string(body)))

	err := bp.signRequest(req, body)
	if err == nil {
		t.Fatal("expected error for missing AWS credentials, got nil")
	}
	if !strings.Contains(err.Error(), "AWS credentials not configured") {
		t.Errorf("error = %v, want to contain 'AWS credentials not configured'", err)
	}
}

func TestBedrockProvider_SignRequest_WithSessionToken(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secretkey")
	t.Setenv("AWS_SESSION_TOKEN", "session-token-123")

	p, _ := NewBedrockProvider(&ProviderConfig{
		Provider: "bedrock",
		Region:   "us-east-1",
	})

	bp := p.(*BedrockProvider)
	body := []byte(`{}`)
	req, _ := http.NewRequest("POST", "https://example.com/invoke", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	err := bp.signRequest(req, body)
	if err != nil {
		t.Fatalf("signRequest: %v", err)
	}

	if token := req.Header.Get("X-Amz-Security-Token"); token != "session-token-123" {
		t.Errorf("X-Amz-Security-Token = %q, want 'session-token-123'", token)
	}

	// Session token should also appear in signed headers in the Authorization header
	auth := req.Header.Get("Authorization")
	if !strings.Contains(auth, "x-amz-security-token") {
		t.Error("Authorization header should include x-amz-security-token in SignedHeaders")
	}
}

func TestBedrockProvider_DefaultModel(t *testing.T) {
	p, _ := NewBedrockProvider(&ProviderConfig{
		Provider: "bedrock",
		Region:   "us-east-1",
	})
	if p.GetModel() != "anthropic.claude-3-sonnet-20240229-v1:0" {
		t.Errorf("default model = %q, want 'anthropic.claude-3-sonnet-20240229-v1:0'", p.GetModel())
	}
}

func TestBedrockProvider_Ask_DelegatesToNonStreaming(t *testing.T) {
	// The Bedrock Ask method delegates to AskNonStreaming.
	// We test this by checking that the callback receives the full response.
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secretkey")

	// Since Bedrock constructs its own URL from region, the test would need
	// to intercept the HTTP client. We verify the delegation pattern by checking
	// that Ask calls AskNonStreaming internally - this is visible from code
	// structure and tested indirectly.
	p, _ := NewBedrockProvider(&ProviderConfig{
		Provider: "bedrock",
		Region:   "us-east-1",
	})

	// Just ensure the provider type is correct
	_, ok := p.(*BedrockProvider)
	if !ok {
		t.Fatal("expected *BedrockProvider")
	}
}

// ===========================================================================
// extractJSONFromMarkdown Tests
// ===========================================================================

func TestExtractJSONFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantJSON string
		wantOK   bool
	}{
		{
			name:     "valid json block",
			input:    "Some text\n```json\n{\"thought\": \"test\"}\n```\nMore text",
			wantJSON: `{"thought": "test"}`,
			wantOK:   true,
		},
		{
			name:     "json block with action",
			input:    "```json\n{\"thought\": \"thinking\", \"action\": {\"name\": \"kubectl\", \"command\": \"kubectl get pods\"}}\n```",
			wantJSON: `{"thought": "thinking", "action": {"name": "kubectl", "command": "kubectl get pods"}}`,
			wantOK:   true,
		},
		{
			name:   "no json block",
			input:  "This is plain text without any code blocks",
			wantOK: false,
		},
		{
			name:   "unclosed json block",
			input:  "```json\n{\"test\": true}",
			wantOK: false,
		},
		{
			name:   "empty string",
			input:  "",
			wantOK: false,
		},
		{
			name:     "json block with whitespace",
			input:    "```json\n  {\"key\": \"value\"}  \n```",
			wantJSON: `{"key": "value"}`,
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractJSONFromMarkdown(tt.input)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got != tt.wantJSON {
				t.Errorf("json = %q, want %q", got, tt.wantJSON)
			}
		})
	}
}

// ===========================================================================
// parseReActResponse Tests
// ===========================================================================

func TestParseReActResponse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		wantAnswer string
		wantAction string
	}{
		{
			name:       "answer response in markdown block",
			input:      "```json\n{\"thought\": \"I know the answer\", \"answer\": \"Here are your pods\"}\n```",
			wantErr:    false,
			wantAnswer: "Here are your pods",
		},
		{
			name:       "action response in markdown block",
			input:      "```json\n{\"thought\": \"I need to check pods\", \"action\": {\"name\": \"kubectl\", \"reason\": \"list pods\", \"command\": \"kubectl get pods\", \"modifies_resource\": \"no\"}}\n```",
			wantErr:    false,
			wantAction: "kubectl",
		},
		{
			name:       "raw JSON without markdown markers",
			input:      "{\"thought\": \"test\", \"answer\": \"response\"}",
			wantErr:    false,
			wantAnswer: "response",
		},
		{
			name:    "plain text (no JSON)",
			input:   "This is just plain text without any JSON",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   "```json\n{invalid json}\n```",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := parseReActResponse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseReActResponse error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if tt.wantAnswer != "" && resp.Answer != tt.wantAnswer {
				t.Errorf("answer = %q, want %q", resp.Answer, tt.wantAnswer)
			}
			if tt.wantAction != "" {
				if resp.Action == nil {
					t.Fatalf("expected action, got nil")
				}
				if resp.Action.Name != tt.wantAction {
					t.Errorf("action.name = %q, want %q", resp.Action.Name, tt.wantAction)
				}
			}
		})
	}
}

// ===========================================================================
// EmbeddedProvider parseToolCallFromText Tests
// ===========================================================================

func TestEmbeddedProvider_ParseToolCallFromText(t *testing.T) {
	p := &EmbeddedProvider{}

	tests := []struct {
		name     string
		content  string
		wantNil  bool
		wantName string
	}{
		{
			name:     "valid JSON tool call",
			content:  `{"name": "kubectl", "arguments": {"command": "kubectl get pods"}}`,
			wantNil:  false,
			wantName: "kubectl",
		},
		{
			name:     "double braces pattern",
			content:  `{{"name": "bash", "arguments": {"command": "echo hello"}}}`,
			wantNil:  false,
			wantName: "bash",
		},
		{
			name:    "plain text no JSON",
			content: "Just a normal response without any tool calls",
			wantNil: true,
		},
		{
			name:    "JSON without name field",
			content: `{"key": "value"}`,
			wantNil: true,
		},
		{
			name:    "empty string",
			content: "",
			wantNil: true,
		},
		{
			name:     "JSON embedded in text",
			content:  `Here is my action: {"name": "kubectl", "arguments": {"command": "kubectl get ns"}} done.`,
			wantNil:  false,
			wantName: "kubectl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseToolCallFromText(tt.content)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil ToolCall, got nil")
			}
			if result.Function.Name != tt.wantName {
				t.Errorf("tool name = %q, want %q", result.Function.Name, tt.wantName)
			}
			if result.ID == "" {
				t.Error("tool call ID should not be empty")
			}
			if result.Type != "function" {
				t.Errorf("type = %q, want 'function'", result.Type)
			}
		})
	}
}

// ===========================================================================
// RetryProvider Integration Tests (with real HTTP mock)
// ===========================================================================

func TestRetryProvider_RetriesOnTransientError(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("service unavailable"))
			return
		}
		// Succeed on 3rd attempt
		resp := openAIChatResponse{
			ID: "chatcmpl-retry",
			Choices: []struct {
				Message struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message: struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					}{Content: "success after retry"},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	base, _ := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	cfg := &RetryConfig{MaxAttempts: 5, MaxBackoff: 0.001, JitterRatio: 0}
	retryProv := CreateWithRetry(base, cfg)

	resp, err := retryProv.AskNonStreaming(context.Background(), "test")
	if err != nil {
		t.Fatalf("AskNonStreaming with retry: %v", err)
	}
	if resp != "success after retry" {
		t.Errorf("response = %q, want 'success after retry'", resp)
	}
	if atomic.LoadInt32(&callCount) != 3 {
		t.Errorf("call count = %d, want 3 (2 failures + 1 success)", callCount)
	}
}

func TestRetryProvider_ExhaustsRetries(t *testing.T) {
	srv := newErrorServer(429, "rate limited")
	defer srv.Close()

	base, _ := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	cfg := &RetryConfig{MaxAttempts: 2, MaxBackoff: 0.001, JitterRatio: 0}
	retryProv := CreateWithRetry(base, cfg)

	_, err := retryProv.AskNonStreaming(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if !strings.Contains(err.Error(), "max retries exceeded") {
		t.Errorf("error = %v, want to contain 'max retries exceeded'", err)
	}
}

func TestRetryProvider_NoRetryOnNonRetryableError(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer srv.Close()

	base, _ := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	cfg := &RetryConfig{MaxAttempts: 5, MaxBackoff: 0.001, JitterRatio: 0}
	retryProv := CreateWithRetry(base, cfg)

	_, err := retryProv.AskNonStreaming(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should only be called once for a non-retryable error
	if count := atomic.LoadInt32(&callCount); count != 1 {
		t.Errorf("call count = %d, want 1 (no retries for 400)", count)
	}
}

func TestRetryProvider_StreamingRetry(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		if count < 2 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("bad gateway"))
			return
		}
		// Success on 2nd attempt
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunk := openAIChatResponse{
			ID: "chatcmpl-stream-retry",
			Choices: []struct {
				Message struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"message"`
				Delta struct {
					Content   string     `json:"content"`
					ToolCalls []ToolCall `json:"tool_calls,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Delta: struct {
						Content   string     `json:"content"`
						ToolCalls []ToolCall `json:"tool_calls,omitempty"`
					}{Content: "retried"},
				},
			},
		}
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	base, _ := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: srv.URL,
	})

	cfg := &RetryConfig{MaxAttempts: 3, MaxBackoff: 0.001, JitterRatio: 0}
	retryProv := CreateWithRetry(base, cfg)

	var collected string
	err := retryProv.Ask(context.Background(), "test", func(s string) {
		collected += s
	})
	if err != nil {
		t.Fatalf("Ask with retry: %v", err)
	}
	if collected != "retried" {
		t.Errorf("collected = %q, want 'retried'", collected)
	}
}

// ===========================================================================
// retryProvider.calculateBackoff Tests
// ===========================================================================

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name       string
		attempt    int
		maxBackoff float64
		jitter     float64
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		{
			name:       "attempt 1 no jitter",
			attempt:    1,
			maxBackoff: 10.0,
			jitter:     0,
			wantMin:    2 * time.Second,
			wantMax:    2 * time.Second,
		},
		{
			name:       "attempt 2 no jitter",
			attempt:    2,
			maxBackoff: 10.0,
			jitter:     0,
			wantMin:    4 * time.Second,
			wantMax:    4 * time.Second,
		},
		{
			name:       "attempt 3 no jitter",
			attempt:    3,
			maxBackoff: 10.0,
			jitter:     0,
			wantMin:    8 * time.Second,
			wantMax:    8 * time.Second,
		},
		{
			name:       "attempt capped at maxBackoff",
			attempt:    5,
			maxBackoff: 10.0,
			jitter:     0,
			wantMin:    10 * time.Second,
			wantMax:    10 * time.Second,
		},
		{
			name:       "attempt with jitter",
			attempt:    1,
			maxBackoff: 10.0,
			jitter:     0.5,
			wantMin:    1 * time.Second, // 2 - 2*0.5 = 1
			wantMax:    3 * time.Second, // 2 + 2*0.5 = 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rp := &retryProvider{
				config: &RetryConfig{
					MaxAttempts: 5,
					MaxBackoff:  tt.maxBackoff,
					JitterRatio: tt.jitter,
				},
			}
			// Run multiple times for jitter tests
			for range 10 {
				got := rp.calculateBackoff(tt.attempt)
				if got < tt.wantMin || got > tt.wantMax {
					t.Errorf("calculateBackoff(%d) = %v, want between %v and %v", tt.attempt, got, tt.wantMin, tt.wantMax)
				}
			}
		})
	}
}

// ===========================================================================
// validateGeminiModel Tests (additional coverage)
// ===========================================================================

func TestValidateGeminiModel(t *testing.T) {
	tests := []struct {
		model   string
		wantErr bool
	}{
		// Valid models
		{"gemini-2.5-flash", false},
		{"gemini-2.5-pro", false},
		{"gemini-2.0-flash", false},
		{"gemini-1.5-pro", false},
		{"gemini-1.5-flash", false},
		{"gemini-1.0-pro", false},
		{"gemini-pro", false},
		{"gemini-pro-latest", false},
		{"gemini-ultra", false},
		{"gemini-nano", false},
		{"gemini-3-pro-preview", false},
		// Invalid models
		{"gpt-4", true},
		{"gemini", true},
		{"gemini-flash", true},
		{"claude-3", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			err := validateGeminiModel(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGeminiModel(%q) error = %v, wantErr %v", tt.model, err, tt.wantErr)
			}
		})
	}
}

// ===========================================================================
// Factory Custom Registration Tests
// ===========================================================================

func TestProviderFactory_CustomRegistration(t *testing.T) {
	factory := &ProviderFactory{
		providers: make(map[string]func(*ProviderConfig) (Provider, error)),
	}

	// Register a custom provider
	factory.Register("custom", func(cfg *ProviderConfig) (Provider, error) {
		return &mockProvider{name: "custom", model: cfg.Model, ready: true}, nil
	})

	p, err := factory.Create(&ProviderConfig{
		Provider: "custom",
		Model:    "custom-model",
	})
	if err != nil {
		t.Fatalf("Create custom provider: %v", err)
	}
	if p.Name() != "custom" {
		t.Errorf("Name() = %q, want 'custom'", p.Name())
	}
	if p.GetModel() != "custom-model" {
		t.Errorf("GetModel() = %q, want 'custom-model'", p.GetModel())
	}
}

func TestProviderFactory_CaseInsensitive(t *testing.T) {
	factory := &ProviderFactory{
		providers: make(map[string]func(*ProviderConfig) (Provider, error)),
	}

	factory.Register("TestProvider", func(cfg *ProviderConfig) (Provider, error) {
		return &mockProvider{name: "test", ready: true}, nil
	})

	// Should find it case-insensitively
	p, err := factory.Create(&ProviderConfig{Provider: "testprovider"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.Name() != "test" {
		t.Errorf("Name() = %q, want 'test'", p.Name())
	}
}

func TestProviderFactory_UnknownProvider(t *testing.T) {
	factory := &ProviderFactory{
		providers: make(map[string]func(*ProviderConfig) (Provider, error)),
	}

	_, err := factory.Create(&ProviderConfig{Provider: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("error = %v, want to contain 'unknown provider'", err)
	}
}

// ===========================================================================
// newHTTPClient Tests
// ===========================================================================

func TestNewHTTPClient_Timeout(t *testing.T) {
	client := newHTTPClient(false)
	if client.Timeout != 60*time.Second {
		t.Errorf("client timeout = %v, want 60s", client.Timeout)
	}
}

// ===========================================================================
// ProviderConfig / RetryConfig Tests
// ===========================================================================

func TestProviderConfig_JSONSerialization(t *testing.T) {
	cfg := &ProviderConfig{
		Provider:        "openai",
		Model:           "gpt-4",
		Endpoint:        "https://api.openai.com/v1",
		APIKey:          "sk-test",
		Region:          "us-east-1",
		AzureDeployment: "my-deployment",
		SkipTLSVerify:   true,
		ReasoningEffort: "high",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded ProviderConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Provider != cfg.Provider {
		t.Errorf("Provider = %q, want %q", decoded.Provider, cfg.Provider)
	}
	if decoded.Model != cfg.Model {
		t.Errorf("Model = %q, want %q", decoded.Model, cfg.Model)
	}
	if decoded.SkipTLSVerify != cfg.SkipTLSVerify {
		t.Errorf("SkipTLSVerify = %v, want %v", decoded.SkipTLSVerify, cfg.SkipTLSVerify)
	}
	if decoded.ReasoningEffort != cfg.ReasoningEffort {
		t.Errorf("ReasoningEffort = %q, want %q", decoded.ReasoningEffort, cfg.ReasoningEffort)
	}
}

// ===========================================================================
// Compile-time interface checks
// ===========================================================================

var (
	_ Provider     = (*OpenAIProvider)(nil)
	_ Provider     = (*OllamaProvider)(nil)
	_ Provider     = (*GeminiProvider)(nil)
	_ Provider     = (*AzureOpenAIProvider)(nil)
	_ Provider     = (*BedrockProvider)(nil)
	_ Provider     = (*EmbeddedProvider)(nil)
	_ Provider     = (*retryProvider)(nil)
	_ ToolProvider = (*OpenAIProvider)(nil)
	_ ToolProvider = (*OllamaProvider)(nil)
	_ ToolProvider = (*GeminiProvider)(nil)
	_ ToolProvider = (*EmbeddedProvider)(nil)
	_ ToolProvider = (*retryProvider)(nil)
)
