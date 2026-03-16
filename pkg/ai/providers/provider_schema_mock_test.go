package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type capturedProviderRequest struct {
	Server  *httptest.Server
	Body    []byte
	Headers http.Header
	Path    string
}

func newOllamaCaptureServer(t *testing.T, content string) *capturedProviderRequest {
	t.Helper()

	rc := &capturedProviderRequest{}
	rc.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rc.Body = body
		rc.Headers = r.Header.Clone()
		rc.Path = r.URL.Path

		resp := ollamaChatResponse{Done: true}
		resp.Message.Content = content
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	return rc
}

func newGeminiCaptureServer(t *testing.T, content string) *capturedProviderRequest {
	t.Helper()

	rc := &capturedProviderRequest{}
	rc.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rc.Body = body
		rc.Headers = r.Header.Clone()
		rc.Path = r.URL.Path

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
	return rc
}

func newAnthropicNonStreamServer(t *testing.T, content string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("x-api-key") == "" {
			t.Error("expected x-api-key header to be set")
		}
		if got := r.Header.Get("anthropic-version"); got != anthropicAPIVersion {
			t.Errorf("anthropic-version = %q, want %q", got, anthropicAPIVersion)
		}

		resp := anthropicResponse{
			ID:         "msg_test",
			Type:       "message",
			Role:       "assistant",
			StopReason: "end_turn",
			Content: []anthropicContentBlock{
				{Type: "text", Text: content},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func newAnthropicStreamServer(t *testing.T, tokens []string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("x-api-key") == "" {
			t.Error("expected x-api-key header to be set")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for _, token := range tokens {
			event := anthropicStreamEvent{
				Type: "content_block_delta",
				Delta: &anthropicStreamDelta{
					Type: "text_delta",
					Text: token,
				},
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}

		stop := anthropicStreamEvent{Type: "message_stop"}
		data, _ := json.Marshal(stop)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}))
}

func newAnthropicCaptureServer(t *testing.T, content string) *capturedProviderRequest {
	t.Helper()

	rc := &capturedProviderRequest{}
	rc.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rc.Body = body
		rc.Headers = r.Header.Clone()
		rc.Path = r.URL.Path

		resp := anthropicResponse{
			ID:         "msg_capture",
			Type:       "message",
			Role:       "assistant",
			StopReason: "end_turn",
			Content: []anthropicContentBlock{
				{Type: "text", Text: content},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	return rc
}

func TestOllamaProvider_RequestBuilding(t *testing.T) {
	rc := newOllamaCaptureServer(t, "mock ollama answer")
	defer rc.Server.Close()

	p, err := NewOllamaProvider(&ProviderConfig{
		Provider: "ollama",
		Model:    "gpt-oss:20b",
		Endpoint: rc.Server.URL,
	})
	if err != nil {
		t.Fatalf("NewOllamaProvider: %v", err)
	}

	resp, err := p.AskNonStreaming(context.Background(), "summarize pod restarts")
	if err != nil {
		t.Fatalf("AskNonStreaming: %v", err)
	}
	if resp != "mock ollama answer" {
		t.Fatalf("response = %q, want %q", resp, "mock ollama answer")
	}

	var reqBody ollamaChatRequest
	if err := json.Unmarshal(rc.Body, &reqBody); err != nil {
		t.Fatalf("failed to decode Ollama request: %v", err)
	}
	if rc.Path != "/api/chat" {
		t.Fatalf("request path = %q, want /api/chat", rc.Path)
	}
	if reqBody.Model != "gpt-oss:20b" {
		t.Fatalf("model = %q, want gpt-oss:20b", reqBody.Model)
	}
	if reqBody.Stream {
		t.Fatal("AskNonStreaming should send stream=false to Ollama")
	}
	if len(reqBody.Messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(reqBody.Messages))
	}
	if reqBody.Messages[0].Role != "system" || reqBody.Messages[1].Role != "user" {
		t.Fatalf("unexpected roles in Ollama request: %+v", reqBody.Messages)
	}
	if reqBody.Messages[1].Content != "summarize pod restarts" {
		t.Fatalf("user content = %q", reqBody.Messages[1].Content)
	}
	if got := rc.Headers.Get("Authorization"); got != "" {
		t.Fatalf("Ollama request should not send Authorization, got %q", got)
	}
}

func TestGeminiProvider_RequestBuilding(t *testing.T) {
	rc := newGeminiCaptureServer(t, "mock gemini answer")
	defer rc.Server.Close()

	p, err := NewGeminiProvider(&ProviderConfig{
		Provider: "gemini",
		Model:    "gemini-2.5-flash",
		APIKey:   "gemini-test-key",
		Endpoint: rc.Server.URL,
	})
	if err != nil {
		t.Fatalf("NewGeminiProvider: %v", err)
	}

	resp, err := p.AskNonStreaming(context.Background(), "show deployment drift")
	if err != nil {
		t.Fatalf("AskNonStreaming: %v", err)
	}
	if resp != "mock gemini answer" {
		t.Fatalf("response = %q, want %q", resp, "mock gemini answer")
	}

	var reqBody geminiRequest
	if err := json.Unmarshal(rc.Body, &reqBody); err != nil {
		t.Fatalf("failed to decode Gemini request: %v", err)
	}
	if !strings.Contains(rc.Path, "models/gemini-2.5-flash:generateContent") {
		t.Fatalf("request path = %q, want generateContent model endpoint", rc.Path)
	}
	if got := rc.Headers.Get("x-goog-api-key"); got != "gemini-test-key" {
		t.Fatalf("x-goog-api-key = %q, want gemini-test-key", got)
	}
	if reqBody.SystemInstruction == nil || len(reqBody.SystemInstruction.Parts) == 0 {
		t.Fatal("expected Gemini systemInstruction to be set")
	}
	if len(reqBody.Contents) != 1 || reqBody.Contents[0].Role != "user" {
		t.Fatalf("unexpected Gemini contents: %+v", reqBody.Contents)
	}
	if got := reqBody.Contents[0].Parts[0].Text; got != "show deployment drift" {
		t.Fatalf("user text = %q", got)
	}
}

func TestAnthropicProvider_AskStreaming(t *testing.T) {
	srv := newAnthropicStreamServer(t, []string{"Claude", " via", " SSE"})
	defer srv.Close()

	p, err := NewAnthropicProvider(&ProviderConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-20250514",
		APIKey:   "anthropic-test-key",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewAnthropicProvider: %v", err)
	}

	var collected []string
	err = p.Ask(context.Background(), "explain pending pods", func(s string) {
		collected = append(collected, s)
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}

	if got := strings.Join(collected, ""); got != "Claude via SSE" {
		t.Fatalf("streamed response = %q, want %q", got, "Claude via SSE")
	}
}

func TestAnthropicProvider_AskNonStreaming(t *testing.T) {
	srv := newAnthropicNonStreamServer(t, "Claude mock answer")
	defer srv.Close()

	p, err := NewAnthropicProvider(&ProviderConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-20250514",
		APIKey:   "anthropic-test-key",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewAnthropicProvider: %v", err)
	}

	resp, err := p.AskNonStreaming(context.Background(), "why are pods evicted")
	if err != nil {
		t.Fatalf("AskNonStreaming: %v", err)
	}
	if resp != "Claude mock answer" {
		t.Fatalf("response = %q, want %q", resp, "Claude mock answer")
	}
}

func TestAnthropicProvider_RequestBuilding(t *testing.T) {
	rc := newAnthropicCaptureServer(t, "Claude capture")
	defer rc.Server.Close()

	p, err := NewAnthropicProvider(&ProviderConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-20250514",
		APIKey:   "anthropic-test-key",
		Endpoint: rc.Server.URL,
	})
	if err != nil {
		t.Fatalf("NewAnthropicProvider: %v", err)
	}

	resp, err := p.AskNonStreaming(context.Background(), "summarize cluster events")
	if err != nil {
		t.Fatalf("AskNonStreaming: %v", err)
	}
	if resp != "Claude capture" {
		t.Fatalf("response = %q, want %q", resp, "Claude capture")
	}

	var reqBody anthropicRequest
	if err := json.Unmarshal(rc.Body, &reqBody); err != nil {
		t.Fatalf("failed to decode Anthropic request: %v", err)
	}
	if rc.Path != "/v1/messages" {
		t.Fatalf("request path = %q, want /v1/messages", rc.Path)
	}
	if got := rc.Headers.Get("x-api-key"); got != "anthropic-test-key" {
		t.Fatalf("x-api-key = %q, want anthropic-test-key", got)
	}
	if got := rc.Headers.Get("anthropic-version"); got != anthropicAPIVersion {
		t.Fatalf("anthropic-version = %q, want %q", got, anthropicAPIVersion)
	}
	if reqBody.Model != "claude-sonnet-4-20250514" {
		t.Fatalf("model = %q", reqBody.Model)
	}
	if reqBody.MaxTokens != anthropicDefaultMaxTokens {
		t.Fatalf("max_tokens = %d, want %d", reqBody.MaxTokens, anthropicDefaultMaxTokens)
	}
	if reqBody.System == "" {
		t.Fatal("expected Anthropic system prompt to be set")
	}
	if len(reqBody.Messages) != 1 || reqBody.Messages[0].Role != "user" {
		t.Fatalf("unexpected Anthropic messages: %+v", reqBody.Messages)
	}
	if content, ok := reqBody.Messages[0].Content.(string); !ok || content != "summarize cluster events" {
		t.Fatalf("unexpected Anthropic message content: %#v", reqBody.Messages[0].Content)
	}
}

func TestAnthropicProvider_AskWithTools_WithToolUse(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		var reqBody anthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			resp := anthropicResponse{
				ID:         "msg_tool_1",
				Type:       "message",
				Role:       "assistant",
				StopReason: "tool_use",
				Content: []anthropicContentBlock{
					{Type: "text", Text: "Inspecting the cluster."},
					{
						Type:  "tool_use",
						ID:    "toolu_123",
						Name:  "kubectl",
						Input: json.RawMessage(`{"command":"kubectl get pods -n default"}`),
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if len(reqBody.Messages) < 3 {
			t.Fatalf("expected follow-up Anthropic request to include tool results, got %d messages", len(reqBody.Messages))
		}

		resp := anthropicResponse{
			ID:         "msg_tool_2",
			Type:       "message",
			Role:       "assistant",
			StopReason: "end_turn",
			Content: []anthropicContentBlock{
				{Type: "text", Text: "Found 2 running pods in default."},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	p, err := NewAnthropicProvider(&ProviderConfig{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-20250514",
		APIKey:   "anthropic-test-key",
		Endpoint: srv.URL,
	})
	if err != nil {
		t.Fatalf("NewAnthropicProvider: %v", err)
	}

	toolCalled := false
	var callbackContent strings.Builder
	err = p.(ToolProvider).AskWithTools(
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
			callbackContent.WriteString(s)
		},
		func(call ToolCall) ToolResult {
			toolCalled = true
			if call.Function.Name != "kubectl" {
				t.Fatalf("tool name = %q, want kubectl", call.Function.Name)
			}
			return ToolResult{
				ToolCallID: call.ID,
				Content:    "NAME READY STATUS\npod-a 1/1 Running\npod-b 1/1 Running",
			}
		},
	)
	if err != nil {
		t.Fatalf("AskWithTools: %v", err)
	}
	if !toolCalled {
		t.Fatal("expected Anthropic tool callback to run")
	}
	if !strings.Contains(callbackContent.String(), "Found 2 running pods in default.") {
		t.Fatalf("expected final Anthropic answer in callback, got %q", callbackContent.String())
	}
	if callCount != 2 {
		t.Fatalf("expected 2 Anthropic requests, got %d", callCount)
	}
}

var (
	_ Provider     = (*AnthropicProvider)(nil)
	_ ToolProvider = (*AnthropicProvider)(nil)
)
