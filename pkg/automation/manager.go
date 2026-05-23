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

	if m.ctx.Err() != nil {
		m.releaseIssueLock(job)
		return QueueResult{Ignored: true, Reason: "automation manager is shutting down"}
	}
	m.assignIssueAuthor(job)
	m.postAcceptedIssueComment(job)

	select {
	case m.queue <- job:
		return QueueResult{Accepted: true, JobID: job.ID}
	case <-m.ctx.Done():
		return QueueResult{Ignored: true, Reason: "automation manager is shutting down"}
	}
}

func (m *Manager) HandleIssueCommentEvent(event *IssueCommentEvent) QueueResult {
	if m == nil || !m.cfg.Enabled {
		return QueueResult{Ignored: true, Reason: "github automation is disabled"}
	}
	if event == nil {
		return QueueResult{Ignored: true, Reason: "empty event"}
	}
	if event.EventName != "issue_comment" {
		return QueueResult{Ignored: true, Reason: "only issue_comment webhook events are supported"}
	}
	if event.Action != "created" {
		return QueueResult{Ignored: true, Reason: "only new issue comments are supported"}
	}
	if event.Repository == "" {
		return QueueResult{Ignored: true, Reason: "repository is missing from webhook payload"}
	}
	if len(m.allowedRepos) > 0 {
		if _, ok := m.allowedRepos[event.Repository]; !ok {
			return QueueResult{Ignored: true, Reason: "repository is not in the allowed list"}
		}
	}
	if isReviewCommand(event.CommentBody) {
		if ok, reason := m.githubUserIsOrgMember(event.Repository, event.CommentAuthor, event.CommentAuthorAssociation); !ok {
			return QueueResult{Ignored: true, Reason: reason}
		}
		if m.reporter == nil {
			return QueueResult{Ignored: true, Reason: "github token is required to create pull request reviews"}
		}
		if len(m.cfg.ReviewCommand) == 0 {
			return QueueResult{Ignored: true, Reason: "review_command is not configured"}
		}

		go m.reviewIssuePullRequest(event)
		return QueueResult{Accepted: true}
	}
	if !isMergeCommand(event.CommentBody) {
		return QueueResult{Ignored: true, Reason: "comment does not contain a k13d review or merge command"}
	}
	if !m.cfg.AllowIssueMerge {
		return QueueResult{Ignored: true, Reason: "issue merge command is disabled"}
	}
	if ok, reason := m.githubUserIsOrgMember(event.Repository, event.CommentAuthor, event.CommentAuthorAssociation); !ok {
		return QueueResult{Ignored: true, Reason: reason}
	}
	if m.reporter == nil {
		return QueueResult{Ignored: true, Reason: "github token is required to merge pull requests"}
	}

	go m.mergeIssuePullRequest(event)
	return QueueResult{Accepted: true}
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
		if prErr := m.ensurePullRequest(job, result); prErr != nil {
			job.Warnings = append(job.Warnings, "failed to create pull request: "+prErr.Error())
		}
	}

	if job.PullRequestNumber > 0 {
		m.requestOrganizationReviewers(job)
	}

	if err := m.waitForCI(job, result); err != nil {
		m.finishJob(job, result, err)
		return
	}

	if result != nil && job.PullRequestNumber > 0 && m.reporter != nil && strings.TrimSpace(result.ReviewLog) != "" {
		if reviewErr := m.reporter.CreatePullRequestReview(m.ctx, job.Repository, job.PullRequestNumber, buildReviewBody(job, result, m.cfg)); reviewErr != nil {
			job.Warnings = append(job.Warnings, "failed to create pull request review: "+reviewErr.Error())
		}
	}

	if err := m.deployPreview(job, result); err != nil {
		m.finishJob(job, result, err)
		return
	}

	if err := m.postPullRequestVerificationComment(job); err != nil {
		job.Warnings = append(job.Warnings, "failed to post pull request verification comment: "+err.Error())
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
	body := buildIssueComment(job, result, execErr, m.cfg)
	if err := m.reporter.PostIssueComment(m.ctx, job.Repository, job.IssueNumber, body); err != nil {
		m.updateJob(job.ID, func(current *Job) {
			current.Warnings = append(current.Warnings, "failed to post issue comment: "+err.Error())
		})
	}
}

