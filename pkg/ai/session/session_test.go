package session

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudbro-kube-ai/k13d/pkg/db"
)

// setupTestDB initializes a fresh SQLite database for testing
func setupTestDB(t *testing.T) *Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	if err := db.Init(dbPath); err != nil {
		t.Fatalf("Failed to init test DB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := NewStore()
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	return store
}

func TestStore_Create(t *testing.T) {
	store := setupTestDB(t)

	session, err := store.Create("openai", "gpt-4")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}
	if session.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", session.Provider)
	}
	if session.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", session.Model)
	}
	if session.Title != "New Conversation" {
		t.Errorf("Expected title 'New Conversation', got '%s'", session.Title)
	}
}

func TestStore_AddMessage(t *testing.T) {
	store := setupTestDB(t)

	session, err := store.Create("openai", "gpt-4")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Add user message
	err = store.AddMessage(session.ID, Message{
		Role:    "user",
		Content: "List all pods in the default namespace",
	})
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	// Verify message was added and title updated
	updated, err := store.Get(session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if len(updated.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(updated.Messages))
	}
	if updated.Messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", updated.Messages[0].Role)
	}
	if updated.Title == "New Conversation" {
		t.Error("Title should have been updated from first message")
	}
	if updated.MessageCount != 1 {
		t.Errorf("Expected MessageCount 1, got %d", updated.MessageCount)
	}
}

func TestStore_AddMessageWithToolCalls(t *testing.T) {
	store := setupTestDB(t)

	session, err := store.Create("openai", "gpt-4")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Add assistant message with tool calls
	err = store.AddMessage(session.ID, Message{
		Role:    "assistant",
		Content: "Let me check the pods for you.",
		ToolCalls: []ToolCallRecord{
			{
				ID:        "call_123",
				Name:      "kubectl",
				Arguments: `{"command": "kubectl get pods"}`,
				Result:    "pod1\npod2\npod3",
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	updated, err := store.Get(session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if len(updated.Messages[0].ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call, got %d", len(updated.Messages[0].ToolCalls))
	}
	if updated.Messages[0].ToolCalls[0].Name != "kubectl" {
		t.Errorf("Expected tool name 'kubectl', got '%s'", updated.Messages[0].ToolCalls[0].Name)
	}
}

func TestStore_List(t *testing.T) {
	store := setupTestDB(t)

	// Create multiple sessions
	session1, _ := store.Create("openai", "gpt-4")
	time.Sleep(10 * time.Millisecond)
	session2, _ := store.Create("ollama", "llama2")

	// Add message to session1 to update its timestamp
	time.Sleep(10 * time.Millisecond)
	if err := store.AddMessage(session1.ID, Message{Role: "user", Content: "Test message"}); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}

	summaries, err := store.List()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(summaries) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(summaries))
	}

	// Session1 should be first (most recently updated)
	if summaries[0].ID != session1.ID {
		t.Error("Sessions should be sorted by update time (newest first)")
	}

	// Check that session2 is second
	if summaries[1].ID != session2.ID {
		t.Error("Second session should be session2")
	}
}

func TestStore_Delete(t *testing.T) {
	store := setupTestDB(t)

	session, _ := store.Create("openai", "gpt-4")

	err := store.Delete(session.ID)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify session was deleted
	_, err = store.Get(session.ID)
	if err == nil {
		t.Error("Session should not exist after deletion")
	}

	// Delete non-existent session should error
	err = store.Delete("non-existent")
	if err == nil {
		t.Error("Deleting non-existent session should return error")
	}
}

func TestStore_Clear(t *testing.T) {
	store := setupTestDB(t)

	// Create multiple sessions
	if _, err := store.Create("openai", "gpt-4"); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if _, err := store.Create("ollama", "llama2"); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if _, err := store.Create("gemini", "gemini-pro"); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	err := store.Clear()
	if err != nil {
		t.Fatalf("Failed to clear sessions: %v", err)
	}

	summaries, _ := store.List()
	if len(summaries) != 0 {
		t.Errorf("Expected 0 sessions after clear, got %d", len(summaries))
	}
}

func TestStore_UpdateTitle(t *testing.T) {
	store := setupTestDB(t)

	session, _ := store.Create("openai", "gpt-4")

	err := store.UpdateTitle(session.ID, "My Custom Title")
	if err != nil {
		t.Fatalf("Failed to update title: %v", err)
	}

	updated, _ := store.Get(session.ID)
	if updated.Title != "My Custom Title" {
		t.Errorf("Expected title 'My Custom Title', got '%s'", updated.Title)
	}
}

func TestStore_GetMessages(t *testing.T) {
	store := setupTestDB(t)

	session, _ := store.Create("openai", "gpt-4")

	// Add multiple messages
	for i := 0; i < 10; i++ {
		if err := store.AddMessage(session.ID, Message{
			Role:    "user",
			Content: "Message " + string(rune('0'+i)),
		}); err != nil {
			t.Fatalf("Failed to add message %d: %v", i, err)
		}
	}

	// Test pagination
	messages, err := store.GetMessages(session.ID, 3, 0)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Test offset
	messages, err = store.GetMessages(session.ID, 3, 5)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Test offset beyond messages
	messages, err = store.GetMessages(session.ID, 3, 100)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages when offset exceeds total, got %d", len(messages))
	}

	// Test no limit (get all)
	messages, err = store.GetMessages(session.ID, 0, 0)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}
	if len(messages) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(messages))
	}
}

