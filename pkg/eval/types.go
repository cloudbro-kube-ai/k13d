// Package eval provides AI agent benchmark evaluation framework
// Inspired by k8s-ai-bench: https://github.com/gke-labs/k8s-ai-bench
package eval

import (
	"time"
)

// TaskDifficulty represents the difficulty level of a task
type TaskDifficulty string

const (
	DifficultyEasy   TaskDifficulty = "easy"
	DifficultyMedium TaskDifficulty = "medium"
	DifficultyHard   TaskDifficulty = "hard"
	DifficultyExpert TaskDifficulty = "expert"
)

// TaskCategory represents the category of a benchmark task
type TaskCategory string

const (
	CategoryCreation        TaskCategory = "creation"
	CategoryTroubleshooting TaskCategory = "troubleshooting"
	CategoryScaling         TaskCategory = "scaling"
	CategoryNetworking      TaskCategory = "networking"
	CategoryStorage         TaskCategory = "storage"
	CategoryRBAC            TaskCategory = "rbac"
	CategoryConfiguration   TaskCategory = "configuration"
	CategoryWorkloads       TaskCategory = "workloads"
	CategoryAnalysis        TaskCategory = "analysis"
	CategoryGatekeeper      TaskCategory = "gatekeeper"
	CategoryMultiStep       TaskCategory = "multi-step"
)

// IsolationMode defines how tasks are isolated
type IsolationMode string

const (
	IsolationNone      IsolationMode = ""
	IsolationNamespace IsolationMode = "namespace"
	IsolationCluster   IsolationMode = "cluster"
)

// BenchmarkTask represents a complete benchmark task definition
// Compatible with k8s-ai-bench format
type BenchmarkTask struct {
	// Task identification
	ID          string   `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Tags        []string `yaml:"tags" json:"tags"`

	// Task categorization
	Category   TaskCategory   `yaml:"category" json:"category"`
	Difficulty TaskDifficulty `yaml:"difficulty" json:"difficulty"`

	// Execution settings
	Timeout   string        `yaml:"timeout" json:"timeout"`
	Isolation IsolationMode `yaml:"isolation" json:"isolation"`
	Disabled  bool          `yaml:"disabled" json:"disabled"`

	// Scripts for setup/verify/cleanup
	Setup    string `yaml:"setup" json:"setup"`
	Verifier string `yaml:"verifier" json:"verifier"`
	Cleanup  string `yaml:"cleanup" json:"cleanup"`

	// Prompt configuration
	Script []ScriptStep `yaml:"script" json:"script"`

	// Expectations for output validation
	Expect []Expectation `yaml:"expect" json:"expect"`

	// AI Model specific hints (optional)
	ModelHints map[string]ModelHint `yaml:"model_hints,omitempty" json:"model_hints,omitempty"`
}

// ScriptStep represents a single step in the task script
type ScriptStep struct {
	Prompt     string `yaml:"prompt,omitempty" json:"prompt,omitempty"`
	PromptFile string `yaml:"promptFile,omitempty" json:"promptFile,omitempty"`
}

// Expectation defines what output is expected from the task
type Expectation struct {
	Contains    string `yaml:"contains,omitempty" json:"contains,omitempty"`
	NotContains string `yaml:"notContains,omitempty" json:"notContains,omitempty"`
}

// ModelHint provides AI model-specific information
type ModelHint struct {
	Strength string   `yaml:"strength,omitempty" json:"strength,omitempty"`
	Weakness string   `yaml:"weakness,omitempty" json:"weakness,omitempty"`
	Tips     []string `yaml:"tips,omitempty" json:"tips,omitempty"`
	PassRate float64  `yaml:"pass_rate,omitempty" json:"pass_rate,omitempty"`
	AvgTime  string   `yaml:"avg_time,omitempty" json:"avg_time,omitempty"`
}

// LLMConfig represents LLM configuration for evaluation
type LLMConfig struct {
	ID         string `json:"id" yaml:"id"`
	ProviderID string `json:"provider_id" yaml:"provider_id"`
	ModelID    string `json:"model_id" yaml:"model_id"`
	Endpoint   string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	McpClient  bool   `json:"mcp_client,omitempty" yaml:"mcp_client,omitempty"`
	Quiet      bool   `json:"quiet,omitempty" yaml:"quiet,omitempty"`
}

// TaskResult represents the outcome of a single task evaluation
type TaskResult struct {
	Task      string    `json:"task" yaml:"task"`
	LLMConfig LLMConfig `json:"llm_config" yaml:"llm_config"`
	Result    string    `json:"result" yaml:"result"` // "success", "fail", "error"
	Error     string    `json:"error,omitempty" yaml:"error,omitempty"`
	Failures  []Failure `json:"failures,omitempty" yaml:"failures,omitempty"`
	StartTime time.Time `json:"start_time" yaml:"start_time"`
	EndTime   time.Time `json:"end_time" yaml:"end_time"`
	Duration  string    `json:"duration" yaml:"duration"`

	// Metrics
	ToolCallCount int `json:"tool_call_count" yaml:"tool_call_count"`
	TokensUsed    int `json:"tokens_used,omitempty" yaml:"tokens_used,omitempty"`
	RetryCount    int `json:"retry_count" yaml:"retry_count"`
}

// Failure represents a specific failure in task execution
type Failure struct {
	Message string `json:"message" yaml:"message"`
	Phase   string `json:"phase,omitempty" yaml:"phase,omitempty"` // "setup", "execution", "verification"
}

// AddFailure adds a formatted failure message
func (r *TaskResult) AddFailure(format string, args ...interface{}) {
	r.Failures = append(r.Failures, Failure{
		Message: formatMessage(format, args...),
	})
}

// BenchmarkSuite represents a collection of benchmark tasks
type BenchmarkSuite struct {
	Name        string          `yaml:"name" json:"name"`
	Description string          `yaml:"description" json:"description"`
	Version     string          `yaml:"version" json:"version"`
	Tasks       []BenchmarkTask `yaml:"tasks" json:"tasks"`

	// Suite-level configuration
	DefaultTimeout   string        `yaml:"default_timeout" json:"default_timeout"`
	DefaultIsolation IsolationMode `yaml:"default_isolation" json:"default_isolation"`
}

// BenchmarkReport represents the complete evaluation report
type BenchmarkReport struct {
	// Metadata
	SuiteName     string    `json:"suite_name"`
	RunID         string    `json:"run_id"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	TotalDuration string    `json:"total_duration"`

	// Configuration
	LLMConfigs []LLMConfig `json:"llm_configs"`
	TaskCount  int         `json:"task_count"`

	// Results
	Results []TaskResult `json:"results"`

	// Summary statistics
	Summary ReportSummary `json:"summary"`

	// Model comparison
	ModelComparison []ModelSummary `json:"model_comparison,omitempty"`
}

