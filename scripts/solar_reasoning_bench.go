//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Solar 모델 설정
var solarConfigs = []struct {
	name            string
	reasoningEffort string
}{
	{"solar-pro2 (low)", "low"},
	{"solar-pro2 (medium)", "medium"},
	{"solar-pro2 (high)", "high"},
}

const solarEndpoint = "https://api.upstage.ai/v1"

var solarAPIKey = os.Getenv("SOLAR_API_KEY")

// k8s-ai-bench 태스크들
var benchTasks = []struct {
	id         string
	difficulty string
	prompt     string
	checkFor   []string
}{
	// Easy tasks (6)
	{"create-pod", "easy", "Please create a nginx pod named web-server in the web-server namespace", []string{"kubectl", "run", "nginx"}},
	{"create-pod-resources-limits", "easy", "Create namespace 'limits-test' with a pod 'resource-limits-pod' using httpd:alpine image. Container 'my-container' should have CPU request 60m, limit 600m, and memory request/limit of 62Mi.", []string{"resources", "limits", "requests"}},
	{"fix-pending-pod", "easy", "My pod homepage-pod in homepage-ns is stuck in pending state. Can you fix the issue so that pod status is in running state?", []string{"describe", "events"}},
	{"fix-rbac-wrong-resource", "easy", "Why can't I list pods with my pod-reader service account in simple-rbac-setup namespace? Please fix it.", []string{"role", "rolebinding"}},
	{"deployment-traffic-switch", "easy", "Our new checkout-service-green deployment in the e-commerce namespace has passed all tests. The current live version is checkout-service-blue. Can you switch all live traffic over to the green version now?", []string{"service", "selector"}},
	{"resize-pvc", "easy", "resize the storage volume to 15Gi for the storage-pod in `resize-pv` namespace", []string{"pvc", "storage", "15Gi"}},

	// Medium tasks (14)
	{"fix-crashloop", "medium", "Please fix the error in the deployment named 'app' in namespace 'crashloop-test'", []string{"logs", "describe"}},
	{"fix-image-pull", "medium", "Please fix the error in the deployment named 'app' in namespace 'debug'", []string{"image", "pull"}},
	{"fix-probes", "medium", "My webapp in the orders namespace is not working? Can you please fix it?", []string{"probe", "health"}},
	{"fix-service-routing", "medium", "Please diagnose and fix why the nginx service in namespace 'web' isn't accessible", []string{"service", "endpoint"}},
	{"fix-service-with-no-endpoints", "medium", "Our frontend application in the webshop-frontend namespace is reporting connection errors. Can you help diagnose and fix this issue?", []string{"endpoint", "selector"}},
	{"scale-deployment", "medium", "Scale up the replicas of deployment 'web-app' in namespace 'scale-test' by 100%", []string{"scale", "replicas"}},
	{"scale-down-deployment", "medium", "Scale down 'web-scale-down' deployment in the 'scale-down-ns' namespace to 0 replicas", []string{"scale", "replicas", "0"}},
	{"rolling-update-deployment", "medium", "Update the deployment 'web-app' in namespace 'rolling' from nginx:1.14 to nginx:1.25", []string{"image", "nginx:1.25"}},
	{"create-simple-rbac", "medium", "Please create a ServiceAccount 'dev-sa', Role 'pod-reader', and RoleBinding to allow listing pods in default namespace", []string{"serviceaccount", "role", "rolebinding"}},
	{"create-network-policy", "medium", "Create a NetworkPolicy 'api-allow' that allows pods with label 'app=api' to accept traffic only from pods labeled 'app=frontend' on port 8080", []string{"networkpolicy", "ingress", "8080"}},
	{"debug-app-logs", "medium", "My app in namespace 'logging-test' with deployment 'log-app' is crashing. Find the error from logs.", []string{"logs", "error"}},
	{"create-pod-mount-configmaps", "medium", "Create a pod with nginx that mounts configmap 'app-config' as volume at /etc/config", []string{"configmap", "volume", "mount"}},
	{"multi-container-pod-communication", "medium", "Create a pod with sidecar pattern: main container runs nginx, sidecar runs busybox that logs nginx access logs", []string{"sidecar", "container"}},
	{"list-images-for-pods", "medium", "List all container images used by pods in the kube-system namespace", []string{"image", "kube-system"}},

	// Hard tasks (3)
	{"horizontal-pod-autoscaler", "hard", "Create HPA for deployment 'web-app' in 'autoscale-ns': min 2, max 10 replicas, target CPU 50%", []string{"hpa", "autoscal", "cpu"}},
	{"create-canary-deployment", "hard", "Create a canary deployment setup: 90% traffic to 'app-stable', 10% to 'app-canary' using a single service", []string{"canary", "service", "selector"}},
	{"statefulset-lifecycle", "hard", "Create a StatefulSet 'db' with 3 replicas using postgres:15, with persistent storage of 1Gi per replica", []string{"statefulset", "volumeclaimtemplate", "postgres"}},
}