func (m *Manager) postPullRequestVerificationComment(job *Job) error {
	if m.reporter == nil || job == nil || job.PullRequestNumber <= 0 {
		return nil
	}
	body := buildPullRequestVerificationComment(job, m.cfg)
	if strings.TrimSpace(body) == "" {
		return nil
	}
	return m.reporter.PostPullRequestComment(m.ctx, job.Repository, job.PullRequestNumber, body)
}

func (m *Manager) postAcceptedIssueComment(job *Job) {
	if m.reporter == nil {
		return
	}
	mentions, err := m.organizationMentions(job.Repository)
	if err != nil {
		m.updateJob(job.ID, func(current *Job) {
			current.Warnings = append(current.Warnings, "failed to mention organization members: "+err.Error())
		})
	}
	body := buildAcceptedIssueComment(job, mentions, m.cfg)
	if err := m.reporter.PostIssueComment(m.ctx, job.Repository, job.IssueNumber, body); err != nil {
		m.updateJob(job.ID, func(current *Job) {
			current.Warnings = append(current.Warnings, "failed to post issue acceptance comment: "+err.Error())
		})
	}
}

func (m *Manager) assignIssueAuthor(job *Job) {
	if m.reporter == nil {
		return
	}
	author := strings.TrimSpace(job.IssueAuthor)
	if author == "" {
		return
	}
	if err := m.reporter.AssignIssue(m.ctx, job.Repository, job.IssueNumber, []string{author}); err != nil {
		m.updateJob(job.ID, func(current *Job) {
			current.Warnings = append(current.Warnings, "failed to assign issue author: "+err.Error())
		})
	}
}

func (m *Manager) ensurePullRequest(job *Job, result *ExecutionResult) error {
	if result == nil {
		return nil
	}
	title := fmt.Sprintf("Issue #%d: %s", job.IssueNumber, job.IssueTitle)
	body := buildPullRequestBody(job, result, m.cfg)
	if existing, err := m.reporter.FindOpenPullRequestByHead(m.ctx, job.Repository, result.Branch); err != nil {
		return err
	} else if existing != nil {
		job.PullRequestURL = existing.URL
		job.PullRequestNumber = existing.Number
		return nil
	}

	pr, err := m.reporter.CreatePullRequest(m.ctx, job.Repository, title, result.Branch, effectiveBaseBranch(m.cfg), body, m.cfg.PullRequestDraft)
	if err != nil {
		if existing, findErr := m.reporter.FindOpenPullRequestByHead(m.ctx, job.Repository, result.Branch); findErr == nil && existing != nil {
			job.PullRequestURL = existing.URL
			job.PullRequestNumber = existing.Number
			job.Warnings = append(job.Warnings, "reused existing pull request after create failed: "+err.Error())
			return nil
		}
		return err
	}
	job.PullRequestURL = pr.URL
	job.PullRequestNumber = pr.Number
	return nil
}

func (m *Manager) requestOrganizationReviewers(job *Job) {
	if m.reporter == nil || job.PullRequestNumber <= 0 {
		return
	}
	reviewers, err := m.organizationMembers(job.Repository, m.cfg.MentionMaxMembers)
	if err != nil {
		m.updateJob(job.ID, func(current *Job) {
			current.Warnings = append(current.Warnings, "failed to list organization reviewers: "+err.Error())
		})
		return
	}
	if len(reviewers) == 0 {
		return
	}
	if err := m.reporter.RequestPullRequestReviewers(m.ctx, job.Repository, job.PullRequestNumber, reviewers); err != nil {
		m.updateJob(job.ID, func(current *Job) {
			current.Warnings = append(current.Warnings, "failed to request organization reviewers: "+err.Error())
		})
	}
}

func (m *Manager) organizationMembers(repo string, limit int) ([]string, error) {
	if m.reporter == nil {
		return nil, nil
	}
	owner := repositoryOwner(repo)
	if owner == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()
	members, err := m.reporter.ListOrganizationMembers(ctx, owner, limit)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(members))
	seen := make(map[string]struct{}, len(members))
	for _, member := range members {
		member = strings.TrimSpace(strings.TrimPrefix(member, "@"))
		if member == "" {
			continue
		}
		key := strings.ToLower(member)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, member)
	}
	return out, nil
}

