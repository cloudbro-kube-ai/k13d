package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/eval"
	"gopkg.in/yaml.v3"
)

type TaskList struct {
	Tasks []eval.Task `yaml:"tasks"`
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	aiClient, err := ai.NewClient(&cfg.LLM)
	if err != nil {
		log.Fatalf("Failed to create AI client: %v", err)
	}

	data, err := os.ReadFile("pkg/eval/tasks.yaml")
	if err != nil {
		log.Fatalf("Failed to read tasks file: %v", err)
	}

	var taskList TaskList
	if err := yaml.Unmarshal(data, &taskList); err != nil {
		log.Fatalf("Failed to unmarshal tasks: %v", err)
	}

	fmt.Println("Starting LLM Evaluation...")
	fmt.Println("==========================")

	for _, task := range taskList.Tasks {
		fmt.Printf("Running Task: %s (%s)\n", task.ID, task.Description)
		result := eval.RunEval(context.Background(), aiClient, task)

		if result.Success {
			fmt.Printf("\033[32m[PASS]\033[0m %s\n", task.ID)
		} else {
			fmt.Printf("\033[31m[FAIL]\033[0m %s\n", task.ID)
			if result.Error != "" {
				fmt.Printf("  Error: %s\n", result.Error)
			}
		}
	}
}
