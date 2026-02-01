package testutil

import (
	"context"
	"sync"
)

// MockLLMProvider is a reusable mock for LLM providers.
// Use this instead of creating ad-hoc mocks in each test file.
type MockLLMProvider struct {
	mu sync.Mutex

	NameValue      string
	ModelValue     string
	ReadyValue     bool
	AskError       error
	AskContent     string
	StreamChunks   []string // For testing streaming behavior
	ModelsValue    []string
	ModelsError    error
	CallCount      int
	LastPrompt     string
	SupportsTools  bool
	ToolCallsCount int
	ToolError      error
}

// Name returns the provider name.
func (m *MockLLMProvider) Name() string {
	return m.NameValue
}

// GetModel returns the model name.
func (m *MockLLMProvider) GetModel() string {
	return m.ModelValue
}

// IsReady returns whether the provider is ready.
func (m *MockLLMProvider) IsReady() bool {
	return m.ReadyValue
}

// ListModels returns available models.
func (m *MockLLMProvider) ListModels(_ context.Context) ([]string, error) {
	return m.ModelsValue, m.ModelsError
}

// Ask simulates a streaming LLM call.
func (m *MockLLMProvider) Ask(_ context.Context, prompt string, callback func(string)) error {
	m.mu.Lock()
	m.CallCount++
	m.LastPrompt = prompt
	m.mu.Unlock()

	if m.AskError != nil {
		return m.AskError
	}
	if callback != nil {
		if len(m.StreamChunks) > 0 {
			for _, chunk := range m.StreamChunks {
				callback(chunk)
			}
		} else if m.AskContent != "" {
			callback(m.AskContent)
		}
	}
	return nil
}

// AskNonStreaming simulates a non-streaming LLM call.
func (m *MockLLMProvider) AskNonStreaming(_ context.Context, prompt string) (string, error) {
	m.mu.Lock()
	m.CallCount++
	m.LastPrompt = prompt
	m.mu.Unlock()

	if m.AskError != nil {
		return "", m.AskError
	}
	return m.AskContent, nil
}

// GetCallCount returns the number of calls made (thread-safe).
func (m *MockLLMProvider) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.CallCount
}

// GetLastPrompt returns the last prompt (thread-safe).
func (m *MockLLMProvider) GetLastPrompt() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.LastPrompt
}

// Reset clears all recorded state.
func (m *MockLLMProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallCount = 0
	m.LastPrompt = ""
}

// MockAuditLogger is a mock audit logger for testing.
type MockAuditLogger struct {
	mu      sync.Mutex
	Entries []AuditEntry
	LogErr  error
}

// Log records an audit entry.
func (m *MockAuditLogger) Log(action, resource, details string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.LogErr != nil {
		return m.LogErr
	}
	m.Entries = append(m.Entries, AuditEntry{
		Action:   action,
		Resource: resource,
		Details:  details,
	})
	return nil
}

// Query returns filtered audit entries.
func (m *MockAuditLogger) Query(filter AuditFilter) ([]AuditEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var results []AuditEntry
	for _, e := range m.Entries {
		if filter.Action != "" && e.Action != filter.Action {
			continue
		}
		if filter.Resource != "" && e.Resource != filter.Resource {
			continue
		}
		results = append(results, e)
	}
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}
	return results, nil
}

// GetEntries returns all logged entries (thread-safe).
func (m *MockAuditLogger) GetEntries() []AuditEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]AuditEntry, len(m.Entries))
	copy(result, m.Entries)
	return result
}

// Reset clears all entries.
func (m *MockAuditLogger) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Entries = nil
}

// MockSessionStore is a mock session store for testing.
type MockSessionStore struct {
	mu       sync.Mutex
	Sessions map[string]interface{}
	Err      error
}

// NewMockSessionStore creates a new mock session store.
func NewMockSessionStore() *MockSessionStore {
	return &MockSessionStore{Sessions: make(map[string]interface{})}
}

// Create creates a new session.
func (m *MockSessionStore) Create(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	m.Sessions[id] = struct{}{}
	return nil
}

// Get retrieves a session.
func (m *MockSessionStore) Get(id string) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	s, ok := m.Sessions[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}

// Delete removes a session.
func (m *MockSessionStore) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	delete(m.Sessions, id)
	return nil
}

// List returns all session IDs.
func (m *MockSessionStore) List() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	var ids []string
	for id := range m.Sessions {
		ids = append(ids, id)
	}
	return ids, nil
}
