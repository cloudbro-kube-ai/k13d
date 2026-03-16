package web

import "testing"

func TestAllowAIToolExecution(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		toolName string
		command  string
		allowed  bool
	}{
		{
			name:     "viewer can use read only kubectl",
			role:     "viewer",
			toolName: "kubectl",
			command:  "get pods -A",
			allowed:  true,
		},
		{
			name:     "viewer cannot use bash",
			role:     "viewer",
			toolName: "bash",
			command:  "ls -la",
			allowed:  false,
		},
		{
			name:     "viewer cannot use write kubectl",
			role:     "viewer",
			toolName: "kubectl",
			command:  "delete pod nginx",
			allowed:  false,
		},
		{
			name:     "viewer cannot use interactive kubectl",
			role:     "viewer",
			toolName: "kubectl",
			command:  "exec nginx-pod -- sh",
			allowed:  false,
		},
		{
			name:     "user keeps existing access",
			role:     "user",
			toolName: "bash",
			command:  "ls -la",
			allowed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, reason := allowAIToolExecution(tt.role, tt.toolName, tt.command)
			if allowed != tt.allowed {
				t.Fatalf("allowAIToolExecution() = %v, want %v (reason=%q)", allowed, tt.allowed, reason)
			}
		})
	}
}

func TestBuildAIRestrictionPrompt(t *testing.T) {
	if prompt := buildAIRestrictionPrompt("viewer"); prompt == "" {
		t.Fatal("expected viewer prompt to describe read-only AI restrictions")
	}
	if prompt := buildAIRestrictionPrompt("admin"); prompt != "" {
		t.Fatalf("expected admin prompt to be empty, got %q", prompt)
	}
}
