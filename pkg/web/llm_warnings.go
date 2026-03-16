package web

import (
	"fmt"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func ollamaToolSupportWarning(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Sprintf("k13d requires an Ollama model with tools/function calling support. Text-only Ollama models may connect, but the AI Assistant will not work correctly. Recommended: %s or another Ollama model whose card explicitly lists tools support.", config.DefaultOllamaModel)
	}

	return fmt.Sprintf("Ollama model %q must support tools/function calling. Text-only Ollama models may connect, but the AI Assistant will not work correctly in k13d. Recommended: %s or another Ollama model whose card explicitly lists tools support.", model, config.DefaultOllamaModel)
}

func modelRegistrationWarning(provider, model string) string {
	if strings.TrimSpace(provider) != "ollama" {
		return ""
	}
	return ollamaToolSupportWarning(model)
}
