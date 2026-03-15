package ai

import (
	"fmt"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/tools"
)

const (
	defaultAgenticMaxIterations = 10
	minAgenticIterations        = 2
	maxAgenticIterations        = 30
)

func normalizedAgenticIterations(maxIterations int) int {
	if maxIterations <= 0 {
		return defaultAgenticMaxIterations
	}
	if maxIterations < minAgenticIterations {
		return minAgenticIterations
	}
	if maxIterations > maxAgenticIterations {
		return maxAgenticIterations
	}
	return maxIterations
}

// buildAgenticPrompt adds execution guidance inspired by kubectl-ai's agent loop.
func buildAgenticPrompt(prompt string, registry *tools.Registry, maxIterations int) string {
	var sb strings.Builder

	sb.WriteString("Agent execution guidance:\n")
	sb.WriteString(fmt.Sprintf("- Use the live cluster state and available tools instead of guessing. You have a budget of at most %d tool-use rounds for this turn.\n", normalizedAgenticIterations(maxIterations)))
	sb.WriteString("- Consider a few plausible explanations or next steps before choosing the next tool.\n")
	sb.WriteString("- After each tool result, reassess whether you already have enough evidence for a final answer.\n")
	sb.WriteString("- Prefer autonomous progress and non-interactive commands that can complete in one step.\n")
	sb.WriteString("- Use tools instead of only suggesting commands.\n")
	sb.WriteString("- Always format kubectl commands as 'kubectl <verb> ...'; never place flags before the kubectl verb.\n")

	if registry != nil {
		toolList := registry.List()
		if len(toolList) > 0 {
			names := make([]string, 0, len(toolList))
			for _, tool := range toolList {
				names = append(names, tool.Name)
			}
			sb.WriteString("- Available tools this turn: " + strings.Join(names, ", ") + ".\n")
		}
		if len(registry.GetMCPTools()) > 0 {
			sb.WriteString("- " + strings.TrimSpace(tools.ToolNameInstruction) + "\n")
		}
	}

	sb.WriteString("\nUser request:\n")
	sb.WriteString(prompt)
	return sb.String()
}
