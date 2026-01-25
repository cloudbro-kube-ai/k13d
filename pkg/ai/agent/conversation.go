package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/providers"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/safety"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/tools"
)

// systemPrompt is the default system prompt for the agent
const systemPrompt = `You are k13d, a Kubernetes AI assistant. You help users manage and troubleshoot Kubernetes clusters.

You have access to tools to execute kubectl commands and bash commands. Use these tools to help users.

Guidelines:
- Be concise and helpful
- When asked about Kubernetes resources, use kubectl to get real data
- Explain complex concepts in simple terms when asked
- Always verify the current state before making changes
- Warn users about potentially dangerous operations
- If a command fails, analyze the error and suggest solutions`

// Run starts the agent conversation loop.
// This is the main agentic loop that processes user messages,
// calls the LLM, handles tool calls, and manages approvals.
func (a *Agent) Run(ctx context.Context) error {
	a.runningMu.Lock()
	if a.running {
		a.runningMu.Unlock()
		return fmt.Errorf("agent is already running")
	}
	a.running = true
	a.runningMu.Unlock()

	defer func() {
		a.runningMu.Lock()
		a.running = false
		a.runningMu.Unlock()
	}()

	a.ctx, a.cancel = context.WithCancel(ctx)
	defer a.cancel()

	for {
		select {
		case <-a.ctx.Done():
			a.setState(StateIdle)
			return a.ctx.Err()
		default:
		}

		switch a.State() {
		case StateIdle:
			// Wait for user input
			select {
			case msg := <-a.Input:
				if msg.Type == MsgText {
					if err := a.handleUserMessage(msg.Content); err != nil {
						a.emitError(err)
						continue
					}
					a.setState(StateRunning)
				}
			case <-a.ctx.Done():
				return a.ctx.Err()
			}

		case StateRunning:
			// Call LLM and process response
			if err := a.callLLM(); err != nil {
				a.emitError(err)
				a.setState(StateError)
				continue
			}

		case StateToolAnalysis:
			// Analyze pending tool calls and decide on approval
			a.analyzeToolCalls()

		case StateWaitingForApproval:
			// Wait for user decision (non-blocking via channel)
			if err := a.waitForApproval(); err != nil {
				a.emitError(err)
				a.setState(StateError)
				continue
			}

		case StateDone:
			// Conversation turn complete, return to idle
			a.emitStreamEnd()
			a.setState(StateIdle)
			return nil

		case StateError:
			// Reset to idle after error
			a.setState(StateIdle)
			return nil
		}
	}
}

// Ask sends a single question and processes it through the agent loop.
// This is a convenience method for simple single-turn interactions.
func (a *Agent) Ask(ctx context.Context, question string) error {
	a.SendUserMessage(question)
	return a.Run(ctx)
}

// AskWithContext sends a question with additional context (resource info).
func (a *Agent) AskWithContext(ctx context.Context, question string, resourceContext string) error {
	fullQuestion := question
	if resourceContext != "" {
		fullQuestion = fmt.Sprintf("%s\n\nContext:\n%s", question, resourceContext)
	}
	return a.Ask(ctx, fullQuestion)
}

// handleUserMessage processes a user message
func (a *Agent) handleUserMessage(content string) error {
	if a.session == nil {
		// Auto-create session if not exists
		providerName := "unknown"
		modelName := "unknown"
		if a.provider != nil {
			providerName = a.provider.Name()
			modelName = a.provider.GetModel()
		}
		a.StartSession(providerName, modelName)
	}

	// Add to session history
	a.session.AddMessage("user", content)

	return nil
}

