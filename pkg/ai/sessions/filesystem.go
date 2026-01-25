package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileSystemStore implements Store using the filesystem
type FileSystemStore struct {
	baseDir string
}

// NewFileSystemStore creates a new filesystem-based session store
func NewFileSystemStore(baseDir string) (*FileSystemStore, error) {
	if baseDir == "" {
		// Default to ~/.config/k13d/sessions
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, ".config", "k13d", "sessions")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	return &FileSystemStore{baseDir: baseDir}, nil
}

// Save persists a session to disk
func (s *FileSystemStore) Save(session *Session) error {
	if session == nil || session.ID == "" {
		return fmt.Errorf("invalid session")
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	path := s.sessionPath(session.ID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// Load retrieves a session from disk
func (s *FileSystemStore) Load(id string) (*Session, error) {
	path := s.sessionPath(id)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// Delete removes a session from disk
func (s *FileSystemStore) Delete(id string) error {
	path := s.sessionPath(id)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	return nil
}

// List returns all session metadata
func (s *FileSystemStore) List() ([]SessionMetadata, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SessionMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []SessionMetadata
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		session, err := s.Load(id)
		if err != nil {
			continue // Skip corrupted files
		}

		sessions = append(sessions, session.ToMetadata())
	}

	// Sort by updated time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// Clear removes all sessions
func (s *FileSystemStore) Clear() error {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read sessions directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(s.baseDir, entry.Name())
		if err := os.Remove(path); err != nil {
			// Continue even if some files fail to delete
			continue
		}
	}

	return nil
}

func (s *FileSystemStore) sessionPath(id string) string {
	return filepath.Join(s.baseDir, id+".json")
}
