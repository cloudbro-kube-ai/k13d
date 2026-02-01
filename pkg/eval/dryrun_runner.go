// Package eval provides AI agent benchmark evaluation framework
// dryrun_runner.go implements the dry-run benchmark runner
package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/yaml"
)

// DryRunRunner executes benchmark tasks in dry-run mode
type DryRunRunner struct {
	config       DryRunRunnerConfig
	tasks        map[string]DryRunBenchmarkTask
	validator    *DryRunValidator
	mockExecutor *MockToolExecutor
	mu           sync.Mutex
}

// DryRunRunnerConfig holds configuration for the dry-run runner
type DryRunRunnerConfig struct {
	TasksDir       string
	OutputDir      string
	TaskPattern    string
	Concurrency    int
	LLMConfigs     []LLMRunConfig
	Verbose        bool
	Mode           DryRunMode
	TimeoutPerTask time.Duration
}

// LLMRunConfig represents LLM configuration for dry-run evaluation
type LLMRunConfig struct {
	ID             string
	Provider       string
	Model          string
	Endpoint       string
	APIKey         string
	Temperature    float64
	MaxTokens      int
	EnableToolUse  bool
	EnableMCP      bool
	AutoApprove    bool
}

// NewDryRunRunner creates a new dry-run benchmark runner
func NewDryRunRunner(config DryRunRunnerConfig) *DryRunRunner {
	validator := NewDryRunValidator(DryRunConfig{
		Mode:    config.Mode,
		Verbose: config.Verbose,
	})

	mockExecutor := NewMockToolExecutor(DefaultKubectlMockResponses(), config.Verbose)

	return &DryRunRunner{
		config:       config,
		tasks:        make(map[string]DryRunBenchmarkTask),
		validator:    validator,
		mockExecutor: mockExecutor,
	}
}

