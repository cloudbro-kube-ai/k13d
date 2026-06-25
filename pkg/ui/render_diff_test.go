package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// newTestTable builds a bare tview.Table for offline rendering tests. We do not
// attach it to a screen, so all operations are pure in-memory mutations.
func newTestTable() *tview.Table {
	return tview.NewTable().SetFixed(1, 0)
}

func cfgFor(resource string, headers []string) tableRendererConfig {
	return tableRendererConfig{
		resource:    resource,
		headers:     headers,
		statusColor: nil,
		isStatusCol: nil,
		headerStyler: func(col int, header string) (string, tcell.Color) {
			return header, tcell.ColorYellow
		},
	}
}

func tableDataText(table *tview.Table, resource string) [][]string {
	count := table.GetRowCount() - dataRowOffset
	if count <= 0 {
		return nil
	}
	out := make([][]string, count)
	for r := 0; r < count; r++ {
		ncols := 0
		// Determine row width by reading until nil cells.
		for {
			c := table.GetCell(dataRowOffset+r, ncols)
			if c == nil {
				break
			}
			if c.Text == "" && ncols > 0 {
				// Heuristic: stop at trailing blank cells.
				break
			}
			ncols++
			if ncols > 32 {
				break
			}
		}
		row := make([]string, ncols)
		for c := 0; c < ncols; c++ {
			cell := table.GetCell(dataRowOffset+r, c)
			if cell != nil {
				row[c] = cell.Text
			}
		}
		out[r] = row
	}
	return out
}

func TestSyncDataRows_InsertFromEmpty(t *testing.T) {
	table := newTestTable()
	cfg := cfgFor("pods", []string{"NS", "NAME", "STATUS"})

	rows := [][]string{
		{"default", "p1", "Running"},
		{"default", "p2", "Pending"},
	}
	keys := []string{"default/p1", "default/p2"}

	n := syncDataRows(table, keys, rows, cfg)
	if n != 2 {
		t.Fatalf("n = %d, want 2", n)
	}
	got := tableDataText(table, "pods")
	if len(got) != 2 || got[0][1] != "p1" || got[1][1] != "p2" {
		t.Fatalf("rows = %v", got)
	}
}

func TestSyncDataRows_UpdateInPlace(t *testing.T) {
	table := newTestTable()
	cfg := cfgFor("pods", []string{"NS", "NAME", "STATUS"})

	// Initial render.
	syncDataRows(table, []string{"default/p1"}, [][]string{{"default", "p1", "Pending"}}, cfg)

	// Capture the cell pointer for p1's STATUS to assert it changed in place.
	oldCell := table.GetCell(dataRowOffset+0, 2)

	// Update STATUS only.
	n := syncDataRows(table, []string{"default/p1"}, [][]string{{"default", "p1", "Running"}}, cfg)
	if n != 1 {
		t.Fatalf("n = %d, want 1", n)
	}
	got := tableDataText(table, "pods")
	if got[0][2] != "Running" {
		t.Fatalf("status not updated: %v", got[0])
	}

	newCell := table.GetCell(dataRowOffset+0, 2)
	if oldCell == newCell {
		t.Fatalf("expected STATUS cell to be replaced because text changed")
	}
}

func TestSyncDataRows_NoChangeKeepsCell(t *testing.T) {
	table := newTestTable()
	cfg := cfgFor("pods", []string{"NS", "NAME", "STATUS"})

	syncDataRows(table, []string{"default/p1"}, [][]string{{"default", "p1", "Running"}}, cfg)
	oldCell := table.GetCell(dataRowOffset+0, 1)

	// Re-render identical data.
	syncDataRows(table, []string{"default/p1"}, [][]string{{"default", "p1", "Running"}}, cfg)

	newCell := table.GetCell(dataRowOffset+0, 1)
	if oldCell != newCell {
		t.Fatalf("expected NAME cell to be reused when text is unchanged")
	}
}

func TestSyncDataRows_RemoveRow(t *testing.T) {
	table := newTestTable()
	cfg := cfgFor("pods", []string{"NS", "NAME", "STATUS"})

	syncDataRows(table,
		[]string{"default/p1", "default/p2", "default/p3"},
		[][]string{
			{"default", "p1", "Running"},
			{"default", "p2", "Running"},
			{"default", "p3", "Running"},
		}, cfg)

	// Remove the middle row.
	n := syncDataRows(table,
		[]string{"default/p1", "default/p3"},
		[][]string{
			{"default", "p1", "Running"},
			{"default", "p3", "Running"},
		}, cfg)
	if n != 2 {
		t.Fatalf("n = %d, want 2", n)
	}
	got := tableDataText(table, "pods")
	if len(got) != 2 || got[0][1] != "p1" || got[1][1] != "p3" {
		t.Fatalf("after removal rows = %v", got)
	}
}

func TestSyncDataRows_ReorderRow(t *testing.T) {
	table := newTestTable()
	cfg := cfgFor("pods", []string{"NS", "NAME", "STATUS"})

	syncDataRows(table,
		[]string{"default/p1", "default/p2"},
		[][]string{
			{"default", "p1", "Running"},
			{"default", "p2", "Running"},
		}, cfg)

	// Swap order.
	n := syncDataRows(table,
		[]string{"default/p2", "default/p1"},
		[][]string{
			{"default", "p2", "Running"},
			{"default", "p1", "Running"},
		}, cfg)
	if n != 2 {
		t.Fatalf("n = %d, want 2", n)
	}
	got := tableDataText(table, "pods")
	if len(got) != 2 || got[0][1] != "p2" || got[1][1] != "p1" {
		t.Fatalf("reordered rows = %v", got)
	}
}

func TestSyncDataRows_ClusterScopedKey(t *testing.T) {
	table := newTestTable()
	cfg := cfgFor("nodes", []string{"NAME", "STATUS"})

	n := syncDataRows(table,
		[]string{"node-1", "node-2"},
		[][]string{
			{"node-1", "Ready"},
			{"node-2", "Ready"},
		}, cfg)
	if n != 2 {
		t.Fatalf("n = %d, want 2", n)
	}

	// Remove node-1 (cluster-scoped key = row[0]).
	n = syncDataRows(table,
		[]string{"node-2"},
		[][]string{{"node-2", "Ready"}}, cfg)
	if n != 1 {
		t.Fatalf("n = %d, want 1", n)
	}
	got := tableDataText(table, "nodes")
	if len(got) != 1 || got[0][0] != "node-2" {
		t.Fatalf("after removal rows = %v", got)
	}
}

func TestSyncHeaders_SkipUnchanged(t *testing.T) {
	table := newTestTable()
	cfg := cfgFor("pods", []string{"NS", "NAME"})

	syncHeaders(table, []string{"NS", "NAME"}, cfg)
	oldNSCell := table.GetCell(0, 0)

	// Re-sync identical headers.
	syncHeaders(table, []string{"NS", "NAME"}, cfg)
	newNSCell := table.GetCell(0, 0)
	if oldNSCell != newNSCell {
		t.Fatalf("expected header cell to be reused when unchanged")
	}
}
