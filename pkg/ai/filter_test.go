package ai

import (
	"testing"
)

func TestExtractKubectlCommands(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "single command on its own line",
			text: "kubectl get pods",
			want: []string{"kubectl get pods"},
		},
		{
			name: "multiple commands on separate lines",
			text: "kubectl get pods\nkubectl describe pod nginx",
			want: []string{"kubectl get pods", "kubectl describe pod nginx"},
		},
		{
			name: "no commands",
			text: "This is just regular text without any commands",
			want: nil,
		},
		{
			name: "code block command",
			text: "```\nkubectl apply -f deployment.yaml\n```",
			want: []string{"kubectl apply -f deployment.yaml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractKubectlCommands(tt.text)
			if len(got) != len(tt.want) {
				t.Errorf("ExtractKubectlCommands(%q) = %v, want %v", tt.text, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ExtractKubectlCommands(%q)[%d] = %q, want %q", tt.text, i, got[i], tt.want[i])
				}
			}
		})
	}
}
