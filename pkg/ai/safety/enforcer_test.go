package safety

import (
	"testing"
	"time"

	"github.com/kube-ai-dashbaord/kube-ai-dashboard-cli/pkg/config"
)

func TestPolicyEnforcer_Evaluate(t *testing.T) {
	// Test with default policy
	enforcer := NewDefaultPolicyEnforcer()

	tests := []struct {
		name             string
		command          string
		expectedAllowed  bool
		expectedApproval bool
		expectedCategory string
	}{
		// Read-only should auto-approve with default policy
		{
			name:             "kubectl get pods auto-approved",
			command:          "kubectl get pods",
			expectedAllowed:  true,
			expectedApproval: false,
			expectedCategory: "read-only",
		},
		// Write should require approval
		{
			name:             "kubectl apply requires approval",
			command:          "kubectl apply -f deployment.yaml",
			expectedAllowed:  true,
			expectedApproval: true,
			expectedCategory: "write",
		},
		// Dangerous should require approval (not blocked by default)
		{
			name:             "kubectl delete --all requires approval",
			command:          "kubectl delete pods --all",
			expectedAllowed:  true,
			expectedApproval: true,
			expectedCategory: "dangerous",
		},
		// Unknown should require approval
		{
			name:             "unknown command requires approval",
			command:          "./custom-script.sh",
			expectedAllowed:  true,
			expectedApproval: true,
			expectedCategory: "unknown",
		},
		// Interactive should require approval
		{
			name:             "kubectl exec requires approval",
			command:          "kubectl exec -it nginx -- bash",
			expectedAllowed:  true,
			expectedApproval: true,
			expectedCategory: "interactive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := enforcer.Evaluate(tt.command)

			if decision.Allowed != tt.expectedAllowed {
				t.Errorf("Allowed = %v, want %v", decision.Allowed, tt.expectedAllowed)
			}

			if decision.RequiresApproval != tt.expectedApproval {
				t.Errorf("RequiresApproval = %v, want %v", decision.RequiresApproval, tt.expectedApproval)
			}

			if decision.Category != tt.expectedCategory {
				t.Errorf("Category = %q, want %q", decision.Category, tt.expectedCategory)
			}
		})
	}
}

func TestPolicyEnforcer_BlockDangerous(t *testing.T) {
	policy := config.ToolApprovalPolicy{
		AutoApproveReadOnly:       true,
		RequireApprovalForWrite:   true,
		RequireApprovalForUnknown: true,
		BlockDangerous:            true, // Block dangerous commands
		ApprovalTimeoutSeconds:    60,
	}

	enforcer := NewPolicyEnforcer(policy)

	tests := []struct {
		name            string
		command         string
		expectedAllowed bool
	}{
		{
			name:            "read-only allowed",
			command:         "kubectl get pods",
			expectedAllowed: true,
		},
		{
			name:            "write allowed",
			command:         "kubectl apply -f deployment.yaml",
			expectedAllowed: true,
		},
		{
			name:            "dangerous blocked",
			command:         "kubectl delete pods --all",
			expectedAllowed: false,
		},
		{
			name:            "drain blocked",
			command:         "kubectl drain node-1",
			expectedAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := enforcer.Evaluate(tt.command)

			if decision.Allowed != tt.expectedAllowed {
				t.Errorf("Allowed = %v, want %v (BlockReason: %s)", decision.Allowed, tt.expectedAllowed, decision.BlockReason)
			}
		})
	}
}

func TestPolicyEnforcer_BlockedPatterns(t *testing.T) {
	policy := config.ToolApprovalPolicy{
		AutoApproveReadOnly:       true,
		RequireApprovalForWrite:   true,
		RequireApprovalForUnknown: true,
		BlockDangerous:            false,
		BlockedPatterns:           []string{`rm\s+-rf`, `DROP\s+TABLE`, `production`},
		ApprovalTimeoutSeconds:    60,
	}

	enforcer := NewPolicyEnforcer(policy)

	tests := []struct {
		name            string
		command         string
		expectedAllowed bool
	}{
		{
			name:            "normal command allowed",
			command:         "kubectl get pods",
			expectedAllowed: true,
		},
		{
			name:            "rm -rf blocked",
			command:         "rm -rf /tmp/test",
			expectedAllowed: false,
		},
		{
			name:            "production namespace blocked",
			command:         "kubectl delete namespace production",
			expectedAllowed: false,
		},
		{
			name:            "DROP TABLE blocked",
			command:         "mysql -e 'DROP TABLE users'",
			expectedAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := enforcer.Evaluate(tt.command)

			if decision.Allowed != tt.expectedAllowed {
				t.Errorf("Allowed = %v, want %v (BlockReason: %s)", decision.Allowed, tt.expectedAllowed, decision.BlockReason)
			}
		})
	}
}

