package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ModelEvalReport holds evaluation results for a single model
type ModelEvalReport struct {
	Provider    string        `json:"provider"`
	Model       string        `json:"model"`
	Timestamp   time.Time     `json:"timestamp"`
	TotalTasks  int           `json:"total_tasks"`
	PassedTasks int           `json:"passed_tasks"`
	PassRate    float64       `json:"pass_rate"`
	AvgScore    float64       `json:"avg_score"`
	AvgDuration time.Duration `json:"avg_duration"`
	Results     []EvalResult  `json:"results"`

	ByCategory   map[string]CategoryScore   `json:"by_category"`
	ByDifficulty map[string]DifficultyScore `json:"by_difficulty"`
}

// CategoryScore holds pass rate/score for a category
type CategoryScore struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"pass_rate"`
	AvgScore float64 `json:"avg_score"`
}

// DifficultyScore holds pass rate/score for a difficulty
type DifficultyScore struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	PassRate float64 `json:"pass_rate"`
	AvgScore float64 `json:"avg_score"`
}

// ComparisonReport holds results for multiple models
type ComparisonReport struct {
	Timestamp time.Time         `json:"timestamp"`
	TaskCount int               `json:"task_count"`
	Models    []ModelEvalReport `json:"models"`
}

// BuildModelReport aggregates EvalResults into a ModelEvalReport
func BuildModelReport(providerName, modelName string, results []EvalResult) ModelEvalReport {
	report := ModelEvalReport{
		Provider:     providerName,
		Model:        modelName,
		Timestamp:    time.Now(),
		TotalTasks:   len(results),
		Results:      results,
		ByCategory:   make(map[string]CategoryScore),
		ByDifficulty: make(map[string]DifficultyScore),
	}

	var totalScore float64
	var totalDuration time.Duration

	catScores := make(map[string][]float64)
	catPassed := make(map[string]int)
	catTotal := make(map[string]int)
	diffScores := make(map[string][]float64)
	diffPassed := make(map[string]int)
	diffTotal := make(map[string]int)

	for _, r := range results {
		totalScore += r.Score
		totalDuration += r.Duration

		if r.Success {
			report.PassedTasks++
		}

		cat := r.Category
		if cat == "" {
			cat = "general"
		}
		catTotal[cat]++
		catScores[cat] = append(catScores[cat], r.Score)
		if r.Success {
			catPassed[cat]++
		}

		diff := r.Difficulty
		if diff == "" {
			diff = "medium"
		}
		diffTotal[diff]++
		diffScores[diff] = append(diffScores[diff], r.Score)
		if r.Success {
			diffPassed[diff]++
		}
	}

	if report.TotalTasks > 0 {
		report.PassRate = float64(report.PassedTasks) / float64(report.TotalTasks) * 100
		report.AvgScore = totalScore / float64(report.TotalTasks)
		report.AvgDuration = totalDuration / time.Duration(report.TotalTasks)
	}

	for cat, total := range catTotal {
		avg := 0.0
		for _, s := range catScores[cat] {
			avg += s
		}
		if total > 0 {
			avg /= float64(total)
		}
		report.ByCategory[cat] = CategoryScore{
			Total:    total,
			Passed:   catPassed[cat],
			PassRate: float64(catPassed[cat]) / float64(total) * 100,
			AvgScore: avg,
		}
	}

	for diff, total := range diffTotal {
		avg := 0.0
		for _, s := range diffScores[diff] {
			avg += s
		}
		if total > 0 {
			avg /= float64(total)
		}
		report.ByDifficulty[diff] = DifficultyScore{
			Total:    total,
			Passed:   diffPassed[diff],
			PassRate: float64(diffPassed[diff]) / float64(total) * 100,
			AvgScore: avg,
		}
	}

	return report
}

// FormatComparisonMarkdown generates a Markdown comparison report
func FormatComparisonMarkdown(reports []ModelEvalReport, tasks []Task) string {
	var sb strings.Builder

	sb.WriteString("# k13d AI Model Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated**: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Tasks**: %d\n", len(tasks)))
	sb.WriteString(fmt.Sprintf("**Models Tested**: %d\n\n", len(reports)))

	// Summary table
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Model | Provider | Pass Rate | Avg Score | Avg Response | Passed/Total |\n")
	sb.WriteString("|-------|----------|-----------|-----------|--------------|-------------|\n")
	for _, r := range reports {
		sb.WriteString(fmt.Sprintf("| %s | %s | %.1f%% | %.2f | %.2fs | %d/%d |\n",
			r.Model, r.Provider, r.PassRate, r.AvgScore,
			r.AvgDuration.Seconds(), r.PassedTasks, r.TotalTasks))
	}

	// Category breakdown
	categories := collectCategories(reports)
	if len(categories) > 0 {
		sb.WriteString("\n## Results by Category\n\n")
		header := "| Model |"
		divider := "|-------|"
		for _, cat := range categories {
			header += fmt.Sprintf(" %s |", cat)
			divider += "---------|"
		}
		sb.WriteString(header + "\n")
		sb.WriteString(divider + "\n")
		for _, r := range reports {
			row := fmt.Sprintf("| %s |", r.Model)
			for _, cat := range categories {
				if cs, ok := r.ByCategory[cat]; ok {
					row += fmt.Sprintf(" %.0f%% (%.2f) |", cs.PassRate, cs.AvgScore)
				} else {
					row += " - |"
				}
			}
			sb.WriteString(row + "\n")
		}
	}

	// Difficulty breakdown
	sb.WriteString("\n## Results by Difficulty\n\n")
	sb.WriteString("| Model | Easy | Medium | Hard |\n")
	sb.WriteString("|-------|------|--------|------|\n")
	for _, r := range reports {
		easy := r.ByDifficulty["easy"]
		med := r.ByDifficulty["medium"]
		hard := r.ByDifficulty["hard"]
		sb.WriteString(fmt.Sprintf("| %s | %.0f%% (%.2f) | %.0f%% (%.2f) | %.0f%% (%.2f) |\n",
			r.Model, easy.PassRate, easy.AvgScore, med.PassRate, med.AvgScore, hard.PassRate, hard.AvgScore))
	}

	// Detailed per-task comparison
	sb.WriteString("\n## Per-Task Results\n\n")
	header := "| Task | Difficulty |"
	divider := "|------|------------|"
	for _, r := range reports {
		header += fmt.Sprintf(" %s |", r.Model)
		divider += "------|"
	}
	sb.WriteString(header + "\n")
	sb.WriteString(divider + "\n")

	for _, task := range tasks {
		row := fmt.Sprintf("| %s | %s |", task.ID, task.Difficulty)
		for _, r := range reports {
			found := false
			for _, res := range r.Results {
				if res.TaskID == task.ID {
					if res.Success {
						row += fmt.Sprintf(" %.2f (%.1fs) |", res.Score, res.Duration.Seconds())
					} else {
						row += fmt.Sprintf(" %.2f* (%.1fs) |", res.Score, res.Duration.Seconds())
					}
					found = true
					break
				}
			}
			if !found {
				row += " - |"
			}
		}
		sb.WriteString(row + "\n")
	}
	sb.WriteString("\n*starred = failed (score < 0.6)*\n")

	// Recommendations
	sb.WriteString("\n## Recommendations\n\n")
	if len(reports) > 0 {
		// Find best overall
		best := reports[0]
		for _, r := range reports[1:] {
			if r.AvgScore > best.AvgScore {
				best = r
			}
		}
		sb.WriteString(fmt.Sprintf("- **Best Overall**: %s (%s) — %.2f avg score, %.1f%% pass rate\n",
			best.Model, best.Provider, best.AvgScore, best.PassRate))

		// Find fastest
		fastest := reports[0]
		for _, r := range reports[1:] {
			if r.AvgDuration < fastest.AvgDuration {
				fastest = r
			}
		}
		sb.WriteString(fmt.Sprintf("- **Fastest**: %s (%s) — %.2fs avg response\n",
			fastest.Model, fastest.Provider, fastest.AvgDuration.Seconds()))
	}

	return sb.String()
}

func collectCategories(reports []ModelEvalReport) []string {
	catSet := make(map[string]bool)
	for _, r := range reports {
		for cat := range r.ByCategory {
			catSet[cat] = true
		}
	}
	var cats []string
	for cat := range catSet {
		cats = append(cats, cat)
	}
	sort.Strings(cats)
	return cats
}

// SaveComparisonReport saves the comparison report to disk
func SaveComparisonReport(reports []ModelEvalReport, tasks []Task, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	// Save JSON
	comparison := ComparisonReport{
		Timestamp: time.Now(),
		TaskCount: len(tasks),
		Models:    reports,
	}
	jsonData, err := json.MarshalIndent(comparison, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	jsonPath := filepath.Join(outputDir, "eval-report.json")
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}

	// Save Markdown
	md := FormatComparisonMarkdown(reports, tasks)
	mdPath := filepath.Join(outputDir, "eval-report.md")
	if err := os.WriteFile(mdPath, []byte(md), 0644); err != nil {
		return fmt.Errorf("writing Markdown: %w", err)
	}

	fmt.Printf("Reports saved:\n  - %s\n  - %s\n", jsonPath, mdPath)
	return nil
}
