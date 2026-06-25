package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// render_diff.go implements flicker-free, incremental table rendering.
//
// Instead of clearing the table and rebuilding every cell on each refresh
// (which causes a visible flash), syncDataRows reconciles the tview Table
// against a target list of rows using row keys:
//
//   - rows that disappeared are removed via RemoveRow
//   - rows that appeared are inserted via InsertRow at the correct position
//   - rows present in both are updated in place, cell by cell, but only when
//     the displayed text actually changed (this lets tview's internal dirty
//     tracking skip unchanged cells entirely)
//
// Row identity is derived from rowKey() (NAMESPACE/NAME or NAME). The header
// row lives at index 0 (see SetFixed(1, 0)), so data rows are offset by 1.

// tableRendererConfig carries the per-render inputs needed to (re)build cells.
// Keeping these as a struct avoids leaking half the App into the renderer and
// makes the diff logic unit-testable in isolation.
type tableRendererConfig struct {
	resource     string
	headers      []string
	statusColor  func(status string) tcell.Color // nil-safe: defaults to white
	isStatusCol  func(col int) bool              // nil-safe: defaults to false
	headerStyler func(col int, header string) (displayHeader string, color tcell.Color)
	cellStyler   func(rowIdx int, row []string, col int, text string) (displayText string, color tcell.Color)
}

// dataRowOffset is the index of the first data row (header occupies row 0).
const dataRowOffset = 1

// syncHeaders writes the header row at index 0. Only cells whose text changed
// are refreshed to minimize redraw work.
func syncHeaders(table *tview.Table, headers []string, cfg tableRendererConfig) {
	for i, h := range headers {
		display := h
		color := tcell.ColorYellow
		if cfg.headerStyler != nil {
			display, color = cfg.headerStyler(i, h)
		}
		if existing := table.GetCell(0, i); existing != nil && existing.Text == display {
			// Avoid touching the cell so tview can skip it during draw.
			continue
		}
		cell := tview.NewTableCell(display).
			SetTextColor(color).
			SetAttributes(tcell.AttrBold).
			SetSelectable(false).
			SetExpansion(1)
		table.SetCell(0, i, cell)
	}
	// Remove any extra header columns left over from a previous schema.
	for c := len(headers); c < table.GetColumnCount(); c++ {
		// Overwrite stale header cell text to blank rather than structural removal
		// (tview has no RemoveColumn in this version); SetCell keeps it harmless.
		if existing := table.GetCell(0, c); existing == nil || existing.Text != "" {
			table.SetCell(0, c, tview.NewTableCell("").SetSelectable(false))
		}
	}
}

// syncDataRows reconciles the table's data rows (starting at dataRowOffset)
// with the target rows/keys. The optional oldKeysHint lets callers supply the
// previously-rendered keys (e.g. from the prior snapshot) so removals can be
// detected even when the table is empty (such as in tests).
//
// It returns the number of data rows now rendered.
func syncDataRows(table *tview.Table, keys []string, rows [][]string, cfg tableRendererConfig) int {
	// Build the set of target keys for existence checks.
	want := make(map[string]bool, len(keys))
	for _, k := range keys {
		want[k] = true
	}

	// First pass: remove rows whose key is no longer wanted. Walk from the
	// bottom up so indices of not-yet-processed rows stay valid.
	currentCount := table.GetRowCount() - dataRowOffset
	if currentCount < 0 {
		currentCount = 0
	}
	for r := currentCount - 1; r >= 0; r-- {
		k := cellKeyAt(table, r, cfg.resource)
		if k == "" || !want[k] {
			table.RemoveRow(dataRowOffset + r)
		}
	}

	// Second pass: insert/update target rows in order.
	for targetIdx, key := range keys {
		row := rows[targetIdx]
		tableRow := dataRowOffset + targetIdx

		// Recalculate count since prior iterations may have inserted/removed rows.
		currentCount := table.GetRowCount() - dataRowOffset
		if currentCount < 0 {
			currentCount = 0
		}

		// Find an existing row with this key at or after the target position.
		existingIdx := findRowByKeyFrom(table, key, cfg.resource, targetIdx)

		if existingIdx == -1 {
			// Key not present anywhere: insert a new row at the target position.
			table.InsertRow(tableRow)
			writeRowCells(table, tableRow, row, cfg, nil)
			continue
		}

		if existingIdx != targetIdx {
			// Key exists but is out of position. Move it by removing and
			// reinserting at the correct index.
			table.RemoveRow(dataRowOffset + existingIdx)
			table.InsertRow(tableRow)
			writeRowCells(table, tableRow, row, cfg, nil)
			continue
		}

		// In place: update only changed cells.
		writeRowCells(table, tableRow, row, cfg, nil)
	}

	// Trim any trailing rows beyond the target count (e.g. after dedup or when
	// the table had more rows than keys and none matched for removal above).
	for table.GetRowCount()-dataRowOffset > len(keys) {
		table.RemoveRow(table.GetRowCount() - 1)
	}

	return len(keys)
}

// findRowByKeyFrom searches data rows from startIdx (inclusive) for a row whose
// key matches. Returns the data-row index (0-based among data rows) or -1.
// Note: count is recalculated on each iteration to handle rows being inserted
// or removed during traversal.
func findRowByKeyFrom(table *tview.Table, key, resource string, startIdx int) int {
	for r := startIdx; ; r++ {
		count := table.GetRowCount() - dataRowOffset
		if count < 0 {
			count = 0
		}
		if r >= count {
			break
		}
		if cellKeyAt(table, r, resource) == key {
			return r
		}
	}
	return -1
}

