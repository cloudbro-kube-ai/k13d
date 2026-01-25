package ai

import (
	"regexp"
	"strings"
)

// CommandType represents the classification of a kubectl command
type CommandType string

const (
	CommandTypeReadOnly    CommandType = "read"
	CommandTypeWrite       CommandType = "write"
	CommandTypeUnknown     CommandType = "unknown"
	CommandTypeDangerous   CommandType = "dangerous"
	CommandTypeInteractive CommandType = "interactive"
)

// CommandFilter provides safety validation for kubectl commands
type CommandFilter struct {
	readOnlyOps    []string
	writeOps       []string
	dangerousOps   []string
	interactiveOps []string
}

// NewCommandFilter creates a new command filter with default rules
func NewCommandFilter() *CommandFilter {
	return &CommandFilter{
		readOnlyOps: []string{
			"get", "describe", "logs", "api-resources", "api-versions",
			"can-i", "whoami", "explain", "cluster-info", "top",
			"version", "config view", "config current-context",
			"config get-contexts", "diff",
		},
		writeOps: []string{
			"create", "apply", "delete", "patch", "scale", "edit",
			"label", "annotate", "set", "rollout", "replace",
			"expose", "autoscale", "taint", "cordon", "uncordon",
			"drain", "cp", "attach", "run",
		},
		dangerousOps: []string{
			"--all", "-all", "delete namespace",
			"delete ns", "--force", "--force --grace-period=0",
			"--cascade=orphan", "drain",
		},
		interactiveOps: []string{
			"edit", "exec -it", "exec -ti", "exec --tty",
			"attach", "port-forward", "proxy", "run -it", "run -ti",
		},
	}
}

// ClassifyCommand classifies a kubectl command
func (f *CommandFilter) ClassifyCommand(cmd string) CommandType {
	cmd = strings.TrimSpace(cmd)
	cmdLower := strings.ToLower(cmd)

	// Check for dangerous operations first
	for _, dangerous := range f.dangerousOps {
		if strings.Contains(cmdLower, dangerous) {
			return CommandTypeDangerous
		}
	}

	// Check for interactive operations
	for _, interactive := range f.interactiveOps {
		if strings.Contains(cmdLower, interactive) {
			return CommandTypeInteractive
		}
	}

	// Check for write operations
	for _, write := range f.writeOps {
		if strings.HasPrefix(cmdLower, "kubectl "+write) ||
			strings.HasPrefix(cmdLower, write) {
			return CommandTypeWrite
		}
	}

	// Check for read-only operations
	for _, readonly := range f.readOnlyOps {
		if strings.HasPrefix(cmdLower, "kubectl "+readonly) ||
			strings.HasPrefix(cmdLower, readonly) {
			return CommandTypeReadOnly
		}
	}

	// Check for composite commands (pipes, subshells)
	if isCompositeCommand(cmd) {
		return CommandTypeUnknown
	}

	return CommandTypeUnknown
}

// IsReadOnly returns true if the command is safe to execute without confirmation
func (f *CommandFilter) IsReadOnly(cmd string) bool {
	return f.ClassifyCommand(cmd) == CommandTypeReadOnly
}

// RequiresConfirmation returns true if the command should be confirmed by user
func (f *CommandFilter) RequiresConfirmation(cmd string) bool {
	cmdType := f.ClassifyCommand(cmd)
	return cmdType == CommandTypeWrite || cmdType == CommandTypeDangerous || cmdType == CommandTypeUnknown
}

// IsDangerous returns true if the command is potentially dangerous
func (f *CommandFilter) IsDangerous(cmd string) bool {
	return f.ClassifyCommand(cmd) == CommandTypeDangerous
}

// IsInteractive returns true if the command requires interactive input
func (f *CommandFilter) IsInteractive(cmd string) bool {
	return f.ClassifyCommand(cmd) == CommandTypeInteractive
}

// isCompositeCommand checks for shell command chaining/piping
func isCompositeCommand(cmd string) bool {
	// Check for shell operators
	shellOps := []string{"|", "&&", "||", ";", "`", "$(", "${"}
	for _, op := range shellOps {
		if strings.Contains(cmd, op) {
			return true
		}
	}
	return false
}

// ExtractKubectlCommands extracts kubectl commands from AI response text
func ExtractKubectlCommands(text string) []string {
	var commands []string

	// Pattern for code blocks with kubectl commands
	codeBlockPattern := regexp.MustCompile("```(?:bash|sh|shell)?\\s*\\n([\\s\\S]*?)```")
	matches := codeBlockPattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) > 1 {
			lines := strings.Split(match[1], "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "kubectl ") {
					commands = append(commands, line)
				}
			}
		}
	}

	// Also look for inline kubectl commands
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip lines that are in code blocks (already processed)
		if strings.HasPrefix(line, "```") {
			continue
		}
		// Look for kubectl commands that start a line
		if strings.HasPrefix(line, "kubectl ") {
			commands = append(commands, line)
		}
		// Look for kubectl commands prefixed with $
		if strings.HasPrefix(line, "$ kubectl ") {
			commands = append(commands, strings.TrimPrefix(line, "$ "))
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, cmd := range commands {
		if !seen[cmd] {
			seen[cmd] = true
			unique = append(unique, cmd)
		}
	}

	return unique
}

// CommandSafetyReport provides a safety analysis of a command
type CommandSafetyReport struct {
	Command              string
	Type                 CommandType
	RequiresConfirmation bool
	IsDangerous          bool
	IsInteractive        bool
	Warnings             []string
}

// AnalyzeCommand provides a comprehensive safety analysis
func (f *CommandFilter) AnalyzeCommand(cmd string) *CommandSafetyReport {
	cmdType := f.ClassifyCommand(cmd)
	report := &CommandSafetyReport{
		Command:              cmd,
		Type:                 cmdType,
		RequiresConfirmation: f.RequiresConfirmation(cmd),
		IsDangerous:          cmdType == CommandTypeDangerous,
		IsInteractive:        cmdType == CommandTypeInteractive,
		Warnings:             []string{},
	}

	// Add specific warnings
	cmdLower := strings.ToLower(cmd)

	if strings.Contains(cmdLower, "delete") {
		report.Warnings = append(report.Warnings, "This command will delete resources")
	}
	if strings.Contains(cmdLower, "--all") || strings.Contains(cmdLower, "-all") {
		report.Warnings = append(report.Warnings, "This command affects ALL resources")
	}
	if strings.Contains(cmdLower, "--force") {
		report.Warnings = append(report.Warnings, "Force flag may cause data loss")
	}
	if strings.Contains(cmdLower, "grace-period=0") {
		report.Warnings = append(report.Warnings, "No grace period - immediate termination")
	}
	if strings.Contains(cmdLower, "drain") {
		report.Warnings = append(report.Warnings, "Drain will evict all pods from the node")
	}
	if strings.Contains(cmdLower, "namespace") || strings.Contains(cmdLower, " ns ") {
		if strings.Contains(cmdLower, "delete") {
			report.Warnings = append(report.Warnings, "Deleting namespace removes all resources in it")
		}
	}
	if isCompositeCommand(cmd) {
		report.Warnings = append(report.Warnings, "Composite command detected - review carefully")
	}

	return report
}

// ValidateCommands validates a list of commands and returns safety reports
func (f *CommandFilter) ValidateCommands(commands []string) []*CommandSafetyReport {
	reports := make([]*CommandSafetyReport, len(commands))
	for i, cmd := range commands {
		reports[i] = f.AnalyzeCommand(cmd)
	}
	return reports
}
