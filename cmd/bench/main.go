// Package main provides the CLI for running AI benchmarks
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/bench"
	"github.com/cloudbro-kube-ai/k13d/pkg/eval"
)

const (
	defaultTaskDir     = "benchmarks/tasks"
	defaultOutputDir   = ".build/bench"
	defaultTimeout     = "10m"
	defaultParallelism = 1
)

func main() {
	// Define subcommands
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	dryrunCmd := flag.NewFlagSet("dryrun", flag.ExitOnError)
	analyzeCmd := flag.NewFlagSet("analyze", flag.ExitOnError)
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)

	// Run subcommand flags
	runTaskDir := runCmd.String("task-dir", defaultTaskDir, "Directory containing benchmark tasks")
	runTaskPattern := runCmd.String("task-pattern", "", "Regex pattern to filter tasks")
	runDifficulty := runCmd.String("difficulty", "", "Filter by difficulty (easy, medium, hard)")
	runCategories := runCmd.String("categories", "", "Filter by categories (comma-separated)")
	runTags := runCmd.String("tags", "", "Filter by tags (comma-separated)")
	runParallelism := runCmd.Int("parallelism", defaultParallelism, "Number of parallel workers")
	runTimeout := runCmd.String("timeout", defaultTimeout, "Default task timeout")
	runRetries := runCmd.Int("retries", 0, "Number of retries per task")
	runOutputDir := runCmd.String("output-dir", defaultOutputDir, "Directory for results")
	runOutputFormat := runCmd.String("output-format", "markdown", "Output format (json, jsonl, yaml, markdown)")
	runClusterProvider := runCmd.String("cluster-provider", "existing", "Cluster provider (kind, vcluster, existing)")
	runKubeconfig := runCmd.String("kubeconfig", "", "Path to kubeconfig file")
	runClusterName := runCmd.String("cluster-name", "", "Cluster name (for kind/vcluster)")
	runClusterPolicy := runCmd.String("cluster-creation-policy", "create_if_not", "Cluster creation policy (always, create_if_not, do_not_create)")
	runHostKubeconfig := runCmd.String("host-kubeconfig", "", "Host kubeconfig for vCluster")
	runAgentBin := runCmd.String("agent-bin", "", "Path to external AI agent binary")
	runAgentArgs := runCmd.String("agent-args", "", "Additional agent arguments (comma-separated)")
	runEnableToolUseShim := runCmd.Bool("enable-tool-use-shim", false, "Enable tool use shim for external agent")
	runAgentMaxTurns := runCmd.Int("max-turns", 0, "Max turns for agent (0 = unlimited)")
	runAgentMaxTokens := runCmd.Int("max-tokens", 0, "Max tokens for agent (0 = default)")
	// LLM configuration
	runModels := runCmd.String("models", "", "Multiple LLM models (comma-separated, e.g., 'openai:gpt-4,anthropic:claude-3')")
	runLLMProvider := runCmd.String("llm-provider", "openai", "LLM provider (openai, anthropic, ollama)")
	runLLMModel := runCmd.String("llm-model", "gpt-4", "LLM model name")
	runLLMEndpoint := runCmd.String("llm-endpoint", "", "LLM API endpoint (optional)")
	runLLMAPIKey := runCmd.String("llm-api-key", "", "LLM API key (optional, uses env)")
	runEnableTools := runCmd.Bool("enable-tools", true, "Enable tool/function calling")
	runAutoApprove := runCmd.Bool("auto-approve", true, "Auto-approve tool executions")
	// Output options
	runQuiet := runCmd.Bool("quiet", false, "Suppress progress output")
	runSaveTrace := runCmd.Bool("save-trace", false, "Save trace.yaml per task")
	runSaveLog := runCmd.Bool("save-log", false, "Save log.txt per task")

	// Analyze subcommand flags
	analyzeInputDir := analyzeCmd.String("input-dir", defaultOutputDir, "Directory containing results")
	analyzeOutputFormat := analyzeCmd.String("output-format", "markdown", "Output format (json, jsonl, yaml, markdown)")
	analyzeOutputFile := analyzeCmd.String("output", "", "Output file (stdout if empty)")
	analyzeShowFailures := analyzeCmd.Bool("show-failures", false, "Show only failed results")

	// Dryrun subcommand flags
	dryrunTaskDir := dryrunCmd.String("task-dir", defaultTaskDir, "Directory containing benchmark tasks")
	dryrunTaskPattern := dryrunCmd.String("task-pattern", "", "Regex pattern to filter tasks")
	dryrunOutputDir := dryrunCmd.String("output-dir", defaultOutputDir, "Directory for results")
	dryrunParallelism := dryrunCmd.Int("parallelism", defaultParallelism, "Number of parallel workers")
	dryrunTimeout := dryrunCmd.String("timeout", "5m", "Timeout per task")
	dryrunMode := dryrunCmd.String("mode", "tool-validation", "Dry-run mode (tool-validation, mock-responses, command-analysis)")
	dryrunVerbose := dryrunCmd.Bool("verbose", false, "Verbose output")
	// LLM configuration for dryrun
	dryrunModels := dryrunCmd.String("models", "", "Multiple LLM models (comma-separated, e.g., 'openai:gpt-4,anthropic:claude-3')")
	dryrunLLMProvider := dryrunCmd.String("llm-provider", "openai", "LLM provider (openai, anthropic, ollama)")
	dryrunLLMModel := dryrunCmd.String("llm-model", "gpt-4", "LLM model name")
	dryrunLLMEndpoint := dryrunCmd.String("llm-endpoint", "", "LLM API endpoint (optional)")
	dryrunLLMAPIKey := dryrunCmd.String("llm-api-key", "", "LLM API key (optional, uses env)")
	dryrunEnableTools := dryrunCmd.Bool("enable-tools", true, "Enable tool/function calling")
	dryrunAutoApprove := dryrunCmd.Bool("auto-approve", true, "Auto-approve tool executions")

	// List subcommand flags
	listTaskDir := listCmd.String("task-dir", defaultTaskDir, "Directory containing benchmark tasks")
	listDifficulty := listCmd.String("difficulty", "", "Filter by difficulty")
	listCategories := listCmd.String("categories", "", "Filter by categories")
	listTags := listCmd.String("tags", "", "Filter by tags")

	// Parse arguments
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd.Parse(os.Args[2:])
		if err := executeRun(runCmd, runConfig{
			taskDir:           *runTaskDir,
			taskPattern:       *runTaskPattern,
			difficulty:        *runDifficulty,
			categories:        *runCategories,
			tags:              *runTags,
			parallelism:       *runParallelism,
			timeout:           *runTimeout,
			retries:           *runRetries,
			outputDir:         *runOutputDir,
			outputFormat:      *runOutputFormat,
			clusterProvider:   *runClusterProvider,
			kubeconfig:        *runKubeconfig,
			clusterName:       *runClusterName,
			clusterPolicy:     *runClusterPolicy,
			hostKubeconfig:    *runHostKubeconfig,
			agentBin:          *runAgentBin,
			agentArgs:         *runAgentArgs,
			enableToolUseShim: *runEnableToolUseShim,
			agentMaxTurns:     *runAgentMaxTurns,
			agentMaxTokens:    *runAgentMaxTokens,
			models:            *runModels,
			llmProvider:       *runLLMProvider,
			llmModel:          *runLLMModel,
			llmEndpoint:       *runLLMEndpoint,
			llmAPIKey:         *runLLMAPIKey,
			enableTools:       *runEnableTools,
			autoApprove:       *runAutoApprove,
			quiet:             *runQuiet,
			saveTrace:         *runSaveTrace,
			saveLog:           *runSaveLog,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "dryrun", "dry-run":
		dryrunCmd.Parse(os.Args[2:])
		if err := executeDryRun(dryrunConfig{
			taskDir:     *dryrunTaskDir,
			taskPattern: *dryrunTaskPattern,
			outputDir:   *dryrunOutputDir,
			parallelism: *dryrunParallelism,
			timeout:     *dryrunTimeout,
			mode:        *dryrunMode,
			verbose:     *dryrunVerbose,
			models:      *dryrunModels,
			llmProvider: *dryrunLLMProvider,
			llmModel:    *dryrunLLMModel,
			llmEndpoint: *dryrunLLMEndpoint,
			llmAPIKey:   *dryrunLLMAPIKey,
			enableTools: *dryrunEnableTools,
			autoApprove: *dryrunAutoApprove,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "analyze":
		analyzeCmd.Parse(os.Args[2:])
		if err := executeAnalyze(*analyzeInputDir, *analyzeOutputFormat, *analyzeOutputFile, *analyzeShowFailures); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "list":
		listCmd.Parse(os.Args[2:])
		if err := executeList(*listTaskDir, *listDifficulty, *listCategories, *listTags); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

type runConfig struct {
	taskDir, taskPattern, difficulty, categories, tags string
	parallelism, retries                               int
	timeout, outputDir, outputFormat                   string
	clusterProvider, kubeconfig, clusterName           string
	clusterPolicy, hostKubeconfig                      string
	agentBin, agentArgs                                string
	enableToolUseShim                                  bool
	agentMaxTurns, agentMaxTokens                      int
	models                                             string
	llmProvider, llmModel, llmEndpoint, llmAPIKey      string
	enableTools, autoApprove                           bool
	quiet, saveTrace, saveLog                          bool
}

type dryrunConfig struct {
	taskDir, taskPattern, outputDir string
	parallelism                     int
	timeout, mode                   string
	verbose                         bool
	models                          string
	llmProvider, llmModel           string
	llmEndpoint, llmAPIKey          string
	enableTools, autoApprove        bool
}

func executeRun(cmd *flag.FlagSet, cfg runConfig) error {
	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, cleaning up...")
		cancel()
	}()

	// Build LLM configs (support multiple models)
	var llmConfigs []bench.LLMConfig
	if cfg.models != "" {
		// Parse --models flag: "openai:gpt-4,anthropic:claude-3"
		llmConfigs = parseModelsFlag(cfg.models, cfg.llmEndpoint, cfg.llmAPIKey, cfg.enableTools, cfg.autoApprove)
	} else {
		// Use single LLM config from individual flags
		llmConfigs = []bench.LLMConfig{{
			ID:            fmt.Sprintf("%s-%s", cfg.llmProvider, cfg.llmModel),
			Provider:      cfg.llmProvider,
			Model:         cfg.llmModel,
			Endpoint:      cfg.llmEndpoint,
			APIKey:        cfg.llmAPIKey,
			EnableToolUse: cfg.enableTools,
			AutoApprove:   cfg.autoApprove,
		}}
	}

	// Build run config
	runCfg := &bench.RunConfig{
		TaskDir:               cfg.taskDir,
		TaskPattern:           cfg.taskPattern,
		Difficulty:            cfg.difficulty,
		Categories:            splitAndTrim(cfg.categories),
		Tags:                  splitAndTrim(cfg.tags),
		LLMConfigs:            llmConfigs,
		Parallelism:           cfg.parallelism,
		DefaultTimeout:        cfg.timeout,
		Retries:               cfg.retries,
		ClusterProvider:       cfg.clusterProvider,
		Kubeconfig:            cfg.kubeconfig,
		ClusterName:           cfg.clusterName,
		ClusterCreationPolicy: bench.ClusterPolicy(cfg.clusterPolicy),
		HostKubeconfig:        cfg.hostKubeconfig,
		OutputDir:             cfg.outputDir,
		OutputFormat:          cfg.outputFormat,
		SaveTrace:             cfg.saveTrace,
		SaveLog:               cfg.saveLog,
		AgentBin:              cfg.agentBin,
		AgentArgs:             splitAndTrim(cfg.agentArgs),
		EnableToolUseShim:     cfg.enableToolUseShim,
		AgentMaxTurns:         cfg.agentMaxTurns,
		AgentMaxTokens:        cfg.agentMaxTokens,
		Quiet:                 cfg.quiet,
	}

	// Create and run benchmark
	runner, err := bench.NewRunner(runCfg)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	summary, err := runner.Run(ctx)
	if err != nil {
		return fmt.Errorf("benchmark failed: %w", err)
	}

	// Print summary
	bench.PrintSummary(summary)

	// Generate report
	analyzer := bench.NewAnalyzer(cfg.outputDir, bench.OutputFormat(cfg.outputFormat))
	results := runner.GetResults()
	reportPath := fmt.Sprintf("%s/report.%s", cfg.outputDir, getReportExtension(cfg.outputFormat))
	if err := analyzer.WriteReport(summary, results, reportPath); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}
	fmt.Printf("\nReport written to: %s\n", reportPath)

	return nil
}

func executeDryRun(cfg dryrunConfig) error {
	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, cleaning up...")
		cancel()
	}()

	// Parse timeout
	timeout, err := time.ParseDuration(cfg.timeout)
	if err != nil {
		timeout = 5 * time.Minute
	}

	// Build LLM configs
	var llmConfigs []eval.LLMRunConfig
	if cfg.models != "" {
		for _, m := range splitAndTrim(cfg.models) {
			parts := strings.SplitN(m, ":", 2)
			var provider, model string
			if len(parts) == 2 {
				provider = parts[0]
				model = parts[1]
			} else {
				provider = "openai"
				model = parts[0]
			}
			llmConfigs = append(llmConfigs, eval.LLMRunConfig{
				ID:            fmt.Sprintf("%s-%s", provider, model),
				Provider:      provider,
				Model:         model,
				Endpoint:      cfg.llmEndpoint,
				APIKey:        cfg.llmAPIKey,
				EnableToolUse: cfg.enableTools,
				AutoApprove:   cfg.autoApprove,
			})
		}
	} else {
		llmConfigs = []eval.LLMRunConfig{{
			ID:            fmt.Sprintf("%s-%s", cfg.llmProvider, cfg.llmModel),
			Provider:      cfg.llmProvider,
			Model:         cfg.llmModel,
			Endpoint:      cfg.llmEndpoint,
			APIKey:        cfg.llmAPIKey,
			EnableToolUse: cfg.enableTools,
			AutoApprove:   cfg.autoApprove,
		}}
	}

	// Parse dry-run mode
	var mode eval.DryRunMode
	switch cfg.mode {
	case "tool-validation":
		mode = eval.DryRunToolValidation
	case "mock-responses":
		mode = eval.DryRunMockResponses
	case "command-analysis":
		mode = eval.DryRunCommandAnalysis
	default:
		mode = eval.DryRunToolValidation
	}

	// Create runner config
	runnerConfig := eval.DryRunRunnerConfig{
		TasksDir:       cfg.taskDir,
		OutputDir:      cfg.outputDir,
		TaskPattern:    cfg.taskPattern,
		Concurrency:    cfg.parallelism,
		LLMConfigs:     llmConfigs,
		Verbose:        cfg.verbose,
		Mode:           mode,
		TimeoutPerTask: timeout,
	}

	// Create runner
	runner := eval.NewDryRunRunner(runnerConfig)

	// Load tasks
	if err := runner.LoadTasks(); err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	tasks := runner.GetTasks()
	if len(tasks) == 0 {
		return fmt.Errorf("no tasks found in %s", cfg.taskDir)
	}

	fmt.Printf("Loaded %d tasks for dry-run evaluation\n", len(tasks))
	fmt.Printf("Mode: %s\n", cfg.mode)
	fmt.Printf("Models: %d configured\n\n", len(llmConfigs))

	// Run benchmark
	report, err := runner.Run(ctx)
	if err != nil {
		return fmt.Errorf("dry-run failed: %w", err)
	}

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("DRY-RUN BENCHMARK SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total Tasks:    %d\n", report.Summary.TotalTasks)
	fmt.Printf("Passed:         %d (%.1f%%)\n", report.Summary.PassedTasks, report.Summary.PassRate)
	fmt.Printf("Failed:         %d\n", report.Summary.FailedTasks)
	fmt.Printf("Average Score:  %.2f\n", report.Summary.AverageScore)
	fmt.Printf("Total Duration: %s\n", report.TotalDuration)
	fmt.Println(strings.Repeat("=", 60))

	// Save report
	if err := report.SaveReport(cfg.outputDir); err != nil {
		return fmt.Errorf("failed to save report: %w", err)
	}

	return nil
}

func executeAnalyze(inputDir, outputFormat, outputFile string, showFailures bool) error {
	analyzer := bench.NewAnalyzer(inputDir, bench.OutputFormat(outputFormat))

	results, err := analyzer.LoadResults()
	if err != nil {
		return fmt.Errorf("failed to load results: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("no results found in %s", inputDir)
	}

	// Filter to only failures if requested
	if showFailures {
		results = bench.GetFailedResults(results)
		if len(results) == 0 {
			fmt.Println("No failures found!")
			return nil
		}
		fmt.Printf("Showing %d failed results:\n", len(results))
	}

	summary := analyzer.Analyze(results)
	bench.PrintSummary(summary)

	if err := analyzer.WriteReport(summary, results, outputFile); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	return nil
}

func executeList(taskDir, difficulty, categories, tags string) error {
	loader := bench.NewLoader(taskDir)
	tasks, err := loader.LoadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	tasks, err = loader.FilterTasks(tasks, bench.FilterOptions{
		Difficulty: difficulty,
		Categories: splitAndTrim(categories),
		Tags:       splitAndTrim(tags),
	})
	if err != nil {
		return fmt.Errorf("failed to filter tasks: %w", err)
	}

	fmt.Printf("Found %d tasks:\n\n", len(tasks))
	fmt.Printf("%-25s %-10s %-15s %s\n", "ID", "DIFFICULTY", "CATEGORY", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 80))

	for _, task := range tasks {
		desc := task.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}
		fmt.Printf("%-25s %-10s %-15s %s\n", task.ID, task.Difficulty, task.Category, desc)
	}

	return nil
}

