package web

import (
	"os"
	"regexp"
	"testing"
)

func TestSettingsPage_DoesNotRenderLegacyOllamaQuickSetup(t *testing.T) {
	data, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatalf("ReadFile(static/index.html) error = %v", err)
	}

	html := string(data)
	for _, forbidden := range []string{
		"Ollama - Local AI (Offline Mode)",
		"Run AI models locally without API keys or internet",
		`id="ollama-setup-section"`,
	} {
		if regexp.MustCompile(regexp.QuoteMeta(forbidden)).FindString(html) != "" {
			t.Fatalf("legacy Ollama quick setup markup still present: %q", forbidden)
		}
	}
}
