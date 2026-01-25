package models

import (
	"sync"
	"testing"
)

func TestNewTable(t *testing.T) {
	table := NewTable()
	if table == nil {
		t.Fatal("NewTable returned nil")
	}
	if table.RowCount() != 0 {
		t.Errorf("RowCount() = %d, want 0", table.RowCount())
	}
}

func TestSetData(t *testing.T) {
	table := NewTable()

	headers := []string{"NAME", "STATUS"}
	rows := [][]string{
		{"pod-1", "Running"},
		{"pod-2", "Pending"},
	}

	table.SetData(headers, rows)

	gotHeaders := table.Headers()
	if len(gotHeaders) != 2 {
		t.Errorf("Headers() len = %d, want 2", len(gotHeaders))
	}
	if gotHeaders[0] != "NAME" {
		t.Errorf("Headers()[0] = %s, want NAME", gotHeaders[0])
	}

	if table.RowCount() != 2 {
		t.Errorf("RowCount() = %d, want 2", table.RowCount())
	}
}

func TestGetRow(t *testing.T) {
	table := NewTable()
	table.SetData(
		[]string{"NAME"},
		[][]string{{"a"}, {"b"}, {"c"}},
	)

	row := table.GetRow(1)
	if row == nil {
		t.Fatal("GetRow returned nil")
	}
	if row[0] != "b" {
		t.Errorf("GetRow(1) = %v, want [b]", row)
	}

	// Test out of bounds
	if table.GetRow(-1) != nil {
		t.Error("GetRow(-1) should return nil")
	}
	if table.GetRow(10) != nil {
		t.Error("GetRow(10) should return nil")
	}
}

type mockListener struct {
	dataChangedCalled bool
	dataFailedCalled  bool
	mx                sync.Mutex
}

func (m *mockListener) DataChanged(headers []string, rows [][]string) {
	m.mx.Lock()
	m.dataChangedCalled = true
	m.mx.Unlock()
}

func (m *mockListener) DataFailed(err error) {
	m.mx.Lock()
	m.dataFailedCalled = true
	m.mx.Unlock()
}

func TestSubscribe(t *testing.T) {
	table := NewTable()
	listener := &mockListener{}

	table.Subscribe(listener)
	table.SetData([]string{"A"}, [][]string{{"1"}})

	if !listener.dataChangedCalled {
		t.Error("DataChanged was not called")
	}
}

func TestUnsubscribe(t *testing.T) {
	table := NewTable()
	listener := &mockListener{}

	table.Subscribe(listener)
	table.Unsubscribe(listener)
	table.SetData([]string{"A"}, [][]string{{"1"}})

	if listener.dataChangedCalled {
		t.Error("DataChanged should not be called after unsubscribe")
	}
}

func TestSetError(t *testing.T) {
	table := NewTable()
	listener := &mockListener{}

	table.Subscribe(listener)
	table.SetError(nil)

	if !listener.dataFailedCalled {
		t.Error("DataFailed was not called")
	}
}

func TestClear(t *testing.T) {
	table := NewTable()
	table.SetData([]string{"A"}, [][]string{{"1"}})

	table.Clear()

	if table.RowCount() != 0 {
		t.Errorf("RowCount() = %d after clear, want 0", table.RowCount())
	}
}

func TestRows(t *testing.T) {
	table := NewTable()
	rows := [][]string{{"a"}, {"b"}}
	table.SetData([]string{"X"}, rows)

	// Modify original
	rows[0][0] = "modified"

	// Should not affect table data
	gotRows := table.Rows()
	if gotRows[0][0] != "a" {
		t.Error("Rows() should return a copy")
	}
}

func TestFilteredTable(t *testing.T) {
	ft := NewFilteredTable()
	ft.SetData(
		[]string{"NAME", "STATUS"},
		[][]string{
			{"nginx", "Running"},
			{"redis", "Pending"},
			{"nginx-proxy", "Running"},
		},
	)

	ft.SetFilter("nginx")

	if ft.FilteredRowCount() != 2 {
		t.Errorf("FilteredRowCount() = %d, want 2", ft.FilteredRowCount())
	}

	filtered := ft.FilteredRows()
	if len(filtered) != 2 {
		t.Errorf("FilteredRows() len = %d, want 2", len(filtered))
	}
}

func TestFilteredTableCaseInsensitive(t *testing.T) {
	ft := NewFilteredTable()
	ft.SetData(
		[]string{"NAME"},
		[][]string{
			{"Nginx"},
			{"Redis"},
		},
	)

	ft.SetFilter("nginx")

	if ft.FilteredRowCount() != 1 {
		t.Errorf("FilteredRowCount() = %d, want 1 (case insensitive)", ft.FilteredRowCount())
	}
}

func TestFilteredTableEmptyFilter(t *testing.T) {
	ft := NewFilteredTable()
	ft.SetData(
		[]string{"NAME"},
		[][]string{{"a"}, {"b"}, {"c"}},
	)

	ft.SetFilter("")

	if ft.FilteredRowCount() != 3 {
		t.Errorf("FilteredRowCount() = %d, want 3 (empty filter)", ft.FilteredRowCount())
	}
}

func TestOriginalIndex(t *testing.T) {
	ft := NewFilteredTable()
	ft.SetData(
		[]string{"NAME"},
		[][]string{{"a"}, {"b"}, {"c"}},
	)

	ft.SetFilter("b")

	idx := ft.OriginalIndex(0)
	if idx != 1 {
		t.Errorf("OriginalIndex(0) = %d, want 1", idx)
	}

	// Out of bounds
	if ft.OriginalIndex(-1) != -1 {
		t.Error("OriginalIndex(-1) should return -1")
	}
	if ft.OriginalIndex(10) != -1 {
		t.Error("OriginalIndex(10) should return -1")
	}
}

func TestGetFilter(t *testing.T) {
	ft := NewFilteredTable()
	ft.SetFilter("test")

	if ft.GetFilter() != "test" {
		t.Errorf("GetFilter() = %s, want test", ft.GetFilter())
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"Hello World", "world", true},
		{"Hello World", "HELLO", true},
		{"Hello World", "xyz", false},
		{"", "a", false},
		{"a", "", true},
		{"nginx-pod", "nginx", true},
		{"NGINX-POD", "nginx", true},
	}

	for _, tt := range tests {
		got := containsIgnoreCase(tt.s, tt.substr)
		if got != tt.expected {
			t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.expected)
		}
	}
}
