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

// GeminiProvider implements the Provider and ToolProvider interfaces for Google Gemini
type GeminiProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	endpoint   string
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role  string       `json:"role"`
}

type geminiPart struct {
	Text             string            `json:"text,omitempty"`
	FunctionCall     *geminiFuncCall   `json:"functionCall,omitempty"`
	FunctionResponse *geminiFuncResult `json:"functionResponse,omitempty"`
}

type geminiFuncCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type geminiFuncResult struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type geminiToolDecl struct {
	FunctionDeclarations []geminiFuncDecl `json:"functionDeclarations"`
}

type geminiFuncDecl struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type geminiRequest struct {
	Contents          []geminiContent  `json:"contents"`
	SystemInstruction *geminiContent   `json:"systemInstruction,omitempty"`
	Tools             []geminiToolDecl `json:"tools,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type geminiModelsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

const geminiSystemPrompt = "You are a helpful Kubernetes assistant. Help users manage Kubernetes clusters using natural language. When users ask to create resources, generate the appropriate kubectl commands."

// NewGeminiProvider creates a new Google Gemini provider
func NewGeminiProvider(cfg *ProviderConfig) (Provider, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://generativelanguage.googleapis.com/v1beta"
	}
	endpoint = strings.TrimSuffix(endpoint, "/")

	model := cfg.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}

	// Validate Gemini model name format
	if err := validateGeminiModel(model); err != nil {
		return nil, err
	}

	return &GeminiProvider{
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

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) GetModel() string {
	return p.config.Model
}

func (p *GeminiProvider) IsReady() bool {
	return p.config != nil && p.config.APIKey != ""
}

func (p *GeminiProvider) Ask(ctx context.Context, prompt string, callback func(string)) error {
	endpoint := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s&alt=sse",
		p.endpoint, p.config.Model, p.config.APIKey)

	reqBody := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: geminiSystemPrompt}},
		},
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: prompt}},
			},
		},
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
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "" {
			continue
		}

		var geminiResp geminiResponse
		if err := json.Unmarshal([]byte(data), &geminiResp); err != nil {
			continue
		}

		for _, candidate := range geminiResp.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					callback(part.Text)
				}
			}
		}
	}

	return nil
}

func (p *GeminiProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		p.endpoint, p.config.Model, p.config.APIKey)

	reqBody := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: geminiSystemPrompt}},
		},
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: prompt}},
			},
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

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	var result strings.Builder
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		result.WriteString(part.Text)
	}
	return result.String(), nil
}

func (p *GeminiProvider) ListModels(ctx context.Context) ([]string, error) {
	endpoint := fmt.Sprintf("%s/models?key=%s", p.endpoint, p.config.APIKey)

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

	var modelsResp geminiModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(modelsResp.Models))
	for i, m := range modelsResp.Models {
		// Strip "models/" prefix
		name := strings.TrimPrefix(m.Name, "models/")
		models[i] = name
	}
	return models, nil
}

// AskWithTools implements ToolProvider for Gemini using functionDeclarations/functionCall.
func (p *GeminiProvider) AskWithTools(ctx context.Context, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) error {
	// Convert tools to Gemini format
	var funcDecls []geminiFuncDecl
	for _, tool := range tools {
		funcDecls = append(funcDecls, geminiFuncDecl{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		})
	}

	geminiTools := []geminiToolDecl{{FunctionDeclarations: funcDecls}}

	contents := []geminiContent{
		{
			Role:  "user",
			Parts: []geminiPart{{Text: prompt}},
		},
	}

	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
			p.endpoint, p.config.Model, p.config.APIKey)

		reqBody := geminiRequest{
			SystemInstruction: &geminiContent{
				Parts: []geminiPart{{Text: `You are a Kubernetes expert assistant with DIRECT ACCESS to kubectl and bash tools.
ALWAYS USE TOOLS to execute commands - NEVER just suggest commands.
When asked about Kubernetes resources, IMMEDIATELY use the kubectl tool.`}},
			},
			Contents: contents,
			Tools:    geminiTools,
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
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}

		var geminiResp geminiResponse
		if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
			return fmt.Errorf("no response from Gemini API")
		}

		parts := geminiResp.Candidates[0].Content.Parts

		// Check for function calls in the response
		var funcCalls []geminiPart
		var textParts []string
		for _, part := range parts {
			if part.FunctionCall != nil {
				funcCalls = append(funcCalls, part)
			}
			if part.Text != "" {
				textParts = append(textParts, part.Text)
			}
		}

		// No function calls - return text response
		if len(funcCalls) == 0 {
			if callback != nil {
				for _, text := range textParts {
					callback(text)
				}
			}
			return nil
		}

		// Add model response to contents
		contents = append(contents, geminiContent{
			Role:  "model",
			Parts: parts,
		})

		// Execute each function call and collect results
		var resultParts []geminiPart
		for _, fc := range funcCalls {
			if callback != nil {
				callback(fmt.Sprintf("\n\n🔧 Executing: %s\n", fc.FunctionCall.Name))
			}

			// Convert Gemini function call to ToolCall for the callback
			argsJSON, _ := json.Marshal(fc.FunctionCall.Args)
			tc := ToolCall{
				ID:   fmt.Sprintf("gemini_%d_%s", i, fc.FunctionCall.Name),
				Type: "function",
				Function: FunctionCall{
					Name:      fc.FunctionCall.Name,
					Arguments: string(argsJSON),
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

			resultParts = append(resultParts, geminiPart{
				FunctionResponse: &geminiFuncResult{
					Name: fc.FunctionCall.Name,
					Response: map[string]interface{}{
						"result": result.Content,
					},
				},
			})
		}

		// Add function results to contents
		contents = append(contents, geminiContent{
			Role:  "user",
			Parts: resultParts,
		})
	}

	return fmt.Errorf("exceeded maximum tool call iterations")
}

// validGeminiModelPrefixes lists known valid Gemini model name prefixes.
// Models must start with "gemini-" followed by a version number.
var validGeminiModelPrefixes = []string{
	"gemini-3-",
	"gemini-2.5-",
	"gemini-2.0-",
	"gemini-1.5-",
	"gemini-1.0-",
	"gemini-pro",
	"gemini-ultra",
	"gemini-nano",
}

// validateGeminiModel checks if the model name is a valid Gemini model.
func validateGeminiModel(model string) error {
	if !strings.HasPrefix(model, "gemini-") {
		return fmt.Errorf("invalid Gemini model name %q: must start with 'gemini-' (e.g., gemini-2.0-flash, gemini-1.5-pro)", model)
	}

	for _, prefix := range validGeminiModelPrefixes {
		if strings.HasPrefix(model, prefix) {
			return nil
		}
	}

	return fmt.Errorf("invalid Gemini model name %q: use a versioned name like gemini-2.5-flash, gemini-2.5-pro, gemini-2.0-flash", model)
}
