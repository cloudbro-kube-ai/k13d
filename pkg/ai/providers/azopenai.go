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

// AzureOpenAIProvider implements the Provider interface for Azure OpenAI
type AzureOpenAIProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	endpoint   string
	deployment string
}

// NewAzureOpenAIProvider creates a new Azure OpenAI provider
func NewAzureOpenAIProvider(cfg *ProviderConfig) (Provider, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("Azure OpenAI requires endpoint (e.g., https://YOUR_RESOURCE.openai.azure.com)")
	}

	deployment := cfg.AzureDeployment
	if deployment == "" {
		deployment = cfg.Model // Use model as deployment name if not specified
	}

	return &AzureOpenAIProvider{
		config:     cfg,
		httpClient: newHTTPClient(cfg.SkipTLSVerify),
		endpoint:   strings.TrimSuffix(cfg.Endpoint, "/"),
		deployment: deployment,
	}, nil
}

func (p *AzureOpenAIProvider) Name() string {
	return "azopenai"
}

func (p *AzureOpenAIProvider) GetModel() string {
	return p.deployment
}

func (p *AzureOpenAIProvider) IsReady() bool {
	return p.config != nil && p.config.APIKey != "" && p.endpoint != ""
}

func (p *AzureOpenAIProvider) Ask(ctx context.Context, prompt string, callback func(string)) error {
	// Azure OpenAI uses deployment name in URL
	endpoint := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-02-15-preview",
		p.endpoint, p.deployment)

	reqBody := openAIChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful Kubernetes assistant. Help users manage Kubernetes clusters using natural language. When users ask to create resources, generate the appropriate kubectl commands."},
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
	req.Header.Set("api-key", p.config.APIKey) // Azure uses api-key header

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
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading response: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chatResp openAIChatResponse
		if err := json.Unmarshal([]byte(data), &chatResp); err != nil {
			continue
		}

		for _, choice := range chatResp.Choices {
			if choice.Delta.Content != "" {
				callback(choice.Delta.Content)
			}
		}
	}

	return nil
}

func (p *AzureOpenAIProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	endpoint := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-02-15-preview",
		p.endpoint, p.deployment)

	reqBody := openAIChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful Kubernetes assistant."},
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
	req.Header.Set("api-key", p.config.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (p *AzureOpenAIProvider) ListModels(ctx context.Context) ([]string, error) {
	// Azure doesn't have a models list endpoint - return common deployments
	return []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-35-turbo",
	}, nil
}

// AskWithTools implements the ToolProvider interface for Azure OpenAI
func (p *AzureOpenAIProvider) AskWithTools(ctx context.Context, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) error {
	endpoint := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=2024-02-15-preview",
		p.endpoint, p.deployment)

	messages := []ChatMessage{
		{Role: "system", Content: `You are a Kubernetes expert assistant with DIRECT ACCESS to kubectl and bash tools.
ALWAYS USE TOOLS to execute commands - NEVER just suggest commands.
When asked about Kubernetes resources, IMMEDIATELY use the kubectl tool.`},
		{Role: "user", Content: prompt},
	}

	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		reqBody := azureOpenAIChatRequest{
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
		req.Header.Set("api-key", p.config.APIKey)

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}

		var chatResp azureOpenAIChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		if len(chatResp.Choices) == 0 {
			return fmt.Errorf("no response from API")
		}

		choice := chatResp.Choices[0]
		content := choice.Message.Content
		toolCalls := choice.Message.ToolCalls

		// If no tool calls, return the content
		if len(toolCalls) == 0 {
			if callback != nil && content != "" {
				callback(content)
			}
			return nil
		}

		// Add assistant message with tool calls to history
		assistantMsg := ChatMessage{
			Role:      "assistant",
			Content:   content,
			ToolCalls: toolCalls,
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call and add results
		for _, tc := range toolCalls {
			if callback != nil {
				callback(fmt.Sprintf("\n\n🔧 Executing: %s\n", tc.Function.Name))
			}

			result := toolCallback(tc)

			// Add tool result to messages
			toolMsg := ChatMessage{
				Role:       "tool",
				Content:    result.Content,
				ToolCallID: tc.ID,
			}
			messages = append(messages, toolMsg)

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

	return fmt.Errorf("exceeded maximum iterations")
}

// azureOpenAIChatRequest extends the request with tools support
type azureOpenAIChatRequest struct {
	Messages []ChatMessage    `json:"messages"`
	Stream   bool             `json:"stream"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
}

// azureOpenAIChatResponse includes tool calls
type azureOpenAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}
