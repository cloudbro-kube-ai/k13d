package automation

import (
	"strings"
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func TestBuildBranchNameIsStablePerIssue(t *testing.T) {
	first := buildBranchName("codex/issue-", 123, "first title")
	second := buildBranchName("codex/issue-", 123, "renamed issue title")
	if first != "codex/issue-123" {
		t.Fatalf("branch = %q, want codex/issue-123", first)
	}
	if second != first {
		t.Fatalf("branch changed after title update: %q != %q", second, first)
	}
}

func TestFilterAutomationEnvironmentRemovesGitHubTokens(t *testing.T) {
	env := []string{
		"GITHUB_TOKEN=ghp_secret",
		"GH_TOKEN=ghp_secret2",
		"K13D_GITHUB_AUTOMATION_TOKEN=ghp_secret3",
		"OPENAI_API_KEY=kept",
	}
	got := strings.Join(filterAutomationEnvironment(env), "\n")
	if strings.Contains(got, "GITHUB_TOKEN") || strings.Contains(got, "GH_TOKEN") || strings.Contains(got, "K13D_GITHUB_AUTOMATION_TOKEN") {
		t.Fatalf("filtered env leaked github token names: %q", got)
	}
	if !strings.Contains(got, "OPENAI_API_KEY=kept") {
		t.Fatalf("filtered env = %q, want non-github secrets preserved", got)
	}
}

func TestRedactGitHubSecrets(t *testing.T) {
	cfg := config.GitHubAutomationConfig{
		PersonalAccessToken: "github_pat_abcdefghijklmnopqrstuvwxyz123456",
		WebhookSecret:       "webhook-secret-value",
	}
	got := RedactGitHubSecrets("token=github_pat_abcdefghijklmnopqrstuvwxyz123456 secret=webhook-secret-value", cfg)
	if strings.Contains(got, "github_pat_") || strings.Contains(got, "webhook-secret-value") {
		t.Fatalf("redacted text leaked secret: %q", got)
	}
	if strings.Count(got, redactedSecret) < 2 {
		t.Fatalf("redacted text = %q, want redaction markers", got)
	}
}
