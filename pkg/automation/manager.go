package automation

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/google/uuid"
)

type Manager struct {
	cfg           config.GitHubAutomationConfig
	repoPath      string
	allowedRepos  map[string]struct{}
	executor      Executor
	reporter      GitHubReporter
	jobs          map[string]*Job
	activeByIssue map[string]string
	queue         chan *Job
	ctx           context.Context
	cancel        context.CancelFunc
	now           func() time.Time
	mu            sync.RWMutex
}

func NewManager(cfg config.GitHubAutomationConfig) (*Manager, error) {
	repoPath := strings.TrimSpace(cfg.RepoPath)
	if repoPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			repoPath = cwd
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	manager := &Manager{
		cfg:           cfg,
		repoPath:      repoPath,
		allowedRepos:  make(map[string]struct{}),
		jobs:          make(map[string]*Job),
		activeByIssue: make(map[string]string),
		queue:         make(chan *Job, 32),
		ctx:           ctx,
		cancel:        cancel,
		now:           time.Now,
	}
	if token := strings.TrimSpace(cfg.PersonalAccessToken); token != "" {
		manager.reporter = NewGitHubClient(token)
	}
	manager.executor = NewDefaultExecutor(cfg, repoPath)

	for _, repo := range cfg.AllowedRepositories {
		repo = strings.TrimSpace(repo)
		if repo != "" {
			manager.allowedRepos[repo] = struct{}{}
		}
	}
	if len(manager.allowedRepos) == 0 && repoPath != "" {
		if repo, err := DetectGitHubRepository(context.Background(), repoPath, cfg.Remote); err == nil && repo != "" {
			manager.allowedRepos[repo] = struct{}{}
		}
	}

	workers := cfg.MaxConcurrentJobs
	if workers < 1 {
		workers = 1
	}
	if cfg.Enabled {
		for i := 0; i < workers; i++ {
			go manager.worker()
		}
	}
	return manager, nil
}

func (m *Manager) Enabled() bool {
	return m != nil && m.cfg.Enabled
}

func (m *Manager) Config() config.GitHubAutomationConfig {
	return m.cfg
}

func (m *Manager) Close() {
	if m == nil || m.cancel == nil {
		return
	}
	m.cancel()
}

func (m *Manager) QueueIssueEvent(event *IssueEvent) QueueResult {
	if m == nil || !m.cfg.Enabled {
		return QueueResult{Ignored: true, Reason: "github automation is disabled"}
	}
	if event == nil {
		return QueueResult{Ignored: true, Reason: "empty event"}
	}
	if ok, reason := m.shouldRun(event); !ok {
		return QueueResult{Ignored: true, Reason: reason}
	}

	key := issueKey(event.Repository, event.IssueNumber)
	m.mu.Lock()
	if existingID, ok := m.activeByIssue[key]; ok {
		m.mu.Unlock()
		return QueueResult{Ignored: true, Reason: "job already queued or running", JobID: existingID}
	}
	job := &Job{
		ID:            uuid.NewString(),
		Repository:    event.Repository,
		IssueNumber:   event.IssueNumber,
		IssueTitle:    event.IssueTitle,
		IssueBody:     event.IssueBody,
		IssueURL:      event.IssueURL,
		IssueAuthor:   event.IssueAuthor,
		TriggerAction: event.Action,
		TriggerLabel:  event.TriggeredLabel,
		Status:        JobStatusQueued,
		CreatedAt:     m.now(),
	}
	m.jobs[job.ID] = job
	m.activeByIssue[key] = job.ID
	m.mu.Unlock()

	select {
	case m.queue <- job:
		return QueueResult{Accepted: true, JobID: job.ID}
	case <-m.ctx.Done():
		return QueueResult{Ignored: true, Reason: "automation manager is shutting down"}
	}
}

func (m *Manager) ListJobs() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		copyJob := *job
		out = append(out, &copyJob)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

func (m *Manager) GetJob(id string) (*Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return nil, false
	}
	copyJob := *job
	return &copyJob, true
}

