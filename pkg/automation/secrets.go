package automation

import (
	"os"
	"regexp"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

const redactedSecret = "[REDACTED]"

var githubTokenPattern = regexp.MustCompile(`\b(?:github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9_]{20,})\b`)

func commandEnvironment(placeholders map[string]string) []string {
	return append(filterAutomationEnvironment(osEnvironment()), buildCommandEnv(placeholders)...)
}

var osEnvironment = os.Environ

func filterAutomationEnvironment(env []string) []string {
	out := make([]string, 0, len(env))
	for _, entry := range env {
		name, _, ok := strings.Cut(entry, "=")
		if !ok || isGitHubSecretEnvName(name) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func isGitHubSecretEnvName(name string) bool {
	name = strings.ToUpper(strings.TrimSpace(name))
	switch name {
	case "GITHUB_TOKEN", "GH_TOKEN", "GITHUB_PAT", "K13D_GITHUB_TOKEN", "K13D_GITHUB_AUTOMATION_TOKEN":
		return true
	}
	return strings.Contains(name, "GITHUB") && (strings.Contains(name, "TOKEN") || strings.Contains(name, "PAT"))
}

func RedactGitHubSecrets(text string, cfg config.GitHubAutomationConfig) string {
	return redactSensitiveText(text, sensitiveGitHubValues(cfg))
}

func sensitiveGitHubValues(cfg config.GitHubAutomationConfig) []string {
	values := []string{
		cfg.PersonalAccessToken,
		cfg.WebhookSecret,
	}
	for _, entry := range osEnvironment() {
		name, value, ok := strings.Cut(entry, "=")
		if ok && isGitHubSecretEnvName(name) {
			values = append(values, value)
		}
	}
	return values
}

func redactSensitiveText(text string, values []string) string {
	if text == "" {
		return ""
	}
	redacted := text
	for _, value := range values {
		value = strings.TrimSpace(value)
		if len(value) < 4 {
			continue
		}
		redacted = strings.ReplaceAll(redacted, value, redactedSecret)
	}
	return githubTokenPattern.ReplaceAllString(redacted, redactedSecret)
}
