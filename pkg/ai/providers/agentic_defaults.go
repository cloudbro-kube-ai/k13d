package providers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	defaultToolLoopIterations = 10
	minimumToolLoopIterations = 2
	maximumToolLoopIterations = 30
)

func effectiveMaxIterations(cfg *ProviderConfig) int {
	if cfg == nil || cfg.MaxIterations <= 0 {
		return defaultToolLoopIterations
	}
	if cfg.MaxIterations < minimumToolLoopIterations {
		return minimumToolLoopIterations
	}
	if cfg.MaxIterations > maximumToolLoopIterations {
		return maximumToolLoopIterations
	}
	return cfg.MaxIterations
}

func toolAgentSystemPrompt(maxIterations int) string {
	return fmt.Sprintf(`You are k13d, a Kubernetes AI assistant with DIRECT ACCESS to kubectl, bash, and optional MCP tools.

Core behavior:
- Work from live cluster evidence rather than guesses.
- For Kubernetes questions, inspect the cluster with tools before answering.
- Prefer autonomous progress: gather the next missing evidence yourself when safe.
- Use tools instead of only suggesting commands.
- Prefer non-interactive commands that can complete in one step.
- After each tool result, reassess whether you already have enough evidence for a final answer.
- You have a budget of at most %d tool-use rounds for this turn.

Command rules:
- Always write kubectl commands as 'kubectl <verb> ...'.
- Never place flags before the kubectl verb.
- Use only exact tool names from the function schema. Never invent or abbreviate tool names.

Final answer:
- Be concise, evidence-based, and action-oriented.
- Summarize findings, cite the key evidence, and suggest next steps only when needed.`, maxIterations)
}

func buildToolUseShimSystemPrompt(tools []ToolDefinition, maxIterations int) string {
	return fmt.Sprintf(`You are k13d, a Kubernetes AI assistant with DIRECT ACCESS to kubectl, bash, and optional MCP tools.

## Available tools
<tools>
%s
</tools>

## Instructions
1. Analyze the user request and the latest observations from prior tool runs.
2. Reflect on a few plausible ways to solve the task before choosing the next action.
3. You have a budget of at most %d tool-use rounds for this turn. Stop as soon as you have enough evidence.
4. Prefer autonomous progress and non-interactive commands.
5. Always format kubectl commands as 'kubectl <verb> ...' and never place flags before the verb.
6. Use only exact tool names from this list: %s.
7. Respond ONLY with a JSON code block in one of the following formats.

If you need a tool:
`+"```json"+`
{
  "thought": "Short reasoning about what to do next",
  "action": {
    "name": "One of the available tool names",
    "reason": "Why this tool is the best next step",
    "command": "Complete command to execute",
    "modifies_resource": "yes | no | unknown"
  }
}
`+"```"+`

If you have enough information:
`+"```json"+`
{
  "thought": "Short reasoning about the final answer",
  "answer": "Concise, evidence-based answer with next steps only if needed"
}
`+"```", toolDefinitionsAsJSON(tools), maxIterations, strings.Join(toolDefinitionNames(tools), ", "))
}

func sortedToolDefinitions(defs []ToolDefinition) []ToolDefinition {
	sorted := append([]ToolDefinition(nil), defs...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Function.Name < sorted[j].Function.Name
	})
	return sorted
}

func toolDefinitionNames(defs []ToolDefinition) []string {
	sorted := sortedToolDefinitions(defs)
	names := make([]string, 0, len(sorted))
	for _, def := range sorted {
		names = append(names, def.Function.Name)
	}
	return names
}

func toolDefinitionsAsJSON(defs []ToolDefinition) string {
	sorted := sortedToolDefinitions(defs)
	data, err := json.MarshalIndent(sorted, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(data)
}

const finalToolSummaryPrompt = "Based on the tool execution results above, provide a concise final answer with findings, supporting evidence, and next steps only if needed."
