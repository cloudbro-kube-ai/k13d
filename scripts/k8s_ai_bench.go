//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// k8s-ai-bench 태스크들 (실제 벤치마크에서 가져옴)
var benchTasks = []struct {
	id         string
	difficulty string
	prompt     string
	checkFor   []string // 응답에 포함되어야 할 키워드
}{
	// Easy tasks
	{
		id:         "create-pod",
		difficulty: "easy",
		prompt:     "Please create a nginx pod named web-server in the web-server namespace",
		checkFor:   []string{"kubectl", "run", "nginx"},
	},
	{
		id:         "create-pod-resources-limits",
		difficulty: "easy",
		prompt:     "Create namespace 'limits-test' with a pod 'resource-limits-pod' using httpd:alpine image. Container 'my-container' should have CPU request 60m, limit 600m, and memory request/limit of 62Mi.",
		checkFor:   []string{"resources", "limits", "requests"},
	},
	{
		id:         "fix-pending-pod",
		difficulty: "easy",
		prompt:     "My pod homepage-pod in homepage-ns is stuck in pending state. Can you fix the issue so that pod status is in running state?",
		checkFor:   []string{"describe", "events"},
	},
	{
		id:         "fix-rbac-wrong-resource",
		difficulty: "easy",
		prompt:     "Why can't I list pods with my pod-reader service account in simple-rbac-setup namespace? Please fix it.",
		checkFor:   []string{"role", "rolebinding"},
	},
	{
		id:         "deployment-traffic-switch",
		difficulty: "easy",
		prompt:     "Our new checkout-service-green deployment in the e-commerce namespace has passed all tests. The current live version is checkout-service-blue. Can you switch all live traffic over to the green version now?",
		checkFor:   []string{"service", "selector"},
	},
	{
		id:         "resize-pvc",
		difficulty: "easy",
		prompt:     "resize the storage volume to 15Gi for the storage-pod in `resize-pv` namespace",
		checkFor:   []string{"pvc", "storage", "15Gi"},
	},

	// Medium tasks
	{
		id:         "fix-crashloop",
		difficulty: "medium",
		prompt:     "Please fix the error in the deployment named 'app' in namespace 'crashloop-test'",
		checkFor:   []string{"logs", "describe"},
	},
	{
		id:         "fix-image-pull",
		difficulty: "medium",
		prompt:     "Please fix the error in the deployment named 'app' in namespace 'debug'",
		checkFor:   []string{"image", "pull"},
	},
	{
		id:         "fix-probes",
		difficulty: "medium",
		prompt:     "My webapp in the orders namespace is not working? Can you please fix it?",
		checkFor:   []string{"probe", "health"},
	},
	{
		id:         "fix-service-routing",
		difficulty: "medium",
		prompt:     "Please diagnose and fix why the nginx service in namespace 'web' isn't accessible",
		checkFor:   []string{"service", "endpoint"},
	},
	{
		id:         "fix-service-with-no-endpoints",
		difficulty: "medium",
		prompt:     "Our frontend application in the webshop-frontend namespace is reporting connection errors. The logs show: 'Error: connection to web-app-service.webshop-frontend.svc.cluster.local failed: connection refused'. Can you help diagnose and fix this issue?",
		checkFor:   []string{"endpoint", "selector"},
	},
	{
		id:         "scale-deployment",
		difficulty: "medium",
		prompt:     "Scale up the replicas of deployment 'web-app' in namespace 'scale-test' by 100%",
		checkFor:   []string{"scale", "replicas"},
	},
	{
		id:         "scale-down-deployment",
		difficulty: "medium",
		prompt:     "Scale down the replicas of deployment 'web-service' in namespace 'scale-down-test' by 50%",
		checkFor:   []string{"scale", "replicas"},
	},
	{
		id:         "rolling-update-deployment",
		difficulty: "medium",
		prompt:     "Update the image of the web-app in rollout-test namespace to 1.22 version. Ensure there is zero downtime (or minimize disruption)",
		checkFor:   []string{"set image", "rolling"},
	},
	{
		id:         "create-simple-rbac",
		difficulty: "medium",
		prompt:     "Create a read-only role for pods bound to my reader-sa service account in the create-simple-rbac namespace.",
		checkFor:   []string{"role", "rolebinding", "get", "list"},
	},
	{
		id:         "create-network-policy",
		difficulty: "medium",
		prompt:     "Create a NetworkPolicy named 'np' in namespace 'ns1' that: 1. Allows egress traffic only to pods in namespace 'ns2' (incoming traffic not affected) 2. Allows DNS traffic (port 53 TCP and UDP) 3. Blocks all other outgoing traffic",
		checkFor:   []string{"networkpolicy", "egress"},
	},
	{
		id:         "debug-app-logs",
		difficulty: "medium",
		prompt:     "What wrong with my calc-app-pod in the calc-app namespace?",
		checkFor:   []string{"logs", "kubectl"},
	},
	{
		id:         "create-pod-mount-configmaps",
		difficulty: "medium",
		prompt:     "Create namespace 'color-size-settings' with two ConfigMaps - 'color-settings' with key 'color=blue' and 'size-settings' with key 'size=medium'. Create an nginx:alpine pod 'pod1' in namespace 'color-size-settings'. The pod `pod1` should use the value of 'color' key from 'color-settings' ConfigMap as an env var 'COLOR' and mounts all keys in the 'size-settings' ConfigMap under '/etc/sizes/' directory.",
		checkFor:   []string{"configmap", "volume", "env"},
	},
	{
		id:         "multi-container-pod-communication",
		difficulty: "medium",
		prompt:     "In the multi-container-logging namespace, run a pod called communication-pod with two containers: 1. A 'web-server' nginx instance that serves traffic 2. A 'logger' busybox instance that processes those logs from a shared volume",
		checkFor:   []string{"containers", "volume"},
	},
	{
		id:         "list-images-for-pods",
		difficulty: "medium",
		prompt:     "What images are all pods running in the cluster?",
		checkFor:   []string{"kubectl", "get", "pods"},
	},

	// Hard tasks
	{
		id:         "horizontal-pod-autoscaler",
		difficulty: "hard",
		prompt:     "Create a HorizontalPodAutoscaler for deployment 'web-app' in namespace 'hpa-test' targeting 50% CPU utilization with min=1 and max=3 replicas",
		checkFor:   []string{"hpa", "autoscale", "cpu"},
	},
	{
		id:         "create-canary-deployment",
		difficulty: "hard",
		prompt:     "We want to test a new version of our recommendation engine (image tag 1.29) in production without disturbing the existing stable deployment. Can you deploy it as a canary (as engine-v2-1)? We want about 50% of traffic to go to the new version.",
		checkFor:   []string{"deployment", "service", "selector"},
	},
	{
		id:         "statefulset-lifecycle",
		difficulty: "hard",
		prompt:     "Deploy a 3-replica StatefulSet db in namespace statefulset-test with each pod mounting a 1Gi PVC at /data containing a file `test` populated with the string `initial_data`. Then, scale back down to 1 replicas.",
		checkFor:   []string{"statefulset", "pvc", "volumeclaim"},
	},
}

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
}