func TestPolicyEnforcer_RequireApprovalForAll(t *testing.T) {
	// Policy that requires approval for everything
	policy := config.ToolApprovalPolicy{
		AutoApproveReadOnly:       false, // Require approval even for read-only
		RequireApprovalForWrite:   true,
		RequireApprovalForUnknown: true,
		BlockDangerous:            false,
		ApprovalTimeoutSeconds:    60,
	}

	enforcer := NewPolicyEnforcer(policy)

	// Even read-only should require approval
	decision := enforcer.Evaluate("kubectl get pods")

	if !decision.RequiresApproval {
		t.Error("Expected approval required for read-only command when AutoApproveReadOnly=false")
	}
}

func TestPolicyEnforcer_GetApprovalTimeout(t *testing.T) {
	tests := []struct {
		name            string
		timeoutSeconds  int
		expectedTimeout time.Duration
	}{
		{
			name:            "custom timeout",
			timeoutSeconds:  30,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "default timeout for zero",
			timeoutSeconds:  0,
			expectedTimeout: 60 * time.Second,
		},
		{
			name:            "default timeout for negative",
			timeoutSeconds:  -1,
			expectedTimeout: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := config.ToolApprovalPolicy{
				ApprovalTimeoutSeconds: tt.timeoutSeconds,
			}
			enforcer := NewPolicyEnforcer(policy)

			if got := enforcer.GetApprovalTimeout(); got != tt.expectedTimeout {
				t.Errorf("GetApprovalTimeout() = %v, want %v", got, tt.expectedTimeout)
			}
		})
	}
}

func TestPolicyEnforcer_UpdatePolicy(t *testing.T) {
	enforcer := NewDefaultPolicyEnforcer()

	// Initial policy allows dangerous commands
	decision := enforcer.Evaluate("kubectl delete pods --all")
	if !decision.Allowed {
		t.Error("Expected dangerous command to be allowed with default policy")
	}

	// Update policy to block dangerous
	newPolicy := config.ToolApprovalPolicy{
		AutoApproveReadOnly:       true,
		RequireApprovalForWrite:   true,
		RequireApprovalForUnknown: true,
		BlockDangerous:            true,
		ApprovalTimeoutSeconds:    60,
	}
	enforcer.UpdatePolicy(newPolicy)

	// Now should be blocked
	decision = enforcer.Evaluate("kubectl delete pods --all")
	if decision.Allowed {
		t.Error("Expected dangerous command to be blocked after policy update")
	}
}

func TestPolicyEnforcer_PipedCommands(t *testing.T) {
	// This is the critical security test - piped commands must require approval
	enforcer := NewDefaultPolicyEnforcer()

	tests := []struct {
		name             string
		command          string
		expectedApproval bool
	}{
		{
			name:             "simple get - no approval",
			command:          "kubectl get pods",
			expectedApproval: false,
		},
		{
			name:             "piped get - requires approval",
			command:          "kubectl get pods | grep nginx",
			expectedApproval: true,
		},
		{
			name:             "dangerous pipe - requires approval",
			command:          "kubectl get pods | xargs rm -rf",
			expectedApproval: true,
		},
		{
			name:             "chained commands - requires approval",
			command:          "kubectl get pods && rm -rf /",
			expectedApproval: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := enforcer.Evaluate(tt.command)

			if decision.RequiresApproval != tt.expectedApproval {
				t.Errorf("RequiresApproval = %v, want %v (command: %s)",
					decision.RequiresApproval, tt.expectedApproval, tt.command)
			}
		})
	}
}

func TestDefaultEnforcer(t *testing.T) {
	// Test package-level convenience function
	decision := Evaluate("kubectl get pods")

	if !decision.Allowed {
		t.Error("Expected command to be allowed")
	}

	if decision.RequiresApproval {
		t.Error("Expected no approval required for kubectl get pods")
	}
}
