package web

import (
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func createUsableAIClient(cfg *config.LLMConfig) (*ai.Client, bool, error) {
	if cfg == nil || strings.TrimSpace(cfg.Provider) == "" || strings.TrimSpace(cfg.Model) == "" {
		return nil, false, nil
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		return nil, false, err
	}

	return client, client.IsReady(), nil
}
