// Mock OpenAI API Server for Integration Testing
// Simulates OpenAI API responses for testing k13d without real API calls
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	mockDelay       time.Duration
	supportTools    bool
	requestCounter  int
	toolCallCounter int
)

func init() {
	delayMS, _ := strconv.Atoi(os.Getenv("MOCK_DELAY_MS"))
	if delayMS > 0 {
		mockDelay = time.Duration(delayMS) * time.Millisecond
	}
	supportTools = os.Getenv("MOCK_TOOL_CALLING") == "true"
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Tools    []Tool        `json:"tools,omitempty"`
	Stream   bool          `json:"stream"`
}

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int          `json:"index"`
	Message      *ChatMessage `json:"message,omitempty"`
	Delta        *ChatMessage `json:"delta,omitempty"`
	FinishReason string       `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelData `json:"data"`
}

type ModelData struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func main() {
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/v1/chat/completions", handleChatCompletions)
	http.HandleFunc("/v1/models", handleModels)
	http.HandleFunc("/chat/completions", handleChatCompletions) // Some clients omit /v1
	http.HandleFunc("/models", handleModels)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Mock OpenAI server starting on port %s (tool_calling=%v, delay=%v)", port, supportTools, mockDelay)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleModels(w http.ResponseWriter, r *http.Request) {
	response := ModelsResponse{
		Object: "list",
		Data: []ModelData{
			{ID: "gpt-4", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai"},
			{ID: "gpt-4-turbo", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai"},
			{ID: "gpt-3.5-turbo", Object: "model", Created: time.Now().Unix(), OwnedBy: "openai"},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authorization
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		http.Error(w, `{"error":{"message":"Invalid API key","type":"invalid_request_error"}}`, http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":{"message":"Failed to read request body"}}`, http.StatusBadRequest)
		return
	}

	var req ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, `{"error":{"message":"Invalid JSON"}}`, http.StatusBadRequest)
		return
	}

	requestCounter++
	log.Printf("[Request #%d] Model: %s, Stream: %v, Tools: %d", requestCounter, req.Model, req.Stream, len(req.Tools))

	// Simulate network delay
	if mockDelay > 0 {
		time.Sleep(mockDelay)
	}

	// Handle tool calls if tools are provided and supported
	if supportTools && len(req.Tools) > 0 && shouldCallTool(req) {
		handleToolCallResponse(w, req)
		return
	}

	// Handle streaming
	if req.Stream {
		handleStreamingResponse(w, req)
		return
	}

	// Non-streaming response
	handleNonStreamingResponse(w, req)
}

func shouldCallTool(req ChatRequest) bool {
	// Check if the last message suggests a kubectl command
	if len(req.Messages) > 0 {
		lastMsg := req.Messages[len(req.Messages)-1]
		content := strings.ToLower(lastMsg.Content)

		// Keywords that suggest kubectl commands
		keywords := []string{"pods", "deployments", "services", "nodes", "namespace", "get", "describe", "logs", "kubectl"}
		for _, kw := range keywords {
			if strings.Contains(content, kw) {
				return true
			}
		}
	}
	return false
}

func handleToolCallResponse(w http.ResponseWriter, req ChatRequest) {
	toolCallCounter++

	// Find kubectl tool
	var kubectlTool *Tool
	for _, t := range req.Tools {
		if t.Function.Name == "kubectl" || strings.Contains(t.Function.Name, "kubectl") {
			kubectlTool = &t
			break
		}
	}

	// Generate appropriate kubectl command based on request
	command := generateKubectlCommand(req)

	toolCallID := fmt.Sprintf("call_%d", toolCallCounter)
	toolName := "kubectl"
	if kubectlTool != nil {
		toolName = kubectlTool.Function.Name
	}

	response := ChatResponse{
		ID:      fmt.Sprintf("chatcmpl-mock-%d", requestCounter),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: &ChatMessage{
					Role:    "assistant",
					Content: "",
					ToolCalls: []ToolCall{
						{
							ID:   toolCallID,
							Type: "function",
							Function: FunctionCall{
								Name:      toolName,
								Arguments: fmt.Sprintf(`{"command": "%s"}`, command),
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
		Usage: Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func generateKubectlCommand(req ChatRequest) string {
	if len(req.Messages) == 0 {
		return "kubectl get pods"
	}

	content := strings.ToLower(req.Messages[len(req.Messages)-1].Content)

	switch {
	case strings.Contains(content, "nodes"):
		return "kubectl get nodes -o wide"
	case strings.Contains(content, "deployments"):
		return "kubectl get deployments -A"
	case strings.Contains(content, "services"):
		return "kubectl get svc -A"
	case strings.Contains(content, "namespace"):
		return "kubectl get namespaces"
	case strings.Contains(content, "logs"):
		return "kubectl logs -l app=nginx --tail=100"
	case strings.Contains(content, "describe"):
		return "kubectl describe pods"
	default:
		return "kubectl get pods -A"
	}
}

func handleStreamingResponse(w http.ResponseWriter, req ChatRequest) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	response := generateResponse(req)
	words := strings.Split(response, " ")

	// Stream word by word
	for i, word := range words {
		chunk := ChatResponse{
			ID:      fmt.Sprintf("chatcmpl-mock-%d", requestCounter),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []Choice{
				{
					Index: 0,
					Delta: &ChatMessage{
						Content: word + " ",
					},
					FinishReason: "",
				},
			},
		}

		if i == len(words)-1 {
			chunk.Choices[0].FinishReason = "stop"
		}

		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func handleNonStreamingResponse(w http.ResponseWriter, req ChatRequest) {
	response := ChatResponse{
		ID:      fmt.Sprintf("chatcmpl-mock-%d", requestCounter),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: &ChatMessage{
					Role:    "assistant",
					Content: generateResponse(req),
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func generateResponse(req ChatRequest) string {
	if len(req.Messages) == 0 {
		return "Hello! How can I help you with Kubernetes today?"
	}

	lastMsg := req.Messages[len(req.Messages)-1]
	content := strings.ToLower(lastMsg.Content)

	// Check for tool call results
	if lastMsg.Role == "tool" {
		return "Based on the command output, I can see the current state of your cluster. Is there anything specific you'd like me to explain or any actions you'd like to take?"
	}

	// Simple ping response
	if strings.Contains(content, "ping") || strings.Contains(content, "hear me") || strings.Contains(content, "ok") {
		return "OK"
	}

	// Kubernetes-related responses
	switch {
	case strings.Contains(content, "pods"):
		return "I can help you with pods. Would you like me to list all pods, describe a specific pod, or check pod logs?"
	case strings.Contains(content, "deployment"):
		return "I can help you manage deployments. Would you like to list deployments, scale them, or check their status?"
	case strings.Contains(content, "service"):
		return "I can help you with Kubernetes services. Would you like to list services, expose a deployment, or check service endpoints?"
	case strings.Contains(content, "node"):
		return "I can help you check node status. Would you like to see all nodes, their resources, or check for any issues?"
	case strings.Contains(content, "namespace"):
		return "I can help you work with namespaces. Would you like to list namespaces, create a new one, or switch contexts?"
	case strings.Contains(content, "error") || strings.Contains(content, "problem"):
		return "I'd be happy to help troubleshoot. Can you describe the issue in more detail or share any error messages you're seeing?"
	case strings.Contains(content, "explain"):
		return "This resource defines how Kubernetes should manage your application. Let me break down the key components for you."
	default:
		return "I'm your Kubernetes AI assistant. I can help you with cluster management, troubleshooting, and explaining resources. What would you like to do?"
	}
}
