package safety

import (
	"strings"
)

// CommandType classification
type CommandType string

const (
	TypeReadOnly    CommandType = "read"
	TypeWrite       CommandType = "write"
	TypeDangerous   CommandType = "dangerous"
	TypeInteractive CommandType = "interactive"
	TypeUnknown     CommandType = "unknown"
)

// Report provides detailed command analysis
type Report struct {
	Command          string
	Type             CommandType
	RequiresApproval bool
	IsDangerous      bool
	IsInteractive    bool
	IsReadOnly       bool
	Warnings         []string
	Parsed           *ParsedCommand
}

// Analyzer provides command safety analysis
type Analyzer struct {
	readOnlyVerbs    map[string]bool
	writeVerbs       map[string]bool
	dangerousFlags   []string
	interactiveFlags []string
	dangerousVerbs   []string
}

// NewAnalyzer creates a new safety analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		readOnlyVerbs: map[string]bool{
			"get":           true,
			"describe":      true,
			"logs":          true,
			"explain":       true,
			"top":           true,
			"version":       true,
			"api-resources": true,
			"api-versions":  true,
			"cluster-info":  true,
			"diff":          true,
			"auth":          true, // auth can-i is read-only
			"config":        true, // config view is read-only
		},
		writeVerbs: map[string]bool{
			"create":   true,
			"apply":    true,
			"delete":   true,
			"patch":    true,
			"scale":    true,
			"edit":     true,
			"label":    true,
			"annotate": true,
			"set":      true,
			"rollout":  true,
			"drain":    true,
			"cordon":   true,
			"uncordon": true,
			"taint":    true,
			"replace":  true,
			"expose":   true,
			"run":      true,
			"cp":       true,
		},
		dangerousFlags: []string{
			"--all",
			"-all",
			"--all-namespaces",
			"-A",
			"--force",
			"--grace-period=0",
			"--cascade=orphan",
			"--now",
		},
		interactiveFlags: []string{
			"-it",
			"-ti",
			"--tty",
			"-i",
			"--stdin",
		},
		dangerousVerbs: []string{
			"drain",
			"cordon",
			"taint",
		},
	}
}

// Analyze performs comprehensive command analysis
func (a *Analyzer) Analyze(cmd string) *Report {
	report := &Report{
		Command:  cmd,
		Warnings: make([]string, 0),
	}

	// Parse the command
	parsed := ParseCommand(cmd)
	report.Parsed = parsed

	// Check for parsing errors
	if parsed.ParseError != nil {
		report.Type = TypeUnknown
		report.RequiresApproval = true
		report.Warnings = append(report.Warnings, "Command parsing failed, treating as unknown")
	}

	// Check for piping/chaining - always requires approval
	if parsed.IsPiped {
		report.Warnings = append(report.Warnings, "Piped command detected")
		report.RequiresApproval = true
	}
	if parsed.IsChained {
		report.Warnings = append(report.Warnings, "Chained command detected")
		report.RequiresApproval = true
	}

	// Check for redirects
	if parsed.HasRedirect {
		report.Warnings = append(report.Warnings, "File redirect detected")
		report.RequiresApproval = true
	}

	// Analyze based on program
	switch parsed.Program {
	case "kubectl":
		a.analyzeKubectl(report, parsed)
	case "helm":
		a.analyzeHelm(report, parsed)
	default:
		// Non-kubectl commands default to requiring approval
		report.Type = TypeUnknown
		report.RequiresApproval = true
		report.Warnings = append(report.Warnings, "Non-kubectl command")
	}

	return report
}

// analyzeKubectl analyzes kubectl commands
func (a *Analyzer) analyzeKubectl(report *Report, parsed *ParsedCommand) {
	verb := parsed.Verb

	// Check verb type
	if a.readOnlyVerbs[verb] {
		report.Type = TypeReadOnly
		report.IsReadOnly = true
		// Read-only doesn't require approval by default
	} else if a.writeVerbs[verb] {
		report.Type = TypeWrite
		report.RequiresApproval = true
	} else {
		report.Type = TypeUnknown
		report.RequiresApproval = true
	}

	// Check for dangerous patterns
	a.checkDangerousPatterns(report, parsed)

	// Check for interactive flags
	a.checkInteractivePatterns(report, parsed)

	// Special case: exec is always interactive
	if verb == "exec" {
		report.Type = TypeInteractive
		report.IsInteractive = true
		report.RequiresApproval = true
		report.Warnings = append(report.Warnings, "Interactive exec command")
	}

	// Special case: port-forward is interactive
	if verb == "port-forward" {
		report.Type = TypeInteractive
		report.IsInteractive = true
		report.RequiresApproval = true
		report.Warnings = append(report.Warnings, "Port-forward requires long-running process")
	}
}

