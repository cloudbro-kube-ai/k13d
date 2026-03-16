package llmhttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

// MockProviderServer serves small provider-specific HTTP fixtures that mirror
// the documented request/response envelopes used by the LLM providers.
type MockProviderServer struct {
	provider string
	model    string
	apiKey   string
	reply    string
	server   *httptest.Server
	requests atomic.Int32
}

func NewMockProviderServer(provider, reply string) *MockProviderServer {
	s := &MockProviderServer{
		provider: provider,
		model:    defaultModel(provider),
		apiKey:   defaultAPIKey(provider),
		reply:    reply,
	}
	s.server = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

func (s *MockProviderServer) Close() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *MockProviderServer) URL() string {
	if s.server == nil {
		return ""
	}
	return s.server.URL
}

func (s *MockProviderServer) RequestCount() int32 {
	return s.requests.Load()
}

func (s *MockProviderServer) LLMConfig() *config.LLMConfig {
	return &config.LLMConfig{
		Provider: s.provider,
		Model:    s.model,
		Endpoint: s.URL(),
		APIKey:   s.apiKey,
	}
}

func (s *MockProviderServer) handle(w http.ResponseWriter, r *http.Request) {
	s.requests.Add(1)

	switch s.provider {
	case "openai":
		s.handleOpenAI(w, r)
	case "ollama":
		s.handleOllama(w, r)
	case "gemini":
		s.handleGemini(w, r)
	case "anthropic":
		s.handleAnthropic(w, r)
	default:
		http.Error(w, "unsupported mock provider", http.StatusBadRequest)
	}
}

func (s *MockProviderServer) handleOpenAI(w http.ResponseWriter, r *http.Request) {
	if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
		http.NotFound(w, r)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid openai request", http.StatusBadRequest)
		return
	}
	if _, ok := body["model"].(string); !ok {
		http.Error(w, "missing model", http.StatusBadRequest)
		return
	}
	if _, ok := body["messages"].([]interface{}); !ok {
		http.Error(w, "missing messages", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id": "chatcmpl-mock",
		"choices": []map[string]interface{}{
			{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": s.reply,
				},
				"finish_reason": "stop",
			},
		},
	})
}

func (s *MockProviderServer) handleOllama(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/chat" {
		http.NotFound(w, r)
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid ollama request", http.StatusBadRequest)
		return
	}
	if _, ok := body["model"].(string); !ok {
		http.Error(w, "missing model", http.StatusBadRequest)
		return
	}
	if _, ok := body["messages"].([]interface{}); !ok {
		http.Error(w, "missing messages", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": map[string]interface{}{
			"role":    "assistant",
			"content": s.reply,
		},
		"done": true,
	})
}

func (s *MockProviderServer) handleGemini(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.URL.Path, ":generateContent") && !strings.Contains(r.URL.Path, ":streamGenerateContent") {
		http.NotFound(w, r)
		return
	}
	if r.Header.Get("x-goog-api-key") == "" {
		http.Error(w, "missing gemini api key", http.StatusUnauthorized)
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid gemini request", http.StatusBadRequest)
		return
	}
	if _, ok := body["contents"].([]interface{}); !ok {
		http.Error(w, "missing contents", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"candidates": []map[string]interface{}{
			{
				"content": map[string]interface{}{
					"parts": []map[string]interface{}{
						{"text": s.reply},
					},
				},
			},
		},
	})
}

func (s *MockProviderServer) handleAnthropic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/messages" {
		http.NotFound(w, r)
		return
	}
	if r.Header.Get("x-api-key") == "" {
		http.Error(w, "missing anthropic api key", http.StatusUnauthorized)
		return
	}
	if r.Header.Get("anthropic-version") == "" {
		http.Error(w, "missing anthropic version", http.StatusBadRequest)
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid anthropic request", http.StatusBadRequest)
		return
	}
	if _, ok := body["messages"].([]interface{}); !ok {
		http.Error(w, "missing messages", http.StatusBadRequest)
		return
	}
	if _, ok := body["max_tokens"].(float64); !ok {
		http.Error(w, "missing max_tokens", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          "msg_mock",
		"type":        "message",
		"role":        "assistant",
		"stop_reason": "end_turn",
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": s.reply,
			},
		},
	})
}

func defaultAPIKey(provider string) string {
	switch provider {
	case "ollama":
		return "ollama"
	default:
		return "test-key"
	}
}

func defaultModel(provider string) string {
	switch provider {
	case "openai":
		return "gpt-4o-mini"
	case "ollama":
		return "gpt-oss:20b"
	case "gemini":
		return "gemini-2.5-flash"
	case "anthropic":
		return "claude-sonnet-4-20250514"
	default:
		return "test-model"
	}
}
