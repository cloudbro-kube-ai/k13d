package resources

import (
	"testing"
	"time"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"negative duration", -5 * time.Second, "0s"},
		{"zero", 0, "0s"},
		{"seconds", 45 * time.Second, "45s"},
		{"one minute", 60 * time.Second, "1m"},
		{"minutes", 5 * time.Minute, "5m"},
		{"one hour", 60 * time.Minute, "1h"},
		{"hours", 12 * time.Hour, "12h"},
		{"one day", 24 * time.Hour, "1d"},
		{"days", 3 * 24 * time.Hour, "3d"},
		{"many days", 30 * 24 * time.Hour, "30d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAge(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatAge(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"empty string", "", 10, ""},
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "hel"},
		{"max is 4", "hello world", 4, "h..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestContainsFilter(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		filter   string
		expected bool
	}{
		{"empty filter", "hello world", "", true},
		{"matching filter", "hello world", "world", true},
		{"case insensitive match", "Hello World", "hello", true},
		{"no match", "hello world", "foo", false},
		{"empty text", "", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsFilter(tt.text, tt.filter)
			if result != tt.expected {
				t.Errorf("ContainsFilter(%q, %q) = %v, want %v", tt.text, tt.filter, result, tt.expected)
			}
		})
	}
}

func TestFormatCPU(t *testing.T) {
	tests := []struct {
		name       string
		milliCores int64
		expected   string
	}{
		{"zero", 0, "-"},
		{"negative", -100, "-"},
		{"normal value", 100, "100m"},
		{"large value", 2500, "2500m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCPU(tt.milliCores)
			if result != tt.expected {
				t.Errorf("FormatCPU(%d) = %q, want %q", tt.milliCores, result, tt.expected)
			}
		})
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero", 0, "-"},
		{"negative", -100, "-"},
		{"bytes", 500, "500B"},
		{"kilobytes", 2 * 1024, "2Ki"},
		{"megabytes", 128 * 1024 * 1024, "128Mi"},
		{"gigabytes", 4 * 1024 * 1024 * 1024, "4Gi"},
		{"terabytes", 2 * 1024 * 1024 * 1024 * 1024, "2Ti"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMemory(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatMemory(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatMemoryMB(t *testing.T) {
	tests := []struct {
		name     string
		mb       int64
		expected string
	}{
		{"zero", 0, "-"},
		{"normal", 256, "256MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMemoryMB(tt.mb)
			if result != tt.expected {
				t.Errorf("FormatMemoryMB(%d) = %q, want %q", tt.mb, result, tt.expected)
			}
		})
	}
}

func TestFormatPercentage(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		total    int64
		expected string
	}{
		{"zero total", 50, 0, "-"},
		{"half", 50, 100, "50.0%"},
		{"full", 100, 100, "100.0%"},
		{"quarter", 25, 100, "25.0%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPercentage(tt.value, tt.total)
			if result != tt.expected {
				t.Errorf("FormatPercentage(%d, %d) = %q, want %q", tt.value, tt.total, result, tt.expected)
			}
		})
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		sep      string
		strs     []string
		expected string
	}{
		{"empty", ", ", []string{}, ""},
		{"single", ", ", []string{"hello"}, "hello"},
		{"multiple", ", ", []string{"a", "b", "c"}, "a, b, c"},
		{"with empty strings", ", ", []string{"a", "", "c"}, "a, c"},
		{"all empty", ", ", []string{"", "", ""}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JoinStrings(tt.sep, tt.strs...)
			if result != tt.expected {
				t.Errorf("JoinStrings(%q, %v) = %q, want %q", tt.sep, tt.strs, result, tt.expected)
			}
		})
	}
}
