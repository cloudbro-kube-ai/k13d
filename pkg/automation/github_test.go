package automation

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubClientIsOrganizationMember(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
		wantErr    bool
	}{
		{name: "member", statusCode: http.StatusNoContent, want: true},
		{name: "not member", statusCode: http.StatusNotFound, want: false},
		{name: "api error", statusCode: http.StatusForbidden, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/orgs/cloudbro-kube-ai/members/alice" {
					t.Fatalf("unexpected path %q", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewGitHubClient("token")
			client.baseURL = server.URL
			got, err := client.IsOrganizationMember(context.Background(), "cloudbro-kube-ai", "alice")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("IsOrganizationMember() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("IsOrganizationMember() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitHubClientListOrganizationMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/cloudbro-kube-ai/members" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("per_page") != "2" {
			t.Fatalf("per_page = %q, want 2", r.URL.Query().Get("per_page"))
		}
		if err := json.NewEncoder(w).Encode([]map[string]string{
			{"login": "alice"},
			{"login": "bob"},
		}); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	client := NewGitHubClient("token")
	client.baseURL = server.URL
	got, err := client.ListOrganizationMembers(context.Background(), "cloudbro-kube-ai", 2)
	if err != nil {
		t.Fatalf("ListOrganizationMembers() error = %v", err)
	}
	want := []string{"alice", "bob"}
	if len(got) != len(want) {
		t.Fatalf("members = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("members = %#v, want %#v", got, want)
		}
	}
}

func TestGitHubClientPostPullRequestComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/cloudbro-kube-ai/k13d/issues/12/comments" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		var payload struct {
			Body string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Body != "preview ready" {
			t.Fatalf("body = %q, want preview ready", payload.Body)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewGitHubClient("token")
	client.baseURL = server.URL
	if err := client.PostPullRequestComment(context.Background(), "cloudbro-kube-ai/k13d", 12, "preview ready"); err != nil {
		t.Fatalf("PostPullRequestComment() error = %v", err)
	}
}

func TestGitHubClientAssignIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/cloudbro-kube-ai/k13d/issues/7/assignees" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var payload struct {
			Assignees []string `json:"assignees"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Assignees) != 1 || payload.Assignees[0] != "alice" {
			t.Fatalf("assignees = %#v, want alice", payload.Assignees)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewGitHubClient("token")
	client.baseURL = server.URL
	if err := client.AssignIssue(context.Background(), "cloudbro-kube-ai/k13d", 7, []string{"@alice", "alice"}); err != nil {
		t.Fatalf("AssignIssue() error = %v", err)
	}
}

func TestGitHubClientFindOpenPullRequestByHead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/cloudbro-kube-ai/k13d/pulls" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.URL.Query().Get("state") != "open" {
			t.Fatalf("state = %q, want open", r.URL.Query().Get("state"))
		}
		if r.URL.Query().Get("head") != "cloudbro-kube-ai:codex/issue-7" {
			t.Fatalf("head = %q", r.URL.Query().Get("head"))
		}
		if err := json.NewEncoder(w).Encode([]map[string]interface{}{
			{"number": 12, "html_url": "https://github.com/cloudbro-kube-ai/k13d/pull/12"},
		}); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	client := NewGitHubClient("token")
	client.baseURL = server.URL
	pr, err := client.FindOpenPullRequestByHead(context.Background(), "cloudbro-kube-ai/k13d", "codex/issue-7")
	if err != nil {
		t.Fatalf("FindOpenPullRequestByHead() error = %v", err)
	}
	if pr == nil || pr.Number != 12 {
		t.Fatalf("PR = %#v, want #12", pr)
	}
}

func TestGitHubClientRequestPullRequestReviewers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/cloudbro-kube-ai/k13d/pulls/12/requested_reviewers" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var payload struct {
			Reviewers []string `json:"reviewers"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Reviewers) != 2 || payload.Reviewers[0] != "alice" || payload.Reviewers[1] != "bob" {
			t.Fatalf("reviewers = %#v, want alice,bob", payload.Reviewers)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewGitHubClient("token")
	client.baseURL = server.URL
	if err := client.RequestPullRequestReviewers(context.Background(), "cloudbro-kube-ai/k13d", 12, []string{"alice", "@bob", "alice"}); err != nil {
		t.Fatalf("RequestPullRequestReviewers() error = %v", err)
	}
}

func TestGitHubClientMergePullRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/repos/cloudbro-kube-ai/k13d/pulls/12/merge" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		var payload struct {
			MergeMethod string `json:"merge_method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.MergeMethod != "squash" {
			t.Fatalf("merge_method = %q, want squash", payload.MergeMethod)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sha":     "abc123",
			"merged":  true,
			"message": "merged",
		})
	}))
	defer server.Close()

	client := NewGitHubClient("token")
	client.baseURL = server.URL
	got, err := client.MergePullRequest(context.Background(), "cloudbro-kube-ai/k13d", 12, "invalid", "title", "message")
	if err != nil {
		t.Fatalf("MergePullRequest() error = %v", err)
	}
	if got == nil || got.SHA != "abc123" || !got.Merged {
		t.Fatalf("MergePullRequest() = %#v", got)
	}
}

func TestGitHubClientCloseIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/repos/cloudbro-kube-ai/k13d/issues/7" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		var payload struct {
			State       string `json:"state"`
			StateReason string `json:"state_reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.State != "closed" {
			t.Fatalf("state = %q, want closed", payload.State)
		}
		if payload.StateReason != "completed" {
			t.Fatalf("state_reason = %q, want completed", payload.StateReason)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewGitHubClient("token")
	client.baseURL = server.URL
	if err := client.CloseIssue(context.Background(), "cloudbro-kube-ai/k13d", 7, ""); err != nil {
		t.Fatalf("CloseIssue() error = %v", err)
	}
}

func TestParseIssueCommentEvent(t *testing.T) {
	body := []byte(`{
		"action":"created",
		"repository":{"full_name":"cloudbro-kube-ai/k13d","default_branch":"main"},
		"issue":{
			"number":17,
			"title":"Automate me",
			"body":"Please fix this",
			"html_url":"https://github.com/cloudbro-kube-ai/k13d/issues/17",
			"author_association":"MEMBER",
			"user":{"login":"alice"},
			"labels":[{"name":"codex:auto"}]
		},
		"comment":{
			"body":"k13d merge 해줘",
			"author_association":"MEMBER",
			"user":{"login":"bob"}
		}
	}`)
	event, err := ParseIssueCommentEvent("issue_comment", body)
	if err != nil {
		t.Fatalf("ParseIssueCommentEvent() error = %v", err)
	}
	if event.CommentAuthor != "bob" || event.CommentBody != "k13d merge 해줘" {
		t.Fatalf("event = %#v", event)
	}
}
