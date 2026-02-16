package providers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// BedrockProvider implements the Provider interface for AWS Bedrock (Claude)
type BedrockProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	region     string
}

type bedrockClaudeRequest struct {
	AnthropicVersion string             `json:"anthropic_version"`
	MaxTokens        int                `json:"max_tokens"`
	System           string             `json:"system,omitempty"`
	Messages         []bedrockClaudeMsg `json:"messages"`
}

type bedrockClaudeMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type bedrockClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
}

// NewBedrockProvider creates a new AWS Bedrock provider
func NewBedrockProvider(cfg *ProviderConfig) (Provider, error) {
	region := cfg.Region
	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1"
		}
	}

	model := cfg.Model
	if model == "" {
		model = "anthropic.claude-3-sonnet-20240229-v1:0"
	}

	return &BedrockProvider{
		config: &ProviderConfig{
			Provider: cfg.Provider,
			Model:    model,
			APIKey:   cfg.APIKey, // AWS Secret Access Key
			Endpoint: cfg.Endpoint,
			Region:   region,
		},
		httpClient: newHTTPClient(cfg.SkipTLSVerify),
		region:     region,
	}, nil
}

func (p *BedrockProvider) Name() string {
	return "bedrock"
}

func (p *BedrockProvider) GetModel() string {
	return p.config.Model
}

func (p *BedrockProvider) IsReady() bool {
	// Check for AWS credentials (either in config or environment)
	if p.config.APIKey != "" {
		return true
	}
	return os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != ""
}

func (p *BedrockProvider) Ask(ctx context.Context, prompt string, callback func(string)) error {
	// Bedrock streaming is more complex, using non-streaming for simplicity
	response, err := p.AskNonStreaming(ctx, prompt)
	if err != nil {
		return err
	}
	callback(response)
	return nil
}

func (p *BedrockProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke",
		p.region, p.config.Model)

	reqBody := bedrockClaudeRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        4096,
		System:           "You are a helpful Kubernetes assistant. Help users manage Kubernetes clusters using natural language. When users ask to create resources, generate the appropriate kubectl commands.",
		Messages: []bedrockClaudeMsg{
			{Role: "user", Content: prompt},
		},
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

	// Sign the request with AWS Signature V4
	if err := p.signRequest(req, jsonBody); err != nil {
		return "", fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var bedrockResp bedrockClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&bedrockResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(bedrockResp.Content) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	var result strings.Builder
	for _, content := range bedrockResp.Content {
		result.WriteString(content.Text)
	}
	return result.String(), nil
}

func (p *BedrockProvider) ListModels(ctx context.Context) ([]string, error) {
	// Return common Bedrock Claude models
	return []string{
		"anthropic.claude-3-5-sonnet-20241022-v2:0",
		"anthropic.claude-3-5-haiku-20241022-v1:0",
		"anthropic.claude-3-sonnet-20240229-v1:0",
		"anthropic.claude-3-haiku-20240307-v1:0",
		"anthropic.claude-3-opus-20240229-v1:0",
	}, nil
}

// BedrockToolConfig represents tool configuration for Bedrock Claude
type bedrockToolConfig struct {
	Tools []bedrockTool `json:"tools,omitempty"`
}

type bedrockTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// bedrockClaudeToolRequest extends the request with tools support
type bedrockClaudeToolRequest struct {
	AnthropicVersion string                 `json:"anthropic_version"`
	MaxTokens        int                    `json:"max_tokens"`
	System           string                 `json:"system,omitempty"`
	Messages         []bedrockClaudeMessage `json:"messages"`
	Tools            []bedrockTool          `json:"tools,omitempty"`
}

type bedrockClaudeMessage struct {
	Role    string                 `json:"role"`
	Content []bedrockClaudeContent `json:"content"`
}

type bedrockClaudeContent struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

type bedrockClaudeToolResponse struct {
	Content    []bedrockToolContent `json:"content"`
	StopReason string               `json:"stop_reason"`
}

type bedrockToolContent struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}

