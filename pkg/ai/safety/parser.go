// Package safety provides command safety analysis using shell AST parsing.
package safety

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// ParsedCommand represents a parsed shell command
type ParsedCommand struct {
	Program     string            // Main program (e.g., "kubectl", "bash")
	Args        []string          // Arguments to the program
	Verb        string            // For kubectl: the verb (get, delete, apply, etc.)
	Namespace   string            // For kubectl: the namespace if specified
	Resource    string            // For kubectl: the resource type
	IsPiped     bool              // Command contains pipes
	IsChained   bool              // Command is chained with && or ||
	HasRedirect bool              // Command has file redirects
	Flags       map[string]string // Parsed flags with values
	RawCommand  string            // Original command string
	ParseError  error             // If parsing failed
}

// ParseCommand parses a shell command using proper AST
func ParseCommand(cmd string) *ParsedCommand {
	result := &ParsedCommand{
		RawCommand: cmd,
		Args:       make([]string, 0),
		Flags:      make(map[string]string),
	}

	parser := syntax.NewParser()
	file, err := parser.Parse(strings.NewReader(cmd), "")
	if err != nil {
		result.ParseError = err
		// Fall back to simple parsing
		result.parseSimple(cmd)
		return result
	}

	// Walk the AST
	syntax.Walk(file, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.CallExpr:
			result.parseCallExpr(n)
		case *syntax.BinaryCmd:
			switch n.Op {
			case syntax.Pipe:
				result.IsPiped = true
			case syntax.AndStmt, syntax.OrStmt:
				result.IsChained = true
			}
		case *syntax.Redirect:
			result.HasRedirect = true
		}
		return true
	})

	// Extract program-specific info
	if result.Program == "kubectl" || result.Program == "helm" {
		result.parseKubectlArgs() // Works for both kubectl and helm
	}

	return result
}

// parseCallExpr extracts command info from a call expression
func (p *ParsedCommand) parseCallExpr(expr *syntax.CallExpr) {
	if len(expr.Args) == 0 {
		return
	}

	// Extract program name (first word)
	p.Program = wordToString(expr.Args[0])

	// Extract remaining arguments
	for i := 1; i < len(expr.Args); i++ {
		arg := wordToString(expr.Args[i])
		p.Args = append(p.Args, arg)

		// Parse flags
		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				p.Flags[parts[0]] = parts[1]
			} else {
				p.Flags[arg] = ""
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// Short flag
			p.Flags[arg] = ""
		}
	}
}

// parseKubectlArgs extracts kubectl-specific information
func (p *ParsedCommand) parseKubectlArgs() {
	if len(p.Args) == 0 {
		return
	}

	// First non-flag arg is the verb
	for _, arg := range p.Args {
		if !strings.HasPrefix(arg, "-") {
			p.Verb = arg
			break
		}
	}

	// Find namespace
	for i, arg := range p.Args {
		if arg == "-n" || arg == "--namespace" {
			if i+1 < len(p.Args) {
				p.Namespace = p.Args[i+1]
			}
		}
		if strings.HasPrefix(arg, "-n=") {
			p.Namespace = strings.TrimPrefix(arg, "-n=")
		}
		if strings.HasPrefix(arg, "--namespace=") {
			p.Namespace = strings.TrimPrefix(arg, "--namespace=")
		}
	}

	// Find resource type (usually second non-flag arg)
	nonFlagCount := 0
	for _, arg := range p.Args {
		if !strings.HasPrefix(arg, "-") {
			nonFlagCount++
			if nonFlagCount == 2 {
				p.Resource = arg
				break
			}
		}
	}
}

// parseSimple performs simple string-based parsing as fallback
func (p *ParsedCommand) parseSimple(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	p.Program = parts[0]
	if len(parts) > 1 {
		p.Args = parts[1:]
	}

	// Check for pipes and chains
	p.IsPiped = strings.Contains(cmd, "|")
	p.IsChained = strings.Contains(cmd, "&&") || strings.Contains(cmd, "||")
	p.HasRedirect = strings.Contains(cmd, ">") || strings.Contains(cmd, "<")

	// Extract kubectl verb
	if p.Program == "kubectl" && len(p.Args) > 0 {
		for _, arg := range p.Args {
			if !strings.HasPrefix(arg, "-") {
				p.Verb = arg
				break
			}
		}
	}
}

// wordToString converts a syntax.Word to a string
func wordToString(word *syntax.Word) string {
	var result strings.Builder
	for _, part := range word.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			result.WriteString(p.Value)
		case *syntax.SglQuoted:
			result.WriteString(p.Value)
		case *syntax.DblQuoted:
			for _, qp := range p.Parts {
				if lit, ok := qp.(*syntax.Lit); ok {
					result.WriteString(lit.Value)
				}
			}
		case *syntax.ParamExp:
			// Variable expansion - keep placeholder
			result.WriteString("$")
			result.WriteString(p.Param.Value)
		case *syntax.CmdSubst:
			// Command substitution
			result.WriteString("$(...)") // placeholder
		}
	}
	return result.String()
}

// HasFlag checks if the command has a specific flag
func (p *ParsedCommand) HasFlag(flag string) bool {
	_, ok := p.Flags[flag]
	return ok
}

// HasAnyFlag checks if the command has any of the specified flags
func (p *ParsedCommand) HasAnyFlag(flags ...string) bool {
	for _, flag := range flags {
		if p.HasFlag(flag) {
			return true
		}
	}
	return false
}

// GetFlagValue returns the value of a flag
func (p *ParsedCommand) GetFlagValue(flag string) string {
	return p.Flags[flag]
}
