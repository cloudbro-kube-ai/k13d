package agent

import (
	"context"
	"sync"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/providers"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/safety"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/sessions"
	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/ai/tools"
)

// Agent manages the AI conversation loop with state machine.
// It supports two communication patterns:
// 1. Channel-based (Input/Output) - for async message passing
// 2. Listener-based (k9s pattern) - for event-driven callbacks
//
// Both TUI and WebUI can use either pattern based on their needs.
type Agent struct {
	// State
	state   State
	stateMu sync.RWMutex

	// Configuration
	maxIterations       int
	approvalTimeout     time.Duration
	autoApproveReadOnly bool
	language            string // Display language for responses (e.g., "ko", "en")

	// Dependencies
	provider       providers.Provider
	toolProvider   providers.ToolProvider
	toolRegistry   *tools.Registry
	safetyAnalyzer *safety.Analyzer
	session        *sessions.Session
	sessionStore   sessions.Store

	// Listener (k9s pattern) - preferred for TUI
	listener   AgentListener
	listenerMu sync.RWMutex

	// Approval handler for synchronous approval flow
	approvalHandler   AgentApprovalHandler
	approvalHandlerMu sync.RWMutex

	// I/O Channels (alternative async pattern)
	Input  chan *Message // Messages from UI to Agent
	Output chan *Message // Messages from Agent to UI

	// Internal
	pendingToolCalls []*ToolCallInfo
	ctx              context.Context
	cancel           context.CancelFunc
	running          bool
	runningMu        sync.Mutex
}

// Config holds agent configuration
type Config struct {
	MaxIterations       int
	ApprovalTimeout     time.Duration
	AutoApproveReadOnly bool
	Provider            providers.Provider
	ToolRegistry        *tools.Registry
	SessionStore        sessions.Store
	Language            string // Display language for responses (e.g., "ko", "en")
}

// DefaultConfig returns default agent configuration
func DefaultConfig() *Config {
	return &Config{
		MaxIterations:       10,
		ApprovalTimeout:     30 * time.Second,
		AutoApproveReadOnly: true,
	}
}

// New creates a new Agent
func New(cfg *Config) *Agent {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 10
	}
	if cfg.ApprovalTimeout == 0 {
		cfg.ApprovalTimeout = 30 * time.Second
	}

	a := &Agent{
		state:               StateIdle,
		maxIterations:       cfg.MaxIterations,
		approvalTimeout:     cfg.ApprovalTimeout,
		autoApproveReadOnly: cfg.AutoApproveReadOnly,
		language:            cfg.Language,
		provider:            cfg.Provider,
		toolRegistry:        cfg.ToolRegistry,
		safetyAnalyzer:      safety.NewAnalyzer(),
		sessionStore:        cfg.SessionStore,
		Input:               make(chan *Message, 10),
		Output:              make(chan *Message, 100), // Larger buffer for streaming
		pendingToolCalls:    make([]*ToolCallInfo, 0),
	}

	// Check if provider supports tools
	if tp, ok := cfg.Provider.(providers.ToolProvider); ok {
		a.toolProvider = tp
	}

	return a
}

// State returns the current agent state (thread-safe)
func (a *Agent) State() State {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.state
}

// setState updates state and notifies UI via both channel and listener
func (a *Agent) setState(s State) {
	a.stateMu.Lock()
	oldState := a.state
	a.state = s
	a.stateMu.Unlock()

	// Only notify if state actually changed
	if oldState != s {
		a.emit(NewStateChangeMessage(s))
		a.notifyStateChanged(s)
	}
}

// emit sends a message to the Output channel (non-blocking)
func (a *Agent) emit(msg *Message) {
	select {
	case a.Output <- msg:
	default:
		// Buffer full, skip (shouldn't happen with large buffer)
	}
}

// emitText sends text to the UI via both channel and listener
func (a *Agent) emitText(text string) {
	a.emit(NewTextMessage(text))
	a.notifyText(text)
}

// emitStreamChunk sends a streaming chunk via both channel and listener
func (a *Agent) emitStreamChunk(chunk string) {
	a.emit(NewStreamChunk(chunk))
	a.notifyStreamChunk(chunk)
}

// emitStreamEnd sends stream end via both channel and listener
func (a *Agent) emitStreamEnd() {
	a.emit(NewStreamEnd())
	a.notifyStreamEnd()
}

// emitError sends an error via both channel and listener
func (a *Agent) emitError(err error) {
	a.emit(NewErrorMessage(err))
	a.notifyError(err)
}

// emitToolCallRequest sends tool call request via both channel and listener
func (a *Agent) emitToolCallRequest(tc *ToolCallInfo) {
	a.emit(NewToolCallRequestMessage(tc))
	a.notifyToolCallRequested(tc)
}

// emitToolCallCompleted sends tool call completion via both channel and listener
func (a *Agent) emitToolCallCompleted(tc *ToolCallInfo) {
	a.emit(&Message{
		Type:     MsgToolCallResponse,
		Content:  tc.Result,
		ToolCall: tc,
	})
	a.notifyToolCallCompleted(tc)
}

