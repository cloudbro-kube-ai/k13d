package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// OutputFormat represents the format for benchmark output
type OutputFormat string

const (
	OutputJSON     OutputFormat = "json"
	OutputJSONL    OutputFormat = "jsonl"
	OutputYAML     OutputFormat = "yaml"
	OutputMarkdown OutputFormat = "markdown"
)

// Analyzer processes and reports benchmark results
type Analyzer struct {
	outputDir    string
	outputFormat OutputFormat
}

// NewAnalyzer creates a new result analyzer
func NewAnalyzer(outputDir string, format OutputFormat) *Analyzer {
	if format == "" {
		format = OutputMarkdown
	}
	return &Analyzer{
		outputDir:    outputDir,
		outputFormat: format,
	}
}

// LoadResults loads all results from the output directory
func (a *Analyzer) LoadResults() ([]*EvalResult, error) {
	var results []*EvalResult

	err := filepath.Walk(a.outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var result EvalResult
		if err := json.Unmarshal(data, &result); err != nil {
			return nil // Skip invalid files
		}

		results = append(results, &result)
		return nil
	})

	return results, err
}

// Analyze generates a summary from results
func (a *Analyzer) Analyze(results []*EvalResult) *BenchmarkSummary {
	if len(results) == 0 {
		return &BenchmarkSummary{
			LLMResults: make(map[string]*LLMSummary),
		}
	}

	// Find time range
	var startTime, endTime time.Time
	for _, r := range results {
		if startTime.IsZero() || r.StartTime.Before(startTime) {
			startTime = r.StartTime
		}
		if endTime.IsZero() || r.EndTime.After(endTime) {
			endTime = r.EndTime
		}
	}

	summary := &BenchmarkSummary{
		RunID:      results[0].RunID,
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   endTime.Sub(startTime),
		LLMResults: make(map[string]*LLMSummary),
	}

	// Aggregate results
	for _, result := range results {
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
		llmSummary.AvgDuration += result.Duration
		if result.Result == ResultSuccess {
			llmSummary.SuccessCount++
		} else if result.Result == ResultFail {
			llmSummary.FailCount++
		} else {
			llmSummary.ErrorCount++
		}
	}

	// Calculate rates
	if summary.TotalTasks > 0 {
		summary.PassAt1 = float64(summary.SuccessCount) / float64(summary.TotalTasks) * 100
	}

	for _, llmSummary := range summary.LLMResults {
		if llmSummary.TotalTasks > 0 {
			llmSummary.PassRate = float64(llmSummary.SuccessCount) / float64(llmSummary.TotalTasks) * 100
			llmSummary.AvgDuration = llmSummary.AvgDuration / time.Duration(llmSummary.TotalTasks)
		}
	}

	return summary
}

// WriteReport writes the analysis report
func (a *Analyzer) WriteReport(summary *BenchmarkSummary, results []*EvalResult, outputPath string) error {
	var data []byte
	var err error

	switch a.outputFormat {
	case OutputJSON:
		data, err = a.formatJSON(summary, results)
	case OutputJSONL:
		data, err = a.formatJSONL(results)
	case OutputYAML:
		data, err = a.formatYAML(summary, results)
	case OutputMarkdown:
		data, err = a.formatMarkdown(summary, results)
	default:
		return fmt.Errorf("unknown output format: %s", a.outputFormat)
	}

	if err != nil {
		return err
	}

	if outputPath == "" {
		fmt.Println(string(data))
		return nil
	}

	return os.WriteFile(outputPath, data, 0644)
}

func (a *Analyzer) formatJSON(summary *BenchmarkSummary, results []*EvalResult) ([]byte, error) {
	report := struct {
		Summary *BenchmarkSummary `json:"summary"`
		Results []*EvalResult     `json:"results"`
	}{
		Summary: summary,
		Results: results,
	}
	return json.MarshalIndent(report, "", "  ")
}