func (m *Manager) GetPreviewTarget(slug string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", false
	}
	for _, job := range m.jobs {
		if job.PreviewSlug == slug && strings.TrimSpace(job.PreviewTarget) != "" {
			return job.PreviewTarget, true
		}
	}
	return "", false
}

func (m *Manager) worker() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case job := <-m.queue:
			if job == nil {
				continue
			}
			m.runJob(job)
		}
	}
}

func (m *Manager) runJob(job *Job) {
	m.updateJob(job.ID, func(current *Job) {
		current.Status = JobStatusRunning
		current.StartedAt = m.now()
	})

	result, err := m.executor.Execute(m.ctx, job)
	if err != nil {
		m.finishJob(job, result, err)
		return
	}

	if result != nil {
		job.Branch = result.Branch
		job.WorktreePath = result.WorktreePath
		job.CommitSHA = result.CommitSHA
		job.HasChanges = result.HasChanges
		job.DevelopmentLog = result.DevelopmentLog
		job.ReviewLog = result.ReviewLog
		job.DiffStat = result.DiffStat
	}

	if result != nil && result.HasChanges && m.cfg.AutoPush && m.cfg.AutoCreatePR && m.reporter != nil {
		title := fmt.Sprintf("Issue #%d: %s", job.IssueNumber, job.IssueTitle)
		body := buildPullRequestBody(job, result)
		pr, prErr := m.reporter.CreatePullRequest(m.ctx, job.Repository, title, result.Branch, effectiveBaseBranch(m.cfg), body, m.cfg.PullRequestDraft)
		if prErr != nil {
			job.Warnings = append(job.Warnings, "failed to create pull request: "+prErr.Error())
		} else {
			job.PullRequestURL = pr.URL
			job.PullRequestNumber = pr.Number
		}
	}

	if err := m.waitForCI(job, result); err != nil {
		m.finishJob(job, result, err)
		return
	}

	if result != nil && job.PullRequestNumber > 0 && m.reporter != nil && strings.TrimSpace(result.ReviewLog) != "" {
		if reviewErr := m.reporter.CreatePullRequestReview(m.ctx, job.Repository, job.PullRequestNumber, buildReviewBody(job, result)); reviewErr != nil {
			job.Warnings = append(job.Warnings, "failed to create pull request review: "+reviewErr.Error())
		}
	}

	if err := m.deployPreview(job, result); err != nil {
		m.finishJob(job, result, err)
		return
	}

	job.Status = JobStatusSucceeded
	job.StatusReason = "completed"
	job.FinishedAt = m.now()
	m.persistJob(job)
	m.releaseIssueLock(job)
	m.postCompletionComment(job, result, nil)
}

func (m *Manager) waitForCI(job *Job, result *ExecutionResult) error {
	if !m.cfg.WaitForCI || result == nil || !result.HasChanges || !m.cfg.AutoPush {
		return nil
	}
	if m.reporter == nil {
		job.Warnings = append(job.Warnings, "wait_for_ci is enabled but github token is not configured")
		return nil
	}
	ref := strings.TrimSpace(result.CommitSHA)
	if ref == "" {
		ref = strings.TrimSpace(result.Branch)
	}
	if ref == "" {
		job.Warnings = append(job.Warnings, "wait_for_ci skipped because no commit SHA or branch was produced")
		return nil
	}

	m.updateJob(job.ID, func(current *Job) {
		current.Status = JobStatusWaitingCI
		current.StatusReason = "waiting for GitHub checks"
	})

	ciResult, err := m.reporter.WaitForChecks(
		m.ctx,
		job.Repository,
		ref,
		time.Duration(ciWaitTimeoutSeconds(m.cfg))*time.Second,
		time.Duration(ciPollIntervalSeconds(m.cfg))*time.Second,
	)
	if ciResult != nil {
		job.CIStatus = ciResult.Status
		job.CIConclusion = ciResult.Conclusion
		job.CIURL = ciResult.URL
	}
	if err != nil {
		if job.CIStatus == "" {
			job.CIStatus = "failed"
		}
		if job.CIConclusion == "" {
			job.CIConclusion = "failure"
		}
		return fmt.Errorf("ci checks failed: %w", err)
	}
	if job.CIStatus == "" {
		job.CIStatus = "completed"
	}
	if job.CIConclusion == "" {
		job.CIConclusion = "success"
	}
	return nil
}

