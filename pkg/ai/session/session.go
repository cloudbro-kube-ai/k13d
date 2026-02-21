// Package session provides AI conversation session persistence
package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
)

const (
	// MaxTitleLength is the maximum length for session titles
	MaxTitleLength = 100
	// MaxMessageContentLength is the maximum length for message content (1MB)
	MaxMessageContentLength = 1024 * 1024
	// MaxMessagesPerSession limits messages per session for memory safety
	MaxMessagesPerSession = 10000
	// SessionFilePermission is the file permission for session files (owner read/write only)
	SessionFilePermission = 0600
	// SessionDirPermission is the directory permission for session directory
	SessionDirPermission = 0700
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

// Store manages session persistence
type Store struct {
	baseDir string
	mu      sync.RWMutex
}

// NewStore creates a new session store
func NewStore() (*Store, error) {
	baseDir := filepath.Join(xdg.DataHome, "k13d", "sessions")
	if err := os.MkdirAll(baseDir, SessionDirPermission); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	return &Store{
		baseDir: baseDir,
	}, nil
}

// NewStoreWithDir creates a session store with a custom directory (for testing)
func NewStoreWithDir(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, SessionDirPermission); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	return &Store{
		baseDir: dir,
	}, nil
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
	// Prevent path traversal
	if strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("session ID contains invalid path characters")
	}
	return nil
}

// sanitizeTitle ensures title is safe and within limits
func sanitizeTitle(title string) string {
	// Remove control characters and trim
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
	// Take first 50 chars of the first line
	lines := strings.SplitN(content, "\n", 2)
	title := strings.TrimSpace(lines[0])
	if len(title) > 50 {
		title = title[:47] + "..."
	}
	return sanitizeTitle(title)
}

// Create creates a new session
func (s *Store) Create(provider, model string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &Session{
		ID:        generateID(),
		Title:     "New Conversation",
		Model:     model,
		Provider:  provider,
		Messages:  []Message{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.saveSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

// Get retrieves a session by ID
func (s *Store) Get(id string) (*Session, error) {
	if err := validateSessionID(id); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.loadSession(id)
}

// AddMessage adds a message to a session
func (s *Store) AddMessage(sessionID string, msg Message) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}

	// Validate message content length
	if len(msg.Content) > MaxMessageContentLength {
		return fmt.Errorf("message content exceeds maximum length (%d bytes)", MaxMessageContentLength)
	}

	// Validate role
	validRoles := map[string]bool{"user": true, "assistant": true, "system": true, "tool": true}
	if !validRoles[msg.Role] {
		return fmt.Errorf("invalid message role: %s", msg.Role)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSession(sessionID)
	if err != nil {
		return err
	}

	// Check message limit
	if len(session.Messages) >= MaxMessagesPerSession {
		return fmt.Errorf("session has reached maximum message limit (%d)", MaxMessagesPerSession)
	}

	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	session.Messages = append(session.Messages, msg)
	session.UpdatedAt = time.Now()
	session.MessageCount = len(session.Messages)

	// Update title from first user message
	if session.Title == "New Conversation" && msg.Role == "user" && msg.Content != "" {
		session.Title = generateTitle(msg.Content)
	}

	return s.saveSession(session)
}

// List returns all session summaries, sorted by update time (newest first)
func (s *Store) List() ([]SessionSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var summaries []SessionSummary
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			id := strings.TrimSuffix(entry.Name(), ".json")
			session, err := s.loadSession(id)
			if err != nil {
				continue // Skip invalid sessions
			}

			summaries = append(summaries, SessionSummary{
				ID:           session.ID,
				Title:        session.Title,
				Model:        session.Model,
				Provider:     session.Provider,
				CreatedAt:    session.CreatedAt,
				UpdatedAt:    session.UpdatedAt,
				MessageCount: session.MessageCount,
			})
		}
	}

	// Sort by UpdatedAt descending
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
	})

	return summaries, nil
}

// Delete removes a session
func (s *Store) Delete(id string) error {
	if err := validateSessionID(id); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.baseDir, id+".json")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session not found: %s", id)
		}
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// Clear removes all sessions
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read session directory: %w", err)
	}

	var firstErr error
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			path := filepath.Join(s.baseDir, entry.Name())
			if err := os.Remove(path); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("failed to remove session file %s: %w", entry.Name(), err)
			}
		}
	}

	return firstErr
}

// UpdateTitle updates the session title
func (s *Store) UpdateTitle(id, title string) error {
	if err := validateSessionID(id); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, err := s.loadSession(id)
	if err != nil {
		return err
	}

	session.Title = sanitizeTitle(title)
	session.UpdatedAt = time.Now()

	return s.saveSession(session)
}

// GetMessages returns messages for a session with optional pagination
func (s *Store) GetMessages(sessionID string, limit, offset int) ([]Message, error) {
	if err := validateSessionID(sessionID); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	session, err := s.loadSession(sessionID)
	if err != nil {
		return nil, err
	}

	messages := session.Messages
	total := len(messages)

	if offset >= total {
		return []Message{}, nil
	}

	end := offset + limit
	if end > total || limit <= 0 {
		end = total
	}

	return messages[offset:end], nil
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

	s.mu.RLock()
	defer s.mu.RUnlock()

	session, err := s.loadSession(id)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(session, "", "  ")
}

// Import imports a session from JSON data
func (s *Store) Import(data []byte) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session data: %w", err)
	}

	// Generate new ID to avoid conflicts
	session.ID = generateID()
	session.UpdatedAt = time.Now()

	if err := s.saveSession(&session); err != nil {
		return nil, err
	}

	return &session, nil
}

// saveSession saves a session to disk with atomic write
func (s *Store) saveSession(session *Session) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	path := filepath.Join(s.baseDir, session.ID+".json")
	tmpPath := path + ".tmp"

	// Write to temporary file first (atomic write pattern)
	if err := os.WriteFile(tmpPath, data, SessionFilePermission); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	// Rename to final path (atomic on most filesystems)
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Clean up temp file on failure
		return fmt.Errorf("failed to finalize session file: %w", err)
	}

	return nil
}

// loadSession loads a session from disk
func (s *Store) loadSession(id string) (*Session, error) {
	path := filepath.Join(s.baseDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return &session, nil
}

// GetContextMessages returns messages formatted for LLM context
// It returns messages in chronological order, optionally limiting to recent messages
func (s *Store) GetContextMessages(sessionID string, maxMessages int) ([]Message, error) {
	if err := validateSessionID(sessionID); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	session, err := s.loadSession(sessionID)
	if err != nil {
		return nil, err
	}

	messages := session.Messages
	if maxMessages > 0 && len(messages) > maxMessages {
		messages = messages[len(messages)-maxMessages:]
	}

	return messages, nil
}

// GetBaseDir returns the base directory for session storage (for debugging/admin)
func (s *Store) GetBaseDir() string {
	return s.baseDir
}

// Count returns the total number of sessions
func (s *Store) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read session directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			count++
		}
	}
	return count, nil
}
