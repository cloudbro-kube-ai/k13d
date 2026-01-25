package ai

import (
	"testing"
)

func TestCommandFilter_ClassifyCommand(t *testing.T) {
	filter := NewCommandFilter()

	tests := []struct {
		name    string
		command string
		want    CommandType
	}{
		{
			name:    "read only - get pods",
			command: "kubectl get pods",
			want:    CommandTypeReadOnly,
		},
		{
			name:    "read only - describe deployment",
			command: "kubectl describe deployment nginx",
			want:    CommandTypeReadOnly,
		},
		{
			name:    "read only - logs",
			command: "kubectl logs nginx-pod",
			want:    CommandTypeReadOnly,
		},
		{
			name:    "write - apply",
			command: "kubectl apply -f deployment.yaml",
			want:    CommandTypeWrite,
		},
		{
			name:    "write - create",
			command: "kubectl create namespace test",
			want:    CommandTypeWrite,
		},
		{
			name:    "write - patch",
			command: "kubectl patch deployment nginx -p '{\"spec\":{\"replicas\":3}}'",
			want:    CommandTypeWrite,
		},
		{
			name:    "dangerous - delete all",
			command: "kubectl delete pods --all",
			want:    CommandTypeDangerous,
		},
		{
			name:    "dangerous - drain force",
			command: "kubectl drain node-1 --force",
			want:    CommandTypeDangerous,
		},
		{
			name:    "interactive - exec",
			command: "kubectl exec -it nginx -- /bin/bash",
			want:    CommandTypeInteractive,
		},
		{
			name:    "interactive - port-forward",
			command: "kubectl port-forward svc/nginx 8080:80",
			want:    CommandTypeInteractive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filter.ClassifyCommand(tt.command)
			if got != tt.want {
				t.Errorf("ClassifyCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestCommandFilter_AnalyzeCommand(t *testing.T) {
	filter := NewCommandFilter()

	tests := []struct {
		name        string
		command     string
		wantDanger  bool
		wantWarning bool
	}{
		{
			name:        "safe read command",
			command:     "kubectl get pods",
			wantDanger:  false,
			wantWarning: false,
		},
		{
			name:        "dangerous delete all command",
			command:     "kubectl delete pods --all",
			wantDanger:  true,
			wantWarning: true,
		},
		{
			name:        "very dangerous delete all namespaces",
			command:     "kubectl delete pods --all -A",
			wantDanger:  true,
			wantWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := filter.AnalyzeCommand(tt.command)
			if report.IsDangerous != tt.wantDanger {
				t.Errorf("AnalyzeCommand(%q).IsDangerous = %v, want %v", tt.command, report.IsDangerous, tt.wantDanger)
			}
			hasWarning := len(report.Warnings) > 0
			if hasWarning != tt.wantWarning {
				t.Errorf("AnalyzeCommand(%q) warnings = %v, want warning = %v", tt.command, report.Warnings, tt.wantWarning)
			}
		})
	}
}

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

func TestCommandFilter_IsReadOnly(t *testing.T) {
	filter := NewCommandFilter()

	tests := []struct {
		command string
		want    bool
	}{
		{"kubectl get pods", true},
		{"kubectl describe pod nginx", true},
		{"kubectl logs nginx", true},
		{"kubectl delete pod nginx", false},
		{"kubectl apply -f file.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := filter.IsReadOnly(tt.command)
			if got != tt.want {
				t.Errorf("IsReadOnly(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestCommandFilter_RequiresConfirmation(t *testing.T) {
	filter := NewCommandFilter()

	tests := []struct {
		command string
		want    bool
	}{
		{"kubectl get pods", false},
		{"kubectl delete pod nginx", true},
		{"kubectl apply -f file.yaml", true},
		{"kubectl exec -it nginx -- bash", false}, // interactive, not confirmation
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := filter.RequiresConfirmation(tt.command)
			if got != tt.want {
				t.Errorf("RequiresConfirmation(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}
