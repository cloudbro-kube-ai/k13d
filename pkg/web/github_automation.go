package web

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/cloudbro-kube-ai/k13d/pkg/automation"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func (s *Server) handleGitHubAutomationWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	if s.automation == nil || !s.automation.Enabled() {
		WriteError(w, NewAPIError(ErrCodeNotFound, "GitHub automation is disabled"))
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Failed to read webhook payload"))
		return
	}
	if !automation.VerifyGitHubSignature(s.cfg.GitHub.WebhookSecret, payload, r.Header.Get("X-Hub-Signature-256")) {
		WriteError(w, NewAPIError(ErrCodeUnauthorized, "Invalid GitHub webhook signature"))
		return
	}

	eventName := r.Header.Get("X-GitHub-Event")
	var result automation.QueueResult
	switch eventName {
	case "issues":
		event, err := automation.ParseIssueEvent(eventName, payload)
		if err != nil {
			WriteError(w, NewAPIError(ErrCodeBadRequest, err.Error()))
			return
		}
		result = s.automation.QueueIssueEvent(event)
	case "issue_comment":
		event, err := automation.ParseIssueCommentEvent(eventName, payload)
		if err != nil {
			WriteError(w, NewAPIError(ErrCodeBadRequest, err.Error()))
			return
		}
		result = s.automation.HandleIssueCommentEvent(event)
	default:
		WriteError(w, NewAPIError(ErrCodeBadRequest, "unsupported github event: "+eventName))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	status := http.StatusAccepted
	if result.Ignored {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) handleGitHubAutomationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	jobs := []*automation.Job{}
	if s.automation != nil {
		jobs = s.automation.ListJobs()
	}

	resp := map[string]interface{}{
		"enabled": false,
		"config":  safeGitHubAutomationConfig(s.cfg.GitHub),
		"jobs":    jobs,
	}
	if s.automation != nil {
		resp["enabled"] = s.automation.Enabled()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGitHubAutomationJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	if s.automation == nil {
		WriteError(w, NewAPIError(ErrCodeNotFound, "GitHub automation is not initialized"))
		return
	}

	jobID := strings.TrimPrefix(r.URL.Path, "/api/admin/github-automation/jobs/")
	jobID = strings.TrimSpace(jobID)
	if jobID == "" || strings.Contains(jobID, "/") {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid automation job path"))
		return
	}

	job, ok := s.automation.GetJob(jobID)
	if !ok {
		WriteError(w, NewAPIError(ErrCodeNotFound, "Automation job not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(job)
}

func safeGitHubAutomationConfig(cfg config.GitHubAutomationConfig) map[string]interface{} {
	redactArgs := func(args []string) []string {
		out := make([]string, 0, len(args))
		for _, arg := range args {
			out = append(out, automation.RedactGitHubSecrets(arg, cfg))
		}
		return out
	}

	return map[string]interface{}{
		"enabled":                          cfg.Enabled,
		"webhook_secret_configured":        strings.TrimSpace(cfg.WebhookSecret) != "",
		"personal_access_token_configured": strings.TrimSpace(cfg.PersonalAccessToken) != "",
		"allowed_repositories":             cfg.AllowedRepositories,
		"require_author_org_member":        cfg.RequireAuthorOrgMember,
		"mention_org_members":              cfg.MentionOrgMembers,
		"mention_max_members":              cfg.MentionMaxMembers,
		"review_language":                  cfg.ReviewLanguage,
		"trigger_label":                    cfg.TriggerLabel,
		"base_branch":                      cfg.BaseBranch,
		"remote":                           cfg.Remote,
		"repo_path":                        cfg.RepoPath,
		"worktree_root":                    cfg.WorktreeRoot,
		"branch_prefix":                    cfg.BranchPrefix,
		"development_command":              redactArgs(cfg.DevelopmentCommand),
		"review_command":                   redactArgs(cfg.ReviewCommand),
		"deploy_preview_command":           redactArgs(cfg.DeployPreviewCommand),
		"auto_commit":                      cfg.AutoCommit,
		"auto_push":                        cfg.AutoPush,
		"auto_create_pr":                   cfg.AutoCreatePR,
		"allow_issue_merge":                cfg.AllowIssueMerge,
		"merge_method":                     cfg.MergeMethod,
		"wait_for_ci":                      cfg.WaitForCI,
		"ci_wait_timeout_seconds":          cfg.CIWaitTimeoutSeconds,
		"ci_poll_interval_seconds":         cfg.CIPollIntervalSeconds,
		"auto_deploy_preview":              cfg.AutoDeployPreview,
		"preview_url_base":                 cfg.PreviewURLBase,
		"preview_path_prefix":              cfg.PreviewPathPrefix,
		"pull_request_draft":               cfg.PullRequestDraft,
		"cleanup_worktrees":                cfg.CleanupWorktrees,
		"max_concurrent_jobs":              cfg.MaxConcurrentJobs,
	}
}
