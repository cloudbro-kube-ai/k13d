package db

import (
	"database/sql"
	"fmt"
	"time"
)

// ChatSession represents a conversation session stored in SQLite
type ChatSession struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Model        string        `json:"model"`
	Provider     string        `json:"provider"`
	Messages     []ChatMessage `json:"messages"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	MessageCount int           `json:"message_count"`
}

// ChatSessionSummary is a lightweight representation for listing sessions
type ChatSessionSummary struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Model        string    `json:"model"`
	Provider     string    `json:"provider"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
}

// ChatMessage represents a single message in a conversation
type ChatMessage struct {
	ID         int64     `json:"id"`
	SessionID  string    `json:"session_id"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	ToolCalls  string    `json:"tool_calls,omitempty"`
	ToolCallID string    `json:"tool_call_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	SortOrder  int       `json:"sort_order"`
}

// InitChatSessionsTable creates the chat_sessions and chat_messages tables.
func InitChatSessionsTable() error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	sessionsQuery := `
	CREATE TABLE IF NOT EXISTS chat_sessions (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT 'New Conversation',
		model TEXT DEFAULT '',
		provider TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		message_count INTEGER DEFAULT 0
	);`

	if _, err := DB.Exec(sessionsQuery); err != nil {
		return fmt.Errorf("failed to create chat_sessions table: %w", err)
	}

	messagesQuery := `
	CREATE TABLE IF NOT EXISTS chat_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL DEFAULT '',
		tool_calls TEXT DEFAULT '',
		tool_call_id TEXT DEFAULT '',
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		sort_order INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
	);`

	if _, err := DB.Exec(messagesQuery); err != nil {
		return fmt.Errorf("failed to create chat_messages table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_chat_sessions_updated ON chat_sessions(updated_at DESC);",
		"CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id, sort_order ASC);",
	}
	for _, q := range indexes {
		if _, err := DB.Exec(q); err != nil {
			fmt.Printf("Warning: chat index creation: %v\n", err)
		}
	}

	return nil
}

// CreateChatSession inserts a new chat session
func CreateChatSession(id, title, provider, model string) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	now := time.Now()
	_, err := DB.Exec(
		`INSERT INTO chat_sessions (id, title, provider, model, created_at, updated_at, message_count)
		 VALUES (?, ?, ?, ?, ?, ?, 0)`,
		id, title, provider, model, now, now,
	)
	return err
}

// GetChatSession retrieves a session by ID with all messages
func GetChatSession(id string) (*ChatSession, error) {
	if DB == nil {
		return nil, ErrDBNotInitialized
	}

	session := &ChatSession{}
	err := DB.QueryRow(
		`SELECT id, title, model, provider, created_at, updated_at, message_count
		 FROM chat_sessions WHERE id = ?`, id,
	).Scan(&session.ID, &session.Title, &session.Model, &session.Provider,
		&session.CreatedAt, &session.UpdatedAt, &session.MessageCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, err
	}

	// Load messages
	messages, err := GetChatMessages(id, 0, 0)
	if err != nil {
		return nil, err
	}
	session.Messages = messages

	return session, nil
}

// ListChatSessions returns all session summaries, ordered by updated_at DESC
func ListChatSessions() ([]ChatSessionSummary, error) {
	if DB == nil {
		return nil, ErrDBNotInitialized
	}

	rows, err := DB.Query(
		`SELECT id, title, model, provider, created_at, updated_at, message_count
		 FROM chat_sessions ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []ChatSessionSummary
	for rows.Next() {
		var s ChatSessionSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.Model, &s.Provider,
			&s.CreatedAt, &s.UpdatedAt, &s.MessageCount); err != nil {
			continue
		}
		summaries = append(summaries, s)
	}
	if summaries == nil {
		summaries = []ChatSessionSummary{}
	}
	return summaries, nil
}

// UpdateChatSessionTitle updates the title of a session
func UpdateChatSessionTitle(id, title string) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	result, err := DB.Exec(
		`UPDATE chat_sessions SET title = ?, updated_at = ? WHERE id = ?`,
		title, time.Now(), id,
	)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("session not found: %s", id)
	}
	return nil
}

// DeleteChatSession deletes a session and its messages (via CASCADE)
func DeleteChatSession(id string) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	// Delete messages first (in case foreign keys are not enabled)
	if _, err := DB.Exec(`DELETE FROM chat_messages WHERE session_id = ?`, id); err != nil {
		return err
	}
	result, err := DB.Exec(`DELETE FROM chat_sessions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("session not found: %s", id)
	}
	return nil
}

