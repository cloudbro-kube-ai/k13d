package ui

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/tests/mocks/llmhttp"
)

func TestAskAI_MockProvidersRenderIntoPanel(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		reply    string
	}{
		{name: "openai", provider: "openai", reply: "OpenAI mock answer for the selected workload."},
		{name: "ollama", provider: "ollama", reply: "Ollama mock sees a stable rollout."},
		{name: "gemini", provider: "gemini", reply: "Gemini mock reports healthy services."},
		{name: "claude", provider: "anthropic", reply: "Claude mock summarizes the incident timeline."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := llmhttp.NewMockProviderServer(tc.provider, tc.reply)
			defer mock.Close()

			client, err := ai.NewClient(mock.LLMConfig())
			if err != nil {
				t.Fatalf("failed to create AI client for %s: %v", tc.provider, err)
			}

			app := NewTestApp(TestAppConfig{
				SkipBackgroundLoading: true,
				SkipBriefing:          true,
			})
			app.aiClient = client
			app.refresh()
			app.resetAIConversation()

			app.askAI("Explain whether this resource looks healthy")

			text := app.aiPanel.GetText(false)
			if !strings.Contains(text, tc.reply) {
				t.Fatalf("expected AI panel to contain %q, got:\n%s", tc.reply, text)
			}
			if !strings.Contains(text, "Assistant") {
				t.Fatalf("expected AI transcript header in panel, got:\n%s", text)
			}
			if mock.RequestCount() == 0 {
				t.Fatal("expected mock provider server to receive at least one request")
			}
			if status := app.aiStatusBar.GetText(false); !strings.Contains(status, "Ready") {
				t.Fatalf("expected AI status bar to reset to ready text, got %q", status)
			}
		})
	}
}

func TestAskAI_FollowUpIncludesConversationHistory(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			http.NotFound(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			http.Error(w, "failed to read request", http.StatusInternalServerError)
			return
		}

		mu.Lock()
		requests = append(requests, string(body))
		mu.Unlock()

		reply := "First answer"
		if strings.Contains(string(body), "What did I ask before?") {
			reply = "Your previous question was about failing pods."
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "chatcmpl-mock",
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": reply,
					},
					"finish_reason": "stop",
				},
			},
		}); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	client, err := ai.NewClient(&config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		Endpoint: server.URL,
		APIKey:   "test-key",
	})
	if err != nil {
		t.Fatalf("failed to create AI client: %v", err)
	}

	app := NewTestApp(TestAppConfig{
		SkipBackgroundLoading: true,
		SkipBriefing:          true,
	})
	app.aiClient = client
	app.refresh()
	app.resetAIConversation()

	app.askAI("Show me failing pods")
	app.askAI("What did I ask before?")

	mu.Lock()
	defer mu.Unlock()

	if len(requests) < 2 {
		t.Fatalf("expected at least 2 AI requests, got %d", len(requests))
	}

	secondRequest := requests[len(requests)-1]
	for _, want := range []string{
		"=== CONVERSATION HISTORY ===",
		"Show me failing pods",
		"First answer",
		"What did I ask before?",
	} {
		if !strings.Contains(secondRequest, want) {
			t.Fatalf("expected second AI request to contain %q, got:\n%s", want, secondRequest)
		}
	}
}
