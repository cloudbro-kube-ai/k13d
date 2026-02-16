package bench

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/bench/cluster"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

// Runner executes benchmark tasks
type Runner struct {
	config   *RunConfig
	provider cluster.Provider
	runID    string

	// Runtime state
	mu      sync.Mutex
	results []*EvalResult

	// Logging
	quiet bool
}

// NewRunner creates a new benchmark runner
func NewRunner(cfg *RunConfig) (*Runner, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate task directory
	if cfg.TaskDir == "" {
		return nil, fmt.Errorf("task directory is required")
	}
	if info, err := os.Stat(cfg.TaskDir); err != nil {
		return nil, fmt.Errorf("task directory not accessible: %w", err)
	} else if !info.IsDir() {
		return nil, fmt.Errorf("task directory is not a directory: %s", cfg.TaskDir)
	}

	// Validate LLM configs
	if len(cfg.LLMConfigs) == 0 {
		return nil, fmt.Errorf("at least one LLM config is required")
	}
	for i, llmCfg := range cfg.LLMConfigs {
		if llmCfg.ID == "" {
			return nil, fmt.Errorf("LLM config #%d: ID is required", i+1)
		}
		if llmCfg.Provider == "" && llmCfg.Endpoint == "" {
			return nil, fmt.Errorf("LLM config #%d (%s): provider or endpoint is required", i+1, llmCfg.ID)
		}
	}

	// Set defaults
	if cfg.DefaultTimeout == "" {
		cfg.DefaultTimeout = "10m"
	}
	if cfg.Parallelism == 0 {
		cfg.Parallelism = 1
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = ".build"
	}

	// Create cluster provider
	var provider cluster.Provider
	var err error

	switch cfg.ClusterProvider {
	case "kind":
		provider, err = cluster.NewProvider(cluster.ProviderKind,
			cluster.WithWorkDir(cfg.OutputDir))
	case "vcluster":
		provider, err = cluster.NewProvider(cluster.ProviderVCluster,
			cluster.WithVClusterKubeconfig(cfg.Kubeconfig),
			cluster.WithWorkDir(cfg.OutputDir))
	case "", "existing":
		provider, err = cluster.NewProvider(cluster.ProviderExisting,
			cluster.WithExistingKubeconfig(cfg.Kubeconfig))
	default:
		return nil, fmt.Errorf("unknown cluster provider: %s", cfg.ClusterProvider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create cluster provider: %w", err)
	}

	return &Runner{
		config:   cfg,
		provider: provider,
		runID:    uuid.New().String()[:8],
		results:  make([]*EvalResult, 0),
		quiet:    cfg.Quiet,
	}, nil
}

// Run executes the benchmark
func (r *Runner) Run(ctx context.Context) (*BenchmarkSummary, error) {
	startTime := time.Now()

	// Create output directory
	if err := os.MkdirAll(r.config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Load tasks
	loader := NewLoader(r.config.TaskDir)
	tasks, err := loader.LoadTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	// Filter tasks
	tasks, err = loader.FilterTasks(tasks, FilterOptions{
		Pattern:    r.config.TaskPattern,
		Difficulty: r.config.Difficulty,
		Categories: r.config.Categories,
		Tags:       r.config.Tags,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to filter tasks: %w", err)
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found matching criteria")
	}

	r.log("Found %d tasks to evaluate\n", len(tasks))

	// Setup cluster if needed
	if err := r.setupCluster(ctx); err != nil {
		return nil, fmt.Errorf("failed to setup cluster: %w", err)
	}

	// Create work items (task × LLM config combinations)
	type workItem struct {
		task      *Task
		llmConfig LLMConfig
	}

	var workItems []workItem
	for _, task := range tasks {
		for _, llmCfg := range r.config.LLMConfigs {
			workItems = append(workItems, workItem{task: task, llmConfig: llmCfg})
		}
	}

	// Run evaluations with parallelism
	results := make(chan *EvalResult, len(workItems))
	sem := make(chan struct{}, r.config.Parallelism)

	var wg sync.WaitGroup
	for _, item := range workItems {
		wg.Add(1)
		go func(task *Task, llmCfg LLMConfig) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			result := r.evaluateTask(ctx, task, llmCfg)
			results <- result

			// Save individual result
			r.saveResult(result)
		}(item.task, item.llmConfig)
	}

	// Close results channel when all work is done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		r.mu.Lock()
		r.results = append(r.results, result)
		r.mu.Unlock()

		// Print progress
		status := "✓"
		if result.Result != ResultSuccess {
			status = "✗"
		}
		r.log("[%s] %s (%s) - %s\n", status, result.TaskID, result.LLMConfig.ID, result.Result)
	}

	// Generate summary
	summary := r.generateSummary(startTime, time.Now())

	return summary, nil
}

// evaluateTask runs a single task evaluation
func (r *Runner) evaluateTask(ctx context.Context, task *Task, llmCfg LLMConfig) *EvalResult {
	result := &EvalResult{
		TaskID:       task.ID,
		TaskName:     task.Name,
		TaskCategory: task.Category,
		Difficulty:   task.Difficulty,
		LLMConfig:    llmCfg,
		StartTime:    time.Now(),
		RunID:        r.runID,
		Attempt:      1,
	}

	// Parse timeout
	timeout, err := time.ParseDuration(task.Timeout)
	if err != nil {
		timeout, _ = time.ParseDuration(r.config.DefaultTimeout)
	}

	taskCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Get kubeconfig
	kubeconfigPath, err := r.provider.GetKubeconfigPath(taskCtx, r.config.ClusterName)
	if err != nil {
		result.Result = ResultError
		result.Error = fmt.Sprintf("failed to get kubeconfig: %v", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}
	result.Kubeconfig = kubeconfigPath

	// Create isolated namespace for task
	namespace := fmt.Sprintf("bench-%s-%s", task.ID, r.runID)
	if err := r.createNamespace(taskCtx, kubeconfigPath, namespace); err != nil {
		result.Result = ResultError
		result.Error = fmt.Sprintf("failed to create namespace: %v", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}
	defer r.deleteNamespace(context.Background(), kubeconfigPath, namespace) // Cleanup even if task fails

	// Run setup script
	if task.Setup != "" {
		setupLog, err := r.runScript(taskCtx, task.GetSetupPath(), kubeconfigPath, namespace, task.Dir)
		result.SetupLog = setupLog
		if err != nil {
			result.Result = ResultError
			result.Error = fmt.Sprintf("setup failed: %v", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}
	}

	// Run AI agent
	output, err := r.runAgent(taskCtx, task, llmCfg, kubeconfigPath, namespace)
	result.Output = output
	if err != nil {
		if taskCtx.Err() == context.DeadlineExceeded {
			result.Result = ResultTimeout
			result.Error = "task timed out"
		} else {
			result.Result = ResultError
			result.Error = fmt.Sprintf("agent failed: %v", err)
		}
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Check expectations
	failures := r.checkExpectations(task, output)
	if len(failures) > 0 {
		result.Result = ResultFail
		result.Failures = failures
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Run verifier script
	if task.Verifier != "" {
		verifyLog, err := r.runScript(taskCtx, task.GetVerifierPath(), kubeconfigPath, namespace, task.Dir)
		result.VerifyLog = verifyLog
		if err != nil {
			result.Result = ResultFail
			result.Failures = []Failure{{
				Type:    "verifier",
				Message: fmt.Sprintf("verifier failed: %v", err),
				Actual:  verifyLog,
			}}
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}
	}

	// Run cleanup script (best effort)
	if task.Cleanup != "" {
		cleanupLog, _ := r.runScript(context.Background(), task.GetCleanupPath(), kubeconfigPath, namespace, task.Dir)
		result.CleanupLog = cleanupLog
	}

	result.Result = ResultSuccess
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	return result
}

// runAgent executes the AI agent with the task prompts
func (r *Runner) runAgent(ctx context.Context, task *Task, llmCfg LLMConfig, kubeconfig, namespace string) (string, error) {
	// If agent binary is specified, use it
	if r.config.AgentBin != "" {
		return r.runExternalAgent(ctx, task, llmCfg, kubeconfig, namespace)
	}

	// Otherwise use built-in AI client
	return r.runBuiltinAgent(ctx, task, llmCfg, kubeconfig, namespace)
}

// runExternalAgent runs an external agent binary
func (r *Runner) runExternalAgent(ctx context.Context, task *Task, llmCfg LLMConfig, kubeconfig, namespace string) (string, error) {
	args := append([]string{}, r.config.AgentArgs...)
	args = append(args,
		"--kubeconfig", kubeconfig,
		"--namespace", namespace,
	)

	// Add enhanced agent arguments (k8s-ai-bench compatible)
	if r.config.EnableToolUseShim {
		args = append(args, "--enable-tool-use-shim")
	}
	if r.config.AgentMaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", r.config.AgentMaxTurns))
	}
	if r.config.AgentMaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", r.config.AgentMaxTokens))
	}
	if llmCfg.Provider != "" {
		args = append(args, "--provider", llmCfg.Provider)
	}
	if llmCfg.Model != "" {
		args = append(args, "--model", llmCfg.Model)
	}
	if llmCfg.Endpoint != "" {
		args = append(args, "--endpoint", llmCfg.Endpoint)
	}

	cmd := exec.CommandContext(ctx, r.config.AgentBin, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("KUBECONFIG=%s", kubeconfig),
		fmt.Sprintf("NAMESPACE=%s", namespace),
	)

	// Add API key to environment if provided
	if llmCfg.APIKey != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("LLM_API_KEY=%s", llmCfg.APIKey))
	}

	// Pipe prompts to stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start agent: %w", err)
	}

	// Send prompts
	for _, prompt := range task.Script {
		_, _ = io.WriteString(stdin, prompt.Text+"\n")
	}
	stdin.Close()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return stdout.String(), fmt.Errorf("agent exited with error: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// runBuiltinAgent runs the built-in AI client
func (r *Runner) runBuiltinAgent(ctx context.Context, task *Task, llmCfg LLMConfig, kubeconfig, namespace string) (string, error) {
	// Create AI client from LLM config
	cfg := &config.LLMConfig{
		Provider: llmCfg.Provider,
		Model:    llmCfg.Model,
		Endpoint: llmCfg.Endpoint,
		APIKey:   llmCfg.APIKey,
	}

	client, err := ai.NewClient(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create AI client: %w", err)
	}

	// Build context prompt
	contextPrompt := fmt.Sprintf(`You are a Kubernetes AI assistant. You have access to a Kubernetes cluster.
Kubeconfig: %s
Namespace: %s

Complete the following task:
`, kubeconfig, namespace)

	// Collect prompts
	var prompts []string
	for _, p := range task.Script {
		prompts = append(prompts, p.Text)
	}
	fullPrompt := contextPrompt + strings.Join(prompts, "\n")

	// Run with tool support if enabled
	var output strings.Builder
	if llmCfg.EnableToolUse && client.SupportsTools() {
		approvalCallback := func(toolName, args string) bool {
			return llmCfg.AutoApprove // Auto-approve in benchmark mode
		}
		err = client.AskWithTools(ctx, fullPrompt, func(text string) {
			output.WriteString(text)
		}, approvalCallback)
	} else {
		err = client.Ask(ctx, fullPrompt, func(text string) {
			output.WriteString(text)
		})
	}

	if err != nil {
		return output.String(), fmt.Errorf("AI request failed: %w", err)
	}

	return output.String(), nil
}

// runScript executes a shell script with the given environment
func (r *Runner) runScript(ctx context.Context, scriptPath, kubeconfig, namespace, workDir string) (string, error) {
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("script not found: %s", scriptPath)
	}

	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("KUBECONFIG=%s", kubeconfig),
		fmt.Sprintf("NAMESPACE=%s", namespace),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	return output, err
}

// checkExpectations verifies the output against task expectations
func (r *Runner) checkExpectations(task *Task, output string) []Failure {
	var failures []Failure

	for _, exp := range task.Expect {
		if exp.Contains != "" {
			re, err := regexp.Compile(exp.Contains)
			if err != nil {
				failures = append(failures, Failure{
					Type:     "contains",
					Expected: exp.Contains,
					Message:  fmt.Sprintf("invalid regex: %v", err),
				})
				continue
			}
			if !re.MatchString(output) {
				failures = append(failures, Failure{
					Type:     "contains",
					Expected: exp.Contains,
					Actual:   truncateString(output, 500),
					Message:  fmt.Sprintf("output does not match pattern: %s", exp.Contains),
				})
			}
		}

		if exp.NotContains != "" {
			re, err := regexp.Compile(exp.NotContains)
			if err != nil {
				failures = append(failures, Failure{
					Type:     "notContains",
					Expected: exp.NotContains,
					Message:  fmt.Sprintf("invalid regex: %v", err),
				})
				continue
			}
			if re.MatchString(output) {
				failures = append(failures, Failure{
					Type:     "notContains",
					Expected: fmt.Sprintf("NOT %s", exp.NotContains),
					Actual:   truncateString(output, 500),
					Message:  fmt.Sprintf("output should not match pattern: %s", exp.NotContains),
				})
			}
		}
	}

	return failures
}

// setupCluster ensures the benchmark cluster is ready
func (r *Runner) setupCluster(ctx context.Context) error {
	if r.config.ClusterName == "" {
		r.config.ClusterName = fmt.Sprintf("k13d-bench-%s", r.runID)
	}

	// Handle cluster creation policy (k8s-ai-bench compatible)
	policy := r.config.ClusterCreationPolicy
	if policy == "" {
		policy = ClusterCreateIfNotExist // Default
	}

	// Check if cluster exists
	exists, err := r.provider.Exists(ctx, r.config.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	switch policy {
	case ClusterDoNotCreate:
		// Use existing cluster only, do not create
		if !exists {
			return fmt.Errorf("cluster %s does not exist and ClusterDoNotCreate policy is set", r.config.ClusterName)
		}
		r.log("Using existing cluster %s\n", r.config.ClusterName)

	case ClusterAlwaysCreate:
		// Delete existing and create fresh
		if exists {
			r.log("Deleting existing cluster %s (AlwaysCreate policy)...\n", r.config.ClusterName)
			if err := r.provider.Delete(ctx, r.config.ClusterName); err != nil {
				return fmt.Errorf("failed to delete existing cluster: %w", err)
			}
		}
		r.log("Creating cluster %s...\n", r.config.ClusterName)
		if err := r.provider.Create(ctx, r.config.ClusterName); err != nil {
			return fmt.Errorf("failed to create cluster: %w", err)
		}

	case ClusterCreateIfNotExist:
		// Create only if not exists (default)
		if !exists {
			r.log("Creating cluster %s...\n", r.config.ClusterName)
			if err := r.provider.Create(ctx, r.config.ClusterName); err != nil {
				return fmt.Errorf("failed to create cluster: %w", err)
			}
		} else {
			r.log("Using existing cluster %s\n", r.config.ClusterName)
		}

	default:
		return fmt.Errorf("unknown cluster creation policy: %s", policy)
	}

	return nil
}

// createNamespace creates a namespace in the cluster
func (r *Runner) createNamespace(ctx context.Context, kubeconfig, namespace string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig,
		"create", "namespace", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl create namespace failed: %w, output: %s", err, string(output))
	}
	return nil
}

// deleteNamespace deletes a namespace from the cluster
func (r *Runner) deleteNamespace(ctx context.Context, kubeconfig, namespace string) error {
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig,
		"delete", "namespace", namespace, "--ignore-not-found", "--wait=false")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Log warning but don't fail - cleanup errors are not critical
		r.log("Warning: failed to delete namespace %s: %v (output: %s)\n", namespace, err, string(output))
	}
	return err
}

