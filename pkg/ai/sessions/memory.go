package sessions

import (
	"fmt"
	"sort"
	"sync"
)

// MemoryStore implements Store using in-memory storage
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewMemoryStore creates a new in-memory session store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*Session),
	}
}

// Save persists a session in memory
func (s *MemoryStore) Save(session *Session) error {
	if session == nil || session.ID == "" {
		return fmt.Errorf("invalid session")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Make a copy to avoid external modifications
	copy := *session
	copy.Messages = make([]Message, len(session.Messages))
	for i, msg := range session.Messages {
		copy.Messages[i] = msg
	}

	s.sessions[session.ID] = &copy
	return nil
}

// Load retrieves a session from memory
func (s *MemoryStore) Load(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	// Return a copy to avoid external modifications
	copy := *session
	copy.Messages = make([]Message, len(session.Messages))
	for i, msg := range session.Messages {
		copy.Messages[i] = msg
	}

	return &copy, nil
}

// Delete removes a session from memory
func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
	return nil
}

// List returns all session metadata
func (s *MemoryStore) List() ([]SessionMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]SessionMetadata, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session.ToMetadata())
	}

	// Sort by updated time (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// Clear removes all sessions
func (s *MemoryStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = make(map[string]*Session)
	return nil
}
