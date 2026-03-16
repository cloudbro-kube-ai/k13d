// Package session provides AI conversation session persistence backed by SQLite
package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/cloudbro-kube-ai/k13d/pkg/db"
)

const (
	// MaxTitleLength is the maximum length for session titles
	MaxTitleLength = 100
	// MaxMessageContentLength is the maximum length for message content (1MB)
	MaxMessageContentLength = 1024 * 1024
	// MaxMessagesPerSession limits messages per session for memory safety
	MaxMessagesPerSession = 10000
)

// validSessionIDPattern matches valid session IDs (alphanumeric and hyphen only)
var validSessionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// Message represents a single message in a conversation
type Message struct {
	Role       string           `json:"role"`                   // "user", "assistant", "system", "tool"
	Content    string           `json:"content"`                // Message content
	ToolCalls  []ToolCallRecord `json:"tool_calls,omitempty"`   // Tool calls made by assistant
	ToolCallID string           `json:"tool_call_id,omitempty"` // For tool response messages
	Timestamp  time.Time        `json:"timestamp"`
}

// ToolCallRecord stores information about a tool call
type ToolCallRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Result    string `json:"result,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// Session represents a conversation session
type Session struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`    // Auto-generated from first message
	Model        string    `json:"model"`    // LLM model used
	Provider     string    `json:"provider"` // LLM provider
	Messages     []Message `json:"messages"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
}

// SessionSummary is a lightweight representation for listing sessions
type SessionSummary struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Model        string    `json:"model"`
	Provider     string    `json:"provider"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
}

// Store manages session persistence via SQLite
type Store struct{}

// NewStore creates a new SQLite-backed session store.
// It also migrates any legacy file-based sessions to SQLite.
func NewStore() (*Store, error) {
	store := &Store{}

	// Migrate legacy file-based sessions to SQLite
	legacyDir := filepath.Join(xdg.DataHome, "k13d", "sessions")
	if info, err := os.Stat(legacyDir); err == nil && info.IsDir() {
		migrated, migrateErr := migrateFileSessions(legacyDir)
		if migrateErr != nil {
			fmt.Printf("  Session migration warning: %v\n", migrateErr)
		} else if migrated > 0 {
			fmt.Printf("  Session migration: migrated %d file sessions to SQLite\n", migrated)
		}
	}

	return store, nil
}

// generateID creates a cryptographically secure unique session ID
func generateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// validateSessionID checks if a session ID is valid and safe
func validateSessionID(id string) error {
	if id == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if len(id) > 64 {
		return fmt.Errorf("session ID too long (max 64 characters)")
	}
	if !validSessionIDPattern.MatchString(id) {
		return fmt.Errorf("session ID contains invalid characters")
	}
	if strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("session ID contains invalid path characters")
	}
	return nil
}

// sanitizeTitle ensures title is safe and within limits
func sanitizeTitle(title string) string {
	title = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, title)
	title = strings.TrimSpace(title)

	if len(title) > MaxTitleLength {
		title = title[:MaxTitleLength-3] + "..."
	}
	if title == "" {
		title = "New Conversation"
	}
	return title
}

// generateTitle creates a title from the first user message
func generateTitle(content string) string {
	lines := strings.SplitN(content, "\n", 2)
	title := strings.TrimSpace(lines[0])
	if len(title) > 50 {
		title = title[:47] + "..."
	}
	return sanitizeTitle(title)
}

// Create creates a new session
func (s *Store) Create(provider, model string) (*Session, error) {
	id := generateID()
	title := "New Conversation"
	now := time.Now()

	if err := db.CreateChatSession(id, title, provider, model); err != nil {
		return nil, err
	}

	return &Session{
		ID:        id,
		Title:     title,
		Model:     model,
		Provider:  provider,
		Messages:  []Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Get retrieves a session by ID
func (s *Store) Get(id string) (*Session, error) {
	if err := validateSessionID(id); err != nil {
		return nil, err
	}

	dbSession, err := db.GetChatSession(id)
	if err != nil {
		return nil, err
	}

	return dbSessionToSession(dbSession), nil
}

// AddMessage adds a message to a session
func (s *Store) AddMessage(sessionID string, msg Message) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}

	if len(msg.Content) > MaxMessageContentLength {
		return fmt.Errorf("message content exceeds maximum length (%d bytes)", MaxMessageContentLength)
	}

	validRoles := map[string]bool{"user": true, "assistant": true, "system": true, "tool": true}
	if !validRoles[msg.Role] {
		return fmt.Errorf("invalid message role: %s", msg.Role)
	}

	// Serialize tool calls to JSON
	toolCallsJSON := ""
	if len(msg.ToolCalls) > 0 {
		data, err := json.Marshal(msg.ToolCalls)
		if err != nil {
			return fmt.Errorf("failed to serialize tool calls: %w", err)
		}
		toolCallsJSON = string(data)
	}

	if err := db.AddChatMessage(sessionID, msg.Role, msg.Content, toolCallsJSON, msg.ToolCallID); err != nil {
		return err
	}

	// Update title from first user message
	if msg.Role == "user" && msg.Content != "" {
		dbSession, err := db.GetChatSession(sessionID)
		if err == nil && dbSession.Title == "New Conversation" {
			title := generateTitle(msg.Content)
			_ = db.UpdateChatSessionTitle(sessionID, title)
		}
	}

	return nil
}

