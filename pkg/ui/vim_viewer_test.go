package ui

import (
	"strings"
	"testing"
)

func TestVimViewerSetContent(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"

	// We can't fully test VimViewer without a running tview.Application,
	// but we can test the content splitting logic
	lines := strings.Split(content, "\n")

	if len(lines) != 5 {
		t.Errorf("Expected 5 lines, got %d", len(lines))
	}

	if lines[0] != "line1" {
		t.Errorf("Expected first line 'line1', got %q", lines[0])
	}

	if lines[4] != "line5" {
		t.Errorf("Expected last line 'line5', got %q", lines[4])
	}
}

func TestSearchMatchLogic(t *testing.T) {
	content := `apiVersion: v1
kind: Pod
metadata:
  name: nginx
  namespace: default
spec:
  containers:
  - name: nginx
    image: nginx:latest`

	lines := strings.Split(content, "\n")

	// Simulate search for "nginx"
	pattern := "nginx"
	var matches []int
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
			matches = append(matches, i)
		}
	}

	// Should find 3 matches: name: nginx, - name: nginx, image: nginx:latest
	if len(matches) != 3 {
		t.Errorf("Expected 3 matches for 'nginx', got %d", len(matches))
	}

	// Verify match positions
	expectedMatches := []int{3, 7, 8}
	for i, expected := range expectedMatches {
		if matches[i] != expected {
			t.Errorf("Match %d: expected line %d, got %d", i, expected, matches[i])
		}
	}
}

func TestSearchWrapAround(t *testing.T) {
	matches := []int{0, 5, 10}
	currentMatch := 2 // at index 2 (line 10)

	// Test next wraps to beginning
	nextMatch := currentMatch + 1
	if nextMatch >= len(matches) {
		nextMatch = 0
	}
	if nextMatch != 0 {
		t.Errorf("Expected next match to wrap to 0, got %d", nextMatch)
	}

	// Test prev wraps to end
	currentMatch = 0
	prevMatch := currentMatch - 1
	if prevMatch < 0 {
		prevMatch = len(matches) - 1
	}
	if prevMatch != 2 {
		t.Errorf("Expected prev match to wrap to 2, got %d", prevMatch)
	}
}

func TestTitleParsing(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{" Describe: pods/nginx ", " Describe: pods/nginx "},
		// Note: extractBaseTitle strips the trailing space when it finds a marker
		{" YAML: pods/nginx [green]pattern[white] (1/3)", " YAML: pods/nginx"},
		{" Logs: ns/pod [/search_[white]", " Logs: ns/pod"},
		{" Test [red]error[white] (no matches)", " Test"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := extractBaseTitle(tt.title)
			if result != tt.expected {
				t.Errorf("extractBaseTitle(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}

// extractBaseTitle extracts the base title without search info (helper for test)
func extractBaseTitle(title string) string {
	// Remove search info
	if idx := strings.Index(title, " [/"); idx > 0 {
		return title[:idx]
	}
	if idx := strings.Index(title, " [green]"); idx > 0 {
		return title[:idx]
	}
	if idx := strings.Index(title, " [red]"); idx > 0 {
		return title[:idx]
	}
	return title
}
