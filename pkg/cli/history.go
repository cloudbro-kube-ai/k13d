package cli

import "sync"

const maxHistoryEntries = 500

type CommandHistory struct {
	mu      sync.RWMutex
	entries []string
	cursor  int
}

func NewCommandHistory() *CommandHistory {
	return &CommandHistory{
		entries: make([]string, 0, maxHistoryEntries),
		cursor:  -1,
	}
}

func (h *CommandHistory) Add(cmd string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if cmd == "" {
		return
	}

	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == cmd {
		h.cursor = len(h.entries)
		return
	}

	if len(h.entries) >= maxHistoryEntries {
		h.entries = h.entries[1:]
	}
	h.entries = append(h.entries, cmd)
	h.cursor = len(h.entries)
}

func (h *CommandHistory) Previous() (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.entries) == 0 || h.cursor <= 0 {
		return "", false
	}
	h.cursor--
	return h.entries[h.cursor], true
}

func (h *CommandHistory) Next() (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.cursor >= len(h.entries)-1 {
		h.cursor = len(h.entries)
		return "", false
	}
	h.cursor++
	return h.entries[h.cursor], true
}

func (h *CommandHistory) Entries() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]string, len(h.entries))
	copy(result, h.entries)
	return result
}

func (h *CommandHistory) ResetCursor() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cursor = len(h.entries)
}

func (h *CommandHistory) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.entries)
}

func (h *CommandHistory) PreviousWithIdx(idx *int) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.entries) == 0 || *idx <= 0 {
		return "", false
	}
	*idx--
	return h.entries[*idx], true
}

func (h *CommandHistory) NextWithIdx(idx *int) (string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if *idx >= len(h.entries)-1 {
		*idx = len(h.entries)
		return "", false
	}
	*idx++
	return h.entries[*idx], true
}