// ReportSummary contains aggregate statistics
type ReportSummary struct {
	TotalTasks  int     `json:"total_tasks"`
	PassedTasks int     `json:"passed_tasks"`
	FailedTasks int     `json:"failed_tasks"`
	ErrorTasks  int     `json:"error_tasks"`
	PassRate    float64 `json:"pass_rate"`

	// By category
	CategoryStats map[TaskCategory]CategoryStat `json:"category_stats"`

	// By difficulty
	DifficultyStats map[TaskDifficulty]DifficultyStat `json:"difficulty_stats"`
}

// CategoryStat contains statistics for a category
type CategoryStat struct {
	Total  int     `json:"total"`
	Passed int     `json:"passed"`
	Rate   float64 `json:"rate"`
}

// DifficultyStat contains statistics for a difficulty level
type DifficultyStat struct {
	Total  int     `json:"total"`
	Passed int     `json:"passed"`
	Rate   float64 `json:"rate"`
}

// ModelSummary contains per-model statistics
type ModelSummary struct {
	ModelID      string  `json:"model_id"`
	ProviderID   string  `json:"provider_id"`
	TotalTasks   int     `json:"total_tasks"`
	PassedTasks  int     `json:"passed_tasks"`
	FailedTasks  int     `json:"failed_tasks"`
	ErrorTasks   int     `json:"error_tasks"`
	PassRate     float64 `json:"pass_rate"`
	AvgDuration  string  `json:"avg_duration"`
	AvgToolCalls float64 `json:"avg_tool_calls"`

	// Strengths and weaknesses
	StrongCategories []TaskCategory `json:"strong_categories,omitempty"`
	WeakCategories   []TaskCategory `json:"weak_categories,omitempty"`
}

// PassRateType represents different pass rate calculations
type PassRateType string

const (
	// PassAt1 - Can the agent solve the task on the first try?
	PassAt1 PassRateType = "pass@1"
	// PassAt5 - Can the agent solve the task at least once in 5 attempts?
	PassAt5 PassRateType = "pass@5"
	// PassHat5 - Does the agent solve the task every single time (5/5)?
	PassHat5 PassRateType = "pass^5"
)

// formatMessage formats a message with optional arguments
func formatMessage(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	return format // Simple case, can be expanded with fmt.Sprintf
}
