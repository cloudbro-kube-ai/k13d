//go:build integration

// Integration tests for LLM providers
// Run with: go test -tags=integration ./tests/integration/...
//
// Prerequisites:
// - docker compose -f docker-compose.test.yaml up -d
// - Wait for services to be healthy
//
// Environment variables for testing against real providers:
// - OPENAI_API_KEY: Test against real OpenAI API
// - OLLAMA_HOST: Test against real Ollama (default: http://localhost:11434)

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
)

func TestMockOpenAI_Connection(t *testing.T) {
	// Test against mock OpenAI server
	endpoint := os.Getenv("MOCK_OPENAI_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8081"
	}

	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: endpoint,
		APIKey:   "test-api-key",
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping: cannot create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := client.TestConnection(ctx)
	if !status.Connected {
		t.Skipf("Skipping: mock server not available: %v", status.Error)
	}

	t.Logf("Connected to mock OpenAI: provider=%s, model=%s, latency=%dms",
		status.Provider, status.Model, status.ResponseTime)
}

func TestMockOpenAI_NonStreaming(t *testing.T) {
	endpoint := os.Getenv("MOCK_OPENAI_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8081"
	}

	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: endpoint,
		APIKey:   "test-api-key",
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping: cannot create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := client.AskNonStreaming(ctx, "What pods are running?")
	if err != nil {
		t.Skipf("Skipping: mock server not available: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}
	t.Logf("Response: %s", response)
}

func TestMockOpenAI_Streaming(t *testing.T) {
	endpoint := os.Getenv("MOCK_OPENAI_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8081"
	}

	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: endpoint,
		APIKey:   "test-api-key",
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping: cannot create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var chunks []string
	err = client.Ask(ctx, "Explain Kubernetes pods", func(chunk string) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Skipf("Skipping: mock server not available: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected streaming chunks")
	}
	t.Logf("Received %d chunks", len(chunks))
}

func TestMockOpenAI_ToolCalling(t *testing.T) {
	endpoint := os.Getenv("MOCK_OPENAI_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8081"
	}

	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4",
		Endpoint: endpoint,
		APIKey:   "test-api-key",
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping: cannot create client: %v", err)
	}

	if !client.SupportsTools() {
		t.Log("Provider does not support tools, testing tool support check")
		return
	}

	t.Log("Provider supports tool calling")
}

func TestOllama_Connection(t *testing.T) {
	// Test against Ollama (either local or from docker compose)
	endpoint := os.Getenv("OLLAMA_HOST")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	cfg := &config.LLMConfig{
		Provider: "ollama",
		Model:    "llama3.2:1b", // Small model for testing
		Endpoint: endpoint,
		APIKey:   "ollama", // Ollama doesn't require API key
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping: cannot create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status := client.TestConnection(ctx)
	if !status.Connected {
		t.Skipf("Skipping: Ollama not available: %v", status.Error)
	}

	t.Logf("Connected to Ollama: provider=%s, model=%s, latency=%dms",
		status.Provider, status.Model, status.ResponseTime)
}

func TestOllama_Qwen25_Connection(t *testing.T) {
	// Test against Ollama with Qwen2.5:3b (default model for low-spec environments)
	endpoint := os.Getenv("OLLAMA_HOST")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	cfg := &config.LLMConfig{
		Provider: "ollama",
		Model:    config.DefaultOllamaModel, // qwen2.5:3b
		Endpoint: endpoint,
		APIKey:   "ollama",
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping: cannot create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	status := client.TestConnection(ctx)
	if !status.Connected {
		t.Skipf("Skipping: Ollama with %s not available: %v", config.DefaultOllamaModel, status.Error)
	}

	t.Logf("Connected to Ollama with %s: provider=%s, model=%s, latency=%dms",
		config.DefaultOllamaModel, status.Provider, status.Model, status.ResponseTime)
}

func TestOllama_Qwen25_KoreanResponse(t *testing.T) {
	// Test Korean language support with Qwen2.5:3b
	endpoint := os.Getenv("OLLAMA_HOST")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	cfg := &config.LLMConfig{
		Provider: "ollama",
		Model:    config.DefaultOllamaModel, // qwen2.5:3b
		Endpoint: endpoint,
		APIKey:   "ollama",
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping: cannot create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Test Korean prompt
	response, err := client.AskNonStreaming(ctx, "쿠버네티스 파드란 무엇인가요? 한국어로 간단히 설명해주세요.")
	if err != nil {
		t.Skipf("Skipping: Ollama with %s not available: %v", config.DefaultOllamaModel, err)
	}

	if response == "" {
		t.Error("Expected non-empty Korean response")
	}

	// Check if response contains Korean characters
	hasKorean := false
	for _, r := range response {
		if r >= 0xAC00 && r <= 0xD7A3 { // Korean Hangul range
			hasKorean = true
			break
		}
	}

	if !hasKorean {
		t.Logf("Warning: Response may not contain Korean characters")
	}

	t.Logf("Korean response (truncated): %s...", truncateString(response, 200))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func TestOllama_ListModels(t *testing.T) {
	endpoint := os.Getenv("OLLAMA_HOST")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	cfg := &config.LLMConfig{
		Provider: "ollama",
		Model:    "llama3.2:1b",
		Endpoint: endpoint,
		APIKey:   "ollama",
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping: cannot create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := client.ListModels(ctx)
	if err != nil {
		t.Skipf("Skipping: Ollama not available: %v", err)
	}

	t.Logf("Available models: %v", models)
}

func TestRealOpenAI_Connection(t *testing.T) {
	// Test against real OpenAI (if API key is provided)
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: OPENAI_API_KEY not set")
	}

	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4o-mini", // Use cheaper model for testing
		APIKey:   apiKey,
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status := client.TestConnection(ctx)
	if !status.Connected {
		t.Fatalf("Failed to connect: %v", status.Error)
	}

	t.Logf("Connected to OpenAI: provider=%s, model=%s, latency=%dms",
		status.Provider, status.Model, status.ResponseTime)
}

func TestRealOpenAI_ToolCalling(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: OPENAI_API_KEY not set")
	}

	cfg := &config.LLMConfig{
		Provider: "openai",
		Model:    "gpt-4o-mini",
		APIKey:   apiKey,
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if !client.SupportsTools() {
		t.Error("OpenAI should support tool calling")
	}

	t.Log("OpenAI tool calling support confirmed")
}

func TestAnthropic_Connection(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: ANTHROPIC_API_KEY not set")
	}

	cfg := &config.LLMConfig{
		Provider: "anthropic",
		Model:    "claude-3-haiku-20240307", // Use cheaper model for testing
		APIKey:   apiKey,
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status := client.TestConnection(ctx)
	if !status.Connected {
		t.Fatalf("Failed to connect: %v", status.Error)
	}

	t.Logf("Connected to Anthropic: provider=%s, model=%s, latency=%dms",
		status.Provider, status.Model, status.ResponseTime)
}

func TestGemini_Connection(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping: GEMINI_API_KEY not set")
	}

	cfg := &config.LLMConfig{
		Provider: "gemini",
		Model:    "gemini-1.5-flash",
		APIKey:   apiKey,
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status := client.TestConnection(ctx)
	if !status.Connected {
		t.Fatalf("Failed to connect: %v", status.Error)
	}

	t.Logf("Connected to Gemini: provider=%s, model=%s, latency=%dms",
		status.Provider, status.Model, status.ResponseTime)
}