// LoadTasks loads benchmark tasks with dry-run expectations
func (r *DryRunRunner) LoadTasks() error {
	var taskFilter *regexp.Regexp
	if r.config.TaskPattern != "" {
		var err error
		taskFilter, err = regexp.Compile(r.config.TaskPattern)
		if err != nil {
			return fmt.Errorf("compiling task pattern regex %q: %w", r.config.TaskPattern, err)
		}
	}

	entries, err := os.ReadDir(r.config.TasksDir)
	if err != nil {
		return fmt.Errorf("reading tasks directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskID := entry.Name()
		if taskFilter != nil && !taskFilter.MatchString(taskID) {
			continue
		}

		// Try to load dry-run task definition first
		dryrunFile := filepath.Join(r.config.TasksDir, taskID, "dryrun.yaml")
		taskFile := filepath.Join(r.config.TasksDir, taskID, "task.yaml")

		var task DryRunBenchmarkTask

		// Check for dry-run specific file
		if data, err := os.ReadFile(dryrunFile); err == nil {
			if err := yaml.Unmarshal(data, &task); err != nil {
				return fmt.Errorf("parsing dry-run file %s: %w", dryrunFile, err)
			}
		} else {
			// Fall back to regular task file
			data, err := os.ReadFile(taskFile)
			if err != nil {
				continue // Skip tasks without task.yaml
			}

			if err := yaml.Unmarshal(data, &task.BenchmarkTask); err != nil {
				return fmt.Errorf("parsing task file %s: %w", taskFile, err)
			}

			// Generate default expectations from task
			task.DryRunExpectations = r.generateDefaultExpectations(&task.BenchmarkTask)
		}

		if task.Disabled {
			if r.config.Verbose {
				fmt.Printf("Skipping disabled task: %s\n", taskID)
			}
			continue
		}

		task.ID = taskID
		r.tasks[taskID] = task

		// Register expectations with validator
		r.validator.SetExpectation(taskID, task.DryRunExpectations)
	}

	return nil
}

// generateDefaultExpectations creates default expectations based on task prompt
func (r *DryRunRunner) generateDefaultExpectations(task *BenchmarkTask) DryRunTaskExpectation {
	exp := DryRunTaskExpectation{
		MinToolCalls: 1, // At least one tool call expected
	}

	// Analyze prompt for expected commands
	for _, step := range task.Script {
		prompt := strings.ToLower(step.Prompt)

		// Create commands
		if strings.Contains(prompt, "create") {
			if strings.Contains(prompt, "pod") {
				exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
					Pattern:     `kubectl (create|apply|run).*pod`,
					Description: "Create a pod",
					Required:    true,
				})
			}
			if strings.Contains(prompt, "deployment") {
				exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
					Pattern:     `kubectl (create|apply).*deployment`,
					Description: "Create a deployment",
					Required:    true,
				})
			}
			if strings.Contains(prompt, "service") {
				exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
					Pattern:     `kubectl (create|apply|expose).*service|svc`,
					Description: "Create a service",
					Required:    true,
				})
			}
			if strings.Contains(prompt, "namespace") {
				exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
					Pattern:     `kubectl create namespace|kubectl create ns`,
					Description: "Create a namespace",
					Required:    true,
				})
			}
			if strings.Contains(prompt, "configmap") {
				exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
					Pattern:     `kubectl create configmap|kubectl create cm`,
					Description: "Create a configmap",
					Required:    true,
				})
			}
			if strings.Contains(prompt, "secret") {
				exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
					Pattern:     `kubectl create secret`,
					Description: "Create a secret",
					Required:    true,
				})
			}
		}

		// Scale commands
		if strings.Contains(prompt, "scale") {
			exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
				Pattern:     `kubectl scale`,
				Description: "Scale a resource",
				Required:    true,
			})
		}

		// Delete commands
		if strings.Contains(prompt, "delete") {
			exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
				Pattern:     `kubectl delete`,
				Description: "Delete a resource",
				Required:    true,
			})
		}

		// Label/Annotate commands
		if strings.Contains(prompt, "label") {
			exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
				Pattern:     `kubectl label`,
				Description: "Add/modify labels",
				Required:    true,
			})
		}
		if strings.Contains(prompt, "annotate") || strings.Contains(prompt, "annotation") {
			exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
				Pattern:     `kubectl annotate`,
				Description: "Add/modify annotations",
				Required:    true,
			})
		}

		// Rolling update commands
		if strings.Contains(prompt, "rolling") || strings.Contains(prompt, "rollout") || strings.Contains(prompt, "update") {
			exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
				Pattern:     `kubectl (set image|rollout|patch)`,
				Description: "Perform rolling update",
				Required:    true,
			})
		}

		// Debug commands
		if strings.Contains(prompt, "debug") || strings.Contains(prompt, "troubleshoot") || strings.Contains(prompt, "diagnose") {
			exp.RequiredCommands = append(exp.RequiredCommands, CommandExpectation{
				Pattern:     `kubectl (describe|logs|get events)`,
				Description: "Debug/troubleshoot resources",
				Required:    true,
			})
		}
	}

	// Add forbidden patterns for dangerous operations
	exp.ForbiddenPatterns = []string{
		`--force.*--grace-period=0`, // Force deletion without grace period
		`delete.*--all`,             // Delete all without namespace
		`rm\s+-rf\s+/`,              // Dangerous rm commands
	}

	return exp
}

// Run executes all loaded benchmark tasks in dry-run mode
func (r *DryRunRunner) Run(ctx context.Context) (*DryRunBenchmarkReport, error) {
	if len(r.tasks) == 0 {
		return nil, fmt.Errorf("no tasks loaded")
	}

	report := &DryRunBenchmarkReport{
		SuiteName: "k13d Dry-Run Benchmark",
		RunID:     fmt.Sprintf("dryrun-%d", time.Now().UnixNano()),
		Mode:      string(r.config.Mode),
		StartTime: time.Now(),
		TaskCount: len(r.tasks),
		Results:   make([]DryRunTaskResult, 0),
	}

	fmt.Printf("Starting dry-run benchmark with %d tasks...\n", len(r.tasks))

	concurrency := r.config.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	// Create channels for work distribution
	type taskJob struct {
		taskID string
		task   DryRunBenchmarkTask
	}
	taskCh := make(chan taskJob, len(r.tasks))
	resultsCh := make(chan DryRunTaskResult, len(r.tasks)*len(r.config.LLMConfigs))

	// Load tasks into channel
	for taskID, task := range r.tasks {
		taskCh <- taskJob{taskID: taskID, task: task}
	}
	close(taskCh)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for job := range taskCh {
				for _, llmConfig := range r.config.LLMConfigs {
					result := r.runDryRunTask(ctx, job.taskID, job.task, llmConfig)
					resultsCh <- result
				}
			}
		}(i)
	}

	// Wait for all workers
	wg.Wait()
	close(resultsCh)

	// Collect results
	for result := range resultsCh {
		report.Results = append(report.Results, result)
	}

	report.EndTime = time.Now()
	report.TotalDuration = report.EndTime.Sub(report.StartTime).String()

	// Calculate summary
	r.calculateSummary(report)

	return report, nil
}

