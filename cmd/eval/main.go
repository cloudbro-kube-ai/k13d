// Package main provides the CLI for running LLM evaluation benchmarks.
// It creates providers directly and evaluates response quality across multiple models.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/providers"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/eval"
	"gopkg.in/yaml.v3"
)

type taskList struct {
	Tasks []eval.Task `yaml:"tasks"`
}

func main() {
	// CLI flags
	tasksFile := flag.String("tasks-file", "pkg/eval/tasks.yaml", "Path to tasks YAML file")
	outputDir := flag.String("output-dir", ".build/eval", "Output directory for reports")
	models := flag.String("models", "", "Multiple models (comma-separated, e.g., 'ollama:gemma3:4b,gemini:gemini-2.0-flash,openai:gpt-4o-mini')")
	llmProvider := flag.String("llm-provider", "", "LLM provider (openai, ollama, gemini)")
	llmModel := flag.String("llm-model", "", "LLM model name")
	llmEndpoint := flag.String("llm-endpoint", "", "LLM API endpoint")
	llmAPIKey := flag.String("llm-api-key", "", "LLM API key (default for all providers)")
	openaiAPIKey := flag.String("openai-api-key", "", "OpenAI API key")
	geminiAPIKey := flag.String("gemini-api-key", "", "Gemini API key")
	solarAPIKey := flag.String("solar-api-key", "", "Solar (Upstage) API key")
	solarEndpoint := flag.String("solar-endpoint", "https://api.upstage.ai/v1", "Solar API endpoint")
	verbose := flag.Bool("verbose", false, "Verbose output")
	flag.Parse()

	// Build per-provider API key map
	apiKeys := map[string]string{
		"openai": *openaiAPIKey,
		"gemini": *geminiAPIKey,
		"solar":  *solarAPIKey,
	}
	// Fill from default --llm-api-key for any missing
	for k, v := range apiKeys {
		if v == "" {
			apiKeys[k] = *llmAPIKey
		}
	}

	// Build per-provider endpoint map
	endpoints := map[string]string{
		"solar": *solarEndpoint,
	}
	if *llmEndpoint != "" {
		endpoints["default"] = *llmEndpoint
	}

	// Parse model configs
	modelConfigs := parseModelConfigs(*models, *llmProvider, *llmModel, *llmEndpoint, *llmAPIKey, apiKeys, endpoints)
	if len(modelConfigs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: specify --models or --llm-provider + --llm-model")
		flag.Usage()
		os.Exit(1)
	}

	// Load tasks
	data, err := os.ReadFile(*tasksFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading tasks file: %v\n", err)
		os.Exit(1)
	}

	var tl taskList
	if err := yaml.Unmarshal(data, &tl); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing tasks file: %v\n", err)
		os.Exit(1)
	}

	if len(tl.Tasks) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no tasks found in tasks file")
		os.Exit(1)
	}

	// Context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted, stopping...")
		cancel()
	}()

	fmt.Println("=== k13d LLM Evaluation Benchmark ===")
	fmt.Printf("Tasks: %d\n", len(tl.Tasks))
	fmt.Printf("Models: %d\n\n", len(modelConfigs))

	var allReports []eval.ModelEvalReport

	for _, mc := range modelConfigs {
		fmt.Printf("--- Evaluating: %s/%s ---\n", mc.providerName, mc.modelName)

		// Create provider
		provider, err := createProvider(mc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error creating provider %s/%s: %v\n\n", mc.providerName, mc.modelName, err)
			continue
		}

		if !provider.IsReady() {
			fmt.Fprintf(os.Stderr, "  Provider %s/%s is not ready (check API key)\n\n", mc.providerName, mc.modelName)
			continue
		}

		// Run evaluation
		var results []eval.EvalResult
		for i, task := range tl.Tasks {
			fmt.Printf("  [%d/%d] %s... ", i+1, len(tl.Tasks), task.ID)

			result := eval.RunEval(ctx, provider, task)
			results = append(results, result)

			if result.Error != "" {
				fmt.Printf("ERROR (%.1fs): %s\n", result.Duration.Seconds(), result.Error)
			} else if result.Success {
				fmt.Printf("PASS %.2f (%.1fs)\n", result.Score, result.Duration.Seconds())
			} else {
				fmt.Printf("FAIL %.2f (%.1fs)\n", result.Score, result.Duration.Seconds())
			}

			if *verbose && len(result.Details) > 0 {
				for _, d := range result.Details {
					status := "+"
					if !d.Passed {
						status = "-"
					}
					fmt.Printf("    %s %s: %s\n", status, d.Criterion, d.Detail)
				}
			}
		}

		report := eval.BuildModelReport(mc.providerName, mc.modelName, results)
		allReports = append(allReports, report)

		fmt.Printf("\n  Result: %d/%d passed (%.1f%%), avg score: %.2f, avg time: %.2fs\n\n",
			report.PassedTasks, report.TotalTasks, report.PassRate,
			report.AvgScore, report.AvgDuration.Seconds())
	}

	// Print comparison summary
	if len(allReports) > 1 {
		fmt.Println("=== Model Comparison ===")
		fmt.Printf("%-20s %-12s %-10s %-10s %-12s\n", "Model", "Provider", "Pass Rate", "Avg Score", "Avg Time")
		fmt.Println(strings.Repeat("-", 65))
		for _, r := range allReports {
			fmt.Printf("%-20s %-12s %-10s %-10s %-12s\n",
				r.Model, r.Provider,
				fmt.Sprintf("%.1f%%", r.PassRate),
				fmt.Sprintf("%.2f", r.AvgScore),
				fmt.Sprintf("%.2fs", r.AvgDuration.Seconds()))
		}
		fmt.Println()
	}

	// Save reports
	if err := eval.SaveComparisonReport(allReports, tl.Tasks, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving reports: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done.")
}

