package eval

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/yaml"
)

// RunnerConfig holds configuration for the benchmark runner
type RunnerConfig struct {
	TasksDir    string
	OutputDir   string
	TaskPattern string
	Concurrency int
	KubeConfig  string
	LLMConfigs  []LLMConfig
	Verbose     bool
}

// Runner executes benchmark tasks
type Runner struct {
	config   RunnerConfig
	agentBin string
	tasks    map[string]BenchmarkTask
	mu       sync.Mutex
}

// NewRunner creates a new benchmark runner
func NewRunner(config RunnerConfig, agentBin string) *Runner {
	return &Runner{
		config:   config,
		agentBin: agentBin,
		tasks:    make(map[string]BenchmarkTask),
	}
}

// LoadTasks loads benchmark tasks from the tasks directory
func (r *Runner) LoadTasks() error {
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

		taskFile := filepath.Join(r.config.TasksDir, taskID, "task.yaml")
		data, err := os.ReadFile(taskFile)
		if err != nil {
			continue // Skip tasks without task.yaml
		}

		var task BenchmarkTask
		if err := yaml.Unmarshal(data, &task); err != nil {
			return fmt.Errorf("parsing task file %s: %w", taskFile, err)
		}

		if task.Disabled {
			if r.config.Verbose {
				fmt.Printf("Skipping disabled task: %s\n", taskID)
			}
			continue
		}

		task.ID = taskID
		r.tasks[taskID] = task
	}

	return nil
}

