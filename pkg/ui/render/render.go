// Package render provides resource rendering following k9s patterns.
// Renderers are responsible for converting Kubernetes objects to table rows.
package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
)

// Column represents a table column definition.
type Column struct {
	Name      string    // Column header name
	Width     int       // Fixed width (0 for auto)
	MinWidth  int       // Minimum width
	MaxWidth  int       // Maximum width (0 for unlimited)
	Align     Alignment // Text alignment
	Hide      bool      // Hidden by default
	MX        bool      // Requires metrics (CPU/MEM)
	Wide      bool      // Only shown in wide mode
	Time      bool      // Contains time/age values
	Highlight bool      // Highlight this column in search
	ColorFn   ColorFunc // Custom color function
}

// Alignment represents text alignment.
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// ColorFunc is a function that returns the color for a cell value.
type ColorFunc func(value string) tcell.Color

// Header is a collection of column definitions.
type Header []Column

// Names returns the column names.
func (h Header) Names() []string {
	names := make([]string, len(h))
	for i, col := range h {
		names[i] = col.Name
	}
	return names
}

// VisibleNames returns names of visible columns.
func (h Header) VisibleNames(wide, mx bool) []string {
	var names []string
	for _, col := range h {
		if col.Hide {
			continue
		}
		if col.Wide && !wide {
			continue
		}
		if col.MX && !mx {
			continue
		}
		names = append(names, col.Name)
	}
	return names
}

// IndexOf returns the index of a column by name.
func (h Header) IndexOf(name string) int {
	for i, col := range h {
		if col.Name == name {
			return i
		}
	}
	return -1
}

// Row represents a table row.
type Row struct {
	ID     string   // Unique identifier (namespace/name)
	Fields []string // Cell values
}

// RowEvent represents a change to a row.
type RowEvent struct {
	Kind RowEventKind
	Row  Row
	Old  Row // For updates, the previous row
}

// RowEventKind represents the type of row change.
type RowEventKind int

const (
	RowAdd RowEventKind = iota
	RowUpdate
	RowDelete
)

// Renderer is the interface for resource renderers.
type Renderer interface {
	// Header returns the column definitions.
	Header() Header
	// ColorerFunc returns the row colorer function.
	ColorerFunc() ColorerFunc
}

// ColorerFunc returns colors for a row based on its values.
type ColorerFunc func(namespace string, row Row) tcell.Color

// BaseRenderer provides common rendering functionality.
type BaseRenderer struct {
	header Header
}

// NewBaseRenderer creates a new BaseRenderer.
func NewBaseRenderer(header Header) *BaseRenderer {
	return &BaseRenderer{header: header}
}

// Header returns the column definitions.
func (r *BaseRenderer) Header() Header {
	return r.header
}

// ColorerFunc returns a default colorer (no coloring).
func (r *BaseRenderer) ColorerFunc() ColorerFunc {
	return func(ns string, row Row) tcell.Color {
		return tcell.ColorDefault
	}
}

// Age formatting utilities

// FormatAge formats a duration as a human-readable age string.
func FormatAge(t time.Time) string {
	if t.IsZero() {
		return "<none>"
	}
	return FormatDuration(time.Since(t))
}

// FormatDuration formats a duration as a human-readable string.
func FormatDuration(d time.Duration) string {
	if d < 0 {
		return "<invalid>"
	}

	d = d.Round(time.Second)
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	switch {
	case days >= 365:
		years := days / 365
		months := (days % 365) / 30
		if months > 0 {
			return fmt.Sprintf("%dy%dM", years, months)
		}
		return fmt.Sprintf("%dy", years)
	case days >= 30:
		months := days / 30
		remainDays := days % 30
		if remainDays > 0 {
			return fmt.Sprintf("%dM%dd", months, remainDays)
		}
		return fmt.Sprintf("%dM", months)
	case days > 0:
		if hours > 0 {
			return fmt.Sprintf("%dd%dh", days, hours)
		}
		return fmt.Sprintf("%dd", days)
	case hours > 0:
		if minutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	case minutes > 0:
		if seconds > 0 {
			return fmt.Sprintf("%dm%ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}

// Status color utilities

// StatusColor returns a color based on status string.
func StatusColor(status string) tcell.Color {
	s := strings.ToLower(status)
	switch {
	case s == "running", s == "active", s == "ready", s == "bound", s == "available":
		return tcell.ColorGreen
	case s == "pending", s == "waiting", s == "creating":
		return tcell.ColorYellow
	case s == "failed", s == "error", s == "crashloopbackoff", s == "imagepullbackoff":
		return tcell.ColorRed
	case s == "terminating", s == "evicted":
		return tcell.ColorOrange
	case s == "completed", s == "succeeded":
		return tcell.ColorBlue
	case s == "unknown":
		return tcell.ColorGray
	default:
		return tcell.ColorDefault
	}
}

// ReadyColor returns a color based on ready count (e.g., "2/3").
func ReadyColor(ready string) tcell.Color {
	parts := strings.Split(ready, "/")
	if len(parts) != 2 {
		return tcell.ColorDefault
	}
	if parts[0] == parts[1] && parts[0] != "0" {
		return tcell.ColorGreen
	}
	if parts[0] == "0" {
		return tcell.ColorRed
	}
	return tcell.ColorYellow
}

// RestartColor returns a color based on restart count.
func RestartColor(restarts string) tcell.Color {
	if restarts == "0" {
		return tcell.ColorDefault
	}
	// High restart count
	if len(restarts) >= 3 || (len(restarts) == 2 && restarts[0] >= '5') {
		return tcell.ColorRed
	}
	if len(restarts) >= 2 || (len(restarts) == 1 && restarts[0] >= '5') {
		return tcell.ColorYellow
	}
	return tcell.ColorDefault
}

// Truncation utilities

// Truncate truncates a string to the specified length.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// TruncateMiddle truncates in the middle, keeping start and end.
func TruncateMiddle(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen <= 5 {
		return Truncate(s, maxLen)
	}
	half := (maxLen - 3) / 2
	return s[:half] + "..." + s[len(s)-half:]
}