// DryRunTaskResult represents the result of a single dry-run task
type DryRunTaskResult struct {
	TaskID          string            `json:"task_id"`
	TaskName        string            `json:"task_name"`
	Difficulty      TaskDifficulty    `json:"difficulty"`
	LLMConfig       LLMRunConfig      `json:"llm_config"`
	Success         bool              `json:"success"`
	Score           float64           `json:"score"`
	ToolCalls       []ToolCallRecord  `json:"tool_calls"`
	MatchedPatterns []string          `json:"matched_patterns"`
	MissedPatterns  []string          `json:"missed_patterns"`
	ForbiddenHits   []string          `json:"forbidden_hits,omitempty"`
	Errors          []string          `json:"errors,omitempty"`
	LLMResponse     string            `json:"llm_response,omitempty"`
	StartTime       time.Time         `json:"start_time"`
	EndTime         time.Time         `json:"end_time"`
	Duration        time.Duration     `json:"duration"`
}

// DryRunBenchmarkReport represents the complete dry-run evaluation report
type DryRunBenchmarkReport struct {
	SuiteName     string             `json:"suite_name"`
	RunID         string             `json:"run_id"`
	Mode          string             `json:"mode"`
	StartTime     time.Time          `json:"start_time"`
	EndTime       time.Time          `json:"end_time"`
	TotalDuration string             `json:"total_duration"`
	TaskCount     int                `json:"task_count"`
	Results       []DryRunTaskResult `json:"results"`
	Summary       DryRunSummary      `json:"summary"`
}

// DryRunSummary contains aggregate statistics for dry-run
type DryRunSummary struct {
	TotalTasks      int                           `json:"total_tasks"`
	PassedTasks     int                           `json:"passed_tasks"`
	FailedTasks     int                           `json:"failed_tasks"`
	PassRate        float64                       `json:"pass_rate"`
	AverageScore    float64                       `json:"average_score"`
	ByDifficulty    map[TaskDifficulty]DiffStat   `json:"by_difficulty"`
	ByModel         map[string]ModelDryRunStat    `json:"by_model"`
	TotalToolCalls  int                           `json:"total_tool_calls"`
	AvgToolCalls    float64                       `json:"avg_tool_calls"`
}

