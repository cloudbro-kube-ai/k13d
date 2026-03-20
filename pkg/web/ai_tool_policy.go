package web

import (
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/safety"
	aitools "github.com/cloudbro-kube-ai/k13d/pkg/ai/tools"
)

const bashKubernetesBypassReason = "Use the dedicated kubectl tool instead of bash for Kubernetes operations"

func normalizeAIToolCommand(toolName, command string) string {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return ""
	}

	switch toolName {
	case "kubectl":
		if strings.HasPrefix(trimmed, "kubectl ") {
			return trimmed
		}
		return "kubectl " + trimmed
	default:
		return trimmed
	}
}

func shouldBlockBashForKubernetes(command string) bool {
	parsed := safety.ParseCommand(strings.TrimSpace(command))
	return parsed.Program == "kubectl" || parsed.Program == "helm"
}

func (s *Server) evaluateAIToolDecision(role, toolName, command string) *safety.Decision {
	normalizedCommand := normalizeAIToolCommand(toolName, command)

	if !isAllowedAIToolExecution(role, toolName, normalizedCommand) {
		category := "restricted"
		if normalizedCommand != "" {
			category = classifyCommand(normalizedCommand)
		}
		return &safety.Decision{
			Allowed:     false,
			Category:    category,
			BlockReason: toolExecutionDeniedReason(role, toolName, normalizedCommand),
			Warnings:    []string{"This session is running in read-only AI mode."},
		}
	}

	decision := s.getToolApprovalDecision(normalizedCommand)
	decision.Category = classifyCategoryForTool(toolName, normalizedCommand, decision.Category)

	switch toolName {
	case "kubectl":
		if err := aitools.ValidateKubectlToolCommand(normalizedCommand); err != nil {
			decision.Allowed = false
			decision.RequiresApproval = false
			decision.BlockReason = err.Error()
			decision.Warnings = appendIfMissing(decision.Warnings, "This kubectl command requires an interactive terminal or unsupported workflow.")
			return decision
		}
	case "bash":
		if err := aitools.ValidateBashToolCommand(normalizedCommand); err != nil {
			decision.Allowed = false
			decision.RequiresApproval = false
			decision.BlockReason = err.Error()
			decision.Warnings = appendIfMissing(decision.Warnings, "This request should not be executed through bash.")
			return decision
		}
	}

	if toolName == "bash" {
		decision.RequiresApproval = true
		decision.Warnings = appendIfMissing(decision.Warnings, "Bash is discouraged in k13d AI Assistant. Prefer kubectl whenever possible.")
		if shouldBlockBashForKubernetes(normalizedCommand) {
			decision.Allowed = false
			decision.BlockReason = bashKubernetesBypassReason
			decision.Warnings = appendIfMissing(decision.Warnings, "This request should be handled through the kubectl tool, not bash.")
		}
	}

	return decision
}

func classifyCategoryForTool(toolName, command, fallback string) string {
	if toolName == "bash" {
		parsed := safety.ParseCommand(command)
		if parsed.Program == "kubectl" || parsed.Program == "helm" {
			return classifyCommand(command)
		}
		return "bash"
	}
	return fallback
}

func appendIfMissing(items []string, want string) []string {
	for _, item := range items {
		if item == want {
			return items
		}
	}
	return append(items, want)
}
