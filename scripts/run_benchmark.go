//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/eval"
)

// 테스트할 모델 목록 (10B~20B 이내)
var models = []struct {
	name     string
	provider string
	model    string
	endpoint string
	apiKey   string
}{
	// Ollama 로컬 모델들
	{"qwen2.5:7b", "ollama", "qwen2.5:7b", "http://localhost:11434", ""},
	{"qwen2.5:14b", "ollama", "qwen2.5:14b", "http://localhost:11434", ""},
	{"llama3.2:3b", "ollama", "llama3.2:3b", "http://localhost:11434", ""},
	{"gemma2:9b", "ollama", "gemma2:9b", "http://localhost:11434", ""},
	{"mistral:7b", "ollama", "mistral:7b", "http://localhost:11434", ""},

	// Solar Pro2 (API)
	{"solar-pro2", "solar", "solar-pro2", "https://api.upstage.ai/v1", ""},
}

func main() {
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println("  k13d AI Model Benchmark")
	fmt.Println("  Based on k8s-ai-bench methodology")
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println()

	// Solar API 키 설정
	solarAPIKey := os.Getenv("SOLAR_API_KEY")
	if solarAPIKey == "" {
		fmt.Println("Warning: SOLAR_API_KEY not set, solar models will be skipped")
	}

	// 벤치마크 태스크 로드
	tasksPath := "pkg/eval/benchmark_tasks.yaml"
	benchCfg, err := eval.LoadBenchmarkTasks(tasksPath)
	if err != nil {
		fmt.Printf("Failed to load benchmark tasks: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Loaded %d benchmark tasks\n\n", len(benchCfg.Tasks))

	// Ollama 실행 확인
	if !checkOllama() {
		fmt.Println("Ollama is not running. Starting...")
		startOllama()
		time.Sleep(3 * time.Second)
	}

	var results []*eval.ModelBenchmark
	ctx := context.Background()

	for i, m := range models {
		fmt.Printf("\n[%d/%d] Testing: %s (%s)\n", i+1, len(models), m.name, m.provider)
		fmt.Println(strings.Repeat("-", 50))

		// Ollama 모델 다운로드
		if m.provider == "ollama" {
			if !pullOllamaModel(m.model) {
				fmt.Printf("  Skipping %s (download failed)\n", m.name)
				continue
			}
		}

		// 설정 생성
		cfg := &config.Config{
			LLM: config.LLMConfig{
				Provider: m.provider,
				Model:    m.model,
				Endpoint: m.endpoint,
				APIKey:   m.apiKey,
			},
			Language: "ko",
		}

		// Solar API 키 설정
		if m.provider == "solar" {
			cfg.LLM.APIKey = solarAPIKey
		}

		// 벤치마크 실행
		result, err := eval.RunBenchmark(ctx, cfg, benchCfg.Tasks)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		results = append(results, result)

		// 결과 요약 출력
		fmt.Printf("\n  Pass Rate: %.1f%% (%d/%d)\n",
			result.PassRate, result.PassedTasks, result.TotalTasks)
		fmt.Printf("  Avg Response Time: %.2fs\n", result.AvgRespTime.Seconds())
	}

	// 결과 저장
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Saving results...")

	// JSON 저장
	jsonPath := "benchmark_results.json"
	if err := eval.SaveResults(results, jsonPath); err != nil {
		fmt.Printf("Failed to save JSON: %v\n", err)
	} else {
		fmt.Printf("  JSON: %s\n", jsonPath)
	}

	// Markdown 보고서 생성
	report := eval.GenerateMarkdownReport(results)
	mdPath := "BENCHMARK_RESULTS.md"
	if err := os.WriteFile(mdPath, []byte(report), 0644); err != nil {
		fmt.Printf("Failed to save report: %v\n", err)
	} else {
		fmt.Printf("  Report: %s\n", mdPath)
	}

	fmt.Println("\nDone!")
}

func checkOllama() bool {
	cmd := exec.Command("curl", "-s", "http://localhost:11434/api/tags")
	err := cmd.Run()
	return err == nil
}

func startOllama() {
	cmd := exec.Command("ollama", "serve")
	cmd.Start()
}

func pullOllamaModel(model string) bool {
	fmt.Printf("  Pulling %s...\n", model)
	cmd := exec.Command("ollama", "pull", model)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err == nil
}
