package automation

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

type fakeExecutor struct {
	result       *ExecutionResult
	deployResult *PreviewDeployment
	reviewLog    string
	err          error
	deployErr    error
	reviewErr    error
	block        chan struct{}
}

func (f *fakeExecutor) Execute(ctx context.Context, job *Job) (*ExecutionResult, error) {
	if f.block != nil {
		<-f.block
	}
	return f.result, f.err
}

func (f *fakeExecutor) Review(ctx context.Context, job *Job, branch string) (string, error) {
	return f.reviewLog, f.reviewErr
}

func (f *fakeExecutor) DeployPreview(ctx context.Context, job *Job, result *ExecutionResult) (*PreviewDeployment, error) {
	return f.deployResult, f.deployErr
}

type fakeReporter struct {
	mu            sync.Mutex
	issueComments []string
	prComments    []string
	assignees     []string
	reviewers     []string
	prBody        string
	reviewBody    string
	prCreated     bool
	reviewCreated bool
	mergedPR      bool
	issueClosed   bool
	closeReason   string
	closeErr      error
	existingPR    *PullRequestInfo
	waitedForCI   bool
	orgMember     bool
	orgMemberErr  error
	orgMembers    []string
	orgMembersErr error
}

func (f *fakeReporter) PostIssueComment(ctx context.Context, repo string, issueNumber int, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.issueComments = append(f.issueComments, body)
	return nil
}

func (f *fakeReporter) PostPullRequestComment(ctx context.Context, repo string, prNumber int, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prComments = append(f.prComments, body)
	return nil
}

func (f *fakeReporter) AssignIssue(ctx context.Context, repo string, issueNumber int, assignees []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.assignees = append([]string(nil), assignees...)
	return nil
}

func (f *fakeReporter) FindOpenPullRequestByHead(ctx context.Context, repo, head string) (*PullRequestInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.existingPR == nil {
		return nil, nil
	}
	pr := *f.existingPR
	return &pr, nil
}

func (f *fakeReporter) CreatePullRequest(ctx context.Context, repo, title, head, base, body string, draft bool) (*PullRequestInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prCreated = true
	f.prBody = body
	return &PullRequestInfo{Number: 42, URL: "https://github.com/example/repo/pull/42"}, nil
}

func (f *fakeReporter) RequestPullRequestReviewers(ctx context.Context, repo string, prNumber int, reviewers []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reviewers = append([]string(nil), reviewers...)
	return nil
}

func (f *fakeReporter) CreatePullRequestReview(ctx context.Context, repo string, prNumber int, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reviewCreated = true
	f.reviewBody = body
	return nil
}

func (f *fakeReporter) MergePullRequest(ctx context.Context, repo string, prNumber int, method, title, message string) (*PullRequestMergeInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mergedPR = true
	return &PullRequestMergeInfo{SHA: "merge-sha", Merged: true, Message: "merged"}, nil
}

func (f *fakeReporter) CloseIssue(ctx context.Context, repo string, issueNumber int, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.issueClosed = true
	f.closeReason = reason
	return f.closeErr
}

func (f *fakeReporter) WaitForChecks(ctx context.Context, repo, ref string, timeout, interval time.Duration) (*CIResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.waitedForCI = true
	return &CIResult{Status: "completed", Conclusion: "success", URL: "https://github.com/example/repo/actions/runs/1"}, nil
}

func (f *fakeReporter) IsOrganizationMember(ctx context.Context, org, username string) (bool, error) {
	return f.orgMember, f.orgMemberErr
}

func (f *fakeReporter) ListOrganizationMembers(ctx context.Context, org string, limit int) ([]string, error) {
	if f.orgMembersErr != nil {
		return nil, f.orgMembersErr
	}
	if len(f.orgMembers) > limit && limit > 0 {
		return append([]string(nil), f.orgMembers[:limit]...), nil
	}
	return append([]string(nil), f.orgMembers...), nil
}

func (f *fakeReporter) snapshot() (bool, bool, bool, int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.prCreated, f.reviewCreated, f.waitedForCI, len(f.issueComments), len(f.prComments)
}

func (f *fakeReporter) assignedAndReviewers() ([]string, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.assignees...), append([]string(nil), f.reviewers...)
}

func (f *fakeReporter) mergeSnapshot() (bool, bool, string, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.mergedPR, f.issueClosed, f.closeReason, append([]string(nil), f.issueComments...)
}