func (m *Manager) deployPreview(job *Job, result *ExecutionResult) error {
	deployment, err := m.executor.DeployPreview(m.ctx, job, result)
	if deployment == nil && err == nil {
		return nil
	}
	m.updateJob(job.ID, func(current *Job) {
		current.Status = JobStatusDeploying
		current.StatusReason = "deploying branch preview"
	})
	if deployment != nil {
		job.PreviewSlug = deployment.Slug
		job.PreviewURL = deployment.PublicURL
		job.PreviewTarget = deployment.TargetURL
		job.DeploymentLog = deployment.Log
	}
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) finishJob(job *Job, result *ExecutionResult, execErr error) {
	if result != nil {
		job.Branch = result.Branch
		job.WorktreePath = result.WorktreePath
		job.CommitSHA = result.CommitSHA
		job.HasChanges = result.HasChanges
		job.DevelopmentLog = result.DevelopmentLog
		job.ReviewLog = result.ReviewLog
		job.DiffStat = result.DiffStat
	}
	job.Status = JobStatusFailed
	job.StatusReason = execErr.Error()
	job.Error = execErr.Error()
	job.FinishedAt = m.now()
	m.persistJob(job)
	m.releaseIssueLock(job)
	m.postCompletionComment(job, result, execErr)
}

func ciWaitTimeoutSeconds(cfg config.GitHubAutomationConfig) int {
	if cfg.CIWaitTimeoutSeconds > 0 {
		return cfg.CIWaitTimeoutSeconds
	}
	return 600
}

func ciPollIntervalSeconds(cfg config.GitHubAutomationConfig) int {
	if cfg.CIPollIntervalSeconds > 0 {
		return cfg.CIPollIntervalSeconds
	}
	return 10
}

func (m *Manager) persistJob(job *Job) {
	m.mu.Lock()
	defer m.mu.Unlock()
	copyJob := *job
	m.jobs[job.ID] = &copyJob
}

func (m *Manager) releaseIssueLock(job *Job) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.activeByIssue, issueKey(job.Repository, job.IssueNumber))
}

func (m *Manager) postCompletionComment(job *Job, result *ExecutionResult, execErr error) {
	if m.reporter == nil {
		return
	}
	body := buildIssueComment(job, result, execErr)
	if err := m.reporter.PostIssueComment(m.ctx, job.Repository, job.IssueNumber, body); err != nil {
		m.updateJob(job.ID, func(current *Job) {
			current.Warnings = append(current.Warnings, "failed to post issue comment: "+err.Error())
		})
	}
}

func (m *Manager) updateJob(id string, mutate func(*Job)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return
	}
	mutate(job)
}

func (m *Manager) shouldRun(event *IssueEvent) (bool, string) {
	if event.EventName != "issues" {
		return false, "only issues webhook events are supported"
	}
	if event.Repository == "" {
		return false, "repository is missing from webhook payload"
	}
	if len(m.allowedRepos) > 0 {
		if _, ok := m.allowedRepos[event.Repository]; !ok {
			return false, "repository is not in the allowed list"
		}
	}
	switch event.Action {
	case "opened", "reopened":
		if label := strings.TrimSpace(m.cfg.TriggerLabel); label != "" && !containsLabel(event.Labels, label) {
			return false, fmt.Sprintf("missing trigger label %q", label)
		}
		return true, ""
	case "labeled":
		label := strings.TrimSpace(m.cfg.TriggerLabel)
		if label != "" && !strings.EqualFold(event.TriggeredLabel, label) {
			return false, fmt.Sprintf("label %q does not match trigger label %q", event.TriggeredLabel, label)
		}
		return true, ""
	default:
		return false, "action is not configured for automation"
	}
}