// Run executes all loaded benchmark tasks
func (r *Runner) Run(ctx context.Context) (*BenchmarkReport, error) {
	if len(r.tasks) == 0 {
		return nil, fmt.Errorf("no tasks loaded")
	}

	report := &BenchmarkReport{
		SuiteName:  "k13d Benchmark",
		RunID:      generateRunID(),
		StartTime:  time.Now(),
		LLMConfigs: r.config.LLMConfigs,
		TaskCount:  len(r.tasks),
		Results:    make([]TaskResult, 0),
	}

	concurrency := r.config.Concurrency
	if concurrency <= 0 {
		concurrency = 1
	}

	// Create channels for work distribution
	type taskJob struct {
		taskID string
		task   BenchmarkTask
	}
	taskCh := make(chan taskJob, len(r.tasks))
	resultsCh := make(chan TaskResult, len(r.tasks)*len(r.config.LLMConfigs))

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
					result := r.runTask(ctx, job.taskID, job.task, llmConfig)
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

// runTask executes a single task
func (r *Runner) runTask(ctx context.Context, taskID string, task BenchmarkTask, llmConfig LLMConfig) TaskResult {
	result := TaskResult{
		Task:      taskID,
		LLMConfig: llmConfig,
		StartTime: time.Now(),
	}

	// Parse timeout
	timeout := 10 * time.Minute
	if task.Timeout != "" {
		if parsed, err := time.ParseDuration(task.Timeout); err == nil {
			timeout = parsed
		}
	}

	taskCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	taskDir := filepath.Join(r.config.TasksDir, taskID)

	// Run setup
	if task.Setup != "" {
		if err := r.runScript(taskCtx, taskDir, task.Setup); err != nil {
			result.Result = "error"
			result.Error = fmt.Sprintf("setup failed: %v", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime).String()
			return result
		}
	}

	// Cleanup on exit
	defer func() {
		if task.Cleanup != "" {
			r.runScript(context.Background(), taskDir, task.Cleanup)
		}
	}()

	// Run agent
	agentOutput, err := r.runAgent(taskCtx, taskDir, task, llmConfig)
	if err != nil {
		if taskCtx.Err() == context.DeadlineExceeded {
			result.Result = "fail"
			result.AddFailure("task timed out after %v", timeout)
		} else {
			result.Result = "error"
			result.Error = fmt.Sprintf("agent error: %v", err)
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).String()
		return result
	}

	// Check expectations
	expectationsPassed := true
	for _, expect := range task.Expect {
		if expect.Contains != "" {
			re, err := regexp.Compile(expect.Contains)
			if err != nil {
				result.AddFailure("invalid regex %q: %v", expect.Contains, err)
				expectationsPassed = false
				continue
			}
			if !re.MatchString(agentOutput) {
				result.AddFailure("expected pattern %q not found in output", expect.Contains)
				expectationsPassed = false
			}
		}
		if expect.NotContains != "" {
			re, err := regexp.Compile(expect.NotContains)
			if err != nil {
				result.AddFailure("invalid regex %q: %v", expect.NotContains, err)
				expectationsPassed = false
				continue
			}
			if re.MatchString(agentOutput) {
				result.AddFailure("unexpected pattern %q found in output", expect.NotContains)
				expectationsPassed = false
			}
		}
	}

	// Run verifier
	verifierPassed := false
	if task.Verifier != "" {
		if err := r.runScript(taskCtx, taskDir, task.Verifier); err == nil {
			verifierPassed = true
		} else {
			result.AddFailure("verifier failed: %v", err)
		}
	}

	// Determine result
	if verifierPassed || (len(task.Expect) > 0 && expectationsPassed) {
		result.Result = "success"
	} else {
		result.Result = "fail"
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime).String()
	return result
}

// runScript executes a shell script
func (r *Runner) runScript(ctx context.Context, taskDir, script string) error {
	scriptPath := filepath.Join(taskDir, script)

	cmd := exec.CommandContext(ctx, scriptPath)
	cmd.Dir = taskDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("KUBECONFIG=%s", r.config.KubeConfig),
		fmt.Sprintf("NAMESPACE=%s", "benchmark-"+filepath.Base(taskDir)),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	return nil
}

// runAgent runs the AI agent with the task prompt
func (r *Runner) runAgent(ctx context.Context, taskDir string, task BenchmarkTask, llmConfig LLMConfig) (string, error) {
	// Resolve prompts
	var prompts []string
	for _, step := range task.Script {
		prompt := step.Prompt
		if step.PromptFile != "" {
			promptPath := filepath.Join(taskDir, step.PromptFile)
			data, err := os.ReadFile(promptPath)
			if err != nil {
				return "", fmt.Errorf("reading prompt file %s: %w", promptPath, err)
			}
			prompt = string(data)
		}
		prompts = append(prompts, prompt)
	}

	// Build agent arguments
	args := []string{
		"--kubeconfig", r.config.KubeConfig,
		"--provider", llmConfig.ProviderID,
		"--model", llmConfig.ModelID,
		"--quiet",
		"--skip-permissions",
	}

	// Create agent process with stdin pipe
	cmd := exec.CommandContext(ctx, r.agentBin, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", r.config.KubeConfig))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	// Send prompts
	for _, prompt := range prompts {
		fmt.Fprintln(stdin, prompt)
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return stdout.String(), err
	}

	return stdout.String(), nil
}

// calculateSummary calculates report summary statistics
func (r *Runner) calculateSummary(report *BenchmarkReport) {
	summary := ReportSummary{
		CategoryStats:   make(map[TaskCategory]CategoryStat),
		DifficultyStats: make(map[TaskDifficulty]DifficultyStat),
	}

	modelStats := make(map[string]*ModelSummary)

	for _, result := range report.Results {
		summary.TotalTasks++

		// Get task for metadata
		task := r.tasks[result.Task]

		switch result.Result {
		case "success":
			summary.PassedTasks++
		case "fail":
			summary.FailedTasks++
		default:
			summary.ErrorTasks++
		}

		// Category stats
		cat := task.Category
		if cat == "" {
			cat = "uncategorized"
		}
		cs := summary.CategoryStats[cat]
		cs.Total++
		if result.Result == "success" {
			cs.Passed++
		}
		cs.Rate = float64(cs.Passed) / float64(cs.Total) * 100
		summary.CategoryStats[cat] = cs

		// Difficulty stats
		diff := task.Difficulty
		if diff == "" {
			diff = DifficultyMedium
		}
		ds := summary.DifficultyStats[diff]
		ds.Total++
		if result.Result == "success" {
			ds.Passed++
		}
		ds.Rate = float64(ds.Passed) / float64(ds.Total) * 100
		summary.DifficultyStats[diff] = ds

		// Model stats
		modelKey := result.LLMConfig.ModelID
		if _, ok := modelStats[modelKey]; !ok {
			modelStats[modelKey] = &ModelSummary{
				ModelID:    result.LLMConfig.ModelID,
				ProviderID: result.LLMConfig.ProviderID,
			}
		}
		ms := modelStats[modelKey]
		ms.TotalTasks++
		switch result.Result {
		case "success":
			ms.PassedTasks++
		case "fail":
			ms.FailedTasks++
		default:
			ms.ErrorTasks++
		}
		ms.PassRate = float64(ms.PassedTasks) / float64(ms.TotalTasks) * 100
	}

	if summary.TotalTasks > 0 {
		summary.PassRate = float64(summary.PassedTasks) / float64(summary.TotalTasks) * 100
	}

	report.Summary = summary

	// Convert model stats to slice
	for _, ms := range modelStats {
		report.ModelComparison = append(report.ModelComparison, *ms)
	}
}

// generateRunID generates a unique run ID
func generateRunID() string {
	return fmt.Sprintf("run-%d", time.Now().UnixNano())
}

// GetTasks returns loaded tasks
func (r *Runner) GetTasks() map[string]BenchmarkTask {
	return r.tasks
}

// ResolvePrompt resolves a prompt from inline or file source
func (s *ScriptStep) ResolvePrompt(baseDir string) (string, error) {
	if s.Prompt != "" && s.PromptFile != "" {
		return "", fmt.Errorf("both 'prompt' and 'promptFile' specified")
	}

	if s.PromptFile != "" {
		promptPath := s.PromptFile
		if !filepath.IsAbs(promptPath) {
			promptPath = filepath.Join(baseDir, s.PromptFile)
		}
		content, err := os.ReadFile(promptPath)
		if err != nil {
			return "", fmt.Errorf("reading prompt file %q: %w", promptPath, err)
		}
		return string(content), nil
	}

	if s.Prompt != "" {
		return s.Prompt, nil
	}

	return "", fmt.Errorf("neither 'prompt' nor 'promptFile' specified")
}

// FormatMarkdown formats the report as Markdown
func (r *BenchmarkReport) FormatMarkdown() string {
	var sb strings.Builder

	sb.WriteString("# k13d Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Run ID:** %s\n", r.RunID))
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n", r.TotalDuration))
	sb.WriteString(fmt.Sprintf("**Tasks:** %d\n\n", r.TaskCount))

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Pass Rate:** %.1f%%\n", r.Summary.PassRate))
	sb.WriteString(fmt.Sprintf("- **Passed:** %d\n", r.Summary.PassedTasks))
	sb.WriteString(fmt.Sprintf("- **Failed:** %d\n", r.Summary.FailedTasks))
	sb.WriteString(fmt.Sprintf("- **Errors:** %d\n\n", r.Summary.ErrorTasks))

	// Model Comparison
	if len(r.ModelComparison) > 0 {
		sb.WriteString("## Model Comparison\n\n")
		sb.WriteString("| Model | Provider | Pass Rate | Passed | Failed | Errors |\n")
		sb.WriteString("|-------|----------|-----------|--------|--------|--------|\n")
		for _, m := range r.ModelComparison {
			sb.WriteString(fmt.Sprintf("| %s | %s | %.1f%% | %d | %d | %d |\n",
				m.ModelID, m.ProviderID, m.PassRate, m.PassedTasks, m.FailedTasks, m.ErrorTasks))
		}
		sb.WriteString("\n")
	}

	// Category Stats
	if len(r.Summary.CategoryStats) > 0 {
		sb.WriteString("## By Category\n\n")
		sb.WriteString("| Category | Pass Rate | Passed | Total |\n")
		sb.WriteString("|----------|-----------|--------|-------|\n")
		for cat, stat := range r.Summary.CategoryStats {
			sb.WriteString(fmt.Sprintf("| %s | %.1f%% | %d | %d |\n",
				cat, stat.Rate, stat.Passed, stat.Total))
		}
		sb.WriteString("\n")
	}

	// Difficulty Stats
	if len(r.Summary.DifficultyStats) > 0 {
		sb.WriteString("## By Difficulty\n\n")
		sb.WriteString("| Difficulty | Pass Rate | Passed | Total |\n")
		sb.WriteString("|------------|-----------|--------|-------|\n")
		for diff, stat := range r.Summary.DifficultyStats {
			sb.WriteString(fmt.Sprintf("| %s | %.1f%% | %d | %d |\n",
				diff, stat.Rate, stat.Passed, stat.Total))
		}
		sb.WriteString("\n")
	}

	// Detailed Results
	sb.WriteString("## Detailed Results\n\n")
	sb.WriteString("| Task | Model | Result | Duration |\n")
	sb.WriteString("|------|-------|--------|----------|\n")
	for _, result := range r.Results {
		emoji := "❌"
		if result.Result == "success" {
			emoji = "✅"
		} else if result.Result == "error" {
			emoji = "⚠️"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s %s | %s |\n",
			result.Task, result.LLMConfig.ModelID, emoji, result.Result, result.Duration))
	}

	return sb.String()
}
