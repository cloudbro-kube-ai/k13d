package automation

import "time"

type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
	JobStatusIgnored   JobStatus = "ignored"
)

// Job captures the end-to-end lifecycle of one GitHub issue automation run.
type Job struct {
	ID                string    `json:"id"`
	Repository        string    `json:"repository"`
	IssueNumber       int       `json:"issue_number"`
	IssueTitle        string    `json:"issue_title"`
	IssueBody         string    `json:"issue_body,omitempty"`
	IssueURL          string    `json:"issue_url"`
	IssueAuthor       string    `json:"issue_author,omitempty"`
	TriggerAction     string    `json:"trigger_action"`
	TriggerLabel      string    `json:"trigger_label,omitempty"`
	Status            JobStatus `json:"status"`
	StatusReason      string    `json:"status_reason,omitempty"`
	Branch            string    `json:"branch,omitempty"`
	WorktreePath      string    `json:"worktree_path,omitempty"`
	CommitSHA         string    `json:"commit_sha,omitempty"`
	PullRequestURL    string    `json:"pull_request_url,omitempty"`
	PullRequestNumber int       `json:"pull_request_number,omitempty"`
	HasChanges        bool      `json:"has_changes"`
	DevelopmentLog    string    `json:"development_log,omitempty"`
	ReviewLog         string    `json:"review_log,omitempty"`
	DiffStat          string    `json:"diff_stat,omitempty"`
	Error             string    `json:"error,omitempty"`
	Warnings          []string  `json:"warnings,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	StartedAt         time.Time `json:"started_at,omitempty"`
	FinishedAt        time.Time `json:"finished_at,omitempty"`
}

type IssueEvent struct {
	EventName      string
	Action         string
	Repository     string
	DefaultBranch  string
	IssueNumber    int
	IssueTitle     string
	IssueBody      string
	IssueURL       string
	IssueAuthor    string
	Labels         []string
	TriggeredLabel string
}

type QueueResult struct {
	Accepted bool   `json:"accepted"`
	Ignored  bool   `json:"ignored"`
	Reason   string `json:"reason,omitempty"`
	JobID    string `json:"job_id,omitempty"`
}

type ExecutionResult struct {
	Branch         string
	WorktreePath   string
	CommitSHA      string
	HasChanges     bool
	DevelopmentLog string
	ReviewLog      string
	DiffStat       string
}

type PullRequestInfo struct {
	Number int
	URL    string
}
