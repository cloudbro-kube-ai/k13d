package sessions

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()

	// Test Save and Load
	session := &Session{
		ID:        "test-123",
		Provider:  "openai",
		Model:     "gpt-4",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
	}

	if err := store.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load("test-123")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.ID != session.ID {
		t.Errorf("Load().ID = %v, want %v", loaded.ID, session.ID)
	}

	if len(loaded.Messages) != 2 {
		t.Errorf("Load().Messages length = %v, want 2", len(loaded.Messages))
	}

	// Test List
	list, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 1 {
		t.Errorf("List() length = %v, want 1", len(list))
	}

	// Test Delete
	if err := store.Delete("test-123"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.Load("test-123")
	if err == nil {
		t.Error("Load() after delete should return error")
	}

	// Test Clear
	if err := store.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	list, _ = store.List()
	if len(list) != 0 {
		t.Errorf("List() after clear = %v, want 0", len(list))
	}
}

func TestFileSystemStore(t *testing.T) {
	// Create temp directory
	tmpDir := filepath.Join(os.TempDir(), "k13d-test-sessions")
	defer os.RemoveAll(tmpDir)

	store, err := NewFileSystemStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileSystemStore() error = %v", err)
	}

	// Test Save and Load
	session := &Session{
		ID:        "fs-test-123",
		Provider:  "ollama",
		Model:     "llama2",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
		},
	}

	if err := store.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load("fs-test-123")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Provider != session.Provider {
		t.Errorf("Load().Provider = %v, want %v", loaded.Provider, session.Provider)
	}

	// Test List
	list, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 1 {
		t.Errorf("List() length = %v, want 1", len(list))
	}

	// Test Delete
	if err := store.Delete("fs-test-123"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.Load("fs-test-123")
	if err == nil {
		t.Error("Load() after delete should return error")
	}
}

func TestSession_AddMessage(t *testing.T) {
	session := NewSession("openai", "gpt-4")

	session.AddMessage("user", "Hello")
	session.AddMessage("assistant", "Hi there!")

	if len(session.Messages) != 2 {
		t.Errorf("Messages length = %v, want 2", len(session.Messages))
	}

	if session.Messages[0].Role != "user" {
		t.Errorf("Messages[0].Role = %v, want 'user'", session.Messages[0].Role)
	}

	if session.Messages[1].Content != "Hi there!" {
		t.Errorf("Messages[1].Content = %v, want 'Hi there!'", session.Messages[1].Content)
	}
}

func TestSession_GetLastMessage(t *testing.T) {
	session := NewSession("openai", "gpt-4")

	// Empty session
	if msg := session.GetLastMessage(); msg != nil {
		t.Error("GetLastMessage() on empty session should return nil")
	}

	session.AddMessage("user", "Hello")
	session.AddMessage("assistant", "Hi there!")

	last := session.GetLastMessage()
	if last == nil {
		t.Fatal("GetLastMessage() should not return nil")
	}

	if last.Content != "Hi there!" {
		t.Errorf("GetLastMessage().Content = %v, want 'Hi there!'", last.Content)
	}
}

func TestSession_ToMetadata(t *testing.T) {
	session := NewSession("openai", "gpt-4")
	session.AddMessage("user", "Test message")

	meta := session.ToMetadata()

	if meta.ID != session.ID {
		t.Errorf("ToMetadata().ID = %v, want %v", meta.ID, session.ID)
	}

	if meta.Provider != "openai" {
		t.Errorf("ToMetadata().Provider = %v, want 'openai'", meta.Provider)
	}

	if meta.Messages != 1 {
		t.Errorf("ToMetadata().Messages = %v, want 1", meta.Messages)
	}
}

func TestSession_ClearMessages(t *testing.T) {
	session := NewSession("openai", "gpt-4")
	session.AddMessage("user", "Hello")
	session.AddMessage("assistant", "Hi!")

	session.ClearMessages()

	if len(session.Messages) != 0 {
		t.Errorf("ClearMessages() should empty messages, got %v", len(session.Messages))
	}
}
