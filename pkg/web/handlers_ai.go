package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai"
	"github.com/cloudbro-kube-ai/k13d/pkg/ai/session"
	"github.com/cloudbro-kube-ai/k13d/pkg/config"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
	"github.com/cloudbro-kube-ai/k13d/pkg/log"
)

// LLMCapabilities represents the capabilities of the configured LLM
type LLMCapabilities struct {
	ToolCalling    bool   `json:"tool_calling"`
	JSONMode       bool   `json:"json_mode"`
	Streaming      bool   `json:"streaming"`
	MaxTokens      int    `json:"max_tokens,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

// handleAgentSettings handles agent loop settings (GET/PUT)
func (s *Server) handleAgentSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		s.aiMu.RLock()
		settings := map[string]interface{}{
			"max_iterations":   s.cfg.LLM.MaxIterations,
			"reasoning_effort": s.cfg.LLM.ReasoningEffort,
			"temperature":      s.cfg.LLM.Temperature,
			"max_tokens":       s.cfg.LLM.MaxTokens,
		}
		s.aiMu.RUnlock()
		_ = json.NewEncoder(w).Encode(settings)

	case http.MethodPut:
		role := r.Header.Get("X-User-Role")
		if role != "admin" {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		var req struct {
			MaxIterations   int     `json:"max_iterations"`
			ReasoningEffort string  `json:"reasoning_effort"`
			Temperature     float64 `json:"temperature"`
			MaxTokens       int     `json:"max_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
			return
		}

		// Clamp values to safe ranges
		if req.MaxIterations < 1 {
			req.MaxIterations = 1
		}
		if req.MaxIterations > 30 {
			req.MaxIterations = 30
		}
		if req.Temperature < 0 {
			req.Temperature = 0
		}
		if req.Temperature > 2.0 {
			req.Temperature = 2.0
		}
		if req.MaxTokens < 0 {
			req.MaxTokens = 0
		}

		s.aiMu.Lock()
		s.cfg.LLM.MaxIterations = req.MaxIterations
		s.cfg.LLM.Temperature = req.Temperature
		s.cfg.LLM.MaxTokens = req.MaxTokens
		if req.ReasoningEffort == "low" || req.ReasoningEffort == "medium" || req.ReasoningEffort == "high" {
			s.cfg.LLM.ReasoningEffort = req.ReasoningEffort
		}
		s.aiMu.Unlock()

		// Save to YAML config
		if err := s.cfg.Save(); err != nil {
			fmt.Printf("Warning: failed to save agent settings to YAML: %v\n", err)
		}

		// Persist to SQLite for web UI persistence
		if err := db.SaveWebSettings(map[string]string{
			"agent.max_iterations":   fmt.Sprintf("%d", req.MaxIterations),
			"agent.reasoning_effort": req.ReasoningEffort,
			"agent.temperature":      fmt.Sprintf("%.2f", req.Temperature),
			"agent.max_tokens":       fmt.Sprintf("%d", req.MaxTokens),
		}); err != nil {
			fmt.Printf("Warning: failed to save agent settings to SQLite: %v\n", err)
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "update_agent_settings",
			Resource: "settings",
			Details:  fmt.Sprintf("MaxIterations: %d, Temperature: %.2f, MaxTokens: %d, ReasoningEffort: %s", req.MaxIterations, req.Temperature, req.MaxTokens, req.ReasoningEffort),
		})

		_ = json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
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

		// Update LLM settings (protected by mutex)
		s.aiMu.Lock()
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
			s.aiMu.Unlock()
			apiErr := ParseLLMError(err, llmSettings.Provider)
			WriteError(w, apiErr)
			return
		}
		s.aiClient = newClient
		s.aiMu.Unlock()

		// Save to YAML
		if err := s.cfg.Save(); err != nil {
			WriteError(w, NewAPIError(ErrCodeInternalError, "Failed to save settings"))
			return
		}

		// Also persist LLM settings to SQLite for web UI persistence
		llmDBSettings := map[string]string{
			"llm.provider":         llmSettings.Provider,
			"llm.model":            llmSettings.Model,
			"llm.endpoint":         llmSettings.Endpoint,
			"llm.use_json_mode":    fmt.Sprintf("%v", llmSettings.UseJSONMode),
			"llm.reasoning_effort": llmSettings.ReasoningEffort,
		}
		if llmSettings.APIKey != "" {
			llmDBSettings["llm.api_key"] = llmSettings.APIKey
		}
		if err := db.SaveWebSettings(llmDBSettings); err != nil {
			fmt.Printf("Warning: failed to save LLM settings to SQLite: %v\n", err)
		}

		// Record audit
		username := r.Header.Get("X-Username")
		_ = db.RecordAudit(db.AuditEntry{
			User:     username,
			Action:   "update_llm_settings",
			Resource: "settings",
			Details:  fmt.Sprintf("Provider: %s, Model: %s, JSONMode: %v", llmSettings.Provider, llmSettings.Model, llmSettings.UseJSONMode),
		})

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "ok",
			"message":        "LLM settings updated successfully",
			"provider":       s.cfg.LLM.Provider,
			"model":          s.cfg.LLM.Model,
			"endpoint":       s.aiClient.GetEndpoint(),
			"ready":          s.aiClient.IsReady(),
			"supports_tools": s.aiClient.SupportsTools(),
		})

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
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
		"connected":        status.Connected,
		"provider":         status.Provider,
		"model":            status.Model,
		"endpoint":         status.Endpoint,
		"response_time_ms": status.ResponseTime,
		"capabilities":     capabilities,
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
	_ = db.RecordAudit(db.AuditEntry{
		User:     username,
		Action:   "llm_connection_test",
		Resource: "llm",
		Details:  fmt.Sprintf("Provider: %s, Model: %s, Result: %s, ToolCalling: %v", status.Provider, status.Model, resultText, capabilities.ToolCalling),
	})

	_ = json.NewEncoder(w).Encode(response)
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

	_ = json.NewEncoder(w).Encode(status)
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
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			// If JSON parsing fails, proceed with empty args
			args = make(map[string]interface{})
		}

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
		_ = sse.WriteEvent("approval", string(approvalJSON))

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

		case <-time.After(s.getToolApprovalTimeout()):
			// Timeout - cleanup and reject
			s.pendingApprovalMutex.Lock()
			delete(s.pendingApprovals, approvalID)
			s.pendingApprovalMutex.Unlock()

			_ = sse.WriteEvent("approval_timeout", approvalID)
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
		_ = sse.WriteEvent("tool_execution", string(execJSON))

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

	// Build message with conversation history and language instruction
	message := req.Message
	var currentSessionID string

	// Handle session-based conversation
	log.Debugf("[Session] sessionStore=%v, req.SessionID=%q", s.sessionStore != nil, req.SessionID)
	if s.sessionStore != nil {
		if req.SessionID != "" {
			currentSessionID = req.SessionID
			// Load conversation history
			history, err := s.sessionStore.GetContextMessages(req.SessionID, 20)
			log.Debugf("[Session] Loaded %d messages from session %s (err=%v)", len(history), req.SessionID, err)
			if err == nil && len(history) > 0 {
				var historyBuilder strings.Builder
				historyBuilder.WriteString("IMPORTANT: This is a continuation of an ongoing conversation. You MUST maintain context from the previous messages below.\n\n")
				historyBuilder.WriteString("=== CONVERSATION HISTORY ===\n")
				for i, msg := range history {
					if msg.Role == "user" {
						historyBuilder.WriteString(fmt.Sprintf("[%d] USER: %s\n", i+1, msg.Content))
					} else if msg.Role == "assistant" {
						historyBuilder.WriteString(fmt.Sprintf("[%d] ASSISTANT: %s\n", i+1, msg.Content))
					}
				}
				historyBuilder.WriteString("=== END OF HISTORY ===\n\n")
				historyBuilder.WriteString("Now respond to the user's NEW message below. Remember all context from the conversation history above.\n\n")
				historyBuilder.WriteString("NEW USER MESSAGE: ")
				message = historyBuilder.String() + req.Message
			}
		} else {
			// Create new session
			newSession, err := s.sessionStore.Create(s.cfg.LLM.Provider, s.cfg.LLM.Model)
			log.Debugf("[Session] Created new session: %s (err=%v)", newSession.ID, err)
			if err == nil {
				currentSessionID = newSession.ID
				// Send session ID to client
				sessionJSON, _ := json.Marshal(map[string]string{"session_id": currentSessionID})
				_ = sse.WriteEvent("session", string(sessionJSON))
				log.Debugf("[Session] Sent session ID to client: %s", currentSessionID)
			}
		}

		// Save user message to session
		if currentSessionID != "" {
			err := s.sessionStore.AddMessage(currentSessionID, session.Message{
				Role:      "user",
				Content:   req.Message,
				Timestamp: time.Now(),
			})
			log.Debugf("[Session] Saved user message to session %s (err=%v)", currentSessionID, err)
		}
	}

	// Add language instruction if needed
	if req.Language != "" && req.Language != "en" {
		langInstruction := getLanguageInstruction(req.Language)
		if langInstruction != "" {
			message = langInstruction + "\n\n" + message
		}
	}

	// Collect response for session storage
	var responseBuilder strings.Builder

	// Run agentic chat with tool execution feedback
	err := s.aiClient.AskWithToolsAndExecution(r.Context(), message, func(text string) {
		responseBuilder.WriteString(text)
		escaped := strings.ReplaceAll(text, "\n", "\\n")
		_ = sse.Write(escaped)
	}, toolApprovalCallback, toolExecutionCallback)

	if err != nil {
		apiErr := ParseLLMError(err, s.cfg.LLM.Provider)
		_ = sse.Write(fmt.Sprintf("[ERROR] %s - %s", apiErr.Message, apiErr.Suggestion))
	}

	// Save assistant response to session
	if s.sessionStore != nil && currentSessionID != "" {
		responseContent := responseBuilder.String()
		err := s.sessionStore.AddMessage(currentSessionID, session.Message{
			Role:      "assistant",
			Content:   responseContent,
			Timestamp: time.Now(),
		})
		log.Debugf("[Session] Saved assistant message to session %s (len=%d, err=%v)", currentSessionID, len(responseContent), err)
	}

	_ = sse.Write("[DONE]")
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
		_ = sse.Write(fmt.Sprintf("[ERROR] %s - %s", apiErr.Message, apiErr.Suggestion))
		_ = sse.Write("[DONE]")
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
		_ = sse.Write(response)
		_ = sse.Write("[DONE]")
		return
	}

	// Handle different response formats
	// Format 1: {thought, answer} - extract just the answer
	if jsonResponse.Answer != "" && jsonResponse.Action == "" {
		_ = sse.Write(jsonResponse.Answer)
		_ = sse.Write("[DONE]")
		return
	}

	switch jsonResponse.Action {
	case "execute_command":
		// Send approval request
		category := classifyCommand(jsonResponse.Command)

		if category == "read-only" {
			// Auto-execute read-only commands
			_ = sse.Write(fmt.Sprintf("[Executing] %s\n", jsonResponse.Command))
			// In a real implementation, you would execute the command here
			// For now, we just inform the user
			_ = sse.Write(fmt.Sprintf("[Note] JSON mode cannot execute commands automatically. Please run: %s\n", jsonResponse.Command))
		} else {
			// Require approval for write commands
			approvalJSON, _ := json.Marshal(map[string]interface{}{
				"type":        "json_mode_command",
				"command":     jsonResponse.Command,
				"category":    category,
				"explanation": jsonResponse.Explanation,
				"note":        "JSON mode: Command execution requires manual approval. Please copy and run the command if you approve.",
			})
			_ = sse.WriteEvent("json_command", string(approvalJSON))
			_ = sse.Write(fmt.Sprintf("\n**Suggested Command (%s):**\n```\n%s\n```\n%s", category, jsonResponse.Command, jsonResponse.Explanation))
		}

	case "direct_answer":
		_ = sse.Write(jsonResponse.Answer)

	default:
		// Unknown action - if there's an answer field, use it; otherwise return original
		if jsonResponse.Answer != "" {
			_ = sse.Write(jsonResponse.Answer)
		} else {
			_ = sse.Write(response)
		}
	}

	_ = sse.Write("[DONE]")
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
		_ = sse.Write(chunk)
	})

	if err != nil {
		apiErr := ParseLLMError(err, s.cfg.LLM.Provider)
		_ = sse.Write(fmt.Sprintf("\n\n[ERROR] %s", apiErr.Message))
	}

	_ = sse.Write("[DONE]")
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
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
		if caps.ToolCalling {
			caps.Recommendation = "Full agentic AI features available with tool calling support"
		} else {
			caps.Recommendation = "Consider using GPT-4 or GPT-3.5-turbo for full tool calling support"
		}
	case "anthropic":
		caps.JSONMode = true
		caps.MaxTokens = 200000 // Claude 3
		if caps.ToolCalling {
			caps.Recommendation = "Full agentic AI features available with tool calling support"
		} else {
			caps.Recommendation = "Consider using Claude 3 Opus, Sonnet, or Haiku for tool calling support"
		}
	case "gemini":
		caps.JSONMode = true
		caps.MaxTokens = 32000
		if caps.ToolCalling {
			caps.Recommendation = "Full agentic AI features available with Gemini function calling"
		} else {
			caps.Recommendation = "Consider using Gemini Pro for tool calling support"
		}
	case "ollama":
		caps.JSONMode = true
		if caps.ToolCalling {
			caps.Recommendation = "Full agentic AI features available. Ollama tool calling enabled"
		} else {
			caps.Recommendation = "Ollama models vary in capabilities. Try llama3, mistral, or qwen2.5 for tool calling support"
		}
	case "bedrock":
		caps.JSONMode = true
		if caps.ToolCalling {
			caps.Recommendation = "Full agentic AI features available via AWS Bedrock"
		} else {
			caps.Recommendation = "AWS Bedrock capabilities depend on the selected model"
		}
	case "azopenai":
		caps.JSONMode = true
		if caps.ToolCalling {
			caps.Recommendation = "Full agentic AI features available via Azure OpenAI"
		} else {
			caps.Recommendation = "Azure OpenAI capabilities depend on the deployed model"
		}
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