// callLLM calls the LLM provider and handles the response
func (a *Agent) callLLM() error {
	if a.provider == nil {
		return fmt.Errorf("no LLM provider configured")
	}

	// Build prompt with conversation history
	prompt := a.buildPromptWithHistory()

	// Streaming callback
	streamCallback := func(chunk string) {
		a.emitStreamChunk(chunk)
	}

	// Check if we have tool support
	if a.toolProvider != nil && a.toolRegistry != nil {
		// Use tool-enabled asking
		toolDefs := a.buildToolDefinitions()

		toolCallback := func(call providers.ToolCall) providers.ToolResult {
			return a.handleToolCall(call)
		}

		err := a.toolProvider.AskWithTools(a.ctx, prompt, toolDefs, streamCallback, toolCallback)
		if err != nil {
			return err
		}
	} else {
		// Basic asking without tools
		err := a.provider.Ask(a.ctx, prompt, streamCallback)
		if err != nil {
			return err
		}
	}

	a.setState(StateDone)
	return nil
}

// buildPromptWithHistory constructs the prompt including message history
func (a *Agent) buildPromptWithHistory() string {
	var sb strings.Builder

	// System prompt
	sb.WriteString(systemPrompt)
	sb.WriteString("\n\n")

	// Language instruction
	if a.language != "" && a.language != "en" {
		langInstruction := getLanguageInstruction(a.language)
		if langInstruction != "" {
			sb.WriteString(langInstruction)
			sb.WriteString("\n\n")
		}
	}

	// Previous messages (up to last N for context window management)
	if a.session != nil {
		messages := a.session.Messages
		start := 0
		maxHistory := 10
		if len(messages) > maxHistory {
			start = len(messages) - maxHistory
		}

		for _, msg := range messages[start:] {
			if msg.Role == "user" {
				sb.WriteString("User: ")
			} else {
				sb.WriteString("Assistant: ")
			}
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

// buildToolDefinitions converts registry tools to provider format
func (a *Agent) buildToolDefinitions() []providers.ToolDefinition {
	if a.toolRegistry == nil {
		return nil
	}

	registryTools := a.toolRegistry.List()
	defs := make([]providers.ToolDefinition, 0, len(registryTools))

	for _, tool := range registryTools {
		defs = append(defs, providers.ToolDefinition{
			Type: "function",
			Function: providers.FunctionDef{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	return defs
}

// handleToolCall processes a tool call from the LLM
func (a *Agent) handleToolCall(call providers.ToolCall) providers.ToolResult {
	// Convert to our internal format
	toolCall := &ToolCallInfo{
		ID:      call.ID,
		Name:    call.Function.Name,
		Command: call.Function.Arguments,
	}

	// Parse arguments to extract command
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err == nil {
		if cmd, ok := args["command"].(string); ok {
			toolCall.Command = cmd
		}
	}

	// Analyze safety using shell parser
	safetyReport := a.safetyAnalyzer.Analyze(toolCall.Command)
	toolCall.IsReadOnly = safetyReport.IsReadOnly
	toolCall.IsDangerous = safetyReport.IsDangerous
	toolCall.Warnings = safetyReport.Warnings

	// Emit tool call request to UI
	a.emitToolCallRequest(toolCall)

	// Check if approval is needed
	needsApproval := !toolCall.IsReadOnly || !a.autoApproveReadOnly

	if needsApproval {
		// Request approval
		approved := a.requestAndWaitForApproval(toolCall)
		if !approved {
			return providers.ToolResult{
				ToolCallID: call.ID,
				Content:    "Tool execution cancelled by user",
				IsError:    true,
			}
		}
	}

	// Execute the tool
	startTime := time.Now()
	registryCall := &tools.ToolCall{
		ID:   call.ID,
		Type: call.Type,
		Function: tools.ToolCallFunc{
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		},
	}

	result := a.toolRegistry.Execute(a.ctx, registryCall)
	executionDuration := time.Since(startTime)

	// Store result in tool call info
	toolCall.Result = result.Content
	toolCall.ExecutedAt = time.Now()
	toolCall.Approved = true

	// Record tool execution in session
	if a.session != nil {
		a.session.AddToolExecution(
			call.Function.Name,
			toolCall.Command,
			result.Content,
			true,
			executionDuration.Milliseconds(),
		)
	}

	// Emit result to UI
	a.emitToolCallCompleted(toolCall)

	return providers.ToolResult{
		ToolCallID: result.ToolCallID,
		Content:    result.Content,
		IsError:    result.IsError,
	}
}

// requestAndWaitForApproval requests user approval and waits for response
func (a *Agent) requestAndWaitForApproval(toolCall *ToolCallInfo) bool {
	// Create approval request
	choice := NewApprovalRequest(toolCall.ID, toolCall.Command, toolCall.IsDangerous)

	// Check if we have an approval handler (synchronous mode)
	a.approvalHandlerMu.RLock()
	handler := a.approvalHandler
	a.approvalHandlerMu.RUnlock()

	if handler != nil {
		// Use synchronous approval handler
		var approved bool
		done := make(chan struct{})
		handler.RequestApproval(choice, func(result bool) {
			approved = result
			close(done)
		})
		<-done
		return approved
	}

	// Fall back to channel-based async approval
	a.emitApprovalRequest(choice)

	// Wait for response with timeout
	timeout := time.NewTimer(a.approvalTimeout)
	defer timeout.Stop()

	for {
		select {
		case msg := <-a.Input:
			if msg.Type == MsgUserChoiceResponse {
				return msg.Content == "approved" || msg.Content == "approve"
			}
		case <-timeout.C:
			a.emitText("\n[Approval timeout - command cancelled]")
			return false
		case <-a.ctx.Done():
			return false
		}
	}
}

// analyzeToolCalls analyzes pending tool calls
func (a *Agent) analyzeToolCalls() {
	if len(a.pendingToolCalls) == 0 {
		a.setState(StateRunning)
		return
	}

	// Check if any need approval
	needsApproval := false
	for _, tc := range a.pendingToolCalls {
		if !tc.IsReadOnly || !a.autoApproveReadOnly {
			needsApproval = true
			break
		}
	}

	if needsApproval {
		a.setState(StateWaitingForApproval)
	} else {
		// Auto-execute read-only commands
		a.executeApprovedTools()
		a.setState(StateRunning)
	}
}

// waitForApproval waits for user approval decision
func (a *Agent) waitForApproval() error {
	timeout := time.NewTimer(a.approvalTimeout)
	defer timeout.Stop()

	select {
	case msg := <-a.Input:
		if msg.Type == MsgUserChoiceResponse {
			approved := msg.Content == "approved" || msg.Content == "approve"
			if approved {
				a.executeApprovedTools()
			} else {
				a.cancelPendingTools()
			}
			a.setState(StateRunning)
		}
	case <-timeout.C:
		a.emitText("\n[Approval timeout - commands cancelled]")
		a.cancelPendingTools()
		a.setState(StateDone)
	case <-a.ctx.Done():
		return a.ctx.Err()
	}

	return nil
}

// executeApprovedTools executes all pending approved tools
func (a *Agent) executeApprovedTools() {
	for _, tc := range a.pendingToolCalls {
		tc.Approved = true
		// Execution is handled in handleToolCall
	}
	a.pendingToolCalls = a.pendingToolCalls[:0]
}

// cancelPendingTools cancels all pending tool calls
func (a *Agent) cancelPendingTools() {
	for _, tc := range a.pendingToolCalls {
		tc.Approved = false
	}
	a.pendingToolCalls = a.pendingToolCalls[:0]
}

// isReadOnlyCommand checks if a command is read-only using safety analyzer
func (a *Agent) isReadOnlyCommand(cmd string) bool {
	isReadOnly, _ := safety.QuickCheck(cmd)
	return isReadOnly
}

// isDangerousCommand checks if a command is dangerous using safety analyzer
func (a *Agent) isDangerousCommand(cmd string) bool {
	_, isDangerous := safety.QuickCheck(cmd)
	return isDangerous
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

// SetLanguage sets the display language for the agent
func (a *Agent) SetLanguage(lang string) {
	a.language = lang
}
