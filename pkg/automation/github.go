package automation

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const githubAPIVersion = "2022-11-28"

type GitHubReporter interface {
	PostIssueComment(ctx context.Context, repo string, issueNumber int, body string) error
	PostPullRequestComment(ctx context.Context, repo string, prNumber int, body string) error
	AssignIssue(ctx context.Context, repo string, issueNumber int, assignees []string) error
	FindOpenPullRequestByHead(ctx context.Context, repo, head string) (*PullRequestInfo, error)
	CreatePullRequest(ctx context.Context, repo, title, head, base, body string, draft bool) (*PullRequestInfo, error)
	RequestPullRequestReviewers(ctx context.Context, repo string, prNumber int, reviewers []string) error
	CreatePullRequestReview(ctx context.Context, repo string, prNumber int, body string) error
	MergePullRequest(ctx context.Context, repo string, prNumber int, method, title, message string) (*PullRequestMergeInfo, error)
	CloseIssue(ctx context.Context, repo string, issueNumber int, reason string) error
	WaitForChecks(ctx context.Context, repo, ref string, timeout, interval time.Duration) (*CIResult, error)
	IsOrganizationMember(ctx context.Context, org, username string) (bool, error)
	ListOrganizationMembers(ctx context.Context, org string, limit int) ([]string, error)
}

type GitHubClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		baseURL:    "https://api.github.com",
		httpClient: &http.Client{Timeout: 20 * time.Second},
		token:      strings.TrimSpace(token),
	}
}

func (c *GitHubClient) PostIssueComment(ctx context.Context, repo string, issueNumber int, body string) error {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	payload := map[string]string{"body": body}
	return c.post(ctx, fmt.Sprintf("/repos/%s/issues/%d/comments", repo, issueNumber), payload, nil)
}

func (c *GitHubClient) PostPullRequestComment(ctx context.Context, repo string, prNumber int, body string) error {
	return c.PostIssueComment(ctx, repo, prNumber, body)
}

func (c *GitHubClient) AssignIssue(ctx context.Context, repo string, issueNumber int, assignees []string) error {
	assignees = sanitizeGitHubLogins(assignees)
	if issueNumber <= 0 || len(assignees) == 0 {
		return nil
	}
	payload := map[string][]string{"assignees": assignees}
	return c.post(ctx, fmt.Sprintf("/repos/%s/issues/%d/assignees", repo, issueNumber), payload, nil)
}

func (c *GitHubClient) FindOpenPullRequestByHead(ctx context.Context, repo, head string) (*PullRequestInfo, error) {
	repo = strings.TrimSpace(repo)
	head = strings.TrimSpace(head)
	owner := repositoryOwner(repo)
	if repo == "" || owner == "" || head == "" {
		return nil, nil
	}

	query := url.Values{}
	query.Set("state", "open")
	query.Set("head", owner+":"+head)
	query.Set("per_page", "1")

	resp := []struct {
		Number int    `json:"number"`
		URL    string `json:"html_url"`
	}{}
	if err := c.get(ctx, fmt.Sprintf("/repos/%s/pulls?%s", repo, query.Encode()), &resp); err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, nil
	}
	return &PullRequestInfo{Number: resp[0].Number, URL: resp[0].URL}, nil
}

func (c *GitHubClient) CreatePullRequest(ctx context.Context, repo, title, head, base, body string, draft bool) (*PullRequestInfo, error) {
	resp := struct {
		Number int    `json:"number"`
		URL    string `json:"html_url"`
	}{}
	payload := map[string]interface{}{
		"title": title,
		"head":  head,
		"base":  base,
		"body":  body,
		"draft": draft,
	}
	if err := c.post(ctx, fmt.Sprintf("/repos/%s/pulls", repo), payload, &resp); err != nil {
		return nil, err
	}
	return &PullRequestInfo{Number: resp.Number, URL: resp.URL}, nil
}

func (c *GitHubClient) RequestPullRequestReviewers(ctx context.Context, repo string, prNumber int, reviewers []string) error {
	reviewers = sanitizeGitHubLogins(reviewers)
	if prNumber <= 0 || len(reviewers) == 0 {
		return nil
	}
	payload := map[string][]string{"reviewers": reviewers}
	return c.post(ctx, fmt.Sprintf("/repos/%s/pulls/%d/requested_reviewers", repo, prNumber), payload, nil)
}

