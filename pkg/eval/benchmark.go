package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"gopkg.in/yaml.v3"
)

// BenchmarkTask represents a single benchmark test case
type BenchmarkTask struct {
	ID          string        `yaml:"id" json:"id"`
	Category    string        `yaml:"category" json:"category"`
	Difficulty  string        `yaml:"difficulty" json:"difficulty"`
	Description string        `yaml:"description" json:"description"`
	Prompt      string        `yaml:"prompt" json:"prompt"`
	Expect      []Expectation `yaml:"expect" json:"expect"`
}

// Expectation defines what we expect in the response
type Expectation struct {
	Type   string   `yaml:"type" json:"type"`
	Value  string   `yaml:"value,omitempty" json:"value,omitempty"`
	Values []string `yaml:"values,omitempty" json:"values,omitempty"`
}

// BenchmarkConfig holds the benchmark configuration
type BenchmarkConfig struct {
	Version     string          `yaml:"version"`
	Description string          `yaml:"description"`
	Tasks       []BenchmarkTask `yaml:"tasks"`
}

// BenchmarkResult holds the result of a single task
type BenchmarkResult struct {
	TaskID       string          `json:"task_id"`
	Category     string          `json:"category"`
	Difficulty   string          `json:"difficulty"`
	Passed       bool            `json:"passed"`
	Output       string          `json:"output"`
	ResponseTime time.Duration   `json:"response_time_ms"`
	Checks       map[string]bool `json:"checks"`
	Error        string          `json:"error,omitempty"`
}

// ModelBenchmark holds results for a single model
type ModelBenchmark struct {
	ModelName    string             `json:"model_name"`
	Provider     string             `json:"provider"`
	Timestamp    time.Time          `json:"timestamp"`
	TotalTasks   int                `json:"total_tasks"`
	PassedTasks  int                `json:"passed_tasks"`
	PassRate     float64            `json:"pass_rate"`
	AvgRespTime  time.Duration      `json:"avg_response_time_ms"`
	Results      []BenchmarkResult  `json:"results"`
	ByCategory   map[string]float64 `json:"by_category"`
	ByDifficulty map[string]float64 `json:"by_difficulty"`
}

// LoadBenchmarkTasks loads benchmark tasks from YAML file
func LoadBenchmarkTasks(path string) (*BenchmarkConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read benchmark file: %w", err)
	}

	var cfg BenchmarkConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse benchmark file: %w", err)
	}

	return &cfg, nil
}