// emitApprovalRequest sends approval request via both channel and listener
func (a *Agent) emitApprovalRequest(choice *ChoiceRequest) {
	a.emit(NewChoiceRequestMessage(choice))
	a.notifyApprovalRequested(choice)
}

// IsRunning returns true if the agent is currently processing
func (a *Agent) IsRunning() bool {
	a.runningMu.Lock()
	defer a.runningMu.Unlock()
	return a.running
}

// SetProvider sets the LLM provider
func (a *Agent) SetProvider(p providers.Provider) {
	a.provider = p
	if tp, ok := p.(providers.ToolProvider); ok {
		a.toolProvider = tp
	} else {
		a.toolProvider = nil
	}
}

// SetToolRegistry sets the tool registry
func (a *Agent) SetToolRegistry(r *tools.Registry) {
	a.toolRegistry = r
}

// StartSession begins a new conversation session
func (a *Agent) StartSession(provider, model string) {
	a.session = sessions.NewSession(provider, model)
}

// LoadSession restores a previous session
func (a *Agent) LoadSession(id string) error {
	if a.sessionStore == nil {
		return nil
	}
	session, err := a.sessionStore.Load(id)
	if err != nil {
		return err
	}
	a.session = session
	return nil
}

// SaveSession persists the current session
func (a *Agent) SaveSession() error {
	if a.sessionStore == nil || a.session == nil {
		return nil
	}
	return a.sessionStore.Save(a.session)
}

// GetSession returns the current session
func (a *Agent) GetSession() *sessions.Session {
	return a.session
}

// GetMessages returns conversation history for UI display
func (a *Agent) GetMessages() []sessions.Message {
	if a.session == nil {
		return nil
	}
	return a.session.Messages
}

// ClearSession clears the current session messages
func (a *Agent) ClearSession() {
	if a.session != nil {
		a.session.ClearMessages()
	}
}

// Stop cancels the current agent run
func (a *Agent) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
}

// Close closes the agent and its channels
func (a *Agent) Close() {
	a.Stop()
	close(a.Input)
	close(a.Output)
}

// SendUserMessage sends a user message to the agent
func (a *Agent) SendUserMessage(content string) {
	a.Input <- NewTextMessage(content)
}

// SendApproval sends an approval response to the agent
func (a *Agent) SendApproval(approved bool) {
	a.Input <- NewChoiceResponseMessage(approved)
}

// SetListener sets the agent listener (k9s pattern)
// Pass nil to remove the listener
func (a *Agent) SetListener(l AgentListener) {
	a.listenerMu.Lock()
	defer a.listenerMu.Unlock()
	a.listener = l
}

// GetListener returns the current listener
func (a *Agent) GetListener() AgentListener {
	a.listenerMu.RLock()
	defer a.listenerMu.RUnlock()
	return a.listener
}

// SetApprovalHandler sets the approval handler for synchronous approval flow
func (a *Agent) SetApprovalHandler(h AgentApprovalHandler) {
	a.approvalHandlerMu.Lock()
	defer a.approvalHandlerMu.Unlock()
	a.approvalHandler = h
}

// notifyListener sends event to listener if set
func (a *Agent) notifyListener(fn func(AgentListener)) {
	a.listenerMu.RLock()
	l := a.listener
	a.listenerMu.RUnlock()

	if l != nil {
		fn(l)
	}
}

// notifyText notifies listener of text event
func (a *Agent) notifyText(text string) {
	a.notifyListener(func(l AgentListener) {
		l.AgentTextReceived(text)
	})
}

// notifyStreamChunk notifies listener of stream chunk
func (a *Agent) notifyStreamChunk(chunk string) {
	a.notifyListener(func(l AgentListener) {
		l.AgentStreamChunk(chunk)
	})
}

// notifyStreamEnd notifies listener of stream end
func (a *Agent) notifyStreamEnd() {
	a.notifyListener(func(l AgentListener) {
		l.AgentStreamEnd()
	})
}

// notifyError notifies listener of error
func (a *Agent) notifyError(err error) {
	a.notifyListener(func(l AgentListener) {
		l.AgentError(err)
	})
}

// notifyStateChanged notifies listener of state change
func (a *Agent) notifyStateChanged(state State) {
	a.notifyListener(func(l AgentListener) {
		l.AgentStateChanged(state)
	})
}

// notifyToolCallRequested notifies listener of tool call request
func (a *Agent) notifyToolCallRequested(tc *ToolCallInfo) {
	a.notifyListener(func(l AgentListener) {
		l.AgentToolCallRequested(tc)
	})
}

// notifyToolCallCompleted notifies listener of tool call completion
func (a *Agent) notifyToolCallCompleted(tc *ToolCallInfo) {
	a.notifyListener(func(l AgentListener) {
		l.AgentToolCallCompleted(tc)
	})
}

// notifyApprovalRequested notifies listener of approval request
func (a *Agent) notifyApprovalRequested(choice *ChoiceRequest) {
	a.notifyListener(func(l AgentListener) {
		l.AgentApprovalRequested(choice)
	})
}

// notifyApprovalTimeout notifies listener of approval timeout
func (a *Agent) notifyApprovalTimeout(choiceID string) {
	a.notifyListener(func(l AgentListener) {
		l.AgentApprovalTimeout(choiceID)
	})
}