func TestStore_GetRecentSessions(t *testing.T) {
	store := setupTestDB(t)

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		if _, err := store.Create("openai", "gpt-4"); err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Get recent 3
	recent, err := store.GetRecentSessions(3)
	if err != nil {
		t.Fatalf("Failed to get recent sessions: %v", err)
	}
	if len(recent) != 3 {
		t.Errorf("Expected 3 recent sessions, got %d", len(recent))
	}
}

func TestStore_ExportImport(t *testing.T) {
	store := setupTestDB(t)

	// Create and populate a session
	session, _ := store.Create("openai", "gpt-4")
	if err := store.AddMessage(session.ID, Message{Role: "user", Content: "Hello"}); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}
	if err := store.AddMessage(session.ID, Message{Role: "assistant", Content: "Hi there!"}); err != nil {
		t.Fatalf("Failed to add message: %v", err)
	}
	if err := store.UpdateTitle(session.ID, "Test Export"); err != nil {
		t.Fatalf("Failed to update title: %v", err)
	}

	// Export
	data, err := store.Export(session.ID)
	if err != nil {
		t.Fatalf("Failed to export session: %v", err)
	}

	// Import (same store, new ID)
	imported, err := store.Import(data)
	if err != nil {
		t.Fatalf("Failed to import session: %v", err)
	}

	// Verify imported session
	if imported.Title != "Test Export" {
		t.Errorf("Expected title 'Test Export', got '%s'", imported.Title)
	}
	if len(imported.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(imported.Messages))
	}
	if imported.ID == session.ID {
		t.Error("Imported session should have new ID")
	}
}

func TestStore_GetContextMessages(t *testing.T) {
	store := setupTestDB(t)

	session, _ := store.Create("openai", "gpt-4")

	// Add 10 messages
	for i := 0; i < 10; i++ {
		if err := store.AddMessage(session.ID, Message{
			Role:    "user",
			Content: "Message " + string(rune('A'+i)),
		}); err != nil {
			t.Fatalf("Failed to add message %d: %v", i, err)
		}
	}

	// Get last 5 messages for context
	messages, err := store.GetContextMessages(session.ID, 5)
	if err != nil {
		t.Fatalf("Failed to get context messages: %v", err)
	}

	if len(messages) != 5 {
		t.Errorf("Expected 5 context messages, got %d", len(messages))
	}

	// Should be the last 5 messages (F, G, H, I, J)
	if messages[0].Content != "Message F" {
		t.Errorf("Expected first context message to be 'Message F', got '%s'", messages[0].Content)
	}
}

func TestGenerateTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello world", "Hello world"},
		{"Short", "Short"},
		{"", "New Conversation"},
		{"This is a very long message that exceeds the fifty character limit for titles", "This is a very long message that exceeds the fi..."},
		{"First line\nSecond line", "First line"},
		{"   Whitespace   \n  around  ", "Whitespace"},
	}

	for _, test := range tests {
		result := generateTitle(test.input)
		if result != test.expected {
			t.Errorf("generateTitle(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store := setupTestDB(t)

	session, _ := store.Create("openai", "gpt-4")

	// Sequential writes (SQLite is single-writer; concurrent transactions cause SQLITE_BUSY)
	for i := 0; i < 10; i++ {
		if err := store.AddMessage(session.ID, Message{
			Role:    "user",
			Content: "Message " + string(rune('0'+i)),
		}); err != nil {
			t.Fatalf("AddMessage %d failed: %v", i, err)
		}
	}

	// Verify all messages were added
	updated, _ := store.Get(session.ID)
	if len(updated.Messages) != 10 {
		t.Errorf("Expected 10 messages, got %d", len(updated.Messages))
	}
}

// Enterprise security tests

func TestValidateSessionID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid alphanumeric", "abc123", false},
		{"valid with hyphen", "session-123", false},
		{"valid with underscore", "session_123", false},
		{"empty string", "", true},
		{"path traversal attempt", "../etc/passwd", true},
		{"path traversal attempt 2", "..%2F..%2Fetc", true},
		{"contains slash", "session/id", true},
		{"contains backslash", "session\\id", true},
		{"too long", string(make([]byte, 100)), true},
		{"special characters", "session<script>", true},
		{"null byte", "session\x00id", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSessionID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSessionID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Normal title", "Normal title"},
		{"", "New Conversation"},
		{"   Whitespace   ", "Whitespace"},
		{"Title\nwith\nnewlines", "Titlewith newlines"},
		{"Title\twith\ttabs", "Title\twith\ttabs"},
		{string(make([]byte, 200)), string(make([]byte, 97)) + "..."},
		{"Title with \x00 null", "Title with  null"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeTitle(tt.input)
			if len(result) > MaxTitleLength {
				t.Errorf("sanitizeTitle result too long: %d > %d", len(result), MaxTitleLength)
			}
		})
	}
}

