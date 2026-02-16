package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OllamaProvider implements the Provider and ToolProvider interfaces for Ollama (local LLM)
type OllamaProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	endpoint   string
}

type ollamaChatRequest struct {
	Model    string           `json:"model"`
	Messages []ChatMessage    `json:"messages"`
	Stream   bool             `json:"stream"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
}

type ollamaChatResponse struct {
	Message struct {
		Content   string     `json:"content"`
		ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	} `json:"message"`
	Done bool `json:"done"`
}

type ollamaModelsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

const ollamaSystemPrompt = "You are a helpful Kubernetes assistant. Help users manage Kubernetes clusters using natural language. When users ask to create resources, generate the appropriate kubectl commands."

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(cfg *ProviderConfig) (Provider, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	endpoint = strings.TrimSuffix(endpoint, "/")

	model := cfg.Model
	if model == "" {
		model = "llama3.2" // Default Ollama model
	}

	return &OllamaProvider{
		config: &ProviderConfig{
			Provider: cfg.Provider,
			Model:    model,
			Endpoint: endpoint,
			APIKey:   cfg.APIKey,
		},
		httpClient: newHTTPClient(cfg.SkipTLSVerify),
		endpoint:   endpoint,
	}, nil
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) GetModel() string {
	return p.config.Model
}

func (p *OllamaProvider) IsReady() bool {
	return p.config != nil && p.endpoint != ""
}

func (p *OllamaProvider) Ask(ctx context.Context, prompt string, callback func(string)) error {
	endpoint := p.endpoint + "/api/chat"

	reqBody := ollamaChatRequest{
		Model: p.config.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: ollamaSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading response: %w", err)
		}

		var chatResp ollamaChatResponse
		if err := json.Unmarshal(line, &chatResp); err != nil {
			continue
		}

		if chatResp.Message.Content != "" {
			callback(chatResp.Message.Content)
		}

		if chatResp.Done {
			break
		}
	}

	return nil
}

func (p *OllamaProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	endpoint := p.endpoint + "/api/chat"

	reqBody := ollamaChatRequest{
		Model: p.config.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: ollamaSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if chatResp.Message.Content == "" {
		return "", fmt.Errorf("empty response from Ollama API")
	}

	return chatResp.Message.Content, nil
}

func (p *OllamaProvider) ListModels(ctx context.Context) ([]string, error) {
	endpoint := p.endpoint + "/api/tags"

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models: status %d", resp.StatusCode)
	}

	var modelsResp ollamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(modelsResp.Models))
	for i, m := range modelsResp.Models {
		models[i] = m.Name
	}
	return models, nil
}

// AskWithTools implements ToolProvider for Ollama using the /api/chat endpoint with tools.
// Ollama supports OpenAI-compatible tool calling since v0.3.0.
func (p *OllamaProvider) AskWithTools(ctx context.Context, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) error {
	endpoint := p.endpoint + "/api/chat"

	messages := []ChatMessage{
		{Role: "system", Content: `You are a Kubernetes expert assistant with DIRECT ACCESS to kubectl and bash tools.
ALWAYS USE TOOLS to execute commands - NEVER just suggest commands.
When asked about Kubernetes resources, IMMEDIATELY use the kubectl tool.`},
		{Role: "user", Content: prompt},
	}

	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		reqBody := ollamaChatRequest{
			Model:    p.config.Model,
			Messages: messages,
			Stream:   false,
			Tools:    tools,
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}

		var chatResp ollamaChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		content := chatResp.Message.Content
		toolCalls := chatResp.Message.ToolCalls

		// No tool calls - return the final response
		if len(toolCalls) == 0 {
			if callback != nil && content != "" {
				callback(content)
			}
			return nil
		}

		// Add assistant message with tool calls to history
		messages = append(messages, ChatMessage{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
		})

		// Execute each tool call and add results
		for _, tc := range toolCalls {
			if callback != nil {
				callback(fmt.Sprintf("\n\n🔧 Executing: %s\n", tc.Function.Name))
			}

			result := toolCallback(tc)

			messages = append(messages, ChatMessage{
				Role:       "tool",
				Content:    result.Content,
				ToolCallID: tc.ID,
			})

			if callback != nil {
				if result.IsError {
					callback(fmt.Sprintf("❌ Error: %s\n", result.Content))
				} else {
					output := result.Content
					if len(output) > 1000 {
						output = output[:1000] + "\n... (truncated)"
					}
					callback(fmt.Sprintf("```\n%s\n```\n", output))
				}
			}
		}
	}

	return fmt.Errorf("exceeded maximum tool call iterations")
}