func (f *fakeReporter) completionComments() ([]string, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.issueComments...), append([]string(nil), f.prComments...)
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
			"author_association":"MEMBER",
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
	if event.IssueAuthorAssociation != "MEMBER" {
		t.Fatalf("IssueAuthorAssociation = %s", event.IssueAuthorAssociation)
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
		EventName:              "issues",
		Action:                 "opened",
		Repository:             "cloudbro-kube-ai/k13d",
		IssueNumber:            99,
		IssueTitle:             "Automate me",
		IssueURL:               "https://github.com/cloudbro-kube-ai/k13d/issues/99",
		IssueAuthor:            "alice",
		IssueAuthorAssociation: "MEMBER",
		Labels:                 []string{"codex:auto"},
		TriggeredLabel:         "",
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

	reporter := &fakeReporter{orgMembers: []string{"alice", "bob"}}
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
		EventName:              "issues",
		Action:                 "opened",
		Repository:             "cloudbro-kube-ai/k13d",
		IssueNumber:            101,
		IssueTitle:             "Automate me",
		IssueURL:               "https://github.com/cloudbro-kube-ai/k13d/issues/101",
		IssueAuthor:            "alice",
		IssueAuthorAssociation: "MEMBER",
		Labels:                 []string{"codex:auto"},
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
			prCreated, reviewCreated, waitedForCI, issueCommentCount, prCommentCount := reporter.snapshot()
			if !prCreated {
				t.Fatal("expected PR creation")
			}
			if !waitedForCI {
				t.Fatal("expected CI wait")
			}
			if !reviewCreated {
				t.Fatal("expected PR review creation")
			}
			if issueCommentCount == 0 {
				t.Fatal("expected issue completion comment")
			}
			if prCommentCount == 0 {
				t.Fatal("expected pull request verification comment")
			}
			assignees, reviewers := reporter.assignedAndReviewers()
			if strings.Join(assignees, ",") != "alice" {
				t.Fatalf("assignees = %#v, want issue author", assignees)
			}
			if strings.Join(reviewers, ",") != "alice,bob" {
				t.Fatalf("reviewers = %#v, want organization members", reviewers)
			}
			issueComments, prComments := reporter.completionComments()
			if !strings.Contains(issueComments[0], "@alice @bob") {
				t.Fatalf("acceptance comment = %q, want org mentions", issueComments[0])
			}
			if !strings.Contains(issueComments[len(issueComments)-1], "배포 확인 링크") {
				t.Fatalf("completion comment = %q, want Korean preview link", issueComments[len(issueComments)-1])
			}
			if !strings.Contains(prComments[len(prComments)-1], "CI/CD 확인 경로") {
				t.Fatalf("PR comment = %q, want Korean CI/CD verification heading", prComments[len(prComments)-1])
			}
			if !strings.Contains(prComments[len(prComments)-1], "https://fingerscore.net/previews/codex-issue-101-automate/") {
				t.Fatalf("PR comment = %q, want preview URL", prComments[len(prComments)-1])
			}
			if !strings.Contains(reporter.prBody, "## 요약") {
				t.Fatalf("PR body = %q, want Korean body", reporter.prBody)
			}
			if !strings.Contains(reporter.reviewBody, "## 자동 코드 리뷰") {
				t.Fatalf("review body = %q, want Korean review", reporter.reviewBody)
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

func TestManagerRunJob_ReusesExistingPullRequestForIssueBranch(t *testing.T) {
	cfg := config.NewDefaultConfig().GitHub
	cfg.Enabled = true
	cfg.AllowedRepositories = []string{"cloudbro-kube-ai/k13d"}
	cfg.TriggerLabel = "codex:auto"
	cfg.AutoPush = true
	cfg.AutoCreatePR = true

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	reporter := &fakeReporter{
		existingPR: &PullRequestInfo{Number: 77, URL: "https://github.com/example/repo/pull/77"},
		orgMembers: []string{"alice", "bob"},
	}
	manager.executor = &fakeExecutor{
		result: &ExecutionResult{
			Branch:         "codex/issue-104",
			WorktreePath:   "/tmp/worktree",
			CommitSHA:      "def456",
			HasChanges:     true,
			DevelopmentLog: "continued change",
		},
	}
	manager.reporter = reporter

	result := manager.QueueIssueEvent(&IssueEvent{
		EventName:              "issues",
		Action:                 "labeled",
		Repository:             "cloudbro-kube-ai/k13d",
		IssueNumber:            104,
		IssueTitle:             "Keep working here",
		IssueURL:               "https://github.com/cloudbro-kube-ai/k13d/issues/104",
		IssueAuthor:            "alice",
		IssueAuthorAssociation: "MEMBER",
		Labels:                 []string{"codex:auto"},
		TriggeredLabel:         "codex:auto",
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
			if reporter.prCreated {
				t.Fatal("expected existing PR to be reused instead of creating another PR")
			}
			if job.PullRequestNumber != 77 || job.PullRequestURL != "https://github.com/example/repo/pull/77" {
				t.Fatalf("job PR = #%d %q, want existing PR", job.PullRequestNumber, job.PullRequestURL)
			}
			_, reviewers := reporter.assignedAndReviewers()
			if strings.Join(reviewers, ",") != "alice,bob" {
				t.Fatalf("reviewers = %#v, want organization members", reviewers)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for automation job to finish")
}

func TestManagerHandleIssueCommentEvent_MergesExistingIssuePR(t *testing.T) {
	cfg := config.NewDefaultConfig().GitHub
	cfg.Enabled = true
	cfg.AllowIssueMerge = true
	cfg.AllowedRepositories = []string{"cloudbro-kube-ai/k13d"}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	reporter := &fakeReporter{
		existingPR: &PullRequestInfo{Number: 88, URL: "https://github.com/example/repo/pull/88"},
	}
	manager.reporter = reporter

	result := manager.HandleIssueCommentEvent(&IssueCommentEvent{
		EventName:                "issue_comment",
		Action:                   "created",
		Repository:               "cloudbro-kube-ai/k13d",
		IssueNumber:              105,
		IssueTitle:               "Ship it",
		IssueURL:                 "https://github.com/cloudbro-kube-ai/k13d/issues/105",
		CommentBody:              "k13d main에 merge 해줘",
		CommentAuthor:            "alice",
		CommentAuthorAssociation: "MEMBER",
	})
	if !result.Accepted {
		t.Fatalf("HandleIssueCommentEvent() = %#v, want accepted", result)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		merged, issueClosed, closeReason, comments := reporter.mergeSnapshot()
		if merged && len(comments) >= 2 {
			if !issueClosed {
				t.Fatal("expected issue to be closed after successful merge")
			}
			if closeReason != "completed" {
				t.Fatalf("close reason = %q, want completed", closeReason)
			}
			if !strings.Contains(comments[len(comments)-1], "병합 완료") {
				t.Fatalf("completion comment = %q, want Korean merge success", comments[len(comments)-1])
			}
			if !strings.Contains(comments[len(comments)-1], "이슈를 완료 상태로 닫았습니다") {
				t.Fatalf("completion comment = %q, want issue close confirmation", comments[len(comments)-1])
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for merge command")
}

func TestManagerHandleIssueCommentEvent_MergeSuccessReportsCloseFailure(t *testing.T) {
	cfg := config.NewDefaultConfig().GitHub
	cfg.Enabled = true
	cfg.AllowIssueMerge = true
	cfg.AllowedRepositories = []string{"cloudbro-kube-ai/k13d"}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	reporter := &fakeReporter{
		closeErr:   errors.New("missing issue write permission"),
		existingPR: &PullRequestInfo{Number: 88, URL: "https://github.com/example/repo/pull/88"},
	}
	manager.reporter = reporter

	result := manager.HandleIssueCommentEvent(&IssueCommentEvent{
		EventName:                "issue_comment",
		Action:                   "created",
		Repository:               "cloudbro-kube-ai/k13d",
		IssueNumber:              105,
		IssueTitle:               "Ship it",
		IssueURL:                 "https://github.com/cloudbro-kube-ai/k13d/issues/105",
		CommentBody:              "k13d main에 merge 해줘",
		CommentAuthor:            "alice",
		CommentAuthorAssociation: "MEMBER",
	})
	if !result.Accepted {
		t.Fatalf("HandleIssueCommentEvent() = %#v, want accepted", result)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		merged, issueClosed, _, comments := reporter.mergeSnapshot()
		if merged && issueClosed && len(comments) >= 2 {
			if !strings.Contains(comments[len(comments)-1], "main 반영은 완료되었지만 이슈 닫기는 실패했습니다") {
				t.Fatalf("completion comment = %q, want close failure warning", comments[len(comments)-1])
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for merge command")
}

func TestManagerHandleIssueCommentEvent_RequiresMergeEnabled(t *testing.T) {
	cfg := config.NewDefaultConfig().GitHub
	cfg.Enabled = true
	cfg.AllowedRepositories = []string{"cloudbro-kube-ai/k13d"}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()
	manager.reporter = &fakeReporter{}

	result := manager.HandleIssueCommentEvent(&IssueCommentEvent{
		EventName:                "issue_comment",
		Action:                   "created",
		Repository:               "cloudbro-kube-ai/k13d",
		IssueNumber:              106,
		IssueTitle:               "Ship it",
		CommentBody:              "k13d merge",
		CommentAuthor:            "alice",
		CommentAuthorAssociation: "MEMBER",
	})
	if !result.Ignored || !strings.Contains(result.Reason, "disabled") {
		t.Fatalf("HandleIssueCommentEvent() = %#v, want disabled", result)
	}
}

func TestManagerHandleIssueCommentEvent_RunsCodexReview(t *testing.T) {
	cfg := config.NewDefaultConfig().GitHub
	cfg.Enabled = true
	cfg.AllowedRepositories = []string{"cloudbro-kube-ai/k13d"}
	cfg.ReviewCommand = []string{"./scripts/run-agent-review.sh"}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	reporter := &fakeReporter{
		existingPR: &PullRequestInfo{Number: 89, URL: "https://github.com/example/repo/pull/89"},
	}
	manager.executor = &fakeExecutor{reviewLog: "발견사항 없음. 잔여 리스크는 E2E 범위입니다."}
	manager.reporter = reporter

	result := manager.HandleIssueCommentEvent(&IssueCommentEvent{
		EventName:                "issue_comment",
		Action:                   "created",
		Repository:               "cloudbro-kube-ai/k13d",
		IssueNumber:              107,
		IssueTitle:               "Review it",
		IssueURL:                 "https://github.com/cloudbro-kube-ai/k13d/issues/107",
		CommentBody:              "k13d 코드리뷰 해줘",
		CommentAuthor:            "alice",
		CommentAuthorAssociation: "MEMBER",
	})
	if !result.Accepted {
		t.Fatalf("HandleIssueCommentEvent() = %#v, want accepted", result)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		reporter.mu.Lock()
		reviewCreated := reporter.reviewCreated
		reviewBody := reporter.reviewBody
		comments := append([]string(nil), reporter.issueComments...)
		reporter.mu.Unlock()
		if reviewCreated && len(comments) >= 2 {
			if !strings.Contains(comments[0], "코드 리뷰 요청 접수") {
				t.Fatalf("accepted comment = %q, want Korean review accepted", comments[0])
			}
			if !strings.Contains(comments[len(comments)-1], "코드 리뷰 완료") {
				t.Fatalf("completion comment = %q, want Korean review success", comments[len(comments)-1])
			}
			if !strings.Contains(reviewBody, "## 자동 코드 리뷰") || !strings.Contains(reviewBody, "발견사항 없음") {
				t.Fatalf("review body = %q, want Codex review body", reviewBody)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for review command")
}

func TestManagerQueueIssueEvent_RequiresOrgMember(t *testing.T) {
	cfg := config.NewDefaultConfig().GitHub
	cfg.Enabled = true
	cfg.AllowedRepositories = []string{"cloudbro-kube-ai/k13d"}
	cfg.TriggerLabel = "codex:auto"
	cfg.RequireAuthorOrgMember = true

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()
	manager.reporter = &fakeReporter{orgMember: false}

	result := manager.QueueIssueEvent(&IssueEvent{
		EventName:      "issues",
		Action:         "labeled",
		Repository:     "cloudbro-kube-ai/k13d",
		IssueNumber:    102,
		IssueTitle:     "Automate me",
		IssueURL:       "https://github.com/cloudbro-kube-ai/k13d/issues/102",
		IssueAuthor:    "external-user",
		Labels:         []string{"codex:auto"},
		TriggeredLabel: "codex:auto",
	})
	if !result.Ignored {
		t.Fatalf("QueueIssueEvent() = %#v, want ignored", result)
	}
	if !strings.Contains(result.Reason, "not a member") {
		t.Fatalf("Reason = %q, want membership denial", result.Reason)
	}
}

func TestManagerQueueIssueEvent_AllowsVerifiedOrgMember(t *testing.T) {
	cfg := config.NewDefaultConfig().GitHub
	cfg.Enabled = true
	cfg.AllowedRepositories = []string{"cloudbro-kube-ai/k13d"}
	cfg.TriggerLabel = "codex:auto"
	cfg.RequireAuthorOrgMember = true

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()
	manager.reporter = &fakeReporter{orgMember: true}
	manager.executor = &fakeExecutor{result: &ExecutionResult{}}

	result := manager.QueueIssueEvent(&IssueEvent{
		EventName:      "issues",
		Action:         "labeled",
		Repository:     "cloudbro-kube-ai/k13d",
		IssueNumber:    103,
		IssueTitle:     "Automate me",
		IssueURL:       "https://github.com/cloudbro-kube-ai/k13d/issues/103",
		IssueAuthor:    "org-user",
		Labels:         []string{"codex:auto"},
		TriggeredLabel: "codex:auto",
	})
	if !result.Accepted {
		t.Fatalf("QueueIssueEvent() = %#v, want accepted", result)
	}
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
