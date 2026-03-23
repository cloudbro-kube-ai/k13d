package web

import (
	"os"
	"regexp"
	"testing"
)

func TestSettingsPage_ModelProviderSelectsStayInSync(t *testing.T) {
	data, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatalf("ReadFile(static/index.html) error = %v", err)
	}

	currentProviders := extractProviderOptions(t, string(data), "setting-llm-provider")
	newProfileProviders := extractProviderOptions(t, string(data), "new-model-provider")

	if len(currentProviders) != len(newProfileProviders) {
		t.Fatalf("provider option count mismatch: current=%v new=%v", currentProviders, newProfileProviders)
	}

	for i := range currentProviders {
		if currentProviders[i] != newProfileProviders[i] {
			t.Fatalf("provider option mismatch at index %d: current=%q new=%q", i, currentProviders[i], newProfileProviders[i])
		}
	}
}

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

func extractProviderOptions(t *testing.T, html, selectID string) []string {
	t.Helper()

	selectPattern := regexp.MustCompile(`(?s)<select id="` + regexp.QuoteMeta(selectID) + `".*?</select>`)
	selectMatch := selectPattern.FindString(html)
	if selectMatch == "" {
		t.Fatalf("select %q not found", selectID)
	}

	optionPattern := regexp.MustCompile(`value="([^"]+)"`)
	matches := optionPattern.FindAllStringSubmatch(selectMatch, -1)
	if len(matches) == 0 {
		t.Fatalf("no options found for select %q", selectID)
	}

	values := make([]string, 0, len(matches))
	for _, match := range matches {
		values = append(values, match[1])
	}

	return values
}