type modelConfig struct {
	providerName string
	modelName    string
	endpoint     string
	apiKey       string
}

func parseModelConfigs(models, provider, model, endpoint, apiKey string, apiKeys, endpointMap map[string]string) []modelConfig {
	var configs []modelConfig

	resolveKey := func(prov string) string {
		if k, ok := apiKeys[prov]; ok && k != "" {
			return k
		}
		return apiKey
	}

	resolveEndpoint := func(prov string) string {
		if e, ok := endpointMap[prov]; ok && e != "" {
			return e
		}
		if e, ok := endpointMap["default"]; ok && e != "" {
			return e
		}
		return endpoint
	}

	if models != "" {
		for _, m := range strings.Split(models, ",") {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			parts := strings.SplitN(m, ":", 2)
			if len(parts) == 2 {
				configs = append(configs, modelConfig{
					providerName: parts[0],
					modelName:    parts[1],
					endpoint:     resolveEndpoint(parts[0]),
					apiKey:       resolveKey(parts[0]),
				})
			} else {
				configs = append(configs, modelConfig{
					providerName: "openai",
					modelName:    parts[0],
					endpoint:     resolveEndpoint("openai"),
					apiKey:       resolveKey("openai"),
				})
			}
		}
	} else if provider != "" && model != "" {
		configs = append(configs, modelConfig{
			providerName: provider,
			modelName:    model,
			endpoint:     resolveEndpoint(provider),
			apiKey:       resolveKey(provider),
		})
	}

	return configs
}

func createProvider(mc modelConfig) (providers.Provider, error) {
	cfg := &providers.ProviderConfig{
		Provider: mc.providerName,
		Model:    mc.modelName,
		Endpoint: mc.endpoint,
		APIKey:   mc.apiKey,
	}

	// Set default endpoints for known providers
	if cfg.Endpoint == "" {
		switch mc.providerName {
		case "ollama":
			cfg.Endpoint = "http://localhost:11434"
		}
	}

	// For Ollama, API key is not required
	if mc.providerName == "ollama" && cfg.APIKey == "" {
		cfg.APIKey = "not-needed"
	}

	factory := providers.GetFactory()
	return factory.Create(cfg)
}
