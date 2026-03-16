package ui

import (
	"fmt"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func ollamaModelToolsHint(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return fmt.Sprintf("k13d requires an Ollama model with tools/function calling support. Text-only Ollama models may connect, but the AI Assistant will not work correctly. Recommended: %s.", config.DefaultOllamaModel)
	}

	return fmt.Sprintf("Ollama model %q must support tools/function calling. Text-only Ollama models may connect, but the AI Assistant will not work correctly. Recommended: %s.", model, config.DefaultOllamaModel)
}

func buildLLMInfoText(provider, model, endpoint string, hasAPIKey bool) string {
	infoText := fmt.Sprintf(` [cyan::b]LLM Configuration[white::-]
 Provider: [yellow]%s[white]  Model: [yellow]%s[white]
API Key: %s  Endpoint: %s
`,
		provider, model,
		map[bool]string{true: "[green]Set[white]", false: "[red]Not set[white]"}[hasAPIKey],
		map[bool]string{true: "[green]" + endpoint + "[white]", false: "[gray](default)[white]"}[endpoint != ""])

	if provider == "ollama" {
		infoText += fmt.Sprintf(" [yellow]Tools Required:[white] %s\n", ollamaModelToolsHint(model))
	}

	return infoText
}

func buildToolApprovalInfoText(policy config.ToolApprovalPolicy) string {
	policy = effectiveUIToolApprovalPolicy(policy)

	return fmt.Sprintf(` [cyan::b]Tool Approval Policy[white::-]
 Read-only auto-approve: %s
 Write approval required: %s
 Dangerous commands blocked: %s
 Unknown approval required: %s
 Approval timeout: [yellow]%ds[white]
`,
		map[bool]string{true: "[green]On[white]", false: "[red]Off[white]"}[policy.AutoApproveReadOnly],
		map[bool]string{true: "[green]On[white]", false: "[red]Off[white]"}[policy.RequireApprovalForWrite],
		map[bool]string{true: "[green]On[white]", false: "[red]Off[white]"}[policy.BlockDangerous],
		map[bool]string{true: "[green]On[white]", false: "[red]Off[white]"}[policy.RequireApprovalForUnknown],
		policy.ApprovalTimeoutSeconds,
	)
}
