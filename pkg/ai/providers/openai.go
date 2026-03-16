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

// OpenAIProvider implements the Provider interface for OpenAI and compatible APIs
type OpenAIProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	endpoint   string
}

type openAIChatRequest struct {
	Model           string           `json:"model"`
	Messages        []ChatMessage    `json:"messages"`
	Stream          bool             `json:"stream"`
	Tools           []ToolDefinition `json:"tools,omitempty"`
	ReasoningEffort string           `json:"reasoning_effort,omitempty"` // For Solar Pro2: "minimal" or "high"
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		Delta struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type openAIModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// ReActResponse represents the JSON response format for Tool Use Shim mode
// This allows LLMs that don't support native tool calling to still execute commands
type ReActResponse struct {
	Thought string       `json:"thought"`
	Answer  string       `json:"answer,omitempty"`
	Action  *ReActAction `json:"action,omitempty"`
}

// ReActAction represents a tool invocation in ReAct format
type ReActAction struct {
	Name             string `json:"name"`              // "kubectl" or "bash"
	Reason           string `json:"reason"`            // Why this tool was chosen
	Command          string `json:"command"`           // The command to execute
	ModifiesResource string `json:"modifies_resource"` // "yes", "no", or "unknown"
}

// reasoningEffort returns the reasoning_effort value only for models that support it
// (OpenAI o-series: o1, o3, o4-mini, etc. and Solar Pro2).
// For standard GPT models, returns "" so the field is omitted via omitempty.
func reasoningEffortForModel(model, effort string) string {
	if effort == "" {
		return ""
	}
	m := strings.ToLower(model)
	// OpenAI o-series models (o1, o1-mini, o1-pro, o3, o3-mini, o4-mini, etc.)
	if len(m) >= 2 && m[0] == 'o' && m[1] >= '0' && m[1] <= '9' {
		return effort
	}
	// Solar Pro2
	if strings.Contains(m, "solar-pro2") {
		return effort
	}
	return ""
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(cfg *ProviderConfig) (Provider, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	endpoint = strings.TrimSuffix(endpoint, "/")

	return &OpenAIProvider{
		config:     cfg,
		httpClient: newHTTPClient(cfg.SkipTLSVerify),
		endpoint:   endpoint,
	}, nil
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) GetModel() string {
	return p.config.Model
}

func (p *OpenAIProvider) IsReady() bool {
	return p.config != nil && p.config.APIKey != ""
}

func (p *OpenAIProvider) Ask(ctx context.Context, prompt string, callback func(string)) error {
	endpoint := p.endpoint + "/chat/completions"

	reqBody := openAIChatRequest{
		Model: p.config.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful Kubernetes assistant. Help users manage Kubernetes clusters using natural language. When users ask to create resources, generate the appropriate kubectl commands."},
			{Role: "user", Content: prompt},
		},
		Stream:          true,
		ReasoningEffort: reasoningEffortForModel(p.config.Model, p.config.ReasoningEffort),
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
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

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

func (p *OpenAIProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	endpoint := p.endpoint + "/chat/completions"

	reqBody := openAIChatRequest{
		Model: p.config.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a helpful Kubernetes assistant. Help users manage Kubernetes clusters using natural language. When users ask to create resources, generate the appropriate kubectl commands."},
			{Role: "user", Content: prompt},
		},
		Stream:          false,
		ReasoningEffort: reasoningEffortForModel(p.config.Model, p.config.ReasoningEffort),
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
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

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

func (p *OpenAIProvider) ListModels(ctx context.Context) ([]string, error) {
	endpoint := p.endpoint + "/models"

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models: status %d", resp.StatusCode)
	}

	var modelsResp openAIModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(modelsResp.Data))
	for i, m := range modelsResp.Data {
		models[i] = m.ID
	}
	return models, nil
}

// AskWithTools implements the ToolProvider interface for agentic tool calling
// Supports both native tool calling and Tool Use Shim mode for LLMs without tool support
func (p *OpenAIProvider) AskWithTools(ctx context.Context, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) error {
	endpoint := p.endpoint + "/chat/completions"
	tools = sortedToolDefinitions(tools)

	// First, try with native tool calling
	nativeToolSuccess := p.tryNativeToolCalling(ctx, endpoint, prompt, tools, callback, toolCallback)
	if nativeToolSuccess {
		return nil
	}

	// Fallback to Tool Use Shim mode (ReAct format)
	log.Debugf("Native tool calling failed or not supported, using Tool Use Shim mode")
	return p.askWithToolsShim(ctx, endpoint, prompt, tools, callback, toolCallback)
}

// tryNativeToolCalling attempts to use native OpenAI-style tool calling
// Returns true if tool calls were made, false if model doesn't support it
func (p *OpenAIProvider) tryNativeToolCalling(ctx context.Context, endpoint, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) bool {
	tools = sortedToolDefinitions(tools)
	maxIterations := effectiveMaxIterations(p.config)
	messages := []ChatMessage{
		{Role: "system", Content: toolAgentSystemPrompt(maxIterations)},
		{Role: "user", Content: prompt},
	}

	toolCallMade := false
	for i := 0; i < maxIterations; i++ {
		reqBody := openAIChatRequest{
			Model:           p.config.Model,
			Messages:        messages,
			Stream:          false, // Non-streaming for first request to detect tool support
			Tools:           tools,
			ReasoningEffort: reasoningEffortForModel(p.config.Model, p.config.ReasoningEffort),
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			log.Debugf("Failed to marshal request: %v", err)
			return false
		}

		// Debug: Log request with tools
		log.Debugf("tryNativeToolCalling - Model: %s, Tools count: %d, Iteration: %d", p.config.Model, len(tools), i+1)

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
		if err != nil {
			log.Debugf("Failed to create request: %v", err)
			return false
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

		resp, err := p.httpClient.Do(req)
		if err != nil {
			log.Debugf("Request failed: %v", err)
			return false
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			log.Debugf("API error (status %d): %s", resp.StatusCode, string(body))
			return false
		}

		// Parse non-streaming response
		var chatResp openAIChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			resp.Body.Close()
			log.Debugf("Failed to decode response: %v", err)
			return false
		}
		resp.Body.Close()

		if len(chatResp.Choices) == 0 {
			log.Debugf("No choices in response")
			return false
		}

		choice := chatResp.Choices[0]
		content := choice.Message.Content
		toolCalls := choice.Message.ToolCalls
		finishReason := choice.FinishReason

		// Debug: Log response
		log.Debugf("Native tool calling response - FinishReason: %s, ToolCalls: %d, Content length: %d", finishReason, len(toolCalls), len(content))

		// If no tool calls, check if model supports tool calling
		if len(toolCalls) == 0 {
			if i == 0 {
				// First request with no tool calls - model doesn't support tool calling
				log.Debugf("No tool calls on first request. Model may not support tool calling.")
				return false // Signal to use fallback
			}
			if content != "" {
				messages = append(messages, ChatMessage{
					Role:    "assistant",
					Content: content,
				})
			}
			if callback != nil {
				if toolCallMade {
					p.streamFinalResponse(ctx, endpoint, messages, callback)
				} else if content != "" {
					callback(content)
				}
			}
			return toolCallMade
		}

		toolCallMade = true

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
					// Truncate long outputs
					output := result.Content
					if len(output) > 1000 {
						output = output[:1000] + "\n... (truncated)"
					}
					callback(fmt.Sprintf("```\n%s\n```\n", output))
				}
			}
		}
	}

	if toolCallMade && callback != nil {
		p.streamFinalResponse(ctx, endpoint, messages, callback)
	}
	return toolCallMade
}

