// Package bench provides AI benchmarking capabilities for Kubernetes tasks.
// Inspired by https://github.com/gke-labs/k8s-ai-bench
package bench

import (
	"time"
)

// TaskDifficulty represents the difficulty level of a benchmark task
type TaskDifficulty string

const (
	DifficultyEasy   TaskDifficulty = "easy"
	DifficultyMedium TaskDifficulty = "medium"
	DifficultyHard   TaskDifficulty = "hard"
)

// TaskIsolation specifies how the task should be isolated
type TaskIsolation string

const (
	IsolationNamespace TaskIsolation = "namespace" // Isolate within a namespace
	IsolationCluster   TaskIsolation = "cluster"   // Create isolated cluster (Kind/vCluster)
	IsolationNone      TaskIsolation = ""          // No isolation
)

// TaskResult represents the outcome of a single task evaluation
type TaskResult string

const (
	ResultSuccess TaskResult = "success"
	ResultFail    TaskResult = "fail"
	ResultError   TaskResult = "error"
	ResultTimeout TaskResult = "timeout"
	ResultSkipped TaskResult = "skipped"
)

// Prompt represents a single prompt in a task script
type Prompt struct {
	Text    string `yaml:"prompt,omitempty"`     // Inline prompt text
	File    string `yaml:"promptFile,omitempty"` // Path to prompt file
	WaitFor string `yaml:"waitFor,omitempty"`    // Wait condition before next prompt
	Timeout string `yaml:"timeout,omitempty"`    // Timeout for this prompt
}

// Expectation defines what output is expected from the AI agent
type Expectation struct {
	Contains    string `yaml:"contains,omitempty"`    // Regex pattern that output should contain
	NotContains string `yaml:"notContains,omitempty"` // Regex pattern that output should NOT contain
	ExitCode    *int   `yaml:"exitCode,omitempty"`    // Expected exit code (for verifier)
}

// Task represents a benchmark task definition loaded from task.yaml
type Task struct {
	// Metadata
	ID          string         `yaml:"id,omitempty"`          // Task identifier (defaults to directory name)
	Name        string         `yaml:"name,omitempty"`        // Human-readable name
	Description string         `yaml:"description,omitempty"` // Task description
	Category    string         `yaml:"category,omitempty"`    // Task category (troubleshooting, creation, etc.)
	Difficulty  TaskDifficulty `yaml:"difficulty,omitempty"`  // easy, medium, hard
	Disabled    bool           `yaml:"disabled,omitempty"`    // Skip this task if true
	Tags        []string       `yaml:"tags,omitempty"`        // Tags for filtering

	// Execution
	Script    []Prompt      `yaml:"script"`              // Prompts to send to the AI agent
	Prompt    string        `yaml:"prompt,omitempty"`    // Single prompt (legacy, converted to Script)
	Setup     string        `yaml:"setup,omitempty"`     // Setup script path (relative to task dir)
	Verifier  string        `yaml:"verifier,omitempty"`  // Verifier script path
	Cleanup   string        `yaml:"cleanup,omitempty"`   // Cleanup script path
	Timeout   string        `yaml:"timeout,omitempty"`   // Task timeout (default: 10m)
	Isolation TaskIsolation `yaml:"isolation,omitempty"` // Isolation level

	// Expectations
	Expect []Expectation `yaml:"expect,omitempty"` // Output expectations

	// Runtime (populated by loader)
	Dir string `yaml:"-"` // Directory containing the task
}

// LLMConfig represents the configuration for an LLM provider
type LLMConfig struct {
	ID       string `yaml:"id"`       // Unique identifier for this config
	Provider string `yaml:"provider"` // Provider name (openai, anthropic, ollama, etc.)
	Model    string `yaml:"model"`    // Model name
	Endpoint string `yaml:"endpoint"` // API endpoint (optional)
	APIKey   string `yaml:"apiKey"`   // API key (optional, can use env)

	// Behavioral settings
	Temperature   float64 `yaml:"temperature,omitempty"`
	MaxTokens     int     `yaml:"maxTokens,omitempty"`
	EnableToolUse bool    `yaml:"enableToolUse,omitempty"` // Enable tool/function calling
	EnableMCP     bool    `yaml:"enableMcp,omitempty"`     // Enable MCP integration
	AutoApprove   bool    `yaml:"autoApprove,omitempty"`   // Auto-approve tool executions
}

