package ai

import (
	"regexp"
	"strings"
)

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
