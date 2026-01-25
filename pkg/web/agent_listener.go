package web

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/agent"
)

// SSEAgentListener implements agent.AgentListener for WebUI.
// It sends Server-Sent Events to the browser for each agent event.
type SSEAgentListener struct {
	writer *SSEWriter
	mu     sync.Mutex

	// Approval handling
	pendingApproval *agent.ChoiceRequest
	approvalChan    chan bool
}

// NewSSEAgentListener creates a new SSE-based agent listener
func NewSSEAgentListener(w *SSEWriter) *SSEAgentListener {
	return &SSEAgentListener{
		writer:       w,
		approvalChan: make(chan bool, 1),
	}
}

// Ensure SSEAgentListener implements AgentListener
var _ agent.AgentListener = (*SSEAgentListener)(nil)

// sendEvent sends an SSE event with the given type and data
func (l *SSEAgentListener) sendEvent(eventType string, data interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	// Format: event: <type>\ndata: <json>\n\n
	msg := fmt.Sprintf("event: %s\ndata: %s", eventType, string(jsonData))
	l.writer.Write(msg)
}

// AgentTextReceived handles text events
func (l *SSEAgentListener) AgentTextReceived(text string) {
	l.sendEvent("text", map[string]string{
		"content": text,
	})
}

// AgentStreamChunk handles streaming chunks
func (l *SSEAgentListener) AgentStreamChunk(chunk string) {
	l.sendEvent("chunk", map[string]string{
		"content": chunk,
	})
}

// AgentStreamEnd handles stream end events
func (l *SSEAgentListener) AgentStreamEnd() {
	l.sendEvent("stream_end", map[string]interface{}{})
}

// AgentError handles error events
func (l *SSEAgentListener) AgentError(err error) {
	l.sendEvent("error", map[string]string{
		"error": err.Error(),
	})
}

// AgentStateChanged handles state change events
func (l *SSEAgentListener) AgentStateChanged(state agent.State) {
	l.sendEvent("state", map[string]string{
		"state": state.String(),
	})
}

// AgentToolCallRequested handles tool call request events
func (l *SSEAgentListener) AgentToolCallRequested(tc *agent.ToolCallInfo) {
	l.sendEvent("tool_request", map[string]interface{}{
		"id":           tc.ID,
		"name":         tc.Name,
		"command":      tc.Command,
		"is_readonly":  tc.IsReadOnly,
		"is_dangerous": tc.IsDangerous,
		"warnings":     tc.Warnings,
	})
}

// AgentToolCallCompleted handles tool call completion events
func (l *SSEAgentListener) AgentToolCallCompleted(tc *agent.ToolCallInfo) {
	l.sendEvent("tool_result", map[string]interface{}{
		"id":      tc.ID,
		"name":    tc.Name,
		"command": tc.Command,
		"result":  tc.Result,
	})
}

// AgentApprovalRequested handles approval request events
func (l *SSEAgentListener) AgentApprovalRequested(choice *agent.ChoiceRequest) {
	l.mu.Lock()
	l.pendingApproval = choice
	l.mu.Unlock()

	l.sendEvent("approval", map[string]interface{}{
		"id":          choice.ID,
		"title":       choice.Title,
		"description": choice.Description,
		"command":     choice.Command,
	})
}

// AgentApprovalTimeout handles approval timeout events
func (l *SSEAgentListener) AgentApprovalTimeout(choiceID string) {
	l.mu.Lock()
	l.pendingApproval = nil
	l.mu.Unlock()

	l.sendEvent("approval_timeout", map[string]string{
		"id": choiceID,
	})
}

// HandleApproval processes an approval response from the browser
// Returns true if the approval was handled, false if there was no pending approval
func (l *SSEAgentListener) HandleApproval(choiceID string, approved bool) bool {
	l.mu.Lock()
	pending := l.pendingApproval
	l.mu.Unlock()

	if pending == nil || pending.ID != choiceID {
		return false
	}

	select {
	case l.approvalChan <- approved:
		return true
	default:
		return false
	}
}

// WaitForApproval waits for an approval response
// This is used when the agent is using the approval handler pattern
func (l *SSEAgentListener) WaitForApproval() bool {
	return <-l.approvalChan
}

// SSEApprovalHandler implements agent.AgentApprovalHandler for WebUI
type SSEApprovalHandler struct {
	listener *SSEAgentListener
}

// NewSSEApprovalHandler creates a new SSE-based approval handler
func NewSSEApprovalHandler(listener *SSEAgentListener) *SSEApprovalHandler {
	return &SSEApprovalHandler{listener: listener}
}

// Ensure SSEApprovalHandler implements AgentApprovalHandler
var _ agent.AgentApprovalHandler = (*SSEApprovalHandler)(nil)

// RequestApproval handles synchronous approval requests
func (h *SSEApprovalHandler) RequestApproval(choice *agent.ChoiceRequest, callback func(bool)) {
	// Store the pending approval
	h.listener.AgentApprovalRequested(choice)

	// Wait for response in a goroutine and call callback
	go func() {
		approved := h.listener.WaitForApproval()
		callback(approved)
	}()
}