func (m *Manager) organizationMentions(repo string) ([]string, error) {
	if !m.cfg.MentionOrgMembers {
		return nil, nil
	}
	members, err := m.organizationMembers(repo, m.cfg.MentionMaxMembers)
	if err != nil {
		return nil, err
	}
	mentions := make([]string, 0, len(members))
	for _, member := range members {
		mentions = append(mentions, "@"+member)
	}
	return mentions, nil
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
	if ok, reason := m.matchesTrigger(event); !ok {
		return false, reason
	}
	return m.authorCanRun(event)
}

func (m *Manager) matchesTrigger(event *IssueEvent) (bool, string) {
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

func (m *Manager) authorCanRun(event *IssueEvent) (bool, string) {
	if !m.cfg.RequireAuthorOrgMember {
		return true, ""
	}

	return m.githubUserCanRun(event.Repository, event.IssueAuthor, event.IssueAuthorAssociation)
}

func (m *Manager) githubUserCanRun(repo, username, association string) (bool, string) {
	if !m.cfg.RequireAuthorOrgMember {
		return true, ""
	}
	return m.githubUserIsOrgMember(repo, username, association)
}

func (m *Manager) githubUserIsOrgMember(repo, username, association string) (bool, string) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, "github user is missing from webhook payload"
	}
	owner := repositoryOwner(repo)
	if owner == "" {
		return false, "repository owner is missing from webhook payload"
	}

	switch strings.ToUpper(strings.TrimSpace(association)) {
	case "OWNER", "MEMBER":
		return true, ""
	}

	if m.reporter == nil {
		return false, fmt.Sprintf("github token is required to verify github user %q is a member of organization %q", username, owner)
	}
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()
	ok, err := m.reporter.IsOrganizationMember(ctx, owner, username)
	if err != nil {
		return false, fmt.Sprintf("failed to verify github user organization membership: %v", err)
	}
	if !ok {
		return false, fmt.Sprintf("github user %q is not a member of organization %q", username, owner)
	}
	return true, ""
}

func (m *Manager) mergeIssuePullRequest(event *IssueCommentEvent) {
	branch := buildBranchName(m.cfg.BranchPrefix, event.IssueNumber, event.IssueTitle)
	accepted := buildMergeAcceptedComment(event, branch, m.cfg)
	_ = m.reporter.PostIssueComment(m.ctx, event.Repository, event.IssueNumber, accepted)

	pr, err := m.reporter.FindOpenPullRequestByHead(m.ctx, event.Repository, branch)
	if err != nil {
		m.postMergeCompletionComment(event, branch, nil, nil, err, nil)
		return
	}
	if pr == nil {
		m.postMergeCompletionComment(event, branch, nil, nil, fmt.Errorf("no open pull request found for branch %q", branch), nil)
		return
	}

	title := fmt.Sprintf("Merge issue #%d: %s", event.IssueNumber, event.IssueTitle)
	message := fmt.Sprintf("Merged from issue %s by @%s via k13d issue automation.", event.IssueURL, event.CommentAuthor)
	merge, err := m.reporter.MergePullRequest(m.ctx, event.Repository, pr.Number, m.cfg.MergeMethod, title, message)
	var closeErr error
	if err == nil {
		closeErr = m.reporter.CloseIssue(m.ctx, event.Repository, event.IssueNumber, "completed")
	}
	m.postMergeCompletionComment(event, branch, pr, merge, err, closeErr)
}