// AskWithTools implements the ToolProvider interface for Bedrock Claude
func (p *BedrockProvider) AskWithTools(ctx context.Context, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) error {
	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke",
		p.region, p.config.Model)

	// Convert tools to Bedrock format
	bedrockTools := make([]bedrockTool, 0, len(tools))
	for _, t := range tools {
		bedrockTools = append(bedrockTools, bedrockTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		})
	}

	messages := []bedrockClaudeMessage{
		{
			Role: "user",
			Content: []bedrockClaudeContent{
				{Type: "text", Text: prompt},
			},
		},
	}

	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		reqBody := bedrockClaudeToolRequest{
			AnthropicVersion: "bedrock-2023-05-31",
			MaxTokens:        4096,
			System:           "You are a helpful Kubernetes assistant with DIRECT ACCESS to kubectl and bash tools. ALWAYS USE TOOLS to execute commands - NEVER just suggest commands.",
			Messages:         messages,
			Tools:            bedrockTools,
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

		if err := p.signRequest(req, jsonBody); err != nil {
			return fmt.Errorf("failed to sign request: %w", err)
		}

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}

		var bedrockResp bedrockClaudeToolResponse
		if err := json.NewDecoder(resp.Body).Decode(&bedrockResp); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		// Process response content
		var textContent strings.Builder
		var toolUses []bedrockToolContent

		for _, content := range bedrockResp.Content {
			if content.Type == "text" {
				textContent.WriteString(content.Text)
			} else if content.Type == "tool_use" {
				toolUses = append(toolUses, content)
			}
		}

		// Output text content
		if callback != nil && textContent.Len() > 0 {
			callback(textContent.String())
		}

		// If no tool uses, we're done
		if len(toolUses) == 0 {
			return nil
		}

		// Build assistant message with content
		assistantContent := make([]bedrockClaudeContent, 0)
		if textContent.Len() > 0 {
			assistantContent = append(assistantContent, bedrockClaudeContent{
				Type: "text",
				Text: textContent.String(),
			})
		}
		for _, tu := range toolUses {
			assistantContent = append(assistantContent, bedrockClaudeContent{
				Type:  "tool_use",
				ID:    tu.ID,
				Name:  tu.Name,
				Input: tu.Input,
			})
		}
		messages = append(messages, bedrockClaudeMessage{
			Role:    "assistant",
			Content: assistantContent,
		})

		// Execute tool calls
		toolResults := make([]bedrockClaudeContent, 0)
		for _, tu := range toolUses {
			if callback != nil {
				callback(fmt.Sprintf("\n\n🔧 Executing: %s\n", tu.Name))
			}

			// Convert input to JSON string for ToolCall
			inputJSON, _ := json.Marshal(tu.Input)

			tc := ToolCall{
				ID:   tu.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      tu.Name,
					Arguments: string(inputJSON),
				},
			}

			result := toolCallback(tc)

			toolResults = append(toolResults, bedrockClaudeContent{
				Type:      "tool_result",
				ToolUseID: tu.ID,
				Content:   result.Content,
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

		// Add tool results as user message
		messages = append(messages, bedrockClaudeMessage{
			Role:    "user",
			Content: toolResults,
		})
	}

	return fmt.Errorf("exceeded maximum iterations")
}

// signRequest signs the request with AWS Signature V4
func (p *BedrockProvider) signRequest(req *http.Request, body []byte) error {
	// Get credentials from environment or config
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")

	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("AWS credentials not configured (set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY)")
	}

	// Create signing time
	t := time.Now().UTC()
	amzDate := t.Format("20060102T150405Z")
	dateStamp := t.Format("20060102")

	// Create canonical request
	host := req.URL.Host
	method := req.Method
	canonicalURI := req.URL.Path
	canonicalQueryString := ""

	// Hash the payload
	payloadHash := sha256Hex(body)

	// Set required headers
	req.Header.Set("Host", host)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", sessionToken)
	}

	// Create canonical headers
	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"), host, payloadHash, amzDate)

	if sessionToken != "" {
		signedHeaders += ";x-amz-security-token"
		canonicalHeaders += fmt.Sprintf("x-amz-security-token:%s\n", sessionToken)
	}

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		method, canonicalURI, canonicalQueryString, canonicalHeaders, signedHeaders, payloadHash)

	// Create string to sign
	algorithm := "AWS4-HMAC-SHA256"
	credentialScope := fmt.Sprintf("%s/%s/bedrock/aws4_request", dateStamp, p.region)
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm, amzDate, credentialScope, sha256Hex([]byte(canonicalRequest)))

	// Create signing key
	signingKey := getSignatureKey(secretKey, dateStamp, p.region, "bedrock")

	// Create signature
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Add authorization header
	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)

	return nil
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getSignatureKey(key, dateStamp, regionName, serviceName string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+key), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(regionName))
	kService := hmacSHA256(kRegion, []byte(serviceName))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}
