package automation

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

type fakeExecutor struct {
	result       *ExecutionResult
	deployResult *PreviewDeployment
	err          error
	deployErr    error
	block        chan struct{}
}

func (f *fakeExecutor) Execute(ctx context.Context, job *Job) (*ExecutionResult, error) {
	if f.block != nil {
		<-f.block
	}
	return f.result, f.err
}

func (f *fakeExecutor) DeployPreview(ctx context.Context, job *Job, result *ExecutionResult) (*PreviewDeployment, error) {
	return f.deployResult, f.deployErr
}

type fakeReporter struct {
	mu            sync.Mutex
	issueComments []string
	prCreated     bool
	reviewCreated bool
	waitedForCI   bool
}

func (f *fakeReporter) PostIssueComment(ctx context.Context, repo string, issueNumber int, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.issueComments = append(f.issueComments, body)
	return nil
}

func (f *fakeReporter) CreatePullRequest(ctx context.Context, repo, title, head, base, body string, draft bool) (*PullRequestInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prCreated = true
	return &PullRequestInfo{Number: 42, URL: "https://github.com/example/repo/pull/42"}, nil
}

func (f *fakeReporter) CreatePullRequestReview(ctx context.Context, repo string, prNumber int, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reviewCreated = true
	return nil
}

func (f *fakeReporter) WaitForChecks(ctx context.Context, repo, ref string, timeout, interval time.Duration) (*CIResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.waitedForCI = true
	return &CIResult{Status: "completed", Conclusion: "success", URL: "https://github.com/example/repo/actions/runs/1"}, nil
}

func (f *fakeReporter) snapshot() (bool, bool, bool, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.prCreated, f.reviewCreated, f.waitedForCI, len(f.issueComments)
}

func TestVerifyGitHubSignature(t *testing.T) {
	payload := []byte(`{"hello":"world"}`)
	secret := "topsecret"
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !VerifyGitHubSignature(secret, payload, signature) {
		t.Fatal("expected signature verification to succeed")
	}
	if VerifyGitHubSignature(secret, payload, "sha256=deadbeef") {
		t.Fatal("expected invalid signature to fail")
	}
}

func TestParseIssueEvent(t *testing.T) {
	body := []byte(`{
		"action":"labeled",
		"repository":{"full_name":"cloudbro-kube-ai/k13d","default_branch":"main"},
		"issue":{
			"number":17,
			"title":"Automate me",
			"body":"Please fix this",
			"html_url":"https://github.com/cloudbro-kube-ai/k13d/issues/17",
			"user":{"login":"alice"},
			"labels":[{"name":"bug"},{"name":"codex:auto"}]
		},
		"label":{"name":"codex:auto"}
	}`)
	event, err := ParseIssueEvent("issues", body)
	if err != nil {
		t.Fatalf("ParseIssueEvent() error = %v", err)
	}
	if event.Repository != "cloudbro-kube-ai/k13d" {
		t.Fatalf("Repository = %s", event.Repository)
	}
	if event.TriggeredLabel != "codex:auto" {
		t.Fatalf("TriggeredLabel = %s", event.TriggeredLabel)
	}
}

func TestManagerQueueIssueEvent_DedupesActiveIssue(t *testing.T) {
	cfg := config.NewDefaultConfig().GitHub
	cfg.Enabled = true
	cfg.AllowedRepositories = []string{"cloudbro-kube-ai/k13d"}
	cfg.TriggerLabel = "codex:auto"

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()
	block := make(chan struct{})
	manager.executor = &fakeExecutor{result: &ExecutionResult{}, block: block}
	manager.reporter = &fakeReporter{}

	event := &IssueEvent{
		EventName:      "issues",
		Action:         "opened",
		Repository:     "cloudbro-kube-ai/k13d",
		IssueNumber:    99,
		IssueTitle:     "Automate me",
		IssueURL:       "https://github.com/cloudbro-kube-ai/k13d/issues/99",
		Labels:         []string{"codex:auto"},
		TriggeredLabel: "",
	}
	first := manager.QueueIssueEvent(event)
	if !first.Accepted {
		t.Fatalf("first queue result = %#v", first)
	}
	second := manager.QueueIssueEvent(event)
	if !second.Ignored || second.Reason == "" {
		t.Fatalf("second queue result = %#v, want ignored duplicate", second)
	}
	close(block)
}

