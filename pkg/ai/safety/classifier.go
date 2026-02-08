package safety

// CommandClassification represents the unified classification result
// for both TUI and Web UI tool approval flows.
type CommandClassification struct {
	// Category is the command category: "read-only", "write", "dangerous", "interactive", "unknown"
	Category string

	// RequiresApproval indicates if user confirmation is needed before execution
	RequiresApproval bool

	// IsDangerous indicates if the command could cause significant damage
	IsDangerous bool

	// IsInteractive indicates if the command requires interactive terminal
	IsInteractive bool

	// IsReadOnly indicates if the command only reads data
	IsReadOnly bool

	// Warnings contains human-readable warning messages
	Warnings []string

	// RawReport contains the full analysis report for debugging
	RawReport *Report
}

// Classifier provides unified command classification for both TUI and Web UI.
// It wraps the existing Analyzer to provide a consistent API.
type Classifier struct {
	analyzer *Analyzer
}

// NewClassifier creates a new unified classifier
func NewClassifier() *Classifier {
	return &Classifier{
		analyzer: NewAnalyzer(),
	}
}

// Classify analyzes a command and returns unified classification result.
// This is the primary entry point for command safety classification.
func (c *Classifier) Classify(command string) *CommandClassification {
	report := c.analyzer.Analyze(command)

	return &CommandClassification{
		Category:         c.typeToCategory(report.Type),
		RequiresApproval: report.RequiresApproval,
		IsDangerous:      report.IsDangerous,
		IsInteractive:    report.IsInteractive,
		IsReadOnly:       report.IsReadOnly,
		Warnings:         report.Warnings,
		RawReport:        report,
	}
}

// typeToCategory converts CommandType to web-compatible category string
func (c *Classifier) typeToCategory(t CommandType) string {
	switch t {
	case TypeReadOnly:
		return "read-only"
	case TypeWrite:
		return "write"
	case TypeDangerous:
		return "dangerous"
	case TypeInteractive:
		return "interactive"
	default:
		return "unknown"
	}
}

// ClassifyQuick performs a fast classification without full parsing.
// Use for performance-critical paths where detailed warnings aren't needed.
func (c *Classifier) ClassifyQuick(command string) *CommandClassification {
	isReadOnly, isDangerous := QuickCheck(command)

	category := "unknown"
	if isDangerous {
		category = "dangerous"
	} else if isReadOnly {
		category = "read-only"
	}

	return &CommandClassification{
		Category:         category,
		RequiresApproval: !isReadOnly || isDangerous,
		IsDangerous:      isDangerous,
		IsReadOnly:       isReadOnly,
		Warnings:         nil,
	}
}

// DefaultClassifier is a package-level classifier instance for convenience
var DefaultClassifier = NewClassifier()

// Classify is a convenience function using the default classifier
func Classify(command string) *CommandClassification {
	return DefaultClassifier.Classify(command)
}