func TestStore_PathTraversalPrevention(t *testing.T) {
	store := setupTestDB(t)

	maliciousIDs := []string{
		"../../../etc/passwd",
		"..\\..\\windows\\system32",
		"valid/../../../etc/passwd",
		"session%2F..%2F..%2Fetc",
	}

	for _, id := range maliciousIDs {
		_, err := store.Get(id)
		if err == nil {
			t.Errorf("Expected error for malicious ID %q, got nil", id)
		}
	}
}

func TestStore_InvalidMessageRole(t *testing.T) {
	store := setupTestDB(t)

	session, _ := store.Create("openai", "gpt-4")

	err := store.AddMessage(session.ID, Message{
		Role:    "invalid_role",
		Content: "Test message",
	})
	if err == nil {
		t.Error("Expected error for invalid message role, got nil")
	}
}

func TestStore_MessageContentLimit(t *testing.T) {
	store := setupTestDB(t)

	session, _ := store.Create("openai", "gpt-4")

	// Create a message that exceeds the limit
	largeContent := string(make([]byte, MaxMessageContentLength+1))
	err := store.AddMessage(session.ID, Message{
		Role:    "user",
		Content: largeContent,
	})
	if err == nil {
		t.Error("Expected error for oversized message content, got nil")
	}
}

func TestStore_Count(t *testing.T) {
	store := setupTestDB(t)

	// Initial count should be 0
	count, err := store.Count()
	if err != nil {
		t.Fatalf("Failed to count sessions: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 sessions, got %d", count)
	}

	// Create some sessions
	if _, err := store.Create("openai", "gpt-4"); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if _, err := store.Create("ollama", "llama2"); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	if _, err := store.Create("gemini", "gemini-pro"); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	count, err = store.Count()
	if err != nil {
		t.Fatalf("Failed to count sessions: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 sessions, got %d", count)
	}
}

func TestStore_GetBaseDir(t *testing.T) {
	store := setupTestDB(t)

	if store.GetBaseDir() != "SQLite database" {
		t.Errorf("Expected 'SQLite database', got '%s'", store.GetBaseDir())
	}
}

func TestStore_RapidWrites(t *testing.T) {
	store := setupTestDB(t)

	session, _ := store.Create("openai", "gpt-4")

	// Add multiple messages rapidly
	for i := 0; i < 100; i++ {
		err := store.AddMessage(session.ID, Message{
			Role:    "user",
			Content: "Message " + string(rune('0'+i%10)),
		})
		if err != nil {
			t.Fatalf("Failed to add message %d: %v", i, err)
		}
	}

	// Verify session is still valid and readable
	updated, err := store.Get(session.ID)
	if err != nil {
		t.Fatalf("Failed to get session after rapid writes: %v", err)
	}
	if len(updated.Messages) != 100 {
		t.Errorf("Expected 100 messages, got %d", len(updated.Messages))
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := generateID()
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true

		// Verify ID format is valid
		if err := validateSessionID(id); err != nil {
			t.Errorf("Generated ID failed validation: %v", err)
		}
	}
}

func TestStore_ImportInvalidJSON(t *testing.T) {
	store := setupTestDB(t)

	invalidJSONs := [][]byte{
		[]byte("not json"),
		[]byte("{invalid json}"),
		[]byte(""),
		nil,
	}

	for _, data := range invalidJSONs {
		_, err := store.Import(data)
		if err == nil {
			t.Errorf("Expected error for invalid JSON %q, got nil", string(data))
		}
	}
}

func TestStore_GetNonExistentSession(t *testing.T) {
	store := setupTestDB(t)

	_, err := store.Get("nonexistent-session-id")
	if err == nil {
		t.Error("Expected error for non-existent session, got nil")
	}
}

func TestStore_DeleteNonExistentSession(t *testing.T) {
	store := setupTestDB(t)

	err := store.Delete("nonexistent-session-id")
	if err == nil {
		t.Error("Expected error for deleting non-existent session, got nil")
	}
}

func TestStore_AddMessageToNonExistentSession(t *testing.T) {
	store := setupTestDB(t)

	err := store.AddMessage("nonexistent-session-id", Message{
		Role:    "user",
		Content: "Hello",
	})
	if err == nil {
		t.Error("Expected error for adding message to non-existent session, got nil")
	}
}
