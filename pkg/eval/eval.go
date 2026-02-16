package eval

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/providers"
)

// Task defines a benchmark evaluation task
type Task struct {
	ID          string   `yaml:"id"`
	Description string   `yaml:"description"`
	Category    string   `yaml:"category"`
	Difficulty  string   `yaml:"difficulty"`
	Prompt      string   `yaml:"prompt"`
	Expect      []Expect `yaml:"expect"`
}

// Expect defines a single evaluation criterion
type Expect struct {
	Contains    string  `yaml:"contains"`
	NotContains string  `yaml:"not_contains"`
	Weight      float64 `yaml:"weight"`
}

// ExpectResult captures the result of a single expectation check
type ExpectResult struct {
	Criterion string  `json:"criterion"`
	Passed    bool    `json:"passed"`
	Weight    float64 `json:"weight"`
	Detail    string  `json:"detail"`
}

// EvalResult holds the result of evaluating a single task
type EvalResult struct {
	TaskID     string         `json:"task_id"`
	Category   string         `json:"category"`
	Difficulty string         `json:"difficulty"`
	Success    bool           `json:"success"`
	Score      float64        `json:"score"`
	Output     string         `json:"output"`
	Error      string         `json:"error,omitempty"`
	Duration   time.Duration  `json:"duration"`
	Details    []ExpectResult `json:"details"`
}

// RunEval executes a single task against a provider and returns the result.
func RunEval(ctx context.Context, provider providers.Provider, task Task) EvalResult {
	result := EvalResult{
		TaskID:     task.ID,
		Category:   task.Category,
		Difficulty: task.Difficulty,
	}

	start := time.Now()

	output, err := provider.AskNonStreaming(ctx, task.Prompt)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		result.Success = false
		result.Score = 0
		return result
	}

	result.Output = output
	outputLower := strings.ToLower(output)

	var totalWeight float64
	var earnedWeight float64

	for _, exp := range task.Expect {
		weight := exp.Weight
		if weight <= 0 {
			weight = 1.0
		}
		totalWeight += weight

		er := ExpectResult{Weight: weight}

		if exp.Contains != "" {
			er.Criterion = fmt.Sprintf("contains: %s", exp.Contains)
			re, compErr := regexp.Compile("(?i)" + exp.Contains)
			if compErr != nil {
				er.Passed = false
				er.Detail = fmt.Sprintf("invalid regex: %v", compErr)
			} else if re.MatchString(outputLower) || re.MatchString(output) {
				er.Passed = true
				er.Detail = "matched"
				earnedWeight += weight
			} else {
				er.Passed = false
				er.Detail = "not found in output"
			}
		}

		if exp.NotContains != "" {
			er.Criterion = fmt.Sprintf("not_contains: %s", exp.NotContains)
			re, compErr := regexp.Compile("(?i)" + exp.NotContains)
			if compErr != nil {
				er.Passed = false
				er.Detail = fmt.Sprintf("invalid regex: %v", compErr)
			} else if re.MatchString(outputLower) || re.MatchString(output) {
				er.Passed = false
				er.Detail = "forbidden pattern found"
				// not_contains failure: deduct weight
			} else {
				er.Passed = true
				er.Detail = "correctly absent"
				earnedWeight += weight
			}
		}

		result.Details = append(result.Details, er)
	}

	if totalWeight > 0 {
		result.Score = earnedWeight / totalWeight
	} else {
		result.Score = 1.0
	}

	result.Success = result.Score >= 0.6 // Pass threshold: 60%

	return result
}
