package tools

import "testing"

func TestExtractCommandForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		argsJSON string
		want     string
	}{
		{
			name:     "standard command field",
			toolName: "bash",
			argsJSON: `{"command":"echo hello","timeout":30}`,
			want:     "echo hello",
		},
		{
			name:     "cmd alias for bash-like tool",
			toolName: "bash",
			argsJSON: `{"cmd":"echo hello"}`,
			want:     "echo hello",
		},
		{
			name:     "script alias for shell tool",
			toolName: "shell",
			argsJSON: `{"script":"echo hello"}`,
			want:     "echo hello",
		},
		{
			name:     "nested arguments wrapper",
			toolName: "bash",
			argsJSON: `{"arguments":{"command":"echo hello"}}`,
			want:     "echo hello",
		},
		{
			name:     "args array is joined",
			toolName: "kubectl",
			argsJSON: `{"args":["get","pods","-A"]}`,
			want:     "get pods -A",
		},
		{
			name:     "json string payload",
			toolName: "bash",
			argsJSON: `"echo hello"`,
			want:     "echo hello",
		},
		{
			name:     "raw non json payload",
			toolName: "bash",
			argsJSON: `echo hello`,
			want:     "echo hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractCommandForDisplay(tt.toolName, tt.argsJSON); got != tt.want {
				t.Fatalf("ExtractCommandForDisplay(%q, %q) = %q, want %q", tt.toolName, tt.argsJSON, got, tt.want)
			}
		})
	}
}
