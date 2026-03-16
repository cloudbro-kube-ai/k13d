package web

import (
	"fmt"
	"strings"
)

func buildAIRestrictionPrompt(role string) string {
	if role == "admin" || role == "user" {
		return ""
	}

	return "IMPORTANT: The authenticated user is in read-only AI mode. " +
		"You may ONLY use the kubectl tool for read-only commands such as get, describe, and logs. " +
		"Never use bash or MCP tools. Never use exec, attach, cp, port-forward, edit, apply, create, delete, patch, scale, restart, drain, cordon, or any other mutating or interactive command. " +
		"If a task needs elevated access, explain the limitation and provide the exact command for a privileged user to run manually."
}

func allowAIToolExecution(role, toolName, command string) (bool, string) {
	if isAllowedAIToolExecution(role, toolName, command) {
		return true, ""
	}
	return false, toolExecutionDeniedReason(role, toolName, command)
}

func isAllowedAIToolExecution(role, toolName, command string) bool {
	if role == "" {
		role = "viewer"
	}

	if role == "admin" || role == "user" {
		return true
	}

	if toolName != "kubectl" {
		return false
	}

	normalizedCommand := strings.TrimSpace(command)
	if normalizedCommand != "" && !strings.HasPrefix(normalizedCommand, "kubectl ") {
		normalizedCommand = "kubectl " + normalizedCommand
	}

	if hasRestrictedKubectlFlags(normalizedCommand) {
		return false
	}

	category := classifyCommand(normalizedCommand)
	if isReadOnlyKubectlCommand(normalizedCommand) {
		return true
	}

	switch category {
	case "read-only":
		return true
	case "interactive":
		return false
	default:
		return false
	}
}

func toolExecutionDeniedReason(role, toolName, command string) string {
	if role == "" {
		role = "viewer"
	}

	if role == "admin" || role == "user" {
		return ""
	}

	if toolName != "kubectl" {
		return fmt.Sprintf("role %s can only use read-only kubectl commands in AI Assistant", role)
	}

	normalizedCommand := strings.TrimSpace(command)
	if normalizedCommand != "" && !strings.HasPrefix(normalizedCommand, "kubectl ") {
		normalizedCommand = "kubectl " + normalizedCommand
	}

	if hasRestrictedKubectlFlags(normalizedCommand) {
		return fmt.Sprintf("role %s cannot override kubectl authentication or cluster targeting in AI Assistant", role)
	}

	if classifyCommand(normalizedCommand) == "interactive" {
		return fmt.Sprintf("role %s cannot run interactive AI commands", role)
	}

	return fmt.Sprintf("role %s is limited to read-only AI actions", role)
}

func isReadOnlyKubectlCommand(command string) bool {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return false
	}

	if fields[0] == "kubectl" {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return false
	}

	verbIdx := 0
	for verbIdx < len(fields) && strings.HasPrefix(fields[verbIdx], "-") {
		verbIdx++
		if verbIdx < len(fields) &&
			!strings.HasPrefix(fields[verbIdx], "-") &&
			!strings.Contains(fields[verbIdx-1], "=") {
			verbIdx++
		}
	}
	if verbIdx >= len(fields) {
		return false
	}

	verb := fields[verbIdx]
	subcommand := ""
	if verbIdx+1 < len(fields) {
		subcommand = fields[verbIdx+1]
	}

	switch verb {
	case "get", "describe", "logs", "top", "explain", "diff", "version", "cluster-info", "api-resources", "api-versions":
		return true
	case "auth":
		return subcommand == "can-i"
	default:
		return false
	}
}

func hasRestrictedKubectlFlags(command string) bool {
	restrictedFlags := []string{
		"--as",
		"--as-group",
		"--token",
		"--kubeconfig",
		"--server",
		"--user",
		"--username",
		"--password",
		"--client-certificate",
		"--client-key",
		"--certificate-authority",
	}

	for _, flag := range restrictedFlags {
		if strings.Contains(command, flag+"=") || strings.Contains(command, flag+" ") {
			return true
		}
	}

	return false
}
