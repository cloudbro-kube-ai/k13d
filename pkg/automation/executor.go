package automation

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

type Executor interface {
	Execute(ctx context.Context, job *Job) (*ExecutionResult, error)
	DeployPreview(ctx context.Context, job *Job, result *ExecutionResult) (*PreviewDeployment, error)
}

type DefaultExecutor struct {
	cfg      config.GitHubAutomationConfig
	repoPath string
	now      func() time.Time
}

func NewDefaultExecutor(cfg config.GitHubAutomationConfig, repoPath string) *DefaultExecutor {
	return &DefaultExecutor{
		cfg:      cfg,
		repoPath: repoPath,
		now:      time.Now,
	}
}

func (e *DefaultExecutor) Execute(ctx context.Context, job *Job) (*ExecutionResult, error) {
	if len(e.cfg.DevelopmentCommand) == 0 {
		return nil, fmt.Errorf("github automation development_command is not configured")
	}

	repoPath := strings.TrimSpace(e.repoPath)
	if repoPath == "" {
		return nil, fmt.Errorf("github automation repo_path is not configured")
	}
	worktreeRoot := strings.TrimSpace(e.cfg.WorktreeRoot)
	if worktreeRoot == "" {
		worktreeRoot = config.DefaultGitHubAutomationWorktreeRoot()
	}
	if err := os.MkdirAll(worktreeRoot, 0o750); err != nil {
		return nil, err
	}

	branch := buildBranchName(e.cfg.BranchPrefix, job.IssueNumber, job.IssueTitle)
	worktreePath := filepath.Join(worktreeRoot, fmt.Sprintf("issue-%d-%d", job.IssueNumber, e.now().Unix()))
	baseRef := e.resolveBaseRef(ctx)
	if _, err := e.runCommand(ctx, repoPath, []string{"git", "worktree", "add", "--force", "-B", branch, worktreePath, baseRef}, nil); err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	cleanup := func() {
		if e.cfg.CleanupWorktrees {
			_, _ = e.runCommand(context.Background(), repoPath, []string{"git", "worktree", "remove", "--force", worktreePath}, nil)
		}
	}

	placeholders := jobPlaceholders(job, repoPath, worktreePath, branch, e.cfg.BaseBranch)
	devLog, err := e.runCommand(ctx, worktreePath, expandArgs(e.cfg.DevelopmentCommand, placeholders), placeholders)
	if err != nil {
		cleanup()
		return &ExecutionResult{
			Branch:         branch,
			WorktreePath:   worktreePath,
			DevelopmentLog: devLog,
		}, fmt.Errorf("development command failed: %w", err)
	}

	reviewLog := ""
	if len(e.cfg.ReviewCommand) > 0 {
		reviewLog, err = e.runCommand(ctx, worktreePath, expandArgs(e.cfg.ReviewCommand, placeholders), placeholders)
		if err != nil {
			cleanup()
			return &ExecutionResult{
				Branch:         branch,
				WorktreePath:   worktreePath,
				DevelopmentLog: devLog,
				ReviewLog:      reviewLog,
			}, fmt.Errorf("review command failed: %w", err)
		}
	}

	statusOut, err := e.runCommand(ctx, worktreePath, []string{"git", "status", "--porcelain"}, nil)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("git status: %w", err)
	}
	diffStat, _ := e.runCommand(ctx, worktreePath, []string{"git", "diff", "--stat"}, nil)
	hasChanges := strings.TrimSpace(statusOut) != ""
	commitSHA := ""

	if hasChanges && e.cfg.AutoCommit {
		if _, err := e.runCommand(ctx, worktreePath, []string{"git", "add", "-A"}, nil); err != nil {
			cleanup()
			return nil, fmt.Errorf("git add: %w", err)
		}
		commitMsg := buildCommitMessage(job)
		if _, err := e.runCommand(ctx, worktreePath, []string{"git", "commit", "-m", commitMsg}, nil); err != nil {
			cleanup()
			return nil, fmt.Errorf("git commit: %w", err)
		}
	}

	if hasChanges {
		if sha, err := e.runCommand(ctx, worktreePath, []string{"git", "rev-parse", "HEAD"}, nil); err == nil {
			commitSHA = strings.TrimSpace(sha)
		}
		if e.cfg.AutoPush {
			remote := strings.TrimSpace(e.cfg.Remote)
			if remote == "" {
				remote = "origin"
			}
			if _, err := e.runCommand(ctx, worktreePath, []string{"git", "push", "-u", remote, branch}, nil); err != nil {
				cleanup()
				return nil, fmt.Errorf("git push: %w", err)
			}
		}
	}

	cleanup()
	return &ExecutionResult{
		Branch:         branch,
		WorktreePath:   worktreePath,
		CommitSHA:      commitSHA,
		HasChanges:     hasChanges,
		DevelopmentLog: devLog,
		ReviewLog:      reviewLog,
		DiffStat:       strings.TrimSpace(diffStat),
	}, nil
}

