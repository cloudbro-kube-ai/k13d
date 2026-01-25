package eval

import (
	"context"
	"fmt"
	"regexp"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai"
)

type Task struct {
	ID          string   `yaml:"id"`
	Description string   `yaml:"description"`
	Prompt      string   `yaml:"prompt"`
	Expect      []Expect `yaml:"expect"`
}

type Expect struct {
	Contains string `yaml:"contains"`
}

type EvalResult struct {
	TaskID  string
	Success bool
	Output  string
	Error   string
}

func RunEval(ctx context.Context, aiClient *ai.Client, task Task) EvalResult {
	result := EvalResult{TaskID: task.ID}

	var output string
	err := aiClient.Ask(ctx, task.Prompt, func(text string) {
		output += text
	})

	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return result
	}

	result.Output = output
	result.Success = true

	for _, exp := range task.Expect {
		re, err := regexp.Compile(exp.Contains)
		if err != nil {
			result.Error = fmt.Sprintf("invalid regex: %v", err)
			result.Success = false
			return result
		}
		if !re.MatchString(output) {
			result.Success = false
			break
		}
	}

	return result
}
