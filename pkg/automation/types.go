package automation

import "time"

type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusWaitingCI JobStatus = "waiting_for_ci"
	JobStatusDeploying JobStatus = "deploying"
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
	RequestAuthor     string    `json:"request_author,omitempty"`
	TriggerAction     string    `json:"trigger_action"`
	TriggerLabel      string    `json:"trigger_label,omitempty"`
	Status            JobStatus `json:"status"`
	StatusReason      string    `json:"status_reason,omitempty"`
	Branch            string    `json:"branch,omitempty"`
	WorktreePath      string    `json:"worktree_path,omitempty"`
	CommitSHA         string    `json:"commit_sha,omitempty"`
	PullRequestURL    string    `json:"pull_request_url,omitempty"`
	PullRequestNumber int       `json:"pull_request_number,omitempty"`
	CIStatus          string    `json:"ci_status,omitempty"`
	CIConclusion      string    `json:"ci_conclusion,omitempty"`
	CIURL             string    `json:"ci_url,omitempty"`
	PreviewSlug       string    `json:"preview_slug,omitempty"`
	PreviewURL        string    `json:"preview_url,omitempty"`
	PreviewTarget     string    `json:"preview_target,omitempty"`
	HasChanges        bool      `json:"has_changes"`
	DevelopmentLog    string    `json:"development_log,omitempty"`
	ReviewLog         string    `json:"review_log,omitempty"`
	DeploymentLog     string    `json:"deployment_log,omitempty"`
	DiffStat          string    `json:"diff_stat,omitempty"`
	Error             string    `json:"error,omitempty"`
	Warnings          []string  `json:"warnings,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	StartedAt         time.Time `json:"started_at,omitempty"`
	FinishedAt        time.Time `json:"finished_at,omitempty"`
}

type IssueEvent struct {
	EventName              string
	Action                 string
	Repository             string
	DefaultBranch          string
	IssueNumber            int
	IssueTitle             string
	IssueBody              string
	IssueURL               string
	IssueAuthor            string
	IssueAuthorAssociation string
	Labels                 []string
	TriggeredLabel         string
}

type IssueCommentEvent struct {
	EventName                string
	Action                   string
	Repository               string
	DefaultBranch            string
	IssueNumber              int
	IssueTitle               string
	IssueBody                string
	IssueURL                 string
	IssueAuthor              string
	IssueAuthorAssociation   string
	CommentID                int64
	CommentBody              string
	CommentAuthor            string
	CommentAuthorAssociation string
	Labels                   []string
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

type CIResult struct {
	Status     string
	Conclusion string
	URL        string
	Summary    string
}

type PreviewDeployment struct {
	Slug      string
	PublicURL string
	TargetURL string
	Log       string
}

type PullRequestInfo struct {
	Number int
	URL    string
}

type PullRequestMergeInfo struct {
	SHA     string
	Merged  bool
	Message string
}
