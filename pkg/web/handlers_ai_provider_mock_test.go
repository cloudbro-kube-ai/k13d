package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/tests/mocks/llmhttp"
)

func TestHandleAgenticChat_MockProvidersStreamIntoSSE(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		reply    string
	}{
		{name: "openai", provider: "openai", reply: "OpenAI mock says the pods are healthy."},
		{name: "ollama", provider: "ollama", reply: "Ollama mock found 2 deployments."},
		{name: "gemini", provider: "gemini", reply: "Gemini mock detected no rollout drift."},
		{name: "claude", provider: "anthropic", reply: "Claude mock explains the restart spike."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := llmhttp.NewMockProviderServer(tc.provider, tc.reply)
			defer mock.Close()

			client, err := ai.NewClient(mock.LLMConfig())
			if err != nil {
				t.Fatalf("failed to create AI client for %s: %v", tc.provider, err)
			}

			s := setupAITestServer(t, false)
			s.aiClient = client
			s.cfg.LLM = *mock.LLMConfig()

			body, _ := json.Marshal(ChatRequest{Message: "Explain the current cluster state"})
			req := httptest.NewRequest(http.MethodPost, "/api/chat/agentic", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			s.handleAgenticChat(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d\n%s", w.Code, w.Body.String())
			}

			respBody := w.Body.String()
			if !strings.Contains(respBody, tc.reply) {
				t.Fatalf("expected SSE body to contain %q, got:\n%s", tc.reply, respBody)
			}
			if !strings.Contains(respBody, "data: [DONE]") {
				t.Fatalf("expected SSE body to end with [DONE], got:\n%s", respBody)
			}
			if mock.RequestCount() == 0 {
				t.Fatal("expected mock provider server to receive at least one request")
			}
		})
	}
}