func (e *DefaultExecutor) DeployPreview(ctx context.Context, job *Job, result *ExecutionResult) (*PreviewDeployment, error) {
	if !e.cfg.AutoDeployPreview || len(e.cfg.DeployPreviewCommand) == 0 || result == nil {
		return nil, nil
	}
	worktreePath := strings.TrimSpace(result.WorktreePath)
	if worktreePath == "" {
		return nil, fmt.Errorf("preview deployment requires a worktree path")
	}

	slug := buildPreviewSlug(result.Branch)
	previewPath := previewPathForSlug(e.cfg.PreviewPathPrefix, slug)
	publicURL := previewPublicURL(e.cfg.PreviewURLBase, previewPath)
	placeholders := jobPlaceholders(job, e.repoPath, worktreePath, result.Branch, e.cfg.BaseBranch)
	placeholders["preview_slug"] = slug
	placeholders["preview_path"] = previewPath
	placeholders["preview_url"] = publicURL

	log, err := e.runCommand(ctx, worktreePath, expandArgs(e.cfg.DeployPreviewCommand, placeholders), placeholders)
	if err != nil {
		return &PreviewDeployment{Slug: slug, PublicURL: publicURL, Log: log}, fmt.Errorf("preview deployment command failed: %w", err)
	}
	targetURL, outputURL := parsePreviewDeploymentOutput(log)
	if outputURL != "" {
		publicURL = outputURL
	}
	if targetURL == "" && publicURL == "" {
		return &PreviewDeployment{Slug: slug, Log: log}, fmt.Errorf("preview deployment did not report a target or public URL")
	}
	return &PreviewDeployment{
		Slug:      slug,
		PublicURL: publicURL,
		TargetURL: targetURL,
		Log:       log,
	}, nil
}

func (e *DefaultExecutor) resolveBaseRef(ctx context.Context) string {
	baseBranch := strings.TrimSpace(e.cfg.BaseBranch)
	if baseBranch == "" {
		baseBranch = "main"
	}
	remote := strings.TrimSpace(e.cfg.Remote)
	if remote == "" {
		remote = "origin"
	}
	_, _ = e.runCommand(ctx, e.repoPath, []string{"git", "fetch", remote, baseBranch}, nil)

	remoteRef := remote + "/" + baseBranch
	if _, err := e.runCommand(ctx, e.repoPath, []string{"git", "rev-parse", "--verify", remoteRef}, nil); err == nil {
		return remoteRef
	}
	if _, err := e.runCommand(ctx, e.repoPath, []string{"git", "rev-parse", "--verify", baseBranch}, nil); err == nil {
		return baseBranch
	}
	return "HEAD"
}

func (e *DefaultExecutor) runCommand(ctx context.Context, dir string, args []string, placeholders map[string]string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("empty command")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) // #nosec G204 -- commands come from local admin-only github_automation config.
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), buildCommandEnv(placeholders)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	errOutput := strings.TrimSpace(stderr.String())
	combined := strings.TrimSpace(strings.Join(compactStrings(output, errOutput), "\n"))
	if err != nil {
		if combined == "" {
			combined = err.Error()
		}
		return truncateLog(combined), err
	}
	return truncateLog(combined), nil
}

func compactStrings(values ...string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func buildCommandEnv(placeholders map[string]string) []string {
	if len(placeholders) == 0 {
		return nil
	}
	keys := []string{
		"issue_number", "issue_title", "issue_body", "issue_url",
		"issue_author", "repository", "branch", "worktree",
		"repo_path", "base_branch", "preview_slug", "preview_path", "preview_url",
	}
	env := make([]string, 0, len(keys))
	for _, key := range keys {
		if value := placeholders[key]; value != "" {
			env = append(env, "K13D_GHA_"+strings.ToUpper(strings.ReplaceAll(key, "-", "_"))+"="+value)
		}
	}
	return env
}

func expandArgs(args []string, placeholders map[string]string) []string {
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		expanded = append(expanded, applyPlaceholders(arg, placeholders))
	}
	return expanded
}