func issueKey(repo string, issueNumber int) string {
	return fmt.Sprintf("%s#%d", repo, issueNumber)
}

func containsLabel(labels []string, want string) bool {
	for _, label := range labels {
		if strings.EqualFold(strings.TrimSpace(label), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}

func effectiveBaseBranch(cfg config.GitHubAutomationConfig) string {
	if strings.TrimSpace(cfg.BaseBranch) != "" {
		return cfg.BaseBranch
	}
	return "main"
}

func buildIssueComment(job *Job, result *ExecutionResult, execErr error) string {
	var b strings.Builder
	if execErr != nil {
		fmt.Fprintf(&b, "## k13d issue automation failed\n\n")
		fmt.Fprintf(&b, "- Issue: #%d\n", job.IssueNumber)
		fmt.Fprintf(&b, "- Repository: `%s`\n", job.Repository)
		fmt.Fprintf(&b, "- Error: `%s`\n", execErr.Error())
		if result != nil && strings.TrimSpace(result.DevelopmentLog) != "" {
			b.WriteString("\n### Development output\n\n```text\n")
			b.WriteString(result.DevelopmentLog)
			b.WriteString("\n```\n")
		}
		return b.String()
	}

	fmt.Fprintf(&b, "## k13d issue automation completed\n\n")
	fmt.Fprintf(&b, "- Issue: #%d\n", job.IssueNumber)
	fmt.Fprintf(&b, "- Branch: `%s`\n", job.Branch)
	if job.CommitSHA != "" {
		fmt.Fprintf(&b, "- Commit: `%s`\n", job.CommitSHA)
	}
	if job.PullRequestURL != "" {
		fmt.Fprintf(&b, "- Pull request: %s\n", job.PullRequestURL)
	}
	if job.CIConclusion != "" {
		fmt.Fprintf(&b, "- CI: `%s`\n", job.CIConclusion)
	}
	if job.CIURL != "" {
		fmt.Fprintf(&b, "- CI details: %s\n", job.CIURL)
	}
	if job.PreviewURL != "" {
		fmt.Fprintf(&b, "- Preview: %s\n", job.PreviewURL)
	}
	if !job.HasChanges {
		b.WriteString("- Result: no file changes were produced\n")
	}
	if strings.TrimSpace(job.DiffStat) != "" {
		b.WriteString("\n### Diff summary\n\n```text\n")
		b.WriteString(job.DiffStat)
		b.WriteString("\n```\n")
	}
	if strings.TrimSpace(job.ReviewLog) != "" {
		b.WriteString("\n### Review summary\n\n```text\n")
		b.WriteString(job.ReviewLog)
		b.WriteString("\n```\n")
	}
	if strings.TrimSpace(job.DeploymentLog) != "" {
		b.WriteString("\n### Deployment output\n\n```text\n")
		b.WriteString(job.DeploymentLog)
		b.WriteString("\n```\n")
	}
	if len(job.Warnings) > 0 {
		b.WriteString("\n### Warnings\n")
		for _, warning := range job.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}
	return b.String()
}

func buildPullRequestBody(job *Job, result *ExecutionResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Summary\n")
	fmt.Fprintf(&b, "- automated from GitHub issue #%d\n", job.IssueNumber)
	fmt.Fprintf(&b, "- source issue: %s\n", job.IssueURL)
	if strings.TrimSpace(result.DiffStat) != "" {
		b.WriteString("\n## Diff Summary\n\n```text\n")
		b.WriteString(result.DiffStat)
		b.WriteString("\n```\n")
	}
	if strings.TrimSpace(result.ReviewLog) != "" {
		b.WriteString("\n## Review Notes\n\n```text\n")
		b.WriteString(result.ReviewLog)
		b.WriteString("\n```\n")
	}
	return b.String()
}

func buildReviewBody(job *Job, result *ExecutionResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Automated review for issue #%d.\n\n", job.IssueNumber)
	if strings.TrimSpace(result.ReviewLog) != "" {
		b.WriteString(result.ReviewLog)
	} else {
		b.WriteString("No separate review command was configured.")
	}
	return b.String()
}
