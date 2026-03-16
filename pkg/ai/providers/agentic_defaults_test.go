package providers

import (
	"strings"
	"testing"
)

func TestEffectiveMaxIterations(t *testing.T) {
	tests := []struct {
		name string
		cfg  *ProviderConfig
		want int
	}{
		{name: "nil config uses default", cfg: nil, want: defaultToolLoopIterations},
		{name: "zero uses default", cfg: &ProviderConfig{MaxIterations: 0}, want: defaultToolLoopIterations},
		{name: "below minimum clamps", cfg: &ProviderConfig{MaxIterations: 1}, want: minimumToolLoopIterations},
		{name: "in range preserved", cfg: &ProviderConfig{MaxIterations: 7}, want: 7},
		{name: "above maximum clamps", cfg: &ProviderConfig{MaxIterations: 99}, want: maximumToolLoopIterations},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveMaxIterations(tt.cfg); got != tt.want {
				t.Fatalf("effectiveMaxIterations() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildToolUseShimSystemPromptUsesSortedToolInventory(t *testing.T) {
	tools := []ToolDefinition{
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "zeta_tool",
				Description: "Zeta",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "alpha_tool",
				Description: "Alpha",
				Parameters:  map[string]interface{}{"type": "object"},
			},
		},
	}

	prompt := buildToolUseShimSystemPrompt(tools, 6)
	if !strings.Contains(prompt, "budget of at most 6 tool-use rounds") {
		t.Fatalf("prompt should include configured tool budget, got %q", prompt)
	}
	if !strings.Contains(prompt, "Use only exact tool names from this list: alpha_tool, zeta_tool.") {
		t.Fatalf("prompt should list sorted tool names, got %q", prompt)
	}

	alphaIdx := strings.Index(prompt, `"name": "alpha_tool"`)
	zetaIdx := strings.Index(prompt, `"name": "zeta_tool"`)
	if alphaIdx == -1 || zetaIdx == -1 {
		t.Fatalf("prompt should include serialized tool definitions, got %q", prompt)
	}
	if alphaIdx > zetaIdx {
		t.Fatalf("prompt should serialize tools in sorted order, got %q", prompt)
	}
}