// Failure represents a single test failure
type Failure struct {
	Type     string `json:"type"`     // Expectation type that failed
	Expected string `json:"expected"` // What was expected
	Actual   string `json:"actual"`   // What was received
	Message  string `json:"message"`  // Human-readable message
}

// EvalResult represents the result of evaluating a single task with a specific LLM
type EvalResult struct {
	// Task info
	TaskID       string         `json:"taskId"`
	TaskName     string         `json:"taskName"`
	TaskCategory string         `json:"taskCategory,omitempty"`
	Difficulty   TaskDifficulty `json:"difficulty,omitempty"`

	// LLM info
	LLMConfig LLMConfig `json:"llmConfig"`

	// Result
	Result   TaskResult `json:"result"`
	Failures []Failure  `json:"failures,omitempty"`
	Error    string     `json:"error,omitempty"`

	// Timing
	StartTime time.Time     `json:"startTime"`
	EndTime   time.Time     `json:"endTime"`
	Duration  time.Duration `json:"duration"`

	// Output
	Output     string `json:"output,omitempty"`     // AI agent output
	SetupLog   string `json:"setupLog,omitempty"`   // Setup script output
	VerifyLog  string `json:"verifyLog,omitempty"`  // Verifier script output
	CleanupLog string `json:"cleanupLog,omitempty"` // Cleanup script output

	// Trace/Log files (k8s-ai-bench compatible)
	TracePath string      `json:"tracePath,omitempty"` // Path to trace.yaml
	LogPath   string      `json:"logPath,omitempty"`   // Path to log.txt
	Trace     *AgentTrace `json:"trace,omitempty"`     // Agent trace data

	// Metadata
	Attempt    int    `json:"attempt"`              // Attempt number (for retry)
	RunID      string `json:"runId,omitempty"`      // Unique run identifier
	Kubeconfig string `json:"kubeconfig,omitempty"` // Kubeconfig used
}

// AgentTrace represents the trace of an agent execution (k8s-ai-bench compatible)
type AgentTrace struct {
	Steps      []TraceStep `json:"steps,omitempty"`      // Execution steps
	TotalSteps int         `json:"totalSteps,omitempty"` // Total step count
	ToolCalls  int         `json:"toolCalls,omitempty"`  // Total tool calls
}

// TraceStep represents a single step in agent execution
type TraceStep struct {
	Type      string `json:"type,omitempty"`      // Step type (prompt, tool_call, response)
	Content   string `json:"content,omitempty"`   // Step content
	ToolName  string `json:"toolName,omitempty"`  // Tool name (if tool_call)
	ToolArgs  string `json:"toolArgs,omitempty"`  // Tool arguments (if tool_call)
	ToolOut   string `json:"toolOut,omitempty"`   // Tool output (if tool_call)
	Timestamp string `json:"timestamp,omitempty"` // ISO timestamp
}

// BenchmarkSummary provides aggregated results across all tasks and LLMs
type BenchmarkSummary struct {
	RunID     string        `json:"runId"`
	StartTime time.Time     `json:"startTime"`
	EndTime   time.Time     `json:"endTime"`
	Duration  time.Duration `json:"duration"`

	// Task statistics
	TotalTasks   int `json:"totalTasks"`
	SuccessCount int `json:"successCount"`
	FailCount    int `json:"failCount"`
	ErrorCount   int `json:"errorCount"`
	SkippedCount int `json:"skippedCount"`

	// Per-difficulty breakdown
	EasySuccess   int `json:"easySuccess"`
	EasyTotal     int `json:"easyTotal"`
	MediumSuccess int `json:"mediumSuccess"`
	MediumTotal   int `json:"mediumTotal"`
	HardSuccess   int `json:"hardSuccess"`
	HardTotal     int `json:"hardTotal"`

	// Per-LLM breakdown
	LLMResults map[string]*LLMSummary `json:"llmResults"`

	// Pass rates
	PassAt1 float64 `json:"passAt1"` // Single attempt pass rate
	PassAt5 float64 `json:"passAt5"` // Pass within 5 attempts
}

