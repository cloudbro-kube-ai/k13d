package safety

import (
	"regexp"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
)

// PolicyEnforcer applies approval policy to command classifications.
// It provides a unified decision-making layer for both TUI and Web UI.
type PolicyEnforcer struct {
	classifier      *Classifier
	policy          config.ToolApprovalPolicy
	blockedPatterns []*regexp.Regexp
}

// Decision represents the policy decision for a command execution request.
type Decision struct {
	// Allowed indicates if the command can proceed (false if blocked by policy)
	Allowed bool

	// RequiresApproval indicates if user confirmation is needed
	RequiresApproval bool

	// Category is the command classification: "read-only", "write", "dangerous", "interactive", "unknown"
	Category string

	// Warnings contains human-readable warning messages
	Warnings []string

	// BlockReason explains why the command was blocked (empty if allowed)
	BlockReason string

	// Classification contains the full classification result
	Classification *CommandClassification
}

// NewPolicyEnforcer creates a new enforcer with the given policy.
func NewPolicyEnforcer(policy config.ToolApprovalPolicy) *PolicyEnforcer {
	e := &PolicyEnforcer{
		classifier:      NewClassifier(),
		policy:          policy,
		blockedPatterns: make([]*regexp.Regexp, 0),
	}

	// Compile blocked patterns
	for _, pattern := range policy.BlockedPatterns {
		if re, err := regexp.Compile(pattern); err == nil {
			e.blockedPatterns = append(e.blockedPatterns, re)
		}
	}

	return e
}

// NewDefaultPolicyEnforcer creates an enforcer with default policy.
func NewDefaultPolicyEnforcer() *PolicyEnforcer {
	return NewPolicyEnforcer(config.DefaultToolApprovalPolicy())
}

// Evaluate applies policy to determine if a command should proceed.
// Returns a Decision indicating allowed/blocked status and approval requirements.
func (e *PolicyEnforcer) Evaluate(command string) *Decision {
	classification := e.classifier.Classify(command)

	decision := &Decision{
		Category:       classification.Category,
		Warnings:       classification.Warnings,
		Classification: classification,
		Allowed:        true, // Default to allowed
	}

	// Check blocked patterns first
	for _, re := range e.blockedPatterns {
		if re.MatchString(command) {
			decision.Allowed = false
			decision.BlockReason = "Command matches blocked pattern: " + re.String()
			return decision
		}
	}

	// Apply policy based on category
	switch classification.Category {
	case "read-only":
		decision.Allowed = true
		decision.RequiresApproval = !e.policy.AutoApproveReadOnly

	case "write":
		decision.Allowed = true
		decision.RequiresApproval = e.policy.RequireApprovalForWrite

	case "dangerous":
		if e.policy.BlockDangerous {
			decision.Allowed = false
			decision.BlockReason = "Dangerous commands are blocked by policy"
		} else {
			decision.Allowed = true
			decision.RequiresApproval = true // Always require approval for dangerous
		}

	case "interactive":
		decision.Allowed = true
		decision.RequiresApproval = true // Always require approval for interactive
		if len(decision.Warnings) == 0 || !containsWarning(decision.Warnings, "Interactive") {
			decision.Warnings = append(decision.Warnings, "Interactive commands may not work as expected in this context")
		}

	default: // "unknown"
		decision.Allowed = true
		decision.RequiresApproval = e.policy.RequireApprovalForUnknown
	}

	return decision
}

// GetApprovalTimeout returns the configured approval timeout duration.
func (e *PolicyEnforcer) GetApprovalTimeout() time.Duration {
	seconds := e.policy.ApprovalTimeoutSeconds
	if seconds <= 0 {
		seconds = 60 // Default 60 seconds
	}
	return time.Duration(seconds) * time.Second
}

// GetPolicy returns the current policy configuration.
func (e *PolicyEnforcer) GetPolicy() config.ToolApprovalPolicy {
	return e.policy
}

// UpdatePolicy updates the enforcer's policy and recompiles patterns.
func (e *PolicyEnforcer) UpdatePolicy(policy config.ToolApprovalPolicy) {
	e.policy = policy
	e.blockedPatterns = make([]*regexp.Regexp, 0)

	for _, pattern := range policy.BlockedPatterns {
		if re, err := regexp.Compile(pattern); err == nil {
			e.blockedPatterns = append(e.blockedPatterns, re)
		}
	}
}

// containsWarning checks if a warning message already exists
func containsWarning(warnings []string, substring string) bool {
	for _, w := range warnings {
		if len(w) >= len(substring) {
			for i := 0; i <= len(w)-len(substring); i++ {
				if w[i:i+len(substring)] == substring {
					return true
				}
			}
		}
	}
	return false
}

// DefaultEnforcer is a package-level enforcer instance for convenience
var DefaultEnforcer = NewDefaultPolicyEnforcer()

// Evaluate is a convenience function using the default enforcer
func Evaluate(command string) *Decision {
	return DefaultEnforcer.Evaluate(command)
}
