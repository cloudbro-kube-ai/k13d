package agent

import "sync"

// AgentListener defines the interface for receiving agent events.
// This follows the k9s listener pattern for loose coupling between
// the Agent (model) and UI components (view/ui).
//
// Both TUI and WebUI implement this interface to receive agent events.
type AgentListener interface {
	// Text events
	AgentTextReceived(text string)
	AgentStreamChunk(chunk string)
	AgentStreamEnd()
	AgentError(err error)

	// State events
	AgentStateChanged(state State)

	// Tool events
	AgentToolCallRequested(toolCall *ToolCallInfo)
	AgentToolCallCompleted(toolCall *ToolCallInfo)

	// Approval events
	AgentApprovalRequested(choice *ChoiceRequest)
	AgentApprovalTimeout(choiceID string)
}

// AgentApprovalHandler defines the interface for handling approval requests.
// UI components implement this to provide approval responses back to the agent.
type AgentApprovalHandler interface {
	// RequestApproval is called when the agent needs user approval.
	// The handler should display the approval UI and call the callback
	// with the user's decision.
	RequestApproval(choice *ChoiceRequest, callback func(approved bool))
}

// NullListener is a no-op implementation of AgentListener.
// Useful for testing or when no listener is needed.
type NullListener struct{}

func (NullListener) AgentTextReceived(string)              {}
func (NullListener) AgentStreamChunk(string)               {}
func (NullListener) AgentStreamEnd()                       {}
func (NullListener) AgentError(error)                      {}
func (NullListener) AgentStateChanged(State)               {}
func (NullListener) AgentToolCallRequested(*ToolCallInfo)  {}
func (NullListener) AgentToolCallCompleted(*ToolCallInfo)  {}
func (NullListener) AgentApprovalRequested(*ChoiceRequest) {}
func (NullListener) AgentApprovalTimeout(string)           {}

// MultiListener broadcasts events to multiple listeners.
// Follows k9s pattern for supporting multiple subscribers.
// Thread-safe for concurrent Add/Remove/broadcast operations.
type MultiListener struct {
	mu        sync.RWMutex
	listeners []AgentListener
}

// NewMultiListener creates a new MultiListener
func NewMultiListener() *MultiListener {
	return &MultiListener{
		listeners: make([]AgentListener, 0),
	}
}

// Add adds a listener (thread-safe)
func (m *MultiListener) Add(l AgentListener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, l)
}

// Remove removes a listener (thread-safe)
func (m *MultiListener) Remove(l AgentListener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, listener := range m.listeners {
		if listener == l {
			m.listeners = append(m.listeners[:i], m.listeners[i+1:]...)
			return
		}
	}
}

// getListeners returns a snapshot of listeners for safe iteration
func (m *MultiListener) getListeners() []AgentListener {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to avoid race during iteration
	snapshot := make([]AgentListener, len(m.listeners))
	copy(snapshot, m.listeners)
	return snapshot
}

// AgentTextReceived broadcasts to all listeners (thread-safe)
func (m *MultiListener) AgentTextReceived(text string) {
	for _, l := range m.getListeners() {
		l.AgentTextReceived(text)
	}
}

// AgentStreamChunk broadcasts to all listeners (thread-safe)
func (m *MultiListener) AgentStreamChunk(chunk string) {
	for _, l := range m.getListeners() {
		l.AgentStreamChunk(chunk)
	}
}

// AgentStreamEnd broadcasts to all listeners (thread-safe)
func (m *MultiListener) AgentStreamEnd() {
	for _, l := range m.getListeners() {
		l.AgentStreamEnd()
	}
}

// AgentError broadcasts to all listeners (thread-safe)
func (m *MultiListener) AgentError(err error) {
	for _, l := range m.getListeners() {
		l.AgentError(err)
	}
}

// AgentStateChanged broadcasts to all listeners (thread-safe)
func (m *MultiListener) AgentStateChanged(state State) {
	for _, l := range m.getListeners() {
		l.AgentStateChanged(state)
	}
}

// AgentToolCallRequested broadcasts to all listeners (thread-safe)
func (m *MultiListener) AgentToolCallRequested(toolCall *ToolCallInfo) {
	for _, l := range m.getListeners() {
		l.AgentToolCallRequested(toolCall)
	}
}

// AgentToolCallCompleted broadcasts to all listeners (thread-safe)
func (m *MultiListener) AgentToolCallCompleted(toolCall *ToolCallInfo) {
	for _, l := range m.getListeners() {
		l.AgentToolCallCompleted(toolCall)
	}
}

// AgentApprovalRequested broadcasts to all listeners (thread-safe)
func (m *MultiListener) AgentApprovalRequested(choice *ChoiceRequest) {
	for _, l := range m.getListeners() {
		l.AgentApprovalRequested(choice)
	}
}

// AgentApprovalTimeout broadcasts to all listeners (thread-safe)
func (m *MultiListener) AgentApprovalTimeout(choiceID string) {
	for _, l := range m.getListeners() {
		l.AgentApprovalTimeout(choiceID)
	}
}