type SolarChatRequest struct {
	Model           string    `json:"model"`
	Messages        []Message `json:"messages"`
	ReasoningEffort string    `json:"reasoning_effort,omitempty"`
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
	Config     string
	TaskID     string
	Difficulty string
	Passed     bool
	Time       time.Duration
}

func main() {
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println("  Solar Pro2 Reasoning Effort Comparison")
	fmt.Println("  Tasks: 23 (6 easy, 14 medium, 3 hard)")
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println()

	var allResults []Result
	configStats := make(map[string]struct {
		easy, medium, hard                int
		easyTotal, mediumTotal, hardTotal int
		totalTime                         time.Duration
	})

	for _, cfg := range solarConfigs {
		fmt.Printf("\n### %s\n", cfg.name)
		fmt.Println(strings.Repeat("-", 60))

		stats := configStats[cfg.name]

		for _, task := range benchTasks {
			start := time.Now()
			response, err := callSolarAPI(cfg.reasoningEffort, task.prompt)
			elapsed := time.Since(start)

			passed := false
			if err == nil {
				passed = checkResponse(response, task.checkFor)
			}

			result := Result{
				Config:     cfg.name,
				TaskID:     task.id,
				Difficulty: task.difficulty,
				Passed:     passed,
				Time:       elapsed,
			}
			allResults = append(allResults, result)

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

		configStats[cfg.name] = stats
		total := stats.easy + stats.medium + stats.hard
		totalTasks := stats.easyTotal + stats.mediumTotal + stats.hardTotal
		fmt.Printf("\n  Summary: %d/%d (%.0f%%) | Easy: %d/%d | Medium: %d/%d | Hard: %d/%d\n",
			total, totalTasks, float64(total)/float64(totalTasks)*100,
			stats.easy, stats.easyTotal,
			stats.medium, stats.mediumTotal,
			stats.hard, stats.hardTotal)
	}

	// Final summary
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("  FINAL RESULTS - Solar Pro2 Reasoning Effort Comparison")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
	fmt.Printf("%-25s %6s %8s %6s %8s %10s\n", "Config", "Easy", "Medium", "Hard", "Total", "Avg Time")
	fmt.Println(strings.Repeat("-", 70))

	for _, cfg := range solarConfigs {
		stats := configStats[cfg.name]
		total := stats.easy + stats.medium + stats.hard
		totalTasks := stats.easyTotal + stats.mediumTotal + stats.hardTotal
		avgTime := stats.totalTime / time.Duration(totalTasks)
		fmt.Printf("%-25s %d/%d    %d/%d     %d/%d   %5.1f%%  %8.3fs\n",
			cfg.name,
			stats.easy, stats.easyTotal,
			stats.medium, stats.mediumTotal,
			stats.hard, stats.hardTotal,
			float64(total)/float64(totalTasks)*100,
			avgTime.Seconds())
	}
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

func callSolarAPI(reasoningEffort, prompt string) (string, error) {
	systemPrompt := `You are a Kubernetes expert. When asked to perform tasks, provide the exact kubectl commands or YAML manifests needed. Be concise and practical.`

	reqBody := SolarChatRequest{
		Model: "solar-pro2",
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		ReasoningEffort: reasoningEffort,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", solarEndpoint+"/chat/completions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+solarAPIKey)

	client := &http.Client{Timeout: 180 * time.Second}
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
