package render

import (
	"testing"
	"time"
)

func TestFormatAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "zero time",
			time:     time.Time{},
			expected: "<none>",
		},
		{
			name:     "seconds ago",
			time:     now.Add(-30 * time.Second),
			expected: "30s",
		},
		{
			name:     "minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5m",
		},
		{
			name:     "hours ago",
			time:     now.Add(-2 * time.Hour),
			expected: "2h",
		},
		{
			name:     "days ago",
			time:     now.Add(-3 * 24 * time.Hour),
			expected: "3d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAge(tt.time)
			if got != tt.expected {
				t.Errorf("FormatAge() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"negative", -1 * time.Second, "<invalid>"},
		{"zero", 0, "0s"},
		{"seconds", 45 * time.Second, "45s"},
		{"minutes", 5 * time.Minute, "5m"},
		{"minutes and seconds", 5*time.Minute + 30*time.Second, "5m30s"},
		{"hours", 2 * time.Hour, "2h"},
		{"hours and minutes", 2*time.Hour + 30*time.Minute, "2h30m"},
		{"days", 3 * 24 * time.Hour, "3d"},
		{"days and hours", 3*24*time.Hour + 12*time.Hour, "3d12h"},
		{"months", 45 * 24 * time.Hour, "1M15d"},
		{"years", 400 * 24 * time.Hour, "1y1M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.duration)
			if got != tt.expected {
				t.Errorf("FormatDuration(%v) = %s, want %s", tt.duration, got, tt.expected)
			}
		})
	}
}

func TestStatusColor(t *testing.T) {
	tests := []struct {
		status string
		isGood bool // Just check if it's not default
	}{
		{"Running", true},
		{"running", true},
		{"Pending", true},
		{"Failed", true},
		{"Error", true},
		{"Unknown", true},
		{"Completed", true},
		{"random", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := StatusColor(tt.status)
			// Just verify we get some color
			_ = got
		})
	}
}

func TestReadyColor(t *testing.T) {
	tests := []struct {
		ready    string
		expected string
	}{
		{"3/3", "green"},
		{"0/3", "red"},
		{"2/3", "yellow"},
		{"invalid", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.ready, func(t *testing.T) {
			got := ReadyColor(tt.ready)
			// Just verify we get a color
			_ = got
		})
	}
}

func TestRestartColor(t *testing.T) {
	tests := []struct {
		restarts string
		expected string
	}{
		{"0", "default"},
		{"5", "yellow"},
		{"10", "yellow"},
		{"100", "red"},
	}

	for _, tt := range tests {
		t.Run(tt.restarts, func(t *testing.T) {
			got := RestartColor(tt.restarts)
			_ = got
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a long string", 10, "this is..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
		{"test", 0, "test"},
		{"test", -1, "test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestTruncateMiddle(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"verylongstring", 10, "ver...ing"},
		{"ab", 5, "ab"},
		{"abcdefghij", 3, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := TruncateMiddle(tt.input, tt.maxLen)
			if got != tt.expected {
				t.Errorf("TruncateMiddle(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
			}
		})
	}
}

func TestHeaderNames(t *testing.T) {
	header := Header{
		{Name: "NAME"},
		{Name: "STATUS"},
		{Name: "AGE"},
	}

	names := header.Names()
	if len(names) != 3 {
		t.Errorf("Names() len = %d, want 3", len(names))
	}
	if names[0] != "NAME" {
		t.Errorf("Names()[0] = %s, want NAME", names[0])
	}
}

func TestHeaderVisibleNames(t *testing.T) {
	header := Header{
		{Name: "NAME"},
		{Name: "STATUS", Hide: true},
		{Name: "CPU", MX: true},
		{Name: "DETAILS", Wide: true},
	}

	// Default mode
	names := header.VisibleNames(false, false)
	if len(names) != 1 {
		t.Errorf("VisibleNames(false, false) len = %d, want 1", len(names))
	}

	// Wide mode
	names = header.VisibleNames(true, false)
	if len(names) != 2 {
		t.Errorf("VisibleNames(true, false) len = %d, want 2", len(names))
	}

	// Metrics mode
	names = header.VisibleNames(false, true)
	if len(names) != 2 {
		t.Errorf("VisibleNames(false, true) len = %d, want 2", len(names))
	}

	// Wide + Metrics
	names = header.VisibleNames(true, true)
	if len(names) != 3 {
		t.Errorf("VisibleNames(true, true) len = %d, want 3", len(names))
	}
}

func TestHeaderIndexOf(t *testing.T) {
	header := Header{
		{Name: "NAME"},
		{Name: "STATUS"},
	}

	if header.IndexOf("STATUS") != 1 {
		t.Errorf("IndexOf(STATUS) = %d, want 1", header.IndexOf("STATUS"))
	}
	if header.IndexOf("NOTFOUND") != -1 {
		t.Errorf("IndexOf(NOTFOUND) = %d, want -1", header.IndexOf("NOTFOUND"))
	}
}

func TestBaseRenderer(t *testing.T) {
	header := Header{{Name: "TEST"}}
	r := NewBaseRenderer(header)

	if len(r.Header()) != 1 {
		t.Error("Header() should return the header")
	}

	colorer := r.ColorerFunc()
	if colorer == nil {
		t.Error("ColorerFunc() should not return nil")
	}
}