type Result struct {
	Model      string
	TaskID     string
	Difficulty string
	Passed     bool
	Time       time.Duration
}

func main() {
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println("  k8s-ai-bench Style Evaluation")
	fmt.Println("  Tasks: 23 (6 easy, 14 medium, 3 hard)")
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println()

	var allResults []Result
	modelStats := make(map[string]struct {
		easy, medium, hard                int
		easyTotal, mediumTotal, hardTotal int
		totalTime                         time.Duration
	})

	for _, model := range models {
		fmt.Printf("\n### %s\n", model.name)
		fmt.Println(strings.Repeat("-", 60))

		stats := modelStats[model.name]

		for _, task := range benchTasks {
			start := time.Now()
			response, err := callAPI(model.endpoint, model.apiKey, model.name, task.prompt)
			elapsed := time.Since(start)

			passed := false
			if err == nil {
				passed = checkResponse(response, task.checkFor)
			}

			result := Result{
				Model:      model.name,
				TaskID:     task.id,
				Difficulty: task.difficulty,
				Passed:     passed,
				Time:       elapsed,
			}
			allResults = append(allResults, result)

			// Update stats
			switch task.difficulty {
			case "easy":
				stats.easyTotal++
				if passed {
					stats.easy++
				}
			case "medium":
				stats.mediumTotal++
				if passed {
					stats.medium++
				}
			case "hard":
				stats.hardTotal++
				if passed {
					stats.hard++
				}
			}
			stats.totalTime += elapsed

			status := "✓"
			if !passed {
				status = "✗"
			}
			fmt.Printf("  [%s] %-35s (%s) %.1fs\n", status, task.id, task.difficulty, elapsed.Seconds())
		}

		modelStats[model.name] = stats
		total := stats.easy + stats.medium + stats.hard
		totalTasks := stats.easyTotal + stats.mediumTotal + stats.hardTotal
		fmt.Printf("\n  Summary: %d/%d (%.0f%%) | Easy: %d/%d | Medium: %d/%d | Hard: %d/%d\n",
			total, totalTasks, float64(total)/float64(totalTasks)*100,
			stats.easy, stats.easyTotal,
			stats.medium, stats.mediumTotal,
			stats.hard, stats.hardTotal)
	}

	// Final summary
	printFinalSummary(modelStats)
	saveReport(allResults, modelStats)
}

