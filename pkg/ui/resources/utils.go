// Package resources provides resource view generators for the TUI dashboard.
// Each resource type has a dedicated function that fetches and formats
// Kubernetes resources for table display.
package resources

import (
	"fmt"
	"strings"
	"time"
)

// FormatAge formats a duration into a human-readable age string.
// Examples: "5d", "12h", "30m", "45s"
func FormatAge(dur time.Duration) string {
	if dur < 0 {
		return "0s"
	}
	if dur.Hours() >= 24 {
		days := int(dur.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
	if dur.Hours() >= 1 {
		return fmt.Sprintf("%dh", int(dur.Hours()))
	}
	if dur.Minutes() >= 1 {
		return fmt.Sprintf("%dm", int(dur.Minutes()))
	}
	return fmt.Sprintf("%ds", int(dur.Seconds()))
}

// FormatAgeSince calculates the age from a given time until now.
func FormatAgeSince(t time.Time) string {
	return FormatAge(time.Since(t))
}

// TruncateString truncates a string to maxLen characters, adding "..." if truncated.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// ContainsFilter checks if the text contains the filter string (case-insensitive).
func ContainsFilter(text, filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(filter))
}

// FormatCPU formats CPU millicores for display.
// Examples: "100m", "1500m", "-"
func FormatCPU(milliCores int64) string {
	if milliCores <= 0 {
		return "-"
	}
	return fmt.Sprintf("%dm", milliCores)
}

// FormatMemory formats memory in bytes to a human-readable string.
// Examples: "128Mi", "2Gi", "-"
func FormatMemory(bytes int64) string {
	if bytes <= 0 {
		return "-"
	}

	const (
		Ki = 1024
		Mi = Ki * 1024
		Gi = Mi * 1024
		Ti = Gi * 1024
	)

	switch {
	case bytes >= Ti:
		return fmt.Sprintf("%dTi", bytes/Ti)
	case bytes >= Gi:
		return fmt.Sprintf("%dGi", bytes/Gi)
	case bytes >= Mi:
		return fmt.Sprintf("%dMi", bytes/Mi)
	case bytes >= Ki:
		return fmt.Sprintf("%dKi", bytes/Ki)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// FormatMemoryMB formats memory in megabytes for display.
func FormatMemoryMB(mb int64) string {
	if mb <= 0 {
		return "-"
	}
	return fmt.Sprintf("%dMB", mb)
}

// FormatPercentage formats a percentage value.
func FormatPercentage(value, total int64) string {
	if total == 0 {
		return "-"
	}
	pct := float64(value) / float64(total) * 100
	return fmt.Sprintf("%.1f%%", pct)
}

// FormatCount formats a count with proper pluralization suffix.
func FormatCount(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

// JoinStrings joins non-empty strings with a separator.
func JoinStrings(sep string, strs ...string) string {
	var nonEmpty []string
	for _, s := range strs {
		if s != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}
	return strings.Join(nonEmpty, sep)
}
