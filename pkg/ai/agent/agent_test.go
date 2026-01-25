package agent

import (
	"context"
	"testing"
	"time"
)

func TestStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     State
		to       State
		expected bool
	}{
		{"Idle to Running", StateIdle, StateRunning, true},
		{"Running to ToolAnalysis", StateRunning, StateToolAnalysis, true},
		{"Running to Done", StateRunning, StateDone, true},
		{"Running to Error", StateRunning, StateError, true},
		{"ToolAnalysis to WaitingForApproval", StateToolAnalysis, StateWaitingForApproval, true},
		{"ToolAnalysis to Running", StateToolAnalysis, StateRunning, true},
		{"WaitingForApproval to Running", StateWaitingForApproval, StateRunning, true},
		{"WaitingForApproval to Done", StateWaitingForApproval, StateDone, true},
		{"Done to Idle", StateDone, StateIdle, true},
		{"Error to Idle", StateError, StateIdle, true},
		// Invalid transitions
		{"Idle to Done", StateIdle, StateDone, false},
		{"Done to Running", StateDone, StateRunning, false},
		{"Error to Running", StateError, StateRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			if result != tt.expected {
				t.Errorf("CanTransitionTo(%s, %s) = %v, want %v",
					tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateIdle, "idle"},
		{StateRunning, "running"},
		{StateToolAnalysis, "analyzing"},
		{StateWaitingForApproval, "waiting"},
		{StateDone, "done"},
		{StateError, "error"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("State.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgentNew(t *testing.T) {
	agent := New(nil)

	if agent == nil {
		t.Fatal("New(nil) returned nil")
	}

	if agent.State() != StateIdle {
		t.Errorf("Initial state = %v, want %v", agent.State(), StateIdle)
	}

	if agent.maxIterations != 10 {
		t.Errorf("maxIterations = %d, want 10", agent.maxIterations)
	}

	if agent.approvalTimeout != 30*time.Second {
		t.Errorf("approvalTimeout = %v, want 30s", agent.approvalTimeout)
	}

	if agent.Input == nil {
		t.Error("Input channel is nil")
	}

	if agent.Output == nil {
		t.Error("Output channel is nil")
	}
}

func TestAgentNewWithConfig(t *testing.T) {
	cfg := &Config{
		MaxIterations:       20,
		ApprovalTimeout:     60 * time.Second,
		AutoApproveReadOnly: false,
	}

	agent := New(cfg)

	if agent.maxIterations != 20 {
		t.Errorf("maxIterations = %d, want 20", agent.maxIterations)
	}

	if agent.approvalTimeout != 60*time.Second {
		t.Errorf("approvalTimeout = %v, want 60s", agent.approvalTimeout)
	}

	if agent.autoApproveReadOnly != false {
		t.Error("autoApproveReadOnly should be false")
	}
}

func TestAgentStateChange(t *testing.T) {
	agent := New(nil)

	// Drain initial state
	go func() {
		for range agent.Output {
		}
	}()

	agent.setState(StateRunning)

	if agent.State() != StateRunning {
		t.Errorf("State = %v, want %v", agent.State(), StateRunning)
	}
}

func TestAgentSession(t *testing.T) {
	agent := New(nil)

	agent.StartSession("openai", "gpt-4")

	session := agent.GetSession()
	if session == nil {
		t.Fatal("Session is nil after StartSession")
	}

	if session.Provider != "openai" {
		t.Errorf("Provider = %s, want openai", session.Provider)
	}

	if session.Model != "gpt-4" {
		t.Errorf("Model = %s, want gpt-4", session.Model)
	}

	messages := agent.GetMessages()
	if len(messages) != 0 {
		t.Errorf("Initial messages count = %d, want 0", len(messages))
	}
}

func TestMessageCreation(t *testing.T) {
	t.Run("NewTextMessage", func(t *testing.T) {
		msg := NewTextMessage("hello")
		if msg.Type != MsgText {
			t.Errorf("Type = %v, want MsgText", msg.Type)
		}
		if msg.Content != "hello" {
			t.Errorf("Content = %s, want hello", msg.Content)
		}
	})

	t.Run("NewStreamChunk", func(t *testing.T) {
		msg := NewStreamChunk("chunk")
		if msg.Type != MsgStreamChunk {
			t.Errorf("Type = %v, want MsgStreamChunk", msg.Type)
		}
		if !msg.IsStreaming {
			t.Error("IsStreaming should be true")
		}
	})

	t.Run("NewErrorMessage", func(t *testing.T) {
		err := context.DeadlineExceeded
		msg := NewErrorMessage(err)
		if msg.Type != MsgError {
			t.Errorf("Type = %v, want MsgError", msg.Type)
		}
		if msg.Error != err {
			t.Errorf("Error = %v, want %v", msg.Error, err)
		}
	})

	t.Run("NewChoiceResponseMessage", func(t *testing.T) {
		msg := NewChoiceResponseMessage(true)
		if msg.Type != MsgUserChoiceResponse {
			t.Errorf("Type = %v, want MsgUserChoiceResponse", msg.Type)
		}
		if msg.Content != "approved" {
			t.Errorf("Content = %s, want approved", msg.Content)
		}

		msg = NewChoiceResponseMessage(false)
		if msg.Content != "rejected" {
			t.Errorf("Content = %s, want rejected", msg.Content)
		}
	})
}

func TestApprovalRequest(t *testing.T) {
	req := NewApprovalRequest("id-123", "kubectl delete pod nginx", false)

	if req.ID != "id-123" {
		t.Errorf("ID = %s, want id-123", req.ID)
	}

	if req.Command != "kubectl delete pod nginx" {
		t.Errorf("Command = %s, want kubectl delete pod nginx", req.Command)
	}

	if req.Title != "Command Approval Required" {
		t.Errorf("Title = %s, want Command Approval Required", req.Title)
	}

	// Test dangerous command
	req = NewApprovalRequest("id-456", "kubectl delete ns default", true)
	if req.Title != "DANGEROUS Command - Approval Required" {
		t.Errorf("Title = %s, want DANGEROUS Command - Approval Required", req.Title)
	}
}

func TestAgentSendApproval(t *testing.T) {
	agent := New(nil)

	// Start a goroutine to receive the message
	done := make(chan bool)
	go func() {
		msg := <-agent.Input
		if msg.Type != MsgUserChoiceResponse {
			t.Errorf("Type = %v, want MsgUserChoiceResponse", msg.Type)
		}
		if msg.Content != "approved" {
			t.Errorf("Content = %s, want approved", msg.Content)
		}
		done <- true
	}()

	agent.SendApproval(true)

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("Timeout waiting for approval message")
	}
}

func TestAgentStop(t *testing.T) {
	agent := New(nil)
	ctx, cancel := context.WithCancel(context.Background())
	agent.ctx = ctx
	agent.cancel = cancel

	agent.Stop()

	select {
	case <-agent.ctx.Done():
		// Success
	case <-time.After(time.Second):
		t.Error("Context not cancelled after Stop()")
	}
}

func TestAgentEmitFunctions(t *testing.T) {
	agent := New(nil)

	// Start a goroutine to drain the output channel
	received := make(chan *Message, 10)
	go func() {
		for msg := range agent.Output {
			received <- msg
		}
	}()

	t.Run("emitText", func(t *testing.T) {
		agent.emitText("test text")
		select {
		case msg := <-received:
			if msg.Type != MsgText {
				t.Errorf("Type = %v, want MsgText", msg.Type)
			}
			if msg.Content != "test text" {
				t.Errorf("Content = %s, want 'test text'", msg.Content)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for emitText message")
		}
	})

	t.Run("emitStreamChunk", func(t *testing.T) {
		agent.emitStreamChunk("chunk data")
		select {
		case msg := <-received:
			if msg.Type != MsgStreamChunk {
				t.Errorf("Type = %v, want MsgStreamChunk", msg.Type)
			}
			if msg.Content != "chunk data" {
				t.Errorf("Content = %s, want 'chunk data'", msg.Content)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for emitStreamChunk message")
		}
	})

	t.Run("emitStreamEnd", func(t *testing.T) {
		agent.emitStreamEnd()
		select {
		case msg := <-received:
			if msg.Type != MsgStreamEnd {
				t.Errorf("Type = %v, want MsgStreamEnd", msg.Type)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for emitStreamEnd message")
		}
	})

	t.Run("emitError", func(t *testing.T) {
		testErr := context.Canceled
		agent.emitError(testErr)
		select {
		case msg := <-received:
			if msg.Type != MsgError {
				t.Errorf("Type = %v, want MsgError", msg.Type)
			}
			if msg.Error != testErr {
				t.Errorf("Error = %v, want %v", msg.Error, testErr)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for emitError message")
		}
	})

	t.Run("emitToolCallRequest", func(t *testing.T) {
		tc := &ToolCallInfo{ID: "tc-1", Name: "kubectl", Command: "get pods"}
		agent.emitToolCallRequest(tc)
		select {
		case msg := <-received:
			if msg.Type != MsgToolCallRequest {
				t.Errorf("Type = %v, want MsgToolCallRequest", msg.Type)
			}
			if msg.ToolCall.ID != "tc-1" {
				t.Errorf("ToolCall.ID = %s, want tc-1", msg.ToolCall.ID)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for emitToolCallRequest message")
		}
	})

	t.Run("emitToolCallCompleted", func(t *testing.T) {
		tc := &ToolCallInfo{ID: "tc-2", Name: "kubectl", Result: "pod/nginx"}
		agent.emitToolCallCompleted(tc)
		select {
		case msg := <-received:
			if msg.Type != MsgToolCallResponse {
				t.Errorf("Type = %v, want MsgToolCallResponse", msg.Type)
			}
			if msg.Content != "pod/nginx" {
				t.Errorf("Content = %s, want 'pod/nginx'", msg.Content)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for emitToolCallCompleted message")
		}
	})

	t.Run("emitApprovalRequest", func(t *testing.T) {
		choice := &ChoiceRequest{ID: "choice-1", Command: "kubectl delete pod nginx"}
		agent.emitApprovalRequest(choice)
		select {
		case msg := <-received:
			if msg.Type != MsgUserChoiceRequest {
				t.Errorf("Type = %v, want MsgUserChoiceRequest", msg.Type)
			}
			if msg.Choice.ID != "choice-1" {
				t.Errorf("Choice.ID = %s, want choice-1", msg.Choice.ID)
			}
		case <-time.After(time.Second):
			t.Error("Timeout waiting for emitApprovalRequest message")
		}
	})
}

func TestAgentSetListener(t *testing.T) {
	agent := New(nil)

	// Initially no listener
	if agent.GetListener() != nil {
		t.Error("Initial listener should be nil")
	}

	// Set a listener
	listener := &testAgentListener{}
	agent.SetListener(listener)

	if agent.GetListener() != listener {
		t.Error("GetListener should return the set listener")
	}

	// Remove listener
	agent.SetListener(nil)
	if agent.GetListener() != nil {
		t.Error("Listener should be nil after setting nil")
	}
}

func TestAgentNotifyFunctions(t *testing.T) {
	agent := New(nil)

	// Drain output channel
	go func() {
		for range agent.Output {
		}
	}()

	var events []string
	listener := &testAgentListener{
		onText:   func(s string) { events = append(events, "text:"+s) },
		onChunk:  func(s string) { events = append(events, "chunk:"+s) },
		onEnd:    func() { events = append(events, "end") },
		onError:  func(e error) { events = append(events, "error:"+e.Error()) },
		onState:  func(s State) { events = append(events, "state:"+s.String()) },
		onTool:   func(tc *ToolCallInfo) { events = append(events, "tool:"+tc.Name) },
		onChoice: func(c *ChoiceRequest) { events = append(events, "choice:"+c.ID) },
	}
	agent.SetListener(listener)

	// Test all notify functions
	agent.notifyText("hello")
	agent.notifyStreamChunk("chunk1")
	agent.notifyStreamEnd()
	agent.notifyError(context.Canceled)
	agent.notifyStateChanged(StateRunning)
	agent.notifyToolCallRequested(&ToolCallInfo{Name: "kubectl"})
	agent.notifyToolCallCompleted(&ToolCallInfo{Name: "bash"})
	agent.notifyApprovalRequested(&ChoiceRequest{ID: "c1"})
	agent.notifyApprovalTimeout("timeout-1")

	expected := []string{
		"text:hello",
		"chunk:chunk1",
		"end",
		"error:context canceled",
		"state:running",
		"tool:kubectl",
		"tool:bash",
		"choice:c1",
	}

	if len(events) != len(expected) {
		t.Errorf("Events count = %d, want %d", len(events), len(expected))
	}

	for i, e := range expected {
		if i < len(events) && events[i] != e {
			t.Errorf("Event[%d] = %s, want %s", i, events[i], e)
		}
	}
}

func TestAgentSetApprovalHandler(t *testing.T) {
	agent := New(nil)

	handler := &testApprovalHandler{}
	agent.SetApprovalHandler(handler)

	// Verify it's set (we can't directly check, but we can test the flow)
	agent.approvalHandlerMu.RLock()
	h := agent.approvalHandler
	agent.approvalHandlerMu.RUnlock()

	if h != handler {
		t.Error("ApprovalHandler was not set correctly")
	}
}

func TestAgentIsRunning(t *testing.T) {
	agent := New(nil)

	if agent.IsRunning() {
		t.Error("Agent should not be running initially")
	}

	agent.runningMu.Lock()
	agent.running = true
	agent.runningMu.Unlock()

	if !agent.IsRunning() {
		t.Error("Agent should be running after setting running=true")
	}
}

func TestAgentClearSession(t *testing.T) {
	agent := New(nil)
	agent.StartSession("test", "model")

	// Add a message
	agent.session.AddMessage("user", "hello")

	if len(agent.GetMessages()) != 1 {
		t.Errorf("Messages count = %d, want 1", len(agent.GetMessages()))
	}

	agent.ClearSession()

	if len(agent.GetMessages()) != 0 {
		t.Errorf("Messages count after clear = %d, want 0", len(agent.GetMessages()))
	}
}

func TestAgentLoadSessionNoStore(t *testing.T) {
	agent := New(nil)

	// Without session store, LoadSession should return nil
	err := agent.LoadSession("any-id")
	if err != nil {
		t.Errorf("LoadSession without store should return nil, got %v", err)
	}
}

func TestAgentSaveSessionNoStore(t *testing.T) {
	agent := New(nil)

	// Without session store, SaveSession should return nil
	err := agent.SaveSession()
	if err != nil {
		t.Errorf("SaveSession without store should return nil, got %v", err)
	}
}

func TestAgentSetProvider(t *testing.T) {
	agent := New(nil)

	// Test setting a non-tool provider
	provider := &mockBasicProvider{}
	agent.SetProvider(provider)

	if agent.provider != provider {
		t.Error("Provider was not set correctly")
	}
	if agent.toolProvider != nil {
		t.Error("toolProvider should be nil for non-ToolProvider")
	}
}

func TestAgentSetToolRegistry(t *testing.T) {
	agent := New(nil)

	if agent.toolRegistry != nil {
		t.Error("Initial toolRegistry should be nil")
	}

	// We can't easily import tools package here without circular deps,
	// so just check the setter works with nil
	agent.SetToolRegistry(nil)

	if agent.toolRegistry != nil {
		t.Error("toolRegistry should be nil after SetToolRegistry(nil)")
	}
}

func TestAgentClose(t *testing.T) {
	agent := New(nil)
	ctx, cancel := context.WithCancel(context.Background())
	agent.ctx = ctx
	agent.cancel = cancel

	// Close should not panic and should close channels
	agent.Close()

	// Verify context is cancelled
	select {
	case <-ctx.Done():
		// Success
	default:
		t.Error("Context should be cancelled after Close")
	}
}

func TestMessageTypeString(t *testing.T) {
	tests := []struct {
		mt       MessageType
		expected string
	}{
		{MsgText, "text"},
		{MsgError, "error"},
		{MsgToolCallRequest, "tool_call_request"},
		{MsgToolCallResponse, "tool_call_response"},
		{MsgUserChoiceRequest, "user_choice_request"},
		{MsgUserChoiceResponse, "user_choice_response"},
		{MsgStateChange, "state_change"},
		{MsgStreamChunk, "stream_chunk"},
		{MsgStreamEnd, "stream_end"},
		{MessageType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.mt.String(); got != tt.expected {
				t.Errorf("MessageType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStateIsTerminal(t *testing.T) {
	tests := []struct {
		state    State
		expected bool
	}{
		{StateIdle, false},
		{StateRunning, false},
		{StateToolAnalysis, false},
		{StateWaitingForApproval, false},
		{StateDone, true},
		{StateError, true},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.expected {
				t.Errorf("State.IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// mockBasicProvider implements only Provider (not ToolProvider)
type mockBasicProvider struct{}

func (m *mockBasicProvider) Name() string     { return "mock-basic" }
func (m *mockBasicProvider) GetModel() string { return "basic-model" }
func (m *mockBasicProvider) Ask(ctx context.Context, prompt string, cb func(string)) error {
	return nil
}
func (m *mockBasicProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	return "", nil
}
func (m *mockBasicProvider) IsReady() bool                                    { return true }
func (m *mockBasicProvider) ListModels(ctx context.Context) ([]string, error) { return nil, nil }

// testAgentListener is a configurable test listener for agent tests
type testAgentListener struct {
	onText    func(string)
	onChunk   func(string)
	onEnd     func()
	onError   func(error)
	onState   func(State)
	onTool    func(*ToolCallInfo)
	onChoice  func(*ChoiceRequest)
	onTimeout func(string)
}

func (l *testAgentListener) AgentTextReceived(text string) {
	if l.onText != nil {
		l.onText(text)
	}
}

func (l *testAgentListener) AgentStreamChunk(chunk string) {
	if l.onChunk != nil {
		l.onChunk(chunk)
	}
}

func (l *testAgentListener) AgentStreamEnd() {
	if l.onEnd != nil {
		l.onEnd()
	}
}

func (l *testAgentListener) AgentError(err error) {
	if l.onError != nil {
		l.onError(err)
	}
}

func (l *testAgentListener) AgentStateChanged(state State) {
	if l.onState != nil {
		l.onState(state)
	}
}

func (l *testAgentListener) AgentToolCallRequested(tc *ToolCallInfo) {
	if l.onTool != nil {
		l.onTool(tc)
	}
}

func (l *testAgentListener) AgentToolCallCompleted(tc *ToolCallInfo) {
	if l.onTool != nil {
		l.onTool(tc)
	}
}

func (l *testAgentListener) AgentApprovalRequested(choice *ChoiceRequest) {
	if l.onChoice != nil {
		l.onChoice(choice)
	}
}

func (l *testAgentListener) AgentApprovalTimeout(choiceID string) {
	if l.onTimeout != nil {
		l.onTimeout(choiceID)
	}
}

// testApprovalHandler is a test approval handler
type testApprovalHandler struct {
	autoApprove bool
}

func (h *testApprovalHandler) RequestApproval(choice *ChoiceRequest, callback func(approved bool)) {
	callback(h.autoApprove)
}