func applyPlaceholders(text string, placeholders map[string]string) string {
	for key, value := range placeholders {
		text = strings.ReplaceAll(text, "{"+key+"}", value)
	}
	return text
}

func jobPlaceholders(job *Job, repoPath, worktreePath, branch, baseBranch string) map[string]string {
	return map[string]string{
		"issue_number": fmt.Sprintf("%d", job.IssueNumber),
		"issue_title":  job.IssueTitle,
		"issue_body":   job.IssueBody,
		"issue_url":    job.IssueURL,
		"issue_author": job.IssueAuthor,
		"repository":   job.Repository,
		"repo_path":    repoPath,
		"worktree":     worktreePath,
		"branch":       branch,
		"base_branch":  baseBranch,
	}
}

func buildCommitMessage(job *Job) string {
	title := strings.TrimSpace(job.IssueTitle)
	if title == "" {
		return fmt.Sprintf("feat: automate issue #%d", job.IssueNumber)
	}
	title = strings.ReplaceAll(title, "\n", " ")
	if len(title) > 64 {
		title = title[:64]
	}
	return fmt.Sprintf("feat: issue #%d %s", job.IssueNumber, title)
}

func buildBranchName(prefix string, issueNumber int, title string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "codex/issue-"
	}
	slug := sanitizeBranchToken(title)
	if slug == "" {
		slug = "work"
	}
	branch := fmt.Sprintf("%s%d-%s", prefix, issueNumber, slug)
	if len(branch) > 120 {
		branch = branch[:120]
	}
	return strings.TrimSuffix(branch, "-")
}

func buildPreviewSlug(branch string) string {
	slug := sanitizeBranchToken(strings.ReplaceAll(branch, "/", "-"))
	if slug == "" {
		return "preview"
	}
	return slug
}

func previewPathForSlug(prefix, slug string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "/previews"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	prefix = strings.TrimRight(prefix, "/")
	return prefix + "/" + slug + "/"
}

func previewPublicURL(baseURL, previewPath string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return previewPath
	}
	return baseURL + previewPath
}

func parsePreviewDeploymentOutput(output string) (targetURL, publicURL string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			key, value, ok = strings.Cut(line, ":")
		}
		if !ok {
			continue
		}
		key = strings.ToUpper(strings.TrimSpace(key))
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if value == "" {
			continue
		}
		switch key {
		case "K13D_PREVIEW_TARGET", "PREVIEW_TARGET", "TARGET_URL":
			if isHTTPURL(value) {
				targetURL = value
			}
		case "K13D_PREVIEW_URL", "PREVIEW_URL", "PUBLIC_URL":
			if strings.HasPrefix(value, "/") || isHTTPURL(value) {
				publicURL = value
			}
		}
	}
	return targetURL, publicURL
}

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

var branchSanitizer = regexp.MustCompile(`[^a-z0-9._/-]+`)

func sanitizeBranchToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = branchSanitizer.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-./")
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-'
	})
	if len(parts) > 6 {
		parts = parts[:6]
	}
	return strings.Join(parts, "-")
}

func truncateLog(text string) string {
	const limit = 12000
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "\n... (truncated)"
}

func DetectGitHubRepository(ctx context.Context, repoPath, remote string) (string, error) {
	if strings.TrimSpace(repoPath) == "" {
		return "", fmt.Errorf("repo path is empty")
	}
	if strings.TrimSpace(remote) == "" {
		remote = "origin"
	}
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "get-url", remote) // #nosec G204 -- repo path and remote come from local admin config.
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return parseGitHubRemoteURL(strings.TrimSpace(string(out)))
}

func parseGitHubRemoteURL(remoteURL string) (string, error) {
	remoteURL = strings.TrimSpace(remoteURL)
	switch {
	case strings.HasPrefix(remoteURL, "git@github.com:"):
		remoteURL = strings.TrimPrefix(remoteURL, "git@github.com:")
	case strings.HasPrefix(remoteURL, "https://github.com/"):
		remoteURL = strings.TrimPrefix(remoteURL, "https://github.com/")
	case strings.HasPrefix(remoteURL, "http://github.com/"):
		remoteURL = strings.TrimPrefix(remoteURL, "http://github.com/")
	default:
		return "", fmt.Errorf("unsupported github remote: %s", remoteURL)
	}
	remoteURL = strings.TrimSuffix(remoteURL, ".git")
	parts := strings.Split(remoteURL, "/")
	parts = slices.DeleteFunc(parts, func(part string) bool { return strings.TrimSpace(part) == "" })
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid github remote path: %s", remoteURL)
	}
	return parts[0] + "/" + parts[1], nil
}
