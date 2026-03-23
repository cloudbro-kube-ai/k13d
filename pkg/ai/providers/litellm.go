package providers

const (
	defaultLiteLLMEndpoint = "http://localhost:4000"
	defaultLiteLLMAPIKey   = "litellm"
)

// LiteLLMProvider wraps the OpenAI-compatible provider so the runtime can
// distinguish a LiteLLM gateway from direct OpenAI usage.
type LiteLLMProvider struct {
	*OpenAIProvider
}

// NewLiteLLMProvider creates an OpenAI-compatible provider targeting a LiteLLM
// proxy. LiteLLM commonly fronts multiple upstream models behind a single
// OpenAI-style endpoint, so this mode is intended for gradual migration rather
// than replacing direct providers all at once.
func NewLiteLLMProvider(cfg *ProviderConfig) (Provider, error) {
	clone := &ProviderConfig{}
	if cfg != nil {
		copied := *cfg
		clone = &copied
	}
	clone.Provider = "litellm"
	if clone.Endpoint == "" {
		clone.Endpoint = defaultLiteLLMEndpoint
	}
	if clone.APIKey == "" {
		clone.APIKey = defaultLiteLLMAPIKey
	}

	base, err := NewOpenAIProvider(clone)
	if err != nil {
		return nil, err
	}

	return &LiteLLMProvider{OpenAIProvider: base.(*OpenAIProvider)}, nil
}

func (p *LiteLLMProvider) Name() string {
	return "litellm"
}
