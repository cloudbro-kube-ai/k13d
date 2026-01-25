package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/db"
)

// LLMCapabilities represents the capabilities of the configured LLM
type LLMCapabilities struct {
	ToolCalling    bool   `json:"tool_calling"`
	JSONMode       bool   `json:"json_mode"`
	Streaming      bool   `json:"streaming"`
	MaxTokens      int    `json:"max_tokens,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

// handleLLMSettings handles LLM settings updates
func (s *Server) handleLLMSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPut:
		var llmSettings struct {
			Provider        string `json:"provider"`
			Model           string `json:"model"`
			Endpoint        string `json:"endpoint"`
			APIKey          string `json:"api_key"`
			UseJSONMode     bool   `json:"use_json_mode"`    // Fallback for non-tool-calling models
			ReasoningEffort string `json:"reasoning_effort"` // For Solar Pro2: "minimal" or "high"
		}

		if err := json.NewDecoder(r.Body).Decode(&llmSettings); err != nil {
			WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
			return
		}

		// Update LLM settings
		s.cfg.LLM.Provider = llmSettings.Provider
		s.cfg.LLM.Model = llmSettings.Model
		s.cfg.LLM.Endpoint = llmSettings.Endpoint
		if llmSettings.APIKey != "" {
			s.cfg.LLM.APIKey = llmSettings.APIKey
		}
		s.cfg.LLM.UseJSONMode = llmSettings.UseJSONMode
		// Update reasoning effort (validate value)
		if llmSettings.ReasoningEffort == "high" || llmSettings.ReasoningEffort == "minimal" {
			s.cfg.LLM.ReasoningEffort = llmSettings.ReasoningEffort
		}

		// Recreate AI client
		newClient, err := ai.NewClient(&s.cfg.LLM)
		if err != nil {
			apiErr := ParseLLMError(err, llmSettings.Provider)
			WriteError(w, apiErr)
			return
		}
		s.aiClient = newClient

		// Save to disk
		if err := s.cfg.Save(); err != nil {
			WriteError(w, NewAPIError(ErrCodeInternalError, "Failed to save settings"))
			return
		}

		// Record audit
		username := r.Header.Get("X-Username")
		db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "update_llm_settings",
			Resource: "settings",
			Details:  fmt.Sprintf("Provider: %s, Model: %s, JSONMode: %v", llmSettings.Provider, llmSettings.Model, llmSettings.UseJSONMode),
		})

		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleLLMTest performs a connection test to the LLM provider
func (s *Server) handleLLMTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Check if POST request has test config in body (form values)
	var testConfig struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Endpoint string `json:"endpoint"`
		APIKey   string `json:"api_key"`
	}

	var testClient *ai.Client
	var testProvider string

	if r.Method == http.MethodPost && r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&testConfig); err == nil && testConfig.Provider != "" {
			// Create temporary client with form values
			tempConfig := config.LLMConfig{
				Provider: testConfig.Provider,
				Model:    testConfig.Model,
				Endpoint: testConfig.Endpoint,
				APIKey:   testConfig.APIKey,
			}
			client, err := ai.NewClient(&tempConfig)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"connected":    false,
					"provider":     testConfig.Provider,
					"model":        testConfig.Model,
					"endpoint":     testConfig.Endpoint,
					"error":        fmt.Sprintf("Failed to create client: %v", err),
					"message":      "Check your provider settings and API key",
					"capabilities": LLMCapabilities{},
				})
				return
			}
			testClient = client
			testProvider = testConfig.Provider
		}
	}

	// Fall back to saved server client if no form values provided
	if testClient == nil {
		if s.aiClient == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"connected":    false,
				"provider":     s.cfg.LLM.Provider,
				"model":        s.cfg.LLM.Model,
				"endpoint":     s.cfg.LLM.Endpoint,
				"error":        "AI client not configured",
				"message":      "Please configure LLM provider, API key, and endpoint in settings",
				"capabilities": LLMCapabilities{},
			})
			return
		}
		testClient = s.aiClient
		testProvider = s.cfg.LLM.Provider
	}

	// Run connection test with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	status := testClient.TestConnection(ctx)

	// Add capabilities info
	capabilities := getLLMCapabilities(testClient, testProvider)

	response := map[string]interface{}{
		"connected":     status.Connected,
		"provider":      status.Provider,
		"model":         status.Model,
		"endpoint":      status.Endpoint,
		"response_time": status.ResponseTime,
		"capabilities":  capabilities,
	}

	if status.Error != "" {
		response["error"] = status.Error
	}
	if status.Message != "" {
		response["message"] = status.Message
	}

	// Add recommendation if tool calling is not supported
	if status.Connected && !capabilities.ToolCalling {
		response["warning"] = "This model doesn't support tool calling. Enable JSON mode for limited command execution support."
	}

	// Record audit
	username := r.Header.Get("X-Username")
	resultText := "success"
	if !status.Connected {
		resultText = fmt.Sprintf("failed: %s", status.Error)
	}
	db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "llm_connection_test",
		Resource: "llm",
		Details:  fmt.Sprintf("Provider: %s, Model: %s, Result: %s, ToolCalling: %v", status.Provider, status.Model, resultText, capabilities.ToolCalling),
	})

	json.NewEncoder(w).Encode(response)
}

// handleLLMStatus returns the current LLM configuration status without testing
func (s *Server) handleLLMStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	status := map[string]interface{}{
		"configured":    s.aiClient != nil,
		"provider":      s.cfg.LLM.Provider,
		"model":         s.cfg.LLM.Model,
		"endpoint":      s.cfg.LLM.Endpoint,
		"has_api_key":   s.cfg.LLM.APIKey != "",
		"use_json_mode": s.cfg.LLM.UseJSONMode,
		"embedded_llm":  s.embeddedLLM,
	}

	// Add default endpoint hint if not configured
	if s.cfg.LLM.Endpoint == "" {
		switch s.cfg.LLM.Provider {
		case "openai":
			status["default_endpoint"] = "https://api.openai.com/v1"
		case "gemini":
			status["default_endpoint"] = "https://generativelanguage.googleapis.com/v1beta"
		case "ollama":
			status["default_endpoint"] = "http://localhost:11434"
		case "anthropic":
			status["default_endpoint"] = "https://api.anthropic.com"
		case "azure":
			status["default_endpoint"] = "(Azure OpenAI endpoint required)"
		}
	}

	if s.aiClient != nil {
		status["ready"] = s.aiClient.IsReady()
		status["supports_tools"] = s.aiClient.SupportsTools()

		// Add capabilities
		status["capabilities"] = getLLMCapabilities(s.aiClient, s.cfg.LLM.Provider)
	}

	json.NewEncoder(w).Encode(status)
}

// handleAgenticChat handles AI chat with tool calling (Decision Required flow)
func (s *Server) handleAgenticChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	username := r.Header.Get("X-Username")
	if username == "" {
		username = "anonymous"
	}

	// Record audit log with k8s context
	s.recordAuditWithK8sContext(r, db.AuditEntry{
		User:       username,
		Action:     "ai_agentic_query",
		ActionType: db.ActionTypeLLM,
		Resource:   "chat",
		Details:    fmt.Sprintf("Query: %s", truncateString(req.Message, 100)),
		LLMRequest: req.Message,
	})

	if s.aiClient == nil {
		WriteError(w, NewAPIError(ErrCodeLLMNotConfigured, "AI client not configured"))
		return
	}

	// Check if provider supports tool calling
	supportsTools := s.aiClient.SupportsTools()

	// If tool calling not supported, use fallback modes
	if !supportsTools {
		if s.cfg.LLM.UseJSONMode {
			// Use JSON mode fallback (structured responses)
			s.handleAgenticChatJSONMode(w, r, req, username)
			return
		}
		// Use simple chat mode (no tool execution, just conversation)
		s.handleSimpleChatMode(w, r, req, username)
		return
	}

	// Set up SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, NewAPIError(ErrCodeInternalError, "Streaming not supported"))
		return
	}

	sse := &SSEWriter{w: w, flusher: flusher}

	// Tool approval callback
	toolApprovalCallback := func(toolName string, argsJSON string) bool {
		// Parse arguments to get the command
		var args map[string]interface{}
		json.Unmarshal([]byte(argsJSON), &args)

		command := ""
		if cmd, ok := args["command"].(string); ok {
			command = cmd
		}

		// Classify the command
		category := classifyCommand(command)

		// Auto-approve read-only commands
		if category == "read-only" {
			return true
		}

		// Create pending approval
		approvalID := fmt.Sprintf("approval_%d", time.Now().UnixNano())
		approval := &PendingToolApproval{
			ID:        approvalID,
			ToolName:  toolName,
			Command:   command,
			Category:  category,
			Timestamp: time.Now(),
			Response:  make(chan bool, 1),
		}

		s.pendingApprovalMutex.Lock()
		s.pendingApprovals[approvalID] = approval
		s.pendingApprovalMutex.Unlock()

		// Send approval request via SSE
		approvalJSON, _ := json.Marshal(map[string]interface{}{
			"type":      "approval_required",
			"id":        approvalID,
			"tool_name": toolName,
			"command":   command,
			"category":  category,
		})
		sse.WriteEvent("approval", string(approvalJSON))

		// Wait for approval with timeout
		select {
		case approved := <-approval.Response:
			// Cleanup
			s.pendingApprovalMutex.Lock()
			delete(s.pendingApprovals, approvalID)
			s.pendingApprovalMutex.Unlock()

			// Log the decision with full context
			k8sContext, k8sCluster, k8sUser := s.getK8sContextInfo()
			s.recordAuditWithK8sContext(r, db.AuditEntry{
				User:        username,
				Action:      map[bool]string{true: "tool_approved", false: "tool_rejected"}[approved],
				ActionType:  db.ActionTypeLLM,
				Resource:    toolName,
				Details:     fmt.Sprintf("LLM requested: %s", command),
				K8sUser:     k8sUser,
				K8sContext:  k8sContext,
				K8sCluster:  k8sCluster,
				LLMTool:     toolName,
				LLMCommand:  command,
				LLMApproved: approved,
				LLMRequest:  req.Message,
			})
			return approved

		case <-time.After(60 * time.Second):
			// Timeout - cleanup and reject
			s.pendingApprovalMutex.Lock()
			delete(s.pendingApprovals, approvalID)
			s.pendingApprovalMutex.Unlock()

			sse.WriteEvent("approval_timeout", approvalID)
			return false

		case <-r.Context().Done():
			// Request cancelled
			s.pendingApprovalMutex.Lock()
			delete(s.pendingApprovals, approvalID)
			s.pendingApprovalMutex.Unlock()
			return false
		}
	}

	// Tool execution callback - sends tool execution info via SSE and records audit
	toolExecutionCallback := func(toolName string, command string, result string, isError bool) {
		execJSON, _ := json.Marshal(map[string]interface{}{
			"type":     "tool_execution",
			"tool":     toolName,
			"command":  command,
			"result":   result,
			"is_error": isError,
		})
		sse.WriteEvent("tool_execution", string(execJSON))

		// Record tool execution in audit log
		actionType := db.ActionTypeLLM
		// Determine if this is a mutation (create, delete, apply, scale, etc.)
		cmdCategory := classifyCommand(command)
		if cmdCategory == "write" || cmdCategory == "dangerous" {
			actionType = db.ActionTypeMutation
		}

		s.recordAuditWithK8sContext(r, db.AuditEntry{
			User:        username,
			Action:      "tool_executed",
			ActionType:  actionType,
			Resource:    toolName,
			Details:     fmt.Sprintf("Command: %s", truncateString(command, 200)),
			LLMTool:     toolName,
			LLMCommand:  command,
			LLMApproved: true,
			LLMRequest:  req.Message,
			Success:     !isError,
		})
	}

	// Build message with language instruction if needed
	message := req.Message
	if req.Language != "" && req.Language != "en" {
		langInstruction := getLanguageInstruction(req.Language)
		if langInstruction != "" {
			message = langInstruction + "\n\n" + req.Message
		}
	}

	// Run agentic chat with tool execution feedback
	err := s.aiClient.AskWithToolsAndExecution(r.Context(), message, func(text string) {
		escaped := strings.ReplaceAll(text, "\n", "\\n")
		sse.Write(escaped)
	}, toolApprovalCallback, toolExecutionCallback)

	if err != nil {
		apiErr := ParseLLMError(err, s.cfg.LLM.Provider)
		sse.Write(fmt.Sprintf("[ERROR] %s - %s", apiErr.Message, apiErr.Suggestion))
	}

	sse.Write("[DONE]")
}

// handleAgenticChatJSONMode handles AI chat using JSON mode fallback for models without tool calling
func (s *Server) handleAgenticChatJSONMode(w http.ResponseWriter, r *http.Request, req ChatRequest, username string) {
	// Set up SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, NewAPIError(ErrCodeInternalError, "Streaming not supported"))
		return
	}

	sse := &SSEWriter{w: w, flusher: flusher}

	// Get language instruction if needed
	langInstruction := ""
	if req.Language != "" && req.Language != "en" {
		langInstruction = getLanguageInstruction(req.Language)
	}

	// Create a prompt that instructs the LLM to respond in JSON format
	langPart := ""
	if langInstruction != "" {
		langPart = " " + langInstruction
	}
	jsonModePrompt := fmt.Sprintf(`You are a Kubernetes assistant.%s

The user's question is: "%s"

If you need to execute a kubectl command to answer the question, respond ONLY with a JSON object in this exact format:
{
  "action": "execute_command",
  "command": "kubectl <your command here>",
  "explanation": "Brief explanation of what this command does"
}

If you can answer the question directly without executing a command, respond ONLY with:
{
  "action": "direct_answer",
  "answer": "Your answer here"
}

IMPORTANT: Your response must be valid JSON only. No markdown, no extra text.`, langPart, req.Message)

	// Get response from LLM
	response, err := s.aiClient.AskNonStreaming(r.Context(), jsonModePrompt)
	if err != nil {
		apiErr := ParseLLMError(err, s.cfg.LLM.Provider)
		sse.Write(fmt.Sprintf("[ERROR] %s - %s", apiErr.Message, apiErr.Suggestion))
		sse.Write("[DONE]")
		return
	}

	// Clean up response (remove markdown code blocks if present)
	cleanResponse := strings.TrimSpace(response)
	cleanResponse = strings.TrimPrefix(cleanResponse, "```json")
	cleanResponse = strings.TrimPrefix(cleanResponse, "```")
	cleanResponse = strings.TrimSuffix(cleanResponse, "```")
	cleanResponse = strings.TrimSpace(cleanResponse)

	// Try to parse JSON response - support multiple formats
	var jsonResponse struct {
		Action      string `json:"action"`
		Command     string `json:"command"`
		Explanation string `json:"explanation"`
		Answer      string `json:"answer"`
		Thought     string `json:"thought"` // Alternative format
	}

	if err := json.Unmarshal([]byte(cleanResponse), &jsonResponse); err != nil {
		// If JSON parsing fails, just return the response as-is (plain text)
		sse.Write(response)
		sse.Write("[DONE]")
		return
	}

	// Handle different response formats
	// Format 1: {thought, answer} - extract just the answer
	if jsonResponse.Answer != "" && jsonResponse.Action == "" {
		sse.Write(jsonResponse.Answer)
		sse.Write("[DONE]")
		return
	}

	switch jsonResponse.Action {
	case "execute_command":
		// Send approval request
		category := classifyCommand(jsonResponse.Command)

		if category == "read-only" {
			// Auto-execute read-only commands
			sse.Write(fmt.Sprintf("[Executing] %s\n", jsonResponse.Command))
			// In a real implementation, you would execute the command here
			// For now, we just inform the user
			sse.Write(fmt.Sprintf("[Note] JSON mode cannot execute commands automatically. Please run: %s\n", jsonResponse.Command))
		} else {
			// Require approval for write commands
			approvalJSON, _ := json.Marshal(map[string]interface{}{
				"type":        "json_mode_command",
				"command":     jsonResponse.Command,
				"category":    category,
				"explanation": jsonResponse.Explanation,
				"note":        "JSON mode: Command execution requires manual approval. Please copy and run the command if you approve.",
			})
			sse.WriteEvent("json_command", string(approvalJSON))
			sse.Write(fmt.Sprintf("\n**Suggested Command (%s):**\n```\n%s\n```\n%s", category, jsonResponse.Command, jsonResponse.Explanation))
		}

	case "direct_answer":
		sse.Write(jsonResponse.Answer)

	default:
		// Unknown action - if there's an answer field, use it; otherwise return original
		if jsonResponse.Answer != "" {
			sse.Write(jsonResponse.Answer)
		} else {
			sse.Write(response)
		}
	}

	sse.Write("[DONE]")
}