func printUsage() {
	fmt.Println(`k13d-bench - AI Benchmark Tool for Kubernetes

USAGE:
    k13d-bench <command> [options]

COMMANDS:
    run       Run benchmark evaluations (requires cluster)
    dryrun    Run dry-run benchmark (no cluster required)
    analyze   Analyze and report benchmark results
    list      List available benchmark tasks
    help      Show this help message

EXAMPLES:
    # Run all benchmarks with GPT-4
    k13d-bench run --llm-provider openai --llm-model gpt-4

    # Run with multiple LLMs for comparison
    k13d-bench run --models "openai:gpt-4,anthropic:claude-3-sonnet"

    # Run only easy tasks
    k13d-bench run --difficulty easy

    # Run with a specific task pattern
    k13d-bench run --task-pattern "fix-.*"

    # Run using Kind cluster with fresh creation
    k13d-bench run --cluster-provider kind --cluster-name bench-cluster --cluster-creation-policy always

    # Run with trace and log saving
    k13d-bench run --save-trace --save-log --output-dir ./results

    # Run in quiet mode
    k13d-bench run --quiet --output-format json

    # DRY-RUN: Validate tool calls without cluster (no cluster required!)
    k13d-bench dryrun --verbose

    # DRY-RUN: Compare multiple LLMs without cluster
    k13d-bench dryrun --models "openai:gpt-4,anthropic:claude-3-sonnet"

    # DRY-RUN: Run specific tasks
    k13d-bench dryrun --task-pattern "create-.*" --verbose

    # DRY-RUN: Mock response mode for testing
    k13d-bench dryrun --mode mock-responses --verbose

    # Analyze previous results
    k13d-bench analyze --input-dir .build/bench --output-format markdown

    # Show only failures from previous run
    k13d-bench analyze --input-dir .build/bench --show-failures

    # List available tasks
    k13d-bench list --task-dir benchmarks/tasks

Run 'k13d-bench <command> --help' for more information on a command.`)
}

// parseModelsFlag parses the --models flag format: "provider:model,provider:model"
func parseModelsFlag(models, defaultEndpoint, defaultAPIKey string, enableTools, autoApprove bool) []bench.LLMConfig {
	var configs []bench.LLMConfig
	for _, m := range splitAndTrim(models) {
		parts := strings.SplitN(m, ":", 2)
		var provider, model string
		if len(parts) == 2 {
			provider = parts[0]
			model = parts[1]
		} else {
			// Assume openai if no provider specified
			provider = "openai"
			model = parts[0]
		}
		configs = append(configs, bench.LLMConfig{
			ID:            fmt.Sprintf("%s-%s", provider, model),
			Provider:      provider,
			Model:         model,
			Endpoint:      defaultEndpoint,
			APIKey:        defaultAPIKey,
			EnableToolUse: enableTools,
			AutoApprove:   autoApprove,
		})
	}
	return configs
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func getReportExtension(format string) string {
	switch format {
	case "json":
		return "json"
	case "jsonl":
		return "jsonl"
	case "yaml":
		return "yaml"
	default:
		return "md"
	}
}