// DiffStat holds statistics for a difficulty level
type DiffStat struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"pass_rate"`
	AvgScore float64 `json:"avg_score"`
}

// ModelDryRunStat holds per-model statistics
type ModelDryRunStat struct {
	ModelID      string  `json:"model_id"`
	Provider     string  `json:"provider"`
	TotalTasks   int     `json:"total_tasks"`
	PassedTasks  int     `json:"passed_tasks"`
	PassRate     float64 `json:"pass_rate"`
	AverageScore float64 `json:"average_score"`
	AvgToolCalls float64 `json:"avg_tool_calls"`
}

// runDryRunTask executes a single task in dry-run mode
func (r *DryRunRunner) runDryRunTask(ctx context.Context, taskID string, task DryRunBenchmarkTask, llmConfig LLMRunConfig) DryRunTaskResult {
	result := DryRunTaskResult{
		TaskID:     taskID,
		TaskName:   task.Name,
		Difficulty: task.Difficulty,
		LLMConfig:  llmConfig,
		StartTime:  time.Now(),
	}

	timeout := r.config.TimeoutPerTask
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	taskCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Get prompt
	prompt := ""
	for _, step := range task.Script {
		prompt += step.Prompt + "\n"
	}

	if r.config.Verbose {
		fmt.Printf("[%s] Running with %s/%s...\n", taskID, llmConfig.Provider, llmConfig.Model)
	}

	// Simulate LLM call and collect tool calls
	toolCalls, llmResponse, err := r.simulateLLMWithToolCalls(taskCtx, prompt, llmConfig)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("LLM error: %v", err))
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	result.ToolCalls = toolCalls
	result.LLMResponse = llmResponse

	// Validate tool calls
	validation := r.validator.Validate(taskID, toolCalls)
	result.Success = validation.Success
	result.Score = validation.Score
	result.MatchedPatterns = validation.MatchedPatterns
	result.MissedPatterns = validation.MissedPatterns
	result.ForbiddenHits = validation.ForbiddenHits
	result.Errors = append(result.Errors, validation.Errors...)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if r.config.Verbose {
		status := "PASS"
		if !result.Success {
			status = "FAIL"
		}
		fmt.Printf("[%s] %s (score: %.2f, tool calls: %d)\n", taskID, status, result.Score, len(toolCalls))
	}

	return result
}

// simulateLLMWithToolCalls simulates an LLM call and returns tool calls
// In a real implementation, this would call the actual LLM API
func (r *DryRunRunner) simulateLLMWithToolCalls(ctx context.Context, prompt string, llmConfig LLMRunConfig) ([]ToolCallRecord, string, error) {
	// This is a placeholder for the actual LLM API call
	// In production, you would:
	// 1. Create an AI client with the given config
	// 2. Send the prompt with tool definitions
	// 3. Collect tool calls made by the LLM
	// 4. Return mock responses for each tool call
	// 5. Let LLM continue until it stops making tool calls

	// For now, we'll use a simplified simulation based on the prompt
	toolCalls := r.analyzePromptForExpectedCommands(prompt)

	llmResponse := fmt.Sprintf("Based on your request, I will help you. [Simulated response for: %s]",
		truncateString(prompt, 100))

	return toolCalls, llmResponse, nil
}

// analyzePromptForExpectedCommands analyzes a prompt and generates expected tool calls
func (r *DryRunRunner) analyzePromptForExpectedCommands(prompt string) []ToolCallRecord {
	var toolCalls []ToolCallRecord
	prompt = strings.ToLower(prompt)

	// Generate tool calls based on keywords in the prompt
	callID := 0

	// Pod operations
	if strings.Contains(prompt, "create") && strings.Contains(prompt, "pod") {
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl run nginx-pod --image=nginx:1.25",
			Timestamp: time.Now(),
		})
	}

	// Deployment operations
	if strings.Contains(prompt, "create") && strings.Contains(prompt, "deployment") {
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl create deployment nginx-deployment --image=nginx:1.25",
			Timestamp: time.Now(),
		})
	}

	// Service operations
	if strings.Contains(prompt, "create") && strings.Contains(prompt, "service") {
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl expose deployment nginx-deployment --port=80",
			Timestamp: time.Now(),
		})
	}

	// Namespace operations
	if strings.Contains(prompt, "create") && strings.Contains(prompt, "namespace") {
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl create namespace test-ns",
			Timestamp: time.Now(),
		})
	}

	// ConfigMap operations
	if strings.Contains(prompt, "create") && strings.Contains(prompt, "configmap") {
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl create configmap my-config --from-literal=key=value",
			Timestamp: time.Now(),
		})
	}

	// Secret operations
	if strings.Contains(prompt, "create") && strings.Contains(prompt, "secret") {
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl create secret generic my-secret --from-literal=password=secret",
			Timestamp: time.Now(),
		})
	}

	// Scale operations
	if strings.Contains(prompt, "scale") {
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl scale deployment nginx-deployment --replicas=3",
			Timestamp: time.Now(),
		})
	}

	// Label operations
	if strings.Contains(prompt, "label") {
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl label pod nginx-pod app=web",
			Timestamp: time.Now(),
		})
	}

	// Delete operations
	if strings.Contains(prompt, "delete") {
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl delete pod nginx-pod",
			Timestamp: time.Now(),
		})
	}

	// Debug/troubleshoot operations
	if strings.Contains(prompt, "debug") || strings.Contains(prompt, "troubleshoot") || strings.Contains(prompt, "diagnose") {
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl describe pod nginx-pod",
			Timestamp: time.Now(),
		})
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl logs nginx-pod",
			Timestamp: time.Now(),
		})
	}

	// If no specific commands detected, add a generic get command
	if len(toolCalls) == 0 {
		callID++
		toolCalls = append(toolCalls, ToolCallRecord{
			ID:        fmt.Sprintf("call_%d", callID),
			ToolName:  "kubectl",
			Command:   "kubectl get pods",
			Timestamp: time.Now(),
		})
	}

	return toolCalls
}

// calculateSummary calculates report summary statistics
func (r *DryRunRunner) calculateSummary(report *DryRunBenchmarkReport) {
	summary := DryRunSummary{
		ByDifficulty: make(map[TaskDifficulty]DiffStat),
		ByModel:      make(map[string]ModelDryRunStat),
	}

	totalScore := 0.0
	totalToolCalls := 0

	for _, result := range report.Results {
		summary.TotalTasks++
		totalScore += result.Score
		totalToolCalls += len(result.ToolCalls)

		if result.Success {
			summary.PassedTasks++
		} else {
			summary.FailedTasks++
		}

		// By difficulty
		diff := result.Difficulty
		if diff == "" {
			diff = DifficultyMedium
		}
		ds := summary.ByDifficulty[diff]
		ds.Total++
		if result.Success {
			ds.Passed++
		}
		ds.PassRate = float64(ds.Passed) / float64(ds.Total) * 100
		ds.AvgScore = (ds.AvgScore*float64(ds.Total-1) + result.Score) / float64(ds.Total)
		summary.ByDifficulty[diff] = ds

		// By model
		modelKey := fmt.Sprintf("%s/%s", result.LLMConfig.Provider, result.LLMConfig.Model)
		ms := summary.ByModel[modelKey]
		ms.ModelID = result.LLMConfig.Model
		ms.Provider = result.LLMConfig.Provider
		ms.TotalTasks++
		if result.Success {
			ms.PassedTasks++
		}
		ms.PassRate = float64(ms.PassedTasks) / float64(ms.TotalTasks) * 100
		ms.AverageScore = (ms.AverageScore*float64(ms.TotalTasks-1) + result.Score) / float64(ms.TotalTasks)
		ms.AvgToolCalls = (ms.AvgToolCalls*float64(ms.TotalTasks-1) + float64(len(result.ToolCalls))) / float64(ms.TotalTasks)
		summary.ByModel[modelKey] = ms
	}

	if summary.TotalTasks > 0 {
		summary.PassRate = float64(summary.PassedTasks) / float64(summary.TotalTasks) * 100
		summary.AverageScore = totalScore / float64(summary.TotalTasks)
		summary.TotalToolCalls = totalToolCalls
		summary.AvgToolCalls = float64(totalToolCalls) / float64(summary.TotalTasks)
	}

	report.Summary = summary
}

// FormatMarkdown formats the dry-run report as Markdown
func (r *DryRunBenchmarkReport) FormatMarkdown() string {
	var sb strings.Builder

	sb.WriteString("# k13d Dry-Run Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Run ID:** %s\n", r.RunID))
	sb.WriteString(fmt.Sprintf("**Mode:** %s\n", r.Mode))
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n", r.TotalDuration))
	sb.WriteString(fmt.Sprintf("**Tasks:** %d\n\n", r.TaskCount))

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Pass Rate:** %.1f%%\n", r.Summary.PassRate))
	sb.WriteString(fmt.Sprintf("- **Average Score:** %.2f\n", r.Summary.AverageScore))
	sb.WriteString(fmt.Sprintf("- **Passed:** %d\n", r.Summary.PassedTasks))
	sb.WriteString(fmt.Sprintf("- **Failed:** %d\n", r.Summary.FailedTasks))
	sb.WriteString(fmt.Sprintf("- **Total Tool Calls:** %d (avg: %.1f per task)\n\n", r.Summary.TotalToolCalls, r.Summary.AvgToolCalls))

	// By Difficulty
	sb.WriteString("## Results by Difficulty\n\n")
	sb.WriteString("| Difficulty | Total | Passed | Pass Rate | Avg Score |\n")
	sb.WriteString("|------------|-------|--------|-----------|----------|\n")
	for diff, stat := range r.Summary.ByDifficulty {
		sb.WriteString(fmt.Sprintf("| %s | %d | %d | %.1f%% | %.2f |\n",
			diff, stat.Total, stat.Passed, stat.PassRate, stat.AvgScore))
	}
	sb.WriteString("\n")

	// By Model
	if len(r.Summary.ByModel) > 0 {
		sb.WriteString("## Results by Model\n\n")
		sb.WriteString("| Model | Provider | Total | Pass Rate | Avg Score | Avg Tool Calls |\n")
		sb.WriteString("|-------|----------|-------|-----------|-----------|----------------|\n")
		for _, stat := range r.Summary.ByModel {
			sb.WriteString(fmt.Sprintf("| %s | %s | %d | %.1f%% | %.2f | %.1f |\n",
				stat.ModelID, stat.Provider, stat.TotalTasks, stat.PassRate, stat.AverageScore, stat.AvgToolCalls))
		}
		sb.WriteString("\n")
	}

	// Detailed Results
	sb.WriteString("## Detailed Results\n\n")
	for _, result := range r.Results {
		emoji := "✅"
		if !result.Success {
			emoji = "❌"
		}
		sb.WriteString(fmt.Sprintf("### %s %s\n\n", emoji, result.TaskID))
		sb.WriteString(fmt.Sprintf("- **Difficulty:** %s\n", result.Difficulty))
		sb.WriteString(fmt.Sprintf("- **Model:** %s/%s\n", result.LLMConfig.Provider, result.LLMConfig.Model))
		sb.WriteString(fmt.Sprintf("- **Score:** %.2f\n", result.Score))
		sb.WriteString(fmt.Sprintf("- **Tool Calls:** %d\n", len(result.ToolCalls)))
		sb.WriteString(fmt.Sprintf("- **Duration:** %s\n", result.Duration))

		if len(result.MatchedPatterns) > 0 {
			sb.WriteString("- **Matched:** " + strings.Join(result.MatchedPatterns, ", ") + "\n")
		}
		if len(result.MissedPatterns) > 0 {
			sb.WriteString("- **Missed:** " + strings.Join(result.MissedPatterns, ", ") + "\n")
		}
		if len(result.Errors) > 0 {
			sb.WriteString("- **Errors:** " + strings.Join(result.Errors, "; ") + "\n")
		}

		// Show tool calls
		if len(result.ToolCalls) > 0 {
			sb.WriteString("\n**Tool Calls:**\n```\n")
			for _, tc := range result.ToolCalls {
				sb.WriteString(fmt.Sprintf("- %s\n", tc.Command))
			}
			sb.WriteString("```\n")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// SaveReport saves the report to a file
func (r *DryRunBenchmarkReport) SaveReport(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Save JSON report
	jsonPath := filepath.Join(outputDir, fmt.Sprintf("dryrun-report-%s.json", time.Now().Format("20060102-150405")))
	jsonData, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return err
	}

	// Save Markdown report
	mdPath := filepath.Join(outputDir, fmt.Sprintf("dryrun-report-%s.md", time.Now().Format("20060102-150405")))
	if err := os.WriteFile(mdPath, []byte(r.FormatMarkdown()), 0644); err != nil {
		return err
	}

	fmt.Printf("Reports saved to:\n  - %s\n  - %s\n", jsonPath, mdPath)
	return nil
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// GetTasks returns loaded tasks
func (r *DryRunRunner) GetTasks() map[string]DryRunBenchmarkTask {
	return r.tasks
}