// LLMSummary provides per-LLM aggregated results
type LLMSummary struct {
	LLMConfig    LLMConfig     `json:"llmConfig"`
	TotalTasks   int           `json:"totalTasks"`
	SuccessCount int           `json:"successCount"`
	FailCount    int           `json:"failCount"`
	ErrorCount   int           `json:"errorCount"`
	PassRate     float64       `json:"passRate"`
	AvgDuration  time.Duration `json:"avgDuration"`
}

// RunConfig contains configuration for a benchmark run
type RunConfig struct {
	// Task selection
	TaskDir     string   `yaml:"taskDir"`               // Directory containing tasks
	TaskPattern string   `yaml:"taskPattern,omitempty"` // Regex pattern to filter tasks
	Tags        []string `yaml:"tags,omitempty"`        // Filter by tags
	Categories  []string `yaml:"categories,omitempty"`  // Filter by categories
	Difficulty  string   `yaml:"difficulty,omitempty"`  // Filter by difficulty

	// LLM configuration
	LLMConfigs []LLMConfig `yaml:"llmConfigs"` // LLMs to evaluate

	// Execution settings
	Parallelism    int    `yaml:"parallelism,omitempty"`    // Number of parallel workers
	DefaultTimeout string `yaml:"defaultTimeout,omitempty"` // Default task timeout
	Retries        int    `yaml:"retries,omitempty"`        // Number of retries per task

	// Cluster settings
	ClusterProvider       string        `yaml:"clusterProvider,omitempty"`       // kind, vcluster, or existing
	Kubeconfig            string        `yaml:"kubeconfig,omitempty"`            // Path to kubeconfig
	ClusterName           string        `yaml:"clusterName,omitempty"`           // Cluster name for Kind/vCluster
	ClusterCreationPolicy ClusterPolicy `yaml:"clusterCreationPolicy,omitempty"` // Cluster creation policy
	HostKubeconfig        string        `yaml:"hostKubeconfig,omitempty"`        // Host kubeconfig for vCluster

	// Output settings
	OutputDir    string `yaml:"outputDir,omitempty"`    // Directory for results
	OutputFormat string `yaml:"outputFormat,omitempty"` // json, yaml, markdown
	SaveTrace    bool   `yaml:"saveTrace,omitempty"`    // Save trace.yaml per task
	SaveLog      bool   `yaml:"saveLog,omitempty"`      // Save log.txt per task

	// Agent settings
	AgentBin          string   `yaml:"agentBin,omitempty"`          // Path to AI agent binary
	AgentArgs         []string `yaml:"agentArgs,omitempty"`         // Additional agent arguments
	EnableToolUseShim bool     `yaml:"enableToolUseShim,omitempty"` // Enable tool use shim for external agent
	AgentMaxTurns     int      `yaml:"agentMaxTurns,omitempty"`     // Max turns for agent
	AgentMaxTokens    int      `yaml:"agentMaxTokens,omitempty"`    // Max tokens for agent
	AgentSystemPrompt string   `yaml:"agentSystemPrompt,omitempty"` // Custom system prompt

	// UI settings
	Quiet bool `yaml:"quiet,omitempty"` // Suppress progress output
}

// ClusterPolicy defines how to handle cluster creation
type ClusterPolicy string

const (
	ClusterAlwaysCreate     ClusterPolicy = "always"        // Delete existing and create fresh
	ClusterCreateIfNotExist ClusterPolicy = "create_if_not" // Reuse if exists
	ClusterDoNotCreate      ClusterPolicy = "do_not_create" // Use provided kubeconfig
)
