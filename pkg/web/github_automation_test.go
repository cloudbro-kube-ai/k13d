package web

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/automation"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func TestHandleGitHubAutomationWebhookAccepted(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.GitHub.Enabled = true
	cfg.GitHub.WebhookSecret = "secret"
	cfg.GitHub.AllowedRepositories = []string{"cloudbro-kube-ai/k13d"}

	manager, err := automation.NewManager(cfg.GitHub)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	s := &Server{cfg: cfg, automation: manager}
	payload := []byte(`{
		"action":"opened",
		"repository":{"full_name":"cloudbro-kube-ai/k13d","default_branch":"main"},
		"issue":{
			"number":1,
			"title":"Automate me",
			"body":"Please automate",
			"html_url":"https://github.com/cloudbro-kube-ai/k13d/issues/1",
			"author_association":"MEMBER",
			"user":{"login":"alice"},
			"labels":[{"name":"codex:auto"}]
		}
	}`)
	mac := hmac.New(sha256.New, []byte(cfg.GitHub.WebhookSecret))
	_, _ = mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/github/automation/webhook", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-Hub-Signature-256", signature)
	rec := httptest.NewRecorder()

	s.handleGitHubAutomationWebhook(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if accepted, _ := resp["accepted"].(bool); !accepted {
		t.Fatalf("response = %v, want accepted=true", resp)
	}
}

func TestHandleGitHubAutomationWebhookRejectsBadSignature(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.GitHub.Enabled = true
	cfg.GitHub.WebhookSecret = "secret"

	manager, err := automation.NewManager(cfg.GitHub)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	s := &Server{cfg: cfg, automation: manager}
	req := httptest.NewRequest(http.MethodPost, "/api/github/automation/webhook", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("X-GitHub-Event", "issues")
	req.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")
	rec := httptest.NewRecorder()

	s.handleGitHubAutomationWebhook(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandleGitHubAutomationJobNotFound(t *testing.T) {
	cfg := config.NewDefaultConfig()
	manager, err := automation.NewManager(cfg.GitHub)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	s := &Server{cfg: cfg, automation: manager}
	req := httptest.NewRequest(http.MethodGet, "/api/admin/github-automation/jobs/missing", nil)
	rec := httptest.NewRecorder()
	s.handleGitHubAutomationJob(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandleGitHubAutomationStatusRedactsSecrets(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.GitHub.Enabled = true
	cfg.GitHub.WebhookSecret = "webhook-secret-value"
	cfg.GitHub.PersonalAccessToken = "github_pat_abcdefghijklmnopqrstuvwxyz123456"
	cfg.GitHub.DevelopmentCommand = []string{"sh", "-c", "echo github_pat_abcdefghijklmnopqrstuvwxyz123456"}

	manager, err := automation.NewManager(cfg.GitHub)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	s := &Server{cfg: cfg, automation: manager}
	req := httptest.NewRequest(http.MethodGet, "/api/admin/github-automation/status", nil)
	rec := httptest.NewRecorder()
	s.handleGitHubAutomationStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if bytes.Contains([]byte(body), []byte(cfg.GitHub.WebhookSecret)) || bytes.Contains([]byte(body), []byte(cfg.GitHub.PersonalAccessToken)) {
		t.Fatalf("status body leaked secret: %s", body)
	}
	if !bytes.Contains([]byte(body), []byte("personal_access_token_configured")) {
		t.Fatalf("status body = %s, want configured flag", body)
	}
}

func TestParsePreviewProxyPath(t *testing.T) {
	slug, upstreamPath, ok := parsePreviewProxyPath("/previews/codex-issue-7/api/health", "/previews")
	if !ok {
		t.Fatal("expected preview path to parse")
	}
	if slug != "codex-issue-7" {
		t.Fatalf("slug = %q", slug)
	}
	if upstreamPath != "/api/health" {
		t.Fatalf("upstreamPath = %q", upstreamPath)
	}
}
