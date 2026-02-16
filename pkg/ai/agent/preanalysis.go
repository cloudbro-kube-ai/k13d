package agent

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/analyzers"
)

// runPreAnalysis runs SRE analyzers on the resource context and returns findings summary.
// Best-effort: returns empty string if parsing fails or no findings.
func (a *Agent) runPreAnalysis(ctx context.Context, resourceContext string) string {
	if a.analyzerRegistry == nil {
		return ""
	}

	info := parseResourceContext(resourceContext)
	if info == nil {
		return ""
	}

	findings := a.analyzerRegistry.AnalyzeAll(ctx, info)
	if len(findings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n[Pre-analysis findings]\n")
	for _, f := range findings {
		sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n", f.Severity, f.Title, f.Details))
		for _, s := range f.Suggestions {
			sb.WriteString(fmt.Sprintf("  Suggestion: %s\n", s))
		}
	}
	return sb.String()
}

var restartsRe = regexp.MustCompile(`(?i)restarts?\s*:\s*(\d+)`)

// parseResourceContext tries to extract resource info from context string.
// Returns nil if parsing fails (best-effort).
func parseResourceContext(ctx string) *analyzers.ResourceInfo {
	if strings.TrimSpace(ctx) == "" {
		return nil
	}

	info := &analyzers.ResourceInfo{}
	lines := strings.Split(ctx, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "kind", "type":
			info.Kind = value
		case "name":
			info.Name = value
		case "namespace":
			info.Namespace = value
		case "status", "phase":
			info.Status = value
		}
	}

	// Parse restart count from the full context
	if m := restartsRe.FindStringSubmatch(ctx); len(m) > 1 {
		if count, err := strconv.ParseInt(m[1], 10, 32); err == nil {
			if len(info.Containers) == 0 {
				info.Containers = append(info.Containers, analyzers.ContainerInfo{
					RestartCount: int32(count),
					State:        info.Status,
				})
			}
		}
	}

	// Need at least some identifying information to be useful
	if info.Kind == "" && info.Name == "" && info.Status == "" {
		return nil
	}

	return info
}
