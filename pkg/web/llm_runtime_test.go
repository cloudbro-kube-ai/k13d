package web

import (
	"path/filepath"
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func TestCreateUsableAIClient(t *testing.T) {
	t.Run("missing API key is not ready", func(t *testing.T) {
		for _, key := range []string{
			"K13D_LLM_API_KEY",
			"OPENAI_API_KEY",
			"UPSTAGE_API_KEY",
			"ANTHROPIC_API_KEY",
			"GOOGLE_API_KEY",
			"AZURE_OPENAI_API_KEY",
			"AZURE_OPENAI_ENDPOINT",
			"OLLAMA_HOST",
			"AWS_ACCESS_KEY_ID",
			"AWS_SECRET_ACCESS_KEY",
			"AWS_SESSION_TOKEN",
			"AWS_REGION",
		} {
			t.Setenv(key, "")
		}

		client, ready, err := createUsableAIClient(&config.LLMConfig{
			Provider: "upstage",
			Model:    "solar-pro2",
			Endpoint: "https://api.upstage.ai/v1",
		})
		if err != nil {
			t.Fatalf("createUsableAIClient() error = %v", err)
		}
		if client == nil {
			t.Fatal("expected client to be created even when not ready")
		}
		if ready {
			t.Fatal("expected ready=false when API key is missing")
		}
	})

	t.Run("environment fallback can make provider ready", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "env-openai-key")

		client, ready, err := createUsableAIClient(&config.LLMConfig{
			Provider: "openai",
			Model:    "gpt-4o",
		})
		if err != nil {
			t.Fatalf("createUsableAIClient() error = %v", err)
		}
		if client == nil {
			t.Fatal("expected client to be created")
		}
		if !ready {
			t.Fatal("expected ready=true when env fallback provides an API key")
		}
	})
}

func TestProcessE2EEnvStripsProviderSecrets(t *testing.T) {
	t.Setenv("K13D_LLM_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "secret")
	t.Setenv("UPSTAGE_API_KEY", "secret")
	t.Setenv("GOOGLE_API_KEY", "secret")
	t.Setenv("AWS_ACCESS_KEY_ID", "secret")
	t.Setenv("SAFE_VAR", "keep-me")

	env := processE2EEnv(filepath.Join(t.TempDir(), "config.yaml"))

	joined := make(map[string]string, len(env))
	for _, entry := range env {
		key, value, ok := splitEnvEntry(entry)
		if !ok {
			continue
		}
		joined[key] = value
	}

	if _, ok := joined["K13D_LLM_PROVIDER"]; ok {
		t.Fatal("expected K13D_* variables to be stripped from process E2E env")
	}
	for _, key := range []string{"OPENAI_API_KEY", "UPSTAGE_API_KEY", "GOOGLE_API_KEY", "AWS_ACCESS_KEY_ID"} {
		if _, ok := joined[key]; ok {
			t.Fatalf("expected %s to be stripped from process E2E env", key)
		}
	}
	if joined["SAFE_VAR"] != "keep-me" {
		t.Fatalf("SAFE_VAR = %q, want keep-me", joined["SAFE_VAR"])
	}
}

func splitEnvEntry(entry string) (string, string, bool) {
	for i := 0; i < len(entry); i++ {
		if entry[i] == '=' {
			return entry[:i], entry[i+1:], true
		}
	}
	return "", "", false
}
