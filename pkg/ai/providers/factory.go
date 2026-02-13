package providers

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ProviderFactory creates LLM providers based on configuration
type ProviderFactory struct {
	mu        sync.RWMutex
	providers map[string]func(*ProviderConfig) (Provider, error)
}

var (
	defaultFactory *ProviderFactory
	factoryOnce    sync.Once
)

// GetFactory returns the singleton provider factory
func GetFactory() *ProviderFactory {
	factoryOnce.Do(func() {
		defaultFactory = &ProviderFactory{
			providers: make(map[string]func(*ProviderConfig) (Provider, error)),
		}
		// Register built-in providers
		defaultFactory.Register("solar", NewOpenAIProvider) // Upstage Solar (OpenAI-compatible)
		defaultFactory.Register("openai", NewOpenAIProvider)
		defaultFactory.Register("ollama", NewOllamaProvider)
		defaultFactory.Register("gemini", NewGeminiProvider)
		defaultFactory.Register("bedrock", NewBedrockProvider)
		defaultFactory.Register("azopenai", NewAzureOpenAIProvider)
		defaultFactory.Register("azure", NewAzureOpenAIProvider) // alias
		defaultFactory.Register("embedded", NewEmbeddedProvider) // Local llama.cpp server
	})
	return defaultFactory
}

// Register adds a new provider constructor to the factory
func (f *ProviderFactory) Register(name string, constructor func(*ProviderConfig) (Provider, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.providers[strings.ToLower(name)] = constructor
}

// Create creates a provider instance based on the configuration
func (f *ProviderFactory) Create(cfg *ProviderConfig) (Provider, error) {
	f.mu.RLock()
	constructor, ok := f.providers[strings.ToLower(cfg.Provider)]
	f.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (available: %s)", cfg.Provider, f.ListProviders())
	}

	return constructor(cfg)
}

// ListProviders returns a comma-separated list of available provider names
func (f *ProviderFactory) ListProviders() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.providers))
	for name := range f.providers {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// CreateWithRetry wraps a provider with retry logic
func CreateWithRetry(provider Provider, cfg *RetryConfig) Provider {
	if cfg == nil {
		cfg = DefaultRetryConfig()
	}
	return &retryProvider{
		provider: provider,
		config:   cfg,
	}
}

// retryProvider wraps a provider with retry logic
type retryProvider struct {
	provider Provider
	config   *RetryConfig
}

func (r *retryProvider) Name() string {
	return r.provider.Name()
}

func (r *retryProvider) GetModel() string {
	return r.provider.GetModel()
}

func (r *retryProvider) IsReady() bool {
	return r.provider.IsReady()
}

func (r *retryProvider) ListModels(ctx context.Context) ([]string, error) {
	return r.provider.ListModels(ctx)
}

func (r *retryProvider) Ask(ctx context.Context, prompt string, callback func(string)) error {
	var lastErr error
	for attempt := 0; attempt < r.config.MaxAttempts; attempt++ {
		if attempt > 0 {
			backoff := r.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := r.provider.Ask(ctx, prompt, callback)
		if err == nil {
			return nil
		}

		if !isRetryableError(err) {
			return err
		}

		lastErr = err
	}
	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (r *retryProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < r.config.MaxAttempts; attempt++ {
		if attempt > 0 {
			backoff := r.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		response, err := r.provider.AskNonStreaming(ctx, prompt)
		if err == nil {
			return response, nil
		}

		if !isRetryableError(err) {
			return "", err
		}

		lastErr = err
	}
	return "", fmt.Errorf("max retries exceeded: %w", lastErr)
}

// AskWithTools implements ToolProvider interface by delegating to the underlying provider
// if it supports tool calling. This ensures the retry wrapper doesn't break tool support.
func (r *retryProvider) AskWithTools(ctx context.Context, prompt string, tools []ToolDefinition, callback func(string), toolCallback ToolCallback) error {
	toolProvider, ok := r.provider.(ToolProvider)
	if !ok {
		return fmt.Errorf("underlying provider does not support tool calling")
	}

	var lastErr error
	for attempt := 0; attempt < r.config.MaxAttempts; attempt++ {
		if attempt > 0 {
			backoff := r.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := toolProvider.AskWithTools(ctx, prompt, tools, callback, toolCallback)
		if err == nil {
			return nil
		}

		if !isRetryableError(err) {
			return err
		}

		lastErr = err
	}
	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// SupportsTools returns true if the underlying provider supports tool calling
func (r *retryProvider) SupportsTools() bool {
	_, ok := r.provider.(ToolProvider)
	return ok
}

func (r *retryProvider) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: 2^attempt seconds, capped at maxBackoff
	backoff := math.Pow(2, float64(attempt))
	if backoff > r.config.MaxBackoff {
		backoff = r.config.MaxBackoff
	}

	// Add jitter
	if r.config.JitterRatio > 0 {
		jitter := backoff * r.config.JitterRatio * (rand.Float64()*2 - 1)
		backoff += jitter
	}

	return time.Duration(backoff * float64(time.Second))
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// HTTP status codes that are retryable
	retryablePatterns := []string{
		"status 429", // Rate limit
		"status 500", // Internal server error
		"status 502", // Bad gateway
		"status 503", // Service unavailable
		"status 504", // Gateway timeout
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"try again",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// newHTTPClient creates an HTTP client with optional TLS skip
func newHTTPClient(skipTLS bool) *http.Client {
	transport := &http.Transport{}
	if skipTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}