// cellKeyAt derives the identity key for the data row at the given 0-based data
// index by reading cells from the table. Returns "" if the row is empty/missing.
func cellKeyAt(table *tview.Table, dataIdx int, resource string) string {
	nameIdx := nameColumnIndex(resource)
	ns := table.GetCell(dataRowOffset+dataIdx, 0)
	if nameIdx == 0 {
		if ns == nil {
			return ""
		}
		return ns.Text
	}
	name := table.GetCell(dataRowOffset+dataIdx, 1)
	if ns == nil || name == nil {
		return ""
	}
	return ns.Text + "/" + name.Text
}

// readRowCells reads up to n cells from a data row as plain text.
func readRowCells(table *tview.Table, dataIdx, n int) []string {
	out := make([]string, n)
	for c := 0; c < n; c++ {
		cell := table.GetCell(dataRowOffset+dataIdx, c)
		if cell != nil {
			out[c] = cell.Text
		}
	}
	return out
}

// writeRowCells writes each cell of a row, skipping cells whose text already
// matches (to keep tview's dirty tracking effective). If prev is non-nil, it is
// used as the previous cell text so we can detect changes reliably (cells
// written in the same frame may not yet reflect their old text via GetCell).
func writeRowCells(table *tview.Table, tableRow int, row []string, cfg tableRendererConfig, prev []string) {
	for c, text := range row {
		color := tcell.ColorWhite
		if cfg.isStatusCol != nil && cfg.isStatusCol(c) {
			if cfg.statusColor != nil {
				color = cfg.statusColor(text)
			}
		}
		displayText := text
		if cfg.cellStyler != nil {
			displayText, color = cfg.cellStyler(tableRow-dataRowOffset, row, c, text)
		}

		var prevText string
		if c < len(prev) {
			prevText = prev[c]
		} else {
			if existing := table.GetCell(tableRow, c); existing != nil {
				prevText = existing.Text
			}
		}

		// Skip untouched cells. Comparing against the raw (non-highlighted) text
		// is sufficient: cellStyler is deterministic, so equal raw text yields
		// equal display text.
		if displayText == prevText {
			continue
		}

		cell := tview.NewTableCell(displayText).
			SetTextColor(color).
			SetExpansion(1)
		table.SetCell(tableRow, c, cell)
	}
}

// applyDiffRender is the App-facing entry point that renders the current store
// snapshot into the main table without a full clear. It preserves the user's
// selection by mapping the selected row to its key before rendering and
// restoring it afterward. Must be called on the UI goroutine.
func (a *App) applyDiffRender() {
	if a.table == nil {
		return
	}

	a.mx.RLock()
	resource := a.currentResource
	sortCol := a.sortColumn
	sortAsc := a.sortAscending
	a.mx.RUnlock()

	headers, keys, rows := a.store.Snapshot()

	// Capture the selected key so we can restore focus after structural changes.
	selRow, _ := a.table.GetSelection()
	var selKey string
	if selRow >= dataRowOffset {
		selKey = cellKeyAt(a.table, selRow-dataRowOffset, resource)
	}

	cfg := a.rendererConfig(resource, headers, sortCol, sortAsc)
	syncHeaders(a.table, headers, cfg)
	count := syncDataRows(a.table, keys, rows, cfg)

	// Restore selection: prefer the same key, else clamp to the first data row.
	a.restoreSelection(selKey, count)

	a.table.SetTitle(fmt.Sprintf(" %s (%d) ", resource, count))
}

// rendererConfig builds the per-render configuration used by syncDataRows,
// reusing the App's status-color and sort-arrow logic.
func (a *App) rendererConfig(resource string, headers []string, sortCol int, sortAsc bool) tableRendererConfig {
	headerColor := tcell.NewRGBColor(224, 175, 104)
	sortColor := tcell.NewRGBColor(125, 207, 255)
	headerBg := tcell.NewRGBColor(36, 40, 59)

	return tableRendererConfig{
		resource:    resource,
		headers:     headers,
		statusColor: a.statusColor,
		isStatusCol: a.isStatusColumn,
		headerStyler: func(col int, header string) (string, tcell.Color) {
			display := header
			color := headerColor
			if col == sortCol {
				if sortAsc {
					display = header + " ▲"
				} else {
					display = header + " ▼"
				}
				color = sortColor
			}
			_ = headerBg // header bg handled via cell style below when needed
			return display, color
		},
	}
}

// restoreSelection re-selects the row identified by selKey, clamping safely.
func (a *App) restoreSelection(selKey string, dataCount int) {
	if dataCount <= 0 {
		return
	}
	if selKey != "" {
		for r := 0; r < dataCount; r++ {
			if cellKeyAt(a.table, r, a.currentResource) == selKey {
				a.table.Select(dataRowOffset+r, 0)
				return
			}
		}
	}
	// Fallback: select the first data row (matches legacy behavior).
	a.table.Select(dataRowOffset, 0)
}

// trimPrefix is a tiny helper kept local to avoid importing strings just for it.
var _ = strings.TrimSpace