// List returns all session summaries, sorted by update time (newest first)
func (s *Store) List() ([]SessionSummary, error) {
	dbSummaries, err := db.ListChatSessions()
	if err != nil {
		return nil, err
	}

	summaries := make([]SessionSummary, len(dbSummaries))
	for i, s := range dbSummaries {
		summaries[i] = SessionSummary{
			ID:           s.ID,
			Title:        s.Title,
			Model:        s.Model,
			Provider:     s.Provider,
			CreatedAt:    s.CreatedAt,
			UpdatedAt:    s.UpdatedAt,
			MessageCount: s.MessageCount,
		}
	}
	return summaries, nil
}

// Delete removes a session
func (s *Store) Delete(id string) error {
	if err := validateSessionID(id); err != nil {
		return err
	}
	return db.DeleteChatSession(id)
}

// Clear removes all sessions
func (s *Store) Clear() error {
	return db.ClearChatSessions()
}

// UpdateTitle updates the session title
func (s *Store) UpdateTitle(id, title string) error {
	if err := validateSessionID(id); err != nil {
		return err
	}
	return db.UpdateChatSessionTitle(id, sanitizeTitle(title))
}

// GetMessages returns messages for a session with optional pagination
func (s *Store) GetMessages(sessionID string, limit, offset int) ([]Message, error) {
	if err := validateSessionID(sessionID); err != nil {
		return nil, err
	}

	dbMessages, err := db.GetChatMessages(sessionID, limit, offset)
	if err != nil {
		return nil, err
	}

	return dbMessagesToMessages(dbMessages), nil
}

// GetRecentSessions returns the most recent n sessions
func (s *Store) GetRecentSessions(n int) ([]SessionSummary, error) {
	summaries, err := s.List()
	if err != nil {
		return nil, err
	}

	if n > 0 && len(summaries) > n {
		summaries = summaries[:n]
	}

	return summaries, nil
}

// Export exports a session to JSON format
func (s *Store) Export(id string) ([]byte, error) {
	if err := validateSessionID(id); err != nil {
		return nil, err
	}

	session, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(session, "", "  ")
}

// Import imports a session from JSON data
func (s *Store) Import(data []byte) (*Session, error) {
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session data: %w", err)
	}

	// Generate new ID to avoid conflicts
	session.ID = generateID()
	session.UpdatedAt = time.Now()

	if err := db.CreateChatSession(session.ID, session.Title, session.Provider, session.Model); err != nil {
		return nil, err
	}

	// Import all messages
	for _, msg := range session.Messages {
		toolCallsJSON := ""
		if len(msg.ToolCalls) > 0 {
			tc, _ := json.Marshal(msg.ToolCalls)
			toolCallsJSON = string(tc)
		}
		if err := db.AddChatMessage(session.ID, msg.Role, msg.Content, toolCallsJSON, msg.ToolCallID); err != nil {
			return nil, err
		}
	}

	return &session, nil
}

// GetContextMessages returns messages formatted for LLM context
func (s *Store) GetContextMessages(sessionID string, maxMessages int) ([]Message, error) {
	if err := validateSessionID(sessionID); err != nil {
		return nil, err
	}

	dbMessages, err := db.GetRecentChatMessages(sessionID, maxMessages)
	if err != nil {
		return nil, err
	}

	return dbMessagesToMessages(dbMessages), nil
}

// Count returns the total number of sessions
func (s *Store) Count() (int, error) {
	return db.CountChatSessions()
}

// GetBaseDir returns a description of the storage backend
func (s *Store) GetBaseDir() string {
	return "SQLite database"
}

// dbSessionToSession converts a db.ChatSession to a session.Session
func dbSessionToSession(dbs *db.ChatSession) *Session {
	messages := dbMessagesToMessages(dbs.Messages)

	return &Session{
		ID:           dbs.ID,
		Title:        dbs.Title,
		Model:        dbs.Model,
		Provider:     dbs.Provider,
		Messages:     messages,
		CreatedAt:    dbs.CreatedAt,
		UpdatedAt:    dbs.UpdatedAt,
		MessageCount: dbs.MessageCount,
	}
}

// dbMessagesToMessages converts db.ChatMessage slice to session.Message slice
func dbMessagesToMessages(dbMsgs []db.ChatMessage) []Message {
	messages := make([]Message, len(dbMsgs))
	for i, m := range dbMsgs {
		var toolCalls []ToolCallRecord
		if m.ToolCalls != "" {
			_ = json.Unmarshal([]byte(m.ToolCalls), &toolCalls)
		}
		messages[i] = Message{
			Role:       m.Role,
			Content:    m.Content,
			ToolCalls:  toolCalls,
			ToolCallID: m.ToolCallID,
			Timestamp:  m.Timestamp,
		}
	}
	return messages
}

// migrateFileSessions migrates legacy JSON file sessions to SQLite
func migrateFileSessions(legacyDir string) (int, error) {
	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		return 0, err
	}

	migrated := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		path := filepath.Join(legacyDir, entry.Name())

		// Skip if already in SQLite
		if _, err := db.GetChatSession(id); err == nil {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		// Insert session
		if err := db.CreateChatSession(session.ID, session.Title, session.Provider, session.Model); err != nil {
			continue
		}

		// Insert messages
		for _, msg := range session.Messages {
			toolCallsJSON := ""
			if len(msg.ToolCalls) > 0 {
				tc, _ := json.Marshal(msg.ToolCalls)
				toolCallsJSON = string(tc)
			}
			_ = db.AddChatMessage(session.ID, msg.Role, msg.Content, toolCallsJSON, msg.ToolCallID)
		}

		// Rename migrated file
		_ = os.Rename(path, path+".migrated")
		migrated++
	}

	return migrated, nil
}