// RunBenchmark executes all benchmark tasks for a model
func RunBenchmark(ctx context.Context, cfg *config.Config, tasks []BenchmarkTask) (*ModelBenchmark, error) {
	// Create AI client
	aiClient, err := ai.NewClient(&cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	benchmark := &ModelBenchmark{
		ModelName:    cfg.LLM.Model,
		Provider:     cfg.LLM.Provider,
		Timestamp:    time.Now(),
		TotalTasks:   len(tasks),
		Results:      make([]BenchmarkResult, 0, len(tasks)),
		ByCategory:   make(map[string]float64),
		ByDifficulty: make(map[string]float64),
	}

	categoryCount := make(map[string]int)
	categoryPass := make(map[string]int)
	difficultyCount := make(map[string]int)
	difficultyPass := make(map[string]int)

	var totalRespTime time.Duration

	for i, task := range tasks {
		fmt.Printf("  [%d/%d] %s... ", i+1, len(tasks), task.ID)

		result := runSingleTask(ctx, aiClient, task, cfg.Language)
		benchmark.Results = append(benchmark.Results, result)

		if result.Passed {
			benchmark.PassedTasks++
			categoryPass[task.Category]++
			difficultyPass[task.Difficulty]++
			fmt.Printf("✓ (%.1fs)\n", result.ResponseTime.Seconds())
		} else {
			fmt.Printf("✗ (%.1fs)\n", result.ResponseTime.Seconds())
		}

		categoryCount[task.Category]++
		difficultyCount[task.Difficulty]++
		totalRespTime += result.ResponseTime
	}

	// Calculate metrics
	benchmark.PassRate = float64(benchmark.PassedTasks) / float64(benchmark.TotalTasks) * 100
	benchmark.AvgRespTime = totalRespTime / time.Duration(len(tasks))

	for cat, count := range categoryCount {
		benchmark.ByCategory[cat] = float64(categoryPass[cat]) / float64(count) * 100
	}
	for diff, count := range difficultyCount {
		benchmark.ByDifficulty[diff] = float64(difficultyPass[diff]) / float64(count) * 100
	}

	return benchmark, nil
}

// runSingleTask executes a single benchmark task
func runSingleTask(ctx context.Context, client *ai.Client, task BenchmarkTask, lang string) BenchmarkResult {
	result := BenchmarkResult{
		TaskID:     task.ID,
		Category:   task.Category,
		Difficulty: task.Difficulty,
		Checks:     make(map[string]bool),
	}

	// Add language instruction if needed
	prompt := task.Prompt
	if lang == "ko" {
		prompt = "다음 질문에 한국어로 답변해주세요. " + prompt
	}

	start := time.Now()
	var output strings.Builder

	err := client.Ask(ctx, prompt, func(text string) {
		output.WriteString(text)
	})

	result.ResponseTime = time.Since(start)
	result.Output = output.String()

	if err != nil {
		result.Error = err.Error()
		result.Passed = false
		return result
	}

	// Check expectations
	result.Passed = true
	for _, exp := range task.Expect {
		checkName := fmt.Sprintf("%s:%s", exp.Type, exp.Value)
		if len(exp.Values) > 0 {
			checkName = fmt.Sprintf("%s:%v", exp.Type, exp.Values)
		}

		passed := checkExpectation(result.Output, exp)
		result.Checks[checkName] = passed
		if !passed {
			result.Passed = false
		}
	}

	return result
}

// checkExpectation verifies if output meets expectation
func checkExpectation(output string, exp Expectation) bool {
	lowerOutput := strings.ToLower(output)

	switch exp.Type {
	case "contains":
		return strings.Contains(lowerOutput, strings.ToLower(exp.Value))

	case "not_contains":
		return !strings.Contains(output, exp.Value)

	case "contains_any":
		for _, v := range exp.Values {
			if strings.Contains(lowerOutput, strings.ToLower(v)) {
				return true
			}
		}
		return false

	case "not_contains_any":
		for _, v := range exp.Values {
			if strings.Contains(lowerOutput, strings.ToLower(v)) {
				return false
			}
		}
		return true

	case "regex":
		re, err := regexp.Compile(exp.Value)
		if err != nil {
			return false
		}
		return re.MatchString(output)

	case "language":
		if exp.Value == "ko" {
			return containsKorean(output)
		}
		return true

	case "max_sentences":
		// Simple sentence count
		sentences := strings.Count(output, ".") + strings.Count(output, "。")
		// Parse max from value (assuming it's a small number)
		var max int
		fmt.Sscanf(exp.Value, "%d", &max)
		return sentences <= max+1 // Allow some margin

	case "sentiment":
		// Simple sentiment check for warning
		if exp.Value == "warning" {
			warningWords := []string{"위험", "주의", "경고", "조심", "삭제", "복구", "danger", "warning", "careful", "caution"}
			for _, w := range warningWords {
				if strings.Contains(lowerOutput, strings.ToLower(w)) {
					return true
				}
			}
			return false
		}
		return true

	default:
		return true
	}
}

// containsKorean checks if text contains Korean characters
func containsKorean(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Hangul, r) {
			return true
		}
	}
	return false
}

// SaveResults saves benchmark results to JSON file
func SaveResults(results []*ModelBenchmark, path string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GenerateMarkdownReport creates a markdown report from results
func GenerateMarkdownReport(results []*ModelBenchmark) string {
	var sb strings.Builder

	sb.WriteString("# k13d AI Model Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Summary table
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Model | Provider | Pass Rate | Avg Response | Tasks |\n")
	sb.WriteString("|-------|----------|-----------|--------------|-------|\n")

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("| %s | %s | %.1f%% | %.2fs | %d/%d |\n",
			r.ModelName, r.Provider, r.PassRate,
			r.AvgRespTime.Seconds(), r.PassedTasks, r.TotalTasks))
	}

	// Category breakdown
	sb.WriteString("\n## By Category\n\n")
	sb.WriteString("| Model | Instruction | Kubernetes | Tool Use | Korean | Safety |\n")
	sb.WriteString("|-------|-------------|------------|----------|--------|--------|\n")

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("| %s | %.0f%% | %.0f%% | %.0f%% | %.0f%% | %.0f%% |\n",
			r.ModelName,
			r.ByCategory["instruction_following"],
			r.ByCategory["kubernetes"],
			r.ByCategory["tool_use"],
			r.ByCategory["korean"],
			r.ByCategory["safety"]))
	}

	// Difficulty breakdown
	sb.WriteString("\n## By Difficulty\n\n")
	sb.WriteString("| Model | Easy | Medium | Hard |\n")
	sb.WriteString("|-------|------|--------|------|\n")

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("| %s | %.0f%% | %.0f%% | %.0f%% |\n",
			r.ModelName,
			r.ByDifficulty["easy"],
			r.ByDifficulty["medium"],
			r.ByDifficulty["hard"]))
	}

	// Detailed results per model
	sb.WriteString("\n## Detailed Results\n\n")

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("### %s (%s)\n\n", r.ModelName, r.Provider))
		sb.WriteString("| Task | Category | Difficulty | Result | Time |\n")
		sb.WriteString("|------|----------|------------|--------|------|\n")

		for _, task := range r.Results {
			status := "✓"
			if !task.Passed {
				status = "✗"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %.2fs |\n",
				task.TaskID, task.Category, task.Difficulty, status, task.ResponseTime.Seconds()))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
