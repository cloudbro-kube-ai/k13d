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
	"time"
)

// EmbeddedProvider implements the Provider interface for the embedded llama.cpp server
// It uses the OpenAI-compatible API provided by llama-server
type EmbeddedProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	endpoint   string
}

// OpenAI-compatible request/response structures for llama.cpp server
type embeddedChatRequest struct {
	Model       string        `json:"model,omitempty"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Tools       []interface{} `json:"tools,omitempty"`
}

type embeddedChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
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
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewEmbeddedProvider creates a new embedded LLM provider
func NewEmbeddedProvider(cfg *ProviderConfig) (Provider, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "http://127.0.0.1:8081"
	}
	endpoint = strings.TrimSuffix(endpoint, "/")

	model := cfg.Model
	if model == "" {
		model = "qwen2.5-0.5b-instruct" // Default embedded model
	}

	return &EmbeddedProvider{
		config: &ProviderConfig{
			Provider: "embedded",
			Model:    model,
			Endpoint: endpoint,
		},
		httpClient: &http.Client{},
		endpoint:   endpoint,
	}, nil
}

func (p *EmbeddedProvider) Name() string {
	return "embedded"
}

func (p *EmbeddedProvider) GetModel() string {
	return p.config.Model
}

func (p *EmbeddedProvider) IsReady() bool {
	if p.config == nil || p.endpoint == "" {
		return false
	}

	// Check if server is responding
	resp, err := p.httpClient.Get(p.endpoint + "/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (p *EmbeddedProvider) Ask(ctx context.Context, prompt string, callback func(string)) error {
	endpoint := p.endpoint + "/v1/chat/completions"

	reqBody := embeddedChatRequest{
		Model: p.config.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful Kubernetes assistant. Help users manage Kubernetes clusters using natural language. When users ask to create resources, generate the appropriate kubectl commands. Be concise and practical."},
			{Role: "user", Content: prompt},
		},
		Stream:      true,
		Temperature: 0.7,
		MaxTokens:   1024,
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
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
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
		if line == "" || line == "data: [DONE]" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var chatResp embeddedChatResponse
		if err := json.Unmarshal([]byte(data), &chatResp); err != nil {
			continue
		}

		if len(chatResp.Choices) > 0 {
			content := chatResp.Choices[0].Delta.Content
			if content != "" {
				callback(content)
			}

			if chatResp.Choices[0].FinishReason == "stop" {
				break
			}
		}
	}

	return nil
}

func (p *EmbeddedProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	endpoint := p.endpoint + "/v1/chat/completions"

	reqBody := embeddedChatRequest{
		Model: p.config.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful Kubernetes assistant."},
			{Role: "user", Content: prompt},
		},
		Stream:      false,
		Temperature: 0.7,
		MaxTokens:   1024,
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
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp embeddedChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (p *EmbeddedProvider) ListModels(ctx context.Context) ([]string, error) {
	// Embedded server typically has only one model loaded
	return []string{p.config.Model}, nil
}

// AskWithTools implements tool calling for the embedded provider
func (p *EmbeddedProvider) AskWithTools(ctx context.Context, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) error {
	endpoint := p.endpoint + "/v1/chat/completions"

	// Convert tools to the format expected by the API
	var apiTools []interface{}
	for _, tool := range tools {
		apiTools = append(apiTools, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Function.Name,
				"description": tool.Function.Description,
				"parameters":  tool.Function.Parameters,
			},
		})
	}

	messages := []ChatMessage{
		{Role: "system", Content: "You are a helpful Kubernetes assistant with tool calling capabilities. Use the provided tools when appropriate to help users manage their Kubernetes clusters."},
		{Role: "user", Content: prompt},
	}

	// Tool calling loop
	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		reqBody := embeddedChatRequest{
			Model:       p.config.Model,
			Messages:    messages,
			Stream:      false,
			Temperature: 0.7,
			MaxTokens:   1024,
			Tools:       apiTools,
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

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}

		var chatResp embeddedChatResponse
		if err := json.Unmarshal(body, &chatResp); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		if len(chatResp.Choices) == 0 {
			return fmt.Errorf("no response from model")
		}

		choice := chatResp.Choices[0]

		// Check for tool calls from API
		if len(choice.Message.ToolCalls) > 0 {
			// Add assistant message with tool calls
			messages = append(messages, ChatMessage{
				Role:      "assistant",
				Content:   choice.Message.Content,
				ToolCalls: choice.Message.ToolCalls,
			})

			// Execute each tool call
			for _, toolCall := range choice.Message.ToolCalls {
				result := toolCallback(toolCall)
				messages = append(messages, ChatMessage{
					Role:       "tool",
					Content:    result.Content,
					ToolCallID: toolCall.ID,
				})
			}

			continue // Loop for next response
		}

		// Fallback: Parse tool calls from text output (for small models that output JSON as text)
		content := choice.Message.Content
		if parsedToolCall := p.parseToolCallFromText(content); parsedToolCall != nil {
			// Execute the parsed tool call
			result := toolCallback(*parsedToolCall)

			// Add messages for the tool call flow
			messages = append(messages, ChatMessage{
				Role:      "assistant",
				Content:   content,
				ToolCalls: []ToolCall{*parsedToolCall},
			})
			messages = append(messages, ChatMessage{
				Role:       "tool",
				Content:    result.Content,
				ToolCallID: parsedToolCall.ID,
			})

			continue // Loop for next response with tool result
		}

		// No tool calls, return final response
		if content != "" {
			callback(content)
		}
		return nil
	}

	return fmt.Errorf("max tool call iterations exceeded")
}

// parseToolCallFromText attempts to extract tool call information from text output
// Small models sometimes output tool calls as JSON text instead of using the API properly
func (p *EmbeddedProvider) parseToolCallFromText(content string) *ToolCall {
	// Common patterns for tool calls in text:
	// 1. {{"name": "...", "arguments": {...}}}
	// 2. {"name": "...", "arguments": {...}}
	// 3. ```json\n{...}\n```

	content = strings.TrimSpace(content)

	// Try to find JSON object in the content
	var jsonStr string

	// Pattern 1: Double braces (common LLM quirk)
	if idx := strings.Index(content, "{{"); idx != -1 {
		if endIdx := strings.LastIndex(content, "}}"); endIdx > idx {
			// Extract and fix double braces
			jsonStr = content[idx+1 : endIdx+1]
		}
	}

	// Pattern 2: Single braces
	if jsonStr == "" {
		if idx := strings.Index(content, "{"); idx != -1 {
			if endIdx := strings.LastIndex(content, "}"); endIdx > idx {
				jsonStr = content[idx : endIdx+1]
			}
		}
	}

	if jsonStr == "" {
		return nil
	}

	// Try to parse the JSON
	var toolCallData struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &toolCallData); err != nil {
		return nil
	}

	// Validate we got a valid tool name
	if toolCallData.Name == "" {
		return nil
	}

	// Convert arguments to string
	argsStr := string(toolCallData.Arguments)
	if argsStr == "" || argsStr == "null" {
		argsStr = "{}"
	}

	return &ToolCall{
		ID:   fmt.Sprintf("call_%d", time.Now().UnixNano()),
		Type: "function",
		Function: FunctionCall{
			Name:      toolCallData.Name,
			Arguments: argsStr,
		},
	}
}
