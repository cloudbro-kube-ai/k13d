package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/providers"
	"github.com/cloudbro-kube-ai/k13d/pkg/ai/safety"
	"github.com/cloudbro-kube-ai/k13d/pkg/ai/tools"
)

func TestAgentRun_AlreadyRunning(t *testing.T) {
	agent := New(nil)
	agent.runningMu.Lock()
	agent.running = true
	agent.runningMu.Unlock()

	err := agent.Run(context.Background())
	if err == nil {
		t.Error("Expected error when agent is already running")
	}
	if err.Error() != "agent is already running" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestAgentRun_ContextCancelled(t *testing.T) {
	agent := New(nil)

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := agent.Run(ctx)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestAgentRun_WithUserMessage(t *testing.T) {
	agent := New(nil)
	agent.provider = &mockProvider{
		streamResponse: "Hello from AI",
	}

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	// Send message before running
	go func() {
		time.Sleep(50 * time.Millisecond)
		agent.SendUserMessage("Hello")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := agent.Run(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestHandleUserMessage(t *testing.T) {
	agent := New(nil)

	// Test auto-create session
	err := agent.handleUserMessage("test message")
	if err != nil {
		t.Errorf("handleUserMessage failed: %v", err)
	}

	if agent.session == nil {
		t.Error("Session should be auto-created")
	}

	messages := agent.GetMessages()
	if len(messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("Message role = %s, want user", messages[0].Role)
	}

	if messages[0].Content != "test message" {
		t.Errorf("Message content = %s, want 'test message'", messages[0].Content)
	}
}

func TestHandleUserMessage_WithExistingSession(t *testing.T) {
	agent := New(nil)
	agent.StartSession("openai", "gpt-4")

	err := agent.handleUserMessage("another message")
	if err != nil {
		t.Errorf("handleUserMessage failed: %v", err)
	}

	messages := agent.GetMessages()
	if len(messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(messages))
	}
}

func TestBuildPromptWithHistory(t *testing.T) {
	agent := New(nil)
	agent.StartSession("test", "model")

	// Add some messages
	agent.session.AddMessage("user", "Hello")
	agent.session.AddMessage("assistant", "Hi there!")
	agent.session.AddMessage("user", "How are you?")

	prompt := agent.buildPromptWithHistory()

	// Should contain system prompt
	if !contains(prompt, "You are k13d") {
		t.Error("Prompt should contain system prompt")
	}

	// Should contain conversation history
	if !contains(prompt, "User: Hello") {
		t.Error("Prompt should contain user message")
	}
	if !contains(prompt, "Assistant: Hi there!") {
		t.Error("Prompt should contain assistant message")
	}
	if !contains(prompt, "User: How are you?") {
		t.Error("Prompt should contain second user message")
	}
}

func TestBuildPromptWithHistory_LimitHistory(t *testing.T) {
	agent := New(nil)
	agent.StartSession("test", "model")

	// Add more than maxHistory (10) messages
	for i := 0; i < 15; i++ {
		agent.session.AddMessage("user", "message "+string(rune('A'+i)))
	}

	prompt := agent.buildPromptWithHistory()

	// Should only contain last 10 messages
	// First 5 messages (A-E) should not be present
	if contains(prompt, "message A") {
		t.Error("Prompt should not contain old messages beyond history limit")
	}

	// Last messages should be present
	if !contains(prompt, "message O") {
		t.Error("Prompt should contain recent messages")
	}
}

func TestCallLLM_NoProvider(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	err := agent.callLLM()
	if err == nil {
		t.Error("Expected error when no provider configured")
	}
	if err.Error() != "no LLM provider configured" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCallLLM_BasicProvider(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()
	agent.provider = &mockProvider{
		streamResponse: "Test response",
	}
	agent.StartSession("test", "model")

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	err := agent.callLLM()
	if err != nil {
		t.Errorf("callLLM failed: %v", err)
	}

	if agent.State() != StateDone {
		t.Errorf("State = %v, want StateDone", agent.State())
	}
}

func TestCallLLM_ProviderError(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()
	agent.provider = &mockProvider{
		err: errors.New("provider error"),
	}
	agent.StartSession("test", "model")

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	err := agent.callLLM()
	if err == nil {
		t.Error("Expected error from provider")
	}
	if err.Error() != "provider error" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCallLLM_WithToolProvider(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()

	mockTP := &mockToolProvider{
		streamResponse: "Tool response",
	}
	agent.provider = mockTP
	agent.toolProvider = mockTP
	agent.toolRegistry = tools.NewRegistry()
	agent.StartSession("test", "model")

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	err := agent.callLLM()
	if err != nil {
		t.Errorf("callLLM failed: %v", err)
	}
}

func TestBuildToolDefinitions(t *testing.T) {
	agent := New(nil)

	// No registry
	defs := agent.buildToolDefinitions()
	if defs != nil {
		t.Error("buildToolDefinitions should return nil when no registry")
	}

	// With registry
	agent.toolRegistry = tools.NewRegistry()

	defs = agent.buildToolDefinitions()
	if defs == nil {
		t.Error("buildToolDefinitions should return non-nil with registry")
	}
}

func TestAnalyzeToolCalls_Empty(t *testing.T) {
	agent := New(nil)
	agent.pendingToolCalls = []*ToolCallInfo{}

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	agent.analyzeToolCalls()

	if agent.State() != StateRunning {
		t.Errorf("State = %v, want StateRunning", agent.State())
	}
}

func TestAnalyzeToolCalls_ReadOnly(t *testing.T) {
	agent := New(nil)
	agent.autoApproveReadOnly = true
	agent.pendingToolCalls = []*ToolCallInfo{
		{ID: "1", Name: "kubectl", IsReadOnly: true},
	}

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	agent.analyzeToolCalls()

	if agent.State() != StateRunning {
		t.Errorf("State = %v, want StateRunning for auto-approved read-only", agent.State())
	}
}

func TestAnalyzeToolCalls_NeedsApproval(t *testing.T) {
	agent := New(nil)
	agent.autoApproveReadOnly = true
	agent.pendingToolCalls = []*ToolCallInfo{
		{ID: "1", Name: "kubectl", IsReadOnly: false},
	}

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	agent.analyzeToolCalls()

	if agent.State() != StateWaitingForApproval {
		t.Errorf("State = %v, want StateWaitingForApproval", agent.State())
	}
}

func TestWaitForApproval_Approved(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()
	agent.pendingToolCalls = []*ToolCallInfo{
		{ID: "1", Name: "kubectl"},
	}

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	// Send approval
	go func() {
		time.Sleep(50 * time.Millisecond)
		agent.Input <- &Message{
			Type:    MsgUserChoiceResponse,
			Content: "approved",
		}
	}()

	err := agent.waitForApproval()
	if err != nil {
		t.Errorf("waitForApproval failed: %v", err)
	}

	if agent.State() != StateRunning {
		t.Errorf("State = %v, want StateRunning after approval", agent.State())
	}
}

func TestWaitForApproval_Rejected(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()
	agent.pendingToolCalls = []*ToolCallInfo{
		{ID: "1", Name: "kubectl"},
	}

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	// Send rejection
	go func() {
		time.Sleep(50 * time.Millisecond)
		agent.Input <- &Message{
			Type:    MsgUserChoiceResponse,
			Content: "rejected",
		}
	}()

	err := agent.waitForApproval()
	if err != nil {
		t.Errorf("waitForApproval failed: %v", err)
	}

	if agent.State() != StateRunning {
		t.Errorf("State = %v, want StateRunning after rejection", agent.State())
	}
}

func TestWaitForApproval_Timeout(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()
	agent.approvalTimeout = 100 * time.Millisecond
	agent.pendingToolCalls = []*ToolCallInfo{
		{ID: "1", Name: "kubectl"},
	}

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	err := agent.waitForApproval()
	if err != nil {
		t.Errorf("waitForApproval failed: %v", err)
	}

	if agent.State() != StateDone {
		t.Errorf("State = %v, want StateDone after timeout", agent.State())
	}
}

func TestWaitForApproval_ContextCancelled(t *testing.T) {
	agent := New(nil)
	ctx, cancel := context.WithCancel(context.Background())
	agent.ctx = ctx
	agent.approvalTimeout = 5 * time.Second

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	// Cancel context
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := agent.waitForApproval()
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestExecuteApprovedTools(t *testing.T) {
	agent := New(nil)
	agent.pendingToolCalls = []*ToolCallInfo{
		{ID: "1", Name: "kubectl", Approved: false},
		{ID: "2", Name: "bash", Approved: false},
	}

	agent.executeApprovedTools()

	// All should be approved
	for _, tc := range agent.pendingToolCalls {
		if !tc.Approved {
			t.Errorf("ToolCall %s should be approved", tc.ID)
		}
	}
}

func TestCancelPendingTools(t *testing.T) {
	agent := New(nil)
	agent.pendingToolCalls = []*ToolCallInfo{
		{ID: "1", Name: "kubectl", Approved: true},
		{ID: "2", Name: "bash", Approved: true},
	}

	agent.cancelPendingTools()

	if len(agent.pendingToolCalls) != 0 {
		t.Errorf("pendingToolCalls should be empty after cancel, got %d", len(agent.pendingToolCalls))
	}
}

func TestSafetyQuickCheck(t *testing.T) {
	tests := []struct {
		cmd              string
		expectReadOnly   bool
		expectDangerous  bool
	}{
		{"kubectl get pods", true, false},
		{"kubectl delete pod nginx", false, true},
		{"kubectl describe node worker-1", true, false},
		{"kubectl delete ns production", false, true},
		{"rm -rf /", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			isReadOnly, isDangerous := safety.QuickCheck(tt.cmd)
			if isReadOnly != tt.expectReadOnly {
				t.Errorf("QuickCheck(%s) readOnly = %v, want %v", tt.cmd, isReadOnly, tt.expectReadOnly)
			}
			if isDangerous != tt.expectDangerous {
				t.Errorf("QuickCheck(%s) dangerous = %v, want %v", tt.cmd, isDangerous, tt.expectDangerous)
			}
		})
	}
}

func TestAsk(t *testing.T) {
	agent := New(nil)
	agent.provider = &mockProvider{
		streamResponse: "response",
	}

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := agent.Ask(ctx, "test question")
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Ask failed: %v", err)
	}
}

func TestAskWithContext(t *testing.T) {
	agent := New(nil)
	agent.provider = &mockProvider{
		streamResponse: "response",
	}

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := agent.AskWithContext(ctx, "explain this", "pod: nginx\nstatus: Running")
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("AskWithContext failed: %v", err)
	}
}

func TestRequestAndWaitForApproval_WithHandler(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()

	// Set approval handler that auto-approves
	agent.SetApprovalHandler(&testApprovalHandler{autoApprove: true})

	toolCall := &ToolCallInfo{ID: "tc-1", Command: "kubectl get pods"}

	approved := agent.requestAndWaitForApproval(toolCall)
	if !approved {
		t.Error("Expected approval with auto-approve handler")
	}

	// Test with rejection
	agent.SetApprovalHandler(&testApprovalHandler{autoApprove: false})

	approved = agent.requestAndWaitForApproval(toolCall)
	if approved {
		t.Error("Expected rejection with auto-reject handler")
	}
}

func TestRequestAndWaitForApproval_ChannelBased(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()
	agent.approvalTimeout = 5 * time.Second

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	toolCall := &ToolCallInfo{ID: "tc-1", Command: "kubectl get pods"}

	// Send approval via channel
	go func() {
		time.Sleep(50 * time.Millisecond)
		agent.Input <- &Message{
			Type:    MsgUserChoiceResponse,
			Content: "approve",
		}
	}()

	approved := agent.requestAndWaitForApproval(toolCall)
	if !approved {
		t.Error("Expected approval from channel")
	}
}

func TestRequestAndWaitForApproval_Timeout(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()
	agent.approvalTimeout = 100 * time.Millisecond

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	toolCall := &ToolCallInfo{ID: "tc-1", Command: "kubectl get pods"}

	approved := agent.requestAndWaitForApproval(toolCall)
	if approved {
		t.Error("Expected timeout (no approval)")
	}
}

func TestHandleToolCall(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()
	agent.autoApproveReadOnly = true
	agent.toolRegistry = tools.NewRegistry()
	agent.StartSession("test", "model")

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	t.Run("ReadOnly tool auto-approved", func(t *testing.T) {
		call := providers.ToolCall{
			ID:   "tc-readonly",
			Type: "function",
			Function: providers.FunctionCall{
				Name:      "kubectl",
				Arguments: `{"command": "kubectl get pods"}`,
			},
		}

		result := agent.handleToolCall(call)

		// Read-only commands should be auto-approved (autoApproveReadOnly = true)
		// The tool doesn't exist in registry, but the call completes
		if result.ToolCallID != "tc-readonly" {
			t.Errorf("ToolCallID = %s, want tc-readonly", result.ToolCallID)
		}
	})

	t.Run("Write tool needs approval - with handler", func(t *testing.T) {
		// Set approval handler that auto-approves
		agent.SetApprovalHandler(&testApprovalHandler{autoApprove: true})

		call := providers.ToolCall{
			ID:   "tc-write",
			Type: "function",
			Function: providers.FunctionCall{
				Name:      "kubectl",
				Arguments: `{"command": "kubectl delete pod nginx"}`,
			},
		}

		result := agent.handleToolCall(call)

		// Tool doesn't exist, but approval should have been granted
		if result.ToolCallID != "tc-write" {
			t.Errorf("ToolCallID = %s, want tc-write", result.ToolCallID)
		}
	})

	t.Run("Write tool rejected", func(t *testing.T) {
		// Set approval handler that rejects
		agent.SetApprovalHandler(&testApprovalHandler{autoApprove: false})

		call := providers.ToolCall{
			ID:   "tc-rejected",
			Type: "function",
			Function: providers.FunctionCall{
				Name:      "kubectl",
				Arguments: `{"command": "kubectl delete pod nginx"}`,
			},
		}

		result := agent.handleToolCall(call)

		if !result.IsError {
			t.Error("Expected error for rejected tool call")
		}
		if result.Content != "Tool execution cancelled by user" {
			t.Errorf("Unexpected content: %s", result.Content)
		}
	})
}

func TestHandleToolCall_InvalidJSON(t *testing.T) {
	agent := New(nil)
	agent.ctx = context.Background()
	agent.autoApproveReadOnly = true
	agent.toolRegistry = tools.NewRegistry()
	agent.StartSession("test", "model")
	// Set approval handler to avoid timeout
	agent.SetApprovalHandler(&testApprovalHandler{autoApprove: true})

	// Drain output
	go func() {
		for range agent.Output {
		}
	}()

	call := providers.ToolCall{
		ID:   "tc-invalid",
		Type: "function",
		Function: providers.FunctionCall{
			Name:      "kubectl",
			Arguments: `invalid json`,
		},
	}

	result := agent.handleToolCall(call)

	// Should still process (will use raw arguments as command)
	if result.ToolCallID != "tc-invalid" {
		t.Errorf("ToolCallID = %s, want tc-invalid", result.ToolCallID)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mockProvider implements providers.Provider for testing
type mockProvider struct {
	streamResponse string
	err            error
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) GetModel() string {
	return "mock-model"
}

func (m *mockProvider) Ask(ctx context.Context, prompt string, streamCallback func(string)) error {
	if m.err != nil {
		return m.err
	}
	if streamCallback != nil && m.streamResponse != "" {
		streamCallback(m.streamResponse)
	}
	return nil
}

func (m *mockProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.streamResponse, nil
}

func (m *mockProvider) IsReady() bool {
	return true
}

func (m *mockProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

// mockToolProvider implements both Provider and ToolProvider
type mockToolProvider struct {
	streamResponse string
	toolCalls      []providers.ToolCall
	err            error
}

func (m *mockToolProvider) Name() string {
	return "mock-tool"
}

func (m *mockToolProvider) GetModel() string {
	return "mock-tool-model"
}

func (m *mockToolProvider) Ask(ctx context.Context, prompt string, streamCallback func(string)) error {
	if m.err != nil {
		return m.err
	}
	if streamCallback != nil && m.streamResponse != "" {
		streamCallback(m.streamResponse)
	}
	return nil
}

func (m *mockToolProvider) AskNonStreaming(ctx context.Context, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.streamResponse, nil
}

func (m *mockToolProvider) IsReady() bool {
	return true
}

func (m *mockToolProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"mock-tool-model"}, nil
}

func (m *mockToolProvider) AskWithTools(ctx context.Context, prompt string, tools []providers.ToolDefinition, streamCallback func(string), toolCallback providers.ToolCallback) error {
	if m.err != nil {
		return m.err
	}
	if streamCallback != nil && m.streamResponse != "" {
		streamCallback(m.streamResponse)
	}
	for _, tc := range m.toolCalls {
		if toolCallback != nil {
			toolCallback(tc)
		}
	}
	return nil
}

// Test getLanguageInstruction function
func TestGetLanguageInstruction(t *testing.T) {
	tests := []struct {
		name      string
		lang      string
		wantEmpty bool
		contains  string
	}{
		{
			name:      "korean returns instruction",
			lang:      "ko",
			wantEmpty: false,
			contains:  "Korean",
		},
		{
			name:      "chinese returns instruction",
			lang:      "zh",
			wantEmpty: false,
			contains:  "Chinese",
		},
		{
			name:      "japanese returns instruction",
			lang:      "ja",
			wantEmpty: false,
			contains:  "Japanese",
		},
		{
			name:      "english returns empty",
			lang:      "en",
			wantEmpty: true,
		},
		{
			name:      "unknown language returns empty",
			lang:      "fr",
			wantEmpty: true,
		},
		{
			name:      "empty string returns empty",
			lang:      "",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLanguageInstruction(tt.lang)

			if tt.wantEmpty && result != "" {
				t.Errorf("getLanguageInstruction(%q) = %q, want empty", tt.lang, result)
			}

			if !tt.wantEmpty && result == "" {
				t.Errorf("getLanguageInstruction(%q) returned empty, want non-empty", tt.lang)
			}

			if tt.contains != "" && result != "" {
				if !containsString(result, tt.contains) {
					t.Errorf("getLanguageInstruction(%q) = %q, want to contain %q", tt.lang, result, tt.contains)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test Agent SetLanguage method
func TestAgentSetLanguage(t *testing.T) {
	agent := New(nil)

	// Initial language should be empty
	if agent.language != "" {
		t.Errorf("Initial language = %q, want empty", agent.language)
	}

	// Set language to Korean
	agent.SetLanguage("ko")
	if agent.language != "ko" {
		t.Errorf("After SetLanguage(\"ko\"), language = %q, want \"ko\"", agent.language)
	}

	// Set language to English
	agent.SetLanguage("en")
	if agent.language != "en" {
		t.Errorf("After SetLanguage(\"en\"), language = %q, want \"en\"", agent.language)
	}
}

// Test Agent Config with Language
func TestAgentConfigWithLanguage(t *testing.T) {
	cfg := &Config{
		MaxIterations: 5,
		Language:      "ko",
	}

	agent := New(cfg)

	if agent.language != "ko" {
		t.Errorf("Agent language = %q, want \"ko\"", agent.language)
	}

	if agent.maxIterations != 5 {
		t.Errorf("Agent maxIterations = %d, want 5", agent.maxIterations)
	}
}
