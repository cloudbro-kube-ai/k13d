package ui

import (
	"testing"
)

func TestGetCommandDescription(t *testing.T) {
	tests := []struct {
		cmd      string
		expected string
	}{
		{"kubectl delete pod nginx", "Delete resource"},
		{"kubectl apply -f deployment.yaml", "Apply configuration"},
		{"kubectl create namespace test", "Create resource"},
		{"kubectl scale deployment nginx --replicas=3", "Scale resource"},
		{"kubectl rollout restart deployment nginx", "Rollout operation"},
		{"kubectl patch pod nginx", "Patch resource"},
		{"kubectl edit deployment nginx", "Edit resource"},
		{"kubectl drain node-1", "Drain node"},
		{"kubectl cordon node-1", "Cordon node"},
		{"kubectl uncordon node-1", "Uncordon node"},
		{"kubectl get pods", "get"},
		{"delete pod", "Delete resource"},
		{"apply", "apply"}, // Single word returns itself
		{"", ""},
		{"kubectl", "kubectl"},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			result := getCommandDescription(tt.cmd)
			if result != tt.expected {
				t.Errorf("getCommandDescription(%q) = %q, want %q", tt.cmd, result, tt.expected)
			}
		})
	}
}

func TestPendingDecisionStruct(t *testing.T) {
	// Test that PendingDecision struct can hold all necessary fields
	decision := PendingDecision{
		Command:     "kubectl delete pod nginx",
		Description: "Delete resource",
		IsDangerous: true,
		Warnings:    []string{"This will delete the pod", "Cannot be undone"},
		ToolName:    "kubectl",
		ToolArgs:    `{"command": "delete pod nginx"}`,
		IsToolCall:  true,
	}

	if decision.Command != "kubectl delete pod nginx" {
		t.Errorf("Expected Command 'kubectl delete pod nginx', got %q", decision.Command)
	}
	if !decision.IsDangerous {
		t.Error("Expected IsDangerous to be true")
	}
	if len(decision.Warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(decision.Warnings))
	}
	if !decision.IsToolCall {
		t.Error("Expected IsToolCall to be true")
	}
}