func checkResponse(response string, keywords []string) bool {
	lower := strings.ToLower(response)
	for _, kw := range keywords {
		if !strings.Contains(lower, strings.ToLower(kw)) {
			return false
		}
	}
	return true
}

func callAPI(endpoint, apiKey, model, prompt string) (string, error) {
	systemPrompt := `You are a Kubernetes expert. When asked to perform tasks, provide the exact kubectl commands or YAML manifests needed. Be concise and practical.`

	reqBody := ChatRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", endpoint+"/chat/completions", bytes.NewBuffer(jsonData))
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
	json.Unmarshal(body, &chatResp)

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response")
	}
	return chatResp.Choices[0].Message.Content, nil
}

func printFinalSummary(stats map[string]struct {
	easy, medium, hard                int
	easyTotal, mediumTotal, hardTotal int
	totalTime                         time.Duration
}) {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("  FINAL RESULTS (k8s-ai-bench style)")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
	fmt.Printf("%-18s %8s %8s %8s %10s %10s\n", "Model", "Easy", "Medium", "Hard", "Total", "Avg Time")
	fmt.Println(strings.Repeat("-", 70))

	type modelScore struct {
		name  string
		score float64
		stats struct {
			easy, medium, hard                int
			easyTotal, mediumTotal, hardTotal int
			totalTime                         time.Duration
		}
	}

	var scores []modelScore
	for name, s := range stats {
		total := s.easy + s.medium + s.hard
		totalTasks := s.easyTotal + s.mediumTotal + s.hardTotal
		scores = append(scores, modelScore{
			name:  name,
			score: float64(total) / float64(totalTasks) * 100,
			stats: s,
		})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	for _, m := range scores {
		s := m.stats
		total := s.easy + s.medium + s.hard
		totalTasks := s.easyTotal + s.mediumTotal + s.hardTotal
		avgTime := s.totalTime / time.Duration(totalTasks)
		fmt.Printf("%-18s %4d/%-3d %4d/%-3d %4d/%-3d %5.1f%% %10s\n",
			m.name,
			s.easy, s.easyTotal,
			s.medium, s.mediumTotal,
			s.hard, s.hardTotal,
			float64(total)/float64(totalTasks)*100,
			avgTime.Round(time.Millisecond))
	}
}

func saveReport(results []Result, stats map[string]struct {
	easy, medium, hard                int
	easyTotal, mediumTotal, hardTotal int
	totalTime                         time.Duration
}) {
	var sb strings.Builder

	sb.WriteString("# k8s-ai-bench Evaluation Results\n\n")
	sb.WriteString(fmt.Sprintf("**Date**: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Tasks**: %d (Easy: 6, Medium: 14, Hard: 3)\n\n", len(benchTasks)))

	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Model | Easy | Medium | Hard | Total | Avg Time |\n")
	sb.WriteString("|-------|------|--------|------|-------|----------|\n")

	type modelScore struct {
		name  string
		score float64
		stats struct {
			easy, medium, hard                int
			easyTotal, mediumTotal, hardTotal int
			totalTime                         time.Duration
		}
	}

	var scores []modelScore
	for name, s := range stats {
		total := s.easy + s.medium + s.hard
		totalTasks := s.easyTotal + s.mediumTotal + s.hardTotal
		scores = append(scores, modelScore{name: name, score: float64(total) / float64(totalTasks) * 100, stats: s})
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })

	for _, m := range scores {
		s := m.stats
		total := s.easy + s.medium + s.hard
		totalTasks := s.easyTotal + s.mediumTotal + s.hardTotal
		avgTime := s.totalTime / time.Duration(totalTasks)
		sb.WriteString(fmt.Sprintf("| %s | %d/%d | %d/%d | %d/%d | %.1f%% | %s |\n",
			m.name, s.easy, s.easyTotal, s.medium, s.mediumTotal, s.hard, s.hardTotal,
			float64(total)/float64(totalTasks)*100, avgTime.Round(time.Millisecond)))
	}

	sb.WriteString("\n## Detailed Results\n\n")
	for _, model := range models {
		sb.WriteString(fmt.Sprintf("### %s\n\n", model.name))
		sb.WriteString("| Task | Difficulty | Result |\n")
		sb.WriteString("|------|------------|--------|\n")
		for _, r := range results {
			if r.Model == model.name {
				status := "✓"
				if !r.Passed {
					status = "✗"
				}
				sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", r.TaskID, r.Difficulty, status))
			}
		}
		sb.WriteString("\n")
	}

	os.WriteFile("K8S_AI_BENCH_RESULTS.md", []byte(sb.String()), 0644)
	fmt.Println("\nReport saved to: K8S_AI_BENCH_RESULTS.md")
}