// analyzeHelm analyzes helm commands
func (a *Analyzer) analyzeHelm(report *Report, parsed *ParsedCommand) {
	verb := parsed.Verb

	readOnlyHelmVerbs := map[string]bool{
		"list":   true,
		"status": true,
		"get":    true,
		"show":   true,
		"search": true,
		"repo":   true, // repo list, etc.
	}

	writeHelmVerbs := map[string]bool{
		"install":   true,
		"upgrade":   true,
		"uninstall": true,
		"delete":    true,
		"rollback":  true,
	}

	if readOnlyHelmVerbs[verb] {
		report.Type = TypeReadOnly
		report.IsReadOnly = true
	} else if writeHelmVerbs[verb] {
		report.Type = TypeWrite
		report.RequiresApproval = true
	} else {
		report.Type = TypeUnknown
		report.RequiresApproval = true
	}

	// Check for dangerous patterns
	a.checkDangerousPatterns(report, parsed)
}

// checkDangerousPatterns checks for dangerous command patterns
func (a *Analyzer) checkDangerousPatterns(report *Report, parsed *ParsedCommand) {
	// Check dangerous flags
	for _, flag := range a.dangerousFlags {
		if parsed.HasFlag(flag) {
			report.Type = TypeDangerous
			report.IsDangerous = true
			report.RequiresApproval = true
			report.Warnings = append(report.Warnings, "Dangerous flag: "+flag)
		}
	}

	// Check dangerous verbs
	for _, verb := range a.dangerousVerbs {
		if parsed.Verb == verb {
			report.Type = TypeDangerous
			report.IsDangerous = true
			report.RequiresApproval = true
			report.Warnings = append(report.Warnings, "Dangerous operation: "+verb)
		}
	}

	// Special case: delete with namespace
	if parsed.Verb == "delete" {
		if parsed.Resource == "namespace" || parsed.Resource == "ns" {
			report.IsDangerous = true
			report.Warnings = append(report.Warnings, "Deleting namespace removes all resources in it")
		}

		// delete with --all
		if parsed.HasFlag("--all") || parsed.HasFlag("-all") {
			report.IsDangerous = true
			report.Warnings = append(report.Warnings, "Delete all resources of this type")
		}

		// delete in all namespaces
		if parsed.HasFlag("--all-namespaces") || parsed.HasFlag("-A") {
			report.IsDangerous = true
			report.Warnings = append(report.Warnings, "Delete affects all namespaces")
		}
	}

	// Check for rm -rf or similar bash patterns
	if parsed.Program == "rm" {
		if parsed.HasAnyFlag("-rf", "-fr", "-r", "--recursive") {
			report.Type = TypeDangerous
			report.IsDangerous = true
			report.RequiresApproval = true
			report.Warnings = append(report.Warnings, "Recursive file deletion")
		}
	}
}

// checkInteractivePatterns checks for interactive command patterns
func (a *Analyzer) checkInteractivePatterns(report *Report, parsed *ParsedCommand) {
	for _, flag := range a.interactiveFlags {
		if parsed.HasFlag(flag) {
			report.Type = TypeInteractive
			report.IsInteractive = true
			report.RequiresApproval = true
			report.Warnings = append(report.Warnings, "Interactive mode not fully supported")
			return
		}
	}
}

// QuickCheck performs a quick safety check without full parsing
// Use for performance-critical paths
func QuickCheck(cmd string) (isReadOnly, isDangerous bool) {
	cmdLower := strings.ToLower(cmd)

	// Quick read-only check
	readOnlyPrefixes := []string{
		"kubectl get",
		"kubectl describe",
		"kubectl logs",
		"kubectl explain",
		"kubectl top",
		"kubectl version",
		"kubectl api-",
		"kubectl cluster-info",
		"kubectl auth can-i",
	}

	for _, prefix := range readOnlyPrefixes {
		if strings.HasPrefix(cmdLower, prefix) {
			isReadOnly = true
			break
		}
	}

	// Quick dangerous check
	dangerousPatterns := []string{
		"delete", "--all", "--force", "--grace-period=0",
		"drain", "cordon", "rm -rf",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, pattern) {
			isDangerous = true
			break
		}
	}

	return
}
