package ui

import (
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/safety"
	aitools "github.com/cloudbro-kube-ai/k13d/pkg/ai/tools"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

const tuiBashKubernetesBypassReason = "Use the dedicated kubectl tool instead of bash for Kubernetes operations"

func normalizeAIToolCommand(toolName, command string) string {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return ""
	}

	if toolName == "kubectl" && !strings.HasPrefix(trimmed, "kubectl ") {
		return "kubectl " + trimmed
	}

	return trimmed
}

func shouldBlockBashForKubernetes(command string) bool {
	parsed := safety.ParseCommand(strings.TrimSpace(command))
	return parsed.Program == "kubectl" || parsed.Program == "helm"
}

func effectiveUIToolApprovalPolicy(policy config.ToolApprovalPolicy) config.ToolApprovalPolicy {
	defaults := config.DefaultToolApprovalPolicy()
	if !policy.AutoApproveReadOnly &&
		!policy.RequireApprovalForWrite &&
		!policy.RequireApprovalForUnknown &&
		!policy.BlockDangerous &&
		policy.ApprovalTimeoutSeconds == 0 &&
		len(policy.BlockedPatterns) == 0 {
		return defaults
	}

	if policy.ApprovalTimeoutSeconds <= 0 {
		policy.ApprovalTimeoutSeconds = defaults.ApprovalTimeoutSeconds
	}
	if policy.BlockedPatterns == nil {
		policy.BlockedPatterns = []string{}
	}
	return policy
}

func appendWarningIfMissing(warnings []string, want string) []string {
	for _, warning := range warnings {
		if warning == want {
			return warnings
		}
	}
	return append(warnings, want)
}

func (a *App) currentToolApprovalPolicy() config.ToolApprovalPolicy {
	if a == nil || a.config == nil {
		return config.DefaultToolApprovalPolicy()
	}
	return effectiveUIToolApprovalPolicy(a.config.Authorization.ToolApproval)
}

func (a *App) evaluateAIToolDecision(toolName, command string) *safety.Decision {
	normalizedCommand := normalizeAIToolCommand(toolName, command)
	decision := safety.NewPolicyEnforcer(a.currentToolApprovalPolicy()).Evaluate(normalizedCommand)

	switch toolName {
	case "kubectl":
		if err := aitools.ValidateKubectlToolCommand(normalizedCommand); err != nil {
			decision.Allowed = false
			decision.RequiresApproval = false
			decision.BlockReason = err.Error()
			decision.Warnings = appendWarningIfMissing(decision.Warnings, "This kubectl command requires an interactive terminal or unsupported workflow.")
			return decision
		}
	case "bash":
		if err := aitools.ValidateBashToolCommand(normalizedCommand); err != nil {
			decision.Allowed = false
			decision.RequiresApproval = false
			decision.BlockReason = err.Error()
			decision.Warnings = appendWarningIfMissing(decision.Warnings, "This request should not be executed through bash.")
			return decision
		}
	}

	if toolName == "bash" {
		decision.RequiresApproval = true
		decision.Warnings = appendWarningIfMissing(decision.Warnings, "Bash is discouraged in k13d AI Assistant. Prefer kubectl whenever possible.")
		if shouldBlockBashForKubernetes(normalizedCommand) {
			decision.Allowed = false
			decision.BlockReason = tuiBashKubernetesBypassReason
			decision.Warnings = appendWarningIfMissing(decision.Warnings, "This request should be handled through the kubectl tool, not bash.")
		}
	}

	return decision
}

func (a *App) getToolApprovalTimeout() time.Duration {
	seconds := a.currentToolApprovalPolicy().ApprovalTimeoutSeconds
	if seconds <= 0 {
		seconds = config.DefaultToolApprovalPolicy().ApprovalTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}