// saveResult saves an individual result to a file
func (r *Runner) saveResult(result *EvalResult) error {
	taskDir := filepath.Join(r.config.OutputDir, result.TaskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.json", result.LLMConfig.ID, timestamp)
	resultPath := filepath.Join(taskDir, filename)

	data, err := marshalJSON(result)
	if err != nil {
		return err
	}

	if err := os.WriteFile(resultPath, data, 0644); err != nil {
		return err
	}

	// Save trace.yaml if enabled (k8s-ai-bench compatible)
	if r.config.SaveTrace && result.Trace != nil {
		tracePath := filepath.Join(taskDir, fmt.Sprintf("%s_%s_trace.yaml", result.LLMConfig.ID, timestamp))
		traceData, err := marshalYAML(result.Trace)
		if err == nil {
			if err := os.WriteFile(tracePath, traceData, 0644); err == nil {
				result.TracePath = tracePath
			}
		}
	}

	// Save log.txt if enabled (k8s-ai-bench compatible)
	if r.config.SaveLog {
		logPath := filepath.Join(taskDir, fmt.Sprintf("%s_%s_log.txt", result.LLMConfig.ID, timestamp))
		logContent := r.buildLogContent(result)
		if err := os.WriteFile(logPath, []byte(logContent), 0644); err == nil {
			result.LogPath = logPath
		}
	}

	return nil
}

// buildLogContent creates a formatted log file content
func (r *Runner) buildLogContent(result *EvalResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Task: %s\n", result.TaskID))
	sb.WriteString(fmt.Sprintf("LLM: %s (%s)\n", result.LLMConfig.ID, result.LLMConfig.Model))
	sb.WriteString(fmt.Sprintf("Result: %s\n", result.Result))
	sb.WriteString(fmt.Sprintf("Duration: %s\n", result.Duration))
	sb.WriteString(fmt.Sprintf("Start: %s\n", result.StartTime.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("End: %s\n", result.EndTime.Format(time.RFC3339)))
	sb.WriteString("\n--- Setup Log ---\n")
	sb.WriteString(result.SetupLog)
	sb.WriteString("\n--- Agent Output ---\n")
	sb.WriteString(result.Output)
	sb.WriteString("\n--- Verify Log ---\n")
	sb.WriteString(result.VerifyLog)
	sb.WriteString("\n--- Cleanup Log ---\n")
	sb.WriteString(result.CleanupLog)

	if result.Error != "" {
		sb.WriteString("\n--- Error ---\n")
		sb.WriteString(result.Error)
	}

	if len(result.Failures) > 0 {
		sb.WriteString("\n--- Failures ---\n")
		for i, f := range result.Failures {
			sb.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, f.Type, f.Message))
		}
	}

	return sb.String()
}