func (c *GitHubClient) CreatePullRequestReview(ctx context.Context, repo string, prNumber int, body string) error {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	payload := map[string]string{
		"body":  body,
		"event": "COMMENT",
	}
	return c.post(ctx, fmt.Sprintf("/repos/%s/pulls/%d/reviews", repo, prNumber), payload, nil)
}

func (c *GitHubClient) MergePullRequest(ctx context.Context, repo string, prNumber int, method, title, message string) (*PullRequestMergeInfo, error) {
	method = normalizeMergeMethod(method)
	payload := map[string]string{
		"merge_method": method,
	}
	if strings.TrimSpace(title) != "" {
		payload["commit_title"] = title
	}
	if strings.TrimSpace(message) != "" {
		payload["commit_message"] = message
	}
	resp := struct {
		SHA     string `json:"sha"`
		Merged  bool   `json:"merged"`
		Message string `json:"message"`
	}{}
	if err := c.put(ctx, fmt.Sprintf("/repos/%s/pulls/%d/merge", repo, prNumber), payload, &resp); err != nil {
		return nil, err
	}
	if !resp.Merged {
		return &PullRequestMergeInfo{SHA: resp.SHA, Merged: resp.Merged, Message: resp.Message}, fmt.Errorf("github did not merge pull request: %s", strings.TrimSpace(resp.Message))
	}
	return &PullRequestMergeInfo{SHA: resp.SHA, Merged: resp.Merged, Message: resp.Message}, nil
}

func (c *GitHubClient) CloseIssue(ctx context.Context, repo string, issueNumber int, reason string) error {
	if issueNumber <= 0 {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "completed"
	}
	payload := map[string]string{
		"state":        "closed",
		"state_reason": reason,
	}
	return c.patch(ctx, fmt.Sprintf("/repos/%s/issues/%d", repo, issueNumber), payload, nil)
}

func (c *GitHubClient) IsOrganizationMember(ctx context.Context, org, username string) (bool, error) {
	org = strings.TrimSpace(org)
	username = strings.TrimSpace(username)
	if org == "" || username == "" {
		return false, nil
	}
	if strings.TrimSpace(c.token) == "" {
		return false, fmt.Errorf("github automation token is not configured")
	}

	path := fmt.Sprintf("/orgs/%s/members/%s", url.PathEscape(org), url.PathEscape(username))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	req.Header.Set("User-Agent", "k13d-github-automation")

	resp, err := c.httpClient.Do(req) // #nosec G704 -- GitHub API base URL is fixed and request paths are generated internally.
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNoContent:
		return true, nil
	case resp.StatusCode == http.StatusNotFound:
		return false, nil
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return true, nil
	default:
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return false, fmt.Errorf("github api %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
}

func (c *GitHubClient) ListOrganizationMembers(ctx context.Context, org string, limit int) ([]string, error) {
	org = strings.TrimSpace(org)
	if org == "" || limit == 0 {
		return nil, nil
	}
	if limit < 0 {
		limit = 100
	}
	if strings.TrimSpace(c.token) == "" {
		return nil, fmt.Errorf("github automation token is not configured")
	}

	type member struct {
		Login string `json:"login"`
	}
	members := make([]string, 0, min(limit, 100))
	for page := 1; len(members) < limit; page++ {
		perPage := min(100, limit-len(members))
		path := fmt.Sprintf("/orgs/%s/members?per_page=%d&page=%d", url.PathEscape(org), perPage, page)
		var pageMembers []member
		if err := c.get(ctx, path, &pageMembers); err != nil {
			return nil, err
		}
		if len(pageMembers) == 0 {
			break
		}
		before := len(members)
		for _, item := range pageMembers {
			if login := strings.TrimSpace(item.Login); login != "" {
				members = append(members, login)
				if len(members) >= limit {
					break
				}
			}
		}
		if len(members) == before {
			break
		}
	}
	return members, nil
}

func sanitizeGitHubLogins(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		login := strings.TrimSpace(strings.TrimPrefix(value, "@"))
		if login == "" {
			continue
		}
		key := strings.ToLower(login)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, login)
	}
	return out
}