// ClearChatSessions deletes all sessions and messages
func ClearChatSessions() error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM chat_messages`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM chat_sessions`); err != nil {
		return err
	}
	return tx.Commit()
}

// CountChatSessions returns the total number of sessions
func CountChatSessions() (int, error) {
	if DB == nil {
		return 0, ErrDBNotInitialized
	}

	var count int
	err := DB.QueryRow(`SELECT COUNT(*) FROM chat_sessions`).Scan(&count)
	return count, err
}

// AddChatMessage inserts a message and updates session metadata in a transaction
func AddChatMessage(sessionID, role, content, toolCallsJSON, toolCallID string) error {
	if DB == nil {
		return ErrDBNotInitialized
	}

	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Get next sort_order
	var maxOrder int
	err = tx.QueryRow(
		`SELECT COALESCE(MAX(sort_order), 0) FROM chat_messages WHERE session_id = ?`,
		sessionID,
	).Scan(&maxOrder)
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = tx.Exec(
		`INSERT INTO chat_messages (session_id, role, content, tool_calls, tool_call_id, timestamp, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sessionID, role, content, toolCallsJSON, toolCallID, now, maxOrder+1,
	)
	if err != nil {
		return err
	}

	// Update session metadata
	_, err = tx.Exec(
		`UPDATE chat_sessions SET message_count = message_count + 1, updated_at = ? WHERE id = ?`,
		now, sessionID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetChatMessages retrieves messages for a session with optional limit/offset.
// If limit <= 0, all messages are returned.
func GetChatMessages(sessionID string, limit, offset int) ([]ChatMessage, error) {
	if DB == nil {
		return nil, ErrDBNotInitialized
	}

	var rows *sql.Rows
	var err error

	if limit > 0 {
		rows, err = DB.Query(
			`SELECT id, session_id, role, content, tool_calls, tool_call_id, timestamp, sort_order
			 FROM chat_messages WHERE session_id = ?
			 ORDER BY sort_order ASC LIMIT ? OFFSET ?`,
			sessionID, limit, offset,
		)
	} else {
		rows, err = DB.Query(
			`SELECT id, session_id, role, content, tool_calls, tool_call_id, timestamp, sort_order
			 FROM chat_messages WHERE session_id = ?
			 ORDER BY sort_order ASC`,
			sessionID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content,
			&m.ToolCalls, &m.ToolCallID, &m.Timestamp, &m.SortOrder); err != nil {
			continue
		}
		messages = append(messages, m)
	}
	if messages == nil {
		messages = []ChatMessage{}
	}
	return messages, nil
}

// GetRecentChatMessages returns the last N messages for a session (for LLM context)
func GetRecentChatMessages(sessionID string, maxMessages int) ([]ChatMessage, error) {
	if DB == nil {
		return nil, ErrDBNotInitialized
	}

	if maxMessages <= 0 {
		return GetChatMessages(sessionID, 0, 0)
	}

	// Use subquery to get last N messages, then order ascending
	rows, err := DB.Query(
		`SELECT id, session_id, role, content, tool_calls, tool_call_id, timestamp, sort_order
		 FROM (
		     SELECT id, session_id, role, content, tool_calls, tool_call_id, timestamp, sort_order
		     FROM chat_messages WHERE session_id = ?
		     ORDER BY sort_order DESC LIMIT ?
		 ) sub ORDER BY sort_order ASC`,
		sessionID, maxMessages,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content,
			&m.ToolCalls, &m.ToolCallID, &m.Timestamp, &m.SortOrder); err != nil {
			continue
		}
		messages = append(messages, m)
	}
	if messages == nil {
		messages = []ChatMessage{}
	}
	return messages, nil
}
