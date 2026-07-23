package ui

import (
	"sort"
	"sync"
)

// KeyedRow pairs a stable identity key with its display row.
// The key uniquely identifies a row within a resource view (see rowKey).
type KeyedRow struct {
	Key string
	Row []string
}

// ResourceStore is an in-memory, insertion-ordered store of resource rows keyed
// by a stable identity. It is the single source of truth for the table model and
// supports both full-snapshot replacement (initial load, relist, fallback) and
// incremental upsert/delete (watch deltas).
//
// The store is safe for concurrent use. UI-facing rendering reads a snapshot via
// Snapshot() and performs diffing on the calling (UI) goroutine.
type ResourceStore struct {
	mu sync.RWMutex

	headers []string
	order   []string            // insertion order of keys (preserves stable display order)
	rows    map[string][]string // key -> row cells
}

// NewResourceStore returns an empty ResourceStore.
func NewResourceStore() *ResourceStore {
	return &ResourceStore{
		headers: nil,
		order:   make([]string, 0),
		rows:    make(map[string][]string),
	}
}

// SetFull replaces the entire store contents with the provided headers and rows.
// The previous selection order is discarded. Rows are stored in the given order.
// New rows are deep-copied so callers cannot mutate store state after insertion.
func (s *ResourceStore) SetFull(headers []string, rows []KeyedRow) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.headers = copyStrings(headers)
	s.order = make([]string, 0, len(rows))
	s.rows = make(map[string][]string, len(rows))
	for _, kr := range rows {
		if _, exists := s.rows[kr.Key]; !exists {
			s.order = append(s.order, kr.Key)
		}
		s.rows[kr.Key] = copyStrings(kr.Row)
	}
}

// Upsert inserts or updates a single row by key. When inserting a new key, it is
// appended to the end of the display order. Returns true if the key was newly
// added (vs updated in place).
func (s *ResourceStore) Upsert(key string, row []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, existed := s.rows[key]
	s.rows[key] = copyStrings(row)
	if !existed {
		s.order = append(s.order, key)
	}
	return !existed
}

// Delete removes a row by key. Returns true if the key was present.
func (s *ResourceStore) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.rows[key]; !ok {
		return false
	}
	delete(s.rows, key)
	for i, k := range s.order {
		if k == key {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
}

// Reset clears all rows and headers without reallocating internal buffers.
func (s *ResourceStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.headers = nil
	s.order = s.order[:0]
	s.rows = make(map[string][]string)
}

// SetHeaders replaces only the headers, leaving rows intact.
func (s *ResourceStore) SetHeaders(headers []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.headers = copyStrings(headers)
}

// Headers returns a copy of the current headers.
func (s *ResourceStore) Headers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyStrings(s.headers)
}

// Snapshot returns the current headers and an ordered, deep-copied slice of rows.
// The key list (second return) preserves display order and is used for diffing.
func (s *ResourceStore) Snapshot() (headers []string, keys []string, rows [][]string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	headers = copyStrings(s.headers)
	keys = make([]string, len(s.order))
	copy(keys, s.order)
	rows = make([][]string, len(s.order))
	for i, k := range s.order {
		rows[i] = copyStrings(s.rows[k])
	}
	return headers, keys, rows
}

// RowByKey returns a copy of the row for the given key, or nil if not found.
func (s *ResourceStore) RowByKey(key string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	row, ok := s.rows[key]
	if !ok {
		return nil
	}
	return copyStrings(row)
}

// IndexOf returns the display-order index of key, or -1 if absent.
func (s *ResourceStore) IndexOf(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i, k := range s.order {
		if k == key {
			return i
		}
	}
	return -1
}

// KeyAt returns the key at display-order index idx, or "" if out of range.
func (s *ResourceStore) KeyAt(idx int) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if idx < 0 || idx >= len(s.order) {
		return ""
	}
	return s.order[idx]
}

// Len returns the number of stored rows.
func (s *ResourceStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.order)
}

// Sort reorders rows in place by the given comparator. The comparator receives
// full row slices and their keys; it must be consistent across calls.
func (s *ResourceStore) Sort(less func(aKey, bKey string, aRow, bRow []string) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sort.SliceStable(s.order, func(i, j int) bool {
		ki, kj := s.order[i], s.order[j]
		return less(ki, kj, s.rows[ki], s.rows[kj])
	})
}

// rowKey returns the stable identity key for a row in the given resource view.
// It mirrors nameColumnIndex() semantics: cluster-scoped resources are keyed by
// their single name column; namespace-scoped resources by NAMESPACE/NAME.
//
// Using a single helper here (shared by the store, row builders, and diff
// renderer) prevents key mismatches between layers.
func rowKey(resource string, row []string) string {
	if len(row) == 0 {
		return ""
	}
	if nameColumnIndex(resource) == 0 {
		return row[0]
	}
	if len(row) > 1 {
		return row[0] + "/" + row[1]
	}
	return row[0]
}

// keyedRows converts a flat [][]string list into KeyedRow entries using rowKey.
// Rows whose key cannot be derived (empty) are still included with their
// computed key so that display parity is preserved.
func keyedRows(resource string, rows [][]string) []KeyedRow {
	out := make([]KeyedRow, len(rows))
	for i, r := range rows {
		out[i] = KeyedRow{Key: rowKey(resource, r), Row: r}
	}
	return out
}

// copyStrings returns a defensive copy of s (nil-safe).
func copyStrings(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)
	return out
}