// generateSummary creates a summary of all results
func (r *Runner) generateSummary(startTime, endTime time.Time) *BenchmarkSummary {
	summary := &BenchmarkSummary{
		RunID:      r.runID,
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   endTime.Sub(startTime),
		LLMResults: make(map[string]*LLMSummary),
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, result := range r.results {
		summary.TotalTasks++

		switch result.Result {
		case ResultSuccess:
			summary.SuccessCount++
		case ResultFail:
			summary.FailCount++
		case ResultError, ResultTimeout:
			summary.ErrorCount++
		case ResultSkipped:
			summary.SkippedCount++
		}

		// Difficulty breakdown
		switch result.Difficulty {
		case DifficultyEasy:
			summary.EasyTotal++
			if result.Result == ResultSuccess {
				summary.EasySuccess++
			}
		case DifficultyMedium:
			summary.MediumTotal++
			if result.Result == ResultSuccess {
				summary.MediumSuccess++
			}
		case DifficultyHard:
			summary.HardTotal++
			if result.Result == ResultSuccess {
				summary.HardSuccess++
			}
		}

		// Per-LLM breakdown
		llmID := result.LLMConfig.ID
		if _, ok := summary.LLMResults[llmID]; !ok {
			summary.LLMResults[llmID] = &LLMSummary{
				LLMConfig: result.LLMConfig,
			}
		}
		llmSummary := summary.LLMResults[llmID]
		llmSummary.TotalTasks++
		if result.Result == ResultSuccess {
			llmSummary.SuccessCount++
		} else if result.Result == ResultFail {
			llmSummary.FailCount++
		} else {
			llmSummary.ErrorCount++
		}
	}

	// Calculate pass rates
	if summary.TotalTasks > 0 {
		summary.PassAt1 = float64(summary.SuccessCount) / float64(summary.TotalTasks) * 100
	}

	// Calculate per-LLM pass rates
	for _, llmSummary := range summary.LLMResults {
		if llmSummary.TotalTasks > 0 {
			llmSummary.PassRate = float64(llmSummary.SuccessCount) / float64(llmSummary.TotalTasks) * 100
		}
	}

	return summary
}

// GetResults returns all collected results
func (r *Runner) GetResults() []*EvalResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*EvalResult{}, r.results...)
}

// log prints a message unless quiet mode is enabled
func (r *Runner) log(format string, args ...interface{}) {
	if !r.quiet {
		fmt.Printf(format, args...)
	}
}

// helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