// handleToolApprove handles user approval/rejection of tool calls
// handleSimpleChatMode handles AI chat using simple streaming mode (no tool execution)
// This is used when the model doesn't support tool calling and JSON mode is disabled.
func (s *Server) handleSimpleChatMode(w http.ResponseWriter, r *http.Request, req ChatRequest, username string) {
	// Set up SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		WriteError(w, NewAPIError(ErrCodeInternalError, "Streaming not supported"))
		return
	}

	sse := &SSEWriter{w: w, flusher: flusher}

	// Get language instruction if needed
	langInstruction := ""
	if req.Language != "" && req.Language != "en" {
		langInstruction = getLanguageInstruction(req.Language)
	}

	// Create a helpful prompt for Kubernetes assistance
	langPart := ""
	if langInstruction != "" {
		langPart = " " + langInstruction
	}

	systemPrompt := fmt.Sprintf(`You are a helpful Kubernetes assistant.%s

You can help users understand Kubernetes concepts, explain resources, suggest kubectl commands, and provide guidance.

IMPORTANT: You cannot execute commands directly. If the user asks you to perform an action, provide the kubectl command they should run manually.

When suggesting commands, format them clearly so users can copy and paste them.`, langPart)

	// Build the full prompt
	fullPrompt := fmt.Sprintf("%s\n\nUser: %s", systemPrompt, req.Message)

	// Use streaming if available
	err := s.aiClient.Ask(r.Context(), fullPrompt, func(chunk string) {
		sse.Write(chunk)
	})

	if err != nil {
		apiErr := ParseLLMError(err, s.cfg.LLM.Provider)
		sse.Write(fmt.Sprintf("\n\n[ERROR] %s", apiErr.Message))
	}

	sse.Write("[DONE]")
}