func normalizeMergeMethod(method string) string {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "merge", "rebase", "squash":
		return strings.ToLower(strings.TrimSpace(method))
	default:
		return "squash"
	}
}

func (c *GitHubClient) WaitForChecks(ctx context.Context, repo, ref string, timeout, interval time.Duration) (*CIResult, error) {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		result, done, err := c.checkRuns(ctx, repo, ref)
		if err != nil {
			return nil, err
		}
		if done {
			if result.Conclusion == "success" {
				return result, nil
			}
			return result, fmt.Errorf("github checks completed with conclusion %q", result.Conclusion)
		}

		select {
		case <-ctx.Done():
			if result != nil && result.Summary != "" {
				return result, fmt.Errorf("timed out waiting for github checks: %s", result.Summary)
			}
			return result, fmt.Errorf("timed out waiting for github checks")
		case <-ticker.C:
		}
	}
}

func (c *GitHubClient) checkRuns(ctx context.Context, repo, ref string) (*CIResult, bool, error) {
	resp := struct {
		TotalCount int `json:"total_count"`
		CheckRuns  []struct {
			Name       string `json:"name"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			HTMLURL    string `json:"html_url"`
		} `json:"check_runs"`
	}{}
	path := fmt.Sprintf("/repos/%s/commits/%s/check-runs?per_page=100", repo, ref)
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, false, err
	}
	if resp.TotalCount == 0 {
		return &CIResult{Status: "pending", Summary: "no check runs found yet"}, false, nil
	}

	pending := 0
	failed := 0
	detailsURL := ""
	for _, run := range resp.CheckRuns {
		if detailsURL == "" {
			detailsURL = run.HTMLURL
		}
		if run.Status != "completed" {
			pending++
			continue
		}
		if !isSuccessfulCheckConclusion(run.Conclusion) {
			failed++
			if run.HTMLURL != "" {
				detailsURL = run.HTMLURL
			}
		}
	}
	if pending > 0 {
		return &CIResult{
			Status:  "pending",
			URL:     detailsURL,
			Summary: fmt.Sprintf("%d/%d check runs still pending", pending, resp.TotalCount),
		}, false, nil
	}
	if failed > 0 {
		return &CIResult{
			Status:     "completed",
			Conclusion: "failure",
			URL:        detailsURL,
			Summary:    fmt.Sprintf("%d/%d check runs failed", failed, resp.TotalCount),
		}, true, nil
	}
	return &CIResult{
		Status:     "completed",
		Conclusion: "success",
		URL:        detailsURL,
		Summary:    fmt.Sprintf("%d check runs passed", resp.TotalCount),
	}, true, nil
}

func isSuccessfulCheckConclusion(conclusion string) bool {
	switch conclusion {
	case "success", "neutral", "skipped":
		return true
	default:
		return false
	}
}

func (c *GitHubClient) post(ctx context.Context, path string, payload interface{}, out interface{}) error {
	return c.doJSON(ctx, http.MethodPost, path, payload, out)
}

func (c *GitHubClient) put(ctx context.Context, path string, payload interface{}, out interface{}) error {
	return c.doJSON(ctx, http.MethodPut, path, payload, out)
}

func (c *GitHubClient) patch(ctx context.Context, path string, payload interface{}, out interface{}) error {
	return c.doJSON(ctx, http.MethodPatch, path, payload, out)
}

func (c *GitHubClient) doJSON(ctx context.Context, method, path string, payload interface{}, out interface{}) error {
	if strings.TrimSpace(c.token) == "" {
		return fmt.Errorf("github automation token is not configured")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	req.Header.Set("User-Agent", "k13d-github-automation")

	resp, err := c.httpClient.Do(req) // #nosec G704 -- GitHub API base URL is fixed and request paths are generated internally.
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github api %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return err
		}
	}
	return nil
}

func (c *GitHubClient) get(ctx context.Context, path string, out interface{}) error {
	if strings.TrimSpace(c.token) == "" {
		return fmt.Errorf("github automation token is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	req.Header.Set("User-Agent", "k13d-github-automation")

	resp, err := c.httpClient.Do(req) // #nosec G704 -- GitHub API base URL is fixed and request paths are generated internally.
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github api %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return err
		}
	}
	return nil
}

func VerifyGitHubSignature(secret string, payload []byte, signatureHeader string) bool {
	if strings.TrimSpace(secret) == "" {
		return true
	}
	const prefix = "sha256="
	if !strings.HasPrefix(signatureHeader, prefix) {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	expected := prefix + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signatureHeader))
}

func ParseIssueEvent(eventName string, body []byte) (*IssueEvent, error) {
	if eventName != "issues" {
		return nil, fmt.Errorf("unsupported github event: %s", eventName)
	}

	var payload struct {
		Action     string `json:"action"`
		Repository struct {
			FullName      string `json:"full_name"`
			DefaultBranch string `json:"default_branch"`
		} `json:"repository"`
		Issue struct {
			Number            int    `json:"number"`
			Title             string `json:"title"`
			Body              string `json:"body"`
			HTMLURL           string `json:"html_url"`
			AuthorAssociation string `json:"author_association"`
			User              struct {
				Login string `json:"login"`
			} `json:"user"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
		} `json:"issue"`
		Label struct {
			Name string `json:"name"`
		} `json:"label"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	event := &IssueEvent{
		EventName:              eventName,
		Action:                 payload.Action,
		Repository:             payload.Repository.FullName,
		DefaultBranch:          payload.Repository.DefaultBranch,
		IssueNumber:            payload.Issue.Number,
		IssueTitle:             payload.Issue.Title,
		IssueBody:              payload.Issue.Body,
		IssueURL:               payload.Issue.HTMLURL,
		IssueAuthor:            payload.Issue.User.Login,
		IssueAuthorAssociation: payload.Issue.AuthorAssociation,
	}
	if payload.Label.Name != "" {
		event.TriggeredLabel = payload.Label.Name
	}
	for _, label := range payload.Issue.Labels {
		event.Labels = append(event.Labels, label.Name)
	}
	return event, nil
}

func ParseIssueCommentEvent(eventName string, body []byte) (*IssueCommentEvent, error) {
	if eventName != "issue_comment" {
		return nil, fmt.Errorf("unsupported github event: %s", eventName)
	}

	var payload struct {
		Action     string `json:"action"`
		Repository struct {
			FullName      string `json:"full_name"`
			DefaultBranch string `json:"default_branch"`
		} `json:"repository"`
		Issue struct {
			Number            int    `json:"number"`
			Title             string `json:"title"`
			Body              string `json:"body"`
			HTMLURL           string `json:"html_url"`
			AuthorAssociation string `json:"author_association"`
			User              struct {
				Login string `json:"login"`
			} `json:"user"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
			PullRequest *struct{} `json:"pull_request"`
		} `json:"issue"`
		Comment struct {
			Body              string `json:"body"`
			AuthorAssociation string `json:"author_association"`
			User              struct {
				Login string `json:"login"`
			} `json:"user"`
		} `json:"comment"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Issue.PullRequest != nil {
		return nil, fmt.Errorf("issue_comment events on pull requests are not supported")
	}

	event := &IssueCommentEvent{
		EventName:                eventName,
		Action:                   payload.Action,
		Repository:               payload.Repository.FullName,
		DefaultBranch:            payload.Repository.DefaultBranch,
		IssueNumber:              payload.Issue.Number,
		IssueTitle:               payload.Issue.Title,
		IssueBody:                payload.Issue.Body,
		IssueURL:                 payload.Issue.HTMLURL,
		IssueAuthor:              payload.Issue.User.Login,
		IssueAuthorAssociation:   payload.Issue.AuthorAssociation,
		CommentBody:              payload.Comment.Body,
		CommentAuthor:            payload.Comment.User.Login,
		CommentAuthorAssociation: payload.Comment.AuthorAssociation,
	}
	for _, label := range payload.Issue.Labels {
		event.Labels = append(event.Labels, label.Name)
	}
	return event, nil
}
