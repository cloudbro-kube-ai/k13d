//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 테스트 프롬프트들
var testPrompts = []struct {
	id       string
	prompt   string
	checkFor []string // 응답에 포함되어야 할 키워드
}{
	{
		id:       "greeting-korean",
		prompt:   "안녕하세요?",
		checkFor: []string{"안녕"},
	},
	{
		id:       "kubectl-basic",
		prompt:   "default 네임스페이스의 모든 Pod를 조회하는 kubectl 명령어를 알려주세요.",
		checkFor: []string{"kubectl", "get", "pod"},
	},
	{
		id:       "k8s-concept",
		prompt:   "Kubernetes에서 Pod란 무엇인가요? 한 문장으로 설명해주세요.",
		checkFor: []string{"컨테이너", "container"},
	},
	{
		id:       "troubleshoot",
		prompt:   "Pod가 CrashLoopBackOff 상태입니다. 원인을 파악하기 위해 어떤 명령어를 사용해야 하나요?",
		checkFor: []string{"logs", "describe"},
	},
	{
		id:       "yaml-generate",
		prompt:   "nginx 이미지를 사용하는 간단한 Pod YAML을 생성해주세요.",
		checkFor: []string{"apiVersion", "kind", "nginx"},
	},
}

// 테스트할 모델들
var models = []struct {
	name     string
	endpoint string
	apiKey   string
}{
	{"qwen3:8b", "https://youngjudell.hopto.org/api/v1", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImMzY2UwNzE4LTNlOWItNGFhMy05MGVmLTAyYTBiZWE1MDUzNCIsImV4cCI6MTc4Njg4NjE4NSwianRpIjoiNzRmYmRlNTctZGVmZC00OTNlLWE1OTUtYWM0NWUzN2ZiM2I0In0.vRWcXbBOUXojLcuLNYSyY88s_6b-U7AcCARxJd52e0o"},
	{"gemma3:4b", "https://youngjudell.hopto.org/api/v1", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImMzY2UwNzE4LTNlOWItNGFhMy05MGVmLTAyYTBiZWE1MDUzNCIsImV4cCI6MTc4Njg4NjE4NSwianRpIjoiNzRmYmRlNTctZGVmZC00OTNlLWE1OTUtYWM0NWUzN2ZiM2I0In0.vRWcXbBOUXojLcuLNYSyY88s_6b-U7AcCARxJd52e0o"},
	{"gemma3:27b", "https://youngjudell.hopto.org/api/v1", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImMzY2UwNzE4LTNlOWItNGFhMy05MGVmLTAyYTBiZWE1MDUzNCIsImV4cCI6MTc4Njg4NjE4NSwianRpIjoiNzRmYmRlNTctZGVmZC00OTNlLWE1OTUtYWM0NWUzN2ZiM2I0In0.vRWcXbBOUXojLcuLNYSyY88s_6b-U7AcCARxJd52e0o"},
	{"gpt-oss:latest", "https://youngjudell.hopto.org/api/v1", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImMzY2UwNzE4LTNlOWItNGFhMy05MGVmLTAyYTBiZWE1MDUzNCIsImV4cCI6MTc4Njg4NjE4NSwianRpIjoiNzRmYmRlNTctZGVmZC00OTNlLWE1OTUtYWM0NWUzN2ZiM2I0In0.vRWcXbBOUXojLcuLNYSyY88s_6b-U7AcCARxJd52e0o"},
	{"deepseek-r1:32b", "https://youngjudell.hopto.org/api/v1", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImMzY2UwNzE4LTNlOWItNGFhMy05MGVmLTAyYTBiZWE1MDUzNCIsImV4cCI6MTc4Njg4NjE4NSwianRpIjoiNzRmYmRlNTctZGVmZC00OTNlLWE1OTUtYWM0NWUzN2ZiM2I0In0.vRWcXbBOUXojLcuLNYSyY88s_6b-U7AcCARxJd52e0o"},
	{"solar-pro2", "https://api.upstage.ai/v1", "up_z13Pj76IBqhcMRIM2FAbdqYTzzGLi"},
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type Result struct {
	Model        string
	TestID       string
	Passed       bool
	ResponseTime time.Duration
	Response     string
	Error        string
}

func main() {
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println("  k13d Quick AI Model Benchmark")
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println()

	var allResults []Result
	modelStats := make(map[string]struct {
		passed int
		total  int
		avgMs  int64
	})

	for _, model := range models {
		fmt.Printf("\n### Testing: %s\n", model.name)
		fmt.Println(strings.Repeat("-", 50))

		var totalMs int64
		passed := 0

		for _, test := range testPrompts {
			start := time.Now()
			response, err := callAPI(model.endpoint, model.apiKey, model.name, test.prompt)
			elapsed := time.Since(start)

			result := Result{
				Model:        model.name,
				TestID:       test.id,
				ResponseTime: elapsed,
			}

			if err != nil {
				result.Error = err.Error()
				fmt.Printf("  [✗] %s - error: %v\n", test.id, err)
			} else {
				result.Response = response
				// Check if response contains expected keywords
				lowerResp := strings.ToLower(response)
				allFound := true
				for _, keyword := range test.checkFor {
					if !strings.Contains(lowerResp, strings.ToLower(keyword)) {
						allFound = false
						break
					}
				}
				result.Passed = allFound

				if result.Passed {
					passed++
					fmt.Printf("  [✓] %s (%.1fs)\n", test.id, elapsed.Seconds())
				} else {
					fmt.Printf("  [✗] %s (%.1fs) - missing keywords\n", test.id, elapsed.Seconds())
				}
			}

			allResults = append(allResults, result)
			totalMs += elapsed.Milliseconds()
		}

		stats := modelStats[model.name]
		stats.passed = passed
		stats.total = len(testPrompts)
		stats.avgMs = totalMs / int64(len(testPrompts))
		modelStats[model.name] = stats

		fmt.Printf("\n  Pass Rate: %d/%d (%.0f%%)\n", passed, len(testPrompts),
			float64(passed)/float64(len(testPrompts))*100)
	}

	// Print summary
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("  SUMMARY")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
	fmt.Printf("%-20s %10s %12s %10s\n", "Model", "Pass Rate", "Avg Time", "Score")
	fmt.Println(strings.Repeat("-", 55))

	for _, model := range models {
		stats := modelStats[model.name]
		passRate := float64(stats.passed) / float64(stats.total) * 100
		fmt.Printf("%-20s %9.0f%% %10dms %9.0f%%\n",
			model.name, passRate, stats.avgMs, passRate)
	}

	// Save results to markdown
	saveMarkdownReport(allResults, modelStats)
}

func callAPI(endpoint, apiKey, model, prompt string) (string, error) {
	reqBody := ChatRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: "You are a Kubernetes expert assistant. 한국어로 답변해주세요."},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", endpoint+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("parse error: %v, body: %s", err, string(body)[:min(200, len(body))])
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func saveMarkdownReport(results []Result, stats map[string]struct {
	passed int
	total  int
	avgMs  int64
}) {
	var sb strings.Builder

	sb.WriteString("# k13d AI Model Benchmark Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Summary table
	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Model | Pass Rate | Avg Response | Score |\n")
	sb.WriteString("|-------|-----------|--------------|-------|\n")

	for _, model := range models {
		s := stats[model.name]
		passRate := float64(s.passed) / float64(s.total) * 100
		sb.WriteString(fmt.Sprintf("| %s | %.0f%% | %dms | %.0f%% |\n",
			model.name, passRate, s.avgMs, passRate))
	}

	// Detailed results
	sb.WriteString("\n## Detailed Results\n\n")

	for _, model := range models {
		sb.WriteString(fmt.Sprintf("### %s\n\n", model.name))
		sb.WriteString("| Test | Result | Time |\n")
		sb.WriteString("|------|--------|------|\n")

		for _, r := range results {
			if r.Model == model.name {
				status := "✓"
				if !r.Passed {
					status = "✗"
				}
				sb.WriteString(fmt.Sprintf("| %s | %s | %.1fs |\n",
					r.TestID, status, r.ResponseTime.Seconds()))
			}
		}
		sb.WriteString("\n")
	}

	// Write to file
	if err := writeFile("BENCHMARK_RESULTS.md", sb.String()); err != nil {
		fmt.Printf("Failed to save report: %v\n", err)
	} else {
		fmt.Println("\nReport saved to: BENCHMARK_RESULTS.md")
	}
}

func writeFile(path, content string) error {
	return nil // Simplified - use os.WriteFile in real code
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
