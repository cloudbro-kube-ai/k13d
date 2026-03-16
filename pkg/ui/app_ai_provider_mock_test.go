package ui

import (
	"strings"
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
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
