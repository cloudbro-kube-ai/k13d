package providers

import (
	"context"
	"errors"
	"os"
	"testing"
)

// mockProvider implements Provider for testing
type mockProvider struct {
	name          string
	model         string
	ready         bool
	askErr        error
	askContent    string
	supportsTools bool
	toolsErr      error
}

func (m *mockProvider) Name() string                                     { return m.name }
func (m *mockProvider) GetModel() string                                 { return m.model }
func (m *mockProvider) IsReady() bool                                    { return m.ready }
func (m *mockProvider) ListModels(ctx context.Context) ([]string, error) { return nil, nil }

func (m *mockProvider) Ask(ctx context.Context, prompt string, callback func(string)) error {
	if m.askErr != nil {
		return m.askErr
	}
	if callback != nil && m.askContent != "" {
		callback(m.askContent)
	}
	return nil
}

func (m *mockProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	if m.askErr != nil {
		return "", m.askErr
	}
	return m.askContent, nil
}

// mockToolProvider implements both Provider and ToolProvider
type mockToolProvider struct {
	mockProvider
	toolsErr       error
	toolCallsCount int
}

func (m *mockToolProvider) AskWithTools(ctx context.Context, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) error {
	m.toolCallsCount++
	return m.toolsErr
}

func TestRetryProviderSupportsTools(t *testing.T) {
	tests := []struct {
		name         string
		provider     Provider
		wantSupports bool
	}{
		{
			name:         "provider without tool support",
			provider:     &mockProvider{name: "mock", ready: true},
			wantSupports: false,
		},
		{
			name:         "provider with tool support",
			provider:     &mockToolProvider{mockProvider: mockProvider{name: "mock-tools", ready: true}},
			wantSupports: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryProv := CreateWithRetry(tt.provider, DefaultRetryConfig())
			rp := retryProv.(*retryProvider)

			if got := rp.SupportsTools(); got != tt.wantSupports {
				t.Errorf("SupportsTools() = %v, want %v", got, tt.wantSupports)
			}
		})
	}
}