// streamFinalResponse makes a streaming request after tool execution for better UX
func (p *OpenAIProvider) streamFinalResponse(ctx context.Context, endpoint string, messages []ChatMessage, callback func(string)) {
	// Copy messages slice to avoid aliasing the caller's slice
	finalMessages := make([]ChatMessage, len(messages), len(messages)+1)
	copy(finalMessages, messages)
	finalMessages = append(finalMessages, ChatMessage{
		Role:    "user",
		Content: finalToolSummaryPrompt,
	})

	reqBody := openAIChatRequest{
		Model:           p.config.Model,
		Messages:        finalMessages,
		Stream:          true,
		ReasoningEffort: reasoningEffortForModel(p.config.Model, p.config.ReasoningEffort),
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Debugf("Failed to marshal streaming request: %v", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Debugf("Failed to create streaming request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Debugf("Streaming request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		log.Debugf("Streaming API error (status %d): %s", resp.StatusCode, string(body))
		return
	}

	// Stream the response
	callback("\n\n")
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Debugf("Error reading streaming response: %v", err)
			return
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
}

// extractJSONFromMarkdown extracts JSON from markdown code blocks
func extractJSONFromMarkdown(s string) (string, bool) {
	const jsonBlockMarker = "```json"
	const endMarker = "```"

	first := strings.Index(s, jsonBlockMarker)
	if first == -1 {
		return "", false
	}

	// Find the closing ```
	rest := s[first+len(jsonBlockMarker):]
	last := strings.Index(rest, endMarker)
	if last == -1 {
		return "", false
	}

	data := rest[:last]
	data = strings.TrimSpace(data)
	return data, true
}

// parseReActResponse parses a ReAct JSON response from the LLM
func parseReActResponse(input string) (*ReActResponse, error) {
	cleaned, found := extractJSONFromMarkdown(input)
	if !found {
		// Try to find JSON without markdown markers
		cleaned = strings.TrimSpace(input)
		if !strings.HasPrefix(cleaned, "{") {
			return nil, fmt.Errorf("no JSON found in response")
		}
	}

	var reActResp ReActResponse
	if err := json.Unmarshal([]byte(cleaned), &reActResp); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return &reActResp, nil
}

// askWithToolsShim implements Tool Use Shim mode using ReAct format
// This is used when the LLM doesn't support native tool calling
func (p *OpenAIProvider) askWithToolsShim(ctx context.Context, endpoint, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) error {
	maxIterations := effectiveMaxIterations(p.config)
	messages := []ChatMessage{
		{Role: "system", Content: buildToolUseShimSystemPrompt(tools, maxIterations)},
		{Role: "user", Content: prompt},
	}

	for i := 0; i < maxIterations; i++ {
		// Request without tools (using ReAct prompting instead)
		reqBody := openAIChatRequest{
			Model:           p.config.Model,
			Messages:        messages,
			Stream:          false, // Non-streaming for easier parsing
			ReasoningEffort: reasoningEffortForModel(p.config.Model, p.config.ReasoningEffort),
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		log.Debugf("Tool Use Shim request - Iteration: %d", i+1)

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}

		var chatResp openAIChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		if len(chatResp.Choices) == 0 {
			return fmt.Errorf("no response from API")
		}

		content := chatResp.Choices[0].Message.Content
		log.Debugf("Tool Use Shim response length: %d", len(content))

		// Parse the ReAct response
		reActResp, err := parseReActResponse(content)
		if err != nil {
			// If parsing fails, show the raw content to user
			if callback != nil {
				callback(content)
			}
			return nil
		}

		// Note: Thought process is intentionally not shown to users for cleaner output
		// The thought field is used internally for ReAct reasoning but not displayed

		// If there's a final answer, we're done
		if reActResp.Answer != "" {
			if callback != nil {
				callback(reActResp.Answer)
			}
			return nil
		}

		// If there's an action, execute it
		if reActResp.Action != nil {
			action := reActResp.Action

			if callback != nil {
				callback(fmt.Sprintf("🔧 Executing: %s\n", action.Command))
			}

			// Check if approval needed for write operations
			needsApproval := action.ModifiesResource == "yes" || action.ModifiesResource == "unknown"

			// Create a ToolCall from the action
			tc := ToolCall{
				ID:   fmt.Sprintf("shim_%d", i),
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      action.Name,
					Arguments: fmt.Sprintf(`{"command":"%s"}`, strings.ReplaceAll(action.Command, `"`, `\"`)),
				},
			}

			// Execute via callback (which handles approval)
			result := toolCallback(tc)

			// Show result
			if callback != nil {
				if result.IsError {
					callback(fmt.Sprintf("❌ Error: %s\n\n", result.Content))
				} else {
					output := result.Content
					if len(output) > 1500 {
						output = output[:1500] + "\n... (truncated)"
					}
					callback(fmt.Sprintf("```\n%s\n```\n\n", output))
				}
			}

			// Add the exchange to messages for context
			messages = append(messages, ChatMessage{
				Role:    "assistant",
				Content: content,
			})

			// Add result as user message (observation)
			resultStatus := "succeeded"
			if result.IsError {
				resultStatus = "failed"
			}
			observation := fmt.Sprintf("Command %s. Result:\n%s", resultStatus, result.Content)
			messages = append(messages, ChatMessage{
				Role:    "user",
				Content: fmt.Sprintf("Observation: %s\n\nNow provide your final answer or next action.", observation),
			})

			// Skip approval check for read-only operations
			_ = needsApproval // Used by toolCallback internally
		} else {
			// No action and no answer - unexpected
			if callback != nil {
				callback(content)
			}
			return nil
		}
	}

	return fmt.Errorf("exceeded maximum iterations (%d)", maxIterations)
}
