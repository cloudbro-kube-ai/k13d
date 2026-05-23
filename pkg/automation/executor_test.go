package automation

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func TestDefaultExecutorReclaimsStaleIssueWorktree(t *testing.T) {
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo")
	worktreeRoot := filepath.Join(tmp, "worktrees")
	staleWorktreePath := filepath.Join(worktreeRoot, "issue-103-100")

	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		t.Fatal(err)
	}
	runTestCommand(t, repoPath, "git", "init")
	runTestCommand(t, repoPath, "git", "checkout", "-B", "main")
	runTestCommand(t, repoPath, "git", "config", "user.email", "k13d@example.invalid")
	runTestCommand(t, repoPath, "git", "config", "user.name", "k13d test")
	runTestCommand(t, repoPath, "git", "commit", "--allow-empty", "-m", "initial")

	cfg := config.GitHubAutomationConfig{
		BaseBranch:         "main",
		WorktreeRoot:       worktreeRoot,
		BranchPrefix:       "codex/issue-",
		DevelopmentCommand: []string{"git", "status", "--short"},
		CleanupWorktrees:   true,
	}
	job := &Job{
		IssueNumber: 103,
		IssueTitle:  "Test automation",
		Repository:  "cloudbro-kube-ai/k13d",
	}
	branch := buildBranchName(cfg.BranchPrefix, job.IssueNumber, job.IssueTitle)
	runTestCommand(t, repoPath, "git", "worktree", "add", "--force", "-B", branch, staleWorktreePath, "main")

	executor := NewDefaultExecutor(cfg, repoPath)
	executor.now = func() time.Time {
		return time.Unix(200, 0)
	}
	result, err := executor.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Branch != branch {
		t.Fatalf("Branch = %q, want %q", result.Branch, branch)
	}
	if _, err := os.Stat(staleWorktreePath); !os.IsNotExist(err) {
		t.Fatalf("stale worktree still exists or stat failed: %v", err)
	}
}

func TestParseGitWorktreeList(t *testing.T) {
	entries := parseGitWorktreeList(`worktree /repo
HEAD abc123
branch refs/heads/main

worktree /tmp/worktrees/issue-7-123
HEAD def456
branch refs/heads/codex/issue-7-fix
`)
	if len(entries) != 2 {
		t.Fatalf("entries = %#v, want 2 entries", entries)
	}
	if entries[1].path != "/tmp/worktrees/issue-7-123" {
		t.Fatalf("path = %q", entries[1].path)
	}
	if entries[1].branch != "refs/heads/codex/issue-7-fix" {
		t.Fatalf("branch = %q", entries[1].branch)
	}
}

func runTestCommand(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(out))
	}
}
