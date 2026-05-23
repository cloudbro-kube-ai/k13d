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
	"strings"
	"time"
)

const githubAPIVersion = "2022-11-28"

type GitHubReporter interface {
	PostIssueComment(ctx context.Context, repo string, issueNumber int, body string) error
	CreatePullRequest(ctx context.Context, repo, title, head, base, body string, draft bool) (*PullRequestInfo, error)
	CreatePullRequestReview(ctx context.Context, repo string, prNumber int, body string) error
	WaitForChecks(ctx context.Context, repo, ref string, timeout, interval time.Duration) (*CIResult, error)
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
	if strings.TrimSpace(c.token) == "" {
		return fmt.Errorf("github automation token is not configured")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	req.Header.Set("User-Agent", "k13d-github-automation")

	resp, err := c.httpClient.Do(req)
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

	resp, err := c.httpClient.Do(req)
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
			Number  int    `json:"number"`
			Title   string `json:"title"`
			Body    string `json:"body"`
			HTMLURL string `json:"html_url"`
			User    struct {
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
		EventName:     eventName,
		Action:        payload.Action,
		Repository:    payload.Repository.FullName,
		DefaultBranch: payload.Repository.DefaultBranch,
		IssueNumber:   payload.Issue.Number,
		IssueTitle:    payload.Issue.Title,
		IssueBody:     payload.Issue.Body,
		IssueURL:      payload.Issue.HTMLURL,
		IssueAuthor:   payload.Issue.User.Login,
	}
	if payload.Label.Name != "" {
		event.TriggeredLabel = payload.Label.Name
	}
	for _, label := range payload.Issue.Labels {
		event.Labels = append(event.Labels, label.Name)
	}
	return event, nil
}