func (a *Analyzer) formatJSONL(results []*EvalResult) ([]byte, error) {
	var lines []string
	for _, r := range results {
		line, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		lines = append(lines, string(line))
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func (a *Analyzer) formatYAML(summary *BenchmarkSummary, results []*EvalResult) ([]byte, error) {
	report := struct {
		Summary *BenchmarkSummary `yaml:"summary"`
		Results []*EvalResult     `yaml:"results"`
	}{
		Summary: summary,
		Results: results,
	}
	return yaml.Marshal(report)
}

func (a *Analyzer) formatMarkdown(summary *BenchmarkSummary, results []*EvalResult) ([]byte, error) {
	var sb strings.Builder

	// Header
	sb.WriteString("# K13D AI Benchmark Results\n\n")
	sb.WriteString(fmt.Sprintf("**Run ID:** %s\n", summary.RunID))
	sb.WriteString(fmt.Sprintf("**Duration:** %s\n", summary.Duration.Round(time.Second)))
	sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", summary.StartTime.Format(time.RFC3339)))

	// Overall Summary
	sb.WriteString("## Overall Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Total Tasks | %d |\n", summary.TotalTasks))
	sb.WriteString(fmt.Sprintf("| Success | %d |\n", summary.SuccessCount))
	sb.WriteString(fmt.Sprintf("| Failed | %d |\n", summary.FailCount))
	sb.WriteString(fmt.Sprintf("| Errors | %d |\n", summary.ErrorCount))
	sb.WriteString(fmt.Sprintf("| Pass@1 | %.1f%% |\n", summary.PassAt1))
	sb.WriteString("\n")

	// Difficulty Breakdown
	sb.WriteString("## Results by Difficulty\n\n")
	sb.WriteString("| Difficulty | Success | Total | Rate |\n")
	sb.WriteString("|------------|---------|-------|------|\n")
	if summary.EasyTotal > 0 {
		rate := float64(summary.EasySuccess) / float64(summary.EasyTotal) * 100
		sb.WriteString(fmt.Sprintf("| Easy | %d | %d | %.1f%% |\n", summary.EasySuccess, summary.EasyTotal, rate))
	}
	if summary.MediumTotal > 0 {
		rate := float64(summary.MediumSuccess) / float64(summary.MediumTotal) * 100
		sb.WriteString(fmt.Sprintf("| Medium | %d | %d | %.1f%% |\n", summary.MediumSuccess, summary.MediumTotal, rate))
	}
	if summary.HardTotal > 0 {
		rate := float64(summary.HardSuccess) / float64(summary.HardTotal) * 100
		sb.WriteString(fmt.Sprintf("| Hard | %d | %d | %.1f%% |\n", summary.HardSuccess, summary.HardTotal, rate))
	}
	sb.WriteString("\n")

	// Per-LLM Results
	sb.WriteString("## Results by LLM\n\n")
	sb.WriteString("| Model | Success | Failed | Errors | Pass Rate | Avg Duration |\n")
	sb.WriteString("|-------|---------|--------|--------|-----------|-------------|\n")

	// Sort LLMs by ID
	var llmIDs []string
	for id := range summary.LLMResults {
		llmIDs = append(llmIDs, id)
	}
	sort.Strings(llmIDs)

	for _, id := range llmIDs {
		llm := summary.LLMResults[id]
		sb.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %.1f%% | %s |\n",
			llm.LLMConfig.Model,
			llm.SuccessCount,
			llm.FailCount,
			llm.ErrorCount,
			llm.PassRate,
			llm.AvgDuration.Round(time.Second)))
	}
	sb.WriteString("\n")

	// Detailed Results
	sb.WriteString("## Detailed Results\n\n")

	// Group by task
	taskResults := make(map[string][]*EvalResult)
	for _, r := range results {
		taskResults[r.TaskID] = append(taskResults[r.TaskID], r)
	}

	// Sort task IDs
	var taskIDs []string
	for id := range taskResults {
		taskIDs = append(taskIDs, id)
	}
	sort.Strings(taskIDs)

	for _, taskID := range taskIDs {
		taskRes := taskResults[taskID]
		if len(taskRes) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("### %s\n\n", taskID))
		if taskRes[0].TaskName != "" {
			sb.WriteString(fmt.Sprintf("**%s** (%s)\n\n", taskRes[0].TaskName, taskRes[0].Difficulty))
		}

		sb.WriteString("| LLM | Result | Duration | Notes |\n")
		sb.WriteString("|-----|--------|----------|-------|\n")

		for _, r := range taskRes {
			notes := ""
			if r.Error != "" {
				notes = truncateString(r.Error, 50)
			} else if len(r.Failures) > 0 {
				notes = truncateString(r.Failures[0].Message, 50)
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				r.LLMConfig.Model,
				resultEmoji(r.Result),
				r.Duration.Round(time.Millisecond),
				notes))
		}
		sb.WriteString("\n")
	}

	return []byte(sb.String()), nil
}

func resultEmoji(r TaskResult) string {
	switch r {
	case ResultSuccess:
		return "✅"
	case ResultFail:
		return "❌"
	case ResultError:
		return "⚠️"
	case ResultTimeout:
		return "⏱️"
	case ResultSkipped:
		return "⏭️"
	default:
		return "❓"
	}
}

// PrintSummary prints a quick summary to stdout
func PrintSummary(summary *BenchmarkSummary) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("BENCHMARK SUMMARY")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Run ID:     %s\n", summary.RunID)
	fmt.Printf("Duration:   %s\n", summary.Duration.Round(time.Second))
	fmt.Printf("Total:      %d tasks\n", summary.TotalTasks)
	fmt.Printf("Success:    %d (%.1f%%)\n", summary.SuccessCount, summary.PassAt1)
	fmt.Printf("Failed:     %d\n", summary.FailCount)
	fmt.Printf("Errors:     %d\n", summary.ErrorCount)
	fmt.Println(strings.Repeat("=", 50))

	if len(summary.LLMResults) > 1 {
		fmt.Println("\nPer-LLM Results:")
		for id, llm := range summary.LLMResults {
			fmt.Printf("  %s: %.1f%% (%d/%d)\n", id, llm.PassRate, llm.SuccessCount, llm.TotalTasks)
		}
	}
}

// marshalJSON is a helper to marshal JSON with pretty printing
func marshalJSON(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// marshalYAML is a helper to marshal YAML
func marshalYAML(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

// GetFailedResults returns only failed results for --show-failures flag
func GetFailedResults(results []*EvalResult) []*EvalResult {
	var failed []*EvalResult
	for _, r := range results {
		if r.Result != ResultSuccess {
			failed = append(failed, r)
		}
	}
	return failed
}