func (m *Manager) reviewIssuePullRequest(event *IssueCommentEvent) {
	branch := buildBranchName(m.cfg.BranchPrefix, event.IssueNumber, event.IssueTitle)
	accepted := buildReviewAcceptedComment(event, branch, m.cfg)
	_ = m.reporter.PostIssueComment(m.ctx, event.Repository, event.IssueNumber, accepted)

	pr, err := m.reporter.FindOpenPullRequestByHead(m.ctx, event.Repository, branch)
	if err != nil {
		m.postReviewCompletionComment(event, branch, nil, "", err)
		return
	}
	if pr == nil {
		m.postReviewCompletionComment(event, branch, nil, "", fmt.Errorf("no open pull request found for branch %q", branch))
		return
	}

	job := &Job{
		Repository:        event.Repository,
		IssueNumber:       event.IssueNumber,
		IssueTitle:        event.IssueTitle,
		IssueBody:         event.IssueBody,
		IssueURL:          event.IssueURL,
		IssueAuthor:       event.IssueAuthor,
		Branch:            branch,
		PullRequestNumber: pr.Number,
		PullRequestURL:    pr.URL,
	}
	reviewLog, err := m.executor.Review(m.ctx, job, branch)
	if err == nil {
		result := &ExecutionResult{Branch: branch, ReviewLog: reviewLog}
		if reviewErr := m.reporter.CreatePullRequestReview(m.ctx, event.Repository, pr.Number, buildReviewBody(job, result, m.cfg)); reviewErr != nil {
			err = fmt.Errorf("failed to create pull request review: %w", reviewErr)
		}
	}
	m.postReviewCompletionComment(event, branch, pr, reviewLog, err)
}

func (m *Manager) postMergeCompletionComment(event *IssueCommentEvent, branch string, pr *PullRequestInfo, merge *PullRequestMergeInfo, err, closeErr error) {
	body := buildMergeCompletionComment(event, branch, pr, merge, err, closeErr, m.cfg)
	_ = m.reporter.PostIssueComment(m.ctx, event.Repository, event.IssueNumber, body)
}

func (m *Manager) postReviewCompletionComment(event *IssueCommentEvent, branch string, pr *PullRequestInfo, reviewLog string, err error) {
	body := buildReviewCompletionComment(event, branch, pr, reviewLog, err, m.cfg)
	_ = m.reporter.PostIssueComment(m.ctx, event.Repository, event.IssueNumber, body)
}

func issueKey(repo string, issueNumber int) string {
	return fmt.Sprintf("%s#%d", repo, issueNumber)
}