func (s *Server) handleToolApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		ID       string `json:"id"`
		Approved bool   `json:"approved"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	s.pendingApprovalMutex.RLock()
	approval, exists := s.pendingApprovals[req.ID]
	s.pendingApprovalMutex.RUnlock()

	if !exists {
		WriteError(w, NewAPIError(ErrCodeNotFound, "Approval not found or expired"))
		return
	}

	// Send response (non-blocking)
	select {
	case approval.Response <- req.Approved:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		WriteError(w, NewAPIError(ErrCodeConflict, "Approval already processed"))
	}
}

// getLLMCapabilities returns the capabilities of the configured LLM
func getLLMCapabilities(client *ai.Client, provider string) LLMCapabilities {
	caps := LLMCapabilities{
		Streaming: true, // Most providers support streaming
	}

	if client == nil {
		return caps
	}

	caps.ToolCalling = client.SupportsTools()

	// Determine JSON mode support based on provider
	switch provider {
	case "openai":
		caps.JSONMode = true
		caps.MaxTokens = 128000 // GPT-4 turbo
		if !caps.ToolCalling {
			caps.Recommendation = "Consider using GPT-4 or GPT-3.5-turbo for full tool calling support"
		}
	case "anthropic":
		caps.JSONMode = true
		caps.MaxTokens = 200000 // Claude 3
		if !caps.ToolCalling {
			caps.Recommendation = "Consider using Claude 3 Opus, Sonnet, or Haiku for tool calling support"
		}
	case "gemini":
		caps.JSONMode = true
		caps.MaxTokens = 32000
		if !caps.ToolCalling {
			caps.Recommendation = "Consider using Gemini Pro for tool calling support"
		}
	case "ollama":
		caps.JSONMode = true
		caps.Recommendation = "Ollama models vary in capabilities. Try llama3, mistral, or codellama for better results"
	case "bedrock":
		caps.JSONMode = true
		caps.Recommendation = "AWS Bedrock capabilities depend on the selected model"
	case "azopenai":
		caps.JSONMode = true
		caps.Recommendation = "Azure OpenAI capabilities depend on the deployed model"
	default:
		caps.JSONMode = false
		caps.Recommendation = "Unknown provider - capabilities may be limited"
	}

	return caps
}

// WriteEvent writes an SSE event with a specific event type
func (s *SSEWriter) WriteEvent(event string, data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, data)
	if err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

// handleAIPing checks if the AI client is configured and can connect
func (s *Server) handleAIPing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.aiClient == nil {
		WriteError(w, NewAPIError(ErrCodeLLMNotConfigured, "AI client not configured"))
		return
	}

	// Simple ping - just check if the client exists and is configured
	// A more thorough check could send a minimal request to the LLM
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"provider": s.cfg.LLM.Provider,
		"model":    s.cfg.LLM.Model,
	})
}

// getLanguageInstruction returns the language instruction for the given language code
func getLanguageInstruction(lang string) string {
	switch lang {
	case "ko":
		return "IMPORTANT: You MUST respond in Korean (한국어). All explanations, descriptions, and conversations should be in Korean. Technical terms and commands can remain in English, but all other text must be in Korean."
	case "zh":
		return "IMPORTANT: You MUST respond in Chinese (中文). All explanations, descriptions, and conversations should be in Chinese. Technical terms and commands can remain in English, but all other text must be in Chinese."
	case "ja":
		return "IMPORTANT: You MUST respond in Japanese (日本語). All explanations, descriptions, and conversations should be in Japanese. Technical terms and commands can remain in English, but all other text must be in Japanese."
	default:
		return ""
	}
}

// ==================== Ollama Helper Endpoints ====================

// OllamaStatusResponse represents the response from Ollama status check
type OllamaStatusResponse struct {
	Running bool                     `json:"running"`
	Models  []map[string]interface{} `json:"models"`
	Error   string                   `json:"error,omitempty"`
}

// handleOllamaStatus checks if Ollama is running and lists available models
func (s *Server) handleOllamaStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Try to connect to Ollama's default endpoint
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")

	if err != nil {
		json.NewEncoder(w).Encode(OllamaStatusResponse{
			Running: false,
			Error:   "Ollama not running or not accessible",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		json.NewEncoder(w).Encode(OllamaStatusResponse{
			Running: false,
			Error:   fmt.Sprintf("Ollama returned status %d", resp.StatusCode),
		})
		return
	}

	var tagsResponse struct {
		Models []map[string]interface{} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		json.NewEncoder(w).Encode(OllamaStatusResponse{
			Running: true,
			Models:  []map[string]interface{}{},
			Error:   "Failed to parse model list",
		})
		return
	}

	json.NewEncoder(w).Encode(OllamaStatusResponse{
		Running: true,
		Models:  tagsResponse.Models,
	})
}

// OllamaPullRequest represents a request to pull a model
type OllamaPullRequest struct {
	Model string `json:"model"`
}

// handleOllamaPull pulls a model from Ollama registry
func (s *Server) handleOllamaPull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req OllamaPullRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	if req.Model == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Model name is required",
		})
		return
	}

	// Send pull request to Ollama
	client := &http.Client{Timeout: 10 * time.Minute} // Model pull can take a while

	pullBody, _ := json.Marshal(map[string]interface{}{
		"name":   req.Model,
		"stream": false,
	})

	resp, err := client.Post("http://localhost:11434/api/pull", "application/json", strings.NewReader(string(pullBody)))
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("Failed to connect to Ollama: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("Ollama returned status %d", resp.StatusCode),
		})
		return
	}

	// Read response (Ollama returns progress updates)
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Even if we can't parse, the pull might have succeeded
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Model %s pull initiated", req.Model),
		})
		return
	}

	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": errMsg,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Model %s pulled successfully", req.Model),
	})
}

// SafetyAnalysisRequest represents a request to analyze K8s command safety
type SafetyAnalysisRequest struct {
	Command   string `json:"command"`             // The command or action to analyze
	Context   string `json:"context,omitempty"`   // Additional context (e.g., namespace, resource)
	Namespace string `json:"namespace,omitempty"` // Target namespace
}

// SafetyAnalysisResponse represents the safety analysis result
type SafetyAnalysisResponse struct {
	Safe             bool     `json:"safe"`              // Overall safety assessment
	RiskLevel        string   `json:"risk_level"`        // safe, warning, dangerous, critical
	RequiresApproval bool     `json:"requires_approval"` // Whether user confirmation is needed
	Warnings         []string `json:"warnings"`          // List of warning messages
	Recommendations  []string `json:"recommendations"`   // Suggested alternatives or precautions
	Category         string   `json:"category"`          // read-only, write, delete, admin
	AffectedScope    string   `json:"affected_scope"`    // pod, namespace, cluster
	Explanation      string   `json:"explanation"`       // Human-readable explanation
}

// handleSafetyAnalysis analyzes K8s commands/actions for safety
func (s *Server) handleSafetyAnalysis(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req SafetyAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
		return
	}

	if req.Command == "" {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Command is required"))
		return
	}

	// Analyze the command
	response := analyzeK8sSafety(req)

	json.NewEncoder(w).Encode(response)
}

// analyzeK8sSafety performs comprehensive K8s safety analysis
func analyzeK8sSafety(req SafetyAnalysisRequest) SafetyAnalysisResponse {
	cmd := strings.ToLower(req.Command)
	response := SafetyAnalysisResponse{
		Safe:             true,
		RiskLevel:        "safe",
		RequiresApproval: false,
		Warnings:         []string{},
		Recommendations:  []string{},
		Category:         "read-only",
		AffectedScope:    "pod",
	}

	// Critical operations - cluster-wide impact
	criticalPatterns := []struct {
		pattern     string
		explanation string
		scope       string
	}{
		{"delete namespace", "Deleting a namespace removes ALL resources within it permanently", "namespace"},
		{"delete ns ", "Deleting a namespace removes ALL resources within it permanently", "namespace"},
		{"delete all", "Deleting all resources can cause severe service disruption", "namespace"},
		{"--all-namespaces", "Operation affects ALL namespaces in the cluster", "cluster"},
		{"-A ", "Operation affects ALL namespaces in the cluster", "cluster"},
		{"drain node", "Draining a node evicts all pods and can cause service disruption", "cluster"},
		{"cordon node", "Cordoning prevents new pods from scheduling on the node", "cluster"},
		{"delete node", "Deleting a node removes it from the cluster", "cluster"},
		{"delete pv ", "Deleting PersistentVolumes can cause permanent data loss", "cluster"},
		{"delete pvc ", "Deleting PersistentVolumeClaims can cause data loss", "namespace"},
		{"delete clusterrole", "Deleting ClusterRoles affects cluster-wide permissions", "cluster"},
		{"delete clusterrolebinding", "Deleting ClusterRoleBindings affects cluster-wide access", "cluster"},
		{"--force --grace-period=0", "Force deletion bypasses graceful termination", "pod"},
		{"rm -rf", "Recursive file deletion is dangerous", "pod"},
		{"kubectl exec", "Executing commands in pods requires caution", "pod"},
	}

	// Dangerous operations
	dangerousPatterns := []struct {
		pattern     string
		explanation string
		category    string
	}{
		{"delete deployment", "Deleting deployments stops all associated pods", "delete"},
		{"delete statefulset", "Deleting StatefulSets can cause data inconsistency", "delete"},
		{"delete daemonset", "Deleting DaemonSets stops pods on all nodes", "delete"},
		{"delete service", "Deleting services breaks network connectivity", "delete"},
		{"delete ingress", "Deleting ingress rules breaks external access", "delete"},
		{"delete secret", "Deleting secrets can break dependent applications", "delete"},
		{"delete configmap", "Deleting ConfigMaps can break dependent applications", "delete"},
		{"scale --replicas=0", "Scaling to zero stops all pods", "write"},
		{"rollout undo", "Rolling back can introduce previous bugs", "write"},
		{"patch ", "Patching resources modifies their configuration", "write"},
		{"edit ", "Editing resources modifies their configuration", "write"},
		{"replace ", "Replacing resources can cause downtime", "write"},
	}

	// Warning operations
	warningPatterns := []struct {
		pattern     string
		explanation string
		category    string
	}{
		{"delete pod", "Deleting pods causes temporary unavailability", "delete"},
		{"delete job", "Deleting jobs stops running tasks", "delete"},
		{"scale ", "Scaling changes the number of running pods", "write"},
		{"rollout restart", "Restarting causes temporary pod unavailability", "write"},
		{"apply ", "Applying changes modifies cluster state", "write"},
		{"create ", "Creating new resources modifies cluster state", "write"},
		{"label ", "Labeling can affect service selectors", "write"},
		{"annotate ", "Annotations can affect controller behavior", "write"},
		{"taint ", "Taints affect pod scheduling", "admin"},
	}

	// Read-only operations (safe)
	readOnlyPatterns := []string{
		"get ", "describe ", "logs ", "top ", "explain ",
		"api-resources", "api-versions", "cluster-info",
		"auth can-i", "config view", "version",
	}

	// Check read-only first
	isReadOnly := false
	for _, pattern := range readOnlyPatterns {
		if strings.Contains(cmd, pattern) {
			isReadOnly = true
			break
		}
	}

	if isReadOnly && !strings.Contains(cmd, "delete") && !strings.Contains(cmd, "apply") {
		response.Explanation = "This is a read-only operation that does not modify the cluster"
		return response
	}

	// Check critical patterns
	for _, p := range criticalPatterns {
		if strings.Contains(cmd, p.pattern) {
			response.Safe = false
			response.RiskLevel = "critical"
			response.RequiresApproval = true
			response.Category = "admin"
			response.AffectedScope = p.scope
			response.Warnings = append(response.Warnings, p.explanation)
			response.Explanation = "CRITICAL: This operation has severe cluster-wide impact and could cause service disruption or data loss"
		}
	}

	// Check dangerous patterns
	if response.RiskLevel != "critical" {
		for _, p := range dangerousPatterns {
			if strings.Contains(cmd, p.pattern) {
				response.Safe = false
				response.RiskLevel = "dangerous"
				response.RequiresApproval = true
				response.Category = p.category
				response.AffectedScope = "namespace"
				response.Warnings = append(response.Warnings, p.explanation)
				response.Explanation = "This operation modifies or deletes resources and should be performed with caution"
			}
		}
	}

	// Check warning patterns
	if response.RiskLevel == "safe" {
		for _, p := range warningPatterns {
			if strings.Contains(cmd, p.pattern) {
				response.RiskLevel = "warning"
				response.RequiresApproval = true
				response.Category = p.category
				response.AffectedScope = "namespace"
				response.Warnings = append(response.Warnings, p.explanation)
				response.Explanation = "This operation modifies cluster state - review before proceeding"
			}
		}
	}

	// Add namespace-specific warnings
	if req.Namespace != "" {
		sensitiveNamespaces := []string{"kube-system", "kube-public", "kube-node-lease", "default"}
		for _, ns := range sensitiveNamespaces {
			if req.Namespace == ns {
				response.Warnings = append(response.Warnings, fmt.Sprintf("Operating on sensitive namespace '%s'", ns))
				if response.RiskLevel == "warning" {
					response.RiskLevel = "dangerous"
				}
				break
			}
		}
	}

	// Check for production indicators in the command
	productionIndicators := []string{"prod", "production", "live", "main", "master"}
	for _, indicator := range productionIndicators {
		if strings.Contains(cmd, indicator) || (req.Namespace != "" && strings.Contains(req.Namespace, indicator)) {
			response.Warnings = append(response.Warnings, "Possible production environment detected - extra caution recommended")
			response.RequiresApproval = true
			break
		}
	}

	// Add recommendations based on risk level
	switch response.RiskLevel {
	case "critical":
		response.Recommendations = append(response.Recommendations,
			"Consider using --dry-run=client first to preview changes",
			"Ensure you have recent backups before proceeding",
			"Verify you're operating on the correct cluster context",
			"Consider scheduling this during a maintenance window",
		)
	case "dangerous":
		response.Recommendations = append(response.Recommendations,
			"Use --dry-run=client to preview the operation",
			"Verify the target namespace and resources",
			"Consider backing up affected resources first",
		)
	case "warning":
		response.Recommendations = append(response.Recommendations,
			"Review the affected resources before proceeding",
			"Consider using --dry-run=client for verification",
		)
	}

	return response
}
