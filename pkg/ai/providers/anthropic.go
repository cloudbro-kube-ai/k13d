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

	"github.com/cloudbro-kube-ai/k13d/pkg/log"
)

// AnthropicProvider implements the Provider and ToolProvider interfaces
// for the Anthropic Messages API (https://docs.anthropic.com/en/api/messages).
type AnthropicProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	endpoint   string
}

const anthropicAPIVersion = "2023-06-01"
const anthropicDefaultMaxTokens = 4096

// Anthropic request/response types

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream,omitempty"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string      `json:"role"` // "user" or "assistant"
	Content interface{} `json:"content"`
}

type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// Response types

type anthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type anthropicContentBlock struct {
	Type  string          `json:"type"` // "text" or "tool_use"
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`    // tool_use
	Name  string          `json:"name,omitempty"`  // tool_use
	Input json.RawMessage `json:"input,omitempty"` // tool_use
}

// Streaming event types

type anthropicStreamEvent struct {
	Type         string                 `json:"type"`
	Index        int                    `json:"index,omitempty"`
	ContentBlock *anthropicContentBlock `json:"content_block,omitempty"`
	Delta        *anthropicStreamDelta  `json:"delta,omitempty"`
	Message      *anthropicResponse     `json:"message,omitempty"`
	Usage        *struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

type anthropicStreamDelta struct {
	Type        string `json:"type"` // "text_delta" or "input_json_delta"
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(cfg *ProviderConfig) (Provider, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.anthropic.com"
	}
	endpoint = strings.TrimSuffix(endpoint, "/")

	return &AnthropicProvider{
		config:     cfg,
		httpClient: newHTTPClient(cfg.SkipTLSVerify),
		endpoint:   endpoint,
	}, nil
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) GetModel() string {
	return p.config.Model
}

func (p *AnthropicProvider) IsReady() bool {
	return p.config != nil && p.config.APIKey != ""
}

func (p *AnthropicProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
}

// Ask sends a prompt and streams the response via callback
func (p *AnthropicProvider) Ask(ctx context.Context, prompt string, callback func(string)) error {
	reqBody := anthropicRequest{
		Model:     p.config.Model,
		MaxTokens: anthropicDefaultMaxTokens,
		System:    "You are a helpful Kubernetes assistant. Help users manage Kubernetes clusters using natural language. When users ask to create resources, generate the appropriate kubectl commands.",
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
		Stream: true,
	}

	return p.doStreamingRequest(ctx, reqBody, callback)
}

// AskNonStreaming sends a prompt and returns the full response
func (p *AnthropicProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     p.config.Model,
		MaxTokens: anthropicDefaultMaxTokens,
		System:    "You are a helpful Kubernetes assistant. Help users manage Kubernetes clusters using natural language. When users ask to create resources, generate the appropriate kubectl commands.",
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	resp, err := p.doRequest(ctx, reqBody)
	if err != nil {
		return "", err
	}

	return extractTextFromResponse(resp), nil
}

// ListModels returns known Claude models (Anthropic has no list-models endpoint)
func (p *AnthropicProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{
		"claude-opus-4-20250514",
		"claude-sonnet-4-20250514",
		"claude-haiku-4-5-20251001",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
	}, nil
}

// AskWithTools implements ToolProvider for Anthropic with native tool calling
func (p *AnthropicProvider) AskWithTools(ctx context.Context, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) error {
	// Convert tools to Anthropic format
	anthropicTools := make([]anthropicTool, 0, len(tools))
	for _, t := range tools {
		anthropicTools = append(anthropicTools, anthropicTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		})
	}

	systemPrompt := `You are a Kubernetes expert assistant with DIRECT ACCESS to kubectl and bash tools.
ALWAYS USE TOOLS to execute commands - NEVER just suggest commands.
When asked about Kubernetes resources, IMMEDIATELY use the kubectl tool.`

	messages := []anthropicMessage{
		{Role: "user", Content: prompt},
	}

	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		reqBody := anthropicRequest{
			Model:     p.config.Model,
			MaxTokens: anthropicDefaultMaxTokens,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     anthropicTools,
		}

		log.Debugf("Anthropic AskWithTools - Model: %s, Tools: %d, Iteration: %d", p.config.Model, len(anthropicTools), i+1)

		resp, err := p.doRequest(ctx, reqBody)
		if err != nil {
			return err
		}

		// Collect text and tool_use blocks from response
		var textParts []string
		var toolUseBlocks []anthropicContentBlock
		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				textParts = append(textParts, block.Text)
			case "tool_use":
				toolUseBlocks = append(toolUseBlocks, block)
			}
		}

		// Emit text to callback
		if callback != nil && len(textParts) > 0 {
			callback(strings.Join(textParts, ""))
		}

		// If no tool calls, we're done
		if len(toolUseBlocks) == 0 || resp.StopReason != "tool_use" {
			return nil
		}

		// Build assistant message with the full content array
		messages = append(messages, anthropicMessage{
			Role:    "assistant",
			Content: resp.Content,
		})

		// Execute tool calls and build tool results
		var toolResults []interface{}
		for _, block := range toolUseBlocks {
			if callback != nil {
				callback(fmt.Sprintf("\n\n🔧 Executing: %s\n", block.Name))
			}

			// Convert to ToolCall format for the callback
			argsJSON := string(block.Input)
			tc := ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      block.Name,
					Arguments: argsJSON,
				},
			}

			result := toolCallback(tc)

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

			toolResults = append(toolResults, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": block.ID,
				"content":     result.Content,
				"is_error":    result.IsError,
			})
		}

		// Add tool results as user message
		messages = append(messages, anthropicMessage{
			Role:    "user",
			Content: toolResults,
		})
	}

	return nil
}

// doRequest sends a non-streaming request and returns the parsed response
func (p *AnthropicProvider) doRequest(ctx context.Context, reqBody anthropicRequest) (*anthropicResponse, error) {
	endpoint := p.endpoint + "/v1/messages"

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if anthropicResp.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s", anthropicResp.Error.Type, anthropicResp.Error.Message)
	}

	return &anthropicResp, nil
}

// doStreamingRequest sends a streaming request and calls callback for each text chunk
func (p *AnthropicProvider) doStreamingRequest(ctx context.Context, reqBody anthropicRequest, callback func(string)) error {
	endpoint := p.endpoint + "/v1/messages"

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Type == "text_delta" && event.Delta.Text != "" {
				if callback != nil {
					callback(event.Delta.Text)
				}
			}
		case "message_stop":
			return nil
		case "error":
			if event.Message != nil && event.Message.Error != nil {
				return fmt.Errorf("stream error: %s - %s", event.Message.Error.Type, event.Message.Error.Message)
			}
		}
	}

	return nil
}

// extractTextFromResponse concatenates all text blocks from the response
func extractTextFromResponse(resp *anthropicResponse) string {
	var parts []string
	for _, block := range resp.Content {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "")
}