// handleAvailableModels fetches available models from the current LLM provider.
// GET: uses the existing AI client.
// POST: accepts provider/api_key/endpoint in request body (avoids API key in URL).
func (s *Server) handleAvailableModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var provider, apiKey, endpoint string

	if r.Method == http.MethodPost {
		// POST: read credentials from request body (avoids API key in URL)
		var req struct {
			Provider string `json:"provider"`
			APIKey   string `json:"api_key"`
			Endpoint string `json:"endpoint"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			provider = req.Provider
			apiKey = req.APIKey
			endpoint = req.Endpoint
		}
	} else {
		// GET: only non-sensitive params from query (uses existing client)
		provider = r.URL.Query().Get("provider")
		endpoint = r.URL.Query().Get("endpoint")
	}

	var client *ai.Client
	if provider != "" && apiKey != "" {
		// Create temporary client with provided config
		tempConfig := config.LLMConfig{
			Provider: provider,
			Model:    "temp",
			Endpoint: endpoint,
			APIKey:   apiKey,
		}
		var err error
		client, err = ai.NewClient(&tempConfig)
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"models": []string{},
				"error":  fmt.Sprintf("Failed to create client: %v", err),
			})
			return
		}
	} else if s.aiClient != nil {
		client = s.aiClient
	} else {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []string{},
			"error":  "No AI client configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	models, err := client.ListModels(ctx)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"models": []string{},
			"error":  fmt.Sprintf("Failed to list models: %v", err),
		})
		return
	}

	// For Gemini, filter to only generateContent-capable models
	if provider == "gemini" || s.cfg.LLM.Provider == "gemini" {
		var filtered []string
		for _, m := range models {
			if strings.HasPrefix(m, "gemini-") {
				filtered = append(filtered, m)
			}
		}
		models = filtered
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"models": models,
	})
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
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
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

	// Use configured Ollama endpoint, falling back to default
	ollamaEndpoint := "http://localhost:11434"
	if s.cfg.LLM.Provider == "ollama" && s.cfg.LLM.Endpoint != "" {
		ollamaEndpoint = strings.TrimSuffix(s.cfg.LLM.Endpoint, "/")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ollamaEndpoint + "/api/tags")

	if err != nil {
		_ = json.NewEncoder(w).Encode(OllamaStatusResponse{
			Running: false,
			Error:   "Ollama not running or not accessible",
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_ = json.NewEncoder(w).Encode(OllamaStatusResponse{
			Running: false,
			Error:   fmt.Sprintf("Ollama returned status %d", resp.StatusCode),
		})
		return
	}

	var tagsResponse struct {
		Models []map[string]interface{} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tagsResponse); err != nil {
		_ = json.NewEncoder(w).Encode(OllamaStatusResponse{
			Running: true,
			Models:  []map[string]interface{}{},
			Error:   "Failed to parse model list",
		})
		return
	}

	_ = json.NewEncoder(w).Encode(OllamaStatusResponse{
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
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Invalid request body",
		})
		return
	}

	if req.Model == "" {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Model name is required",
		})
		return
	}

	// Use configured Ollama endpoint, falling back to default
	ollamaEndpoint := "http://localhost:11434"
	if s.cfg.LLM.Provider == "ollama" && s.cfg.LLM.Endpoint != "" {
		ollamaEndpoint = strings.TrimSuffix(s.cfg.LLM.Endpoint, "/")
	}

	client := &http.Client{Timeout: 10 * time.Minute} // Model pull can take a while

	pullBody, _ := json.Marshal(map[string]interface{}{
		"name":   req.Model,
		"stream": false,
	})

	resp, err := client.Post(ollamaEndpoint+"/api/pull", "application/json", strings.NewReader(string(pullBody)))
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("Failed to connect to Ollama: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("Ollama returned status %d", resp.StatusCode),
		})
		return
	}

	// Read response (Ollama returns progress updates)
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// Even if we can't parse, the pull might have succeeded
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Model %s pull initiated", req.Model),
		})
		return
	}

	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": errMsg,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
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

	_ = json.NewEncoder(w).Encode(response)
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

// ==================== Session Management Handlers ====================

// handleSessions handles session list and creation
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.sessionStore == nil {
		WriteError(w, NewAPIError(ErrCodeInternalError, "Session store not initialized"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		// List sessions
		sessions, err := s.sessionStore.List()
		if err != nil {
			WriteError(w, NewAPIError(ErrCodeInternalError, fmt.Sprintf("Failed to list sessions: %v", err)))
			return
		}
		_ = json.NewEncoder(w).Encode(sessions)

	case http.MethodPost:
		// Create new session
		newSession, err := s.sessionStore.Create(s.cfg.LLM.Provider, s.cfg.LLM.Model)
		if err != nil {
			WriteError(w, NewAPIError(ErrCodeInternalError, fmt.Sprintf("Failed to create session: %v", err)))
			return
		}
		_ = json.NewEncoder(w).Encode(newSession)

	case http.MethodDelete:
		// Clear all sessions
		if err := s.sessionStore.Clear(); err != nil {
			WriteError(w, NewAPIError(ErrCodeInternalError, fmt.Sprintf("Failed to clear sessions: %v", err)))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})

	default:
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleSession handles single session operations
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.sessionStore == nil {
		WriteError(w, NewAPIError(ErrCodeInternalError, "Session store not initialized"))
		return
	}

	// Extract session ID from URL path
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if sessionID == "" {
		WriteError(w, NewAPIError(ErrCodeBadRequest, "Session ID required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get session with messages
		sess, err := s.sessionStore.Get(sessionID)
		if err != nil {
			WriteError(w, NewAPIError(ErrCodeNotFound, fmt.Sprintf("Session not found: %v", err)))
			return
		}
		_ = json.NewEncoder(w).Encode(sess)

	case http.MethodDelete:
		// Delete session
		if err := s.sessionStore.Delete(sessionID); err != nil {
			WriteError(w, NewAPIError(ErrCodeNotFound, fmt.Sprintf("Failed to delete session: %v", err)))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

	case http.MethodPut:
		// Update session title
		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, NewAPIError(ErrCodeBadRequest, "Invalid request body"))
			return
		}
		if err := s.sessionStore.UpdateTitle(sessionID, req.Title); err != nil {
			WriteError(w, NewAPIError(ErrCodeNotFound, fmt.Sprintf("Failed to update session: %v", err)))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	default:
		WriteErrorSimple(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}