func repositoryOwner(repo string) string {
	owner, _, ok := strings.Cut(strings.TrimSpace(repo), "/")
	if !ok {
		return ""
	}
	return strings.TrimSpace(owner)
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

func buildAcceptedIssueComment(job *Job, mentions []string, cfg config.GitHubAutomationConfig) string {
	var b strings.Builder
	if koreanReview(cfg) {
		fmt.Fprintf(&b, "## k13d 이슈 자동화 접수\n\n")
		if len(mentions) > 0 {
			fmt.Fprintf(&b, "%s\n\n", strings.Join(mentions, " "))
		}
		fmt.Fprintf(&b, "- 이슈: #%d\n", job.IssueNumber)
		fmt.Fprintf(&b, "- 작성자: @%s\n", job.IssueAuthor)
		fmt.Fprintf(&b, "- 상태: 자동화 작업을 큐에 등록했습니다.\n")
		b.WriteString("- 리뷰 언어: 이슈 검토와 코드 리뷰는 한국어로 진행합니다.\n")
		b.WriteString("\n작업이 끝나면 PR, CI 결과, 배포 확인 링크가 포함된 완료 코멘트를 남깁니다.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "## k13d issue automation accepted\n\n")
	if len(mentions) > 0 {
		fmt.Fprintf(&b, "%s\n\n", strings.Join(mentions, " "))
	}
	fmt.Fprintf(&b, "- Issue: #%d\n", job.IssueNumber)
	fmt.Fprintf(&b, "- Author: @%s\n", job.IssueAuthor)
	fmt.Fprintf(&b, "- Status: queued for automation\n")
	b.WriteString("\nWhen the job finishes, k13d will comment with the PR, CI result, and preview deployment link.\n")
	return b.String()
}

func buildMergeAcceptedComment(event *IssueCommentEvent, branch string, cfg config.GitHubAutomationConfig) string {
	var b strings.Builder
	if koreanReview(cfg) {
		fmt.Fprintf(&b, "## k13d 병합 요청 접수\n\n")
		fmt.Fprintf(&b, "- 이슈: #%d\n", event.IssueNumber)
		fmt.Fprintf(&b, "- 요청자: @%s\n", event.CommentAuthor)
		fmt.Fprintf(&b, "- 대상 브랜치: `%s`\n", branch)
		fmt.Fprintf(&b, "- 병합 방식: `%s`\n", normalizeMergeMethod(cfg.MergeMethod))
		b.WriteString("\n연결된 open PR을 찾아 GitHub 병합 API로 처리합니다. 브랜치 보호 규칙이나 CI가 막으면 실패 코멘트를 남깁니다.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "## k13d merge request accepted\n\n")
	fmt.Fprintf(&b, "- Issue: #%d\n", event.IssueNumber)
	fmt.Fprintf(&b, "- Requested by: @%s\n", event.CommentAuthor)
	fmt.Fprintf(&b, "- Target branch: `%s`\n", branch)
	fmt.Fprintf(&b, "- Merge method: `%s`\n", normalizeMergeMethod(cfg.MergeMethod))
	b.WriteString("\nk13d will find the linked open PR and ask GitHub to merge it. Branch protection or pending CI can still block the merge.\n")
	return b.String()
}

func buildReviewAcceptedComment(event *IssueCommentEvent, branch string, cfg config.GitHubAutomationConfig) string {
	var b strings.Builder
	if koreanReview(cfg) {
		fmt.Fprintf(&b, "## k13d 코드 리뷰 요청 접수\n\n")
		fmt.Fprintf(&b, "- 이슈: #%d\n", event.IssueNumber)
		fmt.Fprintf(&b, "- 요청자: @%s\n", event.CommentAuthor)
		fmt.Fprintf(&b, "- 대상 브랜치: `%s`\n", branch)
		b.WriteString("\n연결된 open PR을 찾아 설정된 `review_command`로 Codex 리뷰를 실행하고 PR Review로 남깁니다.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "## k13d code review request accepted\n\n")
	fmt.Fprintf(&b, "- Issue: #%d\n", event.IssueNumber)
	fmt.Fprintf(&b, "- Requested by: @%s\n", event.CommentAuthor)
	fmt.Fprintf(&b, "- Target branch: `%s`\n", branch)
	b.WriteString("\nk13d will find the linked open PR, run the configured review command, and post a pull request review.\n")
	return b.String()
}

func buildMergeCompletionComment(event *IssueCommentEvent, branch string, pr *PullRequestInfo, merge *PullRequestMergeInfo, mergeErr, closeErr error, cfg config.GitHubAutomationConfig) string {
	var b strings.Builder
	if koreanReview(cfg) {
		if mergeErr != nil {
			fmt.Fprintf(&b, "## k13d 병합 실패\n\n")
			fmt.Fprintf(&b, "- 이슈: #%d\n", event.IssueNumber)
			fmt.Fprintf(&b, "- 대상 브랜치: `%s`\n", branch)
			if pr != nil {
				fmt.Fprintf(&b, "- Pull Request: %s\n", pr.URL)
			}
			fmt.Fprintf(&b, "- 오류: `%s`\n", RedactGitHubSecrets(mergeErr.Error(), cfg))
			b.WriteString("\nCI 상태, branch protection, reviewer 승인 조건을 확인한 뒤 이슈에 다시 `k13d merge 해줘`라고 남기면 재시도할 수 있습니다.\n")
			return b.String()
		}
		fmt.Fprintf(&b, "## k13d 병합 완료\n\n")
		fmt.Fprintf(&b, "- 이슈: #%d\n", event.IssueNumber)
		if pr != nil {
			fmt.Fprintf(&b, "- Pull Request: %s\n", pr.URL)
		}
		if merge != nil && merge.SHA != "" {
			fmt.Fprintf(&b, "- 병합 커밋: `%s`\n", merge.SHA)
		}
		if closeErr != nil {
			fmt.Fprintf(&b, "\nmain 반영은 완료되었지만 이슈 닫기는 실패했습니다. 오류: `%s`\n", RedactGitHubSecrets(closeErr.Error(), cfg))
			b.WriteString("권한을 확인한 뒤 필요하면 이슈를 수동으로 닫아주세요.\n")
			return b.String()
		}
		b.WriteString("\nmain 반영이 완료되었고 이슈를 완료 상태로 닫았습니다.\n")
		return b.String()
	}

	if mergeErr != nil {
		fmt.Fprintf(&b, "## k13d merge failed\n\n")
		fmt.Fprintf(&b, "- Issue: #%d\n", event.IssueNumber)
		fmt.Fprintf(&b, "- Target branch: `%s`\n", branch)
		if pr != nil {
			fmt.Fprintf(&b, "- Pull request: %s\n", pr.URL)
		}
		fmt.Fprintf(&b, "- Error: `%s`\n", RedactGitHubSecrets(mergeErr.Error(), cfg))
		b.WriteString("\nCheck CI, branch protection, and review requirements, then comment `k13d merge` to retry.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "## k13d merge completed\n\n")
	fmt.Fprintf(&b, "- Issue: #%d\n", event.IssueNumber)
	if pr != nil {
		fmt.Fprintf(&b, "- Pull request: %s\n", pr.URL)
	}
	if merge != nil && merge.SHA != "" {
		fmt.Fprintf(&b, "- Merge commit: `%s`\n", merge.SHA)
	}
	if closeErr != nil {
		fmt.Fprintf(&b, "\nMerged into main, but closing the issue failed: `%s`\n", RedactGitHubSecrets(closeErr.Error(), cfg))
		b.WriteString("Please check token permissions and close the issue manually if needed.\n")
		return b.String()
	}
	b.WriteString("\nMerged into main and closed the issue as completed.\n")
	return b.String()
}

func buildReviewCompletionComment(event *IssueCommentEvent, branch string, pr *PullRequestInfo, reviewLog string, reviewErr error, cfg config.GitHubAutomationConfig) string {
	var b strings.Builder
	if koreanReview(cfg) {
		if reviewErr != nil {
			fmt.Fprintf(&b, "## k13d 코드 리뷰 실패\n\n")
			fmt.Fprintf(&b, "- 이슈: #%d\n", event.IssueNumber)
			fmt.Fprintf(&b, "- 대상 브랜치: `%s`\n", branch)
			if pr != nil {
				fmt.Fprintf(&b, "- Pull Request: %s\n", pr.URL)
			}
			fmt.Fprintf(&b, "- 오류: `%s`\n", RedactGitHubSecrets(reviewErr.Error(), cfg))
			b.WriteString("\n설정된 `review_command`, Codex 인증 상태, PR 상태를 확인한 뒤 이슈에 다시 `k13d 코드리뷰 해줘`라고 남기면 재시도할 수 있습니다.\n")
			return b.String()
		}
		fmt.Fprintf(&b, "## k13d 코드 리뷰 완료\n\n")
		fmt.Fprintf(&b, "- 이슈: #%d\n", event.IssueNumber)
		if pr != nil {
			fmt.Fprintf(&b, "- Pull Request: %s\n", pr.URL)
		}
		if strings.TrimSpace(reviewLog) != "" {
			b.WriteString("\nPR Review에 Codex 코드 리뷰 결과를 남겼습니다.\n")
		}
		return b.String()
	}

	if reviewErr != nil {
		fmt.Fprintf(&b, "## k13d code review failed\n\n")
		fmt.Fprintf(&b, "- Issue: #%d\n", event.IssueNumber)
		fmt.Fprintf(&b, "- Target branch: `%s`\n", branch)
		if pr != nil {
			fmt.Fprintf(&b, "- Pull request: %s\n", pr.URL)
		}
		fmt.Fprintf(&b, "- Error: `%s`\n", RedactGitHubSecrets(reviewErr.Error(), cfg))
		b.WriteString("\nCheck review_command, Codex authentication, and PR state, then comment `k13d review` to retry.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "## k13d code review completed\n\n")
	fmt.Fprintf(&b, "- Issue: #%d\n", event.IssueNumber)
	if pr != nil {
		fmt.Fprintf(&b, "- Pull request: %s\n", pr.URL)
	}
	return b.String()
}

func buildIssueComment(job *Job, result *ExecutionResult, execErr error, cfg config.GitHubAutomationConfig) string {
	var b strings.Builder
	if koreanReview(cfg) {
		return buildKoreanIssueComment(job, result, execErr)
	}
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

func buildKoreanIssueComment(job *Job, result *ExecutionResult, execErr error) string {
	var b strings.Builder
	if execErr != nil {
		fmt.Fprintf(&b, "## k13d 이슈 자동화 실패\n\n")
		fmt.Fprintf(&b, "- 이슈: #%d\n", job.IssueNumber)
		fmt.Fprintf(&b, "- 저장소: `%s`\n", job.Repository)
		fmt.Fprintf(&b, "- 오류: `%s`\n", execErr.Error())
		if result != nil && strings.TrimSpace(result.DevelopmentLog) != "" {
			b.WriteString("\n### 개발 명령 출력\n\n```text\n")
			b.WriteString(result.DevelopmentLog)
			b.WriteString("\n```\n")
		}
		return b.String()
	}

	fmt.Fprintf(&b, "## k13d 이슈 자동화 완료\n\n")
	fmt.Fprintf(&b, "- 이슈: #%d\n", job.IssueNumber)
	fmt.Fprintf(&b, "- 브랜치: `%s`\n", job.Branch)
	if job.CommitSHA != "" {
		fmt.Fprintf(&b, "- 커밋: `%s`\n", job.CommitSHA)
	}
	if job.PullRequestURL != "" {
		fmt.Fprintf(&b, "- Pull Request: %s\n", job.PullRequestURL)
	}
	if job.CIConclusion != "" {
		fmt.Fprintf(&b, "- CI 결과: `%s`\n", job.CIConclusion)
	}
	if job.CIURL != "" {
		fmt.Fprintf(&b, "- CI 상세: %s\n", job.CIURL)
	}
	if job.PreviewURL != "" {
		fmt.Fprintf(&b, "- 배포 확인 링크: %s\n", job.PreviewURL)
	}
	if !job.HasChanges {
		b.WriteString("- 결과: 파일 변경이 생성되지 않았습니다.\n")
	}
	if strings.TrimSpace(job.DiffStat) != "" {
		b.WriteString("\n### 변경 요약\n\n```text\n")
		b.WriteString(job.DiffStat)
		b.WriteString("\n```\n")
	}
	if strings.TrimSpace(job.ReviewLog) != "" {
		b.WriteString("\n### 코드 리뷰 요약\n\n```text\n")
		b.WriteString(job.ReviewLog)
		b.WriteString("\n```\n")
	}
	if strings.TrimSpace(job.DeploymentLog) != "" {
		b.WriteString("\n### 배포 출력\n\n```text\n")
		b.WriteString(job.DeploymentLog)
		b.WriteString("\n```\n")
	}
	if len(job.Warnings) > 0 {
		b.WriteString("\n### 경고\n")
		for _, warning := range job.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}
	return b.String()
}

func buildPullRequestVerificationComment(job *Job, cfg config.GitHubAutomationConfig) string {
	if job == nil || job.PullRequestNumber <= 0 {
		return ""
	}
	hasCI := strings.TrimSpace(job.CIStatus) != "" || strings.TrimSpace(job.CIConclusion) != "" || strings.TrimSpace(job.CIURL) != ""
	hasPreview := strings.TrimSpace(job.PreviewURL) != ""
	if !hasCI && !hasPreview {
		return ""
	}

	var b strings.Builder
	if koreanReview(cfg) {
		b.WriteString("## k13d CI/CD 확인 경로\n\n")
		fmt.Fprintf(&b, "- 이슈: #%d\n", job.IssueNumber)
		if strings.TrimSpace(job.Branch) != "" {
			fmt.Fprintf(&b, "- 브랜치: `%s`\n", job.Branch)
		}
		if hasCI {
			fmt.Fprintf(&b, "- CI 상태: `%s/%s`\n", valueOrDefault(job.CIStatus, "completed"), valueOrDefault(job.CIConclusion, "success"))
		}
		if strings.TrimSpace(job.CIURL) != "" {
			fmt.Fprintf(&b, "- CI 로그: %s\n", job.CIURL)
		}
		if hasPreview {
			fmt.Fprintf(&b, "- 배포 확인 링크: %s\n", job.PreviewURL)
		}
		b.WriteString("\nCI/CD가 끝난 뒤 PR 화면에서 바로 검증할 수 있도록 자동으로 남긴 코멘트입니다.\n")
		return b.String()
	}

	b.WriteString("## k13d CI/CD verification\n\n")
	fmt.Fprintf(&b, "- Issue: #%d\n", job.IssueNumber)
	if strings.TrimSpace(job.Branch) != "" {
		fmt.Fprintf(&b, "- Branch: `%s`\n", job.Branch)
	}
	if hasCI {
		fmt.Fprintf(&b, "- CI status: `%s/%s`\n", valueOrDefault(job.CIStatus, "completed"), valueOrDefault(job.CIConclusion, "success"))
	}
	if strings.TrimSpace(job.CIURL) != "" {
		fmt.Fprintf(&b, "- CI logs: %s\n", job.CIURL)
	}
	if hasPreview {
		fmt.Fprintf(&b, "- Preview URL: %s\n", job.PreviewURL)
	}
	b.WriteString("\nPosted automatically after CI/CD so reviewers can verify the PR from this page.\n")
	return b.String()
}

func valueOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func buildPullRequestBody(job *Job, result *ExecutionResult, cfg config.GitHubAutomationConfig) string {
	var b strings.Builder
	if koreanReview(cfg) {
		fmt.Fprintf(&b, "## 요약\n")
		fmt.Fprintf(&b, "- GitHub 이슈 #%d에서 자동 생성된 변경입니다.\n", job.IssueNumber)
		fmt.Fprintf(&b, "- 연결된 이슈: Closes #%d\n", job.IssueNumber)
		fmt.Fprintf(&b, "- 원본 이슈: %s\n", job.IssueURL)
		b.WriteString("- 이슈 검토와 코드 리뷰는 한국어로 진행합니다.\n")
		if strings.TrimSpace(result.DiffStat) != "" {
			b.WriteString("\n## 변경 요약\n\n```text\n")
			b.WriteString(result.DiffStat)
			b.WriteString("\n```\n")
		}
		if strings.TrimSpace(result.ReviewLog) != "" {
			b.WriteString("\n## 리뷰 메모\n\n```text\n")
			b.WriteString(result.ReviewLog)
			b.WriteString("\n```\n")
		}
		return b.String()
	}
	fmt.Fprintf(&b, "## Summary\n")
	fmt.Fprintf(&b, "- automated from GitHub issue #%d\n", job.IssueNumber)
	fmt.Fprintf(&b, "- linked issue: Closes #%d\n", job.IssueNumber)
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

func buildReviewBody(job *Job, result *ExecutionResult, cfg config.GitHubAutomationConfig) string {
	var b strings.Builder
	if koreanReview(cfg) {
		fmt.Fprintf(&b, "## 자동 코드 리뷰\n\n")
		fmt.Fprintf(&b, "이 리뷰는 이슈 #%d 자동화 결과에 대해 한국어로 작성되었습니다.\n\n", job.IssueNumber)
		if strings.TrimSpace(result.ReviewLog) != "" {
			b.WriteString(result.ReviewLog)
		} else {
			b.WriteString("별도의 리뷰 명령이 설정되어 있지 않아 자동 리뷰 요약은 생성되지 않았습니다.")
		}
		return b.String()
	}
	fmt.Fprintf(&b, "Automated review for issue #%d.\n\n", job.IssueNumber)
	if strings.TrimSpace(result.ReviewLog) != "" {
		b.WriteString(result.ReviewLog)
	} else {
		b.WriteString("No separate review command was configured.")
	}
	return b.String()
}

func koreanReview(cfg config.GitHubAutomationConfig) bool {
	language := strings.ToLower(strings.TrimSpace(cfg.ReviewLanguage))
	return language == "" || language == "ko" || strings.HasPrefix(language, "ko-") || strings.Contains(language, "korean")
}

func isMergeCommand(body string) bool {
	text := strings.ToLower(strings.TrimSpace(body))
	if text == "" || !strings.Contains(text, "k13d") {
		return false
	}
	for _, negative := range []string{"don't merge", "do not merge", "not merge", "머지하지", "병합하지"} {
		if strings.Contains(text, negative) {
			return false
		}
	}
	for _, positive := range []string{"merge", "머지", "병합", "main에 반영", "main 반영"} {
		if strings.Contains(text, positive) {
			return true
		}
	}
	return false
}

func isReviewCommand(body string) bool {
	text := strings.ToLower(strings.TrimSpace(body))
	if text == "" || !strings.Contains(text, "k13d") {
		return false
	}
	for _, negative := range []string{"don't review", "do not review", "not review", "리뷰하지", "검토하지"} {
		if strings.Contains(text, negative) {
			return false
		}
	}
	for _, positive := range []string{"review", "code review", "리뷰", "코드리뷰", "검토"} {
		if strings.Contains(text, positive) {
			return true
		}
	}
	return false
}
