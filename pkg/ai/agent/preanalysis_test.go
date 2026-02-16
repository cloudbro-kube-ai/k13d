package agent

import (
	"context"
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/ai/analyzers"
)

func TestParseResourceContext_Full(t *testing.T) {
	ctx := "Kind: Pod\nName: my-pod\nNamespace: default\nStatus: CrashLoopBackOff\nRestarts: 15"

	info := parseResourceContext(ctx)
	if info == nil {
		t.Fatal("parseResourceContext returned nil")
	}

	if info.Kind != "Pod" {
		t.Errorf("Kind = %q, want %q", info.Kind, "Pod")
	}
	if info.Name != "my-pod" {
		t.Errorf("Name = %q, want %q", info.Name, "my-pod")
	}
	if info.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", info.Namespace, "default")
	}
	if info.Status != "CrashLoopBackOff" {
		t.Errorf("Status = %q, want %q", info.Status, "CrashLoopBackOff")
	}
	if len(info.Containers) != 1 {
		t.Fatalf("Containers count = %d, want 1", len(info.Containers))
	}
	if info.Containers[0].RestartCount != 15 {
		t.Errorf("RestartCount = %d, want 15", info.Containers[0].RestartCount)
	}
}

func TestParseResourceContext_Empty(t *testing.T) {
	info := parseResourceContext("")
	if info != nil {
		t.Error("parseResourceContext(\"\") should return nil")
	}
}

func TestParseResourceContext_Whitespace(t *testing.T) {
	info := parseResourceContext("   \n  \n  ")
	if info != nil {
		t.Error("parseResourceContext with only whitespace should return nil")
	}
}

func TestParseResourceContext_MinimalKind(t *testing.T) {
	info := parseResourceContext("Kind: Deployment")
	if info == nil {
		t.Fatal("parseResourceContext returned nil for kind-only context")
	}
	if info.Kind != "Deployment" {
		t.Errorf("Kind = %q, want %q", info.Kind, "Deployment")
	}
}

func TestParseResourceContext_StatusOnly(t *testing.T) {
	info := parseResourceContext("Status: Running")
	if info == nil {
		t.Fatal("parseResourceContext returned nil for status-only context")
	}
	if info.Status != "Running" {
		t.Errorf("Status = %q, want %q", info.Status, "Running")
	}
}

func TestParseResourceContext_NoIdentifiers(t *testing.T) {
	// Lines without recognized key-value pairs
	info := parseResourceContext("some random text\nanother line")
	if info != nil {
		t.Error("parseResourceContext should return nil when no identifiers found")
	}
}

func TestParseResourceContext_TypeAlias(t *testing.T) {
	info := parseResourceContext("Type: Service\nName: my-svc")
	if info == nil {
		t.Fatal("parseResourceContext returned nil")
	}
	if info.Kind != "Service" {
		t.Errorf("Kind = %q, want %q", info.Kind, "Service")
	}
}

func TestParseResourceContext_PhaseAlias(t *testing.T) {
	info := parseResourceContext("Kind: Pod\nPhase: Pending")
	if info == nil {
		t.Fatal("parseResourceContext returned nil")
	}
	if info.Status != "Pending" {
		t.Errorf("Status = %q, want %q", info.Status, "Pending")
	}
}

func TestParseResourceContext_RestartsVariants(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantN   int32
		wantNil bool
	}{
		{
			name:  "Restarts field",
			input: "Kind: Pod\nRestarts: 5",
			wantN: 5,
		},
		{
			name:  "Restart field (no s)",
			input: "Kind: Pod\nRestart: 3",
			wantN: 3,
		},
		{
			name:    "No restarts",
			input:   "Kind: Pod\nStatus: Running",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := parseResourceContext(tt.input)
			if info == nil {
				t.Fatal("parseResourceContext returned nil")
			}
			if tt.wantNil {
				if len(info.Containers) != 0 {
					t.Errorf("Containers count = %d, want 0", len(info.Containers))
				}
				return
			}
			if len(info.Containers) != 1 {
				t.Fatalf("Containers count = %d, want 1", len(info.Containers))
			}
			if info.Containers[0].RestartCount != tt.wantN {
				t.Errorf("RestartCount = %d, want %d", info.Containers[0].RestartCount, tt.wantN)
			}
		})
	}
}

func TestRunPreAnalysis_WithFindings(t *testing.T) {
	registry := analyzers.NewRegistry()
	registry.Register(&analyzers.PodAnalyzer{})

	agent := New(&Config{AnalyzerRegistry: registry})

	result := agent.runPreAnalysis(context.Background(), "Kind: Pod\nStatus: CrashLoopBackOff\nRestarts: 10")
	if result == "" {
		t.Error("runPreAnalysis should return findings for CrashLoopBackOff pod")
	}
	if !containsString(result, "[Pre-analysis findings]") {
		t.Error("Result should contain [Pre-analysis findings] header")
	}
}

func TestRunPreAnalysis_NoFindings(t *testing.T) {
	registry := analyzers.NewRegistry()
	// Empty registry = no analyzers

	agent := New(&Config{AnalyzerRegistry: registry})

	result := agent.runPreAnalysis(context.Background(), "Kind: Pod\nStatus: Running")
	if result != "" {
		t.Errorf("runPreAnalysis with empty registry should return empty, got %q", result)
	}
}