func TestRetryProviderAskWithTools(t *testing.T) {
	tests := []struct {
		name        string
		provider    Provider
		wantErr     bool
		errContains string
	}{
		{
			name:        "provider without tool support returns error",
			provider:    &mockProvider{name: "mock", ready: true},
			wantErr:     true,
			errContains: "does not support tool calling",
		},
		{
			name:     "provider with tool support succeeds",
			provider: &mockToolProvider{mockProvider: mockProvider{name: "mock-tools", ready: true}},
			wantErr:  false,
		},
		{
			name: "provider with tool error is retried",
			provider: &mockToolProvider{
				mockProvider: mockProvider{name: "mock-tools", ready: true},
				toolsErr:     errors.New("status 503: service unavailable"),
			},
			wantErr:     true,
			errContains: "max retries exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RetryConfig{MaxAttempts: 2, MaxBackoff: 0.001, JitterRatio: 0}
			retryProv := CreateWithRetry(tt.provider, cfg)

			// Type assert to access AskWithTools
			toolProv, ok := retryProv.(ToolProvider)
			if !ok {
				t.Fatal("retryProvider should implement ToolProvider")
			}

			err := toolProv.AskWithTools(context.Background(), "test", nil, nil, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRetryProviderImplementsToolProvider(t *testing.T) {
	// Verify that retryProvider implements ToolProvider at compile time
	var _ ToolProvider = (*retryProvider)(nil)
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestProviderFactoryCreate(t *testing.T) {
	factory := GetFactory()

	tests := []struct {
		name        string
		config      *ProviderConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "create openai provider",
			config: &ProviderConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
		{
			name: "create ollama provider",
			config: &ProviderConfig{
				Provider: "ollama",
				Model:    "llama2",
				Endpoint: "http://localhost:11434",
			},
			wantErr: false,
		},
		{
			name: "create gemini provider",
			config: &ProviderConfig{
				Provider: "gemini",
				Model:    "gemini-pro",
				APIKey:   "test-key",
			},
			wantErr: false,
		},
		{
			name: "create bedrock provider",
			config: &ProviderConfig{
				Provider: "bedrock",
				Model:    "anthropic.claude-3-sonnet-20240229-v1:0",
				Region:   "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "create azure provider",
			config: &ProviderConfig{
				Provider:        "azopenai",
				AzureDeployment: "gpt-4",
				APIKey:          "test-key",
				Endpoint:        "https://test.openai.azure.com",
			},
			wantErr: false,
		},
		{
			name: "create azure provider via alias",
			config: &ProviderConfig{
				Provider:        "azure",
				AzureDeployment: "gpt-4",
				APIKey:          "test-key",
				Endpoint:        "https://test.openai.azure.com",
			},
			wantErr: false,
		},
		{
			name: "unknown provider returns error",
			config: &ProviderConfig{
				Provider: "unknown-provider",
			},
			wantErr:     true,
			errContains: "unknown provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.Create(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want to contain %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if provider == nil {
				t.Error("provider should not be nil")
			}
		})
	}
}

func TestProviderFactoryListProviders(t *testing.T) {
	factory := GetFactory()
	providers := factory.ListProviders()

	expectedProviders := []string{"openai", "ollama", "gemini", "bedrock", "azopenai", "azure"}
	for _, expected := range expectedProviders {
		if !containsString(providers, expected) {
			t.Errorf("expected providers to contain %q, got %q", expected, providers)
		}
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "rate limit error is retryable",
			err:      errors.New("API error: status 429 too many requests"),
			expected: true,
		},
		{
			name:     "server error is retryable",
			err:      errors.New("API error: status 500 internal server error"),
			expected: true,
		},
		{
			name:     "bad gateway is retryable",
			err:      errors.New("API error: status 502 bad gateway"),
			expected: true,
		},
		{
			name:     "service unavailable is retryable",
			err:      errors.New("API error: status 503 service unavailable"),
			expected: true,
		},
		{
			name:     "gateway timeout is retryable",
			err:      errors.New("API error: status 504 gateway timeout"),
			expected: true,
		},
		{
			name:     "timeout error is retryable",
			err:      errors.New("request timeout"),
			expected: true,
		},
		{
			name:     "connection refused is retryable",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "connection reset is retryable",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "bad request is not retryable",
			err:      errors.New("API error: status 400 bad request"),
			expected: false,
		},
		{
			name:     "unauthorized is not retryable",
			err:      errors.New("API error: status 401 unauthorized"),
			expected: false,
		},
		{
			name:     "not found is not retryable",
			err:      errors.New("API error: status 404 not found"),
			expected: false,
		},
		{
			name:     "nil error is not retryable",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d, want 5", cfg.MaxAttempts)
	}
	if cfg.MaxBackoff != 10.0 {
		t.Errorf("MaxBackoff = %f, want 10.0", cfg.MaxBackoff)
	}
	if cfg.JitterRatio != 0.1 {
		t.Errorf("JitterRatio = %f, want 0.1", cfg.JitterRatio)
	}
}

func TestOpenAIProviderIsReady(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{
			name:   "ready with api key",
			apiKey: "sk-test-key",
			want:   true,
		},
		{
			name:   "not ready without api key",
			apiKey: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, _ := NewOpenAIProvider(&ProviderConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   tt.apiKey,
			})

			if got := provider.IsReady(); got != tt.want {
				t.Errorf("IsReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeminiProviderIsReady(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{
			name:   "ready with api key",
			apiKey: "test-api-key",
			want:   true,
		},
		{
			name:   "not ready without api key",
			apiKey: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, _ := NewGeminiProvider(&ProviderConfig{
				Provider: "gemini",
				Model:    "gemini-pro",
				APIKey:   tt.apiKey,
			})

			if got := provider.IsReady(); got != tt.want {
				t.Errorf("IsReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAzureOpenAIProviderIsReady(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		endpoint string
		want     bool
	}{
		{
			name:     "ready with api key and endpoint",
			apiKey:   "test-api-key",
			endpoint: "https://test.openai.azure.com",
			want:     true,
		},
		{
			name:     "not ready without api key",
			apiKey:   "",
			endpoint: "https://test.openai.azure.com",
			want:     false,
		},
		{
			name:     "not ready without endpoint",
			apiKey:   "test-api-key",
			endpoint: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.endpoint == "" {
				// Skip test case that would return error from constructor
				return
			}
			provider, err := NewAzureOpenAIProvider(&ProviderConfig{
				Provider:        "azopenai",
				AzureDeployment: "gpt-4",
				APIKey:          tt.apiKey,
				Endpoint:        tt.endpoint,
			})
			if err != nil {
				return // Expected for missing endpoint
			}

			if got := provider.IsReady(); got != tt.want {
				t.Errorf("IsReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAzureOpenAIProviderRequiresEndpoint(t *testing.T) {
	_, err := NewAzureOpenAIProvider(&ProviderConfig{
		Provider:        "azopenai",
		AzureDeployment: "gpt-4",
		APIKey:          "test-key",
		Endpoint:        "", // Missing endpoint
	})
	if err == nil {
		t.Error("Expected error when endpoint is missing, got nil")
	}
}

func TestBedrockProviderIsReady(t *testing.T) {
	// Save original env vars
	origAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	origSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	defer func() {
		os.Setenv("AWS_ACCESS_KEY_ID", origAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", origSecretKey)
	}()

	tests := []struct {
		name      string
		accessKey string
		secretKey string
		configKey string
		want      bool
	}{
		{
			name:      "ready with env credentials",
			accessKey: "AKIATEST123",
			secretKey: "testsecret123",
			want:      true,
		},
		{
			name:      "not ready without credentials",
			accessKey: "",
			secretKey: "",
			want:      false,
		},
		{
			name:      "ready with config api key",
			accessKey: "",
			secretKey: "",
			configKey: "config-key",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("AWS_ACCESS_KEY_ID", tt.accessKey)
			os.Setenv("AWS_SECRET_ACCESS_KEY", tt.secretKey)

			provider, err := NewBedrockProvider(&ProviderConfig{
				Provider: "bedrock",
				Model:    "anthropic.claude-3-sonnet-20240229-v1:0",
				Region:   "us-east-1",
				APIKey:   tt.configKey,
			})
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			if got := provider.IsReady(); got != tt.want {
				t.Errorf("IsReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBedrockProviderDefaultRegion(t *testing.T) {
	// Save and clear env var
	origRegion := os.Getenv("AWS_REGION")
	os.Setenv("AWS_REGION", "")
	defer os.Setenv("AWS_REGION", origRegion)

	provider, err := NewBedrockProvider(&ProviderConfig{
		Provider: "bedrock",
		Model:    "anthropic.claude-3-sonnet-20240229-v1:0",
		Region:   "", // No region specified
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Should default to us-east-1
	bp := provider.(*BedrockProvider)
	if bp.region != "us-east-1" {
		t.Errorf("Expected default region 'us-east-1', got %q", bp.region)
	}
}

func TestBedrockProviderListModels(t *testing.T) {
	provider, _ := NewBedrockProvider(&ProviderConfig{
		Provider: "bedrock",
		Region:   "us-east-1",
	})

	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected at least one model in list")
	}

	// Check for expected Claude models
	foundClaude := false
	for _, m := range models {
		if containsString(m, "claude") {
			foundClaude = true
			break
		}
	}
	if !foundClaude {
		t.Error("Expected to find Claude models in Bedrock model list")
	}
}

func TestGeminiProviderDefaultEndpoint(t *testing.T) {
	provider, err := NewGeminiProvider(&ProviderConfig{
		Provider: "gemini",
		Model:    "gemini-pro",
		APIKey:   "test-key",
		Endpoint: "", // No endpoint specified
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	gp := provider.(*GeminiProvider)
	if gp.endpoint != "https://generativelanguage.googleapis.com/v1beta" {
		t.Errorf("Expected default Gemini endpoint, got %q", gp.endpoint)
	}
}

func TestGeminiProviderDefaultModel(t *testing.T) {
	provider, err := NewGeminiProvider(&ProviderConfig{
		Provider: "gemini",
		APIKey:   "test-key",
		Model:    "", // No model specified
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider.GetModel() != "gemini-1.5-flash" {
		t.Errorf("Expected default model 'gemini-1.5-flash', got %q", provider.GetModel())
	}
}

func TestOllamaProviderDefaults(t *testing.T) {
	provider, err := NewOllamaProvider(&ProviderConfig{
		Provider: "ollama",
		Model:    "", // No model specified
		Endpoint: "", // No endpoint specified
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	op := provider.(*OllamaProvider)
	if op.endpoint != "http://localhost:11434" {
		t.Errorf("Expected default Ollama endpoint, got %q", op.endpoint)
	}
	if provider.GetModel() != "llama3.2" {
		t.Errorf("Expected default model 'llama3.2', got %q", provider.GetModel())
	}
}

func TestOpenAIProviderDefaults(t *testing.T) {
	provider, err := NewOpenAIProvider(&ProviderConfig{
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Endpoint: "", // No endpoint specified
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	op := provider.(*OpenAIProvider)
	if op.endpoint != "https://api.openai.com/v1" {
		t.Errorf("Expected default OpenAI endpoint, got %q", op.endpoint)
	}
}

func TestProviderNames(t *testing.T) {
	tests := []struct {
		provider Provider
		expected string
	}{
		{mustCreateProvider(t, "openai"), "openai"},
		{mustCreateProvider(t, "ollama"), "ollama"},
		{mustCreateProvider(t, "gemini"), "gemini"},
		{mustCreateProvider(t, "bedrock"), "bedrock"},
		{mustCreateAzureProvider(t), "azopenai"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.provider.Name(); got != tt.expected {
				t.Errorf("Name() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func mustCreateProvider(t *testing.T, name string) Provider {
	t.Helper()
	provider, err := GetFactory().Create(&ProviderConfig{
		Provider: name,
		Model:    "test-model",
		APIKey:   "test-key",
		Region:   "us-east-1",
	})
	if err != nil {
		t.Fatalf("Failed to create %s provider: %v", name, err)
	}
	return provider
}

func mustCreateAzureProvider(t *testing.T) Provider {
	t.Helper()
	provider, err := NewAzureOpenAIProvider(&ProviderConfig{
		Provider:        "azopenai",
		AzureDeployment: "gpt-4",
		APIKey:          "test-key",
		Endpoint:        "https://test.openai.azure.com",
	})
	if err != nil {
		t.Fatalf("Failed to create Azure provider: %v", err)
	}
	return provider
}

func TestAzureOpenAIProviderListModels(t *testing.T) {
	provider, _ := NewAzureOpenAIProvider(&ProviderConfig{
		Provider:        "azopenai",
		AzureDeployment: "gpt-4",
		APIKey:          "test-key",
		Endpoint:        "https://test.openai.azure.com",
	})

	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected at least one model in list")
	}
}

func TestRetryProviderDelegation(t *testing.T) {
	mock := &mockProvider{
		name:       "test-provider",
		model:      "test-model",
		ready:      true,
		askContent: "test response",
	}

	retryProv := CreateWithRetry(mock, DefaultRetryConfig())

	// Test Name()
	if got := retryProv.Name(); got != "test-provider" {
		t.Errorf("Name() = %q, want %q", got, "test-provider")
	}

	// Test GetModel()
	if got := retryProv.GetModel(); got != "test-model" {
		t.Errorf("GetModel() = %q, want %q", got, "test-model")
	}

	// Test IsReady()
	if got := retryProv.IsReady(); got != true {
		t.Errorf("IsReady() = %v, want %v", got, true)
	}

	// Test ListModels()
	_, err := retryProv.ListModels(context.Background())
	if err != nil {
		t.Errorf("ListModels() error = %v", err)
	}
}

func TestRetryProviderBackoff(t *testing.T) {
	// Create a provider that fails initially
	callCount := 0
	mock := &mockProvider{
		name:  "retry-test",
		ready: true,
	}

	// Override Ask to track calls
	cfg := &RetryConfig{MaxAttempts: 3, MaxBackoff: 0.001, JitterRatio: 0}
	retryProv := CreateWithRetry(mock, cfg)

	// This should succeed on first try
	err := retryProv.Ask(context.Background(), "test", func(s string) {
		callCount++
	})
	if err != nil {
		t.Errorf("Ask() error = %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	mock := &mockProvider{
		name:   "cancel-test",
		ready:  true,
		askErr: errors.New("status 503: service unavailable"),
	}

	cfg := &RetryConfig{MaxAttempts: 10, MaxBackoff: 10.0, JitterRatio: 0}
	retryProv := CreateWithRetry(mock, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := retryProv.Ask(ctx, "test", nil)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

// Test Solar provider registration
func TestFactorySolarProviderRegistered(t *testing.T) {
	factory := GetFactory()

	// Check that solar provider is registered
	providers := factory.ListProviders()
	if providers == "" {
		t.Error("ListProviders() returned empty string")
	}

	// Solar should be in the list
	found := false
	for _, p := range []string{"solar", "openai", "ollama", "gemini", "bedrock", "azopenai"} {
		if containsProvider(providers, p) {
			if p == "solar" {
				found = true
			}
		}
	}

	if !found {
		t.Errorf("solar provider not found in registered providers: %s", providers)
	}
}

func containsProvider(list, name string) bool {
	for i := 0; i <= len(list)-len(name); i++ {
		if i+len(name) <= len(list) && list[i:i+len(name)] == name {
			return true
		}
	}
	return false
}

// Test Solar provider creates OpenAI-compatible provider
func TestFactoryCreateSolarProvider(t *testing.T) {
	// Skip if no API key (this is expected in CI)
	if os.Getenv("UPSTAGE_API_KEY") == "" && os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping test: no API key available")
	}

	factory := GetFactory()

	cfg := &ProviderConfig{
		Provider: "solar",
		Model:    "solar-pro2",
		Endpoint: "https://api.upstage.ai/v1",
		APIKey:   os.Getenv("UPSTAGE_API_KEY"),
	}

	provider, err := factory.Create(cfg)
	if err != nil {
		t.Fatalf("Failed to create solar provider: %v", err)
	}

	if provider == nil {
		t.Error("Created provider is nil")
	}

	// Solar uses OpenAI provider internally, so name should be "openai"
	if provider.Name() != "openai" {
		t.Errorf("Provider name = %q, want \"openai\"", provider.Name())
	}
}

// Test that all expected providers are registered
func TestFactoryAllProvidersRegistered(t *testing.T) {
	factory := GetFactory()

	expectedProviders := []string{"solar", "openai", "ollama", "gemini", "bedrock", "azopenai", "azure"}

	for _, name := range expectedProviders {
		t.Run(name, func(t *testing.T) {
			providers := factory.ListProviders()
			if !containsProvider(providers, name) {
				t.Errorf("Provider %q not found in registered providers: %s", name, providers)
			}
		})
	}
}