func TestManagerRunJob_CreatesPRAndReview(t *testing.T) {
	cfg := config.NewDefaultConfig().GitHub
	cfg.Enabled = true
	cfg.AllowedRepositories = []string{"cloudbro-kube-ai/k13d"}
	cfg.TriggerLabel = "codex:auto"
	cfg.AutoPush = true
	cfg.AutoCreatePR = true
	cfg.WaitForCI = true
	cfg.AutoDeployPreview = true

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	reporter := &fakeReporter{}
	manager.executor = &fakeExecutor{
		result: &ExecutionResult{
			Branch:         "codex/issue-101-automate",
			WorktreePath:   "/tmp/worktree",
			CommitSHA:      "abc123",
			HasChanges:     true,
			DevelopmentLog: "implemented change",
			ReviewLog:      "looks good overall",
			DiffStat:       " foo.go | 2 +-",
		},
		deployResult: &PreviewDeployment{
			Slug:      "codex-issue-101-automate",
			PublicURL: "https://fingerscore.net/previews/codex-issue-101-automate/",
			TargetURL: "http://127.0.0.1:18101",
			Log:       "K13D_PREVIEW_TARGET=http://127.0.0.1:18101",
		},
	}
	manager.reporter = reporter

	result := manager.QueueIssueEvent(&IssueEvent{
		EventName:   "issues",
		Action:      "opened",
		Repository:  "cloudbro-kube-ai/k13d",
		IssueNumber: 101,
		IssueTitle:  "Automate me",
		IssueURL:    "https://github.com/cloudbro-kube-ai/k13d/issues/101",
		Labels:      []string{"codex:auto"},
	})
	if !result.Accepted {
		t.Fatalf("QueueIssueEvent() = %#v", result)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := manager.GetJob(result.JobID)
		if ok && (job.Status == JobStatusSucceeded || job.Status == JobStatusFailed) {
			if job.Status != JobStatusSucceeded {
				t.Fatalf("job status = %s, error = %s", job.Status, job.Error)
			}
			prCreated, reviewCreated, waitedForCI, issueComments := reporter.snapshot()
			if !prCreated {
				t.Fatal("expected PR creation")
			}
			if !waitedForCI {
				t.Fatal("expected CI wait")
			}
			if !reviewCreated {
				t.Fatal("expected PR review creation")
			}
			if issueComments == 0 {
				t.Fatal("expected issue completion comment")
			}
			if job.PreviewURL == "" || job.PreviewTarget == "" {
				t.Fatalf("expected preview details, got url=%q target=%q", job.PreviewURL, job.PreviewTarget)
			}
			target, ok := manager.GetPreviewTarget("codex-issue-101-automate")
			if !ok || target != "http://127.0.0.1:18101" {
				t.Fatalf("GetPreviewTarget() = %q, %v", target, ok)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for automation job to finish")
}

func TestParsePreviewDeploymentOutput(t *testing.T) {
	target, public := parsePreviewDeploymentOutput(`starting
K13D_PREVIEW_TARGET=http://127.0.0.1:18081
K13D_PREVIEW_URL=https://fingerscore.net/previews/codex-issue-1/
done`)
	if target != "http://127.0.0.1:18081" {
		t.Fatalf("target = %q", target)
	}
	if public != "https://fingerscore.net/previews/codex-issue-1/" {
		t.Fatalf("public = %q", public)
	}
}

func TestParseGitHubRemoteURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"git@github.com:cloudbro-kube-ai/k13d.git", "cloudbro-kube-ai/k13d"},
		{"https://github.com/cloudbro-kube-ai/k13d.git", "cloudbro-kube-ai/k13d"},
	}
	for _, tt := range tests {
		got, err := parseGitHubRemoteURL(tt.input)
		if err != nil {
			t.Fatalf("parseGitHubRemoteURL(%q) error = %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("parseGitHubRemoteURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
