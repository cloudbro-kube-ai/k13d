package sessions

import (
	"time"
)

// Message represents a single message in a conversation
type Message struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Session represents a conversation session
type Session struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Title     string    `json:"title,omitempty"` // Auto-generated from first message
	Namespace string    `json:"namespace,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	// Agent state tracking
	AgentState  string          `json:"agent_state,omitempty"`
	ToolHistory []ToolExecution `json:"tool_history,omitempty"`
}

// ToolExecution records a tool execution in the session
type ToolExecution struct {
	Timestamp time.Time `json:"timestamp"`
	ToolName  string    `json:"tool_name"`
	Command   string    `json:"command"`
	Result    string    `json:"result"`
	Approved  bool      `json:"approved"`
	Duration  int64     `json:"duration_ms"` // Execution time in milliseconds
}

// SessionMetadata holds summary info for listing sessions
type SessionMetadata struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Messages  int       `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store defines the interface for session persistence
type Store interface {
	// Save persists a session
	Save(session *Session) error

	// Load retrieves a session by ID
	Load(id string) (*Session, error)

	// Delete removes a session
	Delete(id string) error

	// List returns all session metadata
	List() ([]SessionMetadata, error)

	// Clear removes all sessions
	Clear() error
}

// NewSession creates a new session with generated ID
func NewSession(provider, model string) *Session {
	now := time.Now()
	return &Session{
		ID:        generateID(),
		Provider:  provider,
		Model:     model,
		Messages:  make([]Message, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddMessage adds a message to the session
func (s *Session) AddMessage(role, content string) {
	s.Messages = append(s.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	s.UpdatedAt = time.Now()

	// Set title from first user message if not set
	if s.Title == "" && role == "user" && len(content) > 0 {
		s.Title = truncateTitle(content)
	}
}

// GetLastMessage returns the most recent message
func (s *Session) GetLastMessage() *Message {
	if len(s.Messages) == 0 {
		return nil
	}
	return &s.Messages[len(s.Messages)-1]
}

// ClearMessages removes all messages from the session
func (s *Session) ClearMessages() {
	s.Messages = make([]Message, 0)
	s.UpdatedAt = time.Now()
}

// AddToolExecution records a tool execution in the session
func (s *Session) AddToolExecution(toolName, command, result string, approved bool, durationMs int64) {
	if s.ToolHistory == nil {
		s.ToolHistory = make([]ToolExecution, 0)
	}
	s.ToolHistory = append(s.ToolHistory, ToolExecution{
		Timestamp: time.Now(),
		ToolName:  toolName,
		Command:   command,
		Result:    result,
		Approved:  approved,
		Duration:  durationMs,
	})
	s.UpdatedAt = time.Now()
}

// SetAgentState updates the agent state in the session
func (s *Session) SetAgentState(state string) {
	s.AgentState = state
	s.UpdatedAt = time.Now()
}

// ToMetadata converts session to metadata
func (s *Session) ToMetadata() SessionMetadata {
	return SessionMetadata{
		ID:        s.ID,
		Title:     s.Title,
		Provider:  s.Provider,
		Model:     s.Model,
		Messages:  len(s.Messages),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// generateID creates a unique session ID
func generateID() string {
	return time.Now().Format("20060102-150405")
}

// truncateTitle truncates a string to create a session title
func truncateTitle(s string) string {
	maxLen := 50
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
