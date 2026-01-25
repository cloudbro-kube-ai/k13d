// Package models provides data models for the TUI following k9s patterns.
// Models are responsible for data storage, transformation, and listener notification.
package models

import (
	"sync"
)

// TableListener is notified when table data changes.
type TableListener interface {
	// DataChanged is called when the table data is updated.
	DataChanged(headers []string, rows [][]string)
	// DataFailed is called when data fetching fails.
	DataFailed(err error)
}

// Table represents a generic table data model.
type Table struct {
	headers   []string
	rows      [][]string
	listeners []TableListener
	mx        sync.RWMutex
}

// NewTable creates a new Table model.
func NewTable() *Table {
	return &Table{
		headers:   make([]string, 0),
		rows:      make([][]string, 0),
		listeners: make([]TableListener, 0),
	}
}

// SetData updates the table data and notifies listeners.
func (t *Table) SetData(headers []string, rows [][]string) {
	t.mx.Lock()
	// Make deep copy of headers
	t.headers = make([]string, len(headers))
	copy(t.headers, headers)
	// Make deep copy of rows
	t.rows = make([][]string, len(rows))
	for i, row := range rows {
		t.rows[i] = make([]string, len(row))
		copy(t.rows[i], row)
	}
	listeners := make([]TableListener, len(t.listeners))
	copy(listeners, t.listeners)
	t.mx.Unlock()

	// Notify listeners outside lock
	for _, l := range listeners {
		l.DataChanged(headers, rows)
	}
}

// SetError notifies listeners of a data fetch error.
func (t *Table) SetError(err error) {
	t.mx.RLock()
	listeners := make([]TableListener, len(t.listeners))
	copy(listeners, t.listeners)
	t.mx.RUnlock()

	for _, l := range listeners {
		l.DataFailed(err)
	}
}

// Headers returns the current headers.
func (t *Table) Headers() []string {
	t.mx.RLock()
	defer t.mx.RUnlock()
	result := make([]string, len(t.headers))
	copy(result, t.headers)
	return result
}

// Rows returns the current rows.
func (t *Table) Rows() [][]string {
	t.mx.RLock()
	defer t.mx.RUnlock()
	result := make([][]string, len(t.rows))
	for i, row := range t.rows {
		result[i] = make([]string, len(row))
		copy(result[i], row)
	}
	return result
}

// RowCount returns the number of rows.
func (t *Table) RowCount() int {
	t.mx.RLock()
	defer t.mx.RUnlock()
	return len(t.rows)
}

// GetRow returns a specific row by index.
func (t *Table) GetRow(index int) []string {
	t.mx.RLock()
	defer t.mx.RUnlock()
	if index < 0 || index >= len(t.rows) {
		return nil
	}
	result := make([]string, len(t.rows[index]))
	copy(result, t.rows[index])
	return result
}

// Subscribe adds a listener for data changes.
func (t *Table) Subscribe(listener TableListener) {
	t.mx.Lock()
	defer t.mx.Unlock()
	t.listeners = append(t.listeners, listener)
}

// Unsubscribe removes a listener.
func (t *Table) Unsubscribe(listener TableListener) {
	t.mx.Lock()
	defer t.mx.Unlock()
	for i, l := range t.listeners {
		if l == listener {
			t.listeners = append(t.listeners[:i], t.listeners[i+1:]...)
			return
		}
	}
}

// Clear removes all data.
func (t *Table) Clear() {
	t.SetData(nil, nil)
}

// FilteredTable wraps a Table with filter functionality.
type FilteredTable struct {
	*Table
	filter        string
	filteredRows  [][]string
	filterIndices []int // Maps filtered index to original index
	filterMx      sync.RWMutex
}

// NewFilteredTable creates a new FilteredTable.
func NewFilteredTable() *FilteredTable {
	return &FilteredTable{
		Table:         NewTable(),
		filteredRows:  make([][]string, 0),
		filterIndices: make([]int, 0),
	}
}

// SetFilter sets the filter string and re-filters data.
func (ft *FilteredTable) SetFilter(filter string) {
	ft.filterMx.Lock()
	ft.filter = filter
	ft.filterMx.Unlock()
	ft.applyFilter()
}

// GetFilter returns the current filter.
func (ft *FilteredTable) GetFilter() string {
	ft.filterMx.RLock()
	defer ft.filterMx.RUnlock()
	return ft.filter
}

// applyFilter filters the rows based on the current filter.
func (ft *FilteredTable) applyFilter() {
	ft.mx.RLock()
	rows := ft.rows
	ft.mx.RUnlock()

	ft.filterMx.Lock()
	filter := ft.filter
	ft.filterMx.Unlock()

	if filter == "" {
		ft.filterMx.Lock()
		ft.filteredRows = rows
		ft.filterIndices = make([]int, len(rows))
		for i := range rows {
			ft.filterIndices[i] = i
		}
		ft.filterMx.Unlock()
		return
	}

	filtered := make([][]string, 0)
	indices := make([]int, 0)

	for i, row := range rows {
		for _, cell := range row {
			if containsIgnoreCase(cell, filter) {
				filtered = append(filtered, row)
				indices = append(indices, i)
				break
			}
		}
	}

	ft.filterMx.Lock()
	ft.filteredRows = filtered
	ft.filterIndices = indices
	ft.filterMx.Unlock()
}

// FilteredRows returns the filtered rows.
func (ft *FilteredTable) FilteredRows() [][]string {
	ft.filterMx.RLock()
	defer ft.filterMx.RUnlock()
	result := make([][]string, len(ft.filteredRows))
	for i, row := range ft.filteredRows {
		result[i] = make([]string, len(row))
		copy(result[i], row)
	}
	return result
}

// FilteredRowCount returns the count of filtered rows.
func (ft *FilteredTable) FilteredRowCount() int {
	ft.filterMx.RLock()
	defer ft.filterMx.RUnlock()
	return len(ft.filteredRows)
}

// OriginalIndex returns the original row index for a filtered index.
func (ft *FilteredTable) OriginalIndex(filteredIndex int) int {
	ft.filterMx.RLock()
	defer ft.filterMx.RUnlock()
	if filteredIndex < 0 || filteredIndex >= len(ft.filterIndices) {
		return -1
	}
	return ft.filterIndices[filteredIndex]
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	// Simple case-insensitive search
	sLower := toLower(s)
	substrLower := toLower(substr)

	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase (ASCII only for performance).
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
