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
