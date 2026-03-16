package web

import (
	"strings"
	"testing"

	"github.com/cloudbro-kube-ai/k13d/pkg/config"
)

func TestNormalizeAIToolCommand_PrefixesKubectl(t *testing.T) {
	if got := normalizeAIToolCommand("kubectl", "get pods -A"); got != "kubectl get pods -A" {
		t.Fatalf("normalizeAIToolCommand() = %q, want %q", got, "kubectl get pods -A")
	}
}

func TestEvaluateAIToolDecision_KubectlWriteRequiresApproval(t *testing.T) {
	s := &Server{
		cfg: &config.Config{
			Authorization: config.AuthorizationConfig{
				ToolApproval: config.DefaultToolApprovalPolicy(),
			},
		},
	}

	decision := s.evaluateAIToolDecision("admin", "kubectl", "apply -f deploy.yaml")
	if !decision.Allowed {
		t.Fatalf("expected kubectl apply to be allowed with approval, got blocked: %s", decision.BlockReason)
	}
	if !decision.RequiresApproval {
		t.Fatal("expected kubectl apply to require approval")
	}
	if decision.Category != "write" {
		t.Fatalf("decision.Category = %q, want write", decision.Category)
	}
}

func TestEvaluateAIToolDecision_KubectlReadOnlyRequiresApprovalByDefault(t *testing.T) {
	s := &Server{
		cfg: &config.Config{
			Authorization: config.AuthorizationConfig{
				ToolApproval: config.DefaultToolApprovalPolicy(),
			},
		},
	}

	decision := s.evaluateAIToolDecision("admin", "kubectl", "get pods -n default")
	if !decision.Allowed {
		t.Fatalf("expected kubectl get to stay allowed, got blocked: %s", decision.BlockReason)
	}
	if !decision.RequiresApproval {
		t.Fatal("expected kubectl get to require approval by default")
	}
	if decision.Category != "read-only" {
		t.Fatalf("decision.Category = %q, want read-only", decision.Category)
	}
}

func TestEvaluateAIToolDecision_BashKubectlIsBlocked(t *testing.T) {
	s := &Server{
		cfg: &config.Config{
			Authorization: config.AuthorizationConfig{
				ToolApproval: config.DefaultToolApprovalPolicy(),
			},
		},
	}

	decision := s.evaluateAIToolDecision("admin", "bash", "kubectl apply -f deploy.yaml")
	if decision.Allowed {
		t.Fatal("expected bash-wrapped kubectl command to be blocked")
	}
	if !strings.Contains(decision.BlockReason, "kubectl tool") {
		t.Fatalf("unexpected block reason: %q", decision.BlockReason)
	}
}

func TestEvaluateAIToolDecision_BashAlwaysRequiresApproval(t *testing.T) {
	s := &Server{
		cfg: &config.Config{
			Authorization: config.AuthorizationConfig{
				ToolApproval: config.ToolApprovalPolicy{
					AutoApproveReadOnly:       true,
					RequireApprovalForWrite:   false,
					RequireApprovalForUnknown: false,
					ApprovalTimeoutSeconds:    60,
				},
			},
		},
	}

	decision := s.evaluateAIToolDecision("admin", "bash", "date")
	if !decision.Allowed {
		t.Fatalf("expected plain bash command to remain allowed, got blocked: %s", decision.BlockReason)
	}
	if !decision.RequiresApproval {
		t.Fatal("expected bash command to always require approval")
	}
	if decision.Category != "bash" {
		t.Fatalf("decision.Category = %q, want bash", decision.Category)
	}
}
